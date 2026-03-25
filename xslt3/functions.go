package xslt3

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/sequence"
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
		"nilled":                    &xsltFunc{min: 0, max: 1, fn: ec.fnNilled},
		"current":                   &xsltFunc{min: 0, max: 0, fn: ec.fnCurrent, noDynRef: true, dynRefError: errCodeXTDE1360},
		"document":                  &xsltFunc{min: 1, max: 2, fn: ec.fnDocument},
		"key":                       &xsltFunc{min: 2, max: 3, fn: ec.fnKey},
		"generate-id":               &xsltFunc{min: 0, max: 1, fn: ec.fnGenerateID},
		"system-property":           &xsltFunc{min: 1, max: 1, fn: ec.fnSystemProperty},
		"unparsed-entity-uri":       &xsltFunc{min: 1, max: 2, fn: ec.fnUnparsedEntityURI},
		"unparsed-entity-public-id": &xsltFunc{min: 1, max: 2, fn: ec.fnUnparsedEntityPublicID},
		"element-available":         &xsltFunc{min: 1, max: 1, fn: ec.fnElementAvailable},
		"function-available":        &xsltFunc{min: 1, max: 2, fn: ec.fnFunctionAvailable},
		"type-available":            &xsltFunc{min: 1, max: 1, fn: ec.fnTypeAvailable},
		"current-group":             &xsltFunc{min: 0, max: 0, fn: ec.fnCurrentGroup, noDynRef: true, dynRefError: errCodeXTDE1061},
		"current-grouping-key":      &xsltFunc{min: 0, max: 0, fn: ec.fnCurrentGroupingKey, noDynRef: true, dynRefError: errCodeXTDE1071},
		"accumulator-before": &xsltFunc{min: 1, max: 1, fn: func(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
			return ec.accumulatorLookup(ctx, args, "accumulator-before", func() (map[helium.Node]map[string]xpath3.Sequence, map[helium.Node]map[string]error) {
				return ec.accumulatorBeforeByNode, ec.accumulatorBeforeErrorByNode
			})
		}},
		"accumulator-after": &xsltFunc{min: 1, max: 1, fn: func(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
			return ec.accumulatorLookup(ctx, args, "accumulator-after", func() (map[helium.Node]map[string]xpath3.Sequence, map[helium.Node]map[string]error) {
				return ec.accumulatorAfterByNode, ec.accumulatorAfterErrorByNode
			})
		}},
		"copy-of":                     &xsltFunc{min: 0, max: 1, fn: ec.fnCopyOf},
		"snapshot":                    &xsltFunc{min: 0, max: 1, fn: ec.fnSnapshot},
		"regex-group":                 &regexGroupFunc{ec: ec},
		"transform":                   &xsltFunc{min: 1, max: 1, fn: ec.fnTransform},
		"available-system-properties": &xsltFunc{min: 0, max: 0, fn: ec.fnAvailableSystemProperties},
		"stream-available":            &xsltFunc{min: 1, max: 1, fn: ec.fnStreamAvailable},
		"current-output-uri":          &xsltFunc{min: 0, max: 0, fn: ec.fnCurrentOutputURI},
	}
	return ec.cachedFns
}

type xsltFunc struct {
	min         int
	max         int
	fn          func(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error)
	noDynRef    bool   // true if the function must not be called via dynamic function reference
	dynRefError string // error code to raise on dynamic call
}

func (f *xsltFunc) MinArity() int { return f.min }
func (f *xsltFunc) MaxArity() int { return f.max }
func (f *xsltFunc) Call(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	return f.fn(ctx, args)
}

// NoDynamicRef returns true if this function should not be available
// for named function references (e.g. current-group#0).
func (f *xsltFunc) NoDynamicRef() bool { return f.noDynRef }

// DynRefErrorCode returns the error code to raise when this function
// is called via dynamic reference.
func (f *xsltFunc) DynRefErrorCode() string { return f.dynRefError }

// regexGroupFunc implements xpath3.Function and DynamicRefSnapshotProvider
// for the regex-group() XSLT function. When used as a function reference
// (regex-group#1), it captures the current regex groups as a closure.
type regexGroupFunc struct {
	ec *execContext
}

func (f *regexGroupFunc) MinArity() int { return 1 }
func (f *regexGroupFunc) MaxArity() int { return 1 }
func (f *regexGroupFunc) Call(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	return f.ec.fnRegexGroup(ctx, args)
}

