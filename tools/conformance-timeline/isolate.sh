#!/usr/bin/env bash
#
# Find the cases that a release cannot survive -- it hangs on them, or it exhausts
# memory -- so the rest of the suite can be measured.
#
# Why this exists
# ---------------
# A case that hangs or OOMs takes the whole test binary down with it, so the suite
# yields no result at all and the release's score for that suite is a blank. A blank
# is not honest: it hides both what the release passes AND the fact that it died. So:
# identify the offending case, record it as a FAILURE (crashers/<tag>-<suite>.txt),
# skip it, and re-run until the suite completes. The release is then scored on every
# case, with the ones that killed it counted against it.
#
# Two rules keep the numbers honest:
#
#   1. -parallel 1. With parallel subtests an OOM lands on whichever goroutine happens
#      to allocate next, which is usually an innocent bystander (observed: a case blamed
#      for an OOM passed in 2s when run alone), and peak memory is the SUM of concurrent
#      cases rather than any one case's. Serialized, the case that is running when the
#      binary dies is the case that killed it.
#
#   2. Every culprit is re-checked against the REFERENCE tag before being recorded. If a
#      case also dies on the reference, it is our harness/fixture at fault, not the old
#      release -- charging it to the release would invent a failure. Those are recorded
#      as "harness" and excluded from the release's failure count.
#
# Usage: isolate.sh <tag> <suite> [max-rounds]
set -uo pipefail

TAG="${1:?usage: isolate.sh <tag> <suite> [max-rounds]}"
SUITE="${2:?usage: isolate.sh <tag> <suite> [max-rounds]}"
MAXROUNDS="${3:-15}"

STALL_SECS=${STALL_SECS:-180}   # no new verdict for this long => hung case
MEM_KB=${MEM_KB:-12582912}      # 12 GiB address-space cap; a runaway dies loudly

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HELIUM_ROOT="$(git -C "$HERE" rev-parse --show-toplevel)"
MAIN_ROOT="$(cd "$(git -C "$HELIUM_ROOT" rev-parse --path-format=absolute --git-common-dir)/.." && pwd)"
HARNESS="$(cd "$MAIN_ROOT/.." && pwd)/helium-w3c-tests"
CRASHERS="$HERE/crashers"
WORK="$HERE/results"
mkdir -p "$CRASHERS" "$WORK"

case "$SUITE" in
  qt3)    PKG=./qt3;    ROOT='^TestQT3W3C$';    PREFIX=TestQT3W3C ;;
  xsd10)  PKG=./xsd10;  ROOT='^TestXSD10W3C$';  PREFIX=TestXSD10W3C ;;
  xsd11)  PKG=./xsd11;  ROOT='^TestXSD11W3C$';  PREFIX=TestXSD11W3C ;;
  xslt30) PKG=./xslt3;  ROOT='^TestXSLT30W3C$'; PREFIX=TestXSLT30W3C ;;
  xml)    PKG=./xml;    ROOT='^TestXMLW3C$';    PREFIX=TestXMLW3C ;;
  *) echo "unknown suite $SUITE" >&2; exit 2 ;;
esac

REFTAG=$(git -C "$MAIN_ROOT" tag --sort=creatordate | grep -E '^v[0-9]' | tail -1)
GO_MINOR="$(go version | awk '{print $3}' | sed 's/^go//')"

# hbase_for <tag> -> harness worktree to run that tag against (adapter-patched if needed)
hbase_for() {
  local tag=$1
  if [ "$tag" = "$REFTAG" ] || [ ! -f "$HERE/harness-adapters/$tag.patch" ]; then
    echo "$HARNESS"; return
  fi
  local hb="$HARNESS/.worktrees/adapt-$tag"
  if [ ! -d "$hb" ]; then
    git -C "$HARNESS" worktree add -q --detach "$hb" HEAD >&2
    git -C "$hb" apply "$HERE/harness-adapters/$tag.patch" >&2
    ln -sfn "$HARNESS/testdata" "$hb/testdata"
  fi
  echo "$hb"
}

# workfile_for <tag> -> throwaway go.work pinning helium to that tag's pristine worktree
workfile_for() {
  local tag=$1 hb=$2
  local probe="$MAIN_ROOT/.worktrees/probe-$tag"
  [ -d "$probe" ] || git -C "$MAIN_ROOT" worktree add -q --detach "$probe" "$tag" >&2
  local wf="$WORK/$tag.work"
  printf 'go %s\nuse %s\nreplace github.com/lestrrat-go/helium => %s\n' "$GO_MINOR" "$hb" "$probe" > "$wf"
  echo "$wf"
}

# run_suite <hbase> <workfile> <json-out> <skip-pattern>  -> writes json; echoes "ok"|"died"
# Kills the binary if it stops producing verdicts for STALL_SECS (a hang produces no
# output at all, so a stalled stream is the only signal a spinning case gives us).
run_suite() {
  local hb=$1 wf=$2 json=$3 skip=$4
  local skipargs=()
  [ -n "$skip" ] && skipargs=(-skip "$skip")
  ( ulimit -v "$MEM_KB"
    GOWORK="$wf" GOMAXPROCS=2 \
      go -C "$hb" test -json -run "$ROOT" -parallel 1 -timeout 170m "${skipargs[@]}" "$PKG"
  ) > "$json" 2>"$json.err" &
  local gopid=$!

  while kill -0 "$gopid" 2>/dev/null; do
    local before after
    before=$(stat -c%s "$json" 2>/dev/null || echo 0)
    sleep "$STALL_SECS"
    kill -0 "$gopid" 2>/dev/null || break
    after=$(stat -c%s "$json" 2>/dev/null || echo 0)
    if [ "$before" = "$after" ]; then
      echo "    [watchdog] no verdict for ${STALL_SECS}s -- killing (hung case)" >&2
      pkill -9 -P "$gopid" 2>/dev/null
      kill -9 "$gopid" 2>/dev/null
      break
    fi
  done
  wait "$gopid" 2>/dev/null
}

