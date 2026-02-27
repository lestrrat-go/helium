package relaxng

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

const rngNS = "http://relaxng.org/ns/structure/1.0"
const xsdDatatypeLibrary = "http://www.w3.org/2001/XMLSchema-datatypes"

// compiler holds state during schema compilation.
type compiler struct {
	grammar      *Grammar
	baseDir      string
	filename     string
	errors       strings.Builder
	warnings     strings.Builder
	includeStack []string // recursion guard for include/externalRef
	includeLimit int
	// grammarStack tracks nested grammars for parentRef resolution.
	grammarStack []*grammarScope
}

// grammarScope represents a grammar level during compilation.
type grammarScope struct {
	defines map[string]*defineEntry
}

// defineEntry tracks a define and its combine mode.
type defineEntry struct {
	pattern *pattern
	combine string // "choice" or "interleave", or "" for none
}

func compileSchema(doc *helium.Document, baseDir string, cfg *compileConfig) (*Grammar, error) {
	c := &compiler{
		grammar: &Grammar{
			defines: make(map[string]*pattern),
		},
		baseDir:      baseDir,
		filename:     cfg.filename,
		includeLimit: 1000,
	}

	root := findDocumentElement(doc)
	if root == nil {
		c.errors.WriteString(rngParserError("xmlRelaxNGParse: could not find root element"))
		c.grammar.compileErrors = c.errors.String()
		return c.grammar, nil
	}

	c.pushGrammar()

	var startPat *pattern
	if isRNG(root, "grammar") {
		c.parseGrammarContent(root)
		startPat = c.resolveStart()
	} else {
		// Bare pattern (e.g. <element> at root)
		startPat = c.parsePattern(root)
	}

	c.resolveRefs()
	c.checkRefCycles()
	c.popGrammar()

	c.grammar.start = startPat
	c.grammar.compileErrors = c.errors.String()
	c.grammar.compileWarnings = c.warnings.String()

	return c.grammar, nil
}

func (c *compiler) pushGrammar() {
	c.grammarStack = append(c.grammarStack, &grammarScope{
		defines: make(map[string]*defineEntry),
	})
}

func (c *compiler) popGrammar() {
	if len(c.grammarStack) > 0 {
		c.grammarStack = c.grammarStack[:len(c.grammarStack)-1]
	}
}

func (c *compiler) currentGrammar() *grammarScope {
	if len(c.grammarStack) == 0 {
		return nil
	}
	return c.grammarStack[len(c.grammarStack)-1]
}

func (c *compiler) resolveStart() *pattern {
	g := c.currentGrammar()
	if g == nil {
		return nil
	}
	entry, ok := g.defines["##start"]
	if !ok {
		return nil
	}
	return entry.pattern
}

func (c *compiler) resolveRefs() {
	g := c.currentGrammar()
	if g == nil {
		return
	}
	// Copy defines to grammar for validation-time lookup.
	for name, entry := range g.defines {
		if name != "##start" {
			c.grammar.defines[name] = entry.pattern
		}
	}
}

// checkRefCycles detects cycles in inline references (refs that lead back to
// the same define without passing through an element pattern).
func (c *compiler) checkRefCycles() {
	for name := range c.grammar.defines {
		visiting := map[string]bool{name: true}
		if ref := c.findCycleInPattern(c.grammar.defines[name], visiting); ref != nil {
			c.errors.WriteString(rngParserErrorAt(c.filename, ref.line, "ref",
				fmt.Sprintf("Detected a cycle in %s references", ref.name)))
			return
		}
	}
}

// findCycleInPattern walks a pattern tree looking for ref cycles.
// Element patterns break the chain (refs inside elements don't create content cycles).
// Returns the offending ref pattern if a cycle is found.
func (c *compiler) findCycleInPattern(pat *pattern, visiting map[string]bool) *pattern {
	if pat == nil {
		return nil
	}
	switch pat.kind {
	case patternElement:
		// Elements break the cycle chain — don't recurse into content
		return nil
	case patternRef, patternParentRef:
		if visiting[pat.name] {
			return pat
		}
		def, ok := c.grammar.defines[pat.name]
		if !ok {
			return nil
		}
		visiting[pat.name] = true
		result := c.findCycleInPattern(def, visiting)
		delete(visiting, pat.name)
		return result
	default:
		for _, child := range pat.children {
			if ref := c.findCycleInPattern(child, visiting); ref != nil {
				return ref
			}
		}
		// Also check attrs
		for _, attr := range pat.attrs {
			if ref := c.findCycleInPattern(attr, visiting); ref != nil {
				return ref
			}
		}
		return nil
	}
}

