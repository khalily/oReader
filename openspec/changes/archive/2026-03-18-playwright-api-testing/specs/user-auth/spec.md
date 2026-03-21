# 用户认证增量规格

## 修改的需求

### 需求：双令牌认证的用户登录
系统应使用邮箱和密码认证用户，颁发存储在 HttpOnly cookies 中的双令牌（访问和刷新），并在响应体中返回 CSRF 令牌。

#### 场景：成功登录
- **当** 用户向 POST /api/v1/auth/login 提交正确的邮箱和密码
- **则** 系统设置 access_token cookie（HttpOnly，开发环境 SameSite=Lax，15 分钟）
- **且** 系统设置 csrf_token cookie（可被 JS 读取，15 分钟）
- **且** 系统设置 refresh_token cookie（HttpOnly，开发环境 SameSite=Lax，7 天，Path=/api/v1/auth/refresh）
- **且** 系统在响应体中返回用户资料
- **且** 系统在响应体中返回 csrf_token 供客户端使用

#### 场景：无效凭据
- **当** 用户提交不正确的邮箱或密码
- **则** 系统返回 401 Unauthorized
- **且** 系统不透露哪个字段不正确
