package xslt3

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// xsltFunctions returns the XSLT-specific functions that need to be
// registered with the XPath evaluator by local name (no namespace prefix).
// The map is cached on ec after the first call.
func (ec *execContext) xsltFunctions() map[string]xpath3.Function {
	if ec.cachedFns != nil {
		return ec.cachedFns
	}
	ec.cachedFns = map[string]xpath3.Function{
		"current":             &xsltFunc{min: 0, max: 0, fn: ec.fnCurrent},
		"document":            &xsltFunc{min: 1, max: 2, fn: ec.fnDocument},
		"key":                 &xsltFunc{min: 2, max: 3, fn: ec.fnKey},
		"generate-id":         &xsltFunc{min: 0, max: 1, fn: ec.fnGenerateID},
		"system-property":     &xsltFunc{min: 1, max: 1, fn: ec.fnSystemProperty},
		"unparsed-entity-uri":       &xsltFunc{min: 1, max: 2, fn: ec.fnUnparsedEntityURI},
		"unparsed-entity-public-id": &xsltFunc{min: 1, max: 2, fn: ec.fnUnparsedEntityPublicID},
		"element-available":    &xsltFunc{min: 1, max: 1, fn: ec.fnElementAvailable},
		"function-available":   &xsltFunc{min: 1, max: 2, fn: ec.fnFunctionAvailable},
		"type-available":       &xsltFunc{min: 1, max: 1, fn: ec.fnTypeAvailable},
		"current-group":        &xsltFunc{min: 0, max: 0, fn: ec.fnCurrentGroup},
		"current-grouping-key": &xsltFunc{min: 0, max: 0, fn: ec.fnCurrentGroupingKey},
		"accumulator-before":   &xsltFunc{min: 1, max: 1, fn: ec.fnAccumulatorBefore},
		"accumulator-after":    &xsltFunc{min: 1, max: 1, fn: ec.fnAccumulatorAfter},
		"copy-of":              &xsltFunc{min: 0, max: 1, fn: ec.fnCopyOf},
		"snapshot":             &xsltFunc{min: 0, max: 1, fn: ec.fnSnapshot},
	}
	return ec.cachedFns
}

type xsltFunc struct {
	min int
	max int
	fn  func(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error)
}

func (f *xsltFunc) MinArity() int { return f.min }
func (f *xsltFunc) MaxArity() int { return f.max }
func (f *xsltFunc) Call(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	return f.fn(ctx, args)
}

// current() returns the current item (node or atomic value being processed).
func (ec *execContext) fnCurrent(_ context.Context, _ []xpath3.Sequence) (xpath3.Sequence, error) {
	// For-each over atomic values: return the current atomic item
	if ec.contextItem != nil {
		return xpath3.Sequence{ec.contextItem}, nil
	}
	if ec.currentNode == nil {
		return xpath3.EmptySequence(), nil
	}
	return xpath3.SingleNode(ec.currentNode), nil
}

// document(uri, base?) loads an external XML document.
// Per XSLT spec 14.1:
//   - First argument can be a string or a sequence of strings/nodes.
//   - When it is a sequence, each item is atomized to a URI and the
//     corresponding documents are returned as a sequence.
//   - An empty string returns the stylesheet document itself.
//   - Fragment identifiers (#frag) are stripped before loading.
//   - Second argument (optional) is a node whose base URI is used for
//     resolving relative URIs instead of the stylesheet base URI.
func (ec *execContext) fnDocument(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return xpath3.EmptySequence(), nil
	}

	// Determine the base URI for resolving relative URIs.
	// Default: the base URI of the current template's module, falling
	// back to the main stylesheet base URI.
	// If a second argument is provided, use that node's document's URL
	// as the base URI instead.
	baseDir := ""
	if ec.currentTemplate != nil && ec.currentTemplate.BaseURI != "" {
		baseDir = filepath.Dir(ec.currentTemplate.BaseURI)
	} else if ec.stylesheet.baseURI != "" {
		baseDir = filepath.Dir(ec.stylesheet.baseURI)
	}
	if len(args) >= 2 && len(args[1]) > 0 {
		if ni, ok := args[1][0].(xpath3.NodeItem); ok {
			if nodeBase := documentBaseURI(ni.Node); nodeBase != "" {
				baseDir = filepath.Dir(nodeBase)
			}
		}
	}

	// Iterate over all items in the first argument sequence.
	seen := make(map[string]struct{})
	var result xpath3.Sequence
	for _, item := range args[0] {
		av, err := xpath3.AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		uri, err := xpath3.AtomicToString(av)
		if err != nil {
			return nil, err
		}

		doc, err := ec.loadDocument(ctx, uri, baseDir)
		if err != nil {
			return nil, err
		}

		// Deduplicate by resolved URI (same document returned once).
		resolvedKey := ec.resolveDocumentURI(uri, baseDir)
		if _, dup := seen[resolvedKey]; dup {
			continue
		}
		seen[resolvedKey] = struct{}{}
		result = append(result, xpath3.NodeItem{Node: doc})
	}
	return result, nil
}

