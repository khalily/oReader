## ADDED Requirements

### Requirement: UserItemStateRepository 测试覆盖
UserItemStateRepository SHALL 为所有公共方法提供单元测试覆盖，特别是复杂事务逻辑。

#### Scenario: BulkMarkRead 空列表
- **WHEN** 使用空 itemIDs 列表调用 BulkMarkRead
- **THEN** 方法 SHALL 立即返回 nil，不执行任何数据库操作

#### Scenario: BulkMarkRead 混合状态
- **WHEN** 部分文章已有 UserItemState 记录，部分没有
- **THEN** 方法 SHALL 更新现有记录的 is_read 为 true
- **AND** 为没有记录的文章创建新的 UserItemState 记录

#### Scenario: MarkAllRead 事务完整性
- **WHEN** 调用 MarkAllRead 标记某 Feed 的所有文章为已读
- **THEN** 所有操作 SHALL 在单个事务中完成
- **AND** 如果任何操作失败，整个事务 SHALL 回滚

### Requirement: UserFeedRepository 测试覆盖
UserFeedRepository SHALL 为所有公共方法提供单元测试覆盖。

#### Scenario: GetByUserAndFeedIncludingDeleted
- **WHEN** 查询已软删除的 UserFeed 记录
- **THEN** 方法 SHALL 返回包括已删除的记录

#### Scenario: GetMaxPosition 空用户
- **WHEN** 用户没有任何订阅
- **THEN** GetMaxPosition SHALL 返回 0

### Requirement: ImportJobRepository 测试覆盖
ImportJobRepository SHALL 为所有公共方法提供单元测试覆盖。

#### Scenario: 创建导入任务
- **WHEN** 创建新的 ImportJob 记录
- **THEN** 记录 SHALL 包含正确的 UserID, TotalFeeds, Status 字段

#### Scenario: 查询不存在的任务
- **WHEN** 使用不存在的 JobID 查询
- **THEN** 方法 SHALL 返回 gorm.ErrRecordNotFound 错误
