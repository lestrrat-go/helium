package xslt3

import (
	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

func (c *compiler) compileLocalVariable(elem *helium.Element) (*VariableInst, error) {
	name := getAttr(elem, "name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:variable requires name attribute")
	}

	asAttr := c.resolveAsType(getAttr(elem, "as"))
	if err := c.validateAsSequenceType(asAttr, "xsl:variable "+name); err != nil {
		return nil, err
	}

	inst := &VariableInst{
		Name: resolveQName(name, c.nsBindings),
		As:   asAttr,
	}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		// XTSE0620: select and non-empty content are mutually exclusive.
		if err := c.validateEmptyElement(elem, "xsl:variable"); err != nil {
			return nil, staticError(errCodeXTSE0620, "xsl:variable %q has both @select and content", name)
		}
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

func (c *compiler) compileMessage(elem *helium.Element) (*MessageInst, error) {
	inst := &MessageInst{}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	}
	// XSLT 3.0: both select and body content are allowed
	body, err := c.compileChildren(elem)
	if err != nil {
		return nil, err
	}
	inst.Body = body

	termAttr := getAttr(elem, "terminate")
	if termAttr != "" {
		avt, err := compileAVT(termAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		// If the AVT is a static value, validate it is "yes" or "no"
		if sv, ok := avt.staticValue(); ok {
			if sv != lexicon.ValueYes && sv != lexicon.ValueNo {
				return nil, staticError(errCodeXTSE0020, "%q is not a valid value for xsl:message/@terminate", sv)
			}
		}
		inst.Terminate = avt
	}

	errorCodeAttr := getAttr(elem, "error-code")
	if errorCodeAttr != "" {
		avt, err := compileAVT(errorCodeAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.ErrorCode = avt
	}

	return inst, nil
}

func (c *compiler) compileMap(elem *helium.Element) (*MapInst, error) {
	body, err := c.compileChildren(elem)
	if err != nil {
		return nil, err
	}
	return &MapInst{Body: body}, nil
}

func (c *compiler) compileMapEntry(elem *helium.Element) (*MapEntryInst, error) {
	inst := &MapEntryInst{}

	keyAttr := getAttr(elem, "key")
	if keyAttr != "" {
		expr, err := compileXPath(keyAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Key = expr
	} else {
		return nil, staticError(errCodeXTSE0010, "xsl:map-entry requires key attribute")
	}

	selectAttr := getAttr(elem, "select")
	hasBody := hasSignificantChildren(elem)
	if selectAttr != "" && hasBody {
		return nil, staticError("XTSE3280", "xsl:map-entry must not have both a select attribute and a sequence constructor body")
	}
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

// compileAssert compiles xsl:assert.
// Required attribute: test.
// Optional attributes: select, error-code.
// The body provides the error message if no select attribute is present.
func (c *compiler) compileAssert(elem *helium.Element) (Instruction, error) {
	testAttr := getAttr(elem, "test")
	if testAttr == "" {
		return nil, staticError(errCodeXTSE0010, "xsl:assert requires a test attribute")
	}

	testExpr, err := compileXPath(testAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &AssertInst{
		Test:      testExpr,
		ErrorCode: errCodeXTMM9001, // default error code per XSLT 3.0 spec
	}

	// xpath-default-namespace
	if xdn := getAttr(elem, "xpath-default-namespace"); xdn != "" {
		inst.XPathDefaultNS = xdn
		inst.HasXPathDefaultNS = true
	}

	// error-code attribute
	if ec := getAttr(elem, "error-code"); ec != "" {
		inst.ErrorCode = resolveQName(ec, c.nsBindings)
	}

	// select attribute
	if sel := getAttr(elem, "select"); sel != "" {
		selExpr, err := compileXPath(sel, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = selExpr
	}

	// Compile body (error message content)
	body, err := c.compileChildren(elem)
	if err != nil {
		return nil, err
	}
	inst.Body = body

	return inst, nil
}
