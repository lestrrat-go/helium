package xmldsig1

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

func signEnveloped(ctx context.Context, cfg *signerConfig, doc *helium.Document, parent *helium.Element, key any) error {
	if len(cfg.references) == 0 {
		return ErrNoReferences
	}
	if cfg.signatureAlgorithm == "" {
		return fmt.Errorf("%w: signature algorithm not set", ErrInvalidSignature)
	}

	// Reject weak (SHA-1) algorithms before building or mutating any nodes, so a
	// rejected default-SHA-1 request leaves the input tree untouched.
	if err := preflightSignerWeakAlgorithms(cfg); err != nil {
		return err
	}

	// Reject invalid transform pipelines before building or mutating any nodes,
	// so a rejected pipeline leaves the input tree untouched.
	if err := preflightSignerTransforms(cfg); err != nil {
		return err
	}

	sigElem, signedInfo, sigValueElem, err := buildSignatureSkeleton(doc, cfg)
	if err != nil {
		return err
	}

	// Insert the Signature element into the parent.
	if err := parent.AddChild(sigElem); err != nil {
		return err
	}

	// Process references: compute digests and add Reference elements.
	for i, ref := range cfg.references {
		if err := processReference(ctx, doc, sigElem, signedInfo, ref, cfg.allowSHA1, nil); err != nil {
			// Detach the signature on failure.
			helium.UnlinkNode(sigElem)
			return &ReferenceError{Op: "sign", Reference: i, URI: ref.URI, Err: err}
		}
	}

	// Canonicalize SignedInfo and sign.
	if err := computeAndSetSignatureValue(cfg, sigElem, signedInfo, sigValueElem, doc, key); err != nil {
		helium.UnlinkNode(sigElem)
		return err
	}

	// Build KeyInfo if configured.
	if cfg.keyInfoBuilder != nil {
		keyInfoElem, err := cfg.keyInfoBuilder.BuildKeyInfo(ctx, doc, key)
		if err != nil {
			helium.UnlinkNode(sigElem)
			return err
		}
		if err := sigElem.AddChild(keyInfoElem); err != nil {
			helium.UnlinkNode(sigElem)
			return err
		}
	}

	return nil
}

func signEnveloping(ctx context.Context, cfg *signerConfig, doc *helium.Document, content []helium.Node, key any) (*helium.Element, error) {
	if len(cfg.references) == 0 {
		return nil, ErrNoReferences
	}
	if cfg.signatureAlgorithm == "" {
		return nil, fmt.Errorf("%w: signature algorithm not set", ErrInvalidSignature)
	}

	// Reject weak (SHA-1) algorithms before building nodes or moving caller
	// content into the <Object>, so a rejected default-SHA-1 request leaves the
	// input nodes unmoved and the input tree untouched.
	if err := preflightSignerWeakAlgorithms(cfg); err != nil {
		return nil, err
	}

	// Reject invalid transform pipelines before building nodes or moving caller
	// content into the <Object>, so a rejected pipeline leaves the input nodes
	// unmoved and the input tree untouched.
	if err := preflightSignerTransforms(cfg); err != nil {
		return nil, err
	}

	sigElem, signedInfo, sigValueElem, err := buildSignatureSkeleton(doc, cfg)
	if err != nil {
		return nil, err
	}

	// Narrow preflight for the built-in empty X509DataKeyInfo. An x509DataKeyInfo
	// with zero certificates always fails with ErrInvalidKeyInfo; detecting that
	// here — before the <Object> is created or any caller content is moved into
	// it — leaves the caller's input nodes unmoved and the input tree untouched.
	// Arbitrary caller-provided builders keep the established timing (their
	// BuildKeyInfo runs after the content is wrapped, in the block below).
	if b, ok := cfg.keyInfoBuilder.(*x509DataKeyInfo); ok && len(b.certs) == 0 {
		return nil, fmt.Errorf("%w: X509DataKeyInfo requires at least one certificate", ErrInvalidKeyInfo)
	}

	// Create Object element to wrap the content.
	objElem, err := doc.CreateElement("Object")
	if err != nil {
		return nil, err
	}
	if err := objElem.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, err
	}
	for _, n := range content {
		mn, ok := n.(helium.MutableNode)
		if !ok {
			continue
		}
		if err := objElem.AddChild(mn); err != nil {
			return nil, err
		}
	}
	if err := sigElem.AddChild(objElem); err != nil {
		return nil, err
	}

	// Process references. A same-document reference (URI="#id") may point INTO
	// the Signature's own <Object> content — e.g. a <Manifest> or
	// <SignatureProperties> carrying an Id — so the Signature itself is searched
	// as an extra resolution root and a target found there is canonicalized while
	// the Signature stays detached. The Signature is never inserted into the
	// caller's document, so a reference to a document element (URI="#root") sees
	// an unchanged subtree and produces byte-identical output.
	for i, ref := range cfg.references {
		if err := processReference(ctx, doc, sigElem, signedInfo, ref, cfg.allowSHA1, sigElem); err != nil {
			return nil, &ReferenceError{Op: "sign", Reference: i, URI: ref.URI, Err: err}
		}
	}

	if err := computeAndSetSignatureValue(cfg, sigElem, signedInfo, sigValueElem, doc, key); err != nil {
		return nil, err
	}

	if cfg.keyInfoBuilder != nil {
		keyInfoElem, err := cfg.keyInfoBuilder.BuildKeyInfo(ctx, doc, key)
		if err != nil {
			return nil, err
		}
		// The XML-DSig schema content model is (SignedInfo, SignatureValue,
		// KeyInfo?, Object*), so KeyInfo must precede the Object. Append KeyInfo
		// (landing it after SignatureValue, before Object), then re-append the
		// Object so it moves to the end after KeyInfo. AddChild auto-unlinks the
		// already-linked Object before relinking it, so this is a move, not a
		// duplicate.
		if err := sigElem.AddChild(keyInfoElem); err != nil {
			return nil, err
		}
		if err := sigElem.AddChild(objElem); err != nil {
			return nil, err
		}
	}

	return sigElem, nil
}

