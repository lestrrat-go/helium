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
	"github.com/lestrrat-go/helium/internal/xmlchar"
	"github.com/lestrrat-go/helium/xpath1"
)

// errUnknownScheme is returned by evaluatePart for unrecognized XPointer
// schemes. The cascade continues past unknown-scheme errors but aborts on
// syntax errors from known schemes, matching libxml2 behavior.
var errUnknownScheme = errors.New("xpointer: unknown scheme")

// ErrNilExpression is returned by [Expression.Evaluate] when the receiver is a
// nil or uncompiled *Expression.
var ErrNilExpression = errors.New("xpointer: nil or uncompiled expression")

// ErrNilDocument is returned by [Expression.Evaluate] and [Evaluate] when the
// document to evaluate against is nil.
var ErrNilDocument = errors.New("xpointer: nil document")

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
				return nil, fmt.Errorf("xpointer: XPath compilation failed in %s(%s): %w", p.scheme, p.body, cerr)
			}
			compiled[i] = c
		case "xmlns":
			prefix, _, ok := parseXmlnsBody(p.body)
			if !ok {
				return nil, fmt.Errorf("xpointer: invalid xmlns() body %q", p.body)
			}
			if err := validateXmlnsPrefix(prefix); err != nil {
				return nil, err
			}
			// Reserved-prefix / reserved-namespace bindings are accepted here:
			// per the XPointer xmlns() scheme they are no-ops at evaluation
			// time (they leave the binding context unchanged), not errors.
		}
	}
	return &Expression{parts: parts, compiled: compiled}, nil
}

