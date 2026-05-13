// Package xpointer implements XPointer framework and element() scheme resolution
// (libxml2: xpointer module / xmlXPtrEval).
package xpointer

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath1"
)

// errUnknownScheme is returned by evaluatePart for unrecognized XPointer
// schemes. The cascade continues past unknown-scheme errors but aborts on
// syntax errors from known schemes, matching libxml2 behavior.
var errUnknownScheme = errors.New("xpointer: unknown scheme")

// xptrPart represents a single parsed XPointer scheme(body) part.
type xptrPart struct {
	scheme string
	body   string
}

// Expression is a parsed XPointer expression with any embedded XPath
// fragments already compiled. Reuse an Expression across multiple Evaluate
// calls to amortize parsing and XPath compilation when the same XPointer
// is applied to many documents (e.g. an XInclude that references the same
// xpointer="..." against repeated includes).
//
// Expression is safe for concurrent use by multiple goroutines.
type Expression struct {
	parts []xptrPart
	// compiled[i] holds the pre-compiled XPath for parts[i] when the part's
	// scheme is xpointer or xpath1; nil for other schemes. Slot indices
	// match parts indices 1:1.
	compiled []*xpath1.Expression
}

// Compile parses an XPointer expression and pre-compiles any XPath
// fragments inside xpointer() or xpath1() schemes. XPath syntax errors
// and malformed xmlns() bodies are reported here. element() and
// shorthand child-sequence bodies are validated lazily during
// [Expression.Evaluate] (e.g. non-integer child indices). Use
// [Expression.Evaluate] to apply the compiled expression to one or
// more documents.
func Compile(expr string) (*Expression, error) {
	parts, err := parseParts(expr)
	if err != nil {
		return nil, err
	}
	compiled := make([]*xpath1.Expression, len(parts))
	for i, p := range parts {
		switch p.scheme {
		case "xpointer", "xpath1":
			c, cerr := xpath1.Compile(p.body)
			if cerr != nil {
				return nil, fmt.Errorf("xpointer: XPath evaluation failed: %w", cerr)
			}
			compiled[i] = c
		case "xmlns":
			if _, _, ok := parseXmlnsBody(p.body); !ok {
				return nil, fmt.Errorf("xpointer: invalid xmlns() body %q", p.body)
			}
		}
	}
	return &Expression{parts: parts, compiled: compiled}, nil
}

