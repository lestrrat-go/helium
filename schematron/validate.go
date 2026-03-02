package schematron

import (
	"fmt"
	"math"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath"
)

func validateDocument(doc *helium.Document, schema *Schema, cfg *validateConfig) (string, bool) {
	filename := cfg.filename
	var out strings.Builder
	valid := true

	xctx := &xpath.Context{
		Namespaces: schema.namespaces,
	}

	for _, pat := range schema.patterns {
		for _, r := range pat.rules {
			result, err := r.contextExpr.EvaluateWith(doc, xctx)
			if err != nil {
				continue
			}
			if result.Type != xpath.NodeSetResult {
				continue
			}

			for _, node := range result.NodeSet {
				if node.Type() != helium.ElementNode {
					continue
				}

				// Set up let variables for this rule.
				// Variables are accumulated so that each let can
				// reference previously registered variables, matching
				// libxml2's xmlSchematronRegisterVariables behavior.
				ruleCtx := xctx
				if len(r.lets) > 0 {
					vars := make(map[string]any)
					for k, v := range xctx.Variables {
						vars[k] = v
					}
					ruleCtx = &xpath.Context{
						Namespaces: xctx.Namespaces,
						Variables:  vars,
					}
					for _, lb := range r.lets {
						letResult, err := lb.expr.EvaluateWith(node, ruleCtx)
						if err == nil {
							vars[lb.name] = xpathResultToValue(letResult)
						}
					}
				}

				for _, t := range r.tests {
					testResult, err := t.compiled.EvaluateWith(node, ruleCtx)
					if err != nil {
						continue
					}

					boolVal := xpathResultToBool(testResult)

					// Assert: fire error when false.
					// Report: fire error when true.
					fire := false
					if t.typ == testAssert && !boolVal {
						fire = true
					} else if t.typ == testReport && boolVal {
						fire = true
					}

					if fire {
						valid = false
						if cfg.errorHandler != nil {
							msg := formatMessage(t.message, node, ruleCtx, &out)
							cfg.errorHandler.HandleError(ValidationError{
								Filename: filename,
								Line:     node.Line(),
								Element:  node.Name(),
								Path:     getNodePath(node),
								Message:  msg,
							})
						} else if !cfg.quiet {
							msg := formatMessage(t.message, node, ruleCtx, &out)
							nodePath := getNodePath(node)
							out.WriteString(schematronError(filename, node.Line(), node.Name(), nodePath, msg))
						}
					}
				}
			}
		}
	}

	if valid {
		out.WriteString(filename + " validates\n")
	} else {
		out.WriteString(filename + " fails to validate\n")
	}
	return out.String(), valid
}

// formatMessage interpolates message parts against a context node.
// If a value-of evaluation fails, it emits an XPath error to out and
// stops processing further parts (matching libxml2 behavior).
//
// Whitespace normalization matches libxml2's xmlSchematronFormatReport:
// after each segment (text, name, value-of), if the accumulated buffer
// ends with whitespace, all trailing whitespace is replaced with a
// single space. Internal whitespace within segments is preserved.
func formatMessage(parts []messagePart, node helium.Node, xctx *xpath.Context, out *strings.Builder) string {
	var buf []byte
	for _, part := range parts {
		switch p := part.(type) {
		case textPart:
			buf = append(buf, p.text...)
		case namePart:
			if p.expr != nil {
				result, err := p.expr.EvaluateWith(node, xctx)
				if err == nil {
					buf = append(buf, xpathResultToName(result)...)
				}
			}
		case valueOfPart:
			if p.expr == nil {
				// Compile-time error — should not happen (caught during compilation).
				return string(buf)
			}
			result, err := p.expr.EvaluateWith(node, xctx)
			if err != nil {
				// Runtime XPath error — emit error line and stop processing.
				fmt.Fprintf(out, "XPath error : %s\n", formatXPathError(err))
				return string(buf)
			}
			buf = append(buf, xpathResultToString(result)...)
		}
		buf = trimTrailingWS(buf)
	}
	return string(buf)
}

// trimTrailingWS replaces trailing whitespace in buf with a single space.
// Matches libxml2's per-segment whitespace normalization in
// xmlSchematronFormatReport (schematron.c:1515-1533).
func trimTrailingWS(buf []byte) []byte {
	if len(buf) == 0 {
		return buf
	}
	c := buf[len(buf)-1]
	if c != ' ' && c != '\n' && c != '\r' && c != '\t' {
		return buf
	}
	end := len(buf)
	for end > 0 {
		c = buf[end-1]
		if c != ' ' && c != '\n' && c != '\r' && c != '\t' {
			break
		}
		end--
	}
	buf = buf[:end]
	return append(buf, ' ')
}

