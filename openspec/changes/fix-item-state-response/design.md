## 原因

`is_starred` 和 `is_read` 功能不工作，因为后端 API 将这些字段作为文章对象上的扁平属性返回，但前端期望它们在嵌套的 `user_state` 对象中。这导致前端始终显示默认值（false），因为 `article.user_state?.is_read` 评估为 undefined。

此外，E2E 测试对 star/read 端点使用 POST 请求，但后端仅接受 PUT 请求，导致测试失败。

## 变更内容

### 后端 API 响应结构（**破坏性变更**）
- 更改 `ItemWithState` 以将 `user_state` 作为嵌套对象而非扁平字段返回
- 将 `feed_title` 字符串更改为包含 `id` 和 `title` 的 `feed` 对象
- 响应结构从：
  ```json
  { "is_starred": true, "is_read": false, "feed_title": "Feed" }
  ```
  更改为：
  ```json
  { "user_state": { "is_starred": true, "is_read": false }, "feed": { "title": "Feed" } }
  ```

### E2E 测试修复
- 向测试夹具添加 `authPut` 辅助函数
- 更新 star/read 端点测试以使用 PUT 而非 POST

## 能力

### 新增能力
- 无

### 修改的能力
- `article-management`：API 响应结构更改为包含嵌套的 `user_state` 和 `feed` 对象。需求保持不变，但 JSON 序列化格式变更。

## 影响

### 后端 (Go)
- `internal/service/interfaces.go` - 使用嵌套结构体更新 `ItemWithState` 结构体
- `internal/service/item_service.go` - 更新 `buildItemWithState()` 函数
- `internal/repository/item_repository.go` - 更新 `buildItemWithState()` 函数
- 所有返回文章的处理程序将自动使用新结构

### 前端 (TypeScript) - 无需更改
- 前端已期望正确的结构（`user_state`、`feed` 对象）
- `web/src/types/feed.ts` 已定义正确的 `Article` 类型

### 测试
- `web/tests/e2e/fixtures/auth.ts` - 添加 `authPut` 辅助函数
- `web/tests/e2e/items.spec.ts` - 将 star/read 测试的 POST 改为 PUT

### API 契约
- **破坏性变更**：所有文章端点的 JSON 响应结构变更
- 前端代码已期望新结构，因此无需前端更改
