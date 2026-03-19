package xsd

import (
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
)

// ValidateSimpleValue validates a lexical value against a compiled simple type.
func ValidateSimpleValue(value string, td *TypeDef) error {
	if td == nil {
		return fmt.Errorf("nil type definition")
	}
	if td.ContentType != ContentTypeSimple {
		return fmt.Errorf("type %q is not a simple type", typeQualifiedName(td))
	}
	var out strings.Builder
	return validateValue(value, td, "", "", 0, &out)
}

// ValidateElementAgainstType validates an element's content against a compiled
// type definition. This is used by XSLT xsl:type validation where the element
// is constructed in the result tree and must conform to the given type.
func ValidateElementAgainstType(elem *helium.Element, td *TypeDef, schema *Schema) error {
	if td == nil {
		return fmt.Errorf("nil type definition")
	}
	cfg := &validateConfig{}
	var out strings.Builder
	err := validateElementContent(elem, nil, td, schema, cfg, "", &out)
	if err != nil {
		msg := out.String()
		if msg != "" {
			return fmt.Errorf("%s", strings.TrimSpace(msg))
		}
		return err
	}
	return nil
}