// formatXPathError converts XPath error messages to libxml2-compatible format.
func formatXPathError(err error) string {
	msg := err.Error()
	// Map helium xpath error messages to libxml2 format.
	if strings.HasPrefix(msg, "xpath: unknown function: ") {
		return "Unregistered function: " + strings.TrimPrefix(msg, "xpath: unknown function: ")
	}
	if strings.HasPrefix(msg, "unknown function: ") {
		return "Unregistered function: " + strings.TrimPrefix(msg, "unknown function: ")
	}
	return msg
}

// xpathResultToBool converts an XPath result to a boolean.
func xpathResultToBool(r *xpath.Result) bool {
	switch r.Type {
	case xpath.BooleanResult:
		return r.Boolean
	case xpath.NumberResult:
		return r.Number != 0 && !math.IsNaN(r.Number)
	case xpath.StringResult:
		return r.String != ""
	case xpath.NodeSetResult:
		return len(r.NodeSet) > 0
	}
	return false
}

// xpathResultToString converts an XPath result to a string.
func xpathResultToString(r *xpath.Result) string {
	switch r.Type {
	case xpath.StringResult:
		return r.String
	case xpath.NumberResult:
		if r.Number == math.Trunc(r.Number) && !math.IsInf(r.Number, 0) && !math.IsNaN(r.Number) {
			return fmt.Sprintf("%g", r.Number)
		}
		return fmt.Sprintf("%g", r.Number)
	case xpath.BooleanResult:
		if r.Boolean {
			return "True"
		}
		return "False"
	case xpath.NodeSetResult:
		if len(r.NodeSet) == 0 {
			return ""
		}
		var sb strings.Builder
		for i, n := range r.NodeSet {
			if i > 0 {
				sb.WriteByte(' ')
			}
			sb.WriteString(n.Name())
		}
		return sb.String()
	}
	return ""
}

// xpathResultToName extracts a node name from an XPath result.
// Only returns a name for element and attribute nodes (matching libxml2 behavior).
func xpathResultToName(r *xpath.Result) string {
	if r.Type == xpath.NodeSetResult && len(r.NodeSet) > 0 {
		n := r.NodeSet[0]
		if n.Type() == helium.ElementNode {
			return n.Name()
		}
		// Use type assertion for attributes since Attribute.Type() may not be set correctly.
		if attr, ok := n.(*helium.Attribute); ok {
			return attr.Name()
		}
	}
	return ""
}

// xpathResultToValue converts an XPath result to a value suitable for variable binding.
func xpathResultToValue(r *xpath.Result) any {
	switch r.Type {
	case xpath.NodeSetResult:
		return r.NodeSet
	case xpath.StringResult:
		return r.String
	case xpath.NumberResult:
		return r.Number
	case xpath.BooleanResult:
		return r.Boolean
	}
	return nil
}

// getNodePath returns the XPath path to a node (equivalent to libxml2's xmlGetNodePath).
// For elements: /root/parent/child[N] where [N] is added only when siblings share the name.
func getNodePath(n helium.Node) string {
	if n == nil {
		return ""
	}

	var parts []string
	for cur := n; cur != nil; cur = cur.Parent() {
		if cur.Type() == helium.DocumentNode {
			break
		}
		if cur.Type() != helium.ElementNode {
			continue
		}
		name := cur.Name()
		pos := siblingPosition(cur)
		if pos > 0 {
			parts = append(parts, fmt.Sprintf("%s[%d]", name, pos))
		} else {
			parts = append(parts, name)
		}
	}

	// Reverse.
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}

	return "/" + strings.Join(parts, "/")
}

// siblingPosition returns the 1-based position among same-named siblings,
// or 0 if the element is the only one with that name among its siblings.
func siblingPosition(n helium.Node) int {
	name := n.Name()
	parent := n.Parent()
	if parent == nil {
		return 0
	}

	count := 0
	for sib := parent.FirstChild(); sib != nil; sib = sib.NextSibling() {
		if sib.Type() == helium.ElementNode && sib.Name() == name {
			count++
		}
	}

	if count <= 1 {
		return 0 // unique name, no position needed
	}

	// Count position.
	pos := 0
	for sib := parent.FirstChild(); sib != nil; sib = sib.NextSibling() {
		if sib.Type() == helium.ElementNode && sib.Name() == name {
			pos++
			if sib == n {
				return pos
			}
		}
	}
	return 0
}

