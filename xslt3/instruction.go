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
	Select    *xpath3.Expression
	Separator *AVT // default " " for 3.0, absent for 1.0
	Body      []Instruction
}

func (*ValueOfInst) instructionTag() {}

// TextInst represents xsl:text.
type TextInst struct {
	Value                string
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
	Name      *AVT
	Namespace *AVT
	Body      []Instruction
}

func (*ElementInst) instructionTag() {}

// AttributeInst represents xsl:attribute.
type AttributeInst struct {
	Name      *AVT
	Namespace *AVT
	Select    *xpath3.Expression
	Body      []Instruction
	Separator *AVT
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
	Required bool
	Tunnel   bool
}

func (*ParamInst) instructionTag() {}

// CopyInst represents xsl:copy.
type CopyInst struct {
	Select *xpath3.Expression
	Body   []Instruction
}

func (*CopyInst) instructionTag() {}

// CopyOfInst represents xsl:copy-of.
type CopyOfInst struct {
	Select *xpath3.Expression
}

func (*CopyOfInst) instructionTag() {}

// LiteralResultElement represents a non-XSLT element in the stylesheet body.
type LiteralResultElement struct {
	Name       string // qualified name (prefix:local)
	Namespace  string // namespace URI
	Prefix     string
	LocalName  string
	Attrs      []*LiteralAttribute
	Namespaces map[string]string // prefix -> URI declarations to copy
	Body       []Instruction
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

// XSLSequenceInst represents xsl:sequence with a select attribute.
type XSLSequenceInst struct {
	Select *xpath3.Expression
}

func (*XSLSequenceInst) instructionTag() {}

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
	Select     *xpath3.Expression // xsl:try select attribute
	Try        []Instruction
	Catch      []Instruction
	CatchSelect *xpath3.Expression // xsl:catch select attribute
}

func (*TryCatchInst) instructionTag() {}

// ForEachGroupInst represents xsl:for-each-group.
type ForEachGroupInst struct {
	Select            *xpath3.Expression
	GroupBy           *xpath3.Expression
	GroupAdjacent     *xpath3.Expression
	GroupStartingWith *Pattern
	GroupEndingWith   *Pattern
	Sort              []*SortKey
	Body              []Instruction
}

func (*ForEachGroupInst) instructionTag() {}
