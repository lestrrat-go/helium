package relaxng

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

// validator holds state during document validation.
type validator struct {
	ctx              context.Context
	grammar          *Grammar
	filename         string
	errorHandler     helium.ErrorHandler
	structuredErrors []ValidationError
	valid            bool
	suppressDepth    int // when > 0, errors are suppressed (inside choice branches)
	depth            int // recursion depth guard
}

const maxValidationDepth = 500

func validateDocument(ctx context.Context, doc *helium.Document, grammar *Grammar, cfg *validateConfig) (bool, []ValidationError) {
	if grammar == nil || grammar.start == nil {
		return false, nil
	}

	v := &validator{
		ctx:          ctx,
		grammar:      grammar,
		filename:     cfg.filename,
		errorHandler: cfg.errorHandler,
		valid:        true,
	}

	root := findDocElement(doc)
	if root == nil {
		v.valid = false
		return false, nil
	}

	// Create initial state: the root element
	state := &validState{
		seq: []helium.Node{root},
	}

	ret := v.validatePattern(grammar.start, state)
	if ret != 0 {
		v.valid = false
	}

	// Check for extra content after the root
	remaining := skipIgnored(state.seq)
	if len(remaining) > 0 {
		v.valid = false
	}

	return v.valid, v.structuredErrors
}

// validState tracks the current position during validation.
type validState struct {
	seq []helium.Node // remaining siblings to validate
}

func (s *validState) clone() *validState {
	return &validState{
		seq: append([]helium.Node(nil), s.seq...),
	}
}

// validatePattern validates a pattern against the current state.
// Returns 0 for success, -1 for failure.
func (v *validator) validatePattern(pat *pattern, state *validState) int {
	if pat == nil {
		return -1
	}

	v.depth++
	if v.depth > maxValidationDepth {
		v.depth--
		return -1
	}
	defer func() { v.depth-- }()

	switch pat.kind {
	case patternEmpty:
		return 0
	case patternNotAllowed:
		return -1
	case patternText:
		return v.validateText(state)
	case patternElement:
		return v.validateElement(pat, state)
	case patternAttribute:
		return 0
	case patternGroup:
		return v.validateGroup(pat, state)
	case patternChoice:
		return v.validateChoice(pat, state)
	case patternInterleave:
		return v.validateInterleave(pat, state)
	case patternOptional:
		return v.validateOptional(pat, state)
	case patternZeroOrMore:
		return v.validateZeroOrMore(pat, state)
	case patternOneOrMore:
		return v.validateOneOrMore(pat, state)
	case patternRef:
		return v.validateRef(pat, state)
	case patternParentRef:
		return v.validateRef(pat, state)
	case patternData:
		return v.validateData(pat, state)
	case patternValue:
		return v.validateValue(pat, state)
	case patternList:
		return v.validateList(pat, state)
	case patternMixed:
		return v.validateInterleave(pat, state)
	default:
		return -1
	}
}

func (v *validator) validateText(state *validState) int {
	// Text pattern matches any text content (including empty).
	// Consume text nodes from the sequence.
	for len(state.seq) > 0 {
		node := state.seq[0]
		switch node.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			state.seq = state.seq[1:]
		default:
			return 0
		}
	}
	return 0
}

func (v *validator) validateElement(pat *pattern, state *validState) int {
	// Skip whitespace text nodes
	state.seq = skipIgnored(state.seq)

	if len(state.seq) == 0 {
		return -1
	}

	node := state.seq[0]
	elem, ok := node.(*helium.Element)
	if !ok {
		return -1
	}

	// Match element name (with namespace-aware error messages)
	if !v.elementMatchesWithErrors(pat, elem) {
		return -1
	}

	// Consume this element from the sequence
	state.seq = state.seq[1:]

	// Validate attributes and content together.
	// Build child node list, skipping non-content nodes (DTD artifacts, PIs, comments).
	var children []helium.Node
	for child := range helium.Children(elem) {
		switch child.Type() {
		case helium.EntityRefNode, helium.EntityNode, helium.ProcessingInstructionNode, helium.CommentNode:
			continue
		default:
			children = append(children, child)
		}
	}

	// Collect instance attributes (skip xmlns declarations)
	allAttrs := elem.Attributes()
	var instanceAttrs []*helium.Attribute
	for _, attr := range allAttrs {
		if attr.Prefix() == "xmlns" || (attr.Prefix() == "" && attr.LocalName() == "xmlns") {
			continue
		}
		instanceAttrs = append(instanceAttrs, attr)
	}

	// Try to validate using the element's attrs + children content patterns together.
	contentState := &validState{seq: children}
	attrUsed := make([]bool, len(instanceAttrs))

	// Once we've consumed the element from the parent's sequence, any errors
	// from inside are "definitive" — they should be emitted even inside suppressed
	// contexts (choice/zeroOrMore). Save and clear suppressDepth temporarily.
	savedSuppress := v.suppressDepth
	v.suppressDepth = 0

	errLenBefore := len(v.structuredErrors)
	if ret := v.validateElementBody(pat, elem, instanceAttrs, attrUsed, contentState); ret != 0 {
		// For top-level choice content, check for unknown child elements
		// and insert their errors before the body validation errors.
		topChoice := len(pat.children) == 1 && pat.children[0].kind == patternChoice
		if topChoice {
			bodyErrors := append([]ValidationError(nil), v.structuredErrors[errLenBefore:]...)
			v.structuredErrors = v.structuredErrors[:errLenBefore]
			for _, n := range skipIgnored(contentState.seq) {
				if e, ok := n.(*helium.Element); ok {
					if !v.isKnownChildElement(pat, e.LocalName(), elemNS(e)) {
						v.addError(elem, fmt.Sprintf("Did not expect element %s there", e.LocalName()))
					}
				}
			}
			v.structuredErrors = append(v.structuredErrors, bodyErrors...)
		}
		if len(v.structuredErrors) == errLenBefore {
			v.addError(elem, fmt.Sprintf("Element %s failed to validate content", elem.LocalName()))
		}
		v.suppressDepth = savedSuppress
		return -1
	}

	// Check all attrs consumed
	for i, attr := range instanceAttrs {
		if !attrUsed[i] {
			v.addError(elem, fmt.Sprintf("Invalid attribute %s for element %s", attr.LocalName(), elem.LocalName()))
			v.suppressDepth = savedSuppress
			return -1
		}
	}

	// Check all content consumed
	remaining := skipIgnored(contentState.seq)
	if len(remaining) > 0 {
		// When the element's content is a top-level choice, remaining content
		// errors are reported on the parent element (libxml2 behavior).
		topChoice := len(pat.children) == 1 && pat.children[0].kind == patternChoice
		if topChoice {
			for _, n := range remaining {
				if e, ok := n.(*helium.Element); ok {
					if !v.isKnownChildElement(pat, e.LocalName(), elemNS(e)) {
						v.addError(elem, fmt.Sprintf("Did not expect element %s there", e.LocalName()))
					}
				}
			}
			v.addError(elem, fmt.Sprintf("Element %s failed to validate content", elem.LocalName()))
		} else {
			hasChildError := false
			for _, n := range remaining {
				if e, ok := n.(*helium.Element); ok {
					v.addErrorOnNode(e, fmt.Sprintf("Did not expect element %s there", e.LocalName()))
					hasChildError = true
				}
			}
			if !hasChildError {
				v.addError(elem, fmt.Sprintf("Element %s failed to validate content", elem.LocalName()))
			}
		}
		v.suppressDepth = savedSuppress
		return -1
	}

	v.suppressDepth = savedSuppress
	return 0
}

