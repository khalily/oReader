## 新增需求

### 需求：文章响应包含嵌套的 user_state 对象
系统应在嵌套的 `user_state` 对象中返回用户特定状态（is_starred、is_read），而非文章上的扁平字段。

#### 场景：带用户状态的文章
- **当** 用户请求任何文章端点（list、get、star、read）
- **则** 系统返回包含 `user_state` 对象的文章，其中有 `item_id`、`is_starred`、`is_read`、`read_at`
- **且** 如用户-文章对无状态，`user_state` 为 `null`

#### 场景：用户状态默认值
- **当** 用户查看从未交互过的文章
- **则** `user_state` 为 `null`
- **且** 前端将 null 视为 `{ is_starred: false, is_read: false }`

### 需求：文章响应包含嵌套的 feed 对象
系统应在嵌套的 `feed` 对象中返回订阅源信息，而非 `feed_title` 字符串。

#### 场景：Feed 对象结构
- **当** 用户请求任何文章端点
- **则** 系统返回包含 `feed` 对象的文章，其中有 `id`、`title`、`feed_url`、`description`、`image_url`
- **且** `feed` 对象匹配 Feed 数据模型

### 需求：响应结构一致性
所有文章端点应返回一致的响应结构。

#### 场景：列出文章响应
- **当** 用户调用 GET /api/v1/items
- **则** 响应中每篇文章包含嵌套的 `user_state` 和 `feed` 对象

#### 场景：获取文章响应
- **当** 用户调用 GET /api/v1/items/:id
- **则** 响应包含嵌套的 `user_state` 和 `feed` 对象

#### 场景：收藏文章响应
- **当** 用户调用 PUT /api/v1/items/:id/star
- **则** 响应包含更新后的文章，带有嵌套的 `user_state` 和 `feed` 对象

#### 场景：已读文章响应
- **当** 用户调用 PUT /api/v1/items/:id/read
- **则** 响应包含更新后的文章，带有嵌套的 `user_state` 和 `feed` 对象

## 修改的需求

### 需求：收藏/取消收藏文章
系统应允许已认证用户将文章标记为收藏。

#### 场景：收藏文章
- **当** 已认证用户调用 PUT /api/v1/items/:id/star，请求体为 { starred: true }
- **则** 系统在 UserItemState 中设置 is_starred = true
- **且** 系统返回更新后的文章，嵌套 `user_state` 对象显示 `is_starred: true`

#### 场景：取消收藏文章
- **当** 已认证用户调用 PUT /api/v1/items/:id/star，请求体为 { starred: false }
- **则** 系统在 UserItemState 中设置 is_starred = false
- **且** 系统返回更新后的文章，嵌套 `user_state` 对象显示 `is_starred: false`

### 需求：标记文章为已读/未读
系统应允许已认证用户将文章标记为已读。

#### 场景：标记为已读
- **当** 已认证用户调用 PUT /api/v1/items/:id/read，请求体为 { read: true }
- **则** 系统在 UserItemState 中设置 is_read = true
- **且** 系统将 read_at 设置为当前时间戳
- **且** 系统返回更新后的文章，嵌套 `user_state` 对象显示 `is_read: true`

#### 场景：标记为未读
- **当** 已认证用户调用 PUT /api/v1/items/:id/read，请求体为 { read: false }
- **则** 系统在 UserItemState 中设置 is_read = false
- **且** 系统将 read_at 设置为 null
- **且** 系统返回更新后的文章，嵌套 `user_state` 对象显示 `is_read: false`
