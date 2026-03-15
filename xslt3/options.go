package xslt3

import (
	"context"
	"io"
	"maps"

	"github.com/lestrrat-go/helium"
)

// URIResolver resolves URIs to readable content. Used for xsl:import
// and xsl:include during compilation.
type URIResolver interface {
	Resolve(uri string) (io.ReadCloser, error)
}

// --- Compile configuration (context-based) ---

// compileConfigKey is used to store compile configuration in context.Context.
type compileConfigKey struct{}

type compileConfig struct {
	baseURI  string
	resolver URIResolver
}

func getCompileConfig(ctx context.Context) *compileConfig {
	if ctx == nil {
		return nil
	}
	cfg, _ := ctx.Value(compileConfigKey{}).(*compileConfig)
	return cfg
}

func deriveCompileConfig(ctx context.Context) *compileConfig {
	if cfg := getCompileConfig(ctx); cfg != nil {
		cp := *cfg
		return &cp
	}
	return &compileConfig{}
}

func withCompileConfig(ctx context.Context, cfg *compileConfig) context.Context {
	return context.WithValue(ctx, compileConfigKey{}, cfg)
}

func updateCompileConfig(ctx context.Context, fn func(*compileConfig)) context.Context {
	cfg := deriveCompileConfig(ctx)
	fn(cfg)
	return withCompileConfig(ctx, cfg)
}

// WithCompileBaseURI sets the base URI for resolving relative URIs in
// xsl:import and xsl:include during compilation.
func WithCompileBaseURI(ctx context.Context, uri string) context.Context {
	return updateCompileConfig(ctx, func(c *compileConfig) {
		c.baseURI = uri
	})
}

// WithCompileURIResolver sets a custom URI resolver for loading external
// stylesheets and documents during compilation.
func WithCompileURIResolver(ctx context.Context, r URIResolver) context.Context {
	return updateCompileConfig(ctx, func(c *compileConfig) {
		c.resolver = r
	})
}

// --- Transform configuration (context-based) ---

// transformConfigKey is used to store transform configuration in context.Context.
type transformConfigKey struct{}

type transformConfig struct {
	params          map[string]string
	msgHandler      func(msg string, terminate bool)
	initialTemplate string
	resultDocHandler func(href string, doc *helium.Document)
}

func getTransformConfig(ctx context.Context) *transformConfig {
	if ctx == nil {
		return nil
	}
	cfg, _ := ctx.Value(transformConfigKey{}).(*transformConfig)
	return cfg
}

func deriveTransformConfig(ctx context.Context) *transformConfig {
	if cfg := getTransformConfig(ctx); cfg != nil {
		cp := *cfg
		return &cp
	}
	return &transformConfig{}
}

func withTransformConfig(ctx context.Context, cfg *transformConfig) context.Context {
	return context.WithValue(ctx, transformConfigKey{}, cfg)
}

func updateTransformConfig(ctx context.Context, fn func(*transformConfig)) context.Context {
	cfg := deriveTransformConfig(ctx)
	fn(cfg)
	return withTransformConfig(ctx, cfg)
}

// WithParameter sets a global stylesheet parameter value.
func WithParameter(ctx context.Context, name, value string) context.Context {
	return updateTransformConfig(ctx, func(c *transformConfig) {
		c.params = maps.Clone(c.params)
		if c.params == nil {
			c.params = make(map[string]string)
		}
		c.params[name] = value
	})
}

// WithInitialTemplate sets the initial named template to call instead of
// applying templates to the source document.
func WithInitialTemplate(ctx context.Context, name string) context.Context {
	return updateTransformConfig(ctx, func(c *transformConfig) {
		c.initialTemplate = name
	})
}

// WithMessageHandler sets a handler for xsl:message output.
// If terminate is true, the transformation should stop after the message.
func WithMessageHandler(ctx context.Context, fn func(msg string, terminate bool)) context.Context {
	return updateTransformConfig(ctx, func(c *transformConfig) {
		c.msgHandler = fn
	})
}

// WithResultDocumentHandler sets a handler for secondary result documents
// produced by xsl:result-document instructions. The handler is called with
// the evaluated href and the result document for each secondary output.
func WithResultDocumentHandler(ctx context.Context, fn func(href string, doc *helium.Document)) context.Context {
	return updateTransformConfig(ctx, func(c *transformConfig) {
		c.resultDocHandler = fn
	})
}
