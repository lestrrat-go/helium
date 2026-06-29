package relaxng

import "github.com/lestrrat-go/helium/internal/xsdregex"

// Grammar is a compiled RELAX NG schema, analogous to [xsd.Schema].
// (libxml2: xmlRelaxNGPtr)
type Grammar struct {
	start   *pattern
	defines map[string]*pattern
}

// patternKind enumerates RELAX NG pattern types.
type patternKind int

const (
	patternEmpty patternKind = iota
	patternNotAllowed
	patternText
	patternElement
	patternAttribute
	patternGroup
	patternInterleave
	patternChoice
	patternOptional
	patternZeroOrMore
	patternOneOrMore
	patternRef
	patternParentRef
	patternExternalRef
	patternData
	patternValue
	patternList
	patternMixed
	patternGrammar
)

// pattern is a node in the compiled RELAX NG pattern tree.
type pattern struct {
	kind      patternKind
	name      string     // element/attribute local name (or define name for ref)
	ns        string     // namespace URI
	value     string     // for value patterns
	dataType  *dataType  // for data/value patterns
	children  []*pattern // child patterns (group/choice/interleave members, element content, etc.)
	attrs     []*pattern // attribute patterns (for element)
	nameClass *nameClass // for element/attribute name matching
	params    []*param   // for data patterns
	line      int        // source line number
	// resolved is the define pattern a patternRef/patternParentRef points at,
	// fixed at compile time against the node's lexical grammar scope (ref →
	// current grammar scope, parentRef → parent grammar scope). Validation and
	// the compile-time ref checks follow this pointer instead of doing a
	// by-name lookup in a single flat global define map, so nested grammars
	// that reuse a define name keep distinct scopes.
	resolved *pattern
}

// dataType identifies a datatype from a datatype library.
type dataType struct {
	library string // datatype library URI
	name    string // type name (e.g. "integer", "string")
	// libraryDeclared records whether datatypeLibrary was explicitly present on
	// this element or an ancestor (even as ""). It distinguishes a truly-absent
	// library (no datatypeLibrary anywhere) from an explicit datatypeLibrary=""
	// reset under an inherited library. Only the absent case enables the
	// libxml2-compat bare-XSD-name fallback in matchData/matchValue; an explicit
	// "" reset selects the built-in library and rejects bare XSD names.
	libraryDeclared bool
}

// param is a facet parameter for data patterns.
type param struct {
	name  string
	value string

	// compiledPattern is the XSD-regex compilation of a "pattern" facet's value,
	// populated once at compile time (checkDataFacets) so validation reuses it
	// instead of recompiling per value. patternChecked guards that one-time work
	// (and its error report) against a data pattern reached more than once through
	// shared <ref> definitions.
	compiledPattern *xsdregex.Regexp
	patternChecked  bool
}

// nameClassKind enumerates name class types.
type nameClassKind int

const (
	ncName nameClassKind = iota
	ncAnyName
	ncNsName
	ncChoice
	// ncNoMatch never matches any name. It is installed for a schema name
	// whose prefix is unbound or whose lexical form is not a valid NCName, so
	// that even on the default (no error collector) compile path validation
	// cannot spuriously succeed against an unintended no-namespace name.
	ncNoMatch
)

// nameClass represents an element/attribute name class for matching.
type nameClass struct {
	kind   nameClassKind
	name   string     // for ncName
	ns     string     // for ncName, ncNsName
	left   *nameClass // for ncChoice
	right  *nameClass // for ncChoice
	except *nameClass // for ncAnyName, ncNsName
}

// collectAttrPatternsFlat recursively extracts all patternAttribute nodes from
// a pattern slice, walking into wrapper patterns (zeroOrMore, oneOrMore,
// optional, group, interleave). Does NOT walk into choice because attributes
// in different choice branches are alternatives and cannot conflict.
func collectAttrPatternsFlat(pats []*pattern) []*pattern {
	var result []*pattern
	for _, p := range pats {
		if p == nil {
			continue
		}
		switch p.kind {
		case patternAttribute:
			result = append(result, p)
		case patternZeroOrMore, patternOneOrMore, patternOptional,
			patternGroup, patternInterleave:
			result = append(result, collectAttrPatternsFlat(p.children)...)
			result = append(result, collectAttrPatternsFlat(p.attrs)...)
		}
	}
	return result
}

