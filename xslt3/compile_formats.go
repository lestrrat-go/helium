package xslt3

import (
	"strings"
	"unicode"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func (c *compiler) compileCharacterMap(elem *helium.Element) error {
	saved := c.pushElementNamespaces(elem)
	defer func() { c.nsBindings = saved }()

	name := getAttr(elem, "name")
	if name == "" {
		return staticError(errCodeXTSE0110, "xsl:character-map requires name attribute")
	}
	name = resolveQName(name, c.nsBindings)

	cm := &CharacterMapDef{
		Name:     name,
		Mappings: make(map[rune]string),
	}

	if ucm := getAttr(elem, "use-character-maps"); ucm != "" {
		for _, n := range strings.Fields(ucm) {
			cm.UseCharacterMaps = append(cm.UseCharacterMaps, resolveQName(n, c.nsBindings))
		}
	}

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() != NSXSLT || childElem.LocalName() != "output-character" {
			continue
		}
		charAttr := getAttr(childElem, "character")
		if charAttr == "" {
			return staticError(errCodeXTSE0010,
				"xsl:output-character requires 'character' attribute")
		}
		r := firstRune(charAttr)
		str := getAttr(childElem, "string")
		cm.Mappings[r] = str
	}

	if c.stylesheet.characterMaps == nil {
		c.stylesheet.characterMaps = make(map[string]*CharacterMapDef)
	}
	// XTSE1580: duplicate character-map declarations with same name at same
	// import precedence. Imported character maps merge silently.
	if _, ok := c.stylesheet.characterMaps[name]; ok && !c.insideImport {
		return staticError(errCodeXTSE1580,
			"duplicate character-map declaration %q", name)
	}
	if existing, ok := c.stylesheet.characterMaps[name]; ok {
		for r, s := range cm.Mappings {
			existing.Mappings[r] = s
		}
		existing.UseCharacterMaps = append(existing.UseCharacterMaps, cm.UseCharacterMaps...)
	} else {
		c.stylesheet.characterMaps[name] = cm
	}
	return nil
}

func (c *compiler) compileKey(elem *helium.Element) error {
	// Collect local namespace declarations (e.g., xmlns:ex="..." on xsl:key)
	c.collectNamespaces(elem)

	name := getAttr(elem, "name")
	if name == "" {
		return staticError(errCodeXTSE0110, "xsl:key requires name attribute")
	}

	matchAttr := getAttr(elem, "match")
	if matchAttr == "" {
		return staticError(errCodeXTSE0110, "xsl:key requires match attribute")
	}

	matchPat, err := compilePattern(matchAttr, c.nsBindings, c.xpathDefaultNS)
	if err != nil {
		return err
	}

	expandedName := resolveQName(name, c.nsBindings)
	composite, _ := parseXSDBool(getAttr(elem, "composite"))

	// XTSE1222: if there is already a key with the same name, the composite
	// attribute must agree
	if existingDefs, ok := c.stylesheet.keys[expandedName]; ok && len(existingDefs) > 0 {
		if existingDefs[0].Composite != composite {
			return staticError(errCodeXTSE1222,
				"xsl:key declarations with name %q have conflicting composite attributes", expandedName)
		}
	}

	kd := &KeyDef{
		Name:      expandedName,
		Match:     matchPat,
		Composite: composite,
	}

	useAttr := getAttr(elem, "use")
	if useAttr != "" {
		// XTSE0010: xsl:key with use attribute must not have child elements
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Type() == helium.ElementNode {
				return staticError(errCodeXTSE0010,
					"xsl:key with use attribute must not have child elements")
			}
		}
		useExpr, err := compileXPath(useAttr, c.nsBindings)
		if err != nil {
			return err
		}
		kd.Use = useExpr
	} else {
		// XSLT 2.0+: key may use body content instead of use attribute.
		body, _, err := c.compileTemplateBody(elem)
		if err != nil {
			return err
		}
		kd.Body = body
	}

	c.stylesheet.keys[expandedName] = append(c.stylesheet.keys[expandedName], kd)
	return nil
}

