package xslt3

import (
	"context"
	"strconv"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
)

func (ec *execContext) fnElementAvailable(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || (args[0] == nil || sequence.Len(args[0]) == 0) {
		return xpath3.SingleBoolean(false), nil
	}
	av, err := xpath3.AtomizeItem(args[0].Get(0))
	if err != nil {
		return xpath3.SingleBoolean(false), nil
	}
	name, _ := xpath3.AtomicToString(av)

	// XTDE1440: element name must be a valid QName or EQName
	if !strings.HasPrefix(name, "Q{") && !isValidQName(name) {
		return nil, dynamicError(errCodeXTDE1440,
			"element-available: %q is not a valid EQName", name)
	}

	// Resolve prefix:local or Q{uri}local to namespace + local
	local := name
	ns := lexicon.NamespaceXSLT
	if strings.HasPrefix(name, "Q{") {
		if end := strings.IndexByte(name, '}'); end > 2 {
			ns = name[2:end]
			local = name[end+1:]
		}
	} else if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		local = name[idx+1:]
		resolved := false
		if uri, ok := ec.stylesheet.namespaces[prefix]; ok {
			ns = uri
			resolved = true
		}
		// XTDE1440: prefix has no namespace declaration
		if !resolved {
			return nil, dynamicError(errCodeXTDE1440,
				"element-available: prefix %q has no namespace declaration", prefix)
		}
	}
	if ns != lexicon.NamespaceXSLT {
		return xpath3.SingleBoolean(false), nil
	}
	// Check if the element is known and implemented
	if !elems.IsKnown(local) || !elems.IsImplemented(local) {
		return xpath3.SingleBoolean(false), nil
	}
	// Check if the element is available in the current stylesheet version
	if minVer := elems.MinVersion(local); ec.stylesheet.version != "" && ec.stylesheet.version < minVer {
		return xpath3.SingleBoolean(false), nil
	}
	return xpath3.SingleBoolean(true), nil
}

// function-available(name, arity?) returns true if the named function is available.
func (ec *execContext) fnFunctionAvailable(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || (args[0] == nil || sequence.Len(args[0]) == 0) {
		return xpath3.SingleBoolean(false), nil
	}
	av, err := xpath3.AtomizeItem(args[0].Get(0))
	if err != nil {
		return xpath3.SingleBoolean(false), nil
	}
	name, _ := xpath3.AtomicToString(av)

	// XTDE1400: function name must be a valid QName or EQName
	if !strings.HasPrefix(name, "Q{") && !isValidQName(name) {
		return nil, dynamicError(errCodeXTDE1400,
			"function-available: %q is not a valid QName", name)
	}

	// Extract optional arity parameter
	arity := -1 // -1 means any arity
	if len(args) >= 2 && args[1] != nil && sequence.Len(args[1]) > 0 {
		av2, err2 := xpath3.AtomizeItem(args[1].Get(0))
		if err2 == nil {
			s2, _ := xpath3.AtomicToString(av2)
			if n, err3 := strconv.Atoi(strings.TrimSpace(s2)); err3 == nil {
				arity = n
			}
		}
	}

	// Check XSLT functions by local name
	fns := ec.xsltFunctions()
	if fn, ok := fns[name]; ok {
		if arity < 0 || (arity >= fn.MinArity() && arity <= fn.MaxArity()) {
			return xpath3.SingleBoolean(true), nil
		}
		return xpath3.SingleBoolean(false), nil
	}

	// Handle EQName syntax: Q{uri}local
	if strings.HasPrefix(name, "Q{") {
		if closeIdx := strings.IndexByte(name, '}'); closeIdx > 0 {
			uri := name[2:closeIdx]
			local := name[closeIdx+1:]
			qn := xpath3.QualifiedName{URI: uri, Name: local}
			if ec.findXSLFunction(qn, arity) != nil {
				return xpath3.SingleBoolean(true), nil
			}
			// Check XPath built-in functions by namespace
			if xpath3.IsBuiltinFunctionNS(uri, local) {
				if arity < 0 || xpath3.BuiltinFunctionAcceptsArity(uri, local, arity) {
					return xpath3.SingleBoolean(true), nil
				}
				return xpath3.SingleBoolean(false), nil
			}
			return xpath3.SingleBoolean(false), nil
		}
	}

	// Check user-defined functions (prefixed)
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		local := name[idx+1:]
		uri, ok := ec.stylesheet.namespaces[prefix]
		if !ok {
			// XTDE1400: undeclared namespace prefix in function name
			return nil, dynamicError(errCodeXTDE1400,
				"undeclared namespace prefix %q in function-available(%q)", prefix, name)
		}
		qn := xpath3.QualifiedName{URI: uri, Name: local}
		if ec.findXSLFunction(qn, arity) != nil {
			return xpath3.SingleBoolean(true), nil
		}
		// Check runtime NS functions (including available-system-properties, etc.)
		if fnsNS := ec.xsltFunctionsNS(); fnsNS != nil {
			if fn, ok := fnsNS[qn]; ok {
				if arity < 0 || (arity >= fn.MinArity() && arity <= fn.MaxArity()) {
					return xpath3.SingleBoolean(true), nil
				}
				return xpath3.SingleBoolean(false), nil
			}
		}
		// Check XPath built-in functions by namespace
		if xpath3.IsBuiltinFunctionNS(uri, local) {
			if arity < 0 || xpath3.BuiltinFunctionAcceptsArity(uri, local, arity) {
				return xpath3.SingleBoolean(true), nil
			}
			return xpath3.SingleBoolean(false), nil
		}
	}

	// Check XPath built-in functions by local name (unprefixed → fn: namespace).
	if xpath3.IsBuiltinFunctionNS(xpath3.NSFn, name) {
		if arity < 0 || xpath3.BuiltinFunctionAcceptsArity(xpath3.NSFn, name, arity) {
			return xpath3.SingleBoolean(true), nil
		}
		return xpath3.SingleBoolean(false), nil
	}

	return xpath3.SingleBoolean(false), nil
}

