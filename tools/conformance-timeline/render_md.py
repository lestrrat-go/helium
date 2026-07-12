#!/usr/bin/env python3
"""Render CONFORMANCE.md (repo root) + its SVG chart from data.json.

Both files are GENERATED -- do not hand-edit them; change this renderer and re-run
`python3 tools/conformance-timeline/aggregate.py`, which calls into here.

The chart is a plain SVG committed to the repo and referenced from the markdown, so it
renders on GitHub (which strips <script> but keeps shapes, text, and a <style> block --
hence the prefers-color-scheme rules for dark mode).
"""
import json
import os

# Suite line colours: distinguishable in both themes. The lines converge at the top, so a
# legend identifies them -- direct end-labels would collide into an unreadable pile.
COLORS = {
    "xml":    "#b5359c",
    "xsd10":  "#008300",
    "xsd11":  "#d98800",
    "xslt30": "#2a78d6",
    "qt3":    "#1baf7a",
}

SHORT = {"xml": "XML", "xsd10": "XSD 1.0", "xsd11": "XSD 1.1",
         "xslt30": "XSLT 3.0", "qt3": "QT3 (XPath/XQuery)"}

W, H = 900, 720
LEG_Y = 34                      # legend band
PANELS = [                      # (title, y-min, tick-step, top, height)
    ("Full range", 0, 10, 64, 300),
    ("Detail — the 94–100% band, where every suite ends up", 94, 1, 430, 210),
]
PL, PR = 62, 48                 # plot left/right margins (right leaves room for the last date)


def _x(i, n):
    return PL + (i * (W - PL - PR) / max(1, n - 1))


def _y(pct, ymin, top, h):
    return top + (100.0 - pct) * h / (100.0 - ymin)


def render_svg(data, path):
    """Two stacked panels: the full 0-100% range, and a zoom on the 94-100% band.

    A single 0-100% chart is useless here -- XSLT, QT3 and both XSD lines all sit within a
    fraction of a percent of each other at the top, so they overlap into one line and any
    direct end-label collides with the others. The zoom panel separates them; the legend
    (rather than end-labels) keeps the series identifiable in both.
    """
    tags, suites, rows = data["tags"], data["suites"], data["rows"]
    by = {(r["tag"], r["suite"]): r for r in rows}
    n = len(tags)
    out = [f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {W} {H}" width="{W}" height="{H}" '
           f'role="img" aria-label="helium W3C conformance by release">']
    out.append("""<style>
    .bg{fill:#ffffff}.grid{stroke:#e1e0d9}.axis{stroke:#c3c2b7}
    .tick{fill:#6b6a66;font:11px system-ui,sans-serif}
    .leg{fill:#3d3c39;font:12px system-ui,sans-serif}
    .ttl{fill:#0b0b0b;font:13px system-ui,sans-serif;font-weight:600}
    .pt{fill:#3d3c39;font:11.5px system-ui,sans-serif;font-weight:600}
    .sub{fill:#6b6a66;font:10.5px system-ui,sans-serif}
    @media (prefers-color-scheme: dark){
      .bg{fill:#0d0d0d}.grid{stroke:#2c2c2a}.axis{stroke:#3a3a37}
      .tick,.sub{fill:#8f8e88}.ttl{fill:#f2f2f0}.leg,.pt{fill:#d7d6d0}
    }
  </style>""")
    out.append(f'<rect class="bg" x="0" y="0" width="{W}" height="{H}"/>')
    out.append(f'<text class="ttl" x="{PL}" y="20">Share of each suite\'s applicable cases passed, by release</text>')

    # legend
    lx = PL
    for s in suites:
        c = COLORS[s["key"]]
        out.append(f'<line x1="{lx}" y1="{LEG_Y}" x2="{lx + 22}" y2="{LEG_Y}" stroke="{c}" stroke-width="3" '
                   f'stroke-linecap="round"/>')
        out.append(f'<circle cx="{lx + 11}" cy="{LEG_Y}" r="3.4" fill="{c}"/>')
        label = SHORT[s["key"]]
        out.append(f'<text class="leg" x="{lx + 29}" y="{LEG_Y + 4}">{label}</text>')
        lx += 34 + 7.2 * len(label) + 22

    for title, ymin, step, top, h in PANELS:
        out.append(f'<text class="pt" x="{PL}" y="{top - 12}">{title}</text>')
        for v in range(ymin, 101, step):
            y = _y(v, ymin, top, h)
            out.append(f'<line class="grid" x1="{PL}" y1="{y:.1f}" x2="{W - PR}" y2="{y:.1f}"/>')
            out.append(f'<text class="tick" x="{PL - 8}" y="{y + 4:.1f}" text-anchor="end">{v}%</text>')
        for i, t in enumerate(tags):
            x = _x(i, n)
            out.append(f'<line class="grid" x1="{x:.1f}" y1="{top}" x2="{x:.1f}" y2="{top + h}"/>')
            out.append(f'<text class="tick" x="{x:.1f}" y="{top + h + 17}" text-anchor="middle">{t}</text>')
            out.append(f'<text class="sub" x="{x:.1f}" y="{top + h + 30}" text-anchor="middle">'
                       f'{data["dates"].get(t, "")}</text>')
        out.append(f'<line class="axis" x1="{PL}" y1="{top + h}" x2="{W - PR}" y2="{top + h}"/>')

        for s in suites:
            k = s["key"]
            c = COLORS[k]
            segs, cur = [], []
            for i, t in enumerate(tags):
                r = by.get((t, k))
                # in the zoom panel a series below the floor simply leaves the frame; break
                # the line rather than clamp it to the axis and imply a value it never had
                if not r or not r["measured"] or r["pass_pct"] < ymin:
                    if cur:
                        segs.append(cur)
                        cur = []
                    continue
                cur.append((_x(i, n), _y(r["pass_pct"], ymin, top, h), r))
            if cur:
                segs.append(cur)
            for seg in segs:
                d = " ".join(f"{'M' if j == 0 else 'L'}{p[0]:.1f},{p[1]:.1f}" for j, p in enumerate(seg))
                out.append(f'<path d="{d}" fill="none" stroke="{c}" stroke-width="2.2" '
                           f'stroke-linejoin="round" stroke-linecap="round"/>')
                for x, y, r in seg:
                    fill = "none" if r["partial"] else c   # hollow = suite not fully enumerated
                    out.append(f'<circle cx="{x:.1f}" cy="{y:.1f}" r="3.4" fill="{fill}" stroke="{c}" '
                               f'stroke-width="2"/>')

    out.append(f'<text class="sub" x="{PL}" y="{H - 8}">'
               'Hollow dot = the release could not enumerate the whole suite (its compiler chokes first). '
               'A line missing from the lower panel is below 94% at that release.</text>')
    out.append("</svg>")
    with open(path, "w") as f:
        f.write("\n".join(out) + "\n")
    print("wrote", path)


