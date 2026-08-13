package xmlenc1

import (
	"context"
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
)

// refScope is the same-document resolution context of one decrypt: the subtree
// a reference may name, the id index over it, and the targets already taken.
// One scope is built per Decrypt/DecryptBytes call and threaded through the
// parse on parseState, so N references cost one document walk rather than N.
//
// It is per-call state and never shared between calls, which is what keeps a
// Decryptor safe to fire from several goroutines: the index and the visited
// set below are both written during resolution.
type refScope struct {
	// root is the subtree a same-document URI may name. It is the TOPMOST
	// ELEMENT ancestor of the EncryptedData, not OwnerDocument's document
	// element: an EncryptedData a caller holds detached from any document
	// still resolves references into its own subtree, which is where its
	// EncryptedKey then has to be. A document element's own walk up ends at
	// itself, so an attached EncryptedData sees the whole document.
	root *helium.Element

	// baseURI is the base URI of the EncryptedData, used to name the document
	// a refused reference was refused in. It takes no part in resolution:
	// XMLDSig core §4.4.3.2 defines a same-document reference as URI="" or a
	// bare "#..." fragment, so nothing here is ever resolved RELATIVE to a
	// base, and a URI carrying a path is external however that path compares
	// to this one.
	baseURI string

	// index maps every id in root's subtree to every element carrying it. It
	// is nil until the first reference actually needs it and is built exactly
	// once thereafter, so a document whose references are all skipped (a Type
	// this package does not implement) never pays for the walk.
	index map[string][]*helium.Element

	// visited records the EncryptedKey elements already taken as candidates.
	// xmlenc-core1 §3.5.3 permits several ds:RetrievalMethods, and two of them
	// naming one EncryptedKey describe one key: it must be charged and
	// trial-decrypted once, not once per reference. This is DEDUPLICATION, not
	// a loop guard — see parseRetrievalMethod for why no cycle is expressible.
	visited map[*helium.Element]struct{}
}

// newRefScope builds the resolution scope for a decrypt of elem. A nil elem
// yields a nil scope, which resolves nothing; the parse rejects a nil
// EncryptedData before any reference is read.
func newRefScope(elem *helium.Element) *refScope {
	if elem == nil {
		return nil
	}
	root := elem
	for parent := elem.Parent(); parent != nil; parent = parent.Parent() {
		anc, ok := helium.AsNode[*helium.Element](parent)
		if !ok {
			// The Document node above a document element, and nothing else in
			// a well-formed tree. Either way the element below it is the top.
			break
		}
		root = anc
	}
	return &refScope{
		root:    root,
		baseURI: helium.NodeGetBase(elem.OwnerDocument(), elem),
		visited: make(map[*helium.Element]struct{}),
	}
}

// markVisited records elem as taken and reports whether it already was, so a
// second reference to one EncryptedKey adds no second candidate. A nil scope
// records nothing and reports nothing as seen.
func (s *refScope) markVisited(elem *helium.Element) bool {
	if s == nil {
		return false
	}
	if _, seen := s.visited[elem]; seen {
		return true
	}
	s.visited[elem] = struct{}{}
	return false
}

// in names the document a reference was refused in, for the two reference
// errors. It is empty when the document has no base URI, which is the ordinary
// case for a document parsed from memory.
func (s *refScope) in() string {
	if s == nil || s.baseURI == "" {
		return ""
	}
	return fmt.Sprintf(" in %q", s.baseURI)
}