func (c *compiler) compileOutput(elem *helium.Element) error {
	saved := c.pushElementNamespaces(elem)
	defer func() { c.nsBindings = saved }()

	name := getAttr(elem, "name")
	outDef := &OutputDef{
		Name:     name,
		Method:   strings.ToLower(getAttr(elem, "method")),
		Encoding: getAttr(elem, "encoding"),
		Version:  getAttr(elem, "version"),
	}

	if outDef.Method == "" {
		outDef.Method = "xml"
	}
	if outDef.Encoding == "" {
		outDef.Encoding = "UTF-8"
	}

	// Validate and parse boolean output attributes.
	// SEPM0016: invalid boolean values.
	if v := getAttr(elem, "indent"); v != "" {
		b, ok := parseXSDBool(v)
		if !ok {
			return staticError(errCodeSEPM0016, "%q is not a valid value for xsl:output/@indent", v)
		}
		outDef.Indent = b
	}
	if v := getAttr(elem, "omit-xml-declaration"); v != "" {
		b, ok := parseXSDBool(v)
		if !ok {
			return staticError(errCodeSEPM0016, "%q is not a valid value for xsl:output/@omit-xml-declaration", v)
		}
		outDef.OmitDeclaration = b
	}
	if v := getAttr(elem, "undeclare-prefixes"); v != "" {
		b, ok := parseXSDBool(v)
		if !ok {
			return staticError(errCodeSEPM0016, "%q is not a valid value for xsl:output/@undeclare-prefixes", v)
		}
		outDef.UndeclarePrefixes = b
	}
	// Validate other boolean output attributes.
	for _, boolAttr := range []string{"byte-order-mark", "escape-uri-attributes", "include-content-type"} {
		if v := getAttr(elem, boolAttr); v != "" {
			if _, ok := parseXSDBool(v); !ok {
				return staticError(errCodeSEPM0016, "%q is not a valid value for xsl:output/@%s", v, boolAttr)
			}
		}
	}
	// Validate standalone: must be "yes", "no", "omit", or boolean equivalents.
	if v := getAttr(elem, "standalone"); v != "" {
		v = strings.TrimSpace(v)
		switch v {
		case "yes", "no", "omit":
			// valid as-is
		case "true", "1":
			v = "yes"
		case "false", "0":
			v = "no"
		default:
			return staticError(errCodeSEPM0016, "%q is not a valid value for xsl:output/@standalone", v)
		}
		outDef.Standalone = v
	} else {
		outDef.Standalone = ""
	}
	outDef.DoctypePublic = getAttr(elem, "doctype-public")
	outDef.DoctypeSystem = getAttr(elem, "doctype-system")
	outDef.MediaType = getAttr(elem, "media-type")

	cdataStr := getAttr(elem, "cdata-section-elements")
	if cdataStr != "" {
		outDef.CDATASections = strings.Fields(cdataStr)
	}

	if is := getAttr(elem, "item-separator"); is != "" {
		outDef.ItemSeparator = &is
	}

	if ucm := getAttr(elem, "use-character-maps"); ucm != "" {
		for _, n := range strings.Fields(ucm) {
			outDef.UseCharacterMaps = append(outDef.UseCharacterMaps, resolveQName(n, c.nsBindings))
		}
	}

	// Per XSLT 3.0 §9.6.1: multiple xsl:output declarations with the same
	// name are merged. use-character-maps values accumulate; other attributes
	// are taken from the later declaration when explicitly specified.
	if existing, ok := c.stylesheet.outputs[name]; ok {
		// Accumulate use-character-maps from both declarations.
		if len(existing.UseCharacterMaps) > 0 {
			outDef.UseCharacterMaps = append(existing.UseCharacterMaps, outDef.UseCharacterMaps...)
		}
		// Preserve fields from earlier declaration when not explicitly set here.
		if getAttr(elem, "method") == "" {
			outDef.Method = existing.Method
		}
		if getAttr(elem, "encoding") == "" {
			outDef.Encoding = existing.Encoding
		}
		if getAttr(elem, "version") == "" {
			outDef.Version = existing.Version
		}
		if getAttr(elem, "indent") == "" {
			outDef.Indent = existing.Indent
		}
		if getAttr(elem, "omit-xml-declaration") == "" {
			outDef.OmitDeclaration = existing.OmitDeclaration
		}
		if getAttr(elem, "standalone") == "" {
			outDef.Standalone = existing.Standalone
		}
		if getAttr(elem, "undeclare-prefixes") == "" {
			outDef.UndeclarePrefixes = existing.UndeclarePrefixes
		}
		if getAttr(elem, "doctype-public") == "" {
			outDef.DoctypePublic = existing.DoctypePublic
		}
		if getAttr(elem, "doctype-system") == "" {
			outDef.DoctypeSystem = existing.DoctypeSystem
		}
		if getAttr(elem, "media-type") == "" {
			outDef.MediaType = existing.MediaType
		}
		if getAttr(elem, "cdata-section-elements") == "" {
			outDef.CDATASections = existing.CDATASections
		}
		if getAttr(elem, "item-separator") == "" {
			outDef.ItemSeparator = existing.ItemSeparator
		}
		if outDef.IncludeContentType == nil {
			outDef.IncludeContentType = existing.IncludeContentType
		}
	}

	c.stylesheet.outputs[name] = outDef
	return nil
}

