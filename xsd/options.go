package xsd

import helium "github.com/lestrrat-go/helium"

type compileConfig struct {
	filename     string // XSD filename for error messages
	baseDir      string // base directory for resolving relative includes
	errorHandler helium.ErrorHandler
}

type validateConfig struct {
	filename       string
	errorHandler   helium.ErrorHandler
	annotations    *TypeAnnotations
	nilledElements *NilledElements
}

// TypeAnnotations maps document nodes to their XSD type names.
// Type names use the "xs:localName" format for built-in types and
// "Q{ns}localName" for user-defined types.
type TypeAnnotations map[helium.Node]string

// NilledElements tracks elements whose xsi:nil="true" was confirmed valid
// during schema validation (i.e. the element declaration is nillable).
type NilledElements map[*helium.Element]struct{}
