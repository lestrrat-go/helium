package schematron

import (
	"errors"
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath"
)

const (
	nsISO  = "http://purl.oclc.org/dsdl/schematron"
	nsASCC = "http://www.ascc.net/xml/schematron"
)

var (
	errNoRootElement   = errors.New("schematron: no root element")
	errNotSchemaElement = errors.New("schematron: root element is not a Schematron <schema>")
)

func compileSchema(doc *helium.Document, _ *compileConfig) (*Schema, error) {
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

	var errors strings.Builder
	var warnings strings.Builder

	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		localName := stripPrefix(elem.Name())

		switch localName {
		case "title":
			schema.title = elemTextContent(elem)
		case "ns":
			addNamespace(schema.namespaces, elem)
		case "pattern":
			if p := compilePattern(elem, schNS, &errors); p != nil {
				schema.patterns = append(schema.patterns, p)
			}
		}
	}

	schema.compileErrors = errors.String()
	schema.compileWarnings = warnings.String()
	return schema, nil
}

func compilePattern(elem *helium.Element, schNS string, errors *strings.Builder) *pattern {
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
			continue
		}

		if r := compileRule(ruleElem, schNS, errors); r != nil {
			p.rules = append(p.rules, r)
		}
	}

	return p
}

func compileRule(elem *helium.Element, schNS string, errors *strings.Builder) *rule {
	context := getAttr(elem, "context")
	if context == "" {
		return nil
	}

	xpathExpr := contextToXPath(context)

	compiled, err := xpath.Compile(xpathExpr)
	if err != nil {
		fmt.Fprintf(errors, "element rule: Failed to compile context expression '%s': %s\n", context, err)
		return nil
	}

	r := &rule{
		context:     context,
		contextExpr: compiled,
		line:        elem.Line(),
	}

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		compileRuleChild(r, childElem, schNS, errors)
	}

	return r
}

// compileRuleChild processes a single child element of a <rule>.
func compileRuleChild(r *rule, childElem *helium.Element, schNS string, errors *strings.Builder) {
	switch stripPrefix(childElem.Name()) {
	case "let":
		lb, err := compileLet(childElem)
		if err != nil {
			fmt.Fprintf(errors, "element let: Failed to compile expression: %s\n", err)
			return
		}
		if lb != nil {
			r.lets = append(r.lets, lb)
		}
	case "assert":
		if t := compileTest(childElem, testAssert, schNS, errors); t != nil {
			r.tests = append(r.tests, t)
		}
	case "report":
		if t := compileTest(childElem, testReport, schNS, errors); t != nil {
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

	compiled, err := xpath.Compile(value)
	if err != nil {
		return nil, fmt.Errorf("schematron: compile let expression: %w", err)
	}

	return &letBinding{
		name: name,
		expr: compiled,
	}, nil
}

func compileTest(elem *helium.Element, typ testType, schNS string, errors *strings.Builder) *test {
	testExpr := getAttr(elem, "test")
	if testExpr == "" {
		return nil
	}

	compiled, err := xpath.Compile(testExpr)
	if err != nil {
		fmt.Fprintf(errors, "element %s: Failed to compile test expression '%s': %s\n", testTypeName(typ), testExpr, err)
		return nil
	}

	msg := parseMessage(elem, schNS, errors)

	return &test{
		typ:      typ,
		expr:     testExpr,
		compiled: compiled,
		message:  msg,
		line:     elem.Line(),
	}
}

func parseMessage(elem *helium.Element, _ string, errors *strings.Builder) []messagePart {
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
			parts = parseMessageElement(childElem, parts, errors)
		}
	}

	return parts
}

// parseMessageElement processes a single element child of a message/assert/report,
// appending the appropriate messagePart to parts and returning the updated slice.
func parseMessageElement(childElem *helium.Element, parts []messagePart, errors *strings.Builder) []messagePart {
	switch stripPrefix(childElem.Name()) {
	case "name":
		path := getAttr(childElem, "path")
		if path == "" {
			path = "."
		}
		compiled, err := xpath.Compile(path)
		if err != nil {
			fmt.Fprintf(errors, "element name: Failed to compile path '%s': %s\n", path, err)
			return append(parts, namePart{path: path})
		}
		return append(parts, namePart{path: path, expr: compiled})
	case "value-of":
		sel := getAttr(childElem, "select")
		if sel == "" {
			return parts
		}
		compiled, err := xpath.Compile(sel)
		if err != nil {
			// XPath compile error — record in errors but still add the part
			// so validation can emit the error at runtime
			return append(parts, valueOfPart{sel: sel})
		}
		return append(parts, valueOfPart{sel: sel, expr: compiled})
	}
	return parts
}

// contextToXPath converts a Schematron rule context pattern to an XPath expression.
// Absolute patterns (starting with /) are used as-is.
// Relative patterns get "//" prefixed to match anywhere in the document.
func contextToXPath(context string) string {
	trimmed := strings.TrimSpace(context)
	if strings.HasPrefix(trimmed, "/") {
		return trimmed
	}
	return "//" + trimmed
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
	for _, a := range elem.Attributes() {
		if a.LocalName() == name {
			return a.Value()
		}
	}
	return ""
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
