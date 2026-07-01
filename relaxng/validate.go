package relaxng

import (
	"cmp"
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"unicode/utf8"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xsd/value"
	"github.com/lestrrat-go/helium/internal/xsdregex"
)

// validator holds state during document validation.
type validator struct {
	grammar       *Grammar
	filename      string
	errorHandler  helium.ErrorHandler
	pendingErrors []error // buffered errors, flushed to errorHandler at end
	valid         bool
	suppressDepth int // when > 0, errors are suppressed (inside choice branches)
	depth         int // recursion depth guard

	// groupMemo caches results of validateGroupChildren/validateGroupSeq so the
	// recursive group backtracker (backtrackGroupFlexible/backtrackGroupNaive)
	// does not re-explore overlapping (child-range, input-position) subproblems.
	// Without it the cascading retry is exponential in the number of flexible
	// members. Keyed by the full input that determines the result; a hit
	// reproduces the original call's effect byte-for-byte.
	groupMemo map[groupMemoKey]*groupMemoEntry
}

// groupMemoKey identifies a group-validation subproblem. Two calls with an equal
// key are guaranteed to produce identical results and side effects.
type groupMemoKey struct {
	first *pattern        // children[0] — uniquely identifies the child sub-range start
	n     int             // len(children) — sub-range length
	elem  *helium.Element // owning element (content path); pins the element's attrs,
	// which group content may consult even when the child sequence is empty
	pos      helium.Node // first remaining node (nil when the input sequence is empty)
	seqLen   int         // len(state.seq); with pos, fully identifies the sibling run
	attrKey  string      // packed attrUsed bits ("" for the naive path)
	suppress bool        // v.suppressDepth > 0 (governs whether errors are emitted)
	content  bool        // element-content path vs naive path discriminator
}

// groupMemoEntry records the full effect of a memoized group-validation call so a
// cache hit can reproduce it exactly: the resulting input position, attribute
// usage, any errors the call appended, and its return value.
type groupMemoEntry struct {
	result   int
	seq      []helium.Node
	attrUsed []bool
	errs     []error
}

func (e *groupMemoEntry) apply(state *validState, attrUsed []bool, v *validator) int {
	state.seq = append([]helium.Node(nil), e.seq...)
	if attrUsed != nil {
		copy(attrUsed, e.attrUsed)
	}
	if len(e.errs) > 0 {
		v.pendingErrors = append(v.pendingErrors, e.errs...)
		v.valid = false
	}
	return e.result
}

// groupMemoLookupKey builds the cache key for a group-validation call, or reports
// false when the sub-range is empty (trivially handled by the callers).
func (v *validator) groupMemoLookupKey(children []*pattern, elem *helium.Element, attrUsed []bool, state *validState, content bool) (groupMemoKey, bool) {
	if len(children) == 0 {
		return groupMemoKey{}, false
	}
	var pos helium.Node
	if len(state.seq) > 0 {
		pos = state.seq[0]
	}
	var attrKey string
	if len(attrUsed) > 0 {
		buf := make([]byte, len(attrUsed))
		for i, used := range attrUsed {
			if used {
				buf[i] = '1'
			} else {
				buf[i] = '0'
			}
		}
		attrKey = string(buf)
	}
	return groupMemoKey{
		first:    children[0],
		n:        len(children),
		elem:     elem,
		pos:      pos,
		seqLen:   len(state.seq),
		attrKey:  attrKey,
		suppress: v.suppressDepth > 0,
		content:  content,
	}, true
}

const maxValidationDepth = 500

