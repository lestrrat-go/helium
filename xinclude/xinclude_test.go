package xinclude_test

import (
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xinclude"
	"github.com/stretchr/testify/require"
)

func parseXML(t *testing.T, s string) *helium.Document {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
	require.NoError(t, err)
	return doc
}

// fileURIFromPath builds a proper "file:///" URI from a native absolute path on
// any OS. Setting url.URL.Path to a Windows drive path ("C:/x") yields
// "file://C:/x", where "C:" is mis-parsed as the host; ensuring a single
// leading slash before the slash-normalized path yields "file:///C:/x" (and
// keeps "file:///tmp/x" on POSIX).
func fileURIFromPath(path string) string {
	p := filepath.ToSlash(path)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return (&url.URL{Scheme: "file", Path: p}).String()
}

func docElement(doc *helium.Document) *helium.Element {
	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		if n.Type() == helium.ElementNode {
			return n.(*helium.Element)
		}
	}
	return nil
}

// stringResolver is a test resolver that returns canned content.
type stringResolver struct {
	files map[string]string
}

func (r *stringResolver) Resolve(href, _ string) (io.ReadCloser, error) {
	content, ok := r.files[href]
	if !ok {
		return nil, &resolveError{href: href}
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

type resolveError struct {
	href string
}

func (e *resolveError) Error() string {
	return "not found: " + e.href
}

func TestXIncludeNewFSResolver(t *testing.T) {
	t.Parallel()

	t.Run("reads through injected FS", func(t *testing.T) {
		t.Parallel()
		doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
			<xi:include href="fs-target.xml"/>
		</root>`)

		fsys := fstest.MapFS{
			"fs-target.xml": &fstest.MapFile{Data: []byte(`<loaded>FromFS</loaded>`)},
		}
		count, err := xinclude.NewProcessor().
			Resolver(xinclude.NewFSResolver(fsys)).
			NoXIncludeMarkers().NoBaseFixup().
			Process(t.Context(), doc)
		require.NoError(t, err)
		require.Equal(t, 1, count)

		root := docElement(doc)
		require.NotNil(t, root)
		var found bool
		for c := root.FirstChild(); c != nil; c = c.NextSibling() {
			if c.Type() == helium.ElementNode && c.(*helium.Element).LocalName() == "loaded" {
				found = true
				require.Equal(t, "FromFS", string(c.Content()))
			}
		}
		require.True(t, found, "loaded element from FS not found")
	})

	t.Run("nil FS yields a permissive default", func(t *testing.T) {
		t.Parallel()
		require.NotNil(t, xinclude.NewFSResolver(nil))
	})

	t.Run("path is filepath.Clean'd before fs.Open", func(t *testing.T) {
		t.Parallel()

		// xi:include href="sub/../target.xml" resolves to "target.xml"
		// after cleaning. Without cleaning, fstest.MapFS would reject
		// the un-canonicalized name with fs.ErrInvalid.
		doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
			<xi:include href="sub/../target.xml"/>
		</root>`)

		fsys := fstest.MapFS{
			"target.xml": &fstest.MapFile{Data: []byte(`<ok/>`)},
		}
		count, err := xinclude.NewProcessor().
			Resolver(xinclude.NewFSResolver(fsys)).
			NoXIncludeMarkers().NoBaseFixup().
			Process(t.Context(), doc)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})

	t.Run("OS-native href separators are normalized for fs.FS", func(t *testing.T) {
		t.Parallel()

		// Upstream resolveURI uses filepath.Join which on Windows produces
		// backslash-separated paths. The fs.FS contract requires slash-
		// only names, so the resolver must call filepath.ToSlash before
		// Open. On Linux filepath.Join already returns slashes, so this
		// test is a no-op there; it exercises the conversion only on
		// Windows.
		href := filepath.Join("dir", "target.xml")
		fsys := fstest.MapFS{
			"dir/target.xml": &fstest.MapFile{Data: []byte(`<ok/>`)},
		}
		rc, err := xinclude.NewFSResolver(fsys).Resolve(href, "")
		require.NoError(t, err)
		t.Cleanup(func() { _ = rc.Close() })
	})

	t.Run("escaping href is rejected by os.DirFS-style FS", func(t *testing.T) {
		t.Parallel()

		// xi:include href="../escape.xml" resolves to "../escape.xml" after
		// cleaning. os.DirFS enforces fs.ValidPath and returns fs.ErrInvalid
		// for names that start with "..", proving path-traversal containment.
		// (Note: fstest.MapFS also rejects via fs.ValidPath but surfaces
		// fs.ErrNotExist, so we use os.DirFS to assert the specific reason.)
		doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
			<xi:include href="../escape.xml"/>
		</root>`)

		fsys := os.DirFS(t.TempDir())
		_, err := xinclude.NewProcessor().
			Resolver(xinclude.NewFSResolver(fsys)).
			NoXIncludeMarkers().NoBaseFixup().
			Process(t.Context(), doc)
		require.ErrorIs(t, err, fs.ErrInvalid, "expected fs.ValidPath rejection when href escapes FS root")
	})
}

// TestXIncludeSubdirRelativeBase verifies that an href relative to a base
// URI that itself lives in a subdirectory is resolved exactly once. The
// processor resolves the href against the base before handing it to the
// resolver, so the FS resolver must not join the base directory a second
// time (which would open dir/dir/inc.xml instead of dir/inc.xml).
func TestXIncludeSubdirRelativeBase(t *testing.T) {
	t.Parallel()

	t.Run("single subdirectory include", func(t *testing.T) {
		t.Parallel()
		doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
			<xi:include href="inc.xml"/>
		</root>`)

		fsys := fstest.MapFS{
			"dir/inc.xml": &fstest.MapFile{Data: []byte(`<loaded>Sub</loaded>`)},
		}
		count, err := xinclude.NewProcessor().
			Resolver(xinclude.NewFSResolver(fsys)).
			BaseURI("dir/main.xml").
			NoXIncludeMarkers().NoBaseFixup().
			Process(t.Context(), doc)
		require.NoError(t, err)
		require.Equal(t, 1, count)

		root := docElement(doc)
		require.NotNil(t, root)
		var found bool
		for c := root.FirstChild(); c != nil; c = c.NextSibling() {
			if c.Type() == helium.ElementNode && c.(*helium.Element).LocalName() == "loaded" {
				found = true
				require.Equal(t, "Sub", string(c.Content()))
			}
		}
		require.True(t, found, "included element from dir/inc.xml not found")
	})

	t.Run("chained base across nested subdirectory include", func(t *testing.T) {
		t.Parallel()
		// main.xml lives in dir/; it includes dir/outer.xml, which in turn
		// includes a sibling dir/inner.xml. The nested href must resolve
		// against the included file's directory, again exactly once.
		doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
			<xi:include href="outer.xml"/>
		</root>`)

		fsys := fstest.MapFS{
			"dir/outer.xml": &fstest.MapFile{Data: []byte(`<outer xmlns:xi="http://www.w3.org/2001/XInclude"><xi:include href="inner.xml"/></outer>`)},
			"dir/inner.xml": &fstest.MapFile{Data: []byte(`<inner/>`)},
		}
		count, err := xinclude.NewProcessor().
			Resolver(xinclude.NewFSResolver(fsys)).
			BaseURI("dir/main.xml").
			NoXIncludeMarkers().NoBaseFixup().
			Process(t.Context(), doc)
		require.NoError(t, err)
		require.Equal(t, 2, count)

		root := docElement(doc)
		var outer *helium.Element
		for c := root.FirstChild(); c != nil; c = c.NextSibling() {
			if c.Type() == helium.ElementNode {
				outer = c.(*helium.Element)
				break
			}
		}
		require.NotNil(t, outer)
		require.Equal(t, "outer", outer.LocalName())

		var inner *helium.Element
		for c := outer.FirstChild(); c != nil; c = c.NextSibling() {
			if c.Type() == helium.ElementNode {
				inner = c.(*helium.Element)
				break
			}
		}
		require.NotNil(t, inner)
		require.Equal(t, "inner", inner.LocalName())
	})

	t.Run("text include relative to subdirectory base", func(t *testing.T) {
		t.Parallel()
		doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
			<xi:include href="data.txt" parse="text"/>
		</root>`)

		fsys := fstest.MapFS{
			"dir/data.txt": &fstest.MapFile{Data: []byte(`Hello Sub`)},
		}
		count, err := xinclude.NewProcessor().
			Resolver(xinclude.NewFSResolver(fsys)).
			BaseURI("dir/main.xml").
			NoXIncludeMarkers().NoBaseFixup().
			Process(t.Context(), doc)
		require.NoError(t, err)
		require.Equal(t, 1, count)

		root := docElement(doc)
		require.Equal(t, "Hello Sub", strings.TrimSpace(string(root.Content())))
	})
}

