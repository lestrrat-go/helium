package xslt3_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// dispatchIndexConflictStylesheet builds a mode-"" template list long enough
// to trigger the dispatch index (threshold 16 in dispatch_index.go), with two
// templates of equal priority that both genuinely match the same <a/> element
// but land in DIFFERENT index buckets: "a" is name-indexed (byName), and
// ".[self::a]" is a FilterExpr around ContextItemExpr — unindexed, per
// dispatch_index.go's altSignature — so it is always probed regardless of the
// dispatched node's name. Every filler template has strictly lower priority,
// so the two real candidates are scanned (in either declaration order) before
// classifyConflictCandidate's precedence/priority early-break would ever stop
// the scan on a filler.
func dispatchIndexConflictStylesheet(nFillers int) string {
	var b strings.Builder
	b.WriteString(`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">`)
	b.WriteString(`<xsl:template match="/"><out><xsl:apply-templates select="root/a"/></out></xsl:template>`)
	b.WriteString(`<xsl:template match="a" priority="1"><x/></xsl:template>`)
	b.WriteString(`<xsl:template match=".[self::a]" priority="1"><y/></xsl:template>`)
	for i := range nFillers {
		fmt.Fprintf(&b, `<xsl:template match="filler%d" priority="-1"><f/></xsl:template>`, i)
	}
	b.WriteString(`</xsl:stylesheet>`)
	return b.String()
}

// TestDispatchIndexConflictAcrossBuckets is the design step 7 requirement: a
// conflicting pair split across a name-indexed bucket and the unindexed
// bucket must still raise XTDE0540 under on-multiple-match="fail", proving
// hasConflictingMatch's index-aware path (hasConflictingMatchIn in
// execute_templates.go) preserves the precedence/priority early-break and
// still finds a conflict the plain linear scan would have found.
func TestDispatchIndexConflictAcrossBuckets(t *testing.T) {
	const nFillers = 20 // total mode-list length: 3 + 20 = 23, above the threshold of 16

	src := dispatchIndexConflictStylesheet(nFillers)
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a/></root>`))
	require.NoError(t, err)

	_, err = ss.Transform(source).
		OnMultipleMatch(xslt3.OnMultipleMatchFail).
		Serialize(t.Context())
	require.Error(t, err, "a name-indexed and an unindexed template of equal priority both matching <a/> must conflict")
	require.Contains(t, err.Error(), "XTDE0540")
}

// TestDispatchIndexKeyPatternAlwaysProbed is the design step 8 / risk 6
// requirement: a rule whose pattern is a key() lookup is always unindexed
// (dispatch_index.go altSignature falls through to its default case for any
// expression matchPatternAlt does not match without a matchByEvaluation
// fallback), so it must still be probed — and its key() call must still run,
// populating the key table as a side effect — for every dispatched node, even
// once the mode list is long enough for the dispatch index to exist.
func TestDispatchIndexKeyPatternAlwaysProbed(t *testing.T) {
	const nFillers = 20

	var b strings.Builder
	b.WriteString(`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">`)
	b.WriteString(`<xsl:key name="byId" match="item" use="@id"/>`)
	b.WriteString(`<xsl:template match="/"><out><xsl:apply-templates select="root/item"/></out></xsl:template>`)
	// A bare key() call as the WHOLE pattern is a plain FunctionCall AST node
	// — not a LocationPath/PathStepExpr/RootExpr/UnionExpr — so altSignature's
	// default case marks it unindexed (risk 6 in the design doc). It must
	// still be probed, and its key() side effect must still populate the key
	// table, for every dispatched node regardless of the node's own name or
	// the index's byName/byKind buckets built from the OTHER (name-indexed)
	// filler templates.
	b.WriteString(`<xsl:template match="key('byId', '2')" priority="1"><hit/></xsl:template>`)
	b.WriteString(`<xsl:template match="item" priority="0"><miss/></xsl:template>`)
	for i := range nFillers {
		fmt.Fprintf(&b, `<xsl:template match="filler%d" priority="-1"><f/></xsl:template>`, i)
	}
	b.WriteString(`</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(b.String()))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(), []byte(
		`<root><item id="1"/><item id="2"/><item id="3"/></root>`))
	require.NoError(t, err)

	out, err := ss.Transform(source).Serialize(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(out, "<hit/>"), "exactly the item with id=2 must match via key()")
	require.Equal(t, 2, strings.Count(out, "<miss/>"))
}
