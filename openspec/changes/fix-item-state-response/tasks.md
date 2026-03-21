## 1. TDD 阶段 1 - 编写失败测试 (RED)

- [ ] 1.1 为带有嵌套 `user_state` 对象的 `ItemWithState` JSON 序列化编写失败单元测试
- [ ] 1.2 为带有嵌套 `feed` 对象的 `ItemWithState` JSON 序列化编写失败单元测试
- [ ] 1.3 为 `user_state` 为 null 的 `ItemWithState` 编写失败单元测试
- [ ] 1.4 为 ListItems 响应结构编写失败集成测试
- [ ] 1.5 为 GetItem 响应结构编写失败集成测试
- [ ] 1.6 为 SetStar 响应结构编写失败集成测试
- [ ] 1.7 为 SetRead 响应结构编写失败集成测试
- [ ] 1.8 运行测试验证它们因正确原因失败

## 2. TDD 阶段 2 - 实现 (GREEN)

- [ ] 2.1 在 `internal/service/interfaces.go` 中创建 `FeedInfo` 结构体
- [ ] 2.2 在 `internal/service/interfaces.go` 中创建 `UserItemState` 响应结构体
- [ ] 2.3 更新 `ItemWithState` 以使用嵌套的 `Feed` 和 `UserState` 指针
- [ ] 2.4 更新 `internal/service/item_service.go` 中的 `buildItemWithState()`
- [ ] 2.5 更新 `internal/repository/item_repository.go` 中的 `buildItemWithState()`
- [ ] 2.6 运行所有测试验证通过

## 3. TDD 阶段 3 - 重构

- [ ] 3.1 检查并移除 buildItemWithState 函数中的任何代码重复
- [ ] 3.2 确保所有响应路径的 null 处理一致
- [ ] 3.3 验证测试覆盖率达到 80% 阈值

## 4. 修复 E2E 测试

- [ ] 4.1 向 `web/tests/e2e/fixtures/auth.ts` 添加 `authPut` 辅助函数
- [ ] 4.2 更新 `web/tests/e2e/items.spec.ts` 中的 star 测试以使用 `authPut`
- [ ] 4.3 更新 `web/tests/e2e/items.spec.ts` 中的 read 测试以使用 `authPut`
- [ ] 4.4 运行 E2E 测试验证通过

## 5. 验证

- [ ] 5.1 运行完整 Go 测试套件 (`go test ./...`)
- [ ] 5.2 运行完整前端测试套件 (`npm test`)
- [ ] 5.3 运行 E2E 测试 (`npx playwright test`)
- [ ] 5.4 浏览器手动验证：点击 star/read 按钮正常工作
