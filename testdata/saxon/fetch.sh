#!/bin/bash
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TARGET_DIR="$SCRIPT_DIR/source"

if [ -d "$TARGET_DIR/.git" ]; then
    echo "Saxon-HE already cloned at $TARGET_DIR, skipping."
    exit 0
fi

echo "Cloning Saxon-HE into $TARGET_DIR ..."
git clone --depth 1 https://github.com/Saxonica/Saxon-HE.git "$TARGET_DIR"
echo "Done."
