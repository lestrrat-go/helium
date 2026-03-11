#!/usr/bin/env bash
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TARGET_DIR="$SCRIPT_DIR/source"

# Pinned commit to ensure reproducible test inputs.
# Override via QT3TESTS_COMMIT_SHA env var if you need to update it.
QT3TESTS_COMMIT_SHA="${QT3TESTS_COMMIT_SHA:-83993587711dbd5c18ed846385ec37d079d6e492}"

if [ -d "$TARGET_DIR/.git" ]; then
    echo "QT3 test suite already cloned at $TARGET_DIR, skipping."
    exit 0
fi

echo "Cloning W3C QT3 test suite into $TARGET_DIR ..."
git clone https://github.com/w3c/qt3tests.git "$TARGET_DIR"
echo "Checking out pinned commit $QT3TESTS_COMMIT_SHA ..."
cd "$TARGET_DIR"
git checkout "$QT3TESTS_COMMIT_SHA"
echo "Done."
