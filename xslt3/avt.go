package xslt3

import (
	"context"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// AVT is a compiled Attribute Value Template. It consists of alternating
// literal strings and XPath expressions enclosed in curly braces.
type AVT struct {
	parts []avtPart
}

type avtPart struct {
	literal string
	expr    *xpath3.Expression
}

// hasFunction returns true if any expression part of the AVT uses the
// named function (no namespace prefix).
func (a *AVT) hasFunction(name string) bool {
	if a == nil {
		return false
	}
	for _, p := range a.parts {
		if p.expr != nil && xpath3.ExprUsesFunction(p.expr, name) {
			return true
		}
	}
	return false
}

// compileAVT compiles an attribute value template string.
// AVTs contain literal text interspersed with {expr} XPath expressions.
// {{ and }} are escape sequences for literal { and }.
func compileAVT(s string, nsBindings map[string]string) (*AVT, error) {
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
	return &AVT{parts: parts}, nil
}

// findAVTExprEnd finds the index of the closing '}' for an AVT expression,
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

// staticValue returns the literal string value if the AVT contains no
// dynamic XPath expressions, and true. If the AVT contains any dynamic
// parts, it returns ("", false).
func (a *AVT) staticValue() (string, bool) {
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

// evaluate evaluates the AVT in the given context.
func (a *AVT) evaluate(ctx context.Context, node helium.Node) (string, error) {
	if a == nil {
		return "", nil
	}

	// If the context carries an execContext, use its xpath context for variables/functions
	xpathCtx := ctx
	if ec := getExecContext(ctx); ec != nil {
		xpathCtx = ec.newXPathContext(node)
	}

	var sb strings.Builder
	for _, p := range a.parts {
		if p.expr != nil {
			result, err := p.expr.Evaluate(xpathCtx, node)
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
func stringifyResult(r *xpath3.Result) string {
	return stringifySequenceWithSep(r.Sequence(), " ")
}

func stringifyResultWithSep(r *xpath3.Result, sep string) string {
	return stringifySequenceWithSep(r.Sequence(), sep)
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
	if len(seq) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, item := range seq {
		if i > 0 {
			sb.WriteString(sep)
		}
		av, err := xpath3.AtomizeItem(item)
		if err != nil {
			continue
		}
		s, err := xpath3.AtomicToString(av)
		if err != nil {
			continue
		}
		sb.WriteString(s)
	}
	return sb.String()
}

// flattenArraysInSequence recursively replaces any ArrayItem in the
// sequence with its flattened members.  Non-array items pass through
// unchanged.  This implements the XSLT 3.0 rule that arrays appearing
// in sequence constructors, apply-templates select, value-of, etc.
// are expanded before further processing.
func flattenArraysInSequence(seq xpath3.Sequence) xpath3.Sequence {
	hasArray := false
	for _, item := range seq {
		if _, ok := item.(xpath3.ArrayItem); ok {
			hasArray = true
			break
		}
	}
	if !hasArray {
		return seq
	}
	var result xpath3.Sequence
	for _, item := range seq {
		if arr, ok := item.(xpath3.ArrayItem); ok {
			result = append(result, arr.Flatten()...)
		} else {
			result = append(result, item)
		}
	}
	return result
}
