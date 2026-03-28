package relaxng

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

// compiler holds state during schema compilation.
type compiler struct {
	grammar      *Grammar
	baseDir      string
	filename     string
	errorHandler helium.ErrorHandler
	errorCount   int
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
	pattern   *pattern
	combine   string // "choice" or "interleave", or "" for none
	noCombine int    // count of definitions with no combine attribute
}


func compileSchema(ctx context.Context, doc *helium.Document, baseDir string, cfg *compileConfig) (*Grammar, error) { //nolint:unparam // error always nil but callers check for future-proofing
	var eh helium.ErrorHandler = helium.NilErrorHandler{}
	if cfg.errorHandler != nil {
		eh = cfg.errorHandler
	}

	label := cfg.label
	if label == "" {
		label = doc.URL()
	}
	if label == "" {
		label = "(string)"
	}

	c := &compiler{
		grammar: &Grammar{
			defines: make(map[string]*pattern),
		},
		baseDir:      baseDir,
		filename:     label,
		errorHandler: eh,
		includeLimit: 1000,
	}

	root := findDocumentElement(doc)
	if root == nil {
		msg := rngParserError("xmlRelaxNGParse: could not find root element")
		c.errorHandler.Handle(ctx, helium.NewLeveledError(msg, helium.ErrorLevelFatal)) //nolint:contextcheck
		c.errorCount++
		return c.grammar, nil
	}

	c.pushGrammar(ctx)

	var startPat *pattern
	if isRNG(root, "grammar") {
		c.parseGrammarContent(ctx, root) //nolint:contextcheck
		startPat = c.resolveStart(ctx)
	} else {
		// Bare pattern (e.g. <element> at root)
		startPat = c.parsePattern(ctx, root) //nolint:contextcheck
	}

	c.resolveRefs(ctx)
	c.checkRefCycles(ctx) //nolint:contextcheck
	c.popGrammar(ctx)

	c.grammar.start = startPat
	c.checkRules(ctx) //nolint:contextcheck

	return c.grammar, nil
}

func (c *compiler) pushGrammar(ctx context.Context) {
	c.grammarStack = append(c.grammarStack, &grammarScope{
		defines: make(map[string]*defineEntry),
	})
}

func (c *compiler) popGrammar(ctx context.Context) {
	if len(c.grammarStack) > 0 {
		c.grammarStack = c.grammarStack[:len(c.grammarStack)-1]
	}
}

func (c *compiler) currentGrammar(ctx context.Context) *grammarScope {
	if len(c.grammarStack) == 0 {
		return nil
	}
	return c.grammarStack[len(c.grammarStack)-1]
}

func (c *compiler) resolveStart(ctx context.Context) *pattern {
	g := c.currentGrammar(ctx)
	if g == nil {
		return nil
	}
	entry, ok := g.defines["##start"]
	if !ok {
		return nil
	}
	return entry.pattern
}

func (c *compiler) resolveRefs(ctx context.Context) {
	g := c.currentGrammar(ctx)
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
func (c *compiler) checkRefCycles(ctx context.Context) {
	for name := range c.grammar.defines {
		visiting := map[string]bool{name: true}
		if ref := c.findCycleInPattern(ctx, c.grammar.defines[name], visiting); ref != nil {
			msg := rngParserErrorAt(c.filename, ref.line, "ref",
				fmt.Sprintf("Detected a cycle in %s references", ref.name))
			c.errorHandler.Handle(ctx, helium.NewLeveledError(msg, helium.ErrorLevelFatal))
			c.errorCount++
			return
		}
	}
}

// findCycleInPattern walks a pattern tree looking for ref cycles.
// Element patterns break the chain (refs inside elements don't create content cycles).
// Returns the offending ref pattern if a cycle is found.
func (c *compiler) findCycleInPattern(ctx context.Context, pat *pattern, visiting map[string]bool) *pattern {
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
		result := c.findCycleInPattern(ctx, def, visiting)
		delete(visiting, pat.name)
		return result
	default:
		for _, child := range pat.children {
			if ref := c.findCycleInPattern(ctx, child, visiting); ref != nil {
				return ref
			}
		}
		// Also check attrs
		for _, attr := range pat.attrs {
			if ref := c.findCycleInPattern(ctx, attr, visiting); ref != nil {
				return ref
			}
		}
		return nil
	}
}

