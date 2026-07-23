package xmldsig1

import (
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium/internal/xmlbase64"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath1"
)

type transformValueKind uint8

const (
	transformValueNodeSet transformValueKind = iota + 1
	transformValueOctets
)

// transformValue is the current value flowing through an ordered Reference
// transform list. Constructors keep the node-set and octet states distinct even
// when either payload is empty.
type transformValue struct {
	kind   transformValueKind
	nodes  *nodeSetValue
	octets []byte
}

// nodeSetValue carries both explicit node membership and the document that owns
// those nodes. The initial same-document selection stays lazy until a transform
// needs explicit membership, preserving the existing whole-document, subtree,
// enveloped, and detached canonicalization paths byte-for-byte.
type nodeSetValue struct {
	doc          *helium.Document
	nodes        []helium.Node
	materialized bool

	// referenceSelection remains true after the initial set is materialized. It
	// distinguishes that set from a document parsed after an octet boundary, where
	// an enveloped transform can no longer identify the containing Signature.
	referenceSelection bool
	includeComments    bool
	origin             *referenceNodeSetOrigin
}

type referenceNodeSetOrigin struct {
	doc              *helium.Document
	target           *helium.Element
	wholeDoc         bool
	includeComments  bool
	sigElem          *helium.Element
	internalRoot     *helium.Element
	envelopedPending bool
}

type transformContract struct {
	input  transformValueKind
	output transformValueKind
	xpath  *xpath1.Expression
}

// transformRuntime contains the immutable policy and dependencies for one
// pipeline execution. external marks resolver-supplied starting octets so their
// first required XML parse keeps the established ErrReferenceNotFound class.
type transformRuntime struct {
	parser          helium.Parser
	xsltTransformer XSLTTransformer
	signature       *helium.Element

	allowEnveloped bool
	signing        bool
	external       bool
}

func newNodeSetTransformValue(nodes *nodeSetValue) transformValue {
	return transformValue{kind: transformValueNodeSet, nodes: nodes}
}

func newOctetTransformValue(octets []byte) transformValue {
	return transformValue{kind: transformValueOctets, octets: octets}
}

func newReferenceNodeSetValue(doc *helium.Document, target, sigElem *helium.Element, wholeDoc, includeComments bool, internalRoot *helium.Element) transformValue {
	origin := &referenceNodeSetOrigin{
		doc:             doc,
		target:          target,
		wholeDoc:        wholeDoc,
		includeComments: includeComments,
		sigElem:         sigElem,
		internalRoot:    internalRoot,
	}
	return newNodeSetTransformValue(&nodeSetValue{
		doc:                doc,
		referenceSelection: true,
		includeComments:    includeComments,
		origin:             origin,
	})
}

// transformSteps converts a ReferenceConfig's typed Transform list into the
// algorithm-agnostic steps consumed by the ordered transform executor, so signing
// preflight and the per-reference digest path interpret a transform list
// identically.
func transformSteps(ref ReferenceConfig) []transformStep {
	steps := make([]transformStep, len(ref.Transforms))
	for i, t := range ref.Transforms {
		step := transformStep{algorithm: t.URI()}
		if exc, ok := t.(excC14NTransform); ok {
			step.prefixes = exc.prefixes
		}
		steps[i] = step
	}
	return steps
}

// preflightSignerTransforms validates every Reference's complete transform list
// BEFORE any DOM mutation or node moves. Execution uses the same validator, so
// preflight cannot accept a transform that signing later ignores.
func preflightSignerTransforms(cfg *signerConfig) error {
	for i, ref := range cfg.references {
		_, _, _, sameDocument := referenceURIForm(ref.URI)
		kind := transformValueOctets
		if sameDocument {
			kind = transformValueNodeSet
		}
		runtime := transformRuntime{
			parser:         cfg.parser(),
			allowEnveloped: sameDocument,
			signing:        true,
			external:       !sameDocument,
		}
		_, err := validateTransformSteps(runtime, kind, transformSteps(ref))
		if err != nil {
			// Carry the failing reference's index and URI so a caller signing
			// over a multi-reference configuration can pinpoint the offending
			// Reference, symmetric with the per-reference digest loop. The
			// underlying sentinel (e.g. ErrUnsupportedTransform) stays matchable
			// via errors.Is through ReferenceError.Unwrap.
			return &ReferenceError{Op: opSign, Reference: i, URI: ref.URI, Err: err}
		}
	}
	return nil
}

