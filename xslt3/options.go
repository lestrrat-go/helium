package xslt3

import (
	"io"
	"net/http"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
)

// URIResolver resolves URIs to readable content. Used for xsl:import
// and xsl:include during compilation.
type URIResolver interface {
	Resolve(uri string) (io.ReadCloser, error)
}

// xpathURIResolverAdapter adapts a compile-time xslt3 URIResolver (method
// Resolve) to the xpath3.URIResolver interface (method ResolveURI), so a
// compile-time use-when expression that calls doc-available()/doc() retrieves
// resources through the same resolver that loads stylesheet modules.
type xpathURIResolverAdapter struct {
	r URIResolver
}

func (a xpathURIResolverAdapter) ResolveURI(uri string) (io.ReadCloser, error) {
	return a.r.Resolve(uri)
}

// PackageResolver resolves package name URIs to file paths or readers.
// Used during compilation when xsl:use-package is encountered.
type PackageResolver interface {
	ResolvePackage(name string, version string) (io.ReadCloser, string, error)
}

// --- Compile configuration (internal) ---

type compileConfig struct {
	baseURI          string
	resolver         URIResolver
	packageResolver  PackageResolver
	staticParams     map[string]xpath3.Sequence // externally supplied static param values
	importSchemas    []*xsd.Schema              // pre-compiled schemas for xsl:import-schema resolution
	isSubPackage     bool                       // true when compiling a sub-package (via xsl:use-package)
	maxResourceBytes int64                      // per-resource read cap; 0 = MaxResourceBytes default, <0 = unbounded
	// allowExternalEntities opts into the legacy permissive parse of external
	// stylesheet modules (resolver-mediated external entity / DTD loading).
	// Default false: XXE is blocked.
	allowExternalEntities bool
	// parser is the caller-injected base parser governing parse policy for
	// stylesheet/schema parsing. nil = use the hardened default.
	parser *helium.Parser
}

// --- Transform configuration (internal) ---

type transformConfig struct {
	sequenceParams        map[string]xpath3.Sequence
	msgHandler            MessageHandler
	initialTemplate       string
	initialMode           string
	initialModeParams     map[string]xpath3.Sequence
	initialModeTunnel     map[string]xpath3.Sequence
	initialTemplateParams map[string]xpath3.Sequence
	initialTemplateTunnel map[string]xpath3.Sequence
	initialFunction       string            // QName of initial function
	initialFunctionParams []xpath3.Sequence // positional params for initial function
	resultDocHandler      ResultDocumentHandler
	collectionResolver    xpath3.CollectionResolver
	uriResolver           xpath3.URIResolver // resolver for fn:doc, fn:unparsed-text, fn:json-doc
	httpClient            *http.Client       // explicit HTTP client for fn:doc/fn:unparsed-text — opt-in network
	onMultipleMatch       string             // "use-last" or "fail" — overrides default mode's on-multiple-match
	rawResultHandler      RawResultHandler
	baseOutputURI         string // base output URI for current-output-uri()
	annotationHandler     AnnotationHandler
	initialMatchSelection xpath3.Sequence // initial match selection for apply-templates entry
	rawCapture            bool            // enable captureItems on output frame for raw delivery
	rawCapturedItems      xpath3.Sequence // items captured during raw-delivery transform (set by executeTransform)
	primaryItems          xpath3.Sequence // items captured from primary output for json/adaptive serialization
	primaryItemsHandler   PrimaryItemsHandler
	sourceSchemas         []*xsd.Schema // pre-compiled schemas for source document validation
	traceWriter           io.Writer     // destination for fn:trace output (nil = os.Stderr)
	resolvedOutputDef     *OutputDef    // resolved primary output def (set by executeTransform)
	globalContextSelect   string        // XPath for global context item (evaluated after strip-space)
	globalContextItem     xpath3.Item   // explicit global context item (fn:transform global-context-item option)
	maxResourceBytes      int64         // per-resource read cap; 0 = MaxResourceBytes default, <0 = unbounded
	// allowExternalEntities opts into the legacy permissive parse of runtime
	// documents loaded via fn:doc/document()/xsl:source-document/xsl:merge
	// (resolver-mediated external entity / DTD loading). Default false: XXE is
	// blocked.
	allowExternalEntities bool
	// parser is the caller-injected base parser governing parse policy for
	// runtime source / fn:doc / document() parses. nil = use the hardened
	// default.
	parser *helium.Parser
}

// MessageHandler handles xsl:message output during transformation.
// A non-nil error aborts the transform immediately.
//
// Handler methods are called from the goroutine executing Do/Serialize/WriteTo.
// If you run transforms concurrently, your implementation must be safe for
// concurrent use.
type MessageHandler interface {
	HandleMessage(msg string, terminate bool) error
}

// ResultDocumentHandler handles secondary result documents produced
// by xsl:result-document. The outDef contains the effective output
// definition (method, encoding, indent, etc.) for this result document.
// A non-nil error aborts the transform.
//
// Handler methods are called from the goroutine executing Do/Serialize/WriteTo.
// If you run transforms concurrently, your implementation must be safe for
// concurrent use.
type ResultDocumentHandler interface {
	HandleResultDocument(href string, doc *helium.Document, outDef *OutputDef) error
}

// RawResultHandler receives the raw XDM result sequence from the primary
// output before it is serialized into the result document tree.
// A non-nil error aborts the transform.
//
// Handler methods are called from the goroutine executing Do/Serialize/WriteTo.
// If you run transforms concurrently, your implementation must be safe for
// concurrent use.
type RawResultHandler interface {
	HandleRawResult(seq xpath3.Sequence) error
}

// PrimaryItemsHandler receives non-node items captured from the primary
// output during transformation (needed for json/adaptive serialization).
// A non-nil error aborts the transform.
//
// Handler methods are called from the goroutine executing Do/Serialize/WriteTo.
// If you run transforms concurrently, your implementation must be safe for
// concurrent use.
type PrimaryItemsHandler interface {
	HandlePrimaryItems(seq xpath3.Sequence) error
}

// AnnotationHandler receives type annotations and schema declarations
// from schema-aware transformations. A non-nil error aborts the transform.
//
// Handler methods are called from the goroutine executing Do/Serialize/WriteTo.
// If you run transforms concurrently, your implementation must be safe for
// concurrent use.
type AnnotationHandler interface {
	HandleAnnotations(annotations map[helium.Node]string, declarations xpath3.SchemaDeclarations) error
}