// parseGrammarContent parses children of a <grammar> element.
func (c *compiler) parseGrammarContent(grammarElem *helium.Element) {
	for child := grammarElem.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if !isRNGElement(elem) {
			continue
		}

		switch elem.LocalName() {
		case "start":
			c.parseStart(elem)
		case "define":
			c.parseDefine(elem)
		case "include":
			c.parseInclude(elem)
		case "div":
			// <div> is transparent — recurse into its children
			c.parseGrammarContent(elem)
		}
	}
}

// parseStart parses a <start> element.
func (c *compiler) parseStart(elem *helium.Element) {
	combine := getAttr(elem, "combine")
	pat := c.parseChildren(elem)
	if pat == nil {
		return
	}

	g := c.currentGrammar()
	existing, ok := g.defines["##start"]
	if !ok {
		g.defines["##start"] = &defineEntry{pattern: pat, combine: combine}
		return
	}

	// Multiple <start> — combine
	combineMode := combine
	if combineMode == "" {
		combineMode = existing.combine
	}
	existing.pattern = combinePatterns(existing.pattern, pat, combineMode)
	if combineMode != "" {
		existing.combine = combineMode
	}
}

// parseDefine parses a <define> element.
func (c *compiler) parseDefine(elem *helium.Element) {
	name := getAttr(elem, "name")
	if name == "" {
		return
	}
	combine := getAttr(elem, "combine")
	pat := c.parseChildren(elem)
	if pat == nil {
		pat = &pattern{kind: patternEmpty}
	}

	g := c.currentGrammar()
	existing, ok := g.defines[name]
	if !ok {
		g.defines[name] = &defineEntry{pattern: pat, combine: combine}
		return
	}

	// Multiple <define> with same name — combine
	combineMode := combine
	if combineMode == "" {
		combineMode = existing.combine
	}
	existing.pattern = combinePatterns(existing.pattern, pat, combineMode)
	if combineMode != "" {
		existing.combine = combineMode
	}
}

// combinePatterns combines two patterns with the given combine mode.
func combinePatterns(existing, incoming *pattern, mode string) *pattern {
	switch mode {
	case "interleave":
		return &pattern{
			kind:     patternInterleave,
			children: []*pattern{existing, incoming},
		}
	case "choice":
		return &pattern{
			kind:     patternChoice,
			children: []*pattern{existing, incoming},
		}
	default:
		// Default to group if no combine specified
		return &pattern{
			kind:     patternGroup,
			children: []*pattern{existing, incoming},
		}
	}
}

