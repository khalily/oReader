# Monorepo 目录结构重构设计

> 日期：2026-03-30
> 状态：已批准
> 范围：项目目录布局、Makefile、Docker、开发工作流全面重构

## 1. 动机

当前项目结构将 Go 后端、Python converter、前端代码分散在根目录，导致：

- 根目录文件过多（go.mod、Dockerfile、docker-compose.yml 等混杂）
- 各服务没有明确的目录边界
- Python 依赖用 requirements.txt，不够现代
- 前端通过 Go embed 嵌入二进制，增加构建复杂度

重构目标：清晰的 monorepo 布局，各服务自包含，统一开发体验。

## 2. 目标目录结构

```
oreader/
├── backend/                        # Go API 服务器 (独立 Go module)
│   ├── cmd/
│   │   └── server/                 # 主入口 (去除 embed 逻辑)
│   │   └── migrate-to-markdown/    # 迁移工具
│   ├── internal/                   # 业务代码 (结构不变)
│   │   ├── config/
│   │   ├── handler/
│   │   ├── infra/
│   │   │   └── grpc/              # gRPC client
│   │   │   └── markdown/
│   │   ├── middleware/
│   │   ├── model/
│   │   ├── repository/
│   │   ├── service/
│   │   ├── testutil/
│   │   └── worker/
│   ├── migrations/                 # SQL 迁移文件
│   ├── go.mod                      # module github.com/khalily/oreader
│   ├── go.sum
│   ├── Makefile
│   ├── Dockerfile                  # 生产 (纯 Go 构建，无 embed)
│   ├── Dockerfile.dev              # 开发 (air hot-reload)
│   ├── .air.toml
│   └── .dockerignore
│
├── frontend/                       # React 前端 (原 web/)
│   ├── src/
│   ├── public/
│   ├── package.json / package-lock.json
│   ├── tsconfig.*.json
│   ├── vite.config.ts
│   ├── vitest.config.ts
│   ├── eslint.config.js
│   ├── tailwind.config.js
│   ├── postcss.config.js
│   ├── index.html
│   ├── Makefile
│   ├── Dockerfile                  # 生产 (nginx 托管)
│   ├── Dockerfile.dev              # 开发 (vite dev server)
│   ├── nginx.conf                  # SPA 路由 + API 代理
│   └── .dockerignore
│
├── services/
│   └── converter/                  # Python gRPC 服务 (uv 管理)
│       ├── src/
│       │   ├── __init__.py
│       │   ├── server.py
│       │   └── converter.py
│       ├── tests/
│       │   ├── __init__.py
│       │   └── test_converter.py
│       ├── pyproject.toml
│       ├── uv.lock
│       ├── Makefile
│       ├── Dockerfile
│       └── .dockerignore
│
├── proto/                          # Protobuf 定义 + 生成代码
│   ├── paper.proto
│   ├── go/                         # 生成的 Go 代码 (独立 Go module)
│   │   ├── go.mod                  # module github.com/khalily/oreader/proto/go
│   │   ├── paper.pb.go
│   │   └── paper_grpc.pb.go
│   └── python/                     # 生成的 Python 代码
│       ├── __init__.py
│       ├── paper_pb2.py
│       └── paper_pb2_grpc.py
│
├── docker/                         # Docker Compose 管理
│   ├── docker-compose.yml          # 开发环境
│   ├── docker-compose.prod.yml     # 生产环境
│   ├── docker-compose.test.yml     # 测试环境
│   ├── .env.example
│   └── .env.test
│
├── docs/
│   ├── openapi.yaml
│   └── superpowers/
│
├── scripts/
│   └── check-versions.sh
│
├── .github/
│   └── workflows/
│       ├── ci.yml
│       ├── deploy.yml
│       └── integration.yml
│
├── Makefile                        # 根：委托调用 + docker 管理
├── versions.env
├── CLAUDE.md
├── README.md
├── .gitignore
├── .env.example
└── .nvmrc / .python-version
```

## 3. 镜像源统一规范