// referenceURIForm classifies a URI into the same-document forms XMLDSig core
// defines (§4.4.3.2-3), fail-closed. It reports:
//
//   - id: the bare id to resolve (empty for the whole-document forms);
//   - wholeDoc: the reference selects the root of the document;
//   - ok: the URI is a same-document reference at all (false → the caller
//     fails closed with ErrReferenceNotFound).
//
// The four recognized forms are:
//
//   - URI=""                     → the whole document.
//   - URI="#id"                  → the element with that id.
//   - URI="#xpointer(/)"         → the whole document.
//   - URI="#xpointer(id('id'))"  → the element with that id.
//
// Every other URI — an external reference, or any other #xpointer(...) scheme
// — is not a same-document reference and stays fail-closed. This mirrors
// xmldsig1's classification of the same four forms; it reports no
// comment-inclusion flag, because a ds:RetrievalMethod resolves to a single
// ELEMENT rather than to a node-set, so the comment semantics that distinguish
// the bare-name forms from the XPointer ones make no difference here.
func referenceURIForm(uri string) (string, bool, bool) {
	if uri == "" {
		return "", true, true
	}
	if !strings.HasPrefix(uri, "#") {
		// External reference: not supported.
		return "", false, false
	}
	frag := uri[1:]
	if !strings.HasPrefix(frag, "xpointer(") {
		// Bare-name "#id" (no XPointer scheme). Any "#name" without a "(" is
		// a bare id.
		if strings.ContainsAny(frag, "()") {
			return "", false, false
		}
		return frag, false, true
	}
	// Full XPointer form: #xpointer(<expr>). Only the two bare-names
	// SHOULD-support schemes are honored.
	if !strings.HasSuffix(frag, ")") {
		return "", false, false
	}
	expr := strings.TrimSpace(frag[len("xpointer(") : len(frag)-1])
	if expr == "/" {
		return "", true, true
	}
	if id, ok := parseXPointerID(expr); ok {
		return id, false, true
	}
	return "", false, false
}

// parseXPointerID matches the XPointer id() form id('X') or id("X") and returns
// the quoted id. Anything else (a bare argument, a nested call, an unbalanced or
// mismatched quote) is rejected so only the two SHOULD-support schemes resolve.
func parseXPointerID(expr string) (string, bool) {
	if !strings.HasPrefix(expr, "id(") || !strings.HasSuffix(expr, ")") {
		return "", false
	}
	arg := strings.TrimSpace(expr[len("id(") : len(expr)-1])
	if len(arg) < 2 {
		return "", false
	}
	q := arg[0]
	if (q != '\'' && q != '"') || arg[len(arg)-1] != q {
		return "", false
	}
	inner := arg[1 : len(arg)-1]
	// The id itself must not contain the quote character (no embedded quote /
	// second argument), keeping this a strict single-id match.
	if strings.IndexByte(inner, q) >= 0 {
		return "", false
	}
	return inner, true
}

// resolveSameDocument returns the single element a same-document URI names
// within scope, building scope's id index on first use.
//
// It fails closed in three ways, all with a sentinel the caller can match: a
// URI that is not a same-document reference, and one matching no element, are
// ErrReferenceNotFound; one matching more than one element is
// ErrAmbiguousReference, because taking either of two elements answering to one
// id would let whoever injected the second choose the key.
//
// The index is built through domutil.BuildIDIndex, so the ID-name rule here is
// the same FROZEN rule xmldsig1 resolves references under and the two cannot
// disagree about which elements carry an id. That walk observes ctx once per
// node, per this package's per-node polling rule: its trip count is the size of
// a document the caller did not write.
func resolveSameDocument(ctx context.Context, scope *refScope, uri string) (*helium.Element, error) {
	id, wholeDoc, ok := referenceURIForm(uri)
	if !ok {
		return nil, fmt.Errorf("%w: %q is not a same-document reference", ErrReferenceNotFound, uri)
	}
	if scope == nil || scope.root == nil {
		return nil, fmt.Errorf("%w: %q has no document to resolve against", ErrReferenceNotFound, uri)
	}
	if wholeDoc {
		return scope.root, nil
	}
	if scope.index == nil {
		index, err := domutil.BuildIDIndex(ctx, scope.root)
		if err != nil {
			return nil, err
		}
		scope.index = index
	}
	matches := scope.index[id]
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("%w: no element carries id %q%s", ErrReferenceNotFound, id, scope.in())
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("%w: id %q is carried by %d elements%s", ErrAmbiguousReference, id, len(matches), scope.in())
	}
}
