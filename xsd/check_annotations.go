package xsd

import (
	"context"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xmlchar"
)

// checkAnnotations enforces the schema-for-schemas XML representation of
// <xs:annotation> / <xs:appinfo> / <xs:documentation> (XSD Structures §3.13.2)
// that the various component parsers do not verify uniformly:
//
//   - an xs:annotation's content model is (appinfo | documentation)*: any other
//     element child (a stray XSD element such as a nested xs:annotation, or a
//     foreign element) and any non-whitespace character content is an error;
//   - the only unqualified attribute allowed on xs:annotation is id (foreign
//     namespaced attributes are allowed);
//   - xs:appinfo allows only the unqualified attribute source, whose value is
//     xs:anyURI;
//   - xs:documentation allows only the unqualified attribute source and the
//     xml:lang attribute; source is xs:anyURI and xml:lang (after xs:language
//     whiteSpace=collapse) must be a valid xs:language — an empty or
//     whitespace-only value is invalid;
//   - every XSD element whose content model is (annotation?, ...) admits at most
//     one xs:annotation child. Only xs:schema admits repeated annotations, so the
//     "at most one" rule applies to every XSD-namespace element except xs:schema
//     (and the annotation family itself, whose own content model is checked above).
//
// These are version-independent structural constraints, so the walk runs in both
// XSD 1.0 and 1.1. It mirrors checkNotations / checkSchemaComponentIDs and is
// invoked on the entry document and every included / imported / redefined /
// overridden document. It does NOT descend into xs:appinfo / xs:documentation
// payload, whose content is lax (arbitrary well-formed XML application data, not
// schema components).
func (c *compiler) checkAnnotations(ctx context.Context, root *helium.Element) {
	if c.filename == "" {
		return
	}
	c.walkAnnotations(ctx, root)
}

func (c *compiler) walkAnnotations(ctx context.Context, elem *helium.Element) {
	if elem.URI() == lexicon.NamespaceXSD {
		local := elem.LocalName()
		switch local {
		case elemAnnotation:
			c.checkAnnotationDecl(ctx, elem)
		case elemSchema, elemRedefine, elemOverride:
			// xs:schema, xs:redefine and xs:override have the open
			// (annotation | component)* content model and admit repeated
			// annotations, so no "at most one" cardinality check applies.
		default:
			c.checkAnnotationCardinality(ctx, elem)
		}
	}

	for child := range helium.Children(elem) {
		ce, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if ce.URI() == lexicon.NamespaceXSD && (ce.LocalName() == elemAppinfo || ce.LocalName() == elemDocumentation) {
			continue
		}
		c.walkAnnotations(ctx, ce)
	}
}

// checkAnnotationCardinality enforces the (annotation?, ...) rule: an XSD element
// may carry at most one xs:annotation child. A second annotation is a schema
// error. Position ("must be first") is left to the individual component parsers.
func (c *compiler) checkAnnotationCardinality(ctx context.Context, elem *helium.Element) {
	src := c.diagSource()
	if src == "" {
		return
	}
	seen := false
	for child := range helium.Children(elem) {
		ce, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if ce.URI() != lexicon.NamespaceXSD || ce.LocalName() != elemAnnotation {
			continue
		}
		if seen {
			c.schemaError(ctx, schemaParserError(src, ce.Line(), elem.LocalName(), elem.LocalName(),
				"Only one annotation is allowed. The content is not valid."))
			return
		}
		seen = true
	}
}

