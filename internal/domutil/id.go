package domutil

import (
	"context"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
)

// isIDAttribute reports whether attr is treated as an ID attribute under the
// FROZEN name/type rule documented on FindElementsByID. Both FindElementsByID
// and BuildIDIndex apply it identically, so there is exactly one place the
// rule lives.
func isIDAttribute(attr *helium.Attribute) bool {
	name := attr.Name()
	return name == "Id" || name == "ID" || name == "id" || name == "xml:id" || attr.AType() == enum.AttrID
}

// FindElementsByID walks the subtree rooted at root (root included) and
// returns every element whose ID matches the given value. root may be nil, in
// which case it returns no matches. The walk is exhaustive — it never
// short-circuits — so that duplicate IDs are surfaced to the caller and never
// silently masked. We do NOT consult Document.GetElementByID: its
// underlying ID table is keyed by ID value and Document.RegisterID
// overwrites on collision, which would hide the duplicate-xml:id case that
// XSW hardening relies on.
//
// An attribute is treated as an ID when it is any of:
//   - declared ID-typed by a DTD or schema (AType == enum.AttrID);
//   - xml:id (ID-typed by the W3C xml:id Recommendation);
//   - the "id" attribute token in the casings "Id", "ID", or "id".
//
// This name set is FROZEN: it recognizes the "id" identifier token in the
// three casings above plus xml:id, and MUST NOT grow to distinct convention
// tokens such as "wsu:Id" or "AssertionID". Those are not universal ID names
// — they are ID-typed only by their own schemas — so a document that relies
// on them must declare that typing (DTD/schema, or by marking the attribute
// AType == enum.AttrID), which this heuristic will never guess for it.
func FindElementsByID(root helium.Node, id string) []*helium.Element {
	var matches []*helium.Element
	var walk func(helium.Node)
	walk = func(n helium.Node) {
		elem, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			return
		}
		for _, attr := range elem.Attributes() {
			if !isIDAttribute(attr) {
				continue
			}
			// xs:ID derives from xs:NCName, which collapses whitespace;
			// match libxml2/helium normalization for xml:id.
			if strings.TrimSpace(attr.Value()) == id {
				matches = append(matches, elem)
				break
			}
		}
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			walk(child)
		}
	}
	walk(root)
	return matches
}

// BuildIDIndex walks the subtree rooted at root (root included) exactly once
// and returns a map from every id value found to every element carrying it,
// applying the identical FROZEN ID-name rule documented on FindElementsByID.
// It exists so a caller resolving many ids against the same subtree pays for
// one walk in place of one per id, while preserving the same exhaustive,
// non-short-circuiting duplicate collection: a caller can refuse a duplicate
// id by checking len(index[id]) > 1 exactly as a single FindElementsByID call
// would report it.
//
// ctx is observed once per node visited and its error is returned as soon as
// it is done, because the trip count of this walk is the size of a document the
// caller did not write. A cancelled walk returns a nil index, never a
// partial one, so a caller cannot mistake "the id is absent" for "the walk
// stopped before reaching it". A nil ctx is treated as uncancelled, matching
// the packages whose own entry points accept one.
func BuildIDIndex(ctx context.Context, root helium.Node) (map[string][]*helium.Element, error) {
	index := make(map[string][]*helium.Element)
	var walk func(helium.Node) error
	walk = func(n helium.Node) error {
		if ctx != nil {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		elem, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			return nil
		}
		for _, attr := range elem.Attributes() {
			if !isIDAttribute(attr) {
				continue
			}
			// xs:ID derives from xs:NCName, which collapses whitespace;
			// match libxml2/helium normalization for xml:id.
			id := strings.TrimSpace(attr.Value())
			// Every id on the element is recorded, so an element carrying
			// two ID attributes cannot hide the second one from a caller
			// checking len(index[id]) > 1. Only a repeat of the SAME id on
			// the SAME element is dropped, matching the value-conditional
			// break in FindElementsByID. Checking the last entry suffices:
			// the walk runs one element's attributes to completion before
			// descending into its children, so an element's appends for a
			// given id are never interleaved with another element's.
			existing := index[id]
			if len(existing) > 0 && existing[len(existing)-1] == elem {
				continue
			}
			index[id] = append(index[id], elem)
		}
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(root); err != nil {
		return nil, err
	}
	return index, nil
}
