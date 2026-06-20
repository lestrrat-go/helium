package xslt3

import (
	"context"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xmlchar"
	"github.com/lestrrat-go/helium/xpath3"
)

func (c *compiler) compileGlobalContextItem(ctx context.Context, elem *helium.Element) error {
	if err := c.validateXSLTAttrs(ctx, elem, map[string]struct{}{
		"as": {}, "use": {}, xslAttrUseWhen: {},
	}); err != nil {
		return err
	}
	asAttr := getAttr(elem, "as")

	// Capture the declaration-site static namespace context BEFORE validating
	// the @as sequence type, so that schema-element()/schema-attribute() (and
	// plain element/attribute tests) resolve their prefixes against the context
	// in scope at this element rather than against the runtime stylesheet-wide
	// map. The in-scope bindings are derived from the element's own and ancestor
	// xmlns declarations on the DOM tree — that is authoritative and immune to
	// pollution of the mutable c.nsBindings by earlier xsl:include processing.
	nsBindings := inScopeNamespaces(elem)

	// xpath-default-namespace on this element takes precedence over the
	// inherited value. An explicitly present empty value clears any inherited
	// default element namespace, so distinguish absent from empty via the
	// presence-aware accessor.
	xpathDefaultNS := c.xpathDefaultNS
	hasXPathDefaultNS := c.xpathDefaultNS != ""
	if xdn, has := elem.GetAttribute("xpath-default-namespace"); has {
		xpathDefaultNS = xdn
		hasXPathDefaultNS = true
	}

	if err := c.validateAsSequenceTypeWithNS(ctx, asAttr, "xsl:global-context-item", nsBindings, xpathDefaultNS, hasXPathDefaultNS); err != nil {
		return err
	}
	def := &globalContextItemDef{
		Use: getAttr(elem, "use"),
		As:  asAttr,
	}
	if def.Use == "" {
		def.Use = ctxItemOptional
	}
	def.Namespaces = nsBindings
	def.XPathDefaultNS = xpathDefaultNS
	def.HasXPathDefaultNS = hasXPathDefaultNS
	if def.Use == ctxItemAbsent && def.As != "" {
		return staticError(errCodeXTSE3089, "xsl:global-context-item with use=\"absent\" must not specify @as")
	}
	moduleKey := c.moduleKey
	if moduleKey == "" {
		moduleKey = "<main>"
	}
	if c.stylesheet.globalContextModules == nil {
		c.stylesheet.globalContextModules = make(map[string]*globalContextItemDef)
	}
	// XTSE3087: more than one declaration in the same stylesheet module.
	if _, exists := c.stylesheet.globalContextModules[moduleKey]; exists {
		return staticError(errCodeXTSE3087, "multiple xsl:global-context-item declarations in stylesheet module")
	}
	c.stylesheet.globalContextModules[moduleKey] = def

	// XTSE3087: declarations in different modules of one package must agree.
	// Compare the @as types by their CANONICAL EXPANDED form: each declaration's
	// QNames are resolved against ITS OWN captured declaration-site namespace and
	// xpath-default-namespace context, then compared as {uri}local. Two
	// declarations that write the same element/schema type with different lexical
	// prefixes (or different xpath-default-namespace) are equivalent and must NOT
	// conflict; two that write the same lexical form bound to different URIs DO
	// conflict.
	if existing := c.stylesheet.globalContextItem; existing != nil {
		if existing.Use != def.Use || canonicalGlobalContextAs(existing) != canonicalGlobalContextAs(def) {
			return staticError(errCodeXTSE3087, "conflicting xsl:global-context-item declarations")
		}
		return nil
	}
	c.stylesheet.globalContextItem = def
	return nil
}

// canonicalGlobalContextAs returns a canonical, prefix-independent form of a
// global-context-item @as sequence type. Whitespace is collapsed and every
// QName inside an element()/attribute()/schema-element()/schema-attribute()
// test is rewritten to its expanded Q{uri}local form using the declaration's
// own captured namespace context — so two declarations whose @as types denote
// the same type compare equal regardless of the lexical prefixes used, and two
// that denote different types compare unequal even with identical lexical text.
func canonicalGlobalContextAs(def *globalContextItemDef) string {
	collapsed := strings.Join(strings.Fields(def.As), "")
	return expandSequenceTypeQNames(collapsed, def.Namespaces, def.XPathDefaultNS, def.HasXPathDefaultNS)
}

