# 实现任务（TDD + 阶段提交）

> **方法论**：测试驱动开发（TDD）
> - 🔴 先编写失败测试
> - 🟢 编写最小代码使测试通过
> - 🔵 如需要则重构
>
> **Git 策略**：每个完成的阶段后提交

---

## ⚠️ 优先修复清单（来自架构评审）

以下项目被架构师评审标记为关键/高优先级，应在实现过程中解决：

### 关键（必须实现）
| # | 问题 | 阶段 | 状态 |
|---|-------|-------|--------|
| C1 | 添加超出 SameSite cookies 的显式 CSRF 保护 | 阶段 3 | 完成 |
| C2 | 定义 repository/service 接口以实现适当的分层 | 阶段 2 | 完成 |
| C3 | 添加 RSS 订阅源内容的内容清理 | 阶段 5 | 完成 |
| C4 | 阻止订阅源 URL 获取中的 SSRF 攻击 | 阶段 5 | 完成 |
| C5 | 文档化所有环境变量和配置 | 阶段 1 | 完成 |

### 高优先级（应该实现）
| # | 问题 | 阶段 | 状态 |
|---|-------|-------|--------|
| H1 | 使用 golang-migrate 替代 AutoMigrate | 阶段 2 | 完成 |
| H2 | 设计带接口的速率限制器（内存 + Redis） | 阶段 4 | 完成 |
| H3 | 添加结构化日志（zerolog） | 阶段 2 | 完成 |
| H4 | 添加安全头中间件 | 阶段 2 | 完成 |
| H5 | 定义一致的 API 错误响应格式 | 阶段 2 | 完成 |

---

## 阶段 1：项目设置

- [x] 1.1 初始化 Go 模块 (`go mod init oreader`)
- [x] 1.2 创建项目目录结构 (`cmd/`、`internal/`、`web/`、`migrations/`)
- [x] 1.3 创建 Makefile，包含 `test`、`build`、`run`、`migrate` 命令
- [x] 1.4 添加 Go 测试依赖 (`testify`、`mockery`)
- [x] 1.5 配置测试覆盖率报告
- [x] 1.6 使用 Vite + TypeScript 初始化 React 前端
- [x] 1.7 添加前端依赖
- [x] 1.8 配置 Tailwind CSS 和 shadcn/ui
- [x] 1.9 创建包含所有已文档化环境变量的 `.env.example` **[C5]**
- [x] 1.10 在 CI 配置中添加 `govulncheck`
- [x] 1.11 在 CI 配置中添加 `npm audit`

**Git 提交**: `git commit -m "feat: project setup with Go + React structure"`

---

## 阶段 2：后端核心基础设施

### 🔴 编写测试
- [x] 2.1 编写配置加载测试
- [x] 2.2 编写数据库连接测试
- [x] 2.3 编写数据模型验证测试

### 🟢 实现
- [x] 2.4 使用 viper 实现配置加载
- [x] 2.5 定义包含所有必需字段的配置结构体
- [x] 2.6 使用 GORM 创建数据库连接
- [x] 2.7 定义支持多租户的数据模型
- [x] 2.8 **[H1]** 创建 golang-migrate 迁移文件（非 AutoMigrate）
- [x] 2.9 **[C2]** 在 `internal/service/interfaces.go` 中定义 repository 接口
- [x] 2.10 **[H3]** 初始化 zerolog，支持基于环境的格式化
- [x] 2.11 **[H4]** 实现安全头中间件
- [x] 2.12 **[H5]** 实现带标准格式的错误响应辅助函数
- [x] 2.13 创建带路由组的 Gin 路由器
- [x] 2.14 实现 CORS 中间件
- [x] 2.15 实现带 request_id 的请求日志中间件
- [x] 2.16 创建 main.go 入口点

