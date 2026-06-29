package xslt3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/iofs"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/internal/uripath"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
)

// XSLT/XPath function names referenced from multiple sites. Kept as constants
// alongside xsltFunctionArities (the single source of truth for arities) so the
// repeated literals stay consistent across the package.
const (
	fnNameCurrent            = "current"
	fnNameDocument           = "document"
	fnNameElementAvailable   = "element-available"
	fnNameFunctionAvailable  = "function-available"
	fnNameTypeAvailable      = "type-available"
	fnNameCurrentGroup       = "current-group"
	fnNameCurrentGroupingKey = "current-grouping-key"
	fnNameCopyOf             = "copy-of"
	fnNameTransform          = "transform"
	fnNameCurrentOutputURI   = "current-output-uri"
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
		fnNameCurrent:               &xsltFunc{min: 0, max: 0, fn: ec.fnCurrent, noDynRef: true, dynRefError: errCodeXTDE1360},
		fnNameDocument:              &xsltFunc{min: 1, max: 2, fn: ec.fnDocument},
		"key":                       &xsltFunc{min: 2, max: 3, fn: ec.fnKey},
		"generate-id":               &xsltFunc{min: 0, max: 1, fn: ec.fnGenerateID},
		funcSystemProperty:          &xsltFunc{min: 1, max: 1, fn: ec.fnSystemProperty},
		"unparsed-entity-uri":       &xsltFunc{min: 1, max: 2, fn: ec.fnUnparsedEntityURI},
		"unparsed-entity-public-id": &xsltFunc{min: 1, max: 2, fn: ec.fnUnparsedEntityPublicID},
		fnNameElementAvailable:      &xsltFunc{min: 1, max: 1, fn: ec.fnElementAvailable},
		fnNameFunctionAvailable:     &xsltFunc{min: 1, max: 2, fn: ec.fnFunctionAvailable},
		fnNameTypeAvailable:         &xsltFunc{min: 1, max: 1, fn: ec.fnTypeAvailable},
		fnNameCurrentGroup:          &xsltFunc{min: 0, max: 0, fn: ec.fnCurrentGroup, noDynRef: true, dynRefError: errCodeXTDE1061},
		fnNameCurrentGroupingKey:    &xsltFunc{min: 0, max: 0, fn: ec.fnCurrentGroupingKey, noDynRef: true, dynRefError: errCodeXTDE1071},
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
		fnNameCopyOf:                  &xsltFunc{min: 0, max: 1, fn: ec.fnCopyOf},
		funcSnapshot:                  &xsltFunc{min: 0, max: 1, fn: ec.fnSnapshot},
		"regex-group":                 &regexGroupFunc{ec: ec},
		fnNameTransform:               &xsltFunc{min: 1, max: 1, fn: ec.fnTransform},
		funcAvailableSystemProperties: &xsltFunc{min: 0, max: 0, fn: ec.fnAvailableSystemProperties},
		"stream-available":            &xsltFunc{min: 1, max: 1, fn: ec.fnStreamAvailable},
		fnNameCurrentOutputURI:        &xsltFunc{min: 0, max: 0, fn: ec.fnCurrentOutputURI},
	}
	return ec.cachedFns
}

// xsltFunctionArities is the static arity table for XSLT-defined functions that
// live in the fn: namespace. It is the single source of truth consulted by
// compile-time checks (e.g. validatePatternFunctions) that run without an
// execContext. The runtime registries in xsltFunctions / xsltFunctionsNS must
// stay consistent with this table; TestXSLTFunctionAritiesMatchRegistry guards
// against drift.
var xsltFunctionArities = map[string][2]int{
	"nilled":                      {0, 1},
	fnNameCurrent:                 {0, 0},
	fnNameDocument:                {1, 2},
	"doc":                         {1, 1},
	"key":                         {2, 3},
	"generate-id":                 {0, 1},
	funcSystemProperty:            {1, 1},
	"unparsed-entity-uri":         {1, 2},
	"unparsed-entity-public-id":   {1, 2},
	fnNameElementAvailable:        {1, 1},
	fnNameFunctionAvailable:       {1, 2},
	fnNameTypeAvailable:           {1, 1},
	fnNameCurrentGroup:            {0, 0},
	fnNameCurrentGroupingKey:      {0, 0},
	"current-merge-group":         {0, 1},
	"current-merge-key":           {0, 0},
	"accumulator-before":          {1, 1},
	"accumulator-after":           {1, 1},
	fnNameCopyOf:                  {0, 1},
	funcSnapshot:                  {0, 1},
	"regex-group":                 {1, 1},
	fnNameTransform:               {1, 1},
	funcAvailableSystemProperties: {0, 0},
	"stream-available":            {1, 1},
	fnNameCurrentOutputURI:        {0, 0},
}

