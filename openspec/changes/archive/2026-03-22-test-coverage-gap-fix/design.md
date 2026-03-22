## Context

oReader 项目是一个 Go (Gin) 后端 + React (TypeScript) 前端的 RSS 阅读器。当前测试存在以下问题：

**后端现状**：
- Repository 层：`user_item_state_repository.go` 包含复杂事务逻辑（`BulkMarkRead`、`MarkAllRead`）但没有测试
- Handler 层：`import_handler.go` 处理文件上传但完全没有测试
- 安全测试：`internal/testutil/security_patterns_test.go` 被显式排除

**前端现状**：
- 核心组件：`ErrorBoundary.tsx` 和 `ItemsPage.tsx` 没有测试
- Hooks 和 Stores：`useKeyboardShortcuts.ts` 和 `itemsStore.ts` 没有测试

**现有测试模式参考**：
- Repository 测试：`feed_repository_test.go`, `item_repository_test.go`
- Handler 测试：`feed_handler_test.go`, `item_handler_test.go`
- 组件测试：`FeedCard.test.tsx`
- Hook 测试：`useAuth.test.tsx`

## Goals / Non-Goals

**Goals:**
- 补齐 Repository 层测试（user_item_state, user_feed, import_job）
- 补齐 Handler 层测试（import_handler）
- 激活并应用安全测试模式
- 补齐前端核心组件测试（ErrorBoundary, ItemsPage, useKeyboardShortcuts, itemsStore）
- 测试覆盖率达到 80%+

**Non-Goals:**
- 不修改现有功能代码
- 不添加新功能
- 不修改 CI/CD 配置
- 不测试 shadcn/ui 等第三方 UI 组件

## Decisions

### 1. 测试框架选择
**决策**：使用现有测试框架
- 后端：Go testing + testify（已有模式）
- 前端：Vitest + Testing Library + MSW（已有模式）

**理由**：保持一致性，避免引入新依赖

### 2. 测试数据管理
**决策**：使用 `internal/testutil/` 现有工具
- `secrets.go` - 测试密钥管理
- `fixtures.go` - 测试数据
- `helpers.go` - 辅助函数

**理由**：复用现有基础设施，避免重复代码

### 3. 安全测试策略
**决策**：创建专门的安全测试文件，而不是修改 `security_patterns_test.go`
- `internal/handler/auth_security_test.go`
- `internal/handler/feed_security_test.go`

**理由**：保持模板文件作为参考，在实际模块中应用

### 4. 前端测试优先级
**决策**：优先测试关键用户路径
1. ErrorBoundary - 应用稳定性
2. ItemsPage - 主页面功能
3. useKeyboardShortcuts - 键盘交互
4. itemsStore - 状态管理

**理由**：确保核心功能有测试覆盖

## Risks / Trade-offs

**风险 1：测试执行时间增加**
→ 缓解：使用 `-short` 标志跳过慢测试，并行执行

**风险 2：测试维护成本**
→ 缓解：遵循现有测试模式，保持测试简洁

**风险 3：边界条件遗漏**
→ 缓解：参考 `security_patterns_test.go` 的攻击模式列表
