package xslt3

import (
	"context"
)

// executeInstruction dispatches execution of a compiled XSLT instruction.
func (ec *execContext) executeInstruction(ctx context.Context, inst instruction) error {
	// Track source location for xsl:catch error variables ($err:line-number, $err:module)
	if s, ok := inst.(interface {
		getSourceLine() int
		getSourceModule() string
	}); ok {
		if line := s.getSourceLine(); line > 0 {
			ec.errSourceLine = line
			ec.errSourceModule = s.getSourceModule()
		}
	}

	// Apply per-instruction xpath-default-namespace if set
	if h, ok := inst.(xpathNSHolder); ok && h.xpathNSIsSet() {
		savedNS := ec.xpathDefaultNS
		savedHas := ec.hasXPathDefaultNS
		ec.xpathDefaultNS = h.getXPathDefaultNS()
		ec.hasXPathDefaultNS = true
		defer func() {
			ec.xpathDefaultNS = savedNS
			ec.hasXPathDefaultNS = savedHas
		}()
	}
	switch v := inst.(type) {
	case *applyTemplatesInst:
		return ec.execApplyTemplates(ctx, v)
	case *callTemplateInst:
		return ec.execCallTemplate(ctx, v)
	case *valueOfInst:
		return ec.execValueOf(ctx, v)
	case *textInst:
		return ec.execText(v)
	case *literalTextInst:
		return ec.execLiteralText(v)
	case *elementInst:
		return ec.execElement(ctx, v)
	case *attributeInst:
		return ec.execAttribute(ctx, v)
	case *commentInst:
		return ec.execComment(ctx, v)
	case *piInst:
		return ec.execPI(ctx, v)
	case *ifInst:
		return ec.execIf(ctx, v)
	case *chooseInst:
		return ec.execChoose(ctx, v)
	case *forEachInst:
		return ec.execForEach(ctx, v)
	case *variableInst:
		return ec.execVariable(ctx, v)
	case *paramInst:
		return ec.execParam(ctx, v)
	case *copyInst:
		return ec.execCopy(ctx, v)
	case *copyOfInst:
		return ec.execCopyOf(ctx, v)
	case *literalResultElement:
		return ec.execLiteralResultElement(ctx, v)
	case *messageInst:
		return ec.execMessage(ctx, v)
	case *numberInst:
		return ec.execNumber(ctx, v)
	case *sequenceInst:
		if v.DefaultValidation != "" {
			saved := ec.defaultValidation
			ec.defaultValidation = v.DefaultValidation
			err := ec.executeSequenceConstructor(ctx, v.Body)
			ec.defaultValidation = saved
			return err
		}
		return ec.executeSequenceConstructor(ctx, v.Body)
	case *resultDocumentInst:
		return ec.execResultDocument(ctx, v)
	case *xslSequenceInst:
		return ec.execXSLSequence(ctx, v)
	case *performSortInst:
		return ec.execPerformSort(ctx, v)
	case *nextMatchInst:
		return ec.execNextMatch(ctx, v)
	case *applyImportsInst:
		return ec.execApplyImports(ctx, v)
	case *wherePopulatedInst:
		return ec.execWherePopulated(ctx, v)
	case *onEmptyInst:
		return ec.execOnEmpty(ctx, v)
	case *onNonEmptyInst:
		return ec.execOnNonEmpty(ctx, v)
	case *tryCatchInst:
		return ec.execTryCatch(ctx, v)
	case *forEachGroupInst:
		return ec.execForEachGroup(ctx, v)
	case *namespaceInst:
		return ec.execNamespace(ctx, v)
	case *sourceDocumentInst:
		return ec.execSourceDocument(ctx, v)
	case *iterateInst:
		return ec.execIterate(ctx, v)
	case *forkInst:
		return ec.execFork(ctx, v)
	case *breakInst:
		return ec.execBreak(ctx, v)
	case *nextIterationInst:
		return ec.execNextIteration(ctx, v)
	case *mergeInst:
		return ec.execMerge(ctx, v)
	case *mapInst:
		return ec.execMap(ctx, v)
	case *mapEntryInst:
		return ec.execMapEntry(ctx, v)
	case *analyzeStringInst:
		return ec.execAnalyzeString(ctx, v)
	case *assertInst:
		return ec.execAssert(ctx, v)
	case *documentInst:
		return ec.execDocument(ctx, v)
	case *evaluateInst:
		return ec.execEvaluate(ctx, v)
	case *fallbackInst:
		return ec.execFallback(ctx, v)
	default:
		return dynamicError(errCodeXTDE0820, "unsupported instruction type %T", inst)
	}
}

// execFallback executes a forwards-compatible unknown instruction.
// If fallback instructions are present, they are executed; otherwise XTDE1450.
func (ec *execContext) execFallback(ctx context.Context, inst *fallbackInst) error {
	if !inst.HasFallback {
		return dynamicError(errCodeXTDE1450,
			"no xsl:fallback was found for forwards-compatible instruction %s", inst.Name)
	}
	return ec.executeSequenceConstructor(ctx, inst.Body)
}