func contractForTransform(step transformStep) (transformContract, error) {
	switch step.algorithm {
	case C14N10, C14N10Comments, ExcC14N10, ExcC14N10Comments, C14N11URI, C14N11Comments:
		return transformContract{input: transformValueNodeSet, output: transformValueOctets}, nil
	case TransformBase64:
		return transformContract{input: transformValueOctets, output: transformValueOctets}, nil
	case TransformXPath:
		if strings.TrimSpace(step.xpathExpr) == "" {
			return transformContract{}, fmt.Errorf("%w: XPath transform has empty expression", ErrUnsupportedTransform)
		}
		return transformContract{input: transformValueNodeSet, output: transformValueNodeSet}, nil
	case TransformEnvelopedSignature:
		return transformContract{input: transformValueNodeSet, output: transformValueNodeSet}, nil
	case TransformXSLT:
		if len(step.stylesheet) == 0 {
			return transformContract{}, fmt.Errorf("%w: XSLT transform missing xsl:stylesheet element", ErrUnsupportedTransform)
		}
		return transformContract{input: transformValueOctets, output: transformValueOctets}, nil
	default:
		return transformContract{}, fmt.Errorf("%w: %s", ErrUnsupportedTransform, step.algorithm)
	}
}

// validateTransformSteps checks the complete list before execution. It validates
// algorithms, parameters, runtime capabilities, and enveloped-transform identity
// requirements without rejecting valid changes between node sets and octets.
func validateTransformSteps(runtime transformRuntime, initialKind transformValueKind, steps []transformStep) ([]transformContract, error) {
	contracts := make([]transformContract, len(steps))
	kind := initialKind
	hasSignatureIdentity := runtime.allowEnveloped && kind == transformValueNodeSet
	bearingNodeAvailable := kind == transformValueNodeSet
	for i, step := range steps {
		if runtime.signing {
			switch step.algorithm {
			case TransformXPath:
				return nil, fmt.Errorf("transform %d (%s): %w: XPath filter transform is not supported for signing", i, step.algorithm, ErrUnsupportedTransform)
			case TransformXSLT:
				return nil, fmt.Errorf("transform %d (%s): %w: XSLT transform is not supported for signing", i, step.algorithm, ErrUnsupportedTransform)
			}
		}
		contract, err := contractForTransform(step)
		if err != nil {
			return nil, fmt.Errorf("transform %d (%s): %w", i, step.algorithm, err)
		}
		if step.algorithm == TransformXPath {
			eval := newDSigXPathEvaluator(step.xpathNS, step.xpathHere, defaultXPathOpLimit)
			contract.xpath, err = compileXPathFilterExpression(step.xpathExpr, eval)
			if err != nil {
				return nil, fmt.Errorf("transform %d (%s): %w", i, step.algorithm, err)
			}
			if expressionReferencesHere(step.xpathExpr) && (!bearingNodeAvailable || step.xpathHere == nil) {
				return nil, fmt.Errorf("transform %d (%s): %w: here() is unavailable for octet input or after an octet boundary", i, step.algorithm, ErrHereUnavailable)
			}
		}
		if step.algorithm == TransformXSLT && isNilInterface(runtime.xsltTransformer) {
			return nil, fmt.Errorf("transform %d (%s): %w: XSLT transform requires a configured XSLTTransformer", i, step.algorithm, ErrUnsupportedTransform)
		}
		if step.algorithm == TransformEnvelopedSignature {
			if !runtime.allowEnveloped {
				return nil, fmt.Errorf("transform %d (%s): %w: enveloped-signature transform is not valid for this input", i, step.algorithm, ErrUnsupportedTransform)
			}
			if !hasSignatureIdentity {
				return nil, fmt.Errorf("transform %d (%s): %w: enveloped-signature transform cannot follow an octet boundary", i, step.algorithm, ErrUnsupportedTransform)
			}
		}

		// Base64 has an algorithm-specific node-set input rule. It consumes a
		// node-set directly when present, rather than triggering generic C14N.
		if step.algorithm == TransformBase64 && kind == transformValueNodeSet {
			kind = transformValueOctets
			hasSignatureIdentity = false
			bearingNodeAvailable = false
		} else {
			if kind != contract.input {
				hasSignatureIdentity = false
				bearingNodeAvailable = false
			}
			kind = contract.output
			if kind == transformValueOctets {
				hasSignatureIdentity = false
				bearingNodeAvailable = false
			}
		}
		contracts[i] = contract
	}
	return contracts, nil
}

