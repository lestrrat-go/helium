package xslt3

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// Sentinel errors for xsl:break and xsl:next-iteration control flow.
var errBreak = errors.New("xsl:break")
var errNextIter = errors.New("xsl:next-iteration")

// execSourceDocument executes xsl:source-document by loading the referenced
// document into a DOM tree and executing the body with that document as context.
func (ec *execContext) execSourceDocument(ctx context.Context, inst *SourceDocumentInst) error {
	// Evaluate the href AVT to get the URI string.
	uri, err := inst.Href.evaluate(ctx, ec.contextNode)
	if err != nil {
		return err
	}

	// Check the document cache first.
	doc, ok := ec.docCache[uri]
	if !ok {
		// Resolve URI relative to stylesheet base URI.
		resolvedURI := uri
		if ec.stylesheet.baseURI != "" && !strings.Contains(uri, "://") && !filepath.IsAbs(uri) {
			baseDir := filepath.Dir(ec.stylesheet.baseURI)
			resolvedURI = filepath.Join(baseDir, uri)
		}

		data, err := os.ReadFile(resolvedURI)
		if err != nil {
			return dynamicError("FODC0002", "xsl:source-document cannot load %q: %v", uri, err)
		}

		doc, err = helium.Parse(ctx, data)
		if err != nil {
			return dynamicError("FODC0002", "xsl:source-document cannot parse %q: %v", uri, err)
		}

		if ec.docCache == nil {
			ec.docCache = make(map[string]*helium.Document)
		}
		ec.docCache[uri] = doc
	}

	// Save and restore source document and context nodes.
	savedSource := ec.sourceDoc
	savedContext := ec.contextNode
	savedCurrent := ec.currentNode
	ec.sourceDoc = doc
	ec.contextNode = doc
	ec.currentNode = doc
	defer func() {
		ec.sourceDoc = savedSource
		ec.contextNode = savedContext
		ec.currentNode = savedCurrent
	}()

	// Execute the body with the loaded document as context.
	for _, child := range inst.Body {
		if err := ec.executeInstruction(ctx, child); err != nil {
			return err
		}
	}
	return nil
}

// execIterate executes xsl:iterate, processing each item in the selected
// sequence with mutable iteration parameters.
func (ec *execContext) execIterate(ctx context.Context, inst *IterateInst) error {
	// Evaluate the select expression.
	xpathCtx := ec.newXPathContext(ec.contextNode)
	result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
	if err != nil {
		return err
	}
	seq := result.Sequence()

	// Initialize iterate params from their defaults.
	paramVals := make(map[string]xpath3.Sequence, len(inst.Params))
	for _, p := range inst.Params {
		if p.Select != nil {
			pCtx := ec.newXPathContext(ec.contextNode)
			pResult, err := p.Select.Evaluate(pCtx, ec.contextNode)
			if err != nil {
				return err
			}
			paramVals[p.Name] = pResult.Sequence()
		} else if len(p.Body) > 0 {
			val, err := ec.evaluateBody(ctx, p.Body)
			if err != nil {
				return err
			}
			paramVals[p.Name] = val
		} else {
			paramVals[p.Name] = xpath3.EmptySequence()
		}
	}

	// Save and restore context.
	savedCurrent := ec.currentNode
	savedContext := ec.contextNode
	savedPos := ec.position
	savedSize := ec.size
	savedItem := ec.contextItem
	defer func() {
		ec.currentNode = savedCurrent
		ec.contextNode = savedContext
		ec.position = savedPos
		ec.size = savedSize
		ec.contextItem = savedItem
	}()

	ec.size = len(seq)

	completed := true
	for i, item := range seq {
		ec.position = i + 1

		// Set context item/node.
		if ni, ok := item.(xpath3.NodeItem); ok {
			ec.currentNode = ni.Node
			ec.contextNode = ni.Node
			ec.contextItem = nil
		} else {
			ec.contextItem = item
		}

		// Push var scope and set iterate param values.
		ec.pushVarScope()
		for name, val := range paramVals {
			ec.setVar(name, val)
		}

		// Execute body.
		var bodyErr error
		for _, child := range inst.Body {
			bodyErr = ec.executeInstruction(ctx, child)
			if bodyErr != nil {
				break
			}
		}

		ec.popVarScope()

		if bodyErr != nil {
			if errors.Is(bodyErr, errBreak) {
				completed = false
				break
			}
			if errors.Is(bodyErr, errNextIter) {
				// Update params from next-iteration with-params.
				if ec.nextIterParams != nil {
					for name, val := range ec.nextIterParams {
						paramVals[name] = val
					}
					ec.nextIterParams = nil
				}
				continue
			}
			return bodyErr
		}
	}

	if !completed {
		// xsl:break was executed — output the break value if any.
		if ec.breakValue != nil {
			if err := ec.outputSequence(ec.breakValue); err != nil {
				return err
			}
			ec.breakValue = nil
		}
	} else if len(inst.OnCompletion) > 0 {
		// Execute on-completion if present and loop completed normally.
		ec.pushVarScope()
		for name, val := range paramVals {
			ec.setVar(name, val)
		}
		for _, child := range inst.OnCompletion {
			if err := ec.executeInstruction(ctx, child); err != nil {
				ec.popVarScope()
				return err
			}
		}
		ec.popVarScope()
	}

	return nil
}

// execFork executes xsl:fork by running each branch sequentially.
// In a true streaming implementation branches would run concurrently,
// but for the DOM-materialization strategy sequential execution is correct.
func (ec *execContext) execFork(ctx context.Context, inst *ForkInst) error {
	for _, branch := range inst.Branches {
		for _, child := range branch {
			if err := ec.executeInstruction(ctx, child); err != nil {
				return err
			}
		}
	}
	return nil
}

// execBreak executes xsl:break, which terminates the enclosing xsl:iterate.
func (ec *execContext) execBreak(ctx context.Context, inst *BreakInst) error {
	if inst.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		ec.breakValue = result.Sequence()
	} else if len(inst.Body) > 0 {
		val, err := ec.evaluateBody(ctx, inst.Body)
		if err != nil {
			return err
		}
		ec.breakValue = val
	}
	return errBreak
}

// execNextIteration executes xsl:next-iteration, which signals the enclosing
// xsl:iterate to advance to the next item with updated parameter values.
func (ec *execContext) execNextIteration(ctx context.Context, inst *NextIterationInst) error {
	params := make(map[string]xpath3.Sequence, len(inst.Params))
	for _, wp := range inst.Params {
		val, err := ec.evaluateWithParam(ctx, wp)
		if err != nil {
			return err
		}
		params[wp.Name] = val
	}
	ec.nextIterParams = params
	return errNextIter
}