// fnDoc is an XSLT-aware wrapper around fn:doc() that applies
// xsl:strip-space rules to loaded documents.
func (ec *execContext) fnDoc(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return xpath3.EmptySequence(), nil
	}

	av, err := xpath3.AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	uri, err := xpath3.AtomicToString(av)
	if err != nil {
		return nil, err
	}

	// Determine base directory from current template or stylesheet.
	baseDir := ""
	if ec.currentTemplate != nil && ec.currentTemplate.BaseURI != "" {
		baseDir = filepath.Dir(ec.currentTemplate.BaseURI)
	} else if ec.stylesheet.baseURI != "" {
		baseDir = filepath.Dir(ec.stylesheet.baseURI)
	}

	doc, err := ec.loadDocument(ctx, uri, baseDir)
	if err != nil {
		return nil, err
	}
	return xpath3.SingleNode(doc), nil
}

// loadDocument loads a single XML document by URI, using baseDir for
// resolving relative paths.
func (ec *execContext) loadDocument(ctx context.Context, uri string, baseDir string) (*helium.Document, error) {
	// Empty string means the stylesheet module itself (XSLT spec 14.1).
	// When called from an included/imported module, return that module's
	// document, not the top-level stylesheet.
	if uri == "" {
		if ec.currentTemplate != nil && ec.currentTemplate.BaseURI != "" {
			if modDoc, ok := ec.stylesheet.moduleDocs[ec.currentTemplate.BaseURI]; ok {
				return modDoc, nil
			}
		}
		return ec.stylesheet.sourceDoc, nil
	}

	// Strip fragment identifier before loading.
	cleanURI := uri
	if idx := strings.IndexByte(cleanURI, '#'); idx >= 0 {
		cleanURI = cleanURI[:idx]
	}

	// Check cache by clean URI.
	resolvedURI := ec.resolveDocumentURI(cleanURI, baseDir)
	if doc, ok := ec.docCache[resolvedURI]; ok {
		return doc, nil
	}

	data, err := os.ReadFile(resolvedURI)
	if err != nil {
		return nil, dynamicError("FODC0002", "cannot load document %q: %v", uri, err)
	}

	doc, err := helium.Parse(ctx, data)
	if err != nil {
		return nil, dynamicError("FODC0002", "cannot parse document %q: %v", uri, err)
	}

	doc.SetURL(resolvedURI)

	// Apply xsl:strip-space rules to the loaded document so that
	// whitespace-only text nodes are removed consistently with how
	// the source document is treated.
	if len(ec.stylesheet.stripSpace) > 0 {
		ec.stripWhitespaceFromDoc(doc)
	}

	if ec.docCache == nil {
		ec.docCache = make(map[string]*helium.Document)
	}
	ec.docCache[resolvedURI] = doc
	return doc, nil
}

// resolveDocumentURI resolves a URI against a base directory.
func (ec *execContext) resolveDocumentURI(uri string, baseDir string) string {
	if uri == "" {
		return ""
	}
	// Strip fragment identifier.
	cleanURI := uri
	if idx := strings.IndexByte(cleanURI, '#'); idx >= 0 {
		cleanURI = cleanURI[:idx]
	}
	if strings.Contains(cleanURI, "://") || filepath.IsAbs(cleanURI) {
		return cleanURI
	}
	if baseDir != "" {
		return filepath.Join(baseDir, cleanURI)
	}
	return cleanURI
}

// documentBaseURI walks up a node to its owning Document and returns its URL.
func documentBaseURI(n helium.Node) string {
	for n != nil {
		if doc, ok := n.(*helium.Document); ok {
			return doc.URL()
		}
		n = n.Parent()
	}
	return ""
}

