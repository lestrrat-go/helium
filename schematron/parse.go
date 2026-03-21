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
)

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

	var eh helium.ErrorHandler = helium.NilErrorHandler{}
	if cfg != nil && cfg.errorHandler != nil {
		eh = cfg.errorHandler
	}

	// Phase-based parsing matching libxml2's xmlSchematronParse ordering:
	// title, then ns elements, then pattern elements.
	elem := nextSchematronElement(root.FirstChild())

	// Phase 1: optional title
	if elem != nil && stripPrefix(elem.Name()) == "title" {
		schema.title = elemTextContent(elem)
		elem = nextSchematronElement(elem.NextSibling())
	}

	// Phase 2: ns elements
	for elem != nil && stripPrefix(elem.Name()) == "ns" {
		addNamespace(schema.namespaces, elem)
		elem = nextSchematronElement(elem.NextSibling())
	}

	// Phase 3: pattern elements (anything else is an error)
	for elem != nil {
		if stripPrefix(elem.Name()) == "pattern" {
			if p := compilePattern(compileCtx, elem, schNS, eh); p != nil {
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

	return schema, nil
}

func compilePattern(compileCtx context.Context, elem *helium.Element, schNS string, eh helium.ErrorHandler) *pattern {
	p := &pattern{
		name: getAttr(elem, "name"),
	}
	if p.name == "" {
		p.name = getAttr(elem, "id")
	}

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		ruleElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if stripPrefix(ruleElem.Name()) != "rule" {
			eh.Handle(compileCtx, helium.NewLeveledError(fmt.Sprintf("Expecting a rule element instead of %s\n", ruleElem.Name()), helium.ErrorLevelFatal))
			continue
		}

		if r := compileRule(compileCtx, ruleElem, schNS, eh); r != nil {
			p.rules = append(p.rules, r)
		}
	}

	if len(p.rules) == 0 {
		eh.Handle(compileCtx, helium.NewLeveledError("Pattern has no rule element\n", helium.ErrorLevelFatal))
	}

	return p
}

func compileRule(compileCtx context.Context, elem *helium.Element, schNS string, eh helium.ErrorHandler) *rule {
	ctxExpr := getAttr(elem, "context")
	if ctxExpr == "" {
		eh.Handle(compileCtx, helium.NewLeveledError("rule has an empty context attribute\n", helium.ErrorLevelFatal))
		return nil
	}

	xpathExpr := contextToXPath(ctxExpr)

	compiled, err := xpath1.Compile(xpathExpr)
	if err != nil {
		eh.Handle(compileCtx, helium.NewLeveledError(fmt.Sprintf("element rule: Failed to compile context expression '%s': %s\n", ctxExpr, err), helium.ErrorLevelFatal))
		return nil
	}

	r := &rule{
		context:     ctxExpr,
		contextExpr: compiled,
		line:        elem.Line(),
	}

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		compileRuleChild(compileCtx, r, childElem, schNS, eh)
	}

	if len(r.tests) == 0 {
		eh.Handle(compileCtx, helium.NewLeveledError("rule has no assert nor report element\n", helium.ErrorLevelFatal))
	}

	return r
}

// compileRuleChild processes a single child element of a <rule>.
func compileRuleChild(compileCtx context.Context, r *rule, childElem *helium.Element, schNS string, eh helium.ErrorHandler) {
	switch stripPrefix(childElem.Name()) {
	case "let":
		lb, err := compileLet(childElem)
		if err != nil {
			eh.Handle(compileCtx, helium.NewLeveledError(fmt.Sprintf("element let: Failed to compile expression: %s\n", err), helium.ErrorLevelFatal))
			return
		}
		if lb != nil {
			// Prepend to match libxml2's LIFO ordering: each new let
			// is inserted at the head of the list, so evaluation
			// proceeds in reverse-definition order.
			r.lets = append([]*letBinding{lb}, r.lets...)
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
	name := getAttr(elem, "name")
	value := getAttr(elem, "value")
	if name == "" || value == "" {
		return nil, nil
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
	testExpr := getAttr(elem, "test")
	if testExpr == "" {
		return nil
	}

	compiled, err := xpath1.Compile(testExpr)
	if err != nil {
		eh.Handle(compileCtx, helium.NewLeveledError(fmt.Sprintf("element %s: Failed to compile test expression '%s': %s\n", testTypeName(typ), testExpr, err), helium.ErrorLevelFatal))
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

func parseMessage(compileCtx context.Context, elem *helium.Element, _ string, eh helium.ErrorHandler) []messagePart {
	var parts []messagePart

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Type() {
		case helium.TextNode:
			parts = append(parts, textPart{text: string(child.Content())})
		case helium.ElementNode:
			childElem, ok := child.(*helium.Element)
			if !ok {
				continue
			}
			parts = parseMessageElement(compileCtx, childElem, parts, eh)
		}
	}

	return parts
}

// parseMessageElement processes a single element child of a message/assert/report,
// appending the appropriate messagePart to parts and returning the updated slice.
func parseMessageElement(compileCtx context.Context, childElem *helium.Element, parts []messagePart, eh helium.ErrorHandler) []messagePart {
	switch stripPrefix(childElem.Name()) {
	case "name":
		path := getAttr(childElem, "path")
		if path == "" {
			path = "."
		}
		compiled, err := xpath1.Compile(path)
		if err != nil {
			eh.Handle(compileCtx, helium.NewLeveledError(fmt.Sprintf("element name: Failed to compile path '%s': %s\n", path, err), helium.ErrorLevelFatal))
			return append(parts, namePart{path: path})
		}
		return append(parts, namePart{path: path, expr: compiled})
	case "value-of":
		sel := getAttr(childElem, "select")
		if sel == "" {
			eh.Handle(compileCtx, helium.NewLeveledError("value-of has no select attribute\n", helium.ErrorLevelFatal))
			return parts
		}
		compiled, err := xpath1.Compile(sel)
		if err != nil {
			// XPath compile error — record in handler but still add the part
			// so validation can emit the error at runtime
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

	for i := 0; i < len(s); i++ {
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
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

// getAttr returns the value of an attribute on the element by local name.
func getAttr(elem *helium.Element, name string) string {
	attr, ok := elem.FindAttribute(helium.LocalNamePredicate(name))
	if !ok {
		return ""
	}
	return attr.Value()
}

// elemTextContent returns the concatenated text content of an element's children.
func elemTextContent(elem *helium.Element) string {
	var sb strings.Builder
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
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
	prefix := getAttr(elem, "prefix")
	uri := getAttr(elem, "uri")
	if prefix != "" && uri != "" {
		namespaces[prefix] = uri
	}
}
