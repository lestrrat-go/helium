package xslt3

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"sync"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
)

// invocationKind identifies the entry mode for a stylesheet invocation.
type invocationKind uint8

const (
	// invocationTransform is the default apply-templates entry with a source document.
	invocationTransform invocationKind = iota + 1
	// invocationApplyTemplates applies templates with explicit mode/selection control.
	invocationApplyTemplates
	// invocationCallTemplate calls a named template directly.
	invocationCallTemplate
	// invocationCallFunction calls a named function directly.
	invocationCallFunction
)

// OnMultipleMatchMode controls behavior when multiple templates match.
type OnMultipleMatchMode uint8

const (
	// OnMultipleMatchDefault uses the stylesheet's declared behavior.
	OnMultipleMatchDefault OnMultipleMatchMode = iota
	// OnMultipleMatchUseLast selects the last matching template.
	OnMultipleMatchUseLast
	// OnMultipleMatchFail raises XTRE0540 on ambiguous matches.
	OnMultipleMatchFail
)

func (m OnMultipleMatchMode) String() string {
	switch m {
	case OnMultipleMatchUseLast:
		return "use-last"
	case OnMultipleMatchFail:
		return "fail"
	default:
		return ""
	}
}

// Invocation configures and executes an XSLT transformation.
// It is a value-style wrapper: fluent methods return updated copies
// and the original is never mutated. Terminal methods (Do, Serialize,
// WriteTo) create an internal execCtx immediately.
type Invocation struct {
	cfg *invocationConfig
}

// invocationConfig holds the configuration for an Invocation.
type invocationConfig struct {
	ss *Stylesheet

	kind invocationKind

	source          *helium.Document
	initialTemplate string
	initialFunction string
	initialArgs     []xpath3.Sequence

	mode             string
	matchSelection   xpath3.Sequence
	parameters       *Parameters
	tunnelParameters *Parameters

	// initialTemplateParams and initialModeParams hold non-tunnel
	// xsl:with-param values for the initial template/mode invocation.
	initialTemplateParams *Parameters
	initialModeParams     *Parameters

	msgHandler          MessageHandler
	resultDocHandler    ResultDocumentHandler
	rawResultHandler    RawResultHandler
	primaryItemsHandler PrimaryItemsHandler
	annotationHandler   AnnotationHandler
	collectionResolver  xpath3.CollectionResolver
	uriResolver         xpath3.URIResolver
	httpClient          *http.Client
	baseOutputURI       string
	sourceSchemas       []*xsd.Schema
	onMultipleMatch     OnMultipleMatchMode
	traceWriter         io.Writer
	globalContextSelect string // XPath for global context item (evaluated post-strip-space)
	maxResourceBytes    int64  // per-resource read cap; 0 = inherit compiler/default, <0 = unbounded
	maxResourceBytesSet bool   // true once MaxResourceBytes is explicitly configured

	// resolved holds the effective output definition for the primary result
	// after a terminal method (Do/Serialize/WriteTo) completes. It is stored
	// behind a pointer with its own mutex so that concurrent terminal-method
	// calls on the same Invocation value (which share this *invocationConfig)
	// do not race when recording/reading the resolved output def.
	resolved *resolvedOutputState
}

// resolvedOutputState guards the effective output definition recorded by a
// terminal method. The mutex lives behind a pointer so copying
// invocationConfig (in clone) never copies a lock.
type resolvedOutputState struct {
	mu  sync.Mutex
	def *OutputDef
}

// store records the resolved primary output def. It deep-clones so the stored
// snapshot is independent of the per-call transformConfig.
func (r *resolvedOutputState) store(def *OutputDef) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.def = cloneOutputDef(def)
}

// snapshot returns an independent clone of the recorded output def, or nil if
// no terminal method has recorded one yet.
func (r *resolvedOutputState) snapshot() *OutputDef {
	r.mu.Lock()
	defer r.mu.Unlock()
	return cloneOutputDef(r.def)
}

func newInvocation(ss *Stylesheet, kind invocationKind) Invocation {
	return Invocation{cfg: &invocationConfig{ss: ss, kind: kind, resolved: &resolvedOutputState{}}}
}

