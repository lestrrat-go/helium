package xpath3

import (
	"context"
	"strings"
	"time"

	"github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// EvalState holds pre-allocated evaluation state that can be reused
// across multiple EvaluateReuse calls on the same Expression.
// Not safe for concurrent use.
//
// The returned Result from EvaluateReuse is only valid until the next
// EvaluateReuse call on the same EvalState — the result's backing
// storage may be overwritten.
type EvalState struct {
	ec      evalContext
	oneItem [1]Item // reusable backing for single-item results
}

// NewEvalState creates a reusable evaluation state from the given context.
// The context.Context should carry the same XPath evaluation settings
// (variables, functions, namespaces) that would be passed to Evaluate.
// The node parameter sets the initial context node (may be nil).
func NewEvalState(ctx context.Context, node helium.Node) *EvalState {
	opCount := 0
	now := time.Now()
	s := &EvalState{}
	ec := &s.ec
	ec.goCtx = ctx
	ec.node = node
	ec.position = 1
	ec.size = 1
	ec.opCount = &opCount
	ec.docOrder = &ixpath.DocOrderCache{}
	ec.maxNodes = maxNodeSetLength
	ec.currentTime = &now
	ec.docCache = make(map[string]helium.Node)

	if cfg := getEvalConfig(ctx); cfg != nil {
		ec.namespaces = cfg.namespaces
		ec.vars = cfg.varScope
		ec.opLimit = cfg.opLimit
		if cfg.currentTime != nil {
			ec.currentTime = cfg.currentTime
		}
		ec.functions = cfg.functions
		ec.fnsNS = cfg.functionsNS
		ec.implicitTimezone = cfg.implicitTimezone
		ec.defaultLanguage = cfg.defaultLanguage
		ec.defaultCollation = cfg.defaultCollation
		if cfg.defaultDecimal != nil {
			df := *cfg.defaultDecimal
			ec.defaultDecimalFormat = &df
		}
		ec.decimalFormats = cfg.decimalFormats
		ec.baseURI = cfg.baseURI
		ec.uriResolver = cfg.uriResolver
		ec.collectionResolver = cfg.collectionResolver
		ec.httpClient = cfg.httpClient
		ec.typeAnnotations = cfg.typeAnnotations
		ec.variableResolver = cfg.variableResolver
		ec.strictPrefixes = cfg.strictPrefixes
		ec.schemaDeclarations = cfg.schemaDeclarations
		if cfg.position > 0 {
			ec.position = cfg.position
		}
		if cfg.size > 0 {
			ec.size = cfg.size
		}
		if cfg.contextItem != nil {
			ec.contextItem = cfg.contextItem
			ec.node = nil
		}
	}
	return s
}

// SetContextItem sets the non-node context item on the eval state.
// This is used when the context is an atomic value rather than a node
// (e.g., sorting a sequence of atomic items).
func (s *EvalState) SetContextItem(item Item) {
	s.ec.contextItem = item
	if item != nil {
		s.ec.node = nil
	}
}

// SetPosition sets the context position on the eval state.
func (s *EvalState) SetPosition(pos int) { s.ec.position = pos }

// SetSize sets the context size on the eval state.
func (s *EvalState) SetSize(size int) { s.ec.size = size }

// EvaluateReuse evaluates the expression using pre-allocated state,
// resetting per-evaluation fields rather than allocating new.
// The node parameter replaces the context node for this evaluation.
//
// The returned Result is only valid until the next EvaluateReuse call
// on the same EvalState. Callers must extract all needed values from
// the Result before calling EvaluateReuse again.
func (e *Expression) EvaluateReuse(state *EvalState, node helium.Node) (Result, error) {
	ec := &state.ec
	ec.node = node
	if node != nil {
		ec.contextItem = nil
	}
	ec.depth = 0
	*ec.opCount = 0

	// Fast path for "." — skip eval entirely, reuse backing array
	if e.program != nil && e.program.stream.isContextItem {
		if ec.contextItem != nil {
			state.oneItem[0] = ec.contextItem
			return Result{seq: state.oneItem[:]}, nil
		}
		if ec.node == nil {
			return Result{}, &XPathError{Code: errCodeXPDY0002, Message: "context item is absent"}
		}
		state.oneItem[0] = nodeItemFor(ec, ec.node)
		return Result{seq: state.oneItem[:]}, nil
	}

	if err := e.prefixPlan.Validate(ec.namespaces, ec.strictPrefixes, ec.schemaDeclarations); err != nil {
		return Result{}, err
	}
	seq, err := e.evaluate(ec)
	if err != nil {
		return Result{}, err
	}
	return Result{seq: seq}, nil
}

// StringValue returns the XPath string value of the result sequence.
// For single-node results, this returns the node's string value directly,
// avoiding the AtomizeItem → AtomicValue → AtomicToString round-trip.
func (r Result) StringValue() string {
	if len(r.seq) == 0 {
		return ""
	}
	if len(r.seq) == 1 {
		switch v := r.seq[0].(type) {
		case NodeItem:
			return ixpath.StringValue(v.Node)
		case AtomicValue:
			s, _ := AtomicToString(v)
			return s
		}
	}
	var sb strings.Builder
	for i, item := range r.seq {
		if i > 0 {
			sb.WriteByte(' ')
		}
		av, err := AtomizeItem(item)
		if err != nil {
			continue
		}
		s, err := AtomicToString(av)
		if err != nil {
			continue
		}
		sb.WriteString(s)
	}
	return sb.String()
}