// recordingResolver captures the (href, base) pairs it is asked to resolve
// and serves canned content keyed by the resolved href.
type recordingResolver struct {
	files map[string]string
	calls []struct{ href, base string }
}

func (r *recordingResolver) Resolve(href, base string) (io.ReadCloser, error) {
	r.calls = append(r.calls, struct{ href, base string }{href, base})
	content, ok := r.files[href]
	if !ok {
		return nil, &resolveError{href: href}
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

// TestXIncludeResolverReceivesResolvedHref documents the Resolver contract:
// the processor resolves the href against the effective base before calling
// Resolve, so the resolver receives the fully-resolved location and must not
// re-resolve it against base.
func TestXIncludeResolverReceivesResolvedHref(t *testing.T) {
	t.Parallel()

	t.Run("href resolved against document base", func(t *testing.T) {
		t.Parallel()
		doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
			<xi:include href="inc.xml"/>
		</root>`)

		resolver := &recordingResolver{
			files: map[string]string{
				// keyed by the already-resolved subdirectory path
				"dir/inc.xml": `<loaded/>`,
			},
		}
		count, err := xinclude.NewProcessor().
			Resolver(resolver).
			BaseURI("dir/main.xml").
			NoXIncludeMarkers().NoBaseFixup().
			Process(t.Context(), doc)
		require.NoError(t, err)
		require.Equal(t, 1, count)

		require.Len(t, resolver.calls, 1)
		require.Equal(t, "dir/inc.xml", resolver.calls[0].href,
			"resolver must receive the href already resolved against the base")
		require.Equal(t, "dir/main.xml", resolver.calls[0].base,
			"resolver should still receive the base URI as informational context")
	})

	t.Run("href and base reflect ancestor xml:base", func(t *testing.T) {
		t.Parallel()
		// An ancestor xml:base shifts the effective base into sub/, so the
		// href must resolve against dir/sub/ and the base argument handed to
		// the resolver must be the effective base, not the document base.
		doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude" xml:base="sub/">
			<xi:include href="inc.xml"/>
		</root>`)

		resolver := &recordingResolver{
			files: map[string]string{
				"dir/sub/inc.xml": `<loaded/>`,
			},
		}
		count, err := xinclude.NewProcessor().
			Resolver(resolver).
			BaseURI("dir/main.xml").
			NoXIncludeMarkers().NoBaseFixup().
			Process(t.Context(), doc)
		require.NoError(t, err)
		require.Equal(t, 1, count)

		require.Len(t, resolver.calls, 1)
		require.Equal(t, "dir/sub/inc.xml", resolver.calls[0].href,
			"href must be resolved against the effective base (ancestor xml:base applied)")
		require.Equal(t, "dir/sub/", resolver.calls[0].base,
			"base must be the effective base URI, not the document base")
	})
}

func TestXIncludeBasicXML(t *testing.T) {
	t.Parallel()
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="included.xml"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			includedXMLFile: `<chapter>Hello</chapter>`,
		},
	}

	count, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// The root should now contain a <chapter> element instead of xi:include
	root := docElement(doc)
	require.NotNil(t, root)

	var found bool
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			if c.(*helium.Element).LocalName() == "chapter" {
				found = true
				require.Equal(t, "Hello", string(c.Content()))
			}
		}
	}
	require.True(t, found, "included <chapter> element not found")
}

