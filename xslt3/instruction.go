package xslt3

import (
	"github.com/lestrrat-go/helium/xpath3"
)

// instruction is the interface implemented by all compiled XSLT instructions.
type instruction interface {
	instructionTag()
}

// xpathNSHolder is implemented by instructions that may carry a
// per-instruction xpath-default-namespace override.
type xpathNSHolder interface {
	getXPathDefaultNS() string
	xpathNSIsSet() bool
}

// xpathNS is embedded in instructions that support xpath-default-namespace.
type xpathNS struct {
	XPathDefaultNS         string
	HasXPathDefaultNS      bool // true when xpath-default-namespace is set (explicit or inherited)
	HasLocalXPathDefaultNS bool // true only when xpath-default-namespace is declared on this element
}

func (x xpathNS) getXPathDefaultNS() string { return x.XPathDefaultNS }
func (x xpathNS) xpathNSIsSet() bool        { return x.HasXPathDefaultNS }

// collationScopeInst wraps an instruction with a default-collation override.
// Emitted by the compiler when default-collation changes on an instruction element.
type collationScopeInst struct {
	DefaultCollation string
	Inner            instruction
}

func (*collationScopeInst) instructionTag() {}

// sourceInfo records the source location of an instruction in the stylesheet.
// Embedded in instruction types to report location for xsl:catch error variables.
type sourceInfo struct {
	SourceLine    int    // line number in the stylesheet
	SourceModule  string // stylesheet URI (module)
	StaticBaseURI string // effective static base URI from xml:base (non-empty when overridden)
}

func (s *sourceInfo) setSourceInfo(line int, module string) {
	s.SourceLine = line
	s.SourceModule = module
}

func (s *sourceInfo) getSourceLine() int        { return s.SourceLine }
func (s *sourceInfo) getSourceModule() string   { return s.SourceModule }
func (s *sourceInfo) getStaticBaseURI() string  { return s.StaticBaseURI }
func (s *sourceInfo) setStaticBaseURI(v string) { s.StaticBaseURI = v }

// applyTemplatesInst represents xsl:apply-templates.
type applyTemplatesInst struct {
	sourceInfo
	xpathNS
	Select *xpath3.Expression // nil = "child::node()"
	Mode   string             // "" = current mode, "#default", "#current"
	Sort   []*sortKey
	Params []*withParam
}

func (*applyTemplatesInst) instructionTag() {}

// callTemplateInst represents xsl:call-template.
type callTemplateInst struct {
	sourceInfo
	Name   string
	Params []*withParam
}

func (*callTemplateInst) instructionTag() {}

// withParam is a compiled xsl:with-param.
type withParam struct {
	Name   string
	Select *xpath3.Expression
	Body   []instruction
	As     string
	Tunnel bool
}

// valueOfInst represents xsl:value-of.
type valueOfInst struct {
	sourceInfo
	xpathNS
	Select                *xpath3.Expression
	Separator             *avt // nil = absent; non-nil = present (even if empty)
	Body                  []instruction
	DisableOutputEscaping bool
}

func (*valueOfInst) instructionTag() {}

// textInst represents xsl:text.
type textInst struct {
	sourceInfo
	Value                 string
	TVT                   *avt // text value template (non-nil when expand-text is active)
	DisableOutputEscaping bool
}

func (*textInst) instructionTag() {}

// literalTextInst represents literal text content in the stylesheet.
type literalTextInst struct {
	sourceInfo
	Value string
	TVT   *avt // text value template (non-nil when expand-text is active)
}

func (*literalTextInst) instructionTag() {}

// elementInst represents xsl:element.
type elementInst struct {
	sourceInfo
	Name              *avt
	Namespace         *avt
	NSBindings        map[string]string // compile-time in-scope namespace bindings
	Body              []instruction
	TypeName          string   // XSD type annotation (e.g., "xs:integer")
	Validation        string   // "strict", "lax", "preserve", "strip"
	UseAttrSets       []string // xsl:use-attribute-sets (resolved QNames)
	InheritNamespaces bool     // inherit-namespaces="yes" (default true)
}

func (*elementInst) instructionTag() {}

