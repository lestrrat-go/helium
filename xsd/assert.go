package xsd

import (
	"context"
	"fmt"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// assertionNamespaces captures the in-scope namespace bindings for an XPath in an
// xs:assert/xs:assertion, then sets the default element namespace per
// xpathDefaultNamespace. The default element namespace for an XSD XPath is
// governed SOLELY by xpathDefaultNamespace, not by an xmlns="..." default in the
// schema document, so the bare xmlns default (key "") is removed unless
// xpathDefaultNamespace reinstates one. The effective default is resolved by the
// SHARED resolveXPathDefaultNS helper (read_elements.go) used by the idc/CTA paths
// too: it honors a locally-present @xpathDefaultNamespace (collapsed and resolved
// via resolveXPathDefaultNSToken) and otherwise inherits the schema-level URI
// (c.schemaXPathDefaultNS), already resolved against the schema root.
func (c *compiler) assertionNamespaces(elem *helium.Element) map[string]string {
	ns := collectNSContext(elem)
	delete(ns, "")
	if def := c.resolveXPathDefaultNS(elem); def != "" {
		ns[""] = def
	}
	return ns
}

// parseAssert reads an XSD 1.1 <xs:assert> element, capturing its @test
// expression and the in-scope namespace bindings, and pre-compiling the test as
// an XPath 3.1 expression. A missing @test or a malformed XPath is a fatal
// schema error (it returns nil), mirroring how a malformed identity-constraint
// XPath is treated — silently dropping the assertion would let an invalid schema
// validate documents as if the constraint were absent.
//
// Callers must only invoke this in XSD 1.1 mode; xs:assert is not part of the
// 1.0 grammar.
func (c *compiler) parseAssert(ctx context.Context, elem *helium.Element) *Assertion {
	return c.parseAssertion(ctx, elem, elemAssert)
}

// parseAssertion is the shared reader behind parseAssert (xs:assert on complex
// types) and the xs:assertion simple-type facet. elemKind selects the element
// name used in diagnostics.
func (c *compiler) parseAssertion(ctx context.Context, elem *helium.Element, elemKind string) *Assertion {
	if !hasAttr(elem, attrTest) {
		if c.filename != "" {
			c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elemKind,
				"The attribute 'test' is required but missing."))
		}
		return nil
	}
	test := getAttr(elem, attrTest)
	a := &Assertion{
		Test:       test,
		Namespaces: c.assertionNamespaces(elem),
		Line:       elem.Line(),
		Source:     c.diagSource(),
	}
	compiled, err := xpath3.NewCompiler().Compile(test)
	if err != nil {
		if c.filename != "" {
			c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elemKind,
				fmt.Sprintf("The XPath expression '%s' of the assertion is not valid: %v.", test, err)))
		}
		return nil
	}
	if err := compiled.Validate(a.Namespaces); err != nil {
		if c.filename != "" {
			c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elemKind,
				fmt.Sprintf("The XPath expression '%s' of the assertion has an unbound namespace prefix: %v.", test, err)))
		}
		return nil
	}
	a.compiled = compiled
	return a
}