func (f *regexGroupFunc) DynamicRefSnapshot(_ context.Context, arity int) (xpath3.FunctionItem, bool) {
	// Per XSLT 3.0 §5.3.4, a dynamic function reference regex-group#1
	// does NOT capture the regex context. When called dynamically it
	// always returns a zero-length string, because the call is not
	// lexically within xsl:matching-substring.
	fi := xpath3.FunctionItem{
		Arity:     arity,
		Name:      "regex-group",
		Namespace: "http://www.w3.org/1999/XSL/Transform",
		Invoke: func(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
			return xpath3.SingleString(""), nil
		},
	}
	return fi, true
}

// fnNilled overrides the XPath nilled() function to support schema-aware
// nilled checking via the exec context's type annotations and schema registry.
func (ec *execContext) fnNilled(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	var node helium.Node
	if len(args) == 0 {
		// nilled() with no arguments: use context node.
		node = ec.contextNode
	} else if args[0] == nil || sequence.Len(args[0]) == 0 {
		// nilled(()) or empty sequence argument: return empty sequence.
		return nil, nil
	} else {
		ni, ok := args[0].Get(0).(xpath3.NodeItem)
		if !ok {
			return nil, nil
		}
		node = ni.Node
	}
	if node == nil || node.Type() != helium.ElementNode {
		return nil, nil
	}
	elem, ok := node.(*helium.Element)
	if !ok {
		return xpath3.SingleBoolean(false), nil
	}
	// When input-type-annotations="strip", nilled is always false.
	if ec.stylesheet.inputTypeAnnotations == validationStrip {
		return xpath3.SingleBoolean(false), nil
	}
	// nilled() returns true only when the element was confirmed nilled during
	// schema validation (tracked in nilledElements). Documents loaded via
	// doc() get type annotations but are not tracked as nilled since fn:nilled()
	// should only return true for explicitly schema-validated source documents.
	if ec.nilledElements != nil {
		if _, ok := ec.nilledElements[elem]; ok {
			return xpath3.SingleBoolean(true), nil
		}
	}
	return xpath3.SingleBoolean(false), nil
}

