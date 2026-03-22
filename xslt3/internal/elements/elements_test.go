package elements_test

import (
	"testing"

	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xslt3/internal/elements"
	"github.com/stretchr/testify/require"
)

// xsltElementVersion mirrors the version map from xslt3/functions.go.
// Every element listed here must be known to the registry.
var xsltElementVersion = map[string]string{
	// XSLT 1.0
	"apply-imports": "1.0", "apply-templates": "1.0", "attribute": "1.0",
	"attribute-set": "1.0", "call-template": "1.0", "choose": "1.0",
	"comment": "1.0", "copy": "1.0", "copy-of": "1.0",
	"decimal-format": "1.0", "element": "1.0",
	"fallback": "1.0", "for-each": "1.0", "if": "1.0",
	"import": "1.0", "include": "1.0", "key": "1.0",
	"message": "1.0", "namespace-alias": "1.0",
	"number": "1.0", "otherwise": "1.0", "output": "1.0",
	"param": "1.0", "processing-instruction": "1.0",
	"sort": "1.0", "strip-space": "1.0",
	"preserve-space": "1.0", "stylesheet": "1.0", "template": "1.0",
	"text": "1.0", "transform": "1.0", "value-of": "1.0",
	"variable": "1.0", "when": "1.0", "with-param": "1.0",
	// XSLT 2.0
	"analyze-string": "2.0", "character-map": "2.0", "document": "2.0",
	"for-each-group": "2.0", "function": "2.0",
	"import-schema": "2.0", "matching-substring": "2.0",
	"namespace": "2.0", "next-match": "2.0",
	"non-matching-substring": "2.0", "output-character": "2.0",
	"perform-sort": "2.0", "result-document": "2.0", "sequence": "2.0",
	// XSLT 3.0
	"try": "3.0", "catch": "3.0", "evaluate": "3.0", "where-populated": "3.0",
	"on-empty": "3.0", "on-non-empty": "3.0", "merge": "3.0",
	"merge-source": "3.0", "merge-action": "3.0", "merge-key": "3.0",
	"assert": "3.0", "accumulator": "3.0", "accumulator-rule": "3.0",
	"fork": "3.0", "iterate": "3.0", "break": "3.0",
	"next-iteration": "3.0", "map": "3.0", "map-entry": "3.0",
	"array": "3.0", "accept": "3.0", "expose": "3.0",
	"override": "3.0", "use-package": "3.0", "package": "3.0",
	"global-context-item": "3.0", "context-item": "3.0",
	"source-document": "3.0", "mode": "3.0", "on-completion": "3.0",
}

func TestAllVersionedElementsAreKnown(t *testing.T) {
	r := elements.NewRegistry()
	for name := range xsltElementVersion {
		require.True(t, r.IsKnown(name), "element %q from xsltElementVersion is not in registry", name)
	}
}

func TestTopLevelElements(t *testing.T) {
	r := elements.NewRegistry()
	topLevel := []string{
		"import", "include", "use-package", "template",
		"variable", "param", "key", "output",
		"strip-space", "preserve-space", "function",
		"decimal-format", "mode", "import-schema",
		"accumulator", "attribute-set", "character-map",
		"namespace-alias", "expose", "global-context-item",
	}
	for _, name := range topLevel {
		require.True(t, r.IsTopLevel(name), "element %q should be top-level", name)
	}
	// Non-top-level elements
	for _, name := range []string{"if", "choose", "for-each", "value-of", "sort"} {
		require.False(t, r.IsTopLevel(name), "element %q should NOT be top-level", name)
	}
}

func TestInstructionElements(t *testing.T) {
	r := elements.NewRegistry()
	instructions := []string{
		"apply-templates", "call-template", "value-of", "text",
		"element", "attribute", "comment", "processing-instruction",
		"if", "choose", "for-each", "variable", "copy", "copy-of",
		"number", "message", "namespace", "sequence", "perform-sort",
		"next-match", "apply-imports", "document", "result-document",
		"where-populated", "on-empty", "on-non-empty", "try",
		"for-each-group", "map", "map-entry", "assert",
		"analyze-string", "evaluate", "source-document",
		"iterate", "fork", "merge",
	}
	for _, name := range instructions {
		require.True(t, r.IsInstruction(name), "element %q should be an instruction", name)
	}
	// Non-instruction elements
	for _, name := range []string{"template", "output", "key", "sort", "when", "otherwise"} {
		require.False(t, r.IsInstruction(name), "element %q should NOT be an instruction", name)
	}
}

func TestRootElements(t *testing.T) {
	r := elements.NewRegistry()
	for _, name := range []string{
		lexicon.XSLTElementStylesheet,
		lexicon.XSLTElementTransform,
		lexicon.XSLTElementPackage,
	} {
		info, ok := r.AllowedAttrs(name)
		_ = info
		require.True(t, ok, "root element %q should be known", name)
	}
}

