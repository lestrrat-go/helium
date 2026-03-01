package xpointer

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath"
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
func Evaluate(doc *helium.Document, expr string) ([]helium.Node, error) {
	parts, err := parseParts(expr)
	if err != nil {
		return nil, err
	}

	// Evaluate parts left-to-right with cascading fallback.
	// xmlns() parts accumulate namespace bindings; other parts are
	// tried in order and the first non-empty result is returned.
	var nsMap map[string]string
	var lastErr error
	for _, p := range parts {
		if p.scheme == "xmlns" {
			prefix, uri, ok := parseXmlnsBody(p.body)
			if !ok {
				return nil, fmt.Errorf("xpointer: invalid xmlns() body %q", p.body)
			}
			if nsMap == nil {
				nsMap = make(map[string]string)
			}
			nsMap[prefix] = uri
			continue
		}

		nodes, err := evaluatePart(doc, p, nsMap)
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

	// All parts exhausted with no non-empty result.
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, nil
}

// evaluatePart evaluates a single non-xmlns XPointer part.
func evaluatePart(doc *helium.Document, p xptrPart, nsMap map[string]string) ([]helium.Node, error) {
	switch p.scheme {
	case "xpointer", "xpath1":
		if len(nsMap) > 0 {
			return findWithContext(doc, p.body, nsMap)
		}
		return xpath.Find(doc, p.body)
	case "element":
		return evaluateElement(doc, p.body)
	case "": // shorthand pointer (bare name = ID)
		elem := doc.GetElementByID(p.body)
		if elem == nil {
			return nil, nil
		}
		return []helium.Node{elem}, nil
	default:
		return nil, fmt.Errorf("%w: %s", errUnknownScheme, p.scheme)
	}
}

// findWithContext compiles an XPath expression and evaluates it with
// namespace bindings, returning a node-set.
func findWithContext(node helium.Node, expr string, nsMap map[string]string) ([]helium.Node, error) {
	xctx := &xpath.Context{Namespaces: nsMap}
	r, err := xpath.EvaluateWithContext(node, expr, xctx)
	if err != nil {
		return nil, err
	}
	if r.Type != xpath.NodeSetResult {
		return nil, xpath.ErrNotNodeSet
	}
	return r.NodeSet, nil
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
	idx := strings.IndexByte(expr, '(')
	if idx < 0 {
		// No parenthesis: shorthand pointer (bare name)
		return "", expr, "", nil
	}

	scheme = expr[:idx]
	rest := expr[idx+1:]

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
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
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
