package xslt3

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
)

func (ec *execContext) fnCopyOf(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	// copy-of() with argument: deep-copy the argument node(s)
	// copy-of() with no args: deep-copy the context node (streaming snapshot)
	var nodes []helium.Node
	var atomics xpath3.ItemSlice
	if len(args) > 0 {
		// Explicit argument: copy the given node(s), pass atomics through.
		// If the argument is empty, return empty (don't fall through to context).
		if args[0] == nil || sequence.Len(args[0]) == 0 {
			return xpath3.EmptySequence(), nil
		}
		for item := range sequence.Items(args[0]) {
			ni, ok := item.(xpath3.NodeItem)
			if !ok {
				atomics = append(atomics, item)
				continue
			}
			nodes = append(nodes, ni.Node)
		}
		if len(nodes) == 0 && len(atomics) > 0 {
			return atomics, nil
		}
	} else {
		// No argument: copy the context node (streaming snapshot).
		// Prefer XPath dynamic context node (set by evaluator during path
		// steps like transaction/copy-of()) over XSLT execution context.
		if n := xpath3.FnContextNode(ctx); n != nil {
			nodes = append(nodes, n)
		} else if ec.contextNode != nil {
			nodes = append(nodes, ec.contextNode)
		} else {
			return nil, dynamicError(errCodeXPDY0002, "copy-of() with no arguments requires a context item")
		}
	}
	var result xpath3.ItemSlice
	for _, node := range nodes {
		// XTTE0950: copying a standalone attribute with namespace-sensitive
		// content (xs:QName, xs:NOTATION) via copy-of() loses namespace bindings.
		if node.Type() == helium.AttributeNode && ec.nodeHasNamespaceSensitiveContent(node) {
			return nil, dynamicError(errCodeXTTE0950,
				"copy-of(): cannot copy attribute with namespace-sensitive content")
		}
		copied, err := helium.CopyNode(node, ec.resultDoc)
		if err != nil {
			result = append(result, xpath3.NodeItem{Node: node})
			continue
		}
		// copy-of preserves all in-scope namespaces, not just those
		// declared directly on the element.  CopyNode only copies the
		// element's own nsDefs, so we must add inherited ones here.
		if srcElem, ok := node.(*helium.Element); ok {
			if dstElem, ok2 := copied.(*helium.Element); ok2 {
				addInScopeNamespaces(srcElem, dstElem)
			}
		}
		// Transfer nilled status from original to copy so fn:nilled()
		// works correctly on copied nodes.
		ec.transferNilledStatus(node, copied)
		// Transfer accumulator state from original to copy so that
		// accumulator-before()/accumulator-after() work on the copy.
		if len(ec.stylesheet.accumulators) > 0 {
			_ = ec.ensureAccumulatorStates(ctx, node)
			ec.transferAccumulatorStates(node, copied)
			// Mark the copy's document as computed to prevent recomputation.
			copyRoot := documentRoot(copied)
			if ec.accumulatorComputedDocs == nil {
				ec.accumulatorComputedDocs = make(map[helium.Node]struct{})
			}
			ec.accumulatorComputedDocs[copyRoot] = struct{}{}
		}
		result = append(result, xpath3.NodeItem{Node: copied})
	}
	return result, nil
}

// snapshot() produces a deep copy of the node that also preserves ancestor
// information.  Each ancestor element is shallow-copied (name, attributes,
// namespace declarations) and the chain is connected so that
// ancestor::*/parent::*/.. navigation works on the snapshot.
func (ec *execContext) fnSnapshot(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) > 0 {
		if args[0] == nil || sequence.Len(args[0]) == 0 {
			return xpath3.EmptySequence(), nil
		}
		var result xpath3.ItemSlice
		for item := range sequence.Items(args[0]) {
			ni, ok := item.(xpath3.NodeItem)
			if !ok {
				// Non-node items (atomics, functions) pass through unchanged.
				result = append(result, item)
				continue
			}
			snapped, err := ec.snapshotNode(ctx, ni.Node)
			if err != nil {
				result = append(result, xpath3.NodeItem{Node: ni.Node})
				continue
			}
			ec.transferNilledStatus(ni.Node, snapped)
			result = append(result, xpath3.NodeItem{Node: snapped})
		}
		return result, nil
	}

	// No argument: snapshot the context node.
	var node helium.Node
	if n := xpath3.FnContextNode(ctx); n != nil {
		node = n
	} else if ec.contextNode != nil {
		node = ec.contextNode
	} else {
		return xpath3.EmptySequence(), nil
	}
	snapped, err := ec.snapshotNode(ctx, node)
	if err != nil {
		return xpath3.ItemSlice{xpath3.NodeItem{Node: node}}, nil
	}
	ec.transferNilledStatus(node, snapped)
	return xpath3.ItemSlice{xpath3.NodeItem{Node: snapped}}, nil
}

