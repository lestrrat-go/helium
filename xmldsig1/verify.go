package xmldsig1

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/lestrrat-go/helium/internal/xmlbase64"
)

// isNilInterface reports whether v is effectively nil. A plain `v == nil` only
// catches an untyped-nil interface; a typed-nil pointer (e.g.
// `var ks *myKeySource; NewVerifier(ks)`, or `Verifier.XSLTTransformer((*t)(nil))`)
// yields a non-nil interface whose underlying value is nil, so calling a
// pointer-receiver method on it would panic on the nil-receiver dereference.
// Detect that case reflectively for any nil-capable underlying kind. It backs
// both the KeySource and XSLTTransformer nil guards so a typed-nil value the
// builder stored verbatim fails closed (ErrNoKeySource / ErrUnsupportedTransform)
// instead of panicking.
func isNilInterface(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Chan, reflect.Func, reflect.Slice, reflect.Interface:
		return rv.IsNil()
	default:
		return false
	}
}

// parsedSignature holds the parsed structure of a ds:Signature element.
type parsedSignature struct {
	signedInfoElem *helium.Element
	c14nMethod     string
	c14nPrefixes   []string // ec:InclusiveNamespaces PrefixList on CanonicalizationMethod
	signatureAlg   string
	references     []parsedReference
	signatureValue []byte
	keyInfoElem    *helium.Element
}

type parsedReference struct {
	uri             string
	refType         string // Reference Type attribute (e.g. TypeManifest); "" when absent
	digestAlgorithm string
	digestValue     []byte
	transforms      []parsedTransform
	prepared        *preparedReference
}

type parsedTransform struct {
	algorithm  string
	prefixes   []string          // for Exclusive C14N InclusiveNamespaces
	xpathExpr  string            // for the XPath filter transform (ds:Transform/XPath text)
	xpathNS    map[string]string // in-scope namespace bindings on the ds:Transform/XPath element
	xpathHere  helium.Node       // the ds:XPath element bearing the expression (here() resolves to it)
	stylesheet []byte            // for the XSLT transform: the serialized xsl:stylesheet subtree
}

// verifyBudget bounds the parse-time work verification performs on an
// attacker-controlled Signature element BEFORE the SignatureValue is checked. An
// unsigned document can otherwise force large base64 decodes (many/large
// DigestValue/SignatureValue/X509Certificate) and a per-cert x509.ParseCertificate
// for every embedded certificate. The three caps come from the Verifier config
// (with conservative defaults); a non-positive effective cap disables that check.
// The context is polled separately inside the KeyInfo/Reference parse loops so a
// cancelled context stops the work promptly rather than only at their boundaries.
type verifyBudget struct {
	maxRefs    int
	maxEntries int
	maxDecoded int
	references int
	entries    int
	decoded    int
}

func newVerifyBudget(cfg *verifierConfig) *verifyBudget {
	return &verifyBudget{
		maxRefs:    cfg.maxReferencesLimit(),
		maxEntries: cfg.maxKeyInfoEntriesLimit(),
		maxDecoded: cfg.maxDecodedBytesLimit(),
	}
}

// addReference counts one ds:Reference and fails closed once the cap is passed.
func (b *verifyBudget) addReference() error {
	b.references++
	if b.maxRefs > 0 && b.references > b.maxRefs {
		return fmt.Errorf("%w: too many References (limit %d)", ErrResourceLimitExceeded, b.maxRefs)
	}
	return nil
}

// addKeyInfoEntry counts one KeyInfo/X509Data entry and fails closed past the cap.
func (b *verifyBudget) addKeyInfoEntry() error {
	b.entries++
	if b.maxEntries > 0 && b.entries > b.maxEntries {
		return fmt.Errorf("%w: too many KeyInfo entries (limit %d)", ErrResourceLimitExceeded, b.maxEntries)
	}
	return nil
}

// consume adds n decoded bytes to the running total and fails closed past the cap.
func (b *verifyBudget) consume(n int) error {
	b.decoded += n
	if b.maxDecoded > 0 && b.decoded > b.maxDecoded {
		return fmt.Errorf("%w: decoded byte budget exceeded (limit %d)", ErrResourceLimitExceeded, b.maxDecoded)
	}
	return nil
}

