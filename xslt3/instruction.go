package xslt3

import (
	"github.com/lestrrat-go/helium/xpath3"
)

// Instruction is the interface implemented by all compiled XSLT instructions.
type Instruction interface {
	instructionTag()
}

// xpathNSHolder is implemented by instructions that may carry a
// per-instruction xpath-default-namespace override.
type xpathNSHolder interface {
	getXPathDefaultNS() string
}

// xpathNS is embedded in instructions that support xpath-default-namespace.
type xpathNS struct {
	XPathDefaultNS    string
	HasXPathDefaultNS bool // true when xpath-default-namespace is explicitly set
}

func (x xpathNS) getXPathDefaultNS() string { return x.XPathDefaultNS }
func (x xpathNS) xpathNSIsSet() bool        { return x.HasXPathDefaultNS }

// ApplyTemplatesInst represents xsl:apply-templates.
type ApplyTemplatesInst struct {
	xpathNS
	Select *xpath3.Expression // nil = "child::node()"
	Mode   string             // "" = current mode, "#default", "#current"
	Sort   []*SortKey
	Params []*WithParam
}

func (*ApplyTemplatesInst) instructionTag() {}

// CallTemplateInst represents xsl:call-template.
type CallTemplateInst struct {
	Name   string
	Params []*WithParam
}

func (*CallTemplateInst) instructionTag() {}

// WithParam is a compiled xsl:with-param.
type WithParam struct {
	Name   string
	Select *xpath3.Expression
	Body   []Instruction
	As     string
	Tunnel bool
}

// ValueOfInst represents xsl:value-of.
type ValueOfInst struct {
	xpathNS
	Select       *xpath3.Expression
	Separator    *AVT // default " " for 3.0, absent for 1.0
	HasSeparator bool // true when separator attribute is explicitly specified
	Body         []Instruction
}

func (*ValueOfInst) instructionTag() {}

// TextInst represents xsl:text.
type TextInst struct {
	Value                 string
	TVT                   *AVT // text value template (non-nil when expand-text is active)
	DisableOutputEscaping bool
}

func (*TextInst) instructionTag() {}

// LiteralTextInst represents literal text content in the stylesheet.
type LiteralTextInst struct {
	Value string
	TVT   *AVT // text value template (non-nil when expand-text is active)
}

func (*LiteralTextInst) instructionTag() {}

// ElementInst represents xsl:element.
type ElementInst struct {
	Name             *AVT
	Namespace        *AVT
	Body             []Instruction
	TypeName         string   // XSD type annotation (e.g., "xs:integer")
	UseAttributeSets []string // xsl:use-attribute-sets (resolved QNames)
	UseAttrSets      []string // attribute set names (raw)
}

func (*ElementInst) instructionTag() {}

// AttributeInst represents xsl:attribute.
type AttributeInst struct {
	Name      *AVT
	Namespace *AVT
	Select    *xpath3.Expression
	Body      []Instruction
	Separator *AVT
	TypeName  string // XSD type annotation (e.g., "xs:ID")
}

func (*AttributeInst) instructionTag() {}

// CommentInst represents xsl:comment.
type CommentInst struct {
	Select *xpath3.Expression
	Body   []Instruction
}

func (*CommentInst) instructionTag() {}

// PIInst represents xsl:processing-instruction.
type PIInst struct {
	Name   *AVT
	Select *xpath3.Expression
	Body   []Instruction
}

func (*PIInst) instructionTag() {}

// IfInst represents xsl:if.
type IfInst struct {
	xpathNS
	Test *xpath3.Expression
	Body []Instruction
}

func (*IfInst) instructionTag() {}

// ChooseInst represents xsl:choose.
type ChooseInst struct {
	xpathNS
	When              []*WhenClause
	Otherwise         []Instruction
	OtherwiseXPNS     string // xpath-default-namespace on xsl:otherwise
	HasOtherwiseXPNS  bool
}

func (*ChooseInst) instructionTag() {}

// WhenClause is one xsl:when branch in xsl:choose.
type WhenClause struct {
	xpathNS
	Test *xpath3.Expression
	Body []Instruction
}

// ForEachInst represents xsl:for-each.
type ForEachInst struct {
	xpathNS
	Select *xpath3.Expression
	Sort   []*SortKey
	Body   []Instruction
}

func (*ForEachInst) instructionTag() {}

