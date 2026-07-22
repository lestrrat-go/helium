# Releasing

Helium releases are **dispatch-driven**, not tag-triggered. A release is started
manually with a version input; the conformance gate runs *before* the tag exists,
so a failed conformance run never leaves a public tag or partial release behind.

## Cutting a release

1. Make sure `main` is at the commit you want to release and CI is green.
2. **Measure the release candidate into the conformance timeline and commit it.**
   The release workflow refuses to run unless the version already has a row in
   `tools/conformance-timeline/data.json` (see *Conformance-timeline row* below).
   Measure the exact commit you are about to release, then commit the regenerated
   docs to `main`:

   ```
   tools/conformance-timeline/run.sh --ref "$(git rev-parse HEAD)" --as v0.5.2
   git add tools/conformance-timeline/data.json \
           tools/conformance-timeline/conformance-timeline.svg \
           tools/conformance-timeline/conformance-timeline.html CONFORMANCE.md
   git commit -m "measure v0.5.2 into the conformance timeline"
   ```

   `--ref/--as` measures an *untagged* commit under the version label (the tag does
   not exist yet); the row is marked as an untagged release candidate. After the
   release tags the commit, a later `tools/conformance-timeline/run.sh v0.5.2`
   re-measures it at the real tag and supersedes the candidate row cleanly.
3. Run the Release workflow from `main`:
   - GitHub UI: **Actions → Release → Run workflow**, branch `main`, fill in
     `version` (e.g. `v0.5.2`). Leave `harness_ref` at its pinned default.
   - or: `gh workflow run release.yml --ref main -f version=v0.5.2`
4. The `timeline-presence` job checks in seconds that `version` has a row in the
   committed timeline (from step 2). If it does not, the run stops here — fix step
   2 and re-dispatch.
5. The `conformance-gate` job runs the slow XSLT 3.0 suite against the pinned
   `harness_ref`. If it fails, the run stops here — **no tag, no release.**
6. On green, the `release` job waits for **environment approval** (the `release`
   environment: maintainer reviewer, restricted to `main`). Approve it in the run.
7. After approval it creates and pushes the `version` tag, then runs goreleaser
   to publish the GitHub release + binaries.

`version` must be a `vX.Y.Z` (optionally `-prerelease`) tag; the job fails fast
on a malformed version or one that already exists on a *different* commit. A
hand-pushed tag no longer triggers the **Release** workflow — dispatch is the
only path that releases (a raw `push` still runs ordinary CI).

## Conformance-timeline row (presence gate)

Every release must appear in the committed cross-release conformance timeline
(`CONFORMANCE.md` + `tools/conformance-timeline/data.json`), which charts how much
of *today's* W3C suites each tagged release passes. The `timeline-presence` job in
`release.yml` fails the release unless `version` already has a row in `data.json`.

This gate exists because **v0.6.0 was released without a timeline row** — the tag
was cut after the timeline's last regeneration, so the release shipped absent from
its own conformance history and the gap was only noticed later. Requiring the row
up front (measured against the exact release commit, step 2 above) keeps the
timeline complete by construction: a release cannot be published until it has been
measured. The check is a fast metadata read, so it runs before the multi-minute
slow suite and fails in seconds when the row is missing.

The row is seeded from an *untagged* measurement (`run.sh --ref <sha> --as vX.Y.Z`)
because the tag does not exist at dispatch time; it is marked as a release
candidate and superseded by a real-tag re-measure afterwards. See
`tools/conformance-timeline/README.md` for the measurement method.

## Conformance gate

A release cannot be tagged/published unless the slow XSLT 3.0 conformance suite
passes with **0 failures**. The full slow suite is **release-gating, not
PR-gating**: the heavyweight W3C conformance suites are never run on ordinary
pushes or pull requests (they clone large upstream fixture sets and, with the
performance-gated slow tests enabled, take many minutes), so they must not block
day-to-day PR CI.

### How it is enforced

`.github/workflows/release.yml`'s `conformance-gate` job runs the reusable
`.github/workflows/conformance-run.yml` with `suite: xslt30` and `slow: true`
(which sets `HELIUM_SLOW_TESTS=1`). The `release` job declares
`needs: conformance-gate` and runs *after* the gate, so the tag is created and
goreleaser runs **only** on a green suite — a red gate stops the workflow before
any tag exists.

The reusable run gates on the reported failure **count**: after running the
suite it reads `<testsuites failures="N">` from the JUnit report and fails the
job when `N` is not `0`, in addition to the harness's own non-zero exit on a
failing run. A skipped or absent report also fails the gate.

### Running the suite manually

The same reusable workflow backs the nightly/manual `Conformance` workflow.
Trigger it from the Actions tab (`workflow_dispatch`), choosing the suite and
whether to enable slow tests; it also runs nightly against `xslt30` with slow
tests on.

## The pinned harness_ref

`release.yml` pins `harness_ref` to a known-good `helium-w3c-tests` commit so the
release gate is **reproducible** and cannot be red-blocked by unrelated churn on
`helium-w3c-tests@main`. Each release records exactly which harness certified it.

**Bumping the pin:** the nightly `Conformance` run tracks `helium-w3c-tests@main`
(unpinned) — a green nightly means that harness commit passes against helium
`main`. To certify releases against newer upstream tests, set the `harness_ref`
default in `release.yml` to the harness SHA from the latest green nightly:

```
gh api repos/lestrrat-go/helium-w3c-tests/commits/main --jq .sha
```

Bump the pin in its own PR. It only advances which tests gate a release; nothing
else depends on it.

## Environment approval

The `release` GitHub Environment gates the tag/publish job: it requires a
maintainer reviewer and restricts deployment to the `main` branch. Anyone with
write access can *start* a dispatch, but tag + publish only proceeds after
approval. Manage reviewers under **Settings → Environments → release**.

## Recovery if goreleaser fails after tagging

The gate guarantees no side effects from a *conformance* failure. There is one
narrow window it does not cover: if the tag is pushed but goreleaser then fails
(e.g. a transient GitHub API error), a tag exists with no published release.

To recover, either re-run the failed `release` job — the tag step is idempotent,
so when the tag already exists at that commit it is reused and goreleaser re-runs
against it — or delete the tag and re-dispatch:

```
git push origin :refs/tags/v0.5.2   # delete remote tag
# then re-run the Release workflow with the same version
```
