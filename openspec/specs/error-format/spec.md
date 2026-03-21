# API 错误格式规格

## 新增需求

### 需求：一致的错误响应结构
系统应以一致的 JSON 结构返回所有 API 错误。

#### 场景：错误响应格式
- **当** API 返回错误
- **则** 响应体遵循以下结构：
  ```json
  {
    "error": {
      "code": "ERROR_CODE",
      "message": "人类可读的消息",
      "details": {}
    }
  }
  ```
- **且** HTTP 状态码与错误类型匹配

### 需求：标准错误码
系统应对常见错误类型使用标准错误码。

#### 场景：错误码列表
| 错误码 | HTTP 状态 | 描述 |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | 输入验证失败 |
| `UNAUTHORIZED` | 401 | 需要认证 |
| `TOKEN_EXPIRED` | 401 | 访问令牌已过期 |
| `FORBIDDEN` | 403 | 权限拒绝 |
| `NOT_FOUND` | 404 | 资源未找到 |
| `CONFLICT` | 409 | 资源冲突（如重复） |
| `RATE_LIMIT_EXCEEDED` | 429 | 请求过多 |
| `INTERNAL_ERROR` | 500 | 服务器错误 |

### 需求：验证错误详情
系统应在验证错误中包含字段级别的详情。

#### 场景：字段验证错误
- **当** 输入验证失败
- **则** 错误响应包含字段名和验证消息
- **且** 响应格式：
  ```json
  {
    "error": {
      "code": "VALIDATION_ERROR",
      "message": "输入无效",
      "details": {
        "field": "email",
        "value": "invalid-email",
        "rule": "email_format"
      }
    }
  }
  ```

#### 场景：多个验证错误
- **当** 多个字段验证失败
- **则** details 包含所有错误的数组：
  ```json
  {
    "error": {
      "code": "VALIDATION_ERROR",
      "message": "多个验证错误",
      "details": {
        "errors": [
          {"field": "email", "rule": "required"},
          {"field": "password", "rule": "min_length", "value": 8}
        ]
      }
    }
  }
  ```

### 需求：令牌过期处理
系统应区分令牌过期与其他认证错误。

#### 场景：访问令牌过期
- **当** 访问令牌已过期
- **则** 错误码为 `TOKEN_EXPIRED`（而非 `UNAUTHORIZED`）
- **且** 响应：
  ```json
  {
    "error": {
      "code": "TOKEN_EXPIRED",
      "message": "访问令牌已过期"
    }
  }
  ```
- **且** 前端使用此信息触发自动刷新

### 需求：速率限制错误头
系统应在 429 响应中包含速率限制信息。

#### 场景：超出速率限制
- **当** 超出速率限制
- **则** 响应包含以下头信息：
  - `X-RateLimit-Limit`：时间窗口内最大请求数
  - `X-RateLimit-Remaining`：0
  - `X-RateLimit-Reset`：限制重置时的 Unix 时间戳
- **且** 响应体：
  ```json
  {
    "error": {
      "code": "RATE_LIMIT_EXCEEDED",
      "message": "请求过多",
      "details": {
        "retry_after": 60
      }
    }
  }
  ```

### 需求：内部错误处理
系统应对客户端隐藏内部错误详情。

#### 场景：内部服务器错误
- **当** 发生意外错误
- **则** 响应使用通用消息：
  ```json
  {
    "error": {
      "code": "INTERNAL_ERROR",
      "message": "发生意外错误"
    }
  }
  ```
- **且** 完整的错误详情在服务器端记录日志
- **且** 响应包含 `X-Request-ID` 用于支持查询