func verifySignature(ctx context.Context, cfg *verifierConfig, doc *helium.Document, sigElem *helium.Element) (*VerifyResult, error) {
	// Honor an already-cancelled or already-expired context before any work. All
	// of the pre-loop steps below — signature-element parse, weak-algorithm
	// preflight, KeyInfo parse (x509.ParseCertificate per cert), KeySource
	// resolution, SignedInfo canonicalization, and one SignatureValue crypto
	// verify — are bounded but non-trivial, so a context the caller cancelled
	// before calling must short-circuit here rather than run them to completion.
	// The per-Reference loop below repeats this check each iteration.
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// A zero-value Verifier{} constructed directly (bypassing NewVerifier) has
	// a nil cfg, and a nil KeySource (e.g. NewVerifier(nil)) cannot resolve a
	// key. isNilInterface also catches a typed-nil pointer KeySource, whose
	// non-nil interface would otherwise slip past a plain == nil check and panic
	// inside ResolveKey below. Reject all of these up front so config-controlled
	// cases return a typed error instead of panicking on a nil dereference.
	if cfg == nil || isNilInterface(cfg.keySource) {
		return nil, ErrNoKeySource
	}

	// budget bounds the pre-SignatureValue parse work (base64 decodes, per-cert
	// x509 parses) and carries ctx for in-loop cancellation polling. It is shared
	// across the signature-element parse and the KeyInfo parse below so the
	// decoded-byte total spans both.
	budget := newVerifyBudget(cfg)

	parsed, err := parseSignatureElement(ctx, budget, sigElem)
	if err != nil {
		return nil, err
	}

	// Reject weak (SHA-1) signature/digest algorithms before resolving KeyInfo
	// or invoking KeySource, so a rejected SHA-1 input returns ErrWeakAlgorithm
	// without triggering key resolution or surfacing unrelated key/signature
	// errors.
	if err := preflightParsedWeakAlgorithms(parsed, cfg.allowSHA1); err != nil {
		return nil, err
	}

	// Compile and statically validate every Reference's XPath-bearing state
	// before resolving KeyInfo resources or doing any other callback-driven
	// work. Besides making the prepared state reusable in the digest loop, this
	// prevents any Reference resolver or XSLT transformer from running when a
	// later Reference carries an invalid XPath filter or general XPointer
	// expression.
	if err := preflightVerifierReferences(ctx, cfg, doc, parsed); err != nil {
		return nil, err
	}

	// Resolve key.
	var keyInfoData *KeyInfoData
	if parsed.keyInfoElem != nil {
		keyInfoData, err = parseKeyInfo(ctx, budget, parsed.keyInfoElem)
		if err != nil {
			return nil, err
		}
		// A second, resolution-aware pass dereferences any ds:RetrievalMethod and
		// merges the retrieved certificate material into keyInfoData before key
		// resolution. It inherits the external-Reference fail-closed posture
		// (opt-in resolver, size cap, base-URI join).
		if err := resolveRetrievalMethods(ctx, budget, cfg, doc, parsed.keyInfoElem, keyInfoData); err != nil {
			return nil, err
		}
	}

	key, err := cfg.keySource.ResolveKey(ctx, keyInfoData, parsed.signatureAlg)
	if err != nil {
		return nil, err
	}

	// Canonicalize SignedInfo, honoring any ec:InclusiveNamespaces PrefixList
	// declared on its CanonicalizationMethod (relevant for Exclusive C14N).
	canonical, err := canonicalizeSubtree(parsed.c14nMethod, parsed.signedInfoElem, parsed.c14nPrefixes)
	if err != nil {
		return nil, err
	}

	// Verify signature value. SHA-1-based signature algorithms are rejected
	// here unless the caller opted in via Verifier.AllowSHA1(true).
	if err := verifyBytes(parsed.signatureAlg, key, canonical, parsed.signatureValue, cfg.allowSHA1); err != nil {
		return nil, &VerificationError{Reference: -1, Err: err}
	}

	// Verify each reference and record the resolved element so callers can
	// confirm that the element they intend to consume is actually covered.
	result := &VerifyResult{Signature: sigElem}
	for i, ref := range parsed.references {
		// Honor context cancellation between references: a signature with very
		// many References must not be digested to completion once the caller's
		// context is cancelled or its deadline has passed.
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		target, external, err := verifyReference(ctx, cfg, doc, sigElem, ref)
		if err != nil {
			return nil, &VerificationError{Reference: i, URI: ref.uri, Err: err}
		}
		result.References = append(result.References, VerifiedReference{
			URI:             ref.uri,
			Element:         target,
			External:        external,
			DigestAlgorithm: ref.digestAlgorithm,
			Type:            ref.refType,
		})
	}

	// After every top-level Reference digest has verified, optionally walk the
	// inner references of any Manifest-typed Reference (XMLDSig core §5.1). This
	// is opt-in (Verifier.ValidateManifests) and purely advisory: inner digests
	// are recorded but never change whether Verify succeeds, and coverage is
	// never attributed through a Manifest. Taking addresses into
	// result.References is safe because that slice is complete here.
	if cfg.validateManifests {
		for i, ref := range parsed.references {
			if ref.refType != TypeManifest {
				continue
			}
			// A Manifest whose top-level Reference resolved externally (Element
			// nil) has no in-document Manifest element to walk; skip it.
			manifestElem := result.References[i].Element
			if manifestElem == nil {
				continue
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			result.Manifests = append(result.Manifests, ManifestResult{
				Reference:  &result.References[i],
				Element:    manifestElem,
				References: validateManifestReferences(ctx, budget, cfg, doc, sigElem, manifestElem),
			})
		}
	}

	return result, nil
}

func verifyReference(ctx context.Context, cfg *verifierConfig, doc *helium.Document, sigElem *helium.Element, ref parsedReference) (*helium.Element, bool, error) {
	target, canonical, external, err := canonicalizeReference(ctx, cfg, doc, sigElem, ref)
	if err != nil {
		return nil, false, err
	}

	// Compute and compare digest. A SHA-1 digest is rejected unless the
	// caller opted in via Verifier.AllowSHA1(true).
	computed, err := computeDigest(ref.digestAlgorithm, canonical, cfg.allowSHA1)
	if err != nil {
		return nil, false, err
	}

	if !digestEqual(computed, ref.digestValue) {
		return nil, false, ErrDigestMismatch
	}

	return target, external, nil
}

// preflightVerifierReferences prepares every top-level Reference before the
// digest loop starts. The pass is deliberately side-effect free: it only
// interprets transform lists and compiles/validates XPath filter and general
// XPointer expressions. Resolver and transformer calls remain in the execution
// path after the complete pass succeeds.
func preflightVerifierReferences(ctx context.Context, cfg *verifierConfig, doc *helium.Document, parsed *parsedSignature) error {
	for i := range parsed.references {
		if err := ctx.Err(); err != nil {
			return err
		}
		ref := &parsed.references[i]
		prepared, err := prepareReferenceForVerification(cfg, doc, *ref)
		if err != nil {
			return &VerificationError{Reference: i, URI: ref.uri, Err: err}
		}
		ref.prepared = prepared
	}
	return nil
}

func prepareReferenceForVerification(cfg *verifierConfig, doc *helium.Document, ref parsedReference) (*preparedReference, error) {
	steps := make([]transformStep, len(ref.transforms))
	for i, t := range ref.transforms {
		steps[i] = transformStep(t)
	}
	pipe, err := resolveTransformPipeline(steps)
	if err != nil {
		return nil, err
	}

	prepared := &preparedReference{
		pipeline:        pipe,
		hasExplicitC14N: stepsHaveC14N(steps),
	}
	if !cfg.allowXPointer {
		return prepared, nil
	}
	if _, _, _, ok := referenceURIForm(ref.uri); ok {
		return prepared, nil
	}
	overrides, expr, matched := parseGeneralXPointer(ref.uri)
	if !matched {
		return prepared, nil
	}
	prepared.generalXPointer, err = prepareGeneralXPointer(doc, overrides, expr)
	if err != nil {
		return nil, err
	}
	return prepared, nil
}

// canonicalizeReference resolves a Reference URI and applies its transform
// pipeline, returning the resolved target element (nil for an external
// reference), the canonical octet stream that the DigestValue is computed over,
// and whether the reference was satisfied externally. It is the shared reference
// node-set → octet path for the verify digest check.
func canonicalizeReference(ctx context.Context, cfg *verifierConfig, doc *helium.Document, sigElem *helium.Element, ref parsedReference) (*helium.Element, []byte, bool, error) {
	prepared := ref.prepared
	if prepared == nil {
		var err error
		prepared, err = prepareReferenceForVerification(cfg, doc, ref)
		if err != nil {
			return nil, nil, false, err
		}
	}

	// A URI that is not one of the four same-document forms is either a general
	// XPointer (opt-in) or an external reference.
	if _, _, _, ok := referenceURIForm(ref.uri); !ok {
		// A general XPointer is resolved only when Verifier.AllowXPointer(true)
		// opted in; otherwise it stays fail-closed as an external reference, so the
		// default four-form behavior is byte-identical.
		if cfg.allowXPointer {
			target, canonical, handled, err := canonicalizeGeneralXPointer(ctx, cfg, doc, sigElem, prepared)
			if handled {
				if err != nil {
					return nil, nil, false, err
				}
				return target, canonical, false, nil
			}
		}
		ref.prepared = prepared
		octets, err := resolveExternalReference(ctx, cfg, doc, ref)
		if err != nil {
			return nil, nil, false, err
		}
		return nil, octets, true, nil
	}

	target, err := resolveReference(doc, ref.uri)
	if err != nil {
		return nil, nil, false, err
	}

	// Classify the URI's node-set form (§4.3.3.2-3). wholeDoc selects the
	// document root; includeComments governs whether comment nodes are part of
	// the selected node-set.
	_, wholeDoc, includeComments, _ := referenceURIForm(ref.uri)

	canonical, err := applyReferenceTransforms(ctx, cfg, doc, sigElem, target, wholeDoc, includeComments, prepared.pipeline)
	if err != nil {
		return nil, nil, false, err
	}
	return target, canonical, false, nil
}

// applyReferenceTransforms interprets a Reference's transform list as an ordered
// pipeline and returns the canonical octet stream its DigestValue is computed
// over, given the already-resolved target element and the reference form's
// wholeDoc / includeComments classification. It is shared by the four same-document
// forms and the general XPointer resolver so both interpret a transform list
// identically.
//
// Fail closed: any transform whose URI cannot be applied, or one ordered after an
// octet-producing c14n transform, is rejected before digesting — otherwise a
// Reference could declare an unsupported or mis-ordered transform and still verify
// against the untransformed canonical bytes. When no c14n transform is declared
// the default node-set->octet conversion is inclusive Canonical XML 1.0.
func applyReferenceTransforms(ctx context.Context, cfg *verifierConfig, doc *helium.Document, sigElem, target *helium.Element, wholeDoc, includeComments bool, pipe transformPipeline) ([]byte, error) {
	// The base64 decode transform ends the pipeline with decoded octets that are
	// digested directly (XMLDSig core §6.6.2): the resolved node-set's XPath 1.0
	// string-value is base64-decoded and no canonicalization runs afterward.
	// resolveTransformPipeline already fails closed on any transform ordered
	// after base64; combining it with a preceding node-set transform
	// (enveloped-signature or XPath filter) is not supported and is rejected
	// fail-closed here rather than digesting an unintended string-value.
	if pipe.base64 {
		if pipe.hasEnveloped || len(pipe.xpathFilters) > 0 {
			return nil, fmt.Errorf("%w: base64 transform combined with a node-set transform", ErrUnsupportedTransform)
		}
		return base64TransformOctets(target)
	}

	// A C14N WithComments method only emits comment nodes present in the set, so
	// when the form excludes comments the method is downgraded to its plain
	// variant — keeping "#id"/"" free of comments even under a WithComments c14n,
	// and reserving comments for the #xpointer forms.
	c14nMethod := effectiveC14NMethod(pipe.c14nMethod, includeComments)

	// Compute the pre-XSLT octets. When one or more XPath filter transforms are
	// present the reference is processed as an explicit node-set: build the
	// initial node-set, apply the enveloped-signature removal and each XPath
	// filter in order, then canonicalize the surviving node-set. Otherwise the
	// enveloped/whole-document/subtree canonicalization applies. For enveloped
	// signatures the Signature element and its descendants must be omitted from
	// the canonical input; canonicalizeEnveloped does this on a deep copy of the
	// document, never mutating the caller's live DOM (which would race with
	// concurrent readers and risk leaving the tree corrupted if a restore failed).
	// None of these paths change when no XSLT transform is present, so a Reference
	// without XSLT canonicalizes byte-identically.
	var (
		canonical []byte
		err       error
	)
	switch {
	case len(pipe.xpathFilters) > 0:
		canonical, err = canonicalizeWithXPathFilters(ctx, doc, target, sigElem, wholeDoc, c14nMethod, pipe)
	case pipe.hasEnveloped:
		canonical, err = canonicalizeEnveloped(c14nMethod, doc, target, sigElem, wholeDoc, pipe.prefixes)
	case wholeDoc:
		canonical, err = canonicalize(c14nMethod, doc, pipe.prefixes)
	default:
		canonical, err = canonicalizeSubtree(c14nMethod, target, pipe.prefixes)
	}
	if err != nil {
		return nil, err
	}

	// XSLT transform (verify-only, opt-in): octet-in -> octet-out. The octets
	// above are the pre-XSLT input; hand them plus the stylesheet to the injected
	// transformer and digest its output. Fail closed with ErrUnsupportedTransform
	// when no transformer is configured, mirroring the "no HTTP resolver shipped"
	// stance — helium never runs attacker-controlled XSLT on its own.
	if pipe.xslt != nil {
		if isNilInterface(cfg.xsltTransformer) {
			return nil, fmt.Errorf("%w: XSLT transform requires a configured XSLTTransformer", ErrUnsupportedTransform)
		}
		canonical, err = cfg.xsltTransformer.TransformXSLT(ctx, pipe.xslt, canonical)
		if err != nil {
			return nil, err
		}
	}
	return canonical, nil
}

// canonicalizeGeneralXPointer resolves a general XPointer Reference URI (opt-in,
// XPointer framework: zero+ xmlns() parts then one xpointer(<expr>)) to its
// single element apex and applies the Reference's transform pipeline over that
// subtree. handled reports whether the prepared state matched the general
// XPointer shape at all: when it did not, the caller falls through to
// external-reference handling; when it did, the returned err (if any) is the
// fail-closed resolution result.
//
// The apex is enforced to a SINGLE element (the XSW defense, singleElementApex):
// an empty node-set is ErrReferenceNotFound and a scattered/multi-element or
// non-element node-set is ErrAmbiguousReference. The full-XPointer forms include
// comment nodes, so includeComments is true. here() is NOT registered for a
// URI-borne XPointer, so an xpointer(here()...) fails closed.
func canonicalizeGeneralXPointer(ctx context.Context, cfg *verifierConfig, doc *helium.Document, sigElem *helium.Element, prepared *preparedReference) (*helium.Element, []byte, bool, error) {
	if prepared.generalXPointer == nil {
		return nil, nil, false, nil
	}
	target, err := resolvePreparedGeneralXPointerTarget(ctx, doc, prepared.generalXPointer)
	if err != nil {
		return nil, nil, true, err
	}
	canonical, err := applyReferenceTransforms(ctx, cfg, doc, sigElem, target, false, true, prepared.pipeline)
	if err != nil {
		return nil, nil, true, err
	}
	return target, canonical, true, nil
}

// resolveExternalReference dereferences an external Reference URI through the
// configured ReferenceResolver and returns the octet stream its DigestValue is
// computed over. Without a resolver it stays fail-closed with the same
// ErrReferenceNotFound the same-document resolver returns, so a nil-resolver
// Verifier is byte-identical to before. The URI is joined against the document's
// base URI before resolution; the resolved octets then run through the
// Reference's transform pipeline (see externalReferenceDigestInput).
func resolveExternalReference(ctx context.Context, cfg *verifierConfig, doc *helium.Document, ref parsedReference) ([]byte, error) {
	if cfg.referenceResolver == nil {
		return nil, fmt.Errorf("%w: unsupported reference URI: %s", ErrReferenceNotFound, ref.uri)
	}

	prepared := ref.prepared
	if prepared == nil {
		var err error
		prepared, err = prepareReferenceForVerification(cfg, doc, ref)
		if err != nil {
			return nil, err
		}
	}

	joined, err := joinReferenceURI(doc.URL(), ref.uri)
	if err != nil {
		return nil, err
	}
	octets, err := resolveReferenceOctets(ctx, cfg.referenceResolver, joined)
	if err != nil {
		return nil, err
	}
	preXSLT, err := externalReferenceDigestInput(ctx, octets, prepared.pipeline, prepared.hasExplicitC14N, cfg.parser())
	if err != nil {
		return nil, err
	}

	// Apply the XSLT transform (verify-only, opt-in) exactly as the same-document
	// path does: preXSLT is the pre-XSLT octet stream, so fail closed with
	// ErrUnsupportedTransform when no transformer is configured (or a typed-nil one
	// was stored), otherwise hand the stylesheet plus those octets to the injected
	// transformer and digest its output. externalReferenceDigestInput does not
	// consult pipe.xslt, so without this an external Reference declaring an XSLT
	// transform would silently digest the untransformed octets, bypassing the
	// fail-closed invariant. The sign path never reaches here with an XSLT step:
	// preflightSignerTransforms rejects it fail-closed before dereferencing.
	if prepared.pipeline.xslt != nil {
		if isNilInterface(cfg.xsltTransformer) {
			return nil, fmt.Errorf("%w: XSLT transform requires a configured XSLTTransformer", ErrUnsupportedTransform)
		}
		return cfg.xsltTransformer.TransformXSLT(ctx, prepared.pipeline.xslt, preXSLT)
	}
	return preXSLT, nil
}

// canonicalizeWithXPathFilters processes a Reference that carries one or more
// XPath filter transforms. It materializes the initial node-set for the
// reference form, drops the enveloped Signature's own subtree when the
// enveloped-signature transform is present, applies each XPath filter in
// declared order, and canonicalizes the surviving node-set.
func canonicalizeWithXPathFilters(ctx context.Context, doc *helium.Document, target, sigElem *helium.Element, wholeDoc bool, c14nMethod string, pipe transformPipeline) ([]byte, error) {
	var nodes []helium.Node
	if wholeDoc {
		nodes = collectDocumentNodes(doc)
	} else {
		nodes = collectSubtreeNodes(target)
	}
	if pipe.hasEnveloped {
		nodes = removeSignatureNodes(nodes, sigElem)
	}
	for _, f := range pipe.xpathFilters {
		filtered, err := applyXPathFilter(ctx, nodes, f)
		if err != nil {
			return nil, err
		}
		nodes = filtered
	}
	return canonicalizeNodeSet(c14nMethod, nodes, doc, pipe.prefixes)
}

func digestEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	// Constant-time comparison to avoid timing attacks.
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

func parseSignatureElement(ctx context.Context, budget *verifyBudget, sigElem *helium.Element) (*parsedSignature, error) {
	parsed := &parsedSignature{}

	// The XML-Signature schema mandates exactly one SignedInfo and exactly one
	// SignatureValue per ds:Signature. This MUST be enforced rather than
	// last-one-wins: only a single SignedInfo is canonicalized and checked
	// against SignatureValue (see verifySignature), yet every SignedInfo's
	// References were being appended to the result. An attacker could prepend
	// a second, UNSIGNED SignedInfo whose References carry attacker-computed,
	// self-consistent digests; those References would then be reported as
	// verified even though they were never covered by the signature. Reject
	// duplicate SignedInfo / SignatureValue / KeyInfo outright.
	var signatureValueSeen bool
	for child := sigElem.FirstChild(); child != nil; child = child.NextSibling() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		elem, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		// Only elements in the XML-Signature namespace count as core signature
		// children. Matching on local name alone lets a foreign-namespace
		// element masquerade as a core child (e.g. an <evil:Reference> passing
		// the at-least-one-Reference check), so namespace must be enforced. Core
		// children live ONLY in the core xmldsig# namespace; the 1.1 xmldsig11#
		// namespace is for new 1.1 elements and must not satisfy this check.
		if !isDSigCoreNS(elem) {
			continue
		}
		switch domutil.LocalName(elem) {
		case "SignedInfo":
			if parsed.signedInfoElem != nil {
				return nil, fmt.Errorf("%w: multiple SignedInfo elements", ErrInvalidSignature)
			}
			parsed.signedInfoElem = elem
			if err := parseSignedInfo(ctx, budget, elem, parsed); err != nil {
				return nil, err
			}
		case "SignatureValue":
			if signatureValueSeen {
				return nil, fmt.Errorf("%w: multiple SignatureValue elements", ErrInvalidSignature)
			}
			signatureValueSeen = true
			decoded, err := xmlbase64.DecodeString(domutil.TextContent(elem))
			if err != nil {
				return nil, fmt.Errorf("%w: invalid SignatureValue base64: %v", ErrInvalidSignature, err)
			}
			if err := budget.consume(len(decoded)); err != nil {
				return nil, err
			}
			parsed.signatureValue = decoded
		case "KeyInfo":
			if parsed.keyInfoElem != nil {
				return nil, fmt.Errorf("%w: multiple KeyInfo elements", ErrInvalidSignature)
			}
			parsed.keyInfoElem = elem
		}
	}

	if parsed.signedInfoElem == nil {
		return nil, fmt.Errorf("%w: missing SignedInfo", ErrInvalidSignature)
	}
	if !signatureValueSeen {
		return nil, fmt.Errorf("%w: missing SignatureValue", ErrInvalidSignature)
	}

	return parsed, nil
}

