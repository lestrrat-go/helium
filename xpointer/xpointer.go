package xpointer

import (
	"fmt"
	"strconv"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath"
)

// Evaluate evaluates an XPointer expression against a document and returns
// the matching nodes. It supports:
//   - xpointer(expr) scheme: delegates to XPath
//   - element(/1/2/3) scheme: child-sequence navigation
//   - shorthand pointer: looks up by ID via Document.GetElementByID
func Evaluate(doc *helium.Document, expr string) ([]helium.Node, error) {
	scheme, body, err := parseScheme(expr)
	if err != nil {
		return nil, err
	}

	switch scheme {
	case "xpointer":
		return xpath.Find(doc, body)
	case "element":
		return evaluateElement(doc, body)
	case "": // shorthand pointer (bare name = ID)
		elem := doc.GetElementByID(body)
		if elem == nil {
			return nil, nil
		}
		return []helium.Node{elem}, nil
	default:
		return nil, fmt.Errorf("xpointer: unsupported scheme %q", scheme)
	}
}

// ParseFragmentID splits a URI fragment into its XPointer scheme and body.
// For a bare name like "foo", it returns scheme="" and body="foo".
// For "xpointer(//p)", it returns scheme="xpointer" and body="//p".
func ParseFragmentID(fragment string) (scheme, body string, err error) {
	return parseScheme(fragment)
}

// parseScheme parses an XPointer expression into scheme and body.
// Uses balanced parenthesis matching per the XPointer framework.
func parseScheme(expr string) (scheme, body string, err error) {
	if expr == "" {
		return "", "", fmt.Errorf("xpointer: empty expression")
	}

	// Look for scheme(body) pattern
	idx := strings.IndexByte(expr, '(')
	if idx < 0 {
		// No parenthesis: shorthand pointer (bare name)
		return "", expr, nil
	}

	scheme = expr[:idx]
	rest := expr[idx+1:]

	// Find matching closing paren using balanced parenthesis counting
	depth := 1
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return scheme, rest[:i], nil
			}
		}
	}

	return "", "", fmt.Errorf("xpointer: unbalanced parentheses in %q", expr)
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
func nthElementChild(n helium.Node, index int) helium.Node {
	count := 0
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			count++
			if count == index {
				return c
			}
		}
	}
	return nil
}
