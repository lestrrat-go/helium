package xslt3

import (
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// compileSourceDocument compiles an xsl:source-document element.
func (c *compiler) compileSourceDocument(elem *helium.Element) (Instruction, error) {
	hrefAttr := getAttr(elem, "href")
	if hrefAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:source-document requires href attribute")
	}

	hrefAVT, err := compileAVT(hrefAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &SourceDocumentInst{
		Href:       hrefAVT,
		Streamable: xsdBoolTrue(getAttr(elem, "streamable")),
		BaseURI:    stylesheetBaseURI(elem, c.baseURI),
		Validation: getAttr(elem, "validation"),
	}
	if typeName := getAttr(elem, "type"); typeName != "" {
		inst.TypeName = resolveQName(typeName, c.nsBindings)
	}
	if useAccumulators := getAttr(elem, "use-accumulators"); useAccumulators != "" {
		for _, name := range strings.Fields(useAccumulators) {
			inst.UseAccumulators = append(inst.UseAccumulators, resolveQName(name, c.nsBindings))
		}
	}

	body, err := c.compileChildren(elem)
	if err != nil {
		return nil, err
	}
	inst.Body = body

	return inst, nil
}

// compileIterate compiles an xsl:iterate element.
func (c *compiler) compileIterate(elem *helium.Element) (Instruction, error) {
	selectAttr := getAttr(elem, "select")
	if selectAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:iterate requires select attribute")
	}

	expr, err := compileXPath(selectAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &IterateInst{Select: expr}

	// Collect leading xsl:param children, then xsl:on-completion, then body.
	// Validate ordering: params first, then optional on-completion, then body.
	inParams := true
	seenOnCompletion := false
	paramNames := make(map[string]struct{})

	// Save/restore iterate context.
	savedIterDepth := c.iterateDepth
	savedBreakAllowed := c.breakAllowed
	c.iterateDepth++
	c.breakAllowed = true
	defer func() {
		c.iterateDepth = savedIterDepth
		c.breakAllowed = savedBreakAllowed
	}()

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			// Text nodes in the body (after params)
			if text, ok := child.(*helium.Text); ok {
				t := string(text.Content())
				if !c.shouldStripText(t) {
					inParams = false
					inst.Body = append(inst.Body, &LiteralTextInst{Value: t})
				}
			}
			continue
		}

		if childElem.URI() == NSXSLT {
			switch childElem.LocalName() {
			case "param":
				if !inParams {
					// XTSE0010: xsl:param must come before other content
					return nil, staticError(errCodeXTSE0010, "xsl:param in xsl:iterate must precede all other children")
				}
				p, err := c.compileIterateParam(childElem)
				if err != nil {
					return nil, err
				}
				// Check for duplicate param names (XTSE0580).
				if _, dup := paramNames[p.Name]; dup {
					return nil, staticError(errCodeXTSE0580_, "duplicate parameter name %q in xsl:iterate", p.Name)
				}
				paramNames[p.Name] = struct{}{}
				inst.Params = append(inst.Params, p)
				continue
			case "on-completion":
				inParams = false
				if seenOnCompletion {
					return nil, staticError(errCodeXTSE0010, "xsl:iterate must have at most one xsl:on-completion child")
				}
				seenOnCompletion = true
				// XTSE3125: on-completion must not have both select and body.
				selAttr := getAttr(childElem, "select")
				hasBody := false
				for ch := childElem.FirstChild(); ch != nil; ch = ch.NextSibling() {
					if chElem, ok := ch.(*helium.Element); ok {
						if chElem.URI() == NSXSLT && chElem.LocalName() == "fallback" {
							continue
						}
						hasBody = true
						break
					}
					if txt, ok := ch.(*helium.Text); ok {
						t := string(txt.Content())
						if !c.shouldStripText(t) {
							hasBody = true
							break
						}
					}
				}
				if selAttr != "" && hasBody {
					return nil, staticError(errCodeXTSE3125, "xsl:on-completion must not have both select attribute and sequence constructor")
				}
				if selAttr != "" {
					selExpr, selErr := compileXPath(selAttr, c.nsBindings)
					if selErr != nil {
						return nil, selErr
					}
					inst.OnCompletion = []Instruction{&XSLSequenceInst{Select: selExpr}}
				} else {
					body, err := c.compileChildren(childElem)
					if err != nil {
						return nil, err
					}
					inst.OnCompletion = body
				}
				continue
			}
		}

		inParams = false
		childInst, err := c.compileInstruction(childElem)
		if err != nil {
			return nil, err
		}
		if childInst != nil {
			inst.Body = append(inst.Body, childInst)
		}
	}

	// XTSE3130: Validate that xsl:next-iteration with-params only reference
	// declared xsl:iterate parameters.
	if err := validateNextIterationParams(inst.Body, paramNames); err != nil {
		return nil, err
	}

	// XTSE3520: xsl:iterate param without select/default when type doesn't allow empty.
	for _, p := range inst.Params {
		if p.Select == nil && len(p.Body) == 0 && p.As != "" {
			// The default value is the empty sequence. If the type is not
			// optional (doesn't end with ? or *), it requires a value.
			as := p.As
			if !strings.HasSuffix(as, "?") && !strings.HasSuffix(as, "*") {
				return nil, staticError(errCodeXTSE3520, "xsl:iterate parameter %q with type %q requires a select attribute or default value", p.Name, as)
			}
		}
	}

	return inst, nil
}