// parseInclude parses an <include> element.
func (c *compiler) parseInclude(elem *helium.Element) {
	href := getAttr(elem, "href")
	if href == "" {
		c.errors.WriteString(rngParserError("include has no href attribute"))
		return
	}

	// Resolve path
	path := href
	if !filepath.IsAbs(path) && c.baseDir != "" {
		path = filepath.Join(c.baseDir, path)
	}

	// Recursion guard
	for _, p := range c.includeStack {
		if p == path {
			c.errors.WriteString(rngParserError(fmt.Sprintf("Detected an Include recursion for %s", href)))
			return
		}
	}
	if len(c.includeStack) >= c.includeLimit {
		c.errors.WriteString(rngParserError("Include limit reached"))
		return
	}

	c.includeStack = append(c.includeStack, path)
	defer func() { c.includeStack = c.includeStack[:len(c.includeStack)-1] }()

	// Read and parse the included file
	data, err := os.ReadFile(path)
	if err != nil {
		c.errors.WriteString(rngParserError(fmt.Sprintf("xmlRelaxNGParse: could not load %s", href)))
		return
	}
	doc, err := helium.Parse(data)
	if err != nil {
		c.errors.WriteString(rngParserError(fmt.Sprintf("xmlRelaxNGParse: could not load %s", href)))
		return
	}

	root := findDocumentElement(doc)
	if root == nil || !isRNG(root, "grammar") {
		c.errors.WriteString(rngParserError(fmt.Sprintf("xmlRelaxNGParse: included file %s is not a grammar", href)))
		return
	}

	// Parse the included grammar content
	oldBaseDir := c.baseDir
	c.baseDir = filepath.Dir(path)
	c.parseGrammarContent(root)
	c.baseDir = oldBaseDir

	// Collect override names from <include> children, then delete them from
	// the current grammar scope so the overrides replace (not combine with)
	// the included definitions.
	overrideNames := c.collectOverrideNames(elem)
	g := c.currentGrammar()
	if g != nil {
		for _, name := range overrideNames {
			delete(g.defines, name)
		}
	}

	// Process overrides from the <include> element's children
	c.parseIncludeOverrides(elem)
}

// collectOverrideNames returns the define/start names overridden by an include element.
func (c *compiler) collectOverrideNames(elem *helium.Element) []string {
	var names []string
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if !isRNGElement(childElem) {
			continue
		}
		switch childElem.LocalName() {
		case "start":
			names = append(names, "##start")
		case "define":
			name := getAttr(childElem, "name")
			if name != "" {
				names = append(names, name)
			}
		case "div":
			names = append(names, c.collectOverrideNames(childElem)...)
		}
	}
	return names
}

// parseIncludeOverrides processes override children of an <include> element.
func (c *compiler) parseIncludeOverrides(elem *helium.Element) {
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if !isRNGElement(childElem) {
			continue
		}
		switch childElem.LocalName() {
		case "start":
			c.parseStart(childElem)
		case "define":
			c.parseDefine(childElem)
		case "div":
			c.parseIncludeOverrides(childElem)
		}
	}
}

// parsePattern parses a single RNG pattern element.
func (c *compiler) parsePattern(node *helium.Element) *pattern {
	if node == nil || !isRNGElement(node) {
		return nil
	}

	switch node.LocalName() {
	case "element":
		return c.parseElement(node)
	case "attribute":
		return c.parseAttribute(node)
	case "empty":
		return &pattern{kind: patternEmpty, line: node.Line()}
	case "text":
		return &pattern{kind: patternText, line: node.Line()}
	case "notAllowed":
		return &pattern{kind: patternNotAllowed, line: node.Line()}
	case "zeroOrMore":
		p := &pattern{kind: patternZeroOrMore, line: node.Line()}
		p.children = c.parsePatternChildren(node)
		return p
	case "oneOrMore":
		p := &pattern{kind: patternOneOrMore, line: node.Line()}
		p.children = c.parsePatternChildren(node)
		return p
	case "optional":
		p := &pattern{kind: patternOptional, line: node.Line()}
		p.children = c.parsePatternChildren(node)
		return p
	case "choice":
		p := &pattern{kind: patternChoice, line: node.Line()}
		p.children = c.parsePatternChildren(node)
		return p
	case "group":
		p := &pattern{kind: patternGroup, line: node.Line()}
		p.children = c.parsePatternChildren(node)
		return p
	case "interleave":
		p := &pattern{kind: patternInterleave, line: node.Line()}
		p.children = c.parsePatternChildren(node)
		return p
	case "mixed":
		// mixed is interleave with text
		p := &pattern{kind: patternInterleave, line: node.Line()}
		children := c.parsePatternChildren(node)
		textPat := &pattern{kind: patternText}
		if len(children) > 1 {
			groupPat := &pattern{kind: patternGroup, children: children}
			p.children = []*pattern{textPat, groupPat}
		} else if len(children) == 1 {
			p.children = []*pattern{textPat, children[0]}
		} else {
			p.children = []*pattern{textPat}
		}
		return p
	case "ref":
		name := getAttr(node, "name")
		if name == "" {
			c.errors.WriteString(rngParserError("ref has no name"))
			return nil
		}
		return &pattern{kind: patternRef, name: name, line: node.Line()}
	case "parentRef":
		name := getAttr(node, "name")
		if name == "" {
			c.errors.WriteString(rngParserError("parentRef has no name"))
			return nil
		}
		return &pattern{kind: patternParentRef, name: name, line: node.Line()}
	case "externalRef":
		return c.parseExternalRef(node)
	case "data":
		return c.parseData(node)
	case "value":
		return c.parseValue(node)
	case "list":
		p := &pattern{kind: patternList, line: node.Line()}
		p.children = c.parsePatternChildren(node)
		return p
	case "grammar":
		// Nested grammar
		c.pushGrammar()
		c.parseGrammarContent(node)
		startPat := c.resolveStart()
		c.resolveRefs()
		c.popGrammar()
		return startPat
	default:
		return nil
	}
}