// parseGrammarContent parses children of a <grammar> element.
func (c *compiler) parseGrammarContent(ctx context.Context, grammarElem *helium.Element) {
	for child := range helium.Children(grammarElem) {
		elem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if !isRNGElement(elem) {
			continue
		}

		switch elem.LocalName() {
		case "start": //nolint:goconst
			c.parseStart(ctx, elem)
		case "define": //nolint:goconst
			c.parseDefine(ctx, elem)
		case "include":
			c.parseInclude(ctx, elem)
		case "div": //nolint:goconst
			// <div> is transparent — recurse into its children
			c.parseGrammarContent(ctx, elem)
		}
	}
}

// parseStart parses a <start> element.
func (c *compiler) parseStart(ctx context.Context, elem *helium.Element) {
	combine := getAttr(elem, "combine")
	pat := c.parseChildren(ctx, elem)
	if pat == nil {
		return
	}

	g := c.currentGrammar(ctx)
	existing, ok := g.defines["##start"]
	if !ok {
		noCombine := 0
		if combine == "" {
			noCombine = 1
		}
		g.defines["##start"] = &defineEntry{pattern: pat, combine: combine, noCombine: noCombine}
		return
	}

	// Multiple <start> — validate combine modes.
	if combine == "" {
		existing.noCombine++
	}
	if existing.noCombine > 1 {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(rngParserError("Some <start> element miss the combine attribute"), helium.ErrorLevelFatal))
		c.errorCount++
	}
	if combine != "" && existing.combine != "" && combine != existing.combine {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(rngParserError("<start> use both 'interleave' and 'choice'"), helium.ErrorLevelFatal))
		c.errorCount++
	}

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
func (c *compiler) parseDefine(ctx context.Context, elem *helium.Element) {
	name := getAttr(elem, "name")
	if name == "" {
		return
	}
	combine := getAttr(elem, "combine")
	pat := c.parseChildren(ctx, elem)
	if pat == nil {
		pat = &pattern{kind: patternEmpty}
	}

	g := c.currentGrammar(ctx)
	existing, ok := g.defines[name]
	if !ok {
		noCombine := 0
		if combine == "" {
			noCombine = 1
		}
		g.defines[name] = &defineEntry{pattern: pat, combine: combine, noCombine: noCombine}
		return
	}

	// Multiple <define> with same name — validate combine modes.
	if combine == "" {
		existing.noCombine++
	}
	if existing.noCombine > 1 {
		msg := fmt.Sprintf("Some defines for %s needs the combine attribute", name)
		c.errorHandler.Handle(ctx, helium.NewLeveledError(rngParserError(msg), helium.ErrorLevelFatal))
		c.errorCount++
	}
	if combine != "" && existing.combine != "" && combine != existing.combine {
		msg := fmt.Sprintf("Defines for %s use both 'interleave' and 'choice'", name)
		c.errorHandler.Handle(ctx, helium.NewLeveledError(rngParserError(msg), helium.ErrorLevelFatal))
		c.errorCount++
	}

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
	case "interleave": //nolint:goconst
		return &pattern{
			kind:     patternInterleave,
			children: []*pattern{existing, incoming},
		}
	case "choice": //nolint:goconst
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

// resolveHref resolves a relative href against xml:base ancestors and the
// compiler's baseDir. This mirrors libxml2's xmlRelaxNGCleanupTree xml:base
// fixup for <include> and <externalRef> hrefs.
func (c *compiler) resolveHref(ctx context.Context, elem *helium.Element, href string) string {
	if filepath.IsAbs(href) {
		return href
	}
	if doc := elem.OwnerDocument(); doc != nil {
		base := helium.NodeGetBase(doc, elem)
		if base != "" {
			return helium.BuildURI(href, base)
		}
	}
	if c.baseDir != "" {
		return filepath.Join(c.baseDir, href)
	}
	return href
}

// parseInclude parses an <include> element.
func (c *compiler) parseInclude(ctx context.Context, elem *helium.Element) {
	href := getAttr(elem, "href")
	if href == "" {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(rngParserError("include has no href attribute"), helium.ErrorLevelFatal))
		c.errorCount++
		return
	}

	path := c.resolveHref(ctx, elem, href)

	// Recursion guard
	for _, p := range c.includeStack {
		if p == path {
			msg := fmt.Sprintf("Detected an Include recursion for %s", href)
			c.errorHandler.Handle(ctx, helium.NewLeveledError(rngParserError(msg), helium.ErrorLevelFatal))
			c.errorCount++
			return
		}
	}
	if len(c.includeStack) >= c.includeLimit {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(rngParserError("Include limit reached"), helium.ErrorLevelFatal))
		c.errorCount++
		return
	}

	c.includeStack = append(c.includeStack, path)
	defer func() { c.includeStack = c.includeStack[:len(c.includeStack)-1] }()

	// Read and parse the included file
	data, err := os.ReadFile(path)
	if err != nil {
		msg := fmt.Sprintf("xmlRelaxNGParse: could not load %s", href)
		c.errorHandler.Handle(ctx, helium.NewLeveledError(rngParserError(msg), helium.ErrorLevelFatal))
		c.errorCount++
		return
	}
	doc, err := helium.NewParser().Parse(ctx, data)
	if err != nil {
		msg := fmt.Sprintf("xmlRelaxNGParse: could not load %s", href)
		c.errorHandler.Handle(ctx, helium.NewLeveledError(rngParserError(msg), helium.ErrorLevelFatal))
		c.errorCount++
		return
	}
	doc.SetURL(path)

	root := findDocumentElement(doc)
	if root == nil || !isRNG(root, "grammar") {
		msg := fmt.Sprintf("xmlRelaxNGParse: included file %s is not a grammar", href)
		c.errorHandler.Handle(ctx, helium.NewLeveledError(rngParserError(msg), helium.ErrorLevelFatal))
		c.errorCount++
		return
	}

	// Parse the included grammar content
	oldBaseDir := c.baseDir
	c.baseDir = filepath.Dir(path)
	c.parseGrammarContent(ctx, root)
	c.baseDir = oldBaseDir

	// Collect override names from <include> children, then delete them from
	// the current grammar scope so the overrides replace (not combine with)
	// the included definitions.
	overrideNames := c.collectOverrideNames(ctx, elem)
	g := c.currentGrammar(ctx)
	if g != nil {
		for _, name := range overrideNames {
			delete(g.defines, name)
		}
	}

	// Process overrides from the <include> element's children
	c.parseIncludeOverrides(ctx, elem)
}