// validateNextIterationParams walks the instruction tree checking that any
// NextIterationInst only references declared iterate parameter names.
func validateNextIterationParams(body []Instruction, paramNames map[string]struct{}) error {
	for _, inst := range body {
		if ni, ok := inst.(*NextIterationInst); ok {
			for _, wp := range ni.Params {
				if _, exists := paramNames[wp.Name]; !exists {
					return staticError(errCodeXTSE3130, "xsl:next-iteration references undeclared parameter %q", wp.Name)
				}
			}
		}
		// Recurse into sub-instructions.
		for _, child := range instructionChildren(inst) {
			if err := validateNextIterationParams(child, paramNames); err != nil {
				return err
			}
		}
	}
	return nil
}

// instructionChildren returns the sub-instruction slices of an instruction
// for recursive walking.
func instructionChildren(inst Instruction) [][]Instruction {
	switch v := inst.(type) {
	case *IfInst:
		return [][]Instruction{v.Body}
	case *ChooseInst:
		var result [][]Instruction
		for _, w := range v.When {
			result = append(result, w.Body)
		}
		if v.Otherwise != nil {
			result = append(result, v.Otherwise)
		}
		return result
	case *ForEachInst:
		return [][]Instruction{v.Body}
	case *TryCatchInst:
		result := [][]Instruction{v.Try}
		for _, c := range v.Catches {
			result = append(result, c.Body)
		}
		return result
	case *SequenceInst:
		return [][]Instruction{v.Body}
	case *VariableInst:
		return [][]Instruction{v.Body}
	case *LiteralResultElement:
		return [][]Instruction{v.Body}
	case *ElementInst:
		return [][]Instruction{v.Body}
	case *CopyInst:
		return [][]Instruction{v.Body}
	case *CopyOfInst:
		return nil
	default:
		return nil
	}
}

// compileIterateParam compiles an xsl:param inside xsl:iterate.
func (c *compiler) compileIterateParam(elem *helium.Element) (*IterateParam, error) {
	name := getAttr(elem, "name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:param requires name attribute")
	}

	p := &IterateParam{
		Name: resolveQName(name, c.nsBindings),
		As:   getAttr(elem, "as"),
	}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		p.Select = expr
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		p.Body = body
	}

	return p, nil
}

// compileFork compiles an xsl:fork element.
func (c *compiler) compileFork(elem *helium.Element) (Instruction, error) {
	inst := &ForkInst{}

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}

		// Each child of xsl:fork is a branch (typically xsl:sequence or
		// xsl:for-each-group, but any instruction is valid per spec).
		branch, err := c.compileForkBranch(childElem)
		if err != nil {
			return nil, err
		}
		if branch != nil {
			inst.Branches = append(inst.Branches, branch)
		}
	}

	return inst, nil
}

