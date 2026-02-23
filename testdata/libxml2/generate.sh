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

# Copy SAX2 golden files
sax2count=0
for input in "$DEST"/*; do
    [ -f "$input" ] || continue
    base="$(basename "$input")"

    # Skip .expected and .err files
    case "$base" in
        *.expected|*.err) continue ;;
    esac

    # The SAX2 result file is result/<base>.sax2
    sax2result="$SOURCE/result/$base.sax2"
    [ -f "$sax2result" ] || continue

    cp "$sax2result" "$DEST/$base.sax2.expected"
    sax2count=$((sax2count + 1))
done

echo "Copied $sax2count SAX2 golden files into $DEST"

# Fix C buffer artifacts in SAX2 golden files.
# xmllint's C code uses %.4s to print attribute values, which reads 4 bytes
# from a raw pointer regardless of string length, leaking adjacent buffer
# contents. Normalize by trimming displayed value to the reported length.
fixed=0
for f in "$DEST"/*.sax2.expected; do
    [ -f "$f" ] || continue
    if perl -0777 -i -pe '
        s{=\047(.{1,4})\.\.\.\047(, )(\d+)}{
            my($d,$s,$n)=($1,$2,$3);
            $d = substr($d,0,$n) if length($d) > $n;
            "=\047${d}...\047$s$n"
        }gse
    ' "$f"; then
        fixed=$((fixed + 1))
    fi
done
echo "Normalized $fixed SAX2 golden files (C buffer truncation fix)"
