# Monorepo 目录结构重构 执行计划

> **规格文档:** `docs/superpowers/specs/2026-03-30-monorepo-restructure-design.md`
> **目标:** 将分散在根目录的 Go 后端、Python converter、React 前端重构为清晰的 monorepo 布局

---

## 变更概览

| 区域 | 当前 | 重构后 |
|------|------|--------|
| Go module | `oreader`（根目录） | `github.com/khalily/oreader`（`backend/`） |
| 前端 | `web/` | `frontend/`（nginx 托管，取消 embed） |
| Converter | `converter/`（requirements.txt） | `services/converter/`（pyproject.toml + uv） |
| Proto | `converter/proto/` | `proto/`（独立 Go module） |
| Docker Compose | 根目录单文件 + profiles | `docker/` 三份 compose 文件 |
| Dockerfile | 根目录（embed 前端） | 各服务独立 Dockerfile |
| Makefile | 根目录（混合所有命令） | 根 + 各服务子 Makefile |
| 镜像源 | 清华源 / 混合 | 统一阿里云镜像 |

---

## Phase 1: 创建目标目录结构 + git mv 文件移动

### Task 1.1: 创建目标目录骨架

```bash
mkdir -p backend
mkdir -p services/converter/src services/converter/tests
mkdir -p proto/go proto/python
mkdir -p docker
```

### Task 1.2: 移动 Go 后端代码到 backend/

```bash
# Go 源码（包含 cmd/server/ 和 cmd/migrate-to-markdown/）
git mv cmd backend/cmd
git mv internal backend/internal
git mv migrations backend/migrations

# Go module 文件
git mv go.mod backend/go.mod
git mv go.sum backend/go.sum

# Go 开发配置
git mv .air.toml backend/.air.toml

# 构建产物目录（保留 .gitkeep）
mkdir -p backend/build backend/coverage
```

> **注意：** `cmd/migrate-to-markdown/`（HTML→Markdown 迁移工具）随 `cmd/` 一并移动，
> 其 import 路径在 Task 2.2 中统一替换。

### Task 1.2.1: 更新 backend/.air.toml 配置

`.air.toml` 从根目录移至 `backend/` 后，需要更新内部路径（相对位置变化）：

```toml
# 关键变更：路径现在是相对于 backend/ 目录
[build]
  cmd = "go build -o /tmp/oreader ./cmd/server"
  bin = "/tmp/oreader"
  # 监控目录改为 backend/ 的子目录
  include_dir = ["cmd", "internal"]
  # 排除前端和 converter 目录
  exclude_dir = ["build", "coverage"]
```

### Task 1.3: 移动前端代码 web/ → frontend/

```bash
git mv web frontend
```

### Task 1.4: 移动 proto 文件到 proto/

```bash
# 移动 proto 定义文件
git mv converter/proto/paper.proto proto/paper.proto

# ⚠️ 生成的 Go/Python 代码不要直接 git mv — 需要 Task 1.4.1 重新生成
# 旧的生成代码包含硬编码的旧路径，无法通过移动修复
rm -f converter/proto/paper.pb.go converter/proto/paper_grpc.pb.go
rm -f converter/proto/paper_pb2.py converter/proto/paper_pb2_grpc.py
rm -f converter/proto/paper_pb2.pyi

touch proto/python/__init__.py
```

### Task 1.4.1: 更新 proto 定义并重新生成代码

⚠️ **关键步骤：** `paper.proto` 的 `go_package` 和生成的代码内部都嵌入了旧路径，
**不能仅靠 `git mv`**，必须修改源文件并重新生成。

**步骤 1：** 更新 `proto/paper.proto` 中的 `go_package`：

```protobuf
// 旧：
option go_package = "oreader/converter/proto";

// 新：
option go_package = "github.com/khalily/oreader/proto/go";
```

**步骤 2：** 重新生成 Go 代码（需要 `protoc` + `protoc-gen-go` + `protoc-gen-go-grpc`）：

```bash
cd proto

# 安装 protoc 插件（如果未安装）
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 生成 Go 代码
protoc --go_out=go --go_opt=paths=source_relative \
       --go-grpc_out=go --go-grpc_opt=paths=source_relative \
       paper.proto
```

**步骤 3：** 重新生成 Python 代码（需要 `grpcio-tools`）：

```bash
cd proto

# 生成 Python 代码
python -m grpc_tools.protoc \
  --python_out=python \
  --grpc_python_out=python \
  paper.proto

# 修复 Python import（生成的 _grpc.py 引用 _pb2 的方式需要调整）
sed -i 's/import paper_pb2/from . import paper_pb2/' python/paper_pb2_grpc.py
```

**验证：**
```bash
# 检查 Go 代码中的 go_package
head -5 proto/go/paper.pb.go | grep -o 'go_package.*'
# 期望看到：go_package = "github.com/khalily/oreader/proto/go"

# 检查 Python import
grep 'import paper_pb2' proto/python/paper_pb2_grpc.py
# 期望看到：from . import paper_pb2
```

### Task 1.5: 移动 converter 代码到 services/converter/

```bash
git mv converter/converter.py services/converter/src/converter.py
git mv converter/server.py services/converter/src/server.py
git mv converter/tests/test_converter.py services/converter/tests/test_converter.py
git mv converter/tests/__init__.py services/converter/tests/__init__.py
touch services/converter/src/__init__.py
```

### Task 1.6: 移动 Docker Compose 相关

```bash
git mv docker-compose.yml docker/docker-compose.yml
git mv .env.test docker/.env.test
git mv .env.prod.example docker/.env.example
```

#### 修正 docker/.env.test 内容

当前 `.env.test` 使用了 SQLite 风格的 `DATABASE_URL`，需修正为 MySQL：

```bash
# 修正前：
# DATABASE_URL=oreader_test.db
# RATE_LIMIT_ENABLED=false
# OREADER_CONFIG=testing
# TEST_JWT_SECRET=test-jwt-secret-for-testing-must-be-32ch
# TEST_OAUTH_CLIENT_SECRET=test-oauth-client-secret-placeholder

# 修正后：
cat > docker/.env.test << 'EOF'
# Test Environment Configuration (MySQL)
DATABASE_URL=mysql://oreader:oreader@tcp(db:3306)/oreader_test?charset=utf8mb4&parseTime=True&loc=Local
RATE_LIMIT_ENABLED=false
OREADER_CONFIG=testing

# Test Secrets (placeholder values for CI/testing only)
TEST_JWT_SECRET=test-jwt-secret-for-testing-must-be-32ch
TEST_OAUTH_CLIENT_SECRET=test-oauth-client-secret-placeholder
EOF
```

### Task 1.7: 清理根目录冗余文件

删除以下文件（重构后被新文件替代）：

```bash
# 脚本（由 Makefile 替代）
rm -f dev.sh run.sh test-all.sh

# 构建产物
rm -f handler.test service.test server oreader.db
rm -f coverage.out coverage_before.out coverage_after.out coverage_repo.out coverage_service.out
rm -rf coverage/ build/

# 根目录 node_modules（不属于根目录）
rm -rf node_modules/
rm -f package-lock.json

# 旧的 .dockerignore（各服务会有自己的）
rm -f .dockerignore

# embed 相关文件（取消 embed）— 注意：Task 1.2 已将 cmd/ 移到 backend/cmd/
rm -f backend/cmd/server/embed.go
rm -f backend/cmd/server/embed_stub.go
```

---

## Phase 2: Go Module 路径变更

### Task 2.1: 更新 backend/go.mod module 路径

