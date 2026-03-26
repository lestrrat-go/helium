package xpath3

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

const (
	maxRecursionDepth = ixpath.DefaultMaxRecursionDepth
	maxNodeSetLength  = ixpath.DefaultMaxNodeSetLength
)

// evalContext holds the evaluation state for an XPath 3.1 expression.
type evalContext struct {
	goCtx                context.Context
	node                 helium.Node
	contextItem          Item // non-nil when context item is not a node (simple map over atomics)
	position             int
	size                 int
	vars                 *variableScope
	namespaces           map[string]string
	functions            map[string]Function
	fnsNS                map[QualifiedName]Function
	depth                int
	opCount              *int // shared via pointer across copies; safe because eval is single-goroutine
	opLimit              int
	docOrder             *ixpath.DocOrderCache
	maxNodes             int
	currentTime          *time.Time     // set once at construction for stable fn:current-*
	implicitTimezone     *time.Location // for fn:adjust-*-to-timezone (1-arg form)
	defaultLanguage      string         // for fn:default-language and formatting fallbacks
	defaultCollation     string         // for string functions without explicit collation
	defaultDecimalFormat *DecimalFormat
	decimalFormats       map[QualifiedName]DecimalFormat
	docCache             map[string]helium.Node
	baseURI              string      // static base URI for resolving relative URIs
	uriResolver          URIResolver // custom URI resolver for fn:unparsed-text, fn:doc, etc.
	collectionResolver   CollectionResolver
	// httpClient is intentionally stored here (not only in Context) so that
	// built-in functions can access it through getFnContext without an extra
	// indirection. It is pointer-sized and nil when unused, so copies via
	// withNode/withContextItem are negligible. The net/http dependency is
	// already transitively required by golang.org/x/text.
	httpClient         *http.Client
	typeAnnotations        map[helium.Node]string // node → xs:... type (from xslt3 schema awareness)
	preservedIDAnnotations map[helium.Node]string // ID/IDREF annotations preserved after input-type-annotations="strip"
	variableResolver       VariableResolver       // lazy resolver for variables not in static scope
	functionResolver   FunctionResolver       // lazy resolver for functions (not visible to function-lookup)
	strictPrefixes     bool                   // skip defaultPrefixNS fallback in prefix validation
	schemaDeclarations SchemaDeclarations     // schema element/attribute declarations for schema-element()/schema-attribute() tests
	allowXML11Chars    bool                   // when true, codepoints-to-string allows XML 1.1 restricted characters (0x01-0x1F)
	traceWriter        io.Writer              // destination for fn:trace output (nil = os.Stderr)
}

type variableScope struct {
	parent      *variableScope
	name        string
	value       Sequence
	singleValue bool
	values      map[string]Sequence
}

func newVariableScope(vars map[string]Sequence) *variableScope {
	return scopeWithBindings(nil, vars)
}

func scopeWithBinding(parent *variableScope, name string, value Sequence) *variableScope {
	return &variableScope{
		parent:      parent,
		name:        name,
		value:       value,
		singleValue: true,
	}
}

func scopeWithBindings(parent *variableScope, bindings map[string]Sequence) *variableScope {
	if len(bindings) == 0 {
		return parent
	}
	if len(bindings) == 1 {
		for name, value := range bindings {
			return scopeWithBinding(parent, name, value)
		}
	}
	values := make(map[string]Sequence, len(bindings))
	for name, seq := range bindings {
		values[name] = seq
	}
	return &variableScope{parent: parent, values: values}
}

func (s *variableScope) Lookup(name string) (Sequence, bool) {
	for scope := s; scope != nil; scope = scope.parent {
		if scope.singleValue {
			if scope.name == name {
				return scope.value, true
			}
			continue
		}
		if seq, ok := scope.values[name]; ok {
			return seq, true
		}
	}
	return nil, false
}

type evalContextFrame struct {
	node        helium.Node
	contextItem Item
	position    int
	size        int
}

func (ec *evalContext) pushNodeContext(n helium.Node, pos, size int) evalContextFrame {
	frame := evalContextFrame{
		node:        ec.node,
		contextItem: ec.contextItem,
		position:    ec.position,
		size:        ec.size,
	}
	ec.node = n
	ec.contextItem = nil
	ec.position = pos
	ec.size = size
	return frame
}

func (ec *evalContext) pushContextItem(item Item, pos, size int) evalContextFrame {
	frame := evalContextFrame{
		node:        ec.node,
		contextItem: ec.contextItem,
		position:    ec.position,
		size:        ec.size,
	}
	ec.contextItem = item
	ec.node = nil
	ec.position = pos
	ec.size = size
	return frame
}

func (ec *evalContext) restoreContext(frame evalContextFrame) {
	ec.node = frame.node
	ec.contextItem = frame.contextItem
	ec.position = frame.position
	ec.size = frame.size
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

// resolveDefaultCollation returns the collation implementation for the
// dynamic context's default collation URI.  Returns nil when the default
// is the codepoint collation (the XPath default), so callers can fast-path.
func (ec *evalContext) resolveDefaultCollation() *collationImpl {
	if ec.defaultCollation == "" || ec.defaultCollation == lexicon.CollationCodepoint {
		return nil
	}
	coll, err := resolveCollation(ec.defaultCollation, "")
	if err != nil {
		return nil
	}
	return coll
}

func (ec *evalContext) getDefaultLanguage() string {
	if ec.defaultLanguage != "" {
		return ec.defaultLanguage
	}
	return "en"
}

func (ec *evalContext) withVar(name string, val Sequence) *evalContext {
	cp := *ec
	cp.vars = scopeWithBinding(ec.vars, name, val)
	return &cp
}

func (ec *evalContext) withScope(scope *variableScope) *evalContext {
	cp := *ec
	cp.vars = scope
	return &cp
}

// pushScope sets ec.vars to scope in place and returns the previous scope.
// The caller must call restoreScope when done to avoid corrupting state.
func (ec *evalContext) pushScope(scope *variableScope) *variableScope {
	old := ec.vars
	ec.vars = scope
	return old
}

// restoreScope restores ec.vars to a previous scope saved by pushScope.
func (ec *evalContext) restoreScope(old *variableScope) {
	ec.vars = old
}

func (ec *evalContext) countOps(n int) error {
	// Check context cancellation on every op count call
	if err := ec.goCtx.Err(); err != nil {
		return err
	}
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
