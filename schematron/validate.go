package schematron

import (
	"context"
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

func validateDocument(ctx context.Context, doc *helium.Document, schema *Schema, cfg *validateConfig, handler helium.ErrorHandler) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	filename := cfg.label
	if filename == "" {
		filename = doc.URL()
	}
	if filename == "" {
		filename = "(string)"
	}
	valid := true
	paths := make(nodePathCache)
	_, discardDiagnostics := handler.(helium.NilErrorHandler)

	ev := schema.engine.runner(schema.namespaces)

	for _, pat := range schema.patterns {
		// ISO Schematron: within a pattern, each node is processed by only
		// the FIRST rule whose context matches it. Track nodes already
		// claimed by an earlier rule in this pattern and skip them in later
		// rules. The set is reset for each pattern.
		matched := make(map[helium.Node]struct{})
		for _, r := range pat.rules {
			if err := ctx.Err(); err != nil {
				return err
			}
			result, err := ev.evaluate(ctx, r.contextExpr, doc)
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			if err != nil {
				// A rule context that cannot be evaluated means none of
				// the rule's assertions can be checked. Treat this as a
				// validation failure and surface the error. Silently
				// skipping the rule would let a broken schema report a
				// false "valid" result.
				valid = false
				handler.Handle(ctx, helium.NewLeveledError(fmt.Sprintf("XPath error : %s\n", formatXPathError(err)), helium.ErrorLevelError))
				continue
			}
			// A rule context that selects anything other than nodes claims
			// no node at all. Every XPath node kind is a valid rule context.
			for _, node := range result.nodeSet() {
				if err := ctx.Err(); err != nil {
					return err
				}
				// Skip nodes already matched by an earlier rule in this
				// pattern (first-match-only semantics).
				if _, ok := matched[node]; ok {
					continue
				}
				matched[node] = struct{}{}

				// Set up let variables for this rule.
				// Variables are accumulated so that each let can
				// reference previously registered variables, matching
				// libxml2's xmlSchematronRegisterVariables behavior.
				ruleEv := ev
				for _, lb := range r.lets {
					if err := ctx.Err(); err != nil {
						return err
					}
					letResult, err := ruleEv.evaluate(ctx, lb.expr, node)
					if ctxErr := ctx.Err(); ctxErr != nil {
						return ctxErr
					}
					if err != nil {
						// A let whose expression cannot be evaluated must
						// not be silently dropped: a later let or test that
						// references it would then break in confusing ways.
						// Treat this as a validation failure and surface the
						// error and never swallow it -- otherwise an
						// otherwise-passing rule with a broken let would make
						// Validate report a false "valid" result (and with a
						// nil handler the error would be invisible).
						valid = false
						handler.Handle(ctx, helium.NewLeveledError(fmt.Sprintf("XPath error : %s\n", formatXPathError(err)), helium.ErrorLevelError))
						continue
					}
					ruleEv = ruleEv.bind(lb.name, letResult)
				}

				for _, t := range r.tests {
					if err := ctx.Err(); err != nil {
						return err
					}
					boolVal, err := ruleEv.test(ctx, t.compiled, node)
					if ctxErr := ctx.Err(); ctxErr != nil {
						return ctxErr
					}

					// A test XPath that cannot be evaluated, or whose result
					// has no effective boolean value, must not be treated as
					// satisfied. Surface the error and treat the test as
					// false: an assert (fires when false) then fails, while
					// a report (fires when true) stays silent — mirroring
					// libxml2's xmlSchematronRunTest, which returns 0
					// (false) on evaluation failure.
					if err != nil {
						valid = false
						boolVal = false
						handler.Handle(ctx, helium.NewLeveledError(fmt.Sprintf("XPath error : %s\n", formatXPathError(err)), helium.ErrorLevelError))
					}
					if err := ctx.Err(); err != nil {
						return err
					}

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
						msg, xpathErr := formatMessage(ctx, ruleEv, t.message, node)
						if err := ctx.Err(); err != nil {
							return err
						}
						if xpathErr != "" {
							handler.Handle(ctx, helium.NewLeveledError(xpathErr+"\n", helium.ErrorLevelError))
						}
						if !cfg.quiet && !discardDiagnostics {
							path, err := paths.path(ctx, node)
							if err != nil {
								return err
							}
							ve := ValidationError{
								Filename: filename,
								Line:     node.Line(),
								Element:  node.Name(),
								Path:     path,
								Message:  msg,
							}
							handler.Handle(ctx, &ve)
						}
					}
				}
			}
		}
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	if !valid {
		return ErrValidationFailed
	}
	return nil
}

