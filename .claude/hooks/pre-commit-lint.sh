#!/usr/bin/env bash
# Pre-commit lint hook for Claude Code.
# Runs relevant linters based on staged file types before allowing a git commit.
#
# Exit codes:
#   0 - Allow the commit (no lintable files, or all lints pass)
#   2 - Block the commit (lint failure or missing tool)

set -euo pipefail

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

log()  { echo "[pre-commit-lint] $*" >&2; }
fail() { echo "[pre-commit-lint] ERROR: $*" >&2; }

require_tool() {
  if ! command -v "$1" >/dev/null 2>&1; then
    fail "$1 is not installed. Please install it first."
    case "$1" in
      golangci-lint) log "  Install: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest" ;;
      npx)           log "  Install: npm install -g npx (or install Node.js)" ;;
      flake8)        log "  Install: pip install flake8" ;;
      black)         log "  Install: pip install black" ;;
    esac
    return 1
  fi
}

# ---------------------------------------------------------------------------
# Step 1: Read stdin JSON and check if this is a git commit command
# ---------------------------------------------------------------------------

INPUT=$(cat)

# Extract the command from the JSON payload.
# Uses python3 (already required by the project) to avoid a jq dependency.
COMMAND=$(printf '%s' "$INPUT" \
  | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('tool_input',{}).get('command',''))" \
  2>/dev/null || true)

# If we can't parse the command, allow the tool call to proceed.
if [ -z "$COMMAND" ]; then
  exit 0
fi

# Only intercept git commit (with or without flags).
# Strip leading/trailing whitespace for robust matching.
COMMAND_TRIMMED=$(echo "$COMMAND" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')

# Must start with "git commit" (possibly via a full path like /usr/bin/git).
if ! echo "$COMMAND_TRIMMED" | grep -qE 'git\s+commit'; then
  exit 0
fi

# Allow --no-verify to skip the hook entirely.
if echo "$COMMAND" | grep -q '\-\-no-verify'; then
  log "Detected --no-verify, skipping lint checks."
  exit 0
fi

# ---------------------------------------------------------------------------
# Step 2: Gather staged files
# ---------------------------------------------------------------------------

cd "$(git rev-parse --show-toplevel)"

STAGED_GO=$(git diff --cached --name-only --diff-filter=ACM -- '*.go' 2>/dev/null || true)
STAGED_TS=$(git diff --cached --name-only --diff-filter=ACM -- '*.ts' '*.tsx' 2>/dev/null || true)
STAGED_PY=$(git diff --cached --name-only --diff-filter=ACM -- '*.py' 2>/dev/null || true)
STAGED_OPENAPI=$(git diff --cached --name-only -- 'docs/openapi.yaml' 2>/dev/null || true)

# Quick exit if nothing to lint.
if [ -z "$STAGED_GO" ] && [ -z "$STAGED_TS" ] && [ -z "$STAGED_PY" ] && [ -z "$STAGED_OPENAPI" ]; then
  log "No lintable files staged, skipping lint checks."
  exit 0
fi

HAS_FAILURE=0

# ---------------------------------------------------------------------------
# Phase 1: Go (golangci-lint)
# ---------------------------------------------------------------------------

if [ -n "$STAGED_GO" ]; then
  # Exclude generated protobuf files.
  GO_LINTABLE=$(echo "$STAGED_GO" | grep -v '^converter/proto/.*\.pb\.go$' || true)

  if [ -n "$GO_LINTABLE" ]; then
    if ! require_tool golangci-lint; then
      HAS_FAILURE=1
    else
      log "--- Go (golangci-lint) ---"
      # Extract unique package directories.
      GO_DIRS=$(echo "$GO_LINTABLE" | xargs -I{} dirname "{}" | sort -u)
      log "Linting packages: $GO_DIRS"

      if ! golangci-lint run $GO_DIRS; then
        HAS_FAILURE=1
      else
        log "Go lint passed."
      fi
    fi
  fi
fi

# ---------------------------------------------------------------------------
# Phase 2: TypeScript / React (eslint)
# ---------------------------------------------------------------------------

if [ -n "$STAGED_TS" ]; then
  # Only lint files under web/.
  TS_IN_WEB=$(echo "$STAGED_TS" | grep '^web/' || true)

  if [ -n "$TS_IN_WEB" ]; then
    if ! require_tool npx; then
      HAS_FAILURE=1
    else
      log "--- TypeScript/React (eslint) ---"
      # Strip web/ prefix since eslint runs inside web/ to pick up the flat config.
      TS_RELATIVE=$(echo "$TS_IN_WEB" | sed 's|^web/||')
      log "Linting files: $(echo $TS_RELATIVE | tr '\n' ' ')"

      if ! (cd web && npx eslint $TS_RELATIVE); then
        HAS_FAILURE=1
      else
        log "ESLint passed."
      fi
    fi
  fi
fi

# ---------------------------------------------------------------------------
# Phase 3: Python (flake8 + black)
# ---------------------------------------------------------------------------

if [ -n "$STAGED_PY" ]; then
  # Only lint files under converter/, excluding generated protobuf code.
  PY_LINTABLE=$(echo "$STAGED_PY" | grep '^converter/' | grep -v '^converter/proto/' || true)

  if [ -n "$PY_LINTABLE" ]; then
    log "--- Python (flake8 + black) ---"

    if ! require_tool flake8; then
      HAS_FAILURE=1
    else
      log "Running flake8..."
      if ! flake8 $PY_LINTABLE --max-line-length=120; then
        HAS_FAILURE=1
      else
        log "flake8 passed."
      fi
    fi

    if ! require_tool black; then
      HAS_FAILURE=1
    else
      log "Running black --check..."
      if ! black --check --line-length=120 $PY_LINTABLE; then
        HAS_FAILURE=1
      else
        log "black check passed."
      fi
    fi
  fi
fi

# ---------------------------------------------------------------------------
# Phase 4: OpenAPI (redocly)
# ---------------------------------------------------------------------------

if [ -n "$STAGED_OPENAPI" ]; then
  if ! require_tool npx; then
    HAS_FAILURE=1
  else
    log "--- OpenAPI (redocly) ---"
    if ! npx @redocly/cli lint docs/openapi.yaml --config .redocly.yaml; then
      HAS_FAILURE=1
    else
      log "OpenAPI lint passed."
    fi
  fi
fi

# ---------------------------------------------------------------------------
# Final result
# ---------------------------------------------------------------------------

if [ "$HAS_FAILURE" -eq 1 ]; then
  echo "" >&2
  fail "Pre-commit lint checks failed. Fix the issues above and try again."
  log "Skip with: git commit --no-verify"
  exit 2
fi

echo "" >&2
log "All pre-commit lint checks passed."
exit 0
