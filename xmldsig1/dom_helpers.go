package xmldsig1

import (
	helium "github.com/lestrrat-go/helium"
)

// sigAnchor remembers a Signature element's exact location in its sibling
// chain so it can be reinserted in the same spot after a temporary detach
// during enveloped-signature processing. Without this, naive
// detach/AddChild reattachment moves the Signature to the end of its
// parent's child list and silently restructures the document — a quiet
// corruption that confuses downstream consumers and can mask XSW shapes.
type sigAnchor struct {
	parent      *helium.Element
	nextSibling helium.Node // nil if Signature was the last child
}

// captureAnchor records the current location of sigElem.
func captureAnchor(sigElem *helium.Element) sigAnchor {
	a := sigAnchor{}
	if p, ok := helium.AsNode[*helium.Element](sigElem.Parent()); ok {
		a.parent = p
	}
	a.nextSibling = sigElem.NextSibling()
	return a
}

// restore reattaches sigElem at the anchored position. If nextSibling is
// nil, the node is appended at the end (equivalent to AddChild). Otherwise
// the node is spliced in before nextSibling, preserving the original
// document layout.
func (a sigAnchor) restore(sigElem *helium.Element) error {
	if a.parent == nil {
		return nil
	}
	if a.nextSibling == nil {
		return a.parent.AddChild(sigElem)
	}
	return insertBefore(a.parent, sigElem, a.nextSibling)
}

// insertBefore inserts newChild into parent's child list immediately before
// ref. ref must be a current child of parent and newChild must currently
// be detached. Implemented via MutableNode.Replace so we don't depend on
// helium's unexported docnode internals.
func insertBefore(parent *helium.Element, newChild *helium.Element, ref helium.Node) error {
	refMut, ok := ref.(helium.MutableNode)
	if !ok {
		// Fall back to AddChild (appends) so we don't lose the node.
		return parent.AddChild(newChild)
	}
	// Replace ref with [newChild, ref] — Replace patches parent's
	// firstChild/lastChild pointers and rewrites the sibling chain
	// correctly even when ref is the first child of parent.
	return refMut.Replace(newChild, ref)
}
