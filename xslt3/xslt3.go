package xslt3

import (
	"context"
	"io"

	"github.com/lestrrat-go/helium"
)

// secureXMLParser returns a parser hardened against XML External Entity (XXE)
// attacks: external DTD/entity loading is blocked and network access is
// forbidden. This is the default for every xslt3-internal parse of
// externally-sourced XML (resolver/HTTP-fetched stylesheet modules and runtime
// documents).
func secureXMLParser(baseURI string) helium.Parser {
	p := helium.NewParser().BlockXXE(true).LoadExternalDTD(false).AllowNetwork(false)
	if baseURI != "" {
		p = p.BaseURI(baseURI)
	}
	return p
}

// externalEntityLoader loads the bytes for an external DTD / general entity
// referenced by an opted-in permissive parse. It is backed by the XSLT engine's
// configured URIResolver / HTTPClient (and its default-deny + resource-limit
// policy), so opted-in external entities are fetched through the SAME
// resolver-mediated, bounded channel as the parent document — never via the
// parser's default os.Open / network.
type externalEntityLoader func(ctx context.Context, uri string) ([]byte, error)

// parseExternalXML parses externally-sourced XML loaded through a resolver or
// HTTP client. By default XXE is blocked. When allowExternalEntities is true
// the legacy permissive behavior is restored (external DTD / general entity
// loading via LoadExternalDTD + SubstituteEntities), but the loads are routed
// through entityLoader (the configured URIResolver / HTTPClient, subject to the
// configured resource limits) rather than the parser's raw filesystem/network.
// When entityLoader is nil the permissive parse falls back to the parser's
// default resource access. extraOpts lets callers layer additional parser
// options (e.g. DefaultDTDAttributes); they are applied in BOTH branches. In
// the secure branch the XXE guards (BlockXXE / AllowNetwork / LoadExternalDTD)
// are re-asserted AFTER extraOpts so an extraOpts hook cannot accidentally
// re-enable external loading — only internal-subset behaviors (e.g.
// DefaultDTDAttributes) survive.
func parseExternalXML(ctx context.Context, data []byte, baseURI string, allowExternalEntities bool, entityLoader externalEntityLoader, extraOpts func(helium.Parser) helium.Parser) (*helium.Document, error) {
	var p helium.Parser
	if allowExternalEntities {
		// NewParser now blocks external loading by default; this branch is the
		// explicit opt-in, so lift the block. Loads are still confined to the
		// configured loader (or the permissive fallback) selected below.
		p = helium.NewParser().BlockXXE(false).LoadExternalDTD(true).SubstituteEntities(true)
		if entityLoader != nil {
			// Route opted-in external DTD/entity resolution through the
			// configured resolver-backed loader instead of the parser default.
			p = p.FS(schemaResolverFS{ctx: ctx, load: entityLoader})
		} else {
			// No loader configured: restore the historical permissive resource
			// access (NewParser now defaults to a deny-all FS).
			p = p.FS(helium.PermissiveFS())
		}
		if extraOpts != nil {
			p = extraOpts(p)
		}
	} else {
		// SubstituteEntities(true) keeps INTERNAL general entities (declared in
		// the internal subset) expanding to their replacement text; BlockXXE /
		// LoadExternalDTD(false) / AllowNetwork(false) still block external DTD,
		// external entity, and network access. Without SubstituteEntities a
		// resolver-loaded doc() with an internal entity ref would surface an
		// EntityRefNode instead of substituted text.
		p = helium.NewParser().SubstituteEntities(true).BlockXXE(true).LoadExternalDTD(false).AllowNetwork(false)
		if extraOpts != nil {
			p = extraOpts(p)
		}
		// Re-assert the XXE guards so extraOpts cannot weaken the secure posture;
		// only internal-subset behaviors layered by extraOpts (e.g.
		// DefaultDTDAttributes, internal-entity substitution) are kept.
		p = p.SubstituteEntities(true).BlockXXE(true).AllowNetwork(false).LoadExternalDTD(false)
	}
	if baseURI != "" {
		p = p.BaseURI(baseURI)
	}
	return p.Parse(ctx, data)
}

// parseStylesheetDocument parses an externally-sourced stylesheet module
// (xsl:import / xsl:include / xsl:use-package / fn:transform stylesheets).
// XXE is blocked unless allowExternalEntities opts into the legacy behavior.
func parseStylesheetDocument(ctx context.Context, data []byte, baseURI string, allowExternalEntities bool, entityLoader externalEntityLoader) (*helium.Document, error) {
	return parseExternalXML(ctx, data, baseURI, allowExternalEntities, entityLoader, nil)
}

// CompileStylesheet compiles a parsed XSLT stylesheet document.
// This is a convenience wrapper over NewCompiler().Compile(ctx, doc).
func CompileStylesheet(ctx context.Context, doc *helium.Document) (*Stylesheet, error) {
	return NewCompiler().Compile(ctx, doc)
}

// Transform applies the compiled stylesheet to the source document.
// This is a convenience wrapper over ss.Transform(source).Do(ctx).
func Transform(ctx context.Context, source *helium.Document, ss *Stylesheet) (*helium.Document, error) {
	if ss == nil {
		return nil, errNilStylesheet
	}
	return ss.Transform(source).Do(ctx)
}

// TransformString applies the compiled stylesheet and returns the serialized result.
// This is a convenience wrapper over ss.Transform(source).Serialize(ctx).
func TransformString(ctx context.Context, source *helium.Document, ss *Stylesheet) (string, error) {
	if ss == nil {
		return "", errNilStylesheet
	}
	return ss.Transform(source).Serialize(ctx)
}

// TransformToWriter applies the compiled stylesheet and writes the result to w.
// This is a convenience wrapper over ss.Transform(source).WriteTo(ctx, w).
func TransformToWriter(ctx context.Context, source *helium.Document, ss *Stylesheet, w io.Writer) error {
	if ss == nil {
		return errNilStylesheet
	}
	return ss.Transform(source).WriteTo(ctx, w)
}
