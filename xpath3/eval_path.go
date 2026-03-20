package xpath3

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

func evalLiteral(e LiteralExpr) (Sequence, error) {
	switch v := e.Value.(type) {
	case string:
		return SingleString(v), nil
	case *big.Int:
		return SingleIntegerBig(v), nil
	case *big.Rat:
		return SingleDecimal(v), nil
	case float64:
		return SingleDouble(v), nil
	}
	return nil, fmt.Errorf("%w: literal %T", ErrUnsupportedExpr, e.Value)
}

func evalVariable(ec *evalContext, e VariableExpr) (Sequence, error) {
	if ec.vars != nil {
		// Try exact name first
		if v, ok := ec.vars.Lookup(e.Name); ok {
			return enrichNodeItems(ec, v), nil
		}
		// If EQName (Q{uri}local), normalize to {uri}local and retry
		if strings.HasPrefix(e.Name, "Q{") {
			resolved := e.Name[1:] // strip leading "Q"
			if v, ok := ec.vars.Lookup(resolved); ok {
				return enrichNodeItems(ec, v), nil
			}
		}
		// If prefixed, resolve to {uri}local and retry
		if e.Prefix != "" {
			if uri, ok := ec.namespaces[e.Prefix]; ok {
				local := e.Name[len(e.Prefix)+1:] // strip "prefix:"
				resolved := "{" + uri + "}" + local
				if v, ok := ec.vars.Lookup(resolved); ok {
					return enrichNodeItems(ec, v), nil
				}
			}
		}
	}
	// Fallback to lazy variable resolver (e.g. for XSLT global variables)
	if ec.variableResolver != nil {
		if v, ok, err := ec.variableResolver.ResolveVariable(ec.goCtx, e.Name); err != nil {
			return nil, err
		} else if ok {
			return enrichNodeItems(ec, v), nil
		}
	}
	return nil, fmt.Errorf("%w: $%s", ErrUndefinedVariable, e.Name)
}

// enrichNodeItems ensures NodeItems in a sequence have their TypeAnnotation
// and related fields set from the evalContext's type annotations map. This is
// needed because variables may store NodeItems created before type annotations
// were available (e.g., validated after construction).
func enrichNodeItems(ec *evalContext, seq Sequence) Sequence {
	if ec == nil || ec.typeAnnotations == nil {
		return seq
	}
	needsEnrich := false
	for _, item := range seq {
		if ni, ok := item.(NodeItem); ok && ni.TypeAnnotation == "" {
			if _, hasAnn := ec.typeAnnotations[ni.Node]; hasAnn {
				needsEnrich = true
				break
			}
		}
	}
	if !needsEnrich {
		return seq
	}
	result := make(Sequence, len(seq))
	for i, item := range seq {
		if ni, ok := item.(NodeItem); ok && ni.TypeAnnotation == "" {
			enriched := nodeItemFor(ec, ni.Node)
			result[i] = enriched
		} else {
			result[i] = item
		}
	}
	return result
}

func evalSequenceExpr(evalFn exprEvaluator, ec *evalContext, e SequenceExpr) (Sequence, error) {
	if len(e.Items) == 0 {
		return nil, nil
	}
	var result Sequence
	for _, item := range e.Items {
		seq, err := evalFn(ec, item)
		if err != nil {
			return nil, err
		}
		result = append(result, seq...)
	}
	return result, nil
}

func evalLocationPath(evalFn exprEvaluator, ec *evalContext, lp *LocationPath) (Sequence, error) {
	var nodes []helium.Node

	if lp.Absolute {
		if ec.node == nil {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: "context item is absent"}
		}
		root := ixpath.DocumentRoot(ec.node)
		nodes = []helium.Node{root}
	} else {
		if ec.node == nil {
			return nil, &XPathError{Code: errCodeXPDY0002, Message: "context item is absent"}
		}
		nodes = []helium.Node{ec.node}
	}

	var err error
	for _, step := range lp.Steps {
		if len(step.Predicates) > 0 {
			nodes, err = evalStepWithPredicates(evalFn, ec, nodes, step)
		} else {
			nodes, err = evalStepNoPredicates(ec, nodes, step)
		}
		if err != nil {
			return nil, err
		}
	}

	result := make(Sequence, len(nodes))
	for i, n := range nodes {
		result[i] = nodeItemFor(ec, n)
	}
	return result, nil
}

