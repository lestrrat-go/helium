package xslt3

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// xsltFunctions returns the XSLT-specific functions that need to be
// registered with the XPath evaluator by local name (no namespace prefix).
func (ec *execContext) xsltFunctions() map[string]xpath3.Function {
	return map[string]xpath3.Function{
		"current":             &xsltFunc{min: 0, max: 0, fn: ec.fnCurrent},
		"document":            &xsltFunc{min: 1, max: 2, fn: ec.fnDocument},
		"key":                 &xsltFunc{min: 2, max: 3, fn: ec.fnKey},
		"generate-id":         &xsltFunc{min: 0, max: 1, fn: ec.fnGenerateID},
		"system-property":     &xsltFunc{min: 1, max: 1, fn: ec.fnSystemProperty},
		"unparsed-entity-uri": &xsltFunc{min: 1, max: 1, fn: ec.fnUnparsedEntityURI},
	}
}

type xsltFunc struct {
	min int
	max int
	fn  func(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error)
}

func (f *xsltFunc) MinArity() int { return f.min }
func (f *xsltFunc) MaxArity() int { return f.max }
func (f *xsltFunc) Call(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	return f.fn(ctx, args)
}

// current() returns the current node (the node being matched/processed).
func (ec *execContext) fnCurrent(_ context.Context, _ []xpath3.Sequence) (xpath3.Sequence, error) {
	if ec.currentNode == nil {
		return xpath3.EmptySequence(), nil
	}
	return xpath3.SingleNode(ec.currentNode), nil
}

// document(uri, base?) loads an external XML document.
func (ec *execContext) fnDocument(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return xpath3.EmptySequence(), nil
	}

	av, err := xpath3.AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	uri, err := xpath3.AtomicToString(av)
	if err != nil {
		return nil, err
	}

	// Empty string means the stylesheet document itself
	if uri == "" {
		// Return the stylesheet source document
		return xpath3.SingleNode(ec.sourceDoc), nil
	}

	// Check cache
	if doc, ok := ec.docCache[uri]; ok {
		return xpath3.SingleNode(doc), nil
	}

	// Load and parse
	data, err := os.ReadFile(uri)
	if err != nil {
		return nil, dynamicError("FODC0002", "cannot load document %q: %v", uri, err)
	}

	doc, err := helium.Parse(ctx, data)
	if err != nil {
		return nil, dynamicError("FODC0002", "cannot parse document %q: %v", uri, err)
	}

	if ec.docCache == nil {
		ec.docCache = make(map[string]*helium.Document)
	}
	ec.docCache[uri] = doc
	return xpath3.SingleNode(doc), nil
}

// key(name, value, doc?) looks up nodes by key.
func (ec *execContext) fnKey(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) < 2 {
		return nil, dynamicError(errCodeXTDE1170, "key() requires at least 2 arguments")
	}

	nameAV, err := xpath3.AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	name, err := xpath3.AtomicToString(nameAV)
	if err != nil {
		return nil, err
	}

	valAV, err := xpath3.AtomizeItem(args[1][0])
	if err != nil {
		return nil, err
	}
	value, err := xpath3.AtomicToString(valAV)
	if err != nil {
		return nil, err
	}

	nodes, err := ec.lookupKey(name, value)
	if err != nil {
		return nil, err
	}

	seq := make(xpath3.Sequence, len(nodes))
	for i, n := range nodes {
		seq[i] = xpath3.NodeItem{Node: n}
	}
	return seq, nil
}

// generate-id(node?) returns a unique string identifier for a node.
func (ec *execContext) fnGenerateID(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	var node helium.Node
	if len(args) == 0 || len(args[0]) == 0 {
		node = ec.contextNode
	} else {
		ni, ok := args[0][0].(xpath3.NodeItem)
		if !ok {
			return nil, dynamicError("XPTY0004", "generate-id() argument must be a node")
		}
		node = ni.Node
	}
	if node == nil {
		return xpath3.SingleString(""), nil
	}
	// Use fmt.Sprintf %p to get a unique pointer-based ID
	id := fmt.Sprintf("id%p", node)
	return xpath3.SingleString(id), nil
}

// system-property(name) returns XSLT system properties.
func (ec *execContext) fnSystemProperty(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	if len(args) == 0 || len(args[0]) == 0 {
		return xpath3.SingleString(""), nil
	}

	av, err := xpath3.AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	name, err := xpath3.AtomicToString(av)
	if err != nil {
		return nil, err
	}

	// Strip prefix
	local := name
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		local = name[idx+1:]
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
		return xpath3.SingleString("no"), nil
	case "supports-serialization":
		return xpath3.SingleString("yes"), nil
	case "supports-backwards-compatibility":
		return xpath3.SingleString("yes"), nil
	default:
		return xpath3.SingleString(""), nil
	}
}

// unparsed-entity-uri(name) returns the URI of an unparsed entity.
func (ec *execContext) fnUnparsedEntityURI(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	// Not commonly used; return empty string
	return xpath3.SingleString(""), nil
}