// VariableInst represents xsl:variable (local).
type VariableInst struct {
	Name   string
	Select *xpath3.Expression
	Body   []Instruction
	As     string // type declaration (e.g., "item()*"); empty = wrap body in document node
}

func (*VariableInst) instructionTag() {}

// ParamInst represents xsl:param (local, in a template).
type ParamInst struct {
	Name     string
	Select   *xpath3.Expression
	Body     []Instruction
	As       string // type declaration (e.g., "xs:integer")
	Required bool
	Tunnel   bool
}

func (*ParamInst) instructionTag() {}

// CopyInst represents xsl:copy.
type CopyInst struct {
	Select            *xpath3.Expression
	Body              []Instruction
	Validation        string   // "strict", "lax", "preserve", "strip"
	UseAttributeSets  []string // xsl:use-attribute-sets (resolved QNames)
	UseAttrSets       []string // attribute set names (raw)
	CopyNamespaces    bool     // copy-namespaces="yes" (default true)
	InheritNamespaces bool     // inherit-namespaces="yes" (default true)
}

func (*CopyInst) instructionTag() {}

// CopyOfInst represents xsl:copy-of.
type CopyOfInst struct {
	Select         *xpath3.Expression
	Validation     string // "strict", "lax", "preserve", "strip"
	CopyNamespaces bool   // copy-namespaces="yes" (default true)
}

func (*CopyOfInst) instructionTag() {}

// LiteralResultElement represents a non-XSLT element in the stylesheet body.
type LiteralResultElement struct {
	Name             string // qualified name (prefix:local)
	Namespace        string // namespace URI
	Prefix           string
	LocalName        string
	Attrs            []*LiteralAttribute
	Namespaces       map[string]string // prefix -> URI declarations to copy
	UseAttributeSets []string          // xsl:use-attribute-sets (resolved QNames)
	Body             []Instruction
	UseAttrSets      []string // names of attribute sets to apply (raw)
}

func (*LiteralResultElement) instructionTag() {}

// LiteralAttribute is a computed attribute on a literal result element.
type LiteralAttribute struct {
	Name      string // qualified name
	Namespace string
	Prefix    string
	LocalName string
	Value     *AVT
}

// NumberInst represents xsl:number.
type NumberInst struct {
	Level             string             // "single", "multiple", "any"
	Count             *Pattern
	From              *Pattern
	Value             *xpath3.Expression
	Format            *AVT
	GroupingSeparator *AVT
	GroupingSize      *AVT
	Ordinal           *AVT
	StartAt           *AVT              // XSLT 3.0 start-at attribute
	Select            *xpath3.Expression // XSLT 3.0 select attribute
	Lang              *AVT              // language for word/ordinal numbering
	LetterValue       *AVT              // "alphabetic" or "traditional"
}

func (*NumberInst) instructionTag() {}

// MessageInst represents xsl:message.
type MessageInst struct {
	Select    *xpath3.Expression
	Body      []Instruction
	Terminate *AVT // defaults to "no"
	ErrorCode *AVT // defaults to "XTMM9000"
}

func (*MessageInst) instructionTag() {}

// SequenceInst represents a sequence of instructions (implicit block).
type SequenceInst struct {
	Body []Instruction
}

func (*SequenceInst) instructionTag() {}

// NamespaceInst represents xsl:namespace.
type NamespaceInst struct {
	Name   *AVT
	Select *xpath3.Expression
	Body   []Instruction
}

func (*NamespaceInst) instructionTag() {}

// ResultDocumentInst represents xsl:result-document.
// The body output goes to a secondary result, not the primary output.
type ResultDocumentInst struct {
	Href *AVT
	Body []Instruction
}

func (*ResultDocumentInst) instructionTag() {}

// XSLSequenceInst represents xsl:sequence with a select attribute.
type XSLSequenceInst struct {
	Select *xpath3.Expression
}

func (*XSLSequenceInst) instructionTag() {}

// MapInst represents xsl:map.
type MapInst struct {
	Body []Instruction // child xsl:map-entry instructions
}

func (*MapInst) instructionTag() {}

// MapEntryInst represents xsl:map-entry.
type MapEntryInst struct {
	xpathNS
	Key    *xpath3.Expression // key expression
	Select *xpath3.Expression // optional select expression for value
	Body   []Instruction      // optional body for value
}