### 🔵 验证和重构
- [x] 2.17 运行所有测试：`make test`
- [x] 2.18 确保 config 和 models 覆盖率 >80%
- [x] 2.19 验证迁移使用 `make migrate-up` 成功运行
- [x] 2.20 **验证多用户场景**：两个用户订阅同一 RSS 源，各自标记阅读/收藏状态互不影响

**Git 提交**: `git commit -m "feat(backend): core infrastructure with models, interfaces, and logging"`

---

## 阶段 3：认证系统

### 🔴 编写测试
- [x] 3.1 编写 JWT 令牌生成/验证测试
- [x] 3.2 编写刷新令牌 CRUD 测试
- [x] 3.3 编写密码哈希测试（bcrypt cost 12）
- [x] 3.4 编写 CSRF 令牌生成和验证测试 **[C1]**
- [x] 3.5 编写认证服务测试（register、login、logout、refresh）
- [x] 3.6 编写认证处理程序端点测试
- [x] 3.7 编写认证中间件测试
- [x] 3.8 编写 CSRF 中间件测试 **[C1]**

### 🟢 实现
- [x] 3.9 实现 JWT 服务（`internal/infra/jwt/`），使用 HS256 和 256-bit 密钥
- [x] 3.10 实现刷新令牌生成和哈希
- [x] 3.11 **[C1]** 实现 CSRF 令牌生成和验证
- [x] 3.12 实现 cookie 工具（`internal/infra/cookie/`），使用 HttpOnly + SameSite=Strict
- [x] 3.13 使用 bcrypt（cost 12）实现密码哈希
- [x] 3.14 实现 User repository（实现接口）
- [x] 3.15 实现 RefreshToken repository（实现接口）
- [x] 3.16 实现认证服务
- [x] 3.17 实现认证处理程序（register、login、logout、refresh、me）
- [x] 3.18 实现 JWT 认证中间件
- [x] 3.19 **[C1]** 为状态变更请求实现 CSRF 中间件
- [x] 3.20 添加认证事件日志（login、logout、refresh）

### 🔵 验证和重构
- [x] 3.21 运行所有测试：`make test`
- [x] 3.22 确保 auth 包覆盖率 >80%
- [x] 3.23 手动测试认证流程，包括 CSRF
- [x] 3.24 验证 TOKEN_EXPIRED 错误码正确返回

**Git 提交**: `git commit -m "feat(auth): dual-token authentication with HttpOnly cookies and CSRF protection"`

---

## 阶段 4-18（摘要）

其余阶段已完成并提交。完整任务列表见原始文件。

---

## 总结

| 阶段 | 描述 | TDD | 关键修复 | 高优先级修复 | 提交 |
|-------|-------------|-----|----------------|------------|--------|
| 1 | 项目设置 | - | C5 | - | ✅ |
| 2 | 后端核心 | ✅ | C2 | H1, H3, H4, H5 | ✅ |
| 3 | 认证 | ✅ | C1 | - | ✅ |
| 4 | 速率限制 | ✅ | - | H2 | ✅ |
| 5 | RSS/Feeds | ✅ | C3, C4 | - | ✅ |
| 6 | 文章 | ✅ | - | - | ✅ |
| 7 | 后台 Worker | ✅ | - | - | ✅ |
| 8 | OPML 导入/导出 | ✅ | - | - | ✅ |
| 9 | OAuth（预留） | ✅ | - | - | ✅ |
| 10 | 前端核心 | - | C1 | - | ✅ |
| 11-14 | 前端功能 | - | - | - | ✅ |
| 15 | 静态嵌入 | - | - | - | ✅ |
| 16 | Docker 部署 | - | - | - | ✅ |
| 17 | 测试和文档 | - | - | - | ✅ |
| 18 | 最终验证 | - | - | - | ✅ |

**总计：18 个阶段，18 次 Git 提交**

**关键修复：C1-CSRF、C2-接口、C3-清理、C4-SSRF、C5-配置**
**高优先级修复：H1-迁移、H2-速率限制器接口、H3-日志、H4-安全头、H5-错误格式**