func (inv Invocation) clone() Invocation {
	if inv.cfg == nil {
		return Invocation{cfg: &invocationConfig{resolved: &resolvedOutputState{}}}
	}
	cp := *inv.cfg
	// A cloned invocation is a distinct invocation; give it its own resolved
	// slot so recording a terminal result on the clone never writes through
	// to the source invocation's shared state.
	cp.resolved = &resolvedOutputState{}
	return Invocation{cfg: &cp}
}

// SourceDocument sets the source document for the invocation.
// This is needed for CallTemplate/CallFunction when the transform
// requires a source document for fn:doc("") or similar.
func (inv Invocation) SourceDocument(doc *helium.Document) Invocation {
	inv = inv.clone()
	inv.cfg.source = doc
	return inv
}

// Mode sets the initial mode for template matching.
func (inv Invocation) Mode(name string) Invocation {
	inv = inv.clone()
	inv.cfg.mode = name
	return inv
}

// Selection sets the initial match selection.
// Only valid for ApplyTemplates invocations; rejected at validation
// time for Transform, CallTemplate, and CallFunction.
func (inv Invocation) Selection(seq xpath3.Sequence) Invocation {
	inv = inv.clone()
	inv.cfg.matchSelection = seq
	return inv
}

// GlobalParameters sets the global stylesheet parameters.
// The Parameters collection is cloned.
func (inv Invocation) GlobalParameters(p *Parameters) Invocation {
	inv = inv.clone()
	inv.cfg.parameters = p.Clone()
	return inv
}

// TunnelParameters sets the tunnel parameters.
// The Parameters collection is cloned.
func (inv Invocation) TunnelParameters(p *Parameters) Invocation {
	inv = inv.clone()
	inv.cfg.tunnelParameters = p.Clone()
	return inv
}

// SetParameter sets a single global parameter value.
func (inv Invocation) SetParameter(name string, value xpath3.Sequence) Invocation {
	inv = inv.clone()
	if inv.cfg.parameters == nil {
		inv.cfg.parameters = NewParameters()
	} else {
		inv.cfg.parameters = inv.cfg.parameters.Clone()
	}
	inv.cfg.parameters.Set(name, value)
	return inv
}

// SetTunnelParameter sets a single tunnel parameter value.
func (inv Invocation) SetTunnelParameter(name string, value xpath3.Sequence) Invocation {
	inv = inv.clone()
	if inv.cfg.tunnelParameters == nil {
		inv.cfg.tunnelParameters = NewParameters()
	} else {
		inv.cfg.tunnelParameters = inv.cfg.tunnelParameters.Clone()
	}
	inv.cfg.tunnelParameters.Set(name, value)
	return inv
}

// SetInitialTemplateParameter sets a non-tunnel xsl:with-param for the
// initial template call (CallTemplate entry).
func (inv Invocation) SetInitialTemplateParameter(name string, value xpath3.Sequence) Invocation {
	inv = inv.clone()
	if inv.cfg.initialTemplateParams == nil {
		inv.cfg.initialTemplateParams = NewParameters()
	} else {
		inv.cfg.initialTemplateParams = inv.cfg.initialTemplateParams.Clone()
	}
	inv.cfg.initialTemplateParams.Set(name, value)
	return inv
}

// SetInitialModeParameter sets a non-tunnel xsl:with-param for the
// initial mode invocation (Transform / ApplyTemplates entry).
func (inv Invocation) SetInitialModeParameter(name string, value xpath3.Sequence) Invocation {
	inv = inv.clone()
	if inv.cfg.initialModeParams == nil {
		inv.cfg.initialModeParams = NewParameters()
	} else {
		inv.cfg.initialModeParams = inv.cfg.initialModeParams.Clone()
	}
	inv.cfg.initialModeParams.Set(name, value)
	return inv
}

// MessageHandler sets the handler for xsl:message output.
func (inv Invocation) MessageHandler(r MessageHandler) Invocation {
	inv = inv.clone()
	inv.cfg.msgHandler = r
	return inv
}

// ResultDocumentHandler sets the handler for secondary result documents
// produced by xsl:result-document.
func (inv Invocation) ResultDocumentHandler(r ResultDocumentHandler) Invocation {
	inv = inv.clone()
	inv.cfg.resultDocHandler = r
	return inv
}