// type-available(name) returns true if the named type is available.
func (ec *execContext) fnTypeAvailable(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || (args[0] == nil || sequence.Len(args[0]) == 0) {
		return xpath3.SingleBoolean(false), nil
	}
	av, err := xpath3.AtomizeItem(args[0].Get(0))
	if err != nil {
		return xpath3.SingleBoolean(false), nil
	}
	name, _ := xpath3.AtomicToString(av)

	// XTDE1428: type name must be a valid QName or EQName
	if !strings.HasPrefix(name, "Q{") && !isValidQName(name) {
		return nil, dynamicError(errCodeXTDE1428,
			"type-available: %q is not a valid EQName", name)
	}

	// XTDE1428: if it's a prefixed QName, the prefix must have a namespace declaration
	if !strings.HasPrefix(name, "Q{") {
		if idx := strings.IndexByte(name, ':'); idx >= 0 {
			prefix := name[:idx]
			if prefix != "xs" && prefix != "xsd" {
				if _, ok := ec.stylesheet.namespaces[prefix]; !ok {
					return nil, dynamicError(errCodeXTDE1428,
						"type-available: prefix %q has no namespace declaration", prefix)
				}
			}
		}
	}

	// Resolve QName to canonical xs:... form
	resolved := resolveQName(name, ec.stylesheet.namespaces)
	// If it resolved to {uri}local, normalize XSD namespace to xs: prefix
	if strings.HasPrefix(resolved, "{"+lexicon.NamespaceXSD+"}") {
		local := resolved[len("{"+lexicon.NamespaceXSD+"}"):]
		resolved = "xs:" + local
	} else if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		local := name[idx+1:]
		if prefix == "xs" || prefix == "xsd" {
			resolved = "xs:" + local
		} else if uri, ok := ec.stylesheet.namespaces[prefix]; ok && uri == lexicon.NamespaceXSD {
			resolved = "xs:" + local
		}
	}

	// xs:error is never available (it's the error type, not a real XSD type).
	// xs:dateTimeStamp is a valid XSD 1.1 type that we support, so it is NOT excluded.
	switch resolved {
	case "xs:error":
		return xpath3.SingleBoolean(false), nil
	}
	if xpath3.IsKnownXSDType(resolved) {
		return xpath3.SingleBoolean(true), nil
	}

	// Check imported schemas
	for _, schema := range ec.stylesheet.schemas {
		local := resolved
		ns := ""
		if strings.HasPrefix(resolved, "xs:") {
			local = resolved[3:]
			ns = lexicon.NamespaceXSD
		} else if strings.HasPrefix(resolved, "{") {
			// {uri}local form from resolveQName
			closeIdx := strings.IndexByte(resolved, '}')
			if closeIdx > 0 {
				ns = resolved[1:closeIdx]
				local = resolved[closeIdx+1:]
			}
		} else if idx := strings.IndexByte(resolved, ':'); idx >= 0 {
			local = resolved[idx+1:]
			// prefix was not resolved to a namespace — try schema target namespace
			ns = schema.TargetNamespace()
		} else {
			// No prefix — look up in the schema's target namespace
			ns = schema.TargetNamespace()
		}
		if _, ok := schema.LookupType(local, ns); ok {
			return xpath3.SingleBoolean(true), nil
		}
	}

	return xpath3.SingleBoolean(false), nil
}

