#!/usr/bin/env python3
"""Aggregate per-tag W3C conformance JUnit results into data.json + a rendered
self-contained HTML timeline.

Metric (see README): the denominator for each suite is the case set enumerated by
the newest release run with the UNMODIFIED harness (the "reference" tag, v0.4.0).
A release's score for a suite is (passing cases) / (reference-suite size). Cases an
older release cannot even enumerate (its parser/compiler chokes on the schema, so
the harness never runs them) therefore count as not-passing — which is honest:
the release cannot produce a passing result for them.

Reads   tools/conformance-timeline/results/<tag>-<suite>-junit.xml
Writes  tools/conformance-timeline/data.json
        tools/conformance-timeline/conformance-timeline.html  (template + inlined data)
"""
import json
import os
import re
import subprocess
import sys
import xml.etree.ElementTree as ET

import render_md

HERE = os.path.dirname(os.path.abspath(__file__))


def git_out(*args):
    """Return the trimmed stdout of `git -C HERE <args>`, or '' on any failure.

    Used to resolve tag / commit dates; a ref that does not resolve (an --as
    label is not a git ref) simply yields ''."""
    try:
        r = subprocess.run(["git", "-C", HERE, *args], capture_output=True, text=True)
    except OSError:
        return ""
    return r.stdout.strip() if r.returncode == 0 else ""
RESULTS = os.path.join(HERE, "results")
CRASHERS = os.path.join(HERE, "crashers")
SUITES = ["xml", "xsd10", "xsd11", "xslt30", "qt3",
          "xmldsig2ed", "xmldsig11", "merlinxmldsig"]
SUITE_LABEL = {"xml": "XML 1.0/1.1", "xsd10": "XSD 1.0", "xsd11": "XSD 1.1",
               "xslt30": "XSLT 3.0", "qt3": "XPath/XQuery (QT3)",
               "xmldsig2ed": "XMLDSig 2ed interop", "xmldsig11": "XMLDSig 1.1 interop",
               "merlinxmldsig": "XMLDSig merlin baseline"}
REFERENCE_TAG = "v4"  # placeholder; resolved to the newest tag below


def tag_sort_key(tag):
    return [int(x) for x in re.findall(r"\d+", tag)]


def load_crashers(tag, suite):
    """Cases the release cannot survive: it hangs on them or exhausts memory.

    They are skipped when the suite is run (otherwise they take the whole binary down
    and the suite yields nothing at all), so they never appear in the JUnit -- but they
    are the release's failures and are counted as such here. See isolate.sh.

    Lines: <case-id> TAB <kind> TAB <note>, kind = "fail" (the release's fault) or
    "harness" (also dies on the reference tag, so it is our bug, not the release's --
    charging it to the release would invent a failure).
    """
    path = os.path.join(CRASHERS, f"{tag}-{suite}.txt")
    failed, harness = [], []
    if not os.path.exists(path):
        return failed, harness
    for line in open(path):
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        parts = line.split("\t")
        case, kind = parts[0], (parts[1] if len(parts) > 1 else "fail")
        (harness if kind == "harness" else failed).append(case)
    return failed, harness


def _is_xfail(tc):
    """True if the harness marked this case an expected failure (system-out `xfail (...)`)."""
    out = tc.find("system-out")
    return out is not None and "xfail (" in (out.text or "")


def parse_junit(path):
    """Return dict caseName -> 'pass'|'fail'|'skip', or None if file is a setup-fail stub/missing."""
    if not os.path.exists(path) or os.path.getsize(path) == 0:
        return None
    try:
        root = ET.parse(path).getroot()
    except ET.ParseError:
        return None
    cases = {}
    for tc in root.iter("testcase"):
        name = tc.get("name") or ""
        # strip the leading TestXxxW3C/ harness prefix for a stable case id
        cid = re.sub(r"^Test[A-Za-z0-9]+W3C/", "", name)
        if tc.find("failure") is not None or tc.find("error") is not None:
            outcome = "fail"
        elif tc.find("skipped") is not None:
            outcome = "skip"
        elif _is_xfail(tc):
            # An expected failure is a PASSING Go test (the harness asserts the divergence),
            # so it lands in the JUnit pass bucket with only a marker in system-out. It is
            # not a passing conformance result: helium does not produce what the case wants.
            # Counting it as a pass would claim 8 XML cases we do not actually pass.
            outcome = "xfail"
        else:
            outcome = "pass"
        cases[cid] = outcome
    # A single synthetic "setup" case means the suite never really ran.
    if len(cases) <= 1 and any("setup" in k for k in cases):
        return None
    return cases


def load_committed():
    """Load the committed data.json, or None if absent/corrupt.

    results/ is a LOCAL cache that is NOT committed, so a fresh checkout has no
    per-tag JUnit at all -- the committed data.json is the ONLY record of past
    measurements. This aggregator therefore MERGES: freshly measured (tag, suite)
    cells override, and every other cell carries over verbatim from here. Rebuilding
    wholesale from results/ would silently wipe every tag/suite that was not just
    re-measured (hours of measurement lost)."""
    path = os.path.join(HERE, "data.json")
    if not os.path.exists(path):
        return None
    try:
        return json.load(open(path))
    except (ValueError, OSError):
        return None