func TestXIncludeText(t *testing.T) {
	t.Parallel()
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="data.txt" parse="text"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			"data.txt": "Hello World",
		},
	}

	count, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	root := docElement(doc)
	require.Contains(t, string(root.Content()), "Hello World")
}

func TestXIncludeFallback(t *testing.T) {
	t.Parallel()
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="missing.xml">
			<xi:fallback><fallback-content/></xi:fallback>
		</xi:include>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{},
	}

	count, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	root := docElement(doc)
	var found bool
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			if c.(*helium.Element).LocalName() == "fallback-content" {
				found = true
			}
		}
	}
	require.True(t, found, "fallback content not found")
}

// TestXIncludeTextEmptyEncoding verifies that a present-but-empty encoding=""
// on a parse="text" include is treated as an unsupported encoding (a resource
// error), not as an absent encoding that would let the raw bytes be consumed as
// UTF-8. Without a fallback the inclusion must fail; with an xi:fallback the
// fallback content must be used.
func TestXIncludeTextEmptyEncoding(t *testing.T) {
	t.Parallel()

	t.Run("no fallback errors", func(t *testing.T) {
		t.Parallel()
		doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
			<xi:include href="data.txt" parse="text" encoding=""/>
		</root>`)

		resolver := &stringResolver{
			files: map[string]string{
				"data.txt": "Hello World",
			},
		}

		_, err := xinclude.NewProcessor().
			Resolver(resolver).
			NoXIncludeMarkers().
			NoBaseFixup().
			Process(t.Context(), doc)
		require.Error(t, err, "present-but-empty encoding must be an error, not silently UTF-8")
	})

	t.Run("fallback used", func(t *testing.T) {
		t.Parallel()
		doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
			<xi:include href="data.txt" parse="text" encoding="">
				<xi:fallback><fallback-content/></xi:fallback>
			</xi:include>
		</root>`)

		resolver := &stringResolver{
			files: map[string]string{
				"data.txt": "Hello World",
			},
		}

		count, err := xinclude.NewProcessor().
			Resolver(resolver).
			NoXIncludeMarkers().
			NoBaseFixup().
			Process(t.Context(), doc)
		require.NoError(t, err)
		require.Equal(t, 1, count)

		root := docElement(doc)
		var found bool
		for c := root.FirstChild(); c != nil; c = c.NextSibling() {
			if c.Type() == helium.ElementNode {
				if c.(*helium.Element).LocalName() == "fallback-content" {
					found = true
				}
			}
		}
		require.True(t, found, "fallback content not used for empty encoding")
	})
}

func TestXIncludeMissingNoFallback(t *testing.T) {
	t.Parallel()
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="missing.xml"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{},
	}

	_, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		Process(t.Context(), doc)
	require.Error(t, err)
}

func TestXIncludeCircularDetection(t *testing.T) {
	t.Parallel()
	// self.xml includes self.xml — should be detected as circular
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="self.xml"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			"self.xml": `<root xmlns:xi="http://www.w3.org/2001/XInclude"><xi:include href="self.xml"/></root>`,
		},
	}

	_, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "circular")
}

func TestXIncludeMarkerNodes(t *testing.T) {
	t.Parallel()
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="included.xml"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			includedXMLFile: `<chapter>Hello</chapter>`,
		},
	}

	count, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoBaseFixup().
		Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Should have XIncludeStart, <chapter>, XIncludeEnd
	root := docElement(doc)
	var types []helium.ElementType
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		types = append(types, c.Type())
	}

	// There may be text nodes (whitespace) mixed in
	var nonText []helium.ElementType
	for _, t := range types {
		if t != helium.TextNode {
			nonText = append(nonText, t)
		}
	}
	require.Len(t, nonText, 3)
	require.Equal(t, helium.XIncludeStartNode, nonText[0])
	require.Equal(t, helium.ElementNode, nonText[1])
	require.Equal(t, helium.XIncludeEndNode, nonText[2])
}

func TestXIncludeMultiple(t *testing.T) {
	t.Parallel()
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="a.xml"/>
		<xi:include href="b.xml"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			"a.xml": `<a/>`,
			"b.xml": `<b/>`,
		},
	}

	count, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	root := docElement(doc)
	var elems []string
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			elems = append(elems, c.(*helium.Element).LocalName())
		}
	}
	require.Equal(t, []string{"a", "b"}, elems)
}

