package schematron

import (
	"context"
	"errors"
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/xpath1/lexer"
)

const (
	nsISO  = "http://purl.oclc.org/dsdl/schematron"
	nsASCC = "http://www.ascc.net/xml/schematron"
)

var (
	errNoRootElement    = errors.New("schematron: no root element")
	errNotSchemaElement = errors.New("schematron: root element is not a Schematron <schema>")

	// ErrCompileFailed is returned by [Compiler.Compile] and
	// [Compiler.CompileFile] when the Schematron document contains one or
	// more fatal compilation errors (for example, a schema with no valid
	// pattern, or a malformed rule). When this error is returned the
	// resulting *Schema is nil. Individual errors are also delivered to the
	// configured [helium.ErrorHandler], if any.
	ErrCompileFailed = errors.New("schematron: compilation failed")
)

// fatalTrackingHandler wraps an ErrorHandler and records whether any
// fatal-level error has passed through it. This lets compileSchema fail the
// overall Compile when fatal errors occur, regardless of whether the caller
// supplied an error handler.
type fatalTrackingHandler struct {
	inner helium.ErrorHandler
	fatal bool
}

func (h *fatalTrackingHandler) Handle(ctx context.Context, err error) {
	if l, ok := err.(helium.ErrorLeveler); ok && l.ErrorLevel() == helium.ErrorLevelFatal {
		h.fatal = true
	}
	if h.inner != nil {
		h.inner.Handle(ctx, err)
	}
}

// schemaParser holds the state every compilation step needs: the Schematron
// namespace the document uses, the fatal-tracking error handler, and the
// engine that compiles expressions in the schema's query language binding.
type schemaParser struct {
	schNS  string
	eh     *fatalTrackingHandler
	engine engine
}

func compileSchema(compileCtx context.Context, doc *helium.Document, cfg *compileConfig) (*Schema, error) {
	root := findDocumentElement(doc)
	if root == nil {
		return nil, errNoRootElement
	}

	schNS := detectNamespace(root)
	if schNS == "" {
		return nil, errNotSchemaElement
	}

	var inner helium.ErrorHandler = helium.NilErrorHandler{}
	if cfg != nil && cfg.errorHandler != nil {
		inner = cfg.errorHandler
	}
	eh := &fatalTrackingHandler{inner: inner}

	binding, err := resolveQueryBinding(cfg, root)
	if err != nil {
		// An unimplemented query language binding is fatal: evaluating the
		// schema's expressions as if they were written in a binding we do
		// support would report results the schema never asked for.
		eh.Handle(compileCtx, helium.NewLeveledError(fmt.Sprintf("%s\n", err), helium.ErrorLevelFatal))
		return nil, err
	}
	eng := binding.newEngine()

	schema := &Schema{
		namespaces: make(map[string]string),
		binding:    binding,
		engine:     eng,
	}
	sp := &schemaParser{schNS: schNS, eh: eh, engine: eng}

	// Phase-based parsing matching libxml2's xmlSchematronParse ordering:
	// title, then ns elements, then pattern elements.
	elem := nextSchematronElement(root.FirstChild())

	// Phase 1: optional title
	if elem != nil && isSchematronElement(elem, schNS, "title") {
		schema.title = elemTextContent(elem)
		elem = nextSchematronElement(elem.NextSibling())
	}

	// Phase 2: ns elements
	for elem != nil && isSchematronElement(elem, schNS, "ns") {
		addNamespace(schema.namespaces, elem)
		elem = nextSchematronElement(elem.NextSibling())
	}

	// Phase 3: pattern elements (anything else is an error)
	for elem != nil {
		if isSchematronElement(elem, schNS, "pattern") {
			if p := sp.compilePattern(compileCtx, elem); p != nil {
				schema.patterns = append(schema.patterns, p)
			}
		} else {
			eh.Handle(compileCtx, helium.NewLeveledError(fmt.Sprintf("Expecting a pattern element instead of %s\n", elem.Name()), helium.ErrorLevelFatal))
		}
		elem = nextSchematronElement(elem.NextSibling())
	}

	if len(schema.patterns) == 0 {
		eh.Handle(compileCtx, helium.NewLeveledError("schema has no pattern element\n", helium.ErrorLevelFatal))
	}

	if eh.fatal {
		return nil, ErrCompileFailed
	}

	return schema, nil
}

