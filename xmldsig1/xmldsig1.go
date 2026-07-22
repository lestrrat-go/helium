package xmldsig1

import (
	"context"
	"fmt"
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

// NewEnvelopedReference returns a ReferenceConfig for a WHOLE-DOCUMENT
// enveloped signature: an empty URI, an enveloped-signature transform +
// Exclusive C14N, and a SHA-256 digest. The empty URI always resolves to the
// document element, so the reference covers the entire document regardless of
// which parent element the Signature is inserted into by [Signer.SignEnveloped].
//
// To envelope-sign a specific nested element by its id (for example a SAML
// Assertion inside a Response), use [NewEnvelopedReferenceByID] instead — an
// empty URI does NOT scope coverage to the SignEnveloped parent.
func NewEnvelopedReference() ReferenceConfig {
	return ReferenceConfig{
		URI:             "",
		DigestAlgorithm: DigestSHA256,
		Transforms:      []Transform{Enveloped(), ExcC14NTransform()},
	}
}

// NewEnvelopedReferenceByID returns a ReferenceConfig for an enveloped
// signature that covers the single element carrying the given id (URI="#id"),
// with an enveloped-signature transform + Exclusive C14N and a SHA-256 digest.
// This is the correct choice for signing a specific nested element — for
// example a SAML Assertion by its AssertionID/ID — where [NewEnvelopedReference]
// (empty URI) would cover the whole document instead.
//
// The id must be recognized as an ID attribute per the package's ID rules: a
// DTD/schema-declared ID-typed attribute, xml:id, or the "id" token in the
// casings "Id", "ID", or "id" (see [Verifier.Verify]). More than one element
// matching the id makes the reference ambiguous (ErrAmbiguousReference).
func NewEnvelopedReferenceByID(id string) ReferenceConfig {
	return ReferenceConfig{
		URI:             "#" + id,
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
	referenceResolver  ReferenceResolver
	referenceParser    *helium.Parser
}

// parser returns the parser used for external reference octets on the signing
// side: the configured ReferenceParser, or the locked-down default. Symmetric
// with verifierConfig.parser so sign and verify parse identical external content
// identically.
func (cfg *signerConfig) parser() helium.Parser {
	if cfg.referenceParser != nil {
		return *cfg.referenceParser
	}
	return helium.NewParser()
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
	// Deep-copy each Reference's Transforms slice (and each transform's internal
	// prefix slice). Copying the outer references slice alone still shares the
	// per-Reference Transforms backing array, so a caller that mutates a
	// Transforms slice it passed to Reference could otherwise alter this Signer
	// or race with an in-flight sign.
	for i := range cp.references {
		cp.references[i].Transforms = cloneReferenceTransforms(cp.references[i].Transforms)
	}
	return Signer{cfg: &cp}
}

// config returns the Signer's configuration, substituting NewSigner defaults
// when the Signer was constructed directly (a zero-value Signer{} whose cfg is
// nil). This mirrors clone's nil handling so the sign terminals never
// dereference a nil cfg: a zero-value Signer signs as a default Signer with no
// references, returning ErrNoReferences rather than panicking.
func (s Signer) config() *signerConfig {
	if s.cfg == nil {
		return &signerConfig{c14nMethod: ExcC14N10}
	}
	return s.cfg
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

// Reference adds a Reference to be signed. The Reference's Transforms slice is
// copied at ingress, so a later mutation of the slice the caller passed cannot
// alter this Signer or race with an in-flight signing operation.
func (s Signer) Reference(ref ReferenceConfig) Signer {
	s = s.clone()
	ref.Transforms = cloneReferenceTransforms(ref.Transforms)
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

// ReferenceResolver configures a [ReferenceResolver] that dereferences external
// Reference URIs during signing, so a detached signature can cover content
// outside the document. It is opt-in and symmetric with
// [Verifier.ReferenceResolver]: the default is nil, leaving an external Reference
// URI fail-closed with [ErrReferenceNotFound]. When set, an external Reference
// URI is joined against the document's base URI, passed to r, and the resolved
// octets are run through the same transform pipeline the verifier applies, so
// the signed digest is byte-identical to what verification recomputes for the
// same input.
func (s Signer) ReferenceResolver(r ReferenceResolver) Signer {
	s = s.clone()
	s.cfg.referenceResolver = r
	return s
}

// ReferenceParser configures the [helium.Parser] used to parse an external
// reference's resolved octets into XML when the Reference's transform chain needs
// a node-set. It is symmetric with [Verifier.ReferenceParser] and defaults to the
// same locked-down parser, so sign and verify parse the same external content the
// same way.
func (s Signer) ReferenceParser(p helium.Parser) Signer {
	s = s.clone()
	s.cfg.referenceParser = &p
	return s
}

// SignEnveloped creates an enveloped signature inside the given parent
// element of the document. The key is a concrete *rsa.PrivateKey,
// *ecdsa.PrivateKey, or ed25519.PrivateKey, any crypto.Signer whose public key
// is one of those types (for example an HSM/KMS/PKCS#11-backed key), or []byte
// for HMAC.
func (s Signer) SignEnveloped(ctx context.Context, doc *helium.Document, parent *helium.Element, key any) error {
	return signEnveloped(ctx, s.config(), doc, parent, key)
}

// SignEnveloping creates an enveloping signature wrapping the given content
// nodes in a <ds:Object>. Returns the (detached) Signature element for the
// caller to place. A configured Reference may point at an element inside the
// content by its Id (URI="#id") — for example a <ds:Manifest> or
// <ds:SignatureProperties> — and it is resolved and digested during signing
// without ever inserting the Signature into the caller's document: an
// in-Object target is canonicalized on its own, and a target in the document
// (URI="#root", even the document element) is digested over its unchanged
// subtree, byte-identical to a signature with no such internal reference. An
// id that matches in both the document and the Signature's own Object content
// is rejected as an ambiguous reference (ErrAmbiguousReference).
//
// An in-Object target is canonicalized under a proxy that reproduces the full
// inherited canonicalization context the target will have once the caller
// places the Signature under the document element — every in-scope namespace
// declaration plus the inherited xml:* attributes, copied per the C14N version
// to match exactly what helium's own canonicalizer inherits to a node-set apex
// (Canonical XML 1.0 inherits every xml:* attribute including xml:id; Canonical
// XML 1.1 inherits only xml:lang/xml:space and lexically joins xml:base) — so a
// reference into the Object verifies under inclusive Canonical XML 1.0 or 1.1.
// Exclusive Canonical XML inherits no xml:*, so its digests are unaffected.
//
// Placement (inclusive C14N only): the same proxy canonicalizes SignedInfo, and
// an in-Object Reference is digested under the context of the signing document
// element. When SignedInfo's CanonicalizationMethod or an in-Object Reference
// uses inclusive Canonical XML (C14N10 / C14N11), the caller MUST place the
// returned Signature directly under the document element, or under an element
// with the same in-scope namespaces and inherited xml:* attributes. Placing it
// under an element that contributes extra in-scope namespace declarations or
// xml:* attributes changes the inclusively-canonicalized bytes, so verification
// recomputes a different canonical form and fails. Exclusive Canonical XML (the
// [NewSigner] default, [ExcC14NTransform]) inherits no such context and is
// unaffected by placement.
//
// Every content entry must be a movable node (helium.MutableNode); an ordinary
// DOM element qualifies. A nil, typed-nil, or read-only content entry (e.g. a
// namespace-node wrapper) is rejected with an indexed error wrapping
// ErrInvalidSignature before any node is moved, rather than being silently
// dropped from the Object. Moving the content into the Object detaches it from
// the caller's tree; if signing then fails at any later step, every moved node
// is restored to its exact original position (parent, siblings, and order),
// leaving the caller's document byte-identical to before the call.
//
// Lifetime: the returned Signature is allocated from doc's slab storage (its
// nodes are created via doc.CreateElement) and is owned by doc, but a successful
// sign leaves it safe to keep after doc.Free(). Canonicalizing SignedInfo grafts
// the live Signature into a throwaway document, a cross-document move that marks
// doc's slab as escaped; doc.Free() then becomes a no-op and never recycles the
// chunks backing the Signature. So the returned Signature stays valid after
// doc.Free() — the caller does NOT need to move it into another document first
// to keep it.
func (s Signer) SignEnveloping(ctx context.Context, doc *helium.Document, content []helium.Node, key any) (*helium.Element, error) {
	return signEnveloping(ctx, s.config(), doc, content, key)
}

// SignDetached creates a detached Signature element referencing URIs
// specified in the configured References. Returns the Signature element.
//
// Placement (inclusive C14N only): SignedInfo is canonicalized under a proxy
// carrying the signing document element's inherited canonicalization context. If
// SignedInfo is canonicalized with inclusive Canonical XML (C14N10 / C14N11) —
// the SignedInfo CanonicalizationMethod, not the Reference transforms — the
// caller MUST place the returned Signature directly under the document element,
// or under an element with the same in-scope namespaces and inherited xml:*
// attributes. Placing it under an element that contributes extra in-scope
// namespace declarations or xml:* attributes changes the bytes inclusive C14N
// canonicalizes for SignedInfo, so verification recomputes a different canonical
// form and fails. Exclusive Canonical XML (the [NewSigner] default,
// [ExcC14NTransform]) inherits no such context and is unaffected by placement.
//
// Lifetime: the returned Signature is allocated from doc's slab storage (its
// nodes are created via doc.CreateElement) and is owned by doc, but a successful
// sign leaves it safe to keep after doc.Free(). Canonicalizing SignedInfo grafts
// the live Signature into a throwaway document, a cross-document move that marks
// doc's slab as escaped; doc.Free() then becomes a no-op and never recycles the
// chunks backing the Signature. So the returned Signature stays valid after
// doc.Free() — the caller does NOT need to move it into another document first
// to keep it.
func (s Signer) SignDetached(ctx context.Context, doc *helium.Document, key any) (*helium.Element, error) {
	return signDetached(ctx, s.config(), doc, key)
}

// Conservative default parse-time resource caps for verification. They bound
// the decode/parse work an unsigned, attacker-controlled Signature element can
// force BEFORE the SignatureValue is checked, while sitting comfortably above
// any legitimate signature so existing documents verify unchanged. A zero
// config field selects the matching default; a negative field disables that cap.
const (
	defaultMaxReferences     = 1024
	defaultMaxKeyInfoEntries = 256
	defaultMaxDecodedBytes   = 10 << 20 // 10 MiB
)

// verifierConfig holds the configuration for a Verifier.
type verifierConfig struct {
	keySource         KeySource
	allowSHA1         bool
	validateManifests bool
	referenceResolver ReferenceResolver
	// referenceParser is the parser used to parse an external reference's octets
	// into XML for a c14n/XPath transform chain. nil selects the locked-down
	// default (see parser).
	referenceParser *helium.Parser
	// xsltTransformer applies the XSLT transform. nil (the default) keeps the XSLT
	// transform fail-closed with ErrUnsupportedTransform.
	xsltTransformer XSLTTransformer
	// Parse-time resource caps (see the default* constants). Zero selects the
	// default; negative disables the cap.
	maxReferences     int
	maxKeyInfoEntries int
	maxDecodedBytes   int
	// allowXPointer opts into resolving a general XPointer Reference URI (an
	// xmlns()*xpointer(expr) framework form) beyond the four safe same-document
	// forms. Default false keeps a general XPointer fail-closed as an external
	// reference, so default verification is byte-identical.
	allowXPointer bool
}

// maxReferencesLimit returns the effective cap on the number of ds:Reference
// elements: the configured value, or the default when unset (zero). A negative
// value is returned as-is, disabling the cap at the enforcement site (which
// gates on limit > 0).
func (cfg *verifierConfig) maxReferencesLimit() int {
	if cfg.maxReferences == 0 {
		return defaultMaxReferences
	}
	return cfg.maxReferences
}

// maxKeyInfoEntriesLimit returns the effective cap on the number of KeyInfo
// entries (X509Data children plus KeyInfo children), defaulting when unset.
func (cfg *verifierConfig) maxKeyInfoEntriesLimit() int {
	if cfg.maxKeyInfoEntries == 0 {
		return defaultMaxKeyInfoEntries
	}
	return cfg.maxKeyInfoEntries
}

// maxDecodedBytesLimit returns the effective cap on total base64-decoded bytes
// across DigestValue/SignatureValue/X509Certificate, defaulting when unset.
func (cfg *verifierConfig) maxDecodedBytesLimit() int {
	if cfg.maxDecodedBytes == 0 {
		return defaultMaxDecodedBytes
	}
	return cfg.maxDecodedBytes
}

// parser returns the parser used for external reference octets: the configured
// ReferenceParser, or a locked-down default (helium.NewParser(): XXE blocked, no
// filesystem, no network) when none was set. The default fails closed on
// external-entity, DTD, and network access so parsing attacker-supplied external
// content cannot reach the host.
func (cfg *verifierConfig) parser() helium.Parser {
	if cfg.referenceParser != nil {
		return *cfg.referenceParser
	}
	return helium.NewParser()
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

// ReferenceResolver configures a [ReferenceResolver] that dereferences external
// Reference URIs (those that are not one of the four supported same-document
// forms). It is opt-in: the default is nil, which keeps external references
// fail-closed with [ErrReferenceNotFound], byte-identical to a Verifier without
// a resolver. When set, an external Reference URI is joined against the
// document's base URI and passed to r; the resolved octets are then run through
// the Reference's transform pipeline before digesting.
//
// A Reference satisfied via the resolver is marked External in the result (see
// [VerifiedReference]); [VerifyResult.Covers] and [VerifyResult.SignedElement]
// never report an external reference as covering in-document content, since it
// resolves to bytes outside the document rather than an element.
func (v Verifier) ReferenceResolver(r ReferenceResolver) Verifier {
	v = v.clone()
	v.cfg.referenceResolver = r
	return v
}

// ValidateManifests controls whether the inner ds:Reference children of a
// Manifest-typed Reference are digested and reported (XMLDSig core §5.1). It is
// opt-in: the default is false, which leaves [VerifyResult.Manifests] nil and
// walks no inner references, byte-identical to a Verifier without it.
//
// When enabled, after a top-level Reference whose Type is [TypeManifest] has
// itself verified (its own digest over the ds:Manifest subtree is checked
// exactly as any other Reference), that Manifest's inner references are each
// resolved, transformed, and digested through the same fail-closed pipeline,
// with the per-reference outcome recorded in [ManifestResult].
//
// Inner-reference results are ADVISORY: per §5.1 the application decides how to
// treat a Manifest, so an inner-reference digest mismatch or an unresolved or
// unsupported inner reference does NOT fail Verify — the top-level Manifest
// Reference's own digest is what the signature commits to. Inner references
// never contribute to [VerifyResult.Covers] or [VerifyResult.SignedElement], so
// coverage is never attributed through a Manifest. Only one level is walked: a
// Manifest nested inside a Manifest is digested but not recursively expanded.
//
// It is off by default because inner references may pull in transforms or
// external URIs the top-level policy did not intend, so evaluating them is left
// to callers who want the report.
func (v Verifier) ValidateManifests(validate bool) Verifier {
	v = v.clone()
	v.cfg.validateManifests = validate
	return v
}

// ReferenceParser configures the [helium.Parser] used to parse an external
// reference's resolved octets into XML when the Reference's transform chain needs
// a node-set (a canonicalization or XPath filter transform). The default is a
// locked-down parser (helium.NewParser(): XXE blocked, no filesystem access, no
// network), so parsing attacker-supplied external content cannot reach the host.
// Override it only to relax those defaults deliberately. It has no effect on a
// reference whose octets are digested directly (an empty or base64-only chain).
func (v Verifier) ReferenceParser(p helium.Parser) Verifier {
	v = v.clone()
	v.cfg.referenceParser = &p
	return v
}

// XSLTTransformer configures the [XSLTTransformer] that applies the XSLT
// transform (http://www.w3.org/TR/1999/REC-xslt-19991116) to a Reference that
// carries one. It is opt-in: the default is nil, which keeps the XSLT transform
// fail-closed with [ErrUnsupportedTransform], byte-identical to a Verifier
// without one. When set, a Reference's ds:Transform/xsl:stylesheet subtree is
// serialized and passed to t together with the pre-XSLT octet stream, and t's
// output becomes the digest input.
//
// XSLT is a powerful language and both the stylesheet and its input are
// attacker-controlled on verify, so the transformer owns all resource and XXE
// policy (see [XSLTTransformer]). helium ships no transformer.
func (v Verifier) XSLTTransformer(t XSLTTransformer) Verifier {
	v = v.clone()
	v.cfg.xsltTransformer = t
	return v
}

// MaxReferences caps the number of ds:Reference elements the verifier parses
// out of SignedInfo. A document whose Signature declares more References than
// the cap is rejected with [ErrResourceLimitExceeded] before any Reference is
// digested, bounding the per-Reference canonicalization work an unsigned,
// attacker-controlled document can force. n <= 0 has special meaning: 0 (the
// default) selects the conservative built-in cap, and a negative n disables the
// cap entirely. The default sits well above any legitimate signature.
func (v Verifier) MaxReferences(n int) Verifier {
	v = v.clone()
	v.cfg.maxReferences = n
	return v
}

// MaxKeyInfoEntries caps the number of KeyInfo entries the verifier parses: the
// KeyInfo element's own children plus every X509Data child (each
// X509Certificate is parsed with x509.ParseCertificate, which is not free). A
// KeyInfo carrying more entries than the cap is rejected with
// [ErrResourceLimitExceeded]. n <= 0 has special meaning: 0 (the default)
// selects the conservative built-in cap, and a negative n disables the cap.
func (v Verifier) MaxKeyInfoEntries(n int) Verifier {
	v = v.clone()
	v.cfg.maxKeyInfoEntries = n
	return v
}

// MaxDecodedBytes caps the running total of base64-decoded bytes the verifier
// produces while parsing the Signature element before the SignatureValue check
// — across DigestValue, SignatureValue, and X509Certificate content. Exceeding
// the cap is rejected with [ErrResourceLimitExceeded], bounding the decode
// allocation an attacker-controlled document can force. n <= 0 has special
// meaning: 0 (the default) selects the conservative built-in cap, and a
// negative n disables the cap.
func (v Verifier) MaxDecodedBytes(n int) Verifier {
	v = v.clone()
	v.cfg.maxDecodedBytes = n
	return v
}

// AllowXPointer opts into resolving a general XPointer Reference URI — an
// XPointer framework URI of the form "#xmlns(prefix=uri)...xpointer(<expr>)"
// (zero or more xmlns() namespace-binding parts followed by one xpointer() XPath
// expression) — in addition to the four safe same-document forms (see
// [Verifier.Verify]). It is opt-in and fail-closed by default: with AllowXPointer
// off (the default) a general XPointer URI is treated as an external reference
// and, without a [ReferenceResolver], rejected with [ErrReferenceNotFound], so
// default verification is byte-identical.
//
// When enabled, the xpointer() expression is evaluated on a bounded XPath 1.0
// evaluator under the document element's in-scope namespaces overlaid with the
// xmlns() bindings, and its result MUST identify a single element apex — the XML
// Signature Wrapping defense: an empty node-set is [ErrReferenceNotFound], and a
// multi-element or non-element node-set is [ErrAmbiguousReference]. A literal
// xpointer(id('X')) keeps the duplicate-detecting id resolution (never a
// last-one-wins id table). The XMLDSig here() function is not available inside a
// URI-borne XPointer.
func (v Verifier) AllowXPointer(allow bool) Verifier {
	v = v.clone()
	v.cfg.allowXPointer = allow
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
//
// Verification honors ctx: an already-cancelled or already-expired context
// short-circuits before any work, and cancellation is rechecked between
// References. Because a SignedInfo may carry arbitrarily many References and each
// empty-URI enveloped Reference canonicalizes a copy of the whole document, the
// per-Reference work scales with the number of References; bound it by passing a
// ctx with a deadline. On cancellation the context error (ctx.Err()) is returned.
func (v Verifier) Verify(ctx context.Context, doc *helium.Document) (*VerifyResult, error) {
	// Honor an already-cancelled or already-expired context before the
	// signature-discovery walk. findSignatureElements below traverses the whole
	// document, which is unbounded on a large or attacker-supplied input, so a
	// context the caller cancelled before calling must short-circuit here rather
	// than pay for the full walk only to return ErrSignatureNotFound.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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
//
// Verification honors ctx the same way as [Verifier.Verify]: an already-cancelled
// or already-expired context short-circuits before any work, cancellation is
// rechecked between References, and a ctx deadline is the lever for bounding the
// per-Reference work of a SignedInfo that carries many References.
func (v Verifier) VerifyElement(ctx context.Context, doc *helium.Document, sig *helium.Element) (*VerifyResult, error) {
	// Honor an already-cancelled or already-expired context before any work,
	// including the nil / local-name / namespace validation guards below. The
	// caller supplies sig directly, so on an attacker-controlled element a
	// cancelled context must short-circuit here rather than pay for the
	// validation before verifySignature's own ctx check would catch it.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Verify reaches verifySignature only through findSignatureElements, which
	// already gates on local-name Signature in the core XML-Signature namespace.
	// VerifyElement takes the target element straight from the caller, so it must
	// perform the same validation before any work: a nil sig (e.g. a caller's
	// lookup that matched nothing) would otherwise nil-deref in
	// parseSignatureElement, and a non-Signature/wrong-namespace element would be
	// parsed as if it were a ds:Signature.
	if sig == nil {
		return nil, fmt.Errorf("%w: nil Signature element", ErrInvalidSignature)
	}
	if domutil.LocalName(sig) != "Signature" || !isDSigCoreNS(sig) {
		return nil, fmt.Errorf("%w: element is not a ds:Signature", ErrInvalidSignature)
	}
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

// isDSigMoreNS reports whether e is in the xmldsig-more namespace
// (http://www.w3.org/2001/04/xmldsig-more#). RFC 4050's legacy ECDSAKeyValue
// and its DomainParameters/NamedCurve/PublicKey children live ONLY in this
// namespace; as with the core and 1.1 checks, matching on local name alone
// would let a foreign-namespace look-alike supply attacker-chosen key material,
// so the exact namespace is required.
func isDSigMoreNS(e *helium.Element) bool {
	return elementNamespaceURI(e) == NamespaceDSigMore
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