// expandSequenceTypeQNames rewrites EVERY QName position of an
// element()/attribute()/schema-element()/schema-attribute() kind test in a
// (whitespace-collapsed) sequence-type string to its expanded Q{uri}local form.
//
// For the element/attribute NAME argument, prefixes resolve against nsBindings
// and an unprefixed name resolves against xpathDefaultNS (when present) —
// identical to the runtime resolveSchemaQName decision used to interpret the
// same declaration. For the TYPE argument of element(name, type) and
// attribute(name, type), the type-name resolution semantics apply instead: the
// default element namespace is NOT applied (an unprefixed non-builtin type name
// is in no namespace), matching the runtime normalizeTypeName logic. The single
// argument of schema-element()/schema-attribute() is a NAME argument.
//
// Wildcards ("*") and names already in Q{...} form are left unchanged. Because
// document-node(element(...)) and other nested item-type structures are handled
// by scanning all kind tests in the string, every QName in the (possibly
// nested) item type is canonicalized, making the comparison total.
func expandSequenceTypeQNames(as string, nsBindings map[string]string, xpathDefaultNS string, hasXPathDefaultNS bool) string {
	resolve := nsResolverFromMap(nsBindings)
	for _, kind := range []string{"schema-element", "schema-attribute", "element", "attribute"} {
		// element()/attribute() carry an optional second TYPE argument;
		// schema-element()/schema-attribute() take a single NAME argument.
		expandType := kind == "element" || kind == "attribute"
		// element()/schema-element() name arguments take the default element
		// namespace; attribute()/schema-attribute() name arguments never do.
		nameKind := qnameElementName
		if kind == "attribute" || kind == "schema-attribute" {
			nameKind = qnameAttributeName
		}
		as = expandKindTestQNames(as, kind, nameKind, resolve, xpathDefaultNS, hasXPathDefaultNS, expandType)
	}
	return as
}

// expandKindTestQNames rewrites the QName arguments of every kind(...) test in
// as. The first argument (the element/attribute name) is expanded with nameKind's
// resolution rule. When expandType is true (element/attribute kind tests), the
// second argument (the type name) is also expanded, using the type-name
// resolution rule (no default-namespace application).
func expandKindTestQNames(as, kind string, nameKind qnameKind, resolve nsResolver, xpathDefaultNS string, hasXPathDefaultNS, expandType bool) string {
	search := kind + "("
	var b strings.Builder
	rest := as
	for {
		idx := indexKindTest(rest, kind)
		if idx < 0 {
			b.WriteString(rest)
			break
		}
		b.WriteString(rest[:idx+len(search)])
		rest = rest[idx+len(search):]
		// The name argument ends at the first ',' (element(name, type)) or ')'.
		nameEnd := strings.IndexAny(rest, ",)")
		if nameEnd < 0 {
			b.WriteString(rest)
			break
		}
		name := strings.TrimSpace(rest[:nameEnd])
		b.WriteString(expandTestName(name, nameKind, resolve, xpathDefaultNS, hasXPathDefaultNS))
		// If this kind test carries a second TYPE argument, expand it too.
		if expandType && rest[nameEnd] == ',' {
			b.WriteByte(',')
			rest = rest[nameEnd+1:]
			typeEnd := strings.IndexByte(rest, ')')
			if typeEnd < 0 {
				b.WriteString(rest)
				break
			}
			typeName := strings.TrimSpace(rest[:typeEnd])
			b.WriteString(expandTestName(typeName, qnameTypeName, resolve, xpathDefaultNS, hasXPathDefaultNS))
			rest = rest[typeEnd:]
			continue
		}
		rest = rest[nameEnd:]
	}
	return b.String()
}

// indexKindTest finds the next occurrence of kind+"(" in s that is a real kind
// test rather than a suffix of a longer kind name (e.g. "element" inside
// "schema-element"). It returns the index of the kind name, or -1.
func indexKindTest(s, kind string) int {
	search := kind + "("
	offset := 0
	for {
		idx := strings.Index(s[offset:], search)
		if idx < 0 {
			return -1
		}
		abs := offset + idx
		prev := byte(0)
		if abs > 0 {
			prev = s[abs-1]
		}
		// A kind name preceded by an NCName char (letter, digit, '-', '.', '_')
		// is part of a longer name (e.g. "element" within "schema-element");
		// skip it.
		if !isNCNameByte(prev) {
			return abs
		}
		offset = abs + len(search)
	}
}

