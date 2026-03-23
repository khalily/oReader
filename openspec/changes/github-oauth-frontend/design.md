## Context

oReader 是一个在线 RSS 阅读器，采用 Go 后端 + React 前端架构。后端已完整实现 GitHub OAuth 流程：
- `GET /api/v1/auth/github` - 发起 OAuth，生成 state 参数
- `GET /api/v1/auth/github/callback` - 处理回调，设置 cookie
- State 参数防 CSRF
- 通过 email 自动关联现有账户

当前前端仅有邮箱/密码登录入口，缺少 GitHub OAuth 入口。

**技术栈：**
- 前端：React + TypeScript + Tailwind CSS + shadcn/ui
- 状态管理：Zustand + React Query
- 后端：Gin + GORM + JWT

## Goals / Non-Goals

**Goals:**
- 在登录和注册页面添加 GitHub 登录按钮
- 支持 OAuth 错误提示
- 保持现有邮箱/密码登录体验

**Non-Goals:**
- 不添加其他 OAuth 提供商（Google、Apple）
- 不实现账户关联设置页面
- 不修改 JWT token 结构或认证流程

## Decisions

### 1. 跳转方式 vs Popup

**选择：页面跳转**

理由：
- 后端已实现 cookie-based 认证，跳转方式完美适配
- 实现简单可靠，无浏览器兼容性问题
- 主流产品（GitHub、GitLab、Netlify）都采用这种方式

**替代方案：Popup 窗口**
- 需要后端修改回调返回 HTML 而非重定向
- 可能被浏览器拦截
- 增加前端复杂度

### 2. 按钮位置

**选择：表单上方**

理由：
- "Sign in with GitHub" 优先于邮箱登录，符合低摩擦注册理念
- 视觉层次清晰：社交登录 → 分隔线 → 传统登录

### 3. 错误处理

**选择：URL 参数 + 重定向**

理由：
- 保持无状态，无需前端存储错误信息
- 与后端重定向方式一致
- 简单可靠

## Risks / Trade-offs

| 风险 | 缓解措施 |
|------|----------|
| 用户点击后离开页面 | 这是预期行为，OAuth 流程需要跳转 |
| 后端错误响应需修改为重定向 | 轻量修改，不影响现有流程 |
| 浏览器可能阻止 cookie（第三方） | 应用部署在同域下，不涉及第三方 cookie |
