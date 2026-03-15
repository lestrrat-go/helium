package xslt3

import (
	"io"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
)

// URIResolver resolves URIs to readable content. Used for xsl:import
// and xsl:include during compilation.
type URIResolver interface {
	Resolve(uri string) (io.ReadCloser, error)
}

// PackageResolver resolves package name URIs to file paths or readers.
// Used during compilation when xsl:use-package is encountered.
type PackageResolver interface {
	ResolvePackage(name string, version string) (io.ReadCloser, string, error)
}

// --- Compile configuration (internal) ---

type compileConfig struct {
	baseURI         string
	resolver        URIResolver
	packageResolver PackageResolver
	staticParams    map[string]xpath3.Sequence // externally supplied static param values
	importSchemas   []*xsd.Schema             // pre-compiled schemas for xsl:import-schema resolution
}

// --- Transform configuration (internal) ---

type transformConfig struct {
	sequenceParams     map[string]xpath3.Sequence
	msgHandler         MessageHandler
	initialTemplate    string
	initialMode        string
	initialModeParams       map[string]xpath3.Sequence
	initialModeTunnel       map[string]xpath3.Sequence
	initialTemplateParams   map[string]xpath3.Sequence
	initialTemplateTunnel   map[string]xpath3.Sequence
	initialFunction        string              // QName of initial function
	initialFunctionParams  []xpath3.Sequence   // positional params for initial function
	resultDocHandler   ResultDocumentHandler
	collectionResolver xpath3.CollectionResolver
	onMultipleMatch    string // "use-last" or "fail" — overrides default mode's on-multiple-match
	rawResultHandler        RawResultHandler
	baseOutputURI           string // base output URI for current-output-uri()
	annotationHandler       AnnotationHandler
	initialMatchSelection    xpath3.Sequence // initial match selection for apply-templates entry
	rawCapture               bool            // enable captureItems on output frame for raw delivery
	rawCapturedItems         xpath3.Sequence // items captured during raw-delivery transform (set by executeTransform)
	primaryItems             xpath3.Sequence // items captured from primary output for json/adaptive serialization
	primaryItemsHandler     PrimaryItemsHandler
	sourceSchemas            []*xsd.Schema // pre-compiled schemas for source document validation
	traceWriter              io.Writer     // destination for fn:trace output (nil = os.Stderr)
	resolvedOutputDef        *OutputDef    // resolved primary output def (set by executeTransform)
}
