package xslt3

import "io"

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

// URIResolver resolves URIs to readable content. Used for xsl:import,
// xsl:include, and the document() function.
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

// TransformOption configures a transformation.
type TransformOption func(*transformConfig)

type transformConfig struct {
	params     map[string]string
	msgHandler func(msg string, terminate bool)
}

// WithParameter sets a global stylesheet parameter value.
func WithParameter(name, value string) TransformOption {
	return func(c *transformConfig) {
		if c.params == nil {
			c.params = make(map[string]string)
		}
		c.params[name] = value
	}
}

// WithMessageHandler sets a handler for xsl:message output.
// If terminate is true, the transformation should stop after the message.
func WithMessageHandler(fn func(msg string, terminate bool)) TransformOption {
	return func(c *transformConfig) {
		c.msgHandler = fn
	}
}
