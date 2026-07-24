package xmldsig1

import (
	"context"
	"fmt"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
)

type preparedManifestReference struct {
	result ManifestReference
	ref    parsedReference
}

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
//
// All direct inner references are parsed and statically prepared before any is
// executed. A preparation failure records an advisory error for the failing
// reference, marks its peers unexecuted with an error wrapping that failure,
// and prevents resolver or transformer callbacks for the whole Manifest.
func validateManifestReferences(ctx context.Context, budget *verifyBudget, cfg *verifierConfig, doc *helium.Document, sigElem, manifestElem *helium.Element) []ManifestReference {
	var prepared []preparedManifestReference
	preparationFailed := false
	var preparationErr error
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
		item := prepareManifestReference(ctx, budget, cfg, doc, elem)
		if item.result.Err != nil {
			preparationFailed = true
			if preparationErr == nil {
				preparationErr = item.result.Err
			}
		}
		prepared = append(prepared, item)
	}

	results := make([]ManifestReference, 0, len(prepared))
	if preparationFailed {
		for _, item := range prepared {
			if item.result.Err == nil {
				item.result.Err = fmt.Errorf("xmldsig1: Manifest reference was not executed because another reference failed static validation: %w", preparationErr)
			}
			results = append(results, item.result)
		}
		return results
	}

	for _, item := range prepared {
		if err := ctx.Err(); err != nil {
			break
		}
		results = append(results, validateManifestReference(ctx, cfg, doc, sigElem, item))
	}
	return results
}

// prepareManifestReference parses one inner Reference and compiles every
// XPath-bearing part of its transform and URI state without invoking resolver
// or transformer callbacks.
func prepareManifestReference(ctx context.Context, budget *verifyBudget, cfg *verifierConfig, doc *helium.Document, refElem *helium.Element) preparedManifestReference {
	uri, _ := refElem.GetAttribute("URI")
	item := preparedManifestReference{result: ManifestReference{URI: uri}}

	ref, err := parseReferenceElement(ctx, budget, refElem)
	if err != nil {
		item.result.Err = err
		return item
	}
	item.result.URI = ref.uri
	item.result.DigestAlgorithm = ref.digestAlgorithm

	ref.prepared, err = prepareReferenceForVerification(cfg, doc, ref)
	if err != nil {
		item.result.Err = err
		return item
	}
	item.ref = ref
	return item
}

// validateManifestReference resolves one inner ds:Reference of a Manifest,
// applies its transform pipeline, recomputes its digest, and compares it to the
// declared DigestValue. It never returns an error: every failure mode is folded
// into the returned ManifestReference's Err (with Valid left false) so the
// caller can surface it advisorily.
func validateManifestReference(ctx context.Context, cfg *verifierConfig, doc *helium.Document, sigElem *helium.Element, item preparedManifestReference) ManifestReference {
	result := item.result
	ref := item.ref
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
