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
func resolveRetrievalMethods(ctx context.Context, cfg *verifierConfig, doc *helium.Document, keyInfoElem, sigElem *helium.Element, data *KeyInfoData) error {
	visited := make(map[string]struct{})
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
		if err := processRetrievalMethod(ctx, cfg, doc, sigElem, elem, data, visited, 0); err != nil {
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
func processRetrievalMethod(ctx context.Context, cfg *verifierConfig, doc *helium.Document, sigElem, rm *helium.Element, data *KeyInfoData, visited map[string]struct{}, depth int) error {
	if depth > maxRetrievalMethodDepth {
		return fmt.Errorf("%w: chain exceeds %d links", ErrRetrievalMethodLoop, maxRetrievalMethodDepth)
	}
	uri, _ := rm.GetAttribute("URI")
	typ, _ := rm.GetAttribute("Type")

	if _, seen := visited[uri]; seen {
		return fmt.Errorf("%w: %q revisited", ErrRetrievalMethodLoop, uri)
	}
	visited[uri] = struct{}{}

	// An external URI (not one of the four same-document forms) is dereferenced
	// only through a configured ReferenceResolver, fail-closed otherwise.
	if _, _, _, ok := referenceURIForm(uri); !ok {
		octets, err := resolveRetrievalOctets(ctx, cfg, doc, uri)
		if err != nil {
			return err
		}
		return interpretRetrievalOctets(ctx, cfg, octets, typ, data)
	}

	// Same-document target, resolved through the XSW-hardened resolver (a
	// duplicate-id match is rejected as ErrAmbiguousReference).
	target, err := resolveReference(doc, uri, sigElem)
	if err != nil {
		return err
	}
	// A RetrievalMethod may point at another RetrievalMethod; follow the chain
	// under the depth/visited guard rather than misinterpreting it as key material.
	if isDSigCoreNS(target) && domutil.LocalName(target) == "RetrievalMethod" {
		return processRetrievalMethod(ctx, cfg, doc, sigElem, target, data, visited, depth+1)
	}
	return interpretRetrievalElement(target, typ, data)
}

// resolveRetrievalOctets dereferences an external RetrievalMethod URI through the
// configured ReferenceResolver, joining it against the document's base URI first.
// Without a resolver it stays fail-closed with ErrReferenceNotFound, identical to
// the external-Reference posture, and it inherits the resolver's size cap.
func resolveRetrievalOctets(ctx context.Context, cfg *verifierConfig, doc *helium.Document, uri string) ([]byte, error) {
	if cfg.referenceResolver == nil {
		return nil, fmt.Errorf("%w: unsupported RetrievalMethod URI: %s", ErrReferenceNotFound, uri)
	}
	joined, err := joinReferenceURI(doc.URL(), uri)
	if err != nil {
		return nil, err
	}
	return cfg.referenceResolver.ResolveReference(ctx, joined)
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
