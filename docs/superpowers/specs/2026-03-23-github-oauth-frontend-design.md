# GitHub OAuth 登录 - 前端设计

## 概述

为 oReader 添加 GitHub OAuth 登录入口。后端 OAuth 流程已完整实现，本次仅需要添加前端 UI 入口。

## 需求

- 在登录页面和注册页面添加 "Sign in with GitHub" 按钮
- 按钮采用 GitHub 官方风格（黑色背景 + GitHub 图标）
- 按钮位于表单上方，邮箱/密码登录作为备选
- 支持错误提示（用户取消、state 过期等）

## 架构

```
┌──────────────────────────────────────────────────────────────────┐
│                        Frontend (React)                          │
├──────────────────────────────────────────────────────────────────┤
│  LoginPage.tsx          RegisterPage.tsx                         │
│  ┌─────────────────┐    ┌─────────────────┐                     │
│  │ [GitHub Button] │    │ [GitHub Button] │                     │
│  │ ─────────────── │    │ ─────────────── │                     │
│  │ Email/Password  │    │ Email/Password  │                     │
│  └─────────────────┘    └─────────────────┘                     │
│           │                      │                               │
│           ▼                      ▼                               │
│     window.location.href = '/api/v1/auth/github'                 │
└──────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│                     Backend (已实现)                              │
├──────────────────────────────────────────────────────────────────┤
│  GET /api/v1/auth/github          → 重定向到 GitHub 授权页        │
│  GET /api/v1/auth/github/callback → 处理回调，设置 cookie，       │
│                                     重定向到 /                    │
└──────────────────────────────────────────────────────────────────┘
```

**认证流程：**
1. 用户点击 GitHub 按钮 → 跳转到 `/api/v1/auth/github`
2. 后端生成 state 参数，重定向到 GitHub 授权页
3. 用户授权后，GitHub 回调到 `/api/v1/auth/github/callback`
4. 后端验证 state，交换 code 获取用户信息
5. 后端查找/创建用户，设置认证 cookie
6. 重定向到 `/`
7. 前端 App 初始化时调用 `/auth/me` 获取用户信息

## 组件设计

### 新增组件

**文件：** `web/src/components/auth/SocialLoginButton.tsx`

```tsx
interface SocialLoginButtonProps {
  provider: 'github'  // 预留扩展其他提供商
  className?: string
}
```

**视觉规范：**
- 背景色：`#24292F`（GitHub 官方黑）
- 文字：白色，"Sign in with GitHub"
- 图标：GitHub Octocat SVG（内置，无外部依赖）
- 尺寸：与现有 Button 组件一致（h-10）
- 圆角：与 Card 组件一致（rounded-md）

**行为：**
- 点击后设置 `window.location.href = '/api/v1/auth/github'`
- 无需 loading 状态（页面立即跳转）

### 修改现有文件

| 文件 | 变更 |
|------|------|
| `web/src/pages/auth/LoginPage.tsx` | 在表单上方添加 `<SocialLoginButton provider="github" />` + 分隔线 |
| `web/src/pages/auth/RegisterPage.tsx` | 同上 |

### 页面布局

```
┌─────────────────────────────────┐
│  Sign In                        │
│  Enter your credentials...      │
├─────────────────────────────────┤
│  [🐙 Sign in with GitHub]       │  ← 新增
│                                 │
│  ─────── or continue with ──────│  ← 新增分隔线
│                                 │
│  Email                          │
│  [_______________]              │
│  Password                       │
│  [_______________]              │
│                                 │
│  [Sign In]                      │
└─────────────────────────────────┘
```

## 错误处理

### OAuth 错误场景

| 场景 | 后端行为 | 前端处理 |
|------|----------|----------|
| 用户拒绝授权 | 重定向到 `/login?oauth_error=access_denied` | 显示 "GitHub 登录已取消" |
| State 过期/无效 | 重定向到 `/login?oauth_error=invalid_state` | 显示 "登录已过期，请重试" |
| GitHub 服务异常 | 重定向到 `/login?oauth_error=github_error` | 显示 "GitHub 服务暂时不可用" |

### 前端实现

在 `LoginPage.tsx` 中读取 URL 参数并显示错误：

```tsx
const searchParams = new URLSearchParams(location.search)
const oauthError = searchParams.get('oauth_error')

// 错误消息映射
const errorMessages: Record<string, string> = {
  access_denied: 'GitHub 登录已取消',
  invalid_state: '登录已过期，请重试',
  github_error: 'GitHub 服务暂时不可用',
}
```

### 后端修改

修改 `oauth_handler.go` 中的 `GitHubCallback` 方法，将错误响应改为重定向：

```go
// 用户拒绝授权
if errorCode := c.Query("error"); errorCode != "" {
    c.Redirect(http.StatusFound, "/login?oauth_error=access_denied")
    return
}

// State 无效
if oauthState == nil || err != nil {
    c.Redirect(http.StatusFound, "/login?oauth_error=invalid_state")
    return
}
```

## 测试策略

| 层级 | 测试内容 | 方式 |
|------|----------|------|
| **组件测试** | SocialLoginButton 渲染、点击跳转 | Vitest + Testing Library |
| **页面测试** | LoginPage/RegisterPage 包含按钮、错误提示显示 | Vitest + Testing Library |
| **集成测试** | OAuth 错误参数解析和消息显示 | Vitest + MemoryRouter |

**测试覆盖重点：**
1. `SocialLoginButton` 点击后 `window.location.href` 被正确设置
2. 页面正确渲染 GitHub 按钮
3. URL 带有 `oauth_error` 参数时显示对应错误消息

**不需要测试的部分：**
- 后端 OAuth 流程（已有完整测试覆盖）

## 文件清单

### 新增文件
- `web/src/components/auth/SocialLoginButton.tsx` - GitHub 登录按钮组件
- `web/src/components/auth/__tests__/SocialLoginButton.test.tsx` - 组件测试

### 修改文件
- `web/src/pages/auth/LoginPage.tsx` - 添加按钮和错误处理
- `web/src/pages/auth/RegisterPage.tsx` - 添加按钮
- `web/src/pages/auth/__tests__/LoginPage.test.tsx` - 更新测试
- `web/src/pages/auth/__tests__/RegisterPage.test.tsx` - 更新测试
- `internal/handler/oauth_handler.go` - 错误时重定向而非返回 JSON

## 安全考虑

1. **State 参数** - 后端已实现，防止 CSRF 攻击
2. **HttpOnly Cookie** - 后端已实现，防止 XSS 窃取 token
3. **账户关联** - 后端已实现，通过 email 自动关联现有账户

## 扩展性

设计预留了扩展其他 OAuth 提供商的空间：
- `SocialLoginButton` 的 `provider` 属性支持添加 'google'、'apple' 等
- 后端已有 `GoogleInitiate`、`AppleInitiate` 占位实现
