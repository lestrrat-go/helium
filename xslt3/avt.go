package xslt3

import (
	"context"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/internal/xpathstream"
	"github.com/lestrrat-go/helium/xpath3"
)

// avt is a compiled Attribute Value template. It consists of alternating
// literal strings and XPath expressions enclosed in curly braces.
type avt struct {
	parts []avtPart
}

type avtPart struct {
	literal string
	expr    *xpath3.Expression
}

// hasFunction returns true if any expression part of the avt uses the
// named function (no namespace prefix).
func (a *avt) hasFunction(name string) bool {
	if a == nil {
		return false
	}
	for _, p := range a.parts {
		if p.expr != nil && xpathstream.ExprUsesFunction(p.expr, name) {
			return true
		}
	}
	return false
}

// compileAVT compiles an attribute value template string.
// AVTs contain literal text interspersed with {expr} XPath expressions.
// {{ and }} are escape sequences for literal { and }.
func compileAVT(s string, nsBindings map[string]string) (*avt, error) {
	var parts []avtPart
	i := 0
	for i < len(s) {
		switch s[i] {
		case '{':
			if i+1 < len(s) && s[i+1] == '{' {
				// escaped {{
				parts = appendLiteral(parts, "{")
				i += 2
				continue
			}
			// Find matching '}' tracking nesting depth and string literals.
			end := findAVTExprEnd(s[i+1:])
			if end < 0 {
				return nil, staticError(errCodeXTSE0580, "unterminated AVT expression in %q", s)
			}
			exprStr := s[i+1 : i+1+end]
			// Empty expression {} or comment-only expression {(: ... :)}
			// produce empty string per XSLT 3.0 spec.
			trimmed := strings.TrimSpace(exprStr)
			if trimmed == "" || isXPathCommentOnly(trimmed) {
				parts = appendLiteral(parts, "")
			} else {
				expr, err := compileXPath(exprStr, nsBindings)
				if err != nil {
					return nil, staticError(errCodeXTSE0580, "invalid XPath in AVT: %v", err)
				}
				parts = append(parts, avtPart{expr: expr})
			}
			i = i + 1 + end + 1
		case '}':
			if i+1 < len(s) && s[i+1] == '}' {
				// escaped }}
				parts = appendLiteral(parts, "}")
				i += 2
				continue
			}
			return nil, staticError(errCodeXTSE0580, "unmatched } in AVT %q", s)
		default:
			// literal text until next { or }
			j := i + 1
			for j < len(s) && s[j] != '{' && s[j] != '}' {
				j++
			}
			parts = appendLiteral(parts, s[i:j])
			i = j
		}
	}
	return &avt{parts: parts}, nil
}

// findAVTExprEnd finds the index of the closing '}' for an avt expression,
// tracking brace nesting (for EQName Q{uri}local, map{}, array{}) and
// skipping braces inside string literals.
func findAVTExprEnd(s string) int {
	depth := 0
	commentDepth := 0
	inSingle := false
	inDouble := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		// Handle XPath comments (: ... :)
		if commentDepth > 0 {
			if i+1 < len(s) && ch == ':' && s[i+1] == ')' {
				commentDepth--
				i++ // skip ')'
			} else if i+1 < len(s) && ch == '(' && s[i+1] == ':' {
				commentDepth++
				i++ // skip ':'
			}
			continue
		}
		switch {
		case i+1 < len(s) && ch == '(' && s[i+1] == ':' && !inSingle && !inDouble:
			commentDepth++
			i++ // skip ':'
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
		case ch == '"' && !inSingle:
			inDouble = !inDouble
		case ch == '{' && !inSingle && !inDouble:
			depth++
		case ch == '}' && !inSingle && !inDouble:
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

// isXPathCommentOnly checks if a string consists only of XPath comments.
// E.g., "(: this is a comment :)" or "(: nested (: comment :) :)"
func isXPathCommentOnly(s string) bool {
	i := 0
	for i < len(s) {
		if i+1 < len(s) && s[i] == '(' && s[i+1] == ':' {
			// Find matching :)
			depth := 1
			j := i + 2
			for j < len(s) && depth > 0 {
				if j+1 < len(s) && s[j] == '(' && s[j+1] == ':' {
					depth++
					j += 2
					continue
				}
				if j+1 < len(s) && s[j] == ':' && s[j+1] == ')' {
					depth--
					j += 2
					continue
				}
				j++
			}
			if depth != 0 {
				return false
			}
			i = j
		} else if s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' {
			i++
		} else {
			return false
		}
	}
	return i > 0
}

func appendLiteral(parts []avtPart, s string) []avtPart {
	if len(parts) > 0 && parts[len(parts)-1].expr == nil {
		parts[len(parts)-1].literal += s
		return parts
	}
	return append(parts, avtPart{literal: s})
}

// staticValue returns the literal string value if the avt contains no
// dynamic XPath expressions, and true. If the avt contains any dynamic
// parts, it returns ("", false).
func (a *avt) staticValue() (string, bool) {
	if a == nil {
		return "", false
	}
	var sb strings.Builder
	for _, p := range a.parts {
		if p.expr != nil {
			return "", false
		}
		sb.WriteString(p.literal)
	}
	return sb.String(), true
}

// evaluate evaluates the avt in the given context.
func (a *avt) evaluate(ctx context.Context, node helium.Node) (string, error) {
	if a == nil {
		return "", nil
	}

	// If the context carries an execContext, use the Evaluator-based path
	ec := getExecContext(ctx)

	var sb strings.Builder
	for _, p := range a.parts {
		if p.expr != nil {
			var result *xpath3.Result
			var err error
			if ec != nil {
				result, err = ec.evalXPath(ctx, p.expr, node)
			} else {
				result, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, p.expr, node)
			}
			if err != nil {
				return "", &XSLTError{Code: errCodeXTDE0045, Message: "AVT evaluation error: " + err.Error(), Cause: err}
			}
			sb.WriteString(stringifyResult(result))
		} else {
			sb.WriteString(p.literal)
		}
	}
	return sb.String(), nil
}

