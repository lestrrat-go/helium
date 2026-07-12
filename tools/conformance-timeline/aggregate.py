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
import sys
import xml.etree.ElementTree as ET

import render_md

HERE = os.path.dirname(os.path.abspath(__file__))
RESULTS = os.path.join(HERE, "results")
CRASHERS = os.path.join(HERE, "crashers")
SUITES = ["xml", "xsd10", "xsd11", "xslt30", "qt3"]
SUITE_LABEL = {"xml": "XML 1.0/1.1", "xsd10": "XSD 1.0", "xsd11": "XSD 1.1",
               "xslt30": "XSLT 3.0", "qt3": "XPath/XQuery (QT3)"}
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


def main():
    # Discover tags present in results/
    tags = set()
    for fn in os.listdir(RESULTS):
        m = re.match(r"(v\d+\.\d+\.\d+)-(\w+)-junit\.xml$", fn)
        if m:
            tags.add(m.group(1))
    tags = sorted(tags, key=tag_sort_key)
    if not tags:
        sys.exit("no result junit files found in " + RESULTS)
    reference_tag = tags[-1]

    # Build the per-suite universe (denominator) from the reference tag's run: the cases a
    # conformant processor is actually expected to handle.
    #
    # Cases the harness SKIPS on the reference are excluded. They are not failures and never
    # could be: they are inapplicable by design -- XML 1.0 editions 1-4 when helium
    # implements the 5th, XML 1.1 / Namespaces 1.1 when helium targets 1.0, and the
    # equivalent dependency gates in the other suites. Counting them against a release is
    # just as dishonest as counting them as passes: it silently marks "not applicable" as
    # "failed" and understates every release (it drags v0.5.1's XML from 100% to 77%).
    #
    # A case that kills even the reference is skipped there too, so it is missing from its
    # JUnit -- add it back, or the suite would shrink for EVERY release and the case would
    # vanish from the scoring instead of counting against the releases it kills.
    universe, inapplicable = {}, {}
    for suite in SUITES:
        ref = parse_junit(os.path.join(RESULTS, f"{reference_tag}-{suite}-junit.xml"))
        ref = ref or {}
        applicable = {c for c, o in ref.items() if o != "skip"}
        ref_fail, ref_harness = load_crashers(reference_tag, suite)
        universe[suite] = applicable | set(ref_fail) | set(ref_harness)
        inapplicable[suite] = sum(1 for o in ref.values() if o == "skip")

    # Load tag dates from git (best-effort; falls back to empty).
    dates = {}
    for tag in tags:
        s = os.popen(f"git -C {HERE} log -1 --format=%cs {tag} 2>/dev/null").read().strip()
        dates[tag] = s

    rows = []
    for tag in tags:
        for suite in SUITES:
            cases = parse_junit(os.path.join(RESULTS, f"{tag}-{suite}-junit.xml"))
            denom = len(universe[suite]) or 0
            crash_fail, crash_harness = load_crashers(tag, suite)
            if cases is None:
                rows.append(dict(tag=tag, date=dates.get(tag, ""), suite=suite,
                                 measured=False, partial=False, passed=0, failed=0, skipped=0,
                                 unrun=0, xfail=0, crashed=0, harness_excluded=0,
                                 enumerated=0, denom=denom, not_enumerated=denom,
                                 pass_pct=0.0))
                continue
            uni = universe[suite]
            passed = sum(1 for c, o in cases.items() if o == "pass" and (not uni or c in uni))
            failed = sum(1 for c, o in cases.items() if o == "fail" and (not uni or c in uni))
            # A skip INSIDE the applicable set is a case the reference RUNS but this release
            # did not -- most of them the performance-gated XSLT cases, which helium only
            # started running in v0.4.0. It is not applicable-and-excused and it is not a
            # pass: the release has no passing result for a case in scope, so it is counted
            # as a FAILURE (tracked separately as `unrun` so the reason stays visible).
            unrun = sum(1 for c, o in cases.items() if o == "skip" and (not uni or c in uni))
            failed += unrun
            skipped = 0
            # Expected failures: helium deliberately diverges (it is namespace-aware and the
            # case expects a namespace-unaware result). Documented as "not a helium gap", but
            # it is still not a passing conformance result, so it counts against the score
            # and stays visible instead of being rounded into a perfect 100%.
            xfail = sum(1 for c, o in cases.items() if o == "xfail" and (not uni or c in uni))
            failed += xfail
            # Cases that killed the binary were skipped to let the suite finish, so they
            # are absent from the JUnit. They are failures of this release: count them.
            # (Guard against double-counting if one somehow did emit a verdict.)
            crashed = sum(1 for c in crash_fail
                          if (not uni or c in uni) and c not in cases)
            harness_excluded = sum(1 for c in crash_harness if (not uni or c in uni))
            failed += crashed
            enumerated = passed + failed + skipped
            not_enum = max(0, denom - enumerated)
            pct = round(100.0 * passed / denom, 2) if denom else 0.0
            # "partial" = the release did not run the whole suite: its compiler chokes so
            # the harness enumerates far fewer cases than today's suite. Those cases count
            # as not-passing, but the point is flagged so the chart can distinguish it from
            # a clean low score.
            partial = denom > 0 and enumerated < 0.95 * denom
            rows.append(dict(tag=tag, date=dates.get(tag, ""), suite=suite,
                             measured=True, partial=partial,
                             passed=passed, failed=failed, skipped=skipped, unrun=unrun, xfail=xfail,
                             crashed=crashed, harness_excluded=harness_excluded,
                             enumerated=enumerated, denom=denom, not_enumerated=not_enum,
                             pass_pct=pct))

    data = dict(
        generated=os.popen(f"git -C {HERE} log -1 --format=%cs 2>/dev/null").read().strip(),
        reference_tag=reference_tag,
        suites=[dict(key=s, label=SUITE_LABEL[s], denom=len(universe[s]),
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
