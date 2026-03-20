package xslt3

import (
	"bytes"
	"context"
	"errors"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
)

func (ec *execContext) execMessage(ctx context.Context, inst *MessageInst) error {
	var value string
	var bodySeq xpath3.Sequence
	if inst.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			// Errors evaluating message content are recoverable
			value = err.Error()
		} else {
			bodySeq = result.Sequence()
			value = serializeMessageSequence(bodySeq)
		}
	}
	if len(inst.Body) > 0 {
		// XSLT 2.0: message body is temporary output state (XTDE1480).
		// XSLT 3.0 relaxes this restriction.
		isV2TempOutput := ec.stylesheet.version != "" && ec.stylesheet.version < "3.0"
		if isV2TempOutput {
			ec.temporaryOutputDepth++
		}
		val, err := ec.evaluateBody(ctx, inst.Body)
		if isV2TempOutput {
			ec.temporaryOutputDepth--
		}
		if err != nil {
			// XTDE1480 is a fatal error, not recoverable.
			if isXSLTError(err, errCodeXTDE1480) {
				return err
			}
			// Other errors evaluating message body are recoverable
			value += err.Error()
		} else {
			bodySeq = append(bodySeq, val...)
			value += serializeMessageSequence(val)
		}
	}

	terminate := false
	if inst.Terminate != nil {
		termStr, err := inst.Terminate.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		termStr = strings.TrimSpace(termStr)
		terminate = termStr == "yes" || termStr == "true" || termStr == "1"
	}

	if ec.msgHandler != nil {
		ec.msgHandler(value, terminate)
	}

	if terminate {
		errorCode := errCodeXTMM9000
		if inst.ErrorCode != nil {
			code, err := inst.ErrorCode.evaluate(ctx, ec.contextNode)
			if err == nil && code != "" {
				errorCode = code
			}
		}
		return &XSLTError{
			Code:    errorCode,
			Message: value,
			Value:   bodySeq,
			Cause:   ErrTerminated,
		}
	}
	return nil
}

// serializeMessageSequence converts a sequence to a string suitable for
// xsl:message output. Node items (elements, documents) are serialized as
// XML so that the message preserves markup structure. Atomic values are
// converted to their string representations and joined with spaces.
func serializeMessageSequence(seq xpath3.Sequence) string {
	if len(seq) == 0 {
		return ""
	}
	var sb strings.Builder
	prevWasAtomic := false
	for _, item := range seq {
		ni, isNode := item.(xpath3.NodeItem)
		if !isNode {
			// Atomic value — stringify and space-separate
			av, err := xpath3.AtomizeItem(item)
			if err != nil {
				continue
			}
			s, err := xpath3.AtomicToString(av)
			if err != nil {
				continue
			}
			if prevWasAtomic && sb.Len() > 0 {
				sb.WriteByte(' ')
			}
			sb.WriteString(s)
			prevWasAtomic = true
			continue
		}
		prevWasAtomic = false
		var buf bytes.Buffer
		switch n := ni.Node.(type) {
		case *helium.Document:
			_ = n.XML(&buf, helium.WithNoDecl())
		case *helium.Element:
			_ = n.XML(&buf)
		case *helium.Comment:
			// Serialize comments as XML so they roundtrip properly
			buf.WriteString("<!--")
			buf.Write(n.Content())
			buf.WriteString("-->")
		case *helium.ProcessingInstruction:
			buf.WriteString("<?")
			buf.WriteString(n.Name())
			if c := n.Content(); len(c) > 0 {
				buf.WriteByte(' ')
				buf.Write(c)
			}
			buf.WriteString("?>")
		default:
			// Text, CDATA, etc. — use string value
			sb.WriteString(string(ni.Node.Content()))
			continue
		}
		sb.WriteString(strings.TrimSpace(buf.String()))
	}
	return sb.String()
}