文件: `backend/go.mod`

将 `module oreader` 改为 `module github.com/khalily/oreader`

### Task 2.2: 批量替换所有 Go 文件中的 import 路径

76 个文件中的 `"oreader/` 前缀需要替换为 `"github.com/khalily/oreader/`。

⚠️ **重要：分两步执行，先处理通用路径，再修正 proto 路径。**

`paper_client.go:8` 的 proto import 是 `pb "oreader/converter/proto"`，
通用 sed 会将其错误地替换为 `"github.com/khalily/oreader/converter/proto"`（不存在的路径）。
正确目标应该是 `"github.com/khalily/oreader/proto/go"`（独立 proto module）。

```bash
# 步骤 1：通用 import 路径替换
find backend/ -name '*.go' -exec sed -i 's|"oreader/|"github.com/khalily/oreader/|g' {} +

# 步骤 2：修正 proto import（被步骤 1 错误替换的路径）
# "github.com/khalily/oreader/converter/proto" → "github.com/khalily/oreader/proto/go"
find backend/ -name '*.go' -exec sed -i 's|"github.com/khalily/oreader/converter/proto"|"github.com/khalily/oreader/proto/go"|g' {} +
```

验证替换结果：
```bash
# 确保没有残留的旧路径
grep -r '"oreader/' backend/ || echo "OK: no old imports"
# 确保没有错误的 converter/proto 路径
grep -r 'oreader/converter/proto' backend/ || echo "OK: no stale proto imports"
# 查看新的 proto import
grep -r 'oreader/proto/go' backend/
```

### Task 2.3: 创建 proto/go 独立 Go module

创建文件 `proto/go/go.mod`：
```
module github.com/khalily/oreader/proto/go

go 1.25.0

require (
	google.golang.org/grpc v1.79.3
	google.golang.org/protobuf v1.36.11
)

require (
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
)
```

运行 `cd proto/go && go mod tidy`

### Task 2.4: 在 backend/go.mod 中添加 proto module 依赖

在 `backend/go.mod` 末尾添加：
```
require github.com/khalily/oreader/proto/go v0.0.0

replace github.com/khalily/oreader/proto/go => ../proto/go
```

运行 `cd backend && go mod tidy` 验证。确保 `go.sum` 正确生成。

### Task 2.5: 验证 proto import（已在 Task 2.2 步骤 2 中修正）

`backend/internal/infra/grpc/paper_client.go` 中的 import 应变为：
```go
pb "github.com/khalily/oreader/proto/go"
```

如果仍有问题，手动修正此文件的 import 行。

### Task 2.6: 删除 backend/cmd/server/embed.go 和 embed_stub.go

删除两个 embed 文件，并从 main.go 中移除所有 embed 相关代码（见 Task 2.7）。

```bash
rm -f backend/cmd/server/embed.go
rm -f backend/cmd/server/embed_stub.go
```

> `embed.go` 提供 `//go:build !noembed` 构建标签下的 `StaticFS()` 实现，
> `embed_stub.go` 提供 `//go:build noembed` 下的空实现。两者都需要删除。

### Task 2.7: 重写 backend/cmd/server/main.go — 移除 embed 逻辑

从 `main.go` 中移除：
- `"io/fs"` import
- `"os"` 中用于检查 `web/dist` 的代码
- `staticFS`、`fileServer` 相关逻辑（约第 270-341 行）
- `StaticFS()` 调用
- SPA fallback 路由
- `/assets/*filepath` 路由
- `FRONTEND_NOT_AVAILABLE` 错误处理

保留：
- 所有 API 路由（`/health`、`/api/v1/*`）
- 中间件
- 数据库连接
- Worker 启动

移除后，`NoRoute` handler 简化为仅返回 404 JSON。

### Task 2.8: 验证 Go 编译

```bash
cd backend
go build ./cmd/server
go vet ./internal/...
```

---

## Phase 3: Dockerfile 重写

### Task 3.1: 创建 backend/Dockerfile（生产，纯 Go 构建）

文件: `backend/Dockerfile`

⚠️ **注意：** `backend/go.mod` 的 `replace` directive 指向 `../proto/go`，
Docker 构建时需要 proto 目录在 context 中可访问。
有两种方案：
- **方案 A（推荐）：** Docker Compose / CI 中用项目根目录作为 context，`dockerfile: backend/Dockerfile`
- **方案 B：** Dockerfile 内先 COPY proto module

这里采用 **方案 A**，与 dev compose 保持一致。

```dockerfile
ARG GO_VERSION
ARG ALPINE_VERSION

FROM golang:${GO_VERSION}-alpine AS builder
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --no-cache git ca-certificates tzdata
ENV CGO_ENABLED=0
ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,direct

# 构建 context 是项目根目录，先复制 proto module（replace directive 需要）
WORKDIR /src
COPY proto/go/ /src/proto/go/
COPY backend/go.mod backend/go.sum /src/backend/

WORKDIR /src/backend
RUN go mod download

# 复制 backend 源码
COPY backend/ /src/backend/

RUN go build -ldflags="-w -s" -o /oreader ./cmd/server

FROM alpine:${ALPINE_VERSION}
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --no-cache ca-certificates wget tzdata
COPY --from=builder /oreader /oreader
RUN addgroup -g 1000 oreader && adduser -D -u 1000 -G oreader oreader
RUN mkdir -p /data && chown -R oreader:oreader /data
USER oreader
EXPOSE 8080
HEALTHCHECK CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1
ENTRYPOINT ["/oreader"]
```

对应的 Docker Compose / CI build 配置：
```yaml
backend:
  build:
    context: ..           # 项目根目录
    dockerfile: backend/Dockerfile
```

### Task 3.2: 创建 backend/Dockerfile.dev（开发，air hot-reload）

文件: `backend/Dockerfile.dev`

⚠️ **注意：** 与生产 Dockerfile 相同的原因（proto replace directive），build context 需要是项目根目录。

```dockerfile
ARG GO_VERSION

FROM golang:${GO_VERSION}-alpine AS development
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --no-cache git make wget bash ca-certificates
ENV CGO_ENABLED=0
ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,direct

# 构建 context 是项目根目录
WORKDIR /project

# 先复制 proto module（replace directive 需要）
COPY proto/go/ /project/proto/go/
COPY backend/go.mod backend/go.sum /project/backend/

WORKDIR /project/backend
RUN go mod download

RUN go install github.com/air-verse/air@latest
ENV PATH="${PATH}:${HOME}/go/bin"

CMD ["air"]
```

Docker Compose 中挂载完整项目目录覆盖 COPY：
```yaml
volumes:
  - ..:/project
working_dir: /project/backend
```

### Task 3.3: 创建 backend/.dockerignore

```
build/
coverage/
*.test
*.out
```

### Task 3.4: 创建 frontend/Dockerfile（生产，nginx 托管）

文件: `frontend/Dockerfile`

```dockerfile
ARG NODE_VERSION

FROM node:${NODE_VERSION}-alpine AS builder
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm config set registry https://registry.npmmirror.com
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

### Task 3.5: 创建 frontend/Dockerfile.dev（开发，Vite dev server）

文件: `frontend/Dockerfile.dev`

```dockerfile
ARG NODE_VERSION

