#!/bin/bash

# oReader - Run All Tests (Go + Frontend + Converter)
# Usage: ./test-all.sh

set -e

PASS=0
FAIL=0
RESULTS=""

run_test() {
    local name="$1"
    local cmd="$2"
    echo ""
    echo "=========================================="
    echo "  Running: $name"
    echo "=========================================="
    if eval "$cmd"; then
        RESULTS="$RESULTS\n  [PASS] $name"
        PASS=$((PASS + 1))
    else
        RESULTS="$RESULTS\n  [FAIL] $name"
        FAIL=$((FAIL + 1))
    fi
}

echo "oReader Test Suite"
echo "=================="

# Go tests
run_test "Go Backend Tests" "go test -race ./internal/..."

# Frontend tests
run_test "Frontend Tests" "cd web && npm test -- --run"

# Converter tests (only if Python is available)
if command -v python &> /dev/null; then
    run_test "Converter Tests" "cd converter && python -m pytest tests/ -v"
else
    echo ""
    echo "Skipping Converter Tests (Python not found)"
    RESULTS="$RESULTS\n  [SKIP] Converter Tests (Python not found)"
fi

# Summary
echo ""
echo "=========================================="
echo "  Test Summary"
echo "=========================================="
echo -e "$RESULTS"
echo ""
echo "  Passed: $PASS  Failed: $FAIL"
echo "=========================================="

if [ $FAIL -gt 0 ]; then
    exit 1
fi