func TestXIncludeNoHref(t *testing.T) {
	t.Parallel()
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include/>
	</root>`)

	resolver := &stringResolver{files: map[string]string{}}

	_, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		Process(t.Context(), doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing href")
}

func TestXIncludeBaseFixup(t *testing.T) {
	t.Parallel()
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="sub/included.xml"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			"sub/included.xml": `<chapter>Hello</chapter>`,
		},
	}

	count, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// The included element should have xml:base set
	root := docElement(doc)
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			elem := c.(*helium.Element)
			if elem.LocalName() == "chapter" {
				found := false
				for _, a := range elem.Attributes() {
					if a.Name() == "xml:base" {
						found = true
						require.Equal(t, "sub/included.xml", a.Value())
					}
				}
				require.True(t, found, "xml:base attribute not found on included element")
			}
		}
	}
}

func TestXIncludeNested(t *testing.T) {
	t.Parallel()
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="outer.xml"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			"outer.xml": `<outer xmlns:xi="http://www.w3.org/2001/XInclude"><xi:include href="inner.xml"/></outer>`,
			"inner.xml": `<inner/>`,
		},
	}

	count, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// root > outer > inner
	root := docElement(doc)
	var outer *helium.Element
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			outer = c.(*helium.Element)
			break
		}
	}
	require.NotNil(t, outer)
	require.Equal(t, "outer", outer.LocalName())

	var inner *helium.Element
	for c := outer.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			inner = c.(*helium.Element)
			break
		}
	}
	require.NotNil(t, inner)
	require.Equal(t, "inner", inner.LocalName())
}

func TestXIncludeNoIncludes(t *testing.T) {
	t.Parallel()
	doc := parseXML(t, `<root><a/><b/></root>`)

	count, err := xinclude.NewProcessor().NoXIncludeMarkers().Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

// --- New tests for added features ---

func TestXIncludeNewNamespace(t *testing.T) {
	t.Parallel()
	// Test with 2003 XInclude namespace
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2003/XInclude">
		<xi:include href="included.xml"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			includedXMLFile: `<chapter>Hello 2003</chapter>`,
		},
	}

	count, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	root := docElement(doc)
	var found bool
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			if c.(*helium.Element).LocalName() == "chapter" {
				found = true
				require.Equal(t, "Hello 2003", string(c.Content()))
			}
		}
	}
	require.True(t, found, "included <chapter> element not found")
}

func TestXIncludeNewNamespaceFallback(t *testing.T) {
	t.Parallel()
	// Test that fallback works with 2003 namespace
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2003/XInclude">
		<xi:include href="missing.xml">
			<xi:fallback><fallback-2003/></xi:fallback>
		</xi:include>
	</root>`)

	resolver := &stringResolver{files: map[string]string{}}

	count, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	root := docElement(doc)
	var found bool
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			if c.(*helium.Element).LocalName() == "fallback-2003" {
				found = true
			}
		}
	}
	require.True(t, found, "fallback content not found with 2003 namespace")
}

func TestXIncludeDepthLimit(t *testing.T) {
	t.Parallel()
	// Create a chain that would exceed maxDepth (40)
	resolver := &stringResolver{files: make(map[string]string)}
	for i := range 50 {
		next := i + 1
		resolver.files[fmt.Sprintf("level%d.xml", i)] = fmt.Sprintf(
			`<level xmlns:xi="http://www.w3.org/2001/XInclude"><xi:include href="level%d.xml"/></level>`, next)
	}
	resolver.files["level50.xml"] = `<leaf/>`

	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="level0.xml"/>
	</root>`)

	_, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "depth")
}

func TestXIncludeSameURLTwice(t *testing.T) {
	t.Parallel()
	// Same URL included at two non-nested positions should work (not circular)
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="shared.xml"/>
		<xi:include href="shared.xml"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			"shared.xml": `<shared/>`,
		},
	}

	count, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	root := docElement(doc)
	var elems []string
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			elems = append(elems, c.(*helium.Element).LocalName())
		}
	}
	require.Equal(t, []string{"shared", "shared"}, elems)
}

func TestXIncludeTextEncoding(t *testing.T) {
	t.Parallel()
	// Test that encoding attribute is honored for text inclusion
	// Create ISO-8859-1 encoded text: "caf\xe9" = "café" in latin1
	latin1Data := []byte{0x63, 0x61, 0x66, 0xe9}

	resolver := &byteResolver{
		files: map[string][]byte{
			"latin.txt": latin1Data,
		},
	}

	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="latin.txt" parse="text" encoding="ISO-8859-1"/>
	</root>`)

	count, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	root := docElement(doc)
	content := string(root.Content())
	require.Contains(t, content, "café")
}

func TestXIncludeTextUnsupportedEncoding(t *testing.T) {
	t.Parallel()
	// An unsupported requested encoding must be treated as a resource error:
	// the raw bytes must NOT be silently read as UTF-8. Use ASCII-safe data so
	// validateXMLChars cannot mask the bug by rejecting the bytes anyway.
	resolver := &byteResolver{
		files: map[string][]byte{
			"data.txt": []byte("hello"),
		},
	}

	t.Run("no fallback errors", func(t *testing.T) {
		t.Parallel()
		doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
			<xi:include href="data.txt" parse="text" encoding="bogus-encoding"/>
		</root>`)

		_, err := xinclude.NewProcessor().
			Resolver(resolver).
			NoXIncludeMarkers().
			NoBaseFixup().
			Process(t.Context(), doc)
		require.Error(t, err)
	})

	t.Run("fallback used", func(t *testing.T) {
		t.Parallel()
		doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
			<xi:include href="data.txt" parse="text" encoding="bogus-encoding">
				<xi:fallback><fallback-content/></xi:fallback>
			</xi:include>
		</root>`)

		count, err := xinclude.NewProcessor().
			Resolver(resolver).
			NoXIncludeMarkers().
			NoBaseFixup().
			Process(t.Context(), doc)
		require.NoError(t, err)
		require.Equal(t, 1, count)

		root := docElement(doc)
		var found bool
		for c := root.FirstChild(); c != nil; c = c.NextSibling() {
			if c.Type() == helium.ElementNode {
				if c.(*helium.Element).LocalName() == "fallback-content" {
					found = true
				}
			}
		}
		require.True(t, found, "fallback content not found")
	})
}

