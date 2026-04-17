# Sidebar 优化功能 — Bug 与问题复盘

> **日期**: 2026-04-05
> **功能**: 分类侧边栏优化（12 Tasks, Subagent-Driven Development）
> **变更规模**: 40 文件, +4005/-886 行, 13 个 commits

---

## 一、实现 Bug（6 个）

### Bug #1: Category 模型缺少唯一约束

| 属性 | 值 |
|------|-----|
| **任务** | Task 1 — Category Model + Migration |
| **发现阶段** | Code Quality Review |
| **严重性** | 高（数据完整性） |
| **问题** | GORM tag 使用 `index:idx_user_cat_type`（普通索引）而非 `uniqueIndex`，允许同一用户创建同名同类型分类 |
| **根因** | 实现者混淆了 `index` 和 `uniqueIndex` 的 GORM tag 语法 |
| **修复** | `backend/internal/model/category.go` — 改为 `uniqueIndex:idx_user_cat_name` |

### Bug #2: FK 字段缺少 ON DELETE SET NULL

| 属性 | 值 |
|------|-----|
| **任务** | Task 1 — Category Model + Migration |
| **发现阶段** | Code Quality Review |
| **严重性** | 高（数据完整性） |
| **问题** | `UserFeed.CategoryID` 和 `Paper.CategoryID` 外键字段没有指定级联策略，删除分类时会导致外键约束错误 |
| **根因** | 实现者添加了 FK 字段但遗漏了 referential action |
| **修复** | `models.go` 和 `paper.go` 的 GORM tag 添加 `constraint:OnDelete:SET NULL` |

### Bug #3: GetByID/Delete 未按 userID 隔离（安全漏洞）

| 属性 | 值 |
|------|-----|
| **任务** | Task 2 — Category Repository + Service Interface |
| **发现阶段** | Code Quality Review |
| **严重性** | 高（安全 — 越权访问） |
| **问题** | `CategoryRepository.GetByID()` 和 `Delete()` 没有 `WHERE user_id = ?` 条件，任何已认证用户可通过 ID 访问或删除他人分类 |
| **根因** | 实现者按基础 CRUD 模式编写，未应用 defense-in-depth 原则 |
| **修复** | 接口签名加入 `userID` 参数，实现层 WHERE 条件增加 `user_id = ?` 过滤 |

### Bug #4: GetUserFeeds 改动导致 5 个现有测试 nil pointer panic

| 属性 | 值 |
|------|-----|
| **任务** | Task 5 / Task 12 |
| **发现阶段** | E2E Verification |
| **严重性** | 高（5 个测试崩溃） |
| **问题** | Task 5 给 `GetUserFeeds` 新增 `s.userFeedRepo.ListByUserID()` 调用，但 5 个已有测试传入 `nil` 作为 `userFeedRepo` 参数，导致运行时 nil pointer dereference |
| **受影响测试** | `Success`, `ItemCountError`, `EmptyResult`, `RepoError`, `IsolatedByUser` |
| **根因** | 修改 service 方法依赖时未全量搜索相关测试 |
| **修复** | 分三轮修复（上下文窗口限制中断两次），逐一添加 mock `ListByUserID` 返回值 |

### Bug #5: 前后端 API 响应类型不匹配（运行时 Bug）

| 属性 | 值 |
|------|-----|
| **任务** | Final Code Review |
| **发现阶段** | 最终代码审查 |
| **严重性** | 严重（关键用户流程静默失败） |
| **问题** | 后端 `CreateCategory`/`RenameCategory` 返回 `{"category": Category}`，前端类型声明为 `Promise<Category>`。`newCategory.id` 实际为 `undefined`，导致"移动到新建分类"功能完全失效 |
| **根因** | 前后端 subagent 独立工作，各自假设了不同的响应格式。TypeScript 泛型 `apiClient.post<Category>` 不校验实际 JSON 结构 |
| **修复** | `useCategories.ts` 返回类型改为 `{ category: Category }`，`ItemsPage.tsx` 改用 `newCategory.category.id` |

### Bug #6: 缺少"取消分类"功能

| 属性 | 值 |
|------|-----|
| **任务** | Final Code Review |
| **发现阶段** | 最终代码审查 |
| **严重性** | 中（功能不完整） |
| **问题** | 用户可以将 feed/paper 移入分类，但无法从分类中移出 |
| **根因** | 实现者只关注了"正向操作"，忽略了反向操作 |
| **修复** | 后端 Move 方法支持空 `category_id`（设 NULL），前端 FeedRow/PaperRow 添加"移除分类"菜单项 |