// checkAssertions evaluates the XSD 1.1 xs:assert constraints that apply to an
// element against its (already content-validated) instance, walking the type's
// base chain so an assertion inherited from a base type is enforced too. Each
// test is evaluated with the element as the context node; the element is invalid
// unless the test's effective boolean value is true.
//
// The in-scope namespaces captured at the xs:assert element are supplied to the
// evaluator; the standard prefixes (xs, fn, math, …) keep their default bindings
// (StrictPrefixes is intentionally NOT set) so common assertion idioms such as
// xs:integer(...) or string-length(...) work without redeclaration.
//
// When the element's type has SIMPLE content (a simpleContent complex type, or a
// simple type used directly), $value is bound to the typed atomic value of the
// content per XSD 1.1 §3.13.4 (a sequence for a list type). For complex content
// $value is the empty sequence.
//
// The test is evaluated against an isolated copy rooted at the assessed element
// and stripped of comment/PI nodes, so it cannot navigate to ancestors/siblings.
func (vc *validationContext) checkAssertions(ctx context.Context, elem *helium.Element, edecl *ElementDecl, td *TypeDef) error {
	if !typeHasAssertions(td) {
		return nil
	}
	valueSeq := vc.assertValueSequence(ctx, elem, edecl, td)
	// XSD 1.1 §3.13.4.2: an xs:assert test is evaluated against a tree whose root
	// is the element being assessed, isolated from the rest of the document and
	// stripped of comment/PI nodes. Build that isolated tree once (carrying the
	// PSVI type annotations onto the copy) and evaluate every assertion against it
	// so an expression cannot navigate to ancestors/siblings.
	root, annotations := vc.isolatedAssertTree(ctx, elem)
	decls := vc.assertSchemaDecls()
	var firstErr error
	for cur := range baseChain(td) {
		for _, a := range cur.Assertions {
			if a.compiled == nil {
				continue
			}
			ev := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
				Namespaces(a.Namespaces).
				Variables(map[string]xpath3.Sequence{"value": valueSeq}).
				QNameValueNoDefaultNamespace()
			if annotations != nil {
				ev = ev.TypeAnnotations(annotations)
			}
			if decls != nil {
				ev = ev.SchemaDeclarations(decls)
			}
			res, err := ev.Evaluate(ctx, a.compiled, root)
			if err != nil {
				vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem),
					fmt.Sprintf("Failed to evaluate the assertion '%s': %v.", a.Test, err))
				if firstErr == nil {
					firstErr = fmt.Errorf("assertion evaluation failed")
				}
				continue
			}
			ok, err := xpath3.EBV(res.Sequence())
			if err != nil {
				vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem),
					fmt.Sprintf("Failed to evaluate the assertion '%s': %v.", a.Test, err))
				if firstErr == nil {
					firstErr = fmt.Errorf("assertion evaluation failed")
				}
				continue
			}
			if !ok {
				vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem),
					fmt.Sprintf("The assertion '%s' is not satisfied.", a.Test))
				if firstErr == nil {
					firstErr = fmt.Errorf("assertion not satisfied")
				}
			}
		}
	}
	return firstErr
}

// typeHasAssertions reports whether td or any type up its base chain declares an
// xs:assert constraint.
func typeHasAssertions(td *TypeDef) bool {
	for cur := range baseChain(td) {
		if len(cur.Assertions) > 0 {
			return true
		}
	}
	return false
}

// isolatedAssertTree builds the isolated XDM tree an xs:assert is evaluated
// against: a deep copy of elem rooted in a fresh document, with comment and
// processing-instruction nodes removed (XSD 1.1 §3.13.4.2). The returned
// annotation map carries the PSVI type annotations from the live tree onto the
// corresponding copied element/attribute nodes so typed atomization (e.g. a
// typed attribute in a value comparison) still works. If the copy fails for any
// reason it falls back to the live element and annotations (the documented
// non-isolated behavior) rather than skipping the assertion.
func (vc *validationContext) isolatedAssertTree(ctx context.Context, elem *helium.Element) (helium.Node, map[helium.Node]string) {
	live := map[helium.Node]string(vc.assertAnnotations)
	doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
	copied, err := helium.CopyNode(elem, doc)
	if err != nil {
		return elem, live
	}
	ce, ok := helium.AsNode[*helium.Element](copied)
	if !ok {
		return elem, live
	}
	// The copy is left parentless (NOT linked as a document child): per XSD 1.1
	// the element is the ROOT of the assertion tree, so an absolute-path
	// expression ("/" or "//"), whose root must be a document node, raises
	// XPDY0050 — the assertion cannot navigate outside the element's subtree.

	// Materialize the original element's in-scope namespaces onto the copy root.
	// CopyNode re-declares only an element's own (and active) namespaces, so a
	// prefix declared on a now-excluded ancestor would otherwise be unbound on the
	// copy — breaking node-scope resolution of a QName-valued attribute/content
	// (e.g. @name = "xsd:element"). The inherited DEFAULT namespace (prefix "") is
	// re-declared too (when in scope and not already on the copy), so
	// namespace-uri-for-prefix('', .) and unprefixed name resolution survive
	// isolation.
	existing := make(map[string]bool)
	for _, ns := range ce.Namespaces() {
		existing[ns.Prefix()] = true
	}
	for prefix, uri := range collectNSContext(elem) {
		if existing[prefix] {
			continue
		}
		if prefix == "" && uri == "" {
			continue // no default namespace in scope; nothing to re-declare
		}
		// Only prefixes not already declared on (or actively used by) the copy reach
		// here, drawn from the well-formed source's in-scope context, so the prefix
		// conflict is unreachable and NewNamespace never returns nil.
		_ = ce.AddNamespaceDecl(helium.NewNamespace(prefix, uri))
	}

	var ann map[helium.Node]string
	if vc.assertAnnotations != nil {
		ann = make(map[helium.Node]string, len(vc.assertAnnotations))
		vc.mapAssertAnnotations(ctx, elem, ce, vc.assertAnnotations, ann, true)
	}
	stripCommentsAndPIs(ce)
	return ce, ann
}

