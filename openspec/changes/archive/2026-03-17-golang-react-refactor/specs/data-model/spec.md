# 数据模型规格

## 新增需求

### 需求：多租户订阅源共享
系统应允许多个用户订阅同一个 RSS 订阅源 URL 而不重复内容。

#### 场景：共享订阅源订阅
- **当** 用户 A 和用户 B 都订阅 `https://example.com/feed.xml`
- **则** 只存在一条 `feed_url = https://example.com/feed.xml` 的 Feed 记录
- **且** 该订阅源只存在一组 Item 记录
- **且** 用户 A 和用户 B 各自拥有指向此共享 Feed 的 UserFeed 记录

#### 场景：独立的用户状态
- **当** 用户 A 将一篇文章标记为已读
- **则** 用户 B 对同一篇文章的视图仍为未读
- **且** UserItemState 记录按用户独立

### 需求：用户特定的文章状态
系统应按用户而非按文章存储 `is_starred` 和 `is_read` 状态。

#### 场景：收藏文章
- **当** 已认证用户将文章标记为收藏
- **则** 系统为 (user_id, item_id) 创建或更新 UserItemState 记录
- **且** 其他用户对此文章的状态不受影响

#### 场景：已读文章
- **当** 已认证用户将文章标记为已读
- **则** 系统创建或更新 UserItemState 记录，设置 is_read=true, read_at=now
- **且** 其他用户对此文章的状态不受影响

### 需求：按 GUID 去重文章
系统应使用 RSS 文章 GUID 防止重复文章。

#### 场景：新 GUID 的新文章
- **当** RSS 订阅源包含数据库中不存在的 GUID 文章
- **则** 系统创建新的 Item 记录

#### 场景：已存在的文章 GUID
- **当** RSS 订阅源包含数据库中已存在的 GUID 文章
- **则** 系统更新现有 Item 记录（如内容已变更）
- **且** 不创建重复的 Item

## 数据模型定义

### 数据表

#### users
| 字段 | 类型 | 描述 |
|--------|------|-------------|
| id | UUID v7 | 主键 |
| email | VARCHAR(255) | 唯一，非空 |
| password_hash | VARCHAR(255) | 可空，用于 OAuth 用户 |
| nickname | VARCHAR(100) | 显示名称 |
| avatar_url | VARCHAR(500) | 可空 |
| auth_provider | VARCHAR(20) | 'local' 或 'github' |
| github_id | VARCHAR(50) | 可空，唯一 |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP | |

#### feeds（共享的 RSS 订阅源元数据）
| 字段 | 类型 | 描述 |
|--------|------|-------------|
| id | UUID v7 | 主键 |
| feed_url | VARCHAR(500) | 唯一，RSS/Atom URL |
| title | VARCHAR(255) | 订阅源标题 |
| description | TEXT | 可空 |
| image_url | VARCHAR(500) | 可空，订阅源图片/favicon |
| site_url | VARCHAR(500) | 可空，网站链接 |
| last_fetched_at | TIMESTAMP | 可空 |
| last_fetch_status | VARCHAR(20) | 'success'、'error'、'timeout' |
| last_fetch_error | TEXT | 可空，错误信息 |
| consecutive_failures | INTEGER | 默认 0 |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP | |

#### user_feeds（用户订阅关系）
| 字段 | 类型 | 描述 |
|--------|------|-------------|
| id | UUID v7 | 主键 |
| user_id | UUID v7 | 外键 -> users.id |
| feed_id | UUID v7 | 外键 -> feeds.id |
| position | INTEGER | 默认 0，用于排序 |
| created_at | TIMESTAMP | |

**唯一约束**: (user_id, feed_id)

#### items（共享的文章内容）
| 字段 | 类型 | 描述 |
|--------|------|-------------|
| id | UUID v7 | 主键 |
| feed_id | UUID v7 | 外键 -> feeds.id |
| guid | VARCHAR(500) | 来自 RSS 的唯一标识符 |
| title | VARCHAR(255) | |
| link | VARCHAR(500) | 文章 URL |
| description | TEXT | 可空，简短摘要 |
| content | TEXT | 可空，完整内容 |
| creator | VARCHAR(255) | 可空，作者名称 |
| pub_date | TIMESTAMP | 可空 |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP | |

**唯一约束**: (feed_id, guid)
**索引**: feed_id, pub_date

#### user_item_states（按用户的文章状态）
| 字段 | 类型 | 描述 |
|--------|------|-------------|
| id | UUID v7 | 主键 |
| user_id | UUID v7 | 外键 -> users.id |
| item_id | UUID v7 | 外键 -> items.id |
| is_starred | BOOLEAN | 默认 false |
| is_read | BOOLEAN | 默认 false |
| read_at | TIMESTAMP | 可空 |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP | |

**唯一约束**: (user_id, item_id)
**索引**: user_id, is_starred, is_read

## 关系

```
User ──1:N──▶ UserFeed ──N:1──▶ Feed ──1:N──▶ Item ──1:1──▶ UserItemState
  │                                                      ▲
  └──────────────────────────────────────────────────────┘
```

## 迁移注意事项

### 从 Flask（当前）到 Go（新版）

当前 Flask 实现具有：
- Feed.user_id（每个用户有自己的 Feed 副本）
- Item.feed_id（文章属于用户特定的订阅源）
- Item.star（直接存储在 Item 上）

这意味着：
1. 同一 RSS URL 被多次存储（每个用户一次）
2. 同一文章内容被多次存储
3. 用户状态隔离但以数据重复为代价

### 迁移路径

1. **导出**：使用 Flask 版本的 OPML 导出功能
2. **导入**：导入到新的 Go 版本
3. **状态**：收藏的文章无法迁移（ID 方案不同）

### 新模型的优势

| 方面 | 当前 (Flask) | 新版 (Go) |
|--------|-----------------|----------|
| 存储 | 每用户重复 | 共享，最小重复 |
| 用户状态 | 按文章（可用但低效） | 显式关联表 |
| 订阅源更新 | 每个用户的订阅源单独刷新 | 单次刷新更新所有订阅者 |
| 可扩展性 | 随用户线性增长 | 亚线性增长（共享订阅源） |