// validateElementBody validates the attribute and content patterns of an element
// against the instance attributes and content state.
func (v *validator) validateElementBody(pat *pattern, elem *helium.Element,
	attrs []*helium.Attribute, attrUsed []bool, contentState *validState) int {

	// Validate direct attr patterns
	for _, attrPat := range pat.attrs {
		if !v.matchOneAttr(attrPat, attrs, attrUsed, elem) {
			v.addError(elem, fmt.Sprintf("Element %s failed to validate attributes", elem.LocalName()))
			return -1
		}
	}

	// Validate content patterns
	for _, contentPat := range pat.children {
		if ret := v.validateContentPat(contentPat, elem, attrs, attrUsed, contentState); ret != 0 {
			return -1
		}
	}

	return 0
}

// validateContentPat validates a content pattern, handling attributes
// that appear in mixed-mode (attributes inside groups/choices within elements).
func (v *validator) validateContentPat(pat *pattern, elem *helium.Element,
	attrs []*helium.Attribute, attrUsed []bool, state *validState) int {

	if pat == nil {
		return -1
	}

	v.depth++
	if v.depth > maxValidationDepth {
		v.depth--
		return -1
	}
	defer func() { v.depth-- }()

	switch pat.kind {
	case patternAttribute:
		if !v.matchOneAttr(pat, attrs, attrUsed, elem) {
			v.addError(elem, fmt.Sprintf("Element %s failed to validate attributes", elem.LocalName()))
			return -1
		}
		return 0

	case patternGroup:
		return v.validateGroupContent(pat, elem, attrs, attrUsed, state)

	case patternChoice:
		savedLen := len(v.structuredErrors)
		savedValid := v.valid
		v.suppressDepth++
		// Prefer branches that make progress; fall back to no-progress match.
		noProgressMatch := false
		lastBranchLen := savedLen
		lastBranchValid := savedValid
		for _, child := range pat.children {
			savedState := state.clone()
			savedAttrUsed := make([]bool, len(attrUsed))
			copy(savedAttrUsed, attrUsed)

			// Reset errors before each branch so branches don't accumulate.
			v.structuredErrors = v.structuredErrors[:savedLen]
			v.valid = savedValid

			if ret := v.validateContentPat(child, elem, attrs, attrUsed, state); ret == 0 {
				if !seqEqual(state.seq, savedState.seq) || !boolSliceEqual(attrUsed, savedAttrUsed) {
					// Branch made progress — use it.
					v.suppressDepth--
					v.structuredErrors = v.structuredErrors[:savedLen]
					v.valid = savedValid
					return 0
				}
				// Succeeded but no progress — remember and try others.
				noProgressMatch = true
				*state = *savedState
				copy(attrUsed, savedAttrUsed)
			} else {
				lastBranchLen = len(v.structuredErrors)
				lastBranchValid = v.valid
				*state = *savedState
				copy(attrUsed, savedAttrUsed)
			}
		}
		v.suppressDepth--
		if noProgressMatch {
			v.structuredErrors = v.structuredErrors[:savedLen]
			v.valid = savedValid
			return 0
		}
		// If errors grew from inside matched elements (e.g. attribute/value failures),
		// keep those errors so the diagnostic chain is preserved.
		if lastBranchLen > savedLen {
			v.structuredErrors = v.structuredErrors[:lastBranchLen]
			v.valid = lastBranchValid
		} else {
			v.structuredErrors = v.structuredErrors[:savedLen]
			v.valid = savedValid
		}
		if isValueChoice(pat) {
			v.addError(elem, "Error validating value ")
		}
		v.addError(elem, fmt.Sprintf("Element %s failed to validate content", elem.LocalName()))
		return -1

	case patternOptional:
		savedState := state.clone()
		savedAttrUsed := make([]bool, len(attrUsed))
		copy(savedAttrUsed, attrUsed)
		savedLen := len(v.structuredErrors)
		savedValid := v.valid
		v.suppressDepth++
		content := wrapChildren(pat.children)
		ret := v.validateContentPat(content, elem, attrs, attrUsed, state)
		v.suppressDepth--
		if ret != 0 {
			*state = *savedState
			copy(attrUsed, savedAttrUsed)
			v.structuredErrors = v.structuredErrors[:savedLen]
			v.valid = savedValid
		}
		return 0

	case patternZeroOrMore:
		content := wrapChildren(pat.children)
		for {
			savedState := state.clone()
			savedAttrUsed := make([]bool, len(attrUsed))
			copy(savedAttrUsed, attrUsed)
			savedLen := len(v.structuredErrors)
			savedValid := v.valid
			v.suppressDepth++
			ret := v.validateContentPat(content, elem, attrs, attrUsed, state)
			v.suppressDepth--
			if ret != 0 {
				// Check if hard errors were emitted (from inside matched elements).
				if len(v.structuredErrors) > savedLen {
					// Hard failure — propagate errors.
					return -1
				}
				*state = *savedState
				copy(attrUsed, savedAttrUsed)
				v.structuredErrors = v.structuredErrors[:savedLen]
				v.valid = savedValid
				break
			}
			if seqEqual(state.seq, savedState.seq) && boolSliceEqual(attrUsed, savedAttrUsed) {
				break
			}
		}
		return 0

	case patternOneOrMore:
		content := wrapChildren(pat.children)
		// Must match at least once
		if ret := v.validateContentPat(content, elem, attrs, attrUsed, state); ret != 0 {
			return -1
		}
		// Then zero or more
		for {
			savedState := state.clone()
			savedAttrUsed := make([]bool, len(attrUsed))
			copy(savedAttrUsed, attrUsed)
			savedLen := len(v.structuredErrors)
			savedValid := v.valid
			v.suppressDepth++
			ret := v.validateContentPat(content, elem, attrs, attrUsed, state)
			v.suppressDepth--
			if ret != 0 {
				if len(v.structuredErrors) > savedLen {
					return -1
				}
				*state = *savedState
				copy(attrUsed, savedAttrUsed)
				v.structuredErrors = v.structuredErrors[:savedLen]
				v.valid = savedValid
				break
			}
			if seqEqual(state.seq, savedState.seq) && boolSliceEqual(attrUsed, savedAttrUsed) {
				break
			}
		}
		return 0

	case patternInterleave:
		if len(pat.children) == 0 {
			return 0
		}
		// Determine which children are repeatable (zeroOrMore/oneOrMore/text).
		// Text is inherently repeatable in interleave (text nodes appear between elements).
		isRepeatable := make([]bool, len(pat.children))
		for i, child := range pat.children {
			isRepeatable[i] = child.kind == patternZeroOrMore || child.kind == patternOneOrMore || child.kind == patternText
		}

		// For children that resolve to groups, track per-member progress.
		// This allows group members to be matched one-by-one with other
		// interleave children consuming elements between them.
		type groupState struct {
			members []*pattern
			pos     int
		}
		groupStates := make([]*groupState, len(pat.children))
		for i, child := range pat.children {
			if grp := v.resolveToGroup(child); grp != nil {
				groupStates[i] = &groupState{members: grp.children, pos: 0}
			}
		}

		// Track which children ever consumed something.
		consumed := make([]bool, len(pat.children))
		// Track which single-use children are "done" (already matched once).
		done := make([]bool, len(pat.children))
		progress := true
		var extraElemNode *helium.Element // node of the extra (duplicate) element
		for progress {
			progress = false
			for i, child := range pat.children {
				if done[i] {
					// Single-use pattern already consumed. Check if the next element
					// would match it again (= "Extra element in interleave").
					if extraElemNode == nil {
						remaining := skipIgnored(state.seq)
						if len(remaining) > 0 {
							if e, ok := remaining[0].(*helium.Element); ok {
								savedState := state.clone()
								savedAttrUsed := make([]bool, len(attrUsed))
								copy(savedAttrUsed, attrUsed)
								savedLen := len(v.structuredErrors)
								savedValid := v.valid
								v.suppressDepth++
								ret := v.validateContentPat(child, elem, attrs, attrUsed, state)
								v.suppressDepth--
								if ret == 0 && !seqEqual(state.seq, savedState.seq) {
									extraElemNode = e
								}
								*state = *savedState
								copy(attrUsed, savedAttrUsed)
								v.structuredErrors = v.structuredErrors[:savedLen]
								v.valid = savedValid
							}
						}
					}
					continue
				}

				// Group children: match member-by-member so other interleave
				// children can consume elements between group members.
				if gs := groupStates[i]; gs != nil {
					for gs.pos < len(gs.members) {
						member := gs.members[gs.pos]
						savedState := state.clone()
						savedAttrUsed := make([]bool, len(attrUsed))
						copy(savedAttrUsed, attrUsed)
						savedLen := len(v.structuredErrors)
						savedValid := v.valid
						v.suppressDepth++
						ret := v.validateContentPat(member, elem, attrs, attrUsed, state)
						v.suppressDepth--
						if ret == 0 && (!seqEqual(state.seq, savedState.seq) || !boolSliceEqual(attrUsed, savedAttrUsed)) {
							// Member consumed something — advance.
							consumed[i] = true
							progress = true
							gs.pos++
							v.structuredErrors = v.structuredErrors[:savedLen]
							v.valid = savedValid
							// Continue trying next members in same round
							// (they might also match immediately).
						} else {
							// Member didn't consume. If nullable, skip it
							// and try the next member.
							*state = *savedState
							copy(attrUsed, savedAttrUsed)
							v.structuredErrors = v.structuredErrors[:savedLen]
							v.valid = savedValid
							if v.isNullable(member) {
								gs.pos++
								continue
							}
							// Not nullable — stop trying this group for now.
							// Other interleave children may consume first.
							break
						}
					}
					if gs.pos >= len(gs.members) {
						if !isRepeatable[i] {
							done[i] = true
						}
					}
					continue
				}

				// Non-group children: try atomic matching.
				savedState := state.clone()
				savedAttrUsed := make([]bool, len(attrUsed))
				copy(savedAttrUsed, attrUsed)
				savedLen := len(v.structuredErrors)
				savedValid := v.valid
				v.suppressDepth++
				ret := v.validateContentPat(child, elem, attrs, attrUsed, state)
				v.suppressDepth--
				if ret == 0 && (!seqEqual(state.seq, savedState.seq) || !boolSliceEqual(attrUsed, savedAttrUsed)) {
					consumed[i] = true
					progress = true
					if !isRepeatable[i] {
						done[i] = true
					}
					v.structuredErrors = v.structuredErrors[:savedLen]
					v.valid = savedValid
				} else {
					*state = *savedState
					copy(attrUsed, savedAttrUsed)
					v.structuredErrors = v.structuredErrors[:savedLen]
					v.valid = savedValid
				}
			}
		}
		if extraElemNode != nil {
			v.addBareError(fmt.Sprintf("Extra element %s in interleave", extraElemNode.LocalName()))
			v.addErrorOnNode(extraElemNode, fmt.Sprintf("Element %s failed to validate content", elem.LocalName()))
			return -1
		}
		// Check for required children that were never consumed.
		for i, child := range pat.children {
			if !consumed[i] && !v.isNullable(child) {
				// For group children with partial progress, check if remaining members are all nullable.
				if gs := groupStates[i]; gs != nil && gs.pos > 0 {
					allNullable := true
					for j := gs.pos; j < len(gs.members); j++ {
						if !v.isNullable(gs.members[j]) {
							allNullable = false
							break
						}
					}
					if allNullable {
						continue
					}
				}
				isAttr := child.kind == patternAttribute
				if !isAttr {
					eName := v.patternElementName(child)
					if eName != "" {
						v.addError(elem, fmt.Sprintf("Expecting an element %s, got nothing", eName))
					}
				}
				v.addError(elem, "Invalid sequence in interleave")
				if isAttr {
					v.addError(elem, fmt.Sprintf("Element %s failed to validate attributes", elem.LocalName()))
				} else {
					v.addError(elem, fmt.Sprintf("Element %s failed to validate content", elem.LocalName()))
				}
				return -1
			}
		}
		return 0

	case patternRef, patternParentRef:
		def, ok := v.grammar.defines[pat.name]
		if !ok {
			return -1
		}
		return v.validateContentPat(def, elem, attrs, attrUsed, state)

	case patternList:
		text := v.collectText(state)
		if ret := v.matchListContent(pat, text, elem); ret != 0 {
			v.addError(elem, "Error validating list")
			v.addError(elem, fmt.Sprintf("Element %s failed to validate content", elem.LocalName()))
			return -1
		}
		return 0

	default:
		// For element/text/data/value/etc., delegate to normal validation
		return v.validatePattern(pat, state)
	}
}

