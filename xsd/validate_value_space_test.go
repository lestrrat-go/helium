package xsd_test

// Shared test constants for the value-space comparison tests
// (validate_enumeration_value_space_test.go and
// validate_fixed_value_space_test.go). Hoisting the repeated XSD type names and
// literals here keeps each literal defined once, satisfying goconst.
const (
	xsDecimalType   = "xs:decimal"
	xsBooleanType   = "xs:boolean"
	xsStringType    = "xs:string"
	xsHexBinaryType = "xs:hexBinary"

	nanLexical = "NaN"
	abcLiteral = "abc"
)
