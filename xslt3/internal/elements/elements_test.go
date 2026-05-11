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
	"apply-imports": lexicon.XSLTVersion10, "apply-templates": lexicon.XSLTVersion10, "attribute": lexicon.XSLTVersion10,
	"attribute-set": lexicon.XSLTVersion10, "call-template": lexicon.XSLTVersion10, "choose": lexicon.XSLTVersion10,
	"comment": lexicon.XSLTVersion10, "copy": lexicon.XSLTVersion10, "copy-of": lexicon.XSLTVersion10,
	"decimal-format": lexicon.XSLTVersion10, "element": lexicon.XSLTVersion10,
	"fallback": lexicon.XSLTVersion10, "for-each": lexicon.XSLTVersion10, "if": lexicon.XSLTVersion10,
	"import": lexicon.XSLTVersion10, "include": lexicon.XSLTVersion10, "key": lexicon.XSLTVersion10,
	"message": lexicon.XSLTVersion10, "namespace-alias": lexicon.XSLTVersion10,
	"number": lexicon.XSLTVersion10, "otherwise": lexicon.XSLTVersion10, "output": lexicon.XSLTVersion10,
	"param": lexicon.XSLTVersion10, "processing-instruction": lexicon.XSLTVersion10,
	"sort": lexicon.XSLTVersion10, "strip-space": lexicon.XSLTVersion10,
	"preserve-space": lexicon.XSLTVersion10, "stylesheet": lexicon.XSLTVersion10, "template": lexicon.XSLTVersion10,
	"text": lexicon.XSLTVersion10, "transform": lexicon.XSLTVersion10, "value-of": lexicon.XSLTVersion10,
	"variable": lexicon.XSLTVersion10, "when": lexicon.XSLTVersion10, "with-param": lexicon.XSLTVersion10,
	// XSLT 2.0
	"analyze-string": lexicon.XSLTVersion20, "character-map": lexicon.XSLTVersion20, "document": lexicon.XSLTVersion20,
	"for-each-group": lexicon.XSLTVersion20, "function": lexicon.XSLTVersion20,
	"import-schema": lexicon.XSLTVersion20, "matching-substring": lexicon.XSLTVersion20,
	"namespace": lexicon.XSLTVersion20, "next-match": lexicon.XSLTVersion20,
	"non-matching-substring": lexicon.XSLTVersion20, "output-character": lexicon.XSLTVersion20,
	"perform-sort": lexicon.XSLTVersion20, "result-document": lexicon.XSLTVersion20, "sequence": lexicon.XSLTVersion20,
	// XSLT 3.0
	"try": lexicon.XSLTVersion30, "catch": lexicon.XSLTVersion30, "evaluate": lexicon.XSLTVersion30, "where-populated": lexicon.XSLTVersion30,
	"on-empty": lexicon.XSLTVersion30, "on-non-empty": lexicon.XSLTVersion30, "merge": lexicon.XSLTVersion30,
	"merge-source": lexicon.XSLTVersion30, "merge-action": lexicon.XSLTVersion30, "merge-key": lexicon.XSLTVersion30,
	"assert": lexicon.XSLTVersion30, "accumulator": lexicon.XSLTVersion30, "accumulator-rule": lexicon.XSLTVersion30,
	"fork": lexicon.XSLTVersion30, "iterate": lexicon.XSLTVersion30, "break": lexicon.XSLTVersion30,
	"next-iteration": lexicon.XSLTVersion30, "map": lexicon.XSLTVersion30, "map-entry": lexicon.XSLTVersion30,
	"array": lexicon.XSLTVersion30, "accept": lexicon.XSLTVersion30, "expose": lexicon.XSLTVersion30,
	"override": lexicon.XSLTVersion30, "use-package": lexicon.XSLTVersion30, "package": lexicon.XSLTVersion30,
	"global-context-item": lexicon.XSLTVersion30, "context-item": lexicon.XSLTVersion30,
	"source-document": lexicon.XSLTVersion30, "mode": lexicon.XSLTVersion30, "on-completion": lexicon.XSLTVersion30,
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
	require.Equal(t, lexicon.XSLTVersion10, r.MinVersion("apply-templates"))
	require.Equal(t, lexicon.XSLTVersion10, r.MinVersion("if"))
	require.Equal(t, lexicon.XSLTVersion10, r.MinVersion("variable"))
	require.Equal(t, lexicon.XSLTVersion10, r.MinVersion("stylesheet"))

	// XSLT 2.0
	require.Equal(t, lexicon.XSLTVersion20, r.MinVersion("analyze-string"))
	require.Equal(t, lexicon.XSLTVersion20, r.MinVersion("function"))
	require.Equal(t, lexicon.XSLTVersion20, r.MinVersion("sequence"))

	// XSLT 3.0
	require.Equal(t, lexicon.XSLTVersion30, r.MinVersion("try"))
	require.Equal(t, lexicon.XSLTVersion30, r.MinVersion("iterate"))
	require.Equal(t, lexicon.XSLTVersion30, r.MinVersion("merge"))
	require.Equal(t, lexicon.XSLTVersion30, r.MinVersion("package"))

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