// nameClassesOverlap returns true if two name classes can potentially match
// the same attribute name. Uses conservative analysis (anyName overlaps with
// everything regardless of except clauses).
func nameClassesOverlap(a, b *nameClass) bool {
	if a == nil || b == nil {
		return false
	}

	// ncChoice: overlap if either branch overlaps. Decompose choices before the
	// anyName handling below so each ncName leaf is tested against the other
	// class's <except>. Otherwise a choice of names that are all excluded by an
	// anyName <except> would be wrongly flagged as overlapping, because the
	// anyName branch short-circuits before reaching the choice decomposition.
	if a.kind == ncChoice {
		return nameClassesOverlap(a.left, b) || nameClassesOverlap(a.right, b)
	}
	if b.kind == ncChoice {
		return nameClassesOverlap(a, b.left) || nameClassesOverlap(a, b.right)
	}

	// anyName: anyName-except-E matches every name NOT in E, so it is disjoint
	// from the other class b exactly when E fully CONTAINS b (every name b can
	// match is excluded). This generalises the single-ncName case to nsName and
	// choice — e.g. anyName except nsName(X) does not overlap nsName(X).
	if a.kind == ncAnyName {
		if a.except != nil && nameClassContains(a.except, b) {
			return false
		}
		return true
	}
	if b.kind == ncAnyName {
		if b.except != nil && nameClassContains(b.except, a) {
			return false
		}
		return true
	}

	// nsName vs nsName
	if a.kind == ncNsName && b.kind == ncNsName {
		return a.ns == b.ns
	}

	// nsName vs ncName (with except support)
	if a.kind == ncNsName && b.kind == ncName {
		if a.ns != b.ns {
			return false
		}
		if a.except != nil && nameClassMatches(a.except, b.name, b.ns) {
			return false
		}
		return true
	}
	if a.kind == ncName && b.kind == ncNsName {
		if a.ns != b.ns {
			return false
		}
		if b.except != nil && nameClassMatches(b.except, a.name, a.ns) {
			return false
		}
		return true
	}

	// ncName vs ncName
	if a.kind == ncName && b.kind == ncName {
		return a.name == b.name && a.ns == b.ns
	}

	return false
}

// nameClassContains reports whether outer definitely matches every name that
// inner can match (outer ⊇ inner). It is CONSERVATIVE: it returns true only
// when containment is certain, so a caller subtracting an <except> never
// concludes "disjoint" for a pair that might actually overlap. Any inner
// <except> only shrinks inner, so it is safe to ignore for containment.
func nameClassContains(outer, inner *nameClass) bool {
	if outer == nil || inner == nil {
		return false
	}
	switch inner.kind {
	case ncChoice:
		return nameClassContains(outer, inner.left) && nameClassContains(outer, inner.right)
	case ncName:
		return nameClassMatches(outer, inner.name, inner.ns)
	case ncNsName:
		// inner = nsName(inner.ns) minus inner.except, so outer need only cover
		// the names inner actually matches — namespace inner.ns with inner.except
		// removed — NOT all of inner.ns. Threading inner.except lets an outer
		// nsName/anyName carrying its OWN finite except still contain inner.
		return nameClassCoversNSExcept(outer, inner.ns, inner.except)
	case ncAnyName:
		return nameClassCoversAll(outer)
	}
	return false
}

