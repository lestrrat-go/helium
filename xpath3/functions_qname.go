package xpath3

import (
	"context"
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
	// Validate argument types: $paramURI as xs:string?, $paramQName as xs:string
	if len(args[0]) > 0 {
		a, err := AtomizeItem(args[0][0])
		if err != nil {
			return nil, err
		}
		if a.TypeName != TypeString && a.TypeName != TypeUntypedAtomic && a.TypeName != TypeAnyURI {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "fn:QName namespace argument must be a string"}
		}
	}
	if len(args[1]) > 0 {
		a, err := AtomizeItem(args[1][0])
		if err != nil {
			return nil, err
		}
		if a.TypeName != TypeString && a.TypeName != TypeUntypedAtomic {
			return nil, &XPathError{Code: errCodeXPTY0004, Message: "fn:QName QName argument must be a string"}
		}
	}

	uri := seqToString(args[0])
	qname := seqToString(args[1])
	prefix := ""
	local := qname
	if idx := strings.IndexByte(qname, ':'); idx >= 0 {
		prefix = qname[:idx]
		local = qname[idx+1:]
		// Empty prefix with colon (e.g., ":person") is invalid
		if prefix == "" {
			return nil, &XPathError{Code: errCodeFOCA0002, Message: "invalid QName: " + qname}
		}
	}
	// Validate: if there's a prefix, namespace must be non-empty
	if prefix != "" && uri == "" {
		return nil, &XPathError{Code: errCodeFOCA0002, Message: "namespace must not be empty when QName has a prefix"}
	}
	// Validate: prefix (if present) must be a valid NCName
	if prefix != "" && !isValidNCName(prefix) {
		return nil, &XPathError{Code: errCodeFOCA0002, Message: "invalid prefix in QName: " + prefix}
	}
	// Validate: local part must be a valid NCName
	if !isValidNCName(local) {
		return nil, &XPathError{Code: errCodeFOCA0002, Message: "invalid local name in QName: " + local}
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
	if a.TypeName != TypeQName {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "expected QName"}
	}
	return SingleAtomic(AtomicValue{TypeName: TypeAnyURI, Value: a.QNameVal().URI}), nil
}

func fnNamespaceURIForPrefix(_ context.Context, args []Sequence) (Sequence, error) {
	prefix := seqToString(args[0])
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
	qnameStr := seqToString(args[0])
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

	prefix := ""
	local := qnameStr
	if idx := strings.IndexByte(qnameStr, ':'); idx >= 0 {
		prefix = qnameStr[:idx]
		local = qnameStr[idx+1:]
	}

	uri := ""
	if prefix != "" {
		ns := helium.LookupNSByPrefix(elem, prefix)
		if ns == nil {
			return nil, &XPathError{Code: "FONS0004", Message: "resolve-QName: no namespace binding for prefix " + prefix}
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

	// Collect in-scope prefixes by walking up the tree
	prefixes := make(map[string]bool)
	prefixes["xml"] = true
	for cur := elem; cur != nil; {
		for _, ns := range cur.Namespaces() {
			prefix := ns.Prefix()
			if !prefixes[prefix] {
				prefixes[prefix] = true
			}
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
	for prefix := range prefixes {
		result = append(result, AtomicValue{TypeName: TypeString, Value: prefix})
	}
	return result, nil
}
