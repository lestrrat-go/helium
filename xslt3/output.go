package xslt3

import (
	"context"
	"io"
	"strings"

	"github.com/lestrrat-go/helium"
	htmlpkg "github.com/lestrrat-go/helium/html"
	"github.com/lestrrat-go/helium/stream"
	"github.com/lestrrat-go/helium/xpath3"
)

// outputFrame represents the current output target during transformation.
type outputFrame struct {
	doc                 *helium.Document // result document being built
	current             helium.Node      // current insertion point
	captureItems        bool             // when true, xsl:sequence adds to pendingItems instead of DOM
	separateTextNodes   bool             // when true, text nodes are captured as separate string items (prevents DOM merging)
	sequenceMode        bool             // when true, all nodes (text, element, attr, comment, PI) are captured as separate items
	mapConstructor      bool             // when true, xsl:map-entry emits single-entry maps into pendingItems
	pendingItems        xpath3.Sequence  // captured items from xsl:sequence
	prevWasAtomic       bool             // true when last xsl:sequence output was an atomic value (for inter-call space separation)
	wherePopulated      bool             // when true, xsl:document emits document node (not children) so xsl:where-populated can check emptiness
	itemSeparator       *string          // item-separator serialization parameter; nil means default (" " between adjacent atomics)
	outputSerial        int              // monotonically increases whenever visible output is produced
	conditionalScopes   []conditionalScope
}

type conditionalKind int

const (
	conditionalOnEmpty conditionalKind = iota + 1
	conditionalOnNonEmpty
)

type conditionalAction struct {
	ctx         context.Context
	kind        conditionalKind
	content     xpath3.Sequence
	placeholder helium.Node
}

type conditionalScope struct {
	hasOutput bool
	actions   []conditionalAction
}

func (out *outputFrame) noteOutput() {
	out.outputSerial++
}

// SerializeResult writes the result document to a writer according to the
// output definition. If outDef is nil, defaults to XML output.
func SerializeResult(w io.Writer, doc *helium.Document, outDef *OutputDef) error {
	var charMap map[rune]string
	if outDef != nil {
		charMap = outDef.ResolvedCharMap
	}
	return serializeResult(w, doc, outDef, charMap)
}

func serializeResult(w io.Writer, doc *helium.Document, outDef *OutputDef, charMaps ...map[rune]string) error {
	if outDef == nil {
		outDef = defaultOutputDef()
	}

	var charMap map[rune]string
	if len(charMaps) > 0 {
		charMap = charMaps[0]
	}

	switch outDef.Method {
	case "text":
		return serializeText(w, doc, charMap)
	case "html":
		return serializeHTML(w, doc, outDef)
	default:
		return serializeXML(w, doc, outDef, charMap)
	}
}

func defaultOutputDef() *OutputDef {
	return &OutputDef{
		Method:   "xml",
		Encoding: "UTF-8",
		Version:  "1.0",
	}
}

func serializeXML(w io.Writer, doc *helium.Document, outDef *OutputDef, charMap map[rune]string) error {
	if len(charMap) > 0 || hasDOEMarkers(doc) {
		return serializeXMLWithCharMap(w, doc, outDef, charMap)
	}
	// Set encoding on the document so the XML declaration includes it.
	if outDef.Encoding != "" && doc.Encoding() == "utf8" {
		doc.SetEncoding(outDef.Encoding)
	}
	// Add DOCTYPE if doctype-public or doctype-system is specified and
	// the document doesn't already have a DTD.
	if (outDef.DoctypePublic != "" || outDef.DoctypeSystem != "") && doc.IntSubset() == nil {
		rootName := "html" // default
		if root := doc.DocumentElement(); root != nil {
			rootName = root.Name()
		}
		if _, err := doc.CreateInternalSubset(rootName, outDef.DoctypePublic, outDef.DoctypeSystem); err != nil {
			return err
		}
	}
	var opts []helium.WriteOption
	if outDef.Indent {
		opts = append(opts, helium.WithFormat())
	}
	if outDef.OmitDeclaration {
		opts = append(opts, helium.WithNoDecl())
	}
	// The helium XML serializer adds a newline after the XML declaration
	// and after each top-level child (libxml2 behavior). For XSLT output,
	// when indent="no" we strip the newline after the declaration to
	// produce a conformant XSLT serialization.
	if !outDef.Indent && !outDef.OmitDeclaration {
		var buf strings.Builder
		if err := doc.XML(&buf, opts...); err != nil {
			return err
		}
		out := buf.String()
		// Remove the newline that immediately follows the XML declaration.
		if idx := strings.Index(out, "?>\n"); idx >= 0 {
			out = out[:idx+2] + out[idx+3:]
		}
		_, err := io.WriteString(w, out)
		return err
	}
	return doc.XML(w, opts...)
}