// snapshotNode creates a deep copy of node and wraps it in shallow copies
// of all its ancestors up to and including the document root.
func (ec *execContext) snapshotNode(ctx context.Context, node helium.Node) (helium.Node, error) {
	// For a document node, just do a full deep copy.
	if node.Type() == helium.DocumentNode {
		doc, ok := node.(*helium.Document)
		if !ok {
			return nil, fmt.Errorf("unexpected DocumentNode type %T", node)
		}
		return helium.CopyDoc(doc)
	}

	// For namespace nodes, build the ancestor chain, declare the namespace
	// on the innermost shell element, and return a NamespaceNodeWrapper.
	if node.Type() == helium.NamespaceNode {
		return ec.snapshotNamespaceNode(node)
	}

	// For attribute nodes, build the ancestor chain, attach the attribute
	// to the innermost shell element, and return the copied attribute.
	if node.Type() == helium.AttributeNode {
		return ec.snapshotAttributeNode(node)
	}

	// Build the ancestor chain from node's parent up to (but not including)
	// the document node, collecting elements in bottom-up order.
	var ancestors []*helium.Element
	var hasDocRoot bool
	var origDoc *helium.Document
	for p := node.Parent(); p != nil; p = p.Parent() {
		if elem, ok := p.(*helium.Element); ok {
			ancestors = append(ancestors, elem)
		}
		if p.Type() == helium.DocumentNode {
			hasDocRoot = true
			origDoc, _ = p.(*helium.Document)
		}
	}

	// Create a new document to own the snapshot.
	snapDoc := helium.NewDefaultDocument()

	// Copy DTD info (entities, notations) from the original document so
	// that unparsed-entity-uri() / unparsed-entity-public-id() work on
	// the snapshot.
	if origDoc != nil {
		helium.CopyDTDInfo(origDoc, snapDoc)
	}

	// Deep-copy the target node itself.
	copied, err := helium.CopyNode(node, snapDoc)
	if err != nil {
		return nil, err
	}

	// Add all in-scope namespaces from the original element directly onto
	// the deep-copied element. This ensures that when the snapshot result
	// is later copied into the result tree (via CopyNode in xsl:sequence),
	// the inherited namespace bindings travel with the element itself
	// rather than relying solely on the ancestor shell chain.
	if srcElem, ok := node.(*helium.Element); ok {
		if dstElem, ok2 := copied.(*helium.Element); ok2 {
			addInScopeNamespaces(srcElem, dstElem)
		}
	}

	// For parentless (orphan) nodes that have no document root ancestor,
	// return just the deep copy without wrapping in a document node.
	// Per XSLT 3.0 spec, snapshot of a parentless node is a deep copy.
	if len(ancestors) == 0 && !hasDocRoot {
		return copied, nil
	}

	// Build the ancestor chain: for each ancestor (bottom-up), create a
	// shallow copy (name + attributes + namespace declarations) and attach
	// the previous level as its only child.
	current := copied
	for _, anc := range ancestors {
		shell, err := shallowCopyElement(anc, snapDoc)
		if err != nil {
			return nil, err
		}
		// Add inherited in-scope namespaces to the shell element.
		addInScopeNamespaces(anc, shell)
		if err := shell.AddChild(current); err != nil {
			return nil, err
		}
		current = shell
	}

	// Attach the top-level element (or the copied node itself if no
	// ancestors) to the snapshot document.
	if err := snapDoc.AddChild(current); err != nil {
		return nil, err
	}

	// Ensure accumulator states are computed for the original document
	// before transferring, since the snapshot may be created before any
	// accumulator-before()/accumulator-after() call triggers lazy computation.
	if err := ec.ensureAccumulatorStates(ctx, node); err != nil {
		return nil, err
	}

	// Transfer accumulator state from original subtree to snapshot copy.
	ec.transferAccumulatorStates(node, copied)
	// Transfer accumulator state for ancestor shells too.
	for i, anc := range ancestors {
		// ancestors[0] is the immediate parent, ancestors[1] is grandparent, etc.
		// The ancestor chain was built bottom-up, so the corresponding shell
		// is: copied.Parent() for i=0, copied.Parent().Parent() for i=1, etc.
		shell := copied.Parent()
		for j := 0; j < i; j++ {
			if shell != nil {
				shell = shell.Parent()
			}
		}
		if shell != nil {
			ec.transferAccumulatorNode(anc, shell)
		}
	}

	// Mark the snapshot document as having computed accumulators so that
	// ensureAccumulatorStates does not re-walk and overwrite the transferred
	// values (the snapshot tree structure differs from the original).
	if ec.accumulatorComputedDocs == nil {
		ec.accumulatorComputedDocs = make(map[helium.Node]struct{})
	}
	ec.accumulatorComputedDocs[snapDoc] = struct{}{}

	return copied, nil
}

