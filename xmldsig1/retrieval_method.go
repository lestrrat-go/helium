package xmldsig1

import (
	"context"
	"crypto/x509"
	"fmt"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/lestrrat-go/helium/internal/xmlbase64"
)

// maxRetrievalMethodDepth caps how many ds:RetrievalMethod links are followed
// when a RetrievalMethod's target is itself a RetrievalMethod. Together with the
// visited-URI set it prevents an unbounded or cyclic chain from being
// dereferenced without limit.
const maxRetrievalMethodDepth = 5

// resolveRetrievalMethods dereferences every ds:RetrievalMethod child of a
// ds:KeyInfo element and merges the retrieved certificate material into data
// before key resolution. It runs as a second, resolution-aware pass after
// parseKeyInfo (which is value-only): parseKeyInfo cannot dereference because it
// lacks the document, config, and Signature context.
//
// A RetrievalMethod inherits the same fail-closed / opt-in-resolver / size-cap /
// base-join posture as an external Reference: a same-document "#id" target is
// resolved through the XSW-hardened resolveReference, while an external URI is
// dereferenced only through a configured ReferenceResolver (none ⇒
// ErrReferenceNotFound) with the resolver's 64 MiB cap and base-URI join.
// Retrieved certificates are appended to data.X509Certificates; parsing a
// certificate never trusts it, so the caller's KeySource and out-of-band trust
// decision still governs an obtained cert exactly as it does an inline one.
func resolveRetrievalMethods(ctx context.Context, cfg *verifierConfig, doc *helium.Document, keyInfoElem *helium.Element, data *KeyInfoData) error {
	for child := keyInfoElem.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		// A RetrievalMethod is a core XML-Signature element; a foreign-namespace
		// look-alike must not steer key retrieval, so require the core namespace.
		if !isDSigCoreNS(elem) || domutil.LocalName(elem) != "RetrievalMethod" {
			continue
		}
		// The visited-URI set breaks a cyclic/unbounded RetrievalMethod chain, so
		// it is scoped to each top-level chain: two independent sibling
		// RetrievalMethods may legitimately target the same URI, and sharing one
		// set across siblings would misreport the second as a loop. It flows only
		// through the recursive processRetrievalMethod calls that follow a chain.
		visited := make(map[string]struct{})
		if err := processRetrievalMethod(ctx, cfg, doc, elem, data, visited, 0); err != nil {
			return err
		}
	}
	return nil
}

