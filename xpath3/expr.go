package xpath3

import (
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// Expr is the interface implemented by all XPath 3.1 AST nodes.
type Expr interface {
	exprNode()
}

// AxisType identifies one of the 13 XPath axes, shared with xpath1.
type AxisType = ixpath.AxisType

const (
	AxisChild            = ixpath.AxisChild
	AxisDescendant       = ixpath.AxisDescendant
	AxisParent           = ixpath.AxisParent
	AxisAncestor         = ixpath.AxisAncestor
	AxisFollowingSibling = ixpath.AxisFollowingSibling
	AxisPrecedingSibling = ixpath.AxisPrecedingSibling
	AxisFollowing        = ixpath.AxisFollowing
	AxisPreceding        = ixpath.AxisPreceding
	AxisAttribute        = ixpath.AxisAttribute
	AxisNamespace        = ixpath.AxisNamespace
	AxisSelf             = ixpath.AxisSelf
	AxisDescendantOrSelf = ixpath.AxisDescendantOrSelf
	AxisAncestorOrSelf   = ixpath.AxisAncestorOrSelf
)

// NodeTest filters nodes selected by an axis.
type NodeTest interface {
	nodeTest()
}

// NodeKind identifies the built-in node kind test types.
type NodeKind int

const (
	NodeKindNode                  NodeKind = iota // node()
	NodeKindText                                  // text()
	NodeKindComment                               // comment()
	NodeKindProcessingInstruction                 // processing-instruction()
)

// --- Node Tests ---

// NameTest matches by qualified name. Local=="*" is a wildcard.
type NameTest struct {
	Prefix string
	Local  string
}

func (NameTest) nodeTest() {}

// TypeTest matches by node kind (node(), text(), comment()).
type TypeTest struct {
	Kind NodeKind
}

func (TypeTest) nodeTest() {}

// PITest matches processing-instruction nodes, optionally by target name.
type PITest struct {
	Target string // empty means match any PI
}

func (PITest) nodeTest() {}

// ElementTest matches element(name?, type?).
type ElementTest struct {
	Name     string
	TypeName string
	Nillable bool
}

func (ElementTest) nodeTest() {}

// AttributeTest matches attribute(name?, type?).
type AttributeTest struct {
	Name     string
	TypeName string
}

func (AttributeTest) nodeTest() {}

// DocumentTest matches document-node(element(...)?)
type DocumentTest struct {
	Inner NodeTest // nil = any document node
}

func (DocumentTest) nodeTest() {}

// SchemaElementTest matches schema-element(name).
type SchemaElementTest struct {
	Name string
}

func (SchemaElementTest) nodeTest() {}

// SchemaAttributeTest matches schema-attribute(name).
type SchemaAttributeTest struct {
	Name string
}

func (SchemaAttributeTest) nodeTest() {}

// NamespaceNodeTest matches namespace-node().
type NamespaceNodeTest struct{}

func (NamespaceNodeTest) nodeTest() {}

// FunctionTest matches function(*).
type FunctionTest struct{}

func (FunctionTest) nodeTest() {}

// MapTest matches map(*).
type MapTest struct{}

func (MapTest) nodeTest() {}

// ArrayTest matches array(*).
type ArrayTest struct{}

func (ArrayTest) nodeTest() {}

// AnyItemTest matches item().
type AnyItemTest struct{}

func (AnyItemTest) nodeTest() {}

// AtomicOrUnionType matches a named atomic/union type (e.g. xs:integer).
type AtomicOrUnionType struct {
	Prefix string
	Name   string
}

func (AtomicOrUnionType) nodeTest() {}

// --- Sequence Type ---

// Occurrence constrains sequence cardinality in SequenceType.
type Occurrence int

const (
	OccurrenceExactlyOne Occurrence = iota // (default)
	OccurrenceZeroOrOne                    // ?
	OccurrenceZeroOrMore                   // *
	OccurrenceOneOrMore                    // +
)

// SequenceType is used in instance-of, cast-as, castable-as, treat-as.
type SequenceType struct {
	ItemTest   NodeTest   // reuses NodeTest interface for item type matching
	Occurrence Occurrence
	Void       bool // true for empty-sequence()
}

// --- Literals & Variables ---

// LiteralExpr represents a string or numeric literal.
type LiteralExpr struct {
	Value any // string, float64, *big.Int, or *big.Rat
}

func (LiteralExpr) exprNode() {}

// VariableExpr represents a variable reference ($prefix:name or $name).
type VariableExpr struct {
	Prefix string
	Name   string
}

func (VariableExpr) exprNode() {}

// RootExpr represents a bare / (document root).
type RootExpr struct{}

func (RootExpr) exprNode() {}

// ContextItemExpr represents a bare . (context item).
type ContextItemExpr struct{}

func (ContextItemExpr) exprNode() {}

// --- Location Paths ---

// LocationPath represents an absolute or relative location path.
type LocationPath struct {
	Absolute bool
	Steps    []Step
}

func (LocationPath) exprNode() {}

// Step is a single step in a location path: axis::node-test[predicates].
type Step struct {
	Axis       AxisType
	NodeTest   NodeTest
	Predicates []Expr
}

// --- Operators ---

// BinaryExpr represents binary operations (arithmetic, comparison, logic).
type BinaryExpr struct {
	Op    TokenType
	Left  Expr
	Right Expr
}

func (BinaryExpr) exprNode() {}

// UnaryExpr represents unary negation (-expr) or unary plus (+expr).
type UnaryExpr struct {
	Operand Expr
}

func (UnaryExpr) exprNode() {}

// ConcatExpr represents string concatenation (||).
type ConcatExpr struct {
	Left  Expr
	Right Expr
}

func (ConcatExpr) exprNode() {}

// SimpleMapExpr represents the simple map operator (!).
type SimpleMapExpr struct {
	Left  Expr
	Right Expr
}

func (SimpleMapExpr) exprNode() {}

// RangeExpr represents a range expression (start to end).
type RangeExpr struct {
	Start Expr
	End   Expr
}

func (RangeExpr) exprNode() {}

// UnionExpr represents the union of two node-sets (expr | expr or expr union expr).
type UnionExpr struct {
	Left  Expr
	Right Expr
}

func (UnionExpr) exprNode() {}

// IntersectExceptExpr represents intersect/except operations.
type IntersectExceptExpr struct {
	Op    TokenType // TokenIntersect or TokenExcept
	Left  Expr
	Right Expr
}

func (IntersectExceptExpr) exprNode() {}

// --- Filter & Path ---

// FilterExpr represents a primary expression followed by predicates.
type FilterExpr struct {
	Expr       Expr
	Predicates []Expr
}

func (FilterExpr) exprNode() {}

// PathExpr represents a filter expression followed by steps.
type PathExpr struct {
	Filter Expr
	Path   *LocationPath
}

func (PathExpr) exprNode() {}

// PathStepExpr represents E1/E2 or E1//E2 where E2 is a non-axis expression
// (function call, variable ref, etc.) used as a step in a path expression.
// Per XPath 3.1, StepExpr := PostfixExpr | AxisStep.
// Evaluation: for each node in E1, evaluate E2 with that node as context.
type PathStepExpr struct {
	Left       Expr
	Right      Expr
	DescOrSelf bool // true for //, inserts descendant-or-self::node() before Right
}

func (PathStepExpr) exprNode() {}

// --- Lookup ---

// LookupExpr represents postfix lookup: expr ? key.
type LookupExpr struct {
	Expr Expr
	Key  Expr // nil when All is true
	All  bool // ? * (all entries)
}

func (LookupExpr) exprNode() {}

// UnaryLookupExpr represents unary lookup: ? key (uses context item).
type UnaryLookupExpr struct {
	Key Expr // nil when All is true
	All bool // ? * (all entries)
}

func (UnaryLookupExpr) exprNode() {}

// --- FLWOR ---

// FLWORExpr represents a for/let/where/order-by/return expression.
type FLWORExpr struct {
	Clauses []FLWORClause
	Return  Expr
}

func (FLWORExpr) exprNode() {}

// FLWORClause is implemented by ForClause, LetClause, WhereClause, OrderByClause.
type FLWORClause interface {
	flworClause()
}

// ForClause represents "for $var in expr" or "for $var at $pos in expr".
type ForClause struct {
	Var    string
	PosVar string // positional variable from "at $pos" (empty if none)
	Expr   Expr
}

func (ForClause) flworClause() {}

// LetClause represents "let $var := expr".
type LetClause struct {
	Var  string
	Expr Expr
}

func (LetClause) flworClause() {}

// WhereClause represents "where predicate".
type WhereClause struct {
	Predicate Expr
}

func (WhereClause) flworClause() {}

// OrderByClause represents "order by spec1, spec2, ...".
type OrderByClause struct {
	Specs  []OrderSpec
	Stable bool
}

func (OrderByClause) flworClause() {}

// OrderSpec is one ordering specification within an order-by clause.
type OrderSpec struct {
	Expr          Expr
	Descending    bool
	EmptyGreatest bool
	Collation     string
}

// --- Control Flow ---

// QuantifiedExpr represents "some/every $var in domain satisfies test".
// QuantifiedBinding is a single "$var in expr" binding in a quantified expression.
type QuantifiedBinding struct {
	Var    string
	Domain Expr
}

type QuantifiedExpr struct {
	Some      bool // true = some, false = every
	Bindings  []QuantifiedBinding
	Satisfies Expr
}

func (QuantifiedExpr) exprNode() {}

// IfExpr represents "if (cond) then thenExpr else elseExpr".
type IfExpr struct {
	Cond Expr
	Then Expr
	Else Expr
}

func (IfExpr) exprNode() {}

// TryCatchExpr represents "try { expr } catch code { expr }".
type TryCatchExpr struct {
	Try     Expr
	Catches []CatchClause
}

func (TryCatchExpr) exprNode() {}

// CatchClause represents a single catch branch.
type CatchClause struct {
	Codes []string // error codes to match; empty = catch all (*)
	Expr  Expr
}

// --- Type Expressions ---

// InstanceOfExpr represents "expr instance of sequenceType".
type InstanceOfExpr struct {
	Expr Expr
	Type SequenceType
}

func (InstanceOfExpr) exprNode() {}

// CastExpr represents "expr cast as atomicType?".
type CastExpr struct {
	Expr       Expr
	Type       AtomicTypeName
	AllowEmpty bool
}

func (CastExpr) exprNode() {}

// CastableExpr represents "expr castable as atomicType?".
type CastableExpr struct {
	Expr       Expr
	Type       AtomicTypeName
	AllowEmpty bool
}

func (CastableExpr) exprNode() {}

// TreatAsExpr represents "expr treat as sequenceType".
type TreatAsExpr struct {
	Expr Expr
	Type SequenceType
}

func (TreatAsExpr) exprNode() {}

// AtomicTypeName identifies a target type for cast/castable.
type AtomicTypeName struct {
	Prefix string
	Name   string
}

// --- Functions ---

// FunctionCall represents a static function invocation: prefix:name(args...).
type FunctionCall struct {
	Prefix string
	Name   string
	Args   []Expr
}

func (FunctionCall) exprNode() {}

// DynamicFunctionCall represents a dynamic function call: $f(args...).
type DynamicFunctionCall struct {
	Func Expr
	Args []Expr
}

func (DynamicFunctionCall) exprNode() {}

// NamedFunctionRef represents a named function reference: fn:name#arity.
type NamedFunctionRef struct {
	Prefix string
	Name   string
	Arity  int
}

func (NamedFunctionRef) exprNode() {}

// InlineFunctionExpr represents an inline function definition.
type InlineFunctionExpr struct {
	Params     []FunctionParam
	ReturnType *SequenceType // nil if not specified
	Body       Expr
}

func (InlineFunctionExpr) exprNode() {}

// PlaceholderExpr represents ? in partial function application.
type PlaceholderExpr struct{}

func (PlaceholderExpr) exprNode() {}

// FunctionParam is a parameter in an inline function definition.
type FunctionParam struct {
	Name     string
	TypeHint *SequenceType // nil if not specified
}

// --- Constructors ---

// MapConstructorExpr represents map { key: value, ... }.
type MapConstructorExpr struct {
	Pairs []MapConstructorPair
}

func (MapConstructorExpr) exprNode() {}

// MapConstructorPair is one key:value entry in a map constructor.
type MapConstructorPair struct {
	Key   Expr
	Value Expr
}

// ArrayConstructorExpr represents [items] or array { items }.
type ArrayConstructorExpr struct {
	Items         []Expr
	SquareBracket bool // true = [a, b], false = array { expr }
}

func (ArrayConstructorExpr) exprNode() {}

// SequenceExpr represents a comma-separated sequence: (a, b, c).
type SequenceExpr struct {
	Items []Expr
}

func (SequenceExpr) exprNode() {}
