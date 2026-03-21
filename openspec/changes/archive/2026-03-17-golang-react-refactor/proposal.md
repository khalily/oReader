## 背景

当前的 oReader RSS 阅读器使用 Flask（后端）和 AngularJS 1.x（前端）构建，这两者都是过时的技术。AngularJS 1.x 已停止维护，且团队更精通 Go。此外，项目未来计划添加更多功能，需要更现代、更易维护的架构。

## 变更内容

**破坏性变更**：完全重写应用技术栈。

### 后端
- 用 Go + Gin 框架替换 Flask
- 用 GORM 替换 SQLAlchemy
- 用 gofeed 库替换自定义 lxml RSS 解析器
- 用 golang-jwt 替换 itsdangerous 令牌
- 添加双令牌认证（访问令牌 + 刷新令牌）
- 添加后台 RSS 刷新 worker（goroutine）

### 前端
- 用 React 18 + TypeScript 替换 AngularJS 1.x
- 用 React Router 替换 angular-route
- 用 React Query + Axios 替换 $resource
- 用 Zustand 替换 angular-store
- 添加 shadcn/ui + Tailwind CSS 作为 UI 组件
- 使用 Vite 作为构建工具

### 安全增强
- 用 UUID v7 替换自增用户 ID
- 实现双令牌机制（15分钟访问 + 7天刷新）
- 将令牌存储在 HttpOnly + SameSite=Strict cookies 中
- 将刷新令牌存储在数据库中以支持撤销
- 添加显式 CSRF 令牌保护（纵深防御）
- 使用 bluemonday 清理所有 RSS 内容（XSS 防护）
- 通过验证订阅源 URL 阻止 SSRF 攻击（私有 IP 阻止）
- 添加安全头中间件（X-Frame-Options、CSP 等）
- 实现基于接口的速率限制（内存 + Redis 就绪）

### 部署
- 单一二进制文件部署（前端嵌入 Go 二进制）
- Docker 容器化
- 开发环境使用 SQLite，生产环境使用 MySQL 8.0

### 功能
- 带 JWT 的多用户注册/登录
- RSS 订阅管理（添加/删除/列表）
- 文章阅读（列表/详情/收藏/已读）
- 定时自动刷新 + 手动刷新
- 预留 GitHub OAuth 登录接口
- API 速率限制防止滥用
- OPML 导入/导出订阅源

### 开发方法论
- 测试驱动开发（TDD）- 先写测试，再实现
- 每个完成阶段后 Git 提交

## 能力

### 新增能力

- `user-auth`：用户注册、登录、登出和 JWT 令牌管理，采用双令牌机制
- `rss-subscription`：RSS 订阅源订阅管理（添加、删除、列出订阅源）
- `rss-parsing`：使用 gofeed 库解析 RSS/Atom 订阅源
- `article-management`：文章列表、阅读、收藏和已读状态管理
- `background-refresh`：自动刷新 RSS 订阅源的后台 worker
- `oauth-integration`：第三方 OAuth 登录支持（GitHub OAuth 预留）
- `rate-limiting`：API 速率限制以防止滥用和保护资源
- `feed-import-export`：订阅源的 OPML 导入和导出
- `security-hardening`：CSRF 保护、内容清理、SSRF 防护、安全头
- `logging-strategy`：带请求上下文和审计事件的结构化 JSON 日志
- `error-format`：带错误码和详情的一致 API 错误响应
- `data-model`：多租户订阅源共享，支持按用户文章状态（UserFeed、UserItemState）

### 修改的能力

无 - 这是完全重写，不修改现有能力。

## 影响

### 代码库
- 使用 Go 和 TypeScript 完全重写
- 遵循 Go 约定的新项目结构（cmd/、internal/、pkg/）
- 采用 React 组件架构的新前端结构

### API
- 所有 API 端点保持 RESTful 但使用新实现
- 新端点：`/api/v1/auth/refresh`、`/api/v1/auth/github`、`/api/v1/auth/github/callback`
- 基于 cookie 的认证替换 Authorization 头

### 依赖
- 后端：Go 1.21+、Gin、GORM、golang-jwt、gofeed、viper、bcrypt
- 前端：React 18、TypeScript、Vite、React Query、Zustand、shadcn/ui、Tailwind CSS
- 数据库：SQLite（开发）/ MySQL 8.0（生产）
- 部署：Docker、单一二进制

### 数据迁移
- 无数据迁移计划 - 使用新数据库模式全新开始
- 新模式使用 UUID 主键而非自增整数