// xsltFunctionAcceptsArity reports whether an XSLT-defined function (fn:
// namespace) with the given local name exists and accepts the given arity.
func xsltFunctionAcceptsArity(name string, arity int) bool {
	bounds, ok := xsltFunctionArities[name]
	if !ok {
		return false
	}
	return arity >= bounds[0] && arity <= bounds[1]
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
		return nil, nil //nolint:nilnil
	} else {
		ni, ok := args[0].Get(0).(xpath3.NodeItem)
		if !ok {
			return nil, nil //nolint:nilnil
		}
		node = ni.Node
	}
	if node == nil || node.Type() != helium.ElementNode {
		return nil, nil //nolint:nilnil
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
				baseDir = documentBaseDir(nodeBase)
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
					itemBaseDir = documentBaseDir(nodeBase)
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
		if _, frag, ok := strings.Cut(uri, "#"); ok {
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
		baseDir = documentBaseDir(effectiveBase)
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

	cacheKey := ec.docCacheKey(resolvedURI)
	if doc, ok := ec.docCache[cacheKey]; ok {
		return doc, nil
	}

	data, err := ec.retrieveDocumentBytes(ctx, resolvedURI)
	if err != nil {
		// XTDE1160: when the URI contains a fragment identifier, use XTDE1160
		// (fragment identifier error) instead of FODC0002.
		if strings.ContainsRune(uri, '#') {
			return nil, dynamicErrorCause(errCodeXTDE1160, err,
				"cannot load document %q with fragment identifier: %v", uri, err)
		}
		return nil, dynamicErrorCause(errCodeFODC0002, err, "cannot load document %q: %v", uri, err)
	}
	// helium's parser currently rejects literal U+FFFD from RuneCursor-backed
	// input, but accepts the equivalent character reference. Normalizing here
	// preserves the infoset for XSLT document() loads over the W3C source tree.
	data = bytes.ReplaceAll(data, []byte("\uFFFD"), []byte("&#xFFFD;"))

	// Parse the retrieved document with XXE blocked by default: external DTDs
	// and external general entities are neither loaded nor substituted, which
	// eliminates the local-file-disclosure / SSRF vector. Callers that need the
	// legacy permissive behavior (e.g. XSLT 3.0 W3C tests such as base-uri-051
	// that resolve external SYSTEM entities) must opt in via
	// Invocation.AllowExternalEntities(true).
	doc, err := parseExternalXML(ctx, ec.injectedParser(), data, resolvedURI, ec.allowExternalEntities(),
		ec.retrieveDocumentBytes,
		func(p helium.Parser) helium.Parser {
			return p.DefaultDTDAttributes(true).FixBaseURIs(false)
		}, ec.resourceLimit())
	if err != nil {
		return nil, dynamicError(errCodeFODC0002, "cannot parse document %q: %v", uri, err)
	}

	doc.SetURL(resolvedURI)

	// Apply xsl:strip-space rules to the loaded document so that
	// whitespace-only text nodes are removed consistently with how
	// the source document is treated.
	if len(ec.effectiveStripSpace()) > 0 {
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
	ec.docCache[cacheKey] = doc

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

// retrieveDocumentBytes fetches the bytes at resolvedURI using the
// transformation's configured URIResolver / HTTPClient. There is no
// implicit os.ReadFile and no implicit http.DefaultClient fallback —
// retrieval is opt-in. Callers grant access by setting
// [Invocation.URIResolver] and/or [Invocation.HTTPClient]. Mirrors the
// secure-by-default fn:doc retrieval landed in #417 for xpath3.
func (ec *execContext) retrieveDocumentBytes(ctx context.Context, resolvedURI string) ([]byte, error) {
	// URI schemes are case-insensitive per RFC 3986; url.Parse lowercases
	// .Scheme so the equality compares are scheme-correct regardless of
	// how the caller spelled "HTTP" / "Https" / ...
	var isHTTP bool
	if u, err := url.Parse(resolvedURI); err == nil {
		isHTTP = u.Scheme == lexicon.SchemeHTTP || u.Scheme == lexicon.SchemeHTTPS
	}
	var resolver xpath3.URIResolver
	var httpClient *http.Client
	if ec.transformConfig != nil {
		resolver = ec.transformConfig.uriResolver
		httpClient = ec.transformConfig.httpClient
	}

	limit := ec.resourceLimit()

	if isHTTP {
		if httpClient != nil {
			return fetchHTTPBytes(ctx, httpClient, resolvedURI, limit)
		}
		if resolver != nil {
			return fetchViaResolver(resolver, resolvedURI, limit)
		}
		return nil, fmt.Errorf("no HTTPClient or URIResolver configured for %q", resolvedURI)
	}

	if resolver != nil {
		return fetchViaResolver(resolver, resolvedURI, limit)
	}
	return nil, fmt.Errorf("no URIResolver configured for %q", resolvedURI)
}

// resourceLimit returns the per-resource read cap for runtime resolver/HTTP
// reads, taken from the transformConfig (0 = MaxResourceBytes default).
func (ec *execContext) resourceLimit() int64 {
	if ec != nil && ec.transformConfig != nil {
		return ec.transformConfig.maxResourceBytes
	}
	return 0
}

// allowExternalEntities reports whether the legacy permissive parse of runtime
// documents (resolver-mediated external entity / DTD loading) is opted into for
// this transformation. Default false: XXE is blocked.
func (ec *execContext) allowExternalEntities() bool {
	return ec != nil && ec.transformConfig != nil && ec.transformConfig.allowExternalEntities
}

// injectedParser returns the caller-injected base parser governing parse policy
// for runtime XML parses (nil = hardened default).
func (ec *execContext) injectedParser() *helium.Parser {
	if ec != nil && ec.transformConfig != nil {
		return ec.transformConfig.parser
	}
	return nil
}

func fetchViaResolver(r xpath3.URIResolver, uri string, limit int64) ([]byte, error) {
	rc, err := r.ResolveURI(uri)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return readResourceBounded(rc, limit)
}

// retrieveDocumentPrefix opens the resource through the configured
// URIResolver / HTTPClient and reads at most n leading bytes before
// closing. It exists so availability/format probes (e.g. fn:stream-available)
// can avoid a full io.ReadAll of large or remote documents. The
// default-deny posture matches retrieveDocumentBytes: with no resolver /
// HTTPClient configured, retrieval is refused.
func (ec *execContext) retrieveDocumentPrefix(ctx context.Context, resolvedURI string, n int) ([]byte, error) {
	var isHTTP bool
	if u, err := url.Parse(resolvedURI); err == nil {
		isHTTP = u.Scheme == lexicon.SchemeHTTP || u.Scheme == lexicon.SchemeHTTPS
	}
	var resolver xpath3.URIResolver
	var httpClient *http.Client
	if ec.transformConfig != nil {
		resolver = ec.transformConfig.uriResolver
		httpClient = ec.transformConfig.httpClient
	}

	if isHTTP {
		if httpClient != nil {
			return fetchHTTPPrefix(ctx, httpClient, resolvedURI, n)
		}
		if resolver != nil {
			return fetchPrefixViaResolver(resolver, resolvedURI, n)
		}
		return nil, fmt.Errorf("no HTTPClient or URIResolver configured for %q", resolvedURI)
	}

	if resolver != nil {
		return fetchPrefixViaResolver(resolver, resolvedURI, n)
	}
	return nil, fmt.Errorf("no URIResolver configured for %q", resolvedURI)
}

// readPrefix reads at most n bytes from r. A short read (io.EOF before n
// bytes) is not an error — it just means the resource is smaller than n.
func readPrefix(r io.Reader, n int) ([]byte, error) {
	buf := make([]byte, n)
	read, err := io.ReadFull(r, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	return buf[:read], nil
}

func fetchPrefixViaResolver(r xpath3.URIResolver, uri string, n int) ([]byte, error) {
	rc, err := r.ResolveURI(uri)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return readPrefix(rc, n)
}

func fetchHTTPPrefix(ctx context.Context, client *http.Client, uri string, n int) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d for %q", resp.StatusCode, uri)
	}
	return readPrefix(resp.Body, n)
}

func fetchHTTPBytes(ctx context.Context, client *http.Client, uri string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d for %q", resp.StatusCode, uri)
	}
	return readResourceBounded(resp.Body, limit)
}

// resolveAgainstBaseURI resolves a relative URI against an effective base URI.
// The base URI may be a file path (e.g., /a/b/style.xsl) or a directory-like
// path from xml:base processing (e.g., /a/b). For file paths (containing a
// dot-extension in the last segment), the directory part is extracted via
// filepath.Dir. For directory-like paths, the path is used directly.
//
// Absoluteness is decided with [xsd.URIScheme] (RFC 3986), not filepath.IsAbs
// or a "://" substring check: an absolute-URI reference may carry a scheme with
// no "//" authority (e.g. "urn:shared", "file:/docs/d.xml") and must be returned
// unchanged, while a relative reference against a URI base must keep the base
// scheme/authority. Only when both base and ref are local filesystem paths is
// filepath.Join used. Resolution of the URI cases is delegated to the shared
// canonical [xsd.ResolveSchemaURI] helper.
func resolveAgainstBaseURI(uri string, baseURI string) string {
	if uri == "" || baseURI == "" {
		return uri
	}
	// Absolute-URI ref, or any ref against a URI base: defer to the shared
	// canonical resolver (RFC 3986 + OmitHost preservation). On error, fall
	// back to the raw ref rather than producing a host-dropping filepath join.
	if xsd.URIScheme(uri) != "" || xsd.URIScheme(baseURI) != "" {
		resolved, err := xsd.ResolveSchemaURI(uri, baseURI)
		if err != nil {
			return uri
		}
		return resolved
	}
	// Both base and ref are local filesystem paths. Resolve with forward-slash
	// (path) semantics so the result uses '/' on every OS; uripath.IsAbsolutePath
	// recognizes both POSIX- and Windows-absolute refs regardless of GOOS.
	if uripath.IsAbsolutePath(uri) {
		return uri
	}
	baseDir := baseURIDir(baseURI)
	return uripath.JoinLocalBaseDir(baseDir, uri)
}

func splitURIFragment(uri string) (string, string) {
	base, fragment, found := strings.Cut(uri, "#")
	if !found {
		return uri, ""
	}
	return base, fragment
}

// documentBaseDir derives the base passed to resolveDocumentURI /
// resolveAgainstBaseURI for a runtime document base such as a stylesheet or
// node base URI.
//
// For a URI base (it has a scheme per [xsd.URIScheme]) the FULL base URI is
// returned unchanged: the URI-aware resolvers delegate to [xsd.ResolveSchemaURI],
// which performs RFC 3986 resolution and replaces the base's last path segment
// itself. Applying filepath.Dir here would instead collapse the "//" authority
// separator (e.g. "mem://pkg/main.xsl" -> "mem:/pkg"), dropping the host so a
// sibling "doc.xml" wrongly resolves to "mem:/pkg/doc.xml" instead of
// "mem://pkg/doc.xml".
//
// For a genuine local filesystem base the containing directory is derived with
// path.Dir over the forward-slash-normalized base ([uripath.ToSlash]) — the
// SAME derivation compile-time module resolution uses for a local baseURI (see
// resolveModuleHref in compile_imports.go), so a relative reference resolves to
// the same directory whether the surrounding expression is compiled statically
// or evaluated dynamically. A Compiler.BaseURI / stylesheet FILE base such as
// "/styles/main" is treated as a file path: its last segment is dropped
// ("/styles"), so doc("data.xml") lands at "/styles/data.xml". A genuine
// directory-form base (a trailing-slash path such as "…/tests/fn/", as an
// xml:base=".." override resolves to) is still handled correctly: path.Dir
// collapses the trailing slash and keeps the directory ("…/tests/fn/" ->
// "…/tests/fn"). Forward-slash semantics keep the result '/'-separated on every
// OS (on Windows filepath.Dir would emit '\' and corrupt the later slash-based
// join).
func documentBaseDir(base string) string {
	if base == "" {
		return ""
	}
	if xsd.URIScheme(base) != "" {
		return base
	}
	return path.Dir(uripath.ToSlash(base))
}

// baseURIDir extracts the directory from a local-filesystem base URI in
// forward-slash form. If the base looks like a file path (last segment contains
// a dot) the last segment is dropped; otherwise the base itself is treated as a
// directory. uripath.LocalBaseDir performs this in slash space on every OS, so
// the result never gains backslashes on Windows.
func baseURIDir(baseURI string) string {
	return uripath.LocalBaseDir(baseURI)
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
	// Convert file:// URIs to local paths. Use iofs.FileURIToPath so a Windows
	// drive-letter URI ("file:///D:/a/b") yields a drive path ("D:\\a\\b" on
	// Windows) rather than a spurious leading-slash path ("/D:/a/b"). The result
	// is then normalized with uripath.ToSlash so a non-drive POSIX-shaped URI
	// ("file:///abs/x.xml") resolves to a forward-slash path ("/abs/x.xml") on
	// EVERY OS — FileURIToPath would otherwise emit "\\abs\\x.xml" on Windows.
	// The forward-slash form is what every other helium resolver consumes (Win32
	// accepts '/'); POSIX behavior is unchanged. On parse failure, fall back to
	// the original strip.
	if strings.HasPrefix(cleanURI, "file:///") {
		if p, err := iofs.FileURIToPath(cleanURI); err == nil {
			return uripath.ToSlash(p)
		}
		// On conversion failure (e.g. a "file:////server/share" UNC URI that
		// FileURIToPath rejects) do NOT strip the "file://" prefix: that would
		// yield "//server/share/..." which becomes a UNC path on Windows,
		// bypassing the local-only policy. Return the original file: URI so a
		// downstream local-path loader rejects it.
		return cleanURI
	}
	// Decide absoluteness with xsd.URIScheme (RFC 3986), not a "://" substring
	// check or filepath.IsAbs: an absolute-URI ref may carry a scheme with no
	// "//" authority (e.g. document('urn:doc'), doc('file:/x.xml')) and must be
	// returned unchanged, while a relative ref against a URI base must keep the
	// base scheme/authority (resolved per RFC 3986) instead of being joined as a
	// local path. Only when both base and ref are local filesystem paths is
	// filepath.Join used. Resolution of the URI cases is delegated to the shared
	// canonical xsd.ResolveSchemaURI helper.
	if xsd.URIScheme(cleanURI) != "" || (baseDir != "" && xsd.URIScheme(baseDir) != "") {
		resolved, err := xsd.ResolveSchemaURI(cleanURI, baseDir)
		if err != nil {
			return cleanURI
		}
		return resolved
	}
	// Both local: resolve with forward-slash (path) semantics so the result uses
	// '/' on every OS. uripath.IsAbsolutePath recognizes both POSIX- and
	// Windows-absolute refs regardless of GOOS.
	if uripath.IsAbsolutePath(cleanURI) {
		return cleanURI
	}
	if baseDir != "" {
		return uripath.JoinLocalBaseDir(baseDir, cleanURI)
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
