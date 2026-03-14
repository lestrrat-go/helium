#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TARGET_DIR="$SCRIPT_DIR/source"

# Pinned commit to ensure reproducible test inputs.
# Override via XSLT30_COMMIT_SHA env var if you need to update it.
XSLT30_COMMIT_SHA="${XSLT30_COMMIT_SHA:-main}"

if [ -d "$TARGET_DIR/.git" ]; then
    echo "XSLT 3.0 test suite repo already exists at $TARGET_DIR."
    cd "$TARGET_DIR"
    if [ "$XSLT30_COMMIT_SHA" != "main" ]; then
        CURRENT_SHA="$(git rev-parse HEAD)"
        if [ "$CURRENT_SHA" = "$XSLT30_COMMIT_SHA" ]; then
            echo "Already at pinned commit $XSLT30_COMMIT_SHA, nothing to do."
            exit 0
        fi
        echo "At $CURRENT_SHA, updating to pinned commit $XSLT30_COMMIT_SHA ..."
        git fetch --filter=blob:none --prune --no-tags origin "$XSLT30_COMMIT_SHA"
    else
        echo "Updating to latest main ..."
        git fetch --filter=blob:none --prune --no-tags origin main
        XSLT30_COMMIT_SHA="origin/main"
    fi
else
    echo "Cloning W3C XSLT 3.0 test suite into $TARGET_DIR ..."
    git clone --filter=blob:none --no-tags https://github.com/w3c/xslt30-test.git "$TARGET_DIR"
    cd "$TARGET_DIR"
    if [ "$XSLT30_COMMIT_SHA" = "main" ]; then
        echo "Done (at latest main)."
        exit 0
    fi
fi

echo "Checking out $XSLT30_COMMIT_SHA ..."
git checkout "$XSLT30_COMMIT_SHA"
echo "Done."
