package xpath3

import (
	"context"
	"time"

	"github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// NewEvalState creates a reusable EvalState from this Evaluator's config.
// ctx carries cancellation/deadlines and optional xslt3 exec context.
// node sets the initial context node (may be nil).
//
// The returned EvalState is not safe for concurrent use. The Result from
// EvaluateReuse is only valid until the next EvaluateReuse call.
func (e Evaluator) NewEvalState(ctx context.Context, node helium.Node) *EvalState {
	opCount := 0
	now := time.Now()
	s := &EvalState{}
	ec := &s.ec
	cfg := e.cfg

	ec.goCtx = ctx
	ec.node = node
	ec.position = 1
	ec.size = 1
	ec.opCount = &opCount
	ec.docOrder = &ixpath.DocOrderCache{}
	ec.maxNodes = maxNodeSetLength
	ec.docCache = make(map[string]helium.Node)

	// namespaces
	ec.namespaces = cfg.namespaces

	// variables
	if cfg.variables != nil && cfg.variables.Len() > 0 {
		ec.vars = newVariableScope(cfg.variables.toMap())
	}

	// functions
	if cfg.functions != nil {
		ec.functions = cfg.functions.localMap()
		ec.fnsNS = cfg.functions.qnameMap()
	}

	// resolvers
	ec.variableResolver = cfg.variableResolver
	ec.functionResolver = cfg.functionResolver
	ec.uriResolver = cfg.uriResolver
	ec.collectionResolver = cfg.collectionResolver
	ec.httpClient = cfg.httpClient

	// limits
	ec.opLimit = cfg.opLimit

	// time
	if cfg.currentTime != nil {
		ec.currentTime = cfg.currentTime
	} else {
		ec.currentTime = &now
	}
	ec.implicitTimezone = cfg.implicitTimezone

	// locale / formatting
	ec.defaultLanguage = cfg.defaultLanguage
	ec.defaultCollation = cfg.defaultCollation
	if cfg.defaultDecimal != nil {
		df := *cfg.defaultDecimal
		ec.defaultDecimalFormat = &df
	}
	ec.decimalFormats = cfg.decimalFormats

	// base URI
	ec.baseURI = cfg.baseURI

	// schema / typing
	ec.typeAnnotations = cfg.typeAnnotations
	ec.schemaDeclarations = cfg.schemaDeclarations
	ec.strictPrefixes = cfg.strictPrefixes
	ec.allowXML11Chars = cfg.allowXML11Chars

	// dynamic focus overrides
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
	if cfg.docOrder != nil {
		ec.docOrder = cfg.docOrder
	}

	return s
}