// Evaluate evaluates the compiled XPointer against a document, returning
// the matching nodes. Cascading fallback semantics match [Evaluate].
func (e *Expression) Evaluate(ctx context.Context, doc *helium.Document) ([]helium.Node, error) {
	// A nil receiver or a zero-value Expression (e.g. var e Expression) has
	// never been through Compile. A successfully compiled expression always
	// holds at least one part because parseParts rejects the empty expression,
	// so an empty parts slice reliably marks an uncompiled value.
	if e == nil || len(e.parts) == 0 {
		return nil, ErrNilExpression
	}
	if doc == nil {
		return nil, ErrNilDocument
	}

	var nsMap map[string]string
	var lastErr error
	for i, p := range e.parts {
		if p.scheme == "xmlns" {
			prefix, uri, _ := parseXmlnsBody(p.body) // validated in Compile
			// Reserved-prefix and reserved-namespace bindings are no-ops per
			// the XPointer xmlns() scheme: leave the binding context unchanged.
			if isXmlnsNoOp(prefix, uri) {
				continue
			}
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
			// evaluateElement validates the leading NCName and every index.
			return evaluateElement(doc, p.body)
		}
		// A shorthand pointer must be a valid XML NCName per the XPointer
		// framework. Reject malformed names (invalid NCName chars, invalid
		// UTF-8) as syntax errors rather than silently resolving to no node,
		// which would let XInclude unlink the include node.
		if !xmlchar.IsValidNCName(p.body) {
			return nil, fmt.Errorf("xpointer: invalid shorthand pointer %q (not an NCName)", p.body)
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
	// Check the nil document before Compile so that a nil document is reported
	// as ErrNilDocument (as documented) even when expr would also fail to
	// compile. Expression.Evaluate keeps its own nil-doc check for the
	// pre-compiled path.
	if doc == nil {
		return nil, ErrNilDocument
	}
	e, err := Compile(expr)
	if err != nil {
		return nil, err
	}
	return e.Evaluate(ctx, doc)
}

// validateXmlnsPrefix rejects xmlns() scheme prefixes that are syntactically
// malformed. The XPointer xmlns() scheme binds a NamespacePrefix (an XML
// NCName) to a namespace URI; a prefix that is not a valid NCName is malformed
// and is a compile-time error.
//
// The reserved prefixes "xml" and "xmlns" are NOT rejected here: per the
// XPointer xmlns() scheme, attempting to (re)bind them is a no-op that leaves
// the namespace binding context unchanged, not an error. The no-op is applied
// during evaluation (see isXmlnsNoOp).
func validateXmlnsPrefix(prefix string) error {
	if !xmlchar.IsValidNCName(prefix) {
		return fmt.Errorf("xpointer: invalid xmlns() prefix %q (not an NCName)", prefix)
	}
	return nil
}

// Reserved namespace URIs that an xmlns() scheme part may not bind a prefix to.
// Per the XPointer xmlns() scheme, attempting to bind any prefix to one of
// these is a no-op (the binding context is left unchanged), not an error.
const (
	xmlNamespaceURI   = "http://www.w3.org/XML/1998/namespace"
	xmlnsNamespaceURI = "http://www.w3.org/2000/xmlns/"
)

// isXmlnsNoOp reports whether an xmlns(prefix=uri) binding must be ignored
// (left as a no-op) per the XPointer xmlns() scheme. The reserved prefixes
// "xml" and "xmlns" may not be (re)bound, and no prefix may be bound to the XML
// or xmlns namespace URIs.
func isXmlnsNoOp(prefix, uri string) bool {
	if prefix == "xml" || prefix == "xmlns" {
		return true
	}
	return uri == xmlNamespaceURI || uri == xmlnsNamespaceURI
}

// xmlSpaceCutset is the set of XML S (whitespace) characters per the XML 1.0
// production S ::= (#x20 | #x9 | #xD | #xA)+. Note this is deliberately NOT the
// Unicode whitespace set used by strings.TrimSpace.
const xmlSpaceCutset = " \t\r\n"

// parseXmlnsBody parses an xmlns() scheme body per the W3C XPointer xmlns()
// grammar:
//
//	XmlnsSchemeData ::= NCName S? '=' S? EscapedNamespaceName
//
// where the escaped namespace name is EscapedData* and S ::= (#x20 | #x9 | #xD |
// #xA)+. Optional XML whitespace is allowed after the NCName prefix and after the
// '='; this function trims that surrounding whitespace from the prefix (right
// side) and the URI (left side) before returning. The prefix is returned
// unvalidated — callers validate it as an NCName via validateXmlnsPrefix. The URI
// is the remaining namespace name with leading S removed but otherwise preserved
// exactly (no internal or trailing trimming); circumflex-escaping in the body has
// already been undone by parseScheme/unescapeXPointer before this function sees it.
//
// Leading whitespace before the NCName is intentionally NOT trimmed: the data
// begins with the NCName, so a leading space leaves a non-NCName prefix that
// validateXmlnsPrefix rejects as malformed.
//
// ok is false when there is no '=' or the prefix portion is empty after
// trimming surrounding whitespace.
func parseXmlnsBody(body string) (prefix, uri string, ok bool) {
	rawPrefix, rawURI, found := strings.Cut(body, "=")
	if !found {
		return "", "", false
	}
	// XmlnsSchemeData ::= NCName S? '=' S? EscapedNamespaceName — the data
	// starts with the NCName, so only whitespace AFTER the prefix (and after
	// '=') is permitted. Trim the prefix on the right only; any leading
	// whitespace stays attached so the NCName validation rejects it.
	prefix = strings.TrimRight(rawPrefix, xmlSpaceCutset)
	if prefix == "" {
		return "", "", false
	}
	uri = strings.TrimLeft(rawURI, xmlSpaceCutset)
	return prefix, uri, true
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

		// The XPointer framework defines SchemeName as a QName. A non-empty
		// scheme name that is not a valid QName is a syntax error, NOT an
		// unknown scheme: rejecting it here prevents a malformed scheme (e.g.
		// "1bad(/x)") from being skipped via the unknown-scheme cascade and
		// letting a later well-formed part succeed, which would bypass the
		// trailing-text rejection below. A syntactically valid but unsupported
		// scheme (e.g. "foo(...)") is still a valid QName and continues to
		// cascade as an unknown scheme during evaluation.
		if scheme != "" && !xmlchar.IsValidQName(scheme) {
			return nil, fmt.Errorf("xpointer: invalid scheme name %q (not a QName)", scheme)
		}

		// A non-scheme trailing token is only valid as the entire pointer on
		// its own. Once scheme-based parsing has started, every remaining part
		// must be a scheme(...) part. Any trailing non-scheme text — a barename
		// (e.g. "foo") OR a child-sequence (e.g. "/1" or "r/1") — appended after
		// a scheme part is malformed and must not be silently ignored, since an
		// ignored-but-malformed pointer would let XInclude unlink the include
		// node instead of reporting the error.
		//
		// The single tolerated exception is a lone unbalanced ")" left over from
		// a scheme body (e.g. "xpointer(/t1))"), which libxml2 accepts and the
		// xinclude coalesce.xml golden test relies on.
		if scheme == "" && len(parts) > 0 {
			if body == ")" {
				break
			}
			return nil, fmt.Errorf("xpointer: trailing non-scheme text %q is not allowed after scheme-based parts", body)
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

	// Validate the ENTIRE body syntactically BEFORE any document lookup, so that
	// a malformed pointer is reported as an error regardless of whether the
	// initial ID exists. A silent empty result would let XInclude unlink the
	// include node instead of surfacing the syntax error.
	//
	// A non-empty leading token must be a valid NCName (the shorthand ID); an
	// empty leading token denotes an absolute "/..." child sequence. Only the
	// very first segment may be empty (the leading "/" of an absolute sequence
	// like "/1/2"). Every child-sequence segment after the first must be a
	// non-empty 1-based integer index matching [1-9][0-9]* (no sign, no leading
	// zero). An empty segment anywhere else is a trailing/doubled slash and is a
	// syntax error.
	if parts[0] != "" && !xmlchar.IsValidNCName(parts[0]) {
		return nil, fmt.Errorf("xpointer: invalid element() id %q (not an NCName)", parts[0])
	}
	childIndexes := make([]int, 0, len(parts)-1)
	for _, part := range parts[1:] {
		if part == "" {
			return nil, fmt.Errorf("xpointer: empty child-sequence segment in element() scheme (trailing or doubled %q)", "/")
		}
		if !isChildIndex(part) {
			return nil, fmt.Errorf("xpointer: invalid child index %q in element() scheme (must match [1-9][0-9]*)", part)
		}
		idx, err := childIndexValue(part)
		if err != nil {
			return nil, err
		}
		childIndexes = append(childIndexes, idx)
	}

	var cur helium.Node = doc

	if parts[0] != "" {
		// Starts with an NCName — look up element by ID.
		elem := doc.GetElementByID(parts[0])
		if elem == nil {
			return nil, nil
		}
		cur = elem
	}

	for _, childIdx := range childIndexes {
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

// isChildIndex reports whether s is a valid XPointer element() child-sequence
// index: it must match [1-9][0-9]* exactly. That rejects the empty string, a
// leading sign ("+1", "-1"), a leading zero ("01"), zero itself ("0"), and any
// non-digit lexeme, all of which Atoi would otherwise silently accept or coerce.
func isChildIndex(s string) bool {
	if s == "" {
		return false
	}
	if s[0] < '1' || s[0] > '9' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// childIndexValue converts a child-sequence index already validated by
// isChildIndex into an int. isChildIndex guarantees the lexical form, but an
// arbitrarily long digit string can still exceed the platform int range (e.g.
// "18446744073709551617"). strconv.Atoi reports such a value as a range error,
// which we surface as a syntax error rather than letting the index wrap around
// and silently select the wrong node.
func childIndexValue(s string) (int, error) {
	idx, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("xpointer: child index %q in element() scheme is out of range", s)
	}
	return idx, nil
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
