# 侧边栏优化设计文档

**日期**: 2026-04-05
**状态**: 已批准

## 目标

优化前端界面布局，移除冗余导航元素，将 Papers 和 Feeds 统一为三栏阅读体验，支持分类管理。

## 核心变更

### 1. 移除的元素

- **移除 all/unread/starred/today 过滤页面** — 不再需要独立的筛选视图
- **移除顶部工具栏** — 所有操作收纳到侧边栏头部
- **移除侧边栏中的 Feed 添加/导入/导出按钮** — 合并为统一的"新增"按钮

### 2. 三栏布局（无工具栏）

完全三栏布局，最大化内容阅读空间：

| 列 | 宽度 | 内容 |
|---|---|---|
| 侧边栏 | 260px | 分类导航 + Feeds + Papers |
| 文章列表 | 320px | 当前来源的文章列表 |
| 阅读区 | 弹性 | 文章/Paper 全文阅读 |

参考：Folo.is (RSSNext/Folo) 界面风格。

### 3. 侧边栏头部

侧边栏顶部一行布局：

```
[● oReader]          [+ 新增] [W]
  Logo                  按钮   头像
```

- **Logo** — 左侧，品牌标识
- **新增按钮** — 右侧，点击弹出统一对话框
- **用户头像** — 最右侧，圆形头像（取用户名首字母，渐变背景）

### 4. "新增"对话框

点击"新增"按钮弹出对话框，提供三个选项：

| 选项 | 操作 |
|---|---|
| 添加 Feed | 输入 RSS/Atom URL 订阅 |
| 导入 Feed | 从 OPML 文件批量导入 |
| 导入 Paper | 上传 PDF 论文并自动解析 |

### 5. 侧边栏分区

侧边栏内容分为上下两个区域，用分割线隔开：

```
┌──────────────────┐
│  Feeds           │
│  ├── Tech Blogs  │  ← 展开
│  │   ├── HN      │
│  │   ├── TC      │
│  │   └── V2      │
│  ├── News        │  ← 折叠
│  └── (未分类)    │
├──────────────────┤
│  Papers          │
│  ├── ML          │  ← 展开
│  │   ├── Attention│
│  │   └── BERT    │
│  ├── NLP         │  ← 折叠
│  └── (未分类)    │
└──────────────────┘
```

- **Feeds 在上，Papers 在下** — RSS 日常浏览优先
- 分类标题显示计数 badge
- 未分类的 Feed/Paper 平铺在各自区域的最下方
- 每个 section header 显示总数

### 6. 分类管理

#### 数据模型

新建统一的 `categories` 表：

```sql
CREATE TABLE categories (
    id         BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id    BIGINT NOT NULL,
    name       VARCHAR(100) NOT NULL,
    type       ENUM('feed', 'paper') NOT NULL,
    position   INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_type (user_id, type)
);
```

- 单层分类，不支持嵌套
- `type` 字段区分 Feed 分类和 Paper 分类
- `position` 字段支持拖拽排序

关联表：

```sql
-- Feed 与分类的多对多关系
ALTER TABLE user_feeds ADD COLUMN category_id BIGINT NULL
    REFERENCES categories(id) ON DELETE SET NULL;

-- Paper 与分类的多对多关系
ALTER TABLE papers ADD COLUMN category_id BIGINT NULL
    REFERENCES categories(id) ON DELETE SET NULL;
```

设计为单分类（一个 Feed/Paper 只属于一个分类或未分类），而非多对多。理由：
- 操作更直观（右键"添加到分类"是移动，不是追加）
- 减少复杂度，分类管理更简单

#### 右键上下文菜单

**右键分类行**：
- 重命名 — 弹出内联输入框修改分类名称
- 删除 — 确认后删除分类，其中的 Feed/Paper 变为未分类

**右键 Feed/Paper 行**：
- 添加到分类 — 弹出分类选择子菜单
- 添加并新建分类 — 创建新分类并自动归入

### 7. 路由变更

移除 `/items` 的 all/unread/starred/today 子路由，简化为：

| 路由 | 页面 |
|---|---|
| `/` | 重定向到 `/feeds` |
| `/feeds` | 统一三栏阅读页（Feeds + Papers） |
| `/feeds/:id` | 文章阅读（三栏选中状态） |
| `/papers/:id` | Paper 阅读（三栏选中状态） |

侧边栏导航通过 Zustand store 管理选中状态（`selectedFeedId` / `selectedPaperId`），驱动三栏布局的内容切换，无需路由变化。

- 点击 Feed → 设置 `selectedFeedId`，中间列显示该 Feed 的文章列表
- 点击 Paper → 设置 `selectedPaperId`，中间列显示同分类的其他 Paper，右侧显示全文
- 点击分类 → 展开/折叠该分类下的子项（`expandedCategoryIds` set）

### 8. 移除的页面/组件

| 组件 | 处置 |
|---|---|
| `all/unread/starred/today` 过滤 | 移除 |
| 侧边栏 Feed 添加/导入/导出按钮 | 移除，合并到"新增"对话框 |
| 顶部工具栏 | 移除 |
| `/papers` 独立管理页面 | 合并到统一阅读页 |

## 不在范围内

- 拖拽排序分类（后续迭代）
- AI 功能（Folo 的右侧 AI 面板）
- 多选/批量操作
- 移动端适配