所有外部包安装源统一使用阿里云镜像，确保构建速度和一致性。

| 包管理器 | 阿里云源 | Dockerfile 配置 |
|----------|---------|----------------|
| Alpine APK | `mirrors.aliyun.com` | `RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories` |
| Debian APT | `mirrors.aliyun.com` | `RUN sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list.d/debian.sources` |
| Go modules | `mirrors.aliyun.com/goproxy` | `ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,direct` |
| npm | `registry.npmmirror.com` | `RUN npm config set registry https://registry.npmmirror.com`（已是阿里运营） |
| Python/pip | `mirrors.aliyun.com/pypi/simple` | `uv pip install --system -i https://mirrors.aliyun.com/pypi/simple/ --trusted-host mirrors.aliyun.com` |

**说明：**
- npm 的 `registry.npmmirror.com` 由阿里巴巴团队维护（原 `registry.npm.taobao.org`），无需更换
- 所有 Dockerfile 都必须使用上表中的源，不再使用清华源或其他第三方源

## 4. Go Module 变更

### 4.1 Module 路径

- **当前**：`module oreader`
- **重构后**：`module github.com/khalily/oreader`

### 4.2 Import 路径映射

| 当前 | 重构后 |
|------|--------|
| `oreader/cmd/server` | `github.com/khalily/oreader/cmd/server` |
| `oreader/internal/config` | `github.com/khalily/oreader/internal/config` |
| `oreader/internal/handler` | `github.com/khalily/oreader/internal/handler` |
| `oreader/internal/infra/grpc` | `github.com/khalily/oreader/internal/infra/grpc` |
| `oreader/internal/infra/markdown` | `github.com/khalily/oreader/internal/infra/markdown` |
| `oreader/internal/middleware` | `github.com/khalily/oreader/internal/middleware` |
| `oreader/internal/model` | `github.com/khalily/oreader/internal/model` |
| `oreader/internal/repository` | `github.com/khalily/oreader/internal/repository` |
| `oreader/internal/service` | `github.com/khalily/oreader/internal/service` |
| `oreader/internal/testutil` | `github.com/khalily/oreader/internal/testutil` |
| `oreader/internal/worker` | `github.com/khalily/oreader/internal/worker` |

### 4.3 Proto Go 代码引用

`proto/go/` 作为独立小 Go module：

- `proto/go/go.mod`：`module github.com/khalily/oreader/proto/go`
- `backend/go.mod` 中添加 replace directive：
  ```
  require github.com/khalily/oreader/proto/go v0.0.0
  replace github.com/khalily/oreader/proto/go => ../proto/go
  ```
- `backend/internal/infra/grpc/` 中的 import 改为 `github.com/khalily/oreader/proto/go`

## 5. 取消前端 Embed

### 5.1 移除的内容

- `cmd/server/` 中的 `//go:embed dist/*` 指令
- `cmd/server/dist/` 目录
- `noembed` build tag（不再需要）
- `build-prepare` Makefile target（不再需要复制前端 dist）

### 5.2 backend/Dockerfile（生产）

简化为纯 Go 构建，不再有 `frontend-builder` stage：

```dockerfile
ARG GO_VERSION
ARG ALPINE_VERSION

FROM golang:${GO_VERSION}-alpine AS builder
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --no-cache git ca-certificates tzdata
ENV CGO_ENABLED=0
ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
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

### 5.3 frontend/Dockerfile（生产）

新增 nginx 托管：

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

### 5.4 frontend/nginx.conf

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

### 5.5 backend/Dockerfile.dev（开发）

```dockerfile
ARG GO_VERSION