func (c *compiler) compileAttributeSet(elem *helium.Element) error {
	name := getAttr(elem, "name")
	if name == "" {
		return staticError(errCodeXTSE0110, "xsl:attribute-set requires name attribute")
	}
	name = resolveQName(name, c.nsBindings)

	asd := &AttributeSetDef{
		Name:       name,
		Visibility: getAttr(elem, "visibility"),
	}

	if uas := getAttr(elem, "use-attribute-sets"); uas != "" {
		for _, n := range strings.Fields(uas) {
			asd.UseAttrSets = append(asd.UseAttrSets, resolveQName(n, c.nsBindings))
		}
	}

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() == NSXSLT && childElem.LocalName() == "attribute" {
			inst, err := c.compileAttribute(childElem)
			if err != nil {
				return err
			}
			asd.Attrs = append(asd.Attrs, inst)
		}
	}

	if c.stylesheet.attributeSets == nil {
		c.stylesheet.attributeSets = make(map[string]*AttributeSetDef)
	}
	// Merge same-named attribute-sets (XSLT spec: union of all definitions)
	if existing, ok := c.stylesheet.attributeSets[name]; ok {
		existing.Attrs = append(existing.Attrs, asd.Attrs...)
		existing.UseAttrSets = append(existing.UseAttrSets, asd.UseAttrSets...)
	} else {
		c.stylesheet.attributeSets[name] = asd
	}
	return nil
}

// checkAttributeSetCycles detects cyclic use-attribute-sets references (XTSE0720).
// Must be called after all attribute-set definitions have been compiled and merged.
func checkAttributeSetCycles(ss *Stylesheet) error {
	const (
		unvisited = 0
		visiting  = 1
		done      = 2
	)
	state := make(map[string]int, len(ss.attributeSets))

	var visit func(name string) error
	visit = func(name string) error {
		switch state[name] {
		case done:
			return nil
		case visiting:
			return staticError(errCodeXTSE0720,
				"attribute-set %q has a circular use-attribute-sets reference", name)
		}
		state[name] = visiting
		if asd := ss.attributeSets[name]; asd != nil {
			for _, ref := range asd.UseAttrSets {
				if _, ok := ss.attributeSets[ref]; !ok {
					continue // unknown ref — not our concern here
				}
				if err := visit(ref); err != nil {
					return err
				}
			}
		}
		state[name] = done
		return nil
	}

	for name := range ss.attributeSets {
		if err := visit(name); err != nil {
			return err
		}
	}
	return nil
}