// groupBound records state at a group child boundary for backtracking.
type groupBound struct {
	state    *validState
	attrUsed []bool
	errLen   int
	valid    bool
}

func saveGroupBound(state *validState, attrUsed []bool, errLen int, valid bool) groupBound {
	return groupBound{
		state:    state.clone(),
		attrUsed: append([]bool(nil), attrUsed...),
		errLen:   errLen,
		valid:    valid,
	}
}

func (b *groupBound) restore(state *validState, attrUsed []bool, v *validator) {
	*state = *b.state
	copy(attrUsed, b.attrUsed)
	v.structuredErrors = v.structuredErrors[:b.errLen]
	v.valid = b.valid
}

// validateGroupContent validates a group pattern's children sequentially with
// backtracking support. When a mandatory child fails because a previous
// flexible child (zeroOrMore, oneOrMore, optional) over-consumed elements,
// we try reducing the flexible child's consumption.
func (v *validator) validateGroupContent(pat *pattern, elem *helium.Element,
	attrs []*helium.Attribute, attrUsed []bool, state *validState) int {

	children := pat.children
	if len(children) == 0 {
		return 0
	}

	// Save state before each child for backtracking.
	bounds := make([]groupBound, 1, len(children)+1)
	bounds[0] = saveGroupBound(state, attrUsed, len(v.structuredErrors), v.valid)

	groupFailed := false
	for gi, child := range children {
		// Track first non-ignored node to detect consumption
		trimmedBefore := skipIgnored(state.seq)
		var firstNodeBefore helium.Node
		if len(trimmedBefore) > 0 {
			firstNodeBefore = trimmedBefore[0]
		}

		errLenBefore := len(v.structuredErrors)
		savedValid := v.valid
		if ret := v.validateContentPat(child, elem, attrs, attrUsed, state); ret != 0 {
			// Check if an element was consumed (sequence advanced)
			trimmedAfter := skipIgnored(state.seq)
			var firstNodeAfter helium.Node
			if len(trimmedAfter) > 0 {
				firstNodeAfter = trimmedAfter[0]
			}
			if firstNodeBefore != nil && firstNodeAfter != firstNodeBefore {
				// Element consumed but content failed — continue to collect
				// errors from remaining group children (like libxml2 does)
				groupFailed = true
				bounds = append(bounds, saveGroupBound(state, attrUsed, len(v.structuredErrors), v.valid))
				continue
			}

			// No element consumed — try backtracking.
			if gi > 0 && v.backtrackGroupFlexible(children, gi, elem, attrs, attrUsed, state, bounds) {
				bounds = append(bounds, saveGroupBound(state, attrUsed, len(v.structuredErrors), v.valid))
				continue
			}

			// No backtracking possible — report errors
			remaining := skipIgnored(state.seq)
			if len(remaining) > 0 {
				if e, ok := remaining[0].(*helium.Element); ok {
					expectedName := v.patternElementName(child)
					if expectedName != "" && expectedName != e.LocalName() && child.kind == patternChoice {
						v.structuredErrors = v.structuredErrors[:errLenBefore]
						v.valid = savedValid
						v.addErrorOnNode(e, fmt.Sprintf("Expecting element %s, got %s", expectedName, e.LocalName()))
						v.addErrorOnNode(e, fmt.Sprintf("Element %s failed to validate content", elem.LocalName()))
					} else if len(v.structuredErrors) == errLenBefore {
						v.addErrorOnNode(e, fmt.Sprintf("Did not expect element %s there", e.LocalName()))
					}
				}
			} else if len(v.structuredErrors) == errLenBefore && v.patternElementName(child) != "" {
				v.addError(elem, "Expecting an element , got nothing")
			}
			return -1
		}

		bounds = append(bounds, saveGroupBound(state, attrUsed, len(v.structuredErrors), v.valid))
	}
	if groupFailed {
		return -1
	}
	return 0
}

