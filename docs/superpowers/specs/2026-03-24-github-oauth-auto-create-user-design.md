# GitHub OAuth 自动创建用户 - 设计文档

## 概述

改进 GitHub OAuth 登录流程，增强用户创建体验：
1. **处理无 Email 情况**：调用 GitHub `/user/emails` API 获取邮箱，仍无则让用户输入
2. **安全的账户关联**：Email 匹配时要求用户验证原账户后确认关联
3. **保存更多信息**：新增 `GitHubLogin` 字段

## 需求

| 需求 | 描述 |
|------|------|
| A. 无 Email 处理 | 调用 `/user/emails` API 获取邮箱；仍无则让用户输入 |
| B. 账户关联策略 | Email 匹配时跳转确认页，用户验证原账户密码后确认关联 |
| C. 额外字段 | User 表添加 `GitHubLogin` 字段（GitHub 用户名，用于展示和未来功能） |

## 架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                        OAuth 两阶段流程                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Phase 1: GitHub 回调                                                │
│  ┌─────────────┐    ┌──────────────┐    ┌──────────────────────┐   │
│  │   GitHub    │───▶│  Callback    │───▶│  生成 PendingToken   │   │
│  │  Callback   │    │  Handler     │    │  (5分钟有效)         │   │
│  └─────────────┘    └──────────────┘    └──────────────────────┘   │
│                              │                                       │
│                              ▼                                       │
│                    ┌──────────────────────┐                         │
│                    │  重定向到前端确认页   │                         │
│                    │  /oauth/pending      │                         │
│                    │  ?token=xxx          │                         │
│                    └──────────────────────┘                         │
│                                                                      │
│  Phase 2: 用户确认/完成                                              │
│  ┌─────────────────────┐                                             │
│  │  前端确认页          │  显示 GitHub 信息，用户决策：               │
│  │  /oauth/pending     │  • 新用户：确认创建（可能需填写邮箱）        │
│  │                     │  • 需关联：验证原账户密码后确认              │
│  └─────────────────────┘                                             │
│           │                                                          │
│           ▼                                                          │
│  ┌─────────────────────┐    ┌──────────────────────────────────┐   │
│  │  POST /api/v1/auth/ │───▶│  验证 token → 创建/关联用户      │   │
│  │  oauth/complete     │    │  → 设置 cookie → 返回成功        │   │
│  │  或 oauth/link      │    │                                  │   │
│  └─────────────────────┘    └──────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 认证流程

```
用户点击 GitHub 登录
       │
       ▼
GET /api/v1/auth/github (发起 OAuth)
       │
       ▼
GitHub 授权页面
       │
       ▼
GET /api/v1/auth/github/callback
       │
       ├─ 已绑定 GitHub ID? ──────────────────────┐
       │                                          │
       │ 否                                        │ 是
       ▼                                          ▼
调用 /user/emails 获取邮箱                   直接登录成功
       │                                      重定向到 /
       ├─ Email 匹配现有用户?
       │   ├─ 是 → 创建 PendingOAuth (action=link)
       │   └─ 否 → 创建 PendingOAuth (action=create)
       │
       ▼
重定向到 /oauth/pending?token=xxx
       │
       ▼
前端显示确认页
       │
       ├─ action=create → 用户确认/填写邮箱 → POST /oauth/complete
       │                                     │
       └─ action=link   → 用户验证密码 → POST /oauth/link
                                             │
                                             ▼
                                       创建/关联用户
                                       设置 cookie
                                             │
                                             ▼
                                       重定向到 /
```

## 数据模型

### 修改 User 模型

```go
// User 模型 - 新增 GitHubLogin 字段
type User struct {
    Base
    Email        string `gorm:"uniqueIndex;type:varchar(255);not null" json:"email"` // 保持 not null
    PasswordHash string `gorm:"type:varchar(255)" json:"-"`
    Nickname     string `gorm:"type:varchar(100)" json:"nickname"`
    AvatarURL    string `gorm:"type:varchar(500)" json:"avatar_url"`
    AuthProvider string `gorm:"type:varchar(50);default:'email'" json:"auth_provider"`
    GitHubID     string `gorm:"type:varchar(100)" json:"github_id,omitempty"`
    GitHubLogin  string `gorm:"type:varchar(100)" json:"github_login,omitempty"` // 新增
}
```

### 新增 PendingOAuth 模型

