#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TARGET_DIR="$SCRIPT_DIR/source"

# Pinned commit to ensure reproducible test inputs.
# Override via QT3TESTS_COMMIT_SHA env var if you need to update it.
QT3TESTS_COMMIT_SHA="${QT3TESTS_COMMIT_SHA:-83993587711dbd5c18ed846385ec37d079d6e492}"

if [ -d "$TARGET_DIR/.git" ]; then
    echo "QT3 test suite repo already exists at $TARGET_DIR."
    cd "$TARGET_DIR"
    CURRENT_SHA="$(git rev-parse HEAD)"
    if [ "$CURRENT_SHA" = "$QT3TESTS_COMMIT_SHA" ]; then
        echo "Already at pinned commit $QT3TESTS_COMMIT_SHA, nothing to do."
        exit 0
    fi
    echo "At $CURRENT_SHA, updating to pinned commit $QT3TESTS_COMMIT_SHA ..."
    git fetch --filter=blob:none --prune --no-tags origin "$QT3TESTS_COMMIT_SHA"
else
    echo "Cloning W3C QT3 test suite into $TARGET_DIR ..."
    git clone --filter=blob:none --no-tags https://github.com/w3c/qt3tests.git "$TARGET_DIR"
    cd "$TARGET_DIR"
fi

echo "Checking out pinned commit $QT3TESTS_COMMIT_SHA ..."
git checkout "$QT3TESTS_COMMIT_SHA"
echo "Done."