// expressionReferencesHere reports whether an XPath expression invokes the
// unqualified here() function outside a string literal. The expression has
// already passed XPath parsing and static validation before this check.
func expressionReferencesHere(expr string) bool {
	const name = "here"
	var quote byte
	for i := 0; i < len(expr); i++ {
		c := expr[i]
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			continue
		}
		if !strings.HasPrefix(expr[i:], name) {
			continue
		}
		if i > 0 && isXPathNameByte(expr[i-1]) {
			continue
		}
		j := i + len(name)
		for j < len(expr) && (expr[j] == ' ' || expr[j] == '\t' || expr[j] == '\r' || expr[j] == '\n') {
			j++
		}
		if j < len(expr) && expr[j] == '(' {
			return true
		}
	}
	return false
}

// executeTransformPipeline applies every transform in document order, inserting
// only the node-set/octet conversion required by the next step. A final node-set
// is converted with inclusive Canonical XML 1.0.
func executeTransformPipeline(ctx context.Context, runtime transformRuntime, initial transformValue, steps []transformStep) ([]byte, error) {
	contracts, err := validateTransformSteps(runtime, initial.kind, steps)
	if err != nil {
		return nil, err
	}
	value := initial
	producerIndex := -1
	producerAlgorithm := ""
	for i, step := range steps {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// XMLDSig §6.6.2 gives Base64 a special node-set conversion: concatenate
		// text-node values, not canonical markup.
		if step.algorithm == TransformBase64 && value.kind == transformValueNodeSet {
			octets, err := base64TransformNodeSetOctets(value.nodes)
			if err != nil {
				return nil, fmt.Errorf("transform %d (%s): %w", i, step.algorithm, err)
			}
			value = newOctetTransformValue(octets)
			producerIndex = i
			producerAlgorithm = step.algorithm
			continue
		}

		contract := contracts[i]
		if value.kind != contract.input {
			value, err = convertTransformValue(ctx, runtime, value, contract.input, producerIndex, producerAlgorithm, i, step.algorithm)
			if err != nil {
				return nil, err
			}
		}

		switch step.algorithm {
		case C14N10, C14N10Comments, ExcC14N10, ExcC14N10Comments, C14N11URI, C14N11Comments:
			octets, err := canonicalizeNodeSetValue(step.algorithm, value.nodes, step.prefixes)
			if err != nil {
				return nil, fmt.Errorf("transform %d (%s): %w", i, step.algorithm, err)
			}
			value = newOctetTransformValue(octets)
		case TransformBase64:
			octets, err := decodeBase64TransformOctets(value.octets)
			if err != nil {
				return nil, fmt.Errorf("transform %d (%s): %w", i, step.algorithm, err)
			}
			value = newOctetTransformValue(octets)
		case TransformXPath:
			nodes, err := materializeNodeSet(value.nodes)
			if err != nil {
				return nil, fmt.Errorf("transform %d (%s): %w", i, step.algorithm, err)
			}
			hereNode := step.xpathHere
			if hereNode != nil && hereNode.OwnerDocument() != nodes.doc {
				hereNode = nil
			}
			filtered, err := applyXPathFilter(ctx, nodes.nodes, xpathFilter{expr: contract.xpath, ns: step.xpathNS, hereNode: hereNode})
			if err != nil {
				return nil, fmt.Errorf("transform %d (%s): %w", i, step.algorithm, err)
			}
			nodes.nodes = filtered
			value = newNodeSetTransformValue(nodes)
		case TransformEnvelopedSignature:
			if err := applyEnvelopedTransform(value.nodes, runtime.signature); err != nil {
				return nil, fmt.Errorf("transform %d (%s): %w", i, step.algorithm, err)
			}
		case TransformXSLT:
			octets, err := runtime.xsltTransformer.TransformXSLT(ctx, step.stylesheet, value.octets)
			if err != nil {
				return nil, err
			}
			value = newOctetTransformValue(octets)
		}

		if value.kind == transformValueOctets {
			producerIndex = i
			producerAlgorithm = step.algorithm
		}
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if value.kind == transformValueNodeSet {
		return canonicalizeNodeSetValue(C14N10, value.nodes, nil)
	}
	return value.octets, nil
}

func convertTransformValue(ctx context.Context, runtime transformRuntime, value transformValue, required transformValueKind, producerIndex int, producerAlgorithm string, consumerIndex int, consumerAlgorithm string) (transformValue, error) {
	switch {
	case value.kind == transformValueOctets && required == transformValueNodeSet:
		doc, err := runtime.parser.Parse(ctx, value.octets)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return transformValue{}, ctxErr
			}
			if runtime.external && producerIndex < 0 {
				return transformValue{}, fmt.Errorf("%w: cannot parse external reference as XML for transform %d (%s): %v", ErrReferenceNotFound, consumerIndex, consumerAlgorithm, err)
			}
			if producerIndex >= 0 {
				return transformValue{}, fmt.Errorf("%w: transform %d (%s) output cannot be parsed for transform %d (%s): %v", ErrUnsupportedTransform, producerIndex, producerAlgorithm, consumerIndex, consumerAlgorithm, err)
			}
			return transformValue{}, fmt.Errorf("%w: octet input cannot be parsed for transform %d (%s): %v", ErrUnsupportedTransform, consumerIndex, consumerAlgorithm, err)
		}
		nodes := &nodeSetValue{doc: doc, nodes: collectDocumentNodes(doc), materialized: true}
		return newNodeSetTransformValue(nodes), nil
	case value.kind == transformValueNodeSet && required == transformValueOctets:
		octets, err := canonicalizeNodeSetValue(C14N10, value.nodes, nil)
		if err != nil {
			return transformValue{}, err
		}
		return newOctetTransformValue(octets), nil
	default:
		return transformValue{}, fmt.Errorf("%w: cannot convert transform value kind %d to %d", ErrUnsupportedTransform, value.kind, required)
	}
}