FROM golang:${GO_VERSION}-alpine AS development
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --no-cache git make wget bash ca-certificates
ENV CGO_ENABLED=0
ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
WORKDIR /src
RUN go install github.com/air-verse/air@latest
ENV PATH="${PATH}:${HOME}/go/bin"
COPY go.mod go.sum ./
RUN go mod download
COPY . .
CMD ["air"]
```

### 5.6 frontend/Dockerfile.dev（开发）

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

## 6. Python Converter uv 迁移

### 6.1 从 requirements.txt 到 pyproject.toml

**当前** `converter/requirements.txt`：
```
grpcio~=1.60.0
grpcio-tools~=1.60.0
magic-pdf[full]>=0.10.0
openai>=1.0.0
python-dotenv~=1.0.0
pytest~=8.3.0
```

**重构后** `services/converter/pyproject.toml`：
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

### 6.2 源码结构

```
services/converter/
├── src/
│   ├── __init__.py
│   ├── server.py          # 原 converter/server.py
│   └── converter.py       # 原 converter/converter.py
├── tests/
│   ├── __init__.py
│   └── test_converter.py
├── pyproject.toml
├── uv.lock
├── Dockerfile
├── Makefile
└── .dockerignore
```

### 6.3 Proto 引用方式

- **本地开发**：`server.py` 通过相对路径 `sys.path.insert(0, "../../proto/python")` 引用生成代码
- **Docker**：Dockerfile 中 `COPY` proto/python 生成代码到容器内

### 6.4 Dockerfile

```dockerfile
ARG PYTHON_VERSION
FROM python:${PYTHON_VERSION}-slim

COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

WORKDIR /app

RUN sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list.d/debian.sources
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgl1 libglib2.0-0 libgomp1 && rm -rf /var/lib/apt/lists/*

COPY pyproject.toml .
RUN uv pip install --system --no-cache -i https://mirrors.aliyun.com/pypi/simple/ --trusted-host mirrors.aliyun.com .

RUN echo '{"layout-config":{"model":"doclayout_yolo"},"formula-config":{"mfd_model":"yolo_v8_mfd","mfr_model":"unimernet_small","enable":false},"table-config":{"model":"tablemaster","enable":false,"max_time":60},"device-mode":"cpu"}' > /root/magic-pdf.json

COPY src/ ./src/

EXPOSE 50051
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD python -c "import socket; s=socket.socket(); s.settimeout(5); s.connect(('localhost', 50051)); s.close()" || exit 1

CMD ["python", "src/server.py"]
```

## 7. 分层 Makefile

### 7.1 设计原则

- **本地不直接运行程序**，所有运行通过 Docker Compose
- 本地 Makefile 只关注：依赖安装、测试、lint/format
- 根 Makefile 做委托调用 + Docker Compose 管理

### 7.2 根 Makefile

```makefile
include versions.env
export GO_VERSION NODE_VERSION PYTHON_VERSION MYSQL_VERSION REDIS_VERSION ALPINE_VERSION

# === 依赖安装 ===
install: install-backend install-frontend install-converter
install-backend:   $(MAKE) -C backend install
install-frontend:  $(MAKE) -C frontend install
install-converter: $(MAKE) -C services/converter install

# === 测试 ===
test:       $(MAKE) -C backend test
test-all:
    $(MAKE) -C backend test
    $(MAKE) -C frontend test
    $(MAKE) -C services/converter test

# === Lint ===
lint:       $(MAKE) -C backend lint
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
migrate-up:     $(MAKE) -C backend migrate-up
migrate-down:   $(MAKE) -C backend migrate-down
migrate-create: $(MAKE) -C backend migrate-create name=$(name)

# === Docker Compose ===
DC_DEV  = docker compose -f docker/docker-compose.yml
DC_PROD = docker compose -f docker/docker-compose.prod.yml
DC_TEST = docker compose -f docker/docker-compose.test.yml

docker-dev:   $(DC_DEV) up --build
docker-prod:  $(DC_PROD) up -d --build
docker-test:  $(DC_TEST) up --build --abort-on-container-exit
docker-down:  $(DC_DEV) down
docker-logs:  $(DC_DEV) logs -f
docker-clean: $(DC_DEV) down -v --rmi local

# === 版本管理 ===
versions:       @echo "..." # 同现有
check-versions: bash scripts/check-versions.sh
```

### 7.3 backend/Makefile

```makefile
COVERAGE_DIR = ./coverage

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

### 7.4 frontend/Makefile

```makefile
install:
    npm install

test:
    npm test -- --run

lint:
    npm run lint

fmt:
    npx prettier --write "src/**/*.{ts,tsx}"
