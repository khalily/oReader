#!/bin/bash
# Load versions.env variables into GITHUB_ENV for CI workflows
# Usage: bash scripts/ci-load-versions.sh
set -euo pipefail

VERSIONS_FILE="${1:-versions.env}"

if [ ! -f "$VERSIONS_FILE" ]; then
    echo "ERROR: $VERSIONS_FILE not found"
    exit 1
fi

echo "Loading versions from $VERSIONS_FILE:"
while IFS='=' read -r key value; do
    # Skip comments and empty lines
    [[ "$key" =~ ^[[:space:]]*# ]] && continue
    [[ -z "$key" ]] && continue
    echo "  $key=$value"
    echo "$key=$value" >> "${GITHUB_ENV:-/dev/null}"
done < "$VERSIONS_FILE"
