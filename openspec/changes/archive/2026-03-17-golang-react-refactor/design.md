## 背景

当前 oReader 是一个 Flask + AngularJS 1.x 单页应用，用于 RSS 阅读。本设计文档概述了使用 Go（后端）和 React（前端）完全重写的架构。

### 当前状态
- Flask 后端，使用 Flask-RESTful 提供 API
- SQLAlchemy ORM，支持 SQLite/PostgreSQL
- 基于 lxml 的自定义 RSS 解析器
- AngularJS 1.x 前端，使用 angular-route、$resource
- HTTP Basic Auth，使用 itsdangerous 令牌
- Heroku 部署，使用 gunicorn

### 约束
- 团队技能：Go > Python，正在学习 React
- 优先单一二进制部署
- 生产环境使用 MySQL，开发环境使用 SQLite
- 必须支持多租户（多用户）

## 目标 / 非目标

**目标：**
- 现代、可维护的技术栈
- 生产级安全（双令牌、HttpOnly cookies、CSRF 保护）
- 嵌入前端的单一二进制部署
- 可扩展的架构以支持未来功能
- 带 background worker 的自动 RSS 刷新
- GitHub OAuth 集成（预留接口）

**非目标：**
- 从旧系统数据迁移（使用 OPML 迁移路径全新开始）
- 实时更新（WebSocket）- 可后续添加
- 移动应用 - Web 优先
- 全文搜索 - 可后续添加
- 社交功能（分享、评论）

**开发方法论：**
- 测试驱动开发（TDD）：先编写失败测试，然后实现使其通过
- Git 提交：每个完成的阶段后提交，附带描述性消息

## 决策

### 1. 后端框架：Gin

**选择：** Gin 而非 Echo、Fiber、stdlib

**理由：**
- 成熟的生态系统，丰富的中间件
- 性能可与其他替代方案媲美
- 熟悉的 Flask 风格路由模式
- 良好的文档和社区

### 2. ORM：GORM

**选择：** GORM 而非 sqlx、ent

**理由：**
- 与 SQLAlchemy 类似的模式（熟悉的过渡）
- 开发环境支持自动迁移
- 良好的 MySQL 和 SQLite 支持
- 活跃的社区

### 3. 认证：双令牌 + HttpOnly Cookies + CSRF 保护

**选择：** 访问令牌（15分钟）+ 刷新令牌（7天），存储在 HttpOnly + SameSite=Strict cookies 中，并显式 CSRF 令牌保护

**理由：**
- 短期访问令牌限制攻击窗口
- 存储在数据库中的刷新令牌可被撤销
- HttpOnly cookies 防止 XSS 令牌窃取
- SameSite=Strict 提供强大的 CSRF 保护
- **显式 CSRF 令牌**作为纵深防御（并非所有浏览器都支持 SameSite）

**流程：**
```
Login → Set-Cookie: access_token (15min, Path=/, HttpOnly, SameSite=Strict)
       → Set-Cookie: refresh_token (7day, Path=/api/v1/auth/refresh, HttpOnly, SameSite=Strict)
       → Set-Cookie: csrf_token (15min, Path=/, HttpOnly=false) [供 JS 读取]
       → Response body includes csrf_token for frontend to store

API Request → Browser sends access_token cookie automatically
            → Frontend includes X-CSRF-Token header
            → Middleware validates JWT + CSRF token match, injects user_id

Token Expired → 401 with code=TOKEN_EXPIRED → Frontend calls /auth/refresh
                                        → Server validates refresh_token from cookie
                                        → Issues new access_token + csrf_token via Set-Cookie
```

### 4. 用户 ID：UUID v7

**选择：** UUID v7 而非自增整数

**理由：**
- 防止用户枚举攻击
- 时间可排序（v7 特性）
- JWT payload 不暴露用户数量
- 无业务信息泄露

### 5. RSS 解析：gofeed

**选择：** gofeed 而非自定义解析器

**理由：**
- 经过实战检验的库
- 支持 RSS 0.90、0.91、0.92、0.93、1.0、2.0 和 Atom 0.3、1.0
- 处理边缘情况和格式错误的订阅源
- 积极维护

### 6. 后台刷新：Goroutine + Ticker

**选择：** 进程内 goroutine 而非外部调度器（cron）

**理由：**
- 单一二进制部署
- 无外部依赖
- 对于此用例实现简单
- 如需要，后续可提取为独立服务