// backtrackGroupFlexible tries to fix a group failure at failIdx by reducing
// consumption of a previous flexible child (zeroOrMore, oneOrMore, optional).
// It tries each flexible child from nearest to furthest, and for each tries
// progressively increasing iteration counts from the minimum up to one less
// than the greedy count, preferring the highest successful count to maximize
// content consumption.
func (v *validator) backtrackGroupFlexible(children []*pattern, failIdx int,
	elem *helium.Element, attrs []*helium.Attribute, attrUsed []bool,
	state *validState, bounds []groupBound) bool {

	for j := failIdx - 1; j >= 0; j-- {
		child := children[j]
		isZeroFlex := child.kind == patternZeroOrMore || child.kind == patternOptional
		isOneMore := child.kind == patternOneOrMore
		if !isZeroFlex && !isOneMore {
			continue
		}
		// Check if child[j] consumed anything.
		seqEq := seqEqual(bounds[j].state.seq, bounds[j+1].state.seq)
		attrEq := boolSliceEqual(bounds[j].attrUsed, bounds[j+1].attrUsed)
		if seqEq && attrEq {
			continue
		}

		minIter := 0
		if isOneMore {
			minIter = 1
		}

		content := wrapChildren(child.children)

		// Try progressively increasing iteration counts from minIter upward.
		// Track the best (highest iter) success to maximize content consumption.
		type btSuccess struct {
			state    *validState
			attrUsed []bool
			errLen   int
			valid    bool
		}
		var best *btSuccess

		for iter := minIter; ; iter++ {
			bounds[j].restore(state, attrUsed, v)

			// Run 'iter' iterations of the flexible child.
			iterOK := true
			for k := 0; k < iter; k++ {
				savedSt := state.clone()
				v.suppressDepth++
				ret := v.validateContentPat(content, elem, attrs, attrUsed, state)
				v.suppressDepth--
				if ret != 0 || seqEqual(state.seq, savedSt.seq) {
					iterOK = false
					break
				}
			}
			if !iterOK {
				break // Can't match 'iter' times
			}

			// Check if we've reached the same state as the greedy pass.
			if seqEqual(state.seq, bounds[j+1].state.seq) &&
				boolSliceEqual(attrUsed, bounds[j+1].attrUsed) {
				break // Same as original greedy — stop
			}

			// Try remaining children [j+1..failIdx].
			// Save error state so failed retries don't leave stale errors.
			retryLen := len(v.structuredErrors)
			retryValid := v.valid
			allOK := true
			for k := j + 1; k <= failIdx; k++ {
				if v.validateContentPat(children[k], elem, attrs, attrUsed, state) != 0 {
					allOK = false
					break
				}
			}
			if allOK {
				best = &btSuccess{
					state:    state.clone(),
					attrUsed: append([]bool(nil), attrUsed...),
					errLen:   len(v.structuredErrors),
					valid:    v.valid,
				}
			}
			// Restore errors from before retry so failed attempts don't leak errors.
			v.structuredErrors = v.structuredErrors[:retryLen]
			v.valid = retryValid
		}

		if best != nil {
			*state = *best.state
			copy(attrUsed, best.attrUsed)
			v.structuredErrors = v.structuredErrors[:best.errLen]
			v.valid = best.valid
			return true
		}

		// Restore before trying the next candidate.
		bounds[j].restore(state, attrUsed, v)
	}
	return false
}

// isKnownChildElement checks if an element name/ns appears as a child element
// defined anywhere in the content patterns of the given element pattern.
func (v *validator) isKnownChildElement(elemPat *pattern, name, ns string) bool {
	visited := make(map[string]bool)
	for _, child := range elemPat.children {
		if v.isKnownElementInPatternImpl(child, name, ns, visited) {
			return true
		}
	}
	return false
}