```go
// PendingOAuth 存储待确认的 OAuth 数据（5分钟过期）
type PendingOAuth struct {
    Base
    Token         string    `gorm:"type:varchar(64);not null;uniqueIndex" json:"token"`
    GitHubID      string    `gorm:"type:varchar(100);not null" json:"github_id"`
    GitHubLogin   string    `gorm:"type:varchar(100);not null" json:"github_login"`
    Email         string    `gorm:"type:varchar(255)" json:"email"`                 // 可空，GitHub 可能无邮箱
    Nickname      string    `gorm:"type:varchar(100)" json:"nickname"`
    AvatarURL     string    `gorm:"type:varchar(500)" json:"avatar_url"`
    Action        string    `gorm:"type:varchar(20);not null" json:"action"`        // "create" | "link"
    ExistingEmail string    `gorm:"type:varchar(255)" json:"existing_email"`        // 仅 link 时使用
    ExpiresAt     time.Time `gorm:"not null" json:"expires_at"`
}

// IsExpired 判断是否过期
func (p *PendingOAuth) IsExpired() bool {
    return time.Now().After(p.ExpiresAt)
}
```

### 数据库迁移

```sql
-- 添加 user 表字段
ALTER TABLE users ADD COLUMN github_login VARCHAR(100);

-- 创建 pending_oauths 表
CREATE TABLE pending_oauths (
    id VARCHAR(36) PRIMARY KEY,
    token VARCHAR(64) NOT NULL UNIQUE,
    github_id VARCHAR(100) NOT NULL,
    github_login VARCHAR(100) NOT NULL,
    email VARCHAR(255),
    nickname VARCHAR(100),
    avatar_url VARCHAR(500),
    action VARCHAR(20) NOT NULL,
    existing_email VARCHAR(255),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_pending_oauths_token (token),
    INDEX idx_pending_oauths_deleted_at (deleted_at)
);
```

## API 端点

### 修改现有端点

| 端点 | 当前行为 | 新行为 |
|------|----------|--------|
| `GET /api/v1/auth/github/callback` | 直接创建/关联用户 → 重定向到 `/` | 创建 PendingOAuth → 重定向到 `/oauth/pending?token=xxx` |

### 新增端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/v1/auth/oauth/pending` | GET | 根据 token 获取待确认的 OAuth 信息 |
| `/api/v1/auth/oauth/complete` | POST | 完成注册（新用户确认创建） |
| `/api/v1/auth/oauth/link` | POST | 关联账户（验证原账户后确认） |

### API 详情

#### GET /api/v1/auth/oauth/pending

> **注意：** 此 API 无需认证（用户尚未登录）。

**请求参数：**
- `token` (query): PendingOAuth token

**响应：**
```json
{
    "action": "create",
    "github_login": "octocat",
    "email": "user@example.com",
    "has_email": true,
    "nickname": "The Octocat",
    "avatar_url": "https://avatars.githubusercontent.com/u/..."
}
```

对于 `action=link` 的情况，额外返回：
```json
{
    "action": "link",
    "existing_email": "existing@example.com",
    ...
}
```

**错误响应：**
- `400`: token 缺失
- `404`: token 不存在或已过期

#### POST /api/v1/auth/oauth/complete

> **注意：** 此 API 无需认证（用户尚未登录）。

**请求体：**
```json
{
    "token": "xxx",
    "email": "user@example.com"   // 如果 GitHub 无邮箱，必须填写；否则可选
}
```

**Email 校验规则：**
- 格式：符合标准邮箱格式
- 唯一性：不能与现有用户邮箱重复

**成功响应：**
- 设置 auth cookie
- 返回用户信息

**错误响应：**
- `400`: token 无效/过期、email 格式错误
- `409`: email 已被使用

#### POST /api/v1/auth/oauth/link

> **注意：** 此 API 无需认证（用户尚未登录）。后端从 `PendingOAuth.ExistingEmail` 获取要关联的邮箱，**忽略请求中的 `email` 字段**，防止账户劫持攻击。

**请求体：**
```json
{
    "token": "xxx",
    "password": "xxx"
}
```

**后端处理逻辑：**
1. 验证 token 有效且未过期
2. 从 `PendingOAuth.ExistingEmail` 获取目标邮箱
3. 验证该邮箱对应的用户密码
4. 关联 GitHub 账户

**成功响应：**
- 设置 auth cookie
- 返回用户信息（已关联 GitHub）

**错误响应：**
- `400`: token 无效/过期
- `401`: 密码错误

## 后端处理流程

### GitHub Callback 改造