func parseSignedInfo(ctx context.Context, budget *verifyBudget, elem *helium.Element, parsed *parsedSignature) error {
	// The XML-Signature schema fixes SignedInfo's content model as
	// (CanonicalizationMethod, SignatureMethod, Reference+) with exactly one
	// CanonicalizationMethod and exactly one SignatureMethod. Enforce that
	// cardinality rather than accepting duplicates last-one-wins: a crafted
	// SignedInfo carrying two SignatureMethod (or CanonicalizationMethod)
	// children is schema-invalid and ambiguous about which algorithm the
	// signature actually commits to, so a conforming verifier must reject it.
	var c14nSeen, sigMethodSeen bool
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if err := ctx.Err(); err != nil {
			return err
		}
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		// Require the XML-Signature namespace: a foreign-namespace
		// <evil:Reference> must not be counted toward the mandatory
		// at-least-one-Reference rule below, which would otherwise re-open the
		// no-content-signature bypass. Only the core xmldsig# namespace counts;
		// the 1.1 xmldsig11# namespace must not satisfy this check.
		if !isDSigCoreNS(e) {
			continue
		}
		switch domutil.LocalName(e) {
		case "CanonicalizationMethod":
			if c14nSeen {
				return fmt.Errorf("%w: multiple CanonicalizationMethod elements", ErrInvalidSignature)
			}
			c14nSeen = true
			alg, ok := e.GetAttribute("Algorithm")
			if !ok {
				return fmt.Errorf("%w: CanonicalizationMethod missing Algorithm", ErrInvalidSignature)
			}
			parsed.c14nMethod = alg
			prefixes, err := parseCanonicalizationParameters(e, alg)
			if err != nil {
				return err
			}
			parsed.c14nPrefixes = prefixes
		case "SignatureMethod":
			if sigMethodSeen {
				return fmt.Errorf("%w: multiple SignatureMethod elements", ErrInvalidSignature)
			}
			sigMethodSeen = true
			alg, ok := e.GetAttribute("Algorithm")
			if !ok {
				return fmt.Errorf("%w: SignatureMethod missing Algorithm", ErrInvalidSignature)
			}
			parsed.signatureAlg = alg
			if err := rejectSignatureMethodParameters(e); err != nil {
				return err
			}
		case "Reference":
			// Count the Reference before parsing it so a flood of References is
			// rejected up front, bounding the per-Reference canonicalization the
			// verify loop would otherwise perform.
			if err := budget.addReference(); err != nil {
				return err
			}
			ref, err := parseReferenceElement(ctx, budget, e)
			if err != nil {
				return err
			}
			parsed.references = append(parsed.references, ref)
		}
	}

	// SignedInfo's content model fixes CanonicalizationMethod and
	// SignatureMethod at exactly one each, not merely at most one. Enforcing
	// only "at most one" lets a SignedInfo missing either element parse OK and
	// fail much later — as an unsupported-algorithm error, sometimes only after
	// key resolution — instead of as a clean ErrInvalidSignature. Reject the
	// absence here so a structurally invalid SignedInfo never reaches
	// canonicalization or key resolution.
	if !c14nSeen {
		return fmt.Errorf("%w: missing CanonicalizationMethod", ErrInvalidSignature)
	}
	if !sigMethodSeen {
		return fmt.Errorf("%w: missing SignatureMethod", ErrInvalidSignature)
	}

	// XML-Signature requires at least one Reference. A SignatureValue computed
	// over a reference-free SignedInfo verifies cryptographically yet covers no
	// document content, so accepting it would attest to nothing. Reject it.
	if len(parsed.references) == 0 {
		return fmt.Errorf("%w: SignedInfo has no Reference", ErrInvalidSignature)
	}
	return nil
}

