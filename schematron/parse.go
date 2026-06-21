package schematron

import (
	"context"
	"errors"
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath1"
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

// fatalErr delivers msg to eh as a fatal-level compilation error.
func fatalErr(ctx context.Context, eh helium.ErrorHandler, msg string) {
	eh.Handle(ctx, helium.NewLeveledError(msg, helium.ErrorLevelFatal))
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

	schema := &Schema{
		namespaces: make(map[string]string),
	}

	var inner helium.ErrorHandler = helium.NilErrorHandler{}
	if cfg != nil && cfg.errorHandler != nil {
		inner = cfg.errorHandler
	}
	eh := &fatalTrackingHandler{inner: inner}

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
			if p := compilePattern(compileCtx, elem, schNS, eh); p != nil {
				schema.patterns = append(schema.patterns, p)
			}
		} else {
			fatalErr(compileCtx, eh, fmt.Sprintf("Expecting a pattern element instead of %s\n", elem.Name()))
		}
		elem = nextSchematronElement(elem.NextSibling())
	}

	if len(schema.patterns) == 0 {
		fatalErr(compileCtx, eh, "schema has no pattern element\n")
	}

	if eh.fatal {
		return nil, ErrCompileFailed
	}

	return schema, nil
}

func compilePattern(compileCtx context.Context, elem *helium.Element, schNS string, eh helium.ErrorHandler) *pattern {
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
		if !isSchematronElement(ruleElem, schNS, "rule") {
			fatalErr(compileCtx, eh, fmt.Sprintf("Expecting a rule element instead of %s\n", ruleElem.Name()))
			continue
		}

		if r := compileRule(compileCtx, ruleElem, schNS, eh); r != nil {
			p.rules = append(p.rules, r)
		}
	}

	if len(p.rules) == 0 {
		fatalErr(compileCtx, eh, "Pattern has no rule element\n")
	}

	return p
}

func compileRule(compileCtx context.Context, elem *helium.Element, schNS string, eh helium.ErrorHandler) *rule {
	ctxExpr := getStructuralAttr(elem, "context")
	if ctxExpr == "" {
		fatalErr(compileCtx, eh, "rule has an empty context attribute\n")
		return nil
	}

	xpathExpr := contextToXPath(ctxExpr)

	compiled, err := xpath1.Compile(xpathExpr)
	if err != nil {
		fatalErr(compileCtx, eh, fmt.Sprintf("element rule: Failed to compile context expression '%s': %s\n", ctxExpr, err))
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
		compileRuleChild(compileCtx, r, childElem, schNS, eh)
	}

	if len(r.tests) == 0 {
		fatalErr(compileCtx, eh, "rule has no assert nor report element\n")
	}

	return r
}

// compileRuleChild processes a single child element of a <rule>.
func compileRuleChild(compileCtx context.Context, r *rule, childElem *helium.Element, schNS string, eh helium.ErrorHandler) {
	// Only Schematron-namespaced children carry structural meaning; foreign
	// elements (e.g. <x:assert>) are ignored rather than executed.
	if !elementInNamespace(childElem, schNS) {
		return
	}
	switch stripPrefix(childElem.Name()) {
	case "let":
		lb, err := compileLet(childElem)
		if err != nil {
			fatalErr(compileCtx, eh, fmt.Sprintf("element let: Failed to compile expression: %s\n", err))
			return
		}
		if lb != nil {
			// Append in document order so each let is bound before any
			// later let that references it (e.g. <let name="b"
			// value="$a"/> after <let name="a" .../>).
			r.lets = append(r.lets, lb)
		}
	case "assert":
		if t := compileTest(compileCtx, childElem, testAssert, schNS, eh); t != nil {
			r.tests = append(r.tests, t)
		}
	case "report":
		if t := compileTest(compileCtx, childElem, testReport, schNS, eh); t != nil {
			r.tests = append(r.tests, t)
		}
	}
}

func compileLet(elem *helium.Element) (*letBinding, error) {
	name := getStructuralAttr(elem, "name")
	value := getStructuralAttr(elem, "value")
	if name == "" || value == "" {
		return nil, nil //nolint:nilnil
	}

	compiled, err := xpath1.Compile(value)
	if err != nil {
		return nil, fmt.Errorf("schematron: compile let expression: %w", err)
	}

	return &letBinding{
		name: name,
		expr: compiled,
	}, nil
}

func compileTest(compileCtx context.Context, elem *helium.Element, typ testType, schNS string, eh helium.ErrorHandler) *test {
	testExpr := getStructuralAttr(elem, "test")
	if testExpr == "" {
		return nil
	}

	compiled, err := xpath1.Compile(testExpr)
	if err != nil {
		fatalErr(compileCtx, eh, fmt.Sprintf("element %s: Failed to compile test expression '%s': %s\n", testTypeName(typ), testExpr, err))
		return nil
	}

	msg := parseMessage(compileCtx, elem, schNS, eh)

	return &test{
		typ:      typ,
		expr:     testExpr,
		compiled: compiled,
		message:  msg,
		line:     elem.Line(),
	}
}

func parseMessage(compileCtx context.Context, elem *helium.Element, schNS string, eh helium.ErrorHandler) []messagePart {
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
			parts = parseMessageElement(compileCtx, childElem, schNS, parts, eh)
		}
	}

	return parts
}

// parseMessageElement processes a single element child of a message/assert/report,
// appending the appropriate messagePart to parts and returning the updated slice.
func parseMessageElement(compileCtx context.Context, childElem *helium.Element, schNS string, parts []messagePart, eh helium.ErrorHandler) []messagePart {
	// Only Schematron-namespaced <name>/<value-of> carry structural meaning;
	// foreign elements contribute nothing to the message.
	if !elementInNamespace(childElem, schNS) {
		return parts
	}
	switch stripPrefix(childElem.Name()) {
	case "name":
		path := getStructuralAttr(childElem, "path")
		if path == "" {
			path = "."
		}
		compiled, err := xpath1.Compile(path)
		if err != nil {
			fatalErr(compileCtx, eh, fmt.Sprintf("element name: Failed to compile path '%s': %s\n", path, err))
			return append(parts, namePart{path: path})
		}
		return append(parts, namePart{path: path, expr: compiled})
	case "value-of":
		sel := getStructuralAttr(childElem, "select")
		if sel == "" {
			fatalErr(compileCtx, eh, "value-of has no select attribute\n")
			return parts
		}
		compiled, err := xpath1.Compile(sel)
		if err != nil {
			// Report the compile error through the handler (mirroring the
			// <name path="..."> case and compileTest), then still add the
			// part so the message structure is preserved.
			fatalErr(compileCtx, eh, fmt.Sprintf("element value-of: Failed to compile select expression '%s': %s\n", sel, err))
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