// processRetrievalMethod dereferences a single ds:RetrievalMethod element and
// merges the obtained certificate material into data. When the target is itself
// a RetrievalMethod the chain is followed, guarded by a depth cap and a
// visited-URI set so a cyclic or unbounded chain fails closed with
// ErrRetrievalMethodLoop.
func processRetrievalMethod(ctx context.Context, cfg *verifierConfig, doc *helium.Document, rm *helium.Element, data *KeyInfoData, visited map[string]struct{}, depth int) error {
	if depth > maxRetrievalMethodDepth {
		return fmt.Errorf("%w: chain exceeds %d links", ErrRetrievalMethodLoop, maxRetrievalMethodDepth)
	}
	uri, _ := rm.GetAttribute("URI")
	typ, _ := rm.GetAttribute("Type")

	if _, seen := visited[uri]; seen {
		return fmt.Errorf("%w: %q revisited", ErrRetrievalMethodLoop, uri)
	}
	visited[uri] = struct{}{}

	// A RetrievalMethod may carry a ds:Transforms just like a Reference; parse and
	// resolve it as an ordered pipeline so an unsupported or mis-ordered transform
	// fails closed rather than being silently ignored while the resolved material
	// is accepted anyway.
	steps, err := parseRetrievalTransforms(rm)
	if err != nil {
		return err
	}
	pipe, err := resolveTransformPipeline(steps)
	if err != nil {
		return err
	}
	// The enveloped-signature transform removes the Signature's own subtree from a
	// node-set; it is meaningless on a RetrievalMethod (which retrieves key
	// material, not the signed content), so reject it fail-closed.
	if pipe.hasEnveloped {
		return fmt.Errorf("%w: enveloped-signature transform is not valid on a RetrievalMethod", ErrUnsupportedTransform)
	}

	// An external URI (not one of the four same-document forms) is dereferenced
	// only through a configured ReferenceResolver, fail-closed otherwise. The
	// resolved octets run through the same transform pipeline the external
	// Reference digest path uses before Type interpretation.
	if _, _, _, ok := referenceURIForm(uri); !ok {
		octets, err := resolveRetrievalOctets(ctx, cfg, doc, uri)
		if err != nil {
			return err
		}
		transformed, err := externalReferenceDigestInput(ctx, octets, pipe, stepsHaveC14N(steps), cfg.parser())
		if err != nil {
			return err
		}
		return interpretRetrievalOctets(ctx, cfg, transformed, typ, data)
	}

	// Same-document target, resolved against the document through the XSW-hardened
	// resolver (a duplicate-id match is rejected as ErrAmbiguousReference).
	target, err := resolveReference(doc, uri)
	if err != nil {
		return err
	}
	// A transform-free RetrievalMethod may point at another RetrievalMethod;
	// follow the chain under the depth/visited guard rather than misinterpreting
	// it as key material.
	if len(steps) == 0 && isDSigCoreNS(target) && domutil.LocalName(target) == "RetrievalMethod" {
		// The wrapper's own Type is interpreted by neither the recursion nor the
		// target's terminal interpretation, so a present-but-unsupported wrapper
		// Type would be silently accepted here while interpretRetrievalElement/
		// interpretRetrievalOctets reject the same value on every other branch.
		// Reject it fail-closed. Type is advisory (XMLDSig §4.4.3), so an absent
		// wrapper Type stays permissive and the chain is followed.
		if typ != "" && typ != TypeRawX509Certificate && typ != TypeX509Data {
			return fmt.Errorf("%w: unsupported RetrievalMethod Type %q", ErrInvalidKeyInfo, typ)
		}
		return processRetrievalMethod(ctx, cfg, doc, target, data, visited, depth+1)
	}
	// Without transforms the resolved element is interpreted directly, keeping the
	// no-transform behavior byte-identical.
	if len(steps) == 0 {
		return interpretRetrievalElement(target, typ, data)
	}
	// With transforms, the target node-set is run through the pipeline to octets,
	// then interpreted by Type exactly as an externally retrieved octet stream.
	octets, err := retrievalTransformOctets(ctx, target, pipe)
	if err != nil {
		return err
	}
	return interpretRetrievalOctets(ctx, cfg, octets, typ, data)
}

// parseRetrievalTransforms parses the optional single ds:Transforms child of a
// RetrievalMethod into transform steps, enforcing at most one core-namespace
// Transforms element. A foreign-namespace look-alike is ignored so it cannot
// steer processing.
func parseRetrievalTransforms(rm *helium.Element) ([]transformStep, error) {
	var transforms []parsedTransform
	seen := false
	for child := rm.FirstChild(); child != nil; child = child.NextSibling() {
		e, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if !isDSigCoreNS(e) || domutil.LocalName(e) != "Transforms" {
			continue
		}
		if seen {
			return nil, fmt.Errorf("%w: multiple Transforms elements in RetrievalMethod", ErrInvalidKeyInfo)
		}
		seen = true
		list, err := parseTransformList(e)
		if err != nil {
			return nil, err
		}
		transforms = list
	}
	steps := make([]transformStep, len(transforms))
	for i, t := range transforms {
		steps[i] = transformStep(t)
	}
	return steps, nil
}

// retrievalTransformOctets applies a same-document RetrievalMethod's transform
// pipeline to the resolved target element's node-set, producing the octet stream
// that is then interpreted by Type. It mirrors the Reference node-set → octet
// path: a base64 transform decodes the target's string-value, one or more XPath
// filters narrow the subtree node-set before canonicalization, and otherwise the
// subtree is canonicalized with the pipeline's effective c14n method.
func retrievalTransformOctets(ctx context.Context, target *helium.Element, pipe transformPipeline) ([]byte, error) {
	if pipe.base64 {
		if len(pipe.xpathFilters) > 0 {
			return nil, fmt.Errorf("%w: base64 transform combined with a node-set transform", ErrUnsupportedTransform)
		}
		return base64TransformOctets(target)
	}
	if len(pipe.xpathFilters) > 0 {
		nodes := collectSubtreeNodes(target)
		for _, f := range pipe.xpathFilters {
			filtered, err := applyXPathFilter(ctx, nodes, f)
			if err != nil {
				return nil, err
			}
			nodes = filtered
		}
		return canonicalizeNodeSet(pipe.c14nMethod, nodes, target.OwnerDocument(), pipe.prefixes)
	}
	return canonicalizeSubtree(pipe.c14nMethod, target, pipe.prefixes)
}

