package xmldsig1

import (
	"context"
	"fmt"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
)

// XSLTTransformer applies the XMLDSig XSLT transform (XMLDSig core §6.6.5):
// stylesheet is the serialized xsl:stylesheet/xsl:transform subtree taken from the
// ds:Transform element, and input is the pre-XSLT octet stream (the reference's
// canonicalized node-set). The result is the octet stream the DigestValue is
// computed over.
//
// It is the opt-in seam for the XSLT transform, mirroring [ReferenceResolver]:
// the XSLT transform is OFF by default and runs only when a transformer is
// configured via [Verifier.XSLTTransformer]; without one an XSLT transform fails
// closed with [ErrUnsupportedTransform]. XSLT is verify-only — signing rejects it
// fail-closed and never invokes a transformer.
//
// SECURITY: both stylesheet and input are attacker-controlled on verify (an
// attacker who controls the signature controls the ds:Transform/xsl:stylesheet
// subtree, and input derives from the signed document). XSLT is a powerful
// language — document(), unbounded recursion, and other unbounded computation —
// so the implementer owns ALL resource and XXE policy (compute/time/memory
// limits, disabling document()/external access). helium ships no transformer for
// exactly this reason, matching the "no HTTP resolver shipped" stance for
// external references.
//
// TransformXSLT must be safe to call from multiple goroutines and should honor
// ctx cancellation.
type XSLTTransformer interface {
	TransformXSLT(ctx context.Context, stylesheet []byte, input []byte) ([]byte, error)
}

// isXSLTNS reports whether e is in the XSLT namespace
// (http://www.w3.org/1999/XSL/Transform). The XSLT transform's stylesheet root
// lives only here; a foreign-namespace look-alike must not be accepted as the
// stylesheet, so the exact namespace is required.
func isXSLTNS(e *helium.Element) bool {
	return elementNamespaceURI(e) == namespaceXSLT
}

// parseXSLTTransform extracts the stylesheet subtree from a ds:Transform element
// whose Algorithm is TransformXSLT. XMLDSig core §6.6.5 gives the transform a
// single stylesheet root child, an xsl:stylesheet (or the synonym xsl:transform)
// element in the XSLT namespace. It fails closed on a missing, duplicate, or
// foreign child so a malformed XSLT transform never digests the wrong bytes.
//
// The stylesheet subtree is serialized to octets with inclusive Canonical XML 1.0
// (canonicalizeSubtree), which walks the in-scope namespace axis so the emitted
// bytes carry the xsl: (and any other in-scope) namespace declarations — a
// lossless, self-contained round-trip an [XSLTTransformer] can re-parse and
// compile as a standalone stylesheet.
func parseXSLTTransform(te *helium.Element) ([]byte, error) {
	var styleElem *helium.Element
	for c := te.FirstChild(); c != nil; c = c.NextSibling() {
		ce, ok := helium.AsNode[*helium.Element](c)
		if !ok {
			continue
		}
		local := domutil.LocalName(ce)
		if !isXSLTNS(ce) || (local != "stylesheet" && local != "transform") {
			return nil, fmt.Errorf("%w: unsupported XSLT transform child %s", ErrUnsupportedTransform, local)
		}
		if styleElem != nil {
			return nil, fmt.Errorf("%w: multiple stylesheet elements in XSLT transform", ErrUnsupportedTransform)
		}
		styleElem = ce
	}
	if styleElem == nil {
		return nil, fmt.Errorf("%w: XSLT transform missing xsl:stylesheet element", ErrUnsupportedTransform)
	}
	octets, err := canonicalizeSubtree(C14N10, styleElem, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot serialize XSLT stylesheet: %v", ErrUnsupportedTransform, err)
	}
	return octets, nil
}
