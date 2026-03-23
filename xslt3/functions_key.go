package xslt3

import (
	"context"
	"os"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
)

func (ec *execContext) fnKey(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) < 2 {
		return nil, dynamicError(errCodeXTDE1170, "key() requires at least 2 arguments")
	}

	nameAV, err := xpath3.AtomizeItem(args[0].Get(0))
	if err != nil {
		return nil, err
	}
	name, err := xpath3.AtomicToString(nameAV)
	if err != nil {
		return nil, err
	}

	// Resolve prefixed key names to expanded names using stylesheet namespaces
	name = resolveQName(name, ec.stylesheet.namespaces)

	if args[1] == nil || sequence.Len(args[1]) == 0 {
		return xpath3.EmptySequence(), nil
	}

	// Determine the root for key lookup.
	// Default: use the document containing the context node (per XSLT spec §16.3).
	// When the 3rd argument is provided, use that node directly as the search root.
	// This allows scoping to a subtree (e.g., key('k', 'v', $tree/sub)).
	var root helium.Node = ec.sourceDoc
	if len(args) >= 3 && args[2] != nil && sequence.Len(args[2]) > 0 {
		ni, ok := args[2].Get(0).(xpath3.NodeItem)
		if ok {
			// XTDE1270: 3rd arg must have a document node as root
			r := ni.Node
			for r.Parent() != nil {
				r = r.Parent()
			}
			if r.Type() != helium.DocumentNode {
				return nil, dynamicError(errCodeXTDE1270,
					"key() third argument: root of tree is not a document node")
			}
			root = ni.Node
		}
	} else {
		// 2-arg form: context node is required
		ctxNode := xpath3.FnContextNode(ctx)
		if ctxNode == nil {
			ctxNode = ec.contextNode
		}
		if ctxNode == nil {
			// XTDE1270: no context node
			return nil, dynamicError(errCodeXTDE1270,
				"key() called with no context node")
		}
		// XTDE1270: root of context node's tree must be a document node
		r := ctxNode
		for r.Parent() != nil {
			r = r.Parent()
		}
		if r.Type() != helium.DocumentNode {
			return nil, dynamicError(errCodeXTDE1270,
				"key() context node's root is not a document node")
		}
		root = documentRoot(ctxNode)
	}

	// Check if this is a composite key
	expandedName := resolveQName(name, ec.stylesheet.namespaces)
	_ = expandedName // name is already resolved above
	isComposite := false
	if defs, ok := ec.stylesheet.keys[name]; ok && len(defs) > 0 {
		isComposite = defs[0].Composite
	}

	if isComposite {
		// Composite key: the entire second argument sequence is a single
		// composite lookup key (matched element-by-element).
		avs := make([]xpath3.AtomicValue, 0, sequence.Len(args[1]))
		for item := range sequence.Items(args[1]) {
			av, err := xpath3.AtomizeItem(item)
			if err != nil {
				return nil, err
			}
			avs = append(avs, av)
		}
		nodes, err := ec.lookupCompositeKey(name, avs, root)
		if err != nil {
			return nil, err
		}
		var seq xpath3.ItemSlice
		for _, n := range nodes {
			seq = append(seq, xpath3.NodeItem{Node: n})
		}
		return seq, nil
	}

	// Non-composite: look up each value individually and union the results
	// (XSLT 2.0+ §16.3.2).
	seen := make(map[helium.Node]struct{})
	var seq xpath3.ItemSlice
	for item := range sequence.Items(args[1]) {
		valAV, err := xpath3.AtomizeItem(item)
		if err != nil {
			return nil, err
		}
		nodes, err := ec.lookupKey(name, valAV, root)
		if err != nil {
			return nil, err
		}
		for _, n := range nodes {
			if _, dup := seen[n]; !dup {
				seen[n] = struct{}{}
				seq = append(seq, xpath3.NodeItem{Node: n})
			}
		}
	}
	return seq, nil
}

// documentRoot walks up to the document root of the given node.
func documentRoot(n helium.Node) helium.Node {
	for n.Parent() != nil {
		n = n.Parent()
	}
	return n
}

