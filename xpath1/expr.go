package xpath1

import (
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// Expr is the interface implemented by all XPath AST nodes.
type Expr interface {
	exprNode()
}

// AxisType identifies one of the 13 XPath axes.
// This is a type alias (not a defined type), so methods on ixpath.AxisType
// (including String()) are inherited and available on xpath1.AxisType.
type AxisType = ixpath.AxisType

// AxisChild and the other AxisType constants identify the thirteen XPath axes.
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

// NodeTestType identifies built-in node test functions.
type NodeTestType int

// NodeTestNode and the other NodeTestType constants identify built-in XPath node test functions.
const (
	NodeTestNode                  NodeTestType = iota // node()
	NodeTestText                                      // text()
	NodeTestComment                                   // comment()
	NodeTestProcessingInstruction                     // processing-instruction()
)

// NameTest matches by qualified name. Local=="*" is a wildcard.
// Prefix may be empty (no namespace qualifier).
type NameTest struct {
	Prefix string
	Local  string
}

func (NameTest) nodeTest() {}

// TypeTest matches by node type (node(), text(), comment()).
type TypeTest struct {
	Type NodeTestType
}

func (TypeTest) nodeTest() {}

// PITest matches processing-instruction nodes, optionally by target name.
type PITest struct {
	Target string // empty means match any PI
}

func (PITest) nodeTest() {}

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

// BinaryExpr represents a binary operation (and, or, =, !=, <, >, +, -, *, div, mod).
type BinaryExpr struct {
	Op    TokenType
	Left  Expr
	Right Expr
}

func (BinaryExpr) exprNode() {}

// UnaryExpr represents unary negation (-expr).
type UnaryExpr struct {
	Operand Expr
}

func (UnaryExpr) exprNode() {}

// LiteralExpr represents a string literal.
type LiteralExpr struct {
	Value string
}

func (LiteralExpr) exprNode() {}

// NumberExpr represents a numeric literal.
type NumberExpr struct {
	Value float64
}

func (NumberExpr) exprNode() {}

// VariableExpr represents a variable reference ($name).
type VariableExpr struct {
	Name string
}

func (VariableExpr) exprNode() {}

// FunctionCall represents a function invocation.
// Prefix is set for namespace-qualified calls (e.g., ext:func()).
type FunctionCall struct {
	Prefix string // optional namespace prefix
	Name   string // local name
	Args   []Expr
}

func (FunctionCall) exprNode() {}

// FilterExpr represents a primary expression followed by predicates.
type FilterExpr struct {
	Expr       Expr
	Predicates []Expr
}

func (FilterExpr) exprNode() {}

// UnionExpr represents the union of two node-sets (expr | expr).
type UnionExpr struct {
	Left  Expr
	Right Expr
}

func (UnionExpr) exprNode() {}

// PathExpr represents a filter expression followed by a relative location path.
type PathExpr struct {
	Filter Expr
	Path   *LocationPath
}

func (PathExpr) exprNode() {}
