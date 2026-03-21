## 背景

oReader 应用的后端 API 响应结构与前端对文章状态（is_starred、is_read）的期望存在不匹配。后端将这些作为文章对象上的扁平字段返回，而前端期望一个嵌套的 `user_state` 对象。这导致前端始终显示默认值。

### 问题 3：All Items 列表顺序随机变化
- **症状**：在 "All Articles" 视图点击 star/read 后，列表顺序发生变化
- **根因**：Go map 遍历顺序随机 + 合并后缺少排序
- **修复**：添加 `sort.Slice` 按 `pub_date DESC NULLS LAST` 排序，ID 作为次要键

### 问题 4：F5 刷新后跳转登录页
- **症状**：页面刷新后被重定向到登录页
- **根因**：Zustand store 状态在刷新后重置，原 `initializeFromStorage` 只恢复 token
- **修复**：使用 Zustand `persist` 中间件自动持久化认证状态

### 当前状态
```go
// internal/service/interfaces.go
type ItemWithState struct {
    *model.Item
    FeedTitle string  `json:"feed_title"`
    IsStarred bool    `json:"is_starred"`  // ← 扁平字段
    IsRead    bool    `json:"is_read"`     // ← 扁平字段
    ReadAt    *string `json:"read_at,omitempty"`
}
```

### 期望状态
```go
type ItemWithState struct {
    *model.Item
    Feed      *FeedInfo      `json:"feed"`
    UserState *UserItemState `json:"user_state"`
}

type FeedInfo struct {
    ID    string `json:"id"`
    Title string `json:"title"`
}

type UserItemState struct {
    ItemID    string  `json:"item_id"`
    IsStarred bool    `json:"is_starred"`
    IsRead    bool    `json:"is_read"`
    ReadAt    *string `json:"read_at"`
}
```

## 目标 / 非目标

**目标：**
- 修复后端响应结构以匹配前端 TypeScript 类型
- 确保 E2E 测试使用正确的 HTTP 方法（PUT 用于 star/read）
- 为任何 API 消费者维护向后兼容性（记录为破坏性变更）

**非目标：**
- 无需前端更改（前端已期望正确的结构）
- 无数据库模式更改
- 无业务逻辑或验证更改

## 决策

### 决策 1：重构 Go 中的 ItemWithState
**选择：** 为响应 DTO 创建嵌套结构体 `Feed` 和 `UserState`。

**理由：** 前端 TypeScript 类型已定义此结构：
```typescript
interface Article extends Item {
  feed: Feed
  user_state: UserItemState | null
}
```

**考虑的替代方案：**
- 更改前端以匹配后端：拒绝 - 前端结构更易维护，关注点分离更好
- 同时添加扁平字段和嵌套字段：拒绝 - 造成混乱和冗余

### 决策 2：TDD 实现方法
**选择：** 遵循严格的 red-green-refactor TDD 循环。

**理由：** 这是一个定义明确的变更，有清晰的预期行为。TDD 确保：
1. 测试捕获新的响应结构需求
2. 实现最小且正确
3. 现有功能无回归

**测试类别：**
1. **单元测试** - `ItemWithState` JSON 序列化
2. **集成测试** - 处理程序响应结构
3. **E2E 测试** - 完整 API 契约验证

### 决策 3：E2E 测试修复策略（已废弃）
**选择：** 添加 `authPut` 辅助函数并更新 star/read 测试。

**理由：** 最小更改以修复 HTTP 方法不匹配。`authPut` 辅助函数遵循与现有 `authPost` 和 `authGet` 相同的模式。

**更新：** Playwright E2E 测试已被移除，原因是维护成本高且环境依赖复杂。后端 Go 测试和前端 Vitest 测试已提供足够的覆盖。

### 决策 4：All Items 排序
**选择：** 在服务层合并 items 后添加 `sort.Slice` 排序。

**实现：**
- 主键：`pub_date DESC NULLS LAST`
- 次要键：`ID DESC`（确保确定性）

**理由：** Go map 遍历顺序是随机的，导致 "All Articles" 视图在每次操作后顺序变化。添加确定性排序确保用户体验一致性。

### 决策 5：前端认证持久化
**选择：** 使用 Zustand persist 中间件。

**配置：**
- 持久化字段：`isAuthenticated`, `user`, `csrfToken`
- localStorage key: `auth-storage`

**理由：** 原有的 `initializeFromStorage` 方法只在初始化时恢复 token，刷新后 `isAuthenticated` 仍为 false。使用 persist 中间件可以自动同步状态到 localStorage，在刷新后自动恢复完整的认证状态。

## 风险 / 权衡

| 风险 | 缓解措施 |
|------|----------|
| 对现有消费者的破坏性 API 变更 | 记录为破坏性变更；前端已期望新结构 |
| 边缘情况测试覆盖不足 | 使用 TDD 确保覆盖所有场景 |
| null UserState 的 JSON 序列化问题 | 显式测试 null 情况；前端正确处理 null |

## 迁移计划

1. **阶段 1：** 为新响应结构编写失败测试
2. **阶段 2：** 更新 `ItemWithState` 结构体和 `buildItemWithState` 函数
3. **阶段 3：** 验证所有测试通过
4. **阶段 4：** 运行 E2E 测试验证完整 API 契约

**回滚：** 如出现问题，简单 git revert - 无数据库更改。

## 待解决问题

无 - 需求在探索阶段已明确定义。