// current() returns the current item (node or atomic value being processed).
func (ec *execContext) fnCurrent(_ context.Context, _ []xpath3.Sequence) (xpath3.Sequence, error) {
	// For-each over atomic values: return the current atomic item
	if ec.contextItem != nil {
		return xpath3.ItemSlice{ec.contextItem}, nil
	}
	if ec.currentNode == nil {
		// XTDE1360: current() called when context item is absent
		return nil, dynamicError(errCodeXTDE1360,
			"current() called when context item is absent")
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
	if len(args) == 0 || (args[0] == nil || sequence.Len(args[0]) == 0) {
		return xpath3.EmptySequence(), nil
	}

	// Determine the base URI for resolving relative URIs.
	// Default: the base URI of the current template's module, falling
	// back to the main stylesheet base URI.
	// If a second argument is provided, use that node's document's URL
	// as the base URI instead.
	baseDir := ec.baseDir()
	if len(args) >= 2 && args[1] != nil && sequence.Len(args[1]) > 0 {
		if ni, ok := args[1].Get(0).(xpath3.NodeItem); ok {
			nodeBase := documentBaseURI(ni.Node)
			if nodeBase != "" {
				baseDir = filepath.Dir(nodeBase)
			} else if !nodeHasDocumentRoot(ni.Node) {
				// XTDE1162: second argument is an orphan node (no
				// document root) with no base URI.
				return nil, dynamicError(errCodeXTDE1162,
					"document() second argument node has no base URI")
			}
			// Node belongs to a document tree but the document has no
			// URL — fall back to the stylesheet-derived baseDir.
		}
	}

	// Iterate over all items in the first argument sequence.
	seen := make(map[string]struct{})
	var result xpath3.ItemSlice
	for item := range sequence.Items(args[0]) {
		itemBaseDir := baseDir
		if len(args) < 2 {
			if ni, ok := item.(xpath3.NodeItem); ok {
				if nodeBase := documentBaseURI(ni.Node); nodeBase != "" {
					itemBaseDir = filepath.Dir(nodeBase)
				} else if !nodeHasDocumentRoot(ni.Node) {
					// XTDE1162: first argument node has no base URI
					// and no document root (parentless text node).
					return nil, dynamicError(errCodeXTDE1162,
						"document() argument node has no base URI")
				}
			}
		}

		av, err := xpath3.AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		uri, err := xpath3.AtomicToString(av)
		if err != nil {
			return nil, err
		}

		doc, err := ec.loadDocument(ctx, uri, itemBaseDir)
		if err != nil {
			return nil, err
		}

		// Deduplicate by resolved URI (same document returned once).
		resolvedKey := ec.resolveDocumentURI(uri, itemBaseDir)
		if _, dup := seen[resolvedKey]; dup {
			continue
		}
		seen[resolvedKey] = struct{}{}

		// If the URI has a fragment identifier, select the element with
		// that ID from the loaded document (XSLT spec 14.1).
		var resultNode helium.Node = doc
		if fragIdx := strings.IndexByte(uri, '#'); fragIdx >= 0 {
			frag := uri[fragIdx+1:]
			if frag != "" {
				if elem := findElementByID(doc, frag); elem != nil {
					resultNode = elem
				} else {
					continue
				}
			}
		}
		result = append(result, xpath3.NodeItem{Node: resultNode})
	}
	return result, nil
}

// fnDoc is an XSLT-aware wrapper around fn:doc() that applies
// xsl:strip-space rules to loaded documents.
func (ec *execContext) fnDoc(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || (args[0] == nil || sequence.Len(args[0]) == 0) {
		return xpath3.EmptySequence(), nil
	}

	av, err := xpath3.AtomizeItem(args[0].Get(0))
	if err != nil {
		return nil, err
	}
	uri, err := xpath3.AtomicToString(av)
	if err != nil {
		return nil, err
	}

	baseDir := ec.baseDir()

	doc, err := ec.loadDocument(ctx, uri, baseDir)
	if err != nil {
		return nil, err
	}
	return xpath3.SingleNode(doc), nil
}

// loadDocument loads a single XML document by URI, using baseDir for
// resolving relative paths.
func (ec *execContext) loadDocument(ctx context.Context, uri string, baseDir string) (*helium.Document, error) {
	// Fragment-only URI ("#id") refers to the source document.
	if strings.HasPrefix(uri, "#") {
		if ec.sourceDoc != nil {
			return ec.sourceDoc, nil
		}
		return nil, dynamicError(errCodeFODC0002, "cannot load document %q: no source document", uri)
	}

	// Empty string means the stylesheet module itself (XSLT spec 14.1).
	// When called from an included/imported module, return that module's
	// document, not the top-level stylesheet.
	// However, if xml:base changes the effective base URI, resolve against
	// that URI instead (the base URI may point to a different file).
	if uri == "" {
		effectiveBase := ec.effectiveStaticBaseURI()
		if ec.currentTemplate != nil && ec.currentTemplate.BaseURI != "" {
			if modDoc, ok := ec.stylesheet.moduleDocs[ec.currentTemplate.BaseURI]; ok {
				// Only return the module doc if the effective base URI
				// matches the template's module. If xml:base overrides,
				// fall through to load the overridden URI.
				if effectiveBase == ec.currentTemplate.BaseURI || effectiveBase == "" {
					return modDoc, nil
				}
			} else if effectiveBase == ec.currentTemplate.BaseURI || effectiveBase == "" {
				return ec.stylesheet.sourceDoc, nil
			}
		} else if effectiveBase == "" || effectiveBase == ec.stylesheet.baseURI {
			return ec.stylesheet.sourceDoc, nil
		}
		// xml:base overrides the base URI — resolve the empty string
		// against the effective base URI to load the target document.
		uri = effectiveBase
		baseDir = filepath.Dir(effectiveBase)
	}

	// Strip fragment identifier before loading.
	cleanURI := uri
	if idx := strings.IndexByte(cleanURI, '#'); idx >= 0 {
		cleanURI = cleanURI[:idx]
	}

	// Check cache by clean URI.
	resolvedURI := ec.resolveDocumentURI(cleanURI, baseDir)

	// If the resolved URI matches the source document, return it directly
	// to preserve node identity (spec: doc(document-uri(X)) is X).
	if ec.sourceDoc != nil && ec.sourceDoc.URL() == resolvedURI {
		return ec.sourceDoc, nil
	}

	if doc, ok := ec.docCache[resolvedURI]; ok {
		return doc, nil
	}

	data, err := os.ReadFile(resolvedURI)
	if err != nil {
		// XTDE1160: when the URI contains a fragment identifier, use XTDE1160
		// (fragment identifier error) instead of FODC0002.
		if strings.ContainsRune(uri, '#') {
			return nil, dynamicError(errCodeXTDE1160,
				"cannot load document %q with fragment identifier: %v", uri, err)
		}
		return nil, dynamicError(errCodeFODC0002, "cannot load document %q: %v", uri, err)
	}
	// helium's parser currently rejects literal U+FFFD from RuneCursor-backed
	// input, but accepts the equivalent character reference. Normalizing here
	// preserves the infoset for XSLT document() loads over the W3C source tree.
	data = bytes.ReplaceAll(data, []byte("\uFFFD"), []byte("&#xFFFD;"))

	p := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true).SubstituteEntities(true).FixBaseURIs(false).BaseURI(resolvedURI)
	doc, err := p.Parse(ctx, data)
	if err != nil {
		return nil, dynamicError(errCodeFODC0002, "cannot parse document %q: %v", uri, err)
	}

	doc.SetURL(resolvedURI)

	// Apply xsl:strip-space rules to the loaded document so that
	// whitespace-only text nodes are removed consistently with how
	// the source document is treated.
	if len(ec.stylesheet.stripSpace) > 0 {
		ec.stripWhitespaceFromDoc(doc)
	}

	// Schema-aware: validate the loaded document against imported schemas and
	// apply type annotations so that schema-aware XPath expressions work
	// correctly on documents loaded via fn:doc() / fn:document().
	if ec.schemaRegistry != nil && len(ec.stylesheet.schemas) > 0 {
		vr, _ := ec.schemaRegistry.ValidateDoc(ctx, doc)
		for node, typeName := range vr.Annotations {
			ec.annotateNode(node, typeName)
		}
		for elem := range vr.NilledElements {
			ec.markNilled(elem)
		}
	}

	if ec.docCache == nil {
		ec.docCache = make(map[string]*helium.Document)
	}
	ec.docCache[resolvedURI] = doc

	// Pre-compute accumulator states for documents loaded via doc()/document()
	// so that accumulator-before()/accumulator-after() work when processing
	// these documents (XSLT 3.0 §14.1: accumulators are always applicable).
	if len(ec.effectiveAccumulators()) > 0 {
		names := append([]string(nil), ec.effectiveStylesheet().accumulatorOrder...)
		if err := ec.computeAccumulatorStates(ctx, doc, names); err != nil {
			return nil, err
		}
		if ec.accumulatorComputedDocs == nil {
			ec.accumulatorComputedDocs = make(map[helium.Node]struct{})
		}
		ec.accumulatorComputedDocs[documentRoot(doc)] = struct{}{}
	}

	return doc, nil
}