// current-group() returns the items in the current group during for-each-group.
func (ec *execContext) fnCurrentGroup(_ context.Context, _ []xpath3.Sequence) (xpath3.Sequence, error) {
	if !ec.inGroupContext {
		return nil, dynamicError(errCodeXTDE1061, "current-group() called outside xsl:for-each-group")
	}
	if ec.currentGroup != nil {
		return ec.currentGroup, nil
	}
	return xpath3.EmptySequence(), nil
}

// current-grouping-key() returns the grouping key for the current group.
// XTDE1071: it is a non-recoverable dynamic error if called when the innermost
// xsl:for-each-group uses group-starting-with or group-ending-with (which do
// not establish a grouping key), or when called outside xsl:for-each-group.
func (ec *execContext) fnCurrentGroupingKey(_ context.Context, _ []xpath3.Sequence) (xpath3.Sequence, error) {
	if !ec.inGroupContext {
		return nil, dynamicError(errCodeXTDE1071, "current-grouping-key() called outside xsl:for-each-group")
	}
	if !ec.groupHasKey {
		return nil, dynamicError(errCodeXTDE1071, "current-grouping-key() called within xsl:for-each-group that uses group-starting-with or group-ending-with")
	}
	if ec.currentGroupKey != nil {
		return ec.currentGroupKey, nil
	}
	return xpath3.EmptySequence(), nil
}

// current-merge-group(source?) returns the current merge group sequence.
// The actual implementation is wired in execute_streaming.go; this stub
// is registered so that function-available() reports the function.
func (ec *execContext) fnCurrentMergeGroup(_ context.Context, _ []xpath3.Sequence) (xpath3.Sequence, error) {
	if !ec.inMergeAction {
		return nil, dynamicError(errCodeXTDE3480, "current-merge-group() called outside xsl:merge-action")
	}
	return xpath3.EmptySequence(), nil
}

// current-merge-key() returns the current merge key value.
// See fnCurrentMergeGroup comment.
func (ec *execContext) fnCurrentMergeKey(_ context.Context, _ []xpath3.Sequence) (xpath3.Sequence, error) {
	if !ec.inMergeAction {
		return nil, dynamicError(errCodeXTDE3510, "current-merge-key() called outside xsl:merge-action")
	}
	return xpath3.EmptySequence(), nil
}

