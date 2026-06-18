package xpath3

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xmlchar"
)

func init() {
	registerFn("QName", 2, 2, fnQName)
	registerFn("prefix-from-QName", 1, 1, fnPrefixFromQName)
	registerFn("local-name-from-QName", 1, 1, fnLocalNameFromQName)
	registerFn("namespace-uri-from-QName", 1, 1, fnNamespaceURIFromQName)
	registerFn("namespace-uri-for-prefix", 2, 2, fnNamespaceURIForPrefix)
	registerFn("in-scope-prefixes", 1, 1, fnInScopePrefixes)
	registerFn("resolve-QName", 2, 2, fnResolveQName)
}

func fnQName(_ context.Context, args []Sequence) (Sequence, error) {
	uri, err := coerceQNameString(args[0], true, true, "fn:QName namespace argument must be a string")
	if err != nil {
		return nil, err
	}
	qname, err := coerceQNameString(args[1], false, false, "fn:QName QName argument must be a string")
	if err != nil {
		return nil, err
	}
	prefix, local, err := parseLexicalQName(qname)
	if err != nil {
		return nil, err
	}
	// Validate: if there's a prefix, namespace must be non-empty
	if prefix != "" && uri == "" {
		return nil, &XPathError{Code: errCodeFOCA0002, Message: "namespace must not be empty when QName has a prefix"}
	}
	return SingleAtomic(AtomicValue{
		TypeName: TypeQName,
		Value:    QNameValue{Prefix: prefix, Local: local, URI: uri},
	}), nil
}

// atomizeQNameArg atomizes a QName-accessor argument and enforces 0-or-1
// cardinality on the ATOMIZED result, so that a single array/list item that
// atomizes to multiple values is rejected as XPTY0004 (not FOTY0013). It
// stops atomization early once a second atom appears. The returned bool is
// true when the atomized sequence is empty (applicable result is the empty
// sequence); otherwise the returned AtomicValue is the single xs:QName.
func atomizeQNameArg(seq Sequence, fname string) (AtomicValue, bool, error) {
	var first AtomicValue
	count := 0
	err := atomizeStream(seq, func(av AtomicValue) (bool, error) {
		count++
		if count == 1 {
			first = av
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return AtomicValue{}, false, err
	}
	if count == 0 {
		return AtomicValue{}, true, nil
	}
	if count > 1 {
		return AtomicValue{}, false, &XPathError{Code: lexicon.ErrXPTY0004, Message: fname + " expects a single QName"}
	}
	first = PromoteSchemaType(first)
	if first.TypeName != TypeQName {
		return AtomicValue{}, false, &XPathError{Code: lexicon.ErrXPTY0004, Message: "expected QName"}
	}
	return first, false, nil
}

func fnPrefixFromQName(_ context.Context, args []Sequence) (Sequence, error) {
	a, empty, err := atomizeQNameArg(args[0], "fn:prefix-from-QName")
	if err != nil {
		return nil, err
	}
	if empty {
		return validNilSequence, nil
	}
	qv := a.QNameVal()
	if qv.Prefix == "" {
		return validNilSequence, nil
	}
	return ItemSlice{AtomicValue{TypeName: TypeNCName, Value: qv.Prefix}}, nil
}

func fnLocalNameFromQName(_ context.Context, args []Sequence) (Sequence, error) {
	a, empty, err := atomizeQNameArg(args[0], "fn:local-name-from-QName")
	if err != nil {
		return nil, err
	}
	if empty {
		return validNilSequence, nil
	}
	return ItemSlice{AtomicValue{TypeName: TypeNCName, Value: a.QNameVal().Local}}, nil
}

func fnNamespaceURIFromQName(_ context.Context, args []Sequence) (Sequence, error) {
	a, empty, err := atomizeQNameArg(args[0], "fn:namespace-uri-from-QName")
	if err != nil {
		return nil, err
	}
	if empty {
		return validNilSequence, nil
	}
	return SingleAtomic(AtomicValue{TypeName: TypeAnyURI, Value: a.QNameVal().URI}), nil
}

func fnNamespaceURIForPrefix(_ context.Context, args []Sequence) (Sequence, error) {
	// The element() second argument is required and must be exactly one
	// element() regardless of the first argument, so validate it FIRST.
	elem, err := requireSingleElement(args[1], "fn:namespace-uri-for-prefix")
	if err != nil {
		return nil, err
	}
	prefix, err := coerceQNameString(args[0], true, false, "fn:namespace-uri-for-prefix prefix argument must be a string")
	if err != nil {
		return nil, err
	}
	if ns := helium.LookupNSByPrefix(elem, prefix); ns != nil {
		return SingleAtomic(AtomicValue{TypeName: TypeAnyURI, Value: ns.URI()}), nil
	}
	return validNilSequence, nil
}

// requireSingleElement validates a required element() argument: it must be
// exactly one node and that node must be a *helium.Element. This is validated
// before any sibling argument is coerced so an invalid element() arg yields
// XPTY0004 rather than an error from atomizing the other argument.
func requireSingleElement(seq Sequence, fname string) (*helium.Element, error) {
	if seqLen(seq) != 1 {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fname + ": expects a single element"}
	}
	ni, ok := seq.Get(0).(NodeItem)
	if !ok {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fname + ": expected element node"}
	}
	elem, ok := ni.Node.(*helium.Element)
	if !ok {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: fname + ": expected element node"}
	}
	return elem, nil
}

