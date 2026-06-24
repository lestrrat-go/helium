package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// TestBuildURI exercises BuildURI across local-path, http, and absolute cases.
func TestBuildURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		systemID string
		base     string
		want     string
	}{
		{"absolute system id is returned verbatim", "http://x/a.dtd", "http://y/", "http://x/a.dtd"},
		{"relative against http base", "a.dtd", "http://host/dir/doc.xml", "http://host/dir/a.dtd"},
		{"relative against file path", "a.dtd", "/dir/doc.xml", "/dir/a.dtd"},
		{"absolute local path", "/abs/a.dtd", "/dir/doc.xml", "/abs/a.dtd"},
		// Windows shapes are plain strings, so the Windows behavior below is
		// exercised on any GOOS. A native Windows base must NOT route the drive
		// letter through url.Parse (which would emit "c:///a.dtd"); it resolves
		// with local-path (forward-slash) semantics.
		{"relative against windows backslash base", "child.xml", `C:\dir\main.xml`, "C:/dir/child.xml"},
		{"relative against windows forward-slash base", "a.dtd", "D:/dir/doc.xml", "D:/dir/a.dtd"},
		{"windows-absolute system id returned verbatim", `C:\abs\a.dtd`, `D:\dir\doc.xml`, `C:\abs\a.dtd`},
		{"interior dot-dot against windows base", "../sib/child.xml", `C:\a\b\main.xml`, "C:/a/sib/child.xml"},
		{"unc base resolves relative ref", "child.xml", `\\host\share\main.xml`, "//host/share/child.xml"},
		// An absolute-URI systemID stands on its own even when the base is a
		// native Windows path. Without the scheme check this collapsed "http://"
		// to "http:/" and joined it onto the drive-letter base (Windows-only
		// regression that broke the W3C resolve-uri/base-uri cluster).
		{"absolute http system id against windows drive base", "http://example.com/a/b", `D:\dir\doc.xsl`, "http://example.com/a/b"},
		{"absolute http system id against windows slash base", "http://example.com/a/b", "D:/dir/doc.xsl", "http://example.com/a/b"},
		{"absolute file system id against windows base", "file:///x/y", `C:\dir\doc.xsl`, "file:///x/y"},
		// A RELATIVE Windows base (backslashes, no drive — what filepath.Join
		// yields on Windows for a relative test path) must keep its directory so a
		// sibling entity resolves inside it. Without backslash-aware handling this
		// dropped to a bare "world.txt" and the external entity could not be found.
		{"sibling against relative windows base", "world.txt", `..\d\e\example.xml`, "../d/e/world.txt"},
		// A file: base with a Windows drive letter must yield a proper file: URI
		// (not the drive-rooted "/D:/..." path url.Parse exposes), so file-URI-aware
		// loaders convert it back to a native path. The POSIX file: base below
		// keeps returning a plain path, proving POSIX is unaffected.
		{"sibling against windows drive file uri", "nested.dtd", "file:///D:/tmp/t/inc.xml", "file:///D:/tmp/t/nested.dtd"},
		{"sibling against posix file uri", "nested.dtd", "file:///tmp/t/inc.xml", "/tmp/t/nested.dtd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, helium.BuildURI(tt.systemID, tt.base))
		})
	}
}

// TestNodeGetBaseAndSet exercises NodeGetBase with xml:base attributes and the
// SetNodeBaseURI override.
func TestNodeGetBaseAndSet(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	doc.SetURL("http://example.com/dir/doc.xml")

	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))

	child := doc.CreateElement("child")
	xmlNS := helium.NewNamespace("xml", "http://www.w3.org/XML/1998/namespace")
	require.NoError(t, child.SetLiteralAttributeNS("base", "sub/", xmlNS))
	require.NoError(t, root.AddChild(child))

	// The child's effective base resolves its xml:base against the doc URL.
	base := helium.NodeGetBase(doc, child)
	require.Contains(t, base, "sub")

	// A nil node yields an empty base.
	require.Equal(t, "", helium.NodeGetBase(doc, nil))

	// SetNodeBaseURI installs an explicit entity base URI that takes precedence.
	helium.SetNodeBaseURI(child, "http://other.example/")
	base = helium.NodeGetBase(doc, child)
	require.Contains(t, base, "other.example")
}

// TestNewLeveledError exercises the leveled-error type and its ErrorLeveler.
func TestNewLeveledError(t *testing.T) {
	t.Parallel()

	err := helium.NewLeveledError("boom", helium.ErrorLevelError)
	require.EqualError(t, err, "boom")

	leveler, ok := err.(helium.ErrorLeveler)
	require.True(t, ok, "leveled error implements ErrorLeveler")
	require.Equal(t, helium.ErrorLevelError, leveler.ErrorLevel())
}

// TestNilErrorHandlerDiscards verifies NilErrorHandler discards without panicking.
func TestNilErrorHandlerDiscards(t *testing.T) {
	t.Parallel()
	var h helium.NilErrorHandler
	h.Handle(t.Context(), helium.NewLeveledError("ignored", helium.ErrorLevelWarning))
}

// TestAttributeNodeMethods exercises the Attribute node-interface methods.
func TestAttributeNodeMethods(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	attr, err := doc.CreateAttribute("a", "v", nil)
	require.NoError(t, err)

	require.Equal(t, "v", attr.Value())
	require.Equal(t, "a", attr.Name())

	attr.SetAType(enum.AttrCDATA)
	require.Equal(t, enum.AttrCDATA, attr.AType())

	attr.SetDefault(true)
	require.True(t, attr.IsDefault())

	// AppendText extends the attribute value (text child).
	require.NoError(t, attr.AppendText([]byte("-more")))

	attr.SetTreeDoc(doc)
}

// TestElementGetAttribute exercises GetAttribute and GetAttributeNS.
func TestElementGetAttribute(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	elem := doc.CreateElement("e")
	_, err := elem.SetAttribute("plain", "p")
	require.NoError(t, err)

	ns := helium.NewNamespace("x", "urn:x")
	_, err = elem.SetAttributeNS("nsed", "n", ns)
	require.NoError(t, err)

	v, ok := elem.GetAttribute("plain")
	require.True(t, ok)
	require.Equal(t, "p", v)

	_, ok = elem.GetAttribute("absent")
	require.False(t, ok)

	v, ok = elem.GetAttributeNS("nsed", "urn:x")
	require.True(t, ok)
	require.Equal(t, "n", v)

	_, ok = elem.GetAttributeNS("nsed", "urn:wrong")
	require.False(t, ok)
}