func nodeItemFor(ec *evalContext, n helium.Node) NodeItem {
	ni := NodeItem{Node: n}
	if ec == nil || ec.typeAnnotations == nil {
		return ni
	}
	ni.TypeAnnotation = ec.typeAnnotations[n]
	ni.AtomizedType = atomizedTypeForAnnotation(ni.TypeAnnotation, ec.schemaDeclarations)
	if ec.schemaDeclarations != nil && ni.TypeAnnotation != "" {
		if itemType, ok := ec.schemaDeclarations.ListItemType(ni.TypeAnnotation); ok {
			ni.ListItemType = itemType
		}
	}
	return ni
}

func evalStepWithPredicates(evalFn exprEvaluator, ec *evalContext, nodes []helium.Node, step Step) ([]helium.Node, error) {
	var allFiltered []helium.Node
	for _, n := range nodes {
		candidates, err := ixpath.TraverseAxis(step.Axis, n, ec.maxNodes)
		if err != nil {
			return nil, err
		}
		if err := ec.countOps(len(candidates)); err != nil {
			return nil, err
		}
		matched := filterByNodeTest(candidates, step.NodeTest, step.Axis, ec)
		for _, pred := range step.Predicates {
			matched, err = applyPredicate(evalFn, ec, matched, pred)
			if err != nil {
				return nil, err
			}
		}
		allFiltered = append(allFiltered, matched...)
	}
	return ixpath.DeduplicateNodes(allFiltered, ec.docOrder, ec.maxNodes)
}

func evalStepNoPredicates(ec *evalContext, nodes []helium.Node, step Step) ([]helium.Node, error) {
	var next []helium.Node
	for _, n := range nodes {
		candidates, err := ixpath.TraverseAxis(step.Axis, n, ec.maxNodes)
		if err != nil {
			return nil, err
		}
		if err := ec.countOps(len(candidates)); err != nil {
			return nil, err
		}
		next = append(next, filterByNodeTest(candidates, step.NodeTest, step.Axis, ec)...)
	}
	return ixpath.DeduplicateNodes(next, ec.docOrder, ec.maxNodes)
}

func filterByNodeTest(candidates []helium.Node, nt NodeTest, axis AxisType, ec *evalContext) []helium.Node {
	var matched []helium.Node
	for _, c := range candidates {
		if matchNodeTest(nt, c, axis, ec) {
			matched = append(matched, c)
		}
	}
	return matched
}

