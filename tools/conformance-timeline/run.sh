#!/usr/bin/env bash
#
# Retroactively measure W3C conformance for every helium tagged release and
# regenerate data.json + conformance-timeline.html.
#
# How it works
# ------------
# Each release's library code is measured UNMODIFIED. The current conformance
# harness (sibling ../helium-w3c-tests) is pointed at a pristine detached
# worktree of helium checked out at the tag, via a throwaway go.work `replace`.
# The harness's test glue targets today's helium API, so for an older tag it is
# adapted on a per-tag branch: this script recreates that harness worktree by
# applying tools/conformance-timeline/harness-adapters/<tag>.patch on top of the
# harness's current HEAD. The newest (reference) tag needs no patch — it runs the
# unmodified harness. Adapters bridge API differences ONLY; they never touch the
# library under test and never fabricate a passing result (features a release
# lacks fail honestly).
#
# The harness did not exist for the pre-reference tags and no date-matched
# harness is possible, so every tag is measured against today's harness — which
# is exactly the "how does each release score under today's suites" view.
#
# Prereqs
# -------
#   * sibling ../helium-w3c-tests checkout, clean, with fixtures fetched:
#       (cd ../helium-w3c-tests && go run ./cmd/w3cgen fetch qt3 xslt30 xsd11 xml)
#   * python3 (for aggregate.py)
#
# Usage: tools/conformance-timeline/run.sh [--force] [--suites "a b"] [tag ...]
#   --force          re-run even if a cached summary exists
#   --suites "..."   subset of: xml xsd10 xsd11 xslt30 qt3  (default: all)
#   tag ...          restrict to specific tags (default: all v* tags)

set -euo pipefail

HELIUM_ROOT="$(git -C "$(dirname "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)"
MAIN_ROOT="$(cd "$(git -C "$HELIUM_ROOT" rev-parse --path-format=absolute --git-common-dir)/.." && pwd)"
HARNESS="$(cd "$MAIN_ROOT/.." && pwd)/helium-w3c-tests"
OUTDIR="$HELIUM_ROOT/tools/conformance-timeline"
RESULTS="$OUTDIR/results"
ADAPTERS="$OUTDIR/harness-adapters"
WTROOT="$MAIN_ROOT/.worktrees"
HWTROOT="$HARNESS/.worktrees"

FORCE=0; SUITES="xml xsd10 xsd11 xslt30 qt3"; ONLY_TAGS=""
while [ $# -gt 0 ]; do
  case "$1" in
    --force) FORCE=1 ;;
    --suites) SUITES="$2"; shift ;;
    *) ONLY_TAGS="$ONLY_TAGS $1" ;;
  esac; shift
done

[ -d "$HARNESS" ] || { echo "harness not found at $HARNESS" >&2; exit 1; }
mkdir -p "$RESULTS"

TAGS=$(git -C "$MAIN_ROOT" tag --sort=creatordate | grep -E '^v[0-9]')
[ -n "${ONLY_TAGS// }" ] && TAGS="$ONLY_TAGS"
REFERENCE_TAG=$(git -C "$MAIN_ROOT" tag --sort=creatordate | grep -E '^v[0-9]' | tail -1)
GO_MINOR="$(go version | awk '{print $3}' | sed 's/^go//')"

echo "helium:    $MAIN_ROOT"
echo "harness:   $HARNESS @ $(git -C "$HARNESS" rev-parse --short HEAD)"
echo "reference: $REFERENCE_TAG"
echo "suites:    $SUITES"
echo

for tag in $TAGS; do
  # pristine helium checkout at the tag
  hwt="$WTROOT/probe-$tag"
  [ -d "$hwt" ] || git -C "$MAIN_ROOT" worktree add -q --detach "$hwt" "$tag"

  # harness worktree: unmodified for the reference tag, patched otherwise
  patch="$ADAPTERS/$tag.patch"
  if [ "$tag" = "$REFERENCE_TAG" ] || [ ! -f "$patch" ]; then
    hbase="$HARNESS"
  else
    hbase="$HWTROOT/adapt-$tag"
    if [ ! -d "$hbase" ]; then
      git -C "$HARNESS" worktree add -q --detach "$hbase" HEAD
      git -C "$hbase" apply "$patch"
    fi
    # fetched fixtures are gitignored and live only in the primary checkout
    ln -sfn "$HARNESS/testdata" "$hbase/testdata"
  fi

  work="$RESULTS/$tag.work"
  printf 'go %s\nuse %s\nreplace github.com/lestrrat-go/helium => %s\n' "$GO_MINOR" "$hbase" "$hwt" > "$work"

  for suite in $SUITES; do
    case "$suite" in
      qt3)    ROOTNAME=TestQT3W3C ;;
      xsd10)  ROOTNAME=TestXSD10W3C ;;
      xsd11)  ROOTNAME=TestXSD11W3C ;;
      xslt30) ROOTNAME=TestXSLT30W3C ;;
      xml)    ROOTNAME=TestXMLW3C ;;
      *) echo "unknown suite $suite" >&2; continue ;;
    esac
    summ="$RESULTS/$tag-$suite-summary.md"
    if [ "$FORCE" -eq 0 ] && [ -s "$summ" ] && grep -q '^| \*\*Total\*\*' "$summ" \
       && ! grep -q '^| \*\*Total\*\* |[^0-9]*1 |' "$summ"; then
      echo "[$tag/$suite] cached"; continue
    fi
    echo "[$tag/$suite] running..."
    # Cases this release cannot survive (hang / OOM) would take the whole test binary
    # down and leave the suite unmeasured. isolate.sh identified them; skip them here so
    # every other case gets a verdict. They are NOT forgiven -- aggregate.py counts them
    # as failures of this release (see crashers/ and aggregate.py:load_crashers).
    skipargs=()
    crashfile="$OUTDIR/crashers/$tag-$suite.txt"
    if [ -s "$crashfile" ]; then
      # go test splits a -skip pattern on '/' and matches one part per name segment, so a
      # case id that contains '/' (xsd and qt3 ids do) must NOT sit inside a
      # "Root/^(a|b)$" group -- its slashes would be read as separators and the case would
      # never actually be skipped. Emit a top-level alternation of full anchored paths;
      # go splits each alternative on its own.
      ids=$(grep -v '^#' "$crashfile" | cut -f1 | grep -v '^$' \
            | sed 's/[].[^$()*+?{}|\\]/\\&/g' \
            | sed "s|^|^$ROOTNAME/|; s|\$|\$|" | paste -sd'|')
      [ -n "$ids" ] && skipargs=(-skip "$ids")
      echo "[$tag/$suite] skipping $(grep -cv '^#' "$crashfile") crasher(s), counted as failures"
    fi
    # -parallel 1: peak memory is one case's, not the sum of concurrent ones, and a
    # crash is attributable to the case that caused it.
    GOMAXPROCS="${GOMAXPROCS:-2}" GOWORK="$work" go -C "$hbase" run ./cmd/w3ctest \
      -out "$RESULTS/$tag-$suite-junit.xml" -summary "$summ" "$suite" \
      -parallel 1 "${skipargs[@]}" \
      >"$RESULTS/$tag-$suite.log" 2>&1 || true
    if [ -s "$summ" ]; then echo "[$tag/$suite] done"; else
      echo "[$tag/$suite] ERROR — see $RESULTS/$tag-$suite.log" >&2; fi
  done
done

echo; echo "Aggregating ..."
python3 "$OUTDIR/aggregate.py"