// RawResultHandler sets the handler that receives the raw XDM result
// sequence from the primary output before serialization.
func (inv Invocation) RawResultHandler(r RawResultHandler) Invocation {
	inv = inv.clone()
	inv.cfg.rawResultHandler = r
	return inv
}

// PrimaryItemsHandler sets the handler that receives non-node items
// captured from the primary output (needed for json/adaptive serialization).
func (inv Invocation) PrimaryItemsHandler(r PrimaryItemsHandler) Invocation {
	inv = inv.clone()
	inv.cfg.primaryItemsHandler = r
	return inv
}

// AnnotationHandler sets the handler that receives type annotations and
// schema declarations from schema-aware transformations.
func (inv Invocation) AnnotationHandler(r AnnotationHandler) Invocation {
	inv = inv.clone()
	inv.cfg.annotationHandler = r
	return inv
}

// CollectionResolver sets a custom resolver for fn:collection.
func (inv Invocation) CollectionResolver(r xpath3.CollectionResolver) Invocation {
	inv = inv.clone()
	inv.cfg.collectionResolver = r
	return inv
}

// URIResolver sets a custom URI resolver used by fn:doc, fn:unparsed-text,
// fn:unparsed-text-lines, fn:unparsed-text-available, fn:doc-available,
// and fn:json-doc during the transformation. Without this (and without an
// HTTPClient) those functions cannot reach the filesystem or network and
// return their spec-mandated retrieval errors.
func (inv Invocation) URIResolver(r xpath3.URIResolver) Invocation {
	inv = inv.clone()
	inv.cfg.uriResolver = r
	return inv
}

// HTTPClient sets the HTTP client used to fetch http/https URIs for
// fn:doc / fn:unparsed-text / fn:json-doc when no URIResolver is supplied.
// The caller owns the client's transport, timeouts, and redirect policy.
// Without this (and without a URIResolver), network retrieval is refused.
func (inv Invocation) HTTPClient(client *http.Client) Invocation {
	inv = inv.clone()
	inv.cfg.httpClient = client
	return inv
}

// BaseOutputURI sets the base output URI for current-output-uri().
func (inv Invocation) BaseOutputURI(uri string) Invocation {
	inv = inv.clone()
	inv.cfg.baseOutputURI = uri
	return inv
}

// SourceSchemas sets pre-compiled XSD schemas for source validation.
// The slice is cloned; schema pointers are shared.
func (inv Invocation) SourceSchemas(schemas ...*xsd.Schema) Invocation {
	inv = inv.clone()
	inv.cfg.sourceSchemas = append([]*xsd.Schema(nil), schemas...)
	return inv
}

// OnMultipleMatch sets the on-multiple-match behavior.
func (inv Invocation) OnMultipleMatch(mode OnMultipleMatchMode) Invocation {
	inv = inv.clone()
	inv.cfg.onMultipleMatch = mode
	return inv
}

// TraceWriter sets the destination for fn:trace output during the
// transformation. When nil, fn:trace writes to os.Stderr.
func (inv Invocation) TraceWriter(w io.Writer) Invocation {
	inv = inv.clone()
	inv.cfg.traceWriter = w
	return inv
}

// GlobalContextSelect sets an XPath expression whose result (evaluated
// against the source document after whitespace stripping) determines the
// global context item.  If the expression evaluates to an empty sequence,
// the global context item is absent and global variables referencing "."
// will raise XPDY0002.
func (inv Invocation) GlobalContextSelect(expr string) Invocation {
	inv = inv.clone()
	inv.cfg.globalContextSelect = expr
	return inv
}