// key(name, value, doc?) looks up nodes by key.
func (ec *execContext) fnKey(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) < 2 {
		return nil, dynamicError(errCodeXTDE1170, "key() requires at least 2 arguments")
	}

	nameAV, err := xpath3.AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	name, err := xpath3.AtomicToString(nameAV)
	if err != nil {
		return nil, err
	}

	// Resolve prefixed key names to expanded names using stylesheet namespaces
	name = resolveQName(name, ec.stylesheet.namespaces)

	if len(args[1]) == 0 {
		return xpath3.EmptySequence(), nil
	}

	// Determine the root for key lookup.
	// Default: use the document containing the context node (per XSLT spec §16.3).
	// When the 3rd argument is provided, use that node directly as the search root.
	// This allows scoping to a subtree (e.g., key('k', 'v', $tree/sub)).
	var root helium.Node = ec.sourceDoc
	if len(args) >= 3 && len(args[2]) > 0 {
		ni, ok := args[2][0].(xpath3.NodeItem)
		if ok {
			root = ni.Node
		}
	} else if xpathNode := xpath3.FnContextNode(ctx); xpathNode != nil {
		root = documentRoot(xpathNode)
	} else if ec.contextNode != nil {
		root = documentRoot(ec.contextNode)
	}

	// When the second argument is a sequence, look up each value and
	// union the results (XSLT 2.0+ §16.3.2).
	seen := make(map[helium.Node]struct{})
	var seq xpath3.Sequence
	for _, item := range args[1] {
		valAV, err := xpath3.AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		value, err := xpath3.AtomicToString(valAV)
		if err != nil {
			return nil, err
		}
		nodes, err := ec.lookupKeyInDoc(name, value, root)
		if err != nil {
			return nil, err
		}
		for _, n := range nodes {
			if _, dup := seen[n]; !dup {
				seen[n] = struct{}{}
				seq = append(seq, xpath3.NodeItem{Node: n})
			}
		}
	}
	return seq, nil
}

// documentRoot walks up to the document root of the given node.
func documentRoot(n helium.Node) helium.Node {
	for n.Parent() != nil {
		n = n.Parent()
	}
	return n
}

// generate-id(node?) returns a unique string identifier for a node.
func (ec *execContext) fnGenerateID(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	var node helium.Node
	if len(args) == 0 || len(args[0]) == 0 {
		// Use the XPath context node (correct inside predicates) rather
		// than the XSLT context node.
		if xpathNode := xpath3.FnContextNode(ctx); xpathNode != nil {
			node = xpathNode
		} else {
			node = ec.contextNode
		}
	} else {
		ni, ok := args[0][0].(xpath3.NodeItem)
		if !ok {
			return nil, dynamicError("XPTY0004", "generate-id() argument must be a node")
		}
		node = ni.Node
	}
	if node == nil {
		return xpath3.SingleString(""), nil
	}
	// Use fmt.Sprintf %p to get a unique pointer-based ID
	id := fmt.Sprintf("id%p", node)
	return xpath3.SingleString(id), nil
}

// system-property(name) returns XSLT system properties.
func (ec *execContext) fnSystemProperty(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return xpath3.SingleString(""), nil
	}

	av, err := xpath3.AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	name, err := xpath3.AtomicToString(av)
	if err != nil {
		return nil, err
	}

	// Strip prefix
	local := name
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		local = name[idx+1:]
	}

	switch local {
	case "version":
		return xpath3.SingleString("3.0"), nil
	case "vendor":
		return xpath3.SingleString("helium"), nil
	case "vendor-url":
		return xpath3.SingleString("https://github.com/lestrrat-go/helium"), nil
	case "product-name":
		return xpath3.SingleString("helium-xslt3"), nil
	case "product-version":
		return xpath3.SingleString("0.1"), nil
	case "is-schema-aware":
		return xpath3.SingleString("yes"), nil
	case "supports-serialization":
		return xpath3.SingleString("yes"), nil
	case "supports-backwards-compatibility":
		return xpath3.SingleString("yes"), nil
	default:
		return xpath3.SingleString(""), nil
	}
}

// lookupUnparsedEntity finds an unparsed entity by name in the source document's DTD.
func (ec *execContext) lookupUnparsedEntity(name string) *helium.Entity {
	doc := ec.sourceDoc
	if doc == nil {
		return nil
	}
	if dtd := doc.IntSubset(); dtd != nil {
		if ent, ok := dtd.LookupEntity(name); ok {
			return ent
		}
	}
	if dtd := doc.ExtSubset(); dtd != nil {
		if ent, ok := dtd.LookupEntity(name); ok {
			return ent
		}
	}
	return nil
}

