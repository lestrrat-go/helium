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
			// find matching }
			end := strings.IndexByte(s[i+1:], '}')
			if end < 0 {
				return nil, staticError(errCodeXTSE0580, "unterminated AVT expression in %q", s)
			}
			exprStr := s[i+1 : i+1+end]
			expr, err := compileXPath(exprStr, nsBindings)
			if err != nil {
				return nil, staticError(errCodeXTSE0580, "invalid XPath in AVT: %v", err)
			}
			parts = append(parts, avtPart{expr: expr})
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

func appendLiteral(parts []avtPart, s string) []avtPart {
	if len(parts) > 0 && parts[len(parts)-1].expr == nil {
		parts[len(parts)-1].literal += s
		return parts
	}
	return append(parts, avtPart{literal: s})
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
				return "", dynamicError(errCodeXTDE0045, "AVT evaluation error: %v", err)
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

func stringifySequenceWithSep(seq xpath3.Sequence, sep string) string {
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