// parseTransformList parses the ds:Transform children of a ds:Transforms element
// into the ordered parsedTransform list, shared by Reference and RetrievalMethod
// parsing so both interpret a transform list identically. A Transform must be in
// the core XML-Signature namespace; a foreign-namespace look-alike (e.g.
// <evil:Transform Algorithm="...">) is ignored, and the 1.1 xmldsig11# namespace
// does not satisfy the check. Each Transform's Algorithm and its validated
// parameters (the XPath expression, or an ec:InclusiveNamespaces prefix list) are
// recorded; an unrecognized parameter child is rejected fail-closed rather than
// digested as if absent.
func parseTransformList(transformsElem *helium.Element) ([]parsedTransform, error) {
	var transforms []parsedTransform
	for tc := transformsElem.FirstChild(); tc != nil; tc = tc.NextSibling() {
		te, ok := helium.AsNode[*helium.Element](tc)
		if !ok {
			continue
		}
		// A Transform element must be in the core XML-Signature namespace; its
		// InclusiveNamespaces child lives in the xml-exc-c14n namespace and is
		// handled separately below.
		if !isDSigCoreNS(te) {
			continue
		}
		if domutil.LocalName(te) != "Transform" {
			continue
		}
		alg, _ := te.GetAttribute("Algorithm")
		// The XPath filter transform carries its expression in a
		// ds:Transform/XPath child element (not an ec:InclusiveNamespaces
		// parameter), so parse it separately; every other transform is validated
		// by the InclusiveNamespaces gate below.
		if alg == TransformXPath {
			expr, nsBindings, hereElem, err := parseXPathTransform(te)
			if err != nil {
				return nil, err
			}
			transforms = append(transforms, parsedTransform{algorithm: alg, xpathExpr: expr, xpathNS: nsBindings, xpathHere: hereElem})
			continue
		}
		// The XSLT transform carries its stylesheet in a
		// ds:Transform/xsl:stylesheet child element (not an
		// ec:InclusiveNamespaces parameter), so capture that subtree
		// separately. Whether the XSLT transform can actually run is decided
		// later (fail-closed unless a Verifier.XSLTTransformer is configured);
		// parsing it here does not accept an unsupported transform, it only
		// records the stylesheet the pipeline will hand to the transformer.
		if alg == TransformXSLT {
			stylesheet, err := parseXSLTTransform(te)
			if err != nil {
				return nil, err
			}
			transforms = append(transforms, parsedTransform{algorithm: alg, stylesheet: stylesheet})
			continue
		}
		// Validate the Transform's child elements by algorithm. For a supported
		// transform those children are algorithm parameters; accepting an unknown
		// one while processing as if it were absent would be fail-open. The only
		// parameter helium honors is ec:InclusiveNamespaces under the exclusive
		// c14n transforms; every other child — and ec:InclusiveNamespaces under a
		// non-exclusive algorithm — is rejected fail-closed.
		prefixes, err := parseInclusiveNamespaceParameters(te, alg, "Transform")
		if err != nil {
			return nil, err
		}
		transforms = append(transforms, parsedTransform{algorithm: alg, prefixes: prefixes})
	}
	return transforms, nil
}

