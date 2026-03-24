# Loop: fix test failures one at a time, forever

## Scope

Run in parent worktree:

- `/home/lestrrat/dev/src/github.com/lestrrat-go/helium/.worktrees/feat-xslt3`

## Goal

Fix test failures one at a time. Never stop. When one is fixed, find the next.

## Timeout Safety

The full test suite may contain unknown hangs. Prefer targeted `-run` patterns.
If running a broad suite, always use `-timeout 600s` and run in background.

## Hard Rules

- ONE failure at a time. No batching. No parallel subagents.
- Find the first actionable failure, launch a subagent to fix it, merge, repeat.
- DO NOT plan ahead. DO NOT create task lists. DO NOT categorize failures.
  Just fix the next one.
- If a subagent fails to fix the same item twice, mark it non-actionable and
  move to the next.

## What Is Not Actionable

Skip these — they require separate feature work:

- accumulator-dependent failures
- `use-when` compile-time evaluation
- UCA collation (requires ICU)
- HTTP document loading (requires network)
- schema-element/schema-attribute node tests (requires deep schema-aware typing)
- backwards_compatibility feature tests (intentionally unsupported)
- failure too truncated to identify root cause

## Loop

### 1. Run one test bucket

Pick a bucket. Run it. Store output in `.tmp/`.

```bash
go test ./xslt3/ -run 'TestW3C_<category>$' -v -count=1 -p 1 -timeout 600s > .tmp/loop.txt 2>&1
```

Rotate buckets each iteration so you don't keep hitting the same passing ones.

### 2. Find first actionable failure

Look at the output. Find the first `--- FAIL:` that isn't in the non-actionable list.
Note the exact test name and error message.

### 3. Launch one subagent

Give it:

- The exact failing test name and error text
- The source files likely involved
- Instructions to: read pre-read docs, create worktree from `feat-xslt3`,
  fix the issue, verify with targeted `-run`, commit, report results,
  do NOT merge or push

### 4. Merge the fix

- Commit if uncommitted
- Verify build: `go build ./xslt3/`
- Merge into `feat-xslt3`, resolve conflicts
- Verify build again
- Rerun the test to confirm
- Clean up worktree + branch
- If fix failed: try once more, then mark non-actionable and move on

### 5. Go to 1