func isNCNameByte(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z':
		return true
	case b >= 'A' && b <= 'Z':
		return true
	case b >= '0' && b <= '9':
		return true
	case b == '-' || b == '.' || b == '_':
		return true
	}
	return false
}

// expandTestName expands a single kind-test QName argument (element name,
// attribute name, or type name, per kind) to its canonical Q{uri}local form
// using the single unified sequence-type QName resolver.
func expandTestName(name string, kind qnameKind, resolve nsResolver, xpathDefaultNS string, hasXPathDefaultNS bool) string {
	if name == "" || name == "*" {
		return name
	}
	if strings.HasPrefix(name, "Q{") {
		return name
	}
	local, ns := resolveSequenceTypeQName(name, kind, resolve, xpathDefaultNS, hasXPathDefaultNS)
	return "Q{" + ns + "}" + local
}

// isBuiltinXSDLocalName reports whether an unprefixed type name denotes a
// built-in XML Schema type (interpreted in the XSD namespace) per the runtime
// normalizeTypeName mapping, rather than a user type in no namespace.
func isBuiltinXSDLocalName(local string) bool {
	switch local {
	case lexicon.TypeString, "integer", "decimal", "double", "float",
		lexicon.TypeBoolean, "date", "dateTime", "time", "duration",
		"dayTimeDuration", "yearMonthDuration", "anyURI", "untypedAtomic":
		return true
	}
	return false
}

// inScopeNamespaces returns the in-scope namespace bindings (prefix→URI) for
// elem by walking the DOM tree from the element up through its ancestors.
// Nearer declarations take precedence over more distant ones, matching XML
// namespace scoping. This reflects the namespaces actually written on the
// element and its ancestors, independent of any mutable compiler state.
func inScopeNamespaces(elem *helium.Element) map[string]string {
	bindings := make(map[string]string)
	for cur := helium.Node(elem); cur != nil; cur = cur.Parent() {
		ce, ok := cur.(*helium.Element)
		if !ok {
			continue
		}
		// Closer declarations win: only record a prefix not already seen on a
		// nearer (already-visited) element.
		for _, ns := range ce.Namespaces() {
			if _, seen := bindings[ns.Prefix()]; !seen {
				bindings[ns.Prefix()] = ns.URI()
			}
		}
	}
	// The xml prefix is implicitly in scope on every element (XML Namespaces
	// spec), so it must resolve even without an explicit xmlns:xml declaration.
	// An explicit declaration recorded above takes precedence over this default.
	if _, seen := bindings[lexicon.PrefixXML]; !seen {
		bindings[lexicon.PrefixXML] = lexicon.NamespaceXML
	}
	return bindings
}

// isReservedFunctionNS returns true if the given namespace URI is reserved
// by the XSLT 3.0 spec and may not be used for user-defined functions.
func isReservedFunctionNS(uri string) bool {
	switch uri {
	case lexicon.NamespaceXSLT, xpath3.NSFn, xpath3.NSMath, xpath3.NSMap, xpath3.NSArray, xpath3.NSXS:
		return true
	}
	return false
}