def load_label_meta(tag):
    """Return (date, untagged) for an --as label recorded by run.sh, or (None, False).

    An --as measurement targets an UNTAGGED committish (a release candidate before its
    tag exists), so git cannot resolve its date. run.sh records `<date>\\tuntagged` in
    results/<label>.meta; a later re-measure at the real tag supersedes it (git then
    resolves the date and the row is no longer untagged)."""
    path = os.path.join(RESULTS, f"{tag}.meta")
    if not os.path.exists(path):
        return None, False
    parts = open(path).read().strip().split("\t")
    if not parts or not parts[0]:
        return None, False
    return parts[0], (len(parts) > 1 and parts[1] == "untagged")


def build_row(tag, suite, uni, denom, date, untagged):
    """Compute one (tag, suite) row from its fresh JUnit + crasher records.

    uni is the reference universe (case-id set) or None to count every enumerated case
    (used only when the reference has no fresh run for this suite)."""
    cases = parse_junit(os.path.join(RESULTS, f"{tag}-{suite}-junit.xml"))
    crash_fail, crash_harness = load_crashers(tag, suite)
    if cases is None:
        return dict(tag=tag, date=date, suite=suite, measured=False, partial=False,
                    passed=0, failed=0, skipped=0, unrun=0, xfail=0, crashed=0,
                    harness_excluded=0, enumerated=0, denom=denom, not_enumerated=denom,
                    pass_pct=0.0, untagged=untagged)

    def inuni(c):
        return uni is None or c in uni

    passed = sum(1 for c, o in cases.items() if o == "pass" and inuni(c))
    failed = sum(1 for c, o in cases.items() if o == "fail" and inuni(c))
    # A skip INSIDE the applicable set is a case the reference RUNS but this release did
    # not -- most of them the performance-gated XSLT cases, which helium only started
    # running in v0.4.0. It is not applicable-and-excused and it is not a pass: the release
    # has no passing result for an in-scope case, so it is counted as a FAILURE (tracked
    # separately as `unrun` so the reason stays visible).
    unrun = sum(1 for c, o in cases.items() if o == "skip" and inuni(c))
    failed += unrun
    skipped = 0
    # Expected failures: helium deliberately diverges (documented "not a helium gap"), but
    # it is still not a passing conformance result, so it counts against the score and
    # stays visible instead of being rounded into a perfect 100%.
    xfail = sum(1 for c, o in cases.items() if o == "xfail" and inuni(c))
    failed += xfail
    # Cases that killed the binary were skipped to let the suite finish, so they are absent
    # from the JUnit. They are failures of this release: count them. (Guard against
    # double-counting if one somehow did emit a verdict.)
    crashed = sum(1 for c in crash_fail if inuni(c) and c not in cases)
    harness_excluded = sum(1 for c in crash_harness if inuni(c))
    failed += crashed
    enumerated = passed + failed + skipped
    not_enum = max(0, denom - enumerated)
    pct = round(100.0 * passed / denom, 2) if denom else 0.0
    # "partial" = the release did not run the whole suite: its compiler chokes so the
    # harness enumerates far fewer cases than today's suite. Those cases count as
    # not-passing, but the point is flagged so the chart distinguishes it from a clean
    # low score.
    partial = denom > 0 and enumerated < 0.95 * denom
    return dict(tag=tag, date=date, suite=suite, measured=True, partial=partial,
                passed=passed, failed=failed, skipped=skipped, unrun=unrun, xfail=xfail,
                crashed=crashed, harness_excluded=harness_excluded,
                enumerated=enumerated, denom=denom, not_enumerated=not_enum,
                pass_pct=pct, untagged=untagged)