func (v *validator) isKnownElementInPatternImpl(pat *pattern, name, ns string, visited map[string]bool) bool {
	if pat == nil {
		return false
	}
	switch pat.kind {
	case patternElement:
		// Check if this element pattern matches the name
		if pat.nameClass != nil {
			if nameClassMatches(pat.nameClass, name, ns) {
				return true
			}
		} else if (pat.name == "" || pat.name == name) && (pat.ns == "" || pat.ns == ns) {
			return true
		}
		// Don't recurse into element children (those define nested content)
		return false
	case patternRef, patternParentRef:
		if visited[pat.name] {
			return false
		}
		visited[pat.name] = true
		def, ok := v.grammar.defines[pat.name]
		if !ok {
			return false
		}
		return v.isKnownElementInPatternImpl(def, name, ns, visited)
	default:
		for _, child := range pat.children {
			if v.isKnownElementInPatternImpl(child, name, ns, visited) {
				return true
			}
		}
		return false
	}
}

// elementMatchesWithErrors checks element matching and generates error messages.
func (v *validator) elementMatchesWithErrors(pat *pattern, elem *helium.Element) bool {
	ns := elemNS(elem)

	// For ncName name classes (the common case), generate namespace-specific errors.
	if pat.nameClass != nil && pat.nameClass.kind == ncName {
		if pat.nameClass.name == elem.LocalName() {
			patNS := pat.nameClass.ns
			if patNS != "" && ns == "" {
				v.addError(elem, fmt.Sprintf("Expecting a namespace for element %s", elem.LocalName()))
				return false
			}
			if patNS != "" && patNS != ns {
				v.addError(elem, fmt.Sprintf("Element %s has wrong namespace: expecting %s", elem.LocalName(), patNS))
				return false
			}
			if patNS == "" && ns != "" {
				v.addError(elem, fmt.Sprintf("Expecting no namespace for element %s", elem.LocalName()))
				return false
			}
			return true
		}
		return false
	}

	if pat.nameClass != nil {
		if nameClassMatches(pat.nameClass, elem.LocalName(), ns) {
			return true
		}
		// Generate namespace-specific error for anyName-except
		if pat.nameClass.kind == ncAnyName && pat.nameClass.except != nil {
			v.addError(elem, fmt.Sprintf("Element %s has wrong namespace: expecting %s", elem.LocalName(), describeExceptNS(pat.nameClass.except)))
		}
		return false
	}

	// No nameClass — use direct name/ns matching
	if pat.name != "" && pat.name == elem.LocalName() {
		if pat.ns != "" && ns == "" {
			v.addError(elem, fmt.Sprintf("Expecting a namespace for element %s", elem.LocalName()))
			return false
		}
		if pat.ns != "" && pat.ns != ns {
			v.addError(elem, fmt.Sprintf("Element %s has wrong namespace: expecting %s", elem.LocalName(), pat.ns))
			return false
		}
		if pat.ns == "" && ns != "" {
			v.addError(elem, fmt.Sprintf("Expecting no namespace for element %s", elem.LocalName()))
			return false
		}
		return true
	}
	return false
}

// describeExceptNS extracts the expected namespace from an except name class.
func describeExceptNS(nc *nameClass) string {
	if nc == nil {
		return ""
	}
	switch nc.kind {
	case ncNsName:
		return nc.ns
	case ncChoice:
		if nc.left != nil {
			return describeExceptNS(nc.left)
		}
	}
	return ""
}

// matchOneAttr tries to match an attribute pattern against instance attributes.
func (v *validator) matchOneAttr(pat *pattern, attrs []*helium.Attribute, used []bool, elem *helium.Element) bool {
	for i, attr := range attrs {
		if used[i] {
			continue
		}
		if v.attributeMatches(pat, attr) {
			if v.validateAttributeValue(pat, attr, elem) == 0 {
				used[i] = true
				return true
			}
		}
	}
	return false
}

func (v *validator) attributeMatches(pat *pattern, attr *helium.Attribute) bool {
	localName := attr.LocalName()
	uri := attr.URI()
	if pat.nameClass != nil {
		return nameClassMatches(pat.nameClass, localName, uri)
	}
	if pat.name != "" && pat.name != localName {
		return false
	}
	if pat.ns != "" && pat.ns != uri {
		return false
	}
	return true
}

func (v *validator) validateAttributeValue(pat *pattern, attr *helium.Attribute, elem *helium.Element) int {
	if len(pat.children) == 0 {
		return 0
	}

	valuePat := pat.children[0]
	return v.matchAttrContent(valuePat, attr.Value(), elem)
}

// matchAttrContent validates an attribute's text value against a content pattern.
func (v *validator) matchAttrContent(pat *pattern, text string, elem *helium.Element) int {
	if pat == nil {
		return -1
	}
	switch pat.kind {
	case patternText:
		return 0
	case patternValue:
		if ret := v.matchValue(pat, text); ret != 0 {
			if elem != nil && pat.dataType != nil && pat.dataType.library == lexicon.NamespaceXSDDatatypes {
				if validateXSDType(pat.dataType.name, strings.TrimSpace(text), nil) != 0 {
					v.addError(elem, fmt.Sprintf("failed to compare type %s", pat.dataType.name))
				}
			}
			return -1
		}
		return 0
	case patternData:
		return v.matchData(pat, text)
	case patternChoice:
		for _, child := range pat.children {
			if v.matchAttrContent(child, text, elem) == 0 {
				return 0
			}
		}
		return -1
	case patternList:
		if ret := v.matchListContent(pat, text, elem); ret != 0 {
			return -1
		}
		return 0
	case patternGroup:
		// Sequential match on tokens
		tokens := strings.Fields(text)
		offset := 0
		for _, child := range pat.children {
			n, ok := v.matchAttrTokens(child, tokens[offset:])
			if !ok {
				return -1
			}
			offset += n
		}
		if offset != len(tokens) {
			return -1
		}
		return 0
	case patternEmpty:
		if strings.TrimSpace(text) == "" {
			return 0
		}
		return -1
	case patternOptional:
		if strings.TrimSpace(text) == "" {
			return 0
		}
		content := wrapChildren(pat.children)
		return v.matchAttrContent(content, text, elem)
	case patternRef:
		def, ok := v.grammar.defines[pat.name]
		if !ok {
			return -1
		}
		return v.matchAttrContent(def, text, elem)
	case patternOneOrMore:
		content := wrapChildren(pat.children)
		if v.matchAttrContent(content, text, elem) != 0 {
			return -1
		}
		return 0
	case patternZeroOrMore:
		return 0
	}
	return 0
}

// matchListContent validates a string against a list pattern.
func (v *validator) matchListContent(pat *pattern, text string, elem *helium.Element) int {
	tokens := strings.Fields(text)
	offset := 0
	for _, child := range pat.children {
		n, ok := v.matchAttrTokens(child, tokens[offset:])
		if !ok {
			if elem != nil {
				if typeName := listDataTypeName(child); typeName != "" {
					v.addError(elem, fmt.Sprintf("failed to validate type %s", typeName))
				}
			}
			return -1
		}
		offset += n
	}
	if offset != len(tokens) {
		if elem != nil {
			v.addError(elem, fmt.Sprintf("Extra data in list: %s", tokens[offset]))
		}
		return -1
	}
	return 0
}

