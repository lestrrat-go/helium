# conformance-timeline

Retroactively measures W3C conformance for every helium tagged release and
renders a timeline graph of how much of *today's* conformance suites each
release passes.

Suites: **XML 1.0/1.1**, **XSD 1.0**, **XSD 1.1**, **XSLT 3.0**, **XPath/XQuery (QT3)**,
and three XML-Signature interop suites — **XMLDSig 2ed interop** (`xmldsig2ed`),
**XMLDSig 1.1 interop** (`xmldsig11`), and **XMLDSig merlin baseline**
(`merlinxmldsig`).

## Output

- `conformance-timeline.html` — self-contained interactive graph (open in a
  browser). Pass-rate lines per suite across releases, legend, hover crosshair,
  and a raw-count data table.
- `data.json` — the aggregated numbers behind the graph (committed: it is what
  `CONFORMANCE.md` and the SVG are rendered from, so the doc can be re-rendered
  without re-measuring). `aggregate.py` **merges** into it: freshly measured
  `(tag, suite)` cells override, everything else carries over verbatim. Because
  `results/` is a local cache a fresh checkout does not have, the committed
  `data.json` is the only record of past measurements — so re-measuring one suite
  (or one tag) never wipes the rest.
- `CONFORMANCE.md` (repo root) + `conformance-timeline.svg` — the committed report.
- `results/` — raw per tag/suite JUnit + summaries. **Not committed**: it is bulky and
  fully recalculable by re-running `run.sh` (which takes hours). Local artifact only.

## Method

Each release's library code is measured **unmodified** by the current harness
(`../helium-w3c-tests`). The harness is pointed at a pristine `git worktree` of
helium checked out at the tag, using a throwaway `go.work` that `replace`s
helium with that worktree.

The harness's Go test glue targets today's helium API, and the harness repo did
not exist for the older tags (so no date-matched harness is possible). Where a tag
differs from that API a small **adapter** bridges the difference — stored as
`harness-adapters/<tag>.patch`, applied on top of the harness HEAD. Adapters:

- **never modify the library under test** (helium stays byte-for-byte as
  released), and
- **never fabricate a passing result.** Where a release lacks a feature (XSD
  1.1, `fn:transform`, schema-awareness, …) the harness degrades honestly and
  those cases fail.

The newest tag is **not** special-cased: `run.sh` applies a tag's patch whenever
one exists, reference tag included. The XML-Signature harness API (the
`xmldsig1.KeyInfoData` fields for DName / IssuerSerial / DSA key resolution)
postdates *every* tagged release, so even the reference tag carries an
xmldsig-only adapter that degrades the missing key-resolution — the affected
cases fail honestly, which is why the reference does not score 100% on the
xmldsig suites. See *XML-Signature suites* below.

The **denominator** for a suite is the set of cases the newest (reference) release
actually **runs** with the unmodified harness: today's suite **minus the cases the
harness skips as inapplicable**. A release's score is `passing cases / denominator`.

Inapplicable cases are excluded because they are not failures and never could be —
they test something helium deliberately does not implement:

| Suite | Excluded | Why |
|-------|---------:|-----|
| XML | 310 | test applies to XML 1.0 editions 1–4; helium implements the 5th |
| XML | 274 | XML 1.1 / Namespaces 1.1; helium targets XML 1.0 |
| XSLT 3.0 | 781 | harness dependency gates (optional features out of scope) |
| QT3 | 141 | harness dependency gates |
| XSD 1.0 / 1.1 | 0 | — |

Counting them against a release is as dishonest as counting them as passes: it
silently records "not applicable" as "failed". With them in the denominator the
reference release scores 77.4% on XML; excluded, it scores its true **100%**.