// attributeInst represents xsl:attribute.
type attributeInst struct {
	sourceInfo
	Name       *avt
	Namespace  *avt
	Select     *xpath3.Expression
	Body       []instruction
	Separator  *avt
	TypeName   string // XSD type annotation (e.g., "xs:ID")
	Validation string // "strict", "lax", "preserve", "strip"
}

func (*attributeInst) instructionTag() {}

// commentInst represents xsl:comment.
type commentInst struct {
	sourceInfo
	Select *xpath3.Expression
	Body   []instruction
}

func (*commentInst) instructionTag() {}

// piInst represents xsl:processing-instruction.
type piInst struct {
	sourceInfo
	Name   *avt
	Select *xpath3.Expression
	Body   []instruction
}

func (*piInst) instructionTag() {}

// ifInst represents xsl:if.
type ifInst struct {
	sourceInfo
	xpathNS
	Test *xpath3.Expression
	Body []instruction
}

func (*ifInst) instructionTag() {}

// chooseInst represents xsl:choose.
type chooseInst struct {
	sourceInfo
	xpathNS
	When             []*whenClause
	Otherwise        []instruction
	OtherwiseXPNS    string // xpath-default-namespace on xsl:otherwise
	HasOtherwiseXPNS bool
	DefaultCollation string // default-collation URI for XPath comparisons
}

func (*chooseInst) instructionTag() {}

// whenClause is one xsl:when branch in xsl:choose.
type whenClause struct {
	xpathNS
	Test             *xpath3.Expression
	Body             []instruction
	Namespaces       map[string]string // per-clause namespace bindings (nil = use stylesheet default)
	DefaultCollation string            // per-clause default-collation (empty = inherit)
}

// forEachInst represents xsl:for-each.
type forEachInst struct {
	sourceInfo
	xpathNS
	Select *xpath3.Expression
	Sort   []*sortKey
	Body   []instruction
}

func (*forEachInst) instructionTag() {}

// variableInst represents xsl:variable (local).
type variableInst struct {
	sourceInfo
	Name   string
	Select *xpath3.Expression
	Body   []instruction
	As     string // type declaration (e.g., "item()*"); empty = wrap body in document node
}

func (*variableInst) instructionTag() {}

// paramInst represents xsl:param (local, in a template).
type paramInst struct {
	sourceInfo
	Name     string
	Select   *xpath3.Expression
	Body     []instruction
	As       string // type declaration (e.g., "xs:integer")
	Required bool
	Tunnel   bool
}

func (*paramInst) instructionTag() {}

// copyInst represents xsl:copy.
type copyInst struct {
	sourceInfo
	Select               *xpath3.Expression
	Body                 []instruction
	Validation           string   // "strict", "lax", "preserve", "strip"
	TypeName             string   // type annotation (e.g., "Q{ns}typeName")
	UseAttributeSets     []string // xsl:use-attribute-sets (resolved QNames)
	UseAttrSets          []string // attribute set names (raw)
	CopyNamespaces       bool     // copy-namespaces="yes" (default true)
	InheritNamespaces    bool     // inherit-namespaces="yes" (default true)
	CopyNamespacesAVT    *avt     // shadow attr _copy-namespaces (overrides CopyNamespaces)
	InheritNamespacesAVT *avt     // shadow attr _inherit-namespaces (overrides InheritNamespaces)
}

func (*copyInst) instructionTag() {}

// copyOfInst represents xsl:copy-of.
type copyOfInst struct {
	sourceInfo
	Select            *xpath3.Expression
	Validation        string // "strict", "lax", "preserve", "strip"
	TypeName          string // type annotation (e.g., "Q{ns}typeName")
	CopyNamespaces    bool   // copy-namespaces="yes" (default true)
	CopyNamespacesAVT *avt   // shadow attr _copy-namespaces (overrides CopyNamespaces)
	CopyAccumulators  bool   // copy-accumulators="yes" (default false)
}

func (*copyOfInst) instructionTag() {}

