package schematron

import (
	"github.com/lestrrat-go/helium/xpath"
)

// Schema is a compiled Schematron schema.
// (libxml2: xmlSchematronPtr)
type Schema struct {
	patterns        []*pattern
	namespaces      map[string]string // prefix -> URI from <ns> elements
	title           string
	compileErrors   string
	compileWarnings string
}

// CompileErrors returns any errors encountered during schema compilation.
func (s *Schema) CompileErrors() string {
	return s.compileErrors
}

// CompileWarnings returns any warnings encountered during schema compilation.
func (s *Schema) CompileWarnings() string {
	return s.compileWarnings
}

type pattern struct {
	name  string // id or name attribute
	rules []*rule
}

type rule struct {
	context     string             // XPath context expression (source)
	contextExpr *xpath.Expression  // compiled XPath
	tests       []*test
	lets        []*letBinding
	line        int
}

type testType int

const (
	testAssert testType = iota + 1
	testReport
)

type test struct {
	typ      testType
	expr     string             // XPath test expression (source)
	compiled *xpath.Expression  // compiled XPath
	message  []messagePart      // parsed message content
	line     int
}

type letBinding struct {
	name string
	expr *xpath.Expression
}

// messagePart is a piece of an assert/report message.
type messagePart interface {
	msgPart()
}

// textPart is literal text in a message.
type textPart struct {
	text string
}

func (textPart) msgPart() {}

// namePart is a <name/> or <name path="..."/> element in a message.
type namePart struct {
	path string             // XPath path expression (default ".")
	expr *xpath.Expression  // compiled path expression
}

func (namePart) msgPart() {}

// valueOfPart is a <value-of select="..."/> element in a message.
type valueOfPart struct {
	sel  string             // XPath select expression
	expr *xpath.Expression  // compiled select expression
}

func (valueOfPart) msgPart() {}