// compileForkBranch compiles one child of xsl:fork into a branch (slice of instructions).
func (c *compiler) compileForkBranch(elem *helium.Element) ([]Instruction, error) {
	// If the child is xsl:sequence, compile its children as the branch body.
	if elem.URI() == NSXSLT && elem.LocalName() == "sequence" {
		selectAttr := getAttr(elem, "select")
		if selectAttr != "" {
			expr, err := compileXPath(selectAttr, c.nsBindings)
			if err != nil {
				return nil, err
			}
			return []Instruction{&XSLSequenceInst{Select: expr}}, nil
		}
		return c.compileChildren(elem)
	}

	// Otherwise, compile the element as a single-instruction branch.
	inst, err := c.compileInstruction(elem)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, nil
	}
	return []Instruction{inst}, nil
}

// compileBreak compiles an xsl:break element.
func (c *compiler) compileBreak(elem *helium.Element) (Instruction, error) {
	// XTSE0010: xsl:break must be lexically within xsl:iterate.
	if c.iterateDepth == 0 {
		return nil, staticError(errCodeXTSE0010, "xsl:break must be within xsl:iterate")
	}
	// XTSE3120: xsl:break must not be inside an element constructor,
	// xsl:for-each, or other non-iterate-body position.
	if !c.breakAllowed {
		return nil, staticError(errCodeXTSE3120, "xsl:break is not allowed in this position within xsl:iterate")
	}

	// XTSE3125: xsl:break must not have both select and body.
	selectAttr := getAttr(elem, "select")
	hasBody := false
	for ch := elem.FirstChild(); ch != nil; ch = ch.NextSibling() {
		if chElem, ok := ch.(*helium.Element); ok {
			if chElem.URI() == NSXSLT && chElem.LocalName() == "fallback" {
				continue
			}
			hasBody = true
			break
		}
		if txt, ok := ch.(*helium.Text); ok {
			t := string(txt.Content())
			if !c.shouldStripText(t) {
				hasBody = true
				break
			}
		}
	}
	if selectAttr != "" && hasBody {
		return nil, staticError(errCodeXTSE3125, "xsl:break must not have both select attribute and sequence constructor")
	}

	inst := &BreakInst{}

	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		inst.Body = body
	}

	return inst, nil
}

// compileNextIteration compiles an xsl:next-iteration element.
func (c *compiler) compileNextIteration(elem *helium.Element) (Instruction, error) {
	// XTSE0010: xsl:next-iteration must be lexically within xsl:iterate.
	if c.iterateDepth == 0 {
		return nil, staticError(errCodeXTSE0010, "xsl:next-iteration must be within xsl:iterate")
	}
	// XTSE3120: xsl:next-iteration must not be inside an element constructor,
	// xsl:for-each, or other non-iterate-body position.
	if !c.breakAllowed {
		return nil, staticError(errCodeXTSE3120, "xsl:next-iteration is not allowed in this position within xsl:iterate")
	}

	inst := &NextIterationInst{}

	// Check for duplicate with-param names (XTSE0670).
	wpNames := make(map[string]struct{})
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() == NSXSLT && childElem.LocalName() == "with-param" {
			wp, err := c.compileWithParam(childElem)
			if err != nil {
				return nil, err
			}
			if wp == nil {
				continue
			}
			if _, dup := wpNames[wp.Name]; dup {
				return nil, staticError(errCodeXTSE0670, "duplicate xsl:with-param name %q in xsl:next-iteration", wp.Name)
			}
			wpNames[wp.Name] = struct{}{}
			inst.Params = append(inst.Params, wp)
		}
	}

	return inst, nil
}

