package xslt3

import (
	"context"
	"fmt"
	"io"
	"maps"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
)

// InvocationKind identifies the entry mode for a stylesheet invocation.
type InvocationKind uint8

const (
	// InvocationTransform is the default apply-templates entry with a source document.
	InvocationTransform InvocationKind = iota + 1
	// InvocationApplyTemplates applies templates with explicit mode/selection control.
	InvocationApplyTemplates
	// InvocationCallTemplate calls a named template directly.
	InvocationCallTemplate
	// InvocationCallFunction calls a named function directly.
	InvocationCallFunction
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
	cfg *invocationCfg
}

type invocationCfg struct {
	ss *Stylesheet

	kind InvocationKind

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

	receiver           any
	collectionResolver xpath3.CollectionResolver
	baseOutputURI      string
	sourceSchemas      []*xsd.Schema
	onMultipleMatch    OnMultipleMatchMode
}

func newInvocation(ss *Stylesheet, kind InvocationKind) Invocation {
	return Invocation{cfg: &invocationCfg{ss: ss, kind: kind}}
}

func (inv Invocation) clone() Invocation {
	cp := *inv.cfg
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

// Selection sets the initial match selection for apply-templates.
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

// Receiver sets the receiver object that may implement any subset of
// MessageReceiver, ResultDocumentReceiver, RawResultReceiver,
// PrimaryItemsReceiver, AnnotationReceiver, etc.
func (inv Invocation) Receiver(r any) Invocation {
	inv = inv.clone()
	inv.cfg.receiver = r
	return inv
}

// CollectionResolver sets a custom resolver for fn:collection.
func (inv Invocation) CollectionResolver(r xpath3.CollectionResolver) Invocation {
	inv = inv.clone()
	inv.cfg.collectionResolver = r
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

// Do executes the transformation and returns the principal result document.
func (inv Invocation) Do(ctx context.Context) (*helium.Document, error) {
	if err := inv.validate(); err != nil {
		return nil, err
	}
	tcfg := inv.toTransformConfig()
	return executeTransform(ctx, inv.cfg.source, inv.cfg.ss, tcfg)
}

// Serialize executes the transformation and returns the serialized result.
// Secondary result documents are delivered through the receiver only.
func (inv Invocation) Serialize(ctx context.Context) (string, error) {
	var buf strings.Builder
	if err := inv.WriteTo(ctx, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// WriteTo executes the transformation and writes the serialized result to w.
// Secondary result documents are delivered through the receiver only.
func (inv Invocation) WriteTo(ctx context.Context, w io.Writer) error {
	if err := inv.validate(); err != nil {
		return err
	}
	tcfg := inv.toTransformConfig()
	resultDoc, err := executeTransform(ctx, inv.cfg.source, inv.cfg.ss, tcfg)
	if err != nil {
		return err
	}
	outDef := inv.cfg.ss.outputs[""]
	return SerializeResult(w, resultDoc, outDef)
}

// validate checks that the invocation config is valid for the entry kind.
func (inv Invocation) validate() error {
	c := inv.cfg
	switch c.kind {
	case InvocationTransform:
		// nil source is allowed: the stylesheet may use xsl:source-document,
		// global-context-item use="absent", or an initial template.
		// executeTransform will raise XTDE0040 if source is truly needed.
	case InvocationApplyTemplates:
		// nil source is allowed when a match selection is provided, or when
		// the stylesheet does not require a source document.
		// executeTransform will raise XTDE0040 if needed.
	case InvocationCallTemplate:
		if c.initialTemplate == "" {
			return fmt.Errorf("xslt3: CallTemplate requires a template name")
		}
		if c.mode != "" {
			return fmt.Errorf("xslt3: Mode is not valid for CallTemplate")
		}
		if c.matchSelection != nil {
			return fmt.Errorf("xslt3: Selection is not valid for CallTemplate")
		}
	case InvocationCallFunction:
		if c.initialFunction == "" {
			return fmt.Errorf("xslt3: CallFunction requires a function name")
		}
		if c.mode != "" {
			return fmt.Errorf("xslt3: Mode is not valid for CallFunction")
		}
		if c.matchSelection != nil {
			return fmt.Errorf("xslt3: Selection is not valid for CallFunction")
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
		baseOutputURI:      c.baseOutputURI,
		sourceSchemas:      c.sourceSchemas,
		onMultipleMatch:    c.onMultipleMatch.String(),
	}

	// Entry mode
	switch c.kind {
	case InvocationCallTemplate:
		tcfg.initialTemplate = c.initialTemplate
	case InvocationCallFunction:
		tcfg.initialFunction = c.initialFunction
		tcfg.initialFunctionParams = c.initialArgs
	case InvocationApplyTemplates:
		if c.mode != "" {
			tcfg.initialMode = c.mode
		}
		if c.matchSelection != nil {
			tcfg.initialMatchSelection = c.matchSelection
		}
	case InvocationTransform:
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
		case InvocationCallTemplate:
			tcfg.initialTemplateTunnel = tunnel
		default:
			tcfg.initialModeTunnel = tunnel
		}
	}

	// Receiver → wire directly into transformConfig.
	if c.receiver != nil {
		rs := extractReceivers(c.receiver)
		tcfg.msgReceiver = rs.message
		tcfg.resultDocReceiver = rs.resultDocument
		tcfg.resultDocOutputReceiver = rs.resultDocumentOutput
		tcfg.rawResultReceiver = rs.rawResult
		tcfg.primaryItemsReceiver = rs.primaryItems
		tcfg.annotationReceiver = rs.annotations
	}

	return tcfg
}
