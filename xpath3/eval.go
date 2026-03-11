package xpath3

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

const (
	maxRecursionDepth = ixpath.DefaultMaxRecursionDepth
	maxNodeSetLength  = ixpath.DefaultMaxNodeSetLength
)

// evalContext holds the evaluation state for an XPath 3.1 expression.
type evalContext struct {
	goCtx            context.Context
	node             helium.Node
	contextItem      Item // non-nil when context item is not a node (simple map over atomics)
	position         int
	size             int
	vars             map[string]Sequence
	namespaces       map[string]string
	functions        map[string]Function
	fnsNS            map[QualifiedName]Function
	depth            int
	opCount          *int // shared via pointer across copies; safe because eval is single-goroutine
	opLimit          int
	docOrder         *ixpath.DocOrderCache
	maxNodes         int
	currentTime      *time.Time     // set once at construction for stable fn:current-*
	implicitTimezone *time.Location // for fn:adjust-*-to-timezone (1-arg form)
	baseURI          string         // static base URI for resolving relative URIs
	uriResolver      URIResolver    // custom URI resolver for fn:unparsed-text, fn:doc, etc.
	// httpClient is intentionally stored here (not only in Context) so that
	// built-in functions can access it through getFnContext without an extra
	// indirection. It is pointer-sized and nil when unused, so copies via
	// withNode/withContextItem are negligible. The net/http dependency is
	// already transitively required by golang.org/x/text.
	httpClient *http.Client
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
		ec.implicitTimezone = xctx.implicitTimezone
		ec.baseURI = xctx.baseURI
		ec.uriResolver = xctx.uriResolver
		ec.httpClient = xctx.httpClient
	}
	return ec
}

func (ec *evalContext) withNode(n helium.Node, pos, size int) *evalContext {
	cp := *ec
	cp.node = n
	cp.contextItem = nil
	cp.position = pos
	cp.size = size
	return &cp
}

// withContextItem sets a non-node context item (for simple map, etc.)
func (ec *evalContext) withContextItem(item Item, pos, size int) *evalContext {
	cp := *ec
	cp.contextItem = item
	cp.node = nil // clear node so functions don't accidentally use an outer node
	cp.position = pos
	cp.size = size
	return &cp
}

// contextStringValue returns the string value of the current context item.
// For node context items, it returns the XPath string value of the node.
// For atomic context items, it returns the string representation.
// Returns ("", false) if no context item is set.
func (ec *evalContext) contextStringValue() (string, bool) {
	if ec.contextItem != nil {
		if av, ok := ec.contextItem.(AtomicValue); ok {
			s, _ := atomicToString(av)
			return s, true
		}
		if ni, ok := ec.contextItem.(NodeItem); ok {
			return ixpath.StringValue(ni.Node), true
		}
		return "", false
	}
	if ec.node != nil {
		return ixpath.StringValue(ec.node), true
	}
	return "", false
}

// getCurrentTime returns the cached current time for this evaluation.
func (ec *evalContext) getCurrentTime() time.Time {
	return *ec.currentTime
}

// getImplicitTimezone returns the implicit timezone for the dynamic context.
// If not explicitly set, it falls back to the system local timezone.
func (ec *evalContext) getImplicitTimezone() *time.Location {
	if ec.implicitTimezone != nil {
		return ec.implicitTimezone
	}
	return time.Local
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

// withVars returns a shallow copy using the given vars map.
func (ec *evalContext) withVars(vars map[string]Sequence) *evalContext {
	cp := *ec
	cp.vars = vars
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
// Depth tracking: withNode/withContextItem copy the parent's depth into the
// new context, so each nested eval chain inherits and increments correctly.
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
			return nil, &XPathError{Code: errCodeXPDY0002, Message: "context item is absent"}
		}
		return Sequence{NodeItem{Node: ec.node}}, nil
	case RootExpr:
		if ec.node == nil {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: "context item is absent"}
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
