#!/bin/bash
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TARGET_DIR="$SCRIPT_DIR/source"

if [ -d "$TARGET_DIR/.git" ]; then
    echo "QT3 test suite already cloned at $TARGET_DIR, skipping."
    exit 0
fi

echo "Cloning W3C QT3 test suite into $TARGET_DIR ..."
git clone --depth 1 https://github.com/w3c/qt3tests.git "$TARGET_DIR"
echo "Done."
