package xslt3

import (
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func (c *compiler) compileGlobalContextItem(elem *helium.Element) error {
	asAttr := getAttr(elem, "as")
	if err := c.validateAsSequenceType(asAttr, "xsl:global-context-item"); err != nil {
		return err
	}
	def := &GlobalContextItemDef{
		Use: getAttr(elem, "use"),
		As:  asAttr,
	}
	if def.Use == "" {
		def.Use = "optional"
	}
	if def.Use == "absent" && def.As != "" {
		return staticError("XTSE3089", "xsl:global-context-item with use=\"absent\" must not specify @as")
	}
	moduleKey := c.moduleKey
	if moduleKey == "" {
		moduleKey = "<main>"
	}
	if c.stylesheet.globalContextModules == nil {
		c.stylesheet.globalContextModules = make(map[string]*GlobalContextItemDef)
	}
	// XTSE3087: more than one declaration in the same stylesheet module.
	if _, exists := c.stylesheet.globalContextModules[moduleKey]; exists {
		return staticError(errCodeXTSE3087, "multiple xsl:global-context-item declarations in stylesheet module")
	}
	c.stylesheet.globalContextModules[moduleKey] = def

	// XTSE3087: declarations in different modules of one package must agree.
	if existing := c.stylesheet.globalContextItem; existing != nil {
		normalizeAs := func(s string) string {
			return strings.Join(strings.Fields(s), "")
		}
		if existing.Use != def.Use || normalizeAs(existing.As) != normalizeAs(def.As) {
			return staticError(errCodeXTSE3087, "conflicting xsl:global-context-item declarations")
		}
		return nil
	}
	c.stylesheet.globalContextItem = def
	return nil
}

// isReservedFunctionNS returns true if the given namespace URI is reserved
// by the XSLT 3.0 spec and may not be used for user-defined functions.
func isReservedFunctionNS(uri string) bool {
	switch uri {
	case NSXSLT, xpath3.NSFn, xpath3.NSMath, xpath3.NSMap, xpath3.NSArray, xpath3.NSXS:
		return true
	}
	return false
}

func (c *compiler) compileFunction(elem *helium.Element) error {
	name := getAttr(elem, "name")
	if name == "" {
		return staticError(errCodeXTSE0110, "xsl:function requires name attribute")
	}

	// Collect namespace declarations from this element
	c.collectNamespaces(elem)

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
	} else if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		local := name[idx+1:]
		if !isValidNCName(prefix) || !isValidNCName(local) {
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
		// Both override and override-extension-function is XTSE0020
		if _, hasOEF := elem.GetAttribute("override-extension-function"); hasOEF {
			return staticError(errCodeXTSE0020,
				"xsl:function must not have both @override and @override-extension-function")
		}
	}
	// XTSE0020: validate new-each-time (yes|no|maybe)
	if net := getAttr(elem, "new-each-time"); net != "" {
		switch net {
		case "yes", "no", "maybe":
			// valid
		default:
			return staticError(errCodeXTSE0020,
				"%q is not a valid value for xsl:function/@new-each-time", net)
		}
	}

	// Compile function body (params + instructions)
	body, params, err := c.compileTemplateBody(elem)
	c.expandText = savedExpandText
	if err != nil {
		return err
	}

	// XTSE0020: xsl:param inside xsl:function must not have @required attribute
	// (even required="no" is disallowed by the spec)
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() == NSXSLT && childElem.LocalName() == "param" {
			if _, hasReq := childElem.GetAttribute("required"); hasReq {
				pname := getAttr(childElem, "name")
				return staticError(errCodeXTSE0020,
					"xsl:param %q in xsl:function must not have @required", pname)
			}
		}
	}

	fnAs := getAttr(elem, "as")
	if err := c.validateAsSequenceType(fnAs, "xsl:function "+name); err != nil {
		return err
	}

	fn := &XSLFunction{
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
	c.stylesheet.functions[fk] = fn
	return nil
}

func (c *compiler) compileMode(elem *helium.Element) error {
	// xsl:mode must be empty (no children)
	if elem.FirstChild() != nil {
		return staticError(errCodeXTSE0010, "xsl:mode must be empty")
	}

	name := strings.TrimSpace(getAttr(elem, "name"))
	if name == "" {
		name = "#default"
	} else if name[0] != '#' {
		// Resolve QName to Clark notation so mode declarations and mode
		// references (on xsl:template/@mode, xsl:apply-templates/@mode)
		// use the same key format.
		name = c.resolveMode(name)
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
		case "text-only-copy", "shallow-copy", "deep-copy", "shallow-skip", "deep-skip", "fail":
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
		case "strict", "lax", "unspecified":
			// valid non-boolean values
		default:
			if _, ok := parseXSDBool(v); !ok {
				return staticError(errCodeXTSE0020, "invalid value %q for typed on xsl:mode", v)
			}
		}
	}

	visibility := strings.TrimSpace(getAttr(elem, "visibility"))
	// XTSE0020: unnamed mode cannot have visibility="public" or "final"
	if name == "#default" && (visibility == visPublic || visibility == visFinal) {
		return staticError(errCodeXTSE0020, "unnamed mode cannot have visibility %q", visibility)
	}

	onMultipleMatch := strings.TrimSpace(getAttr(elem, "on-multiple-match"))
	if onMultipleMatch != "" {
		switch onMultipleMatch {
		case "use-last", "fail":
			// valid
		default:
			return staticError(errCodeXTSE0020, "invalid value %q for on-multiple-match on xsl:mode", onMultipleMatch)
		}
	}

	// Resolve accumulator QNames to expanded names for proper comparison
	// across modules with different namespace prefixes.
	rawUA := getAttr(elem, "use-accumulators")
	var resolvedParts []string
	for _, tok := range strings.Fields(rawUA) {
		if tok == "#all" {
			resolvedParts = append(resolvedParts, tok)
		} else {
			resolvedParts = append(resolvedParts, resolveQName(tok, c.nsBindings))
		}
	}
	useAccumulators := strings.Join(resolvedParts, " ")

	typed := getAttr(elem, "typed")

	md := &ModeDef{
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
		c.stylesheet.modeDefs = make(map[string]*ModeDef)
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
			if existing.UseAccumulators != "" && md.UseAccumulators != "" && !sameAccumulatorSet(existing.UseAccumulators, md.UseAccumulators) {
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
			if md.UseAccumulators != "" {
				existing.UseAccumulators = md.UseAccumulators
			}
			return nil
		}
		// Different precedence: higher precedence wins.
		if c.importPrec > existing.ImportPrec {
			// Preserve conflict flags from the lower-precedence entry,
			// but clear them for attributes the higher-prec explicitly specifies.
			md.conflictStreamable = existing.conflictStreamable && streamableStr == ""
			md.conflictOnNoMatch = existing.conflictOnNoMatch && md.OnNoMatch == ""
			md.conflictVisibility = existing.conflictVisibility && md.Visibility == ""
			md.conflictOnMultiple = existing.conflictOnMultiple && md.OnMultipleMatch == ""
			md.conflictAccumulator = existing.conflictAccumulator && md.UseAccumulators == ""
			c.stylesheet.modeDefs[name] = md
		}
		// Lower precedence than existing: existing already won, ignore this decl.
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
	for _, s := range strings.Fields(a) {
		as[s] = struct{}{}
	}
	bs := make(map[string]struct{})
	for _, s := range strings.Fields(b) {
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
