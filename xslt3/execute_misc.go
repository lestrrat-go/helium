package xslt3

import (
	"context"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/internal/sequence"
)

func (ec *execContext) execAnalyzeString(ctx context.Context, inst *analyzeStringInst) error {
	// Evaluate the select expression
	result, err := ec.evalXPath(inst.Select, ec.contextNode)
	if err != nil {
		return err
	}
	seq := result.Sequence()

	// The select expression must produce a single xs:string value.
	// A sequence of more than one item, or a non-string item, is XPTY0004.
	isV2 := ec.stylesheet.version != "" && ec.stylesheet.version < "3.0"
	if seq == nil || sequence.Len(seq) == 0 {
		if isV2 {
			// XSLT 2.0: empty sequence is XPTY0004
			return dynamicError(errCodeXPTY0004, "xsl:analyze-string select must be a single xs:string, got empty sequence")
		}
		// XSLT 3.0: empty sequence treated as ""
		return nil
	}
	if sequence.Len(seq) > 1 {
		return dynamicError(errCodeXPTY0004, "xsl:analyze-string select must be a single xs:string, got sequence of %d items", sequence.Len(seq))
	}
	av, err := xpath3.AtomizeItem(seq.Get(0))
	if err != nil {
		return dynamicError(errCodeXPTY0004, "xsl:analyze-string select must be xs:string: %v", err)
	}
	// Reject non-string atomic types (xs:integer, etc.)
	if av.TypeName != xpath3.TypeString && av.TypeName != xpath3.TypeUntypedAtomic && av.TypeName != xpath3.TypeAnyURI {
		return dynamicError(errCodeXPTY0004, "xsl:analyze-string select must be xs:string, got %s", av.TypeName)
	}
	input, err := xpath3.AtomicToString(av)
	if err != nil {
		return dynamicError(errCodeXPTY0004, "xsl:analyze-string select must be xs:string: %v", err)
	}

	// Evaluate regex avt
	regex, err := inst.Regex.evaluate(ctx, ec.contextNode)
	if err != nil {
		return err
	}

	// Evaluate flags avt
	flags := ""
	if inst.Flags != nil {
		flags, err = inst.Flags.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
	}

	// Version 2.0 restrictions
	if isV2 {
		// XSLT 2.0: 'q' flag is not allowed (XTDE1145)
		if strings.ContainsRune(flags, 'q') {
			return dynamicError(errCodeXTDE1145, "xsl:analyze-string flag 'q' is not allowed in XSLT 2.0")
		}
		// XSLT 2.0: non-capturing groups (?:...) are not allowed (XTDE1140)
		if strings.Contains(regex, "(?:") {
			return dynamicError(errCodeXTDE1140, "non-capturing groups are not allowed in XSLT 2.0 regex")
		}
	}

	// Compile the regex using XPath regex semantics
	re, err := xpath3.CompileRegex(regex, flags)
	if err != nil {
		// Map XPath regex errors to XSLT error codes
		return dynamicError(errCodeXTDE1140, "xsl:analyze-string invalid regex: %v", err)
	}

	// XSLT 3.0 removed the XTDE1150 error for zero-length regex matches.
	// Zero-length matches are handled by advancing past each one to avoid
	// infinite loops (see below).

	// Find all matches.
	// In XSLT 3.0, zero-length matches are allowed (unlike XSLT 2.0
	// which raised XTDE1150). We handle them by advancing past each
	// zero-length match to avoid infinite loops.
	matches, err := re.FindAllSubmatchIndex(input, -1)
	if err != nil {
		return dynamicError(errCodeXTDE1140, "xsl:analyze-string regex match error: %v", err)
	}

	// Save and restore context state
	savedNode := ec.contextNode
	savedCurrent := ec.currentNode
	savedItem := ec.contextItem
	savedPos := ec.position
	savedSize := ec.size
	savedGroups := ec.regexGroups
	defer func() {
		ec.contextNode = savedNode
		ec.currentNode = savedCurrent
		ec.contextItem = savedItem
		ec.position = savedPos
		ec.size = savedSize
		ec.regexGroups = savedGroups
	}()

	// Build segments: alternating non-match/match segments
	type segment struct {
		text    string
		isMatch bool
		groups  []string // captured groups (only for matches)
	}
	var segments []segment
	pos := 0
	for _, m := range matches {
		start, end := m[0], m[1]
		if start > pos {
			segments = append(segments, segment{text: input[pos:start], isMatch: false})
		}
		// Collect captured groups
		var groups []string
		groups = append(groups, input[start:end]) // group 0 = full match
		for g := 1; g < len(m)/2; g++ {
			gs, ge := m[2*g], m[2*g+1]
			if gs < 0 || ge < 0 {
				groups = append(groups, "")
			} else {
				groups = append(groups, input[gs:ge])
			}
		}
		segments = append(segments, segment{text: input[start:end], isMatch: true, groups: groups})
		pos = end
	}
	if pos < len(input) {
		segments = append(segments, segment{text: input[pos:], isMatch: false})
	}

	// Set size = total number of segments
	totalSegments := len(segments)

	// Execute appropriate body for each segment
	for i, seg := range segments {
		ec.position = i + 1
		ec.size = totalSegments
		ec.contextItem = xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: seg.text}
		ec.contextNode = nil
		ec.currentNode = nil

		if seg.isMatch {
			ec.regexGroups = seg.groups
			for _, bodyInst := range inst.MatchingBody {
				if err := ec.executeInstruction(ctx, bodyInst); err != nil {
					return err
				}
			}
		} else {
			ec.regexGroups = nil
			for _, bodyInst := range inst.NonMatchingBody {
				if err := ec.executeInstruction(ctx, bodyInst); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (ec *execContext) execWherePopulated(ctx context.Context, inst *wherePopulatedInst) error {
	out := ec.currentOutput()

	// When the outer frame captures items (e.g. inside xsl:variable with
	// as="map(*)?"), use sequence mode so maps/arrays aren't rejected as
	// XTDE0450. Delegate filtering to isItemSequencePopulated.
	if out.captureItems {
		return ec.execWherePopulatedSequence(ctx, inst)
	}

	// Execute body into a temporary document, then filter per XSLT 3.0 section 8.4.
	tmpDoc := helium.NewDefaultDocument()
	tmpRoot := tmpDoc.CreateElement("_tmp")
	if err := tmpDoc.AddChild(tmpRoot); err != nil {
		return err
	}

	ec.outputStack = append(ec.outputStack, &outputFrame{doc: tmpDoc, current: tmpRoot, wherePopulated: true})

	if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
		ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
		return err
	}

	ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]

	// Collect surviving attributes (non-empty value) and surviving child nodes.
	var survivingAttrs []*helium.Attribute
	for _, attr := range tmpRoot.Attributes() {
		if len(attr.Content()) > 0 {
			survivingAttrs = append(survivingAttrs, attr)
		}
	}

	hasSignificantChild := false
	for child := range helium.Children(tmpRoot) {
		if isPopulated(child) {
			hasSignificantChild = true
			break
		}
	}

	// If nothing survives (no populated attrs and no populated children),
	// discard the whole result.
	if len(survivingAttrs) == 0 && !hasSignificantChild {
		return nil
	}

	// Emit surviving attributes to the real output.
	for _, attr := range survivingAttrs {
		if elem, ok := out.current.(*helium.Element); ok {
			if elem.FirstChild() != nil {
				return dynamicError(errCodeXTRE0540,
					"cannot add attribute to element after children have been added")
			}
			if attr.URI() != "" {
				ns, _ := out.doc.CreateNamespace(attr.Prefix(), attr.URI())
				if err := elem.SetLiteralAttributeNS(attr.LocalName(), string(attr.Content()), ns); err != nil {
					return err
				}
			} else {
				if _, err := elem.SetAttribute(attr.LocalName(), string(attr.Content())); err != nil {
					return err
				}
			}
			out.noteOutput()
		}
	}

	// Copy significant child nodes to real output.
	for child := range helium.Children(tmpRoot) {
		if !isPopulated(child) {
			continue
		}
		if child.Type() == helium.DocumentNode {
			doc := child.(*helium.Document)
			for dc := range helium.Children(doc) {
				copied, copyErr := helium.CopyNode(dc, ec.resultDoc)
				if copyErr != nil {
					return copyErr
				}
				if err := ec.addNode(copied); err != nil {
					return err
				}
			}
			continue
		}
		copied, copyErr := helium.CopyNode(child, ec.resultDoc)
		if copyErr != nil {
			return copyErr
		}
		if err := ec.addNode(copied); err != nil {
			return err
		}
	}
	return nil
}

// execWherePopulatedSequence handles xsl:where-populated inside a sequence
// context (xsl:variable with as= or similar). The body is evaluated as a
// sequence and items are filtered using the XSLT 3.0 populated-item rules.
func (ec *execContext) execWherePopulatedSequence(ctx context.Context, inst *wherePopulatedInst) error {
	seq, err := ec.evaluateBodyAsSequence(ctx, inst.Body)
	if err != nil {
		return err
	}
	if !isItemSequencePopulated(seq) {
		return nil
	}
	out := ec.currentOutput()
	out.pendingItems = append(out.pendingItems, sequence.Materialize(seq)...)
	out.noteOutput()
	return nil
}

// isPopulated checks if a node is "populated" per XSLT 3.0 xsl:where-populated semantics
// (section 11.1.8). A node N is significant unless:
//   - N is a text node with zero-length string value
//   - N is a comment node with zero-length string value
//   - N is a processing-instruction node with zero-length string value
//   - N is an element or document node with no significant children
//
// Element nodes are always significant.
func isPopulated(node helium.Node) bool {
	switch node.Type() {
	case helium.ElementNode, helium.DocumentNode:
		for child := range helium.Children(node) {
			switch child.Type() {
			case helium.ElementNode:
				return true
			case helium.TextNode:
				if len(child.Content()) > 0 {
					return true
				}
			case helium.CommentNode, helium.ProcessingInstructionNode:
				if len(child.Content()) > 0 {
					return true
				}
			case helium.DocumentNode:
				// Document nodes can appear as children when xsl:document is
				// used inside xsl:where-populated. Recursively check if the
				// document itself is populated.
				if isPopulated(child) {
					return true
				}
			}
		}
		return false
	case helium.TextNode:
		return len(node.Content()) > 0
	case helium.CommentNode, helium.ProcessingInstructionNode:
		return len(node.Content()) > 0
	case helium.AttributeNode:
		// An attribute is populated if its value is non-empty.
		return len(node.Content()) > 0
	default:
		return false
	}
}

// isItemSequencePopulated returns true if the XDM item sequence contains
// at least one "significant" item per XSLT 3.0 xsl:where-populated rules.
//
//   - A map is significant if it has at least one entry.
//   - An array is significant if at least one member (recursively) is
//     a non-empty sequence containing a significant item.
//   - An empty string ("") is not significant.
//   - Any other atomic value, node, or non-empty-string is significant.
func isItemSequencePopulated(items xpath3.Sequence) bool {
	if items == nil {
		return false
	}
	for item := range sequence.Items(items) {
		if isItemSignificant(item) {
			return true
		}
	}
	return false
}

func isItemSignificant(item xpath3.Item) bool {
	switch v := item.(type) {
	case xpath3.MapItem:
		return v.Size() > 0
	case xpath3.ArrayItem:
		for i := 1; i <= v.Size(); i++ {
			member, _ := v.Get(i)
			if isItemSequencePopulated(member) {
				return true
			}
		}
		return false
	case xpath3.AtomicValue:
		s, _ := xpath3.AtomicToString(v)
		return s != ""
	case xpath3.NodeItem:
		return isPopulated(v.Node)
	default:
		return true
	}
}

func (ec *execContext) evaluateConditionalInstruction(ctx context.Context, selectExpr *xpath3.Expression, body []instruction) (xpath3.Sequence, error) {
	if selectExpr != nil {
		result, err := ec.evalXPath(selectExpr, ec.contextNode)
		if err != nil {
			return nil, err
		}
		return result.Sequence(), nil
	}
	return ec.evaluateBodyAsSequence(ctx, body)
}

func (ec *execContext) execOnEmpty(ctx context.Context, inst *onEmptyInst) error {
	out := ec.currentOutput()
	if len(out.conditionalScopes) == 0 || out.current == nil {
		return nil
	}
	content, err := ec.evaluateConditionalInstruction(ctx, inst.Select, inst.Body)
	if err != nil {
		return err
	}
	placeholder := out.doc.CreateComment(nil)
	if err := ec.addNodeUntracked(placeholder); err != nil {
		return err
	}
	scopeIdx := len(out.conditionalScopes) - 1
	out.conditionalScopes[scopeIdx].actions = append(out.conditionalScopes[scopeIdx].actions, conditionalAction{
		ctx:           ctx,
		kind:          conditionalOnEmpty,
		content:       content,
		placeholder:   placeholder,
		prevWasAtomic: out.prevWasAtomic,
	})
	return nil
}

func (ec *execContext) execOnNonEmpty(ctx context.Context, inst *onNonEmptyInst) error {
	out := ec.currentOutput()
	if len(out.conditionalScopes) == 0 || out.current == nil {
		return nil
	}
	content, err := ec.evaluateConditionalInstruction(ctx, inst.Select, inst.Body)
	if err != nil {
		return err
	}
	placeholder := out.doc.CreateComment(nil)
	if err := ec.addNodeUntracked(placeholder); err != nil {
		return err
	}
	scopeIdx := len(out.conditionalScopes) - 1
	out.conditionalScopes[scopeIdx].actions = append(out.conditionalScopes[scopeIdx].actions, conditionalAction{
		ctx:           ctx,
		kind:          conditionalOnNonEmpty,
		content:       content,
		placeholder:   placeholder,
		prevWasAtomic: out.prevWasAtomic,
	})
	return nil
}

// execMap executes an xsl:map instruction, producing a MapItem from child
// xsl:map-entry instructions.
func (ec *execContext) execMap(ctx context.Context, inst *mapInst) error {
	tmpDoc := helium.NewDefaultDocument()
	tmpRoot := tmpDoc.CreateElement("_tmp")
	if err := tmpDoc.AddChild(tmpRoot); err != nil {
		return err
	}

	frame := &outputFrame{
		doc:            tmpDoc,
		current:        tmpRoot,
		captureItems:   true,
		sequenceMode:   true,
		mapConstructor: true,
	}
	ec.outputStack = append(ec.outputStack, frame)
	if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
		ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
		return err
	}
	ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]

	var entries []xpath3.MapEntry
	// Build entries and detect duplicates per XTDE3365.
	// Use a temporary map to check for duplicate keys efficiently.
	dupCheck := xpath3.NewMap(nil)
	for _, item := range frame.pendingItems {
		m, ok := item.(xpath3.MapItem)
		if !ok {
			return dynamicError(errCodeXTDE0450, "xsl:map body produced non-map item %T", item)
		}
		if err := m.ForEach(func(k xpath3.AtomicValue, v xpath3.Sequence) error {
			if _, exists := dupCheck.Get(k); exists {
				ks, _ := xpath3.AtomicToString(k)
				return dynamicError(errCodeXTDE3365, "duplicate key %q in xsl:map", ks)
			}
			entries = append(entries, xpath3.MapEntry{Key: k, Value: v})
			dupCheck = xpath3.NewMap(entries)
			return nil
		}); err != nil {
			return err
		}
	}

	m := xpath3.NewMap(entries)
	out := ec.currentOutput()
	if out.captureItems {
		out.pendingItems = append(out.pendingItems, m)
		out.noteOutput()
		return nil
	}
	// For json/adaptive output methods, capture items instead of XTDE0450.
	if ec.isItemOutputMethod() {
		out.pendingItems = append(out.pendingItems, m)
		out.noteOutput()
		return nil
	}
	return dynamicError(errCodeXTDE0450, "cannot add a map to the result tree")
}

