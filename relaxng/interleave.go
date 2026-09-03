package relaxng

import (
	"context"

	helium "github.com/lestrrat-go/helium"
)

// interleavePartition is the compile-time routing table of one <interleave>
// (libxml2: xmlRelaxNGPartition). Every child node of the validated element is
// routed to the one branch whose reachable name classes accept it; RELAX NG
// §7.4 guarantees the branches are pairwise disjoint, so the route is exact.
type interleavePartition struct {
	branches []*interleaveBranch
	byName   map[ncQName]int // exact element names → branch index
	wild     []int           // branches carrying a non-finite element name class, scanned in order
	text     int             // branch accepting text nodes, or -1
}

// interleaveBranch is the set of leaves one branch can start with, reachable
// without crossing an element or attribute boundary (libxml2:
// xmlRelaxNGGetElements with eora 2 and 1).
type interleaveBranch struct {
	elems []*nameClass
	attrs []*nameClass
	text  bool // a text, data, value or list leaf
}

// elementNameClass returns the name class an element pattern matches with.
func elementNameClass(p *pattern) *nameClass {
	if p.nameClass != nil {
		return p.nameClass
	}
	if p.name == "" {
		return &nameClass{kind: ncNoMatch}
	}
	return &nameClass{kind: ncName, name: p.name, ns: p.ns}
}

// collectInterleaveLeaves gathers the leaves of one interleave branch. It
// descends through the composite patterns and through resolved refs (each
// define once per branch), and stops at element and attribute boundaries.
func collectInterleaveLeaves(p *pattern, b *interleaveBranch, visited map[*pattern]struct{}) {
	if p == nil {
		return
	}
	switch p.kind {
	case patternElement:
		b.elems = append(b.elems, elementNameClass(p))
	case patternAttribute:
		if p.nameClass != nil {
			b.attrs = append(b.attrs, p.nameClass)
		}
	case patternText, patternData, patternValue, patternList:
		b.text = true
	case patternRef, patternParentRef:
		def := p.resolved
		if def == nil {
			return
		}
		if _, seen := visited[def]; seen {
			return
		}
		visited[def] = struct{}{}
		collectInterleaveLeaves(def, b, visited)
	case patternGroup, patternChoice, patternInterleave, patternOptional, patternZeroOrMore, patternOneOrMore:
		for _, child := range p.children {
			collectInterleaveLeaves(child, b, visited)
		}
	}
}

// interleaveBranchesConflict reports whether two branches violate RELAX NG
// §7.4: an element name class in common, or text in both. It uses
// nameClassesOverlap, which is sound (it never reports "disjoint" for classes
// that can share a name) — this is what makes routing exact — but
// conservative on exotic <except> shapes, which could reject a grammar
// libxml2 accepts.
func interleaveBranchesConflict(a, b *interleaveBranch) bool {
	if a.text && b.text {
		return true
	}
	for _, x := range a.elems {
		for _, y := range b.elems {
			if nameClassesOverlap(x, y) {
				return true
			}
		}
	}
	return false
}

// interleaveAttrsConflict reports whether two branches claim an overlapping
// attribute name class.
func interleaveAttrsConflict(a, b *interleaveBranch) bool {
	for _, x := range a.attrs {
		for _, y := range b.attrs {
			if nameClassesOverlap(x, y) {
				return true
			}
		}
	}
	return false
}

// computeInterleavePartition builds the routing table for one interleave
// pattern. For each branch, an element leaf whose name class is a finite
// union of names is entered into byName (first writer wins — a duplicate can
// only happen on a conflicting grammar, which fails to compile); a leaf whose
// name class is not a finite union (nsName/anyName) marks the branch wild.
func computeInterleavePartition(p *pattern) *interleavePartition {
	part := &interleavePartition{byName: make(map[ncQName]int), text: -1}
	for i, child := range p.children {
		b := &interleaveBranch{}
		collectInterleaveLeaves(child, b, make(map[*pattern]struct{}))
		part.branches = append(part.branches, b)
		if b.text && part.text < 0 {
			part.text = i
		}
		wild := false
		for _, nc := range b.elems {
			if nc.kind == ncNoMatch {
				continue
			}
			names, ok := collectFiniteNames(nc)
			if !ok {
				wild = true
				continue
			}
			for _, n := range names {
				if _, dup := part.byName[n]; !dup {
					part.byName[n] = i
				}
			}
		}
		if wild {
			part.wild = append(part.wild, i)
		}
	}
	return part
}

// checkInterleaves computes every recorded interleave's partition and reports
// RELAX NG §7.4 conflicts (libxml2: xmlRelaxNGComputeInterleaves). Conflicts
// are reported only on a grammar with no earlier compile error, matching
// libxml2's gate.
func (c *compiler) checkInterleaves(ctx context.Context) {
	report := c.errorCount == 0
	for _, p := range c.interleaves {
		part := computeInterleavePartition(p)
		p.partition = part
		if !report {
			continue
		}
		elemConflict, attrConflict := false, false
		for i := range part.branches {
			for j := i + 1; j < len(part.branches); j++ {
				if interleaveBranchesConflict(part.branches[i], part.branches[j]) {
					elemConflict = true
				}
				if interleaveAttrsConflict(part.branches[i], part.branches[j]) {
					attrConflict = true
				}
			}
		}
		if elemConflict {
			c.addPatternError(ctx, p, "Element or text conflicts in interleave")
		}
		if attrConflict {
			c.addPatternError(ctx, p, "Attributes conflicts in interleave")
		}
	}
}

// route returns the branch index for node, -1 when no branch accepts it, or
// -2 for a node that is dropped without being routed (a comment, or a
// whitespace-only text node when no branch accepts text).
func (p *interleavePartition) route(node helium.Node) int {
	switch node.Type() {
	case helium.CommentNode:
		return -2
	case helium.TextNode, helium.CDATASectionNode:
		if p.text >= 0 {
			return p.text
		}
		if t, ok := node.(*helium.Text); ok && isXMLSpaceOnly(string(t.Content())) {
			return -2
		}
		return -1
	case helium.ElementNode:
		e, ok := node.(*helium.Element)
		if !ok {
			return -1
		}
		local, ns := e.LocalName(), elemNS(e)
		if i, ok := p.byName[ncQName{name: local, ns: ns}]; ok {
			return i
		}
		for _, i := range p.wild {
			for _, nc := range p.branches[i].elems {
				if nameClassMatches(nc, local, ns) {
					return i
				}
			}
		}
		return -1
	}
	return -1
}