// listDataTypeName extracts the data type name from a pattern for list error reporting.
func listDataTypeName(pat *pattern) string {
	if pat == nil {
		return ""
	}
	switch pat.kind {
	case patternData:
		if pat.dataType != nil {
			return pat.dataType.name
		}
	case patternOneOrMore, patternZeroOrMore, patternGroup:
		for _, child := range pat.children {
			if name := listDataTypeName(child); name != "" {
				return name
			}
		}
	}
	return ""
}

// matchAttrTokens matches tokens against a pattern, returning how many tokens were consumed.
func (v *validator) matchAttrTokens(pat *pattern, tokens []string) (int, bool) {
	if pat == nil {
		return 0, false
	}
	switch pat.kind {
	case patternData:
		if len(tokens) == 0 {
			return 0, false
		}
		if v.matchData(pat, tokens[0]) == 0 {
			return 1, true
		}
		return 0, false
	case patternValue:
		if len(tokens) == 0 {
			return 0, false
		}
		if v.matchValue(pat, tokens[0]) == 0 {
			return 1, true
		}
		return 0, false
	case patternChoice:
		for _, child := range pat.children {
			n, ok := v.matchAttrTokens(child, tokens)
			if ok {
				return n, true
			}
		}
		return 0, false
	case patternOneOrMore:
		content := wrapChildren(pat.children)
		total := 0
		// Must match at least once
		n, ok := v.matchAttrTokens(content, tokens[total:])
		if !ok {
			return 0, false
		}
		total += n
		// Then zero or more
		for total < len(tokens) {
			n, ok = v.matchAttrTokens(content, tokens[total:])
			if !ok {
				break
			}
			if n == 0 {
				break
			}
			total += n
		}
		return total, true
	case patternZeroOrMore:
		content := wrapChildren(pat.children)
		total := 0
		for total < len(tokens) {
			n, ok := v.matchAttrTokens(content, tokens[total:])
			if !ok || n == 0 {
				break
			}
			total += n
		}
		return total, true
	case patternGroup:
		total := 0
		for _, child := range pat.children {
			n, ok := v.matchAttrTokens(child, tokens[total:])
			if !ok {
				return 0, false
			}
			total += n
		}
		return total, true
	case patternText:
		// Text in a list: consume one token
		if len(tokens) > 0 {
			return 1, true
		}
		return 0, true
	case patternEmpty:
		return 0, true
	case patternRef:
		def, ok := v.grammar.defines[pat.name]
		if !ok {
			return 0, false
		}
		return v.matchAttrTokens(def, tokens)
	case patternOptional:
		content := wrapChildren(pat.children)
		n, ok := v.matchAttrTokens(content, tokens)
		if ok && n > 0 {
			return n, true
		}
		return 0, true
	}
	return 0, false
}

func (v *validator) validateGroup(pat *pattern, state *validState) int {
	for _, child := range pat.children {
		if ret := v.validatePattern(child, state); ret != 0 {
			return -1
		}
	}
	return 0
}

func (v *validator) validateChoice(pat *pattern, state *validState) int {
	// Save error state so failed branches don't pollute the output.
	savedLen := len(v.structuredErrors)
	savedValid := v.valid

	v.suppressDepth++
	for _, child := range pat.children {
		saved := state.clone()
		if ret := v.validatePattern(child, saved); ret == 0 {
			v.suppressDepth--
			// Restore error state (successful branch discards errors from prior branches)
			v.structuredErrors = v.structuredErrors[:savedLen]
			v.valid = savedValid
			*state = *saved
			return 0
		}
	}
	v.suppressDepth--

	// All branches failed — restore error state (no branch errors emitted)
	v.structuredErrors = v.structuredErrors[:savedLen]
	v.valid = savedValid
	return -1
}

func (v *validator) validateInterleave(pat *pattern, state *validState) int {
	if len(pat.children) == 0 {
		return 0
	}

	matched := make([]bool, len(pat.children))
	progress := true

	for progress {
		progress = false
		for i, child := range pat.children {
			if matched[i] {
				continue
			}
			saved := state.clone()
			if ret := v.validatePattern(child, saved); ret == 0 {
				if len(saved.seq) < len(state.seq) || !seqEqual(saved.seq, state.seq) {
					*state = *saved
					matched[i] = true
					progress = true
				}
			}
		}
	}

	// Check all non-nullable children were matched
	for i, child := range pat.children {
		if !matched[i] && !v.isNullable(child) {
			return -1
		}
	}

	return 0
}

func (v *validator) validateOptional(pat *pattern, state *validState) int {
	if len(pat.children) == 0 {
		return 0
	}

	saved := state.clone()
	content := wrapChildren(pat.children)
	if ret := v.validatePattern(content, saved); ret == 0 {
		*state = *saved
	}
	return 0
}

func (v *validator) validateZeroOrMore(pat *pattern, state *validState) int {
	if len(pat.children) == 0 {
		return 0
	}

	content := wrapChildren(pat.children)

	for {
		saved := state.clone()
		if ret := v.validatePattern(content, saved); ret != 0 {
			break
		}
		if len(saved.seq) >= len(state.seq) && seqEqual(saved.seq, state.seq) {
			break
		}
		*state = *saved
	}
	return 0
}

func (v *validator) validateOneOrMore(pat *pattern, state *validState) int {
	if len(pat.children) == 0 {
		return -1
	}

	content := wrapChildren(pat.children)

	// Must match at least once
	if ret := v.validatePattern(content, state); ret != 0 {
		return -1
	}

	// Then zero or more
	for {
		saved := state.clone()
		if ret := v.validatePattern(content, saved); ret != 0 {
			break
		}
		if len(saved.seq) >= len(state.seq) && seqEqual(saved.seq, state.seq) {
			break
		}
		*state = *saved
	}
	return 0
}

func (v *validator) validateRef(pat *pattern, state *validState) int {
	def, ok := v.grammar.defines[pat.name]
	if !ok {
		return -1
	}
	return v.validatePattern(def, state)
}

func (v *validator) validateData(pat *pattern, state *validState) int {
	text := v.collectText(state)
	return v.matchData(pat, text)
}

func (v *validator) validateValue(pat *pattern, state *validState) int {
	text := v.collectText(state)
	return v.matchValue(pat, text)
}