func signDetached(ctx context.Context, cfg *signerConfig, doc *helium.Document, key any) (*helium.Element, error) {
	if len(cfg.references) == 0 {
		return nil, ErrNoReferences
	}
	if cfg.signatureAlgorithm == "" {
		return nil, fmt.Errorf("%w: signature algorithm not set", ErrInvalidSignature)
	}

	// Reject weak (SHA-1) algorithms before building or mutating any nodes.
	if err := preflightSignerWeakAlgorithms(cfg); err != nil {
		return nil, err
	}

	// Reject invalid transform pipelines before building or mutating any nodes.
	if err := preflightSignerTransforms(cfg); err != nil {
		return nil, err
	}

	sigElem, signedInfo, sigValueElem, err := buildSignatureSkeleton(doc, cfg)
	if err != nil {
		return nil, err
	}

	for i, ref := range cfg.references {
		if err := processReference(ctx, doc, sigElem, signedInfo, ref, cfg.allowSHA1, nil); err != nil {
			return nil, &ReferenceError{Op: "sign", Reference: i, URI: ref.URI, Err: err}
		}
	}

	if err := computeAndSetSignatureValue(cfg, sigElem, signedInfo, sigValueElem, doc, key); err != nil {
		return nil, err
	}

	if cfg.keyInfoBuilder != nil {
		keyInfoElem, err := cfg.keyInfoBuilder.BuildKeyInfo(ctx, doc, key)
		if err != nil {
			return nil, err
		}
		if err := sigElem.AddChild(keyInfoElem); err != nil {
			return nil, err
		}
	}

	return sigElem, nil
}