func (sp *schemaParser) compilePattern(compileCtx context.Context, elem *helium.Element) *pattern {
	p := &pattern{
		name: getStructuralAttr(elem, "name"),
	}
	if p.name == "" {
		p.name = getStructuralAttr(elem, "id")
	}

	for child := range helium.Children(elem) {
		ruleElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if !isSchematronElement(ruleElem, sp.schNS, "rule") {
			sp.eh.Handle(compileCtx, helium.NewLeveledError(fmt.Sprintf("Expecting a rule element instead of %s\n", ruleElem.Name()), helium.ErrorLevelFatal))
			continue
		}

		if r := sp.compileRule(compileCtx, ruleElem); r != nil {
			p.rules = append(p.rules, r)
		}
	}

	if len(p.rules) == 0 {
		sp.eh.Handle(compileCtx, helium.NewLeveledError("Pattern has no rule element\n", helium.ErrorLevelFatal))
	}

	return p
}

func (sp *schemaParser) compileRule(compileCtx context.Context, elem *helium.Element) *rule {
	ctxExpr := getStructuralAttr(elem, "context")
	if ctxExpr == "" {
		sp.eh.Handle(compileCtx, helium.NewLeveledError("rule has an empty context attribute\n", helium.ErrorLevelFatal))
		return nil
	}

	xpathExpr := contextToXPath(ctxExpr)

	compiled, err := sp.engine.compile(xpathExpr)
	if err != nil {
		sp.eh.Handle(compileCtx, helium.NewLeveledError(fmt.Sprintf("element rule: Failed to compile context expression '%s': %s\n",
			lexer.DiagnosticExcerpt(ctxExpr), err), helium.ErrorLevelFatal))
		return nil
	}

	r := &rule{
		context:     ctxExpr,
		contextExpr: compiled,
		line:        elem.Line(),
	}

	for child := range helium.Children(elem) {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		sp.compileRuleChild(compileCtx, r, childElem)
	}

	if len(r.tests) == 0 {
		sp.eh.Handle(compileCtx, helium.NewLeveledError("rule has no assert nor report element\n", helium.ErrorLevelFatal))
	}

	return r
}

// compileRuleChild processes a single child element of a <rule>.
func (sp *schemaParser) compileRuleChild(compileCtx context.Context, r *rule, childElem *helium.Element) {
	// Only Schematron-namespaced children carry structural meaning; foreign
	// elements (e.g. <x:assert>) are ignored and never executed.
	if !elementInNamespace(childElem, sp.schNS) {
		return
	}
	switch stripPrefix(childElem.Name()) {
	case "let":
		lb, err := sp.compileLet(childElem)
		if err != nil {
			sp.eh.Handle(compileCtx, helium.NewLeveledError(fmt.Sprintf("element let: Failed to compile expression: %s\n", err), helium.ErrorLevelFatal))
			return
		}
		if lb != nil {
			// Append in document order so each let is bound before any
			// later let that references it (e.g. <let name="b"
			// value="$a"/> after <let name="a" .../>).
			r.lets = append(r.lets, lb)
		}
	case "assert":
		if t := sp.compileTest(compileCtx, childElem, testAssert); t != nil {
			r.tests = append(r.tests, t)
		}
	case "report":
		if t := sp.compileTest(compileCtx, childElem, testReport); t != nil {
			r.tests = append(r.tests, t)
		}
	}
}