### 7. 前端构建：Vite + embed

**选择：** Vite + embed 包而非分离部署

**理由：**
- 单一二进制简化部署
- Vite 的快速 HMR 用于开发
- embed 包用于生产静态文件服务
- 开发/生产构建的清晰分离

### 8. 前端状态：React Query + Zustand

**选择：** React Query 用于服务器状态，Zustand 用于客户端状态

**理由：**
- React Query 处理缓存、重新获取、后台更新
- Zustand 用于认证状态的最小样板
- 清晰的关注点分离
- 两者都轻量且维护良好

### 9. UI 组件：shadcn/ui + Tailwind

**选择：** shadcn/ui 而非 Ant Design、Material UI

**理由：**
- 完全控制组件（复制粘贴，非依赖）
- Tailwind 开箱即用集成
- 现代、可访问的默认值
- React 模式的良好学习机会

### 10. 速率限制：Token Bucket + Redis（可选）

**选择：** 单实例使用内存 token bucket，分布式使用基于 Redis 的

**理由：**
- 保护 API 免受滥用和 DoS 攻击
- 滑动窗口算法实现更平滑的速率限制
- 可按端点类型配置限制
- Redis 可选用于水平扩展

### 11. Feed 导入/导出：OPML 2.0

**选择：** OPML 2.0 格式用于订阅源

**理由：**
- 大多数 RSS 阅读器支持的标准格式
- 简单的 XML 结构，易于解析
- 支持嵌套分类（未来使用）
- 人类可读

### 12. 安全头中间件

**选择：** 所有响应使用全面的安全头

**理由：**
- 针对常见 Web 漏洞的纵深防御
- 防止点击劫持、MIME 嗅探、XSS
- 现代浏览器安全功能

### 13. 内容清理：bluemonday

**选择：** bluemonday 用于 RSS 内容的 HTML 清理

**理由：**
- RSS 订阅源可能包含恶意 HTML/JavaScript
- XSS 防护用于用户生成内容显示
- 基于白名单的方法比黑名单更安全

### 14. SSRF 防护：URL 验证

**选择：** 阻止私有 IP 范围并验证 URL 协议

**理由：**
- 用户提供的订阅源 URL 可能针对内部服务
- 防止访问元数据端点（169.254.169.254）
- 限制服务端请求伪造的攻击面

### 15. 日志：zerolog 结构化输出

**选择：** zerolog 用于结构化、高性能日志

**理由：**
- 零分配 JSON 日志，性能优异
- 结构化日志便于搜索和分析
- 可配置日志级别
- 带字段的上下文感知日志

### 16. 数据库迁移：golang-migrate

**选择：** golang-migrate 而非生产环境使用 GORM AutoMigrate

**理由：**
- 版本控制的迁移，支持 up/down
- 生产事故的回滚能力
- 基于 SQL 的迁移实现精细控制
- 迁移历史跟踪

### 17. API 错误响应格式

**选择：** 一致的结构化错误响应

**理由：**
- 可预测的错误结构便于前端处理
- 错误码用于程序化处理
- 人类可读的消息
- 可选详情用于验证错误

### 18. 带接口的分层架构

**选择：** 层间显式接口定义

**理由：**
- 为单元测试启用正确的模拟
- 层间清晰的契约
- 依赖倒置原则
- 更容易替换实现

## 架构

### 后端结构
```
cmd/
└── server/main.go              # 入口点
internal/
├── config/                     # 配置 (viper)
├── domain/                     # 领域实体和业务规则
│   └── entity/                 # User, Feed, Item 实体
├── handler/                    # HTTP 处理程序 (Gin)
├── middleware/                 # Auth, CORS, logging, CSRF, security headers
├── service/                    # 业务逻辑
│   └── interfaces.go           # Repository 接口
├── repository/                 # 数据访问实现 (GORM)
├── model/                      # 数据库模型 (GORM structs)
└── infra/                      # 基础设施工具
    ├── jwt/                    # JWT 生成/验证
    ├── rss/                    # gofeed 包装器 + URL 验证
    ├── cookie/                 # Cookie 工具
    ├── sanitize/               # HTML 清理 (bluemonday)
    └── ratelimit/              # 速率限制 (内存 + Redis 接口)
migrations/                     # 数据库迁移 (golang-migrate)
web/                            # React 前端 (嵌入)
```

