package xslt3

import (
	"github.com/lestrrat-go/helium"
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
		Streamable: getAttr(elem, "streamable") == "yes",
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
	inParams := true
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
				if inParams {
					p, err := c.compileIterateParam(childElem)
					if err != nil {
						return nil, err
					}
					inst.Params = append(inst.Params, p)
					continue
				}
				// param after non-param content is a body instruction
			case "on-completion":
				// Check for select attribute first.
				if selAttr := getAttr(childElem, "select"); selAttr != "" {
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

	return inst, nil
}

// compileIterateParam compiles an xsl:param inside xsl:iterate.
func (c *compiler) compileIterateParam(elem *helium.Element) (*IterateParam, error) {
	name := getAttr(elem, "name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:param requires name attribute")
	}

	p := &IterateParam{
		Name: resolveQName(name, c.nsBindings),
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
	inst := &BreakInst{}

	selectAttr := getAttr(elem, "select")
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
	inst := &NextIterationInst{}

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
		Streamable: getAttr(elem, "streamable") == "yes",
	}

	// Read initial-value attribute
	if iv := getAttr(elem, "initial-value"); iv != "" {
		expr, err := compileXPath(iv, c.nsBindings)
		if err != nil {
			return err
		}
		acc.Initial = expr
	}

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

	// Default phase is "end" if not specified
	if rule.Phase == "" {
		rule.Phase = "end"
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