func (sp *schemaParser) compileLet(elem *helium.Element) (*letBinding, error) {
	name := getStructuralAttr(elem, "name")
	value := getStructuralAttr(elem, "value")
	if name == "" || value == "" {
		return nil, nil //nolint:nilnil
	}

	compiled, err := sp.engine.compile(value)
	if err != nil {
		return nil, fmt.Errorf("schematron: compile let expression: %w", err)
	}

	return &letBinding{
		name: name,
		expr: compiled,
	}, nil
}

func (sp *schemaParser) compileTest(compileCtx context.Context, elem *helium.Element, typ testType) *test {
	testExpr := getStructuralAttr(elem, "test")
	if testExpr == "" {
		return nil
	}

	compiled, err := sp.engine.compile(testExpr)
	if err != nil {
		sp.eh.Handle(compileCtx, helium.NewLeveledError(fmt.Sprintf("element %s: Failed to compile test expression '%s': %s\n",
			testTypeName(typ), lexer.DiagnosticExcerpt(testExpr), err), helium.ErrorLevelFatal))
		return nil
	}

	msg := sp.parseMessage(compileCtx, elem)

	return &test{
		typ:      typ,
		expr:     testExpr,
		compiled: compiled,
		message:  msg,
		line:     elem.Line(),
	}
}

func (sp *schemaParser) parseMessage(compileCtx context.Context, elem *helium.Element) []messagePart {
	var parts []messagePart

	for child := range helium.Children(elem) {
		switch child.Type() {
		case helium.TextNode:
			parts = append(parts, textPart{text: string(child.Content())})
		case helium.ElementNode:
			childElem, ok := child.(*helium.Element)
			if !ok {
				continue
			}
			parts = sp.parseMessageElement(compileCtx, childElem, parts)
		}
	}

	return parts
}

// parseMessageElement processes a single element child of a message/assert/report,
// appending the appropriate messagePart to parts and returning the updated slice.
func (sp *schemaParser) parseMessageElement(compileCtx context.Context, childElem *helium.Element, parts []messagePart) []messagePart {
	// Only Schematron-namespaced <name>/<value-of> carry structural meaning;
	// foreign elements contribute nothing to the message.
	if !elementInNamespace(childElem, sp.schNS) {
		return parts
	}
	switch stripPrefix(childElem.Name()) {
	case "name":
		path := getStructuralAttr(childElem, "path")
		if path == "" {
			path = "."
		}
		compiled, err := sp.engine.compile(path)
		if err != nil {
			sp.eh.Handle(compileCtx, helium.NewLeveledError(fmt.Sprintf("element name: Failed to compile path '%s': %s\n",
				lexer.DiagnosticExcerpt(path), err), helium.ErrorLevelFatal))
			return append(parts, namePart{path: path})
		}
		return append(parts, namePart{path: path, expr: compiled})
	case "value-of":
		sel := getStructuralAttr(childElem, "select")
		if sel == "" {
			sp.eh.Handle(compileCtx, helium.NewLeveledError("value-of has no select attribute\n", helium.ErrorLevelFatal))
			return parts
		}
		compiled, err := sp.engine.compile(sel)
		if err != nil {
			// Report the compile error through the handler (mirroring the
			// <name path="..."> case and compileTest), then still add the
			// part so the message structure is preserved.
			sp.eh.Handle(compileCtx, helium.NewLeveledError(fmt.Sprintf("element value-of: Failed to compile select expression '%s': %s\n",
				lexer.DiagnosticExcerpt(sel), err), helium.ErrorLevelFatal))
			return append(parts, valueOfPart{sel: sel})
		}
		return append(parts, valueOfPart{sel: sel, expr: compiled})
	}
	return parts
}

// contextToXPath converts a Schematron rule context pattern to an XPath expression.
// For union patterns (e.g. "a | b"), each alternative is processed independently:
// relative parts get "//" prefixed, absolute parts are kept as-is.
func contextToXPath(context string) string {
	parts := splitTopLevelUnion(context)
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if !strings.HasPrefix(p, "/") {
			p = "//" + p
		}
		parts[i] = p
	}
	return strings.Join(parts, " | ")
}