```
GitHubCallback 流程：
│
├─ 1. 验证 state 参数
├─ 2. 交换 code 获取 access_token
├─ 3. 调用 /user API 获取 GitHub 用户信息
├─ 4. 如果 user.email 为空，调用 /user/emails API 获取
│       └─ 选择优先级: primary+verified > verified > primary > 第一个
│
├─ 5. 查找逻辑：
│   ├─ 通过 GitHub ID 找到 → 直接登录（更新 GitHubLogin）
│   │
│   └─ 未找到 → 检查 email 是否匹配现有用户
│       ├─ 匹配 → 创建 PendingOAuth (action=link)
│       └─ 不匹配 → 创建 PendingOAuth (action=create)
│
├─ 6. 生成临时 token (UUID v7)
├─ 7. 存储 PendingOAuth (5分钟过期)
└─ 8. 重定向到 /oauth/pending?token=xxx
```

### 获取 GitHub Email

```go
// GitHubEmail 表示 /user/emails 返回的邮箱项
type GitHubEmail struct {
    Email    string `json:"email"`
    Primary  bool   `json:"primary"`
    Verified bool   `json:"verified"`
}

// getGitHubEmails 获取用户邮箱列表
func (h *OAuthHandler) getGitHubEmails(ctx context.Context, accessToken string) ([]GitHubEmail, error) {
    url := fmt.Sprintf("%s/user/emails", h.githubAPIURL)
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
    req.Header.Set("Accept", "application/json")
    // ...
}

// getBestEmail 选择最佳邮箱
// 优先级: primary && verified > verified > primary > 第一个
func getBestEmail(emails []GitHubEmail) string {
    var primaryVerified, verified, primary string
    for _, e := range emails {
        if e.Primary && e.Verified {
            return e.Email  // 最佳选择
        }
        if e.Verified && verified == "" {
            verified = e.Email
        }
        if e.Primary && primary == "" {
            primary = e.Email
        }
    }
    if verified != "" {
        return verified
    }
    if primary != "" {
        return primary
    }
    if len(emails) > 0 {
        return emails[0].Email
    }
    return ""
}
```

## 前端设计

### 新增页面

**路由：** `/oauth/pending?token=xxx`

**文件：** `web/src/pages/oauth/OAuthPendingPage.tsx`

### 页面布局

#### 场景 A1: 新用户 + GitHub 有邮箱

```
┌─────────────────────────────────────────────────────────┐
│  [GitHub 头像]   欢迎来到 oReader！                       │
│                                                          │
│  GitHub 账户: @octocat                                   │
│  昵称: The Octocat                                       │
│  邮箱: octocat@example.com ✓ (来自 GitHub)              │
│                                                          │
│                 [ 完成注册 ]                              │
└─────────────────────────────────────────────────────────┘
```

#### 场景 A2: 新用户 + GitHub 无邮箱

```
┌─────────────────────────────────────────────────────────┐
│  [GitHub 头像]   欢迎来到 oReader！                       │
│                                                          │
│  GitHub 账户: @octocat                                   │
│  昵称: The Octocat                                       │
│                                                          │
│  请提供您的邮箱地址：*                                    │
│  ┌───────────────────────────────────────────────────┐  │
│  │                                                   │  │
│  └───────────────────────────────────────────────────┘  │
│                                                          │
│                 [ 完成注册 ]                              │
└─────────────────────────────────────────────────────────┘
```

#### 场景 B: 账户关联

```
┌─────────────────────────────────────────────────────────┐
│  [GitHub 头像]   关联现有账户                             │
│                                                          │
│  该 GitHub 账户的邮箱已关联到现有账户                     │
│                                                          │
│  GitHub: @octocat                                        │
│  关联邮箱: existing@example.com                          │
│                                                          │
│  请输入原账户密码以完成关联：                             │
│  ┌───────────────────────────────────────────────────┐  │
│  │ 密码                                               │  │
│  └───────────────────────────────────────────────────┘  │
│                                                          │
│                 [ 确认关联 ]                              │
└─────────────────────────────────────────────────────────┘
```

> **安全说明**：邮箱从后端 `PendingOAuth.ExistingEmail` 获取，前端只收集密码验证，防止邮箱篡改攻击。

### 组件结构

```
web/src/pages/oauth/
├── OAuthPendingPage.tsx      # 主页面，根据 action 渲染不同表单
├── OAuthCreateForm.tsx       # 新用户注册表单
├── OAuthLinkForm.tsx         # 账户关联表单
└── __tests__/
    ├── OAuthPendingPage.test.tsx
    ├── OAuthCreateForm.test.tsx
    └── OAuthLinkForm.test.tsx
```