// unparsed-entity-uri(name) returns the URI of an unparsed entity.
func (ec *execContext) fnUnparsedEntityURI(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args[0]) == 0 {
		return xpath3.SingleString(""), nil
	}
	av, err := xpath3.AtomizeItem(args[0][0])
	if err != nil {
		return xpath3.SingleString(""), nil
	}
	name, err := xpath3.AtomicToString(av)
	if err != nil {
		return xpath3.SingleString(""), nil
	}
	ent := ec.lookupUnparsedEntity(name)
	if ent == nil {
		return xpath3.SingleString(""), nil
	}
	// If the entity has a pre-resolved URI, use it. Otherwise resolve
	// the system ID against the source document's base URL.
	uri := ent.URI()
	if uri == ent.SystemID() && ec.sourceDoc != nil {
		if base := ec.sourceDoc.URL(); base != "" {
			if resolved := helium.BuildURI(uri, base); resolved != "" {
				uri = resolved
			}
		}
	}
	return xpath3.SingleString(uri), nil
}

// unparsed-entity-public-id(name) returns the public identifier of an unparsed entity.
func (ec *execContext) fnUnparsedEntityPublicID(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args[0]) == 0 {
		return xpath3.SingleString(""), nil
	}
	av, err := xpath3.AtomizeItem(args[0][0])
	if err != nil {
		return xpath3.SingleString(""), nil
	}
	name, err := xpath3.AtomicToString(av)
	if err != nil {
		return xpath3.SingleString(""), nil
	}
	ent := ec.lookupUnparsedEntity(name)
	if ent == nil {
		return xpath3.SingleString(""), nil
	}
	return xpath3.SingleString(ent.ExternalID()), nil
}

// xslUserFunc wraps an xsl:function for use as an xpath3.Function.
type xslUserFunc struct {
	def *XSLFunction
	ec  *execContext
}

func (f *xslUserFunc) MinArity() int { return len(f.def.Params) }
func (f *xslUserFunc) MaxArity() int { return len(f.def.Params) }

func (f *xslUserFunc) Call(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	// Retrieve the XSLT exec context from the context.Context
	ec := f.ec
	if ecFromCtx := getExecContext(ctx); ecFromCtx != nil {
		ec = ecFromCtx
	}

	// Recursion depth check
	ec.depth++
	if ec.depth > maxRecursionDepth {
		ec.depth--
		return nil, dynamicError(errCodeXTDE0820, "recursion depth exceeded in xsl:function %s", f.def.Name.Name)
	}
	defer func() { ec.depth-- }()

	// Save and restore execution state.
	// xsl:function creates a new scope — tunnel params and current mode
	// are NOT inherited (XSLT 2.0 erratum XT.E19).
	savedContext := ec.contextNode
	savedCurrent := ec.currentNode
	savedPos := ec.position
	savedSize := ec.size
	savedTunnel := ec.tunnelParams
	savedMode := ec.currentMode
	ec.tunnelParams = nil
	ec.currentMode = ec.stylesheet.defaultMode
	defer func() {
		ec.contextNode = savedContext
		ec.currentNode = savedCurrent
		ec.position = savedPos
		ec.size = savedSize
		ec.tunnelParams = savedTunnel
		ec.currentMode = savedMode
	}()

	// Push new variable scope for parameters
	ec.pushVarScope()
	defer ec.popVarScope()

	// Bind parameters
	for i, param := range f.def.Params {
		if i < len(args) {
			ec.setVar(param.Name, args[i])
		} else if param.Select != nil {
			xpathCtx := ec.newXPathContext(ec.contextNode)
			result, err := param.Select.Evaluate(xpathCtx, ec.contextNode)
			if err != nil {
				return nil, err
			}
			ec.setVar(param.Name, result.Sequence())
		} else {
			ec.setVar(param.Name, xpath3.EmptySequence())
		}
	}

	// Execute the function body, collecting result into a temporary document.
	// For functions returning atomic types, use captureItems mode so that
	// attribute nodes returned by xsl:sequence are preserved directly
	// (writing them to a DOM tree loses them as attributes of the wrapper).
	tmpDoc := helium.NewDefaultDocument()
	tmpRoot, _ := tmpDoc.CreateElement("_xsl_fn_result")
	_ = tmpDoc.SetDocumentElement(tmpRoot)

	atomicReturn := f.def.As != "" && isAtomicTypeName(f.def.As)
	frame := &outputFrame{current: tmpRoot, doc: tmpDoc, captureItems: true}
	ec.outputStack = append(ec.outputStack, frame)
	defer func() {
		ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
	}()

	for _, inst := range f.def.Body {
		if err := ec.executeInstruction(ctx, inst); err != nil {
			return nil, err
		}
	}

	// Return captured items if any, otherwise collect from DOM.
	// For atomic return types, atomize the captured items.
	var result xpath3.Sequence
	if len(frame.pendingItems) > 0 {
		if tmpRoot.FirstChild() != nil {
			var seq xpath3.Sequence
			for child := tmpRoot.FirstChild(); child != nil; child = child.NextSibling() {
				seq = append(seq, xpath3.NodeItem{Node: child})
			}
			seq = append(seq, frame.pendingItems...)
			if atomicReturn {
				result = atomizeSequence(seq)
			} else {
				result = seq
			}
		} else if atomicReturn {
			result = atomizeSequence(frame.pendingItems)
		} else {
			result = frame.pendingItems
		}
	} else {
		result = ec.collectSequenceFromNode(tmpRoot)
	}
	return result, nil
}