// transferAccumulatorStates copies accumulator before/after state from
// an original node subtree to a copied node subtree (parallel walk).
func (ec *execContext) transferAccumulatorStates(orig, copy helium.Node) {
	ec.transferAccumulatorNode(orig, copy)
	oc := orig.FirstChild()
	cc := copy.FirstChild()
	for oc != nil && cc != nil {
		ec.transferAccumulatorStates(oc, cc)
		oc = oc.NextSibling()
		cc = cc.NextSibling()
	}
}

// transferAccumulatorNode copies the accumulator before/after maps for a
// single node from orig to copy.
func (ec *execContext) transferAccumulatorNode(orig, copy helium.Node) {
	if ec.accumulatorBeforeByNode != nil {
		if vals, ok := ec.accumulatorBeforeByNode[orig]; ok {
			ec.accumulatorBeforeByNode[copy] = cloneAccumulatorSnapshot(vals)
		}
	}
	if ec.accumulatorAfterByNode != nil {
		if vals, ok := ec.accumulatorAfterByNode[orig]; ok {
			ec.accumulatorAfterByNode[copy] = cloneAccumulatorSnapshot(vals)
		}
	}
	if ec.accumulatorBeforeErrorByNode != nil {
		if errs, ok := ec.accumulatorBeforeErrorByNode[orig]; ok {
			cloned := make(map[string]error, len(errs))
			for k, v := range errs {
				cloned[k] = v
			}
			ec.accumulatorBeforeErrorByNode[copy] = cloned
		}
	}
	if ec.accumulatorAfterErrorByNode != nil {
		if errs, ok := ec.accumulatorAfterErrorByNode[orig]; ok {
			cloned := make(map[string]error, len(errs))
			for k, v := range errs {
				cloned[k] = v
			}
			ec.accumulatorAfterErrorByNode[copy] = cloned
		}
	}
}

// snapshotNamespaceNode creates a snapshot of a namespace node by building
// the ancestor chain and declaring the namespace on the innermost element.
func (ec *execContext) snapshotNamespaceNode(node helium.Node) (helium.Node, error) {
	prefix := node.Name()
	uri := string(node.Content())

	// Build the ancestor chain from the namespace node's parent element.
	var ancestors []*helium.Element
	for p := node.Parent(); p != nil; p = p.Parent() {
		if elem, ok := p.(*helium.Element); ok {
			ancestors = append(ancestors, elem)
		}
	}

	snapDoc := helium.NewDefaultDocument()

	// The innermost shell is the parent element of the namespace.
	// Build it first, then wrap with outer ancestors.
	var innermostShell *helium.Element
	var current helium.Node
	for i, anc := range ancestors {
		shell, err := shallowCopyElement(anc, snapDoc)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			innermostShell = shell
		}
		if current != nil {
			if err := shell.AddChild(current); err != nil {
				return nil, err
			}
		}
		current = shell
	}

	if current != nil {
		if err := snapDoc.AddChild(current); err != nil {
			return nil, err
		}
	}

	// Ensure the namespace is declared on the innermost element.
	if innermostShell != nil {
		// The shallowCopy already copied namespace declarations, so the
		// namespace should already be there. Return a wrapper pointing to
		// the innermost element.
		ns := helium.NewNamespace(prefix, uri)
		return helium.NewNamespaceNodeWrapper(ns, innermostShell), nil
	}

	// No parent element — return a standalone namespace wrapper.
	ns := helium.NewNamespace(prefix, uri)
	return helium.NewNamespaceNodeWrapper(ns, nil), nil
}

