package xmldsig1

import (
	"context"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
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
	allowSHA1          bool
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

// AllowSHA1 controls whether SHA-1-based signature and digest algorithms
// (rsa-sha1, hmac-sha1, sha1) may be used when signing. SHA-1 is rejected by
// default; pass true to opt in for legacy interoperability. SHA-1 is
// cryptographically weak and should not be used for new signatures.
func (s Signer) AllowSHA1(allow bool) Signer {
	s = s.clone()
	s.cfg.allowSHA1 = allow
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
	allowSHA1 bool
}

// Verifier verifies XML Digital Signatures. It uses clone-on-write semantics:
// each builder method returns a new Verifier and the original is never mutated.
type Verifier struct {
	cfg *verifierConfig
}

// NewVerifier creates a new Verifier with the given key source.
func NewVerifier(ks KeySource) Verifier {
	return Verifier{cfg: &verifierConfig{keySource: ks}}
}

func (v Verifier) clone() Verifier {
	if v.cfg == nil {
		return Verifier{cfg: &verifierConfig{}}
	}
	cp := *v.cfg
	return Verifier{cfg: &cp}
}

// AllowSHA1 controls whether SHA-1-based signature and digest algorithms
// (rsa-sha1, hmac-sha1, sha1) are accepted during verification. SHA-1 is
// rejected by default; pass true to opt in for verifying legacy signatures.
// SHA-1 is cryptographically weak and accepting it exposes callers to
// downgrade and collision risks, so only enable it when interoperating with
// systems that cannot be upgraded.
func (v Verifier) AllowSHA1(allow bool) Verifier {
	v = v.clone()
	v.cfg.allowSHA1 = allow
	return v
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
//
// Same-document reference resolution (ds:Reference URI="#id") locates the
// target element by its ID attribute. An attribute is recognized as an ID
// when it is any of:
//
//   - declared ID-typed by a DTD or schema the document was parsed with;
//   - xml:id (ID-typed by the W3C xml:id Recommendation);
//   - the "id" attribute token in the casings "Id", "ID", or "id".
//
// This name set is deliberately limited to the "id" token. Other conventions
// (for example "wsu:Id" or SAML "AssertionID") are ID-typed only by their own
// schemas, so a document relying on them must carry that typing — via its
// DTD/schema, or by marking the attribute's type as an ID before verifying —
// rather than have it inferred from the name. If more than one element matches
// the referenced ID the reference is refused (ErrAmbiguousReference).
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
//
// Same-document reference resolution recognizes the same ID attributes as
// [Verifier.Verify].
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
		if domutil.LocalName(elem) == "Signature" && isDSigCoreNS(elem) {
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

// isDSigCoreNS reports whether e is in the core XML-Signature namespace
// (http://www.w3.org/2000/09/xmldsig#). Core structural elements — Signature,
// SignedInfo, SignatureValue, CanonicalizationMethod, SignatureMethod,
// Reference, Transforms, Transform, DigestMethod, DigestValue, KeyInfo — are
// ALWAYS in this namespace. The XML-Signature 1.1 namespace
// (http://www.w3.org/2009/xmldsig11#) is only for new 1.1-specific elements
// (e.g. ECKeyValue, DEREncodedKeyValue); it is not an alternate spelling of the
// core elements, so a dsig11:Reference must not satisfy a core-element check.
func isDSigCoreNS(e *helium.Element) bool {
	return elementNamespaceURI(e) == NamespaceDSig
}

// isDSig11NS reports whether e is in the XML-Signature 1.1 namespace
// (http://www.w3.org/2009/xmldsig11#). The 1.1-specific elements (ECKeyValue,
// NamedCurve, PublicKey, DEREncodedKeyValue, ...) live ONLY in this namespace;
// they are not part of the core xmldsig# namespace. As with the core check,
// matching such elements on local name alone would let a foreign-namespace
// look-alike supply attacker-chosen key material, so the exact namespace is
// required.
func isDSig11NS(e *helium.Element) bool {
	return elementNamespaceURI(e) == NamespaceDSig11
}

// isExcC14NNS reports whether e is in the Exclusive XML Canonicalization
// namespace (http://www.w3.org/2001/10/xml-exc-c14n#). The InclusiveNamespaces
// element lives only here, not in the core XML-Signature namespace.
func isExcC14NNS(e *helium.Element) bool {
	return elementNamespaceURI(e) == ExcC14N10
}

func elementNamespaceURI(e *helium.Element) string {
	name := e.Name()
	prefix := ""
	if p, _, ok := strings.Cut(name, ":"); ok {
		prefix = p
	}
	// First-match-wins ancestor walk for the prefix (or the default namespace
	// when the name is unprefixed), with no implicit xml predeclaration.
	uri, _ := domutil.LookupNSPrefixURI(e, prefix)
	return uri
}
