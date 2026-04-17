# Category Feature Code Review 修复 — Bug 与问题复盘

> **日期**: 2026-04-05
> **功能**: Category 分类功能的 Code Review 修复 + 运行时 bug 修复
> **变更规模**: 原始功能 41 文件, +4363/-885 行, 14 commits；本轮修复 ~12 文件

---

## 一、实现 Bug（7 个）

### Bug #1: PaperRepository.UpdateCategory 缺少 userID 隔离

| 属性 | 值 |
|------|-----|
| **任务** | 原始 Task — Category Service 实现 |
| **发现阶段** | Code Quality Review |
| **严重性** | Critical（授权漏洞） |
| **问题** | `PaperRepository.UpdateCategory` 只按 `paperID` 过滤，未按 `userID` 过滤 |
| **根因** | 实现时只参照了自身逻辑，未与 `UserFeedRepository.UpdateCategory`（已有 userID 隔离）保持一致 |
| **修复** | `interfaces.go:326`, `paper_repository.go:139-148` — 接口添加 `userID` 参数，查询加 `WHERE user_id = ? AND id = ?`，并添加 `RowsAffected == 0` 检查返回 `ErrPaperNotFound` |

### Bug #2: ItemsPage 测试与组件完全脱节

| 属性 | 值 |
|------|-----|
| **任务** | 原始 Task — ItemsPage 重写 |
| **发现阶段** | Code Quality Review |
| **严重性** | Critical（测试无效） |
| **问题** | 测试文件传递 `filterType` 和 `feedId` props，但组件已改为无 props，使用 `useSidebarStore` 和 `useSearchParams` |
| **根因** | 组件重构后测试未同步更新，测试仍引用旧 API |
| **修复** | `ItemsPage.test.tsx` — 完全重写测试，移除废弃 props，添加 categories MSW handlers，测试 sidebar 驱动行为 |

### Bug #3: Paper authors 字段 JSON.parse 返回 null 导致崩溃

| 属性 | 值 |
|------|-----|
| **任务** | 原始 Task — ItemsPage Paper 列表渲染 |
| **发现阶段** | E2E 验证（生产环境运行时错误） |
| **严重性** | Critical（页面白屏） |
| **问题** | `JSON.parse(p.authors).slice(0, 2)` 当 `authors` 为 `"null"` 字符串时，`JSON.parse("null")` 返回 JS `null`，对 `null` 调用 `.slice()` 抛出 TypeError |
| **根因** | `p.authors ?` 的 truthy 检查无法过滤 `"null"` 字符串（非空字符串为 truthy），但 `JSON.parse("null")` 返回 `null` |
| **修复** | `ItemsPage.tsx:470,641` — 用 `try/catch` + `Array.isArray()` 双重保护 |

### Bug #4: 重复错误检测依赖字符串匹配

| 属性 | 值 |
|------|-----|
| **任务** | 原始 Task — CategoryService |
| **发现阶段** | Code Quality Review |
| **严重性** | High（静默失败风险） |
| **问题** | `strings.Contains(err.Error(), "duplicate")` 依赖错误消息文本，不同 MySQL 版本/配置可能产生不同消息 |
| **根因** | 使用了最简单的实现方式，未考虑数据库错误消息的跨版本兼容性 |
| **修复** | `category_service.go:36-39` — 新增 `isDuplicateKeyError()` 函数，使用 `errors.As` 检查 `mysql.MySQLError.Number == 1062` |

### Bug #5: Category mutation 未刷新 TanStack Query 缓存

| 属性 | 值 |
|------|-----|
| **任务** | 原始 Task — useCategories hook |
| **发现阶段** | Code Quality Review |
| **严重性** | High（数据不一致） |
| **问题** | `useCreateCategory`、`useRenameCategory`、`useDeleteCategory` 等 mutation 无 `onSuccess` 处理器，操作后侧边栏显示过期数据 |
| **根因** | hook 实现时只关注了 mutation 本身，未考虑缓存失效 |
| **修复** | `useCategories.ts` — 每个 mutation 添加 `onSuccess: () => queryClient.invalidateQueries({ queryKey: ['categories'] })` |

### Bug #6: FeedRow 菜单在无分类时完全隐藏