func (c *compiler) compileFunction(ctx context.Context, elem *helium.Element) error {
	if err := c.validateXSLTAttrs(ctx, elem, map[string]struct{}{
		xslAttrName: {}, "as": {}, xslAttrVisibility: {}, "streamable": {},
		"streamability":               {},
		"override-extension-function": {}, "override": {},
		"identity-sensitive": {}, "cache": {}, "new-each-time": {},
		xslAttrUseWhen: {},
	}); err != nil {
		return err
	}
	name := getAttr(elem, xslAttrName)
	if name == "" {
		return staticError(errCodeXTSE0110, "xsl:function requires name attribute")
	}

	// Collect namespace declarations from this element
	c.collectNamespaces(ctx, elem)

	// Resolve the prefixed name to a QualifiedName
	var qn xpath3.QualifiedName
	if strings.HasPrefix(name, "Q{") {
		// EQName: Q{uri}local
		closeBrace := strings.IndexByte(name, '}')
		if closeBrace < 0 {
			return staticError(errCodeXTSE0010, "malformed EQName in xsl:function name %q", name)
		}
		uri := name[2:closeBrace]
		local := name[closeBrace+1:]
		if uri == "" {
			return staticError(errCodeXTSE0010, "xsl:function name %q must be in a non-null namespace", name)
		}
		qn = xpath3.QualifiedName{URI: uri, Name: local}
	} else if prefix, local, ok := strings.Cut(name, ":"); ok {
		if !xmlchar.IsValidNCName(prefix) || !xmlchar.IsValidNCName(local) {
			return staticError(errCodeXTSE0020, "xsl:function name %q is not a valid QName", name)
		}
		uri := c.nsBindings[prefix]
		if uri == "" {
			uri = c.stylesheet.namespaces[prefix]
		}
		if uri == "" {
			return staticError(errCodeXTSE0010, "unresolved namespace prefix %q in xsl:function name %q", prefix, name)
		}
		qn = xpath3.QualifiedName{URI: uri, Name: local}
	} else {
		return staticError(errCodeXTSE0010, "xsl:function name %q must have a namespace prefix", name)
	}

	// XTSE0080: function name must not be in a reserved namespace
	if isReservedFunctionNS(qn.URI) {
		return staticError(errCodeXTSE0080, "xsl:function name %q is in a reserved namespace", name)
	}

	// Handle expand-text on xsl:function (using GetAttribute to catch empty values)
	savedExpandText := c.expandText
	if et, hasET := elem.GetAttribute("expand-text"); hasET {
		if v, ok := parseXSDBool(et); ok {
			c.expandText = v
		} else {
			return staticError(errCodeXTSE0020, "%q is not a valid value for xsl:function/@expand-text", et)
		}
	}

	// XTSE0020: validate override-extension-function (boolean)
	if oef, has := elem.GetAttribute("override-extension-function"); has {
		if err := validateBooleanAttr("xsl:function", "override-extension-function", oef); err != nil {
			return err
		}
	}
	// XTSE0020: validate override (boolean)
	if ov, hasOv := elem.GetAttribute("override"); hasOv {
		if err := validateBooleanAttr("xsl:function", "override", ov); err != nil {
			return err
		}
		// Having both @override and @override-extension-function is only an error
		// when they disagree (XTSE0020 per spec function-0117).
		if oef, hasOEF := elem.GetAttribute("override-extension-function"); hasOEF {
			ovBool, _ := parseXSDBool(ov)
			oefBool, _ := parseXSDBool(oef)
			if ovBool != oefBool {
				return staticError(errCodeXTSE0020,
					"xsl:function has conflicting @override=%q and @override-extension-function=%q", ov, oef)
			}
		}
	}
	// XTSE0020: validate new-each-time (yes|no|maybe)
	if net := getAttr(elem, "new-each-time"); net != "" {
		switch net {
		case lexicon.ValueYes, lexicon.ValueNo, "maybe":
			// valid
		default:
			return staticError(errCodeXTSE0020,
				"%q is not a valid value for xsl:function/@new-each-time", net)
		}
	}

	// Compile function body (params + instructions)
	_, body, params, err := c.compileTemplateBodyEx(ctx, elem, true)
	c.expandText = savedExpandText
	if err != nil {
		return err
	}

	// XTSE0020: xsl:param inside xsl:function must not have required="no"
	// (function params are implicitly required; required="yes" is allowed as redundant)
	for child := range helium.Children(elem) {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() == lexicon.NamespaceXSLT && childElem.LocalName() == lexicon.XSLTElementParam {
			if reqVal, hasReq := childElem.GetAttribute("required"); hasReq {
				if reqVal != lexicon.ValueYes && reqVal != "1" && reqVal != lexicon.ValueTrue {
					pname := getAttr(childElem, "name")
					return staticError(errCodeXTSE0020,
						"xsl:param %q in xsl:function must not have required=%q", pname, reqVal)
				}
			}
			// XTSE0020: tunnel="yes" is not allowed on a function parameter
			if getAttr(childElem, "tunnel") == lexicon.ValueYes {
				return staticError(errCodeXTSE0020,
					"tunnel=\"yes\" is not allowed on a function parameter")
			}
			// XTSE0760: function params must not have a default value
			if getAttr(childElem, "select") != "" {
				return staticError(errCodeXTSE0760,
					"xsl:param %q in xsl:function must not have a select attribute", getAttr(childElem, "name"))
			}
			if childElem.FirstChild() != nil {
				// Check for non-whitespace content (body content = default value)
				for ch := range helium.Children(childElem) {
					if ch.Type() == helium.ElementNode {
						return staticError(errCodeXTSE0760,
							"xsl:param %q in xsl:function must not have content", getAttr(childElem, "name"))
					}
					if ch.Type() == helium.TextNode && strings.TrimSpace(string(ch.Content())) != "" {
						return staticError(errCodeXTSE0760,
							"xsl:param %q in xsl:function must not have content", getAttr(childElem, "name"))
					}
				}
			}
		}
	}

	fnAs := getAttr(elem, "as")
	if err := c.validateAsSequenceType(ctx, fnAs, "xsl:function "+name); err != nil {
		return err
	}

	// XTSE3155: xsl:function with no params and streamability != unclassified
	streamability := getAttr(elem, "streamability")
	if streamability == "" {
		streamability = getAttr(elem, "streamable")
		switch streamability {
		case lexicon.ValueYes:
			streamability = lexicon.StreamAbsorbing
		case lexicon.ValueNo, "":
			streamability = ""
		}
	}
	if len(params) == 0 && streamability != "" && streamability != lexicon.StreamUnclassified {
		return staticError(errCodeXTSE3155,
			"xsl:function %q with no parameters must not have streamability=%q (only unclassified allowed)", name, streamability)
	}

	fn := &xslFunction{
		Name:          qn,
		Params:        params,
		Body:          body,
		As:            fnAs,
		Cache:         xsdBoolTrue(getAttr(elem, "cache")),
		Streamability: getAttr(elem, "streamability"),
		Visibility:    getAttr(elem, "visibility"),
		NewEachTime:   getAttr(elem, "new-each-time"),
		ImportPrec:    c.importPrec,
	}
	if c.stylesheet.isPackage {
		fn.OwnerPackage = c.stylesheet
	}

	// XTSE0770: it is a static error if a stylesheet contains two or more
	// functions with the same expanded QName, the same arity, and the same
	// import precedence. Functions from different import levels are allowed;
	// the one with the highest precedence wins.
	fk := funcKey{Name: qn, Arity: len(fn.Params)}
	if existing, ok := c.stylesheet.functions[fk]; ok {
		if existing.ImportPrec == fn.ImportPrec {
			return staticError(errCodeXTSE0770,
				"duplicate xsl:function %s with arity %d", name, len(fn.Params))
		}
	}
	// XTSE0770: it is a static error if an xsl:function has the same
	// expanded QName and arity as a schema-defined constructor function.
	if len(fn.Params) == 1 {
		for _, sch := range c.stylesheet.schemas {
			if _, found := sch.LookupType(qn.Name, qn.URI); found {
				return staticError(errCodeXTSE0770,
					"xsl:function %s conflicts with schema-defined constructor of the same name", name)
			}
		}
	}
	c.stylesheet.functions[fk] = fn
	return nil
}