// collectSequenceFromNode converts children of a node to an XPath sequence.
func (ec *execContext) collectSequenceFromNode(node helium.Node) xpath3.Sequence {
	var seq xpath3.Sequence
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			text := string(child.Content())
			seq = append(seq, xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: text})
		default:
			seq = append(seq, xpath3.NodeItem{Node: child})
		}
	}
	return seq
}

// isAtomicTypeName returns true if the given type name (from an as="" attribute)
// represents an atomic/simple type (not a node type or function type).
// Handles occurrence indicators (?, *, +) and xs: prefixed types.
func isAtomicTypeName(as string) bool {
	// Strip occurrence indicator
	name := strings.TrimRight(as, "?*+")
	name = strings.TrimSpace(name)
	// xs:string, xs:integer, xs:boolean, xs:double, etc.
	if strings.HasPrefix(name, "xs:") {
		return true
	}
	// unprefixed atomic types (rare but possible)
	switch name {
	case "string", "integer", "boolean", "double", "float", "decimal",
		"date", "dateTime", "time", "duration", "anyURI":
		return true
	}
	return false
}

// xsltElementVersion maps XSLT elements to the minimum version that supports them.
// Elements not in this map are assumed to be available in version 1.0+.
var xsltElementVersion = map[string]string{
	// XSLT 1.0 elements (available in all versions)
	"apply-imports": "1.0", "apply-templates": "1.0", "attribute": "1.0",
	"call-template": "1.0", "choose": "1.0", "comment": "1.0",
	"copy": "1.0", "copy-of": "1.0", "element": "1.0",
	"fallback": "1.0", "for-each": "1.0", "if": "1.0",
	"import": "1.0", "include": "1.0", "message": "1.0",
	"number": "1.0", "otherwise": "1.0", "output": "1.0",
	"param": "1.0", "processing-instruction": "1.0",
	"sort": "1.0", "strip-space": "1.0",
	"preserve-space": "1.0", "stylesheet": "1.0", "template": "1.0",
	"text": "1.0", "transform": "1.0", "value-of": "1.0",
	"variable": "1.0", "when": "1.0", "with-param": "1.0",
	// XSLT 2.0 elements
	"analyze-string": "2.0", "document": "2.0", "for-each-group": "2.0",
	"namespace": "2.0", "next-match": "2.0", "perform-sort": "2.0",
	"result-document": "2.0", "sequence": "2.0",
	// XSLT 3.0 elements
	"try": "3.0", "catch": "3.0", "where-populated": "3.0",
	"on-empty": "3.0", "on-non-empty": "3.0", "merge": "3.0",
	"merge-source": "3.0", "merge-action": "3.0", "merge-key": "3.0",
	"assert": "3.0", "accumulator": "3.0", "accumulator-rule": "3.0",
	"fork": "3.0", "iterate": "3.0", "break": "3.0",
	"next-iteration": "3.0", "map": "3.0", "map-entry": "3.0",
	"array": "3.0", "accept": "3.0", "expose": "3.0",
	"override": "3.0", "use-package": "3.0", "package": "3.0",
	"global-context-item": "3.0", "context-item": "3.0",
	"source-document": "3.0",
}

