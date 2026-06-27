#!/bin/sh
# Clone the W3C XML Schema 1.1 test suite (XSTS) into source/ (gitignored).
# Dev-time only; needed to (re)generate the embedded cases via `go run ./tools/xstsgen`.
# The harness in xsd/w3c_xsts_test.go runs from the embedded gen files, not source/.
set -e
dir="$(cd "$(dirname "$0")" && pwd)"
if [ -d "$dir/source/.git" ]; then
  git -C "$dir/source" pull --ff-only
else
  git clone --depth 1 https://github.com/w3c/xsdtests "$dir/source"
fi
