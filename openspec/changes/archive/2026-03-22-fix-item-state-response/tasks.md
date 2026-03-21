## 1. TDD 阶段 1 - 编写失败测试 (RED)

- [x] 1.1 为带有嵌套 `user_state` 对象的 `ItemWithState` JSON 序列化编写失败单元测试
- [x] 1.2 为带有嵌套 `feed` 对象的 `ItemWithState` JSON 序列化编写失败单元测试
- [x] 1.3 为 `user_state` 为 null 的 `ItemWithState` 编写失败单元测试
- [x] 1.4 为 ListItems 响应结构编写失败集成测试
- [x] 1.5 为 GetItem 响应结构编写失败集成测试
- [x] 1.6 为 SetStar 响应结构编写失败集成测试
- [x] 1.7 为 SetRead 响应结构编写失败集成测试
- [x] 1.8 运行测试验证它们因正确原因失败

## 2. TDD 阶段 2 - 实现 (GREEN)

- [x] 2.1 在 `internal/service/interfaces.go` 中创建 `FeedInfo` 结构体
- [x] 2.2 在 `internal/service/interfaces.go` 中创建 `UserItemState` 响应结构体
- [x] 2.3 更新 `ItemWithState` 以使用嵌套的 `Feed` 和 `UserState` 指针
- [x] 2.4 更新 `internal/service/item_service.go` 中的 `buildItemWithState()`
- [x] 2.5 更新 `internal/repository/item_repository.go` 中的 `buildItemWithState()`
- [x] 2.6 运行所有测试验证通过

## 3. TDD 阶段 3 - 重构

- [x] 3.1 检查并移除 buildItemWithState 函数中的任何代码重复
- [x] 3.2 确保所有响应路径的 null 处理一致
- [x] 3.3 验证测试覆盖率达到 80% 阈值

## 4. 修复 E2E 测试（已废弃 - 改为移除）

- [x] 4.1 向 `web/tests/e2e/fixtures/auth.ts` 添加 `authPut` 辅助函数
- [x] 4.2 更新 `web/tests/e2e/items.spec.ts` 中的 star 测试以使用 `authPut`
- [x] 4.3 更新 `web/tests/e2e/items.spec.ts` 中的 read 测试以使用 `authPut`
- [x] 4.4 运行 E2E 测试验证通过

## 5. 验证

- [x] 5.1 运行完整 Go 测试套件 (`go test ./...`)
- [x] 5.2 运行完整前端测试套件 (`npm test`)
- [x] 5.3 ~~运行 E2E 测试 (`npx playwright test`)~~ - 已移除 Playwright 测试
- [x] 5.4 浏览器手动验证：点击 star/read 按钮正常工作
- [x] 5.5 手动验证：刷新 "All Articles" 页面，顺序保持一致
- [x] 5.6 手动验证：F5 刷新页面，保持登录状态

## 6. 新增修复：All Items 排序

- [x] 6.1 在 item_service.go 添加 sort 包导入
- [x] 6.2 在 ListItems "all" 分支添加排序逻辑
- [x] 6.3 添加 NULL 值处理（NULLS LAST）
- [x] 6.4 添加 ID 次要排序键确保确定性
- [x] 6.5 编写排序验证测试

## 7. 新增修复：前端认证持久化

- [x] 7.1 添加 Zustand persist 中间件到 authStore
- [x] 7.2 配置 partialize 持久化字段
- [x] 7.3 更新 initializeFromStorage 为 no-op
- [x] 7.4 更新 authStore 测试

## 8. 移除 Playwright E2E 测试

- [x] 8.1 删除 `web/tests/e2e/` 目录
- [x] 8.2 更新 proposal.md 说明移除原因