// collectOverrideNames returns the define/start names overridden by an include element.
func (c *compiler) collectOverrideNames(ctx context.Context, elem *helium.Element) []string {
	var names []string
	for child := range helium.Children(elem) {
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
			names = append(names, c.collectOverrideNames(ctx, childElem)...)
		}
	}
	return names
}

// parseIncludeOverrides processes override children of an <include> element.
func (c *compiler) parseIncludeOverrides(ctx context.Context, elem *helium.Element) {
	for child := range helium.Children(elem) {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if !isRNGElement(childElem) {
			continue
		}
		switch childElem.LocalName() {
		case "start":
			c.parseStart(ctx, childElem)
		case "define":
			c.parseDefine(ctx, childElem)
		case "div":
			c.parseIncludeOverrides(ctx, childElem)
		}
	}
}

// parsePattern parses a single RNG pattern element.
func (c *compiler) parsePattern(ctx context.Context, node *helium.Element) *pattern {
	if node == nil || !isRNGElement(node) {
		return nil
	}

	switch node.LocalName() {
	case "element":
		return c.parseElement(ctx, node)
	case "attribute":
		return c.parseAttribute(ctx, node)
	case "empty":
		return &pattern{kind: patternEmpty, line: node.Line()}
	case "text":
		return &pattern{kind: patternText, line: node.Line()}
	case "notAllowed":
		return &pattern{kind: patternNotAllowed, line: node.Line()}
	case "zeroOrMore":
		p := &pattern{kind: patternZeroOrMore, line: node.Line()}
		p.children = c.parsePatternChildren(ctx, node)
		return p
	case "oneOrMore":
		p := &pattern{kind: patternOneOrMore, line: node.Line()}
		p.children = c.parsePatternChildren(ctx, node)
		return p
	case "optional":
		p := &pattern{kind: patternOptional, line: node.Line()}
		p.children = c.parsePatternChildren(ctx, node)
		return p
	case "choice":
		p := &pattern{kind: patternChoice, line: node.Line()}
		p.children = c.parsePatternChildren(ctx, node)
		return p
	case "group":
		p := &pattern{kind: patternGroup, line: node.Line()}
		p.children = c.parsePatternChildren(ctx, node)
		return p
	case "interleave":
		p := &pattern{kind: patternInterleave, line: node.Line()}
		p.children = c.parsePatternChildren(ctx, node)
		return p
	case "mixed":
		// mixed is interleave with text
		p := &pattern{kind: patternInterleave, line: node.Line()}
		children := c.parsePatternChildren(ctx, node)
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
			c.errorHandler.Handle(ctx, helium.NewLeveledError(rngParserError("ref has no name"), helium.ErrorLevelFatal))
			c.errorCount++
			return nil
		}
		return &pattern{kind: patternRef, name: name, line: node.Line()}
	case "parentRef":
		name := getAttr(node, "name")
		if name == "" {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(rngParserError("parentRef has no name"), helium.ErrorLevelFatal))
			c.errorCount++
			return nil
		}
		return &pattern{kind: patternParentRef, name: name, line: node.Line()}
	case "externalRef":
		return c.parseExternalRef(ctx, node)
	case "data":
		return c.parseData(ctx, node)
	case "value":
		return c.parseValue(ctx, node)
	case "list":
		p := &pattern{kind: patternList, line: node.Line()}
		p.children = c.parsePatternChildren(ctx, node)
		return p
	case "grammar":
		// Nested grammar
		c.pushGrammar(ctx)
		c.parseGrammarContent(ctx, node)
		startPat := c.resolveStart(ctx)
		c.resolveRefs(ctx)
		c.popGrammar(ctx)
		return startPat
	default:
		return nil
	}
}

