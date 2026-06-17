package xmldsig1

import (
	"encoding/base64"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

// decodeBase64 decodes the base64 text of an XML Signature field. XML Signature
// base64 fields (SignatureValue, DigestValue, X509Certificate, key-value
// components, ...) are typed xs:base64Binary, whose lexical space permits
// interspersed XML whitespace; real-world signers line-wrap and indent base64.
// Go's base64 decoder happens to skip CR/LF but rejects space and tab, so all
// four XML whitespace characters (space, tab, CR, LF) are stripped before
// decoding. No other characters are removed, so invalid base64 still fails.
func decodeBase64(text string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(stripXMLWhitespace(text))
}

// stripXMLWhitespace removes the four XML whitespace characters (space 0x20,
// tab 0x09, CR 0x0D, LF 0x0A) from s.
func stripXMLWhitespace(s string) string {
	if !strings.ContainsAny(s, " \t\r\n") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := range len(s) {
		switch c := s[i]; c {
		case ' ', '\t', '\r', '\n':
			// drop XML whitespace
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// sigAnchor remembers a Signature element's exact location in its sibling
// chain so it can be reinserted in the same spot after a temporary detach
// during enveloped-signature processing. Without this, naive
// detach/AddChild reattachment moves the Signature to the end of its
// parent's child list and silently restructures the document — a quiet
// corruption that confuses downstream consumers and can mask XSW shapes.
//
// parent is stored as MutableNode (rather than *Element) so that we can
// also anchor a Signature whose parent is the Document itself — i.e., a
// document whose root element is <ds:Signature>. If parent were typed as
// *Element, that case would silently drop the anchor and the detached
// Signature would never be reattached.
type sigAnchor struct {
	parent      helium.MutableNode
	nextSibling helium.Node // nil if Signature was the last child
}

// captureAnchor records the current location of sigElem.
func captureAnchor(sigElem *helium.Element) sigAnchor {
	a := sigAnchor{}
	if p, ok := sigElem.Parent().(helium.MutableNode); ok {
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
func insertBefore(parent helium.MutableNode, newChild *helium.Element, ref helium.Node) error {
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