func (ec *execContext) execTryCatch(ctx context.Context, inst *TryCatchInst) error {
	// When rollback-output="no", write directly to the real output.
	// If the try fails after producing output, raise XTDE3530.
	if !inst.RollbackOutput {
		return ec.execTryCatchNoRollback(ctx, inst)
	}

	// Execute try body into a temporary output buffer.
	// If the try succeeds, copy the buffered output to the real output.
	// If it fails, discard the buffer and execute the catch.
	tmpDoc := helium.NewDefaultDocument()
	tmpRoot, err := tmpDoc.CreateElement("_try")
	if err != nil {
		return err
	}
	if err := tmpDoc.AddChild(tmpRoot); err != nil {
		return err
	}

	// Inherit captureItems from the parent frame so that function items
	// (maps, arrays) produced inside the try body can flow through.
	parentCapture := ec.currentOutput().captureItems
	ec.outputStack = append(ec.outputStack, &outputFrame{doc: tmpDoc, current: tmpRoot, captureItems: parentCapture})

	// Push a new variable scope for the try body so variables
	// declared inside the try are not visible in catch.
	savedVarScope := ec.localVars
	ec.pushVarScope()

	tryErr := func() error {
		if inst.Select != nil {
			xpathCtx := ec.newXPathContext(ec.contextNode)
			result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
			if err != nil {
				return err
			}
			return ec.outputSequence(result.Sequence())
		}
		for _, child := range inst.Try {
			if err := ec.executeInstruction(ctx, child); err != nil {
				return err
			}
		}
		return nil
	}()

	tryFrame := ec.outputStack[len(ec.outputStack)-1]
	ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]

	// xsl:break and xsl:next-iteration are control flow signals, not errors.
	// If one occurred inside the try body, copy the buffered output (produced
	// before the signal) to the real output and propagate the signal.
	if tryErr != nil && (errors.Is(tryErr, errBreak) || errors.Is(tryErr, errNextIter)) {
		for child := tmpRoot.FirstChild(); child != nil; child = child.NextSibling() {
			copied, copyErr := helium.CopyNode(child, ec.resultDoc)
			if copyErr != nil {
				return copyErr
			}
			if err := ec.addNode(copied); err != nil {
				return err
			}
		}
		return tryErr
	}

	if tryErr == nil {
		// Success — copy buffered output to real output
		for child := tmpRoot.FirstChild(); child != nil; child = child.NextSibling() {
			copied, copyErr := helium.CopyNode(child, ec.resultDoc)
			if copyErr != nil {
				return copyErr
			}
			if err := ec.addNode(copied); err != nil {
				return err
			}
		}
		// Transfer captured items (maps, arrays, function items) to parent.
		if len(tryFrame.pendingItems) > 0 {
			out := ec.currentOutput()
			out.pendingItems = append(out.pendingItems, tryFrame.pendingItems...)
			out.noteOutput()
		}
		return nil
	}

	// Extract error code and QName from the error.
	// Unwrap wrapper errors (e.g., AVT XTDE0045 wrapping FOAR0001) to find
	// the most specific XPath/XSLT error code for catch clause matching.
	errNS := lexicon.Err
	errCode := errCodeXSLT0000
	errDesc := tryErr.Error()
	var errQName xpath3.QNameValue
	if xErr, ok := errors.AsType[*XSLTError](tryErr); ok {
		errCode = xErr.Code
		errDesc = xErr.Message
		errQName = xpath3.QNameValue{Prefix: "err", URI: errNS, Local: errCode}
		// If this is a wrapper error (like XTDE0045 for AVT), check inner
		// cause for a more specific error code.
		if xErr.Cause != nil {
			if innerXP, ok := errors.AsType[*xpath3.XPathError](xErr.Cause); ok {
				errCode = innerXP.Code
				errDesc = innerXP.Message
				errQName = innerXP.CodeQName()
			} else if innerXS, ok := errors.AsType[*XSLTError](xErr.Cause); ok {
				errCode = innerXS.Code
				errDesc = innerXS.Message
				errQName = xpath3.QNameValue{Prefix: "err", URI: errNS, Local: innerXS.Code}
			}
		}
	} else if xpErr, ok := errors.AsType[*xpath3.XPathError](tryErr); ok {
		errCode = xpErr.Code
		errDesc = xpErr.Message
		errQName = xpErr.CodeQName()
	}
	if errQName.Local == "" {
		errQName = xpath3.QNameValue{Prefix: "err", URI: errNS, Local: errCode}
	}

	// Build Clark-notation error code for matching against compiled catch patterns
	errClark := errCode
	if errQName.URI != "" {
		errClark = "{" + errQName.URI + "}" + errQName.Local
	}

	// Restore variable scope to before the try body.
	// Variables declared inside the try must not be visible in catch.
	ec.localVars = savedVarScope

	// Find matching catch clause
	var matchedCatch *CatchClause
	for _, clause := range inst.Catches {
		if catchMatches(clause, errClark) {
			matchedCatch = clause
			break
		}
	}
	if matchedCatch == nil {
		// No matching catch — propagate the error
		return tryErr
	}

	// Set XSLT 3.0 error variables in catch scope
	ec.pushVarScope()
	defer ec.popVarScope()

	// $err:code is an xs:QName value with the error code
	errCodeSeq := xpath3.Sequence{xpath3.AtomicValue{
		TypeName: xpath3.TypeQName,
		Value:    errQName,
	}}

	// $err:value carries the sequence associated with the error.
	// For xsl:message terminate="yes", this is the message body content.
	errValueSeq := xpath3.EmptySequence()
	if xErr, ok := errors.AsType[*XSLTError](tryErr); ok {
		if seq, ok := xErr.Value.(xpath3.Sequence); ok && len(seq) > 0 {
			// Copy nodes into the result document so they are usable
			// in the catch body's output tree.
			var copied xpath3.Sequence
			for _, item := range seq {
				if ni, isNode := item.(xpath3.NodeItem); isNode {
					dup, cpErr := helium.CopyNode(ni.Node, ec.resultDoc)
					if cpErr == nil {
						copied = append(copied, xpath3.NodeItem{Node: dup})
					} else {
						copied = append(copied, item)
					}
				} else {
					copied = append(copied, item)
				}
			}
			errValueSeq = copied
		}
	}

	ec.setVar("{"+errNS+"}code", errCodeSeq)
	ec.setVar("{"+errNS+"}description", xpath3.SingleString(errDesc))
	ec.setVar("{"+errNS+"}value", errValueSeq)
	ec.setVar("{"+errNS+"}module", xpath3.SingleString(ec.errSourceModule))
	ec.setVar("{"+errNS+"}line-number", xpath3.SingleInteger(int64(ec.errSourceLine)))
	ec.setVar("{"+errNS+"}column-number", xpath3.SingleInteger(0))

	// Execute matched catch body
	if matchedCatch.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := matchedCatch.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		return ec.outputSequence(result.Sequence())
	}
	return ec.executeSequenceConstructor(ctx, matchedCatch.Body)
}

