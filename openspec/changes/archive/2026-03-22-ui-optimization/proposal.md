# UI Optimization Proposal

## Why

当前 oReader 使用两栏布局，文章详情需要整页跳转，阅读体验不流畅。同时，过滤器缺少计数显示，用户无法直观了解文章规模，也没有时间维度的快速筛选能力。

这个变更将：
1. 提升阅读效率：三栏并排，无需页面跳转
2. 增强信息可见性：所有过滤器显示对应数量
3. 支持时间维度：快速查看今天的新文章

## What Changes

- **三栏布局重构**: Sidebar (订阅源) | ItemList (文章列表) | ArticlePanel (正文)
- **新增 Stats API**: `GET /api/v1/stats` 返回 total/unread/starred/today 计数
- **新增 Today 过滤器**: 支持按"今天发布"筛选文章
- **响应式设计**: 渐进式降级 (桌面三栏 → 平板两栏 → 手机单栏)
- **URL 状态同步**: 选中文章时 URL 更新，支持分享链接

## Capabilities

### New Capabilities

- `stats-api`: 统计数据 API，返回用户的文章计数 (total/unread/starred/today)
- `today-filter`: Today 过滤器，支持筛选今天发布的文章
- `three-column-layout`: 三栏布局，Sidebar | ItemList | ArticlePanel 并排显示
- `article-panel`: ArticlePanel 组件，从 ItemViewPage 提取，显示文章正文和摘要

### Modified Capabilities

- `article-list`: 扩展支持 `published_today` 参数和选中状态高亮
- `sidebar-filters`: 扩展过滤器显示计数，新增 Today 过滤器

## Impact

### 后端 (Go)

| 文件 | 变更 |
|------|------|
| `internal/repository/stats_repository.go` | 新建 - 统计查询 |
| `internal/service/stats_service.go` | 新建 - 统计服务 |
| `internal/handler/stats_handler.go` | 新建 - HTTP handler |
| `internal/repository/item_repository.go` | 修改 - 添加 published_today 参数 |
| `internal/handler/item_handler.go` | 修改 - 解析 published_today |
| `cmd/server/main.go` | 修改 - 注册 /api/v1/stats 路由 |

### 前端 (React)

| 文件 | 变更 |
|------|------|
| `web/src/hooks/useStats.ts` | 新建 - Stats API hook |
| `web/src/components/items/ArticlePanel.tsx` | 新建 - 文章面板组件 |
| `web/src/pages/items/ItemsPage.tsx` | 重构 - 三栏布局 |
| `web/src/components/feed/Sidebar.tsx` | 修改 - 添加计数和 Today 过滤器 |
| `web/src/components/items/ItemList.tsx` | 修改 - 添加选中状态 |
| `web/src/types/feed.ts` | 修改 - 添加 StatsResponse 类型 |

### 测试

- 后端单元测试: `*_test.go` 文件
- 前端单元测试: `*.test.tsx` 文件
- 不使用 E2E 浏览器测试 (本地无桌面环境)

## References

- PRD: `design/PRD/ui-optimization.md`
- Design: `docs/superpowers/specs/2026-03-22-ui-optimization-design.md`
- 数据模型: `openspec/specs/data-model/spec.md`