// splitTopLevelUnion splits s on "|" characters that are not inside
// brackets, parentheses, or string literals.
func splitTopLevelUnion(s string) []string {
	var parts []string
	depth := 0     // tracks [] and () nesting
	var quote byte // tracks ' or " literal state
	start := 0

	for i := range len(s) {
		ch := s[i]
		switch {
		case quote != 0:
			if ch == quote {
				quote = 0
			}
		case ch == '\'' || ch == '"':
			quote = ch
		case ch == '[' || ch == '(':
			depth++
		case ch == ']' || ch == ')':
			if depth > 0 {
				depth--
			}
		case ch == '|' && depth == 0:
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func testTypeName(typ testType) string {
	if typ == testAssert {
		return "assert"
	}
	return "report"
}

// detectNamespace checks if root element is <schema> in a recognized Schematron namespace.
func detectNamespace(root *helium.Element) string {
	name := stripPrefix(root.Name())
	if name != "schema" {
		return ""
	}

	ns := root.Namespace()
	if ns != nil {
		uri := ns.URI()
		switch uri {
		case nsISO, nsASCC:
			return uri
		}
	}

	// Check for default namespace via xmlns attribute on the element.
	for _, nsDef := range root.Namespaces() {
		if nsDef.Prefix() == "" {
			uri := nsDef.URI()
			switch uri {
			case nsISO, nsASCC:
				return uri
			}
		}
	}

	return ""
}

// stripPrefix removes any namespace prefix from a name (e.g. "sch:schema" -> "schema").
func stripPrefix(name string) string {
	if _, after, ok := strings.Cut(name, ":"); ok {
		return after
	}
	return name
}

// getStructuralAttr returns the value of an unqualified (no namespace)
// attribute on the element. Schematron structural attributes such as
// context/test/select are defined to have no namespace; a prefixed attribute
// like x:test belongs to a foreign vocabulary and must not be read as
// Schematron.
func getStructuralAttr(elem *helium.Element, name string) string {
	attr, ok := elem.FindAttribute(helium.NSPredicate{Local: name, NamespaceURI: ""})
	if !ok {
		return ""
	}
	return attr.Value()
}

// elementInNamespace reports whether elem belongs to the given Schematron
// namespace. Foreign-namespaced elements (e.g. <x:rule>) must not be treated
// as Schematron constructs.
func elementInNamespace(elem *helium.Element, schNS string) bool {
	ns := elem.Namespace()
	if ns == nil {
		return false
	}
	return ns.URI() == schNS
}

// isSchematronElement reports whether elem is the named Schematron element in
// the detected Schematron namespace.
func isSchematronElement(elem *helium.Element, schNS, localName string) bool {
	return elementInNamespace(elem, schNS) && elem.LocalName() == localName
}

// elemTextContent returns the concatenated text content of an element's children.
func elemTextContent(elem *helium.Element) string {
	var sb strings.Builder
	for child := range helium.Children(elem) {
		if child.Type() == helium.TextNode {
			sb.Write(child.Content())
		}
	}
	return sb.String()
}

// nextSchematronElement advances from the given node to the next sibling
// that is an *helium.Element, skipping text, comments, and PIs.
// This mirrors libxml2's NEXT_SCHEMATRON macro.
func nextSchematronElement(n helium.Node) *helium.Element {
	for n != nil {
		if elem, ok := n.(*helium.Element); ok {
			return elem
		}
		n = n.NextSibling()
	}
	return nil
}

func findDocumentElement(doc *helium.Document) *helium.Element {
	return doc.DocumentElement()
}

// addNamespace registers a namespace binding from a <ns> element if
// both the prefix and uri attributes are non-empty.
func addNamespace(namespaces map[string]string, elem *helium.Element) {
	prefix := getStructuralAttr(elem, "prefix")
	uri := getStructuralAttr(elem, "uri")
	if prefix != "" && uri != "" {
		namespaces[prefix] = uri
	}
}