// element-available(name) returns true if the named XSLT element is available.
func (ec *execContext) fnElementAvailable(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return xpath3.SingleBoolean(false), nil
	}
	av, err := xpath3.AtomizeItem(args[0][0])
	if err != nil {
		return xpath3.SingleBoolean(false), nil
	}
	name, _ := xpath3.AtomicToString(av)

	// Resolve prefix:local to namespace + local
	local := name
	ns := NSXSLT
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		local = name[idx+1:]
		if uri, ok := ec.stylesheet.namespaces[prefix]; ok {
			ns = uri
		}
	}
	if ns != NSXSLT {
		return xpath3.SingleBoolean(false), nil
	}
	minVersion, ok := xsltElementVersion[local]
	if !ok {
		return xpath3.SingleBoolean(false), nil
	}
	// Check if the element is available in the current stylesheet version
	if ec.stylesheet.version != "" && ec.stylesheet.version < minVersion {
		return xpath3.SingleBoolean(false), nil
	}
	return xpath3.SingleBoolean(true), nil
}

// function-available(name, arity?) returns true if the named function is available.
func (ec *execContext) fnFunctionAvailable(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return xpath3.SingleBoolean(false), nil
	}
	av, err := xpath3.AtomizeItem(args[0][0])
	if err != nil {
		return xpath3.SingleBoolean(false), nil
	}
	name, _ := xpath3.AtomicToString(av)

	// Check XSLT functions by local name
	fns := ec.xsltFunctions()
	if _, ok := fns[name]; ok {
		return xpath3.SingleBoolean(true), nil
	}

	// Check user-defined functions (prefixed)
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		local := name[idx+1:]
		if uri, ok := ec.stylesheet.namespaces[prefix]; ok {
			qn := xpath3.QualifiedName{URI: uri, Name: local}
			if _, ok := ec.stylesheet.functions[qn]; ok {
				return xpath3.SingleBoolean(true), nil
			}
		}
	}

	// Check XPath built-in functions — they're always available.
	// For simplicity, return true for common fn: functions.
	return xpath3.SingleBoolean(false), nil
}

// type-available(name) returns true if the named type is available.
func (ec *execContext) fnTypeAvailable(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return xpath3.SingleBoolean(false), nil
	}
	av, err := xpath3.AtomizeItem(args[0][0])
	if err != nil {
		return xpath3.SingleBoolean(false), nil
	}
	name, _ := xpath3.AtomicToString(av)

	// Resolve QName to canonical xs:... form
	resolved := resolveQName(name, ec.stylesheet.namespaces)
	// If it resolved to {uri}local, normalize XSD namespace to xs: prefix
	if strings.HasPrefix(resolved, "{http://www.w3.org/2001/XMLSchema}") {
		local := resolved[len("{http://www.w3.org/2001/XMLSchema}"):]
		resolved = "xs:" + local
	} else if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		local := name[idx+1:]
		if prefix == "xs" || prefix == "xsd" {
			resolved = "xs:" + local
		} else if uri, ok := ec.stylesheet.namespaces[prefix]; ok && uri == "http://www.w3.org/2001/XMLSchema" {
			resolved = "xs:" + local
		}
	}

	// Check xpath3 built-in types
	if xpath3.IsKnownXSDType(resolved) {
		return xpath3.SingleBoolean(true), nil
	}

	// Check imported schemas
	for _, schema := range ec.stylesheet.schemas {
		local := resolved
		ns := "http://www.w3.org/2001/XMLSchema"
		if strings.HasPrefix(resolved, "xs:") {
			local = resolved[3:]
		} else if idx := strings.IndexByte(resolved, ':'); idx >= 0 {
			local = resolved[idx+1:]
		}
		if _, ok := schema.LookupType(local, ns); ok {
			return xpath3.SingleBoolean(true), nil
		}
	}

	return xpath3.SingleBoolean(false), nil
}

// current-group() returns the items in the current group during for-each-group.
func (ec *execContext) fnCurrentGroup(_ context.Context, _ []xpath3.Sequence) (xpath3.Sequence, error) {
	if ec.currentGroup != nil {
		return ec.currentGroup, nil
	}
	return xpath3.EmptySequence(), nil
}

