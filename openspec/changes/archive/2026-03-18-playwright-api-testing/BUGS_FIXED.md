# E2E 测试期间发现并修复的 Bug

## Bug #1：登录响应缺少 CSRF 令牌

**问题**：登录端点未在响应体中返回 `csrf_token`，导致与注册端点不一致。

**位置**：`internal/handler/auth.go`，第 250-260 行

**根本原因**：`Login` 函数调用 `setAuthCookies()` 发送带有 CSRF 令牌的响应，但随后又尝试发送不包含 CSRF 令牌的重复响应。

**修复**：移除 `Login` 函数中重复的 `c.JSON()` 调用（第 253-259 行），让 `setAuthCookies()` 处理完整响应。

**修改文件**：
- `internal/handler/auth.go`

**影响**：用户现在可以从登录响应中提取 CSRF 令牌，为后续请求启用正确的 CSRF 保护。

## Bug #2：刷新令牌响应缺少 CSRF 令牌

**问题**：刷新端点存在与登录相同的问题 - 重复响应发送。

**位置**：`internal/handler/auth.go`，第 386-394 行

**根本原因**：与 Bug #1 类似，`Refresh` 函数调用 `setAuthCookies()` 后又尝试发送另一个响应。

**修复**：移除 `Refresh` 函数中重复的 `c.JSON()` 调用（第 388-394 行），让 `setAuthCookies()` 处理完整响应。

**修改文件**：
- `internal/handler/auth.go`

**影响**：令牌刷新现在正确返回 CSRF 令牌，供后续认证请求使用。

## 总结

两个 bug 都与相同的模式相关：尝试发送多个 HTTP 响应。`setAuthCookies()` 辅助方法设计为发送包含 CSRF 令牌的完整响应，但调用函数也在尝试发送响应。这导致：

1. 来自 `setAuthCookies` 的第一个响应（带 CSRF 令牌）被忽略/覆盖
2. 来自调用者的第二个响应（不带 CSRF 令牌）被发送
3. 客户端无法提取 CSRF 令牌用于后续请求

修复确保所有认证端点（注册、登录、刷新）一致地在响应体中返回 CSRF 令牌。