// accumulatorLookup implements both accumulator-before() and accumulator-after().
// The caller passes an accessor that returns the per-node value and error maps
// for the appropriate phase. An accessor is used (rather than passing the maps
// directly) because the maps are lazily allocated: ensureAccumulatorStates may
// assign a new map to the execContext field after the call begins.
func (ec *execContext) accumulatorLookup(
	ctx context.Context,
	args []xpath3.Sequence,
	fnName string,
	nodeMaps func() (map[helium.Node]map[string]xpath3.Sequence, map[helium.Node]map[string]error),
) (xpath3.Sequence, error) {
	if len(args) == 0 || (args[0] == nil || sequence.Len(args[0]) == 0) {
		return xpath3.EmptySequence(), nil
	}
	av, err := xpath3.AtomizeItem(args[0].Get(0))
	if err != nil {
		return xpath3.EmptySequence(), nil
	}
	name, err := xpath3.AtomicToString(av)
	if err != nil {
		return xpath3.EmptySequence(), nil
	}
	// XTDE3340: validate the name is a valid EQName
	if !isValidQName(name) && !isValidEQName(name) {
		return nil, dynamicError(errCodeXTDE3340, "%q is not a valid EQName for %s", name, fnName)
	}
	name = resolveQName(name, ec.stylesheet.namespaces)
	if err := ec.checkAccumulatorAccess(name); err != nil {
		return nil, err
	}
	// XTDE3350: in streaming mode, accumulator functions must not be called
	// via a dynamic function reference because the captured context does not
	// reflect the node currently being processed.
	if xpath3.IsDynamicCall(ctx) && ec.isStreamableMode() {
		return nil, dynamicError(errCodeXTDE3350, "%s called via dynamic reference in streaming mode", fnName)
	}
	// XTTE3360: context item must be a node (not atomic), and not an attribute
	// or namespace node. Use the XPath function context node (which reflects
	// path steps like ../accumulator-before(...)) rather than ec.contextNode
	// (which is the XSLT template context and may still be an attribute).
	xpathNode := xpath3.FnContextNode(ctx)
	if xpathNode == nil {
		xpathNode = ec.contextNode
	}
	if !ec.evaluatingAccumulator && xpathNode == nil {
		return nil, dynamicError(errCodeXTTE3360, "%s requires context to be a node", fnName)
	}
	if !ec.evaluatingAccumulator && !isNilNode(xpathNode) {
		nt := xpathNode.Type()
		if nt == helium.AttributeNode || nt == helium.NamespaceDeclNode {
			return nil, dynamicError(errCodeXTTE3360, "%s cannot be called on an attribute or namespace node", fnName)
		}
	}
	if ec.evaluatingAccumulator {
		if ec.accumulatorStateError != nil {
			if deferredErr, ok := ec.accumulatorStateError[name]; ok {
				return nil, deferredErr
			}
		}
		if val, ok := ec.accumulatorState[name]; ok {
			return val, nil
		}
	}
	// Use the XPath function context node for lookups so that path
	// expressions like ../accumulator-before('x') resolve the
	// accumulator value on the stepped-to node, not the template context.
	lookupNode := xpathNode
	// During merge-key evaluation, accumulator-before falls back to
	// the after-values map when the before-values have no entry.
	if ec.evaluatingMergeKey && lookupNode != nil && ec.accumulatorAfterByNode != nil {
		if values, ok := ec.accumulatorAfterByNode[lookupNode]; ok {
			if val, ok := values[name]; ok {
				return val, nil
			}
		}
	}
	if lookupNode != nil {
		valuesByNode, errorsByNode := nodeMaps()
		if errorsByNode != nil {
			if errs, ok := errorsByNode[lookupNode]; ok {
				if deferredErr, ok := errs[name]; ok {
					return nil, deferredErr
				}
			}
		}
		if valuesByNode != nil {
			if values, ok := valuesByNode[lookupNode]; ok {
				if val, ok := values[name]; ok {
					return val, nil
				}
			}
		}
		// Lazily compute accumulator states for unknown documents
		if err := ec.ensureAccumulatorStates(ctx, lookupNode); err != nil {
			return nil, err
		}
		// Re-fetch maps after lazy computation (may have been allocated)
		valuesByNode, errorsByNode = nodeMaps()
		if errorsByNode != nil {
			if errs, ok := errorsByNode[lookupNode]; ok {
				if deferredErr, ok := errs[name]; ok {
					return nil, deferredErr
				}
			}
		}
		if valuesByNode != nil {
			if values, ok := valuesByNode[lookupNode]; ok {
				if val, ok := values[name]; ok {
					return val, nil
				}
			}
		}
	}
	if ec.accumulatorStateError != nil {
		if deferredErr, ok := ec.accumulatorStateError[name]; ok {
			return nil, deferredErr
		}
	}
	if val, ok := ec.accumulatorState[name]; ok {
		return val, nil
	}
	return xpath3.EmptySequence(), nil
}