// current-grouping-key() returns the grouping key for the current group.
func (ec *execContext) fnCurrentGroupingKey(_ context.Context, _ []xpath3.Sequence) (xpath3.Sequence, error) {
	if ec.inGroupContext {
		if ec.currentGroupKey != nil {
			return ec.currentGroupKey, nil
		}
		// Key is absent (e.g., group-starting-with, group-ending-with)
		return nil, dynamicError("XTDE1071", "current-grouping-key() is not available for group-starting-with/group-ending-with")
	}
	return nil, dynamicError("XTDE1071", "current-grouping-key() called outside xsl:for-each-group")
}

// accumulator-before(name) returns the pre-descent value of a named accumulator.
func (ec *execContext) fnAccumulatorBefore(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return xpath3.EmptySequence(), nil
	}
	av, err := xpath3.AtomizeItem(args[0][0])
	if err != nil {
		return xpath3.EmptySequence(), nil
	}
	name, err := xpath3.AtomicToString(av)
	if err != nil {
		return xpath3.EmptySequence(), nil
	}
	name = resolveQName(name, ec.stylesheet.namespaces)
	if val, ok := ec.accumulatorState[name]; ok {
		return val, nil
	}
	return xpath3.EmptySequence(), nil
}

// accumulator-after(name) returns the post-descent value of a named accumulator.
func (ec *execContext) fnAccumulatorAfter(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return xpath3.EmptySequence(), nil
	}
	av, err := xpath3.AtomizeItem(args[0][0])
	if err != nil {
		return xpath3.EmptySequence(), nil
	}
	name, err := xpath3.AtomicToString(av)
	if err != nil {
		return xpath3.EmptySequence(), nil
	}
	name = resolveQName(name, ec.stylesheet.namespaces)
	if val, ok := ec.accumulatorState[name]; ok {
		return val, nil
	}
	return xpath3.EmptySequence(), nil
}

// copy-of() returns a deep copy of the context node (zero-argument XSLT 3.0 streaming function).
func (ec *execContext) fnCopyOf(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	// copy-of() with argument: deep-copy the argument node(s)
	// copy-of() with no args: deep-copy the context node (streaming snapshot)
	var nodes []helium.Node
	if len(args) > 0 {
		// Explicit argument: copy the given node(s).
		// If the argument is empty, return empty (don't fall through to context).
		if len(args[0]) == 0 {
			return xpath3.EmptySequence(), nil
		}
		for _, item := range args[0] {
			ni, ok := item.(xpath3.NodeItem)
			if !ok {
				continue
			}
			nodes = append(nodes, ni.Node)
		}
	} else {
		// No argument: copy the context node (streaming snapshot).
		// Prefer XPath dynamic context node (set by evaluator during path
		// steps like transaction/copy-of()) over XSLT execution context.
		if n := xpath3.FnContextNode(ctx); n != nil {
			nodes = append(nodes, n)
		} else if ec.contextNode != nil {
			nodes = append(nodes, ec.contextNode)
		} else {
			return xpath3.EmptySequence(), nil
		}
	}
	var result xpath3.Sequence
	for _, node := range nodes {
		copied, err := helium.CopyNode(node, ec.resultDoc)
		if err != nil {
			result = append(result, xpath3.NodeItem{Node: node})
			continue
		}
		result = append(result, xpath3.NodeItem{Node: copied})
	}
	return result, nil
}

// snapshot() produces a deep copy of the node that also preserves ancestor
// information.  Each ancestor element is shallow-copied (name, attributes,
// namespace declarations) and the chain is connected so that
// ancestor::*/parent::*/.. navigation works on the snapshot.
func (ec *execContext) fnSnapshot(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	var nodes []helium.Node
	if len(args) > 0 {
		if len(args[0]) == 0 {
			return xpath3.EmptySequence(), nil
		}
		for _, item := range args[0] {
			ni, ok := item.(xpath3.NodeItem)
			if !ok {
				continue
			}
			nodes = append(nodes, ni.Node)
		}
	} else {
		if n := xpath3.FnContextNode(ctx); n != nil {
			nodes = append(nodes, n)
		} else if ec.contextNode != nil {
			nodes = append(nodes, ec.contextNode)
		} else {
			return xpath3.EmptySequence(), nil
		}
	}

	var result xpath3.Sequence
	for _, node := range nodes {
		snapped, err := ec.snapshotNode(node)
		if err != nil {
			// Fall back to the original node on error.
			result = append(result, xpath3.NodeItem{Node: node})
			continue
		}
		result = append(result, xpath3.NodeItem{Node: snapped})
	}
	return result, nil
}