What *does* count as not-passing: cases an old release cannot enumerate (its parser
or schema compiler chokes before running them), cases it **skips that the reference
runs** (that is the release's own gap, not an exemption), and cases that hang or
exhaust its memory.

## Expected failures (`xfail`)

An expected failure is a *passing Go test* (the harness asserts the divergence), so it
lands in the JUnit **pass bucket** with the marker only in `system-out`. It is not a
passing conformance result — helium does not produce what the case asks for — so it is
counted as **not-passing** and shown as `⚠` rather than rounded into a perfect score.

Reference (v0.5.1) xfails: XML 8, XSD 1.0 16, XSD 1.1 1, QT3 4, XSLT 3.0 0.

Note the committed summaries are inconsistent about these: `summary-xml.md` breaks
XFail out as its own row (Pass 1993 + XFail 8), while `xsd/summary-xsd10.md`,
`xsd/summary-xsd11.md` and `xpath3/summary-qt3.md` fold theirs into **Pass** (so their
"Pass 14399 / Fail 0" silently includes 16 expected failures). This tool counts them
uniformly, which is why XSD 1.0 reads 99.9% here and 100% there.

## Performance-gated cases (`HELIUM_SLOW_TESTS`)

The harness skips 481 slow XSLT cases (streaming, heavy source docs) unless
`HELIUM_SLOW_TESTS=1`. These are **not** inapplicable — they are simply not run — so
they must not vanish from the denominator.

helium started actually running them in **v0.4.0** (PR #1015, 2026-07-03), the first
release to ship `xslt3/results-xslt30-slow.xml`. Both scripts detect that cutoff from
the release's own content rather than hardcoding a tag:

- **v0.4.0 and later** are measured *with* `HELIUM_SLOW_TESTS=1`, so the 481 cases get
  real verdicts (all pass: 12,827 / 0 fail / 300 skip, matching `xslt3/CONFORMANCE.md`).
- **Earlier releases** never ran them, so the 481 land inside the applicable set as
  cases with no passing result and are **counted as failures** (`unrun` in `data.json`).

The 300 the reference still skips in slow mode are the genuinely inapplicable ones
(2.0-vs-3.0 divergences, "feature required absent") plus cases too slow to run at all.

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

## XML-Signature suites

The three XML-Signature interop suites (`xmldsig2ed`, `xmldsig11`, `merlinxmldsig`)
exercise helium's `c14n`, `xpath1`, and `xmldsig1` packages. Their harness glue
targets an `xmldsig1` API that landed *after* v0.6.0, so every tagged release needs
some adapting; each tag's patch degrades honestly, never fabricating a pass:

- **All tags** lack the `KeyInfoData` fields for DName / IssuerSerial / DSA key
  resolution, so the adapter stubs that resolution to "no match": the dname,
  X509IssuerSerial, and DSA cases fail honestly. This is why even the reference
  tag scores below 100% on these suites.
- **v0.0.2 – v0.2.0** additionally lack `Verifier.AllowSHA1`; the adapter drops the
  call (those releases allow SHA-1 by default, so the interop vectors still verify).
- **v0.0.1** has no `xmldsig1` package and no `Parser.FS` at all. The adapter builds
  the suite without `xmldsig1`: signature verification degrades to an always-error
  stub (every signature case fails honestly), while the pure Canonical XML 1.1
  node-set cases still run for real against `c14n` + `xpath1`.

## Regenerate

```sh
# one-time: fetch upstream fixtures into the sibling harness
(cd ../helium-w3c-tests && go run ./cmd/w3cgen fetch qt3 xslt30 xsd11 xml \
    xmldsig2ed xmldsig11 merlinxmldsig)

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

## Measuring an untagged release candidate

The release gate (`.github/workflows/release.yml`) requires the version being
released to already have a row in `data.json` before the tag exists. Seed that row
by measuring the candidate commit under its version label:

```sh
tools/conformance-timeline/run.sh --ref "$(git rev-parse HEAD)" --as v0.7.0
```

`--ref/--as` checks out the committish under the label, dates the row from the
ref's commit date, and marks it as an **untagged** release candidate (shown with a
⚑ in `CONFORMANCE.md`). After the release tags the commit, a later
`run.sh v0.7.0` re-measures it at the real tag and supersedes the candidate row.

## Adding a new release

After tagging, run `tools/conformance-timeline/run.sh <newtag>`; it becomes the
new reference (newest tag) automatically and the graph extends. Existing tags
stay cached. A new release needs no adapter patch for the XML/XSD/XSLT/QT3 suites,
but until a tag ships the post-v0.6.0 `xmldsig1` key-resolution API it still needs
the xmldsig-only adapter (copy the newest tag's patch, which touches only
`xmldsig/`).