// compileAccumulator compiles an xsl:accumulator top-level element.
func (c *compiler) compileAccumulator(elem *helium.Element) error {
	name := getAttr(elem, "name")
	if name == "" {
		return staticError(errCodeXTSE0110, "xsl:accumulator requires name attribute")
	}

	expandedName := resolveQName(name, c.nsBindings)

	acc := &AccumulatorDef{
		Name:       expandedName,
		As:         getAttr(elem, "as"),
		Streamable: xsdBoolTrue(getAttr(elem, "streamable")),
		ImportPrec: c.importPrec,
	}

	// Read initial-value attribute (required per XSLT 3.0 §10.4)
	iv := getAttr(elem, "initial-value")
	if iv == "" {
		return staticError(errCodeXTSE0010, "xsl:accumulator %q requires initial-value attribute", expandedName)
	}
	initialExpr, err := compileXPath(iv, c.nsBindings)
	if err != nil {
		return err
	}
	acc.Initial = initialExpr

	// Scan children for accumulator-rule elements
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() != NSXSLT {
			continue
		}
		if childElem.LocalName() == "accumulator-rule" {
			if err := c.compileAccumulatorRule(acc, childElem); err != nil {
				return err
			}
		}
	}

	// XTSE0010: an accumulator must have at least one rule
	if len(acc.Rules) == 0 {
		return staticError(errCodeXTSE0010, "xsl:accumulator %q must have at least one xsl:accumulator-rule", expandedName)
	}

	if existing, exists := c.stylesheet.accumulators[expandedName]; exists {
		// Accumulators in different packages may share the same name.
		// A local accumulator shadows a package-imported one.
		if existing.FromPackage {
			c.stylesheet.accumulators[expandedName] = acc
			return nil
		}
		// XTSE3350: duplicate accumulator name at the same import precedence.
		// Defer the error — a higher-precedence accumulator may resolve the
		// conflict later (e.g. imported file has duplicates but the importer
		// provides its own override).
		if acc.ImportPrec == existing.ImportPrec {
			existing.conflictDuplicate = true
			return nil
		}
		// Different import precedence: highest wins, lower is silently discarded.
		// A higher-precedence declaration also clears any deferred duplicate
		// conflict from a lower level.
		if acc.ImportPrec > existing.ImportPrec {
			c.stylesheet.accumulators[expandedName] = acc
		}
		return nil
	}
	c.stylesheet.accumulatorOrder = append(c.stylesheet.accumulatorOrder, expandedName)
	c.stylesheet.accumulators[expandedName] = acc
	return nil
}

// compileAccumulatorRule compiles an xsl:accumulator-rule element.
func (c *compiler) compileAccumulatorRule(parent *AccumulatorDef, elem *helium.Element) error {
	matchAttr := getAttr(elem, "match")
	if matchAttr == "" {
		return staticError(errCodeXTSE0110, "xsl:accumulator-rule requires match attribute")
	}

	matchPat, err := compilePattern(matchAttr, c.nsBindings, c.xpathDefaultNS)
	if err != nil {
		return err
	}

	rule := &AccumulatorRule{
		Match: matchPat,
		Phase: getAttr(elem, "phase"),
		New:   getAttr(elem, "new-value") == "yes",
	}

	// Default phase is "start" per XSLT 3.0 spec §14
	if rule.Phase == "" {
		rule.Phase = "start"
	}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return err
		}
		rule.Select = expr
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return err
		}
		rule.Body = body
	}

	parent.Rules = append(parent.Rules, rule)
	return nil
}

// compileMerge compiles an xsl:merge element.
func (c *compiler) compileMerge(elem *helium.Element) (Instruction, error) {
	inst := &MergeInst{}

	actionCount := 0
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() != NSXSLT {
			continue
		}
		switch childElem.LocalName() {
		case "merge-source":
			src, err := c.compileMergeSource(childElem)
			if err != nil {
				return nil, err
			}
			inst.Sources = append(inst.Sources, src)
		case "merge-action":
			actionCount++
			if actionCount > 1 {
				return nil, staticError(errCodeXTSE0010, "xsl:merge must have at most one xsl:merge-action child")
			}
			body, err := c.compileChildren(childElem)
			if err != nil {
				return nil, err
			}
			inst.Action = body
		case "fallback":
			// xsl:fallback is allowed only after xsl:merge-action (XTSE0010).
			if actionCount == 0 {
				return nil, staticError(errCodeXTSE0010, "xsl:fallback in xsl:merge must follow xsl:merge-action")
			}
			continue
		default:
			return nil, staticError(errCodeXTSE0010, "unexpected child element xsl:%s in xsl:merge", childElem.LocalName())
		}
	}

	if len(inst.Sources) == 0 {
		return nil, staticError(errCodeXTSE0110, "xsl:merge requires at least one xsl:merge-source child")
	}
	if len(inst.Action) == 0 {
		return nil, staticError(errCodeXTSE0110, "xsl:merge requires an xsl:merge-action child")
	}

	// XTSE2200: all merge-sources must have the same number of merge-keys.
	if len(inst.Sources) > 1 {
		keyCount := len(inst.Sources[0].Keys)
		for i := 1; i < len(inst.Sources); i++ {
			if len(inst.Sources[i].Keys) != keyCount {
				return nil, staticError(errCodeXTSE2200, "all xsl:merge-source children must have the same number of xsl:merge-key children")
			}
		}
	}

	// XTSE0020: merge-source name must be a valid xs:QName.
	for _, src := range inst.Sources {
		if src.Name != "" && !isValidQName(src.Name) {
			return nil, staticError(errCodeXTSE0020, "xsl:merge-source @name %q is not a valid xs:QName", src.Name)
		}
	}

	// Streamability checks when any merge-source has streamable="yes".
	if err := mergeStreamCheck(inst); err != nil {
		return nil, err
	}

	return inst, nil
}

