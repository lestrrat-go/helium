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
	C14N10          Mode = iota // Canonical XML 1.0 (libxml2: XML_C14N_1_0)
	ExclusiveC14N10             // Exclusive Canonical XML 1.0 (libxml2: XML_C14N_EXCLUSIVE_1_0)
	C14N11                      // Canonical XML 1.1 (libxml2: XML_C14N_1_1)
)

// canonicalizerCfg holds the configuration for a Canonicalizer.
type canonicalizerCfg struct {
	mode              Mode
	withComments      bool
	nodeSet           []helium.Node
	inclusivePrefixes []string
	baseURI           string
}

// Canonicalizer configures XML canonicalization. It is a value-style
// wrapper: fluent methods return updated copies and the original is
// never mutated. The terminal methods Canonicalize and CanonicalizeTo
// execute the canonicalization.
type Canonicalizer struct {
	cfg *canonicalizerCfg
}

// NewCanonicalizer creates a new Canonicalizer for the given mode.
func NewCanonicalizer(mode Mode) Canonicalizer {
	return Canonicalizer{cfg: &canonicalizerCfg{mode: mode}}
}

func (c Canonicalizer) clone() Canonicalizer {
	cp := *c.cfg
	return Canonicalizer{cfg: &cp}
}

// Comments enables comment output in the canonical form.
func (c Canonicalizer) Comments() Canonicalizer {
	c = c.clone()
	c.cfg.withComments = true
	return c
}

// NodeSet restricts canonicalization to the given set of nodes.
func (c Canonicalizer) NodeSet(nodes []helium.Node) Canonicalizer {
	c = c.clone()
	c.cfg.nodeSet = append([]helium.Node(nil), nodes...)
	return c
}

// BaseURI specifies the document's base URI. This is needed for
// C14N 1.1 xml:base fixup when using node-set filtering. If not provided,
// xml:base fixup uses an empty base.
func (c Canonicalizer) BaseURI(uri string) Canonicalizer {
	c = c.clone()
	c.cfg.baseURI = uri
	return c
}

// InclusiveNamespaces specifies prefixes that should be treated as
// inclusive when using ExclusiveC14N10 mode. Use "" (empty string) or
// "#default" for the default namespace.
func (c Canonicalizer) InclusiveNamespaces(prefixes []string) Canonicalizer {
	c = c.clone()
	c.cfg.inclusivePrefixes = append([]string(nil), prefixes...)
	return c
}

// Canonicalize writes the canonical form of doc to out.
// (libxml2: xmlC14NDocSaveTo)
func (c Canonicalizer) Canonicalize(doc *helium.Document, out io.Writer) error {
	can := &canonicalizer{
		doc:  doc,
		mode: c.cfg.mode,
		out:  out,
	}
	can.withComments = c.cfg.withComments
	can.baseURI = c.cfg.baseURI
	if len(c.cfg.nodeSet) > 0 {
		can.nodeSet = make(map[helium.Node]struct{}, len(c.cfg.nodeSet))
		for _, n := range c.cfg.nodeSet {
			can.nodeSet[n] = struct{}{}
		}
	}
	if len(c.cfg.inclusivePrefixes) > 0 {
		can.inclusivePrefixes = make(map[string]struct{}, len(c.cfg.inclusivePrefixes))
		for _, p := range c.cfg.inclusivePrefixes {
			if p == "#default" {
				p = ""
			}
			can.inclusivePrefixes[p] = struct{}{}
		}
	}
	return can.process()
}

// CanonicalizeTo returns the canonical form of doc as a byte slice.
// (libxml2: xmlC14NDocSaveTo)
func (c Canonicalizer) CanonicalizeTo(doc *helium.Document) ([]byte, error) {
	var buf bytes.Buffer
	if err := c.Canonicalize(doc, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