---

## 二、流程与基础设施问题（4 个）

### Issue #7: 前端 subagent 未提交代码（Tasks 7-10）

| 属性 | 值 |
|------|-----|
| **严重性** | 中（流程漏洞） |
| **问题** | 4 个前端 subagent 创建/修改了文件但未 commit，控制器需手动 stage + commit |
| **根因** | implementer subagent prompt 未包含明确的 commit 指令 |

### Issue #8: 上下文窗口两次耗尽

| 属性 | 值 |
|------|-----|
| **严重性** | 低（自动恢复） |
| **问题** | 会话累积 171,563 tokens 后两次触发上下文压缩，中断 E2E 验证流程 |
| **根因** | 12 个任务的完整输出（文件读取、测试结果、审查报告）超出上下文窗口 |

### Issue #9: LSP 误报 24 次

| 属性 | 值 |
|------|-----|
| **严重性** | 低（噪音） |
| **问题** | gopls 和 tsserver 报告的错误全部为缓存滞后或工作目录错误导致的误报 |
| **根因** | (a) Go LSP 未在 subagent 写入后重新索引; (b) TS LSP 从 backend/ 目录运行; (c) LSP 缓存延迟 |
| **应对** | 每次均用 `go build` / `go test` / `npx tsc --noEmit` 替代 LSP 诊断验证 |

### Issue #10: 跳过 git worktree 步骤

| 属性 | 值 |
|------|-----|
| **严重性** | 低（流程偏差） |
| **问题** | subagent-driven-development 要求使用 git worktree 隔离，但被跳过 |
| **影响** | 无负面影响，但不符合规范流程 |

---

## 三、预先存在的问题（3 个）

| # | 问题 | 包 | 根因 |
|---|------|-----|------|
| 11 | `TestPaperConfigDefaults` 失败 | config | `.env` 中 `PAPER_GRPC_TIMEOUT=30m` 覆盖默认值 `5m` |
| 12 | MySQL 集成测试并发 flaky | repository, handler, service | 多包并发运行时 MySQL 连接池耗尽 |
| 13 | `ItemsPage.test.tsx` 部分测试失败 | frontend | 异步时序相关的预先存在问题 |

---

## 四、教训总结

### 教训 1: 接口变更必须全量搜索受影响测试

> **来源**: Bug #4
> **规则**: 修改任何 service 方法的依赖调用后，`grep` 所有测试文件中的构造函数调用（如 `NewFeedService`），确保每个 mock 包含新增的依赖。

### 教训 2: 前后端 API 契约必须显式对齐

> **来源**: Bug #5
> **规则**: 在 plan 中定义每个 API 的 request/response JSON 结构。后端 handler 和前端 hook 必须引用同一份契约（OpenAPI spec 或共享类型文件）。TypeScript 泛型不校验实际 JSON 结构。

### 教训 3: 每个状态变更都要考虑反向操作

> **来源**: Bug #6
> **规则**: 实现写操作（assign/move/set）时，必须同时问"用户如何撤销这个操作？"

### 教训 4: Repository 方法必须按 userID 隔离（defense-in-depth）

> **来源**: Bug #3
> **规则**: 所有数据访问方法在 SQL 层面都应包含 `WHERE user_id = ?`，即使 service 层已做权限检查。

### 教训 5: GORM FK tag 必须显式指定级联策略

> **来源**: Bug #2
> **规则**: 每个外键字段必须显式声明 `constraint:OnDelete` 和 `constraint:OnUpdate`。

### 教训 6: Subagent prompt 必须包含明确的 commit 指令

> **来源**: Issue #7
> **规则**: implementer subagent 的 prompt 中必须写明"测试通过后 commit 所有改动"。

---

## 五、统计

| 类别 | 数量 |
|------|------|
| 实现 Bug（已修复） | 6 |
| 流程/基础设施问题 | 4 |
| 预先存在问题 | 3 |
| **总计** | **13** |
| 严重 — Critical | 1 (Bug #5) |
| 严重 — High | 3 (Bug #1, #2, #3) |
| 严重 — High（测试崩溃） | 1 (Bug #4) |
| 严重 — Medium | 2 (Bug #6, Issue #7) |
| 严重 — Low | 6 (Issue #8-#13) |
| 上下文窗口耗尽次数 | 2 |
| LSP 误报次数 | 24 |
