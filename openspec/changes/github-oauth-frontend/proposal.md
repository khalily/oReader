## Why

oReader 目前仅支持邮箱/密码登录方式，用户需要记住额外的一套凭证。添加 GitHub OAuth 登录可以：
- 降低用户注册门槛，一键授权即可使用
- 提供更安全、更便捷的认证方式
- 减少密码管理负担

后端 GitHub OAuth 流程已完整实现，本次仅需要添加前端 UI 入口。

## What Changes

- 在登录页面添加 "Sign in with GitHub" 按钮
- 在注册页面添加 "Sign in with GitHub" 按钮
- 添加 OAuth 错误处理（用户取消、state 过期等）
- 修改后端错误响应为重定向（而非返回 JSON）

## Capabilities

### New Capabilities

- `social-auth`: 社交账号登录能力，支持 GitHub OAuth 认证入口

### Modified Capabilities

无。现有 `auth` 能力的实现细节不变，仅添加新的入口。

## Impact

**前端文件：**
- `web/src/components/auth/SocialLoginButton.tsx` - 新增
- `web/src/pages/auth/LoginPage.tsx` - 修改
- `web/src/pages/auth/RegisterPage.tsx` - 修改

**后端文件：**
- `internal/handler/oauth_handler.go` - 修改错误响应为重定向

**依赖：**
- 后端 OAuth 流程已就绪（`/api/v1/auth/github`, `/api/v1/auth/github/callback`）
- 环境变量配置：`GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`
