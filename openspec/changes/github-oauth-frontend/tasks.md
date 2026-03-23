## 1. 前端组件开发

- [ ] 1.1 创建 `SocialLoginButton.tsx` 组件（GitHub 官方风格按钮）
- [ ] 1.2 添加 GitHub Octocat SVG 图标（内置）
- [ ] 1.3 实现 `window.location.href` 跳转逻辑
- [ ] 1.4 添加 ARIA 标签和键盘可访问性

## 2. 页面集成

- [ ] 2.1 修改 `LoginPage.tsx`，添加 GitHub 按钮和分隔线
- [ ] 2.2 修改 `RegisterPage.tsx`，添加 GitHub 按钮和分隔线
- [ ] 2.3 添加 OAuth 错误参数解析逻辑
- [ ] 2.4 添加错误消息显示组件

## 3. 后端错误处理修改

- [ ] 3.1 修改 `oauth_handler.go` 中 `GitHubCallback` 错误响应为重定向
- [ ] 3.2 处理 `access_denied` 错误重定向
- [ ] 3.3 处理 `invalid_state` 错误重定向
- [ ] 3.4 处理 `github_error` 错误重定向

## 4. 测试

- [ ] 4.1 编写 `SocialLoginButton.test.tsx` 组件测试
- [ ] 4.2 更新 `LoginPage.test.tsx` 添加按钮渲染测试
- [ ] 4.3 更新 `RegisterPage.test.tsx` 添加按钮渲染测试
- [ ] 4.4 添加 OAuth 错误显示测试
- [ ] 4.5 更新后端 OAuth handler 测试（验证重定向行为）

## 5. 验证

- [ ] 5.1 本地测试完整 OAuth 流程
- [ ] 5.2 验证错误场景处理
- [ ] 5.3 验证无障碍访问