# analyze <json> -> prints "done" or the culprit case id (bare, no Test prefix)
analyze() {
  python3 - "$1" "$PREFIX" <<'PY'
import json,sys
jf,prefix=sys.argv[1],sys.argv[2]
evs=[]
for line in open(jf,errors='replace'):
    line=line.strip()
    if not line.startswith('{'): continue
    try: evs.append(json.loads(line))
    except Exception: pass
done=set(); order=[]; fatal=[]
FATAL=('fatal error','out of memory','panic:','test timed out')
for e in evs:
    t,a=e.get('Test'),e.get('Action')
    if not t or '/' not in t: continue
    if a=='run': order.append(t)
    elif a in ('pass','fail','skip'): done.add(t)
    elif a=='output' and any(f in e.get('Output','') for f in FATAL): fatal.append(t)
unfinished=[t for t in order if t not in done]
if not unfinished:
    print("done"); sys.exit(0)
# OOM/panic: the runtime names the running case. Hang: no output at all, so the culprit
# is the last case that was handed control (-parallel 1 => exactly one runs at a time).
cand=[t for t in fatal if t in set(unfinished)]
if cand:
    print("CULPRIT\t%s\t%s" % (cand[0][len(prefix)+1:], "oom"))
else:
    last=None
    for e in evs:
        t,a=e.get('Test'),e.get('Action')
        if t and '/' in t and a in ('run','cont','output') and t not in done: last=t
    if last: print("CULPRIT\t%s\t%s" % (last[len(prefix)+1:], "hang"))
    else: print("unknown")
PY
}

# verdict_on <tag> <case> -> pass|fail|died   (single case, alone)
verdict_on() {
  local tag=$1 case=$2 hb wf rc
  hb=$(hbase_for "$tag"); wf=$(workfile_for "$tag" "$hb")
  ( ulimit -v "$MEM_KB"
    GOWORK="$wf" GOMAXPROCS=2 timeout 120 \
      go -C "$hb" test -run "^$PREFIX\$/^$(printf '%s' "$case" | sed 's/[].[^$()*+?{}|\\]/\\&/g')\$" \
        -parallel 1 "$PKG" >/dev/null 2>&1 )
  rc=$?
  case $rc in
    0) echo pass ;;
    1) echo fail ;;      # ran, asserted a wrong result -- still a measurable verdict
    *) echo died ;;      # 124 = timeout/hang, 2 = fatal
  esac
}

HBASE=$(hbase_for "$TAG"); WF=$(workfile_for "$TAG" "$HBASE")
OUT="$CRASHERS/$TAG-$SUITE.txt"
: > "$OUT.tmp"
skips=()

echo "tag=$TAG suite=$SUITE reference=$REFTAG harness=$HBASE"
for round in $(seq 1 "$MAXROUNDS"); do
  pat=""
  if [ ${#skips[@]} -gt 0 ]; then
    pat="$PREFIX/^($(IFS='|'; echo "${skips[*]}"))\$"
  fi
  echo "=== round $round (${#skips[@]} case(s) skipped)"
  json="$WORK/$TAG-$SUITE-iso.json"
  run_suite "$HBASE" "$WF" "$json" "$pat"

  res=$(analyze "$json")
  if [ "$res" = "done" ]; then
    echo "=== suite COMPLETED: every case reached a verdict"
    mv "$OUT.tmp" "$OUT"
    [ -s "$OUT" ] && { echo "--- recorded crashers:"; cat "$OUT"; } || echo "--- no crashers; suite runs clean"
    exit 0
  fi
  if [ "$res" = "unknown" ]; then
    echo "=== died but could not attribute to a case; see $json.err" >&2
    tail -3 "$json.err" >&2; exit 3
  fi

  case_id=$(printf '%s' "$res" | cut -f2)
  mode=$(printf '%s' "$res" | cut -f3)
  echo "    culprit: $case_id ($mode)"

  # Guard rail: does it also die on the reference? Then it is OUR bug, not the release's.
  ref=$(verdict_on "$REFTAG" "$case_id")
  if [ "$ref" = "died" ]; then
    echo "    !! also dies on reference $REFTAG -- harness/fixture fault, NOT charged to $TAG"
    printf '%s\tharness\t%s also dies on reference %s\n' "$case_id" "$mode" "$REFTAG" >> "$OUT.tmp"
  else
    echo "    confirmed: $mode on $TAG, but '$ref' on reference $REFTAG -> genuine failure"
    printf '%s\tfail\t%s on %s; reference %s=%s\n' "$case_id" "$mode" "$TAG" "$REFTAG" "$ref" >> "$OUT.tmp"
  fi
  skips+=("$(printf '%s' "$case_id" | sed 's/[].[^$()*+?{}|\\]/\\&/g')")
done

echo "=== hit max rounds ($MAXROUNDS) without completing" >&2
mv "$OUT.tmp" "$OUT"
exit 4