// buildSignatureSkeleton creates the Signature element with SignedInfo and
// SignatureValue children, but no References yet.
func buildSignatureSkeleton(doc *helium.Document, cfg *signerConfig) (*helium.Element, *helium.Element, *helium.Element, error) {
	sigElem, err := doc.CreateElement("Signature")
	if err != nil {
		return nil, nil, nil, err
	}
	if err := sigElem.DeclareNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, nil, nil, err
	}
	if err := sigElem.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, nil, nil, err
	}
	if cfg.signatureID != "" {
		if err := sigElem.SetAttribute("Id", cfg.signatureID); err != nil {
			return nil, nil, nil, err
		}
	}

	signedInfo, err := doc.CreateElement("SignedInfo")
	if err != nil {
		return nil, nil, nil, err
	}
	if err := signedInfo.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, nil, nil, err
	}
	if err := sigElem.AddChild(signedInfo); err != nil {
		return nil, nil, nil, err
	}

	c14nMethod, err := doc.CreateElement("CanonicalizationMethod")
	if err != nil {
		return nil, nil, nil, err
	}
	if err := c14nMethod.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, nil, nil, err
	}
	if err := c14nMethod.SetAttribute("Algorithm", cfg.c14nMethod); err != nil {
		return nil, nil, nil, err
	}
	if err := signedInfo.AddChild(c14nMethod); err != nil {
		return nil, nil, nil, err
	}

	sigMethod, err := doc.CreateElement("SignatureMethod")
	if err != nil {
		return nil, nil, nil, err
	}
	if err := sigMethod.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, nil, nil, err
	}
	if err := sigMethod.SetAttribute("Algorithm", cfg.signatureAlgorithm); err != nil {
		return nil, nil, nil, err
	}
	if err := signedInfo.AddChild(sigMethod); err != nil {
		return nil, nil, nil, err
	}

	sigValue, err := doc.CreateElement("SignatureValue")
	if err != nil {
		return nil, nil, nil, err
	}
	if err := sigValue.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, nil, nil, err
	}
	if err := sigElem.AddChild(sigValue); err != nil {
		return nil, nil, nil, err
	}

	return sigElem, signedInfo, sigValue, nil
}

// processReference computes the digest for a single Reference and adds the
// Reference element to SignedInfo.
// internalRoot, when non-nil, is the enveloping Signature whose own (detached)
// <Object> content may hold the reference target; it is searched in addition to
// the document and a target found inside it is canonicalized while detached.
func processReference(_ context.Context, doc *helium.Document, sigElem, signedInfo *helium.Element, ref ReferenceConfig, allowSHA1 bool, internalRoot *helium.Element) error {
	// Resolve the reference target. For an enveloping signature the target may
	// live inside the Signature's own detached Object content, so search it too.
	var target *helium.Element
	var err error
	if internalRoot != nil {
		target, err = resolveReference(doc, ref.URI, internalRoot)
	} else {
		target, err = resolveReference(doc, ref.URI)
	}
	if err != nil {
		return err
	}

	// Interpret the configured transforms as an ordered pipeline so the digest
	// is computed exactly as a verifier reading these same Transform elements
	// would. The output begins as a node-set; an octet-producing c14n transform
	// ends the pipeline, and when no c14n transform is configured the default
	// node-set->octet conversion is inclusive Canonical XML 1.0.
	c14nMethod, prefixes, hasEnveloped, err := resolveTransformPipeline(transformSteps(ref))
	if err != nil {
		return err
	}

	// For enveloped signatures the Signature element and its descendants must
	// be omitted from the canonical input. canonicalizeEnveloped does this on a
	// deep copy of the document, never mutating the caller's live DOM (which
	// would race with concurrent readers and risk leaving the tree corrupted if
	// a restore failed).
	var canonical []byte
	switch {
	case hasEnveloped:
		canonical, err = canonicalizeEnveloped(c14nMethod, doc, target, sigElem, ref.URI == "", prefixes)
	case ref.URI == "":
		canonical, err = canonicalize(c14nMethod, doc, prefixes)
	case internalRoot != nil && isDescendantOrSelf(target, internalRoot):
		// The target lives inside the enveloping Signature's own detached
		// <Object> content; canonicalize it without inserting the Signature
		// into the caller's document.
		canonical, err = canonicalizeDetachedSubtree(c14nMethod, internalRoot, target, prefixes)
	default:
		canonical, err = canonicalizeSubtree(c14nMethod, target, prefixes)
	}
	if err != nil {
		return err
	}

	// Compute digest. A SHA-1 digest is rejected unless the caller opted in
	// via Signer.AllowSHA1(true).
	digest, err := computeDigest(ref.DigestAlgorithm, canonical, allowSHA1)
	if err != nil {
		return err
	}

	// Build the Reference element.
	refElem, err := doc.CreateElement("Reference")
	if err != nil {
		return err
	}
	if err := refElem.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return err
	}
	if ref.URI != "" {
		if err := refElem.SetAttribute("URI", ref.URI); err != nil {
			return err
		}
	} else {
		if err := refElem.SetAttribute("URI", ""); err != nil {
			return err
		}
	}
	if ref.ID != "" {
		if err := refElem.SetAttribute("Id", ref.ID); err != nil {
			return err
		}
	}
	if ref.Type != "" {
		if err := refElem.SetAttribute("Type", ref.Type); err != nil {
			return err
		}
	}

	// Transforms element.
	if len(ref.Transforms) > 0 {
		transformsElem, err := doc.CreateElement("Transforms")
		if err != nil {
			return err
		}
		if err := transformsElem.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
			return err
		}
		for _, t := range ref.Transforms {
			tElem, err := doc.CreateElement("Transform")
			if err != nil {
				return err
			}
			if err := tElem.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
				return err
			}
			if err := tElem.SetAttribute("Algorithm", t.URI()); err != nil {
				return err
			}
			// For Exclusive C14N with prefixes, add InclusiveNamespaces child.
			if exc, ok := t.(excC14NTransform); ok && len(exc.prefixes) > 0 {
				incNS, err := doc.CreateElement("InclusiveNamespaces")
				if err != nil {
					return err
				}
				if err := incNS.DeclareNamespace("ec", "http://www.w3.org/2001/10/xml-exc-c14n#"); err != nil {
					return err
				}
				if err := incNS.SetActiveNamespace("ec", "http://www.w3.org/2001/10/xml-exc-c14n#"); err != nil {
					return err
				}
				prefixList := strings.Join(exc.prefixes, " ")
				if err := incNS.SetAttribute("PrefixList", prefixList); err != nil {
					return err
				}
				if err := tElem.AddChild(incNS); err != nil {
					return err
				}
			}
			if err := transformsElem.AddChild(tElem); err != nil {
				return err
			}
		}
		if err := refElem.AddChild(transformsElem); err != nil {
			return err
		}
	}

	// DigestMethod.
	digestMethod, err := doc.CreateElement("DigestMethod")
	if err != nil {
		return err
	}
	if err := digestMethod.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return err
	}
	if err := digestMethod.SetAttribute("Algorithm", ref.DigestAlgorithm); err != nil {
		return err
	}
	if err := refElem.AddChild(digestMethod); err != nil {
		return err
	}

	// DigestValue.
	digestValue, err := doc.CreateElement("DigestValue")
	if err != nil {
		return err
	}
	if err := digestValue.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return err
	}
	encoded := base64.StdEncoding.EncodeToString(digest)
	if err := digestValue.AddChild(doc.CreateText([]byte(encoded))); err != nil {
		return err
	}
	if err := refElem.AddChild(digestValue); err != nil {
		return err
	}

	return signedInfo.AddChild(refElem)
}