func parseReferenceElement(ctx context.Context, budget *verifyBudget, elem *helium.Element) (parsedReference, error) {
	ref := parsedReference{}
	ref.uri, _ = elem.GetAttribute("URI")
	// The Type attribute is advisory metadata (XMLDSig core §4.3.3.1): it names
	// what the URI points at (e.g. a ds:Manifest). It never affects the
	// top-level digest; it only gates the opt-in Manifest inner-reference walk.
	ref.refType, _ = elem.GetAttribute("Type")

	// The XML-Signature schema fixes Reference's content model as
	// (Transforms?, DigestMethod, DigestValue) with at most one Transforms and
	// exactly one DigestMethod and one DigestValue. Enforce that cardinality
	// rather than accepting duplicates last-one-wins: a crafted Reference with
	// two DigestValue children (the second crafted to match the recomputed
	// digest) is schema-invalid and ambiguous about which digest the signature
	// commits to, so a conforming verifier must reject it.
	var transformsSeen, digestMethodSeen, digestValueSeen bool
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if err := ctx.Err(); err != nil {
			return ref, err
		}
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		// Core Reference children (Transforms/DigestMethod/DigestValue) must be
		// in the core XML-Signature namespace; do not honor foreign-namespace
		// look-alikes, and the 1.1 xmldsig11# namespace must not satisfy this
		// check.
		if !isDSigCoreNS(e) {
			continue
		}
		switch domutil.LocalName(e) {
		case "Transforms":
			if transformsSeen {
				return ref, fmt.Errorf("%w: multiple Transforms elements", ErrInvalidSignature)
			}
			transformsSeen = true
			transforms, err := parseTransformList(e)
			if err != nil {
				return ref, err
			}
			ref.transforms = transforms
		case "DigestMethod":
			if digestMethodSeen {
				return ref, fmt.Errorf("%w: multiple DigestMethod elements", ErrInvalidSignature)
			}
			digestMethodSeen = true
			alg, ok := e.GetAttribute("Algorithm")
			if !ok {
				return ref, fmt.Errorf("%w: DigestMethod missing Algorithm", ErrInvalidSignature)
			}
			ref.digestAlgorithm = alg
		case "DigestValue":
			if digestValueSeen {
				return ref, fmt.Errorf("%w: multiple DigestValue elements", ErrInvalidSignature)
			}
			digestValueSeen = true
			decoded, err := xmlbase64.DecodeString(domutil.TextContent(e))
			if err != nil {
				return ref, fmt.Errorf("%w: invalid DigestValue base64: %v", ErrInvalidSignature, err)
			}
			if err := budget.consume(len(decoded)); err != nil {
				return ref, err
			}
			ref.digestValue = decoded
		}
	}

	// Reference's content model fixes DigestMethod and DigestValue at exactly
	// one each, not merely at most one. Enforcing only "at most one" lets a
	// Reference missing either element parse OK and fail much later — a missing
	// DigestMethod surfaces as an unsupported-digest error and a missing
	// DigestValue as a digest mismatch (the empty digest never matches) —
	// instead of as a clean ErrInvalidSignature. Reject the absence here.
	if !digestMethodSeen {
		return ref, fmt.Errorf("%w: missing DigestMethod", ErrInvalidSignature)
	}
	if !digestValueSeen {
		return ref, fmt.Errorf("%w: missing DigestValue", ErrInvalidSignature)
	}
	return ref, nil
}