// serializeXMLWithCharMap serializes an XML document applying character map
// substitutions. Replacement strings are written raw (not escaped).
func serializeXMLWithCharMap(w io.Writer, doc *helium.Document, outDef *OutputDef, charMap map[rune]string) error {
	sw := stream.NewWriter(w)

	if !outDef.OmitDeclaration {
		enc := outDef.Encoding
		if enc == "" {
			enc = "UTF-8"
		}
		if err := sw.StartDocument("1.0", enc, ""); err != nil {
			return err
		}
	}

	// Add DOCTYPE if doctype-public or doctype-system is specified.
	if outDef.DoctypePublic != "" || outDef.DoctypeSystem != "" {
		rootName := "html"
		if root := doc.DocumentElement(); root != nil {
			rootName = root.Name()
		}
		if err := sw.WriteDTD(rootName, outDef.DoctypePublic, outDef.DoctypeSystem, ""); err != nil {
			return err
		}
	}

	err := serializeXMLNodeWithCharMap(sw, doc, charMap)
	if err != nil {
		return err
	}
	return sw.Flush()
}

func serializeXMLNodeWithCharMap(sw *stream.Writer, n helium.Node, charMap map[rune]string) error {
	doeActive := false
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		// Handle DOE marker PIs
		if child.Type() == helium.ProcessingInstructionNode {
			piName := string(child.Name())
			if piName == "disable-output-escaping" {
				doeActive = true
				continue
			}
			if piName == "enable-output-escaping" {
				doeActive = false
				continue
			}
		}
		switch child.Type() {
		case helium.ElementNode:
			elem := child.(*helium.Element)
			prefix := string(elem.Prefix())
			local := string(elem.LocalName())
			uri := string(elem.URI())
			if err := sw.StartElementNS(prefix, local, uri); err != nil {
				return err
			}
			// Write additional namespace declarations not handled by StartElementNS
			elemPrefix := string(elem.Prefix())
			for _, ns := range elem.Namespaces() {
				if ns.Prefix() == elemPrefix {
					continue // already declared by StartElementNS
				}
				if ns.Prefix() == "" {
					if err := sw.WriteAttribute("xmlns", ns.URI()); err != nil {
						return err
					}
				} else {
					if err := sw.WriteAttribute("xmlns:"+ns.Prefix(), ns.URI()); err != nil {
						return err
					}
				}
			}
			// Write attributes
			for _, attr := range elem.Attributes() {
				attrVal := applyCharacterMap(attr.Value(), charMap)
				if err := sw.WriteAttribute(attr.Name(), attrVal); err != nil {
					return err
				}
			}
			// Recurse into children
			if err := serializeXMLNodeWithCharMap(sw, elem, charMap); err != nil {
				return err
			}
			if err := sw.EndElement(); err != nil {
				return err
			}
		case helium.TextNode, helium.CDATASectionNode:
			text := string(child.Content())
			if doeActive {
				if err := sw.WriteRaw(text); err != nil {
					return err
				}
			} else if err := writeTextWithCharMap(sw, text, charMap); err != nil {
				return err
			}
		case helium.CommentNode:
			if err := sw.WriteComment(string(child.Content())); err != nil {
				return err
			}
		case helium.ProcessingInstructionNode:
			if err := sw.WritePI(string(child.Name()), string(child.Content())); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeTextWithCharMap writes text content, applying character map substitutions.
// Mapped characters are written raw (unescaped), unmapped characters are written
// as normal text (with XML escaping).
func writeTextWithCharMap(sw *stream.Writer, text string, charMap map[rune]string) error {
	var unmapped strings.Builder
	for _, r := range text {
		if repl, ok := charMap[r]; ok {
			// Flush any accumulated unmapped text first
			if unmapped.Len() > 0 {
				if err := sw.WriteString(unmapped.String()); err != nil {
					return err
				}
				unmapped.Reset()
			}
			// Write the replacement raw
			if err := sw.WriteRaw(repl); err != nil {
				return err
			}
		} else {
			unmapped.WriteRune(r)
		}
	}
	if unmapped.Len() > 0 {
		return sw.WriteString(unmapped.String())
	}
	return nil
}

func serializeText(w io.Writer, doc *helium.Document, charMap map[rune]string) error {
	// Text output: just write the text content of the document
	sw := stream.NewWriter(w)
	err := helium.Walk(doc, func(n helium.Node) error {
		switch n.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			text := string(n.Content())
			if len(charMap) > 0 {
				text = applyCharacterMap(text, charMap)
			}
			return sw.WriteRaw(text)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return sw.Flush()
}

func serializeHTML(w io.Writer, doc *helium.Document, outDef *OutputDef) error {
	// Output DOCTYPE if specified.
	if outDef.DoctypePublic != "" || outDef.DoctypeSystem != "" {
		rootName := "html"
		if root := doc.DocumentElement(); root != nil {
			rootName = root.Name()
		}
		_, _ = io.WriteString(w, "<!DOCTYPE ")
		_, _ = io.WriteString(w, rootName)
		if outDef.DoctypePublic != "" {
			_, _ = io.WriteString(w, " PUBLIC \"")
			_, _ = io.WriteString(w, outDef.DoctypePublic)
			_, _ = io.WriteString(w, "\"")
			if outDef.DoctypeSystem != "" {
				_, _ = io.WriteString(w, " \"")
				_, _ = io.WriteString(w, outDef.DoctypeSystem)
				_, _ = io.WriteString(w, "\"")
			}
		} else if outDef.DoctypeSystem != "" {
			_, _ = io.WriteString(w, " SYSTEM \"")
			_, _ = io.WriteString(w, outDef.DoctypeSystem)
			_, _ = io.WriteString(w, "\"")
		}
		_, _ = io.WriteString(w, ">\n")
	}
	opts := []htmlpkg.WriteOption{
		htmlpkg.WithNoDefaultDTD(),
		htmlpkg.WithNoFormat(),
		htmlpkg.WithPreserveCase(),
	}
	return htmlpkg.WriteDoc(w, doc, opts...)
}

// hasDOEMarkers checks if the document contains any disable-output-escaping markers.
func hasDOEMarkers(doc *helium.Document) bool {
	found := false
	_ = helium.Walk(doc, func(n helium.Node) error {
		if n.Type() == helium.ProcessingInstructionNode && string(n.Name()) == "disable-output-escaping" {
			found = true
		}
		return nil
	})
	return found
}

// applyCharacterMap replaces characters in text according to the character map.
func applyCharacterMap(text string, charMap map[rune]string) string {
	var b strings.Builder
	for _, r := range text {
		if repl, ok := charMap[r]; ok {
			b.WriteString(repl)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// resolveCharacterMaps builds a merged character map from a list of map names.
func resolveCharacterMaps(ss *Stylesheet, names []string) map[rune]string {
	if len(names) == 0 || ss == nil || len(ss.characterMaps) == 0 {
		return nil
	}
	merged := make(map[rune]string)
	visited := make(map[string]bool)
	var resolve func(name string)
	resolve = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true
		cm := ss.characterMaps[name]
		if cm == nil {
			return
		}
		// Resolve referenced maps first (lower priority)
		for _, ref := range cm.UseCharacterMaps {
			resolve(ref)
		}
		// This map's entries override
		for r, s := range cm.Mappings {
			merged[r] = s
		}
	}
	for _, name := range names {
		resolve(name)
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}
