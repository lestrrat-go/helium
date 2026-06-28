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

	// baseNode / baseContextItem capture the focus seeded by NewEvalState so
	// each EvaluateReuse call starts from that base focus rather than a focus
	// mutated by a prior call.
	baseNode        helium.Node
	baseContextItem Item

	// explicitTime is true when the Evaluator had an explicitly configured
	// CurrentTime. When false, EvaluateReuse refreshes the clock per call so
	// fn:current-dateTime() does not stay frozen across reuse calls.
	explicitTime bool
}

// SetContextItem sets the non-node context item on the eval state.
// This is used when the context is an atomic value rather than a node
// (e.g., sorting a sequence of atomic items).
//
// The item becomes the base focus, so it persists as the starting focus
// for subsequent EvaluateReuse calls (a reuse call with a non-nil node
// still overrides it for that call only).
func (s *EvalState) SetContextItem(item Item) {
	s.ec.contextItem = item
	s.baseContextItem = item
	if item != nil {
		s.ec.node = nil
		s.baseNode = nil
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
func (e *Expression) EvaluateReuse(ctx context.Context, state *EvalState, node helium.Node) (Result, error) {
	if err := e.requireCompiledProgram(); err != nil {
		return Result{}, err
	}

	ec := &state.ec

	// Start from the base focus seeded by NewEvalState/SetContextItem so a
	// prior reuse call cannot leak its focus into this one. A non-nil node
	// overrides the base focus for this call only.
	if node != nil {
		ec.node = node
		ec.contextItem = nil
	} else {
		ec.node = state.baseNode
		ec.contextItem = state.baseContextItem
	}
	ec.depth = 0
	*ec.opCount = 0

	// When CurrentTime was not explicitly configured on the Evaluator, refresh
	// the clock per call so fn:current-dateTime() / fn:current-date() track
	// wall-clock time instead of staying frozen at NewEvalState construction.
	// A user-pinned CurrentTime is preserved untouched.
	if !state.explicitTime {
		now := time.Now()
		ec.currentTime = &now
	}

	// Fast path for "." — skip eval entirely, reuse backing array
	if e.program != nil && e.program.stream.isContextItem {
		if ec.contextItem != nil {
			state.oneItem[0] = ec.contextItem
			return Result{seq: ItemSlice(state.oneItem[:])}, nil
		}
		if ec.node == nil {
			return Result{}, &XPathError{Code: errCodeXPDY0002, Message: errMsgContextItemAbsent}
		}
		state.oneItem[0] = nodeItemFor(ctx, ec, ec.node)
		return Result{seq: ItemSlice(state.oneItem[:])}, nil
	}

	if err := e.prefixPlan.Validate(ec.namespaces, ec.strictPrefixes, ec.schemaDeclarations); err != nil {
		return Result{}, err
	}
	seq, err := e.evaluate(ctx, ec)
	if err != nil {
		return Result{}, err
	}
	return Result{seq: seq}, nil
}

// StringValue returns the XPath string value of the result sequence.
// For single-node results, this returns the node's string value directly,
// avoiding the AtomizeItem → AtomicValue → AtomicToString round-trip.
func (r Result) StringValue() string {
	if seqLen(r.seq) == 0 {
		return ""
	}
	if seqLen(r.seq) == 1 {
		switch v := r.seq.Get(0).(type) {
		case NodeItem:
			return ixpath.StringValue(v.Node)
		case AtomicValue:
			s, _ := AtomicToString(v)
			return s
		}
	}
	var sb strings.Builder
	i := 0
	for item := range seqItems(r.seq) {
		if i > 0 {
			sb.WriteByte(' ')
		}
		av, err := AtomizeItem(item)
		if err != nil {
			i++
			continue
		}
		s, err := AtomicToString(av)
		if err != nil {
			i++
			continue
		}
		sb.WriteString(s)
		i++
	}
	return sb.String()
}

// NewEvalState creates a reusable EvalState from this Evaluator's config.
// node sets the initial context node (may be nil).
//
// The returned EvalState is not safe for concurrent use. The Result from
// EvaluateReuse is only valid until the next EvaluateReuse call.
func (e Evaluator) NewEvalState(node helium.Node) *EvalState {
	// Build the evaluation context through the shared initialization path so
	// the reuse state stays in lockstep with normal Evaluate: same nil-cfg
	// guard, same custom max-node limit, and the same complete set of copied
	// fields (preserved ID annotations, TraceWriter, etc.).
	s := &EvalState{ec: *e.newEvalCtx(node)}

	// Capture the seeded base focus so each EvaluateReuse call can reset to it.
	// newEvalCtx clears ec.node when a non-node context item is configured, so
	// read both back from the freshly built context.
	s.baseNode = s.ec.node
	s.baseContextItem = s.ec.contextItem

	// Record whether CurrentTime was explicitly configured. If so, the reuse
	// path must keep it pinned; otherwise the reuse path refreshes per call.
	s.explicitTime = e.cfg != nil && e.cfg.currentTime != nil

	return s
}
