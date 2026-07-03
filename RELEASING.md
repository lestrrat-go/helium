# Releasing

## Conformance gate

A release cannot be tagged/published unless the slow XSLT 3.0 conformance suite
passes with **0 failures**. The full slow suite is **release-gating, not
PR-gating**: the heavyweight W3C conformance suites are never run on ordinary
pushes or pull requests (they clone large upstream fixture sets and, with the
performance-gated slow tests enabled, take many minutes), so they must not block
day-to-day PR CI.

### How it is enforced

`.github/workflows/release.yml` (triggered by pushing a `v*` tag) has a
`conformance-gate` job that runs the reusable `.github/workflows/conformance-run.yml`
with `suite: xslt30` and `slow: true` (which sets `HELIUM_SLOW_TESTS=1`). The
`goreleaser` publish job declares `needs: conformance-gate`, so the release
artifacts and GitHub release are **not** published when the slow suite reports
any failure.

The reusable run gates on the reported failure **count**: after running the
suite it reads `<testsuites failures="N">` from the JUnit report and fails the
job when `N` is not `0`, in addition to the harness's own non-zero exit on a
failing run. A skipped or absent report also fails the gate.

Because `release.yml` is triggered by a tag push, the tag itself may already
exist in the repository when the gate runs, but **no release is published**
until the gate is green. Re-run the release workflow (or push a corrected tag)
once the conformance suite passes.

### Running the suite manually

The same reusable workflow backs the nightly/manual `Conformance` workflow.
Trigger it from the Actions tab (`workflow_dispatch`), choosing the suite and
whether to enable slow tests; it also runs nightly against `xslt30` with slow
tests on.