// MaxResourceBytes sets the maximum number of bytes read from a single
// external resource fetched at runtime through the configured URIResolver /
// HTTPClient. It governs the resource reads performed by XSLT's own loader —
// fn:doc / document(), fn:doc-available, xsl:source-document, xsl:merge,
// xsl:result-document parameter documents, xsi:schemaLocation source schemas,
// and fn:transform stylesheet / package sources — which fail with
// [ErrResourceTooLarge] when the cap is exceeded.
//
// A value of 0 inherits the cap configured on the Compiler (or the
// [MaxResourceBytes] default); a negative value disables the bound.
//
// The XPath built-ins fn:unparsed-text, fn:unparsed-text-lines, and
// fn:json-doc read through the xpath3 layer rather than the XSLT loader: they
// honor the same byte cap but do NOT surface [ErrResourceTooLarge]. An over-cap
// fn:unparsed-text / fn:unparsed-text-lines read surfaces FOUT1170
// (fn:unparsed-text-available returns false), and an over-cap fn:json-doc read
// surfaces FODC0002.
func (inv Invocation) MaxResourceBytes(n int64) Invocation {
	inv = inv.clone()
	inv.cfg.maxResourceBytes = n
	inv.cfg.maxResourceBytesSet = true
	return inv
}

// Do executes the transformation and returns the principal result document.
func (inv Invocation) Do(ctx context.Context) (*helium.Document, error) {
	if err := inv.validate(); err != nil {
		return nil, err
	}
	tcfg := inv.toTransformConfig()
	doc, err := executeTransform(ctx, inv.cfg.source, inv.cfg.ss, tcfg)
	inv.cfg.resolved.store(tcfg.resolvedOutputDef)
	return doc, err
}

// ResolvedOutputDef returns the effective output definition for the primary
// result after a terminal method (Do, Serialize, WriteTo) has been called.
// It includes runtime overrides from xsl:result-document targeting the
// primary output. Returns nil if no terminal method has been called yet.
func (inv Invocation) ResolvedOutputDef() *OutputDef {
	if inv.cfg == nil || inv.cfg.resolved == nil {
		return nil
	}
	// Return an independent snapshot so callers cannot mutate shared state and
	// so concurrent terminal-method calls remain safe.
	return inv.cfg.resolved.snapshot()
}

