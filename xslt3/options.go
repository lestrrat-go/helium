package xslt3

import (
	"context"
	"io"
	"maps"
)

// CompileOption configures stylesheet compilation.
type CompileOption func(*compileConfig)

type compileConfig struct {
	baseURI  string
	resolver URIResolver
}

// WithBaseURI sets the base URI for resolving relative URIs in
// xsl:import and xsl:include.
func WithBaseURI(uri string) CompileOption {
	return func(c *compileConfig) {
		c.baseURI = uri
	}
}

// URIResolver resolves URIs to readable content. Used for xsl:import
// and xsl:include during compilation.
type URIResolver interface {
	Resolve(uri string) (io.ReadCloser, error)
}

// WithURIResolver sets a custom URI resolver for loading external
// stylesheets and documents.
func WithURIResolver(r URIResolver) CompileOption {
	return func(c *compileConfig) {
		c.resolver = r
	}
}

// transformConfigKey is used to store transform configuration in context.Context.
type transformConfigKey struct{}

type transformConfig struct {
	params          map[string]string
	msgHandler      func(msg string, terminate bool)
	initialTemplate string
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
		// Shallow-copy the config and let mutators clone maps only when they
		// actually modify them.
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
