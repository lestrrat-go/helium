package xsd

import (
	"fmt"
	"strings"
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
