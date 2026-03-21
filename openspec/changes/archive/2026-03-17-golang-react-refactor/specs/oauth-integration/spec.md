# OAuth 集成规格

## 新增需求

### 需求：GitHub OAuth 登录发起
系统应提供端点以发起 GitHub OAuth 流程。

#### 场景：发起 GitHub OAuth
- **当** 用户调用 GET /api/v1/auth/github
- **则** 系统生成安全的 state 参数
- **且** 系统将 state 存储在 session 或 cookie 中
- **且** 系统重定向到 GitHub 授权 URL，带有 client_id、redirect_uri、scope、state

### 需求：GitHub OAuth 回调处理
系统应处理 GitHub OAuth 回调并创建/关联用户账户。

#### 场景：GitHub OAuth 成功
- **当** GitHub 重定向到 GET /api/v1/auth/github/callback，带有有效的 code 和 state
- **则** 系统验证 state 与存储值匹配
- **且** 系统用 code 交换 GitHub 访问令牌
- **且** 系统获取 GitHub 用户资料
- **且** 系统通过 github_id 查找或创建本地用户
- **且** 系统设置 access_token 和 refresh_token cookies
- **且** 系统重定向到前端并携带用户资料

#### 场景：无效的 state 参数
- **当** 回调的 state 与存储值不匹配
- **则** 系统返回 400 Bad Request
- **且** 系统不继续 OAuth 流程

#### 场景：GitHub API 错误
- **当** GitHub 返回错误或无法响应
- **则** 系统返回适当的错误
- **且** 系统重定向到前端并携带错误消息

### 需求：OAuth 用户账户创建
系统应为新的 OAuth 用户创建账户。

#### 场景：新 GitHub 用户
- **当** GitHub OAuth 对无现有账户的用户成功
- **则** 系统创建新 User，包含：
  - id：UUID
  - email：GitHub 邮箱（如可用且公开）
  - nickname：GitHub login 或 name
  - avatar_url：GitHub 头像 URL
  - auth_provider："github"
  - github_id：GitHub 用户 ID
  - password_hash：null

#### 场景：相同邮箱的现有用户
- **当** GitHub OAuth 返回的邮箱与现有本地账户匹配
- **则** 系统将 GitHub 账户关联到现有用户
- **且** 系统在现有用户上设置 github_id
- **且** 系统不创建重复账户

### 需求：OAuth 用户资料映射
系统应将 GitHub 资料字段映射到本地用户字段。

#### 场景：资料字段映射
- **当** 用户通过 GitHub 认证
- **则** 系统映射：
  - GitHub id → github_id
  - GitHub login → nickname（如 name 不可用）
  - GitHub name → nickname（优先）
  - GitHub avatar_url → avatar_url
  - GitHub email → email（如公开）

### 需求：为未来提供商预留 OAuth 端点
系统应为额外的 OAuth 提供商预留端点模式。

#### 场景：未来提供商端点
- **当** 系统部署
- **则** 以下端点模式被预留：
  - GET /api/v1/auth/google
  - GET /api/v1/auth/google/callback
- **且** 端点在实现前返回 501 Not Implemented