// parseElement parses an <element> pattern.
func (c *compiler) parseElement(ctx context.Context, node *helium.Element) *pattern {
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
	for child := range helium.Children(node) {
		elem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if !isRNGElement(elem) {
			continue
		}

		// Check if this is a name class element
		if p.nameClass == nil && isNameClassElement(elem) {
			p.nameClass = c.parseNameClass(ctx, elem)
			if p.nameClass != nil && p.nameClass.kind == ncName {
				p.name = p.nameClass.name
				p.ns = p.nameClass.ns
			}
			continue
		}

		// Otherwise it's a content pattern
		pat := c.parsePattern(ctx, elem)
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
				c.addSchemaError(ctx, node, "Attributes conflicts in group")
				goto attrConflictDone
			}
		}
	}
attrConflictDone:

	// Check: element with no content (no children, no attrs)
	if len(contentChildren) == 0 && len(p.attrs) == 0 {
		c.addSchemaError(ctx, node, "xmlRelaxNGParseElement: element has no content")
	}

	// Check: content type error (mixing data/value with element/group patterns)
	if hasDataContent(contentChildren) && hasElementContent(contentChildren) {
		eName := p.name
		if eName == "" && p.nameClass != nil {
			eName = p.nameClass.name
		}
		c.addSchemaError(ctx, node, fmt.Sprintf("Element %s has a content type error", eName))
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
func (c *compiler) parseAttribute(ctx context.Context, node *helium.Element) *pattern {
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
	for child := range helium.Children(node) {
		elem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if !isRNGElement(elem) {
			continue
		}

		if p.nameClass == nil && isNameClassElement(elem) {
			p.nameClass = c.parseNameClass(ctx, elem)
			if p.nameClass != nil && p.nameClass.kind == ncName {
				p.name = p.nameClass.name
				p.ns = p.nameClass.ns
			}
			continue
		}

		pat := c.parsePattern(ctx, elem)
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
func (c *compiler) parseData(ctx context.Context, node *helium.Element) *pattern {
	p := &pattern{kind: patternData, line: node.Line()}

	typeName := getAttr(node, "type")
	library := getDatatypeLibrary(node)

	p.dataType = &dataType{
		library: library,
		name:    typeName,
	}

	// Parse <param> and <except> children
	for child := range helium.Children(node) {
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
			excPats := c.parsePatternChildren(ctx, elem)
			if len(excPats) > 0 {
				except := &pattern{kind: patternChoice, children: excPats}
				p.children = append(p.children, except)
			}
		}
	}

	return p
}

// parseValue parses a <value> pattern.
func (c *compiler) parseValue(ctx context.Context, node *helium.Element) *pattern {
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
func (c *compiler) parseExternalRef(ctx context.Context, node *helium.Element) *pattern {
	href := getAttr(node, "href")
	if href == "" {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(rngParserError("externalRef has no href attribute"), helium.ErrorLevelFatal))
		c.errorCount++
		return nil
	}

	path := c.resolveHref(ctx, node, href)

	// Recursion guard
	for _, p := range c.includeStack {
		if p == path {
			msg := fmt.Sprintf("Detected an externalRef recursion for %s", href)
			c.errorHandler.Handle(ctx, helium.NewLeveledError(rngParserError(msg), helium.ErrorLevelFatal))
			c.errorCount++
			return nil
		}
	}
	if len(c.includeStack) >= c.includeLimit {
		c.errorHandler.Handle(ctx, helium.NewLeveledError(rngParserError("Include limit reached"), helium.ErrorLevelFatal))
		c.errorCount++
		return nil
	}

	c.includeStack = append(c.includeStack, path)
	defer func() { c.includeStack = c.includeStack[:len(c.includeStack)-1] }()

	data, err := os.ReadFile(path)
	if err != nil {
		msg := fmt.Sprintf("xmlRelaxNGParse: could not load %s", href)
		c.errorHandler.Handle(ctx, helium.NewLeveledError(rngParserError(msg), helium.ErrorLevelFatal))
		c.errorCount++
		return nil
	}
	doc, err := helium.NewParser().Parse(ctx, data)
	if err != nil {
		msg := fmt.Sprintf("xmlRelaxNGParse: could not load %s", href)
		c.errorHandler.Handle(ctx, helium.NewLeveledError(rngParserError(msg), helium.ErrorLevelFatal))
		c.errorCount++
		return nil
	}
	doc.SetURL(path)

	root := findDocumentElement(doc)
	if root == nil {
		msg := fmt.Sprintf("xmlRelaxNGParse: external ref %s has no root", href)
		c.errorHandler.Handle(ctx, helium.NewLeveledError(rngParserError(msg), helium.ErrorLevelFatal))
		c.errorCount++
		return nil
	}

	oldBaseDir := c.baseDir
	c.baseDir = filepath.Dir(path)
	var result *pattern
	if isRNG(root, "grammar") {
		c.pushGrammar(ctx)
		c.parseGrammarContent(ctx, root)
		result = c.resolveStart(ctx)
		c.resolveRefs(ctx)
		c.popGrammar(ctx)
	} else {
		result = c.parsePattern(ctx, root)
	}
	c.baseDir = oldBaseDir

	return result
}

// parseNameClass parses a name class element.
func (c *compiler) parseNameClass(ctx context.Context, node *helium.Element) *nameClass {
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
		for child := range helium.Children(node) {
			elem, ok := child.(*helium.Element)
			if !ok {
				continue
			}
			if isRNG(elem, "except") {
				nc.except = c.parseNameClassChildren(ctx, elem)
			}
		}
		return nc
	case "nsName":
		ns, hasNS := getAttrOpt(node, "ns")
		if !hasNS {
			ns = getAncestorNS(node)
		}
		nc := &nameClass{kind: ncNsName, ns: ns}
		for child := range helium.Children(node) {
			elem, ok := child.(*helium.Element)
			if !ok {
				continue
			}
			if isRNG(elem, "except") {
				nc.except = c.parseNameClassChildren(ctx, elem)
			}
		}
		return nc
	case "choice":
		var classes []*nameClass
		for child := range helium.Children(node) {
			elem, ok := child.(*helium.Element)
			if !ok {
				continue
			}
			if !isRNGElement(elem) {
				continue
			}
			nc := c.parseNameClass(ctx, elem)
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
func (c *compiler) parseNameClassChildren(ctx context.Context, exceptElem *helium.Element) *nameClass {
	var classes []*nameClass
	for child := range helium.Children(exceptElem) {
		elem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if !isRNGElement(elem) {
			continue
		}
		nc := c.parseNameClass(ctx, elem)
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
func (c *compiler) parseChildren(ctx context.Context, parent *helium.Element) *pattern {
	children := c.parsePatternChildren(ctx, parent)
	if len(children) == 0 {
		return nil
	}
	if len(children) == 1 {
		return children[0]
	}
	return &pattern{kind: patternGroup, children: children}
}

// parsePatternChildren parses all RNG pattern children of an element.
func (c *compiler) parsePatternChildren(ctx context.Context, parent *helium.Element) []*pattern {
	var result []*pattern
	for child := range helium.Children(parent) {
		elem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if !isRNGElement(elem) {
			continue
		}
		pat := c.parsePattern(ctx, elem)
		if pat != nil {
			result = append(result, pat)
		}
	}
	return result
}

// Helper functions

func findDocumentElement(doc *helium.Document) *helium.Element {
	for child := range helium.Children(doc) {
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
	return elem.Namespace() != nil && elemNS(elem) == lexicon.NamespaceRelaxNG
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
	attr, ok := elem.FindAttribute(helium.LocalNamePredicate(name))
	if !ok {
		return ""
	}
	return strings.TrimSpace(attr.Value())
}

// getAttrOpt returns the value and presence of an attribute.
func getAttrOpt(elem *helium.Element, name string) (string, bool) {
	attr, ok := elem.FindAttribute(helium.LocalNamePredicate(name))
	if !ok {
		return "", false
	}
	return strings.TrimSpace(attr.Value()), true
}

// getAncestorNS walks up the RNG element tree to find the ns attribute.
// An explicit ns="" on an ancestor stops the walk (empty namespace).
func getAncestorNS(node *helium.Element) string {
	current := node.Parent()
	for current != nil {
		if elem, ok := current.(*helium.Element); ok {
			if ns, hasNS := getAttrOpt(elem, "ns"); hasNS {
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
func (c *compiler) addSchemaError(ctx context.Context, node *helium.Element, msg string) {
	line := node.Line()
	formatted := fmt.Sprintf("%s:%d: element %s: Relax-NG parser error : %s\n",
		c.filename, line, node.LocalName(), msg)
	c.errorHandler.Handle(ctx, helium.NewLeveledError(formatted, helium.ErrorLevelFatal))
	c.errorCount++
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
		if prefix == lexicon.PrefixXML {
			return localName, lexicon.NamespaceXML
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
	for child := range helium.Children(elem) {
		if t, ok := child.(*helium.Text); ok {
			sb.Write(t.Content())
		}
	}
	return sb.String()
}