// generate-id(node?) returns a unique string identifier for a node.
// With no arguments, returns the ID of the context node.
// With an empty sequence argument, returns the empty string.
func (ec *execContext) fnGenerateID(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	var node helium.Node
	if len(args) == 0 {
		// No argument: use the XPath context node (correct inside
		// predicates) rather than the XSLT context node.
		if xpathNode := xpath3.FnContextNode(ctx); xpathNode != nil {
			node = xpathNode
		} else {
			node = ec.contextNode
		}
	} else if args[0] == nil || sequence.Len(args[0]) == 0 {
		// Argument is an empty sequence: return empty string per spec.
		return xpath3.SingleString(""), nil
	} else {
		ni, ok := args[0].Get(0).(xpath3.NodeItem)
		if !ok {
			return nil, dynamicError(errCodeXPTY0004, "generate-id() argument must be a node")
		}
		node = ni.Node
	}
	if node == nil {
		return xpath3.SingleString(""), nil
	}
	return xpath3.SingleString(xpath3.StableNodeID(node)), nil
}

// system-property(name) returns XSLT system properties.
func (ec *execContext) fnSystemProperty(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || (args[0] == nil || sequence.Len(args[0]) == 0) {
		return xpath3.SingleString(""), nil
	}

	av, err := xpath3.AtomizeItem(args[0].Get(0))
	if err != nil {
		return nil, err
	}
	name, err := xpath3.AtomicToString(av)
	if err != nil {
		return nil, err
	}

	// XTDE1390: validate that the argument is a valid QName or EQName.
	if !strings.HasPrefix(name, "Q{") && !isValidQName(name) {
		return nil, dynamicError(errCodeXTDE1390,
			"system-property argument %q is not a valid QName", name)
	}

	// Resolve QName: only prefixed names in the XSLT namespace return
	// XSLT properties. Unprefixed names are in no namespace and return "".
	local := name
	ns := ""
	if strings.HasPrefix(name, "Q{") {
		// URIQualifiedName: Q{uri}local
		end := strings.IndexByte(name, '}')
		if end > 2 {
			ns = name[2:end]
			local = name[end+1:]
		}
	} else if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		local = name[idx+1:]
		// Resolve prefix via stylesheet-level namespaces.
		resolved := false
		if uri, ok := ec.stylesheet.namespaces[prefix]; ok {
			ns = uri
			resolved = true
		}
		// XTDE1390: prefix has no namespace declaration
		if !resolved {
			return nil, dynamicError(errCodeXTDE1390,
				"system-property prefix %q has no namespace declaration", prefix)
		}
	}

	if ns != lexicon.NamespaceXSLT {
		return xpath3.SingleString(""), nil
	}

	switch local {
	case "version":
		return xpath3.SingleString("3.0"), nil
	case "vendor":
		return xpath3.SingleString("helium"), nil
	case "vendor-url":
		return xpath3.SingleString("https://github.com/lestrrat-go/helium"), nil
	case "product-name":
		return xpath3.SingleString("helium-xslt3"), nil
	case "product-version":
		return xpath3.SingleString("0.1"), nil
	case "is-schema-aware":
		if len(ec.stylesheet.schemas) > 0 {
			return xpath3.SingleString(lexicon.ValueYes), nil
		}
		return xpath3.SingleString(lexicon.ValueNo), nil
	case "supports-serialization":
		return xpath3.SingleString(lexicon.ValueYes), nil
	case "supports-backwards-compatibility":
		return xpath3.SingleString(lexicon.ValueNo), nil
	case "supports-namespace-axis":
		return xpath3.SingleString(lexicon.ValueYes), nil
	case "supports-streaming":
		return xpath3.SingleString(lexicon.ValueYes), nil
	case "supports-dynamic-evaluation":
		return xpath3.SingleString(lexicon.ValueYes), nil
	case "supports-higher-order-functions":
		return xpath3.SingleString(lexicon.ValueYes), nil
	case "xpath-version":
		return xpath3.SingleString("3.1"), nil
	case "xsd-version":
		return xpath3.SingleString("1.1"), nil
	default:
		// Return empty string for unknown properties
		return xpath3.SingleString(""), nil
	}
}

