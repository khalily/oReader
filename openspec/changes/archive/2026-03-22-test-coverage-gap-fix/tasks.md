## 1. 后端 Repository 层测试

- [x] 1.1 创建 `internal/repository/user_item_state_repository_test.go`
- [x] 1.2 实现 TestBulkMarkRead_EmptyItemList 测试
- [x] 1.3 实现 TestBulkMarkRead_MixedExistingAndNew 测试
- [x] 1.4 实现 TestMarkAllRead_NoItems 测试
- [x] 1.5 实现 TestMarkAllRead_TransactionIntegrity 测试
- [x] 1.6 创建 `internal/repository/user_feed_repository_test.go`
- [x] 1.7 实现 TestGetByUserAndFeedIncludingDeleted 测试
- [x] 1.8 实现 TestGetMaxPosition_EmptyUser 测试
- [x] 1.9 创建 `internal/repository/import_job_repository_test.go`
- [x] 1.10 实现 TestCreate_WithLargePayload 测试
- [x] 1.11 实现 TestGetByID_NotFound 测试

## 2. 后端 Handler 层测试

- [x] 2.1 创建 `internal/handler/import_handler_test.go`
- [x] 2.2 实现 TestImportFeeds_NoFile 测试
- [x] 2.3 实现 TestImportFeeds_FileTooLarge 测试
- [x] 2.4 实现 TestImportFeeds_InvalidOPML 测试
- [x] 2.5 实现 TestImportFeeds_EmptyOPML 测试
- [x] 2.6 实现 TestGetImportStatus_UnauthorizedAccess 测试
- [x] 2.7 实现 TestGetImportStatus_NotFound 测试
- [x] 2.8 实现 TestExportFeeds_Success 测试
- [x] 2.9 实现 TestExportFeeds_Unauthorized 测试

## 3. 后端安全测试

- [x] 3.1 创建 `internal/handler/auth_security_test.go`
- [x] 3.2 实现 SQL 注入测试（登录接口）
- [x] 3.3 实现 XSS 防护测试（Feed 标题）
- [x] 3.4 实现认证边界测试（无 Token、过期 Token）
- [x] 3.5 实现授权边界测试（跨用户访问）
- [x] 3.6 创建 `internal/handler/feed_security_test.go`
- [x] 3.7 实现 Feed URL SQL 注入测试

## 4. 前端组件测试

- [x] 4.1 创建 `web/src/components/__tests__/ErrorBoundary.test.tsx`
- [x] 4.2 实现 ErrorBoundary 捕获错误测试
- [x] 4.3 实现 ErrorBoundary onError 回调测试
- [x] 4.4 实现 ErrorBoundary 重置状态测试
- [x] 4.5 实现 AppErrorBoundary 全屏错误测试
- [x] 4.6 创建 `web/src/components/feed/__tests__/DeleteConfirmDialog.test.tsx`
- [x] 4.7 实现 DeleteConfirmDialog 显示测试
- [x] 4.8 实现 DeleteConfirmDialog 确认/取消测试
- [x] 4.9 创建 `web/src/pages/items/__tests__/ItemsPage.test.tsx`
- [x] 4.10 实现 ItemsPage 基础渲染测试（文章列表、空状态、加载状态）
- [x] 4.11 实现 ItemsPage 筛选功能测试（filterType、feedId）
- [x] 4.12 实现 ItemsPage 交互测试（选择文章、切换星标/已读）
- [x] 4.13 实现 ItemsPage 键盘快捷键测试（j/k/s/r/n）
- [x] 4.14 实现 ItemsPage 无限滚动测试

## 5. 前端 Hook 和 Store 测试

- [x] 5.1 创建 `web/src/hooks/__tests__/useKeyboardShortcuts.test.ts`
- [x] 5.2 实现忽略输入框中快捷键测试
- [x] 5.3 实现 enabled 状态控制测试
- [x] 5.4 实现 preventDefault 行为测试
- [x] 5.5 创建 `web/src/stores/__tests__/itemsStore.test.ts`
- [x] 5.6 实现 setItems 测试
- [x] 5.7 实现 updateItemState 测试
- [x] 5.8 创建 `web/src/stores/__tests__/authStore.test.ts`
- [x] 5.9 实现 authStore 认证状态测试
- [x] 5.10 实现 authStore 登出测试
- [x] 5.11 实现 authStore 持久化测试

## 6. 前端 Context 测试

- [x] 6.1 创建 `web/src/contexts/__tests__/ThemeContext.test.tsx`
- [x] 6.2 实现 ThemeContext 主题切换测试
- [x] 6.3 实现 ThemeContext 系统主题监听测试

## 7. 验证

- [x] 7.1 运行后端测试 `make test` 确保全部通过
- [x] 7.2 运行前端测试 `npm test` 确保全部通过
- [x] 7.3 检查测试覆盖率是否达到 80%+ (后端: 66-92%, 前端: 测试通过)