// isStreamableMode returns true when the current mode has streamable="yes".
func (ec *execContext) isStreamableMode() bool {
	mode := ec.currentMode
	md := ec.stylesheet.modeDefs[mode]
	if md == nil && (mode == "" || mode == modeUnnamed) {
		md = ec.stylesheet.modeDefs[modeDefault]
	}
	return md != nil && md.Streamable
}

func (ec *execContext) checkAccumulatorAccess(name string) error {
	// XTDE3340: the name must correspond to a declared accumulator.
	if _, ok := ec.stylesheet.accumulators[name]; !ok {
		return dynamicError(errCodeXTDE3340, "no accumulator %q is declared in this stylesheet", name)
	}
	if ec.evaluatingAccumulator {
		return nil
	}
	// XTDE3362: check mode-level use-accumulators restriction.
	// If the current mode explicitly declares use-accumulators, only
	// those accumulators are accessible from templates in that mode.
	if md := ec.stylesheet.modeDefs[ec.currentMode]; md != nil && md.UseAccumulators != nil {
		switch *md.UseAccumulators {
		case "#all":
			// all accumulators allowed
		case "":
			// use-accumulators="" means NO accumulators allowed
			return dynamicError(errCodeXTDE3362,
				"accumulator %q is not applicable in mode %q (use-accumulators is empty)", name, ec.currentMode)
		default:
			allowed := make(map[string]struct{})
			for _, n := range strings.Fields(*md.UseAccumulators) {
				allowed[n] = struct{}{}
			}
			if _, ok := allowed[name]; !ok {
				return dynamicError(errCodeXTDE3362,
					"accumulator %q is not applicable in mode %q (not in use-accumulators)", name, ec.currentMode)
			}
		}
	}
	// Document-level accumulator applicability
	if ec.activeAccumulators != nil {
		if _, ok := ec.activeAccumulators[name]; !ok {
			return dynamicError(errCodeXTDE3362, "accumulator %q is not applicable in the current source-document", name)
		}
	}
	if !ec.requireStreamableAccums {
		return nil
	}
	def, ok := ec.stylesheet.accumulators[name]
	if !ok || def.Streamable {
		return nil
	}
	return dynamicError(errCodeXTDE3362, "accumulator %q is not streamable", name)
}

// ensureAccumulatorStates lazily computes accumulator states for the document
// tree containing the given node, if not already computed.
func isNilNode(n helium.Node) bool {
	if n == nil {
		return true
	}
	// Guard against typed-nil interface values (e.g. (*Document)(nil)).
	// helium.Node is a sealed interface; all implementations are pointer types.
	switch v := n.(type) {
	case *helium.Document:
		return v == nil
	case *helium.Element:
		return v == nil
	case *helium.Text:
		return v == nil
	case *helium.Comment:
		return v == nil
	case *helium.ProcessingInstruction:
		return v == nil
	case *helium.Attribute:
		return v == nil
	case *helium.DTD:
		return v == nil
	case *helium.EntityRef:
		return v == nil
	case *helium.NamespaceNodeWrapper:
		return v == nil
	case *helium.XIncludeMarker:
		return v == nil
	case *helium.Notation:
		return v == nil
	case *helium.AttributeDecl:
		return v == nil
	case *helium.ElementDecl:
		return v == nil
	default:
		return false
	}
}

func (ec *execContext) ensureAccumulatorStates(ctx context.Context, node helium.Node) error {
	if isNilNode(node) || len(ec.stylesheet.accumulators) == 0 {
		return nil
	}
	root := documentRoot(node)
	if ec.accumulatorComputedDocs == nil {
		ec.accumulatorComputedDocs = make(map[helium.Node]struct{})
	}
	if _, done := ec.accumulatorComputedDocs[root]; done {
		return nil
	}
	ec.accumulatorComputedDocs[root] = struct{}{}
	names := append([]string(nil), ec.stylesheet.accumulatorOrder...)
	return ec.computeAccumulatorStates(ctx, root, names)
}

// copy-of() returns a deep copy of the context node (zero-argument XSLT 3.0 streaming function).
