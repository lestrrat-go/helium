package xpath3

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

func (op vmOpcode) String() string {
	switch op {
	case vmOpLiteral:
		return "literal"
	case vmOpVariable:
		return "variable"
	case vmOpRoot:
		return "root"
	case vmOpContextItem:
		return "context-item"
	case vmOpLocationPath:
		return "location-path"
	case vmOpBinary:
		return "binary"
	case vmOpUnary:
		return "unary"
	case vmOpConcat:
		return "concat"
	case vmOpSimpleMap:
		return "simple-map"
	case vmOpRange:
		return "range"
	case vmOpUnion:
		return "union"
	case vmOpIntersectExcept:
		return "intersect-except"
	case vmOpFilter:
		return "filter"
	case vmOpPath:
		return "path"
	case vmOpPathStep:
		return "path-step"
	case vmOpLookup:
		return "lookup"
	case vmOpUnaryLookup:
		return "unary-lookup"
	case vmOpFLWOR:
		return "flwor"
	case vmOpQuantified:
		return "quantified"
	case vmOpIf:
		return "if"
	case vmOpTryCatch:
		return "try-catch"
	case vmOpInstanceOf:
		return "instance-of"
	case vmOpCast:
		return "cast"
	case vmOpCastable:
		return "castable"
	case vmOpTreatAs:
		return "treat-as"
	case vmOpFunctionCall:
		return "function-call"
	case vmOpDynamicFunctionCall:
		return "dynamic-call"
	case vmOpNamedFunctionRef:
		return "named-function-ref"
	case vmOpInlineFunction:
		return "inline-function"
	case vmOpPlaceholder:
		return "placeholder"
	case vmOpMapConstructor:
		return "map-constructor"
	case vmOpArrayConstructor:
		return "array-constructor"
	case vmOpSequence:
		return "sequence"
	default:
		return fmt.Sprintf("vm-opcode(%d)", op)
	}
}

func (p *vmProgram) dumpTo(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "root @%d\n", p.root); err != nil {
		return err
	}
	for i, inst := range p.instructions {
		marker := ' '
		if i == p.root {
			marker = '*'
		}
		if _, err := fmt.Fprintf(w, "%c%04d %-18s %s\n", marker, i, inst.op.String(), formatVMExpr(inst.payload.(Expr))); err != nil { //nolint:forcetypeassert
			return err
		}
	}
	return nil
}