// computeAndSetSignatureValue canonicalizes SignedInfo, signs it, and sets
// the SignatureValue element text.
func computeAndSetSignatureValue(cfg *signerConfig, sigElem *helium.Element, signedInfo, sigValueElem *helium.Element, doc *helium.Document, key any) error {
	// Canonicalize SignedInfo. When the Signature is already in the document tree
	// (enveloped mode), canonicalize its SignedInfo subtree in place. When the
	// Signature is detached (enveloping/detached mode, sigElem.Parent()==nil),
	// canonicalize SignedInfo through the throwaway-document proxy: the live
	// Signature root is moved into a private document rooted at a proxy that
	// reproduces the caller document element's full inherited canonicalization
	// context (in-scope namespaces + inherited xml:* per C14N version), so
	// SignedInfo inherits EXACTLY what it would under doc.DocumentElement() while
	// the caller's document is never mutated. The move is undone on every exit —
	// normal return, error, or a panic unwinding out of canonicalization.
	var canonical []byte
	var err error
	if sigElem.Parent() == nil {
		canonical, err = canonicalizeDetachedSubtree(cfg.c14nMethod, sigElem, signedInfo, nil)
	} else {
		canonical, err = canonicalizeSubtree(cfg.c14nMethod, signedInfo, nil)
	}
	if err != nil {
		return err
	}

	sigBytes, err := signBytes(cfg.signatureAlgorithm, key, canonical, cfg.allowSHA1)
	if err != nil {
		return err
	}

	encoded := base64.StdEncoding.EncodeToString(sigBytes)
	return sigValueElem.AddChild(doc.CreateText([]byte(encoded)))
}