func (*MapEntryInst) instructionTag() {}

// PerformSortInst represents xsl:perform-sort.
type PerformSortInst struct {
	Select *xpath3.Expression
	Sort   []*SortKey
	Body   []Instruction
}

func (*PerformSortInst) instructionTag() {}

// NextMatchInst represents xsl:next-match.
type NextMatchInst struct {
	Params []*WithParam
}

func (*NextMatchInst) instructionTag() {}

// ApplyImportsInst represents xsl:apply-imports.
type ApplyImportsInst struct {
	Params []*WithParam
}

func (*ApplyImportsInst) instructionTag() {}

// WherePopulatedInst represents xsl:where-populated.
type WherePopulatedInst struct {
	Body []Instruction
}

func (*WherePopulatedInst) instructionTag() {}

// OnEmptyInst represents xsl:on-empty.
// Executes its body/select only if the current output container has no significant content.
type OnEmptyInst struct {
	Body   []Instruction
	Select *xpath3.Expression
}

func (*OnEmptyInst) instructionTag() {}

// TryCatchInst represents xsl:try/xsl:catch.
type TryCatchInst struct {
	Select  *xpath3.Expression // xsl:try select attribute
	Try     []Instruction
	Catches []*CatchClause // multiple catch clauses (matched in order)
}

func (*TryCatchInst) instructionTag() {}

// CatchClause represents a single xsl:catch clause.
type CatchClause struct {
	Errors []string           // error codes to match (empty = "*")
	Select *xpath3.Expression // xsl:catch select attribute
	Body   []Instruction      // xsl:catch body
}

// SourceDocumentInst represents xsl:source-document.
type SourceDocumentInst struct {
	Href       *AVT
	Streamable bool
	Body       []Instruction
}

func (*SourceDocumentInst) instructionTag() {}

// IterateInst represents xsl:iterate.
type IterateInst struct {
	Select       *xpath3.Expression
	Params       []*IterateParam
	OnCompletion []Instruction
	Body         []Instruction
}

func (*IterateInst) instructionTag() {}

// IterateParam is a compiled xsl:param inside xsl:iterate.
type IterateParam struct {
	Name   string
	Select *xpath3.Expression
	Body   []Instruction
	As     string // type declaration (e.g., "element()*")
}

// BreakInst represents xsl:break.
type BreakInst struct {
	Select *xpath3.Expression
	Body   []Instruction
}

func (*BreakInst) instructionTag() {}

// NextIterationInst represents xsl:next-iteration.
type NextIterationInst struct {
	Params []*WithParam
}

func (*NextIterationInst) instructionTag() {}

// ForkInst represents xsl:fork.
// Each entry in Branches is a sequence of instructions from one child.
type ForkInst struct {
	Branches [][]Instruction
}

func (*ForkInst) instructionTag() {}

// ForEachGroupInst represents xsl:for-each-group.
type ForEachGroupInst struct {
	Select            *xpath3.Expression
	GroupBy           *xpath3.Expression
	GroupAdjacent     *xpath3.Expression
	GroupStartingWith *Pattern
	GroupEndingWith   *Pattern
	Composite         bool
	Sort              []*SortKey
	Body              []Instruction
}

func (*ForEachGroupInst) instructionTag() {}

// MergeInst represents xsl:merge.
type MergeInst struct {
	Sources []*MergeSource
	Action  []Instruction
}

func (*MergeInst) instructionTag() {}

// MergeSource represents xsl:merge-source.
type MergeSource struct {
	Name            string
	ForEachSource   *xpath3.Expression // XPath expr evaluating to sequence of URIs
	ForEachItem     *xpath3.Expression // XPath expr evaluating to sequence of items (nodes)
	Select          *xpath3.Expression
	StreamableAttr  bool
	SortBeforeMerge bool
	Keys            []*MergeKey
}

// MergeKey represents xsl:merge-key.
type MergeKey struct {
	Select *xpath3.Expression
	Order  string // "ascending" or "descending"
}

// AnalyzeStringInst represents xsl:analyze-string.
type AnalyzeStringInst struct {
	Select          *xpath3.Expression
	Regex           *AVT
	Flags           *AVT
	MatchingBody    []Instruction
	NonMatchingBody []Instruction
}

func (*AnalyzeStringInst) instructionTag() {}