FROM node:${NODE_VERSION}-alpine AS frontend-dev
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm config set registry https://registry.npmmirror.com
RUN npm install
COPY . .
EXPOSE 5173
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]
```

### Task 3.6: 创建 frontend/nginx.conf

文件: `frontend/nginx.conf`

```nginx
server {
    listen 80;
    root /usr/share/nginx/html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Task 3.7: 创建 frontend/.dockerignore

```
node_modules/
dist/
*.log
```

### Task 3.8: 创建 services/converter/Dockerfile

文件: `services/converter/Dockerfile`

```dockerfile
ARG PYTHON_VERSION
FROM python:${PYTHON_VERSION}-slim

COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

WORKDIR /app

RUN sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list.d/debian.sources
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgl1 libglib2.0-0 libgomp1 && rm -rf /var/lib/apt/lists/*

COPY services/converter/pyproject.toml .
RUN uv pip install --system --no-cache -i https://mirrors.aliyun.com/pypi/simple/ --trusted-host mirrors.aliyun.com .

RUN echo '{"layout-config":{"model":"doclayout_yolo"},"formula-config":{"mfd_model":"yolo_v8_mfd","mfr_model":"unimernet_small","enable":false},"table-config":{"model":"tablemaster","enable":false,"max_time":60},"device-mode":"cpu"}' > /root/magic-pdf.json

# Copy proto generated Python code (context is project root)
# server.py 在 /app/src/，其 ../../../proto/python 解析为 /proto/python
COPY proto/python/ /proto/python/

COPY services/converter/src/ ./src/

EXPOSE 50051
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD python -c "import socket; s=socket.socket(); s.settimeout(5); s.connect(('localhost', 50051)); s.close()" || exit 1

CMD ["python", "src/server.py"]
```

### Task 3.9: 创建 services/converter/.dockerignore

```
__pycache__/
.pytest_cache/
*.pyc
.venv/
tests/
```

---

## Phase 4: Python Converter uv 迁移

### Task 4.1: 创建 services/converter/pyproject.toml

文件: `services/converter/pyproject.toml`

```toml
[project]
name = "oreader-converter"
version = "0.1.0"
description = "PDF to Markdown converter gRPC service for oReader"
requires-python = ">=3.11"
dependencies = [
    "grpcio~=1.60.0",
    "grpcio-tools~=1.60.0",
    "magic-pdf[full]>=0.10.0",
    "openai>=1.0.0",
    "python-dotenv~=1.0.0",
]

[dependency-groups]
dev = [
    "pytest~=8.3.0",
    "ruff>=0.4.0",
]

[tool.ruff]
target-version = "py311"
line-length = 120
```

### Task 4.2: 更新 services/converter/src/server.py 的 proto 引用路径

`server.py` 从 `converter/server.py`（1 层深）移到 `services/converter/src/server.py`（3 层深）。

将原来的：
```python
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "proto"))
```

改为（从 `services/converter/src/` 到项目根的 `proto/python/`，需要 `../../../`）：
```python
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../proto/python"))
```

> ⚠️ **路径计算说明：**
> - `os.path.dirname(__file__)` = `{project}/services/converter/src`
> - `../` = `services/converter/`
> - `../../` = `services/`
> - `../../../` = `{project}/`（项目根目录）
> - 所以 `../../../proto/python` = `{project}/proto/python` ✅

Docker 构建时需要在 Dockerfile 中 COPY proto/python 到容器内对应路径（见 Task 4.3）。

### Task 4.3: 验证 converter Dockerfile 中的 proto COPY

> ⚠️ **已在 Task 3.8 中完成。** 此步骤仅做验证，不需要额外修改 Dockerfile。

验证 `services/converter/Dockerfile` 中包含以下行（应在 `COPY src/ ./src/` 之前）：

```bash
grep 'COPY proto/python/' services/converter/Dockerfile
# 期望输出：COPY proto/python/ /proto/python/
```

如果没有找到，手动添加：
```dockerfile
# Copy proto generated Python code (context is project root)
# server.py 在 /app/src/，其 ../../../proto/python 解析为 /proto/python
COPY proto/python/ /proto/python/
```

> ⚠️ **路径验证：**
> - 容器内 `server.py` 位于 `/app/src/server.py`
> - `os.path.dirname(__file__)` = `/app/src`
> - `../../../proto/python` 从 `/app/src` 解析：
>   - `../` = `/app/`
>   - `../../` = `/`
>   - `../../../proto/python` = `/proto/python` ✅
> - Dockerfile 将 `proto/python/` 复制到 `/proto/python/`，路径匹配

---

## Phase 5: 分层 Makefile

### Task 5.1: 创建 backend/Makefile

文件: `backend/Makefile`

```makefile
COVERAGE_DIR = ./coverage

.PHONY: install test lint fmt migrate-up migrate-down migrate-create

install:
	go mod download

test:
	@mkdir -p $(COVERAGE_DIR)
	go test -v -race -p 1 -coverprofile=$(COVERAGE_DIR)/coverage.out ./internal/...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

migrate-up:
	migrate -path ./migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path ./migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@test -n "$(name)" || (echo "Usage: make migrate-create name=xxx" && exit 1)
	migrate create -ext sql -dir ./migrations -seq $(name)
```

### Task 5.2: 创建 frontend/Makefile

文件: `frontend/Makefile`

```makefile
.PHONY: install test lint fmt

install:
	npm install

test:
	npm test -- --run

lint:
	npm run lint

fmt:
	npx prettier --write "src/**/*.{ts,tsx}"
```

### Task 5.3: 创建 services/converter/Makefile

文件: `services/converter/Makefile`

```makefile
.PHONY: install test lint fmt

install:
	uv sync

test:
	uv run pytest tests/ -v

lint:
	uv run ruff check src/ tests/

fmt:
	uv run ruff format src/ tests/
```

### Task 5.4: 重写根 Makefile（委托调用）

文件: `Makefile`（根目录）

```makefile
include versions.env
export GO_VERSION NODE_VERSION PYTHON_VERSION MYSQL_VERSION REDIS_VERSION ALPINE_VERSION

.PHONY: install install-backend install-frontend install-converter \
        test test-all lint lint-all fmt \
        migrate-up migrate-down migrate-create \
        docker-dev docker-prod docker-test docker-down docker-logs docker-clean \
        versions check-versions help

# === 依赖安装 ===
install: install-backend install-frontend install-converter

install-backend:
	$(MAKE) -C backend install

install-frontend:
	$(MAKE) -C frontend install

install-converter:
	$(MAKE) -C services/converter install

# === 测试 ===
test:
	$(MAKE) -C backend test

test-all:
	$(MAKE) -C backend test
	$(MAKE) -C frontend test
	$(MAKE) -C services/converter test

# === Lint ===
lint:
	$(MAKE) -C backend lint

lint-all:
	$(MAKE) -C backend lint
	$(MAKE) -C frontend lint
	$(MAKE) -C services/converter lint

# === Format ===
fmt:
	$(MAKE) -C backend fmt
	$(MAKE) -C frontend fmt
	$(MAKE) -C services/converter fmt

# === 数据库迁移 ===
migrate-up:
	$(MAKE) -C backend migrate-up

migrate-down:
	$(MAKE) -C backend migrate-down

migrate-create:
	$(MAKE) -C backend migrate-create name=$(name)

# === Docker Compose ===
# ⚠️ --env-file ../.env: Docker Compose 从 docker/ 运行，需要显式指定根目录 .env
DC_ENV   = --env-file ../.env
DC_DEV   = docker compose $(DC_ENV) -f docker/docker-compose.yml
DC_PROD  = docker compose $(DC_ENV) -f docker/docker-compose.prod.yml
DC_TEST  = docker compose $(DC_ENV) -f docker/docker-compose.test.yml

docker-dev:
	$(DC_DEV) up --build

docker-prod:
	$(DC_PROD) up -d --build

docker-test:
	$(DC_TEST) up --build --abort-on-container-exit

docker-down:
	$(DC_DEV) down

docker-logs:
	$(DC_DEV) logs -f

docker-clean:
	$(DC_DEV) down -v --rmi local

# === 版本管理 ===
versions:
	@echo "oReader Tool Versions (from versions.env):"
	@echo "  Go:              $(GO_VERSION)"
	@echo "  Node.js:         $(NODE_VERSION)"
	@echo "  Python:          $(PYTHON_VERSION)"
	@echo "  MySQL:           $(MYSQL_VERSION)"
	@echo "  Redis:           $(REDIS_VERSION)"
	@echo "  Alpine:          $(ALPINE_VERSION)"

check-versions:
	@bash scripts/check-versions.sh

# === 帮助 ===
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "  install           Install all dependencies"
	@echo "  test              Run backend tests"
	@echo "  test-all          Run all tests (backend + frontend + converter)"
	@echo "  lint              Run backend linter"
	@echo "  lint-all          Run all linters"
	@echo "  fmt               Format all code"
	@echo "  docker-dev        Start dev environment"
	@echo "  docker-prod       Start prod environment"
	@echo "  docker-test       Run tests in Docker"
	@echo "  docker-down       Stop dev environment"
	@echo "  docker-logs       Follow Docker logs"
	@echo "  docker-clean      Remove all Docker resources"
	@echo "  migrate-up        Apply database migrations"
	@echo "  migrate-down      Rollback migration"
	@echo "  migrate-create    Create new migration (name=xxx)"
	@echo "  versions          Show tool versions"
	@echo "  check-versions    Verify version consistency"
```

---

## Phase 6: Docker Compose 分层

### Task 6.1: 创建 docker/docker-compose.yml（开发环境）

文件: `docker/docker-compose.yml`

```yaml
services:
  backend:
    build:
      context: ..
      dockerfile: backend/Dockerfile.dev
      args:
        GO_VERSION: ${GO_VERSION?Set in versions.env}
    container_name: oreader-backend
    ports:
      - "0.0.0.0:${PORT:-8080}:8080"
    environment:
      ENV: ${ENV:-development}
      DATABASE_URL: mysql://oreader:${OREADER_DB_PASSWORD:-oreader}@db:3306/oreader?parseTime=true
      JWT_SECRET_KEY: ${JWT_SECRET_KEY:?JWT_SECRET_KEY is required}
      PORT: "8080"
      LOG_LEVEL: ${LOG_LEVEL:-debug}
      FRONTEND_URL: http://${HOST_IP:-localhost}:5173
      PAPER_GRPC_ADDR: converter:50051
      PAPER_UPLOAD_DIR: /data/uploads/papers
      PAPER_MAX_UPLOAD_SIZE: ${PAPER_MAX_UPLOAD_SIZE:-52428800}
      GITHUB_CLIENT_ID: ${GITHUB_CLIENT_ID:-}
      GITHUB_CLIENT_SECRET: ${GITHUB_CLIENT_SECRET:-}
    volumes:
      # ⚠️ 必须挂载项目根目录，否则 proto replace directive 的 ../proto/go 路径无法解析
      - ..:/project
      - go-module-cache:/root/go/pkg/mod
      - dev-uploads:/data/uploads
    working_dir: /project/backend
    command: >
      air
      --build.cmd "go build -o /tmp/oreader ./cmd/server"
      --build.bin "/tmp/oreader"
      --build.delay "1000"
    depends_on:
      db:
        condition: service_healthy
      converter:
        condition: service_started
        required: false
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s
    networks:
      - oreader-net

  frontend:
    build:
      context: ../frontend
      dockerfile: Dockerfile.dev
      args:
        NODE_VERSION: ${NODE_VERSION?Set in versions.env}
    container_name: oreader-frontend
    ports:
      - "0.0.0.0:${FRONTEND_PORT:-5173}:5173"
    environment:
      VITE_API_TARGET: http://backend:8080
    volumes:
      # 挂载完整目录以确保 vite.config.ts、tailwind.config.js 等变更也能热重载
      - ../frontend:/app
      - /app/node_modules
    depends_on:
      - backend
    restart: unless-stopped
    networks:
      - oreader-net

  converter:
    build:
      context: ..
      dockerfile: services/converter/Dockerfile
      args:
        PYTHON_VERSION: ${PYTHON_VERSION?Set in versions.env}
    container_name: oreader-converter
    ports:
      - "0.0.0.0:${GRPC_PORT:-50051}:50051"
    environment:
      GRPC_PORT: "50051"
      LLM_API_KEY: ${LLM_API_KEY:-}
      LLM_BASE_URL: ${LLM_BASE_URL:-https://api.openai.com/v1}
      LLM_MODEL: ${LLM_MODEL:-gpt-4o-mini}
    volumes:
      - converter-model-cache:/root/.cache
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "python", "-c", "import socket; s=socket.socket(); s.settimeout(5); s.connect(('localhost', 50051)); s.close()"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s
    networks:
      - oreader-net

  db:
    image: mysql:${MYSQL_VERSION?Set in versions.env}
    container_name: oreader-db
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD:-rootpassword}
      MYSQL_DATABASE: oreader
      MYSQL_USER: oreader
      MYSQL_PASSWORD: ${OREADER_DB_PASSWORD:-oreader}
    ports:
      - "0.0.0.0:${DB_PORT:-3306}:3306"
    volumes:
      - mysql-data:/var/lib/mysql
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "root", "-p${MYSQL_ROOT_PASSWORD:-rootpassword}"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s
    networks:
      - oreader-net

  # ===========================================================================
  # Tools: phpMyAdmin (可选，数据库调试)
  # ===========================================================================
  phpmyadmin:
    image: phpmyadmin:${PHPMYADMIN_VERSION:-5.2}
    container_name: oreader-phpmyadmin
    environment:
      PMA_HOST: db
      PMA_PORT: 3306
      PMA_USER: root
      PMA_PASSWORD: ${MYSQL_ROOT_PASSWORD:-rootpassword}
    ports:
      - "0.0.0.0:${PMA_PORT:-8081}:80"
    depends_on:
      - db
    restart: unless-stopped
    profiles:
      - tools
    networks:
      - oreader-net

volumes:
  mysql-data:
  go-module-cache:
  dev-uploads:
  converter-model-cache:

networks:
  oreader-net:
    driver: bridge
```

**注意：** Docker Compose 从 `docker/` 目录运行，不会自动读取根目录 `.env` 文件。
根 Makefile 中 DC_DEV 命令需要添加 `--env-file` 参数（见 Task 5.4 修复）。

### Task 6.2: 创建 docker/docker-compose.prod.yml（生产环境）

```yaml
services:
  backend:
    build:
      context: ..
      dockerfile: backend/Dockerfile
      args:
        GO_VERSION: ${GO_VERSION?Set in versions.env}
        ALPINE_VERSION: ${ALPINE_VERSION?Set in versions.env}
    image: oreader:latest
    container_name: oreader-backend
    ports:
      - "0.0.0.0:${PORT:-8080}:8080"
    environment:
      ENV: production
      DATABASE_URL: mysql://oreader:${OREADER_DB_PASSWORD:-oreader}@db:3306/oreader?parseTime=true
      JWT_SECRET_KEY: ${JWT_SECRET_KEY:?JWT_SECRET_KEY is required}
      PORT: "8080"
      LOG_LEVEL: ${LOG_LEVEL:-info}
      PAPER_GRPC_ADDR: converter:50051
      PAPER_UPLOAD_DIR: /data/uploads/papers
      PAPER_MAX_UPLOAD_SIZE: ${PAPER_MAX_UPLOAD_SIZE:-52428800}
    volumes:
      - prod-uploads:/data
    depends_on:
      db:
        condition: service_healthy
      converter:
        condition: service_started
        required: false
    restart: always
    healthcheck:
      test: ["CMD-SHELL", "wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s
    networks:
      - oreader-net

  frontend:
    build:
      context: ../frontend
      dockerfile: Dockerfile
      args:
        NODE_VERSION: ${NODE_VERSION?Set in versions.env}
    image: oreader-frontend:latest
    container_name: oreader-frontend
    ports:
      - "0.0.0.0:${FRONTEND_PORT:-80}:80"
    depends_on:
      - backend
    restart: always
    networks:
      - oreader-net

  converter:
    build:
      context: ..
      dockerfile: services/converter/Dockerfile
      args:
        PYTHON_VERSION: ${PYTHON_VERSION?Set in versions.env}
    image: oreader-converter:latest
    container_name: oreader-converter
    environment:
      GRPC_PORT: "50051"
      LLM_API_KEY: ${LLM_API_KEY:-}
      LLM_BASE_URL: ${LLM_BASE_URL:-https://api.openai.com/v1}
      LLM_MODEL: ${LLM_MODEL:-gpt-4o-mini}
    volumes:
      - converter-model-cache:/root/.cache
    restart: always
    healthcheck:
      test: ["CMD", "python", "-c", "import socket; s=socket.socket(); s.settimeout(5); s.connect(('localhost', 50051)); s.close()"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s
    networks:
      - oreader-net

  db:
    image: mysql:${MYSQL_VERSION?Set in versions.env}
    container_name: oreader-db
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD:-rootpassword}
      MYSQL_DATABASE: oreader
      MYSQL_USER: oreader
      MYSQL_PASSWORD: ${OREADER_DB_PASSWORD:-oreader}
    volumes:
      - mysql-data:/var/lib/mysql
    restart: always
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "root", "-p${MYSQL_ROOT_PASSWORD:-rootpassword}"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s
    networks:
      - oreader-net

  redis:
    image: redis:${REDIS_VERSION?Set in versions.env}-alpine
    container_name: oreader-redis
    ports:
      - "0.0.0.0:${REDIS_PORT:-6379}:6379"
    volumes:
      - redis-data:/data
    restart: always
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    command: redis-server --appendonly yes
    networks:
      - oreader-net

volumes:
  mysql-data:
  prod-uploads:
  converter-model-cache:
  redis-data:

networks:
  oreader-net:
    driver: bridge
```

### Task 6.3: 创建 docker/docker-compose.test.yml（测试环境）

```yaml
services:
  backend-test:
    build:
      context: ..
      dockerfile: backend/Dockerfile.dev
      args:
        GO_VERSION: ${GO_VERSION?Set in versions.env}
    container_name: oreader-test
    environment:
      ENV: testing
      DATABASE_URL: mysql://oreader:${OREADER_DB_PASSWORD:-oreader}@db:3306/oreader_test?parseTime=true
      TEST_DATABASE_URL: oreader:${OREADER_DB_PASSWORD:-oreader}@tcp(db:3306)/oreader_test?charset=utf8mb4&parseTime=True&loc=Local
      JWT_SECRET_KEY: test-jwt-secret-for-testing-minimum-32-characters
      PORT: "8080"
      LOG_LEVEL: debug
      PAPER_GRPC_ADDR: converter:50051
    volumes:
      - ..:/project
      - go-module-cache:/root/go/pkg/mod
    working_dir: /project/backend
    command: ["go", "test", "-v", "-race", "./internal/..."]
    depends_on:
      db:
        condition: service_healthy
      converter:
        condition: service_started
        required: true
    networks:
      - oreader-net

  converter:
    build:
      context: ..
      dockerfile: services/converter/Dockerfile
      args:
        PYTHON_VERSION: ${PYTHON_VERSION?Set in versions.env}
    container_name: oreader-test-converter
    environment:
      GRPC_PORT: "50051"
    networks:
      - oreader-net

  db:
    image: mysql:${MYSQL_VERSION?Set in versions.env}
    container_name: oreader-test-db
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD:-rootpassword}
      MYSQL_DATABASE: oreader_test
      MYSQL_USER: oreader
      MYSQL_PASSWORD: ${OREADER_DB_PASSWORD:-oreader}
    volumes:
      - test-mysql-data:/var/lib/mysql
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "root", "-p${MYSQL_ROOT_PASSWORD:-rootpassword}"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s
    networks:
      - oreader-net

volumes:
  go-module-cache:
  test-mysql-data:

networks:
  oreader-net:
    driver: bridge
```

---

## Phase 7: CI/CD 更新

### Task 7.1: 更新 .github/workflows/ci.yml

关键变更：
- `working-directory: backend` 用于所有 Go 操作
- `working-directory: frontend` 用于所有前端操作（原 `web`）
- `working-directory: services/converter` 用于 converter 操作
- `cache-dependency-path: frontend/package-lock.json`（原 `web/package-lock.json`）
- Docker build: `context: .`（项目根）, `file: backend/Dockerfile`
- Converter lint: `ruff` 替代 `flake8` + `black`
- Converter test: `cd services/converter && uv run pytest`
- OpenAPI test: `working-directory: backend`

**注意**: `load-versions` job 和 `version-drift` job 不变（仍从根目录 `versions.env` 读取）。

```yaml
name: CI

on:
  push:
    branches: ['**']
  pull_request:
    branches: ['**']

concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: true

jobs:
  # =============================================================================
  # Load versions from versions.env → job outputs (single source of truth)
  # =============================================================================
  load-versions:
    name: "Load Versions"
    runs-on: ubuntu-latest
    outputs:
      GO_VERSION: ${{ steps.export.outputs.GO_VERSION }}
      NODE_VERSION: ${{ steps.export.outputs.NODE_VERSION }}
      PYTHON_VERSION: ${{ steps.export.outputs.PYTHON_VERSION }}
      MYSQL_VERSION: ${{ steps.export.outputs.MYSQL_VERSION }}
      REDIS_VERSION: ${{ steps.export.outputs.REDIS_VERSION }}
      GOLANGCI_LINT_VERSION: ${{ steps.export.outputs.GOLANGCI_LINT_VERSION }}
      ALPINE_VERSION: ${{ steps.export.outputs.ALPINE_VERSION }}
    steps:
      - uses: actions/checkout@v4

      - name: Export versions.env to job outputs
        id: export
        run: |
          while IFS='=' read -r key value; do
            [[ "$key" =~ ^#.*$ ]] && continue
            [[ -z "$key" ]] && continue
            echo "$key=$value" >> "$GITHUB_OUTPUT"
            echo "  $key=$value"
          done < versions.env

  # =============================================================================
  # Version Drift Detection
  # =============================================================================
  version-drift:
    name: "Version Consistency"
    runs-on: ubuntu-latest
    needs: [load-versions]
    steps:
      - uses: actions/checkout@v4

      - name: Check version consistency
        run: bash scripts/check-versions.sh

  # =============================================================================
  # OpenAPI: Lint
  # =============================================================================
  openapi-lint:
    name: "OpenAPI: Lint"
    runs-on: ubuntu-latest
    needs: [load-versions, version-drift]
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: ${{ needs.load-versions.outputs.NODE_VERSION }}

      - name: Lint OpenAPI spec
        run: npx @redocly/cli lint docs/openapi.yaml --config .redocly.yaml

  # =============================================================================
  # OpenAPI: Route & Contract Tests (no MySQL required)
  # =============================================================================
  openapi-test:
    name: "OpenAPI: Route & Contract Tests"
    runs-on: ubuntu-latest
    needs: [load-versions, version-drift]
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ needs.load-versions.outputs.GO_VERSION }}
          cache: true
          cache-dependency-path: backend/go.sum

      - name: Download Go modules
        working-directory: backend
        run: go mod download

      - name: Run OpenAPI route consistency test
        working-directory: backend
        run: go test -v -run TestOpenAPIRoutes ./internal/testutil/

      - name: Run OpenAPI contract tests
        working-directory: backend
        run: go test -v -run TestContract_ ./internal/testutil/

  # =============================================================================
  # Backend: Lint
  # =============================================================================
  backend-lint:
    name: "Backend: Lint"
    runs-on: ubuntu-latest
    needs: [load-versions, version-drift]
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ needs.load-versions.outputs.GO_VERSION }}
          cache: true
          cache-dependency-path: backend/go.sum

      - name: Download Go modules
        working-directory: backend
        run: go mod download

      - name: Go vet
        working-directory: backend
        run: go vet ./internal/...

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: ${{ needs.load-versions.outputs.GOLANGCI_LINT_VERSION }}
          install-mode: goinstall
          working-directory: backend
          args: --timeout=5m

  # =============================================================================
  # Backend: Test (requires MySQL)
  # =============================================================================
  backend-test:
    name: "Backend: Test"
    runs-on: ubuntu-latest
    needs: [load-versions, version-drift]
    services:
      mysql:
        image: mysql:${{ needs.load-versions.outputs.MYSQL_VERSION }}
        env:
          MYSQL_ROOT_PASSWORD: rootpassword
          MYSQL_DATABASE: oreader_test
          MYSQL_USER: oreader
          MYSQL_PASSWORD: oreader
        ports:
          - 3306:3306
        options: >-
          --health-cmd="mysqladmin ping -h localhost -u root -prootpassword"
          --health-interval=10s
          --health-timeout=5s
          --health-retries=10
          --health-start-period=30s
    env:
      TEST_DATABASE_URL: "oreader:oreader@tcp(127.0.0.1:3306)/oreader_test?charset=utf8mb4&parseTime=True&loc=Local"
      TEST_JWT_SECRET: "test-jwt-secret-for-ci-must-be-32-characters"
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ needs.load-versions.outputs.GO_VERSION }}
          cache: true
          cache-dependency-path: backend/go.sum

      - name: Install MySQL client
        run: sudo apt-get update && sudo apt-get install -y mysql-client

      - name: Run database migrations
        run: |
          for f in backend/migrations/*.up.sql; do
            echo "Applying $f..."
            mysql -h 127.0.0.1 -u root -prootpassword oreader_test < "$f"
          done

      - name: Run Go tests
        working-directory: backend
        run: |
          mkdir -p coverage
          go test -v -p 1 -race -coverprofile=coverage/coverage.out -covermode=atomic ./internal/...

      - name: Upload coverage
        uses: actions/upload-artifact@v4
        if: always()
        with:
          name: go-coverage
          path: backend/coverage/coverage.out

  # =============================================================================
  # Frontend: Lint
  # =============================================================================
  frontend-lint:
    name: "Frontend: Lint"
    runs-on: ubuntu-latest
    needs: [load-versions, version-drift]
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: ${{ needs.load-versions.outputs.NODE_VERSION }}
          cache: 'npm'
          cache-dependency-path: frontend/package-lock.json

      - name: Install dependencies
        working-directory: frontend
        run: npm ci

      - name: ESLint
        working-directory: frontend
        run: npm run lint

  # =============================================================================
  # Frontend: Test
  # =============================================================================
  frontend-test:
    name: "Frontend: Test"
    runs-on: ubuntu-latest
    needs: [load-versions, version-drift]
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: ${{ needs.load-versions.outputs.NODE_VERSION }}
          cache: 'npm'
          cache-dependency-path: frontend/package-lock.json

      - name: Install dependencies
        working-directory: frontend
        run: npm ci

      - name: Run Vitest
        working-directory: frontend
        run: npm test -- --run --coverage

      - name: Upload coverage
        uses: actions/upload-artifact@v4
        if: always()
        with:
          name: frontend-coverage
          path: frontend/coverage/

  # =============================================================================
  # Frontend: Build
  # =============================================================================
  frontend-build:
    name: "Frontend: Build"
    runs-on: ubuntu-latest
    needs: [load-versions, version-drift]
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: ${{ needs.load-versions.outputs.NODE_VERSION }}
          cache: 'npm'
          cache-dependency-path: frontend/package-lock.json

      - name: Install dependencies
        working-directory: frontend
        run: npm ci

      - name: TypeScript check + Vite build
        working-directory: frontend
        run: npm run build

      - name: Upload dist
        uses: actions/upload-artifact@v4
        with:
          name: frontend-dist
          path: frontend/dist/
          retention-days: 1

  # =============================================================================
  # Converter: Lint
  # =============================================================================
  converter-lint:
    name: "Converter: Lint"
    runs-on: ubuntu-latest
    needs: [load-versions, version-drift]
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-python@v5
        with:
          python-version: ${{ needs.load-versions.outputs.PYTHON_VERSION }}

      - name: Install uv and ruff
        run: pip install uv ruff

      - name: Run ruff check
        working-directory: services/converter
        run: ruff check src/ tests/

      - name: Check ruff formatting
        working-directory: services/converter
        run: ruff format --check src/ tests/

  # =============================================================================
  # Converter: Test
  # =============================================================================
  converter-test:
    name: "Converter: Test"
    runs-on: ubuntu-latest
    needs: [load-versions, version-drift]
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-python@v5
        with:
          python-version: ${{ needs.load-versions.outputs.PYTHON_VERSION }}

      - name: Install uv
        run: pip install uv

      - name: Sync dependencies
        working-directory: services/converter
        run: uv sync

      - name: Run converter tests
        working-directory: services/converter
        run: uv run pytest tests/ -v --tb=short

  # =============================================================================
  # Docker: Build (validates production multi-stage build)
  # =============================================================================
  docker-build:
    name: "Docker: Build"
    runs-on: ubuntu-latest
    needs: [load-versions, version-drift, frontend-build]
    steps:
      - uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Build backend production image
        uses: docker/build-push-action@v6
        with:
          context: .
          file: backend/Dockerfile
          push: false
          load: true
          tags: oreader:ci-test
          cache-from: type=gha
          cache-to: type=gha,mode=max
          build-args: |
            GO_VERSION=${{ needs.load-versions.outputs.GO_VERSION }}
            ALPINE_VERSION=${{ needs.load-versions.outputs.ALPINE_VERSION }}

      - name: Build frontend production image
        uses: docker/build-push-action@v6
        with:
          context: frontend
          file: frontend/Dockerfile
          push: false
          load: true
          tags: oreader-frontend:ci-test
          cache-from: type=gha
          cache-to: type=gha,mode=max
          build-args: |
            NODE_VERSION=${{ needs.load-versions.outputs.NODE_VERSION }}

      - name: Build converter production image
        uses: docker/build-push-action@v6
        with:
          context: .
          file: services/converter/Dockerfile
          push: false
          load: true
          tags: oreader-converter:ci-test
          cache-from: type=gha
          cache-to: type=gha,mode=max
          build-args: |
            PYTHON_VERSION=${{ needs.load-versions.outputs.PYTHON_VERSION }}

      - name: Verify backend binary runs
        run: docker run --rm oreader:ci-test --help || true

  # =============================================================================
  # Security: Scan
  # =============================================================================
  security-scan:
    name: "Security: Scan"
    runs-on: ubuntu-latest
    needs: [load-versions, version-drift, backend-lint]
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ needs.load-versions.outputs.GO_VERSION }}
          cache: true
          cache-dependency-path: backend/go.sum

      - name: govulncheck
        uses: golang/govulncheck-action@v1
        with:
          go-version-input: ${{ needs.load-versions.outputs.GO_VERSION }}
          repo-path: backend

      - name: npm audit
        working-directory: frontend
        run: |
          npm ci
          npm audit --audit-level=high
        continue-on-error: true
```

### Task 7.2: 更新 .github/workflows/deploy.yml

关键变更：
- Backend build: `context: .`（项目根，proto replace directive 需要访问）, `file: backend/Dockerfile`
- Converter build: `context: .`, `file: services/converter/Dockerfile`（需要根目录 context 来访问 proto/）
- **移除** `NODE_VERSION` build arg（backend 不再构建前端）
- 新增 frontend image build（`context: frontend`, `file: frontend/Dockerfile`）

```yaml
name: Deploy

on:
  push:
    branches: [main, master]
    tags: ['v*']
  workflow_dispatch:
    inputs:
      environment:
        description: 'Deployment environment'
        required: true
        default: 'staging'
        type: choice
        options:
          - staging
          - production

jobs:
  load-versions:
    name: "Load Versions"
    runs-on: ubuntu-latest
    outputs:
      GO_VERSION: ${{ steps.export.outputs.GO_VERSION }}
      NODE_VERSION: ${{ steps.export.outputs.NODE_VERSION }}
      PYTHON_VERSION: ${{ steps.export.outputs.PYTHON_VERSION }}
      ALPINE_VERSION: ${{ steps.export.outputs.ALPINE_VERSION }}
    steps:
      - uses: actions/checkout@v4
      - name: Export versions.env to job outputs
        id: export
        run: |
          while IFS='=' read -r key value; do
            [[ "$key" =~ ^#.*$ ]] && continue
            [[ -z "$key" ]] && continue
            echo "$key=$value" >> "$GITHUB_OUTPUT"
          done < versions.env

  build-and-push:
    name: "Build & Push Images"
    runs-on: ubuntu-latest
    needs: [load-versions]
    permissions:
      contents: read
      packages: write

    steps:
      - uses: actions/checkout@v4

      - name: Docker meta
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: |
            ${{ secrets.DOCKER_REGISTRY || 'ghcr.io' }}/${{ github.repository }}
          tags: |
            type=ref,event=branch
            type=semver,pattern={{version}}
            type=semver,pattern={{major}}.{{minor}}
            type=sha,prefix=

      - name: Login to registry
        uses: docker/login-action@v3
        with:
          registry: ${{ secrets.DOCKER_REGISTRY || 'ghcr.io' }}
          username: ${{ secrets.DOCKER_USERNAME || github.actor }}
          password: ${{ secrets.DOCKER_PASSWORD || secrets.GITHUB_TOKEN }}

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Build and push backend image
        uses: docker/build-push-action@v6
        with:
          context: .
          file: backend/Dockerfile
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
          build-args: |
            VERSION=${{ github.ref_name }}
            GO_VERSION=${{ needs.load-versions.outputs.GO_VERSION }}
            ALPINE_VERSION=${{ needs.load-versions.outputs.ALPINE_VERSION }}

      - name: Build and push frontend image
        uses: docker/build-push-action@v6
        with:
          context: frontend
          file: frontend/Dockerfile
          push: true
          tags: ${{ steps.meta.outputs.tags }}-frontend
          cache-from: type=gha
          cache-to: type=gha,mode=max
          build-args: |
            NODE_VERSION=${{ needs.load-versions.outputs.NODE_VERSION }}

      - name: Build and push converter image
        uses: docker/build-push-action@v6
        with:
          context: .
          file: services/converter/Dockerfile
          push: true
          tags: ${{ steps.meta.outputs.tags }}-converter
          cache-from: type=gha
          cache-to: type=gha,mode=max
          build-args: |
            PYTHON_VERSION=${{ needs.load-versions.outputs.PYTHON_VERSION }}

  deploy:
    name: "Deploy to ${{ inputs.environment || 'staging' }}"
    runs-on: ubuntu-latest
    needs: build-and-push
    if: github.ref == 'refs/heads/main' || github.ref == 'refs/heads/master' || startsWith(github.ref, 'refs/tags/')
    environment: ${{ inputs.environment || 'staging' }}

    steps:
      - name: Deploy info
        run: |
          echo "=== Deployment ==="
          echo "Environment: ${{ inputs.environment || 'staging' }}"
          echo "Ref: ${{ github.ref_name }}"
```

### Task 7.3: 更新 .github/workflows/integration.yml

关键变更：
- `paths` 过滤器更新：`backend/internal/**`, `backend/cmd/**`, `services/converter/**`, `docker/**` 等
- Docker compose 命令使用 `docker/docker-compose.test.yml`
- Converter 是 `required` 依赖

```yaml
name: Integration Test

on:
  pull_request:
    branches: ['**']
    paths:
      - 'backend/internal/**'
      - 'backend/cmd/**'
      - 'services/converter/**'
      - 'proto/**'
      - 'docker/docker-compose.test.yml'
      - 'backend/Dockerfile*'
      - 'backend/go.mod'
      - 'backend/go.sum'
      - 'backend/migrations/**'
      - 'versions.env'
  workflow_dispatch:

concurrency:
  group: integration-${{ github.ref }}
  cancel-in-progress: true

jobs:
  load-versions:
    name: "Load Versions"
    runs-on: ubuntu-latest
    outputs:
      GO_VERSION: ${{ steps.export.outputs.GO_VERSION }}
      NODE_VERSION: ${{ steps.export.outputs.NODE_VERSION }}
      PYTHON_VERSION: ${{ steps.export.outputs.PYTHON_VERSION }}
      MYSQL_VERSION: ${{ steps.export.outputs.MYSQL_VERSION }}
      ALPINE_VERSION: ${{ steps.export.outputs.ALPINE_VERSION }}
    steps:
      - uses: actions/checkout@v4
      - name: Export versions.env to job outputs
        id: export
        run: |
          while IFS='=' read -r key value; do
            [[ "$key" =~ ^#.*$ ]] && continue
            [[ -z "$key" ]] && continue
            echo "$key=$value" >> "$GITHUB_OUTPUT"
          done < versions.env

  integration-test:
    name: "Integration: Docker Compose"
    runs-on: ubuntu-latest
    needs: [load-versions]
    timeout-minutes: 20

    steps:
      - uses: actions/checkout@v4

      - name: Run integration tests via Docker Compose
        run: |
          export OREADER_DB_PASSWORD=oreader
          export MYSQL_ROOT_PASSWORD=rootpassword
          export JWT_SECRET_KEY=test-jwt-secret-for-testing-minimum-32-characters
          export DOCKER_BUILDKIT=1

          export GO_VERSION=${{ needs.load-versions.outputs.GO_VERSION }}
          export NODE_VERSION=${{ needs.load-versions.outputs.NODE_VERSION }}
          export PYTHON_VERSION=${{ needs.load-versions.outputs.PYTHON_VERSION }}
          export MYSQL_VERSION=${{ needs.load-versions.outputs.MYSQL_VERSION }}
          export ALPINE_VERSION=${{ needs.load-versions.outputs.ALPINE_VERSION }}

          docker compose -f docker/docker-compose.test.yml up \
            --build \
            --abort-on-container-exit \
            --exit-code-from backend-test \
            --timeout 120

      - name: Collect logs on failure
        if: failure()
        run: |
          docker compose -f docker/docker-compose.test.yml logs --no-color > integration-test.log 2>&1 || true

      - name: Upload test logs
        if: failure()
        uses: actions/upload-artifact@v4
        with:
          name: integration-test-logs
          path: integration-test.log
          retention-days: 7

      - name: Cleanup
        if: always()
        run: docker compose -f docker/docker-compose.test.yml down -v --remove-orphans
```

---

## Phase 8: 辅助文件更新

### Task 8.1: 更新 .gitignore

```gitignore
# IDE
.idea/
*.swp
*.swo
*~

# Worktrees
.worktrees/

# Python
*.pyc
*.pyo
__pycache__/
*.egg-info/
.pytest_cache/
.coverage
htmlcov/

# Go
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out
go.work
go.work.sum

# Backend
backend/build/
backend/coverage/

# Frontend
frontend/node_modules/
frontend/dist/

# Converter
services/converter/.venv/

# Proto
proto/go/go.sum

# Database
*.db
*.sqlite
*.sqlite3

# Environment
.env
.env.local
.env.*.local
!.env.test

# Node
node_modules/
npm-debug.log*
yarn-debug.log*
yarn-error-log*

# OS
.DS_Store
Thumbs.db

# other
.superpowers
uploads
```

**注意：** 移除了通配 `build/` 和 `dist/`（与具体路径冗余）和 `server`（不再在根目录生成）。

### Task 8.2: 删除根目录残留文件

```bash
# 旧的根目录 Dockerfile（已创建新的各服务 Dockerfile）
rm -f Dockerfile Dockerfile.dev

# 清理 converter 残留（源码已移到 services/converter/）
rm -rf converter/
```

### Task 8.3: 重写 scripts/check-versions.sh

`versions.env` 保持在根目录不变（跨服务共享）。

`scripts/check-versions.sh` 需要大幅改写，所有文件路径都需要更新：

| 原路径 | 新路径 |
|--------|--------|
| `go.mod` | `backend/go.mod` |
| `Dockerfile` | `backend/Dockerfile` |
| `Dockerfile.dev` | `backend/Dockerfile.dev` |
| `converter/Dockerfile` | `services/converter/Dockerfile` |
| `docker-compose.yml` | `docker/docker-compose.yml` |

新增检查项：
- `frontend/Dockerfile` — NODE_VERSION 变量引用
- `frontend/Dockerfile.dev` — NODE_VERSION 变量引用
- `docker/docker-compose.yml` — GO_VERSION, PYTHON_VERSION 变量引用
- `docker/docker-compose.prod.yml` — REDIS_VERSION、MYSQL_VERSION 变量引用
- `docker/docker-compose.test.yml` — MYSQL_VERSION、GO_VERSION、PYTHON_VERSION 变量引用
- `proto/go/go.mod` — GO_VERSION 一致性（`go 1.X` 版本号）

移除检查项：
- `Dockerfile` 中的 `NODE_VERSION`（backend 不再构建前端）

更新 `for` 循环中的 Dockerfile 列表：
```bash
# 旧
for dockerfile in Dockerfile Dockerfile.dev converter/Dockerfile; do

# 新
for dockerfile in backend/Dockerfile backend/Dockerfile.dev \
                  frontend/Dockerfile frontend/Dockerfile.dev \
                  services/converter/Dockerfile; do
```

### Task 8.4: 重写 CLAUDE.md

更新 CLAUDE.md 中的目录结构描述、开发命令、Docker 命令等，反映新的 monorepo 布局。

### Task 8.5: 更新 README.md

更新 README.md 中的项目结构、开发指南、构建说明。

### Task 8.6: 创建根目录 .dockerignore

⚠️ **必须创建根目录 `.dockerignore`**，因为 backend 和 converter 的 build context 都是项目根目录。
不配置的话会将 `frontend/node_modules/` 等大目录发送到 Docker daemon。

**注意：** Docker `.dockerignore` 支持否定模式 `!`，但行为与 `.gitignore` 不同——
如果父目录被排除，子目录的 `!` 否定**可能不生效**（取决于 Docker 版本）。
因此采用"先排除所有，再逐个放行"的策略更可靠：

```gitignore
# 根目录 .dockerignore — 适用于以项目根目录为 context 的所有 Docker 构建

# === 先排除所有 ===
*

# === 放行需要的目录 ===
!backend/
!proto/
!services/
!versions.env

# === 在放行的目录中排除不需要的 ===
backend/build/
backend/coverage/
backend/*.test
backend/*.out
services/converter/tests/
services/converter/.venv/
services/converter/__pycache__/

# === 始终排除 ===
.git/
**/node_modules/
**/__pycache__/
**/.venv/
```

> **替代方案（更简单）：** 如果 Docker 版本 ≥ 20.10，也可以直接列出排除项：
> ```gitignore
> .git/
> .github/
> .claude/
> .superpowers/
> docs/
> openspec/
> frontend/node_modules/
> frontend/dist/
> node_modules/
> backend/build/
> backend/coverage/
> services/converter/tests/
> services/converter/.venv/
> ```
> 这种方式不依赖否定模式，更可预测。**推荐使用替代方案。**

---

## Phase 9: 验证

### Task 9.1: 验证 Go 编译 + 测试

```bash
cd backend
go mod tidy
go build ./cmd/server
go vet ./internal/...
```

### Task 9.2: 验证前端构建

```bash
cd frontend
npm ci
npm run build
npm run lint
npm test -- --run
```

### Task 9.3: 验证 Converter

```bash
cd services/converter
uv sync
uv run pytest tests/ -v
```

### Task 9.4: 验证 Docker Compose 启动

```bash
# 开发环境
make docker-dev

# 验证服务健康
docker compose -f docker/docker-compose.yml ps
```

### Task 9.5: 验证 CI 配置语法

检查 YAML 语法、路径引用正确性。

---

## 执行顺序建议

```
Phase 1 (文件移动)
  ↓
Phase 2 (Go module 变更) ← 核心变更，影响最大
  ↓
Phase 3 (Dockerfile) + Phase 4 (Python uv) ← 可并行
  ↓
Phase 5 (Makefile) + Phase 6 (Docker Compose) ← 可并行
  ↓
Phase 7 (CI/CD) + Phase 8 (辅助文件) ← 可并行
  ↓
Phase 9 (验证)
```

**一次性 commit**，整个重构作为单个 commit 提交，确保 git history 清晰。

commit message 建议：
```
refactor: restructure monorepo layout (backend/, frontend/, services/converter/, proto/)

- Move Go code to backend/ with new module path github.com/khalily/oreader
- Move web/ to frontend/ with nginx serving (remove Go embed)
- Move converter/ to services/converter/ with uv + pyproject.toml
- Extract proto/ as independent Go module
- Split Docker Compose into docker/ (dev/prod/test)
- Create per-service Makefiles + Dockerfiles
- Unify mirror sources to Aliyun
```
