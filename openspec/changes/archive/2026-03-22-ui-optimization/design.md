# UI Optimization Design

## Context

oReader 是一个 RSS 阅读器，后端使用 Go + Gin + SQLite，前端使用 React + TypeScript + Tailwind CSS + shadcn/ui。

### Current State

```
当前布局:
┌──────────────┬────────────────────────────────────┐
│   Sidebar    │           Main Content             │
│   (256px)    │            (flex-1)                │
│              │                                    │
│  • Filters   │  ItemsPage OR ItemViewPage         │
│  • Feeds     │  (互斥页面，点击文章跳转)            │
└──────────────┴────────────────────────────────────┘
```

### Constraints

- 保持现有认证、RSS 解析、数据库模型不变
- 响应式设计必须支持移动端
- 不使用 E2E 浏览器测试 (无桌面环境)
- 使用 TDD 流程：测试先行

## Goals / Non-Goals

**Goals:**
- 三栏布局：Sidebar | ItemList | ArticlePanel 并排显示
- 所有过滤器显示计数 (All/Unread/Starred/Today)
- 新增 Today 过滤器，支持时间维度筛选
- URL 状态同步，支持分享链接
- 渐进式响应式降级

**Non-Goals:**
- AI 摘要功能 (后续迭代)
- Feed 文件夹分组 (需要数据模型变更)
- 离线阅读 (需要 PWA 集成)
- 重构状态管理库 (保持 Zustand + React Query)

## Decisions

### D1: 三栏布局实现方式

**Decision**: 使用 CSS Flexbox + Tailwind 响应式类

**Rationale**:
- 项目已使用 Tailwind CSS，无需引入新依赖
- Flexbox 足以处理三栏布局
- 避免过度工程化

**Alternatives Considered**:
- CSS Grid: 功能更强但增加复杂度，Flexbox 足够
- 独立 Layout 组件: 增加抽象层，当前规模不需要

### D2: Stats API 设计

**Decision**: 新建独立端点 `GET /api/v1/stats`

**Rationale**:
- 单次请求返回所有计数，避免 N+1
- 与现有 feeds 端点解耦
- 便于独立缓存和测试

**Response Format**:
```json
{
  "total": 156,
  "unread": 42,
  "starred": 8,
  "today": 12
}
```

### D3: Today 过滤实现

**Decision**: 在 ListItems API 添加 `published_today` 参数

**Rationale**:
- 复用现有 API 结构
- 与现有 `starred`、`read` 参数模式一致
- 后端使用 UTC 时间截断计算今天

**SQL Logic**:
```sql
WHERE i.pub_date >= DATE('now', 'start of day')
```

### D4: ArticlePanel 组件

**Decision**: 从 ItemViewPage 提取内容部分作为独立组件

**Rationale**:
- 复用现有渲染逻辑
- 支持三栏和独立页面两种使用场景
- 保持 ItemViewPage 用于分享链接

### D5: URL 状态管理

**Decision**: 使用 `?filter=&feed=&id=` 查询参数

**Rationale**:
- 保持 URL 可分享
- 支持浏览器历史导航
- 不需要复杂的状态管理库

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Stats API 性能 (大量文章) | 添加数据库索引 `(user_id, pub_date)`；考虑 30s staleTime |
| Today 时区问题 | 统一使用服务器 UTC 时间，前端不传时区 |
| 三栏布局移动端体验 | 渐进式降级：lg:三栏 → md:两栏 → sm:单栏+抽屉 |
| 计数不一致 (star 后) | 操作成功后 invalidate stats query |
| ArticlePanel 重复请求 | React Query 自动缓存，相同 itemId 不重复请求 |

## Migration Plan

### Phase 1: 后端 API (无依赖)

1. 创建 `stats_repository.go` - 数据库查询
2. 创建 `stats_service.go` - 业务逻辑
3. 创建 `stats_handler.go` - HTTP handler
4. 修改 `item_repository.go` - 添加 published_today
5. 修改 `item_handler.go` - 解析参数
6. 注册路由 `/api/v1/stats`
7. 编写后端单元测试

### Phase 2: 前端基础 (依赖 Phase 1)

1. 添加 `StatsResponse` 类型
2. 创建 `useStats.ts` hook
3. 创建 `ArticlePanel.tsx` 组件
4. 编写前端单元测试

### Phase 3: UI 整合 (依赖 Phase 2)

1. 重构 `ItemsPage.tsx` 三栏布局
2. 修改 `Sidebar.tsx` 添加计数和 Today
3. 修改 `ItemList.tsx` 添加选中状态
4. 添加响应式样式

### Phase 4: 收尾

1. 更新键盘快捷键
2. 手动验证清单

### Rollback

- 后端：删除 stats 相关文件，移除路由
- 前端：恢复 ItemsPage 原有布局

## Open Questions

1. ~~是否需要 AI 摘要？~~ → 已确认暂不做
2. ~~Today 定义用 pub_date 还是 created_at？~~ → 已确认用 pub_date
3. ~~E2E 测试方案？~~ → 已确认不用浏览器测试，用 API + MSW mock