// evaluateStatic evaluates the avt at compile time using the given Evaluator.
// This avoids the deprecated Expression.Evaluate(ctx, node) path.
func (a *avt) evaluateStatic(eval xpath3.Evaluator, node helium.Node) (string, error) {
	if a == nil {
		return "", nil
	}
	var sb strings.Builder
	for _, p := range a.parts {
		if p.expr != nil {
			result, err := eval.Evaluate(context.Background(), p.expr, node)
			if err != nil {
				return "", &XSLTError{Code: errCodeXTDE0045, Message: "AVT evaluation error: " + err.Error(), Cause: err}
			}
			sb.WriteString(stringifyResult(result))
		} else {
			sb.WriteString(p.literal)
		}
	}
	return sb.String(), nil
}

// stringifyResult converts an xpath3.Result to its string representation.
// Adjacent text nodes are merged before items are joined with spaces,
// consistent with the XSLT 3.0 simple content construction rules (§5.7.2).
func stringifyResult(r *xpath3.Result) string {
	return stringifySimpleContent(r.Sequence(), " ")
}

// stringifySequence converts a Sequence to its string representation
// by atomizing each item and joining with spaces.
func stringifySequence(seq xpath3.Sequence) string {
	return stringifySequenceWithSep(seq, " ")
}

// stringifyItem converts a single XPath Item to its string representation.
func stringifyItem(item xpath3.Item) string {
	av, err := xpath3.AtomizeItem(item)
	if err != nil {
		return ""
	}
	s, err := xpath3.AtomicToString(av)
	if err != nil {
		return ""
	}
	return s
}

func stringifySequenceWithSep(seq xpath3.Sequence, sep string) string {
	seq = flattenArraysInSequence(seq)
	if seq == nil || sequence.Len(seq) == 0 {
		return ""
	}
	// Atomize the entire sequence at once so that list-typed nodes
	// are decomposed into their individual atomic items (each one
	// rendered in canonical form).
	atoms, err := xpath3.AtomizeSequence(seq)
	if err != nil {
		// Fall back to per-item atomization on error.
		var sb strings.Builder
		i := 0
		for item := range sequence.Items(seq) {
			if i > 0 {
				sb.WriteString(sep)
			}
			av, atomErr := xpath3.AtomizeItem(item)
			if atomErr != nil {
				continue
			}
			s, strErr := xpath3.AtomicToString(av)
			if strErr != nil {
				continue
			}
			sb.WriteString(s)
			i++
		}
		return sb.String()
	}
	var sb strings.Builder
	for i, av := range atoms {
		if i > 0 {
			sb.WriteString(sep)
		}
		s, strErr := xpath3.AtomicToString(av)
		if strErr != nil {
			continue
		}
		sb.WriteString(s)
	}
	return sb.String()
}

// stringifySimpleContent implements XSLT 3.0 §5.7.2 simple content
// construction: adjacent text nodes are merged first, zero-length
// text nodes are removed, then remaining items are separated by the
// given separator string.
func stringifySimpleContent(seq xpath3.Sequence, sep string) string {
	seq = flattenArraysInSequence(seq)
	if seq == nil || sequence.Len(seq) == 0 {
		return ""
	}
	// Step 1: merge adjacent text nodes.
	merged := mergeAdjacentTextNodes(seq)
	// Step 2: remove zero-length text nodes so they don't produce
	// stray separators (e.g. empty TVTs in sequence mode).
	merged = removeEmptyTextNodes(merged)
	// Step 3: stringify each item, separating with sep.
	return stringifySequenceWithSep(merged, sep)
}

// removeEmptyTextNodes filters out text nodes with zero-length content.
func removeEmptyTextNodes(seq xpath3.Sequence) xpath3.Sequence {
	if seq == nil {
		return nil
	}
	seqLen := sequence.Len(seq)
	result := make(xpath3.ItemSlice, 0, seqLen)
	for i := 0; i < seqLen; i++ {
		item := seq.Get(i)
		ni, ok := item.(xpath3.NodeItem)
		if ok && ni.Node.Type() == helium.TextNode && len(ni.Node.Content()) == 0 {
			continue
		}
		result = append(result, item)
	}
	return result
}

// flattenArraysInSequence recursively replaces any ArrayItem in the
// sequence with its flattened members.  Non-array items pass through
// unchanged.  This implements the XSLT 3.0 rule that arrays appearing
// in sequence constructors, apply-templates select, value-of, etc.
// are expanded before further processing.
func flattenArraysInSequence(seq xpath3.Sequence) xpath3.Sequence {
	if seq == nil {
		return nil
	}
	hasArray := false
	for item := range sequence.Items(seq) {
		if _, ok := item.(xpath3.ArrayItem); ok {
			hasArray = true
			break
		}
	}
	if !hasArray {
		return seq
	}
	var result xpath3.ItemSlice
	for item := range sequence.Items(seq) {
		if arr, ok := item.(xpath3.ArrayItem); ok {
			result = append(result, sequence.Materialize(arr.Flatten())...)
		} else {
			result = append(result, item)
		}
	}
	return result
}
