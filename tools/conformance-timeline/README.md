# conformance-timeline

Retroactively measures W3C conformance for every helium tagged release and
renders a timeline graph of how much of *today's* conformance suites each
release passes.

Suites: **XML 1.0/1.1**, **XSD 1.0**, **XSD 1.1**, **XSLT 3.0**, **XPath/XQuery (QT3)**.

## Output

- `conformance-timeline.html` — self-contained interactive graph (open in a
  browser). Pass-rate lines per suite across releases, legend, hover crosshair,
  and a raw-count data table.
- `data.json` — the aggregated numbers behind the graph.
- `results/*-summary.md` — per tag/suite Pass/Skip/Fail evidence.

## Method

Each release's library code is measured **unmodified** by the current harness
(`../helium-w3c-tests`). The harness is pointed at a pristine `git worktree` of
helium checked out at the tag, using a throwaway `go.work` that `replace`s
helium with that worktree.

The harness's Go test glue targets today's helium API, and the harness repo did
not exist for the older tags (so no date-matched harness is possible). For each
pre-reference tag a small **adapter** bridges API differences — stored as
`harness-adapters/<tag>.patch`, applied on top of the harness HEAD. Adapters:

- **never modify the library under test** (helium stays byte-for-byte as
  released), and
- **never fabricate a passing result.** Where a release lacks a feature (XSD
  1.1, `fn:transform`, schema-awareness, …) the harness degrades honestly and
  those cases fail.

The **denominator** for a suite is the case set the newest (reference) release
enumerates with the unmodified harness — i.e. today's full suite. A release's
score is `passing cases / denominator`. Cases an old release cannot even
enumerate (its parser or schema compiler chokes before running them) therefore
count as not-passing — an honest reflection that the release cannot produce a
passing result for them.

## Cases that kill the release (`crashers/`)

A case that hangs or exhausts memory takes the whole test binary down with it, so
the suite yields *no* result and the release's score for it would be a blank. A
blank hides both what the release passes and the fact that it died. Instead:

- `isolate.sh <tag> <suite>` finds the offending case, records it in
  `crashers/<tag>-<suite>.txt`, skips it, and re-runs until the suite completes.
- `run.sh` skips the recorded cases so every other case gets a verdict.
- `aggregate.py` **counts them as failures** of that release. They are skipped to
  let the suite finish, never to forgive them.

Two rules keep this honest:

1. **`-parallel 1`.** With parallel subtests an out-of-memory lands on whichever
   goroutine allocates next — usually an innocent bystander (a case blamed for an
   OOM here passed in 2s when run alone) — and peak memory is the *sum* of the
   concurrent cases rather than any one case's. Serialized, the case running when
   the binary dies is the case that killed it.
2. **Every culprit is re-checked against the reference tag.** If it dies there too,
   it is our harness/fixture at fault, not the old release; it is recorded as
   `harness` and *excluded* from the release's failure count. Charging it to the
   release would invent a failure — the mirror image of fabricating a pass.

Do not "fix" a blank cell by writing it down as a zero: a suite our own machine
failed to run (an OOM caused by the runner's memory cap or parallelism, not by the
release) is a measurement failure, and recording it as the release's zero is as
dishonest as fabricating a pass. Re-measure it; only a case that genuinely kills
the release is its failure.

## Regenerate

```sh
# one-time: fetch upstream fixtures into the sibling harness
(cd ../helium-w3c-tests && go run ./cmd/w3cgen fetch qt3 xslt30 xsd11 xml)

# run all tags × all suites, then aggregate + render
tools/conformance-timeline/run.sh

# re-render only (after editing template.html or data.json)
python3 tools/conformance-timeline/aggregate.py
```

If a tag/suite dies instead of producing a summary, find what kills it and record it,
then measure normally:

```sh
tools/conformance-timeline/isolate.sh v0.1.0 xslt30   # writes crashers/v0.1.0-xslt30.txt
tools/conformance-timeline/run.sh --suites xslt30 v0.1.0
```

`run.sh` measures with `-parallel 1` and `GOMAXPROCS=2`. That is not incidental: the
suites were originally left unmeasured because an unguarded parallel run was OOM-killed
by the machine, which is a property of the *runner*, not of the release.

`run.sh` caches per tag/suite; pass `--force` to re-run, `--suites "xsd10 xsd11"`
to restrict suites, or tag names to restrict releases.

## Adding a new release

After tagging, run `tools/conformance-timeline/run.sh <newtag>`; it becomes the
new reference (newest tag) automatically and the graph extends. Existing tags
stay cached. Newer releases generally need no adapter patch.