func TestXIncludeProcessTree(t *testing.T) {
	t.Parallel()
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<container>
			<xi:include href="a.xml"/>
		</container>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			"a.xml": `<item/>`,
		},
	}

	// Process from the document (same as Process)
	count, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		ProcessTree(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestXIncludeParseFlags(t *testing.T) {
	t.Parallel()
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="included.xml"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			includedXMLFile: `<chapter>Hello</chapter>`,
		},
	}

	count, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// With ParseNoXIncNode, there should be no marker nodes
	root := docElement(doc)
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		require.NotEqual(t, helium.XIncludeStartNode, c.Type(), "should not have XIncludeStart markers")
		require.NotEqual(t, helium.XIncludeEndNode, c.Type(), "should not have XIncludeEnd markers")
	}

	// With ParseNoBaseFix, there should be no xml:base attribute
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			elem := c.(*helium.Element)
			for _, a := range elem.Attributes() {
				require.NotEqual(t, "xml:base", a.Name(), "should not have xml:base with ParseNoBaseFix")
			}
		}
	}
}

// byteResolver is a test resolver that returns raw byte content.
type byteResolver struct {
	files map[string][]byte
}

func (r *byteResolver) Resolve(href, _ string) (io.ReadCloser, error) {
	content, ok := r.files[href]
	if !ok {
		return nil, &resolveError{href: href}
	}
	return io.NopCloser(strings.NewReader(string(content))), nil
}

// --- libxml2 golden file tests ---

func TestLibxml2XIncludeGolden(t *testing.T) {
	t.Parallel()
	docsDir, err := filepath.Abs(filepath.Join("..", "testdata", "libxml2-compat", "xinclude", "docs"))
	require.NoError(t, err)
	resultDir, err := filepath.Abs(filepath.Join("..", "testdata", "libxml2-compat", "xinclude", "result"))
	require.NoError(t, err)

	// Check if the libxml2 test data is available
	if _, statErr := os.Stat(docsDir); os.IsNotExist(statErr) {
		t.Skip("libxml2 test data not available; run testdata/libxml2/generate.sh first")
	}

	// Skip files that have issues beyond XPointer support
	skip := map[string]string{}

	entries, err := os.ReadDir(docsDir)
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".xml") {
			continue
		}

		name := entry.Name()
		if reason, ok := skip[name]; ok {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				t.Skip(reason)
			})
			continue
		}

		// Check for result file (success case) or .err file (error case)
		resultFile := filepath.Join(resultDir, name)
		errFile := filepath.Join(resultDir, name+".err")
		hasResult := false
		hasErr := false
		if _, statErr := os.Stat(resultFile); statErr == nil {
			hasResult = true
		}
		if _, statErr := os.Stat(errFile); statErr == nil {
			hasErr = true
		}
		if !hasResult && !hasErr {
			continue
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			docPath := filepath.Join(docsDir, name)
			data, err := os.ReadFile(docPath) //nolint:gosec // reading test data file from testdata directory
			require.NoError(t, err)

			doc, err := helium.NewParser().Parse(t.Context(), data)
			require.NoError(t, err, "parsing %s", name)

			_, procErr := xinclude.NewProcessor().
				NoXIncludeMarkers().
				BaseURI(docPath).
				Process(t.Context(), doc)

			if hasResult {
				// Success case: compare output against expected result
				require.NoError(t, procErr, "processing %s", name)

				got, err := helium.WriteString(doc)
				require.NoError(t, err)

				expected, err := os.ReadFile(resultFile) //nolint:gosec // reading test expected result from testdata directory
				require.NoError(t, err)

				require.Equal(t, string(expected), got, "output mismatch for %s", name)
			} else {
				// Error case: XInclude processing should have returned an error
				require.Error(t, procErr, "expected error processing %s", name)
			}
		})
	}
}

func TestLibxml2XIncludeWithoutReader(t *testing.T) {
	t.Parallel()
	wrDir, err := filepath.Abs(filepath.Join("..", "testdata", "libxml2-compat", "xinclude", "without-reader"))
	require.NoError(t, err)
	resultDir, err := filepath.Abs(filepath.Join("..", "testdata", "libxml2-compat", "xinclude", "result"))
	require.NoError(t, err)

	if _, statErr := os.Stat(wrDir); os.IsNotExist(statErr) {
		t.Skip("libxml2 without-reader test data not available")
	}

	// Tests that expect success: compare output against result file
	successTests := []string{
		"issue424-1.xml",
		"issue424-2.xml",
		"fallback7.xml",
		"ns1.xml",
	}

	for _, name := range successTests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			docPath := filepath.Join(wrDir, name)
			data, err := os.ReadFile(docPath) //nolint:gosec // reading test data file from testdata directory
			require.NoError(t, err)

			doc, err := helium.NewParser().Parse(t.Context(), data)
			require.NoError(t, err, "parsing %s", name)

			_, err = xinclude.NewProcessor().
				NoXIncludeMarkers().
				BaseURI(docPath).
				Process(t.Context(), doc)
			require.NoError(t, err, "processing %s", name)

			got, err := helium.WriteString(doc)
			require.NoError(t, err)

			resultFile := filepath.Join(resultDir, name)
			expected, err := os.ReadFile(resultFile) //nolint:gosec // reading test expected result
			require.NoError(t, err)

			require.Equal(t, string(expected), got, "output mismatch for %s", name)
		})
	}

	// Tests that expect errors
	errorTests := []struct {
		name    string
		contain string
	}{
		{"loop.xml", "circular"},
		{"max-recurse.xml", "depth"},
	}

	for _, tc := range errorTests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			docPath := filepath.Join(wrDir, tc.name)
			data, err := os.ReadFile(docPath) //nolint:gosec // reading test data file
			require.NoError(t, err)

			doc, err := helium.NewParser().Parse(t.Context(), data)
			require.NoError(t, err, "parsing %s", tc.name)

			_, err = xinclude.NewProcessor().
				NoXIncludeMarkers().
				BaseURI(docPath).
				Process(t.Context(), doc)
			require.Error(t, err, "expected error for %s", tc.name)
			require.Contains(t, err.Error(), tc.contain,
				"error for %s should contain %q", tc.name, tc.contain)
		})
	}
}

