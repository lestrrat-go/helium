# XSLT 3.0 conformance declaration

This document is the explicit, auditable statement of what `xslt3` claims to
conform to. It exists to remove two ambiguities a production reviewer will ask
about: **(1)** exactly which XSLT 3.0 conformance level(s) helium claims, and
**(2)** what the W3C-suite *skip* count actually means. It is a current-state
cache; regenerate the evidence (see below) when the numbers change.

## The claim

`xslt3` targets and implements **Basic XSLT 3.0** — the "Basic XSLT Processor"
conformance level defined in [W3C XSLT 3.0 §27](https://www.w3.org/TR/xslt-30/#conformance).
"Basic XSLT Processor" is the **only required** conformance level; the spec
defines seven further **optional** levels. Helium claims Basic only. Several of
the optional levels are implemented to varying degrees (see the level table),
but only Basic is *claimed as a conformance level*; where an optional level is
partial we say so and do **not** claim it.

### Evidence baseline

Against the W3C XSLT 3.0 test suite (run from the sibling
[`helium-w3c-tests`](https://github.com/lestrrat-go/helium-w3c-tests) module):

| Run | Pass | Skip | Fail | Total |
|-----|-----:|-----:|-----:|------:|
| Default (`go test ./xslt3/`) | 12,346 | 781 | 0 | 13,127 |
| Slow (`HELIUM_SLOW_TESTS=1`) | 12,827 | 300 | 0 | 13,127 |

There are **zero failures** in either run. The slow run is a strict superset of
the default run: it additionally executes the 481 performance-gated cases the
default run skips, all of which pass. Committed point-in-time evidence lives
beside this file:

- `summary-xslt30.md` / `results-xslt30.xml` — default run (the summary's
  "Slow run" section carries the slow counts).
- `results-xslt30-slow.xml` — slow run JUnit output.

The authoritative per-case reasons are the harness `w3cImplicitSkips` map
(`helium-w3c-tests` `xslt3/w3c_helpers_test.go`) and `expectations/xslt30.json`;
`summary-xslt30.md`'s "Skipped by reason" table is regenerated from them. Every
skip below reconciles to those sources.

This taxonomy is also enforced as a machine-readable, CI-gated contract. The
`helium-w3c-tests` module carries a generated per-case skip ledger
(`expectations/xslt30-skip-ledger.json`, one row per default-run skip: test id,
pinned upstream-suite commit, skip-class — the five labels below — reason, spec
dependency, paired passing 3.0 case) and a count contract
(`expectations/xslt30-skip-counts.json`). Both are regenerated from the real
skip sources with `go test ./xslt3 -run TestXSLT30SkipLedger -update-ledger` and
must never be hand-edited. `TestXSLT30SkipLedger` (run without the flag) is a
fast, fixture-free drift-check — wired into the `Conformance Ledger` CI workflow
— that fails if the failure count is nonzero, a skip is unrecorded or
reclassified, a mandatory-facility skip appears (any skip that classifies as
neither one of the four labels nor the enumerated narrow-quirk allowlist), or
the default/slow skip counts drift.

Known limitation: by project policy CI lives in helium, so the `Conformance
Ledger` workflow runs on helium PRs against `helium-w3c-tests@main`;
`helium-w3c-tests` has no CI of its own. A `helium-w3c-tests` PR that edits a
skip source without regenerating the ledger is therefore not gated on that PR —
the drift is caught on the next helium PR instead. Contributors editing skip
sources in `helium-w3c-tests` must regenerate the ledger with
`go test ./xslt3 -run TestXSLT30SkipLedger -update-ledger` (or `go generate ./xslt3`)
and commit it alongside the change.

## Conformance-level table

The eight W3C conformance levels (Basic + seven optional), plus two
capabilities the reviewer asked to see stated explicitly (packages, external
resource loading — these are *not* formal conformance levels). Status words are
used precisely: **implemented** (complete for Basic-3.0 purposes and suite-
exercised), **partial** (works for a defined subset, not the full optional
level), **not claimed** (helium does not assert this conformance level).

| Area | Status | Claimed as a level? | Notes |
|------|--------|---------------------|-------|
| **Basic XSLT Processor** | **implemented** | **Yes (the sole claim)** | All mandatory facilities exercised by the suite (see "What is implemented"). |
| Schema-Awareness | partial | No | `xsl:import-schema` and source-document schemas compile via the `xsd` package (default **XSD 1.1**); constructed-tree validation (`xsl:validation`) runs through `xsd.Validator`. This is *not* full Schema-Aware conformance — 31 suite cases require schema-awareness to be *absent* and 5 require XSD 1.1 absent, and full PSVI-driven type-checking is not claimed. |
| Serialization | implemented | No (optional level) | `xml` / `html` / `text` / `xhtml` output methods, character maps, multiple result documents. One byte-exact XHTML formatting quirk remains (`validation-0201`, see quirks). |
| Streaming | **partial — analysis only, not streaming execution** | No | Streamability **analysis** is implemented (`XTSE3430` reporting); streaming instructions execute by **DOM materialization**, i.e. the source is fully built in memory. Helium does **not** claim true bounded-memory streaming execution. |
| Higher-Order Functions | implemented | No (optional level) | Via `xpath3` (function items, `fn:for-each`, partial application, etc.). |
| XPath 3.1 | implemented | No (optional level) | Full XPath 3.1 data model, maps, arrays, and function library via `xpath3`. |
| Dynamic Evaluation | implemented | No (optional level) | `xsl:evaluate`. Dynamically-compiled expressions run under the same resolver/security model as static ones; external access is default-deny (below). |
| Backwards-Compatible Processing | implemented (**semantics, not grammar**) | No (optional level) | XSLT 1.0 behavior + XPath 1.0 compatibility mode, enabled per element when the effective `[xsl:]version` < 2.0. Compat mode changes **semantics**; XSLT/XPath **1.0/2.0 syntax** is deliberately out of scope. |
| Packages / `xsl:use-package` *(capability, not a level)* | implemented | n/a | Package declaration, import, component visibility. |
| External resource loading *(capability, not a level)* | implemented, **default-deny** | n/a | `document()`, `doc()`, `unparsed-text()`, collections, external/parameter entities are **resolver-mediated and default-deny** — network, arbitrary filesystem, and external-entity access are refused unless a caller opts in. This is a security posture, not a missing feature. |

### What is implemented (Basic XSLT 3.0)

All mandatory Basic XSLT 3.0 facilities work and are exercised by the suite:
template matching and modes; `xsl:apply-templates` / `xsl:call-template` /
`xsl:next-match` / `xsl:apply-imports`; variables and parameters; `xsl:function`;
`xsl:for-each-group`; sorting; `xsl:number`; keys; attribute-sets;
`xsl:result-document` (multiple result documents); accumulators; `xsl:merge`;
`xsl:iterate` / `xsl:fork`; `xsl:try` / `xsl:catch`; maps and arrays;
packages / `xsl:use-package`; and the XPath 3.1 data model. Unsupported
*optional* features are reported with concrete error codes
(e.g. `XTSE0090`, `XTSE0220`, `FOXT0003`) rather than silently misinterpreted.

## Skip taxonomy

A raw skip count is **not** a quality metric. Every one of the 781 default-run
skips falls into exactly one of four stable labels, plus a small residual
**narrow-quirk** bucket. The labels are defined once, here, so the classification
is auditable and does not drift:

- **Expected divergence** — the test asserts XSLT 1.0/2.0 (or XSD 1.0) behavior
  that a conforming XSLT 3.0 processor must **not** produce: a removed error
  code, an older regex/grouping rule, or older function-library availability.
  Our 3.0 output is the correct one; the 2.0 assertion cannot hold on a 3.0
  processor. **Each is paired in the harness with its passing 3.0 variant.**
- **Not claimed** — the case only applies to a processor **without** an optional
  feature we support, or exercises an optional conformance level helium does not
  claim (schema-awareness, disable-output-escaping, dynamic-evaluation-absent,
  XSD 1.1-absent, out-of-range year components, XQuery module loading, …). We
  *exceed* the case's requirement.
- **Deliberately denied** — a capability that exists but is disabled / resolver-
  gated **for security**: network access and external / parameter-entity
  resolution. Refusing these is intended behavior.
- **Performance-gated** — a valid case that **passes** under `HELIUM_SLOW_TESTS=1`
  (and in the on-demand CI workflow); skipped only in the default run to bound
  CI runtime. **Not a capability gap.**
- **Narrow quirk** *(residual)* — an individually-tracked, genuinely-narrow
  defect or fixture/environment dependence. None is a mandatory Basic 3.0
  facility. Enumerated in full in the next section.

### Reconciliation to the 781 default-run skips

| Label | Count | Composition (from `summary-xslt30.md` "Skipped by reason") |
|-------|------:|-------------------------------------------------------------|
| Performance-gated | 605 | slow streaming `big-transactions.xml` (433); large-corpus regex (120); slow source doc (35); slow test (13); large-iteration `xsl:evaluate` (3); large-iteration variable binding (1) |
| Expected divergence | 84 | XSLT 2.0-vs-3.0 removed errors / relaxed syntax / 3.0 library availability (80) + XSD 1.0-vs-1.1 spec-version divergences (4) |
| Not claimed | 76 | feature-present-but-required-absent: schema_aware (31), disabling_output_escaping (8), backwards_compatibility (6), XSD_1.1 (5), dynamic_evaluation (2), XPath_3.1 (1), streaming (1); out-of-range year components required absent (15); XQuery `load-xquery-module` (4); Saxon-format `?select=` glob URIs, W3C-noted non-interoperable (3) |
| Deliberately denied | 3 | network access to saxonica.com (1); external SYSTEM-entity resolution (1); external parameter-entity resolution (1) |
| Narrow quirk | 13 | itemized below |
| **Total** | **781** | |

In the slow run, the Performance-gated bucket drops from 605 to 124 (the 481
`HELIUM_SLOW_TESTS=1`-gated cases now run and pass), so the total falls to 300;
every other label is unchanged.

## Narrow-quirk list (isolated)

The complete residual set — 13 cases. None blocks Basic XSLT 3.0 conformance.
Test IDs are the harness case names (present in `expectations/xslt30.json` or
the generated case tables in `helium-w3c-tests` `xslt3/`).

| Test ID | Why it is NOT a Basic 3.0 blocker | Intended to fix? | Fix risk to correct 3.0 behavior? |
|---------|-----------------------------------|------------------|-----------------------------------|
| `validation-0201` | Transform output is content-correct; only Saxon-identical XHTML *formatting* differs (3-space indent, inline-element whitespace, XML declaration on the root line, `&#xa0;` preservation) — a serialization-cosmetics detail, not a semantic gap. | Eventually (serialization polish) | Low — a formatting-only change, isolated to the XHTML serializer. |
| `regex-syntax-xslt20-0984` | Unicode-version dependency, not a spec divergence: `[\w]` is correctly `[^\p{P}\p{Z}\p{C}]`; the test expects U+2308/U+2309 to match `\w` (pre-Unicode-6.1 category Sm), but Go's current Unicode tables classify them as Ps/Pe, so `\w` correctly excludes them. | No | High — matching the old expectation would mean diverging from Go's current Unicode data; our output is the correct current one. |
| `backwards-041` | Depends on the source nodes' `base-uri()` resolving against a *fixture* base URI rather than the test-set file path — a harness/environment dependence, not a transform defect. | No (fixture-dependent) | Low — no engine behavior involved. |
| `backwards-019` | The XSLT 1.0-only **default output method** (implicit-1.0 result tree serialized `xhtml`→`xml`) is not implemented; this is a backwards-compatibility *serialization default*, not a Basic 3.0 facility. | Maybe (BCM completeness) | Low — gated to effective-version < 2.0 output defaulting. |
| `import-schema-203` | Asserts a **placeholder** error code `XXXX9999` (not a real spec code); helium raises a concrete diagnostic instead. | No | N/A — the expected code is a catalog placeholder. |
| `normalize-unicode-008` | Requires an external fixture `NormalizationTest.txt` that is not shipped; a missing-fixture dependence, not an engine gap. | No (fixture-dependent) | None. |
| `error-FODC0002a-ignore` | Helium raises `FODC0002` on a `document()` retrieval failure instead of silently ignoring it — a stricter, still-conformant reporting choice. | No | Medium — relaxing it would weaken error reporting the paired `FODC0002` cases rely on. |
| `error-1160a` | Premise relied on `os.ReadFile` failing for `http://` URLs; the `loadDocument` retrieval refactor changed that assumption — a harness premise, not a transform defect. | No | Low. |
| `assert-007` | Requires `xsl:assert` evaluation to be **disabled**; helium evaluates assertions (the safer default). "Feature present, test wants it absent." | No | N/A — this is a Not-claimed-shaped residual, tracked here as narrow. |
| `format-number-070` | A user-defined function **shadows** the built-in `format-number` with a mismatched arity; a narrow name-resolution edge, not a core `format-number` gap. | Maybe | Medium — shadowing/arity resolution touches function-library lookup. |
| `select-3401` | Requires **XPath 1.0 grammar** (`div`/`mod` as a name after an operator). Compat mode delivers 1.0 *semantics*, not 1.0 *syntax* — grammar support is deliberately out of scope. | No (by design) | High — parsing 1.0 grammar would change the lexer/parser for all inputs. |
| `docbook-001` | Requires **XPath 1.0 grammar** (unprefixed `function` as a name test). Same design boundary as `select-3401`. | No (by design) | High — same parser-level risk. |
| `docbook-002` | Requires **XPath 1.0 grammar** (empty function arguments). Same design boundary as `select-3401`. | No (by design) | High — same parser-level risk. |

## Where we deliberately under-claim

To keep this declaration honest, three areas are stated **below** what the code
might superficially suggest:

- **Streaming** is listed *partial — analysis only*. Streamability analysis and
  `XTSE3430` reporting are implemented, but streaming instructions execute by
  **DOM materialization**. We do not claim bounded-memory streaming execution.
- **Schema-Awareness** is listed *partial*. `xsl:import-schema` and source/
  constructed-tree validation work against the `xsd` package (XSD 1.1 by
  default), but full Schema-Aware conformance (complete PSVI type propagation)
  is not claimed; 31 + 5 suite cases sit in "Not claimed" precisely because they
  require schema-awareness / XSD 1.1 to be *absent*.
- **External resource loading** is listed *default-deny*. The capability exists
  but refuses network, arbitrary-filesystem, and external-entity access unless a
  caller opts in — the three "Deliberately denied" skips are this posture
  working as intended, not gaps.

Only **Basic XSLT 3.0** is claimed as a conformance level. Everything else in
the level table is stated as implemented / partial / not-claimed strictly per
the current code.
