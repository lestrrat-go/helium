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

	sigElem, signedInfo, sigValueElem, err := buildSignatureSkeleton(doc, cfg)
	if err != nil {
		return err
	}

	// Insert the Signature element into the parent.
	if err := parent.AddChild(sigElem); err != nil {
		return err
	}

	// Process references: compute digests and add Reference elements.
	for _, ref := range cfg.references {
		if err := processReference(ctx, doc, sigElem, signedInfo, ref); err != nil {
			// Detach the signature on failure.
			helium.UnlinkNode(sigElem)
			return err
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

	sigElem, signedInfo, sigValueElem, err := buildSignatureSkeleton(doc, cfg)
	if err != nil {
		return nil, err
	}

	// Create Object element to wrap the content.
	objElem := doc.CreateElement("Object")
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

	for _, ref := range cfg.references {
		if err := processReference(ctx, doc, sigElem, signedInfo, ref); err != nil {
			return nil, err
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
		// Insert KeyInfo before Object.
		if err := sigElem.AddChild(keyInfoElem); err != nil {
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

	sigElem, signedInfo, sigValueElem, err := buildSignatureSkeleton(doc, cfg)
	if err != nil {
		return nil, err
	}

	for _, ref := range cfg.references {
		if err := processReference(ctx, doc, sigElem, signedInfo, ref); err != nil {
			return nil, err
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
	sigElem := doc.CreateElement("Signature")
	if err := sigElem.DeclareNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, nil, nil, err
	}
	if err := sigElem.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, nil, nil, err
	}
	if cfg.signatureID != "" {
		if err := sigElem.SetLiteralAttribute("Id", cfg.signatureID); err != nil {
			return nil, nil, nil, err
		}
	}

	signedInfo := doc.CreateElement("SignedInfo")
	if err := signedInfo.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, nil, nil, err
	}
	if err := sigElem.AddChild(signedInfo); err != nil {
		return nil, nil, nil, err
	}

	c14nMethod := doc.CreateElement("CanonicalizationMethod")
	if err := c14nMethod.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, nil, nil, err
	}
	if err := c14nMethod.SetLiteralAttribute("Algorithm", cfg.c14nMethod); err != nil {
		return nil, nil, nil, err
	}
	if err := signedInfo.AddChild(c14nMethod); err != nil {
		return nil, nil, nil, err
	}

	sigMethod := doc.CreateElement("SignatureMethod")
	if err := sigMethod.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return nil, nil, nil, err
	}
	if err := sigMethod.SetLiteralAttribute("Algorithm", cfg.signatureAlgorithm); err != nil {
		return nil, nil, nil, err
	}
	if err := signedInfo.AddChild(sigMethod); err != nil {
		return nil, nil, nil, err
	}

	sigValue := doc.CreateElement("SignatureValue")
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
func processReference(_ context.Context, doc *helium.Document, sigElem, signedInfo *helium.Element, ref ReferenceConfig) error {
	// Resolve the reference target.
	target, err := resolveReference(doc, ref.URI)
	if err != nil {
		return err
	}

	// Apply transforms to get canonical bytes.
	hasEnveloped := false
	for _, t := range ref.Transforms {
		if _, ok := t.(envelopedTransform); ok {
			hasEnveloped = true
			break
		}
	}

	// For enveloped signature, temporarily detach the Signature element.
	if hasEnveloped {
		helium.UnlinkNode(sigElem)
	}

	var canonical []byte
	// Find the last c14n transform to determine the canonicalization method.
	c14nMethod := ExcC14N10 // default
	var prefixes []string
	for _, t := range ref.Transforms {
		switch tt := t.(type) {
		case c14nTransform:
			c14nMethod = tt.method
		case excC14NTransform:
			c14nMethod = ExcC14N10
			prefixes = tt.prefixes
		}
	}

	if ref.URI == "" {
		// Whole document.
		canonical, err = canonicalize(c14nMethod, doc, prefixes)
	} else {
		canonical, err = canonicalizeSubtree(c14nMethod, target, prefixes)
	}

	// Reattach the Signature element.
	if hasEnveloped {
		parent := target
		if err2 := parent.AddChild(sigElem); err2 != nil {
			return err2
		}
	}

	if err != nil {
		return err
	}

	// Compute digest.
	digest, err := computeDigest(ref.DigestAlgorithm, canonical)
	if err != nil {
		return err
	}

	// Build the Reference element.
	refElem := doc.CreateElement("Reference")
	if err := refElem.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return err
	}
	if ref.URI != "" {
		if err := refElem.SetLiteralAttribute("URI", ref.URI); err != nil {
			return err
		}
	} else {
		if err := refElem.SetLiteralAttribute("URI", ""); err != nil {
			return err
		}
	}
	if ref.ID != "" {
		if err := refElem.SetLiteralAttribute("Id", ref.ID); err != nil {
			return err
		}
	}
	if ref.Type != "" {
		if err := refElem.SetLiteralAttribute("Type", ref.Type); err != nil {
			return err
		}
	}

	// Transforms element.
	if len(ref.Transforms) > 0 {
		transformsElem := doc.CreateElement("Transforms")
		if err := transformsElem.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
			return err
		}
		for _, t := range ref.Transforms {
			tElem := doc.CreateElement("Transform")
			if err := tElem.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
				return err
			}
			if err := tElem.SetLiteralAttribute("Algorithm", t.URI()); err != nil {
				return err
			}
			// For Exclusive C14N with prefixes, add InclusiveNamespaces child.
			if exc, ok := t.(excC14NTransform); ok && len(exc.prefixes) > 0 {
				incNS := doc.CreateElement("InclusiveNamespaces")
				if err := incNS.DeclareNamespace("ec", "http://www.w3.org/2001/10/xml-exc-c14n#"); err != nil {
					return err
				}
				if err := incNS.SetActiveNamespace("ec", "http://www.w3.org/2001/10/xml-exc-c14n#"); err != nil {
					return err
				}
				prefixList := strings.Join(exc.prefixes, " ")
				if err := incNS.SetLiteralAttribute("PrefixList", prefixList); err != nil {
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
	digestMethod := doc.CreateElement("DigestMethod")
	if err := digestMethod.SetActiveNamespace(nsPrefix, NamespaceDSig); err != nil {
		return err
	}
	if err := digestMethod.SetLiteralAttribute("Algorithm", ref.DigestAlgorithm); err != nil {
		return err
	}
	if err := refElem.AddChild(digestMethod); err != nil {
		return err
	}

	// DigestValue.
	digestValue := doc.CreateElement("DigestValue")
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
	// If the Signature element is not in the document tree (detached mode),
	// temporarily attach it so canonicalization can walk the tree.
	needsAttach := sigElem.Parent() == nil
	if needsAttach {
		if err := doc.DocumentElement().AddChild(sigElem); err != nil {
			return err
		}
	}

	canonical, err := canonicalizeSubtree(cfg.c14nMethod, signedInfo, nil)

	if needsAttach {
		helium.UnlinkNode(sigElem)
	}

	if err != nil {
		return err
	}

	sigBytes, err := signBytes(cfg.signatureAlgorithm, key, canonical)
	if err != nil {
		return err
	}

	encoded := base64.StdEncoding.EncodeToString(sigBytes)
	return sigValueElem.AddChild(doc.CreateText([]byte(encoded)))
}