def _fmt(r):
    if r["passed"] == r["denom"]:
        return "**100%**"
    return f"{r['pass_pct']:.2f}%" if r["pass_pct"] >= 99.9 else f"{r['pass_pct']:.1f}%"


def render_md(data, path, svg_rel):
    tags, suites, rows = data["tags"], data["suites"], data["rows"]
    by = {(r["tag"], r["suite"]): r for r in rows}
    ref = data["reference_tag"]
    L = []
    L.append("# W3C conformance across releases\n")
    L.append("<!-- GENERATED by tools/conformance-timeline/aggregate.py -- do not edit by hand. -->\n")
    L.append(f"Every tagged release measured **unmodified** against *today's* W3C suites "
             f"(reference: `{ref}`). See "
             f"[tools/conformance-timeline](tools/conformance-timeline/README.md) for the method.\n")
    L.append(f"![conformance by release]({svg_rel})\n")

    L.append("## Score by release\n")
    L.append("| Release | Date | " + " | ".join(s["label"] for s in suites) + " |")
    L.append("|---|---|" + "---:|" * len(suites))
    for t in tags:
        cells = []
        for s in suites:
            r = by[(t, s["key"])]
            marks = ""
            if r["crashed"]:
                marks += f" ✗{r['crashed']}"
            if r["unrun"]:
                marks += f" ⊘{r['unrun']}"
            if r["xfail"]:
                marks += f" ⚠{r['xfail']}"
            cells.append(_fmt(r) + marks)
        L.append(f"| `{t}` | {data['dates'].get(t,'')} | " + " | ".join(cells) + " |")
    L.append("")
    L.append("✗ hang/OOM crasher · ⊘ in-scope case never run · ⚠ documented expected failure "
             "— **all three count as not-passing**.\n")

    L.append("## How a score is computed\n")
    L.append("The denominator is the set of cases the reference release actually **runs**: today's suite "
             "minus the cases the harness skips as *inapplicable* (wrong XML edition, XML 1.1 when helium "
             "targets 1.0, dependency gates). Those are not failures and never could be, so charging them "
             "to a release would understate it.\n")
    L.append("| Suite | Applicable | Excluded as inapplicable |")
    L.append("|---|---:|---:|")
    for s in suites:
        L.append(f"| {s['label']} | {s['denom']:,} | {s['inapplicable']:,} |")
    L.append("")
    L.append("Everything else counts against the release: cases it fails, cases it **cannot enumerate** "
             "(its compiler chokes before the harness can run them — a hollow dot on the chart), cases it "
             "**skips that the reference runs**, cases that **hang or exhaust its memory**, and documented "
             "**expected failures**.\n")

    L.append("## Full counts\n")
    for s in suites:
        k = s["key"]
        L.append(f"### {s['label']} — {s['denom']:,} applicable cases\n")
        L.append("| Release | Pass | Not passing | ⚠ xfail | ⊘ unrun | ✗ crash | Not enumerated | Score |")
        L.append("|---|---:|---:|---:|---:|---:|---:|---:|")
        for t in tags:
            r = by[(t, k)]
            if not r["measured"]:
                L.append(f"| `{t}` | — | — | — | — | — | — | not measured |")
                continue
            notpass = r["denom"] - r["passed"]
            L.append(f"| `{t}` | {r['passed']:,} | {notpass:,} | {r['xfail'] or '—'} | "
                     f"{r['unrun'] or '—'} | {r['crashed'] or '—'} | "
                     f"{r['not_enumerated'] or '—'} | {_fmt(r)} |")
        L.append("")
    with open(path, "w") as f:
        f.write("\n".join(L))
    print("wrote", path)


def main():
    here = os.path.dirname(os.path.abspath(__file__))
    repo = os.path.abspath(os.path.join(here, "..", ".."))
    data = json.load(open(os.path.join(here, "data.json")))
    svg = os.path.join(here, "conformance-timeline.svg")
    render_svg(data, svg)
    render_md(data, os.path.join(repo, "CONFORMANCE.md"),
              "tools/conformance-timeline/conformance-timeline.svg")


if __name__ == "__main__":
    main()
