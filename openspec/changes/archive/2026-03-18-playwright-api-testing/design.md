## 背景

oReader 后端是一个 Go/Gin REST API，使用基于 cookie 的认证和 CSRF 保护。目前仅存在针对单个组件的单元测试。我们需要端到端 API 测试来验证完整的请求/响应周期，包括认证流程、CSRF 令牌处理和数据库持久化。

关键挑战是正确处理认证流程：
1. Login/Register 设置 `access_token`（HttpOnly）和 `csrf_token`（可读）cookies
2. CSRF 令牌必须从响应体中提取，并在 POST/PUT/DELETE/PATCH 请求中作为 `X-CSRF-Token` 头发送
3. 测试需要隔离的数据库状态以避免运行间污染

## 目标 / 非目标

**目标：**
- 为所有 REST 端点提供全面的 E2E API 测试覆盖
- 测试中正确处理认证和 CSRF 令牌
- 测试运行间重置的隔离测试数据库
- 自动化测试-修复循环以发现并修复 bug
- 无头执行以兼容 CI/CD

**非目标：**
- 前端 UI 测试（仅 API）
- OAuth/GitHub 认证测试（跳过）
- 性能/负载测试
- 视觉回归测试

## 决策

### D1：使用 Playwright 进行 API 测试
**选择：** Playwright（而非直接使用 Jest/Vitest）
**理由：**
- 内置 cookie 管理和自动 cookie 处理
- 与前端技术栈匹配的优秀 TypeScript 支持
- 如需要可扩展到 UI 测试
- 比传统测试运行器更好的 async/await 处理

**考虑的替代方案：**
- Vitest + supertest：更轻量但需要手动 cookie管理
- Go testing：可行但团队偏好 TypeScript 进行 E2E 测试

### D2：独立的测试数据库
**选择：** SQLite 文件 `oreader_test.db`，运行间清理
**理由：**
- 无需外部数据库依赖
- 快速测试执行
- 与开发/生产数据完全隔离

**实现：**
- 在测试环境中设置 `DATABASE_URL=oreader_test.db`
- 每次测试运行前删除并重建数据库
- Go 自动迁移处理模式创建

### D3：绕过速率限制
**选择：** 通过 `RATE_LIMIT_ENABLED=false` 禁用速率限制
**理由：**
- 测试运行快速，无人工延迟
- 速率限制在单元测试中测试
- 避免因速率限制命中导致的测试不稳定

### D4：RSS 订阅源模拟策略
**选择：** 使用真实的 OpenAI News RSS 订阅源
**理由：**
- 测试真实世界的 RSS 解析行为
- OpenAI RSS 稳定且公开可用
- 如需要可回退到 BBC News RSS

**考虑的替代方案：**
- 模拟 RSS 服务器：更多控制但增加复杂性
- 本地 RSS 文件：不测试实际 HTTP 获取

### D5：测试结构
**选择：** 按 API 领域组织（auth、feeds、items、opml）
**理由：**
- 反映 API 结构
- 更易查找和维护测试
- 清晰的关注点分离

```
web/tests/e2e/
├── playwright.config.ts     # 配置
├── fixtures/
│   └── auth.ts              # 认证辅助和夹具
├── auth.spec.ts             # 认证测试
├── feeds.spec.ts            # 订阅源管理测试
├── items.spec.ts            # 文章管理测试
└── opml.spec.ts             # OPML 导入/导出测试
```

## 风险 / 权衡

| 风险 | 缓解措施 |
|------|----------|
| 外部 RSS 订阅源不可用 | 回退到 BBC News RSS；添加重试逻辑 |
| 测试数据库冲突 | 每次测试运行使用唯一数据库文件；拆卸时清理 |
| CSRF 令牌时机问题 | 登录后立即从响应体提取令牌 |
| 异步操作导致测试不稳定 | 使用 Playwright 内置等待和重试 |
| 速率限制意外启用 | 在测试设置中验证 `RATE_LIMIT_ENABLED=false` |