// --- Validation strictness tests ---

func TestXIncludeIncludeInInclude(t *testing.T) {
	t.Parallel()
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="a.xml">
			<xi:include href="b.xml"/>
		</xi:include>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			"a.xml": `<a/>`,
			"b.xml": `<b/>`,
		},
	}

	_, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "xi:include has an 'include' child")
}

func TestXIncludeMultipleFallback(t *testing.T) {
	t.Parallel()
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="missing.xml">
			<xi:fallback><a/></xi:fallback>
			<xi:fallback><b/></xi:fallback>
		</xi:include>
	</root>`)

	resolver := &stringResolver{files: map[string]string{}}

	_, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "xi:include has multiple fallback children")
}

func TestXIncludeFallbackOutsideInclude(t *testing.T) {
	t.Parallel()
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:fallback><a/></xi:fallback>
	</root>`)

	_, err := xinclude.NewProcessor().
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "xi:fallback is not the child of an 'include'")
}

func TestXIncludeURITooLong(t *testing.T) {
	t.Parallel()
	longHref := strings.Repeat("a", 2001) + ".xml"
	doc := parseXML(t, fmt.Sprintf(`<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="%s"/>
	</root>`, longHref))

	resolver := &stringResolver{files: map[string]string{}}

	_, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "URI too long")
}

func TestXIncludeNamespacedAttr(t *testing.T) {
	t.Parallel()
	// xi:include with namespace-qualified xi:href attribute should work
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include xi:href="included.xml"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			includedXMLFile: `<chapter>Namespaced</chapter>`,
		},
	}

	count, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	root := docElement(doc)
	var found bool
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			if c.(*helium.Element).LocalName() == "chapter" {
				found = true
				require.Equal(t, "Namespaced", string(c.Content()))
			}
		}
	}
	require.True(t, found, "included <chapter> element not found with namespace-qualified href")
}

func TestXIncludeParseNoEntWithXPointer(t *testing.T) {
	t.Parallel()
	// Document with entity reference that should be resolved before XPointer
	resolver := &stringResolver{
		files: map[string]string{
			"entities.xml": `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY greeting "Hello World">
]>
<root><item>&greeting;</item></root>`,
		},
	}

	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="entities.xml" xpointer="xpointer(/root/item)"/>
	</root>`)

	count, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	root := docElement(doc)
	var found bool
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			elem := c.(*helium.Element)
			if elem.LocalName() == "item" {
				found = true
				require.Equal(t, "Hello World", string(c.Content()))
			}
		}
	}
	require.True(t, found, "included <item> element not found")
}

// TestXIncludeXPointerNoXXE verifies that an external SYSTEM entity declared
// inside an XPointer-included document (parsed with entity substitution) cannot
// read host files behind a strict resolver's back. The resolver here is a
// custom (non-FS) resolver that only serves canned content, so the inner parser
// must not fall back to the host filesystem for the SYSTEM entity.
func TestXIncludeXPointerNoXXE(t *testing.T) {
	t.Parallel()

	secret := filepath.Join(t.TempDir(), "secret.txt")
	const marker = "TOP-SECRET-XXE-MARKER"
	require.NoError(t, os.WriteFile(secret, []byte(marker), 0o600))

	resolver := &stringResolver{
		files: map[string]string{
			"entities.xml": `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY xxe SYSTEM "` + secret + `">
]>
<root><item>&xxe;</item></root>`,
		},
	}

	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="entities.xml" xpointer="xpointer(/root/item)"/>
	</root>`)

	// Processing may succeed (entity unresolved) or fail; either is acceptable.
	// What must never happen is the secret file's contents leaking into output.
	_, _ = xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)

	got, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.NotContains(t, got, marker, "external SYSTEM entity leaked host file via XPointer include")
}

// fsWrapResolver wraps an FS-backed Resolver (e.g. NewFSResolver) for
// logging/metrics-style decoration while forwarding the optional FS()
// capability so XInclude still recognizes it as FS-backed.
type fsWrapResolver struct {
	inner xinclude.Resolver
	fsys  fs.FS
}

func (r *fsWrapResolver) Resolve(href, base string) (io.ReadCloser, error) {
	return r.inner.Resolve(href, base)
}

func (r *fsWrapResolver) FS() fs.FS { return r.fsys }

