package xmldsig1

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
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
		if err := processReference(ctx, cfg, doc, sigElem, signedInfo, ref, nil); err != nil {
			// Detach the signature on failure.
			helium.UnlinkNode(sigElem)
			return &ReferenceError{Op: opSign, Reference: i, URI: ref.URI, Err: err}
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

	// Reject every non-movable or nil content entry BEFORE building nodes or
	// moving any caller content into the <Object>, so a bad entry never leaves a
	// partially-moved tree and is never silently dropped. An ordinary DOM element
	// implements helium.MutableNode; a nil interface, a typed-nil node pointer, or
	// a read-only node (e.g. a namespace-node wrapper) does not and cannot be
	// relinked into the Object.
	for i, n := range content {
		if !contentMovable(n) {
			return nil, fmt.Errorf("%w: content[%d] is nil or not movable (%T)", ErrInvalidSignature, i, n)
		}
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

	// Snapshot every content node's original position (parent + both sibling
	// anchors) BEFORE moving any node. Moving one node changes the live sibling
	// links of the others, so a per-node capture taken at move time would record a
	// stale anchor and restore the nodes out of order; a pre-move snapshot is the
	// only faithful record.
	snapshots := make([]movedContent, len(content))
	for i, n := range content {
		mn, ok := n.(helium.MutableNode)
		if !ok {
			// Unreachable: the preflight above rejected every non-movable entry.
			return nil, fmt.Errorf("%w: content is nil or not movable (%T)", ErrInvalidSignature, n)
		}
		snapshots[i] = movedContent{node: mn, parent: mn.Parent(), prev: mn.PrevSibling(), next: mn.NextSibling(), ownerDoc: mn.OwnerDocument()}
	}

	// Moving each content node into the Object detaches it from the caller's tree.
	// If ANY later step fails — reference processing, signing, or KeyInfo
	// construction — the returned Signature is discarded and the moved nodes would
	// be stranded under it, silently lost to the caller. Restore every moved node
	// to its exact original location on the error/panic path only; on success they
	// stay in the Object (the intended enveloping semantics). This mirrors the
	// restore-on-every-failure-exit discipline of canonicalizeDetachedSubtree.
	var moved []movedContent
	success := false
	defer func() {
		if success {
			return
		}
		restoreMovedContent(moved)
	}()

	// Splice each node onto the end of the Object with a non-coalescing insert:
	// AddChild for the first node (the Object is empty, so nothing to merge with),
	// then Replace(prev, node) to place each subsequent node immediately after the
	// previous one. Plain AddChild would merge two adjacent Text content entries
	// into one node, corrupting the first node's content so a later rollback could
	// not restore the originals. Record a node in moved only once its move
	// succeeds, so the restore never touches a node left in place.
	var prevMoved helium.MutableNode
	for i := range snapshots {
		mn := snapshots[i].node
		if prevMoved == nil {
			if err := objElem.AddChild(mn); err != nil {
				return nil, err
			}
		} else {
			if err := prevMoved.Replace(prevMoved, mn); err != nil {
				return nil, err
			}
		}
		prevMoved = mn
		moved = append(moved, snapshots[i])
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
		if err := processReference(ctx, cfg, doc, sigElem, signedInfo, ref, sigElem); err != nil {
			return nil, &ReferenceError{Op: opSign, Reference: i, URI: ref.URI, Err: err}
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

	success = true
	return sigElem, nil
}

// movedContent records a caller content node's original tree position so
// signEnveloping can restore it if signing fails after the node has been moved
// into the <Object>. Both sibling anchors are captured: the restore inserts
// before the next sibling when it can, and otherwise (former last child, or a
// read-only next sibling) inserts after the previous sibling — a non-coalescing
// splice that needs no movable next sibling. ownerDoc is the node's original
// OwnerDocument, captured before the move: canonicalizeDetachedSubtree rewrites
// the moved subtree's owners to the signing document, so content that came from
// a different helium.Document must have its owner restored after the structural
// rollback puts the node back in place.
type movedContent struct {
	node     helium.MutableNode
	parent   helium.Node
	prev     helium.Node
	next     helium.Node
	ownerDoc *helium.Document
}

// contentMovable reports whether a SignEnveloping content entry can be relinked
// into the <Object>. An ordinary DOM element qualifies; a nil interface, a
// typed-nil node pointer, and a read-only node that does not implement
// helium.MutableNode (e.g. a namespace-node wrapper) do not.
func contentMovable(n helium.Node) bool {
	if n == nil {
		return false
	}
	if _, ok := n.(helium.MutableNode); !ok {
		return false
	}
	v := reflect.ValueOf(n)
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return false
	}
	return true
}

// restoreMovedContent puts every moved caller content node back at its exact
// original tree position, in the original order, leaving the caller's DOM
// byte-identical to before enveloping signing began. It runs only on the
// error/panic path; on success the moved nodes stay in the <Object>.
//
// A node is restored against a sibling anchor that is itself already back at its
// final position. When the anchor a node needs is still a pending (unrestored)
// moved node, the node is deferred until that anchor is in place, so the restore
// order is independent of the order the nodes were moved.
func restoreMovedContent(moved []movedContent) {
	pending := make(map[helium.Node]struct{}, len(moved))
	for _, m := range moved {
		pending[m.node] = struct{}{}
	}
	for len(pending) > 0 {
		remaining := len(pending)
		for _, m := range moved {
			if _, ok := pending[m.node]; !ok {
				continue
			}
			if !restoreOneContent(m, pending) {
				continue
			}
			delete(pending, m.node)
		}
		if len(pending) < remaining {
			continue
		}
		// No anchor became ready this pass. A well-formed tree cannot reach here
		// (sibling links form a strict order with no cycle), but never spin:
		// detach the rest so they end up cleanly unlinked rather than left
		// double-linked inside the discarded Object.
		for _, m := range moved {
			if _, ok := pending[m.node]; !ok {
				continue
			}
			helium.UnlinkNode(m.node)
			delete(pending, m.node)
		}
	}
	// Structural rollback is complete; every node is back at its original
	// position. canonicalizeDetachedSubtree rewrote each moved subtree's owner to
	// the signing document, so restore each node's original OwnerDocument over its
	// whole subtree. For content that already belonged to the signing document
	// this is a no-op; for content from a different helium.Document it hands the
	// subtree back pointing at its original owner.
	for _, m := range moved {
		m.node.SetTreeDoc(m.ownerDoc)
	}
}

// restoreOneContent reinserts one moved content node at its recorded original
// position, reporting false (restore deferred) when the anchor it must splice
// against is a still-pending moved node.
//
// It inserts the node immediately BEFORE its original next sibling when that
// sibling is a movable node, deferring until the sibling is back at its final
// position. This drives a right-to-left restore that always terminates: a node
// whose next sibling is nil, read-only, or a node that never moved is placed
// without waiting on anything, and each placement frees its left neighbor.
//
// When there is no movable next sibling to insert before (the node was the last
// child, or its next sibling is a read-only node), it inserts the node
// immediately AFTER its previous sibling instead. Both are pointer-level,
// non-coalescing splices (Replace keeps the anchor and moves the node next to
// it), so a restored Text node is never merged into an adjacent Text sibling and
// a former last child lands in its exact slot. Replace also auto-unlinks the
// node from the discarded Object first, so each is a move, not a duplicate.
//
// Invariant (falsifiable): in a tree helium parsed, every content sibling
// (Element/Text/Comment/CDATA/PI/EntityRef) implements helium.MutableNode, so
// m.next/m.prev satisfy the MutableNode branches above and content order is
// restored exactly. A helium.NamespaceNodeWrapper is a virtual XPath-only node
// with nil sibling links and is never linked into a content sibling chain by the
// parser; the "read-only next sibling reverses order" path is reachable only if a
// caller manually splices a non-MutableNode between content nodes, which the
// documented content contract does not permit. m.next/m.prev are position anchors
// snapshotted before the move: a caller-supplied KeyInfoBuilder must not relocate
// them mid-sign, and a builder that does has already invalidated its own snapshot,
// so no snapshot-based restore can define a correct original position.
func restoreOneContent(m movedContent, pending map[helium.Node]struct{}) bool {
	if m.parent == nil && m.prev == nil && m.next == nil {
		// The node was fully detached when the caller handed it in — no parent and
		// no siblings — so return it to that detached state.
		helium.UnlinkNode(m.node)
		return true
	}
	if next, ok := m.next.(helium.MutableNode); ok {
		if _, blocked := pending[m.next]; blocked {
			return false
		}
		if err := next.Replace(m.node, m.next); err == nil {
			return true
		}
	}
	// No movable next sibling to insert before. Insert after the previous sibling
	// when it is movable and already back in place — a non-coalescing splice that
	// restores a former last child (and a node before a read-only next sibling)
	// without merging adjacent Text nodes.
	if prev, ok := m.prev.(helium.MutableNode); ok {
		if _, blocked := pending[m.prev]; !blocked {
			if err := prev.Replace(prev, m.node); err == nil {
				return true
			}
		}
	}
	// No usable sibling anchor: the node was the only child, or a fallback when
	// both original siblings are unavailable. Splice after the parent's current
	// last child with a non-coalescing Replace so an appended Text node is never
	// merged into a residual last Text child; only when the parent is empty (no
	// last child to merge with) does a plain AddChild apply.
	if parent, ok := m.parent.(helium.MutableNode); ok {
		if last, ok := parent.LastChild().(helium.MutableNode); ok {
			if err := last.Replace(last, m.node); err == nil {
				return true
			}
		} else if err := parent.AddChild(m.node); err == nil {
			return true
		}
	}
	// Last resort: leave the node detached rather than double-linked.
	helium.UnlinkNode(m.node)
	return true
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
		if err := processReference(ctx, cfg, doc, sigElem, signedInfo, ref, nil); err != nil {
			return nil, &ReferenceError{Op: opSign, Reference: i, URI: ref.URI, Err: err}
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
// signReferenceOctets computes the canonical byte stream a Reference's
// DigestValue is signed over. It handles a same-document reference (resolving and
// canonicalizing the target subtree/document) and, when a ReferenceResolver is
// configured, an external reference (dereferencing its octets and applying the
// transform pipeline through the SAME externalReferenceDigestInput the verifier
// uses, so the signed digest is byte-identical to what verification recomputes).
// Without a resolver an external URI stays fail-closed with ErrReferenceNotFound.
func signReferenceOctets(ctx context.Context, cfg *signerConfig, doc *helium.Document, sigElem *helium.Element, ref ReferenceConfig, internalRoot *helium.Element) ([]byte, error) {
	// An external reference is dereferenced only through a configured resolver.
	if _, _, _, ok := referenceURIForm(ref.URI); !ok {
		if cfg.referenceResolver == nil {
			return nil, fmt.Errorf("%w: unsupported reference URI: %s", ErrReferenceNotFound, ref.URI)
		}
		steps := transformSteps(ref)
		pipe, err := resolveTransformPipeline(steps)
		if err != nil {
			return nil, err
		}
		joined, err := joinReferenceURI(doc.URL(), ref.URI)
		if err != nil {
			return nil, err
		}
		octets, err := cfg.referenceResolver.ResolveReference(ctx, joined)
		if err != nil {
			return nil, err
		}
		return externalReferenceDigestInput(ctx, octets, pipe, stepsHaveC14N(steps), cfg.parser())
	}

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
		return nil, err
	}

	// Interpret the configured transforms as an ordered pipeline so the digest
	// is computed exactly as a verifier reading these same Transform elements
	// would. The output begins as a node-set; an octet-producing c14n transform
	// ends the pipeline, and when no c14n transform is configured the default
	// node-set->octet conversion is inclusive Canonical XML 1.0.
	pipe, err := resolveTransformPipeline(transformSteps(ref))
	if err != nil {
		return nil, err
	}
	c14nMethod := pipe.c14nMethod
	prefixes := pipe.prefixes
	hasEnveloped := pipe.hasEnveloped

	// Classify the URI's node-set form (§4.3.3.2-3) so the digest is computed
	// exactly as a verifier reading these same elements would. wholeDoc selects
	// the document root; includeComments governs comment membership, and a
	// WithComments c14n is downgraded to its plain variant when the form excludes
	// comments so sign and verify stay symmetric on comment handling.
	_, wholeDoc, includeComments, _ := referenceURIForm(ref.URI)
	c14nMethod = effectiveC14NMethod(c14nMethod, includeComments)

	// For enveloped signatures the Signature element and its descendants must
	// be omitted from the canonical input. canonicalizeEnveloped does this on a
	// deep copy of the document, never mutating the caller's live DOM (which
	// would race with concurrent readers and risk leaving the tree corrupted if
	// a restore failed).
	var canonical []byte
	switch {
	case hasEnveloped:
		canonical, err = canonicalizeEnveloped(c14nMethod, doc, target, sigElem, wholeDoc, prefixes)
	case wholeDoc:
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
		return nil, err
	}
	return canonical, nil
}

func processReference(ctx context.Context, cfg *signerConfig, doc *helium.Document, sigElem, signedInfo *helium.Element, ref ReferenceConfig, internalRoot *helium.Element) error {
	canonical, err := signReferenceOctets(ctx, cfg, doc, sigElem, ref, internalRoot)
	if err != nil {
		return err
	}

	// Compute digest. A SHA-1 digest is rejected unless the caller opted in
	// via Signer.AllowSHA1(true).
	digest, err := computeDigest(ref.DigestAlgorithm, canonical, cfg.allowSHA1)
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