// execTryCatchNoRollback handles xsl:try with rollback-output="no".
// Output goes directly to the real output. If the try body fails after
// producing output, XTDE3530 is raised instead of catching the error.
func (ec *execContext) execTryCatchNoRollback(ctx context.Context, inst *TryCatchInst) error {
	out := ec.currentOutput()
	snapSerial := out.outputSerial

	savedVarScope := ec.localVars
	ec.pushVarScope()

	tryErr := func() error {
		if inst.Select != nil {
			xpathCtx := ec.newXPathContext(ec.contextNode)
			result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
			if err != nil {
				return err
			}
			return ec.outputSequence(result.Sequence())
		}
		for _, child := range inst.Try {
			if err := ec.executeInstruction(ctx, child); err != nil {
				return err
			}
		}
		return nil
	}()

	if tryErr == nil {
		return nil
	}

	// Control flow signals propagate regardless.
	if errors.Is(tryErr, errBreak) || errors.Is(tryErr, errNextIter) {
		return tryErr
	}

	// XTDE3530: output was written and cannot be rolled back.
	if out.outputSerial != snapSerial {
		return dynamicError(errCodeXTDE3530,
			"xsl:try with rollback-output='no': output was written before error and cannot be rolled back")
	}

	// No output was written — safe to execute catch.
	ec.localVars = savedVarScope
	ec.pushVarScope()
	defer ec.popVarScope()

	errNS := lexicon.Err
	errCode := errCodeXSLT0000
	errDesc := tryErr.Error()
	var errQName xpath3.QNameValue
	if xErr, ok := errors.AsType[*XSLTError](tryErr); ok {
		errCode = xErr.Code
		errDesc = xErr.Message
		errQName = xpath3.QNameValue{Prefix: "err", URI: errNS, Local: errCode}
	}
	errCodeSeq := xpath3.Sequence{xpath3.AtomicValue{
		TypeName: xpath3.TypeQName,
		Value:    errQName,
	}}

	ec.setVar("{"+errNS+"}code", errCodeSeq)
	ec.setVar("{"+errNS+"}description", xpath3.SingleString(errDesc))
	ec.setVar("{"+errNS+"}value", xpath3.EmptySequence())
	ec.setVar("{"+errNS+"}module", xpath3.SingleString(ec.errSourceModule))
	ec.setVar("{"+errNS+"}line-number", xpath3.SingleInteger(int64(ec.errSourceLine)))
	ec.setVar("{"+errNS+"}column-number", xpath3.SingleInteger(0))

	for _, c := range inst.Catches {
		if catchMatches(c, errCode) {
			return ec.executeSequenceConstructor(ctx, c.Body)
		}
	}
	return tryErr
}

// catchMatches returns true if a catch clause matches the given error code.
// errClark is the error code in Clark notation: "{uri}local" or just "local" if no namespace.
func catchMatches(clause *CatchClause, errClark string) bool {
	if len(clause.Errors) == 0 {
		return true // no errors attribute = match all
	}

	// Extract local name from Clark notation for wildcard matching.
	errLocal := errClark
	if strings.HasPrefix(errClark, "{") {
		if idx := strings.IndexByte(errClark, '}'); idx >= 0 {
			errLocal = errClark[idx+1:]
		}
	}

	for _, pattern := range clause.Errors {
		if pattern == "*" {
			return true
		}
		// *:local — match any namespace with this local name
		if strings.HasPrefix(pattern, "*:") {
			if pattern[2:] == errLocal {
				return true
			}
			continue
		}
		// Q{ns}* — match any local name in this namespace
		if strings.HasSuffix(pattern, "}*") {
			ns := pattern[1 : len(pattern)-2]
			if strings.HasPrefix(errClark, "{"+ns+"}") {
				return true
			}
			continue
		}
		// Both errClark and pattern are in the same format from resolveQName:
		// - Prefixed names → "{uri}local" (Clark notation)
		// - Unprefixed names → "local" (no namespace)
		if pattern == errClark {
			return true
		}
	}
	return false
}
