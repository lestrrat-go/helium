package xmldsig1

import (
	"context"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
)

// validateManifestReferences digests each inner ds:Reference child of a
// ds:Manifest element and returns one ManifestReference per child, in document
// order. Only the direct core-namespace ds:Reference children are walked (one
// level): a Manifest Reference nested inside this Manifest is digested through
// the same path but not recursively expanded, which bounds the work.
//
// Every inner reference reuses the same fail-closed reference pipeline the
// top-level references use (canonicalizeReference + computeDigest): an
// unsupported transform, an unresolved external reference, or a digest mismatch
// is recorded as that inner reference's Err/Valid=false, never a panic and
// never a silent pass. Because the results are advisory (XMLDSig core §5.1), an
// inner failure is captured rather than propagated.
func validateManifestReferences(ctx context.Context, cfg *verifierConfig, doc *helium.Document, sigElem, manifestElem *helium.Element) []ManifestReference {
	var results []ManifestReference
	for child := manifestElem.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		// Only core XML-Signature ds:Reference children count. A
		// foreign-namespace <evil:Reference> look-alike must not be walked, and
		// the 1.1 xmldsig11# namespace does not spell the core Reference element.
		if !isDSigCoreNS(elem) || domutil.LocalName(elem) != "Reference" {
			continue
		}
		// Bound the walk on cancellation: a Manifest can carry arbitrarily many
		// inner references and each digest resolves and canonicalizes a node-set.
		// Stop with the results gathered so far rather than run to completion.
		if err := ctx.Err(); err != nil {
			break
		}
		results = append(results, validateManifestReference(ctx, cfg, doc, sigElem, elem))
	}
	return results
}

// validateManifestReference resolves one inner ds:Reference of a Manifest,
// applies its transform pipeline, recomputes its digest, and compares it to the
// declared DigestValue. It never returns an error: every failure mode is folded
// into the returned ManifestReference's Err (with Valid left false) so the
// caller can surface it advisorily.
func validateManifestReference(ctx context.Context, cfg *verifierConfig, doc *helium.Document, sigElem, refElem *helium.Element) ManifestReference {
	uri, _ := refElem.GetAttribute("URI")
	result := ManifestReference{URI: uri}

	ref, err := parseReferenceElement(refElem)
	if err != nil {
		result.Err = err
		return result
	}
	result.URI = ref.uri
	result.DigestAlgorithm = ref.digestAlgorithm

	target, canonical, _, err := canonicalizeReference(ctx, cfg, doc, sigElem, ref)
	if err != nil {
		result.Err = err
		return result
	}
	result.Element = target

	computed, err := computeDigest(ref.digestAlgorithm, canonical, cfg.allowSHA1)
	if err != nil {
		result.Err = err
		return result
	}
	if !digestEqual(computed, ref.digestValue) {
		result.Err = ErrDigestMismatch
		return result
	}

	result.Valid = true
	return result
}
