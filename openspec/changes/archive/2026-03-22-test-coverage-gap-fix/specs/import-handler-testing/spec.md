## ADDED Requirements

### Requirement: Import Handler 文件上传验证
ImportFeeds Handler SHALL 验证所有文件上传请求，拒绝无效或危险的输入。

#### Scenario: 无文件上传
- **WHEN** 请求不包含 multipart form file
- **THEN** Handler SHALL 返回 400 Bad Request
- **AND** 错误信息为 "No file uploaded"

#### Scenario: 文件大小超限
- **WHEN** 上传文件大小超过 1MB
- **THEN** Handler SHALL 返回 413 Request Entity Too Large
- **AND** 错误信息为 "File too large (max 1MB)"

#### Scenario: 无效 OPML 格式
- **WHEN** 上传的文件不是有效的 OPML XML
- **THEN** Handler SHALL 返回 400 Bad Request
- **AND** 错误信息为 "Invalid OPML format"

#### Scenario: 空 OPML 文件
- **WHEN** OPML 文件不包含任何 feed 订阅
- **THEN** Handler SHALL 返回 200 OK
- **AND** 响应包含 message: "No feeds found in OPML file"

### Requirement: Import Handler 授权检查
Import Handler SHALL 验证用户对导入任务的访问权限。

#### Scenario: 获取任务状态未授权
- **WHEN** 用户尝试获取不属于自己的导入任务状态
- **THEN** Handler SHALL 返回 403 Forbidden
- **AND** 错误信息为 "Access denied"

#### Scenario: 任务不存在
- **WHEN** 查询不存在的 JobID
- **THEN** Handler SHALL 返回 404 Not Found
- **AND** 错误信息为 "Import job not found"

### Requirement: Export Handler 测试
ExportFeeds Handler SHALL 正确导出用户的订阅列表。

#### Scenario: 成功导出
- **WHEN** 已认证用户请求导出
- **THEN** Handler SHALL 返回 200 OK
- **AND** Content-Type 为 "application/xml"
- **AND** Content-Disposition 包含 "oreader-subscriptions.xml"

#### Scenario: 未授权导出
- **WHEN** 未认证用户请求导出
- **THEN** Handler SHALL 返回 401 Unauthorized