func (v *validator) validateList(pat *pattern, state *validState) int {
	text := v.collectText(state)
	tokens := strings.Fields(text)

	if len(pat.children) == 0 {
		if len(tokens) == 0 {
			return 0
		}
		return -1
	}

	for _, child := range pat.children {
		switch child.kind {
		case patternData:
			if len(tokens) == 0 {
				return -1
			}
			if v.matchData(child, tokens[0]) != 0 {
				return -1
			}
			tokens = tokens[1:]
		case patternValue:
			if len(tokens) == 0 {
				return -1
			}
			if v.matchValue(child, tokens[0]) != 0 {
				return -1
			}
			tokens = tokens[1:]
		case patternOneOrMore:
			if len(tokens) == 0 {
				return -1
			}
			for len(tokens) > 0 {
				matched := false
				for _, cc := range child.children {
					if cc.kind == patternData {
						if v.matchData(cc, tokens[0]) == 0 {
							tokens = tokens[1:]
							matched = true
							break
						}
					}
				}
				if !matched {
					break
				}
			}
		case patternZeroOrMore:
			for len(tokens) > 0 {
				matched := false
				for _, cc := range child.children {
					if cc.kind == patternData {
						if v.matchData(cc, tokens[0]) == 0 {
							tokens = tokens[1:]
							matched = true
							break
						}
					}
				}
				if !matched {
					break
				}
			}
		}
	}

	if len(tokens) > 0 {
		return -1
	}
	return 0
}

// collectText consumes text nodes from the state and returns their content.
func (v *validator) collectText(state *validState) string {
	var sb strings.Builder
loop:
	for len(state.seq) > 0 {
		node := state.seq[0]
		switch node.Type() {
		case helium.TextNode:
			if t, ok := node.(*helium.Text); ok {
				sb.Write(t.Content())
			}
			state.seq = state.seq[1:]
		case helium.CDATASectionNode:
			if t, ok := node.(*helium.CDATASection); ok {
				sb.Write(t.Content())
			}
			state.seq = state.seq[1:]
		default:
			break loop
		}
	}
	return sb.String()
}

// matchValue checks if text matches a value pattern.
func (v *validator) matchValue(pat *pattern, text string) int {
	expected := pat.value

	if pat.dataType != nil {
		switch pat.dataType.library {
		case "":
			if pat.dataType.name == "token" {
				text = normalizeToken(text)
				expected = normalizeToken(expected)
			}
		case lexicon.NamespaceXSDDatatypes:
			// XSD type-aware comparison: normalize both values
			text = strings.TrimSpace(text)
			expected = strings.TrimSpace(expected)
		}
	}

	if text == expected {
		return 0
	}
	return -1
}

// matchData checks if text matches a data type pattern.
func (v *validator) matchData(pat *pattern, text string) int {
	if pat.dataType == nil {
		return 0
	}

	dt := pat.dataType
	text = strings.TrimSpace(text)

	switch dt.library {
	case lexicon.NamespaceXSDDatatypes:
		return validateXSDType(dt.name, text, pat.params)
	case "":
		switch dt.name {
		case "token":
			return 0
		case "string":
			return 0
		}
	}

	return 0
}

// validateXSDType validates a value against an XSD datatype.
func validateXSDType(typeName, value string, params []*param) int {
	switch typeName {
	case "string":
		return validateWithParams(value, params)
	case "normalizedString", "token":
		return 0
	case "integer", "int", "long", "short", "byte",
		"nonNegativeInteger", "positiveInteger",
		"nonPositiveInteger", "negativeInteger",
		"unsignedInt", "unsignedLong", "unsignedShort", "unsignedByte":
		return validateXSDInteger(typeName, value)
	case "decimal":
		return validateXSDDecimal(value)
	case "float", "double":
		return validateXSDFloat(value)
	case "boolean":
		return validateXSDBoolean(value)
	case "date":
		if len(value) < 10 {
			return -1
		}
		return 0
	case "dateTime":
		if len(value) < 19 {
			return -1
		}
		return 0
	case "time":
		if len(value) < 8 {
			return -1
		}
		return 0
	case "duration":
		if len(value) < 2 || value[0] != 'P' {
			return -1
		}
		return 0
	case "anyURI":
		return 0
	case "QName":
		return validateXSDQName(value)
	case "NCName", "ID", "IDREF":
		return validateXSDNCName(value)
	case "Name":
		return validateXSDName(value)
	case "NMTOKEN":
		return validateXSDNMTOKEN(value)
	case "NMTOKENS":
		return validateXSDNMTOKENS(value)
	case "IDREFS":
		tokens := strings.Fields(value)
		if len(tokens) == 0 {
			return -1
		}
		for _, t := range tokens {
			if validateXSDNCName(t) != 0 {
				return -1
			}
		}
		return 0
	case "language":
		return 0
	case "base64Binary":
		return 0
	case "hexBinary":
		return validateXSDHexBinary(value)
	default:
		return 0
	}
}

func validateWithParams(value string, params []*param) int {
	for _, p := range params {
		switch p.name {
		case "pattern":
			matched, err := regexp.MatchString("^(?:"+p.value+")$", value)
			if err != nil || !matched {
				return -1
			}
		case "minLength":
			// simplified: just check length
		case "maxLength":
			// simplified: just check length
		}
	}
	return 0
}

func validateXSDInteger(typeName, value string) int {
	if value == "" {
		return -1
	}
	start := 0
	if value[0] == '+' || value[0] == '-' {
		start = 1
	}
	if start >= len(value) {
		return -1
	}
	for i := start; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return -1
		}
	}
	switch typeName {
	case "nonNegativeInteger", "positiveInteger", "unsignedInt", "unsignedLong", "unsignedShort", "unsignedByte":
		if value[0] == '-' {
			return -1
		}
		if typeName == "positiveInteger" && value == "0" {
			return -1
		}
	case "nonPositiveInteger":
		if value[0] != '-' && value != "0" {
			return -1
		}
	case "negativeInteger":
		if value[0] != '-' || value == "-0" {
			return -1
		}
	}
	return 0
}

func validateXSDDecimal(value string) int {
	if value == "" {
		return -1
	}
	start := 0
	if value[0] == '+' || value[0] == '-' {
		start = 1
	}
	if start >= len(value) {
		return -1
	}
	dotSeen := false
	for i := start; i < len(value); i++ {
		if value[i] == '.' {
			if dotSeen {
				return -1
			}
			dotSeen = true
		} else if value[i] < '0' || value[i] > '9' {
			return -1
		}
	}
	return 0
}

func validateXSDFloat(value string) int {
	if value == "" {
		return -1
	}
	switch value {
	case "INF", "-INF", "+INF", "NaN":
		return 0
	}
	start := 0
	if value[0] == '+' || value[0] == '-' {
		start = 1
	}
	if start >= len(value) {
		return -1
	}
	dotSeen := false
	eSeen := false
	for i := start; i < len(value); i++ {
		if value[i] == '.' {
			if dotSeen || eSeen {
				return -1
			}
			dotSeen = true
		} else if value[i] == 'e' || value[i] == 'E' {
			if eSeen {
				return -1
			}
			eSeen = true
			if i+1 < len(value) && (value[i+1] == '+' || value[i+1] == '-') {
				i++
			}
		} else if value[i] < '0' || value[i] > '9' {
			return -1
		}
	}
	return 0
}

func validateXSDBoolean(value string) int {
	switch value {
	case "true", "false", "1", "0":
		return 0
	}
	return -1
}

func validateXSDQName(value string) int {
	parts := strings.SplitN(value, ":", 2)
	for _, p := range parts {
		if validateXSDNCName(p) != 0 {
			return -1
		}
	}
	return 0
}