// checkAnnotationDecl validates a single xs:annotation element: its attributes
// (only id is allowed) and its (appinfo | documentation)* content model.
func (c *compiler) checkAnnotationDecl(ctx context.Context, elem *helium.Element) {
	src := c.diagSource()
	if src == "" {
		return
	}
	line := elem.Line()

	// Only the unqualified attribute id is allowed; foreign attributes are fine.
	for _, attr := range elem.Attributes() {
		if attr.Prefix() != "" {
			continue
		}
		if attr.LocalName() == "id" {
			continue
		}
		c.schemaError(ctx, schemaParserError(src, line, elemAnnotation, elemAnnotation,
			"The attribute '"+attr.LocalName()+"' is not allowed."))
	}

	// Content model (appinfo | documentation)*: reject non-whitespace character
	// content and any element child other than xs:appinfo / xs:documentation.
	// Report the content-model violation before descending into the appinfo /
	// documentation children so the diagnostics stay in document order.
	invalid := false
	for child := range helium.Children(elem) {
		switch child.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			if !xmlchar.IsAllSpace(child.Content()) {
				invalid = true
			}
		case helium.ElementNode:
			ce, ok := child.(*helium.Element)
			if !ok {
				continue
			}
			if ce.URI() == lexicon.NamespaceXSD && (ce.LocalName() == elemAppinfo || ce.LocalName() == elemDocumentation) {
				continue
			}
			invalid = true
		}
	}
	if invalid {
		c.schemaError(ctx, schemaParserError(src, line, elemAnnotation, elemAnnotation,
			"The content is not valid. Expected is (appinfo | documentation)*."))
	}

	for child := range helium.Children(elem) {
		ce, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if ce.URI() != lexicon.NamespaceXSD {
			continue
		}
		switch ce.LocalName() {
		case elemAppinfo:
			c.checkAppinfo(ctx, ce, src)
		case elemDocumentation:
			c.checkDocumentation(ctx, ce, src)
		}
	}
}

// checkAppinfo validates an xs:appinfo element. Only the unqualified attribute
// source is allowed; its content is lax and is not inspected.
func (c *compiler) checkAppinfo(ctx context.Context, elem *helium.Element, src string) {
	line := elem.Line()
	for _, attr := range elem.Attributes() {
		if attr.Prefix() != "" {
			continue
		}
		if attr.LocalName() == attrSource {
			c.checkAnnotationSource(ctx, src, line, elemAppinfo, string(attr.Content()))
			continue
		}
		c.schemaError(ctx, schemaParserError(src, line, elemAppinfo, elemAppinfo,
			"The attribute '"+attr.LocalName()+"' is not allowed."))
	}
}

// checkDocumentation validates an xs:documentation element. Only the unqualified
// attribute source and the xml:lang attribute are allowed; xml:lang must be a
// valid xs:language (empty / whitespace-only is invalid). Content is lax.
func (c *compiler) checkDocumentation(ctx context.Context, elem *helium.Element, src string) {
	line := elem.Line()
	langPresent := false
	var langValue string
	for _, attr := range elem.Attributes() {
		name := attr.LocalName()
		prefix := attr.Prefix()
		if prefix == lexicon.PrefixXML && name == lexicon.AttrLang {
			langPresent = true
			langValue = string(attr.Content())
			continue
		}
		if prefix != "" {
			continue // other namespaced attributes are allowed
		}
		if name == attrSource {
			c.checkAnnotationSource(ctx, src, line, elemDocumentation, string(attr.Content()))
			continue
		}
		c.schemaError(ctx, schemaParserError(src, line, elemDocumentation, elemDocumentation,
			"The attribute '"+name+"' is not allowed."))
	}

	if langPresent {
		collapsed := normalizeWhiteSpace(langValue, "collapse")
		if !languageRegex.MatchString(collapsed) {
			c.schemaError(ctx, schemaParserErrorAttr(src, line, elemDocumentation, elemDocumentation,
				helium.ClarkName(lexicon.NamespaceXML, lexicon.AttrLang),
				"'"+langValue+"' is not a valid value of the atomic type 'xs:language'."))
		}
	}
}

func (c *compiler) checkAnnotationSource(ctx context.Context, src string, line int, elemName, raw string) {
	collapsed := normalizeWhiteSpace(raw, "collapse")
	if err := validateBuiltinValue(collapsed, lexicon.TypeAnyURI, c.version); err != nil {
		c.schemaError(ctx, schemaParserErrorAttr(src, line, elemName, elemName, attrSource,
			"'"+collapsed+"' is not a valid value of the atomic type 'xs:anyURI'."))
	}
}
