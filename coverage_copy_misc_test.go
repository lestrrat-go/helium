package helium_test

import (
	"os"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestCopyDTDInfo copies the internal-subset DTD declarations from one document
// into another via CopyDTDInfo.
func TestCopyDTDInfo(t *testing.T) {
	t.Parallel()

	in, err := os.ReadFile("test/att12.xml")
	require.NoError(t, err)
	src, err := helium.NewParser().Parse(t.Context(), in)
	require.NoError(t, err)
	require.NotNil(t, src.IntSubset())

	dst := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	helium.CopyDTDInfo(src, dst)

	require.NotNil(t, dst.IntSubset(), "CopyDTDInfo populates the destination internal subset")
	_, ok := dst.IntSubset().LookupNotation("gif")
	require.True(t, ok, "notation copied via CopyDTDInfo")

	// nil arguments are a no-op (no panic).
	helium.CopyDTDInfo(nil, dst)
	helium.CopyDTDInfo(src, nil)
}

// TestCopyExtSubset copies an external DTD subset between documents.
func TestCopyExtSubset(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dtdPath := dir + "/ext.dtd"
	require.NoError(t, os.WriteFile(dtdPath, []byte(`<!ELEMENT root (#PCDATA)>
<!NOTATION gif SYSTEM "viewer.exe">
<!ENTITY ext SYSTEM "data.xml">`), 0600))

	xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "` + dtdPath + `">
<root/>`

	src, err := helium.NewParser().LoadExternalDTD(true).Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	require.NotNil(t, src.ExtSubset())

	dst := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	helium.CopyExtSubset(src, dst)
	require.NotNil(t, dst.ExtSubset(), "external subset copied")

	// nil arguments are a no-op.
	helium.CopyExtSubset(nil, dst)
	helium.CopyExtSubset(src, nil)
}

// TestElementTypeString exercises the ElementType Stringer, including the
// out-of-range fallback.
func TestElementTypeString(t *testing.T) {
	t.Parallel()

	require.Equal(t, "ElementNode", helium.ElementNode.String())
	require.Equal(t, "DocumentNode", helium.DocumentNode.String())

	// An out-of-range value falls back to the numeric form.
	require.Contains(t, helium.ElementType(9999).String(), "ElementType(")
}

// TestNotationNodeInterfaceMethods exercises the remaining Notation node-method
// wrappers (AddSibling, Replace, Free, AddChild).
func TestNotationNodeInterfaceMethods(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("doc", "", "")
	require.NoError(t, err)
	nota, err := dtd.AddNotation("n", "", "sys")
	require.NoError(t, err)

	// These delegate to the shared tree primitives; exercise without asserting
	// implementation-defined success on an already-attached node.
	_ = nota.AddSibling(doc.CreateElement("x"))
	_ = nota.Replace()
	_ = nota.AddChild(doc.CreateText([]byte("t")))
	nota.Free()
}
