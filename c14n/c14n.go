// Package c14n implements XML canonicalization (C14N) as defined by the W3C
// specifications: Canonical XML 1.0, Exclusive Canonical XML 1.0, and
// Canonical XML 1.1.
package c14n

import (
	"bytes"
	"io"

	helium "github.com/lestrrat-go/helium"
)

// Mode specifies the canonicalization algorithm.
type Mode int

// C14N10 selects Canonical XML 1.0, ExclusiveC14N10 selects Exclusive Canonical
// XML 1.0, and C14N11 selects Canonical XML 1.1.
const (
	C14N10          Mode = iota // Canonical XML 1.0
	ExclusiveC14N10             // Exclusive Canonical XML 1.0
	C14N11                      // Canonical XML 1.1
)

// Option configures the canonicalizer.
type Option func(*canonicalizer)

// WithComments enables comment output in the canonical form.
func WithComments() Option {
	return func(c *canonicalizer) { c.withComments = true }
}

// WithNodeSet restricts canonicalization to the given set of nodes.
func WithNodeSet(nodes []helium.Node) Option {
	return func(c *canonicalizer) {
		c.nodeSet = make(map[helium.Node]struct{}, len(nodes))
		for _, n := range nodes {
			c.nodeSet[n] = struct{}{}
		}
	}
}

// WithBaseURI specifies the document's base URI. This is needed for
// C14N 1.1 xml:base fixup when using node-set filtering. If not provided,
// xml:base fixup uses an empty base.
func WithBaseURI(uri string) Option {
	return func(c *canonicalizer) { c.baseURI = uri }
}

// WithInclusiveNamespaces specifies prefixes that should be treated as
// inclusive when using ExclusiveC14N10 mode. Use "" (empty string) or
// "#default" for the default namespace.
func WithInclusiveNamespaces(prefixes []string) Option {
	return func(c *canonicalizer) {
		c.inclusivePrefixes = make(map[string]struct{}, len(prefixes))
		for _, p := range prefixes {
			if p == "#default" {
				p = ""
			}
			c.inclusivePrefixes[p] = struct{}{}
		}
	}
}

// Canonicalize writes the canonical form of doc to out.
func Canonicalize(out io.Writer, doc *helium.Document, mode Mode, opts ...Option) error {
	c := &canonicalizer{
		doc:  doc,
		mode: mode,
		out:  out,
	}
	for _, o := range opts {
		o(c)
	}
	return c.process()
}

// CanonicalizeTo returns the canonical form of doc as a byte slice.
func CanonicalizeTo(doc *helium.Document, mode Mode, opts ...Option) ([]byte, error) {
	var buf bytes.Buffer
	if err := Canonicalize(&buf, doc, mode, opts...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
