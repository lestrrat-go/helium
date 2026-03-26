package xsd

import (
	"context"
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

// Validate validates a lexical value against this simple type definition.
// nsMap provides prefix-to-URI mappings for QName/NOTATION resolution and may be nil.
func (td *TypeDef) Validate(value string, nsMap map[string]string) error {
	if td == nil {
		return fmt.Errorf("nil type definition")
	}
	if td.ContentType != ContentTypeSimple {
		return fmt.Errorf("type %q is not a simple type", typeQualifiedName(td))
	}
	vc := &validationContext{
		ctx:          context.Background(),
		errorHandler: helium.NilErrorHandler{},
	}
	return validateValue(value, nsMap, td, "", "", 0, vc)
}

// ValidateElement validates an element's content against this type definition.
// This is used by XSLT xsl:type validation where the element is constructed
// in the result tree and must conform to the given type.
func (td *TypeDef) ValidateElement(elem *helium.Element, schema *Schema) error {
	if td == nil {
		return fmt.Errorf("nil type definition")
	}
	collector := &validationErrors{}
	vc := newValidationContext(context.Background(), schema, &validateConfig{}, "", collector)
	err := vc.validateElementContent(elem, nil, td)
	if err == nil {
		return nil
	}
	if len(collector.errors) > 0 {
		var b strings.Builder
		for _, e := range collector.errors {
			b.WriteString(e)
		}
		return fmt.Errorf("%s", strings.TrimSpace(b.String()))
	}
	return err
}
