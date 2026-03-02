package relaxng

// Grammar is a compiled RELAX NG schema, analogous to [xsd.Schema].
// (libxml2: xmlRelaxNGPtr)
type Grammar struct {
	start           *pattern
	defines         map[string]*pattern
	compileErrors   string
	compileWarnings string
}

// CompileErrors returns any schema compilation error messages
// in libxml2-compatible format. Empty string means no errors.
func (g *Grammar) CompileErrors() string {
	return g.compileErrors
}

// CompileWarnings returns any schema compilation warning messages.
func (g *Grammar) CompileWarnings() string {
	return g.compileWarnings
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
}

// dataType identifies a datatype from a datatype library.
type dataType struct {
	library string // datatype library URI
	name    string // type name (e.g. "integer", "string")
}

// param is a facet parameter for data patterns.
type param struct {
	name  string
	value string
}

// nameClassKind enumerates name class types.
type nameClassKind int

const (
	ncName nameClassKind = iota
	ncAnyName
	ncNsName
	ncChoice
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

	// anyName: check if except clause excludes the other name class
	if a.kind == ncAnyName {
		if a.except != nil && b.kind == ncName {
			if nameClassMatches(a.except, b.name, b.ns) {
				return false
			}
		}
		return true
	}
	if b.kind == ncAnyName {
		if b.except != nil && a.kind == ncName {
			if nameClassMatches(b.except, a.name, a.ns) {
				return false
			}
		}
		return true
	}

	// ncChoice: overlap if either branch overlaps
	if a.kind == ncChoice {
		return nameClassesOverlap(a.left, b) || nameClassesOverlap(a.right, b)
	}
	if b.kind == ncChoice {
		return nameClassesOverlap(a, b.left) || nameClassesOverlap(a, b.right)
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
