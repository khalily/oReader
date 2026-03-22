## Why

项目存在多个测试盲点，影响代码质量和安全性：
1. **Repository 层**：`user_item_state_repository.go` 包含复杂事务逻辑（`BulkMarkRead`、`MarkAllRead`）但没有测试
2. **Handler 层**：`import_handler.go` 处理文件上传但完全没有测试，存在安全风险
3. **安全测试**：`security_patterns_test.go` 模板被 `//go:build ignore` 排除，安全测试模式未应用到实际代码
4. **前端核心组件**：`ErrorBoundary.tsx` 和 `ItemsPage.tsx` 没有测试

## What Changes

### 后端测试补充
- 添加 `internal/repository/user_item_state_repository_test.go` - 测试复杂事务逻辑
- 添加 `internal/repository/user_feed_repository_test.go` - 测试用户订阅关系
- 添加 `internal/repository/import_job_repository_test.go` - 测试导入任务
- 添加 `internal/handler/import_handler_test.go` - 测试文件上传和安全边界
- 添加 `internal/handler/auth_security_test.go` - 应用安全测试模式

### 前端测试补充
- 添加 `web/src/components/__tests__/ErrorBoundary.test.tsx` - 测试错误边界组件
- 添加 `web/src/pages/items/__tests__/ItemsPage.test.tsx` - 测试主页面（含无限滚动）
- 添加 `web/src/components/feed/__tests__/DeleteConfirmDialog.test.tsx` - 测试删除确认对话框
- 添加 `web/src/hooks/__tests__/useKeyboardShortcuts.test.ts` - 测试键盘快捷键
- 添加 `web/src/stores/__tests__/itemsStore.test.ts` - 测试文章状态管理
- 添加 `web/src/stores/__tests__/authStore.test.ts` - 测试认证状态管理
- 添加 `web/src/contexts/__tests__/ThemeContext.test.tsx` - 测试主题上下文

## Capabilities

### New Capabilities

- `backend-repository-testing`: Repository 层单元测试覆盖（user_item_state, user_feed, import_job）
- `import-handler-testing`: Import Handler 文件上传和安全边界测试
- `security-testing-activation`: 安全测试模式激活（SQL 注入、XSS 测试）
- `frontend-core-testing`: 前端核心组件测试（ErrorBoundary, ItemsPage, useKeyboardShortcuts, itemsStore）

### Modified Capabilities

无 - 此次变更仅添加测试，不修改现有功能需求

## Impact

### 受影响的代码
- `internal/repository/` - 3 个新的测试文件
- `internal/handler/` - 2 个新的测试文件
- `web/src/components/` - 1 个新的测试文件
- `web/src/pages/` - 1 个新的测试文件
- `web/src/hooks/` - 1 个新的测试文件
- `web/src/stores/` - 1 个新的测试文件

### 验证方式
- 后端: `make test` 和 `go test ./... -coverprofile=coverage.out`
- 前端: `npm test` 和 `npm run test:coverage`

### 预期成果
- 测试覆盖率提升至 80%+
- 所有核心功能有对应的单元测试
- 安全测试模式被应用到实际模块
