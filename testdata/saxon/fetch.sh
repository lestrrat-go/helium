#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TARGET_DIR="$SCRIPT_DIR/source"

# Pinned commit to ensure reproducible test/reference inputs.
# Override via SAXON_COMMIT_SHA env var if you need to update it.
SAXON_COMMIT_SHA="${SAXON_COMMIT_SHA:-28d25a31c7b29e263149ae7ccd2c991d56c57e8d}"

if [ -d "$TARGET_DIR/.git" ]; then
    echo "Saxon-HE already cloned at $TARGET_DIR, skipping."
    exit 0
fi

echo "Cloning Saxon-HE into $TARGET_DIR ..."
git clone https://github.com/Saxonica/Saxon-HE.git "$TARGET_DIR"
echo "Checking out pinned commit $SAXON_COMMIT_SHA ..."
cd "$TARGET_DIR"
git checkout "$SAXON_COMMIT_SHA"
echo "Done."