// resolveRetrievalOctets dereferences an external RetrievalMethod URI through the
// configured ReferenceResolver, joining it against the document's base URI first.
// Without a resolver it stays fail-closed with ErrReferenceNotFound, identical to
// the external-Reference posture, and it inherits the shared 64 MiB resolver cap.
func resolveRetrievalOctets(ctx context.Context, cfg *verifierConfig, doc *helium.Document, uri string) ([]byte, error) {
	if cfg.referenceResolver == nil {
		return nil, fmt.Errorf("%w: unsupported RetrievalMethod URI: %s", ErrReferenceNotFound, uri)
	}
	joined, err := joinReferenceURI(doc.URL(), uri)
	if err != nil {
		return nil, err
	}
	return resolveReferenceOctets(ctx, cfg.referenceResolver, joined)
}

// interpretRetrievalOctets interprets externally retrieved octets by the
// RetrievalMethod Type and appends the resulting certificate material to data.
// An unsupported (or absent) Type fails closed.
func interpretRetrievalOctets(ctx context.Context, cfg *verifierConfig, octets []byte, typ string, data *KeyInfoData) error {
	switch typ {
	case TypeRawX509Certificate:
		cert, err := x509.ParseCertificate(octets)
		if err != nil {
			return fmt.Errorf("%w: invalid rawX509Certificate: %v", ErrInvalidKeyInfo, err)
		}
		data.X509Certificates = append(data.X509Certificates, cert)
		return nil
	case TypeX509Data:
		// Parse the retrieved octets with the locked-down reference parser (XXE
		// blocked, no filesystem, no network by default), then reuse parseX509Data.
		extDoc, err := parseRetrievalDoc(ctx, cfg, octets)
		if err != nil {
			return err
		}
		root := extDoc.DocumentElement()
		if root == nil || !isDSigCoreNS(root) || domutil.LocalName(root) != "X509Data" {
			return fmt.Errorf("%w: RetrievalMethod X509Data root is not a ds:X509Data", ErrInvalidKeyInfo)
		}
		return parseX509Data(root, data)
	default:
		return fmt.Errorf("%w: unsupported RetrievalMethod Type %q", ErrInvalidKeyInfo, typ)
	}
}

// parseRetrievalDoc parses externally retrieved octets into a document with the
// locked-down reference parser.
func parseRetrievalDoc(ctx context.Context, cfg *verifierConfig, octets []byte) (*helium.Document, error) {
	parser := cfg.parser()
	extDoc, err := parser.Parse(ctx, octets)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot parse RetrievalMethod resource as XML: %v", ErrReferenceNotFound, err)
	}
	return extDoc, nil
}

// interpretRetrievalElement interprets a same-document RetrievalMethod target
// element by the RetrievalMethod Type and appends the resulting certificate
// material to data. An unsupported (or absent) Type fails closed.
func interpretRetrievalElement(target *helium.Element, typ string, data *KeyInfoData) error {
	switch typ {
	case TypeX509Data:
		if !isDSigCoreNS(target) || domutil.LocalName(target) != "X509Data" {
			return fmt.Errorf("%w: RetrievalMethod X509Data target is not a ds:X509Data", ErrInvalidKeyInfo)
		}
		return parseX509Data(target, data)
	case TypeRawX509Certificate:
		der, err := xmlbase64.DecodeString(domutil.TextContent(target))
		if err != nil {
			return fmt.Errorf("%w: invalid rawX509Certificate base64: %v", ErrInvalidKeyInfo, err)
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return fmt.Errorf("%w: invalid rawX509Certificate: %v", ErrInvalidKeyInfo, err)
		}
		data.X509Certificates = append(data.X509Certificates, cert)
		return nil
	default:
		return fmt.Errorf("%w: unsupported RetrievalMethod Type %q", ErrInvalidKeyInfo, typ)
	}
}
