package xslt3

import (
	"context"
	"maps"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// Compiler configures XSLT 3.0 stylesheet compilation.
// It is a value-style wrapper: fluent methods return updated copies
// and the original is never mutated. The terminal method Compile
// creates an internal compileCtx immediately; downstream compilation
// uses that context, never the Compiler itself.
type Compiler struct {
	cfg *xsltCompilerCfg
}

type xsltCompilerCfg struct {
	baseURI         string
	uriResolver     URIResolver
	packageResolver PackageResolver
	staticParams    *Parameters
}

// NewCompiler creates a new Compiler with default settings.
func NewCompiler() Compiler {
	return Compiler{cfg: &xsltCompilerCfg{}}
}

func (c Compiler) clone() Compiler {
	cp := *c.cfg
	return Compiler{cfg: &cp}
}

// BaseURI sets the base URI for resolving relative URIs in xsl:import
// and xsl:include.
func (c Compiler) BaseURI(uri string) Compiler {
	c = c.clone()
	c.cfg.baseURI = uri
	return c
}

// URIResolver sets a custom URI resolver for loading external
// stylesheets during compilation.
func (c Compiler) URIResolver(r URIResolver) Compiler {
	c = c.clone()
	c.cfg.uriResolver = r
	return c
}

// PackageResolver sets a package resolver for xsl:use-package references.
func (c Compiler) PackageResolver(r PackageResolver) Compiler {
	c = c.clone()
	c.cfg.packageResolver = r
	return c
}

// StaticParameters sets the static parameters for compilation.
// The Parameters collection is cloned.
func (c Compiler) StaticParameters(p *Parameters) Compiler {
	c = c.clone()
	c.cfg.staticParams = p.Clone()
	return c
}

// SetStaticParameter sets a single static parameter value.
// Clone-on-write: the existing parameters are cloned before mutation
// if shared with another Compiler value.
func (c Compiler) SetStaticParameter(name string, value xpath3.Sequence) Compiler {
	c = c.clone()
	if c.cfg.staticParams == nil {
		c.cfg.staticParams = NewParameters()
	} else {
		c.cfg.staticParams = c.cfg.staticParams.Clone()
	}
	c.cfg.staticParams.Set(name, value)
	return c
}

// ClearStaticParameters removes all static parameter bindings.
func (c Compiler) ClearStaticParameters() Compiler {
	c = c.clone()
	c.cfg.staticParams = nil
	return c
}

// Compile compiles a parsed XSLT stylesheet document into a reusable
// Stylesheet. ctx is used for cancellation/deadlines.
func (c Compiler) Compile(ctx context.Context, doc *helium.Document) (*Stylesheet, error) {
	return compile(doc, c.toCompileConfig())
}

// MustCompile is like Compile but panics on error.
// Note: context cancellation or timeout will cause a panic.
func (c Compiler) MustCompile(ctx context.Context, doc *helium.Document) *Stylesheet {
	ss, err := c.Compile(ctx, doc)
	if err != nil {
		panic("xslt3: Compile: " + err.Error())
	}
	return ss
}

// toCompileConfig converts the Compiler config to the internal compileConfig
// used by the existing compile function.
func (c Compiler) toCompileConfig() *compileConfig {
	cfg := &compileConfig{
		baseURI:         c.cfg.baseURI,
		resolver:        c.cfg.uriResolver,
		packageResolver: c.cfg.packageResolver,
	}
	if c.cfg.staticParams != nil {
		cfg.staticParams = maps.Clone(c.cfg.staticParams.toMap())
	}
	return cfg
}