// resolveAgainstBaseURI resolves a relative URI against an effective base URI.
// The base URI may be a file path (e.g., /a/b/style.xsl) or a directory-like
// path from xml:base processing (e.g., /a/b). For file paths (containing a
// dot-extension in the last segment), the directory part is extracted via
// filepath.Dir. For directory-like paths, the path is used directly.
func resolveAgainstBaseURI(uri string, baseURI string) string {
	if uri == "" || baseURI == "" {
		return uri
	}
	if strings.Contains(uri, "://") || filepath.IsAbs(uri) {
		return uri
	}
	baseDir := baseURIDir(baseURI)
	return filepath.Join(baseDir, uri)
}

func splitURIFragment(uri string) (string, string) {
	base, fragment, found := strings.Cut(uri, "#")
	if !found {
		return uri, ""
	}
	return base, fragment
}

// baseURIDir extracts the directory from a base URI. If the base URI looks
// like a file path (last segment contains a dot), filepath.Dir is used.
// Otherwise the base URI itself is treated as a directory.
func baseURIDir(baseURI string) string {
	if strings.HasSuffix(baseURI, "/") || strings.HasSuffix(baseURI, string(filepath.Separator)) {
		return strings.TrimRight(baseURI, "/"+string([]byte{filepath.Separator}))
	}
	base := filepath.Base(baseURI)
	if strings.Contains(base, ".") {
		return filepath.Dir(baseURI)
	}
	// No extension in the last segment — treat the entire path as a directory.
	return baseURI
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
	// Convert file:// URIs to local paths.
	if strings.HasPrefix(cleanURI, "file:///") {
		return cleanURI[len("file://"):]
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

// nodeHasDocumentRoot reports whether n belongs to a tree rooted at a
// Document node (even if that document has no URL/base URI).
func nodeHasDocumentRoot(n helium.Node) bool {
	for n != nil {
		if _, ok := n.(*helium.Document); ok {
			return true
		}
		n = n.Parent()
	}
	return false
}

