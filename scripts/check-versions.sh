#!/bin/bash
# =============================================================================
# check-versions.sh - Verify version consistency across all configuration files
# =============================================================================
# Usage: bash scripts/check-versions.sh
# Called by: make check-versions, CI version-drift job
#
# Validates that all files reference versions consistent with versions.env.
# Dockerfiles have no defaults (ARG without =), so we check they USE the right
# variables rather than checking literal version values in FROM lines.
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Source the single source of truth
if [ ! -f "$PROJECT_ROOT/versions.env" ]; then
    echo "ERROR: versions.env not found at $PROJECT_ROOT/versions.env"
    exit 1
fi
source "$PROJECT_ROOT/versions.env"

ERRORS=0
WARNINGS=0

# check <file> <description> <expected> <grep_pattern> [extract_mode]
#
# extract_mode:
#   "arg"          - extract value after = (ARG GO_VERSION=1.25)
#   "compose_var"  - verify variable name appears in compose image/build-arg context
#   "file"         - read entire file content (for .nvmrc, .python-version)
#   (default)      - extract first X.Y version number
check() {
    local file="$PROJECT_ROOT/$1"
    local desc="$2"
    local expected="$3"
    local pattern="$4"
    local mode="${5:-}"

    if [ ! -f "$file" ]; then
        echo "SKIP: $desc - file not found: $1"
        return
    fi

    local matches
    matches=$(grep -n "$pattern" "$file" 2>/dev/null || true)

    if [ -z "$matches" ]; then
        echo "SKIP: $desc - pattern not found: $pattern in $1"
        return
    fi

    local line
    line=$(echo "$matches" | head -1)

    local actual
    case "$mode" in
        arg)
            # Extract value after = from: ARG GO_VERSION=1.25
            actual=$(echo "$line" | sed 's/.*=\(.*\)/\1/' | tr -d '"' | tr -d "'" | tr -d '}' | xargs)
            ;;
        file)
            # Read entire file content as version
            actual=$(cat "$file" | tr -d '[:space:]')
            ;;
        compose_var)
            # For docker-compose, verify the variable reference exists
            actual="present"
            expected="present"
            ;;
        *)
            # Extract first X.Y version number
            actual=$(echo "$line" | grep -oE '[0-9]+\.[0-9]+' | head -1)
            ;;
    esac

    if [ -z "$actual" ]; then
        echo "WARN: $desc - could not extract version from: $line"
        WARNINGS=$((WARNINGS + 1))
        return
    fi

    if [ "$actual" != "$expected" ]; then
        echo "DRIFT: $desc"
        echo "  File:     $1"
        echo "  Expected: $expected"
        echo "  Actual:   $actual"
        echo "  Line:     $line"
        echo ""
        ERRORS=$((ERRORS + 1))
    else
        echo "  OK: $desc"
    fi
}

# check_var <file> <description> <var_name>
# Verify that a file references the given variable name
check_var() {
    local file="$PROJECT_ROOT/$1"
    local desc="$2"
    local var_name="$3"

    if [ ! -f "$file" ]; then
        echo "SKIP: $desc - file not found: $1"
        return
    fi

    if grep -qE "\\\$\{?$var_name\}?" "$file" 2>/dev/null; then
        echo "  OK: $desc - references $var_name"
    else
        echo "DRIFT: $desc - expected $var_name reference not found in $1"
        ERRORS=$((ERRORS + 1))
    fi
}

echo "=== oReader Version Consistency Check ==="
echo "Source of truth: versions.env"
echo ""

# --- Go version ---
check "backend/go.mod" \
    "backend/go.mod Go version" \
    "$GO_VERSION" \
    "^go "

check "proto/go/go.mod" \
    "proto/go/go.mod Go version" \
    "$GO_VERSION" \
    "^go "

check_var "backend/Dockerfile" \
    "backend/Dockerfile uses \${GO_VERSION}" \
    "GO_VERSION"

check_var "backend/Dockerfile.dev" \
    "backend/Dockerfile.dev uses \${GO_VERSION}" \
    "GO_VERSION"

# --- Node.js version ---
check ".nvmrc" \
    ".nvmrc" \
    "$NODE_VERSION" \
    "." \
    "file"

check_var "frontend/Dockerfile" \
    "frontend/Dockerfile uses \${NODE_VERSION}" \
    "NODE_VERSION"

check_var "frontend/Dockerfile.dev" \
    "frontend/Dockerfile.dev uses \${NODE_VERSION}" \
    "NODE_VERSION"

# --- Python version ---
check ".python-version" \
    ".python-version" \
    "$PYTHON_VERSION" \
    "." \
    "file"

check_var "services/converter/Dockerfile" \
    "services/converter/Dockerfile uses \${PYTHON_VERSION}" \
    "PYTHON_VERSION"

# --- Alpine version ---
check_var "backend/Dockerfile" \
    "backend/Dockerfile uses \${ALPINE_VERSION}" \
    "ALPINE_VERSION"

# --- Docker Compose: verify version variables are referenced ---
check_var "docker/docker-compose.yml" \
    "docker-compose.yml references GO_VERSION" \
    "GO_VERSION"

check_var "docker/docker-compose.yml" \
    "docker-compose.yml references PYTHON_VERSION" \
    "PYTHON_VERSION"

check_var "docker/docker-compose.yml" \
    "docker-compose.yml references MYSQL_VERSION" \
    "MYSQL_VERSION"

check_var "docker/docker-compose.yml" \
    "docker-compose.yml references PHPMYADMIN_VERSION" \
    "PHPMYADMIN_VERSION"

check_var "docker/docker-compose.prod.yml" \
    "docker-compose.prod.yml references REDIS_VERSION" \
    "REDIS_VERSION"

check_var "docker/docker-compose.prod.yml" \
    "docker-compose.prod.yml references MYSQL_VERSION" \
    "MYSQL_VERSION"

check_var "docker/docker-compose.test.yml" \
    "docker-compose.test.yml references MYSQL_VERSION" \
    "MYSQL_VERSION"

check_var "docker/docker-compose.test.yml" \
    "docker-compose.test.yml references GO_VERSION" \
    "GO_VERSION"

check_var "docker/docker-compose.test.yml" \
    "docker-compose.test.yml references PYTHON_VERSION" \
    "PYTHON_VERSION"

# --- Verify ARG declarations exist (no default) ---
for dockerfile in backend/Dockerfile backend/Dockerfile.dev \
                  frontend/Dockerfile frontend/Dockerfile.dev \
                  services/converter/Dockerfile; do
    if [ -f "$PROJECT_ROOT/$dockerfile" ]; then
        # Check ARG lines don't have defaults (should be bare ARG)
        bad_args=$(grep -E '^ARG [A-Z_]+=' "$PROJECT_ROOT/$dockerfile" 2>/dev/null || true)
        if [ -n "$bad_args" ]; then
            echo "DRIFT: $dockerfile has ARG with default value (should be bare):"
            echo "$bad_args" | while read -r line; do echo "  $line"; done
            echo ""
            ERRORS=$((ERRORS + 1))
        fi
    fi
done

echo ""
echo "=========================================="
if [ $WARNINGS -gt 0 ]; then
    echo "WARNINGS: $WARNINGS"
fi
if [ $ERRORS -gt 0 ]; then
    echo "FAILED: $ERRORS version drift(s) detected."
    echo "Update the files above or adjust versions.env."
    exit 1
fi

echo "OK: All versions are consistent."
exit 0