func matchNodeTest(nt NodeTest, n helium.Node, axis AxisType, ec *evalContext) bool {
	switch test := nt.(type) {
	case NameTest:
		return matchNameTest(test, n, axis, ec)
	case TypeTest:
		return matchTypeTest(test, n)
	case PITest:
		if n.Type() != helium.ProcessingInstructionNode {
			return false
		}
		if test.Target == "" {
			return true
		}
		return n.Name() == test.Target
	case ElementTest:
		if n.Type() != helium.ElementNode {
			return false
		}
		if test.Name != "" && test.Name != "*" {
			if ixpath.LocalNameOf(n) != test.Name {
				return false
			}
		}
		if test.TypeName != "" {
			ann := nodeTypeAnnotation(n, ec)
			if ann == "" {
				ann = TypeUntyped // elements default to xs:untyped
			}
			target := resolveTestTypeName(test.TypeName, ec)
			if !isSubtypeOf(ann, target) {
				if ec == nil || ec.schemaDeclarations == nil || !ec.schemaDeclarations.IsSubtypeOf(ann, target) {
					return false
				}
			}
		}
		return true
	case AttributeTest:
		if _, ok := n.(*helium.Attribute); !ok {
			return false
		}
		if test.Name != "" && test.Name != "*" {
			if ixpath.LocalNameOf(n) != test.Name {
				return false
			}
		}
		if test.TypeName != "" {
			ann := nodeTypeAnnotation(n, ec)
			if ann == "" {
				ann = TypeUntypedAtomic
			}
			target := resolveTestTypeName(test.TypeName, ec)
			if !isSubtypeOf(ann, target) {
				if ec == nil || ec.schemaDeclarations == nil || !ec.schemaDeclarations.IsSubtypeOf(ann, target) {
					return false
				}
			}
		}
		return true
	case DocumentTest:
		if n.Type() != helium.DocumentNode {
			return false
		}
		if test.Inner != nil {
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if matchNodeTest(test.Inner, c, AxisChild, ec) {
					return true
				}
			}
			return false
		}
		return true
	case SchemaElementTest:
		if n.Type() != helium.ElementNode {
			return false
		}
		if ec == nil || ec.schemaDeclarations == nil {
			// Without schema, match by name only.
			if test.Name == "" || test.Name == "*" {
				return true
			}
			_, local := splitQName(test.Name)
			return ixpath.LocalNameOf(n) == local
		}
		prefix, local := splitQName(test.Name)
		ns := ""
		if prefix != "" && ec.namespaces != nil {
			ns = ec.namespaces[prefix]
		}
		if ixpath.LocalNameOf(n) != local || ixpath.NodeNamespaceURI(n) != ns {
			return false
		}
		typeName, found := ec.schemaDeclarations.LookupSchemaElement(local, ns)
		if !found {
			return false
		}
		ann := nodeTypeAnnotation(n, ec)
		if ann == "" {
			ann = TypeUntyped
		}
		if isSubtypeOf(ann, typeName) {
			return true
		}
		return ec.schemaDeclarations.IsSubtypeOf(ann, typeName)
	case NamespaceNodeTest:
		return n.Type() == helium.NamespaceNode
	case AnyItemTest:
		return true
	}
	return false
}

func matchNameTest(test NameTest, n helium.Node, axis AxisType, ec *evalContext) bool {
	switch axis {
	case AxisAttribute:
		if _, ok := n.(*helium.Attribute); !ok {
			return false
		}
	case AxisNamespace:
		if n.Type() != helium.NamespaceNode {
			return false
		}
		if test.Local == "*" {
			return true
		}
		return n.Name() == test.Local
	default:
		if n.Type() != helium.ElementNode {
			return false
		}
	}

	if test.Local == "*" {
		if test.URI != "" {
			return ixpath.NodeNamespaceURI(n) == test.URI
		}
		if test.Prefix == "" {
			return true
		}
		return matchPrefix(test.Prefix, n, ec)
	}

	if ixpath.LocalNameOf(n) != test.Local {
		return false
	}

	if test.URI != "" {
		return ixpath.NodeNamespaceURI(n) == test.URI
	}
	if test.Prefix == "*" {
		// *:local matches any namespace
		return true
	}
	if test.Prefix != "" {
		return matchPrefix(test.Prefix, n, ec)
	}
	// Check for default element namespace (xpath-default-namespace).
	// Only applies to element axis, not attributes.
	// Per XPath 3.1 §3.3.2.1: when default element namespace is absent,
	// unprefixed names match only no-namespace elements.
	if axis != AxisAttribute && ec.namespaces != nil {
		return ixpath.NodeNamespaceURI(n) == ec.namespaces[""]
	}
	// No namespace context at all: permissive match (any namespace).
	return true
}