// parseElement parses an <element> pattern.
func (c *compiler) parseElement(node *helium.Element) *pattern {
	p := &pattern{kind: patternElement, line: node.Line()}

	name := getAttr(node, "name")
	if name != "" {
		localName, ns := resolveQName(node, name)
		p.nameClass = &nameClass{kind: ncName, name: localName, ns: ns}
		p.name = localName
		p.ns = ns
	}

	// Parse children: first child may be name class, rest are content patterns
	var contentChildren []*pattern
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if !isRNGElement(elem) {
			continue
		}

		// Check if this is a name class element
		if p.nameClass == nil && isNameClassElement(elem) {
			p.nameClass = c.parseNameClass(elem)
			if p.nameClass != nil && p.nameClass.kind == ncName {
				p.name = p.nameClass.name
				p.ns = p.nameClass.ns
			}
			continue
		}

		// Otherwise it's a content pattern
		pat := c.parsePattern(elem)
		if pat != nil {
			if pat.kind == patternAttribute {
				p.attrs = append(p.attrs, pat)
			} else {
				contentChildren = append(contentChildren, pat)
			}
		}
	}

	// Check: attribute name class conflicts
	allAttrs := collectAttrPatternsFlat(append(p.attrs, contentChildren...))
	for i := 0; i < len(allAttrs); i++ {
		for j := i + 1; j < len(allAttrs); j++ {
			if nameClassesOverlap(allAttrs[i].nameClass, allAttrs[j].nameClass) {
				c.addSchemaError(node, "Attributes conflicts in group")
				goto attrConflictDone
			}
		}
	}
attrConflictDone:

	// Check: element with no content (no children, no attrs)
	if len(contentChildren) == 0 && len(p.attrs) == 0 {
		c.addSchemaError(node, "xmlRelaxNGParseElement: element has no content")
	}

	// Check: content type error (mixing data/value with element/group patterns)
	if hasDataContent(contentChildren) && hasElementContent(contentChildren) {
		eName := p.name
		if eName == "" && p.nameClass != nil {
			eName = p.nameClass.name
		}
		c.addSchemaError(node, fmt.Sprintf("Element %s has a content type error", eName))
	}

	// Wrap content
	if len(contentChildren) > 1 {
		p.children = []*pattern{{kind: patternGroup, children: contentChildren}}
	} else if len(contentChildren) == 1 {
		p.children = contentChildren
	}

	return p
}

// parseAttribute parses an <attribute> pattern.
func (c *compiler) parseAttribute(node *helium.Element) *pattern {
	p := &pattern{kind: patternAttribute, line: node.Line()}

	name := getAttr(node, "name")
	if name != "" {
		localName, ns := resolveQName(node, name)
		// For attributes, the default namespace is "" (not inherited),
		// unless an explicit ns attribute is provided.
		if !strings.Contains(name, ":") {
			ns = getAttr(node, "ns")
		}
		p.nameClass = &nameClass{kind: ncName, name: localName, ns: ns}
		p.name = localName
		p.ns = ns
	}

	// Parse children
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if !isRNGElement(elem) {
			continue
		}

		if p.nameClass == nil && isNameClassElement(elem) {
			p.nameClass = c.parseNameClass(elem)
			if p.nameClass != nil && p.nameClass.kind == ncName {
				p.name = p.nameClass.name
				p.ns = p.nameClass.ns
			}
			continue
		}

		pat := c.parsePattern(elem)
		if pat != nil {
			p.children = append(p.children, pat)
		}
	}

	// If no content specified, attribute has <text/> content by default
	if len(p.children) == 0 {
		p.children = []*pattern{{kind: patternText}}
	}

	return p
}