// lookupUnparsedEntityInDoc looks up an unparsed entity by name in the given document.
func lookupUnparsedEntityInDoc(name string, doc *helium.Document) *helium.Entity {
	if doc == nil {
		return nil
	}
	if dtd := doc.IntSubset(); dtd != nil {
		if ent, ok := dtd.LookupEntity(name); ok {
			return ent
		}
	}
	if dtd := doc.ExtSubset(); dtd != nil {
		if ent, ok := dtd.LookupEntity(name); ok {
			return ent
		}
	}
	return nil
}

// documentNodeFromNode walks up to the root of a node's tree and returns
// the document node if the root is a document, or nil otherwise.
func documentNodeFromNode(n helium.Node) *helium.Document {
	for n.Parent() != nil {
		n = n.Parent()
	}
	if doc, ok := n.(*helium.Document); ok {
		return doc
	}
	return nil
}

// available-system-properties() returns a sequence of QNames for all
// available system properties in the XSLT namespace.
// fnStreamAvailable implements fn:stream-available — returns true when the
// URI identifies a resource that can be streamed.
func (ec *execContext) fnStreamAvailable(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || (args[0] == nil || sequence.Len(args[0]) == 0) {
		return xpath3.SingleBoolean(false), nil
	}
	av, err := xpath3.AtomizeItem(args[0].Get(0))
	if err != nil {
		return xpath3.SingleBoolean(false), nil
	}
	uri, err := xpath3.AtomicToString(av)
	if err != nil || uri == "" {
		return xpath3.SingleBoolean(false), nil
	}
	resolved := ec.resolveDocumentURI(uri, ec.baseDir())
	info, statErr := os.Stat(resolved)
	if statErr != nil || info.IsDir() {
		return xpath3.SingleBoolean(false), nil
	}
	// Only XML files can be streamed. Quick check: try parsing the first few bytes.
	f, err := os.Open(resolved)
	if err != nil {
		return xpath3.SingleBoolean(false), nil
	}
	defer func() { _ = f.Close() }()
	buf := make([]byte, 256)
	n, _ := f.Read(buf)
	content := strings.TrimSpace(string(buf[:n]))
	isXML := strings.HasPrefix(content, "<?xml") || strings.HasPrefix(content, "<")
	return xpath3.SingleBoolean(isXML), nil
}

// fnCurrentOutputURI implements current-output-uri() — returns the URI
// of the current output destination.
// Returns empty sequence in temporary output state or when no base output URI is set.
func (ec *execContext) fnCurrentOutputURI(_ context.Context, _ []xpath3.Sequence) (xpath3.Sequence, error) {
	if ec.inPatternMatch || ec.inSortKeyEval || ec.temporaryOutputDepth > 0 || ec.currentOutputURI == "" {
		return xpath3.EmptySequence(), nil
	}
	return xpath3.ItemSlice{xpath3.AtomicValue{TypeName: xpath3.TypeAnyURI, Value: ec.currentOutputURI}}, nil
}

func (ec *execContext) fnAvailableSystemProperties(_ context.Context, _ []xpath3.Sequence) (xpath3.Sequence, error) {
	props := []string{
		"version", "vendor", "vendor-url", "product-name", "product-version",
		"is-schema-aware", "supports-serialization",
		"supports-backwards-compatibility", "supports-namespace-axis",
		"supports-streaming", "supports-dynamic-evaluation",
		"supports-higher-order-functions", "xpath-version", "xsd-version",
	}
	var seq xpath3.ItemSlice
	for _, p := range props {
		seq = append(seq, xpath3.AtomicValue{
			TypeName: xpath3.TypeQName,
			Value:    xpath3.QNameValue{Prefix: "xsl", URI: lexicon.NamespaceXSLT, Local: p},
		})
	}
	return seq, nil
}