func (c *compiler) compileDecimalFormat(elem *helium.Element) error {
	// Push element-local namespace declarations so prefixed names resolve correctly
	saved := c.pushElementNamespaces(elem)
	defer func() { c.nsBindings = saved }()

	name := getAttr(elem, "name")
	qn := xpath3.QualifiedName{}
	if name != "" {
		qn = xpath3.QualifiedName{Name: name}
		if idx := strings.IndexByte(name, ':'); idx >= 0 {
			prefix := name[:idx]
			local := name[idx+1:]
			if uri, ok := c.nsBindings[prefix]; ok {
				qn = xpath3.QualifiedName{URI: uri, Name: local}
			} else {
				qn.Name = local
			}
		}
	}

	if c.stylesheet.decimalFormats == nil {
		c.stylesheet.decimalFormats = make(map[xpath3.QualifiedName]xpath3.DecimalFormat)
		c.stylesheet.decimalFmtPrec = make(map[xpath3.QualifiedName]int)
		c.stylesheet.decimalFmtSet = make(map[xpath3.QualifiedName]map[string]struct{})
	}

	// XTSE1295: zero-digit must be a Unicode character with digit value zero
	if v := getAttr(elem, "zero-digit"); v != "" {
		zd := firstRune(v)
		if !unicode.IsDigit(zd) || !isDigitZero(zd) {
			return staticError(errCodeXTSE1295,
				"zero-digit %q is not a Unicode digit-zero character", string(zd))
		}
	}

	// Get or create the merged format
	df, exists := c.stylesheet.decimalFormats[qn]
	existingPrec := c.stylesheet.decimalFmtPrec[qn]
	if !exists {
		df = xpath3.DefaultDecimalFormat()
	}

	// XTSE1290: record potential conflicts for deferred checking after all imports.
	// Higher precedence declarations can override lower-precedence conflicts.
	if exists && c.importPrec == existingPrec {
		prevSet := c.stylesheet.decimalFmtSet[qn]
		if err := checkDecimalFormatConflictExplicit(df, elem, prevSet); err != nil {
			// Record the conflict and its precedence for deferred checking
			if c.stylesheet.decimalFmtConflicts == nil {
				c.stylesheet.decimalFmtConflicts = make(map[xpath3.QualifiedName]int)
			}
			c.stylesheet.decimalFmtConflicts[qn] = c.importPrec
		}
	}

	// If current declaration has higher or equal import precedence, merge properties.
	// Lower precedence declarations' explicitly-set properties are inherited.
	if !exists || c.importPrec >= existingPrec {

		// Merge explicitly-set properties
		if v := getAttr(elem, "decimal-separator"); v != "" {
			df.DecimalSeparator = firstRune(v)
		}
		if v := getAttr(elem, "grouping-separator"); v != "" {
			df.GroupingSeparator = firstRune(v)
		}
		if v := getAttr(elem, "percent"); v != "" {
			df.Percent = firstRune(v)
		}
		if v := getAttr(elem, "per-mille"); v != "" {
			df.PerMille = firstRune(v)
		}
		if v := getAttr(elem, "zero-digit"); v != "" {
			df.ZeroDigit = firstRune(v)
		}
		if v := getAttr(elem, "digit"); v != "" {
			df.Digit = firstRune(v)
		}
		if v := getAttr(elem, "pattern-separator"); v != "" {
			df.PatternSeparator = firstRune(v)
		}
		if v := getAttr(elem, "exponent-separator"); v != "" {
			df.ExponentSeparator = firstRune(v)
		}
		if v := getAttr(elem, "infinity"); v != "" {
			df.Infinity = v
		}
		if v := getAttr(elem, "NaN"); v != "" {
			df.NaN = v
		}
		if v := getAttr(elem, "minus-sign"); v != "" {
			df.MinusSign = firstRune(v)
		}

		c.stylesheet.decimalFormats[qn] = df
		c.stylesheet.decimalFmtPrec[qn] = c.importPrec

		// Track which properties were explicitly set
		if c.stylesheet.decimalFmtSet[qn] == nil {
			c.stylesheet.decimalFmtSet[qn] = make(map[string]struct{})
		}
		// If higher precedence, clear previously tracked properties
		if exists && c.importPrec > existingPrec {
			c.stylesheet.decimalFmtSet[qn] = make(map[string]struct{})
		}
		setProps := c.stylesheet.decimalFmtSet[qn]
		for _, attr := range []string{"decimal-separator", "grouping-separator", "percent",
			"per-mille", "zero-digit", "digit", "pattern-separator", "exponent-separator",
			"infinity", "NaN", "minus-sign"} {
			if getAttr(elem, attr) != "" {
				setProps[attr] = struct{}{}
			}
		}
	}
	return nil
}

// isDigitZero returns true if r is a Unicode digit with numeric value 0.
func isDigitZero(r rune) bool {
	// Unicode digit blocks are contiguous runs of 10 codepoints.
	// The zero digit is at a position where (r - block_start) == 0.
	// We check that r-1 is NOT a digit (meaning r is the first in its block).
	return r == '0' || (r > '0' && unicode.IsDigit(r) && !unicode.IsDigit(r-1))
}

// checkDecimalFormatCharConflicts raises XTSE1300 if any two formatting characters
// in the decimal format are the same.
func checkDecimalFormatCharConflicts(df xpath3.DecimalFormat) error {
	// Collect all formatting characters and check for duplicates
	type charRole struct {
		char rune
		role string
	}
	roles := []charRole{
		{df.DecimalSeparator, "decimal-separator"},
		{df.GroupingSeparator, "grouping-separator"},
		{df.Percent, "percent"},
		{df.PerMille, "per-mille"},
		{df.ZeroDigit, "zero-digit"},
		{df.Digit, "digit"},
		{df.PatternSeparator, "pattern-separator"},
		{df.ExponentSeparator, "exponent-separator"},
	}
	for i := 0; i < len(roles); i++ {
		for j := i + 1; j < len(roles); j++ {
			if roles[i].char == roles[j].char {
				return staticError(errCodeXTSE1300,
					"%s and %s use the same character %q",
					roles[i].role, roles[j].role, string(roles[i].char))
			}
		}
	}
	return nil
}