// mapAssertAnnotations walks the live element orig and its structurally identical
// copy in parallel (helium.CopyNode preserves child node types and order), copying
// each element's and attribute's type annotation from src (keyed by live node) into
// dst (keyed by copied node). It runs BEFORE comment/PI stripping so the two trees
// still align node-for-node.
//
// The assertion-tree ROOT element is deliberately left UNannotated (isRoot): an
// xs:assert is part of determining the element's own validity, so its type is not
// yet assigned during evaluation — its typed value (data(.)) is untyped — while
// its attributes and all descendant elements (validated earlier) keep their PSVI
// types. This matches the XSD 1.1 conformance tests (and Saxon).
func (vc *validationContext) mapAssertAnnotations(ctx context.Context, orig, copied *helium.Element, src TypeAnnotations, dst map[helium.Node]string, isRoot bool) {
	if name, ok := src[orig]; ok && !isRoot {
		dst[copied] = name
	}
	// A DEFAULTED/FIXED empty DESCENDANT element: materialize its effective value
	// onto the copy so data(c) atomizes the default, not "". The asserted ROOT's own
	// value is handled by $value and its data(.) is intentionally untyped, so skip it.
	if !isRoot {
		vc.materializeAssertDefault(ctx, orig, copied)
	}
	// Match attributes by expanded QName rather than positional index: the copy
	// may carry a different attribute ordering or extra namespace declarations.
	ca := copied.Attributes()
	for _, oattr := range orig.Attributes() {
		name, ok := src[oattr]
		if !ok {
			continue
		}
		for _, cattr := range ca {
			if cattr.LocalName() == oattr.LocalName() && cattr.URI() == oattr.URI() {
				dst[cattr] = name
				break
			}
		}
	}
	oc := childNodes(orig)
	cc := childNodes(copied)
	for i := range oc {
		if i >= len(cc) {
			break
		}
		oe, ok1 := helium.AsNode[*helium.Element](oc[i])
		ce, ok2 := helium.AsNode[*helium.Element](cc[i])
		if ok1 && ok2 {
			vc.mapAssertAnnotations(ctx, oe, ce, src, dst, false)
		}
	}
}

// materializeAssertDefault inserts the recorded schema default/fixed effective value
// of an EMPTY descendant element (assertEffectiveValues, keyed by the live node)
// into its isolated copy, so an ancestor xs:assert's data(thisElement) atomizes the
// schema-normalized default instead of "". A QName/NOTATION value's prefix is
// declared on the copy (from the DECLARATION's namespace context) so
// resolveQNameFromNode resolves it. Only an empty copy is materialized.
func (vc *validationContext) materializeAssertDefault(ctx context.Context, orig, copied *helium.Element) {
	if vc.assertEffectiveValues == nil {
		return
	}
	ev, ok := vc.assertEffectiveValues[orig]
	if !ok || ev.value == "" {
		return
	}
	if elemTextContent(copied) != "" {
		return // copy already carries content; do not overwrite
	}
	_ = copied.AppendText([]byte(vc.materializeAssertText(ctx, copied, ev)))
}

