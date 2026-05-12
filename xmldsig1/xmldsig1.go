package xmldsig1

import (
	"context"

	helium "github.com/lestrrat-go/helium"
)

// ReferenceConfig describes a single Reference element in a signature.
type ReferenceConfig struct {
	URI             string
	DigestAlgorithm string
	Transforms      []Transform
	ID              string
	Type            string
}

// NewEnvelopedReference returns a ReferenceConfig pre-configured for the
// common SAML enveloped signature pattern: empty URI, enveloped-signature
// transform + Exclusive C14N, SHA-256 digest.
func NewEnvelopedReference() ReferenceConfig {
	return ReferenceConfig{
		URI:             "",
		DigestAlgorithm: DigestSHA256,
		Transforms:      []Transform{Enveloped(), ExcC14NTransform()},
	}
}

// signerConfig holds the configuration for a Signer.
type signerConfig struct {
	signatureAlgorithm string
	c14nMethod         string
	references         []ReferenceConfig
	keyInfoBuilder     KeyInfoBuilder
	signatureID        string
}

// Signer creates XML Digital Signatures. It uses clone-on-write semantics:
// each builder method returns a new Signer and the original is never mutated.
type Signer struct {
	cfg *signerConfig
}

// NewSigner creates a new Signer with default settings.
// Defaults: Exclusive C14N for SignedInfo canonicalization.
func NewSigner() Signer {
	return Signer{cfg: &signerConfig{
		c14nMethod: ExcC14N10,
	}}
}

func (s Signer) clone() Signer {
	if s.cfg == nil {
		return Signer{cfg: &signerConfig{c14nMethod: ExcC14N10}}
	}
	cp := *s.cfg
	cp.references = append([]ReferenceConfig(nil), s.cfg.references...)
	return Signer{cfg: &cp}
}

// SignatureAlgorithm sets the signature algorithm URI.
func (s Signer) SignatureAlgorithm(alg string) Signer {
	s = s.clone()
	s.cfg.signatureAlgorithm = alg
	return s
}

// CanonicalizationMethod sets the canonicalization algorithm for SignedInfo.
func (s Signer) CanonicalizationMethod(method string) Signer {
	s = s.clone()
	s.cfg.c14nMethod = method
	return s
}

// Reference adds a Reference to be signed.
func (s Signer) Reference(ref ReferenceConfig) Signer {
	s = s.clone()
	s.cfg.references = append(s.cfg.references, ref)
	return s
}

// KeyInfo configures KeyInfo element construction.
func (s Signer) KeyInfo(builder KeyInfoBuilder) Signer {
	s = s.clone()
	s.cfg.keyInfoBuilder = builder
	return s
}

// SignatureID sets the Id attribute on the Signature element.
func (s Signer) SignatureID(id string) Signer {
	s = s.clone()
	s.cfg.signatureID = id
	return s
}

// SignEnveloped creates an enveloped signature inside the given parent
// element of the document. The key is a crypto.Signer (rsa.PrivateKey,
// ecdsa.PrivateKey, ed25519.PrivateKey) or []byte for HMAC.
func (s Signer) SignEnveloped(ctx context.Context, doc *helium.Document, parent *helium.Element, key any) error {
	return signEnveloped(ctx, s.cfg, doc, parent, key)
}

// SignEnveloping creates an enveloping signature wrapping the given content
// nodes. Returns the Signature element.
func (s Signer) SignEnveloping(ctx context.Context, doc *helium.Document, content []helium.Node, key any) (*helium.Element, error) {
	return signEnveloping(ctx, s.cfg, doc, content, key)
}

// SignDetached creates a detached Signature element referencing URIs
// specified in the configured References. Returns the Signature element.
func (s Signer) SignDetached(ctx context.Context, doc *helium.Document, key any) (*helium.Element, error) {
	return signDetached(ctx, s.cfg, doc, key)
}

// verifierConfig holds the configuration for a Verifier.
type verifierConfig struct {
	keySource KeySource
}

// Verifier verifies XML Digital Signatures. It uses clone-on-write semantics.
type Verifier struct {
	cfg *verifierConfig
}

// NewVerifier creates a new Verifier with the given key source.
func NewVerifier(ks KeySource) Verifier {
	return Verifier{cfg: &verifierConfig{keySource: ks}}
}

// Verify verifies the Signature element in the document. The document must
// contain exactly one ds:Signature element; if it contains more than one
// the function returns ErrAmbiguousSignature and the caller must use
// VerifyElement to disambiguate.
//
// On success the returned VerifyResult exposes the set of elements actually
// covered by the signature so callers can confirm — by pointer identity —
// that the element they intend to consume was signed. This is the primary
// defense against XML Signature Wrapping (XSW) attacks at the application
// layer.
func (v Verifier) Verify(ctx context.Context, doc *helium.Document) (*VerifyResult, error) {
	sigs := findSignatureElements(doc.DocumentElement())
	switch len(sigs) {
	case 0:
		return nil, ErrSignatureNotFound
	case 1:
		return verifySignature(ctx, v.cfg, doc, sigs[0])
	default:
		return nil, ErrAmbiguousSignature
	}
}

// VerifyElement verifies a specific Signature element. Use this when the
// document contains more than one Signature, or when the caller wants
// explicit control over which Signature is targeted.
func (v Verifier) VerifyElement(ctx context.Context, doc *helium.Document, sig *helium.Element) (*VerifyResult, error) {
	return verifySignature(ctx, v.cfg, doc, sig)
}

// findSignatureElements walks the tree and returns every ds:Signature
// element. The walk is exhaustive so that multiple-Signature documents are
// detected rather than silently resolved to the first match.
func findSignatureElements(n helium.Node) []*helium.Element {
	var out []*helium.Element
	var walk func(helium.Node)
	walk = func(node helium.Node) {
		elem, ok := helium.AsNode[*helium.Element](node)
		if !ok {
			return
		}
		if localName(elem) == "Signature" && isDSigNS(elem) {
			out = append(out, elem)
			// Do not descend into a Signature — a Signature inside
			// another Signature (e.g. inside KeyInfo) is not itself a
			// signature to be verified at the document level.
			return
		}
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			walk(child)
		}
	}
	walk(n)
	return out
}

func isDSigNS(e *helium.Element) bool {
	ns := elementNamespaceURI(e)
	return ns == NamespaceDSig || ns == NamespaceDSig11
}

func elementNamespaceURI(e *helium.Element) string {
	name := e.Name()
	for i := range len(name) {
		if name[i] == ':' {
			prefix := name[:i]
			for _, ns := range e.Namespaces() {
				if ns.Prefix() == prefix {
					return ns.URI()
				}
			}
			// Walk ancestors for the namespace declaration.
			for p := e.Parent(); p != nil; p = p.Parent() {
				if pe, ok := helium.AsNode[*helium.Element](p); ok {
					for _, ns := range pe.Namespaces() {
						if ns.Prefix() == prefix {
							return ns.URI()
						}
					}
				}
			}
			return ""
		}
	}
	// No prefix — look for default namespace.
	for _, ns := range e.Namespaces() {
		if ns.Prefix() == "" {
			return ns.URI()
		}
	}
	for p := e.Parent(); p != nil; p = p.Parent() {
		if pe, ok := helium.AsNode[*helium.Element](p); ok {
			for _, ns := range pe.Namespaces() {
				if ns.Prefix() == "" {
					return ns.URI()
				}
			}
		}
	}
	return ""
}