// execMapEntry is a no-op when called outside xsl:map; entries are handled
// by execMap directly.
func (ec *execContext) execMapEntry(ctx context.Context, inst *mapEntryInst) error {
	out := ec.currentOutput()
	if out.captureItems && out.mapConstructor {
		keyResult, err := ec.evalXPath(inst.Key, ec.contextNode)
		if err != nil {
			return err
		}
		keySeq := keyResult.Sequence()
		if keySeq == nil || sequence.Len(keySeq) != 1 {
			return dynamicError(errCodeXPTY0004, "xsl:map-entry key must be a single atomic value")
		}
		keyAV, err := xpath3.AtomizeItem(keySeq.Get(0))
		if err != nil {
			return err
		}

		var valSeq xpath3.Sequence
		if inst.Select != nil {
			valResult, err := ec.evalXPath(inst.Select, ec.contextNode)
			if err != nil {
				return err
			}
			valSeq = valResult.Sequence()
		} else if len(inst.Body) > 0 {
			valSeq, err = ec.evaluateBodyAsSequence(ctx, inst.Body)
			if err != nil {
				return err
			}
		}

		out.pendingItems = append(out.pendingItems, xpath3.NewMap([]xpath3.MapEntry{{
			Key:   keyAV,
			Value: valSeq,
		}}))
		out.noteOutput()
		return nil
	}

	// When called standalone (outside xsl:map), produce a single-entry map.
	// Per XSLT 3.0 §11.9.4, xsl:map-entry always produces a map item.
	keyResult, err := ec.evalXPath(inst.Key, ec.contextNode)
	if err != nil {
		return err
	}
	keySeq := keyResult.Sequence()
	if keySeq == nil || sequence.Len(keySeq) != 1 {
		return dynamicError(errCodeXPTY0004, "xsl:map-entry key must be a single atomic value")
	}
	keyAV, err := xpath3.AtomizeItem(keySeq.Get(0))
	if err != nil {
		return err
	}

	var valSeq xpath3.Sequence
	if inst.Select != nil {
		valResult, err := ec.evalXPath(inst.Select, ec.contextNode)
		if err != nil {
			return err
		}
		valSeq = valResult.Sequence()
	} else if len(inst.Body) > 0 {
		valSeq, err = ec.evaluateBodyAsSequence(ctx, inst.Body)
		if err != nil {
			return err
		}
	}

	mapItem := xpath3.NewMap([]xpath3.MapEntry{{
		Key:   keyAV,
		Value: valSeq,
	}})

	if out.captureItems {
		out.pendingItems = append(out.pendingItems, mapItem)
		out.noteOutput()
		return nil
	}
	// If not capturing items, output the map as text (fallback)
	s := stringifySequenceWithSep(xpath3.ItemSlice{mapItem}, " ")
	if s != "" {
		text := ec.resultDoc.CreateText([]byte(s))
		return ec.addNode(text)
	}
	return nil
}