// excInclusiveNamespacePrefixes reports whether elem is an ec:InclusiveNamespaces
// element. InclusiveNamespaces is an Exclusive XML Canonicalization element and
// lives ONLY in the exc-c14n namespace (http://www.w3.org/2001/10/xml-exc-c14n#),
// not the core XML-Signature namespace. Matching on local name alone would let a
// foreign-namespace look-alike inject a PrefixList and alter which namespaces are
// canonicalized, so the exact exc-c14n namespace is required. When it matches, the
// PrefixList attribute is split into its individual prefixes.
func excInclusiveNamespacePrefixes(elem *helium.Element) ([]string, bool) {
	if !isExcC14NNS(elem) || domutil.LocalName(elem) != "InclusiveNamespaces" {
		return nil, false
	}
	pl, _ := elem.GetAttribute("PrefixList")
	if pl == "" {
		return nil, true
	}
	return strings.Fields(pl), true
}

// parseCanonicalizationParameters extracts the ec:InclusiveNamespaces PrefixList
// from a CanonicalizationMethod element and fails closed on any other child
// element, which would be a canonicalization parameter we cannot honor. Silently
// ignoring an unknown parameter would canonicalize SignedInfo differently from
// what the signer intended, so it is rejected.
func parseCanonicalizationParameters(elem *helium.Element, alg string) ([]string, error) {
	return parseInclusiveNamespaceParameters(elem, alg, "CanonicalizationMethod")
}