func validateDocument(ctx context.Context, doc *helium.Document, grammar *Grammar, cfg *validateConfig, handler helium.ErrorHandler) bool {
	if grammar == nil || grammar.start == nil {
		return false
	}

	label := cfg.label
	if label == "" {
		label = doc.URL()
	}
	if label == "" {
		label = "(string)"
	}

	v := &validator{
		grammar:      grammar,
		filename:     label,
		errorHandler: handler,
		valid:        true,
	}

	root := findDocElement(doc)
	if root == nil {
		v.valid = false
		return false
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

	// Flush buffered errors to the handler.
	for _, e := range v.pendingErrors {
		handler.Handle(ctx, e)
	}

	return v.valid
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

func (v *validator) validateText(state *validState) int { //nolint:unparam // always 0 but matches validatePattern return contract
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

	errLenBefore := len(v.pendingErrors)
	if ret := v.validateElementBody(pat, elem, instanceAttrs, attrUsed, contentState); ret != 0 {
		// For top-level choice content, check for unknown child elements
		// and insert their errors before the body validation errors.
		topChoice := len(pat.children) == 1 && pat.children[0].kind == patternChoice
		if topChoice {
			bodyErrors := append([]error(nil), v.pendingErrors[errLenBefore:]...)
			v.pendingErrors = v.pendingErrors[:errLenBefore]
			for _, n := range skipIgnored(contentState.seq) {
				if e, ok := n.(*helium.Element); ok {
					if !v.isKnownChildElement(pat, e.LocalName(), elemNS(e)) {
						v.addError(elem, fmt.Sprintf("Did not expect element %s there", e.LocalName()))
					}
				}
			}
			v.pendingErrors = append(v.pendingErrors, bodyErrors...)
		}
		if len(v.pendingErrors) == errLenBefore {
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
					v.addError(e, fmt.Sprintf("Did not expect element %s there", e.LocalName()))
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
		savedLen := len(v.pendingErrors)
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
			v.pendingErrors = v.pendingErrors[:savedLen]
			v.valid = savedValid

			if ret := v.validateContentPat(child, elem, attrs, attrUsed, state); ret == 0 {
				if !seqEqual(state.seq, savedState.seq) || !boolSliceEqual(attrUsed, savedAttrUsed) {
					// Branch made progress — use it.
					v.suppressDepth--
					v.pendingErrors = v.pendingErrors[:savedLen]
					v.valid = savedValid
					return 0
				}
				// Succeeded but no progress — remember and try others.
				noProgressMatch = true
				*state = *savedState
				copy(attrUsed, savedAttrUsed)
			} else {
				lastBranchLen = len(v.pendingErrors)
				lastBranchValid = v.valid
				*state = *savedState
				copy(attrUsed, savedAttrUsed)
			}
		}
		v.suppressDepth--
		if noProgressMatch {
			v.pendingErrors = v.pendingErrors[:savedLen]
			v.valid = savedValid
			return 0
		}
		// If errors grew from inside matched elements (e.g. attribute/value failures),
		// keep those errors so the diagnostic chain is preserved.
		if lastBranchLen > savedLen {
			v.pendingErrors = v.pendingErrors[:lastBranchLen]
			v.valid = lastBranchValid
		} else {
			v.pendingErrors = v.pendingErrors[:savedLen]
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
		savedLen := len(v.pendingErrors)
		savedValid := v.valid
		v.suppressDepth++
		content := wrapChildren(pat.children)
		ret := v.validateContentPat(content, elem, attrs, attrUsed, state)
		v.suppressDepth--
		if ret != 0 {
			*state = *savedState
			copy(attrUsed, savedAttrUsed)
			v.pendingErrors = v.pendingErrors[:savedLen]
			v.valid = savedValid
		}
		return 0

	case patternZeroOrMore:
		content := wrapChildren(pat.children)
		for {
			savedState := state.clone()
			savedAttrUsed := make([]bool, len(attrUsed))
			copy(savedAttrUsed, attrUsed)
			savedLen := len(v.pendingErrors)
			savedValid := v.valid
			v.suppressDepth++
			ret := v.validateContentPat(content, elem, attrs, attrUsed, state)
			v.suppressDepth--
			if ret != 0 {
				// Check if hard errors were emitted (from inside matched elements).
				if len(v.pendingErrors) > savedLen {
					// Hard failure — propagate errors.
					return -1
				}
				*state = *savedState
				copy(attrUsed, savedAttrUsed)
				v.pendingErrors = v.pendingErrors[:savedLen]
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
			savedLen := len(v.pendingErrors)
			savedValid := v.valid
			v.suppressDepth++
			ret := v.validateContentPat(content, elem, attrs, attrUsed, state)
			v.suppressDepth--
			if ret != 0 {
				if len(v.pendingErrors) > savedLen {
					return -1
				}
				*state = *savedState
				copy(attrUsed, savedAttrUsed)
				v.pendingErrors = v.pendingErrors[:savedLen]
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
		// interleave children consuming elements between them. A repeatable
		// child wrapping a group (zeroOrMore/oneOrMore of group(a,b)) is tracked
		// the same way, with repeat=true so a completed iteration restarts at the
		// first member — letting another interleave branch consume between group
		// members across iterations.
		type groupState struct {
			members      []*pattern
			pos          int
			repeat       bool // member-group wrapped in zeroOrMore/oneOrMore
			iterConsumed bool // current iteration consumed at least one item
		}
		groupStates := make([]*groupState, len(pat.children))
		for i, child := range pat.children {
			if grp := v.resolveToGroup(child); grp != nil {
				groupStates[i] = &groupState{members: grp.children, pos: 0}
				continue
			}
			if child.kind == patternZeroOrMore || child.kind == patternOneOrMore {
				if grp := v.resolveToGroup(wrapChildren(child.children)); grp != nil {
					groupStates[i] = &groupState{members: grp.children, pos: 0, repeat: true}
				}
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
								savedLen := len(v.pendingErrors)
								savedValid := v.valid
								v.suppressDepth++
								ret := v.validateContentPat(child, elem, attrs, attrUsed, state)
								v.suppressDepth--
								if ret == 0 && !seqEqual(state.seq, savedState.seq) {
									extraElemNode = e
								}
								*state = *savedState
								copy(attrUsed, savedAttrUsed)
								v.pendingErrors = v.pendingErrors[:savedLen]
								v.valid = savedValid
							}
						}
					}
					continue
				}

				// Group children: match member-by-member so other interleave
				// children can consume elements between group members. For a
				// repeatable member-group, a completed iteration that consumed
				// something restarts at the first member (a fresh group iteration).
				if gs := groupStates[i]; gs != nil {
					for {
						gs.iterConsumed = false
						blocked := false
						for gs.pos < len(gs.members) {
							member := gs.members[gs.pos]
							savedState := state.clone()
							savedAttrUsed := make([]bool, len(attrUsed))
							copy(savedAttrUsed, attrUsed)
							savedLen := len(v.pendingErrors)
							savedValid := v.valid
							v.suppressDepth++
							ret := v.validateContentPat(member, elem, attrs, attrUsed, state)
							v.suppressDepth--
							if ret == 0 && (!seqEqual(state.seq, savedState.seq) || !boolSliceEqual(attrUsed, savedAttrUsed)) {
								// Member consumed something — advance.
								consumed[i] = true
								progress = true
								gs.iterConsumed = true
								gs.pos++
								v.pendingErrors = v.pendingErrors[:savedLen]
								v.valid = savedValid
								// Continue trying next members in same round
								// (they might also match immediately).
							} else {
								// Member didn't consume. If nullable, skip it
								// and try the next member.
								*state = *savedState
								copy(attrUsed, savedAttrUsed)
								v.pendingErrors = v.pendingErrors[:savedLen]
								v.valid = savedValid
								if v.isNullable(member) {
									gs.pos++
									continue
								}
								// Not nullable — stop trying this group for now.
								// Other interleave children may consume first.
								blocked = true
								break
							}
						}
						// Restart a repeatable member-group only after a full
						// iteration that consumed something; otherwise stop.
						if gs.repeat && !blocked && gs.pos >= len(gs.members) && gs.iterConsumed {
							gs.pos = 0
							continue
						}
						break
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
				savedLen := len(v.pendingErrors)
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
					v.pendingErrors = v.pendingErrors[:savedLen]
					v.valid = savedValid
				} else {
					*state = *savedState
					copy(attrUsed, savedAttrUsed)
					v.pendingErrors = v.pendingErrors[:savedLen]
					v.valid = savedValid
				}
			}
		}
		if extraElemNode != nil {
			v.addBareError(fmt.Sprintf("Extra element %s in interleave", extraElemNode.LocalName()))
			v.addError(extraElemNode, fmt.Sprintf("Element %s failed to validate content", elem.LocalName()))
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
		// A repeatable member-group that ends mid-iteration with a non-nullable
		// remaining member has a dangling partial group (e.g. zeroOrMore(group(a,b))
		// fed an unpaired trailing a): that is incomplete content.
		for i := range pat.children {
			gs := groupStates[i]
			if gs == nil || !gs.repeat || gs.pos <= 0 || gs.pos >= len(gs.members) {
				continue
			}
			for j := gs.pos; j < len(gs.members); j++ {
				if !v.isNullable(gs.members[j]) {
					v.addError(elem, "Invalid sequence in interleave")
					v.addError(elem, fmt.Sprintf("Element %s failed to validate content", elem.LocalName()))
					return -1
				}
			}
		}
		return 0

	case patternRef, patternParentRef:
		def := pat.resolved
		if def == nil {
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
	v.pendingErrors = v.pendingErrors[:b.errLen]
	v.valid = b.valid
}

// validateGroupContent validates a group pattern's children sequentially with
// backtracking support. When a mandatory child fails because a previous
// flexible child (zeroOrMore, oneOrMore, optional) over-consumed elements,
// we try reducing the flexible child's consumption.
func (v *validator) validateGroupContent(pat *pattern, elem *helium.Element,
	attrs []*helium.Attribute, attrUsed []bool, state *validState) int {
	return v.validateGroupChildren(pat.children, elem, attrs, attrUsed, state)
}

// validateGroupChildren validates a sequence of group children with backtracking
// support. It is shared by validateGroupContent and by the recursive retry inside
// backtrackGroupFlexible, so a backtracked sub-range that contains its own
// flexible members can itself backtrack. Results are memoized so the cascading
// retry stays polynomial instead of exponential in the flexible-member count.
func (v *validator) validateGroupChildren(children []*pattern, elem *helium.Element,
	attrs []*helium.Attribute, attrUsed []bool, state *validState) int {
	key, ok := v.groupMemoLookupKey(children, elem, attrUsed, state, true)
	if ok {
		if e, hit := v.groupMemo[key]; hit {
			return e.apply(state, attrUsed, v)
		}
	}
	errBase := len(v.pendingErrors)
	result := v.validateGroupChildrenUncached(children, elem, attrs, attrUsed, state)
	if ok {
		if v.groupMemo == nil {
			v.groupMemo = make(map[groupMemoKey]*groupMemoEntry)
		}
		v.groupMemo[key] = &groupMemoEntry{
			result:   result,
			seq:      append([]helium.Node(nil), state.seq...),
			attrUsed: append([]bool(nil), attrUsed...),
			errs:     append([]error(nil), v.pendingErrors[errBase:]...),
		}
	}
	return result
}

func (v *validator) validateGroupChildrenUncached(children []*pattern, elem *helium.Element,
	attrs []*helium.Attribute, attrUsed []bool, state *validState) int {
	if len(children) == 0 {
		return 0
	}

	// Save state before each child for backtracking.
	bounds := make([]groupBound, 1, len(children)+1)
	bounds[0] = saveGroupBound(state, attrUsed, len(v.pendingErrors), v.valid)

	groupFailed := false
	for gi, child := range children {
		// Track first non-ignored node to detect consumption
		trimmedBefore := skipIgnored(state.seq)
		var firstNodeBefore helium.Node
		if len(trimmedBefore) > 0 {
			firstNodeBefore = trimmedBefore[0]
		}

		errLenBefore := len(v.pendingErrors)
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
				bounds = append(bounds, saveGroupBound(state, attrUsed, len(v.pendingErrors), v.valid))
				continue
			}

			// No element consumed — try backtracking.
			if gi > 0 && v.backtrackGroupFlexible(children, gi, elem, attrs, attrUsed, state, bounds) {
				bounds = append(bounds, saveGroupBound(state, attrUsed, len(v.pendingErrors), v.valid))
				continue
			}

			// No backtracking possible — report errors
			remaining := skipIgnored(state.seq)
			if len(remaining) > 0 {
				if e, ok := remaining[0].(*helium.Element); ok {
					expectedName := v.patternElementName(child)
					if expectedName != "" && expectedName != e.LocalName() && child.kind == patternChoice {
						v.pendingErrors = v.pendingErrors[:errLenBefore]
						v.valid = savedValid
						v.addError(e, fmt.Sprintf("Expecting element %s, got %s", expectedName, e.LocalName()))
						v.addError(e, fmt.Sprintf("Element %s failed to validate content", elem.LocalName()))
					} else if len(v.pendingErrors) == errLenBefore {
						v.addError(e, fmt.Sprintf("Did not expect element %s there", e.LocalName()))
					}
				}
			} else if len(v.pendingErrors) == errLenBefore && v.patternElementName(child) != "" {
				v.addError(elem, "Expecting an element , got nothing")
			}
			return -1
		}

		bounds = append(bounds, saveGroupBound(state, attrUsed, len(v.pendingErrors), v.valid))
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
			for range iter {
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
			retryLen := len(v.pendingErrors)
			retryValid := v.valid
			// Validate the remaining children as a group so that any flexible
			// members in [j+1..failIdx] can themselves backtrack. A flat greedy
			// retry would let a second flexible member re-grab content that a
			// later mandatory member needs.
			allOK := v.validateGroupChildren(children[j+1:failIdx+1], elem, attrs, attrUsed, state) == 0
			if allOK {
				best = &btSuccess{
					state:    state.clone(),
					attrUsed: append([]bool(nil), attrUsed...),
					errLen:   len(v.pendingErrors),
					valid:    v.valid,
				}
			}
			// Restore errors from before retry so failed attempts don't leak errors.
			v.pendingErrors = v.pendingErrors[:retryLen]
			v.valid = retryValid
		}

		if best != nil {
			*state = *best.state
			copy(attrUsed, best.attrUsed)
			v.pendingErrors = v.pendingErrors[:best.errLen]
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
	visited := make(map[*pattern]bool)
	for _, child := range elemPat.children {
		if v.isKnownElementInPatternImpl(child, name, ns, visited) {
			return true
		}
	}
	return false
}

func (v *validator) isKnownElementInPatternImpl(pat *pattern, name, ns string, visited map[*pattern]bool) bool {
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
		def := pat.resolved
		if def == nil {
			return false
		}
		if visited[def] {
			return false
		}
		visited[def] = true
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
	// Fail closed: an attribute pattern with no name class and no name/ns is
	// malformed (parseAttribute should have poisoned it). Never treat it as a
	// wildcard that matches any attribute.
	if pat.name == "" && pat.ns == "" {
		return false
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
		// Sequential match on tokens, backtracking across each member's
		// consumption options so a greedy repetition or a zero-token choice
		// branch cannot strand a later mandatory member. Tokenization splits on
		// XML whitespace only (#x20, #x9, #xA, #xD), not arbitrary Unicode
		// whitespace, so a value such as "a b" stays a single token.
		tokens := xmlFields(text)
		if slices.Contains(v.groupCounts(pat.children, tokens), len(tokens)) {
			return 0
		}
		return -1
	case patternEmpty:
		if isXMLSpaceOnly(text) {
			return 0
		}
		return -1
	case patternOptional:
		if isXMLSpaceOnly(text) {
			return 0
		}
		content := wrapChildren(pat.children)
		return v.matchAttrContent(content, text, elem)
	case patternRef, patternParentRef:
		def := pat.resolved
		if def == nil {
			return -1
		}
		return v.matchAttrContent(def, text, elem)
	case patternOneOrMore, patternZeroOrMore:
		// Match the repetition against the attribute's tokens via the
		// backtracking token matcher, requiring that the whole token sequence
		// is consumed. Without the full-consumption check, a zeroOrMore would
		// accept any text (consuming nothing) and a oneOrMore would ignore
		// trailing non-matching tokens. Tokenization must split on XML
		// whitespace only (#x20, #x9, #xA, #xD), not arbitrary Unicode
		// whitespace, so a value such as "a b" stays a single token.
		tokens := xmlFields(text)
		if slices.Contains(v.matchAttrTokensCounts(pat, tokens), len(tokens)) {
			return 0
		}
		return -1
	}
	return 0
}

// matchListContent validates a string against a list pattern.
func (v *validator) matchListContent(pat *pattern, text string, elem *helium.Element) int {
	// <list> tokenization splits on XML whitespace only (#x20, #x9, #xA, #xD),
	// not arbitrary Unicode whitespace (e.g. NBSP stays part of a token).
	tokens := xmlFields(text)
	// Backtrack across each member's consumption options so a greedy
	// repetition or a zero-token choice branch cannot strand a later
	// mandatory member.
	if slices.Contains(v.groupCounts(pat.children, tokens), len(tokens)) {
		return 0
	}

	// No combination consumed the whole list. Report a best-effort error by
	// replaying members greedily to find where matching first stalls.
	if elem != nil {
		offset := 0
		for _, child := range pat.children {
			n, ok := v.matchAttrTokens(child, tokens[offset:])
			if !ok {
				if typeName := listDataTypeName(child); typeName != "" {
					v.addError(elem, fmt.Sprintf("failed to validate type %s", typeName))
				}
				return -1
			}
			offset += n
		}
		if offset < len(tokens) {
			v.addError(elem, fmt.Sprintf("Extra data in list: %s", tokens[offset]))
		}
	}
	return -1
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

// matchAttrTokens matches tokens against a pattern, returning how many tokens
// were consumed. It prefers the greedy (largest) match; callers needing the
// full set of consumption options use matchAttrTokensCounts.
func (v *validator) matchAttrTokens(pat *pattern, tokens []string) (int, bool) {
	counts := v.matchAttrTokensCounts(pat, tokens)
	if len(counts) == 0 {
		return 0, false
	}
	return counts[0], true
}

// matchAttrTokensCounts returns every token-consumption count that pat can
// match against the leading tokens, in greedy-preferred (descending) order
// with duplicates removed. An empty slice means the pattern cannot match.
//
// Returning the full set (rather than a single count) lets the group matcher
// backtrack: a greedy oneOrMore/zeroOrMore that over-consumes can yield tokens
// back when a later mandatory member fails, and a choice with a zero-token
// branch (e.g. empty) does not shadow a consuming branch.
func (v *validator) matchAttrTokensCounts(pat *pattern, tokens []string) []int {
	if pat == nil {
		return nil
	}
	switch pat.kind {
	case patternData:
		if len(tokens) == 0 {
			return nil
		}
		if v.matchData(pat, tokens[0]) == 0 {
			return []int{1}
		}
		return nil
	case patternValue:
		if len(tokens) == 0 {
			return nil
		}
		if v.matchValue(pat, tokens[0]) == 0 {
			return []int{1}
		}
		return nil
	case patternChoice:
		seen := map[int]struct{}{}
		var counts []int
		for _, child := range pat.children {
			for _, n := range v.matchAttrTokensCounts(child, tokens) {
				if _, ok := seen[n]; ok {
					continue
				}
				seen[n] = struct{}{}
				counts = append(counts, n)
			}
		}
		sortDescending(counts)
		return counts
	case patternOneOrMore:
		content := wrapChildren(pat.children)
		return v.repeatCounts(content, tokens, 1)
	case patternZeroOrMore:
		content := wrapChildren(pat.children)
		return v.repeatCounts(content, tokens, 0)
	case patternGroup:
		return v.groupCounts(pat.children, tokens)
	case patternList:
		// A <list> nested in a repetition (e.g. oneOrMore/zeroOrMore) consumes
		// one full run of its children per iteration; return the set of token
		// counts a sequential match of those children can consume so the
		// repetition machinery can chain iterations. This mirrors the
		// full-consumption check matchListContent performs for a standalone list.
		return v.groupCounts(pat.children, tokens)
	case patternText:
		// Text in a list: consume one token (or none when empty).
		if len(tokens) > 0 {
			return []int{1}
		}
		return []int{0}
	case patternEmpty:
		return []int{0}
	case patternRef, patternParentRef:
		def := pat.resolved
		if def == nil {
			return nil
		}
		return v.matchAttrTokensCounts(def, tokens)
	case patternOptional:
		content := wrapChildren(pat.children)
		seen := map[int]struct{}{0: {}}
		counts := []int{}
		for _, n := range v.matchAttrTokensCounts(content, tokens) {
			if n == 0 {
				continue
			}
			if _, ok := seen[n]; ok {
				continue
			}
			seen[n] = struct{}{}
			counts = append(counts, n)
		}
		counts = append(counts, 0)
		sortDescending(counts)
		return counts
	}
	return nil
}

// groupCounts returns every total consumption count for matching children
// sequentially against tokens, backtracking across each member's options.
//
// The set of counts children[ci:] can consume starting at tokens[off:] depends
// only on (ci, off), not on how that offset was reached, so it is memoized.
// Without this, the sibling enumeration is exponential — e.g. a group of N
// optionals over the tokens explores 2^N paths even though it has only N+1
// distinct totals.
func (v *validator) groupCounts(children []*pattern, tokens []string) []int {
	memo := map[[2]int][]int{}
	var rec func(ci, off int) []int
	rec = func(ci, off int) []int {
		if ci >= len(children) {
			return []int{0}
		}
		key := [2]int{ci, off}
		if cached, ok := memo[key]; ok {
			return cached
		}
		seen := map[int]struct{}{}
		var counts []int
		for _, n := range v.matchAttrTokensCounts(children[ci], tokens[off:]) {
			for _, m := range rec(ci+1, off+n) {
				total := n + m
				if _, ok := seen[total]; ok {
					continue
				}
				seen[total] = struct{}{}
				counts = append(counts, total)
			}
		}
		sortDescending(counts)
		memo[key] = counts
		return counts
	}
	return rec(0, 0)
}

// repeatCounts returns every consumption count for matching content repeatedly
// (at least minReps times) against tokens.
func (v *validator) repeatCounts(content *pattern, tokens []string, minReps int) []int {
	seen := map[int]struct{}{}     // offsets already recorded as results
	explored := map[int]struct{}{} // offsets whose deeper recursion is done
	var counts []int
	var recurse func(offset, reps int)
	recurse = func(offset, reps int) {
		if reps >= minReps {
			if _, ok := seen[offset]; !ok {
				seen[offset] = struct{}{}
				counts = append(counts, offset)
			}
		}
		// The offsets reachable from here don't depend on reps, so explore each
		// offset only once. Without this, overlapping repetition paths make the
		// enumeration exponential.
		if _, done := explored[offset]; done {
			return
		}
		explored[offset] = struct{}{}
		for _, n := range v.matchAttrTokensCounts(content, tokens[offset:]) {
			if n == 0 {
				// A zero-width match cannot make progress, so we must not
				// recurse (it would loop forever). But it is still a valid
				// iteration: if taking it once satisfies minReps, record the
				// current offset. This lets oneOrMore match nullable content
				// (optional/empty/text), matching node-level validateOneOrMore.
				if reps+1 >= minReps {
					if _, ok := seen[offset]; !ok {
						seen[offset] = struct{}{}
						counts = append(counts, offset)
					}
				}
				continue
			}
			recurse(offset+n, reps+1)
		}
	}
	recurse(0, 0)
	sortDescending(counts)
	return counts
}

// sortDescending sorts counts in place, largest first (greedy-preferred).
func sortDescending(counts []int) {
	slices.SortFunc(counts, func(a, b int) int { return cmp.Compare(b, a) })
}

func (v *validator) validateGroup(pat *pattern, state *validState) int {
	return v.validateGroupSeq(pat.children, state)
}

// validateGroupSeq validates a sequence of naive-group children with
// backtracking support. It is shared by validateGroup and by the recursive
// retry inside backtrackGroupNaive, so a backtracked sub-range with its own
// flexible members can itself backtrack. Results are memoized (see
// validateGroupChildren) to keep the cascading retry polynomial.
func (v *validator) validateGroupSeq(children []*pattern, state *validState) int {
	key, ok := v.groupMemoLookupKey(children, nil, nil, state, false)
	if ok {
		if e, hit := v.groupMemo[key]; hit {
			return e.apply(state, nil, v)
		}
	}
	errBase := len(v.pendingErrors)
	result := v.validateGroupSeqUncached(children, state)
	if ok {
		if v.groupMemo == nil {
			v.groupMemo = make(map[groupMemoKey]*groupMemoEntry)
		}
		v.groupMemo[key] = &groupMemoEntry{
			result: result,
			seq:    append([]helium.Node(nil), state.seq...),
			errs:   append([]error(nil), v.pendingErrors[errBase:]...),
		}
	}
	return result
}

func (v *validator) validateGroupSeqUncached(children []*pattern, state *validState) int {
	if len(children) == 0 {
		return 0
	}

	// Record state at each child boundary so a later mandatory member that
	// fails can ask a previous flexible member (zeroOrMore/oneOrMore/optional)
	// to yield items back. This mirrors validateGroupContent's backtracking,
	// minus the attribute/element-content bookkeeping that path threads.
	bounds := make([]groupBound, 1, len(children)+1)
	bounds[0] = saveGroupBound(state, nil, len(v.pendingErrors), v.valid)

	for gi, child := range children {
		if ret := v.validatePattern(child, state); ret != 0 {
			if gi > 0 && v.backtrackGroupNaive(children, gi, state, bounds) {
				bounds = append(bounds, saveGroupBound(state, nil, len(v.pendingErrors), v.valid))
				continue
			}
			return -1
		}
		bounds = append(bounds, saveGroupBound(state, nil, len(v.pendingErrors), v.valid))
	}
	return 0
}

// backtrackGroupNaive fixes a naive-group failure at failIdx by reducing the
// consumption of a previous flexible child (zeroOrMore/oneOrMore/optional). It
// tries each flexible child from nearest to furthest, and for each tries
// iteration counts from the minimum upward, preferring the highest count that
// lets the remaining children match (maximizing content consumption). It is the
// validatePattern-based counterpart to backtrackGroupFlexible.
//
// Two scope notes:
//   - The naive group path runs only for the top-level document sequence (a
//     single root element); multi-node sequences are element content, handled by
//     validateGroupContent. So `bounds[failIdx]` is always the freshly-appended
//     boundary and there is no multi-node cascade across separate backtrack calls
//     that could observe a stale intermediate `bounds` entry.
//   - Recovery reduces one flexible child and re-validates the rest via
//     validateGroupSeq, which itself backtracks. So a group with two or more
//     flexible members that must each yield (e.g. group(zeroOrMore(x),
//     zeroOrMore(x), x)) is recovered by cascading the reductions recursively.
//     The element-content path (backtrackGroupFlexible) mirrors this.
func (v *validator) backtrackGroupNaive(children []*pattern, failIdx int,
	state *validState, bounds []groupBound) bool {
	for j := failIdx - 1; j >= 0; j-- {
		child := children[j]
		isZeroFlex := child.kind == patternZeroOrMore || child.kind == patternOptional
		isOneMore := child.kind == patternOneOrMore
		if !isZeroFlex && !isOneMore {
			continue
		}
		// Skip flexible children that consumed nothing — nothing to yield back.
		if seqEqual(bounds[j].state.seq, bounds[j+1].state.seq) {
			continue
		}

		minIter := 0
		if isOneMore {
			minIter = 1
		}

		content := wrapChildren(child.children)

		var bestState *validState
		var bestErrLen int
		var bestValid bool

		for iter := minIter; ; iter++ {
			bounds[j].restore(state, nil, v)

			iterOK := true
			for range iter {
				savedSt := state.clone()
				v.suppressDepth++
				ret := v.validatePattern(content, state)
				v.suppressDepth--
				if ret != 0 || seqEqual(state.seq, savedSt.seq) {
					iterOK = false
					break
				}
			}
			if !iterOK {
				break
			}

			// Stop once we reach the greedy consumption level — that is the
			// state that already failed.
			if seqEqual(state.seq, bounds[j+1].state.seq) {
				break
			}

			retryLen := len(v.pendingErrors)
			retryValid := v.valid
			// Validate the remaining children as a group so that any flexible
			// members in [j+1..failIdx] can themselves backtrack. A flat greedy
			// retry would let a second flexible member re-grab content that a
			// later mandatory member needs.
			allOK := v.validateGroupSeq(children[j+1:failIdx+1], state) == 0
			if allOK {
				bestState = state.clone()
				bestErrLen = len(v.pendingErrors)
				bestValid = v.valid
			}
			v.pendingErrors = v.pendingErrors[:retryLen]
			v.valid = retryValid
		}

		if bestState != nil {
			*state = *bestState
			v.pendingErrors = v.pendingErrors[:bestErrLen]
			v.valid = bestValid
			return true
		}

		bounds[j].restore(state, nil, v)
	}
	return false
}

func (v *validator) validateChoice(pat *pattern, state *validState) int {
	// Save error state so failed branches don't pollute the output.
	savedLen := len(v.pendingErrors)
	savedValid := v.valid

	v.suppressDepth++
	// Prefer a branch that makes progress (consumes input) over a zero-length
	// match, so an early <empty/>/optional branch can't shadow a later
	// consuming branch (mirrors the hardened validateContentPat choice case).
	noProgressMatch := false
	for _, child := range pat.children {
		saved := state.clone()
		if ret := v.validatePattern(child, saved); ret != 0 {
			continue
		}
		if !seqEqual(saved.seq, state.seq) {
			// Branch made progress — use it.
			v.suppressDepth--
			v.pendingErrors = v.pendingErrors[:savedLen]
			v.valid = savedValid
			*state = *saved
			return 0
		}
		// Succeeded but consumed nothing — remember and keep trying.
		noProgressMatch = true
	}
	v.suppressDepth--

	// Restore error state (no branch errors emitted).
	v.pendingErrors = v.pendingErrors[:savedLen]
	v.valid = savedValid
	if noProgressMatch {
		return 0
	}
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

func (v *validator) validateOptional(pat *pattern, state *validState) int { //nolint:unparam // always 0 but matches validatePattern return contract
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

func (v *validator) validateZeroOrMore(pat *pattern, state *validState) int { //nolint:unparam // always 0 but matches validatePattern return contract
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
	def := pat.resolved
	if def == nil {
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
	// Delegate to the shared token matcher so every <list> path — element
	// content and this validatePattern path (lists nested under
	// optional/oneOrMore/choice, or at grammar.start) — uses the same
	// backtracking semantics. The previous hand-rolled matcher here only
	// handled data/value/(zero|one)OrMore-of-data and silently ignored
	// group/choice/optional/nested children. Pass a nil element: the naive
	// path has no element context for per-token error reporting.
	text := v.collectText(state)
	return v.matchListContent(pat, text, nil)
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
		case lexicon.NamespaceXSDDatatypes:
			return matchXSDValue(pat.dataType.name, text, expected)
		case "":
			// Empty datatypeLibrary selects the built-in RELAX NG library, which
			// provides only "string" (lexical equality) and "token" (whiteSpace
			// =collapse equality). Mirror matchData: recognized bare XSD type
			// names are routed through the same XSD value path used there
			// (documented libxml2/golden-compat deviation; see matchData and the
			// token-matcher tests cited there); an unknown name fails rather than
			// silently matching by raw equality.
			switch pat.dataType.name {
			case lexicon.TypeToken:
				text = normalizeToken(text)
				expected = normalizeToken(expected)
			case "string":
				// Lexical equality below.
			default:
				// Only fall back to the XSD value path when datatypeLibrary is
				// genuinely absent; an explicit "" reset rejects bare XSD names.
				if _, ok := xsdDatatypeNames[pat.dataType.name]; ok && !pat.dataType.libraryDeclared {
					return matchXSDValue(pat.dataType.name, text, expected)
				}
				return -1
			}
		default:
			// Unknown datatype library: an unsupported <value> datatype cannot be
			// satisfied, so it must fail rather than fall through to raw equality.
			return -1
		}
	}

	if text == expected {
		return 0
	}
	return -1
}

// xsdValueSpaceTypes is the set of XSD datatypes for which value.Compare
// implements correct value-space equality. It mirrors the XSD layer's
// enumValueSpaceTypes and deliberately excludes string-family and anyURI types:
// value.Compare falls back to decimal comparison for unrecognized builtins, which
// would wrongly treat numeric-looking lexicals as equal (e.g. a string "5"
// matching "5.0"). Those types compare by their whitespace-processed lexical
// value, which equals their value space.
var xsdValueSpaceTypes = map[string]struct{}{
	"decimal": {}, "integer": {}, "nonPositiveInteger": {}, "negativeInteger": {},
	"long": {}, "int": {}, "short": {}, "byte": {},
	"nonNegativeInteger": {}, "unsignedLong": {}, "unsignedInt": {},
	"unsignedShort": {}, "unsignedByte": {}, lexicon.TypePositiveInteger: {},
	"float": {}, "double": {},
	"boolean":  {},
	"dateTime": {}, "date": {}, "time": {}, "duration": {},
	"gYear": {}, "gYearMonth": {}, "gMonth": {}, "gDay": {}, "gMonthDay": {},
	"hexBinary": {}, "base64Binary": {},
}

// matchXSDValue compares an instance value against a <value> literal using the
// XSD value space for the named datatype. For every recognized XSD datatype,
// both the instance text and the <value> literal must first be lexically valid
// for the type (via value.ValidateBuiltin); an invalid lexical never matches,
// even when the two forms are identical (e.g. type="NCName">1foo< does not
// accept <e>1foo</e>, and type="integer">5.0< does not accept <e>5.0</e>). For
// value-space-comparable types (numeric, boolean, date/time, binary) a
// lexically distinct but value-equal form then also matches (e.g. integer
// "5" == "+5" == "05"), agreeing with the XSD layer. String-family and anyURI
// types stay lexical-only (whitespace-processed lexical equality, which equals
// their value space). An unknown datatype name never matches.
func matchXSDValue(typeName, text, expected string) int {
	if _, ok := xsdDatatypeNames[typeName]; !ok {
		return -1
	}
	// Normalize both the instance text and the <value> literal using the
	// datatype's XSD whiteSpace facet (preserve/replace/collapse) before
	// validating or comparing, rather than a blanket TrimSpace. This lets a
	// collapsible token like "a  b" validate and value-match "a b", while leaving
	// xs:string untouched (preserve).
	text = value.Normalize(text, typeName)
	expected = value.Normalize(expected, typeName)

	// Both the instance text and the <value> literal must be lexically valid for
	// the type before either the equality fast-path or value-space comparison may
	// accept. This gate runs for every recognized type so that an invalid lexical
	// (e.g. "1foo" for NCName, or "5.0" for integer) is rejected even when the two
	// forms are byte-identical. value.ValidateBuiltin imposes no constraint on
	// xs:string / xs:anyURI, so those stay effectively lexical-only.
	if value.ValidateBuiltin(text, typeName, value.Version10) != nil || value.ValidateBuiltin(expected, typeName, value.Version10) != nil {
		return -1
	}

	if _, ok := xsdValueSpaceTypes[typeName]; !ok {
		// Lexical-only (string-family/anyURI) type: compare by whitespace-processed
		// lexical value, which equals the value space for these types.
		if text == expected {
			return 0
		}
		return -1
	}

	// Value-space-comparable type: a lexically distinct but value-equal form
	// matches (e.g. integer "5" == "+5" == "05", NaN == NaN for float/double).
	if text == expected {
		return 0
	}
	if (typeName == "float" || typeName == "double") && value.IsFloatNaN(text) && value.IsFloatNaN(expected) {
		return 0
	}
	if cmpResult, ok := value.Compare(text, expected, typeName); ok && cmpResult == 0 {
		return 0
	}
	return -1
}

// matchData checks if text matches a data type pattern, including any
// <except> patterns nested under the <data>. The base datatype must match
// first; then any value excluded by the except patterns is rejected.
func (v *validator) matchData(pat *pattern, text string) int {
	if ret := v.matchDataType(pat, text); ret != 0 {
		return ret
	}
	// A <data> may carry an <except> (stored as a patternChoice child by
	// parseData). The base datatype matched above, so a value that also matches
	// any except pattern must be rejected.
	if v.matchesExcept(pat, text) {
		return -1
	}
	return 0
}

// matchesExcept reports whether text matches any of the <except> patterns
// stored under a <data> pattern. Each <except> is compiled into a patternChoice
// child; its alternatives are value/data patterns (optionally combined with
// nested choice).
func (v *validator) matchesExcept(pat *pattern, text string) bool {
	for _, child := range pat.children {
		if v.matchExceptPattern(child, text) {
			return true
		}
	}
	return false
}

// matchExceptPattern reports whether text matches a single pattern appearing
// inside a <data>'s <except>. Only value, data, and choice patterns are valid
// there per the RELAX NG schema for except-in-data.
func (v *validator) matchExceptPattern(ex *pattern, text string) bool {
	switch ex.kind {
	case patternData:
		return v.matchData(ex, text) == 0
	case patternValue:
		return v.matchValue(ex, text) == 0
	case patternChoice:
		for _, c := range ex.children {
			if v.matchExceptPattern(c, text) {
				return true
			}
		}
	}
	return false
}

// matchDataType checks if text matches a data type pattern's base datatype,
// without considering any <except> patterns.
func (v *validator) matchDataType(pat *pattern, text string) int {
	if pat.dataType == nil {
		return 0
	}

	dt := pat.dataType

	switch dt.library {
	case lexicon.NamespaceXSDDatatypes:
		// validateXSDType applies the per-datatype XSD whiteSpace facet itself, so
		// the raw (un-trimmed) text is passed through.
		return validateXSDType(dt.name, text, pat.params)
	case "":
		// Built-in RELAX NG datatypes: both "token" and "string" accept any text.
		switch dt.name {
		case lexicon.TypeToken:
			return 0
		case "string":
			return 0
		}
		// Spec note: per RELAX NG, an omitted datatypeLibrary selects the EMPTY
		// built-in library, which provides ONLY "string" and "token"; XSD
		// datatypes strictly require datatypeLibrary
		// "http://www.w3.org/2001/XMLSchema-datatypes". A spec-conformant
		// processor would reject <data type="integer"/> here.
		//
		// We deliberately deviate for libxml2/golden compatibility: libxml2's
		// RELAX NG engine resolves bare XSD datatype names against its built-in
		// XSD library. The hand-written token-matcher unit tests rely on this:
		// their schemas use bare <data type="integer"/> and <data type="NMTOKEN"/>
		// with NO datatypeLibrary declared or inherited anywhere (the root is a
		// plain <element xmlns="…relaxng…structure/1.0">, no <grammar> ancestor
		// and no datatypeLibrary attribute). Specifically required by:
		//   - TestTokenMatcherChoiceShadow / ...ChoiceShadowAttr (token_matcher_test.go)
		//   - TestTokenMatcherGroupBacktrack (token_matcher_test.go, bare NMTOKEN)
		//   - TestTokenMatcherNullableOneOrMore / ...NoExponentialBlowup
		//     (token_matcher_review_test.go, bare integer)
		// Verified by grepping those fixtures: removing this fallback breaks them.
		// Routing only RECOGNIZED XSD datatype names through the shared validator
		// keeps those green while still validating the value (so e.g. an
		// out-of-range integer is rejected). Crucially, a truly-unknown name such
		// as type="bogus" is NOT in xsdDatatypeNames and falls through to the
		// failure below, fixing the original "empty library matches anything" bug.
		//
		// The fallback applies ONLY when datatypeLibrary is genuinely ABSENT
		// (libraryDeclared == false). An explicit datatypeLibrary="" — including
		// one that resets an inherited XSD library — selects the built-in library
		// conformantly and so rejects bare XSD names like "integer".
		if _, ok := xsdDatatypeNames[dt.name]; ok && !dt.libraryDeclared {
			return validateXSDType(dt.name, text, pat.params)
		}
	}

	// Unknown built-in datatype name or unknown library: an unsupported datatype
	// cannot be satisfied, so it must fail rather than silently match everything.
	return -1
}

// xsdDatatypeNames is the set of XSD datatype-library names RELAX NG recognizes.
// Validation of any of these is routed through the shared XSD value validator
// (internal/xsd/value) so RELAX NG and XSD agree on lexical/value spaces. Any
// name outside this set is an unknown datatype and is rejected (a <data> against
// an unsupported XSD type cannot be satisfied), rather than silently accepted.
var xsdDatatypeNames = map[string]struct{}{
	"string": {}, "normalizedString": {}, "token": {},
	"integer": {}, "int": {}, "long": {}, "short": {}, "byte": {},
	"nonNegativeInteger": {}, lexicon.TypePositiveInteger: {},
	"nonPositiveInteger": {}, "negativeInteger": {},
	"unsignedInt": {}, "unsignedLong": {}, "unsignedShort": {}, "unsignedByte": {},
	"decimal": {}, "float": {}, "double": {}, "boolean": {},
	"date": {}, "dateTime": {}, "time": {}, "duration": {},
	"gYear": {}, "gYearMonth": {}, "gMonth": {}, "gDay": {}, "gMonthDay": {},
	"anyURI": {}, "QName": {}, "NOTATION": {},
	"NCName": {}, "ID": {}, "IDREF": {}, "ENTITY": {}, "Name": {},
	"NMTOKEN": {}, "NMTOKENS": {}, "IDREFS": {}, "ENTITIES": {},
	"language": {}, "base64Binary": {}, "hexBinary": {},
}

// effectiveXSDDatatype returns the XSD builtin datatype local name a <data>
// dataType resolves to for facet-applicability purposes, and whether it is an
// XSD-validated type. It mirrors matchData's resolution exactly so the
// compile-time facet check (checkDataFacets) and runtime validation agree: an
// explicit XSD datatype library names an XSD type directly, and — when no
// datatypeLibrary was declared anywhere — a bare recognized XSD name resolves to
// its XSD type (the libxml2 compatibility fallback) EXCEPT bare "string"/"token",
// which matchData resolves first as the EMPTY built-in RELAX NG library (accept
// any text, no XSD facets). Anything else (the built-in string/token library, an
// explicit datatypeLibrary="" reset, or an unknown name) is not an XSD-validated
// type and returns ("", false).
func effectiveXSDDatatype(dt *dataType) (string, bool) {
	if dt == nil {
		return "", false
	}
	switch dt.library {
	case lexicon.NamespaceXSDDatatypes:
		_, ok := xsdDatatypeNames[dt.name]
		return dt.name, ok
	case "":
		// matchData resolves bare "string"/"token" as the EMPTY built-in RELAX NG
		// library FIRST (they accept any text and apply no XSD facets), before the
		// libxml2 bare-XSD fallback. So those two names are never XSD-validated at
		// runtime — treating them as XSD here would raise spurious facet errors at
		// compile time (e.g. an ordering facet on the built-in "token"). Only the
		// other bare recognized XSD names take the fallback path.
		if dt.name == lexicon.TypeString || dt.name == lexicon.TypeToken {
			return "", false
		}
		if _, ok := xsdDatatypeNames[dt.name]; ok && !dt.libraryDeclared {
			return dt.name, true
		}
	}
	return "", false
}

// validateXSDType validates a value against an XSD datatype. Recognized type
// names are validated through the shared XSD value validator so RELAX NG and
// XSD stay consistent (date/time/duration value-space ranges, integer subtype
// ranges, binary alphabets, …). xs:string carries RELAX NG <param> facets, so it
// stays on the local param path. Unknown type names are rejected.
func validateXSDType(typeName, text string, params []*param) int {
	if _, ok := xsdDatatypeNames[typeName]; !ok {
		return -1
	}
	// xs:string has whiteSpace=preserve and may carry RELAX NG <param> facets
	// (pattern, length, …); validate those locally against the preserved text.
	if typeName == "string" {
		return validateWithParams(text, typeName, params)
	}
	// Apply the datatype's XSD whiteSpace facet (replace for normalizedString,
	// collapse for everything else here) before lexical validation, so a value
	// such as xs:token "a  b" (collapses to "a b") is accepted.
	text = value.Normalize(text, typeName)
	if value.ValidateBuiltin(text, typeName, value.Version10) != nil {
		return -1
	}
	// Apply the supported length facets to the whitespace-normalized value for
	// every applicable XSD datatype, not just xs:string. The length facets are
	// defined on the value's length in the units fixed by the datatype (see
	// facetLength).
	return validateWithParams(text, typeName, params)
}

// validateWithParams applies the RELAX NG <param> facets carried by a <data>
// pattern to an already whitespace-normalized, lexically-valid instance value.
// The parameter is named text (not value) so the internal/xsd/value package
// stays reachable for the ordering-facet comparisons below.
//
// The length facets (length/minLength/maxLength) are measured via facetLength on
// every applicable type — including xs:QName and xs:NOTATION, where they are
// CONSTRAINING (XSD 1.0 / libxml2 parity, matching the xsd package's facetLength)
// rather than a no-op, so RELAX NG rejects an out-of-length QName/NOTATION exactly
// as the shared xsd validator does;
// the ordering facets (min/maxInclusive, min/maxExclusive) are evaluated through
// the shared XSD value engine (value.Compare) so RELAX NG and XSD agree on the
// value space (e.g. xs:integer "5" really is less than "10"); and the digit
// facets (totalDigits, fractionDigits) are measured via value.CountTotalDigits/
// CountFractionDigits on the xs:decimal family; and the pattern facet is matched
// through the shared XSD-regex engine (xsdregex) so XSD-only constructs (\i, \c,
// \p{...}, …) are honoured. Facet applicability — an ordering facet on a
// non-ordered type, a digit facet on a non-decimal type, a length facet on a
// non-string-derived type, and bound/pattern validity — is enforced at compile
// time (checkDataFacets), so the grammar is already unmatchable when an
// inapplicable or invalid facet is present. Any other param name
// — enumeration, whiteSpace, or an unknown facet — is unsupported and fails
// closed: it cannot be silently accepted, which would let a <data> match values
// the facet was meant to reject.
func validateWithParams(text, typeName string, params []*param) int {
	for _, p := range params {
		switch p.name {
		case "pattern":
			// The pattern facet is an XSD/XPath regular expression, anchored to the
			// whole value. Use the compilation cached at compile time (checkDataFacets);
			// fall back to compiling here for robustness if it was not pre-compiled. A
			// pattern that cannot compile fails closed.
			re := p.compiledPattern
			if re == nil {
				compiled, err := xsdregex.Compile(p.value)
				if err != nil {
					return -1
				}
				re = compiled
			}
			if !re.MatchString(text) {
				return -1
			}
		case "length":
			// The bound is compile-time validated as an xs:nonNegativeInteger and
			// parsed width-safely (big.Int) so a huge-but-valid bound is compared
			// faithfully rather than overflowing int into a reject-all.
			n, ok := parseNonNegFacetBound(p.value)
			if !ok {
				return -1
			}
			length, ok := facetLength(text, typeName)
			if !ok || big.NewInt(int64(length)).Cmp(n) != 0 {
				return -1
			}
		case "minLength":
			n, ok := parseNonNegFacetBound(p.value)
			if !ok {
				return -1
			}
			length, ok := facetLength(text, typeName)
			if !ok || big.NewInt(int64(length)).Cmp(n) < 0 {
				return -1
			}
		case "maxLength":
			n, ok := parseNonNegFacetBound(p.value)
			if !ok {
				return -1
			}
			length, ok := facetLength(text, typeName)
			if !ok || big.NewInt(int64(length)).Cmp(n) > 0 {
				return -1
			}
		case "minInclusive":
			if !facetOrderingOK(text, p.value, typeName, func(cmp int) bool { return cmp >= 0 }) {
				return -1
			}
		case "maxInclusive":
			if !facetOrderingOK(text, p.value, typeName, func(cmp int) bool { return cmp <= 0 }) {
				return -1
			}
		case "minExclusive":
			if !facetOrderingOK(text, p.value, typeName, func(cmp int) bool { return cmp > 0 }) {
				return -1
			}
		case "maxExclusive":
			if !facetOrderingOK(text, p.value, typeName, func(cmp int) bool { return cmp < 0 }) {
				return -1
			}
		case "totalDigits":
			// totalDigits applies only to the xs:decimal family (an inapplicable
			// facet, and a bound that is not a valid xs:positiveInteger, are rejected
			// at compile time). The instance value has already passed lexical
			// validation, so count its significant digits and reject when they exceed
			// the bound. The bound is parsed with big.Int so an arbitrarily large
			// xs:positiveInteger does not overflow into a reject-all.
			n, ok := parseNonNegFacetBound(p.value)
			if !ok || !value.IsDecimalFamily(typeName) {
				return -1
			}
			if big.NewInt(int64(value.CountTotalDigits(text))).Cmp(n) > 0 {
				return -1
			}
		case "fractionDigits":
			n, ok := parseNonNegFacetBound(p.value)
			if !ok || !value.IsDecimalFamily(typeName) {
				return -1
			}
			if big.NewInt(int64(value.CountFractionDigits(text))).Cmp(n) > 0 {
				return -1
			}
		default:
			// Unsupported / unrecognized facet (enumeration, whiteSpace, or an
			// unknown name). Fail closed rather than silently accept: an unenforced
			// facet must not let the value through.
			return -1
		}
	}
	return 0
}

// facetOrderingOK reports whether the instance value satisfies an ordering facet
// (min/maxInclusive, min/maxExclusive) whose bound lexical is facetBound. The
// instance value and the bound are compared in the datatype's value space via the
// shared XSD engine (value.Compare). accept is given the comparison sign of text
// relative to facetBound (-1/0/+1) and decides whether the facet is met.
//
// Ordering facets are defined ONLY on a datatype whose value space is ordered
// (value.Orderable): value.Compare returns a deterministic total order even for
// the non-ordered types (boolean, hexBinary, base64Binary) so enumeration can use
// cmp==0, but that order must never fire a range facet. Such facets are rejected
// at compile time (checkDataFacets makes the grammar unmatchable), so this is
// belt-and-suspenders — a non-ordered type fails closed if ever reached.
//
// For an ordered type, value.Compare can still return ok=false when two VALID
// operands are indeterminate. There are two distinct cases, and XSD treats them
// oppositely:
//
//   - An xs:float/xs:double NaN operand (a NaN instance value OR a NaN bound):
//     NaN does not participate in the order, and XSD bounding facets EXCLUDE
//     incomparable values, so the facet is NOT satisfied. This must reject — a
//     minInclusive=0 must not accept a NaN instance, and a NaN bound must not
//     accept finite values.
//   - A genuinely-indeterminate comparison between two non-NaN ordered values
//     (e.g. a mixed-timezone xs:dateTime comparison): XSD treats this as
//     SATISFYING the range facet, so the value is accepted — matching the XSD
//     layer's semantics.
func facetOrderingOK(text, facetBound, typeName string, accept func(int) bool) bool {
	if !value.Orderable(typeName) {
		return false
	}
	// Normalize the bound per the datatype's XSD whitespace facet (float/double are
	// collapse) BEFORE comparing it: a raw bound such as "<param>" NaN "</param>"
	// carries surrounding whitespace that value.Compare trims away (so it reads NaN
	// and returns indeterminate) but value.IsFloatNaN does not, so the unnormalized
	// " NaN " bound would be seen as non-NaN below and let a finite instance through.
	facetBound = value.Normalize(facetBound, typeName)
	cmp, ok := value.Compare(text, facetBound, typeName)
	if !ok {
		// A NaN operand (instance value or bound) is incomparable for the bounding
		// facets, so the value fails the facet rather than slipping through. Other
		// indeterminate ordered comparisons remain treated as satisfied.
		if value.IsFloatNaN(text) || value.IsFloatNaN(facetBound) {
			return false
		}
		return true
	}
	return accept(cmp)
}

// parseNonNegFacetBound parses an integer facet bound — totalDigits,
// fractionDigits, or the length family (length/minLength/maxLength) — into a
// big.Int. The bound is compile-time validated (checkDataFacets) as a valid
// xs:positiveInteger (totalDigits) or xs:nonNegativeInteger (fractionDigits and
// the length facets), so by the time validation runs it is a well-formed integer
// lexical with only XSD whitespace; it is normalized (XSD collapse — NOT Go's
// unicode-space trimming, which would accept NBSP) before parsing. A big.Int is
// used rather than strconv.Atoi so an arbitrarily large bound is compared
// faithfully instead of overflowing int and collapsing into a reject-all.
func parseNonNegFacetBound(s string) (*big.Int, bool) {
	n, ok := new(big.Int).SetString(value.Normalize(s, "nonNegativeInteger"), 10)
	return n, ok
}

// facetLength returns the value's length in the units the length/minLength/
// maxLength facets are defined in for the given XSD datatype, and ok=true when
// that length is well-defined:
//   - list builtins (NMTOKENS, IDREFS, ENTITIES): the number of XML-whitespace
//     -separated tokens (the list length).
//   - binary builtins (hexBinary, base64Binary): the number of decoded octets.
//   - everything else (string family, NMTOKEN, …): the number of characters
//     (runes) in the value.
//
// For the binary builtins the length is measured in decoded octets, so if the
// value cannot be decoded ok=false is returned and the facet must be treated as
// unsatisfiable: the rune count of an undecodable binary value is meaningless,
// and falling back to it would silently compare lengths in the wrong unit. A
// value that has passed lexical validation for typeName decodes successfully, so
// this only guards against an undecodable value reaching the facet check.
//
// xsd has a same-named facetLength (xsd/simplevalue_facets.go) that deliberately
// approximates binary lengths instead of strict-decoding; the two are kept
// separate on purpose — see that function's comment.
func facetLength(value, typeName string) (int, bool) {
	switch typeName {
	case "NMTOKENS", "IDREFS", "ENTITIES":
		return len(xmlFields(value)), true
	case "hexBinary":
		return hexBinaryOctets(value)
	case "base64Binary":
		octets, ok := decodeBase64Octets(value)
		if !ok {
			return 0, false
		}
		return len(octets), true
	}
	return utf8.RuneCountInString(value), true
}

// hexBinaryOctets returns the decoded octet count of an xs:hexBinary lexical
// value. Each octet is two hex digits, so for a lexically valid value this is
// simply len/2.
func hexBinaryOctets(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if len(s)%2 != 0 {
		return 0, false
	}
	if _, err := hex.DecodeString(s); err != nil {
		return 0, false
	}
	return len(s) / 2, true
}

// decodeBase64Octets decodes an xs:base64Binary lexical value to its octets,
// mirroring the strict decoder used by the xsd/value comparison path so the two
// agree on what counts as a valid base64Binary. XSD permits embedded whitespace,
// which is stripped first; the remainder must then be correctly padded and have
// zero unused trailing bits (Strict()), so unpadded forms such as "TQ" or padded
// forms with non-zero trailing bits such as "TR==" fail to decode rather than
// yielding a bogus octet count.
func decodeBase64Octets(s string) ([]byte, bool) {
	stripped := strings.Map(func(r rune) rune {
		if isXMLSpace(r) {
			return -1
		}
		return r
	}, s)
	decoded, err := base64.StdEncoding.Strict().DecodeString(stripped)
	if err != nil {
		return nil, false
	}
	return decoded, true
}

// normalizeToken collapses runs of XML whitespace (#x20, #x9, #xA, #xD) into a
// single space and trims leading/trailing XML whitespace, matching the
// xs:token whiteSpace=collapse facet. It splits on XML whitespace only (via
// xmlFields), not arbitrary Unicode whitespace, so NBSP is preserved within a
// token.
func normalizeToken(s string) string {
	return strings.Join(xmlFields(s), " ")
}

// isXMLSpace reports whether r is one of the four XML whitespace characters
// (#x20, #x9, #xA, #xD). RELAX NG / XSD list tokenization is defined in terms
// of these only, unlike strings.Fields which splits on all Unicode whitespace
// (e.g. NBSP, which must remain part of a token).
func isXMLSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

// xmlFields splits s into tokens on XML whitespace only, discarding empty
// fields. It is the XML-whitespace analogue of strings.Fields.
func xmlFields(s string) []string {
	return strings.FieldsFunc(s, isXMLSpace)
}

// isXMLSpaceOnly reports whether s consists entirely of XML whitespace (or is
// empty), using XML's whitespace definition rather than Unicode's. Unlike
// strings.TrimSpace-based emptiness checks it does not treat other Unicode
// whitespace (such as NBSP) as whitespace, since those characters are
// significant content in XML and must not be silently discarded.
func isXMLSpaceOnly(s string) bool {
	return strings.IndexFunc(s, func(r rune) bool { return !isXMLSpace(r) }) < 0
}

// addError adds a validation error (suppressed when inside choice branches).
func (v *validator) addError(elem *helium.Element, msg string) {
	if v.suppressDepth > 0 {
		return
	}
	line := elem.Line()
	errStr := validityError(v.filename, line, elem.LocalName(), msg)
	v.pendingErrors = append(v.pendingErrors, helium.NewLeveledError(errStr, helium.ErrorLevelError))
	v.valid = false
}

// addBareError adds a validation error without file/line/element context.
func (v *validator) addBareError(msg string) {
	if v.suppressDepth > 0 {
		return
	}
	errStr := bareValidityError(msg)
	v.pendingErrors = append(v.pendingErrors, helium.NewLeveledError(errStr, helium.ErrorLevelError))
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
	case patternRef, patternParentRef:
		def := pat.resolved
		if def == nil {
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
		return slices.ContainsFunc(pat.children, v.isNullable)
	case patternGroup, patternInterleave:
		for _, child := range pat.children {
			if !v.isNullable(child) {
				return false
			}
		}
		return true
	case patternRef, patternParentRef:
		def := pat.resolved
		if def == nil {
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
				if isXMLSpaceOnly(string(t.Content())) {
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
			def := pat.resolved
			if def == nil {
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