// snapshotNode creates a deep copy of node and wraps it in shallow copies
// of all its ancestors up to and including the document root.
func (ec *execContext) snapshotNode(node helium.Node) (helium.Node, error) {
	// For a document node, just do a full deep copy.
	if node.Type() == helium.DocumentNode {
		doc, ok := node.(*helium.Document)
		if !ok {
			return nil, fmt.Errorf("unexpected DocumentNode type %T", node)
		}
		return helium.CopyDoc(doc)
	}

	// Build the ancestor chain from node's parent up to (but not including)
	// the document node, collecting elements in bottom-up order.
	var ancestors []*helium.Element
	for p := node.Parent(); p != nil; p = p.Parent() {
		if elem, ok := p.(*helium.Element); ok {
			ancestors = append(ancestors, elem)
		}
	}

	// Create a new document to own the snapshot.
	snapDoc := helium.NewDefaultDocument()

	// Deep-copy the target node itself.
	copied, err := helium.CopyNode(node, snapDoc)
	if err != nil {
		return nil, err
	}

	// Build the ancestor chain: for each ancestor (bottom-up), create a
	// shallow copy (name + attributes + namespace declarations) and attach
	// the previous level as its only child.
	current := copied
	for _, anc := range ancestors {
		shell, err := shallowCopyElement(anc, snapDoc)
		if err != nil {
			return nil, err
		}
		if err := shell.AddChild(current); err != nil {
			return nil, err
		}
		current = shell
	}

	// Attach the top-level element (or the copied node itself if no
	// ancestors) to the snapshot document.
	if err := snapDoc.AddChild(current); err != nil {
		return nil, err
	}

	return copied, nil
}

// shallowCopyElement copies an element's name, namespace declarations, and
// attributes but none of its children.
func shallowCopyElement(src *helium.Element, doc *helium.Document) (*helium.Element, error) {
	elem, err := doc.CreateElement(src.LocalName())
	if err != nil {
		return nil, err
	}

	declaredPrefixes := make(map[string]bool)

	// Copy namespace declarations.
	if nc, ok := helium.Node(src).(helium.NamespaceContainer); ok {
		for _, ns := range nc.Namespaces() {
			if err := elem.DeclareNamespace(ns.Prefix(), ns.URI()); err != nil {
				return nil, err
			}
			declaredPrefixes[ns.Prefix()] = true
		}
	}

	// Copy the active namespace.
	if nsr, ok := helium.Node(src).(helium.Namespacer); ok {
		if ns := nsr.Namespace(); ns != nil {
			if ns.Prefix() != "" && !declaredPrefixes[ns.Prefix()] {
				if err := elem.DeclareNamespace(ns.Prefix(), ns.URI()); err != nil {
					return nil, err
				}
			}
			if err := elem.SetActiveNamespace(ns.Prefix(), ns.URI()); err != nil {
				return nil, err
			}
		}
	}

	// Copy attributes.
	for _, a := range src.Attributes() {
		if err := elem.SetAttribute(a.Name(), a.Value()); err != nil {
			return nil, err
		}
	}

	return elem, nil
}

// xsltFunctionsNS returns user-defined xsl:function definitions and
// XSLT built-in functions that need to be callable in the fn: namespace
// as xpath3 functions keyed by qualified name.
func (ec *execContext) xsltFunctionsNS() map[xpath3.QualifiedName]xpath3.Function {
	if ec.cachedFnsNS != nil {
		return ec.cachedFnsNS
	}
	ec.cachedFnsNS = make(map[xpath3.QualifiedName]xpath3.Function, len(ec.stylesheet.functions)+1)

	// Register XSLT document() in the fn: namespace so fn:document() works.
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "document"}] = &xsltFunc{min: 1, max: 2, fn: ec.fnDocument}

	// Override fn:doc to apply xsl:strip-space rules to loaded documents.
	if len(ec.stylesheet.stripSpace) > 0 {
		ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "doc"}] = &xsltFunc{min: 1, max: 1, fn: ec.fnDoc}
	}

	for qn, def := range ec.stylesheet.functions {
		ec.cachedFnsNS[qn] = &xslUserFunc{def: def, ec: ec}
	}
	return ec.cachedFnsNS
}