// literalResultElement represents a non-XSLT element in the stylesheet body.
type literalResultElement struct {
	sourceInfo
	xpathNS
	Name              string // qualified name (prefix:local)
	Namespace         string // namespace URI
	Prefix            string
	LocalName         string
	Attrs             []*literalAttribute
	Namespaces        map[string]string // prefix -> URI declarations to copy
	Body              []instruction
	UseAttrSets       []string // xsl:use-attribute-sets (resolved QNames)
	Validation        string   // xsl:validation="strict"|"lax"|"preserve"|"strip"
	TypeName          string   // xsl:type annotation (e.g., "xs:integer")
	DefaultValidation string   // xsl:default-validation override for this LRE scope
	InheritNamespaces bool     // xsl:inherit-namespaces (default true)
}

func (*literalResultElement) instructionTag() {}

// literalAttribute is a computed attribute on a literal result element.
type literalAttribute struct {
	Name      string // qualified name
	Namespace string
	Prefix    string
	LocalName string
	Value     *avt
}

// numberInst represents xsl:number.
type numberInst struct {
	sourceInfo
	Level             string // "single", "multiple", "any"
	Count             *pattern
	From              *pattern
	Value             *xpath3.Expression
	Format            *avt
	GroupingSeparator *avt
	GroupingSize      *avt
	Ordinal           *avt
	StartAt           *avt               // XSLT 3.0 start-at attribute
	Select            *xpath3.Expression // XSLT 3.0 select attribute
	Lang              *avt               // language for word/ordinal numbering
	LetterValue       *avt               // "alphabetic" or "traditional"
}

func (*numberInst) instructionTag() {}

// messageInst represents xsl:message.
type messageInst struct {
	sourceInfo
	Select    *xpath3.Expression
	Body      []instruction
	Terminate *avt // defaults to "no"
	ErrorCode *avt // defaults to "XTMM9000"
}

func (*messageInst) instructionTag() {}

// sequenceInst represents a sequence of instructions (implicit block).
type sequenceInst struct {
	sourceInfo
	Body              []instruction
	DefaultValidation string // xsl:default-validation override (from xsl:sequence)
}

func (*sequenceInst) instructionTag() {}

// namespaceInst represents xsl:namespace.
type namespaceInst struct {
	sourceInfo
	Name   *avt
	Select *xpath3.Expression
	Body   []instruction
}

func (*namespaceInst) instructionTag() {}

// resultDocumentInst represents xsl:result-document.
// The body output goes to a secondary result, not the primary output.
type resultDocumentInst struct {
	sourceInfo
	Href             *avt
	Body             []instruction
	Format           string   // name of xsl:output to use (static format attribute)
	FormatAVT        *avt     // dynamic format attribute (when it contains {…})
	Method           string   // output method override (from method attribute)
	ItemSeparator    *avt     // item-separator override (avt); nil = not specified
	ItemSeparatorSet bool     // true when item-separator attribute is present (including #absent)
	Validation       string   // "strict", "lax", "preserve", "strip"
	TypeName         string   // type annotation (e.g., "Q{ns}typeName")
	UseCharacterMaps []string // names of character maps to use

	NSBindings map[string]string // compile-time namespace bindings for QName resolution
	// Serialization parameter AVTs (evaluated at runtime).
	OutputVersion           *avt
	Encoding                *avt
	Indent                  *avt
	OmitXMLDeclaration      *avt
	Standalone              *avt
	DoctypeSystem           *avt
	DoctypePublic           *avt
	CDATASectionElements    *avt
	ByteOrderMark           *avt
	MethodAVT               *avt // method as avt (overrides Method if non-nil)
	MediaType               *avt
	HTMLVersion             *avt
	IncludeContentType      *avt
	AllowDuplicateNames     *avt
	EscapeURIAttributes     *avt
	JSONNodeOutputMethodAVT *avt       // json-node-output-method avt
	NormalizationForm       *avt       // normalization-form avt
	SuppressIndentation     []string   // suppress-indentation element names
	ParameterDocAVT         *avt       // parameter-document avt
	ParameterDocOutputDef   *OutputDef // resolved output def from parameter-document (compile-time)
	BuildTree               *bool      // build-tree: nil=default(true), true/false
}

func (*resultDocumentInst) instructionTag() {}