### API 调用

```typescript
// web/src/api/oauth.ts
export const oauthApi = {
  getPending: (token: string) =>
    api.get(`/api/v1/auth/oauth/pending?token=${token}`),

  complete: (data: { token: string; email?: string }) =>
    api.post('/api/v1/auth/oauth/complete', data),

  // link 不需要传 email，后端从 PendingOAuth 获取
  link: (data: { token: string; password: string }) =>
    api.post('/api/v1/auth/oauth/link', data),
};
```

## 文件清单

### 新增文件

| 文件 | 描述 |
|------|------|
| `internal/model/pending_oauth.go` | PendingOAuth 模型 |
| `internal/repository/pending_oauth_repository.go` | PendingOAuth 数据访问层 |
| `internal/repository/pending_oauth_repository_test.go` | Repository 测试 |
| `internal/handler/oauth_complete_handler.go` | 完成注册/关联的 handler |
| `internal/handler/oauth_complete_handler_test.go` | Handler 测试 |
| `web/src/pages/oauth/OAuthPendingPage.tsx` | 确认页面 |
| `web/src/pages/oauth/OAuthCreateForm.tsx` | 注册表单组件 |
| `web/src/pages/oauth/OAuthLinkForm.tsx` | 关联表单组件 |
| `web/src/pages/oauth/__tests__/*.test.tsx` | 组件测试 |
| `web/src/api/oauth.ts` | OAuth API 调用 |

### 修改文件

| 文件 | 变更 |
|------|------|
| `internal/model/models.go` | User 添加 `GitHubLogin` 字段 |
| `internal/handler/oauth_handler.go` | 改造为两阶段流程，添加获取 emails 逻辑 |
| `internal/service/interfaces.go` | 添加 `PendingOAuthRepository` 接口 |
| `internal/handler/oauth_handler_test.go` | 更新测试 |
| `web/src/router/index.tsx` | 添加 `/oauth/pending` 路由 |

## 测试策略

| 层级 | 测试内容 | 方式 |
|------|----------|------|
| **单元测试** | `getBestEmail()` 邮箱选择逻辑 | Go test |
| **单元测试** | `PendingOAuth.IsExpired()` 过期判断 | Go test |
| **集成测试** | Callback 流程：新用户 / 需关联 / 已有 GitHub ID | httptest |
| **集成测试** | `/oauth/complete`：邮箱验证、重复检测 | httptest |
| **集成测试** | `/oauth/link`：密码验证、关联成功 | httptest |
| **组件测试** | 前端表单：有/无邮箱两种场景 | Vitest + Testing Library |

### 关键测试场景

1. **Callback 测试**
   - GitHub 返回 email → 正常流程
   - GitHub 无 email，/user/emails 有 → 使用 emails
   - GitHub 完全无 email → action=create，has_email=false
   - Email 匹配现有用户 → action=link

2. **Complete 测试**
   - 有 email → 直接创建用户
   - 无 email + 用户填写 → 创建用户
   - Email 已存在 → 返回 409

3. **Link 测试**
   - 正确密码 → 关联成功
   - 错误密码 → 返回 401

## 安全考虑

1. **State 参数** - 已实现，防止 CSRF 攻击
2. **PendingOAuth 过期** - 5 分钟有效期，防止 token 被滥用
3. **关联验证** - 必须验证原账户密码；后端从 `PendingOAuth.ExistingEmail` 获取邮箱，**不信任用户输入**，防止账户劫持
4. **HttpOnly Cookie** - 防止 XSS 窃取 token
5. **Email 唯一性** - 数据库约束保证，防止重复注册
6. **Token 机密性** - URL 中的 token 不应被分享，前端应清理 URL 历史

## 数据清理策略

### PendingOAuth 过期清理

| 策略 | 实现方式 |
|------|----------|
| **懒清理** | 查询时检查 `expires_at`，过期则返回 404（不自动删除） |
| **定时清理** | 可选：每日定时任务删除 `expires_at < NOW()` 的记录 |
| **用户触发清理** | 使用后立即删除（无论成功或失败） |

**推荐实现**：懒清理 + 使用后立即删除，无需额外定时任务。

## 扩展性

设计支持未来添加其他 OAuth 提供商：
- `PendingOAuth.Provider` 可扩展为 "google"、"apple" 等
- `Action` 枚举可扩展
- 后端已有 `GoogleInitiate`、`AppleInitiate` 占位实现