func formatVMExpr(expr Expr) string {
	switch v := expr.(type) {
	case compiledExprRef:
		return "@" + strconv.Itoa(v.index)
	case LiteralExpr:
		return formatLiteral(v)
	case VariableExpr:
		return "$" + formatQName(v.Prefix, v.Name)
	case RootExpr:
		return "/"
	case ContextItemExpr:
		return "."
	case vmLocationPathExpr:
		return formatVMLocationPath(v)
	case BinaryExpr:
		return "binary(" + v.Op.String() + ", " + formatVMExpr(v.Left) + ", " + formatVMExpr(v.Right) + ")"
	case UnaryExpr:
		prefix := "+"
		if v.Negate {
			prefix = "-"
		}
		return prefix + formatVMExpr(v.Operand)
	case ConcatExpr:
		return "concat(" + formatVMExpr(v.Left) + ", " + formatVMExpr(v.Right) + ")"
	case SimpleMapExpr:
		return "simple-map(" + formatVMExpr(v.Left) + " ! " + formatVMExpr(v.Right) + ")"
	case RangeExpr:
		return "range(" + formatVMExpr(v.Start) + " to " + formatVMExpr(v.End) + ")"
	case UnionExpr:
		return "union(" + formatVMExpr(v.Left) + ", " + formatVMExpr(v.Right) + ")"
	case IntersectExceptExpr:
		return "binary(" + v.Op.String() + ", " + formatVMExpr(v.Left) + ", " + formatVMExpr(v.Right) + ")"
	case FilterExpr:
		return "filter(" + formatVMExpr(v.Expr) + formatPredicates(v.Predicates) + ")"
	case vmPathExpr:
		if v.Path == nil {
			return "path(" + formatVMExpr(v.Filter) + ")"
		}
		return "path(" + formatVMExpr(v.Filter) + " / " + formatVMLocationPath(*v.Path) + ")"
	case PathStepExpr:
		sep := "/"
		if v.DescOrSelf {
			sep = "//"
		}
		return "path-step(" + formatVMExpr(v.Left) + " " + sep + " " + formatVMExpr(v.Right) + ")"
	case LookupExpr:
		if v.All {
			return "lookup(" + formatVMExpr(v.Expr) + ", *)"
		}
		return "lookup(" + formatVMExpr(v.Expr) + ", " + formatVMExpr(v.Key) + ")"
	case UnaryLookupExpr:
		if v.All {
			return "lookup(., *)"
		}
		return "lookup(., " + formatVMExpr(v.Key) + ")"
	case FLWORExpr:
		return fmt.Sprintf("flwor(clauses=%d, return=%s)", len(v.Clauses), formatVMExpr(v.Return))
	case QuantifiedExpr:
		kind := "every"
		if v.Some {
			kind = "some"
		}
		return fmt.Sprintf("quantified(%s, bindings=%d, satisfies=%s)", kind, len(v.Bindings), formatVMExpr(v.Satisfies))
	case IfExpr:
		return "if(" + formatVMExpr(v.Cond) + ", " + formatVMExpr(v.Then) + ", " + formatVMExpr(v.Else) + ")"
	case TryCatchExpr:
		return fmt.Sprintf("try(%s, catches=%d)", formatVMExpr(v.Try), len(v.Catches))
	case InstanceOfExpr:
		return "instance-of(" + formatVMExpr(v.Expr) + ", " + formatSequenceType(v.Type) + ")"
	case CastExpr:
		return "cast(" + formatVMExpr(v.Expr) + ", " + formatAtomicTypeName(v.Type, v.AllowEmpty) + ")"
	case CastableExpr:
		return "castable(" + formatVMExpr(v.Expr) + ", " + formatAtomicTypeName(v.Type, v.AllowEmpty) + ")"
	case TreatAsExpr:
		return "treat-as(" + formatVMExpr(v.Expr) + ", " + formatSequenceType(v.Type) + ")"
	case FunctionCall:
		return formatQName(v.Prefix, v.Name) + "(" + formatVMExprList(v.Args) + ")"
	case DynamicFunctionCall:
		return "call(" + formatVMExpr(v.Func) + "(" + formatVMExprList(v.Args) + "))"
	case NamedFunctionRef:
		return formatQName(v.Prefix, v.Name) + "#" + strconv.Itoa(v.Arity)
	case InlineFunctionExpr:
		return fmt.Sprintf("inline-function(params=%d, body=%s)", len(v.Params), formatVMExpr(v.Body))
	case PlaceholderExpr:
		return "?"
	case MapConstructorExpr:
		return fmt.Sprintf("map{pairs=%d}", len(v.Pairs))
	case ArrayConstructorExpr:
		return fmt.Sprintf("array{items=%d,square=%t}", len(v.Items), v.SquareBracket)
	case SequenceExpr:
		return "sequence(" + formatVMExprList(v.Items) + ")"
	case vmPositionPredicateExpr:
		return "position() = " + strconv.Itoa(v.Position)
	case vmAttributeExistsPredicateExpr:
		return "attribute-exists(" + formatNodeTest(v.NodeTest) + ")"
	case vmAttributeEqualsStringPredicateExpr:
		return "attribute-equals(" + formatNodeTest(v.NodeTest) + ", " + strconv.Quote(v.Value) + ")"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func formatLiteral(expr LiteralExpr) string {
	switch v := expr.Value.(type) {
	case string:
		return strconv.Quote(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func formatVMLocationPath(expr vmLocationPathExpr) string {
	if len(expr.Steps) == 0 {
		if expr.Absolute {
			return "/"
		}
		return "."
	}

	parts := make([]string, 0, len(expr.Steps))
	for _, step := range expr.Steps {
		parts = append(parts, formatVMLocationStep(step))
	}

	path := strings.Join(parts, "/")
	if expr.Absolute {
		return "/" + path
	}
	return path
}

func formatVMLocationStep(step vmLocationStep) string {
	return formatAxis(step.Axis) + "::" + formatNodeTest(step.NodeTest) + formatPredicates(step.Predicates)
}

func formatPredicates(predicates []Expr) string {
	if len(predicates) == 0 {
		return ""
	}
	var b strings.Builder
	for _, pred := range predicates {
		b.WriteByte('[')
		b.WriteString(formatVMExpr(pred))
		b.WriteByte(']')
	}
	return b.String()
}

func formatVMExprList(exprs []Expr) string {
	if len(exprs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(exprs))
	for _, expr := range exprs {
		parts = append(parts, formatVMExpr(expr))
	}
	return strings.Join(parts, ", ")
}

func formatNodeTest(test NodeTest) string {
	switch v := test.(type) {
	case nil:
		return "<nil>"
	case NameTest:
		switch {
		case v.URI != "":
			return "Q{" + v.URI + "}" + v.Local
		case v.Prefix != "":
			return v.Prefix + ":" + v.Local
		default:
			return v.Local
		}
	case TypeTest:
		switch v.Kind {
		case NodeKindNode:
			return "node()"
		case NodeKindText:
			return "text()"
		case NodeKindComment:
			return "comment()"
		case NodeKindProcessingInstruction:
			return "processing-instruction()"
		default:
			return fmt.Sprintf("node-kind(%d)", v.Kind)
		}
	case PITest:
		if v.Target == "" {
			return "processing-instruction()"
		}
		return "processing-instruction(" + strconv.Quote(v.Target) + ")"
	case ElementTest:
		if v.Name == "" {
			return "element()"
		}
		return "element(" + v.Name + ")"
	case AttributeTest:
		if v.Name == "" {
			return "attribute()"
		}
		return "attribute(" + v.Name + ")"
	case DocumentTest:
		if v.Inner == nil {
			return "document-node()"
		}
		return "document-node(" + formatNodeTest(v.Inner) + ")"
	case SchemaElementTest:
		return "schema-element(" + v.Name + ")"
	case SchemaAttributeTest:
		return "schema-attribute(" + v.Name + ")"
	case NamespaceNodeTest:
		return "namespace-node()"
	case FunctionTest:
		if v.AnyFunction {
			return "function(*)"
		}
		return "function(...)"
	case MapTest:
		if v.AnyType {
			return "map(*)"
		}
		return "map(...)"
	case ArrayTest:
		if v.AnyType {
			return "array(*)"
		}
		return "array(...)"
	case AnyItemTest:
		return "item()"
	case AtomicOrUnionType:
		return formatQName(v.Prefix, v.Name)
	default:
		return fmt.Sprintf("%T", test)
	}
}

func formatAxis(axis AxisType) string {
	switch axis {
	case AxisChild:
		return "child"
	case AxisDescendant:
		return "descendant"
	case AxisParent:
		return "parent"
	case AxisAncestor:
		return "ancestor"
	case AxisFollowingSibling:
		return "following-sibling"
	case AxisPrecedingSibling:
		return "preceding-sibling"
	case AxisFollowing:
		return "following"
	case AxisPreceding:
		return "preceding"
	case AxisAttribute:
		return "attribute"
	case AxisNamespace:
		return "namespace"
	case AxisSelf:
		return "self"
	case AxisDescendantOrSelf:
		return "descendant-or-self"
	case AxisAncestorOrSelf:
		return "ancestor-or-self"
	default:
		return fmt.Sprintf("axis(%d)", axis)
	}
}

func formatQName(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + ":" + name
}

func formatAtomicTypeName(t AtomicTypeName, allowEmpty bool) string {
	name := formatQName(t.Prefix, t.Name)
	if allowEmpty {
		return name + "?"
	}
	return name
}

func formatSequenceType(t SequenceType) string {
	if t.Void {
		return "empty-sequence()"
	}
	name := formatNodeTest(t.ItemTest)
	switch t.Occurrence {
	case OccurrenceZeroOrOne:
		return name + "?"
	case OccurrenceZeroOrMore:
		return name + "*"
	case OccurrenceOneOrMore:
		return name + "+"
	default:
		return name
	}
}
