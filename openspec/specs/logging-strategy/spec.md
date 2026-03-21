# 日志策略规格

## 新增需求

### 需求：结构化 JSON 日志
系统应对所有应用日志使用结构化 JSON 日志。

#### 场景：日志格式
- **当** 写入任何日志消息
- **则** 日志格式为 JSON，包含 timestamp、level、message 和 context 字段
- **且** timestamp 使用带时区的 ISO 8601 格式

#### 场景：日志级别
- **当** 应用记录事件
- **则** 系统使用标准级别：debug、info、warn、error
- **且** 级别可通过 LOG_LEVEL 环境变量配置

### 需求：请求上下文日志
系统应在所有处理程序日志中包含请求上下文。

#### 场景：请求日志
- **当** 处理 HTTP 请求
- **则** 日志包含 request_id、user_id（如已认证）、method、path、status、latency
- **且** request_id 每个请求唯一，并在响应头中返回

#### 场景：错误日志
- **当** 请求处理期间发生错误
- **则** 日志包含错误消息、堆栈跟踪（开发环境）和请求上下文
- **且** 敏感数据（密码、令牌）永不记录

### 需求：订阅源刷新操作日志
系统应记录带有相关指标的订阅源刷新操作。

#### 场景：成功刷新
- **当** 订阅源成功刷新
- **则** 日志包含 feed_id、items_added、duration、source_url

#### 场景：刷新失败
- **当** 订阅源刷新失败
- **则** 日志包含 feed_id、error_type、error_message、duration
- **且** 记录连续失败计数

### 需求：认证事件日志
系统应为安全审计记录认证事件。

#### 场景：登录尝试
- **当** 用户尝试登录
- **则** 日志包含 event_type=login、email、success、ip_address、user_agent

#### 场景：令牌刷新
- **当** 令牌被刷新
- **则** 日志包含 event_type=token_refresh、user_id、success

#### 场景：登出
- **当** 用户登出
- **则** 日志包含 event_type=logout、user_id

### 需求：开发与生产日志
系统应根据环境调整日志格式。

#### 场景：开发模式
- **当** ENV=development
- **则** 日志使用人类可读的控制台格式，带颜色
- **且** 默认启用 debug 级别

#### 场景：生产模式
- **当** ENV=production
- **则** 日志使用 JSON 格式
- **且** 默认为 info 级别
- **且** 无颜色或额外格式化
