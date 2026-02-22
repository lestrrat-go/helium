#!/bin/bash
#
# Copies relevant XML test inputs and expected outputs from a libxml2 clone
# into testdata/libxml2-compat/ for use by Go integration tests.
#
# Usage:
#   bash testdata/libxml2/generate.sh
#
# Prerequisites:
#   bash testdata/libxml2/fetch.sh   (clone libxml2 first)
#
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SOURCE="$SCRIPT_DIR/source"
DEST="$REPO_ROOT/testdata/libxml2-compat"

if [ ! -d "$SOURCE/test" ]; then
    echo "Error: libxml2 source not found at $SOURCE"
    echo "Run: bash testdata/libxml2/fetch.sh"
    exit 1
fi

rm -rf "$DEST"
mkdir -p "$DEST"

count=0

# Iterate over files (not directories) in test/
for input in "$SOURCE"/test/*; do
    [ -f "$input" ] || continue

    base="$(basename "$input")"

    # The result file has the same basename, sitting directly in result/
    result="$SOURCE/result/$base"
    [ -f "$result" ] || continue

    # Copy input and expected output
    cp "$input" "$DEST/$base"
    cp "$result" "$DEST/$base.expected"
    count=$((count + 1))
done

echo "Copied $count test cases into $DEST"