// execAssert implements xsl:assert.
// If the test expression evaluates to false, an error is raised with the
// specified error code (default XTMM9001).
func (ec *execContext) execAssert(ctx context.Context, inst *assertInst) error {
	if inst.Test == nil {
		return nil
	}
	result, err := ec.evalXPath(inst.Test, ec.contextNode)
	if err != nil {
		return err
	}
	b, err := xpath3.EBV(result.Sequence())
	if err != nil {
		return err
	}
	if !b {
		errCode := inst.ErrorCode
		if errCode == "" {
			errCode = errCodeXTMM9001
		}
		// Build error message from body or select
		msg := "assertion failed"
		if inst.Select != nil {
			sel, selErr := ec.evalXPath(inst.Select, ec.contextNode)
			if selErr == nil {
				msg = stringifySequence(sel.Sequence())
			}
		} else if len(inst.Body) > 0 {
			seq, bodyErr := ec.evaluateBody(ctx, inst.Body)
			if bodyErr == nil {
				msg = stringifySequence(seq)
			}
		}
		return dynamicError(errCode, "%s", msg)
	}
	return nil
}

// execEvaluate implements xsl:evaluate — dynamically compile and evaluate
// an XPath expression string at runtime.
func (ec *execContext) execEvaluate(ctx context.Context, inst *evaluateInst) error {
	// 1. Evaluate the xpath attribute expression to get the XPath string.
	xpathResult, err := ec.evalXPath(inst.XPath, ec.contextNode)
	if err != nil {
		return err
	}
	xpathStr, ok := xpathResult.IsString()
	if !ok {
		// Atomize and convert to string
		seq := xpathResult.Sequence()
		if seq == nil || sequence.Len(seq) == 0 {
			return dynamicError(errCodeXTDE3160, "xsl:evaluate: xpath attribute evaluated to empty sequence")
		}
		av, atomErr := xpath3.AtomizeItem(seq.Get(0))
		if atomErr != nil {
			return atomErr
		}
		s, sErr := xpath3.AtomicToString(av)
		if sErr != nil {
			return sErr
		}
		xpathStr = s
	}

	if strings.TrimSpace(xpathStr) == "" {
		return dynamicError(errCodeXTDE3160, "xsl:evaluate: xpath expression is empty")
	}

	// 2. Determine the context item for dynamic evaluation.
	var dynContextNode helium.Node
	var dynContextItem xpath3.Item
	hasContextItem := true
	if inst.ContextItem != nil {
		ciResult, ciErr := ec.evalXPath(inst.ContextItem, ec.contextNode)
		if ciErr != nil {
			return ciErr
		}
		ciSeq := ciResult.Sequence()
		ciLen := 0
		if ciSeq != nil {
			ciLen = sequence.Len(ciSeq)
		}
		if ciLen == 1 {
			switch v := ciSeq.Get(0).(type) {
			case xpath3.NodeItem:
				dynContextNode = v.Node
			default:
				dynContextItem = v
			}
		} else if ciLen > 1 {
			return dynamicError(errCodeXTTE3210, "xsl:evaluate: context-item must be a single item, got %d items", ciLen)
		} else {
			// Empty sequence: no context item
			hasContextItem = false
		}
	} else {
		// Per XSLT 3.0 §20.3.2: when no context-item attribute is present,
		// the context item for the dynamic expression is absent.
		hasContextItem = false
	}

	// 3. Build namespace bindings for the dynamic expression.
	nsBindings := make(map[string]string)

	// Start with stylesheet namespace bindings
	for k, v := range ec.stylesheet.namespaces {
		if k == "" {
			continue // don't inherit default namespace by default
		}
		nsBindings[k] = v
	}

	// If namespace-context is specified, collect namespaces from that node
	if inst.NamespaceContext != nil {
		ncResult, ncErr := ec.evalXPath(inst.NamespaceContext, ec.contextNode)
		if ncErr != nil {
			return ncErr
		}
		ncSeq := ncResult.Sequence()
		ncLen := 0
		if ncSeq != nil {
			ncLen = sequence.Len(ncSeq)
		}
		// XTTE3170: namespace-context must produce a single node.
		if ncLen > 1 {
			return dynamicError(errCodeXTTE3170,
				"xsl:evaluate namespace-context produced %d items; a single node is required", ncLen)
		}
		if ncLen > 0 {
			if ni, nodeOK := ncSeq.Get(0).(xpath3.NodeItem); nodeOK {
				nsNode := ni.Node
				// Walk up to find an element
				for nsNode != nil {
					if elem, elemOK := nsNode.(*helium.Element); elemOK {
						// Collect in-scope namespaces walking up
						seen := make(map[string]struct{})
						var cur helium.Node = elem
						for cur != nil {
							if e, eOK := cur.(*helium.Element); eOK {
								for _, ns := range e.Namespaces() {
									prefix := ns.Prefix()
									if _, exists := seen[prefix]; !exists {
										seen[prefix] = struct{}{}
										nsBindings[prefix] = ns.URI()
									}
								}
							}
							cur = cur.Parent()
						}
						break
					}
					nsNode = nsNode.Parent()
				}
			}
		}
	}

	// 4. Handle xpath-default-namespace for the dynamic expression.
	// Per XSLT 3.0 §3.18.2: when namespace-context is present, the
	// in-scope namespaces of that node define the namespace context for
	// the dynamic expression, including any default namespace. An explicit
	// xpath-default-namespace on xsl:evaluate overrides this. But an
	// *inherited* xpath-default-namespace from the stylesheet must NOT
	// override the namespace-context's default namespace.
	if inst.HasLocalXPathDefaultNS {
		nsBindings[""] = inst.XPathDefaultNS
	} else if inst.NamespaceContext != nil {
		// Keep whatever default namespace came from the namespace-context
		// node (which may be absent, i.e. no key "" in nsBindings).
	} else if inst.HasXPathDefaultNS {
		nsBindings[""] = inst.XPathDefaultNS
	} else if ec.hasXPathDefaultNS {
		nsBindings[""] = ec.xpathDefaultNS
	}

	// 4b. Evaluate schema-aware avt if present
	if inst.SchemaAwareAVT != nil {
		saStr, saErr := inst.SchemaAwareAVT.evaluate(ctx, ec.contextNode)
		if saErr != nil {
			return saErr
		}
		if _, ok := parseXSDBool(saStr); !ok {
			return staticError(errCodeXTSE0020, "xsl:evaluate: invalid value %q for schema-aware attribute", saStr)
		}
	}

	// 5. Compile the dynamic XPath expression.
	dynExpr, compileErr := xpath3.NewCompiler().Compile(xpathStr)
	if compileErr != nil {
		return dynamicError(errCodeXTDE3160, "xsl:evaluate: cannot compile XPath expression %q: %v", xpathStr, compileErr)
	}

	// 5a. XTDE3160: certain XSLT functions are not allowed in xsl:evaluate
	if xpath3.ExprUsesFunction(dynExpr, "current") {
		return dynamicError(errCodeXTDE3160, "xsl:evaluate: current() is not allowed in dynamically evaluated expressions")
	}
	for _, blocked := range []string{"system-property", "current-output-uri", "available-system-properties", "document"} {
		if xpath3.ExprUsesFunction(dynExpr, blocked) {
			return dynamicError(errCodeXTDE3160, "xsl:evaluate: %s() is not allowed in dynamically evaluated expressions", blocked)
		}
	}

	// 6. Build evaluation context with variables from xsl:with-param.
	dynCtx := ec.xpathContext()

	// Collect variables: start with current XSLT variables plus xsl:with-param
	vars := ec.collectAllVars()

	// Add xsl:with-param variables
	for _, wp := range inst.Params {
		paramVal, paramErr := ec.evaluateWithParam(ctx, wp)
		if paramErr != nil {
			return paramErr
		}
		vars[wp.Name] = paramVal
	}

	// Add with-params map variables (higher priority, overrides xsl:with-param)
	if inst.WithParamsExpr != nil {
		wpResult, wpErr := ec.evalXPath(inst.WithParamsExpr, ec.contextNode)
		if wpErr != nil {
			return wpErr
		}
		wpSeq := wpResult.Sequence()
		if wpSeq != nil && sequence.Len(wpSeq) == 1 {
			if wpMap, mapOK := wpSeq.Get(0).(xpath3.MapItem); mapOK {
				forEachErr := wpMap.ForEach(func(key xpath3.AtomicValue, value xpath3.Sequence) error {
					// XTTE3165: with-params map keys must be xs:QName
					if key.TypeName != xpath3.TypeQName {
						return dynamicError(errCodeXTTE3165, "xsl:evaluate: with-params map key must be xs:QName, got %s", key.TypeName)
					}
					qn := key.QNameVal()
					vars[qn.Local] = value
					return nil
				})
				if forEachErr != nil {
					return forEachErr
				}
			}
		}
	}

	// Per XSLT 3.0 section 20.3: the available functions include all
	// functions from the static context of the xsl:evaluate instruction
	// EXCEPT current(), user-defined stylesheet functions (xsl:function),
	// and functions in the XSLT namespace.
	evalFns := ec.xsltEvaluateFunctions()
	eval := xpath3.NewEvaluator(xpath3.EvalBorrowing).
		Variables(xpath3.VariablesFromMap(vars)).
		Functions(xpath3.FunctionLibraryFromMaps(evalFns, ec.xsltEvaluateFunctionsNS()))

	if len(nsBindings) > 0 {
		eval = eval.Namespaces(nsBindings)
	}

	if ec.typeAnnotations != nil {
		eval = eval.TypeAnnotations(ec.typeAnnotations)
	}
	if ec.preservedIDAnnotations != nil {
		eval = eval.PreservedIDAnnotations(ec.preservedIDAnnotations)
	}
	if ec.schemaRegistry != nil {
		eval = eval.SchemaDeclarations(ec.schemaRegistry)
	}

	// Handle base-uri
	if inst.BaseURI != nil {
		baseURI, buErr := inst.BaseURI.evaluate(dynCtx, ec.contextNode)
		if buErr != nil {
			return buErr
		}
		if baseURI != "" {
			eval = eval.BaseURI(ensureFileURI(baseURI))
		}
	} else if ec.stylesheet.baseURI != "" {
		eval = eval.BaseURI(ensureFileURI(ec.stylesheet.baseURI))
	}

	// Default collation
	if ec.defaultCollation != "" {
		eval = eval.DefaultCollation(ec.defaultCollation)
	}

	// Decimal formats (package-scoped)
	dfmts := ec.effectiveDecimalFormats()
	if len(dfmts) > 0 {
		for qn, df := range dfmts {
			if qn == (xpath3.QualifiedName{}) {
				eval = eval.DefaultDecimalFormat(df)
			}
		}
		eval = eval.NamedDecimalFormats(dfmts)
	}

	// Set context item if it's an atomic value
	if dynContextItem != nil {
		eval = eval.ContextItem(dynContextItem)
	}

	if hasContextItem {
		eval = eval.Position(1).Size(1)
	}

	// 7. Evaluate the dynamic expression.
	var evalNode helium.Node
	if dynContextNode != nil {
		evalNode = dynContextNode
	} else if hasContextItem && ec.contextNode != nil {
		evalNode = ec.contextNode
	}

	result, evalErr := eval.Evaluate(dynCtx, dynExpr, evalNode)
	if evalErr != nil {
		return evalErr
	}

	// 8. Check as type constraint (XPTY0004).
	seq := result.Sequence()
	if inst.As != "" {
		if typeErr := ec.checkEvaluateAsType(inst.As, seq); typeErr != nil {
			return typeErr
		}
	}

	// 9. Output the result sequence.
	return ec.outputSequence(seq)
}