// parseData parses a <data> pattern.
func (c *compiler) parseData(node *helium.Element) *pattern {
	p := &pattern{kind: patternData, line: node.Line()}

	typeName := getAttr(node, "type")
	library := getDatatypeLibrary(node)

	p.dataType = &dataType{
		library: library,
		name:    typeName,
	}

	// Parse <param> and <except> children
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if !isRNGElement(elem) {
			continue
		}

		switch elem.LocalName() {
		case "param":
			paramName := getAttr(elem, "name")
			paramValue := textContent(elem)
			p.params = append(p.params, &param{name: paramName, value: paramValue})
		case "except":
			excPats := c.parsePatternChildren(elem)
			if len(excPats) > 0 {
				except := &pattern{kind: patternChoice, children: excPats}
				p.children = append(p.children, except)
			}
		}
	}

	return p
}

// parseValue parses a <value> pattern.
func (c *compiler) parseValue(node *helium.Element) *pattern {
	p := &pattern{kind: patternValue, line: node.Line()}

	typeName := getAttr(node, "type")
	library := getDatatypeLibrary(node)

	if typeName == "" {
		// Default type is "token" from the built-in library
		typeName = "token"
		library = ""
	}

	p.dataType = &dataType{
		library: library,
		name:    typeName,
	}
	p.value = textContent(node)

	// Resolve namespace for the value
	p.ns = getAttr(node, "ns")

	return p
}

// parseExternalRef parses an <externalRef> pattern.
func (c *compiler) parseExternalRef(node *helium.Element) *pattern {
	href := getAttr(node, "href")
	if href == "" {
		c.errors.WriteString(rngParserError("externalRef has no href attribute"))
		return nil
	}

	path := href
	if !filepath.IsAbs(path) && c.baseDir != "" {
		path = filepath.Join(c.baseDir, path)
	}

	// Recursion guard
	for _, p := range c.includeStack {
		if p == path {
			c.errors.WriteString(rngParserError(fmt.Sprintf("Detected an externalRef recursion for %s", href)))
			return nil
		}
	}
	if len(c.includeStack) >= c.includeLimit {
		c.errors.WriteString(rngParserError("Include limit reached"))
		return nil
	}

	c.includeStack = append(c.includeStack, path)
	defer func() { c.includeStack = c.includeStack[:len(c.includeStack)-1] }()

	data, err := os.ReadFile(path)
	if err != nil {
		c.errors.WriteString(rngParserError(fmt.Sprintf("xmlRelaxNGParse: could not load %s", href)))
		return nil
	}
	doc, err := helium.Parse(data)
	if err != nil {
		c.errors.WriteString(rngParserError(fmt.Sprintf("xmlRelaxNGParse: could not load %s", href)))
		return nil
	}

	root := findDocumentElement(doc)
	if root == nil {
		c.errors.WriteString(rngParserError(fmt.Sprintf("xmlRelaxNGParse: external ref %s has no root", href)))
		return nil
	}

	oldBaseDir := c.baseDir
	c.baseDir = filepath.Dir(path)
	var result *pattern
	if isRNG(root, "grammar") {
		c.pushGrammar()
		c.parseGrammarContent(root)
		result = c.resolveStart()
		c.resolveRefs()
		c.popGrammar()
	} else {
		result = c.parsePattern(root)
	}
	c.baseDir = oldBaseDir

	return result
}