def main():
    # MERGE, don't rebuild: the committed data.json is the only record of tag/suite cells
    # not present in the local (uncommitted) results/ cache. Freshly measured cells
    # override; everything else carries over.
    committed = load_committed()
    committed_rows, committed_suites, committed_dates, committed_tags = {}, {}, {}, []
    if committed:
        for r in committed.get("rows", []):
            committed_rows[(r["tag"], r["suite"])] = r
        committed_suites = {s["key"]: s for s in committed.get("suites", [])}
        committed_dates = dict(committed.get("dates", {}))
        committed_tags = list(committed.get("tags", []))

    # Discover freshly measured (tag, suite) cells and any --as label meta in results/.
    # results/ is an uncommitted local cache: on a fresh checkout it is absent entirely,
    # in which case there is nothing fresh and everything comes from the committed data.
    fresh, fresh_tags, meta_tags = set(), set(), set()
    for fn in (os.listdir(RESULTS) if os.path.isdir(RESULTS) else []):
        m = re.match(r"(v\d+\.\d+\.\d+)-(\w+)-junit\.xml$", fn)
        if m:
            fresh.add((m.group(1), m.group(2)))
            fresh_tags.add(m.group(1))
            continue
        m = re.match(r"(v\d+\.\d+\.\d+)\.meta$", fn)
        if m:
            meta_tags.add(m.group(1))

    tags = sorted(set(committed_tags) | fresh_tags | meta_tags, key=tag_sort_key)
    if not tags:
        sys.exit("no results in " + RESULTS + " and no committed data.json to merge")
    reference_tag = tags[-1]

    # Dates + untagged flags. A real git tag always wins (it supersedes an earlier --as
    # label measurement of the same version): resolve its date from git and clear untagged.
    dates, untagged_map = {}, {}
    for tag in tags:
        gitdate = git_out("log", "-1", "--format=%cs", tag)
        if gitdate:
            dates[tag] = gitdate
            untagged_map[tag] = False
            continue
        mdate, ut = load_label_meta(tag)
        untagged_map[tag] = ut
        dates[tag] = mdate or committed_dates.get(tag, "")

    # Build the per-suite universe (denominator) from the reference tag's FRESH run: the
    # cases a conformant processor is expected to handle. Cases the harness SKIPS on the
    # reference are excluded -- they are inapplicable by design (wrong XML edition, XML 1.1
    # when helium targets 1.0, dependency gates), not failures. When the reference has no
    # fresh run for a suite, its committed denominator carries over unchanged.
    #
    # A case that kills even the reference is skipped there too, so it is missing from its
    # JUnit -- add it back, or the suite would shrink for EVERY release and the case would
    # vanish from scoring instead of counting against the releases it kills.
    universe, inapplicable, suite_denom = {}, {}, {}
    for suite in SUITES:
        if (reference_tag, suite) in fresh:
            ref = parse_junit(os.path.join(RESULTS, f"{reference_tag}-{suite}-junit.xml")) or {}
            applicable = {c for c, o in ref.items() if o != "skip"}
            ref_fail, ref_harness = load_crashers(reference_tag, suite)
            universe[suite] = applicable | set(ref_fail) | set(ref_harness)
            suite_denom[suite] = len(universe[suite])
            inapplicable[suite] = sum(1 for o in ref.values() if o == "skip")
        else:
            # No fresh reference run for this suite: reuse the committed metrics verbatim.
            universe[suite] = None
            cs = committed_suites.get(suite, {})
            suite_denom[suite] = cs.get("denom", 0)
            inapplicable[suite] = cs.get("inapplicable", 0)

    rows = []
    for tag in tags:
        for suite in SUITES:
            date, untagged = dates.get(tag, ""), untagged_map.get(tag, False)
            if (tag, suite) in fresh:
                rows.append(build_row(tag, suite, universe[suite], suite_denom[suite],
                                      date, untagged))
                continue
            prev = committed_rows.get((tag, suite))
            if prev is not None:
                # Carry over verbatim; only the volatile date/untagged flags refresh.
                row = dict(prev)
                row["date"] = date
                row["untagged"] = untagged
                rows.append(row)
                continue
            rows.append(dict(tag=tag, date=date, suite=suite, measured=False, partial=False,
                             passed=0, failed=0, skipped=0, unrun=0, xfail=0, crashed=0,
                             harness_excluded=0, enumerated=0, denom=suite_denom[suite],
                             not_enumerated=suite_denom[suite], pass_pct=0.0, untagged=untagged))

    data = dict(
        generated=git_out("log", "-1", "--format=%cs"),
        reference_tag=reference_tag,
        suites=[dict(key=s, label=SUITE_LABEL[s], denom=suite_denom[s],
                     inapplicable=inapplicable[s]) for s in SUITES],
        tags=tags,
        dates=dates,
        rows=rows,
    )
    out = os.path.join(HERE, "data.json")
    with open(out, "w") as f:
        json.dump(data, f, indent=2)
    print("wrote", out)

    # Render the committed CONFORMANCE.md + its SVG chart from the same data, so the doc
    # can never drift from the measurements.
    render_md.render_svg(data, os.path.join(HERE, "conformance-timeline.svg"))
    render_md.render_md(
        data,
        os.path.join(os.path.abspath(os.path.join(HERE, "..", "..")), "CONFORMANCE.md"),
        "tools/conformance-timeline/conformance-timeline.svg",
    )

    # Render the self-contained HTML by inlining data.json into the template.
    tpl = os.path.join(HERE, "template.html")
    if os.path.exists(tpl):
        with open(tpl) as f:
            html = f.read()
        html = html.replace("/*__DATA__*/null", json.dumps(data))
        outhtml = os.path.join(HERE, "conformance-timeline.html")
        with open(outhtml, "w") as f:
            f.write(html)
        print("wrote", outhtml)


if __name__ == "__main__":
    main()