// checkEvaluateAsType checks the xsl:evaluate as= type constraint.
// Returns XPTY0004 if the result does not match the expected type.
// Per XSLT 3.0, the result is coerced to the target type.
// For now, only check obvious mismatches.
func (ec *execContext) checkEvaluateAsType(asType string, seq xpath3.Sequence) error {
	switch asType {
	case "xs:string":
		// xs:string: nodes atomize to xs:untypedAtomic which coerces to string.
		// xs:untypedAtomic also coerces. Other atomic types do NOT coerce.
		for item := range sequence.Items(seq) {
			if _, ok := item.(xpath3.NodeItem); ok {
				continue // nodes atomize to xs:untypedAtomic → coerces to string
			}
			if av, ok := item.(xpath3.AtomicValue); ok {
				switch av.TypeName {
				case xpath3.TypeString, xpath3.TypeUntypedAtomic, xpath3.TypeAnyURI:
					continue
				}
			}
			return dynamicError(errCodeXPTY0004, "xsl:evaluate: result does not match as=%q", asType)
		}
	case "xs:integer":
		for item := range sequence.Items(seq) {
			if av, ok := item.(xpath3.AtomicValue); ok {
				switch av.TypeName {
				case xpath3.TypeInteger, xpath3.TypeUntypedAtomic:
					continue
				}
			}
			return dynamicError(errCodeXPTY0004, "xsl:evaluate: result does not match as=%q", asType)
		}
	case "xs:boolean":
		for item := range sequence.Items(seq) {
			if av, ok := item.(xpath3.AtomicValue); ok {
				switch av.TypeName {
				case xpath3.TypeBoolean, xpath3.TypeUntypedAtomic:
					continue
				}
			}
			return dynamicError(errCodeXPTY0004, "xsl:evaluate: result does not match as=%q", asType)
		}
	case "item()", "item()*":
		// Any item matches
	}
	return nil
}