```

### 7.5 services/converter/Makefile

```makefile
install:
    uv sync

test:
    uv run pytest tests/ -v

lint:
    uv run ruff check src/ tests/

fmt:
    uv run ruff format src/ tests/
```

## 8. Docker Compose 分层

### 8.1 目录结构

```
docker/
├── docker-compose.yml          # 开发环境 (默认)
├── docker-compose.prod.yml     # 生产环境
├── docker-compose.test.yml     # 测试环境
├── .env.example
└── .env.test
```

### 8.2 开发环境 (docker-compose.yml)

- backend: `../backend/Dockerfile.dev` + air hot-reload + 源码挂载
- frontend: `../frontend/Dockerfile.dev` + vite dev server + 源码挂载
- converter: `../services/converter/Dockerfile`
- db: MySQL

### 8.3 生产环境 (docker-compose.prod.yml)

- backend: `../backend/Dockerfile` (纯 Go 二进制)
- frontend: `../frontend/Dockerfile` (nginx 托管，新增)
- converter: `../services/converter/Dockerfile`
- db: MySQL
- redis: 缓存
- 无源码挂载，无 hot-reload

### 8.4 测试环境 (docker-compose.test.yml)

- backend-test: 运行 `go test ./internal/...`
- converter: 运行 `uv run pytest`
- db: MySQL (测试数据库)

## 9. CI/CD 变更

保持单 CI 文件，更新路径引用：

- Go 测试: `working-directory: backend`
- 前端测试: `working-directory: frontend`
- Converter 测试: `working-directory: services/converter`
- Docker 构建: `-f backend/Dockerfile ./backend` 等

## 10. 辅助文件更新

### .gitignore

更新路径引用，新增：
```gitignore
backend/build/
backend/coverage/
frontend/node_modules/
frontend/dist/
services/converter/.venv/
proto/go/go.sum
```

移除旧路径引用。

### CLAUDE.md

全面重写，反映新目录结构、命令、架构描述。

### README.md

全面更新开发指南、构建说明、目录结构说明。

### versions.env

保持根目录不变（跨服务共享）。

## 11. 需删除/清理的文件

| 文件 | 处理 |
|------|------|
| 根目录 `go.mod`、`go.sum` | 移到 `backend/` |
| 根目录 `Dockerfile`、`Dockerfile.dev` | 移到 `backend/`（内容修改） |
| 根目录 `docker-compose.yml` | 移到 `docker/`（拆分三份） |
| 根目录 `.air.toml` | 移到 `backend/` |
| 根目录 `dev.sh`、`run.sh`、`test-all.sh` | 删除（由 Makefile 替代） |
| 根目录 `oreader.db`、`*.test`、`server` | 删除（构建产物） |
| 根目录 `coverage*.out`、`coverage/` | 移到 `backend/` |
| 根目录 `node_modules/`、`package-lock.json` | 删除（根目录无 Node 项目） |
| 根目录 `uploads/` | 移到 `backend/uploads/` 或 Docker volume |
| `converter/` 整个目录 | 移到 `services/converter/` 并重构 |
| `web/` 整个目录 | 重命名为 `frontend/` |
| `migrations/` | 移到 `backend/migrations/` |
| `cmd/`、`internal/` | 移到 `backend/` 下 |

## 12. 执行策略

单次完整迁移，一次性 commit：

1. 创建目标目录结构
2. `git mv` 移动文件
3. 更新 Go import 路径
4. 创建各子 Makefile
5. 创建各 Dockerfile
6. 创建 Docker Compose 文件
7. 创建 proto/go 独立 module
8. 更新 pyproject.toml + uv.lock
9. 更新 CI 配置
10. 更新 .gitignore、CLAUDE.md、README.md
11. 验证：本地测试 + Docker Compose 启动验证
