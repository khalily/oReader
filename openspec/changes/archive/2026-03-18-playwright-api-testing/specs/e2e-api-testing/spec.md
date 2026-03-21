# E2E API 测试规格

## 新增需求

### 需求：Playwright 测试基础设施
系统应提供基于 Playwright 的端到端 API 测试基础设施，支持无头浏览器以兼容 CI/CD。

#### 场景：测试环境设置
- **当** 执行测试
- **则** 系统使用独立的测试数据库 (oreader_test.db)
- **且** 系统禁用速率限制 (RATE_LIMIT_ENABLED=false)
- **且** 系统在无头 Chromium 模式下运行

### 需求：认证测试夹具
系统应提供可复用的认证夹具，处理登录、CSRF 令牌提取和认证请求设置。

#### 场景：认证测试上下文
- **当** 测试需要认证
- **则** 夹具注册/登录测试用户
- **且** 夹具从响应中提取 csrf_token
- **且** 夹具返回包含 cookies 和 CSRF 令牌的上下文用于后续请求

### 需求：认证 API 测试覆盖
系统应为所有认证端点提供全面的测试。

#### 场景：注册测试
- **当** 使用有效邮箱和密码请求 POST /api/v1/auth/register
- **则** 响应为 201 Created，包含用户对象和 csrf_token
- **且** cookies 包含 access_token 和 csrf_token

#### 场景：登录测试
- **当** 使用有效凭据请求 POST /api/v1/auth/login
- **则** 响应为 200 OK，包含用户对象和 csrf_token
- **且** cookies 包含 access_token 和 csrf_token

#### 场景：获取当前用户测试
- **当** 使用有效 access_token cookie 请求 GET /api/v1/auth/me
- **则** 响应为 200 OK，包含用户资料

#### 场景：令牌刷新测试
- **当** 使用有效 refresh_token cookie 请求 POST /api/v1/auth/refresh
- **则** 响应为 200 OK，包含用户资料
- **且** 设置新的 access_token cookie

#### 场景：登出测试
- **当** 请求 POST /api/v1/auth/logout
- **则** 响应为 200 OK，包含成功消息
- **且** 清除所有认证 cookies

### 需求：订阅源 API 测试覆盖
系统应为所有订阅源管理端点提供全面的测试。

#### 场景：创建订阅源订阅
- **当** 使用有效 feed_url 和 X-CSRF-Token 请求头请求 POST /api/v1/feeds
- **则** 响应为 201 Created，包含订阅源对象和 new_item_count

#### 场景：列出订阅源
- **当** 使用有效认证请求 GET /api/v1/feeds
- **则** 响应为 200 OK，包含订阅源数组和总数

#### 场景：获取单个订阅源
- **当** 使用有效认证请求 GET /api/v1/feeds/:id
- **则** 响应为 200 OK，包含订阅源对象和 item_count

#### 场景：删除订阅源
- **当** 使用 X-CSRF-Token 请求头请求 DELETE /api/v1/feeds/:id
- **则** 响应为 204 No Content

#### 场景：刷新订阅源
- **当** 使用 X-CSRF-Token 请求头请求 POST /api/v1/feeds/:id/refresh
- **则** 响应为 200 OK，包含刷新结果

#### 场景：标记所有文章为已读
- **当** 使用 X-CSRF-Token 请求头请求 POST /api/v1/feeds/:id/mark-all-read
- **则** 响应为 200 OK，包含已标记为已读的文章数量

### 需求：文章 API 测试覆盖
系统应为所有文章管理端点提供全面的测试。

#### 场景：分页列出文章
- **当** 使用有效认证请求 GET /api/v1/items
- **则** 响应为 200 OK，包含文章数组、total、has_more 和 next_cursor

#### 场景：获取单个文章
- **当** 使用有效认证请求 GET /api/v1/items/:id
- **则** 响应为 200 OK，包含文章对象

#### 场景：切换收藏状态
- **当** 使用 X-CSRF-Token 请求头和 {starred: true} 请求 POST /api/v1/items/:id/star
- **则** 响应为 200 OK，包含更新后的文章对象

#### 场景：切换已读状态
- **当** 使用 X-CSRF-Token 请求头和 {read: true} 请求 POST /api/v1/items/:id/read
- **则** 响应为 200 OK，包含更新后的文章对象

### 需求：OPML API 测试覆盖
系统应为 OPML 导入/导出端点提供全面的测试。

#### 场景：导出订阅源为 OPML
- **当** 使用有效认证请求 GET /api/v1/opml/export
- **则** 响应为 200 OK，Content-Type: application/xml
- **且** 响应体为有效的 OPML 文档

#### 场景：导入 OPML 文件
- **当** 使用 X-CSRF-Token 请求头和有效 OPML 文件请求 POST /api/v1/opml/import
- **则** 响应为 202 Accepted，包含 job_id 和 total_feeds

#### 场景：获取导入任务状态
- **当** 使用有效认证请求 GET /api/v1/opml/import/:job_id
- **则** 响应为 200 OK，包含任务状态、进度和计数

### 需求：错误场景测试覆盖
系统应为错误场景提供测试，包括认证失败、验证错误和未找到情况。

#### 场景：未授权访问
- **当** 无有效 access_token 访问受保护端点
- **则** 响应为 401 Unauthorized

#### 场景：缺少 CSRF 令牌
- **当** 无 X-CSRF-Token 请求头对受保护端点执行 POST/PUT/DELETE
- **则** 响应为 403 Forbidden，包含 CSRF 错误

#### 场景：资源未找到
- **当** 请求不存在的订阅源或文章
- **则** 响应为 404 Not Found

### 需求：测试数据库隔离
系统应通过独立数据库和运行间清理确保完全的测试隔离。

#### 场景：每次运行使用全新数据库
- **当** 测试套件启动
- **则** 测试数据库全新创建并应用所有迁移
- **且** 不保留之前运行的任何数据