// formatMessage interpolates message parts against a context node.
// If a value-of evaluation fails, it emits an XPath error to out and
// stops processing further parts (matching libxml2 behavior).
//
// Whitespace normalization matches libxml2's xmlSchematronFormatReport:
// after each segment (text, name, value-of), if the accumulated buffer
// ends with whitespace, all trailing whitespace is replaced with a
// single space. Internal whitespace within segments is preserved.
func formatMessage(ctx context.Context, ev runner, parts []messagePart, node helium.Node) (string, string) {
	var buf []byte
	for _, part := range parts {
		if ctx.Err() != nil {
			return string(buf), ""
		}
		switch p := part.(type) {
		case textPart:
			buf = append(buf, p.text...)
		case namePart:
			if p.expr != nil {
				result, err := ev.evaluate(ctx, p.expr, node)
				if err == nil {
					buf = append(buf, result.nodeName()...)
				}
			}
		case valueOfPart:
			if p.expr == nil {
				// The select expression failed to compile; the error was
				// already reported through the handler at compile time, so
				// stop here, emitting no bogus value.
				return string(buf), ""
			}
			result, err := ev.evaluate(ctx, p.expr, node)
			if err != nil {
				// Runtime XPath error — report alongside the validation error.
				return string(buf), fmt.Sprintf("XPath error : %s", formatXPathError(err))
			}
			buf = append(buf, result.stringValue()...)
		}
		buf = trimTrailingWS(buf)
	}
	return string(buf), ""
}

// formatXPathError converts XPath error messages to libxml2-compatible format.
func formatXPathError(err error) string {
	msg := err.Error()
	if after, ok := strings.CutPrefix(msg, "xpath: unknown function: "); ok {
		return "Unregistered function: " + after
	}
	if after, ok := strings.CutPrefix(msg, "unknown function: "); ok {
		return "Unregistered function: " + after
	}
	return msg
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

// nodePathCache indexes each sibling list once per validation run.
// Its keys and positions cannot outlive the run because the document may change.
type nodePathCache map[helium.Node]int

// path returns the XPath path to a node (equivalent to libxml2's xmlGetNodePath).
// For elements: /root/parent/child[N] where [N] is added only when siblings share the name.
func (paths nodePathCache) path(ctx context.Context, n helium.Node) (string, error) {
	if n == nil {
		return "", nil
	}

	// Attribute leaves are not part of the element ancestry walk below
	// (which only collects element nodes). Compute the owning element's
	// path and append /@<name>, matching libxml2's xmlGetNodePath
	// (e.g. /root/@id).
	if n.Type() == helium.AttributeNode {
		parent, err := paths.path(ctx, n.Parent())
		if err != nil {
			return "", err
		}
		if parent == "/" {
			parent = ""
		}
		return parent + "/@" + n.Name(), nil
	}

	var parts []string
	for cur := n; cur != nil; cur = cur.Parent() {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if cur.Type() == helium.DocumentNode {
			break
		}
		if cur.Type() != helium.ElementNode {
			continue
		}
		name := cur.Name()
		pos, err := paths.siblingPosition(ctx, cur)
		if err != nil {
			return "", err
		}
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

	return "/" + strings.Join(parts, "/"), nil
}

// siblingPosition returns the 1-based position among same-named siblings,
// or 0 if the element is the only one with that name among its siblings.
func (paths nodePathCache) siblingPosition(ctx context.Context, n helium.Node) (int, error) {
	if pos, ok := paths[n]; ok {
		return pos, nil
	}
	parent := n.Parent()
	if parent == nil {
		return 0, nil
	}

	counts := make(map[string]int)
	for sib := range helium.Children(parent) {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		if sib.Type() == helium.ElementNode {
			counts[sib.Name()]++
			paths[sib] = counts[sib.Name()]
		}
	}

	for sib := range helium.Children(parent) {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		if sib.Type() == helium.ElementNode && counts[sib.Name()] == 1 {
			paths[sib] = 0
		}
	}
	return paths[n], nil
}