func TestChildOnlyElements(t *testing.T) {
	r := elements.NewRegistry()
	childOnly := []string{
		"sort", "when", "otherwise", "catch", "with-param",
		"fallback", "matching-substring", "non-matching-substring",
		"on-completion", "merge-source", "merge-action", "merge-key",
		"accumulator-rule", "output-character", "context-item",
		"break", "next-iteration", "accept", "override",
	}
	for _, name := range childOnly {
		parents := r.ValidParents(name)
		// fallback has nil parents (any XSLT instruction)
		if name == "fallback" {
			require.Nil(t, parents, "fallback should have nil parents")
			continue
		}
		require.NotNil(t, parents, "child-only element %q should have parents", name)
		require.NotEmpty(t, parents, "child-only element %q should have at least one parent", name)
	}
}

func TestIsValidChild(t *testing.T) {
	r := elements.NewRegistry()

	// Valid combinations
	require.True(t, r.IsValidChild("sort", "apply-templates"))
	require.True(t, r.IsValidChild("sort", "for-each"))
	require.True(t, r.IsValidChild("when", "choose"))
	require.True(t, r.IsValidChild("otherwise", "choose"))
	require.True(t, r.IsValidChild("catch", "try"))
	require.True(t, r.IsValidChild("with-param", "call-template"))
	require.True(t, r.IsValidChild("with-param", "evaluate"))
	require.True(t, r.IsValidChild("matching-substring", "analyze-string"))
	require.True(t, r.IsValidChild("merge-source", "merge"))
	require.True(t, r.IsValidChild("merge-key", "merge-source"))
	require.True(t, r.IsValidChild("on-completion", "iterate"))
	require.True(t, r.IsValidChild("accumulator-rule", "accumulator"))
	require.True(t, r.IsValidChild("output-character", "character-map"))
	require.True(t, r.IsValidChild("context-item", "template"))
	require.True(t, r.IsValidChild("param", "template"))
	require.True(t, r.IsValidChild("param", "function"))
	require.True(t, r.IsValidChild("param", "iterate"))

	// Invalid combinations
	require.False(t, r.IsValidChild("sort", "if"))
	require.False(t, r.IsValidChild("when", "if"))
	require.False(t, r.IsValidChild("catch", "choose"))
	require.False(t, r.IsValidChild("with-param", "for-each"))
	require.False(t, r.IsValidChild("merge-key", "merge"))

	// fallback: always false because Parents is nil
	require.False(t, r.IsValidChild("fallback", "if"))
}

func TestMinVersion(t *testing.T) {
	r := elements.NewRegistry()

	// XSLT 1.0
	require.Equal(t, "1.0", r.MinVersion("apply-templates"))
	require.Equal(t, "1.0", r.MinVersion("if"))
	require.Equal(t, "1.0", r.MinVersion("variable"))
	require.Equal(t, "1.0", r.MinVersion("stylesheet"))

	// XSLT 2.0
	require.Equal(t, "2.0", r.MinVersion("analyze-string"))
	require.Equal(t, "2.0", r.MinVersion("function"))
	require.Equal(t, "2.0", r.MinVersion("sequence"))

	// XSLT 3.0
	require.Equal(t, "3.0", r.MinVersion("try"))
	require.Equal(t, "3.0", r.MinVersion("iterate"))
	require.Equal(t, "3.0", r.MinVersion("merge"))
	require.Equal(t, "3.0", r.MinVersion("package"))

	// Unknown
	require.Equal(t, "", r.MinVersion("nonexistent"))
}

func TestIsKnownUnknown(t *testing.T) {
	r := elements.NewRegistry()
	require.False(t, r.IsKnown("nonexistent"))
	require.False(t, r.IsKnown(""))
	require.False(t, r.IsKnown("xsl:template")) // should be plain local name
}

func TestAllowedAttrs(t *testing.T) {
	r := elements.NewRegistry()

	// template
	attrs, ok := r.AllowedAttrs("template")
	require.True(t, ok)
	require.Contains(t, attrs, "match")
	require.Contains(t, attrs, "name")
	require.Contains(t, attrs, "priority")
	require.Contains(t, attrs, "mode")
	require.Contains(t, attrs, "as")

	// apply-templates
	attrs, ok = r.AllowedAttrs("apply-templates")
	require.True(t, ok)
	require.Contains(t, attrs, "select")
	require.Contains(t, attrs, "mode")

	// sort
	attrs, ok = r.AllowedAttrs("sort")
	require.True(t, ok)
	require.Contains(t, attrs, "select")
	require.Contains(t, attrs, "order")
	require.Contains(t, attrs, "collation")

	// with-param
	attrs, ok = r.AllowedAttrs("with-param")
	require.True(t, ok)
	require.Contains(t, attrs, "name")
	require.Contains(t, attrs, "select")
	require.Contains(t, attrs, "as")
	require.Contains(t, attrs, "tunnel")

	// param has required and static, but not tunnel... wait, param has tunnel
	attrs, ok = r.AllowedAttrs("param")
	require.True(t, ok)
	require.Contains(t, attrs, "required")
	require.Contains(t, attrs, "static")

	// choose has no element-specific attrs
	attrs, ok = r.AllowedAttrs("choose")
	require.True(t, ok)
	require.Nil(t, attrs)

	// unknown element
	_, ok = r.AllowedAttrs("nonexistent")
	require.False(t, ok)
}