// TestXIncludeWrappedFSResolverInSandbox verifies that a resolver which wraps
// NewFSResolver but re-exposes FS() is still recognized as FS-backed: an
// external entity declared inside the included document loads through the SAME
// sandbox FS (not denied), so legitimate in-sandbox references keep working.
func TestXIncludeWrappedFSResolverInSandbox(t *testing.T) {
	t.Parallel()

	const extContent = "FROM-SANDBOX-ENTITY"
	fsys := fstest.MapFS{
		"included.xml": &fstest.MapFile{Data: []byte(`<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY ext SYSTEM "ext.txt">
]>
<root><item>&ext;</item></root>`)},
		"ext.txt": &fstest.MapFile{Data: []byte(extContent)},
	}

	resolver := &fsWrapResolver{inner: xinclude.NewFSResolver(fsys), fsys: fsys}

	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="included.xml" xpointer="xpointer(/root/item)"/>
	</root>`)

	_, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.NoError(t, err)

	got, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Contains(t, got, extContent,
		"wrapped FS-backed resolver should still load in-sandbox external entity")
}

// countingResolver wraps a stringResolver and counts Resolve calls per URI.
type countingResolver struct {
	inner *stringResolver
	calls map[string]int
}

func (r *countingResolver) Resolve(href, base string) (io.ReadCloser, error) {
	if r.calls == nil {
		r.calls = make(map[string]int)
	}
	r.calls[href]++
	return r.inner.Resolve(href, base)
}

func TestXIncludeDocCacheAvoidsReResolve(t *testing.T) {
	t.Parallel()
	// When the same URI is included multiple times, the resolver should
	// only be called once — subsequent includes reuse cached bytes.
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="shared.xml"/>
		<xi:include href="shared.xml"/>
		<xi:include href="shared.xml"/>
	</root>`)

	resolver := &countingResolver{
		inner: &stringResolver{
			files: map[string]string{
				"shared.xml": `<item>cached</item>`,
			},
		},
	}

	count, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 3, count)

	// Resolver should be called only once for "shared.xml"
	require.Equal(t, 1, resolver.calls["shared.xml"])

	// All three inclusions should produce independent nodes
	root := docElement(doc)
	var items int
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			items++
			require.Equal(t, "item", c.(*helium.Element).LocalName())
		}
	}
	require.Equal(t, 3, items)
}

func TestXIncludeEntityMerge(t *testing.T) {
	t.Parallel()
	// Included document defines entities; they should be merged into target's internal subset.
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="entities.xml"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			"entities.xml": `<?xml version="1.0"?>
<!DOCTYPE chapter [
  <!ENTITY greet "hello">
  <!ENTITY farewell "goodbye">
]>
<chapter>&greet;</chapter>`,
		},
	}

	count, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Target document should now have an internal subset with the merged entities
	intSub := doc.IntSubset()
	require.NotNil(t, intSub, "target document should have an internal subset")

	greet, ok := intSub.LookupEntity("greet")
	require.True(t, ok, "entity 'greet' should exist in target")
	require.Equal(t, "hello", string(greet.Content()))

	farewell, ok := intSub.LookupEntity("farewell")
	require.True(t, ok, "entity 'farewell' should exist in target")
	require.Equal(t, "goodbye", string(farewell.Content()))
}

func TestXIncludeEntityMergeConflict(t *testing.T) {
	t.Parallel()
	// Target and included document both define the same entity with different content.
	// Target's definition should win (first-definition-wins) and warning should fire.
	doc, err := helium.NewParser().LoadExternalDTD(true).Parse(t.Context(), []byte(`<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY greeting "target-value">
]>
<root xmlns:xi="http://www.w3.org/2001/XInclude">
	<xi:include href="conflict.xml"/>
</root>`))
	require.NoError(t, err)

	resolver := &stringResolver{
		files: map[string]string{
			"conflict.xml": `<?xml version="1.0"?>
<!DOCTYPE chapter [
  <!ENTITY greeting "included-value">
]>
<chapter>text</chapter>`,
		},
	}

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	count, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		ErrorHandler(collector).
		Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Target's definition wins
	intSub := doc.IntSubset()
	require.NotNil(t, intSub)
	ent, ok := intSub.LookupEntity("greeting")
	require.True(t, ok)
	require.Equal(t, "target-value", string(ent.Content()))

	// Warning should have been emitted
	_ = collector.Close()
	warnings := collector.Errors()
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0].Error(), "greeting")
	require.Contains(t, warnings[0].Error(), "mismatch")
}

func TestXIncludeEntityMergeNoTargetDTD(t *testing.T) {
	t.Parallel()
	// Target has no DTD; included document has entities.
	// An internal subset should be created on the target and entities merged.
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="with-entities.xml"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			"with-entities.xml": `<?xml version="1.0"?>
<!DOCTYPE section [
  <!ENTITY author "Alice">
]>
<section>&author;</section>`,
		},
	}

	count, err := xinclude.NewProcessor().
		Resolver(resolver).
		NoXIncludeMarkers().
		NoBaseFixup().
		Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Verify internal subset was created
	intSub := doc.IntSubset()
	require.NotNil(t, intSub, "internal subset should have been created on target")

	// Verify entity was merged
	ent, ok := intSub.LookupEntity("author")
	require.True(t, ok, "entity 'author' should exist in target")
	require.Equal(t, "Alice", string(ent.Content()))
}

