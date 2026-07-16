package helium_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// dtdLoadDoc references an external subset by SYSTEM id. A valid content model
// for <doc> is declared in the external subset so a validating parse succeeds
// once the subset is actually loaded.
const dtdLoadDocName = "sub.dtd"

func dtdLoadDoc() string {
	return `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE doc SYSTEM "` + dtdLoadDocName + `">` + "\n" +
		`<doc>hello</doc>`
}

func dtdLoadFS() fstest.MapFS {
	return fstest.MapFS{
		dtdLoadDocName: &fstest.MapFile{Data: []byte("<!ELEMENT doc (#PCDATA)>\n")},
	}
}

// warnCaptureSAX embeds the default TreeBuilder (so the full DOM is still
// built) and records every SAX Warning delivered during the parse. A wrapping
// SAX handler takes the generic SAX-dispatch path (pctx.treeBuilder stays nil),
// which is a supported configuration.
type warnCaptureSAX struct {
	*helium.TreeBuilder
	warnings []error
}

func (w *warnCaptureSAX) Warning(_ context.Context, err error) error {
	w.warnings = append(w.warnings, err)
	return nil
}

// The external subset is loaded when load, default-attributes, or validation is
// requested; the three intents are independent and the load decision does not
// depend on the order the setters were called. ValidateDTD(true) must cause the
// external subset to load even when LoadExternalDTD(false) is also set, and the
// result must be identical regardless of call order.
func TestExternalDTDLoadOrderIndependent(t *testing.T) {
	t.Parallel()

	validateThenNoLoad := helium.NewParser().
		BlockXXE(false).
		FS(dtdLoadFS()).
		ValidateDTD(true).
		LoadExternalDTD(false)

	noLoadThenValidate := helium.NewParser().
		BlockXXE(false).
		FS(dtdLoadFS()).
		LoadExternalDTD(false).
		ValidateDTD(true)

	doc1, err1 := validateThenNoLoad.Parse(t.Context(), []byte(dtdLoadDoc()))
	require.NoError(t, err1, "validation should succeed once the external subset is loaded")
	require.NotNil(t, doc1)

	doc2, err2 := noLoadThenValidate.Parse(t.Context(), []byte(dtdLoadDoc()))
	require.NoError(t, err2)
	require.NotNil(t, doc2)

	// Both orders resolve the load decision the same way: validation causes the
	// external subset to load, LoadExternalDTD(false) notwithstanding.
	require.NotNil(t, doc1.ExtSubset(), "validation must load the external subset regardless of LoadExternalDTD(false)")
	require.NotNil(t, doc2.ExtSubset(), "call order must not change the load decision")
}

// Turning DefaultDTDAttributes back off must clear the load intent it set. A
// DefaultDTDAttributes(true) followed by DefaultDTDAttributes(false) must leave
// the parser NOT loading the external subset — the load bit does not get stuck
// on. DefaultDTDAttributes(true) on its own does load.
func TestExternalDTDDefaultAttrsToggleClearsLoad(t *testing.T) {
	t.Parallel()

	toggledOff := helium.NewParser().
		BlockXXE(false).
		FS(dtdLoadFS()).
		DefaultDTDAttributes(true).
		DefaultDTDAttributes(false)

	doc, err := toggledOff.Parse(t.Context(), []byte(dtdLoadDoc()))
	require.NoError(t, err)
	require.NotNil(t, doc)
	require.Nil(t, doc.ExtSubset(), "toggling DefaultDTDAttributes off must not leave the load bit stuck on")

	on := helium.NewParser().
		BlockXXE(false).
		FS(dtdLoadFS()).
		DefaultDTDAttributes(true)

	doc2, err2 := on.Parse(t.Context(), []byte(dtdLoadDoc()))
	require.NoError(t, err2)
	require.NotNil(t, doc2)
	require.NotNil(t, doc2.ExtSubset(), "DefaultDTDAttributes(true) must load the external subset")
}

// A requested external-subset load that fails to open must surface a non-fatal
// warning instead of being silently swallowed. The parse stays lenient (no
// fatal error, document still returned), matching libxml2.
func TestExternalDTDFailedLoadWarns(t *testing.T) {
	t.Parallel()

	capture := &warnCaptureSAX{TreeBuilder: helium.NewTreeBuilder()}

	// Empty filesystem: the SYSTEM "sub.dtd" open fails.
	doc, err := helium.NewParser().
		BlockXXE(false).
		FS(fstest.MapFS{}).
		SAXHandler(capture).
		LoadExternalDTD(true).
		Parse(t.Context(), []byte(dtdLoadDoc()))

	require.NoError(t, err, "a failed external-subset load stays non-fatal")
	require.NotNil(t, doc)
	require.NotEmpty(t, capture.warnings, "a requested-but-failed external-subset load must emit a warning")

	var lvl helium.ErrorLeveler
	require.ErrorAs(t, capture.warnings[0], &lvl)
	require.Equal(t, helium.ErrorLevelWarning, lvl.ErrorLevel())
	require.Contains(t, capture.warnings[0].Error(), dtdLoadDocName,
		"the warning must name the external DTD that failed to load")
}