### 前端结构
```
web/src/
├── components/
│   └── ui/                     # shadcn 组件
├── features/
│   ├── auth/                   # Login, Register, authStore
│   ├── feeds/                  # FeedList, AddFeed
│   └── items/                  # ItemList, ItemView
├── hooks/                      # 自定义 hooks
├── lib/
│   ├── api/                    # 类型化 API 客户端
│   │   ├── client.ts           # Axios 实例
│   │   ├── feeds.ts            # Feed API 方法
│   │   └── items.ts            # Item API 方法
│   └── utils.ts                # 工具函数
├── stores/                     # Zustand stores
│   ├── authStore.ts            # 认证状态 (isAuthenticated, user)
│   └── uiStore.ts              # UI 状态 (sidebarOpen, theme)
├── types/                      # TypeScript 类型
│   └── api.ts                  # API 响应类型
└── App.tsx                     # 路由配置
```

### 数据模型

> **重要**：`is_starred` 和 `is_read` 是按用户的状态，而非按文章的状态。
> 如果直接存储在 Item 上，用户 A 的已读状态会影响用户 B。
> 解决方案：使用 `UserItemState` 关联表存储按用户的文章状态。

```
User (id: UUID v7) ──1:N──▶ UserFeed ──N:1──▶ Feed
       │                              │
       ├──1:N──▶ RefreshToken        └──1:N──▶ Item ──1:1──▶ UserItemState
       │
       └── fields: email, password_hash, nickname, avatar_url, auth_provider, github_id

UserFeed (用户订阅关系)
  └── fields: user_id, feed_id, position (排序), created_at

Feed (RSS源 - 共享)
  └── fields: feed_url (唯一), title, description, image_url, last_fetched_at, last_fetch_status, consecutive_failures

Item (文章内容 - 共享)
  └── fields: feed_id, guid (去重), title, link, description, content, pub_date, creator

UserItemState (用户文章状态 - 私有)
  └── fields: user_id, item_id, is_starred, is_read, read_at, created_at
```

## 风险 / 权衡

### 风险 1：基于 Cookie 的认证和 CORS
**风险：** SameSite=Strict cookies 可能无法与分离的前端域名一起工作
**缓解：**
- 部署为单一源（主要方法）
- 如需分离域名，使用 SameSite=Lax 并配合显式 CSRF 令牌
- 在部署指南中记录回退配置

### 风险 2：同一进程中的后台 Worker
**风险：** 长时间的订阅源刷新可能阻塞 API 响应
**缓解：**
- 使用带 context 超时的 goroutines（每个订阅源 30s）
- 使用信号量限制并发（最多 10 个并发获取）
- 计划在 v2.1 提取为独立 worker 服务

### 风险 3：RSS 订阅源恶意内容
**风险：** 订阅源可能在 HTML 内容中包含 XSS payload
**缓解：**
- 在提供服务前使用 bluemonday 清理所有 HTML 内容
- 对文章内容使用 UGC 策略
- 对订阅源标题/描述使用严格策略

### 风险 4：订阅源获取 SSRF
**风险：** 用户提供的 URL 可能针对内部服务
**缓解：**
- 阻止私有 IP 范围（10.x、172.16-31.x、192.168.x、127.x、169.254.x）
- 仅允许 http/https 协议
- 获取前验证 URL

## 配置

### 环境变量

| 变量 | 必需 | 默认值 | 描述 |
|----------|----------|---------|-------------|
| `DATABASE_URL` | 是 | - | 数据库连接字符串 |
| `JWT_SECRET_KEY` | 是 | - | JWT 签名密钥（256-bit） |
| `JWT_ACCESS_TTL` | 否 | 15m | 访问令牌有效期 |
| `JWT_REFRESH_TTL` | 否 | 168h | 刷新令牌有效期（7天） |
| `REFRESH_INTERVAL` | 否 | 15m | 订阅源刷新间隔 |
| `RATE_LIMIT_ENABLED` | 否 | true | 启用速率限制 |
| `RATE_LIMIT_REDIS_URL` | 否 | - | 分布式速率限制的 Redis URL |
| `LOG_LEVEL` | 否 | info | 日志级别（debug、info、warn、error） |
| `ENV` | 否 | development | 环境（development、production） |
| `GITHUB_CLIENT_ID` | 否 | - | GitHub OAuth 客户端 ID |
| `GITHUB_CLIENT_SECRET` | 否 | - | GitHub OAuth 客户端密钥 |
