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
