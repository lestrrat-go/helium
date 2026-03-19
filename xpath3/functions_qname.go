package xpath3

import (
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
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

func fnPrefixFromQName(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	a, err := AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	a = PromoteSchemaType(a)
	if a.TypeName != TypeQName {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "expected QName"}
	}
	qv := a.QNameVal()
	if qv.Prefix == "" {
		return nil, nil
	}
	return SingleString(qv.Prefix), nil
}

func fnLocalNameFromQName(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	a, err := AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	a = PromoteSchemaType(a)
	if a.TypeName != TypeQName {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "expected QName"}
	}
	return SingleString(a.QNameVal().Local), nil
}

func fnNamespaceURIFromQName(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	a, err := AtomizeItem(args[0][0])
	if err != nil {
		return nil, err
	}
	a = PromoteSchemaType(a)
	if a.TypeName != TypeQName {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "expected QName"}
	}
	return SingleAtomic(AtomicValue{TypeName: TypeAnyURI, Value: a.QNameVal().URI}), nil
}

func fnNamespaceURIForPrefix(_ context.Context, args []Sequence) (Sequence, error) {
	prefix, err := coerceQNameString(args[0], true, false, "fn:namespace-uri-for-prefix prefix argument must be a string")
	if err != nil {
		return nil, err
	}
	if len(args[1]) == 0 {
		return nil, nil
	}
	ni, ok := args[1][0].(NodeItem)
	if !ok {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "expected element node"}
	}
	elem, ok := ni.Node.(*helium.Element)
	if !ok {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "expected element node"}
	}
	if ns := helium.LookupNSByPrefix(elem, prefix); ns != nil {
		return SingleAtomic(AtomicValue{TypeName: TypeAnyURI, Value: ns.URI()}), nil
	}
	return nil, nil
}

func fnResolveQName(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	qnameStr, err := coerceQNameString(args[0], false, false, "resolve-QName: QName argument must be a string")
	if err != nil {
		return nil, err
	}
	if len(args[1]) == 0 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "resolve-QName: element argument is empty"}
	}
	ni, ok := args[1][0].(NodeItem)
	if !ok {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "resolve-QName: expected element node"}
	}
	elem, ok := ni.Node.(*helium.Element)
	if !ok {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "resolve-QName: expected element node"}
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
	if idx := strings.IndexByte(qname, ':'); idx >= 0 {
		prefix = qname[:idx]
		local = qname[idx+1:]
		if prefix == "" {
			return "", "", &XPathError{Code: errCodeFOCA0002, Message: "invalid QName: " + qname}
		}
	}
	if prefix != "" && !isValidNCName(prefix) {
		return "", "", &XPathError{Code: errCodeFOCA0002, Message: "invalid prefix in QName: " + prefix}
	}
	if !isValidNCName(local) {
		return "", "", &XPathError{Code: errCodeFOCA0002, Message: "invalid local name in QName: " + local}
	}
	return prefix, local, nil
}

func coerceQNameString(seq Sequence, allowEmpty, allowAnyURI bool, message string) (string, error) {
	switch len(seq) {
	case 0:
		if allowEmpty {
			return "", nil
		}
		return "", &XPathError{Code: errCodeXPTY0004, Message: message}
	case 1:
	default:
		return "", &XPathError{Code: errCodeXPTY0004, Message: message}
	}

	a, err := AtomizeItem(seq[0])
	if err != nil {
		return "", err
	}
	switch a.TypeName {
	case TypeString, TypeUntypedAtomic:
	case TypeAnyURI:
		if !allowAnyURI {
			return "", &XPathError{Code: errCodeXPTY0004, Message: message}
		}
	default:
		return "", &XPathError{Code: errCodeXPTY0004, Message: message}
	}

	s, ok := a.Value.(string)
	if !ok {
		return "", fmt.Errorf("xpath3: internal error: expected string for %s", a.TypeName)
	}
	return s, nil
}

func fnInScopePrefixes(_ context.Context, args []Sequence) (Sequence, error) {
	if len(args[0]) == 0 {
		return nil, nil
	}
	ni, ok := args[0][0].(NodeItem)
	if !ok {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "expected element node"}
	}
	elem, ok := ni.Node.(*helium.Element)
	if !ok {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "expected element node"}
	}

	// Walk from the context element outward so namespace undeclarations mask
	// ancestor bindings for the same prefix.
	prefixes := map[string]bool{"xml": true}
	resolved := map[string]bool{"xml": true}
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

	result := make(Sequence, 0, len(prefixes))
	for prefix, active := range prefixes {
		if !active {
			continue
		}
		result = append(result, AtomicValue{TypeName: TypeString, Value: prefix})
	}
	return result, nil
}
