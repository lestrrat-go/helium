package xslt3

import (
	"context"
	"io"
	"math"

	"github.com/lestrrat-go/helium"
)

// alignNodeContentCap returns p with its parser-level per-node content cap
// aligned to the resource read cap (resourceLimit) used to fetch the bytes it
// will parse. The bytes handed to these xslt3-internal parses are already read
// fully into memory bounded by resourceLimit, so the parser's separate
// per-node content cap (default 10 MiB) must not independently reject a single
// large node that the raised resource cap already admitted. The 0/negative/
// positive convention (default / unbounded / explicit) matches between the
// resource cap and [helium.Parser.MaxNodeContentSize], so a resourceLimit of 0
// keeps the parser default unchanged.
func alignNodeContentCap(p helium.Parser, resourceLimit int64) helium.Parser {
	return p.MaxNodeContentSize(clampInt64ToInt(resourceLimit))
}

// clampInt64ToInt narrows an int64 limit to int without wrapping on 32-bit
// platforms; the sign is preserved so the 0/negative/positive convention
// survives the conversion.
func clampInt64ToInt(n int64) int {
	if n > int64(math.MaxInt) {
		return math.MaxInt
	}
	if n < int64(math.MinInt) {
		return math.MinInt
	}
	return int(n)
}

// secureXMLParser returns the base parser for an xslt3-internal parse of
// externally-sourced XML (resolver/HTTP-fetched stylesheet modules, schema
// documents, and runtime documents).
//
// When injected is non-nil the caller-supplied parser (Compiler.Parser /
// Invocation.Parser) is used as the base, taking its parse policy as-is. When
// injected is nil the base is hardened against XML External Entity (XXE)
// attacks: external DTD/entity loading is blocked and network access is
// forbidden. The BaseURI is applied in either case.
func secureXMLParser(injected *helium.Parser, baseURI string, resourceLimit int64) helium.Parser {
	var p helium.Parser
	if injected != nil {
		p = *injected
	} else {
		p = helium.NewParser().BlockXXE(true).LoadExternalDTD(false).AllowNetwork(false)
	}
	if baseURI != "" {
		p = p.BaseURI(baseURI)
	}
	return alignNodeContentCap(p, resourceLimit)
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
//
// When injected is non-nil the caller-supplied parser (Compiler.Parser /
// Invocation.Parser) is the base in BOTH branches; the functional opts below
// (and the permissive-branch external-loading opts) are still forced, but the
// secure branch does NOT re-assert the XXE guards on top — the injected policy
// wins. When injected is nil the historical hardened default is the base and the
// guards are re-asserted exactly as before.
func parseExternalXML(ctx context.Context, injected *helium.Parser, data []byte, baseURI string, allowExternalEntities bool, entityLoader externalEntityLoader, extraOpts func(helium.Parser) helium.Parser, resourceLimit int64) (*helium.Document, error) {
	base := func() helium.Parser {
		if injected != nil {
			return *injected
		}
		return helium.NewParser()
	}
	var p helium.Parser
	if allowExternalEntities {
		// NewParser now blocks external loading by default; this branch is the
		// explicit opt-in, so lift the block. Loads are still confined to the
		// configured loader (or the permissive fallback) selected below.
		p = base().BlockXXE(false).LoadExternalDTD(true).SubstituteEntities(true)
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
		p = base().SubstituteEntities(true)
		if injected == nil {
			p = p.BlockXXE(true).LoadExternalDTD(false).AllowNetwork(false)
		}
		if extraOpts != nil {
			p = extraOpts(p)
		}
		// Force internal-entity substitution regardless of base/extraOpts.
		p = p.SubstituteEntities(true)
		// Re-assert the XXE guards so extraOpts cannot weaken the secure posture;
		// only internal-subset behaviors layered by extraOpts (e.g.
		// DefaultDTDAttributes, internal-entity substitution) are kept. When a
		// parser is injected its policy wins, so the guards are NOT re-asserted.
		if injected == nil {
			p = p.BlockXXE(true).AllowNetwork(false).LoadExternalDTD(false)
		}
	}
	if baseURI != "" {
		p = p.BaseURI(baseURI)
	}
	// Align the per-node content cap with the resource read cap. The bytes are
	// already fully in memory bounded by resourceLimit, so the parser's default
	// node-content cap must not independently reject content the resource cap
	// already admitted. Applied last so it survives the XXE-guard re-assertions
	// above (node-content size is not an XXE guard).
	p = alignNodeContentCap(p, resourceLimit)
	return p.Parse(ctx, data)
}

// parseStylesheetDocument parses an externally-sourced stylesheet module
// (xsl:import / xsl:include / xsl:use-package / fn:transform stylesheets).
// XXE is blocked unless allowExternalEntities opts into the legacy behavior.
func parseStylesheetDocument(ctx context.Context, injected *helium.Parser, data []byte, baseURI string, allowExternalEntities bool, entityLoader externalEntityLoader, resourceLimit int64) (*helium.Document, error) {
	return parseExternalXML(ctx, injected, data, baseURI, allowExternalEntities, entityLoader, nil, resourceLimit)
}

// CompileStylesheet compiles a parsed XSLT stylesheet document.
// This is a convenience wrapper over NewCompiler().Compile(ctx, doc).
//
// The returned [*Stylesheet] is immutable and safe for concurrent use by
// multiple goroutines, each with its own source document; it may be reused
// across many transformations. See the package documentation's Concurrency
// section (in particular the source-document caveat for schema-aware
// transforms).
func CompileStylesheet(ctx context.Context, doc *helium.Document) (*Stylesheet, error) {
	return NewCompiler().Compile(ctx, doc)
}

// Transform applies the compiled stylesheet to the source document.
// This is a convenience wrapper over ss.Transform(source).Do(ctx).
//
// ss is not mutated and may be transformed concurrently from multiple
// goroutines, each with its own source document. A schema-aware transform
// mutates source in place, so such a source must not be shared across
// concurrent transforms (see the package documentation's Concurrency section).
func Transform(ctx context.Context, source *helium.Document, ss *Stylesheet) (*helium.Document, error) {
	if ss == nil {
		return nil, errNilStylesheet
	}
	return ss.Transform(source).Do(ctx)
}

// TransformString applies the compiled stylesheet and returns the serialized result.
// This is a convenience wrapper over ss.Transform(source).Serialize(ctx).
//
// ss is not mutated and may be transformed concurrently from multiple
// goroutines, each with its own source document. A schema-aware transform
// mutates source in place, so such a source must not be shared across
// concurrent transforms (see the package documentation's Concurrency section).
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
