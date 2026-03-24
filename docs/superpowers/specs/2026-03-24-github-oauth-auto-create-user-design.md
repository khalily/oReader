# GitHub OAuth 自动创建用户 - 设计文档

## 概述

改进 GitHub OAuth 登录流程，增强用户创建体验：
1. **处理无 Email 情况**：调用 GitHub `/user/emails` API 获取邮箱，仍无则让用户输入
2. **自动关联账户**：Email 匹配现有用户时自动关联 GitHub 账户
3. **保存更多信息**：新增 `GitHubLogin` 字段

## 架构

```
GitHub Callback
       │
       ├─ 通过 GitHub ID 找到用户? ────────────────────┐
       │                                                │
       │ 否                                              │ 是
       ▼                                                 ▼
调用 /user/emails 获取邮箱                          直接登录 ✓
       │                                           重定向到 /
       ├─ 有 Email?
       │   ├─ 是 → Email 匹配现有用户?
       │   │   ├─ 是 → 自动关联 GitHub，直接登录 ✓
       │   │   └─ 否 → 创建新用户，直接登录 ✓
       │   │
       │   └─ 否 → 创建 PendingOAuth
       │            重定向到 /oauth/pending?token=xxx
       │                     │
       │                     ▼
       │            前端显示邮箱输入表单
       │                     │
       │                     ▼
       │            POST /oauth/complete
       │            创建新用户，直接登录 ✓
```

## 数据模型

### 修改 User 模型

```go
// User 模型 - 新增 GitHubLogin 字段
type User struct {
    Base
    Email        string `gorm:"uniqueIndex;type:varchar(255);not null" json:"email"`
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
    Token       string    `gorm:"type:varchar(64);not null;uniqueIndex" json:"token"`
    GitHubID    string    `gorm:"type:varchar(100);not null" json:"github_id"`
    GitHubLogin string    `gorm:"type:varchar(100);not null" json:"github_login"`
    Nickname    string    `gorm:"type:varchar(100)" json:"nickname"`
    AvatarURL   string    `gorm:"type:varchar(500)" json:"avatar_url"`
    ExpiresAt   time.Time `gorm:"not null" json:"expires_at"`
}

// IsExpired 判断是否过期
func (p *PendingOAuth) IsExpired() bool {
    return time.Now().After(p.ExpiresAt)
}
```

## API 端点

### 新增端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/v1/auth/oauth/pending` | GET | 根据 token 获取待确认的 OAuth 信息 |
| `/api/v1/auth/oauth/complete` | POST | 完成注册（用户填写邮箱后创建账号） |

### API 详情

#### GET /api/v1/auth/oauth/pending

**请求参数：**
- `token` (query): PendingOAuth token

**响应：**
```json
{
    "github_login": "octocat",
    "nickname": "The Octocat",
    "avatar_url": "https://avatars.githubusercontent.com/u/..."
}
```

**错误响应：**
- `400`: token 缺失
- `404`: token 不存在或已过期

#### POST /api/v1/auth/oauth/complete

**请求体：**
```json
{
    "token": "xxx",
    "email": "user@example.com"
}
```

**成功响应：**
- 设置 auth cookie
- 返回用户信息

**错误响应：**
- `400`: token 无效/过期、email 格式错误
- `409`: email 已被使用

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
│   └─ 未找到 → 检查 email
│       ├─ 有 email 且匹配现有用户 → 自动关联 GitHub，直接登录
│       ├─ 有 email 且不匹配 → 创建新用户，直接登录
│       └─ 无 email → 创建 PendingOAuth，重定向到 /oauth/pending
│
└─ 6. 直接登录：设置 cookie → 重定向到 /
```

### 获取 GitHub Email

```go
// GitHubEmail 表示 /user/emails 返回的邮箱项
type GitHubEmail struct {
    Email    string `json:"email"`
    Primary  bool   `json:"primary"`
    Verified bool   `json:"verified"`
}

// getBestEmail 选择最佳邮箱
// 优先级: primary && verified > verified > primary > 第一个
func getBestEmail(emails []GitHubEmail) string {
    var verified, primary string
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

## 文件清单

### 新增文件

| 文件 | 描述 |
|------|------|
| `internal/repository/pending_oauth_repository.go` | PendingOAuth 数据访问层 |
| `internal/repository/pending_oauth_repository_test.go` | Repository 测试 |

### 修改文件

| 文件 | 变更 |
|------|------|
| `internal/model/models.go` | User 添加 `GitHubLogin` 字段；新增 `PendingOAuth` 模型 |
| `internal/handler/oauth_handler.go` | 添加 `/user/emails` 调用；添加 `/oauth/pending`、`/oauth/complete` 端点 |
| `internal/handler/oauth_handler_test.go` | 更新测试 |
| `internal/service/interfaces.go` | 添加 `PendingOAuthRepository` 接口 |
| `cmd/server/main.go` | 添加新路由 |
| `web/src/pages/oauth/OAuthPendingPage.tsx` | 邮箱输入页面 |
| `web/src/App.tsx` | 添加 `/oauth/pending` 路由 |

## 测试策略

| 层级 | 测试内容 |
|------|----------|
| **单元测试** | `getBestEmail()` 邮箱选择逻辑 |
| **单元测试** | `PendingOAuth.IsExpired()` 过期判断 |
| **集成测试** | Callback 流程：有邮箱/无邮箱/已绑定 GitHub ID |
| **集成测试** | `/oauth/complete`：邮箱验证、重复检测 |
| **组件测试** | 前端邮箱输入表单 |

## 安全考虑

1. **State 参数** - 防止 CSRF 攻击（已实现）
2. **PendingOAuth 过期** - 5 分钟有效期，防止 token 被滥用
3. **自动关联安全性** - GitHub 已验证用户拥有该邮箱，自动关联是安全的