| 属性 | 值 |
|------|-----|
| **任务** | 原始 Task — FeedRow 组件 |
| **发现阶段** | Code Quality Review |
| **严重性** | Medium（可发现性问题） |
| **问题** | 当 `allFeedCategories.length === 0` 时整个右键菜单不渲染，包括"新建分类"按钮 |
| **根因** | `menuOpen && allFeedCategories.length > 0` 条件过于严格 |
| **修复** | `FeedRow.tsx:119` — 改为 `menuOpen` 始终显示菜单，"新建分类" 始终可选，仅条件渲染已有分类列表 |

### Bug #7: OpenAPI spec 缺失 Category 端点

| 属性 | 值 |
|------|-----|
| **任务** | 原始 Task — Category API 实现 |
| **发现阶段** | Code Quality Review |
| **严重性** | Medium（文档缺失） |
| **问题** | 新增的 5 个 Category 端点未添加到 OpenAPI spec，CI openapi-lint/openapi-test 可能失败 |
| **根因** | 实现后未同步更新 API 文档 |
| **修复** | `docs/openapi.yaml` — 添加 Categories tag、5 个端点、CategoryIdParam、6 个 schemas |

---

## 二、流程与基础设施问题（2 个）

| # | 问题 | 影响 |
|---|------|------|
| 1 | `go-sql-driver/mysql` 原为 indirect 依赖，添加 `isDuplicateKeyError` 后需改为 direct | `go mod tidy` 自动处理，但需注意 CI 中 `dependabot` 可能需要调整 |
| 2 | 测试重写时 MSW handler 缺少 `/categories` 端点，导致初次测试运行可能失败 | 通过添加完整的 MSW handlers 解决 |

---

## 三、预先存在的问题（1 个）

| # | 问题 | 包/模块 | 根因 |
|---|------|---------|------|
| 1 | Paper `authors` 字段值可为 `"null"` 字符串 | backend converter | MinerU/LLM 提取元数据时，若无作者信息，将 authors 设为 JSON `"null"` 而非空字符串 `""` |

---

## 四、教训总结

### 教训 1: Repository 层操作必须按 userID 隔离

> **来源**: Bug #1
> **规则**: 所有涉及用户数据的 Repository 方法（Update/Delete/Get），必须包含 `WHERE user_id = ?` 条件。修改 service 方法签名时，必须 grep 所有调用点和 mock 实现进行同步更新。

### 教训 2: 组件重构后测试必须同步更新

> **来源**: Bug #2
> **规则**: 当组件 API 变更（如移除 props、改用 store/hook 获取状态），对应的测试文件必须同步重写。测试中引用的 props 必须与组件实际 API 完全匹配，否则测试结果毫无意义。

### 教训 3: JSON.parse 不可信数据必须防御

> **来源**: Bug #3
> **规则**: 对数据库存储的 JSON 字符串执行 `JSON.parse` 后，必须验证返回值类型。`JSON.parse("null")` 返回 `null`，`JSON.parse("123")` 返回数字。使用 `Array.isArray()` 或 try/catch 保护。

### 教训 4: 数据库错误检测用错误码而非消息文本

> **来源**: Bug #4
> **规则**: 检测 MySQL 特定错误（如 duplicate key、foreign key violation）时，使用 `errors.As` 提取 `*mysql.MySQLError` 检查 `Number` 字段（如 1062），不依赖 `err.Error()` 文本匹配。

### 教训 5: TanStack Query mutation 必须配置缓存失效

> **来源**: Bug #5
> **规则**: 每个 `useMutation` 如果修改了某个 query 涉及的数据，必须在 `onSuccess` 中调用 `queryClient.invalidateQueries({ queryKey: ['...'] })`，否则用户会看到过期数据。

### 教训 6: 新增 API 端点必须同步更新 OpenAPI spec

> **来源**: Bug #7
> **规则**: 新增/修改/删除 API 端点时，必须同步更新 `docs/openapi.yaml`，包括 paths、parameters、schemas 和 tags。CI 的 openapi-lint 会校验一致性。

---

## 五、统计

| 类别 | 数量 |
|------|------|
| 实现 Bug（已修复） | 7 |
| 流程/基础设施问题 | 2 |
| 预先存在问题 | 1 |
| **总计** | **10** |