// parseInclusiveNamespaceParameters validates the child elements of a
// CanonicalizationMethod (SignedInfo) or per-Reference Transform element and
// returns the ec:InclusiveNamespaces PrefixList when one is present. It is the
// single fail-closed gate for both call sites so they behave identically.
//
// The only honored child is ec:InclusiveNamespaces, and only under the
// exclusive c14n algorithms (ExcC14N10 / ExcC14N10Comments); canonicalize()
// honors the PrefixList for exclusive modes alone. Under any other algorithm
// (enveloped-signature/C14N10/C14N11/...) the prefixes are silently dropped
// during canonicalization, so an ec:InclusiveNamespaces there — even with an
// empty PrefixList — is rejected. Any other child element is an unknown
// parameter we cannot honor; accepting it while digesting as if absent would be
// fail-open, so it too is rejected. A second ec:InclusiveNamespaces is rejected
// rather than silently letting the last one win. The context label
// ("CanonicalizationMethod" / "Transform") only shapes the error message.
func parseInclusiveNamespaceParameters(elem *helium.Element, alg, context string) ([]string, error) {
	var prefixes []string
	var seen bool
	for c := elem.FirstChild(); c != nil; c = c.NextSibling() {
		ce, ok := helium.AsNode[*helium.Element](c)
		if !ok {
			continue
		}
		px, matched := excInclusiveNamespacePrefixes(ce)
		if !matched {
			return nil, fmt.Errorf("%w: unsupported %s parameter %s", ErrUnsupportedTransform, context, domutil.LocalName(ce))
		}
		if err := gateInclusiveNamespaces(alg); err != nil {
			return nil, err
		}
		if seen {
			return nil, fmt.Errorf("%w: multiple ec:InclusiveNamespaces under %s", ErrUnsupportedTransform, context)
		}
		seen = true
		prefixes = px
	}
	return prefixes, nil
}

