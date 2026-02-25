package xmlschema

import (
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

func validateDocument(doc *helium.Document, schema *Schema, cfg *validateConfig) string {
	filename := cfg.filename
	var out strings.Builder
	valid := true

	root := findDocumentElement(doc)
	if root == nil {
		return filename + " fails to validate\n"
	}

	// Walk the document tree.
	_ = helium.Walk(doc, func(n helium.Node) error {
		if n.Type() != helium.ElementNode {
			return nil
		}
		elem := n.(*helium.Element)
		if err := validateElement(elem, schema, filename, &out); err != nil {
			valid = false
		}
		return nil
	})

	if valid {
		out.WriteString(filename + " validates\n")
	} else {
		out.WriteString(filename + " fails to validate\n")
	}
	return out.String()
}

func validateElement(elem *helium.Element, schema *Schema, filename string, out *strings.Builder) error {
	parent := elem.Parent()
	if parent == nil || parent.Type() == helium.DocumentNode {
		// Root element — must match a global element declaration.
		return validateRootElement(elem, schema, filename, out)
	}
	// Non-root elements are validated by their parent's content model.
	return nil
}

func validateRootElement(elem *helium.Element, schema *Schema, filename string, out *strings.Builder) error {
	local := elem.LocalName()
	ns := elem.URI()
	edecl, ok := schema.LookupElement(local, ns)
	if !ok {
		// Try with empty namespace.
		edecl, ok = schema.LookupElement(local, "")
	}
	if !ok {
		out.WriteString(validityError(filename, elem.Line(), local, "This element is not expected."))
		return fmt.Errorf("not expected")
	}

	if edecl.Type == nil {
		return nil
	}

	return validateElementContent(elem, edecl.Type, schema, filename, out)
}

func validateElementContent(elem *helium.Element, td *TypeDef, schema *Schema, filename string, out *strings.Builder) error {
	switch td.ContentType {
	case ContentTypeEmpty:
		return validateEmptyContent(elem, filename, out)
	case ContentTypeSimple:
		return nil // Phase 1 doesn't validate simple type values.
	case ContentTypeElementOnly, ContentTypeMixed:
		if td.ContentModel == nil {
			// No content model means anything goes (for mixed) or empty (for element-only).
			if td.ContentType == ContentTypeElementOnly {
				return validateEmptyContent(elem, filename, out)
			}
			return nil
		}
		return validateContentModel(elem, td.ContentModel, schema, filename, out)
	}
	return nil
}

func validateEmptyContent(elem *helium.Element, filename string, out *strings.Builder) error {
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Type() {
		case helium.ElementNode:
			ce := child.(*helium.Element)
			out.WriteString(validityError(filename, ce.Line(), ce.LocalName(), "This element is not expected."))
			return fmt.Errorf("not expected")
		case helium.TextNode:
			if !isBlank(child.Content()) {
				out.WriteString(validityError(filename, elem.Line(), elem.LocalName(), "Character content is not allowed, because the type definition is simple."))
				return fmt.Errorf("not expected")
			}
		}
	}
	return nil
}

func validateContentModel(elem *helium.Element, mg *ModelGroup, schema *Schema, filename string, out *strings.Builder) error {
	children := collectChildElements(elem)
	return validateContentModelTop(elem, mg, children, schema, filename, out)
}

type childElem struct {
	elem *helium.Element
	name string
}

func collectChildElements(elem *helium.Element) []childElem {
	var children []childElem
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() == helium.ElementNode {
			ce := child.(*helium.Element)
			children = append(children, childElem{elem: ce, name: ce.LocalName()})
		}
	}
	return children
}

func isBlank(b []byte) bool {
	for _, c := range b {
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return false
		}
	}
	return true
}