// compileMergeSource compiles an xsl:merge-source element.
func (c *compiler) compileMergeSource(elem *helium.Element) (*MergeSource, error) {
	src := &MergeSource{
		Name:    getAttr(elem, "name"),
		BaseURI: stylesheetBaseURI(elem, c.baseURI),
	}

	// Parse streamable attribute — must be a valid xs:boolean.
	if streamRaw := getAttr(elem, "streamable"); streamRaw != "" {
		streamVal := strings.TrimSpace(streamRaw)
		if v, ok := parseXSDBool(streamVal); ok {
			src.StreamableAttr = v
		} else {
			return nil, staticError(errCodeXTSE0020, "%q is not a valid value for xsl:merge-source/@streamable", streamVal)
		}
	}

	if sortAttr, hasSBM := elem.GetAttribute("sort-before-merge"); hasSBM {
		sortVal := strings.TrimSpace(sortAttr)
		if v, ok := parseXSDBool(sortVal); ok {
			src.SortBeforeMerge = v
		} else {
			return nil, staticError(errCodeXTSE0020, "%q is not a valid value for xsl:merge-source/@sort-before-merge", sortVal)
		}
	}

	// for-each-source: XPath expression evaluating to sequence of URI strings
	if fes := getAttr(elem, "for-each-source"); fes != "" {
		expr, err := compileXPath(fes, c.nsBindings)
		if err != nil {
			return nil, err
		}
		src.ForEachSource = expr
	}

	// for-each-item: XPath expression evaluating to sequence of items (nodes)
	if fei := getAttr(elem, "for-each-item"); fei != "" {
		expr, err := compileXPath(fei, c.nsBindings)
		if err != nil {
			return nil, err
		}
		src.ForEachItem = expr
	}

	// select: XPath expression for selecting items from each source
	if sel := getAttr(elem, "select"); sel != "" {
		expr, err := compileXPath(sel, c.nsBindings)
		if err != nil {
			return nil, err
		}
		src.Select = expr
	}

	if useAccumulators := getAttr(elem, "use-accumulators"); useAccumulators != "" {
		for _, name := range strings.Fields(useAccumulators) {
			src.UseAccumulators = append(src.UseAccumulators, resolveQName(name, c.nsBindings))
		}
	}

	// Parse xsl:merge-key children — only xsl:merge-key is allowed
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() == NSXSLT && childElem.LocalName() == "merge-key" {
			mk, err := c.compileMergeKey(childElem)
			if err != nil {
				return nil, err
			}
			src.Keys = append(src.Keys, mk)
		} else {
			// Any non-merge-key child is an error
			return nil, staticError(errCodeXTSE0010, "unexpected child element in xsl:merge-source: %s", childElem.Name())
		}
	}

	// XTSE0010: merge-source must have at least a select, for-each-source,
	// or for-each-item attribute.
	if src.Select == nil && src.ForEachSource == nil && src.ForEachItem == nil {
		return nil, staticError(errCodeXTSE0010, "xsl:merge-source requires at least one of select, for-each-source, or for-each-item attributes")
	}

	// XTSE3195: for-each-source and for-each-item are mutually exclusive.
	if src.ForEachSource != nil && src.ForEachItem != nil {
		return nil, staticError(errCodeXTSE3195, "xsl:merge-source must not have both for-each-source and for-each-item attributes")
	}

	// XTSE3430: sort-before-merge="yes" is incompatible with streamable="yes".
	if src.SortBeforeMerge && src.StreamableAttr {
		return nil, staticError(errCodeXTSE3430, "xsl:merge-source must not have both sort-before-merge='yes' and streamable='yes'")
	}
	return src, nil
}