// xslSequenceInst represents xsl:sequence with a select attribute.
type xslSequenceInst struct {
	sourceInfo
	Select *xpath3.Expression
}

func (*xslSequenceInst) instructionTag() {}

// mapInst represents xsl:map.
type mapInst struct {
	sourceInfo
	Body []instruction // child xsl:map-entry instructions
}

func (*mapInst) instructionTag() {}

// mapEntryInst represents xsl:map-entry.
type mapEntryInst struct {
	sourceInfo
	xpathNS
	Key    *xpath3.Expression // key expression
	Select *xpath3.Expression // optional select expression for value
	Body   []instruction      // optional body for value
}

func (*mapEntryInst) instructionTag() {}

// performSortInst represents xsl:perform-sort.
type performSortInst struct {
	sourceInfo
	Select *xpath3.Expression
	Sort   []*sortKey
	Body   []instruction
}

func (*performSortInst) instructionTag() {}

// nextMatchInst represents xsl:next-match.
type nextMatchInst struct {
	sourceInfo
	Params []*withParam
}

func (*nextMatchInst) instructionTag() {}

// applyImportsInst represents xsl:apply-imports.
type applyImportsInst struct {
	sourceInfo
	Params []*withParam
}

func (*applyImportsInst) instructionTag() {}

// wherePopulatedInst represents xsl:where-populated.
type wherePopulatedInst struct {
	sourceInfo
	Body []instruction
}

func (*wherePopulatedInst) instructionTag() {}

// onEmptyInst represents xsl:on-empty.
// Executes its body/select only if the current output container has no significant content.
type onEmptyInst struct {
	sourceInfo
	Body   []instruction
	Select *xpath3.Expression
}

func (*onEmptyInst) instructionTag() {}

// onNonEmptyInst represents xsl:on-non-empty.
// Executes its body/select only if the current output container has significant content.
type onNonEmptyInst struct {
	sourceInfo
	Body   []instruction
	Select *xpath3.Expression
}

func (*onNonEmptyInst) instructionTag() {}

// tryCatchInst represents xsl:try/xsl:catch.
type tryCatchInst struct {
	sourceInfo
	Select         *xpath3.Expression // xsl:try select attribute
	Try            []instruction
	Catches        []*catchClause // multiple catch clauses (matched in order)
	RollbackOutput bool           // rollback-output="yes" (default true)
}

func (*tryCatchInst) instructionTag() {}

// catchClause represents a single xsl:catch clause.
type catchClause struct {
	Errors []string           // error codes to match (empty = "*")
	Select *xpath3.Expression // xsl:catch select attribute
	Body   []instruction      // xsl:catch body
}

// sourceDocumentInst represents xsl:source-document.
type sourceDocumentInst struct {
	sourceInfo
	Href            *avt
	Streamable      bool
	UseAccumulators []string
	BaseURI         string // effective base URI for resolving href
	Validation      string // "strict", "lax", "preserve", "strip"
	TypeName        string // resolved QName for type attribute
	Body            []instruction
}

func (*sourceDocumentInst) instructionTag() {}

// iterateInst represents xsl:iterate.
type iterateInst struct {
	sourceInfo
	Select       *xpath3.Expression
	Params       []*iterateParam
	OnCompletion []instruction
	Body         []instruction
}

func (*iterateInst) instructionTag() {}

// iterateParam is a compiled xsl:param inside xsl:iterate.
type iterateParam struct {
	Name   string
	Select *xpath3.Expression
	Body   []instruction
	As     string // type declaration (e.g., "element()*")
}

// breakInst represents xsl:break.
type breakInst struct {
	sourceInfo
	Select *xpath3.Expression
	Body   []instruction
}

func (*breakInst) instructionTag() {}

// nextIterationInst represents xsl:next-iteration.
type nextIterationInst struct {
	sourceInfo
	Params []*withParam
}

func (*nextIterationInst) instructionTag() {}

// forkInst represents xsl:fork.
// Each entry in Branches is a sequence of instructions from one child.
type forkInst struct {
	sourceInfo
	Branches [][]instruction
}