// nameClassCoversNSExcept reports whether outer certainly matches every name in
// namespace ns that is NOT matched by innerExcept (i.e. outer ⊇ nsName(ns)
// except innerExcept). A finite set of ncName leaves can never cover an
// (infinite) namespace, so only an anyName/nsName whose own except removes
// nothing from ns that innerExcept does not already remove, or a choice
// containing one of those, qualifies. When innerExcept is nil this reduces to
// "outer covers every name in ns".
func nameClassCoversNSExcept(outer *nameClass, ns string, innerExcept *nameClass) bool {
	if outer == nil {
		return false
	}
	switch outer.kind {
	case ncAnyName:
		// anyName except outer.except covers (ns \ innerExcept) iff every name
		// IN ns that outer.except removes is already removed by innerExcept.
		// Names outer.except removes OUTSIDE ns are irrelevant — outer never
		// needed to match them within ns — so only outer.except ∩ ns matters.
		if outer.except == nil {
			return true
		}
		return nameClassCoversWithinNS(innerExcept, outer.except, ns)
	case ncNsName:
		if outer.ns != ns {
			return false
		}
		if outer.except == nil {
			return true
		}
		return nameClassCoversWithinNS(innerExcept, outer.except, ns)
	case ncChoice:
		// A single branch may cover ns\innerExcept on its own...
		if nameClassCoversNSExcept(outer.left, ns, innerExcept) ||
			nameClassCoversNSExcept(outer.right, ns, innerExcept) {
			return true
		}
		// ...or the branches may cover it only by UNION. e.g.
		// (nsName(X) except foo) | name(foo) covers all of nsName(X): the
		// nsName branch matches everything in X but foo, and a sibling branch
		// fills the foo gap.
		return choiceCoversNSByUnion(outer, ns, innerExcept)
	}
	return false
}

// nameClassCoversWithinNS reports whether cover certainly matches every name in
// namespace ns that sub matches — i.e. cover ⊇ (sub ∩ ns). Names sub matches
// OUTSIDE ns are ignored: when establishing that an outer class covers
// nsName(ns) minus some except, an excluded name in a DIFFERENT namespace can
// never have been in ns to begin with, so it does not need to be re-covered.
// This is what makes e.g. anyName except (nsName(X) except name(Y:foo)) cover
// nsName(X): within X, the inner except removes nothing (Y:foo ∉ X), so the
// excluded class is all of X and the anyName matches no name in X. Conservative:
// returns true only when coverage within ns is certain.
func nameClassCoversWithinNS(cover, sub *nameClass, ns string) bool {
	if sub == nil {
		// sub matches nothing, so there is nothing in ns to cover.
		return true
	}
	switch sub.kind {
	case ncChoice:
		return nameClassCoversWithinNS(cover, sub.left, ns) &&
			nameClassCoversWithinNS(cover, sub.right, ns)
	case ncName:
		if sub.ns != ns {
			// A single name outside ns is not part of sub ∩ ns.
			return true
		}
		return cover != nil && nameClassMatches(cover, sub.name, sub.ns)
	case ncNsName:
		if sub.ns != ns {
			// sub matches only its own namespace, which is not ns.
			return true
		}
		// sub ∩ ns = nsName(ns) minus sub.except; cover must cover that.
		return nameClassCoversNSExcept(cover, ns, sub.except)
	case ncAnyName:
		// sub ∩ ns = nsName(ns) minus sub.except; cover must cover that.
		return nameClassCoversNSExcept(cover, ns, sub.except)
	}
	return false
}

