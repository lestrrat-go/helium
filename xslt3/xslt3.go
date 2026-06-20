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

// parseExternalXML parses externally-sourced XML loaded through a resolver or
// HTTP client. By default XXE is blocked. When allowExternalEntities is true
// the legacy permissive behavior is restored (resolver-mediated external entity
// loading via LoadExternalDTD + SubstituteEntities), subject to the configured
// resource limits. extraOpts lets callers layer additional parser options (e.g.
// DefaultDTDAttributes) onto the permissive path.
func parseExternalXML(ctx context.Context, data []byte, baseURI string, allowExternalEntities bool, extraOpts func(helium.Parser) helium.Parser) (*helium.Document, error) {
	var p helium.Parser
	if allowExternalEntities {
		p = helium.NewParser().LoadExternalDTD(true).SubstituteEntities(true)
		if extraOpts != nil {
			p = extraOpts(p)
		}
		if baseURI != "" {
			p = p.BaseURI(baseURI)
		}
	} else {
		p = secureXMLParser(baseURI)
	}
	return p.Parse(ctx, data)
}

// parseStylesheetDocument parses an externally-sourced stylesheet module
// (xsl:import / xsl:include / xsl:use-package / fn:transform stylesheets).
// XXE is blocked unless allowExternalEntities opts into the legacy behavior.
func parseStylesheetDocument(ctx context.Context, data []byte, baseURI string, allowExternalEntities bool) (*helium.Document, error) {
	return parseExternalXML(ctx, data, baseURI, allowExternalEntities, nil)
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
