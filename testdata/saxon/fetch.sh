#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TARGET_DIR="$SCRIPT_DIR/source"

# Pinned commit to ensure reproducible test/reference inputs.
# Override via SAXON_COMMIT_SHA env var if you need to update it.
SAXON_COMMIT_SHA="${SAXON_COMMIT_SHA:-28d25a31c7b29e263149ae7ccd2c991d56c57e8d}"

if [ -d "$TARGET_DIR/.git" ]; then
    echo "Saxon-HE repo already exists at $TARGET_DIR."
    cd "$TARGET_DIR"
    CURRENT_SHA="$(git rev-parse HEAD)"
    if [ "$CURRENT_SHA" = "$SAXON_COMMIT_SHA" ]; then
        echo "Already at pinned commit $SAXON_COMMIT_SHA, nothing to do."
        exit 0
    fi
    echo "At $CURRENT_SHA, updating to pinned commit $SAXON_COMMIT_SHA ..."
    git fetch --depth=1 --no-tags origin "$SAXON_COMMIT_SHA"
else
    echo "Cloning Saxon-HE into $TARGET_DIR ..."
    git clone --depth=1 --no-tags https://github.com/Saxonica/Saxon-HE.git "$TARGET_DIR"
    cd "$TARGET_DIR"
fi

echo "Checking out pinned commit $SAXON_COMMIT_SHA ..."
git checkout "$SAXON_COMMIT_SHA"
echo "Done."
