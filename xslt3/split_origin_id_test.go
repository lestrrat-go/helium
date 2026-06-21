package xslt3

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// splitOriginIDs compiles a stylesheet source and returns, per mode, the
// non-zero splitOriginID values carried by union-split match templates.
func splitOriginIDs(t *testing.T, src string) []int64 {
	t.Helper()
	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	require.NoError(t, err)
	ss, err := compile(context.Background(), doc, &compileConfig{})
	require.NoError(t, err)
	var ids []int64
	for _, tmpls := range ss.modeTemplates {
		for _, tmpl := range tmpls {
			if tmpl.splitOriginID != 0 {
				ids = append(ids, tmpl.splitOriginID)
			}
		}
	}
	return ids
}

// UNRES-8: the union-split origin id must be unique across the whole
// compilation, not merely within a single stylesheet/package compile. A
// per-stylesheet counter restarts at 0 for every compile(), so two independent
// compiles would hand their first union split the same id — and under
// xsl:use-package those splits can be merged into a single mode list where the
// on-multiple-match conflict check treats equal ids as one rule, wrongly
// suppressing a genuine cross-package XTDE0540. A process-global counter
// guarantees splits from different compiles never collide.
func TestSplitOriginIDUniqueAcrossCompiles(t *testing.T) {
	const src = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="a | b"/>
</xsl:stylesheet>`

	idsA := splitOriginIDs(t, src)
	idsB := splitOriginIDs(t, src)
	require.NotEmpty(t, idsA, "union pattern should produce at least one split")
	require.NotEmpty(t, idsB)

	// Within one compile, every branch of the SAME union rule shares one id.
	for _, id := range idsA[1:] {
		require.Equal(t, idsA[0], id, "branches of one union rule must share an origin id")
	}
	for _, id := range idsB[1:] {
		require.Equal(t, idsB[0], id, "branches of one union rule must share an origin id")
	}

	// Across two independent compiles, the ids must NOT collide. With a
	// per-stylesheet counter both would be 1; with a global counter they differ.
	require.NotEqual(t, idsA[0], idsB[0],
		"split origin ids from independent compiles must be distinct so cross-package conflicts are not suppressed")
}

// Distinct union rules within a single compile must also receive distinct
// origin ids (one id per union rule, shared only among its own branches).
func TestSplitOriginIDDistinctPerUnionRule(t *testing.T) {
	const src = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="a | b" mode="m1"/>
  <xsl:template match="c | d" mode="m2"/>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	require.NoError(t, err)
	ss, err := compile(context.Background(), doc, &compileConfig{})
	require.NoError(t, err)

	idByMode := map[string]int64{}
	for mode, tmpls := range ss.modeTemplates {
		for _, tmpl := range tmpls {
			if tmpl.splitOriginID == 0 {
				continue
			}
			if existing, ok := idByMode[mode]; ok {
				require.Equal(t, existing, tmpl.splitOriginID,
					"branches of the same union rule (mode %q) must share an id", mode)
				continue
			}
			idByMode[mode] = tmpl.splitOriginID
		}
	}
	require.Contains(t, idByMode, "m1")
	require.Contains(t, idByMode, "m2")
	require.NotEqual(t, idByMode["m1"], idByMode["m2"],
		"different union rules must receive distinct origin ids")
}
