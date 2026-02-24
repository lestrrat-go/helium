package xinclude_test

import (
	"io"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xinclude"
	"github.com/stretchr/testify/require"
)

func parseXML(t *testing.T, s string) *helium.Document {
	t.Helper()
	doc, err := helium.Parse([]byte(s))
	require.NoError(t, err)
	return doc
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

func TestXIncludeBasicXML(t *testing.T) {
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="included.xml"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			"included.xml": `<chapter>Hello</chapter>`,
		},
	}

	count, err := xinclude.Process(doc,
		xinclude.WithResolver(resolver),
		xinclude.WithNoXIncludeNodes(),
		xinclude.WithNoBaseFixup(),
	)
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
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="data.txt" parse="text"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			"data.txt": "Hello World",
		},
	}

	count, err := xinclude.Process(doc,
		xinclude.WithResolver(resolver),
		xinclude.WithNoXIncludeNodes(),
		xinclude.WithNoBaseFixup(),
	)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	root := docElement(doc)
	require.Contains(t, string(root.Content()), "Hello World")
}

func TestXIncludeFallback(t *testing.T) {
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="missing.xml">
			<xi:fallback><fallback-content/></xi:fallback>
		</xi:include>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{},
	}

	count, err := xinclude.Process(doc,
		xinclude.WithResolver(resolver),
		xinclude.WithNoXIncludeNodes(),
		xinclude.WithNoBaseFixup(),
	)
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

func TestXIncludeMissingNoFallback(t *testing.T) {
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="missing.xml"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{},
	}

	_, err := xinclude.Process(doc,
		xinclude.WithResolver(resolver),
		xinclude.WithNoXIncludeNodes(),
	)
	require.Error(t, err)
}

func TestXIncludeCircularDetection(t *testing.T) {
	// self.xml includes self.xml — should be detected as circular
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="self.xml"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			"self.xml": `<root xmlns:xi="http://www.w3.org/2001/XInclude"><xi:include href="self.xml"/></root>`,
		},
	}

	_, err := xinclude.Process(doc,
		xinclude.WithResolver(resolver),
		xinclude.WithNoXIncludeNodes(),
		xinclude.WithNoBaseFixup(),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "circular")
}

func TestXIncludeMarkerNodes(t *testing.T) {
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="included.xml"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			"included.xml": `<chapter>Hello</chapter>`,
		},
	}

	count, err := xinclude.Process(doc,
		xinclude.WithResolver(resolver),
		xinclude.WithNoBaseFixup(),
	)
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

	count, err := xinclude.Process(doc,
		xinclude.WithResolver(resolver),
		xinclude.WithNoXIncludeNodes(),
		xinclude.WithNoBaseFixup(),
	)
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
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include/>
	</root>`)

	resolver := &stringResolver{files: map[string]string{}}

	_, err := xinclude.Process(doc,
		xinclude.WithResolver(resolver),
		xinclude.WithNoXIncludeNodes(),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing href")
}

func TestXIncludeBaseFixup(t *testing.T) {
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="sub/included.xml"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			"sub/included.xml": `<chapter>Hello</chapter>`,
		},
	}

	count, err := xinclude.Process(doc,
		xinclude.WithResolver(resolver),
		xinclude.WithNoXIncludeNodes(),
	)
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
	doc := parseXML(t, `<root xmlns:xi="http://www.w3.org/2001/XInclude">
		<xi:include href="outer.xml"/>
	</root>`)

	resolver := &stringResolver{
		files: map[string]string{
			"outer.xml": `<outer xmlns:xi="http://www.w3.org/2001/XInclude"><xi:include href="inner.xml"/></outer>`,
			"inner.xml": `<inner/>`,
		},
	}

	count, err := xinclude.Process(doc,
		xinclude.WithResolver(resolver),
		xinclude.WithNoXIncludeNodes(),
		xinclude.WithNoBaseFixup(),
	)
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
	doc := parseXML(t, `<root><a/><b/></root>`)

	count, err := xinclude.Process(doc, xinclude.WithNoXIncludeNodes())
	require.NoError(t, err)
	require.Equal(t, 0, count)
}
