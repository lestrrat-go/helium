package relaxng

import (
	"context"
	"fmt"

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

// interleavePartitionOf returns the compiled partition, computing an
// uncached one for an interleave the compiler did not record. It never
// caches at validation time: Grammar is shared across goroutines.
func interleavePartitionOf(p *pattern) *interleavePartition {
	if p.partition != nil {
		return p.partition
	}
	return computeInterleavePartition(p)
}

// partitionInterleave splits state.seq into one sub-sequence per branch
// (libxml2: the triage loop of xmlRelaxNGValidateInterleave). Routing stops at
// the first node no branch accepts; that node and everything after it are left
// in state.seq for the caller.
func partitionInterleave(part *interleavePartition, state *validState) [][]helium.Node {
	subs := make([][]helium.Node, len(part.branches))
	seq := state.seq
	for len(seq) > 0 {
		i := part.route(seq[0])
		if i == -1 {
			break
		}
		if i >= 0 {
			subs[i] = append(subs[i], seq[0])
		}
		seq = seq[1:]
	}
	state.seq = seq
	return subs
}

// validateInterleaveContent validates an <interleave> inside element content:
// it routes state.seq once (partitionInterleave), then validates each branch
// independently against its own sub-sequence. Because §7.4 guarantees the
// branches are pairwise disjoint, the routing is exact: no branch can ever
// need a node routed to another branch.
//
// Each branch runs under v.suppressDepth++ (errors a branch appends outside a
// consumed element — an attribute pattern's own "failed to validate
// attributes", a group's "Expecting an element , got nothing" — would
// duplicate or contradict the lines the goldens expect). Errors appended
// AFTER validateElement has consumed an element are recorded regardless of
// suppression (validateElement resets suppressDepth to 0 for its own body), so
// the real cause of a branch whose element has bad content survives and
// precedes "Invalid sequence in interleave".
func (v *validator) validateInterleaveContent(pat *pattern, elem *helium.Element,
	attrs []*helium.Attribute, attrUsed []bool, state *validState) int {
	part := interleavePartitionOf(pat)
	before := state.seq
	subs := partitionInterleave(part, state)
	for i, child := range pat.children {
		sub := &validState{seq: subs[i], run: v.newRun()}
		errLen := len(v.pendingErrors)
		savedValid := v.valid
		savedAttrUsed := append([]bool(nil), attrUsed...)
		v.suppressDepth++
		ret := v.validateContentPat(child, elem, attrs, attrUsed, sub)
		v.suppressDepth--
		if ret == 0 && len(skipIgnored(sub.seq)) > 0 {
			// The branch matched but left part of its own sub-sequence behind. No
			// other branch can take those nodes (§7.4 makes the routing exact), so
			// retry with every choice picking the arm that consumes the most.
			sub = &validState{seq: subs[i], run: v.newRun()}
			copy(attrUsed, savedAttrUsed)
			v.pendingErrors = v.pendingErrors[:errLen]
			v.valid = savedValid
			v.suppressDepth++
			v.exactChoice++
			ret = v.validateContentPat(child, elem, attrs, attrUsed, sub)
			v.exactChoice--
			v.suppressDepth--
		}
		if ret != 0 {
			isAttr := child.kind == patternAttribute
			if len(v.pendingErrors) == errLen {
				v.valid = savedValid
				if !isAttr && len(skipIgnored(subs[i])) == 0 {
					if eName := v.patternElementName(child); eName != "" {
						v.addError(elem, fmt.Sprintf("Expecting an element %s, got nothing", eName))
					}
				}
			}
			v.addError(elem, "Invalid sequence in interleave")
			if isAttr {
				v.addError(elem, fmt.Sprintf("Element %s failed to validate attributes", elem.LocalName()))
			} else {
				v.addError(elem, fmt.Sprintf("Element %s failed to validate content", elem.LocalName()))
			}
			state.seq = before
			return -1
		}
		v.pendingErrors = v.pendingErrors[:errLen]
		v.valid = savedValid
		rest := skipIgnored(sub.seq)
		if len(rest) == 0 {
			continue
		}
		if e, ok := rest[0].(*helium.Element); ok {
			v.addBareError(fmt.Sprintf("Extra element %s in interleave", e.LocalName()))
			v.addError(e, fmt.Sprintf("Element %s failed to validate content", elem.LocalName()))
			state.seq = before
			return -1
		}
		v.addError(elem, "Invalid sequence in interleave")
		v.addError(elem, fmt.Sprintf("Element %s failed to validate content", elem.LocalName()))
		state.seq = before
		return -1
	}
	return 0
}

// validateInterleave is the bare-pattern interleave path: a top-level
// <interleave>/<mixed> reached via validatePattern (no element/attribute
// context, so no diagnostics — matches the pre-partition naive path).
func (v *validator) validateInterleave(pat *pattern, state *validState) int {
	if len(pat.children) == 0 {
		return 0
	}
	part := interleavePartitionOf(pat)
	before := state.seq
	subs := partitionInterleave(part, state)
	for i, child := range pat.children {
		sub := &validState{seq: subs[i], run: v.newRun()}
		if ret := v.validatePattern(child, sub); ret != 0 {
			state.seq = before
			return -1
		}
		if len(skipIgnored(sub.seq)) > 0 {
			state.seq = before
			return -1
		}
	}
	return 0
}

// validateChoiceContentExact evaluates every arm of a choice from the saved
// state and keeps the successful arm that leaves the fewest nodes behind (the
// earliest arm wins a tie, stopping early at zero remaining). It reports
// ok=false when no arm succeeds, so the caller falls back to the ordinary
// greedy choice path and its diagnostics.
//
// Only validateInterleaveContent sets v.exactChoice, and only after a branch
// succeeded with leftover content, so no path outside interleave changes. The
// greedy pass still runs first in validateContentPat's patternChoice case
// because exact mode can lose cases the greedy pass wins — e.g.
// group(choice(d, group(d,d)), d) on [d, d]: the group backtracker retries
// flexible members, not choices.
func (v *validator) validateChoiceContentExact(pat *pattern, elem *helium.Element,
	attrs []*helium.Attribute, attrUsed []bool, state *validState) (int, bool) {
	savedState := state.clone()
	savedAttrUsed := append([]bool(nil), attrUsed...)
	savedLen := len(v.pendingErrors)
	savedValid := v.valid
	var bestState *validState
	var bestAttrUsed []bool
	bestRemaining := -1
	v.suppressDepth++
	for _, child := range pat.children {
		*state = *savedState
		copy(attrUsed, savedAttrUsed)
		v.pendingErrors = v.pendingErrors[:savedLen]
		v.valid = savedValid
		if v.validateContentPat(child, elem, attrs, attrUsed, state) != 0 {
			continue
		}
		remaining := len(skipIgnored(state.seq))
		if bestRemaining >= 0 && remaining >= bestRemaining {
			continue
		}
		bestRemaining = remaining
		bestState = state.clone()
		bestAttrUsed = append([]bool(nil), attrUsed...)
		if remaining == 0 {
			break
		}
	}
	v.suppressDepth--
	*state = *savedState
	copy(attrUsed, savedAttrUsed)
	v.pendingErrors = v.pendingErrors[:savedLen]
	v.valid = savedValid
	if bestState == nil {
		return -1, false
	}
	*state = *bestState
	copy(attrUsed, bestAttrUsed)
	return 0, true
}