// unparsed-entity-uri(name [, doc]) returns the URI of an unparsed entity.
func (ec *execContext) fnUnparsedEntityURI(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	// Determine the document node: 2-arg form uses the second arg, 1-arg form uses context node.
	var doc *helium.Document
	if len(args) >= 2 && args[1] != nil && sequence.Len(args[1]) > 0 {
		if ni, ok := args[1].Get(0).(xpath3.NodeItem); ok {
			doc = documentNodeFromNode(ni.Node)
		}
	}
	if doc == nil {
		// 1-arg form: use XPath context node, then XSLT context node
		ctxNode := xpath3.FnContextNode(ctx)
		if ctxNode == nil {
			ctxNode = ec.contextNode
		}
		if ctxNode == nil {
			return nil, dynamicError(errCodeXTDE1370,
				"unparsed-entity-uri called when there is no context node")
		}
		root := ctxNode
		for root.Parent() != nil {
			root = root.Parent()
		}
		if root.Type() != helium.DocumentNode {
			return nil, dynamicError(errCodeXTDE1370,
				"unparsed-entity-uri called when root of context tree is not a document node")
		}
		doc, _ = root.(*helium.Document)
	}

	if args[0] == nil || sequence.Len(args[0]) == 0 {
		return xpath3.SingleString(""), nil
	}
	av, err := xpath3.AtomizeItem(args[0].Get(0))
	if err != nil {
		return xpath3.SingleString(""), nil
	}
	name, err := xpath3.AtomicToString(av)
	if err != nil {
		return xpath3.SingleString(""), nil
	}
	ent := lookupUnparsedEntityInDoc(name, doc)
	if ent == nil {
		return xpath3.SingleString(""), nil
	}
	// If the entity has a pre-resolved URI, use it. Otherwise resolve
	// the system ID against the source document's base URL.
	uri := ent.URI()
	if uri == ent.SystemID() && doc != nil {
		if base := doc.URL(); base != "" {
			if resolved := helium.BuildURI(uri, base); resolved != "" {
				uri = resolved
			}
		}
	}
	return xpath3.ItemSlice{xpath3.AtomicValue{TypeName: xpath3.TypeAnyURI, Value: uri}}, nil
}

// unparsed-entity-public-id(name) returns the public identifier of an unparsed entity.
func (ec *execContext) fnUnparsedEntityPublicID(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	// Determine the document node: 2-arg form uses the second arg, 1-arg form uses context node.
	var doc *helium.Document
	if len(args) >= 2 && args[1] != nil && sequence.Len(args[1]) > 0 {
		if ni, ok := args[1].Get(0).(xpath3.NodeItem); ok {
			doc = documentNodeFromNode(ni.Node)
		}
	}
	if doc == nil {
		// 1-arg form: use XPath context node, then XSLT context node
		ctxNode := xpath3.FnContextNode(ctx)
		if ctxNode == nil {
			ctxNode = ec.contextNode
		}
		if ctxNode == nil {
			return nil, dynamicError(errCodeXTDE1380,
				"unparsed-entity-public-id called when there is no context node")
		}
		root := ctxNode
		for root.Parent() != nil {
			root = root.Parent()
		}
		if root.Type() != helium.DocumentNode {
			return nil, dynamicError(errCodeXTDE1380,
				"unparsed-entity-public-id called when root of context tree is not a document node")
		}
		doc, _ = root.(*helium.Document)
	}

	if args[0] == nil || sequence.Len(args[0]) == 0 {
		return xpath3.SingleString(""), nil
	}
	av, err := xpath3.AtomizeItem(args[0].Get(0))
	if err != nil {
		return xpath3.SingleString(""), nil
	}
	name, err := xpath3.AtomicToString(av)
	if err != nil {
		return xpath3.SingleString(""), nil
	}
	ent := lookupUnparsedEntityInDoc(name, doc)
	if ent == nil {
		return xpath3.SingleString(""), nil
	}
	return xpath3.SingleString(ent.ExternalID()), nil
}

// xslMultiArityFunc wraps multiple xsl:function overloads (same QName,
// different arity) as a single xpath3.Function that dispatches by arg count.