func (*forkInst) instructionTag() {}

// forEachGroupInst represents xsl:for-each-group.
type forEachGroupInst struct {
	sourceInfo
	Select            *xpath3.Expression
	GroupBy           *xpath3.Expression
	GroupAdjacent     *xpath3.Expression
	GroupStartingWith *pattern
	GroupEndingWith   *pattern
	Collation         *avt
	Composite         bool
	Sort              []*sortKey
	Body              []instruction
}

func (*forEachGroupInst) instructionTag() {}

// mergeInst represents xsl:merge.
type mergeInst struct {
	sourceInfo
	Sources []*mergeSource
	Action  []instruction
}

func (*mergeInst) instructionTag() {}

// mergeSource represents xsl:merge-source.
type mergeSource struct {
	Name            string
	ForEachSource   *xpath3.Expression // XPath expr evaluating to sequence of URIs
	ForEachItem     *xpath3.Expression // XPath expr evaluating to sequence of items (nodes)
	Select          *xpath3.Expression
	UseAccumulators []string
	StreamableAttr  bool
	SortBeforeMerge bool
	BaseURI         string // effective base URI for resolving for-each-source URIs
	Keys            []*mergeKey
}

// mergeKey represents xsl:merge-key.
type mergeKey struct {
	Select       *xpath3.Expression
	Body         []instruction // used when select is absent
	Order        string        // "ascending" or "descending" (static)
	OrderAVT     *avt          // non-nil when order is an avt
	DataType     string        // "text" or "number" (static)
	DataTypeAVT  *avt          // non-nil when data-type is an avt
	HasCollation bool          // true when lang, collation, or case-order is specified
	Collation    string        // collation URI (for XTDE2210 mismatch detection)
	CollationAVT *avt          // non-nil when collation is an AVT
	Lang         string        // lang attribute value
	CaseOrder    string        // case-order attribute value
}

// documentInst represents xsl:document.
// It creates a document node wrapping its content body.
type documentInst struct {
	sourceInfo
	Validation string // "strict", "lax", "preserve", "strip"
	TypeName   string // type annotation (e.g., "Q{ns}typeName")
	Body       []instruction
}

func (*documentInst) instructionTag() {}

// analyzeStringInst represents xsl:analyze-string.
type analyzeStringInst struct {
	sourceInfo
	Select          *xpath3.Expression
	Regex           *avt
	Flags           *avt
	MatchingBody    []instruction
	NonMatchingBody []instruction
}

func (*analyzeStringInst) instructionTag() {}

// assertInst represents xsl:assert.
// When test evaluates to false, a dynamic error XTMM9001 is raised
// (or the error-code specified by the error-code attribute).
type assertInst struct {
	sourceInfo
	xpathNS
	Test      *xpath3.Expression
	Select    *xpath3.Expression
	ErrorCode string // default "XTMM9001"
	Body      []instruction
}

func (*assertInst) instructionTag() {}

// evaluateInst represents xsl:evaluate.
type evaluateInst struct {
	sourceInfo
	xpathNS
	XPath            *xpath3.Expression // xpath attribute (expression producing the XPath string)
	ContextItem      *xpath3.Expression // context-item attribute (optional)
	BaseURI          *avt               // base-uri attribute (optional)
	NamespaceContext *xpath3.Expression // namespace-context attribute (optional, expression producing a node)
	WithParamsExpr   *xpath3.Expression // with-params attribute (optional, map expression)
	As               string             // as attribute (optional sequence type)
	SchemaAwareAVT   *avt               // nil = absent; non-nil = schema-aware attribute present
	Params           []*withParam       // child xsl:with-param elements
}

func (*evaluateInst) instructionTag() {}

// fallbackInst represents a forwards-compatible unknown XSLT instruction.
// At runtime, the Body (from xsl:fallback children) is executed.
// If no xsl:fallback was found, XTDE1450 is raised.
type fallbackInst struct {
	sourceInfo
	Body        []instruction // compiled xsl:fallback children
	Name        string        // original instruction name for error messages
	HasFallback bool          // true if at least one xsl:fallback was found
}

func (*fallbackInst) instructionTag() {}
