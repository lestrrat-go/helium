#!/bin/bash
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TARGET_DIR="$SCRIPT_DIR/source"

if [ -d "$TARGET_DIR/.git" ]; then
    echo "libxml2 already cloned at $TARGET_DIR, skipping."
    exit 0
fi

echo "Cloning libxml2 into $TARGET_DIR ..."
git clone --depth 1 https://gitlab.gnome.org/GNOME/libxml2.git "$TARGET_DIR"
echo "Done."