func (c *compiler) compileMode(ctx context.Context, elem *helium.Element) error {
	if err := c.validateXSLTAttrs(ctx, elem, map[string]struct{}{
		xslAttrName: {}, "streamable": {}, "on-no-match": {}, "on-multiple-match": {},
		"warning-on-no-match": {}, "warning-on-multiple-match": {},
		"typed": {}, xslAttrVisibility: {}, xslAttrUseWhen: {}, "use-accumulators": {},
	}); err != nil {
		return err
	}
	// xsl:mode must be empty (no children)
	if elem.FirstChild() != nil {
		return staticError(errCodeXTSE0010, "xsl:mode must be empty")
	}

	name := strings.TrimSpace(getAttr(elem, "name"))
	if name == "" {
		name = modeDefault
	} else if name == modeUnnamed || name == modeAll || name == modeCurrent {
		return staticError(errCodeXTSE0020, "invalid mode name %q on xsl:mode", name)
	} else if name[0] != '#' {
		// Resolve QName to Clark notation so mode declarations and mode
		// references (on xsl:template/@mode, xsl:apply-templates/@mode)
		// use the same key format.
		name = c.resolveMode(ctx, name)
	}

	// Parse streamable with proper xs:boolean validation
	streamableStr := strings.TrimSpace(getAttr(elem, "streamable"))
	streamable := false
	if streamableStr != "" {
		v, ok := parseXSDBool(streamableStr)
		if !ok {
			return staticError(errCodeXTSE0020, "invalid value %q for streamable on xsl:mode", streamableStr)
		}
		streamable = v
	}

	// Validate on-no-match
	onNoMatch := strings.TrimSpace(getAttr(elem, "on-no-match"))
	if onNoMatch != "" {
		switch onNoMatch {
		case onNoMatchTextOnlyCopy, onNoMatchShallowCopy, onNoMatchDeepCopy, onNoMatchShallowSkip, onNoMatchDeepSkip, onNoMatchFail:
			// valid
		default:
			return staticError(errCodeXTSE0020, "invalid value %q for on-no-match on xsl:mode", onNoMatch)
		}
	}
	// Note: we keep onNoMatch as "" when not specified, so we can detect
	// conflicts properly. The default "text-only-copy" is applied later.

	// Validate boolean attributes
	if v := getAttr(elem, "warning-on-no-match"); v != "" {
		if _, ok := parseXSDBool(v); !ok {
			return staticError(errCodeXTSE0020, "invalid value %q for warning-on-no-match on xsl:mode", v)
		}
	}
	if v := getAttr(elem, "warning-on-multiple-match"); v != "" {
		if _, ok := parseXSDBool(v); !ok {
			return staticError(errCodeXTSE0020, "invalid value %q for warning-on-multiple-match on xsl:mode", v)
		}
	}
	if v := getAttr(elem, "typed"); v != "" {
		// typed accepts "yes", "no", "true", "false", "1", "0",
		// "strict", "lax", "unspecified"
		switch v {
		case validationStrict, validationLax, validationUnspecified:
			// valid non-boolean values
		default:
			if _, ok := parseXSDBool(v); !ok {
				return staticError(errCodeXTSE0020, "invalid value %q for typed on xsl:mode", v)
			}
		}
	}

	visibility := strings.TrimSpace(getAttr(elem, "visibility"))
	// XTSE0020: unnamed mode cannot have visibility="public" or "final"
	if name == modeDefault && (visibility == visPublic || visibility == visFinal) {
		return staticError(errCodeXTSE0020, "unnamed mode cannot have visibility %q", visibility)
	}

	onMultipleMatch := strings.TrimSpace(getAttr(elem, "on-multiple-match"))
	if onMultipleMatch != "" {
		switch onMultipleMatch {
		case onMultipleMatchUseLast, onNoMatchFail:
			// valid
		default:
			return staticError(errCodeXTSE0020, "invalid value %q for on-multiple-match on xsl:mode", onMultipleMatch)
		}
	}

	// Resolve accumulator QNames to expanded names for proper comparison
	// across modules with different namespace prefixes.
	rawUA := getAttr(elem, "use-accumulators")
	_, hasUseAccumulators := elem.GetAttribute("use-accumulators")
	var useAccumulators *string
	if hasUseAccumulators {
		var resolvedParts []string
		for tok := range strings.FieldsSeq(rawUA) {
			if tok == "#all" {
				resolvedParts = append(resolvedParts, tok)
			} else {
				resolvedParts = append(resolvedParts, resolveQName(tok, c.nsBindings))
			}
		}
		s := strings.Join(resolvedParts, " ")
		useAccumulators = &s
	}

	typed := getAttr(elem, "typed")

	md := &modeDef{
		Name:            name,
		OnNoMatch:       onNoMatch,
		Typed:           typed,
		Streamable:      streamable,
		Visibility:      visibility,
		OnMultipleMatch: onMultipleMatch,
		UseAccumulators: useAccumulators,
		ImportPrec:      c.importPrec,
	}

	if c.stylesheet.modeDefs == nil {
		c.stylesheet.modeDefs = make(map[string]*modeDef)
	}

	// Check for conflicting declarations at the same import precedence
	if existing, ok := c.stylesheet.modeDefs[name]; ok {
		if existing.ImportPrec == c.importPrec {
			// Same precedence: check for conflicting attribute values (XTSE0545).
			// Instead of erroring immediately, defer the conflict — a higher-
			// precedence declaration may resolve it later.
			if existing.OnNoMatch != "" && md.OnNoMatch != "" && existing.OnNoMatch != md.OnNoMatch {
				existing.conflictOnNoMatch = true
			}
			if streamableStr != "" && existing.Streamable != md.Streamable {
				existing.conflictStreamable = true
			}
			if existing.Visibility != "" && md.Visibility != "" && existing.Visibility != md.Visibility {
				existing.conflictVisibility = true
			}
			if existing.OnMultipleMatch != "" && md.OnMultipleMatch != "" && existing.OnMultipleMatch != md.OnMultipleMatch {
				existing.conflictOnMultiple = true
			}
			if existing.UseAccumulators != nil && md.UseAccumulators != nil && !sameAccumulatorSet(*existing.UseAccumulators, *md.UseAccumulators) {
				existing.conflictAccumulator = true
			}
			// Non-conflicting: merge attributes (use non-empty values from new decl)
			if md.OnNoMatch != "" {
				existing.OnNoMatch = md.OnNoMatch
			}
			if md.Visibility != "" {
				existing.Visibility = md.Visibility
			}
			if md.OnMultipleMatch != "" {
				existing.OnMultipleMatch = md.OnMultipleMatch
			}
			if md.UseAccumulators != nil {
				existing.UseAccumulators = md.UseAccumulators
			}
			return nil
		}
		// Different precedence: higher precedence wins, but inherit
		// unspecified attributes from the lower-precedence declaration.
		if c.importPrec > existing.ImportPrec {
			// Preserve conflict flags from the lower-precedence entry,
			// but clear them for attributes the higher-prec explicitly specifies.
			md.conflictStreamable = existing.conflictStreamable && streamableStr == ""
			md.conflictOnNoMatch = existing.conflictOnNoMatch && md.OnNoMatch == ""
			md.conflictVisibility = existing.conflictVisibility && md.Visibility == ""
			md.conflictOnMultiple = existing.conflictOnMultiple && md.OnMultipleMatch == ""
			md.conflictAccumulator = existing.conflictAccumulator && md.UseAccumulators == nil
			// Inherit attributes not explicitly set in higher-precedence decl
			if md.OnNoMatch == "" {
				md.OnNoMatch = existing.OnNoMatch
			}
			if md.OnMultipleMatch == "" {
				md.OnMultipleMatch = existing.OnMultipleMatch
			}
			if md.UseAccumulators == nil {
				md.UseAccumulators = existing.UseAccumulators
			}
			if md.Visibility == "" {
				md.Visibility = existing.Visibility
			}
			c.stylesheet.modeDefs[name] = md
		} else {
			// Lower precedence than existing: merge unspecified attrs into existing.
			if existing.OnNoMatch == "" && md.OnNoMatch != "" {
				existing.OnNoMatch = md.OnNoMatch
			}
			if existing.OnMultipleMatch == "" && md.OnMultipleMatch != "" {
				existing.OnMultipleMatch = md.OnMultipleMatch
			}
			if existing.UseAccumulators == nil && md.UseAccumulators != nil {
				existing.UseAccumulators = md.UseAccumulators
			}
			if existing.Visibility == "" && md.Visibility != "" {
				existing.Visibility = md.Visibility
			}
		}
		return nil
	}

	c.stylesheet.modeDefs[name] = md
	return nil
}

// sameAccumulatorSet checks whether two use-accumulators values contain
// the same set of accumulator names (order-independent).
func sameAccumulatorSet(a, b string) bool {
	if a == b {
		return true
	}
	as := make(map[string]struct{})
	for s := range strings.FieldsSeq(a) {
		as[s] = struct{}{}
	}
	bs := make(map[string]struct{})
	for s := range strings.FieldsSeq(b) {
		bs[s] = struct{}{}
	}
	if len(as) != len(bs) {
		return false
	}
	for k := range as {
		if _, ok := bs[k]; !ok {
			return false
		}
	}
	return true
}