// checkDecimalFormatConflictExplicit raises XTSE1290 if any property
// explicitly set in elem conflicts with a property explicitly set in a prior
// declaration of the same name at the same import precedence.
func checkDecimalFormatConflictExplicit(existing xpath3.DecimalFormat, elem *helium.Element, prevSet map[string]struct{}) error {
	type runeCheck struct {
		attr string
		cur  rune
	}
	runeChecks := []runeCheck{
		{"decimal-separator", existing.DecimalSeparator},
		{"grouping-separator", existing.GroupingSeparator},
		{"percent", existing.Percent},
		{"per-mille", existing.PerMille},
		{"zero-digit", existing.ZeroDigit},
		{"digit", existing.Digit},
		{"pattern-separator", existing.PatternSeparator},
		{"exponent-separator", existing.ExponentSeparator},
		{"minus-sign", existing.MinusSign},
	}
	for _, rc := range runeChecks {
		v := getAttr(elem, rc.attr)
		if v == "" {
			continue
		}
		// Only conflict if the property was explicitly set in a prior declaration
		if _, wasSet := prevSet[rc.attr]; !wasSet {
			continue
		}
		newVal := firstRune(v)
		if newVal != rc.cur {
			return staticError(errCodeXTSE1290,
				"conflicting xsl:decimal-format declarations: %s was %q, now %q",
				rc.attr, string(rc.cur), string(newVal))
		}
	}
	// String properties
	if v := getAttr(elem, "infinity"); v != "" {
		if _, wasSet := prevSet["infinity"]; wasSet && v != existing.Infinity {
			return staticError(errCodeXTSE1290,
				"conflicting xsl:decimal-format declarations: infinity was %q, now %q",
				existing.Infinity, v)
		}
	}
	if v := getAttr(elem, "NaN"); v != "" {
		if _, wasSet := prevSet["NaN"]; wasSet && v != existing.NaN {
			return staticError(errCodeXTSE1290,
				"conflicting xsl:decimal-format declarations: NaN was %q, now %q",
				existing.NaN, v)
		}
	}
	return nil
}

func firstRune(s string) rune {
	for _, r := range s {
		return r
	}
	return 0
}

func (c *compiler) compileSpaceHandling(elem *helium.Element, strip bool) {
	elements := getAttr(elem, "elements")
	if elements == "" {
		return
	}

	for _, name := range strings.Fields(elements) {
		nt := NameTest{}
		// Handle EQName syntax: Q{uri}local
		if strings.HasPrefix(name, "Q{") {
			if closeIdx := strings.IndexByte(name, '}'); closeIdx > 0 {
				nt.Prefix = "Q{" + name[2:closeIdx] + "}"
				nt.Local = name[closeIdx+1:]
			} else {
				nt.Local = name
			}
		} else if idx := strings.IndexByte(name, ':'); idx >= 0 {
			nt.Prefix = name[:idx]
			nt.Local = name[idx+1:]
		} else {
			nt.Local = name
		}
		if strip {
			c.stylesheet.stripSpace = append(c.stylesheet.stripSpace, nt)
		} else {
			c.stylesheet.preserveSpace = append(c.stylesheet.preserveSpace, nt)
		}
	}
}

// nameTestKey returns a canonical key for a NameTest by resolving the prefix
// to a URI. EQName prefixes of the form "Q{uri}" are used as-is.
func (c *compiler) nameTestKey(nt NameTest) string {
	uri := ""
	if strings.HasPrefix(nt.Prefix, "Q{") {
		uri = nt.Prefix[2 : len(nt.Prefix)-1]
	} else if nt.Prefix != "" {
		uri = c.nsBindings[nt.Prefix]
	}
	return "{" + uri + "}" + nt.Local
}

// checkSpaceConflicts detects NameTests that appear in both xsl:strip-space
// and xsl:preserve-space at the same import precedence (XTSE0270).
func (c *compiler) checkSpaceConflicts() error {
	if len(c.stylesheet.stripSpace) == 0 || len(c.stylesheet.preserveSpace) == 0 {
		return nil
	}
	stripKeys := make(map[string]struct{}, len(c.stylesheet.stripSpace))
	for _, nt := range c.stylesheet.stripSpace {
		stripKeys[c.nameTestKey(nt)] = struct{}{}
	}
	for _, nt := range c.stylesheet.preserveSpace {
		key := c.nameTestKey(nt)
		if _, ok := stripKeys[key]; ok {
			return staticError(errCodeXTSE0270,
				"conflicting xsl:strip-space and xsl:preserve-space for %q", key)
		}
	}
	return nil
}
