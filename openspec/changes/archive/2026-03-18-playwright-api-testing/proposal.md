## 背景

oReader API 目前缺乏自动化端到端测试，难以捕获回归问题并验证所有端点是否正确协同工作。我们需要基于 Playwright 的全面 API 测试，在无头模式下运行，以确保 API 可靠性并在部署前发现 bug。

## 变更内容

- 在 `tests/e2e/` 目录添加 Playwright 测试基础设施
- 为所有端点创建全面的 API 测试套件：
  - 认证 API（register、login、refresh、logout、me）
  - 订阅源管理 API（CRUD、refresh、mark-all-read）
  - 文章管理 API（list、get、star、read）
  - OPML 导入/导出 API
- 使用独立的 SQLite 文件实现测试数据库隔离
- 添加绕过速率限制的测试环境配置
- 修复测试期间发现的任何 bug（测试-修复循环）

## 能力

### 新增能力

- `e2e-api-testing`：全面的基于 Playwright 的端到端 API 测试基础设施，覆盖所有 oReader REST 端点，具有适当的认证、CSRF 处理和测试数据库隔离

### 修改的能力

- `user-auth`：登录端点响应将被修改以在响应体中包含 `csrf_token`（目前缺失，导致与注册端点不一致）

## 影响

- **新增文件**：
  - `web/tests/e2e/` - Playwright 测试目录
  - `web/tests/e2e/playwright.config.ts` - Playwright 配置
  - `web/tests/e2e/fixtures/auth.ts` - 认证测试夹具
  - `web/tests/e2e/*.spec.ts` - 各 API 领域的测试套件
  - `.env.test` - 测试环境配置

- **修改文件**：
  - `web/package.json` - 添加 Playwright 依赖
  - `internal/handler/auth.go` - 修复登录响应以包含 csrf_token

- **测试数据库**：`oreader_test.db`（与生产环境分离）

- **外部依赖**：
  - OpenAI News RSS (`https://openai.com/news/rss.xml`) 用于订阅源测试
  - Playwright 浏览器（Chromium）