// materializeAssertText returns the text to append for a defaulted descendant in the
// isolated assert tree, handling QName/NOTATION values exactly like
// materializeQNameAttrValue: prefixes bound in the DECLARATION context are declared
// on the copy so atomic values and every list token resolve to their intended URI;
// collisions mint a fresh prefix and rewrite only the affected token(s). A non-QName
// value (or a value with no prefix / an unbound prefix) is returned verbatim.
func (vc *validationContext) materializeAssertText(ctx context.Context, copied *helium.Element, ev assertEffectiveValue) string {
	if !ev.qname || ev.ns == nil {
		return ev.value
	}
	if materialized, ok := vc.materializeQNameValue(ctx, copied, ev.td, ev.value, ev.ns); ok {
		return materialized
	}
	return ev.value
}

// stripCommentsAndPIs removes every comment and processing-instruction node from
// elem's subtree, so an assertion XPath (e.g. .//comment()) sees the XSD 1.1
// assertion data model, which excludes them.
func stripCommentsAndPIs(elem *helium.Element) {
	var doomed []helium.Node
	var collect func(n *helium.Element)
	collect = func(n *helium.Element) {
		for _, child := range childNodes(n) {
			switch child.Type() {
			case helium.CommentNode, helium.ProcessingInstructionNode:
				doomed = append(doomed, child)
			case helium.ElementNode:
				if ce, ok := helium.AsNode[*helium.Element](child); ok {
					collect(ce)
				}
			}
		}
	}
	collect(elem)
	for _, n := range doomed {
		if mn, ok := n.(helium.MutableNode); ok {
			helium.UnlinkNode(mn)
		}
	}
}

// childNodes materializes a node's children into a slice (a stable snapshot so a
// later structural edit — comment stripping — does not disturb iteration).
func childNodes(n helium.Node) []helium.Node {
	var out []helium.Node
	for child := range helium.Children(n) {
		out = append(out, child)
	}
	return out
}

// assertValueSequence builds the $value binding for an xs:assert on elem's type:
// the typed simple value when td has simple content, otherwise the empty
// sequence (complex content). For an EMPTY element the declaration's fixed/default
// effective value is substituted first (mirroring validateSimpleContent), so
// $value reflects the schema-normalized value rather than the raw empty text. The
// value is then whitespace-normalized per the content type's effective whiteSpace
// facet before typing.
func (vc *validationContext) assertValueSequence(ctx context.Context, elem *helium.Element, edecl *ElementDecl, td *TypeDef) xpath3.Sequence {
	if td == nil || td.ContentType != ContentTypeSimple {
		return xpath3.EmptySequence()
	}
	// $value carries the effective content simple type composed across the whole
	// simpleContent chain (a narrowing inherited through derived types), matching
	// what validateSimpleContent validates against.
	valueType := effectiveContentSimpleType(td)
	value := elemTextContent(elem)
	isEmpty := value == ""
	if isEmpty && edecl != nil {
		if edecl.Fixed != nil {
			value = *edecl.Fixed
		} else if edecl.Default != nil {
			value = *edecl.Default
		}
	}
	// A QName/NOTATION value substituted from the declaration's fixed/default
	// resolves its prefix against the DECLARATION's namespace context, so $value
	// binds the schema-intended URI rather than an unrelated instance binding.
	valueNS := effectiveValueNS(elem, edecl, isEmpty)
	raw := normalizeWhiteSpace(value, resolveWhiteSpace(valueType))
	return buildValueSequence(ctx, raw, valueNS, valueType, vc)
}
