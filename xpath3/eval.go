package xpath3

import (
	"context"
	"fmt"
	"time"

	helium "github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

const (
	maxRecursionDepth = ixpath.DefaultMaxRecursionDepth
	maxNodeSetLength  = ixpath.DefaultMaxNodeSetLength
)

// evalContext holds the evaluation state for an XPath 3.1 expression.
type evalContext struct {
	goCtx       context.Context
	node        helium.Node
	contextItem Item // non-nil when context item is not a node (simple map over atomics)
	position    int
	size        int
	vars        map[string]Sequence
	namespaces  map[string]string
	functions   map[string]Function
	fnsNS       map[QualifiedName]Function
	depth       int
	opCount     *int
	opLimit     int
	docOrder    *ixpath.DocOrderCache
	maxNodes    int
	currentTime *time.Time // cached current time for stable fn:current-*
}

func newEvalContext(ctx context.Context, node helium.Node) *evalContext {
	opCount := 0
	now := time.Now()
	ec := &evalContext{
		goCtx:       ctx,
		node:        node,
		position:    1,
		size:        1,
		opCount:     &opCount,
		docOrder:    &ixpath.DocOrderCache{},
		maxNodes:    maxNodeSetLength,
		currentTime: &now,
	}
	if xctx := GetContext(ctx); xctx != nil {
		ec.namespaces = xctx.namespaces
		ec.vars = xctx.variables
		ec.opLimit = xctx.opLimit
		ec.functions = xctx.functions
		ec.fnsNS = xctx.functionsNS
	}
	return ec
}

func (ec *evalContext) withNode(n helium.Node, pos, size int) *evalContext {
	return &evalContext{
		goCtx:       ec.goCtx,
		node:        n,
		position:    pos,
		size:        size,
		vars:        ec.vars,
		namespaces:  ec.namespaces,
		functions:   ec.functions,
		fnsNS:       ec.fnsNS,
		depth:       ec.depth,
		opCount:     ec.opCount,
		opLimit:     ec.opLimit,
		docOrder:    ec.docOrder,
		maxNodes:    ec.maxNodes,
		currentTime: ec.currentTime,
	}
}

// withContextItem sets a non-node context item (for simple map, etc.)
func (ec *evalContext) withContextItem(item Item, pos, size int) *evalContext {
	return &evalContext{
		goCtx:       ec.goCtx,
		node:        ec.node,
		contextItem: item,
		position:    pos,
		size:        size,
		vars:        ec.vars,
		namespaces:  ec.namespaces,
		functions:   ec.functions,
		fnsNS:       ec.fnsNS,
		depth:       ec.depth,
		opCount:     ec.opCount,
		opLimit:     ec.opLimit,
		docOrder:    ec.docOrder,
		maxNodes:    ec.maxNodes,
		currentTime: ec.currentTime,
	}
}

// getCurrentTime returns the cached current time for this evaluation,
// lazily initializing it on first access.
func (ec *evalContext) getCurrentTime() time.Time {
	if ec.currentTime == nil {
		now := time.Now()
		ec.currentTime = &now
	}
	return *ec.currentTime
}

func (ec *evalContext) withVar(name string, val Sequence) *evalContext {
	newVars := make(map[string]Sequence, len(ec.vars)+1)
	for k, v := range ec.vars {
		newVars[k] = v
	}
	newVars[name] = val
	cp := *ec
	cp.vars = newVars
	return &cp
}

func (ec *evalContext) countOps(n int) error {
	if ec.opLimit <= 0 {
		return nil
	}
	*ec.opCount += n
	if *ec.opCount > ec.opLimit {
		return ErrOpLimit
	}
	return nil
}

// eval dispatches to the appropriate evaluator for each AST node type.
func eval(ec *evalContext, expr Expr) (Sequence, error) {
	ec.depth++
	if ec.depth > maxRecursionDepth {
		return nil, ErrRecursionLimit
	}
	defer func() { ec.depth-- }()

	switch e := expr.(type) {
	case LiteralExpr:
		return evalLiteral(e)
	case VariableExpr:
		return evalVariable(ec, e)
	case ContextItemExpr:
		if ec.contextItem != nil {
			return Sequence{ec.contextItem}, nil
		}
		if ec.node == nil {
			return nil, &XPathError{Code: "XPDY0002", Message: "context item is absent"}
		}
		return Sequence{NodeItem{Node: ec.node}}, nil
	case RootExpr:
		if ec.node == nil {
			return nil, &XPathError{Code: "XPDY0002", Message: "context item is absent"}
		}
		return Sequence{NodeItem{Node: ixpath.DocumentRoot(ec.node)}}, nil
	case SequenceExpr:
		return evalSequenceExpr(ec, e)
	case *LocationPath:
		return evalLocationPath(ec, e)
	case BinaryExpr:
		return evalBinaryExpr(ec, e)
	case UnaryExpr:
		return evalUnaryExpr(ec, e)
	case ConcatExpr:
		return evalConcatExpr(ec, e)
	case SimpleMapExpr:
		return evalSimpleMapExpr(ec, e)
	case RangeExpr:
		return evalRangeExpr(ec, e)
	case UnionExpr:
		return evalUnionExpr(ec, e)
	case IntersectExceptExpr:
		return evalIntersectExceptExpr(ec, e)
	case FilterExpr:
		return evalFilterExpr(ec, e)
	case PathExpr:
		return evalPathExpr(ec, e)
	case PathStepExpr:
		return evalPathStepExpr(ec, e)
	case LookupExpr:
		return evalLookupExpr(ec, e)
	case UnaryLookupExpr:
		return evalUnaryLookupExpr(ec, e)
	case FLWORExpr:
		return evalFLWOR(ec, e)
	case QuantifiedExpr:
		return evalQuantifiedExpr(ec, e)
	case IfExpr:
		return evalIfExpr(ec, e)
	case TryCatchExpr:
		return evalTryCatchExpr(ec, e)
	case InstanceOfExpr:
		return evalInstanceOfExpr(ec, e)
	case CastExpr:
		return evalCastExpr(ec, e)
	case CastableExpr:
		return evalCastableExpr(ec, e)
	case TreatAsExpr:
		return evalTreatAsExpr(ec, e)
	case FunctionCall:
		return evalFunctionCall(ec, e)
	case DynamicFunctionCall:
		return evalDynamicFunctionCall(ec, e)
	case NamedFunctionRef:
		return evalNamedFunctionRef(ec, e)
	case InlineFunctionExpr:
		return evalInlineFunctionExpr(ec, e)
	case MapConstructorExpr:
		return evalMapConstructorExpr(ec, e)
	case ArrayConstructorExpr:
		return evalArrayConstructorExpr(ec, e)
	default:
		return nil, fmt.Errorf("%w: %T", ErrUnsupportedExpr, expr)
	}
}