// Evaluate evaluates the compiled XPointer against a document, returning
// the matching nodes. Cascading fallback semantics match [Evaluate].
func (e *Expression) Evaluate(ctx context.Context, doc *helium.Document) ([]helium.Node, error) {
	var nsMap map[string]string
	var lastErr error
	for i, p := range e.parts {
		if p.scheme == "xmlns" {
			prefix, uri, _ := parseXmlnsBody(p.body) // validated in Compile
			if nsMap == nil {
				nsMap = make(map[string]string)
			}
			nsMap[prefix] = uri
			continue
		}

		nodes, err := e.evaluatePart(ctx, doc, p, i, nsMap)
		if err != nil {
			// Unknown schemes allow cascade to continue (try next part).
			// Syntax errors from known schemes abort immediately.
			if !errors.Is(err, errUnknownScheme) {
				return nil, err
			}
			lastErr = err
			continue
		}
		if len(nodes) > 0 {
			return nodes, nil
		}
		// Empty result — try next part.
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, nil
}

// evaluatePart evaluates a single non-xmlns XPointer part using the
// pre-compiled state in e.
func (e *Expression) evaluatePart(ctx context.Context, doc *helium.Document, p xptrPart, partIdx int, nsMap map[string]string) ([]helium.Node, error) {
	switch p.scheme {
	case "xpointer", "xpath1":
		ev := xpath1.NewEvaluator()
		if len(nsMap) > 0 {
			ev = ev.AdditionalNamespaces(nsMap)
		}
		r, err := ev.Evaluate(ctx, e.compiled[partIdx], doc)
		if err != nil {
			return nil, fmt.Errorf("xpointer: XPath evaluation failed: %w", err)
		}
		if r.Type != xpath1.NodeSetResult {
			return nil, xpath1.ErrNotNodeSet
		}
		return r.NodeSet, nil
	case "element":
		return evaluateElement(doc, p.body)
	case "": // shorthand pointer or bare child sequence
		if strings.ContainsRune(p.body, '/') {
			// Bare child sequence (/1/2/3) or name+child sequence (name/1/2).
			// libxml2 supports these at top level without element() wrapper.
			return evaluateElement(doc, p.body)
		}
		elem := doc.GetElementByID(p.body)
		if elem == nil {
			return nil, nil
		}
		return []helium.Node{elem}, nil
	default:
		return nil, fmt.Errorf("%w: %s", errUnknownScheme, p.scheme)
	}
}

// Evaluate evaluates an XPointer expression against a document and returns
// the matching nodes. It supports:
//   - xpointer(expr) / xpath1(expr) scheme: delegates to XPath
//   - xmlns(prefix=uri) scheme: registers namespace bindings for subsequent parts
//   - element(/1/2/3) scheme: child-sequence navigation
//   - shorthand pointer: looks up by ID via Document.GetElementByID
//
// Multiple scheme parts are evaluated left-to-right with cascading fallback:
// the first part that produces a non-empty result wins. xmlns() parts
// accumulate namespace bindings for all subsequent parts.
//
// Evaluate is a one-shot convenience wrapper around [Compile] +
// [Expression.Evaluate]. When the same XPointer is reused across documents,
// call Compile once and reuse the resulting [*Expression].
func Evaluate(ctx context.Context, doc *helium.Document, expr string) ([]helium.Node, error) {
	e, err := Compile(expr)
	if err != nil {
		return nil, err
	}
	return e.Evaluate(ctx, doc)
}

// parseXmlnsBody parses "prefix=uri" from an xmlns() body.
func parseXmlnsBody(body string) (prefix, uri string, ok bool) {
	i := strings.IndexByte(body, '=')
	if i < 1 {
		return "", "", false
	}
	return body[:i], body[i+1:], true
}

// ParseFragmentID splits a URI fragment into its XPointer scheme and body.
// For a bare name like "foo", it returns scheme="" and body="foo".
// For "xpointer(//p)", it returns scheme="xpointer" and body="//p".
func ParseFragmentID(fragment string) (scheme, body string, err error) {
	s, b, _, err := parseScheme(fragment)
	return s, b, err
}

// parseParts parses an XPointer expression into a sequence of scheme(body) parts.
// Handles multiple consecutive parts like "xmlns(...)xpath1(...)".
func parseParts(expr string) ([]xptrPart, error) {
	if expr == "" {
		return nil, fmt.Errorf("xpointer: empty expression")
	}

	var parts []xptrPart
	rest := expr
	for rest != "" {
		rest = strings.TrimLeft(rest, " \t\r\n")
		if rest == "" {
			break
		}

		scheme, body, remaining, err := parseScheme(rest)
		if err != nil {
			return nil, err
		}

		parts = append(parts, xptrPart{scheme: scheme, body: body})

		// Bare name (no scheme) consumes everything
		if scheme == "" {
			break
		}
		rest = remaining
	}

	if len(parts) == 0 {
		return nil, fmt.Errorf("xpointer: empty expression")
	}
	return parts, nil
}

// parseScheme parses the first XPointer scheme(body) from expr.
// Returns the scheme, body, remaining string after the closing paren, and any error.
// For bare names (no parenthesis), remaining is empty.
func parseScheme(expr string) (scheme, body, remaining string, err error) {
	if expr == "" {
		return "", "", "", fmt.Errorf("xpointer: empty expression")
	}

	// Look for scheme(body) pattern
	schemePart, rest, found := strings.Cut(expr, "(")
	if !found {
		// No parenthesis: shorthand pointer (bare name)
		return "", expr, "", nil
	}

	scheme = schemePart

	// Find matching closing paren using balanced parenthesis counting.
	// Circumflex (^) escapes the next character per the XPointer framework:
	// ^( and ^) are literal parens, ^^ is a literal circumflex.
	depth := 1
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case '^':
			i++ // skip escaped character
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return scheme, unescapeXPointer(rest[:i]), rest[i+1:], nil
			}
		}
	}

	return "", "", "", fmt.Errorf("xpointer: unbalanced parentheses in %q", expr)
}

// evaluateElement handles the element() scheme.
// Supports element(/1/2/3) child-sequence syntax and element(id/1/2) ID+sequence.
func evaluateElement(doc *helium.Document, body string) ([]helium.Node, error) {
	if body == "" {
		return nil, fmt.Errorf("xpointer: empty element() body")
	}

	parts := strings.Split(body, "/")

	var cur helium.Node = doc
	startIdx := 0

	if parts[0] == "" {
		// Starts with "/" — absolute child sequence from document
		startIdx = 1
	} else {
		// Starts with an NCName — look up element by ID first
		elem := doc.GetElementByID(parts[0])
		if elem == nil {
			return nil, nil
		}
		cur = elem
		startIdx = 1
	}

	for _, part := range parts[startIdx:] {
		if part == "" {
			continue
		}
		childIdx, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("xpointer: invalid child index %q in element() scheme", part)
		}
		cur = nthElementChild(cur, childIdx)
		if cur == nil {
			return nil, nil
		}
	}

	if cur.Type() == helium.DocumentNode {
		return nil, nil
	}
	return []helium.Node{cur}, nil
}

// nthElementChild returns the n-th element child (1-based) of the given node.
// Matches libxml2's xmlXPtrGetNthChild which counts ElementNode, DocumentNode,
// and HTMLDocumentNode.
func nthElementChild(n helium.Node, index int) helium.Node {
	count := 0
	for c := range helium.Children(n) {
		switch c.Type() {
		case helium.ElementNode, helium.DocumentNode, helium.HTMLDocumentNode:
			count++
			if count == index {
				return c
			}
		}
	}
	return nil
}

// unescapeXPointer handles circumflex escaping per the XPointer framework spec:
// ^) -> ), ^( -> (, ^^ -> ^.
func unescapeXPointer(s string) string {
	if !strings.ContainsRune(s, '^') {
		return s
	}
	var sb strings.Builder
	sb.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '^' && i+1 < len(s) {
			next := s[i+1]
			if next == ')' || next == '(' || next == '^' {
				i++
				sb.WriteByte(next)
			} else {
				sb.WriteByte('^')
			}
		} else {
			sb.WriteByte(s[i])
		}
	}
	return sb.String()
}
