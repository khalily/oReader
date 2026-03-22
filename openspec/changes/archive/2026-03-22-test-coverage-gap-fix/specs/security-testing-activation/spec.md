## ADDED Requirements

### Requirement: SQL 注入测试
认证相关 Handler SHALL 抵御 SQL 注入攻击，使用参数化查询或 ORM 方法。

#### Scenario: 登录接口 SQL 注入
- **WHEN** 攻击者在 email 字段注入 SQL 语句（如 `' OR '1'='1`）
- **THEN** 系统 SHALL 拒绝该请求或返回安全结果
- **AND** 数据库 SHALL NOT 执行恶意 SQL

#### Scenario: Feed URL 参数注入
- **WHEN** 攻击者在 feed URL 中注入 SQL 语句
- **THEN** 系统 SHALL 拒绝该请求或安全处理
- **AND** 数据库完整性 SHALL 保持不变

### Requirement: XSS 攻击防护测试
系统 SHALL 对所有用户输入进行 HTML 转义，防止 XSS 攻击。

#### Scenario: Feed 标题 XSS 注入
- **WHEN** RSS feed 包含 `<script>` 标签的标题
- **THEN** 系统 SHALL 在输出时转义 HTML 标签
- **AND** 响应 SHALL NOT 包含未转义的 `<script>` 标签

#### Scenario: 文章内容 XSS 注入
- **WHEN** 文章内容包含恶意 JavaScript 代码
- **THEN** 系统 SHALL 净化内容（使用 sanitize 模块）
- **AND** 危险的 HTML 标签和属性 SHALL 被移除

### Requirement: 认证授权边界测试
系统 SHALL 正确处理认证和授权边界情况。

#### Scenario: 无 Token 访问受保护资源
- **WHEN** 请求不包含 Authorization header
- **THEN** 系统 SHALL 返回 401 Unauthorized

#### Scenario: 过期 Token 访问
- **WHEN** 请求使用已过期的 access token
- **THEN** 系统 SHALL 返回 401 Unauthorized
- **AND** 响应 SHALL 包含刷新 token 的提示

#### Scenario: 跨用户资源访问
- **WHEN** 用户 A 尝试访问用户 B 的 Feed
- **THEN** 系统 SHALL 返回 403 Forbidden
