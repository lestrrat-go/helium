package xslt3

import (
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
)

// compileSourceDocument compiles an xsl:source-document element.
func (c *compiler) compileSourceDocument(elem *helium.Element) (instruction, error) {
	hrefAttr := getAttr(elem, "href")
	if hrefAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:source-document requires href attribute")
	}

	hrefAVT, err := compileAVT(hrefAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &sourceDocumentInst{
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
func (c *compiler) compileIterate(elem *helium.Element) (instruction, error) {
	selectAttr := getAttr(elem, "select")
	if selectAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:iterate requires select attribute")
	}

	expr, err := compileXPath(selectAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &iterateInst{Select: expr}

	// Collect leading xsl:param children, then xsl:on-completion, then body.
	// Validate ordering: params first, then optional on-completion, then body.
	inParams := true
	seenOnCompletion := false
	paramNames := make(map[string]struct{})

	// Save/restore iterate context.
	savedIterDepth := c.iterateDepth
	savedBreakAllowed := c.breakAllowed
	c.iterateDepth++
	// breakAllowed is set per-instruction below (only the tail instruction
	// is allowed to contain xsl:break / xsl:next-iteration).
	c.breakAllowed = false
	defer func() {
		c.iterateDepth = savedIterDepth
		c.breakAllowed = savedBreakAllowed
	}()

	// Collect body children (non-param, non-on-completion) to determine which
	// is the last one (tail position for XTSE3120 break/next-iteration check).
	type bodyChild struct {
		elem *helium.Element // nil for text nodes
		text string          // non-empty for text nodes
	}
	var bodyChildren []bodyChild

	for child := range helium.Children(elem) {
		childElem, ok := child.(*helium.Element)
		if !ok {
			// Text nodes in the body (after params)
			if text, ok := child.(*helium.Text); ok {
				t := string(text.Content())
				if !c.shouldStripText(t) {
					inParams = false
					bodyChildren = append(bodyChildren, bodyChild{text: t})
				}
			}
			continue
		}

		if childElem.URI() == lexicon.NamespaceXSLT {
			switch childElem.LocalName() {
			case lexicon.XSLTElementParam:
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
					return nil, staticError(errCodeXTSE0580, "duplicate parameter name %q in xsl:iterate", p.Name)
				}
				paramNames[p.Name] = struct{}{}
				inst.Params = append(inst.Params, p)
				continue
			case lexicon.XSLTElementOnCompletion:
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
						if chElem.URI() == lexicon.NamespaceXSLT && chElem.LocalName() == lexicon.XSLTElementFallback {
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
					inst.OnCompletion = []instruction{&xslSequenceInst{Select: selExpr}}
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
		bodyChildren = append(bodyChildren, bodyChild{elem: childElem})
	}

	// Compile body children. Only the LAST significant body child is in
	// "tail position" where xsl:break and xsl:next-iteration are allowed
	// (XTSE3120). xsl:fallback compiles to nil so it does not count.
	lastSignificant := -1
	for i := len(bodyChildren) - 1; i >= 0; i-- {
		bc := bodyChildren[i]
		if bc.elem != nil && bc.elem.URI() == lexicon.NamespaceXSLT && bc.elem.LocalName() == lexicon.XSLTElementFallback {
			continue
		}
		lastSignificant = i
		break
	}
	for i, bc := range bodyChildren {
		if bc.elem == nil {
			// Text node
			lit := &literalTextInst{Value: bc.text}
			if c.expandText && strings.ContainsAny(bc.text, "{}") {
				avt, err := compileAVT(bc.text, c.nsBindings)
				if err != nil {
					return nil, err
				}
				lit.TVT = avt
			}
			inst.Body = append(inst.Body, lit)
			continue
		}
		c.breakAllowed = (i == lastSignificant)
		childInst, err := c.compileInstruction(bc.elem)
		if err != nil {
			return nil, err
		}
		if childInst != nil {
			inst.Body = append(inst.Body, childInst)
		}
	}
	c.breakAllowed = false

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
// nextIterationInst only references declared iterate parameter names.
func validateNextIterationParams(body []instruction, paramNames map[string]struct{}) error {
	for _, inst := range body {
		if ni, ok := inst.(*nextIterationInst); ok {
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
func instructionChildren(inst instruction) [][]instruction {
	switch v := inst.(type) {
	case *ifInst:
		return [][]instruction{v.Body}
	case *chooseInst:
		var result [][]instruction
		for _, w := range v.When {
			result = append(result, w.Body)
		}
		if v.Otherwise != nil {
			result = append(result, v.Otherwise)
		}
		return result
	case *forEachInst:
		return [][]instruction{v.Body}
	case *tryCatchInst:
		result := [][]instruction{v.Try}
		for _, c := range v.Catches {
			result = append(result, c.Body)
		}
		return result
	case *sequenceInst:
		return [][]instruction{v.Body}
	case *variableInst:
		return [][]instruction{v.Body}
	case *literalResultElement:
		return [][]instruction{v.Body}
	case *elementInst:
		return [][]instruction{v.Body}
	case *copyInst:
		return [][]instruction{v.Body}
	case *copyOfInst:
		return nil
	case *collationScopeInst:
		if v.Inner != nil {
			return [][]instruction{{v.Inner}}
		}
		return nil
	default:
		return nil
	}
}

// compileIterateParam compiles an xsl:param inside xsl:iterate.
func (c *compiler) compileIterateParam(elem *helium.Element) (*iterateParam, error) {
	name := getAttr(elem, "name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:param requires name attribute")
	}

	p := &iterateParam{
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
func (c *compiler) compileFork(elem *helium.Element) (instruction, error) {
	inst := &forkInst{}

	for child := range helium.Children(elem) {
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
func (c *compiler) compileForkBranch(elem *helium.Element) ([]instruction, error) {
	// If the child is xsl:sequence, compile its children as the branch body.
	if elem.URI() == lexicon.NamespaceXSLT && elem.LocalName() == lexicon.XSLTElementSequence {
		selectAttr := getAttr(elem, "select")
		if selectAttr != "" {
			expr, err := compileXPath(selectAttr, c.nsBindings)
			if err != nil {
				return nil, err
			}
			return []instruction{&xslSequenceInst{Select: expr}}, nil
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
	return []instruction{inst}, nil
}

// compileBreak compiles an xsl:break element.
func (c *compiler) compileBreak(elem *helium.Element) (instruction, error) {
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
			if chElem.URI() == lexicon.NamespaceXSLT && chElem.LocalName() == lexicon.XSLTElementFallback {
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

	inst := &breakInst{}

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
func (c *compiler) compileNextIteration(elem *helium.Element) (instruction, error) {
	// XTSE0010: xsl:next-iteration must be lexically within xsl:iterate.
	if c.iterateDepth == 0 {
		return nil, staticError(errCodeXTSE0010, "xsl:next-iteration must be within xsl:iterate")
	}
	// XTSE3120: xsl:next-iteration must not be inside an element constructor,
	// xsl:for-each, or other non-iterate-body position.
	if !c.breakAllowed {
		return nil, staticError(errCodeXTSE3120, "xsl:next-iteration is not allowed in this position within xsl:iterate")
	}

	inst := &nextIterationInst{}

	// Check for duplicate with-param names (XTSE0670).
	wpNames := make(map[string]struct{})
	for child := range helium.Children(elem) {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() == lexicon.NamespaceXSLT && childElem.LocalName() == lexicon.XSLTElementWithParam {
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

	acc := &accumulatorDef{
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
	for child := range helium.Children(elem) {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() != lexicon.NamespaceXSLT {
			continue
		}
		if childElem.LocalName() == lexicon.XSLTElementAccumulatorRule {
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
func (c *compiler) compileAccumulatorRule(parent *accumulatorDef, elem *helium.Element) error {
	matchAttr := getAttr(elem, "match")
	if matchAttr == "" {
		return staticError(errCodeXTSE0110, "xsl:accumulator-rule requires match attribute")
	}

	// XPST0008: $value is only in scope in the select expression/body of
	// an accumulator-rule, not in the match pattern. Reject it early.
	if containsVarRef(matchAttr, "value") {
		return staticError(errCodeXPST0008, "variable $value is not in scope in accumulator-rule match pattern")
	}

	matchPat, err := compilePattern(matchAttr, c.nsBindings, c.xpathDefaultNS)
	if err != nil {
		return err
	}

	rule := &accumulatorRule{
		Match: matchPat,
		Phase: getAttr(elem, "phase"),
		New:   getAttr(elem, "new-value") == lexicon.ValueYes,
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

// containsVarRef checks whether a string contains a variable reference $name
// that is NOT inside a string literal.
func containsVarRef(s, name string) bool {
	ref := "$" + name
	inSingle := false
	inDouble := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '$':
			if !inSingle && !inDouble && i+len(ref)-1 < len(s) {
				if s[i:i+len(ref)] == ref {
					// Check that the next char is not a name char
					end := i + len(ref)
					if end >= len(s) || !isNameChar(rune(s[end])) {
						return true
					}
				}
			}
		}
	}
	return false
}

// sortAccumulatorOrder topologically sorts the accumulator evaluation order
// so that accumulators that depend on others (via accumulator-before/after
// calls in their rule expressions) are evaluated after their dependencies.
// This ensures that when accumulator A calls accumulator-after('B'), B's
// value for the current node has already been computed.
// Cyclic dependencies (including self-references) are detected and the
// involved accumulators are marked with CyclicDeps=true for XTDE3400.
func sortAccumulatorOrder(ss *Stylesheet) {
	// Build dependency graph by scanning rule select expressions for
	// accumulator-before('name') and accumulator-after('name') calls.
	deps := make(map[string]map[string]struct{})
	for name, acc := range ss.accumulators {
		deps[name] = make(map[string]struct{})
		for _, rule := range acc.Rules {
			var src string
			if rule.Select != nil {
				src = rule.Select.String()
			}
			if src == "" {
				continue
			}
			for _, ref := range extractAccumulatorRefs(src) {
				resolved := resolveQName(ref, ss.namespaces)
				if _, ok := ss.accumulators[resolved]; ok {
					deps[name][resolved] = struct{}{}
				}
			}
		}
	}

	// Detect self-cycles: an accumulator that references itself.
	for name := range deps {
		if _, selfRef := deps[name][name]; selfRef {
			if acc, ok := ss.accumulators[name]; ok {
				acc.CyclicDeps = true
			}
			delete(deps[name], name)
		}
	}

	if len(ss.accumulatorOrder) <= 1 {
		return
	}

	// DFS-based topological sort with cycle detection.
	// States: 0=unvisited, 1=visiting (on stack), 2=done
	state := make(map[string]int)
	var sorted []string
	var visit func(string) bool
	visit = func(name string) bool {
		switch state[name] {
		case 2:
			return false // already processed
		case 1:
			return true // cycle detected
		}
		state[name] = 1
		for dep := range deps[name] {
			if visit(dep) {
				// Mark all accumulators in the cycle
				if acc, ok := ss.accumulators[name]; ok {
					acc.CyclicDeps = true
				}
				if acc, ok := ss.accumulators[dep]; ok {
					acc.CyclicDeps = true
				}
			}
		}
		state[name] = 2
		sorted = append(sorted, name)
		return false
	}
	for _, name := range ss.accumulatorOrder {
		visit(name)
	}
	ss.accumulatorOrder = sorted
}

// extractAccumulatorRefs extracts accumulator names referenced by
// accumulator-before('name') and accumulator-after('name') calls
// from an XPath expression source string.
func extractAccumulatorRefs(src string) []string {
	var refs []string
	for _, prefix := range []string{"accumulator-before(", "accumulator-after("} {
		s := src
		for {
			idx := strings.Index(s, prefix)
			if idx < 0 {
				break
			}
			s = s[idx+len(prefix):]
			// Skip whitespace
			s = strings.TrimSpace(s)
			if len(s) == 0 {
				break
			}
			// Expect a quote character
			q := s[0]
			if q != '\'' && q != '"' {
				break
			}
			s = s[1:]
			end := strings.IndexByte(s, q)
			if end < 0 {
				break
			}
			refs = append(refs, s[:end])
			s = s[end+1:]
		}
	}
	return refs
}

// compileMerge compiles an xsl:merge element.
func (c *compiler) compileMerge(elem *helium.Element) (instruction, error) {
	inst := &mergeInst{}

	actionCount := 0
	for child := range helium.Children(elem) {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() != lexicon.NamespaceXSLT {
			continue
		}
		// Resolve shadow attributes on child XSLT elements (e.g.
		// _select="..." on xsl:merge-source) before compilation.
		if childElem.URI() == lexicon.NamespaceXSLT {
			if err := c.resolveShadowAttributes(childElem); err != nil {
				return nil, err
			}
		}

		switch childElem.LocalName() {
		case lexicon.XSLTElementMergeSource:
			src, err := c.compileMergeSource(childElem)
			if err != nil {
				return nil, err
			}
			inst.Sources = append(inst.Sources, src)
		case lexicon.XSLTElementMergeAction:
			actionCount++
			if actionCount > 1 {
				return nil, staticError(errCodeXTSE0010, "xsl:merge must have at most one xsl:merge-action child")
			}
			body, err := c.compileChildren(childElem)
			if err != nil {
				return nil, err
			}
			inst.Action = body
		case lexicon.XSLTElementFallback:
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

	// XTSE3190: duplicate xsl:merge-source names.
	seenNames := make(map[string]struct{})
	for _, src := range inst.Sources {
		name := src.Name
		if name == "" {
			// Implicit name is the position, so no duplicate check needed for unnamed
			continue
		}
		if _, ok := seenNames[name]; ok {
			return nil, staticError(errCodeXTSE3190, "duplicate xsl:merge-source name %q", name)
		}
		seenNames[name] = struct{}{}
	}

	// Streamability checks when any merge-source has streamable="yes".
	if err := mergeStreamCheck(inst); err != nil {
		return nil, err
	}

	return inst, nil
}

// mergeSourceAllowedAttrs lists the valid attributes on xsl:merge-source.
var mergeSourceAllowedAttrs = map[string]struct{}{
	"name": {}, "for-each-item": {}, "for-each-source": {},
	"select": {}, "streamable": {}, "sort-before-merge": {},
	"use-accumulators": {}, "validation": {}, "type": {},
}

// compileMergeSource compiles an xsl:merge-source element.
func (c *compiler) compileMergeSource(elem *helium.Element) (*mergeSource, error) {
	if err := c.validateXSLTAttrs(elem, mergeSourceAllowedAttrs); err != nil {
		return nil, err
	}
	// XTSE0020: the type attribute requires schema-aware type validation
	// on merge-source items, which is not yet implemented.
	if typeAttr := getAttr(elem, "type"); typeAttr != "" {
		return nil, staticError(errCodeXTSE0020, "xsl:merge-source type=%q attribute validation is not supported", typeAttr)
	}
	src := &mergeSource{
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
	for child := range helium.Children(elem) {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() == lexicon.NamespaceXSLT && childElem.LocalName() == lexicon.XSLTElementMergeKey {
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
	// Fall back to non-streaming rather than raising a fatal error.
	if src.SortBeforeMerge && src.StreamableAttr {
		src.StreamableAttr = false
	}
	return src, nil
}

// compileMergeKey compiles an xsl:merge-key element.
func (c *compiler) compileMergeKey(elem *helium.Element) (*mergeKey, error) {
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

	mk := &mergeKey{
		Order: "ascending", // default
	}

	// Track collation-related attributes for sort verification.
	collation := getAttr(elem, "collation")
	mk.Collation = collation
	mk.Lang = getAttr(elem, "lang")
	mk.CaseOrder = getAttr(elem, "case-order")
	if mk.Lang != "" || mk.CaseOrder != "" {
		mk.HasCollation = true
	} else if collation != "" && collation != lexicon.CollationCodepoint {
		mk.HasCollation = true
	}
	// Compile the collation attribute as an AVT so that expressions like
	// collation="{$collation}" are resolved at runtime.
	if collation != "" {
		collAVT, err := compileAVT(collation, c.nsBindings)
		if err == nil && collAVT != nil {
			mk.CollationAVT = collAVT
		}
	}

	selAttr := getAttr(elem, "select")

	// XTSE3200: merge-key must not have both select and body.
	if selAttr != "" {
		for ch := range helium.Children(elem) {
			if chElem, ok := ch.(*helium.Element); ok {
				if chElem.URI() == lexicon.NamespaceXSLT && chElem.LocalName() == lexicon.XSLTElementFallback {
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
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		mk.Body = body
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

func mergeStreamCheck(inst *mergeInst) error {
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
				return staticError(errCodeXTSE3470, "xsl:merge-key @order must not be an AVT when xsl:merge-source is streamable")
			}
			if mk.DataTypeAVT != nil {
				return staticError(errCodeXTSE3470, "xsl:merge-key @data-type must not be an AVT when xsl:merge-source is streamable")
			}
		}
	}
	// XTSE3430: when a streamable merge-source select uses descendant-or-self
	// axis (//), the merge-key must not navigate upward.
	// Fall back to non-streaming rather than raising a fatal error.
	for _, src := range inst.Sources {
		if !src.StreamableAttr || src.Select == nil {
			continue
		}
		if !xpath3.ExprUsesDescendantOrSelf(src.Select) {
			continue
		}
		for _, mk := range src.Keys {
			if xpath3.ExprUsesUpwardAxis(mk.Select) {
				src.StreamableAttr = false
				break
			}
		}
	}
	return nil
}
