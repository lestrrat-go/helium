# conformance-timeline

Retroactively measures W3C conformance for every helium tagged release and
renders a timeline graph of how much of *today's* conformance suites each
release passes.

Suites: **XSD 1.0**, **XSD 1.1**, **XSLT 3.0**, **XPath/XQuery (QT3)**.

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

## Regenerate

```sh
# one-time: fetch upstream fixtures into the sibling harness
(cd ../helium-w3c-tests && go run ./cmd/w3cgen fetch qt3 xslt30 xsd11)

# run all tags × all suites, then aggregate + render
tools/conformance-timeline/run.sh

# re-render only (after editing template.html or data.json)
python3 tools/conformance-timeline/aggregate.py
```

`run.sh` caches per tag/suite; pass `--force` to re-run, `--suites "xsd10 xsd11"`
to restrict suites, or tag names to restrict releases.

## Adding a new release

After tagging, run `tools/conformance-timeline/run.sh <newtag>`; it becomes the
new reference (newest tag) automatically and the graph extends. Existing tags
stay cached. Newer releases generally need no adapter patch.