// compileMergeKey compiles an xsl:merge-key element.
func (c *compiler) compileMergeKey(elem *helium.Element) (*MergeKey, error) {
	// XTSE0090: validate permitted attributes on xsl:merge-key.
	for _, attr := range elem.Attributes() {
		if attr.URI() != "" {
			continue // namespace-qualified attributes are allowed
		}
		switch attr.LocalName() {
		case "select", "order", "collation", "data-type", "lang", "case-order":
			// permitted
		default:
			return nil, staticError(errCodeXTSE0090, "attribute %q is not permitted on xsl:merge-key", attr.LocalName())
		}
	}

	mk := &MergeKey{
		Order: "ascending", // default
	}

	// Track collation-related attributes for sort verification.
	// When a non-codepoint collation or locale-specific sorting is used,
	// we can't reliably verify sort order so we skip the check.
	collation := getAttr(elem, "collation")
	codepointCollation := "http://www.w3.org/2005/xpath-functions/collation/codepoint"
	mk.Collation = collation
	mk.Lang = getAttr(elem, "lang")
	mk.CaseOrder = getAttr(elem, "case-order")
	if mk.Lang != "" || mk.CaseOrder != "" {
		mk.HasCollation = true
	} else if collation != "" && collation != codepointCollation {
		mk.HasCollation = true
	}

	selAttr := getAttr(elem, "select")

	// XTSE3200: merge-key must not have both select and body.
	if selAttr != "" {
		for ch := elem.FirstChild(); ch != nil; ch = ch.NextSibling() {
			if chElem, ok := ch.(*helium.Element); ok {
				if chElem.URI() == NSXSLT && chElem.LocalName() == "fallback" {
					continue
				}
				return nil, staticError(errCodeXTSE3200, "xsl:merge-key must not have both select attribute and sequence constructor")
			}
			if txt, ok := ch.(*helium.Text); ok {
				t := string(txt.Content())
				if !c.shouldStripText(t) {
					return nil, staticError(errCodeXTSE3200, "xsl:merge-key must not have both select attribute and sequence constructor")
				}
			}
		}
	}

	if selAttr != "" {
		expr, err := compileXPath(selAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		mk.Select = expr
	}

	if order := getAttr(elem, "order"); order != "" {
		if strings.Contains(order, "{") {
			avt, err := compileAVT(order, c.nsBindings)
			if err != nil {
				return nil, err
			}
			mk.OrderAVT = avt
		} else {
			mk.Order = order
		}
	}

	if dataType := getAttr(elem, "data-type"); dataType != "" {
		if strings.Contains(dataType, "{") {
			avt, err := compileAVT(dataType, c.nsBindings)
			if err != nil {
				return nil, err
			}
			mk.DataTypeAVT = avt
		} else {
			mk.DataType = dataType
		}
	}

	return mk, nil
}

func mergeStreamCheck(inst *MergeInst) error {
	hasStreamable := false
	for _, src := range inst.Sources {
		if src.StreamableAttr {
			hasStreamable = true
			break
		}
	}
	if !hasStreamable {
		return nil
	}
	// XTSE3470: merge-key order/data-type must not be AVTs within
	// a streamable merge-source.
	for _, src := range inst.Sources {
		if !src.StreamableAttr {
			continue
		}
		for _, mk := range src.Keys {
			if mk.OrderAVT != nil {
				return staticError("XTSE3470", "xsl:merge-key @order must not be an AVT when xsl:merge-source is streamable")
			}
			if mk.DataTypeAVT != nil {
				return staticError("XTSE3470", "xsl:merge-key @data-type must not be an AVT when xsl:merge-source is streamable")
			}
		}
	}
	// XTSE3430: when a streamable merge-source select uses descendant-or-self
	// axis (//), the merge-key must not navigate upward.
	for _, src := range inst.Sources {
		if !src.StreamableAttr || src.Select == nil {
			continue
		}
		if !xpath3.ExprUsesDescendantOrSelf(src.Select) {
			continue
		}
		for _, mk := range src.Keys {
			if xpath3.ExprUsesUpwardAxis(mk.Select) {
				return staticError(errCodeXTSE3430, "xsl:merge-key select is not streamable: upward axis with descendant-or-self selection")
			}
		}
	}
	return nil
}
