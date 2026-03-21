package xpath3

import (
	"fmt"

	"github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

type exprEvaluator func(*evalContext, Expr) (Sequence, error)

func evalWith(evalFn exprEvaluator, ec *evalContext, expr Expr) (Sequence, error) {
	ec.depth++
	if ec.depth > maxRecursionDepth {
		return nil, ErrRecursionLimit
	}
	defer func() { ec.depth-- }()

	return evalFn(ec, expr)
}

func evalContextItemExpr(ec *evalContext) (Sequence, error) {
	if ec.contextItem != nil {
		return Sequence{ec.contextItem}, nil
	}
	if ixpath.IsNilNode(ec.node) {
		return nil, &XPathError{Code: errCodeXPDY0002, Message: "context item is absent"}
	}
	return Sequence{nodeItemFor(ec, ec.node)}, nil
}

func evalRootExpr(ec *evalContext) (Sequence, error) {
	if ixpath.IsNilNode(ec.node) {
		return nil, &XPathError{Code: errCodeXPDY0002, Message: "context item is absent"}
	}
	root := ixpath.DocumentRoot(ec.node)
	// XPDY0050: the root of the context node's tree must be a document node.
	if root.Type() != helium.DocumentNode {
		return nil, &XPathError{Code: errCodeXPDY0050, Message: "root of the tree containing the context node is not a document node"}
	}
	return Sequence{nodeItemFor(ec, root)}, nil
}

func dispatchExpr(evalFn exprEvaluator, ec *evalContext, expr Expr) (Sequence, error) {
	switch e := expr.(type) {
	case compiledExprRef:
		return nil, fmt.Errorf("%w: compiled expression reference outside VM", ErrUnsupportedExpr)
	case LiteralExpr:
		return evalLiteral(e)
	case VariableExpr:
		return evalVariable(ec, e)
	case ContextItemExpr:
		return evalContextItemExpr(ec)
	case RootExpr:
		return evalRootExpr(ec)
	case SequenceExpr:
		return evalSequenceExpr(evalFn, ec, e)
	case *LocationPath:
		return evalLocationPath(evalFn, ec, e)
	case BinaryExpr:
		return evalBinaryExpr(evalFn, ec, e)
	case UnaryExpr:
		return evalUnaryExpr(evalFn, ec, e)
	case ConcatExpr:
		return evalConcatExpr(evalFn, ec, e)
	case SimpleMapExpr:
		return evalSimpleMapExpr(evalFn, ec, e)
	case RangeExpr:
		return evalRangeExpr(evalFn, ec, e)
	case UnionExpr:
		return evalUnionExpr(evalFn, ec, e)
	case IntersectExceptExpr:
		return evalIntersectExceptExpr(evalFn, ec, e)
	case FilterExpr:
		return evalFilterExpr(evalFn, ec, e)
	case PathExpr:
		return evalPathExpr(evalFn, ec, e)
	case PathStepExpr:
		return evalPathStepExpr(evalFn, ec, e)
	case LookupExpr:
		return evalLookupExpr(evalFn, ec, e)
	case UnaryLookupExpr:
		return evalUnaryLookupExpr(evalFn, ec, e)
	case FLWORExpr:
		return evalFLWOR(evalFn, ec, e)
	case QuantifiedExpr:
		return evalQuantifiedExpr(evalFn, ec, e)
	case IfExpr:
		return evalIfExpr(evalFn, ec, e)
	case TryCatchExpr:
		return evalTryCatchExpr(evalFn, ec, e)
	case InstanceOfExpr:
		return evalInstanceOfExpr(evalFn, ec, e)
	case CastExpr:
		return evalCastExpr(evalFn, ec, e)
	case CastableExpr:
		return evalCastableExpr(evalFn, ec, e)
	case TreatAsExpr:
		return evalTreatAsExpr(evalFn, ec, e)
	case FunctionCall:
		return evalFunctionCall(evalFn, ec, e)
	case DynamicFunctionCall:
		return evalDynamicFunctionCall(evalFn, ec, e)
	case NamedFunctionRef:
		return evalNamedFunctionRef(ec, e)
	case InlineFunctionExpr:
		return evalInlineFunctionExpr(evalFn, ec, e)
	case MapConstructorExpr:
		return evalMapConstructorExpr(evalFn, ec, e)
	case ArrayConstructorExpr:
		return evalArrayConstructorExpr(evalFn, ec, e)
	default:
		return nil, fmt.Errorf("%w: %T", ErrUnsupportedExpr, expr)
	}
}