// gateInclusiveNamespaces rejects an ec:InclusiveNamespaces PrefixList declared
// under a non-exclusive c14n algorithm. The PrefixList is only honored for the
// exclusive c14n modes (ExcC14N10 / ExcC14N10Comments); on any other algorithm
// (C14N10/C14N11/enveloped-signature/...) the prefixes are silently dropped
// during canonicalization, so accepting one would canonicalize differently from
// what the signer declared. Shared by the SignedInfo CanonicalizationMethod and
// per-Reference Transform gating so both fail closed identically.
func gateInclusiveNamespaces(alg string) error {
	if alg != ExcC14N10 && alg != ExcC14N10Comments {
		return fmt.Errorf("%w: ec:InclusiveNamespaces is only valid for exclusive c14n, not %s", ErrUnsupportedTransform, alg)
	}
	return nil
}

// parseXPathTransform extracts the XPath filter transform's expression and the
// in-scope namespace bindings it must be evaluated under, from a ds:Transform
// element whose Algorithm is TransformXPath. The expression is the text of the
// single mandatory ds:XPath child; the namespace bindings are that element's
// in-scope prefix declarations (the XPath transform is evaluated with the
// XPath element's namespace context per XMLDSig core §6.6.3). It fails closed on
// a missing, duplicate, or foreign child so a malformed XPath transform never
// digests an unfiltered node-set. The default (empty-prefix) namespace is not
// forwarded: XPath 1.0 has no default element namespace, so an unprefixed name
// test matches only no-namespace nodes. The ds:XPath element itself is returned
// as the bearing node so the here() function (core §6.6.3.1) resolves to it.
func parseXPathTransform(te *helium.Element) (string, map[string]string, *helium.Element, error) {
	var xpathElem *helium.Element
	for c := te.FirstChild(); c != nil; c = c.NextSibling() {
		ce, ok := helium.AsNode[*helium.Element](c)
		if !ok {
			continue
		}
		if !isDSigCoreNS(ce) || domutil.LocalName(ce) != "XPath" {
			return "", nil, nil, fmt.Errorf("%w: unsupported XPath transform child %s", ErrUnsupportedTransform, domutil.LocalName(ce))
		}
		if xpathElem != nil {
			return "", nil, nil, fmt.Errorf("%w: multiple XPath elements in XPath transform", ErrUnsupportedTransform)
		}
		xpathElem = ce
	}
	if xpathElem == nil {
		return "", nil, nil, fmt.Errorf("%w: XPath transform missing XPath element", ErrUnsupportedTransform)
	}
	expr := strings.TrimSpace(domutil.TextContent(xpathElem))
	if expr == "" {
		return "", nil, nil, fmt.Errorf("%w: XPath transform has empty expression", ErrUnsupportedTransform)
	}
	nsBindings := make(map[string]string)
	for prefix, ns := range domutil.InScopeNamespaces(xpathElem, true) {
		if prefix == "" {
			continue
		}
		nsBindings[prefix] = ns.URI()
	}
	return expr, nsBindings, xpathElem, nil
}

// rejectSignatureMethodParameters fails closed on any child element of
// SignatureMethod. The only standard child is ds:HMACOutputLength, which requests
// a truncated HMAC; helium always computes and compares the full-length MAC, so a
// truncation request is unsupported. Silently ignoring such a parameter would
// verify against bytes that differ from what the signer intended, so any
// SignatureMethod parameter is rejected.
func rejectSignatureMethodParameters(elem *helium.Element) error {
	for c := elem.FirstChild(); c != nil; c = c.NextSibling() {
		ce, ok := helium.AsNode[*helium.Element](c)
		if !ok {
			continue
		}
		return fmt.Errorf("%w: unsupported SignatureMethod parameter %s", ErrUnsupportedAlgorithm, domutil.LocalName(ce))
	}
	return nil
}
