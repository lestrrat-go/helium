package xsd

import (
	"regexp"

	"github.com/lestrrat-go/helium/internal/xsd/value"
)

// validateBuiltinValue validates a value against a builtin XSD type's lexical space.
func validateBuiltinValue(v, builtinLocal string) error {
	return value.ValidateBuiltin(v, builtinLocal)
}

// validateQName validates a QName value.
func validateQName(v string) error {
	return value.ValidateBuiltin(v, "QName")
}

// languageRegex matches the lexical space of xs:language (RFC 3066).
var languageRegex = regexp.MustCompile(`^[a-zA-Z]{1,8}(-[a-zA-Z0-9]{1,8})*$`)