// snapshotAttributeNode creates a snapshot of an attribute node by building
// the ancestor chain and attaching the attribute to the innermost element.
func (ec *execContext) snapshotAttributeNode(node helium.Node) (helium.Node, error) {
	attr, ok := node.(*helium.Attribute)
	if !ok {
		return nil, fmt.Errorf("unexpected AttributeNode type %T", node)
	}

	// Build the ancestor chain: the attribute's parent element, then its ancestors.
	var ancestors []*helium.Element
	for p := node.Parent(); p != nil; p = p.Parent() {
		if elem, ok := p.(*helium.Element); ok {
			ancestors = append(ancestors, elem)
		}
	}

	snapDoc := helium.NewDefaultDocument()

	var innermostShell *helium.Element
	var current helium.Node
	for i, anc := range ancestors {
		shell, err := shallowCopyElement(anc, snapDoc)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			innermostShell = shell
		}
		if current != nil {
			if err := shell.AddChild(current); err != nil {
				return nil, err
			}
		}
		current = shell
	}

	if current != nil {
		if err := snapDoc.AddChild(current); err != nil {
			return nil, err
		}
	}

	// Create a copy of the attribute on the innermost shell.
	if innermostShell != nil {
		// The shallowCopy already includes all attributes, so find the
		// matching attribute on the shell.
		for _, a := range innermostShell.Attributes() {
			if a.LocalName() == attr.LocalName() && a.URI() == attr.URI() {
				return a, nil
			}
		}
	}

	// Fallback: create a standalone copy.
	copied, err := helium.CopyNode(node, snapDoc)
	if err != nil {
		return nil, err
	}
	return copied, nil
}

// shallowCopyElement copies an element's name, namespace declarations, and
// attributes but none of its children.
func shallowCopyElement(src *helium.Element, doc *helium.Document) (*helium.Element, error) {
	elem, err := doc.CreateElement(src.LocalName())
	if err != nil {
		return nil, err
	}

	declaredPrefixes := make(map[string]bool)

	// Copy namespace declarations.
	if nc, ok := helium.Node(src).(helium.NamespaceContainer); ok {
		for _, ns := range nc.Namespaces() {
			if err := elem.DeclareNamespace(ns.Prefix(), ns.URI()); err != nil {
				return nil, err
			}
			declaredPrefixes[ns.Prefix()] = true
		}
	}

	// Copy the active namespace.
	if nsr, ok := helium.Node(src).(helium.Namespacer); ok {
		if ns := nsr.Namespace(); ns != nil {
			if ns.Prefix() != "" && !declaredPrefixes[ns.Prefix()] {
				if err := elem.DeclareNamespace(ns.Prefix(), ns.URI()); err != nil {
					return nil, err
				}
			}
			if err := elem.SetActiveNamespace(ns.Prefix(), ns.URI()); err != nil {
				return nil, err
			}
		}
	}

	// Copy attributes, preserving namespace information.
	for _, a := range src.Attributes() {
		if a.URI() != "" {
			ns := helium.NewNamespace(a.Prefix(), a.URI())
			elem.SetLiteralAttributeNS(a.LocalName(), a.Value(), ns)
			if a.Prefix() != "" && !declaredPrefixes[a.Prefix()] {
				_ = elem.DeclareNamespace(a.Prefix(), a.URI())
				declaredPrefixes[a.Prefix()] = true
			}
		} else {
			elem.SetLiteralAttribute(a.Name(), a.Value())
		}
	}

	return elem, nil
}

// addInScopeNamespaces copies all in-scope namespaces from src to dst that
// are not already declared on dst.  This ensures that inherited namespace
// bindings visible on the source element are preserved on the copy.
func addInScopeNamespaces(src, dst *helium.Element) {
	declared := make(map[string]struct{})
	for _, ns := range dst.Namespaces() {
		declared[ns.Prefix()] = struct{}{}
	}
	for _, ns := range collectInScopeNamespaces(src) {
		if _, exists := declared[ns.Prefix()]; !exists {
			_ = dst.DeclareNamespace(ns.Prefix(), ns.URI())
			declared[ns.Prefix()] = struct{}{}
		}
	}
}

// regex-group(n) returns the nth captured group from the current regex match.
// Returns empty string if called outside xsl:matching-substring or if the
// group number is out of range.
func (ec *execContext) fnRegexGroup(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || (args[0] == nil || sequence.Len(args[0]) == 0) {
		return xpath3.SingleString(""), nil
	}
	av, err := xpath3.AtomizeItem(args[0].Get(0))
	if err != nil {
		return xpath3.SingleString(""), nil
	}
	s, err := xpath3.AtomicToString(av)
	if err != nil {
		return xpath3.SingleString(""), nil
	}
	idx, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return xpath3.SingleString(""), nil
	}
	if idx < 0 || idx >= len(ec.regexGroups) {
		return xpath3.SingleString(""), nil
	}
	return xpath3.SingleString(ec.regexGroups[idx]), nil
}

// xsltFunctionsNS returns user-defined xsl:function definitions and
// XSLT built-in functions that need to be callable in the fn: namespace
// as xpath3 functions keyed by qualified name.
