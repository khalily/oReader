# CI Pipeline Repair — Bug 与问题复盘

> **日期**: 2026-04-06
> **功能**: 修复 oReader-paper 分支首次推送后的全部 CI 失败（3 个 job 失败 + 1 个被跳过的 Docker build 暴露问题）
> **变更规模**: 4 文件, +18/-8 行, 1 个 commit

---

## 一、实现 Bug（4 个）

### Bug #1: OpenAPI Route 测试缺少 6 个 categories 路由

| 属性 | 值 |
|------|-----|
| **任务** | 非本次引入 — categories 功能开发时遗漏 |
| **发现阶段** | CI (OpenAPI: Route & Contract Tests) |
| **严重性** | High（CI 红灯阻塞合并，且掩盖了路由一致性问题） |
| **问题** | `expectedBackendRoutes()` 硬编码了 34 个路由，遗漏了 6 个 categories 路由。测试报 "Routes in spec but NOT in backend (6)"，但实际 backend 的 `main.go` 中已正确注册 |
| **根因** | categories 功能添加时只更新了 OpenAPI spec 和 main.go 路由注册，但没有同步更新 `expectedBackendRoutes()` 这个静态对照表 |
| **修复** | `backend/internal/testutil/openapi_routes_test.go:136-142` — 添加 6 个 categories 路由 |

### Bug #2: TestPaperConfigDefaults 期望值过时

| 属性 | 值 |
|------|-----|
| **任务** | 非本次引入 — PAPER_GRPC_TIMEOUT 默认值变更时遗漏 |
| **发现阶段** | CI (Backend: Test) |
| **严重性** | Medium（测试假阳性，但不会导致功能问题） |
| **问题** | 测试期望 `cfg.Paper.GRPCTimeout == "5m"`，但 `config.go` 中默认值已改为 `"30m"` |
| **根因** | `v.SetDefault("PAPER_GRPC_TIMEOUT", "30m")` 更新了但测试没有同步修改 |
| **修复** | `backend/internal/config/config_test.go:163` — 期望值从 `"5m"` 改为 `"30m"` |

### Bug #3: category_service_test mock 返回错误类型不匹配

| 属性 | 值 |
|------|-----|
| **任务** | 非本次引入 — categories 功能开发时测试 mock 不正确 |
| **发现阶段** | CI (Backend: Test) |
| **严重性** | High（测试永远无法覆盖 duplicate name 分支，mock 绕过了真实的错误类型检查） |
| **问题** | `TestCreateCategory/duplicate_name_error` 和 `TestRenameCategory/duplicate_name_error_on_update` 的 mock 返回 `errors.New("duplicate entry ...")`，但 `isDuplicateKeyError()` 检查的是 `*mysql.MySQLError` with Number 1062 |
| **根因** | 编写 mock 时没有注意到 `isDuplicateKeyError` 使用 `errors.As()` 做类型断言，`errors.New()` 不会被匹配为 MySQL error |
| **修复** | `backend/internal/service/category_service_test.go:186,331` — mock 改为返回 `&mysql.MySQLError{Number: 1062, Message: "Duplicate entry"}`，并添加 `mysql "github.com/go-sql-driver/mysql"` import |

### Bug #4: govulncheck-action 与 monorepo 不兼容

| 属性 | 值 |
|------|-----|
| **任务** | 非本次引入 — CI 配置时未考虑 monorepo 结构 |
| **发现阶段** | CI (Security: Scan) |
| **严重性** | Medium（安全扫描无法运行，但不阻塞开发） |
| **问题** | `golang/govulncheck-action@v1` 的 `repo-path` 参数已被移除，且 action 内部的 setup-go 没有 `cache-dependency-path`，在 monorepo 根目录找不到 `go.sum` |
| **根因** | 该 action 内置的 setup-go 和外显的 setup-go 冲突，且 action 无法配置 `cache-dependency-path` |
| **修复** | `.github/workflows/ci.yml:329-334` — 替换为手动 `go install govulncheck` + `working-directory: backend` |

---

## 二、流程与基础设施问题（2 个）

| # | 问题 | 影响 |
|---|------|------|
| 1 | `expectedBackendRoutes()` 是硬编码静态表，新增路由极易遗漏同步 | 导致 OpenAPI 一致性测试无法保护新增路由的正确性 |
| 2 | Node.js 20 deprecation 警告遍布所有 job | 目前仅为 warning，2026-06-02 后将强制使用 Node.js 24 |

---

## 三、预先存在的问题（4 个）

| # | 问题 | 包/模块 | 根因 |
|---|------|---------|------|
| 1 | Docker: Build frontend `npm run build` exit code 2 | `frontend/Dockerfile` | 前端 TypeScript 编译错误（被前置 CI 失败掩盖，本次修复后才暴露） |
| 2 | Backend Dockerfile `InvalidDefaultArgInFrom` warning | `backend/Dockerfile:4,23` | ARG 默认值 `golang:${GO_VERSION}-alpine` 和 `alpine:${ALPINE_VERSION}` 在 Docker buildx 校验中视为无效（变量未展开） |
| 3 | Frontend ESLint warnings — `useMemo` 和 `useEffect` 依赖 | `ItemsPage.tsx:95`, `ArticlePanel.tsx:78` | items 逻辑表达式导致 useEffect 依赖不稳定；toggleRead 缺少依赖 |
| 4 | 前置 CI 失败掩盖了 Docker build 问题 | CI pipeline | docker-build 依赖 openapi-test 和 backend-test，它们失败时 docker-build 被跳过 |

---

## 四、教训总结

### 教训 1: 硬编码路由表必须在 PR review 中显式检查

> **来源**: Bug #1
> **规则**: 在 `main.go` 中新增或删除路由时，必须同步更新 `expectedBackendRoutes()`，PR review 时 grep 两者的一致性。长远考虑可改为运行时从 Gin router 提取路由列表

### 教训 2: 修改配置默认值后必须搜索所有测试引用

> **来源**: Bug #2
> **规则**: 当 `v.SetDefault()` 的值变更时，grep 测试文件中对该值的硬编码期望。`PAPER_GRPC_TIMEOUT` 从 `"5m"` 改为 `"30m"` 时测试没有同步

### 教训 3: Mock 必须匹配真实的错误类型检查逻辑

> **来源**: Bug #3
> **规则**: 当被测代码使用 `errors.As()` 做类型断言时（如 `isDuplicateKeyError` 检查 `*mysql.MySQLError`），mock 必须返回相同类型的 error。`errors.New()` 无法通过 `errors.As(&mysql.MySQLError{})` 的匹配

### 教训 4: 第三方 GitHub Action 在 monorepo 中需要仔细验证参数兼容性

> **来源**: Bug #4
> **规则**: 使用第三方 action 时，检查其内部是否嵌套了 setup-go/setup-node 等步骤，避免和外显步骤冲突。对于 monorepo，优先选择支持 `working-directory` 或 `cache-dependency-path` 的 action，或直接用 `run` 替代

---

## 五、统计

| 类别 | 数量 |
|------|------|
| 实现 Bug（已修复） | 4 |
| 流程/基础设施问题 | 2 |
| 预先存在问题 | 4 |
| **总计** | **10** |
