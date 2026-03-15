package xslt3

import (
	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
)

const (
	// XSLT namespace URI.
	NSXSLT = "http://www.w3.org/1999/XSL/Transform"
)

// Stylesheet is a compiled XSLT stylesheet ready for transformation.
type Stylesheet struct {
	version         string
	templates       []*Template
	namedTemplates  map[string]*Template
	modeTemplates   map[string][]*Template // mode -> templates sorted by import-precedence then priority
	defaultMode     string
	globalVars      []*Variable // topologically sorted
	globalParams    []*Param
	keys            map[string]*KeyDef
	outputs         map[string]*OutputDef                 // "" = default output
	functions       map[xpath3.QualifiedName]*XSLFunction // xsl:function defs
	stripSpace      []NameTest
	preserveSpace   []NameTest
	namespaces      map[string]string   // prefix -> URI from stylesheet
	excludePrefixes map[string]struct{} // prefixes excluded from output
	decimalFormats  map[xpath3.QualifiedName]xpath3.DecimalFormat // named decimal formats
	modeDefs        map[string]*ModeDef                         // mode name -> mode definition
	attributeSets     map[string]*AttributeSetDef                 // xsl:attribute-set definitions
	accumulators      map[string]*AccumulatorDef                  // accumulator name -> definition
	sourceDoc         *helium.Document                            // the parsed stylesheet document (for document(""))
	baseURI           string                                      // base URI for resolving relative document references
	schemas           []*xsd.Schema                               // imported schemas (xsl:import-schema)
	defaultValidation string                                      // "strict", "lax", "preserve", "strip" (default-validation attr)
}

// ModeDef is a compiled xsl:mode declaration.
type ModeDef struct {
	Name             string
	OnNoMatch        string // "shallow-copy", "deep-copy", "shallow-skip", "deep-skip", "text-only-copy", "fail"
	Streamable       bool
	Visibility       string // "public", "private", "final"
	OnMultipleMatch  string // "use-last", "fail"
	UseAccumulators  string // accumulator names
	ImportPrec       int
}

// AttributeSetDef is a compiled xsl:attribute-set.
type AttributeSetDef struct {
	Name          string
	UseAttrSets   []string      // names of other attribute sets to include
	Attrs         []Instruction // xsl:attribute instructions
}

// XSLFunction is a compiled xsl:function.
type XSLFunction struct {
	Name          xpath3.QualifiedName
	Params        []*Param
	Body          []Instruction
	As            string // return type constraint (e.g., "xs:string?")
	Streamability string // "absorbing", "inspection", etc.; "" = unspecified
}

// Template is a compiled xsl:template.
type Template struct {
	Match          *Pattern
	Name           string
	Mode           string // "" = default, "#all" = all modes
	Priority       float64
	ImportPrec     int
	Params         []*Param
	Body           []Instruction
	XPathDefaultNS string // xpath-default-namespace (inherited or explicit)
}

// Variable is a compiled xsl:variable.
type Variable struct {
	Name   string
	Select *xpath3.Expression
	Body   []Instruction // used when select is absent
	As     string        // type declaration (e.g., "element()*")
}

// Param is a compiled xsl:param.
type Param struct {
	Name     string
	Select   *xpath3.Expression
	Body     []Instruction // used when select is absent
	As       string        // type declaration (e.g., "xs:integer")
	Required bool
	Tunnel   bool
}

// KeyDef is a compiled xsl:key.
type KeyDef struct {
	Name  string
	Match *Pattern
	Use   *xpath3.Expression
	Body  []Instruction // content constructor (when use attribute is absent)
}

// OutputDef is a compiled xsl:output.
type OutputDef struct {
	Name              string
	Method            string // "xml", "html", "text"
	Encoding          string
	Indent            bool
	OmitDeclaration   bool
	Standalone        string // "yes", "no", "omit"
	CDATASections     []string
	DoctypePublic     string
	DoctypeSystem     string
	MediaType         string
	Version           string
	UndeclarePrefixes bool
}

// AccumulatorDef is a compiled xsl:accumulator.
type AccumulatorDef struct {
	Name        string
	As          string // type declaration
	Initial     *xpath3.Expression
	InitialBody []Instruction
	Rules       []*AccumulatorRule
	Streamable  bool
}

// AccumulatorRule is a compiled xsl:accumulator-rule.
type AccumulatorRule struct {
	Match  *Pattern
	Phase  string // "start" or "end"
	Select *xpath3.Expression
	Body   []Instruction
	New    bool // new="yes" starts a fresh value
}

// NameTest is used for xsl:strip-space and xsl:preserve-space element names.
type NameTest struct {
	Prefix string
	Local  string // "*" = wildcard
}