func validateXSDNCName(value string) int {
	if value == "" {
		return -1
	}
	if !isNameStartChar(rune(value[0])) {
		return -1
	}
	for _, r := range value[1:] {
		if !isNameChar(r) || r == ':' {
			return -1
		}
	}
	return 0
}

func validateXSDName(value string) int {
	if value == "" {
		return -1
	}
	if !isNameStartChar(rune(value[0])) && value[0] != ':' {
		return -1
	}
	for _, r := range value[1:] {
		if !isNameChar(r) {
			return -1
		}
	}
	return 0
}

func validateXSDNMTOKEN(value string) int {
	if value == "" {
		return -1
	}
	for _, r := range value {
		if !isNameChar(r) {
			return -1
		}
	}
	return 0
}

func validateXSDNMTOKENS(value string) int {
	tokens := strings.Fields(value)
	if len(tokens) == 0 {
		return -1
	}
	for _, t := range tokens {
		if validateXSDNMTOKEN(t) != 0 {
			return -1
		}
	}
	return 0
}

func validateXSDHexBinary(value string) int {
	if len(value)%2 != 0 {
		return -1
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return -1
		}
	}
	return 0
}

func isNameStartChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
}

func isNameChar(r rune) bool {
	return isNameStartChar(r) || (r >= '0' && r <= '9') || r == '-' || r == '.' || r == ':'
}

// normalizeToken collapses whitespace and trims.
func normalizeToken(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// addError adds a validation error (suppressed when inside choice branches).
func (v *validator) addError(elem *helium.Element, msg string) {
	if v.suppressDepth > 0 {
		return
	}
	line := elem.Line()
	errStr := validityError(v.filename, line, elem.LocalName(), msg)
	v.structuredErrors = append(v.structuredErrors, ValidationError{
		Filename: v.filename,
		Line:     line,
		Element:  elem.LocalName(),
		Message:  msg,
	})
	if v.errorHandler != nil {
		v.errorHandler.Handle(v.ctx, helium.NewLeveledError(errStr, helium.ErrorLevelError))
	}
	v.valid = false
}

// addErrorOnNode adds an error attributed to a specific element node.
func (v *validator) addErrorOnNode(elem *helium.Element, msg string) {
	if v.suppressDepth > 0 {
		return
	}
	line := elem.Line()
	errStr := validityError(v.filename, line, elem.LocalName(), msg)
	v.structuredErrors = append(v.structuredErrors, ValidationError{
		Filename: v.filename,
		Line:     line,
		Element:  elem.LocalName(),
		Message:  msg,
	})
	if v.errorHandler != nil {
		v.errorHandler.Handle(v.ctx, helium.NewLeveledError(errStr, helium.ErrorLevelError))
	}
	v.valid = false
}

// addBareError adds a validation error without file/line/element context.
func (v *validator) addBareError(msg string) {
	if v.suppressDepth > 0 {
		return
	}
	errStr := bareValidityError(msg)
	v.structuredErrors = append(v.structuredErrors, ValidationError{
		Message: msg,
	})
	if v.errorHandler != nil {
		v.errorHandler.Handle(v.ctx, helium.NewLeveledError(errStr, helium.ErrorLevelError))
	}
	v.valid = false
}

// patternElementName extracts the expected element name from a pattern.
// Returns empty string if the pattern does not represent a single named element.
func (v *validator) patternElementName(pat *pattern) string {
	if pat == nil {
		return ""
	}
	switch pat.kind {
	case patternElement:
		return pat.name
	case patternRef:
		def, ok := v.grammar.defines[pat.name]
		if !ok {
			return ""
		}
		return v.patternElementName(def)
	case patternChoice:
		for _, child := range pat.children {
			name := v.patternElementName(child)
			if name != "" {
				return name
			}
		}
	case patternGroup:
		if len(pat.children) > 0 {
			return v.patternElementName(pat.children[0])
		}
	case patternOneOrMore:
		if len(pat.children) > 0 {
			return v.patternElementName(pat.children[0])
		}
	}
	return ""
}

// isNullable checks if a pattern can match empty content.
func (v *validator) isNullable(pat *pattern) bool {
	if pat == nil {
		return true
	}
	switch pat.kind {
	case patternEmpty, patternText:
		return true
	case patternNotAllowed, patternElement, patternData, patternValue, patternList, patternAttribute:
		return false
	case patternOptional, patternZeroOrMore:
		return true
	case patternOneOrMore:
		if len(pat.children) == 0 {
			return true
		}
		for _, child := range pat.children {
			if !v.isNullable(child) {
				return false
			}
		}
		return true
	case patternChoice:
		for _, child := range pat.children {
			if v.isNullable(child) {
				return true
			}
		}
		return false
	case patternGroup, patternInterleave:
		for _, child := range pat.children {
			if !v.isNullable(child) {
				return false
			}
		}
		return true
	case patternRef:
		def, ok := v.grammar.defines[pat.name]
		if !ok {
			return false
		}
		return v.isNullable(def)
	}
	return false
}

func wrapChildren(children []*pattern) *pattern {
	if len(children) == 0 {
		return &pattern{kind: patternEmpty}
	}
	if len(children) == 1 {
		return children[0]
	}
	return &pattern{kind: patternGroup, children: children}
}

// skipIgnored skips whitespace-only text nodes and comment nodes.
func skipIgnored(nodes []helium.Node) []helium.Node {
	for len(nodes) > 0 {
		n := nodes[0]
		if n.Type() == helium.CommentNode {
			nodes = nodes[1:]
			continue
		}
		if n.Type() == helium.TextNode {
			if t, ok := n.(*helium.Text); ok {
				if strings.TrimSpace(string(t.Content())) == "" {
					nodes = nodes[1:]
					continue
				}
			}
		}
		break
	}
	return nodes
}

func boolSliceEqual(a, b []bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func seqEqual(a, b []helium.Node) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// isValueChoice checks if a choice pattern contains only value or data patterns.
func isValueChoice(pat *pattern) bool {
	if pat == nil || pat.kind != patternChoice {
		return false
	}
	for _, child := range pat.children {
		if child.kind != patternValue && child.kind != patternData {
			return false
		}
	}
	return len(pat.children) > 0
}

// resolveToGroup follows refs to find a group pattern. Returns nil if the
// pattern is not a group (or ref chain to a group).
func (v *validator) resolveToGroup(pat *pattern) *pattern {
	for pat != nil {
		switch pat.kind {
		case patternRef, patternParentRef:
			def, ok := v.grammar.defines[pat.name]
			if !ok {
				return nil
			}
			pat = def
		case patternGroup:
			if len(pat.children) > 1 {
				return pat
			}
			return nil
		default:
			return nil
		}
	}
	return nil
}

func findDocElement(doc *helium.Document) *helium.Element {
	for child := range helium.Children(doc) {
		if elem, ok := child.(*helium.Element); ok {
			return elem
		}
	}
	return nil
}