// parseNameClass parses a name class element.
func (c *compiler) parseNameClass(node *helium.Element) *nameClass {
	switch node.LocalName() {
	case "name":
		qname := textContent(node)
		ns, hasNS := getAttrOpt(node, "ns")
		name := qname
		if strings.Contains(qname, ":") {
			localName, resolvedNS := resolveQName(node, qname)
			name = localName
			if !hasNS {
				ns = resolvedNS
			}
		} else if !hasNS {
			ns = getAncestorNS(node)
		}
		return &nameClass{kind: ncName, name: name, ns: ns}
	case "anyName":
		nc := &nameClass{kind: ncAnyName}
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			elem, ok := child.(*helium.Element)
			if !ok {
				continue
			}
			if isRNG(elem, "except") {
				nc.except = c.parseNameClassChildren(elem)
			}
		}
		return nc
	case "nsName":
		ns, hasNS := getAttrOpt(node, "ns")
		if !hasNS {
			ns = getAncestorNS(node)
		}
		nc := &nameClass{kind: ncNsName, ns: ns}
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			elem, ok := child.(*helium.Element)
			if !ok {
				continue
			}
			if isRNG(elem, "except") {
				nc.except = c.parseNameClassChildren(elem)
			}
		}
		return nc
	case "choice":
		var classes []*nameClass
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			elem, ok := child.(*helium.Element)
			if !ok {
				continue
			}
			if !isRNGElement(elem) {
				continue
			}
			nc := c.parseNameClass(elem)
			if nc != nil {
				classes = append(classes, nc)
			}
		}
		return buildNameClassChoice(classes)
	}
	return nil
}

// parseNameClassChildren parses the children of an <except> element as name classes
// and returns them combined as a choice.
func (c *compiler) parseNameClassChildren(exceptElem *helium.Element) *nameClass {
	var classes []*nameClass
	for child := exceptElem.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if !isRNGElement(elem) {
			continue
		}
		nc := c.parseNameClass(elem)
		if nc != nil {
			classes = append(classes, nc)
		}
	}
	return buildNameClassChoice(classes)
}

func buildNameClassChoice(classes []*nameClass) *nameClass {
	if len(classes) == 0 {
		return nil
	}
	if len(classes) == 1 {
		return classes[0]
	}
	// Build a right-recursive choice tree
	result := classes[len(classes)-1]
	for i := len(classes) - 2; i >= 0; i-- {
		result = &nameClass{kind: ncChoice, left: classes[i], right: result}
	}
	return result
}

// parseChildren parses all pattern children and wraps them appropriately.
func (c *compiler) parseChildren(parent *helium.Element) *pattern {
	children := c.parsePatternChildren(parent)
	if len(children) == 0 {
		return nil
	}
	if len(children) == 1 {
		return children[0]
	}
	return &pattern{kind: patternGroup, children: children}
}

// parsePatternChildren parses all RNG pattern children of an element.
func (c *compiler) parsePatternChildren(parent *helium.Element) []*pattern {
	var result []*pattern
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if !isRNGElement(elem) {
			continue
		}
		pat := c.parsePattern(elem)
		if pat != nil {
			result = append(result, pat)
		}
	}
	return result
}

// Helper functions

func findDocumentElement(doc *helium.Document) *helium.Element {
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		if elem, ok := child.(*helium.Element); ok {
			return elem
		}
	}
	return nil
}

func elemNS(elem *helium.Element) string {
	ns := elem.Namespace()
	if ns == nil {
		return ""
	}
	return ns.URI()
}

func isRNGElement(elem *helium.Element) bool {
	ns := elemNS(elem)
	return ns == rngNS || ns == ""
}

func isRNG(elem *helium.Element, localName string) bool {
	return elem.LocalName() == localName && isRNGElement(elem)
}

func isNameClassElement(elem *helium.Element) bool {
	if !isRNGElement(elem) {
		return false
	}
	switch elem.LocalName() {
	case "name", "anyName", "nsName", "choice":
		return true
	}
	return false
}

func getAttr(elem *helium.Element, name string) string {
	for _, attr := range elem.Attributes() {
		if attr.LocalName() == name {
			return attr.Value()
		}
	}
	return ""
}