func TestZeroValueProcessor(t *testing.T) {
	t.Parallel()
	// A document with no xi:include elements — Process should succeed with 0 substitutions.
	doc := parseXML(t, `<root><child>text</child></root>`)
	var proc xinclude.Processor
	count, err := proc.Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestZeroValueProcessorFluent(t *testing.T) {
	t.Parallel()
	doc := parseXML(t, `<root><child>text</child></root>`)
	var proc xinclude.Processor
	count, err := proc.NoXIncludeMarkers().Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

// endlessResolver returns a reader that never reaches EOF, simulating a hostile
// or pathological resolver whose response would exhaust memory if read fully.
type endlessResolver struct{}

func (endlessResolver) Resolve(string, string) (io.ReadCloser, error) {
	return io.NopCloser(repeatingReader{b: 'x'}), nil
}

type repeatingReader struct{ b byte }

func (r repeatingReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = r.b
	}
	return len(p), nil
}

func TestXIncludeMaxIncludeSize(t *testing.T) {
	t.Parallel()

	t.Run("text include over cap fails with ErrIncludeTooLarge", func(t *testing.T) {
		t.Parallel()
		doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
			<xi:include href="huge.txt" parse="text"/>
		</root>`)

		_, err := xinclude.NewProcessor().
			Resolver(endlessResolver{}).
			MaxIncludeSize(1024).
			Process(t.Context(), doc)
		require.Error(t, err)
		require.ErrorIs(t, err, xinclude.ErrIncludeTooLarge)
	})

	t.Run("xml include over cap fails with ErrIncludeTooLarge", func(t *testing.T) {
		t.Parallel()
		doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
			<xi:include href="huge.xml"/>
		</root>`)

		_, err := xinclude.NewProcessor().
			Resolver(endlessResolver{}).
			MaxIncludeSize(1024).
			Process(t.Context(), doc)
		require.Error(t, err)
		require.ErrorIs(t, err, xinclude.ErrIncludeTooLarge)
	})

	t.Run("include at or under cap succeeds", func(t *testing.T) {
		t.Parallel()
		doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
			<xi:include href="small.txt" parse="text"/>
		</root>`)

		res := &stringResolver{files: map[string]string{"small.txt": "hello"}}
		count, err := xinclude.NewProcessor().
			Resolver(res).
			MaxIncludeSize(1024).
			NoXIncludeMarkers().NoBaseFixup().
			Process(t.Context(), doc)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})
}

func TestXIncludeFileURIHref(t *testing.T) {
	t.Parallel()

	// Write a real include target and reference it via an absolute file:// URI.
	// The default permissive resolver must convert the URI to an OS path rather
	// than handing "file:/..." to os.Open verbatim.
	dir := t.TempDir()
	target := filepath.Join(dir, "inc.xml")
	require.NoError(t, os.WriteFile(target, []byte(`<loaded>FromFileURI</loaded>`), 0o600))

	fileURI := fileURIFromPath(target)

	doc := parseXML(t, fmt.Sprintf(`<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="%s"/>
	</root>`, fileURI))

	count, err := xinclude.NewProcessor().
		NoXIncludeMarkers().NoBaseFixup().
		Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	root := docElement(doc)
	require.NotNil(t, root)
	var found bool
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode && c.(*helium.Element).LocalName() == "loaded" {
			found = true
			require.Equal(t, "FromFileURI", string(c.Content()))
		}
	}
	require.True(t, found, "loaded element from file URI not found")
}

func TestXIncludeFileURINestedExternalDTD(t *testing.T) {
	t.Parallel()

	// Regression: a file:// XInclude whose included document references an
	// external DTD that defines a general entity. The included doc is parsed
	// with BaseURI("file:///.../inc.xml"), so the DTD SYSTEM ref resolves to a
	// "file:" URI handed to the inner parser's FS (normalizingFS). That FS must
	// convert the file: URI to a local path too, or the nested external DTD is
	// never found and its entity is silently missed. The entity-merge step then
	// surfaces the entity into the target's internal subset, proving the nested
	// DTD was actually located and parsed.
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "inc.dtd"),
		[]byte(`<!ENTITY greet "hello from nested dtd">`), 0o600))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "inc.xml"),
		[]byte(`<?xml version="1.0"?>`+
			`<!DOCTYPE chapter SYSTEM "inc.dtd">`+
			`<chapter>text</chapter>`), 0o600))

	fileURI := fileURIFromPath(filepath.Join(dir, "inc.xml"))

	doc := parseXML(t, fmt.Sprintf(`<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="%s"/>
	</root>`, fileURI))

	count, err := xinclude.NewProcessor().
		NoXIncludeMarkers().NoBaseFixup().
		Process(t.Context(), doc)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// The entity defined in the included document's EXTERNAL DTD must have been
	// merged into the target's internal subset. This only happens if the nested
	// "file:" DTD reference was converted to a local path and successfully read.
	intSub := doc.IntSubset()
	require.NotNil(t, intSub, "target should have a merged internal subset")
	greet, ok := intSub.LookupEntity("greet")
	require.True(t, ok, "entity from nested external DTD must be merged, proving the file: DTD URI was resolved")
	require.Equal(t, "hello from nested dtd", string(greet.Content()))
}

func TestXIncludeFileURINonLocalHost(t *testing.T) {
	t.Parallel()

	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="file://remotehost/tmp/inc.xml"/>
	</root>`)

	_, err := xinclude.NewProcessor().
		NoXIncludeMarkers().NoBaseFixup().
		Process(t.Context(), doc)
	require.Error(t, err)
	require.ErrorContains(t, err, "non-local file URI host",
		"expected explicit non-local host rejection, got: %v", err)
}

// TestXIncludeParserInjection verifies that a parser injected via
// Processor.Parser governs the resource limits used to parse included
// documents (here, the element-name-length cap), while XInclude continues to
// own the resolver-confined filesystem.
func TestXIncludeParserInjection(t *testing.T) {
	t.Parallel()

	const main = `<root xmlns:xi="http://www.w3.org/2001/XInclude"><xi:include href="inc.xml"/></root>`
	// Included document's element name "longname" is 8 bytes.
	incFS := fstest.MapFS{"inc.xml": &fstest.MapFile{Data: []byte(`<longname/>`)}}

	t.Run("no injection accepts the included name", func(t *testing.T) {
		t.Parallel()
		doc := parseXML(t, main)
		count, err := xinclude.NewProcessor().
			Resolver(xinclude.NewFSResolver(incFS)).
			Process(t.Context(), doc)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})

	t.Run("injected MaxNameLength is enforced on included docs", func(t *testing.T) {
		t.Parallel()
		doc := parseXML(t, main)
		_, err := xinclude.NewProcessor().
			Resolver(xinclude.NewFSResolver(incFS)).
			Parser(helium.NewParser().MaxNameLength(4)).
			Process(t.Context(), doc)
		require.Error(t, err, "injected parser's name-length limit must apply to included documents")
	})
}