func materializeNodeSet(value *nodeSetValue) (*nodeSetValue, error) {
	if value == nil || value.doc == nil {
		return nil, fmt.Errorf("%w: transform node-set has no owning document", ErrUnsupportedTransform)
	}
	if value.materialized {
		return value, nil
	}
	if value.origin == nil || value.origin.target == nil {
		return nil, fmt.Errorf("%w: transform node-set has no reference origin", ErrUnsupportedTransform)
	}
	if value.origin.wholeDoc {
		value.nodes = collectDocumentNodes(value.origin.doc)
	} else {
		value.nodes = collectSubtreeNodes(value.origin.target)
	}
	if !value.origin.includeComments {
		value.nodes = removeCommentNodes(value.nodes)
	}
	if value.origin.envelopedPending {
		value.nodes = removeSignatureNodes(value.nodes, value.origin.sigElem)
	}
	value.origin = nil
	value.materialized = true
	return value, nil
}

func removeCommentNodes(nodes []helium.Node) []helium.Node {
	kept := make([]helium.Node, 0, len(nodes))
	for _, node := range nodes {
		if node.Type() != helium.CommentNode {
			kept = append(kept, node)
		}
	}
	return kept
}

func applyEnvelopedTransform(value *nodeSetValue, sigElem *helium.Element) error {
	if value == nil || !value.referenceSelection {
		return fmt.Errorf("%w: enveloped-signature transform has no original Signature identity", ErrUnsupportedTransform)
	}
	if value.origin != nil && !value.materialized {
		value.origin.envelopedPending = true
		return nil
	}
	value.nodes = removeSignatureNodes(value.nodes, sigElem)
	return nil
}

func canonicalizeNodeSetValue(method string, value *nodeSetValue, prefixes []string) ([]byte, error) {
	if value == nil || value.doc == nil {
		return nil, fmt.Errorf("%w: transform node-set has no owning document", ErrUnsupportedTransform)
	}
	if value.referenceSelection {
		method = effectiveC14NMethod(method, value.includeComments)
	}
	if value.origin != nil && !value.materialized {
		origin := value.origin
		switch {
		case origin.envelopedPending:
			return canonicalizeEnveloped(method, origin.doc, origin.target, origin.sigElem, origin.wholeDoc, prefixes)
		case origin.wholeDoc:
			return canonicalize(method, origin.doc, prefixes)
		case origin.internalRoot != nil && isDescendantOrSelf(origin.target, origin.internalRoot):
			return canonicalizeDetachedSubtree(method, origin.internalRoot, origin.target, prefixes)
		default:
			return canonicalizeSubtree(method, origin.target, prefixes)
		}
	}
	if !value.materialized {
		return nil, fmt.Errorf("%w: transform node-set is not materialized", ErrUnsupportedTransform)
	}
	return canonicalizeNodeSet(method, value.nodes, value.doc, prefixes)
}

func base64TransformNodeSetOctets(value *nodeSetValue) ([]byte, error) {
	if value == nil {
		return nil, fmt.Errorf("%w: transform node-set is nil", ErrUnsupportedTransform)
	}
	materialized, err := materializeNodeSet(value)
	if err != nil {
		return nil, err
	}
	var encoded strings.Builder
	for _, node := range materialized.nodes {
		switch node.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			_, _ = encoded.Write(node.Content())
		}
	}
	return decodeBase64TransformOctets([]byte(encoded.String()))
}

func decodeBase64TransformOctets(encoded []byte) ([]byte, error) {
	decoded, err := xmlbase64.DecodeString(string(encoded))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base64 transform input: %v", ErrInvalidSignature, err)
	}
	return decoded, nil
}