// getAttrOpt returns the value and presence of an attribute.
func getAttrOpt(elem *helium.Element, name string) (string, bool) {
	for _, attr := range elem.Attributes() {
		if attr.LocalName() == name {
			return attr.Value(), true
		}
	}
	return "", false
}

// getAncestorNS walks up the RNG element tree to find the ns attribute.
func getAncestorNS(node *helium.Element) string {
	current := node.Parent()
	for current != nil {
		if elem, ok := current.(*helium.Element); ok {
			ns := getAttr(elem, "ns")
			if ns != "" {
				return ns
			}
			current = elem.Parent()
		} else {
			break
		}
	}
	return ""
}

// getDatatypeLibrary walks up the tree to find the datatypeLibrary attribute.
func getDatatypeLibrary(node *helium.Element) string {
	// Check the element itself first
	lib := getAttr(node, "datatypeLibrary")
	if lib != "" {
		return lib
	}
	// Walk up parents
	current := node.Parent()
	for current != nil {
		if elem, ok := current.(*helium.Element); ok {
			lib = getAttr(elem, "datatypeLibrary")
			if lib != "" {
				return lib
			}
			current = elem.Parent()
		} else {
			break
		}
	}
	return ""
}

// addSchemaError records a schema compilation error.
func (c *compiler) addSchemaError(node *helium.Element, msg string) {
	line := node.Line()
	fmt.Fprintf(&c.errors, "%s:%d: element %s: Relax-NG parser error : %s\n",
		c.filename, line, node.LocalName(), msg)
}

// hasDataContent checks if any pattern in the slice is a data/value/list pattern.
func hasDataContent(pats []*pattern) bool {
	for _, p := range pats {
		switch p.kind {
		case patternData, patternValue, patternList:
			return true
		}
	}
	return false
}

// hasElementContent checks if any pattern in the slice contains an element pattern.
func hasElementContent(pats []*pattern) bool {
	for _, p := range pats {
		if containsElement(p) {
			return true
		}
	}
	return false
}

func containsElement(p *pattern) bool {
	if p == nil {
		return false
	}
	if p.kind == patternElement {
		return true
	}
	for _, child := range p.children {
		if containsElement(child) {
			return true
		}
	}
	return false
}

// resolveQName resolves a QName from a schema element, returning the local name
// and namespace URI. If the name contains a prefix (e.g. "ex1:bar1"), the prefix
// is resolved using the namespace declarations in scope on the schema element.
// If the name has no prefix, the ns is determined by the "ns" attribute (inherited).
func resolveQName(schemaElem *helium.Element, qname string) (localName, ns string) {
	if idx := strings.IndexByte(qname, ':'); idx >= 0 {
		prefix := qname[:idx]
		localName = qname[idx+1:]
		// Walk the schema element and ancestors to find the namespace URI for this prefix.
		var node helium.Node = schemaElem
		for node != nil {
			if el, ok := node.(*helium.Element); ok {
				for _, nsdecl := range el.Namespaces() {
					if nsdecl.Prefix() == prefix {
						return localName, nsdecl.URI()
					}
				}
				node = el.Parent()
			} else {
				break
			}
		}
		// The xml prefix is always bound to the XML namespace by definition.
		if prefix == helium.XMLPrefix {
			return localName, helium.XMLNamespace
		}
		// Prefix not found — return as-is
		return localName, ""
	}
	// No prefix — use the "ns" attribute (inherited from ancestors).
	return qname, getInheritedNS(schemaElem)
}

// getInheritedNS returns the ns attribute value, inheriting from ancestors.
func getInheritedNS(node *helium.Element) string {
	var current helium.Node = node
	for current != nil {
		if elem, ok := current.(*helium.Element); ok {
			for _, attr := range elem.Attributes() {
				if attr.LocalName() == "ns" && attr.Prefix() == "" {
					return attr.Value()
				}
			}
			current = elem.Parent()
		} else {
			break
		}
	}
	return ""
}

// textContent returns the text content of an element.
func textContent(elem *helium.Element) string {
	var sb strings.Builder
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if t, ok := child.(*helium.Text); ok {
			sb.Write(t.Content())
		}
	}
	return sb.String()
}