// choiceCoversNSByUnion reports whether the UNION of a choice's branches
// certainly matches every name in namespace ns that innerExcept does not
// remove. It handles the disjoint-union case nameClassCoversNSExcept's
// branch-by-branch test misses: one branch is nsName(ns) minus a FINITE set of
// names, and sibling branches match each of those removed names. Soundness
// rests on the gap being finitely enumerable — if the nsName branch's own
// except is not a finite set of ncName leaves, no claim is made.
func choiceCoversNSByUnion(choice *nameClass, ns string, innerExcept *nameClass) bool {
	for _, branch := range flattenChoiceBranches(choice) {
		if branch.kind != ncNsName || branch.ns != ns {
			continue
		}
		if branch.except == nil {
			return true
		}
		gap, ok := collectFiniteNames(branch.except)
		if !ok {
			// The branch removes an infinite/unknown set; the gap cannot be
			// enumerated, so coverage cannot be claimed soundly.
			continue
		}
		covered := true
		for _, n := range gap {
			if n.ns != ns {
				// A name outside ns the branch excludes is irrelevant: the
				// nsName branch never matched it within ns anyway.
				continue
			}
			if innerExcept != nil && nameClassMatches(innerExcept, n.name, n.ns) {
				// innerExcept already removes this name, so it need not be
				// covered.
				continue
			}
			// Some OTHER branch must match the gap name; the nsName branch
			// itself excludes it by construction.
			if !nameClassMatches(choice, n.name, n.ns) {
				covered = false
				break
			}
		}
		if covered {
			return true
		}
	}
	return false
}

// ncQName is a fully-qualified name (local + namespace) used when enumerating a
// finite name set out of a name class.
type ncQName struct {
	name string
	ns   string
}

// collectFiniteNames returns the finite set of names a name class matches, or
// ok=false when the class is not a finite union of ncName leaves (i.e. it
// contains an nsName/anyName/ncNoMatch and so matches an infinite or
// indeterminate set).
func collectFiniteNames(nc *nameClass) ([]ncQName, bool) {
	if nc == nil {
		return nil, true
	}
	switch nc.kind {
	case ncName:
		return []ncQName{{name: nc.name, ns: nc.ns}}, true
	case ncChoice:
		left, ok := collectFiniteNames(nc.left)
		if !ok {
			return nil, false
		}
		right, ok := collectFiniteNames(nc.right)
		if !ok {
			return nil, false
		}
		return append(left, right...), true
	}
	return nil, false
}

// flattenChoiceBranches collapses a (possibly nested) ncChoice tree into its
// leaf branches.
func flattenChoiceBranches(nc *nameClass) []*nameClass {
	if nc == nil {
		return nil
	}
	if nc.kind != ncChoice {
		return []*nameClass{nc}
	}
	return append(flattenChoiceBranches(nc.left), flattenChoiceBranches(nc.right)...)
}

// nameClassCoversAll reports whether outer certainly matches every possible
// name (only an except-free anyName, or a choice containing one).
func nameClassCoversAll(outer *nameClass) bool {
	if outer == nil {
		return false
	}
	switch outer.kind {
	case ncAnyName:
		return outer.except == nil
	case ncChoice:
		return nameClassCoversAll(outer.left) || nameClassCoversAll(outer.right)
	}
	return false
}

// nameClassContainsNoMatch reports whether the name class tree contains a
// poisoned (ncNoMatch) leaf. It is used to detect an invalid name class nested
// inside an <except>, which must poison the enclosing anyName/nsName rather than
// be silently treated as an empty exclusion.
func nameClassContainsNoMatch(nc *nameClass) bool {
	if nc == nil {
		return false
	}
	switch nc.kind {
	case ncNoMatch:
		return true
	case ncChoice:
		return nameClassContainsNoMatch(nc.left) || nameClassContainsNoMatch(nc.right)
	case ncAnyName, ncNsName:
		return nameClassContainsNoMatch(nc.except)
	}
	return false
}

// nameClassMatches returns true if the name class matches the given local name and namespace.
func nameClassMatches(nc *nameClass, local, ns string) bool {
	if nc == nil {
		return false
	}
	switch nc.kind {
	case ncName:
		return nc.name == local && nc.ns == ns
	case ncAnyName:
		if nc.except != nil && nameClassMatches(nc.except, local, ns) {
			return false
		}
		return true
	case ncNsName:
		if ns != nc.ns {
			return false
		}
		if nc.except != nil && nameClassMatches(nc.except, local, ns) {
			return false
		}
		return true
	case ncChoice:
		return nameClassMatches(nc.left, local, ns) || nameClassMatches(nc.right, local, ns)
	}
	return false
}