// Serialize executes the transformation and returns the serialized result.
// Secondary result documents are delivered through the handler only.
func (inv Invocation) Serialize(ctx context.Context) (string, error) {
	var buf strings.Builder
	if err := inv.WriteTo(ctx, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// WriteTo executes the transformation and writes the serialized result to w.
// Secondary result documents are delivered through the handler only.
func (inv Invocation) WriteTo(ctx context.Context, w io.Writer) error {
	if err := inv.validate(); err != nil {
		return err
	}
	tcfg := inv.toTransformConfig()
	resultDoc, err := executeTransform(ctx, inv.cfg.source, inv.cfg.ss, tcfg)
	inv.cfg.resolved.store(tcfg.resolvedOutputDef)
	if err != nil {
		return err
	}
	return SerializeResult(w, resultDoc, tcfg.resolvedOutputDef)
}

// validate checks that the invocation config is valid for the entry kind.
func (inv Invocation) validate() error {
	c := inv.cfg
	if c == nil {
		return errZeroInvocation
	}
	if c.ss == nil {
		return errNilStylesheet
	}
	switch c.kind {
	case invocationTransform:
		// nil source is allowed: the stylesheet may use xsl:source-document,
		// global-context-item use="absent", or an initial template.
		// executeTransform will raise XTDE0040 if source is truly needed.
		if c.matchSelection != nil {
			return fmt.Errorf("xslt3: Selection is not valid for Transform (use ApplyTemplates)")
		}
		if c.initialTemplateParams != nil {
			return fmt.Errorf("xslt3: SetInitialTemplateParameter is not valid for Transform (use CallTemplate)")
		}
	case invocationApplyTemplates:
		// nil source is allowed when a match selection is provided, or when
		// the stylesheet does not require a source document.
		// executeTransform will raise XTDE0040 if needed.
		if c.initialTemplateParams != nil {
			return fmt.Errorf("xslt3: SetInitialTemplateParameter is not valid for ApplyTemplates (use CallTemplate)")
		}
	case invocationCallTemplate:
		if c.initialTemplate == "" {
			return fmt.Errorf("xslt3: CallTemplate requires a template name")
		}
		if c.mode != "" {
			return fmt.Errorf("xslt3: Mode is not valid for CallTemplate")
		}
		if c.matchSelection != nil {
			return fmt.Errorf("xslt3: Selection is not valid for CallTemplate")
		}
		if c.initialModeParams != nil {
			return fmt.Errorf("xslt3: SetInitialModeParameter is not valid for CallTemplate (use Transform or ApplyTemplates)")
		}
	case invocationCallFunction:
		if c.initialFunction == "" {
			return fmt.Errorf("xslt3: CallFunction requires a function name")
		}
		if c.mode != "" {
			return fmt.Errorf("xslt3: Mode is not valid for CallFunction")
		}
		if c.matchSelection != nil {
			return fmt.Errorf("xslt3: Selection is not valid for CallFunction")
		}
		if c.initialTemplateParams != nil {
			return fmt.Errorf("xslt3: SetInitialTemplateParameter is not valid for CallFunction (use CallTemplate)")
		}
		if c.initialModeParams != nil {
			return fmt.Errorf("xslt3: SetInitialModeParameter is not valid for CallFunction (use Transform or ApplyTemplates)")
		}
		if c.tunnelParameters != nil {
			return fmt.Errorf("xslt3: TunnelParameters is not valid for CallFunction")
		}
	default:
		return fmt.Errorf("xslt3: invalid invocation kind %d", c.kind)
	}
	return nil
}

// toTransformConfig converts the Invocation config to the internal
// transformConfig used by executeTransform.
func (inv Invocation) toTransformConfig() *transformConfig {
	c := inv.cfg
	tcfg := &transformConfig{
		collectionResolver: c.collectionResolver,
		uriResolver:        c.uriResolver,
		httpClient:         c.httpClient,
		baseOutputURI:      c.baseOutputURI,
		sourceSchemas:      c.sourceSchemas,
		onMultipleMatch:    c.onMultipleMatch.String(),
		traceWriter:        c.traceWriter,
	}

	// Resource cap: a non-zero explicit per-invocation setting wins; an explicit
	// 0 (or no setting at all) inherits the cap configured on the Compiler
	// (stored on the stylesheet). A negative value disables the bound; a stylesheet
	// value of 0 then falls through to the MaxResourceBytes default at the read
	// sites. This keeps MaxResourceBytes(0) meaning "inherit" per its contract.
	switch {
	case c.maxResourceBytesSet && c.maxResourceBytes != 0:
		tcfg.maxResourceBytes = c.maxResourceBytes
	case c.ss != nil:
		tcfg.maxResourceBytes = c.ss.maxResourceBytes
	}

	// Entry mode
	switch c.kind {
	case invocationCallTemplate:
		tcfg.initialTemplate = c.initialTemplate
	case invocationCallFunction:
		tcfg.initialFunction = c.initialFunction
		tcfg.initialFunctionParams = c.initialArgs
	case invocationApplyTemplates:
		if c.mode != "" {
			tcfg.initialMode = c.mode
		}
		if c.matchSelection != nil {
			tcfg.initialMatchSelection = c.matchSelection
		}
	case invocationTransform:
		if c.mode != "" {
			tcfg.initialMode = c.mode
		}
	}

	// Parameters: map from Parameters carrier to the three-tier param maps
	if c.parameters != nil {
		tcfg.sequenceParams = maps.Clone(c.parameters.toMap())
	}

	// Initial template/mode non-tunnel params
	if c.initialTemplateParams != nil {
		tcfg.initialTemplateParams = maps.Clone(c.initialTemplateParams.toMap())
	}
	if c.initialModeParams != nil {
		tcfg.initialModeParams = maps.Clone(c.initialModeParams.toMap())
	}

	// Tunnel parameters → initial mode tunnel or initial template tunnel
	if c.tunnelParameters != nil {
		tunnel := maps.Clone(c.tunnelParameters.toMap())
		switch c.kind {
		case invocationCallTemplate:
			tcfg.initialTemplateTunnel = tunnel
		default:
			tcfg.initialModeTunnel = tunnel
		}
	}

	// Handlers
	tcfg.msgHandler = c.msgHandler
	tcfg.resultDocHandler = c.resultDocHandler
	tcfg.rawResultHandler = c.rawResultHandler
	tcfg.primaryItemsHandler = c.primaryItemsHandler
	tcfg.annotationHandler = c.annotationHandler

	// Global context select
	tcfg.globalContextSelect = c.globalContextSelect

	return tcfg
}