func fnResolveQName(_ context.Context, args []Sequence) (Sequence, error) {
	// The element() second argument is required and must be exactly one
	// element() regardless of whether $qname is empty, so validate it FIRST.
	elem, err := requireSingleElement(args[1], "resolve-QName")
	if err != nil {
		return nil, err
	}

	// Empty $qname yields the empty sequence (after the element() check).
	if seqLen(args[0]) == 0 {
		return validNilSequence, nil
	}
	qnameStr, err := coerceQNameString(args[0], false, false, "resolve-QName: QName argument must be a string")
	if err != nil {
		return nil, err
	}

	prefix, local, err := parseLexicalQName(qnameStr)
	if err != nil {
		return nil, err
	}

	uri := ""
	if prefix != "" {
		ns := helium.LookupNSByPrefix(elem, prefix)
		if ns == nil {
			return nil, &XPathError{Code: errCodeFONS0004, Message: "resolve-QName: no namespace binding for prefix " + prefix}
		}
		uri = ns.URI()
	} else {
		// Default namespace
		ns := helium.LookupNSByPrefix(elem, "")
		if ns != nil {
			uri = ns.URI()
		}
	}

	return SingleAtomic(AtomicValue{
		TypeName: TypeQName,
		Value:    QNameValue{Prefix: prefix, Local: local, URI: uri},
	}), nil
}

func parseLexicalQName(qname string) (string, string, error) {
	prefix := ""
	local := qname
	if p, l, ok := strings.Cut(qname, ":"); ok {
		prefix = p
		local = l
		if prefix == "" {
			return "", "", &XPathError{Code: errCodeFOCA0002, Message: "invalid QName: " + qname}
		}
	}
	if prefix != "" && !xmlchar.IsValidNCName(prefix) {
		return "", "", &XPathError{Code: errCodeFOCA0002, Message: "invalid prefix in QName: " + prefix}
	}
	if !xmlchar.IsValidNCName(local) {
		return "", "", &XPathError{Code: errCodeFOCA0002, Message: "invalid local name in QName: " + local}
	}
	return prefix, local, nil
}

func coerceQNameString(seq Sequence, allowEmpty, allowAnyURI bool, message string) (string, error) {
	switch seqLen(seq) {
	case 0:
		if allowEmpty {
			return "", nil
		}
		return "", &XPathError{Code: lexicon.ErrXPTY0004, Message: message}
	case 1:
	default:
		return "", &XPathError{Code: lexicon.ErrXPTY0004, Message: message}
	}

	a, err := AtomizeItem(seq.Get(0))
	if err != nil {
		return "", err
	}
	switch a.TypeName {
	case TypeString, TypeUntypedAtomic:
	case TypeAnyURI:
		if !allowAnyURI {
			return "", &XPathError{Code: lexicon.ErrXPTY0004, Message: message}
		}
	default:
		return "", &XPathError{Code: lexicon.ErrXPTY0004, Message: message}
	}

	s, ok := a.Value.(string)
	if !ok {
		return "", fmt.Errorf("xpath3: internal error: expected string for %s", a.TypeName)
	}
	return s, nil
}

func fnInScopePrefixes(_ context.Context, args []Sequence) (Sequence, error) {
	if seqLen(args[0]) != 1 {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "fn:in-scope-prefixes expects a single element"}
	}
	ni, ok := args[0].Get(0).(NodeItem)
	if !ok {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "expected element node"}
	}
	elem, ok := ni.Node.(*helium.Element)
	if !ok {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "expected element node"}
	}

	// Walk from the context element outward so namespace undeclarations mask
	// ancestor bindings for the same prefix.
	prefixes := map[string]bool{lexicon.PrefixXML: true}
	resolved := map[string]bool{lexicon.PrefixXML: true}
	for cur := elem; cur != nil; {
		for _, ns := range cur.Namespaces() {
			prefix := ns.Prefix()
			if _, ok := resolved[prefix]; ok {
				continue
			}
			prefixes[prefix] = ns.URI() != ""
			resolved[prefix] = true
		}
		p := cur.Parent()
		if p == nil {
			break
		}
		if pe, ok := p.(*helium.Element); ok {
			cur = pe
		} else {
			break
		}
	}

	// Collect active prefixes into a sorted slice to ensure deterministic
	// output order (the XPath 3.1 spec leaves order implementation-defined).
	sorted := make([]string, 0, len(prefixes))
	for prefix, active := range prefixes {
		if active {
			sorted = append(sorted, prefix)
		}
	}
	sort.Strings(sorted)

	result := make(ItemSlice, 0, len(sorted))
	for _, prefix := range sorted {
		result = append(result, AtomicValue{TypeName: TypeString, Value: prefix})
	}
	return result, nil
}