func matchPrefix(prefix string, n helium.Node, ec *evalContext) bool {
	if ec.namespaces != nil {
		if uri, ok := ec.namespaces[prefix]; ok {
			return ixpath.NodeNamespaceURI(n) == uri
		}
	}
	// The xml prefix is always bound per the XML Namespaces spec.
	if prefix == "xml" {
		return ixpath.NodeNamespaceURI(n) == helium.XMLNamespace
	}
	// Check built-in XPath prefixes (fn, xs, math, map, array, err).
	if uri, ok := defaultPrefixNS[prefix]; ok {
		return ixpath.NodeNamespaceURI(n) == uri
	}
	// Per XPath spec, prefix resolution must come from the static/evaluation
	// namespace bindings, not the document's lexical prefixes. If the prefix
	// is not declared in the namespace context, it cannot match.
	return false
}

func matchTypeTest(test TypeTest, n helium.Node) bool {
	switch test.Kind {
	case NodeKindNode:
		return true
	case NodeKindText:
		return n.Type() == helium.TextNode || n.Type() == helium.CDATASectionNode
	case NodeKindComment:
		return n.Type() == helium.CommentNode
	case NodeKindProcessingInstruction:
		return n.Type() == helium.ProcessingInstructionNode
	}
	return false
}

func applyPredicate(evalFn exprEvaluator, ec *evalContext, nodes []helium.Node, pred Expr) ([]helium.Node, error) {
	if err := ec.countOps(len(nodes)); err != nil {
		return nil, err
	}
	size := len(nodes)
	var result []helium.Node
	for i, n := range nodes {
		pctx := ec.withNode(n, i+1, size)
		r, err := evalFn(pctx, pred)
		if err != nil {
			return nil, err
		}
		match, err := predicateTrue(r, i+1)
		if err != nil {
			return nil, err
		}
		if match {
			result = append(result, n)
		}
	}
	return result, nil
}

// nodeTypeAnnotation returns the type annotation for a node from the
// evalContext's type annotation map.
func nodeTypeAnnotation(n helium.Node, ec *evalContext) string {
	if ec == nil || ec.typeAnnotations == nil {
		return ""
	}
	return ec.typeAnnotations[n]
}

// resolveTestTypeName normalizes a type name from an ElementTest/AttributeTest
// to the canonical annotation format:
//   - "xs:localName" for types in the XSD namespace
//   - "Q{ns}localName" for types in any other namespace
//   - the raw name for names without a prefix (treated as no-namespace)
func resolveTestTypeName(raw string, ec *evalContext) string {
	if strings.HasPrefix(raw, "xs:") {
		return raw
	}
	if idx := strings.IndexByte(raw, ':'); idx >= 0 {
		prefix := raw[:idx]
		local := raw[idx+1:]
		if prefix == "xsd" {
			return "xs:" + local
		}
		if ec != nil && ec.namespaces != nil {
			if uri, ok := ec.namespaces[prefix]; ok {
				if uri == lexicon.XSD {
					return "xs:" + local
				}
				return QAnnotation(uri, local)
			}
		}
	}
	// Unprefixed type names: resolve using the default element namespace
	// (same as unprefixed element names in XPath). This handles types like
	// "addressType" in element(address, addressType) when xpath-default-namespace
	// provides the namespace.
	if ec != nil && ec.namespaces != nil {
		if defNS, ok := ec.namespaces[""]; ok && defNS != "" {
			return QAnnotation(defNS, raw)
		}
	}
	// No namespace: use Q{} annotation form for consistency with
	// xsdTypeNameFromDef which produces Q{}local for no-namespace types.
	return "Q{}" + raw
}

// predicateTrue evaluates a predicate result per XPath spec:
// numeric → compare to position, otherwise → EBV.
func predicateTrue(r Sequence, position int) (bool, error) {
	if len(r) == 1 {
		if av, ok := r[0].(AtomicValue); ok && av.IsNumeric() {
			return av.ToFloat64() == float64(position), nil
		}
	}
	return EBV(r)
}
