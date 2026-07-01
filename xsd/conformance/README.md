# XSD 1.1 W3C conformance evidence

Committed, point-in-time evidence that helium passes the XSD 1.1 W3C conformance
suite. The test infrastructure (generator, harness, fixtures) lives in the
sibling `helium-w3c-tests` module; only the **results** are recorded here.

- `summary.md` — human-readable pass/skip/fail counts + skip-reason breakdown,
  stamped with the pinned upstream suite commit and the helium commit tested.
- `results.xml` — JUnit XML, one testcase per conformance case. Generated with
  `-no-system-out`, so each testcase carries its pass/skip/fail record and (for
  skips/failures) the diagnostic message, but not the per-case captured-stdout
  mirror — keeping the report machine-readable without the bulk.

Refresh from the sibling module (which `replace`s `helium => ../helium`):

```sh
# in ../helium-w3c-tests, after: go run ./cmd/w3cgen fetch xsd11
go run ./cmd/w3ctest -no-system-out \
  -out ../helium/xsd/conformance/results.xml \
  -summary ../helium/xsd/conformance/summary.md \
  -helium-commit "$(git -C ../helium rev-parse --short HEAD)" \
  xsd11
```
