package xpath

// Expr is the interface implemented by all XPath AST nodes.
type Expr interface {
	exprNode()
}

// AxisType identifies one of the 13 XPath axes.
type AxisType int

// AxisChild and the other AxisType constants identify the thirteen XPath axes.
const (
	AxisChild AxisType = iota
	AxisDescendant
	AxisParent
	AxisAncestor
	AxisFollowingSibling
	AxisPrecedingSibling
	AxisFollowing
	AxisPreceding
	AxisAttribute
	AxisNamespace
	AxisSelf
	AxisDescendantOrSelf
	AxisAncestorOrSelf
)

var axisNames = map[AxisType]string{
	AxisChild:            "child",
	AxisDescendant:       "descendant",
	AxisParent:           "parent",
	AxisAncestor:         "ancestor",
	AxisFollowingSibling: "following-sibling",
	AxisPrecedingSibling: "preceding-sibling",
	AxisFollowing:        "following",
	AxisPreceding:        "preceding",
	AxisAttribute:        "attribute",
	AxisNamespace:        "namespace",
	AxisSelf:             "self",
	AxisDescendantOrSelf: "descendant-or-self",
	AxisAncestorOrSelf:   "ancestor-or-self",
}

func (a AxisType) String() string {
	if s, ok := axisNames[a]; ok {
		return s
	}
	return "unknown-axis"
}

// axisFromName maps an axis name string to its AxisType.
// Returns the axis and true if recognized, or AxisChild and false otherwise.
func axisFromName(name string) (AxisType, bool) {
	for k, v := range axisNames {
		if v == name {
			return k, true
		}
	}
	return AxisChild, false
}

// NodeTest filters nodes selected by an axis.
type NodeTest interface {
	nodeTest()
}

// NodeTestType identifies built-in node test functions.
type NodeTestType int

// NodeTestNode and the other NodeTestType constants identify built-in XPath node test functions.
const (
	NodeTestNode NodeTestType = iota // node()
	NodeTestText                     // text()
	NodeTestComment                  // comment()
	NodeTestPI                       // processing-instruction()
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
type FunctionCall struct {
	Name string
	Args []Expr
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
