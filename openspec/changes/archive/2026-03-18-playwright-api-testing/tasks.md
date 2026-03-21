## 1. 基础设施设置

- [x] 1.1 在 web/ 目录安装 Playwright 和依赖
- [x] 1.2 创建 tests/e2e/ 目录结构
- [x] 1.3 创建 playwright.config.ts，配置基础 URL、无头模式和测试数据库设置
- [x] 1.4 创建 .env.test 配置测试环境（DATABASE_URL=oreader_test.db, RATE_LIMIT_ENABLED=false）
- [x] 1.5 添加运行 E2E 测试的 npm 脚本（test:e2e, test:e2e:ui）

## 2. 测试夹具和辅助函数

- [x] 2.1 创建 tests/e2e/fixtures/auth.ts 包含认证辅助函数
- [x] 2.2 实现从响应提取 csrf_token 的 login 辅助函数
- [x] 2.3 实现自动注入 CSRF 头的认证请求辅助函数
- [x] 2.4 创建生成唯一测试用户的测试用户工厂

## 3. Bug 修复 - 登录响应

- [x] 3.1 修复登录端点在响应体中包含 csrf_token（internal/handler/auth.go）
- [x] 3.2 验证登录响应与注册响应结构匹配
- [x] 3.3 手动或用单元测试测试修复

## 4. 认证 API 测试

- [x] 4.1 创建 tests/e2e/auth.spec.ts
- [x] 4.2 测试：POST /api/v1/auth/register - 成功注册
- [x] 4.3 测试：POST /api/v1/auth/register - 重复邮箱错误
- [x] 4.4 测试：POST /api/v1/auth/register - 无效邮箱格式
- [x] 4.5 测试：POST /api/v1/auth/login - 成功登录并在响应中包含 csrf_token
- [x] 4.6 测试：POST /api/v1/auth/login - 无效凭据
- [x] 4.7 测试：GET /api/v1/auth/me - 获取当前用户
- [x] 4.8 测试：POST /api/v1/auth/refresh - 令牌刷新
- [x] 4.9 测试：POST /api/v1/auth/logout - 登出清除 cookies

## 5. 订阅源 API 测试

- [x] 5.1 创建 tests/e2e/feeds.spec.ts
- [x] 5.2 测试：POST /api/v1/feeds - 创建订阅源订阅（使用 OpenAI RSS）
- [x] 5.3 测试：GET /api/v1/feeds - 列出订阅源
- [x] 5.4 测试：GET /api/v1/feeds/:id - 获取单个订阅源
- [x] 5.5 测试：POST /api/v1/feeds/:id/refresh - 刷新订阅源
- [x] 5.6 测试：POST /api/v1/feeds/:id/mark-all-read - 标记所有文章为已读
- [x] 5.7 测试：DELETE /api/v1/feeds/:id - 删除订阅源
- [x] 5.8 测试：错误场景 - 无效 URL、重复订阅源、未找到

## 6. 文章 API 测试

- [x] 6.1 创建 tests/e2e/items.spec.ts
- [x] 6.2 测试：GET /api/v1/items - 分页列出文章
- [x] 6.3 测试：GET /api/v1/items?feed_id=X - 按订阅源筛选
- [x] 6.4 测试：GET /api/v1/items?starred=true - 按收藏筛选
- [x] 6.5 测试：GET /api/v1/items/:id - 获取单个文章
- [x] 6.6 测试：POST /api/v1/items/:id/star - 切换收藏状态
- [x] 6.7 测试：POST /api/v1/items/:id/read - 切换已读状态
- [x] 6.8 测试：错误场景 - 文章未找到、未授权访问

## 7. OPML API 测试

- [x] 7.1 创建 tests/e2e/opml.spec.ts
- [x] 7.2 测试：GET /api/v1/opml/export - 导出订阅源为 OPML
- [x] 7.3 测试：POST /api/v1/opml/import - 导入 OPML 文件
- [x] 7.4 测试：GET /api/v1/opml/import/:job_id - 获取导入状态

## 8. 错误处理测试

- [x] 8.1 测试：401 Unauthorized - 缺少访问令牌
- [x] 8.2 测试：403 Forbidden - POST 缺少 CSRF 令牌
- [x] 8.3 测试：403 Forbidden - CSRF 令牌不匹配
- [x] 8.4 测试：404 Not Found - 不存在的资源
- [x] 8.5 测试：速率限制行为（可选，如果启用速率限制测试）

## 9. 测试-修复循环

- [x] 9.1 运行所有测试并捕获失败
- [x] 9.2 修复每个发现的 bug
- [x] 9.3 重新运行测试直到全部通过
- [x] 9.4 记录发现和修复的额外 bug

## 10. 文档和清理

- [x] 10.1 在 tests/e2e/ 添加 README.md，包含设置和运行说明
- [x] 10.2 更新主 README.md 添加 E2E 测试部分
- [x] 10.3 清理任何测试产物
- [x] 10.4 验证测试在 CI 环境中运行（如适用）
