package lexicon

import "strings"

// IsFnNamespacePrefix reports whether a function-call prefix names the XPath
// functions namespace (http://www.w3.org/2005/xpath-functions) for the purpose
// of streamability analysis. For function calls an empty prefix defaults to that
// namespace, and "fn" is the conventional reserved prefix bound to it, so both
// lexical forms name the same built-in. Streamability special-cases functions
// such as position()/last() by (namespace, local-name) and must therefore treat
// the unprefixed and "fn:"-prefixed spellings identically.
//
// This is a deliberately lexical check, not a namespace-resolved one: the
// analysis runs over the static AST with no access to a prefix->URI map (the map
// only exists at eval time, via the xpath3 evaluator's namespace bindings). We
// make the same assumption the xpath3 evaluator's own static-shape recognition
// already makes (eval_path.go positionCall: Prefix == "" || Prefix == "fn").
// Edge case: a caller that rebinds "fn" to a custom namespace and then calls
// fn:position() would have it resolve to a user function at eval time while
// streamability still treats it as the built-in. Rebinding the reserved "fn"
// prefix is pathological and unsupported here; consistency with the evaluator is
// preferred.
func IsFnNamespacePrefix(prefix string) bool {
	return prefix == "" || prefix == "fn"
}

// StreamFnLocalName resolves a function call's lexical (name, prefix) to its
// local name for streamability analysis, reporting whether the call names the
// XPath functions namespace.
//
// The parser keeps an EQName function call's whole braced spelling in the name
// (e.g. "Q{http://www.w3.org/2005/xpath-functions}position"), so a bare lexical
// comparison against "position"/"last" would miss it. This normalizes that form
// to its local part when the braced URI is the functions namespace. For the
// lexical (unprefixed or "fn:") forms it defers to IsFnNamespacePrefix and
// returns name unchanged.
func StreamFnLocalName(name, prefix string) (string, bool) {
	if strings.HasPrefix(name, "Q{") {
		if idx := strings.Index(name, "}"); idx >= 0 {
			return name[idx+1:], name[2:idx] == NamespaceFn
		}
		return name, false
	}
	return name, IsFnNamespacePrefix(prefix)
}
