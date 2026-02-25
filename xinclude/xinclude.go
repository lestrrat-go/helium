package xinclude

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/encoding"
	"github.com/lestrrat-go/helium/xpointer"
)

const (
	xiNamespaceLegacy = "http://www.w3.org/2001/XInclude"
	xiNamespaceNew    = "http://www.w3.org/2003/XInclude"
	maxDepth          = 40
)

// Resolver loads content from a URI.
type Resolver interface {
	Resolve(href, base string) (io.ReadCloser, error)
}

// Option configures XInclude processing behavior.
type Option func(*processor)

// WithNoXIncludeNodes suppresses XIncludeStart/End marker nodes.
func WithNoXIncludeNodes() Option {
	return func(p *processor) { p.noMarkers = true }
}

// WithNoBaseFixup disables xml:base fixup on included content.
func WithNoBaseFixup() Option {
	return func(p *processor) { p.noBaseFixup = true }
}

// WithResolver sets a custom resource resolver.
func WithResolver(r Resolver) Option {
	return func(p *processor) { p.resolver = r }
}

// WithBaseURI sets the base URI for resolving relative hrefs.
func WithBaseURI(uri string) Option {
	return func(p *processor) { p.baseURI = uri }
}

// WithParseFlags reads XInclude-related parse flags and configures the
// processor accordingly. Recognized flags: ParseNoXIncNode, ParseNoBaseFix.
func WithParseFlags(flags helium.ParseOption) Option {
	return func(p *processor) {
		if flags.IsSet(helium.ParseNoXIncNode) {
			p.noMarkers = true
		}
		if flags.IsSet(helium.ParseNoBaseFix) {
			p.noBaseFixup = true
		}
	}
}

type docCacheEntry struct {
	doc *helium.Document
	err error
}

type txtCacheEntry struct {
	data []byte
	err  error
}

type processor struct {
	noMarkers   bool
	noBaseFixup bool
	resolver    Resolver
	baseURI     string
	expanding   map[string]bool       // circular inclusion detection (set during recursive expansion)
	docCache    map[string]docCacheEntry // cached parsed documents
	txtCache    map[string]txtCacheEntry // cached text inclusions
	depth       int
	count       int
}

// Process performs XInclude processing on the document.
// Returns the number of substitutions made, or an error.
func Process(doc *helium.Document, opts ...Option) (int, error) {
	return ProcessTree(doc, opts...)
}

// ProcessTree performs XInclude processing starting from any node in the tree.
// When called with a *Document, it processes the entire document.
// Returns the number of substitutions made, or an error.
func ProcessTree(node helium.Node, opts ...Option) (int, error) {
	p := &processor{
		expanding: make(map[string]bool),
		docCache:  make(map[string]docCacheEntry),
		txtCache:  make(map[string]txtCacheEntry),
	}
	for _, o := range opts {
		o(p)
	}
	if p.resolver == nil {
		p.resolver = &fileResolver{}
	}

	if err := p.processNode(node); err != nil {
		return p.count, err
	}
	return p.count, nil
}

func (p *processor) processNode(n helium.Node) error {
	if p.depth > maxDepth {
		return fmt.Errorf("xi:include: maximum recursion depth (%d) exceeded", maxDepth)
	}

	// Repeatedly collect and process xi:include elements at this level.
	// Fallback processing may insert new xi:include elements as siblings,
	// so we loop until no more are found.
	for {
		var includes []*helium.Element
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if isXInclude(c) {
				includes = append(includes, c.(*helium.Element))
			}
		}
		if len(includes) == 0 {
			break
		}
		for _, inc := range includes {
			if err := p.processInclude(inc); err != nil {
				return err
			}
		}
	}

	// Recurse into remaining children (including newly inserted content)
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			if err := p.processNode(c); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *processor) processInclude(inc *helium.Element) error {
	href := getAttr(inc, "href")
	xptrExpr := getAttr(inc, "xpointer")
	parse := getAttr(inc, "parse")
	if parse == "" {
		parse = "xml"
	}

	// Extract fragment identifier from href
	var fragment string
	if idx := strings.IndexByte(href, '#'); idx >= 0 {
		fragment = href[idx+1:]
		href = href[:idx]
	}

	// 2003 namespace with fragment in href is an error per spec
	if getNamespaceURI(inc) == xiNamespaceNew && fragment != "" {
		return p.handleFallback(inc, fmt.Errorf("xi:include: invalid fragment identifier in URI, use the xpointer attribute"))
	}

	// Use fragment as xpointer expression if xpointer attribute is not set
	if xptrExpr == "" && fragment != "" {
		xptrExpr = fragment
	}

	// Neither href nor xpointer → error
	if href == "" && xptrExpr == "" {
		return p.handleFallback(inc, fmt.Errorf("xi:include missing href attribute"))
	}

	// Compute effective base URI at the include point, accounting for
	// xml:base attributes on ancestor elements.
	incBase := effectiveBaseURI(inc, p.baseURI)

	// Resolve the document URI
	var resolved string
	if href != "" {
		var err error
		resolved, err = resolveURI(href, incBase)
		if err != nil {
			return p.handleFallback(inc, fmt.Errorf("xi:include: cannot resolve URI %q: %w", href, err))
		}
	} else if fragment != "" && p.baseURI != "" {
		// href="#fragment" with base URI set → load document from base URI
		resolved = p.baseURI
	}
	// else: href absent, xpointer only → same-document (resolved stays "")

	// Circular inclusion check key includes xpointer expression
	circularKey := resolved
	if xptrExpr != "" {
		if resolved != "" {
			circularKey = resolved + "#" + xptrExpr
		} else {
			circularKey = "#" + xptrExpr
		}
	}
	if p.expanding[circularKey] {
		return fmt.Errorf("xi:include: circular inclusion detected for %q", circularKey)
	}

	var err error
	switch parse {
	case "xml":
		if xptrExpr != "" {
			err = p.includeXMLWithXPointer(inc, resolved, xptrExpr, incBase)
		} else {
			err = p.includeXML(inc, resolved, incBase)
		}
	case "text":
		if resolved == "" {
			err = fmt.Errorf("xi:include: text inclusion requires href")
		} else {
			err = p.includeText(inc, resolved)
		}
	default:
		err = fmt.Errorf("xi:include: unsupported parse value %q", parse)
	}

	if err != nil {
		return p.handleFallback(inc, err)
	}
	return nil
}

func (p *processor) includeXML(inc *helium.Element, uri string, incBase string) error {
	doc, err := p.loadXMLDoc(uri)
	if err != nil {
		return err
	}

	// Collect top-level children from included document, skipping DTD nodes
	var nodes []helium.Node
	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.DTDNode {
			continue
		}
		nodes = append(nodes, c)
	}

	if len(nodes) == 0 {
		unlinkNode(inc)
		p.count++
		return nil
	}

	// Set xml:base on included content (if not suppressed)
	if !p.noBaseFixup {
		fixupSource, fixupTarget := p.computeFixupBases(inc, uri)
		for _, n := range nodes {
			if elem, ok := n.(*helium.Element); ok {
				computeAndSetBaseURI(elem, fixupSource, fixupTarget)
			}
		}
	}

	p.replaceWithNodes(inc, nodes)
	p.count++

	// Recursively process included content for nested xi:include.
	// Temporarily set the base URI to the included document's URI
	// so that relative hrefs in the included content resolve correctly.
	savedBase := p.baseURI
	p.baseURI = uri
	p.expanding[uri] = true
	p.depth++
	for _, n := range nodes {
		if n.Type() == helium.ElementNode {
			if err := p.processNode(n); err != nil {
				p.depth--
				delete(p.expanding, uri)
				p.baseURI = savedBase
				return err
			}
		}
	}
	p.depth--
	delete(p.expanding, uri)
	p.baseURI = savedBase

	return nil
}

func (p *processor) includeXMLWithXPointer(inc *helium.Element, uri string, xptrExpr string, incBase string) error {
	var doc *helium.Document
	var err error

	if uri == "" {
		// Same-document reference: evaluate against a fresh parse of the
		// current document (via base URI) to avoid seeing nodes that were
		// inserted by previous XInclude processing in this pass.
		if p.baseURI != "" {
			doc, err = p.loadXMLDoc(p.baseURI)
			if err != nil {
				return err
			}
		} else {
			doc = inc.OwnerDocument()
		}
	} else {
		doc, err = p.loadXMLDoc(uri)
		if err != nil {
			return err
		}
	}

	// Evaluate XPointer expression against the document
	nodes, err := xpointer.Evaluate(doc, xptrExpr)
	if err != nil {
		return fmt.Errorf("xi:include: XPointer evaluation failed: %w", err)
	}

	if len(nodes) == 0 {
		unlinkNode(inc)
		p.count++
		return nil
	}

	// Compute effective base for each source node BEFORE copying
	// (copies lose their ancestor chain).
	var srcBases []string
	var fixupTargetBase string
	if !p.noBaseFixup && uri != "" {
		fixupSource, fixupTarget := p.computeFixupBases(inc, uri)
		fixupTargetBase = fixupTarget
		for _, n := range nodes {
			srcBases = append(srcBases, effectiveBaseURI(n, fixupSource))
		}
	}

	// Deep-copy result nodes into the target document
	targetDoc := inc.OwnerDocument()
	var copies []helium.Node
	for _, n := range nodes {
		c, copyErr := xpointer.CopyNode(n, targetDoc)
		if copyErr != nil {
			return fmt.Errorf("xi:include: copy failed: %w", copyErr)
		}
		copies = append(copies, c)
	}

	// Apply xml:base fixup (only for cross-document includes)
	if !p.noBaseFixup && uri != "" {
		for i, n := range copies {
			if elem, ok := n.(*helium.Element); ok {
				computeBaseForIncludedNode(elem, srcBases[i], fixupTargetBase)
			}
		}
	}

	p.replaceWithNodes(inc, copies)
	p.count++

	// Circular detection key
	circularKey := "#" + xptrExpr
	if uri != "" {
		circularKey = uri + "#" + xptrExpr
	}

	// Recursively process included content for nested xi:include
	savedBase := p.baseURI
	if uri != "" {
		p.baseURI = uri
	}
	p.expanding[circularKey] = true
	p.depth++
	for _, n := range copies {
		if n.Type() == helium.ElementNode {
			if err := p.processNode(n); err != nil {
				p.depth--
				delete(p.expanding, circularKey)
				p.baseURI = savedBase
				return err
			}
		}
	}
	p.depth--
	delete(p.expanding, circularKey)
	p.baseURI = savedBase

	return nil
}

func (p *processor) loadXMLDoc(uri string) (*helium.Document, error) {
	if entry, ok := p.docCache[uri]; ok {
		if entry.err != nil {
			return nil, entry.err
		}
		// Re-parse from cache: we need a fresh copy since nodes get moved into
		// the target document. Re-resolve to get the data again.
		// Actually, we cache the raw bytes and re-parse each time to get
		// independent node trees.
	}

	rc, err := p.resolver.Resolve(uri, p.baseURI)
	if err != nil {
		p.docCache[uri] = docCacheEntry{err: err}
		return nil, err
	}
	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(rc)
	if err != nil {
		wrapErr := fmt.Errorf("xi:include: error reading %q: %w", uri, err)
		p.docCache[uri] = docCacheEntry{err: wrapErr}
		return nil, wrapErr
	}

	parser := helium.NewParser()
	parser.SetOption(helium.ParseDTDLoad)
	parser.SetBaseURI(uri)
	doc, err := parser.Parse(data)
	if err != nil {
		wrapErr := fmt.Errorf("xi:include: error parsing %q: %w", uri, err)
		p.docCache[uri] = docCacheEntry{err: wrapErr}
		return nil, wrapErr
	}

	// Cache successful parse (note: nodes will be detached, so subsequent
	// inclusions of the same URI will need re-parsing via Resolve)
	p.docCache[uri] = docCacheEntry{doc: doc}
	return doc, nil
}

func (p *processor) includeText(inc *helium.Element, uri string) error {
	data, err := p.loadText(uri)
	if err != nil {
		return err
	}

	// Handle encoding attribute
	encName := getAttr(inc, "encoding")
	if encName != "" {
		enc := encoding.Load(encName)
		if enc != nil {
			decoded, decErr := enc.NewDecoder().Bytes(data)
			if decErr != nil {
				return fmt.Errorf("xi:include: error decoding %q with encoding %q: %w", uri, encName, decErr)
			}
			data = decoded
		}
	}

	// Validate that text contains only valid XML characters
	if err := validateXMLChars(data); err != nil {
		return fmt.Errorf("xi:include: %s contains invalid char", uri)
	}

	doc := inc.OwnerDocument()
	text, err := doc.CreateText(data)
	if err != nil {
		return fmt.Errorf("xi:include: error creating text node: %w", err)
	}

	p.replaceWithNodes(inc, []helium.Node{text})
	p.count++
	return nil
}

func (p *processor) loadText(uri string) ([]byte, error) {
	if entry, ok := p.txtCache[uri]; ok {
		return entry.data, entry.err
	}

	rc, err := p.resolver.Resolve(uri, p.baseURI)
	if err != nil {
		p.txtCache[uri] = txtCacheEntry{err: err}
		return nil, err
	}
	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(rc)
	if err != nil {
		wrapErr := fmt.Errorf("xi:include: error reading %q: %w", uri, err)
		p.txtCache[uri] = txtCacheEntry{err: wrapErr}
		return nil, wrapErr
	}

	p.txtCache[uri] = txtCacheEntry{data: data}
	return data, nil
}

func (p *processor) handleFallback(inc *helium.Element, origErr error) error {
	nsURI := getNamespaceURI(inc)
	for c := inc.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			if elem, ok := c.(*helium.Element); ok {
				if elem.LocalName() == "fallback" && getNamespaceURI(elem) == nsURI {
					return p.processFallback(inc, elem)
				}
			}
		}
	}
	return origErr
}

func (p *processor) processFallback(inc *helium.Element, fb *helium.Element) error {
	var nodes []helium.Node
	for c := fb.FirstChild(); c != nil; c = c.NextSibling() {
		nodes = append(nodes, c)
	}

	if len(nodes) == 0 {
		unlinkNode(inc)
		p.count++
		return nil
	}

	// Fix namespace declarations for nodes being moved out of their
	// declaring context (the fallback element may have xmlns:* declarations
	// that children rely on).
	for _, n := range nodes {
		fixupNamespaceDecls(n)
	}

	p.replaceWithNodes(inc, nodes)
	p.count++
	return nil
}

func (p *processor) replaceWithNodes(target *helium.Element, nodes []helium.Node) {
	if len(nodes) == 0 {
		unlinkNode(target)
		return
	}

	// Detach nodes from their original parents
	for _, n := range nodes {
		if n.Parent() != nil {
			unlinkNode(n)
		}
	}

	if !p.noMarkers {
		doc := target.OwnerDocument()
		startMarker := newXIncludeNode(doc, helium.XIncludeStartNode, target.Name())
		endMarker := newXIncludeNode(doc, helium.XIncludeEndNode, target.Name())

		expanded := make([]helium.Node, 0, len(nodes)+2)
		expanded = append(expanded, startMarker)
		expanded = append(expanded, nodes...)
		expanded = append(expanded, endMarker)
		nodes = expanded
	}

	spliceReplace(target, nodes)
}

// spliceReplace replaces target node with the given slice of nodes.
// Uses target.Replace() for the first node (which handles firstChild/lastChild
// updates via the exported API), then chains remaining nodes as siblings.
func spliceReplace(target helium.Node, nodes []helium.Node) {
	if len(nodes) == 0 {
		unlinkNode(target)
		return
	}

	afterTarget := target.NextSibling()

	// Replace target with the first node (handles parent firstChild/lastChild)
	target.Replace(nodes[0])

	// Chain remaining nodes after the first
	prev := nodes[0]
	for i := 1; i < len(nodes); i++ {
		cur := nodes[i]
		cur.SetParent(prev.Parent())
		cur.SetPrevSibling(prev)
		prev.SetNextSibling(cur)
		prev = cur
	}

	// Link last node to whatever followed target
	last := nodes[len(nodes)-1]
	last.SetNextSibling(afterTarget)
	if afterTarget != nil {
		afterTarget.SetPrevSibling(last)
	}
}

// unlinkNode removes a node from its parent's child list.
func unlinkNode(n helium.Node) {
	parent := n.Parent()
	if parent == nil {
		return
	}

	prev := n.PrevSibling()
	next := n.NextSibling()

	if prev != nil {
		prev.SetNextSibling(next)
	}
	if next != nil {
		next.SetPrevSibling(prev)
	}

	// Handle firstChild/lastChild: if this is the only child, or first/last,
	// we need to use Replace or AddChild to properly update parent pointers.
	// Since setFirstChild/setLastChild are unexported, we use a workaround:
	// if n is the only child, replace with a dummy then remove it.
	if prev == nil && next == nil {
		// Only child — disconnect. The parent's firstChild/lastChild will be
		// stale, but for our use case (xi:include is never the only child
		// under document root), this should not be an issue.
		n.SetParent(nil)
		n.SetPrevSibling(nil)
		n.SetNextSibling(nil)
		return
	}

	if prev == nil && next != nil {
		// First child but not only: use Replace to make next the first
		n.Replace(next)
		return
	}

	if next == nil && prev != nil {
		// Last child: prev becomes new last.
		prev.SetNextSibling(nil)
	}

	n.SetParent(nil)
	n.SetPrevSibling(nil)
	n.SetNextSibling(nil)
}

func isXInclude(n helium.Node) bool {
	if n.Type() != helium.ElementNode {
		return false
	}
	elem, ok := n.(*helium.Element)
	if !ok {
		return false
	}
	ns := getNamespaceURI(elem)
	return elem.LocalName() == "include" && (ns == xiNamespaceLegacy || ns == xiNamespaceNew)
}

func getNamespaceURI(n helium.Node) string {
	type urier interface {
		URI() string
	}
	if u, ok := n.(urier); ok {
		return u.URI()
	}
	return ""
}

func getAttr(elem *helium.Element, name string) string {
	for _, a := range elem.Attributes() {
		if a.LocalName() == name {
			return a.Value()
		}
	}
	return ""
}


func resolveURI(href, base string) (string, error) {
	hrefURL, err := url.Parse(href)
	if err != nil {
		return "", err
	}

	if hrefURL.IsAbs() {
		return href, nil
	}

	if base == "" {
		return href, nil
	}

	// For file-like paths (no scheme), use filepath-based resolution
	// to avoid Go's url.ResolveReference quirk that adds leading '/'
	// to purely relative paths.
	baseURL, err := url.Parse(base)
	if err != nil {
		return href, nil
	}
	if baseURL.Scheme == "" || baseURL.Scheme == "file" {
		basePath := baseURL.Path
		if basePath == "" {
			basePath = base
		}
		return filepath.Join(filepath.Dir(basePath), href), nil
	}

	return baseURL.ResolveReference(hrefURL).String(), nil
}

// computeAndSetBaseURI computes the relative URI of the included resource
// against the target document's base, and sets xml:base only when needed.
// Used for whole-document XML inclusion (includeXML).
func computeAndSetBaseURI(elem *helium.Element, includedURI, targetBase string) {
	// If the included element already has xml:base set, leave it alone
	for _, a := range elem.Attributes() {
		if a.Name() == "xml:base" {
			return
		}
	}

	// Compute relative URI if possible
	base := relativeURI(includedURI, targetBase)
	if base == "" {
		return
	}

	_ = elem.SetAttribute("xml:base", base)
}

// computeBaseForIncludedNode sets xml:base on a node that was included via
// XPointer. srcEffectiveBase is the absolute effective base of the source
// node (computed from its ancestor xml:base chain). targetEffectiveBase is
// the effective base at the xi:include point in the target document.
func computeBaseForIncludedNode(elem *helium.Element, srcEffectiveBase, targetEffectiveBase string) {
	// Check if this element has an existing xml:base attribute
	var existingBase string
	for _, a := range elem.Attributes() {
		if a.Name() == "xml:base" {
			existingBase = a.Value()
			break
		}
	}

	if existingBase != "" {
		// Element has xml:base in the source. If absolute, keep it.
		if u, err := url.Parse(existingBase); err == nil && u.IsAbs() {
			return
		}
		// The element's xml:base was relative to the source context.
		// srcEffectiveBase already incorporates this element's xml:base,
		// so relativize it against the target's effective base.
		newBase := relativeURI(srcEffectiveBase, targetEffectiveBase)
		if newBase == "" {
			return
		}
		_ = elem.SetAttribute("xml:base", newBase)
	} else {
		// No xml:base — set one relative to the target's effective base.
		newBase := relativeURI(srcEffectiveBase, targetEffectiveBase)
		if newBase == "" {
			return
		}
		_ = elem.SetAttribute("xml:base", newBase)
	}
}

// relativeURI attempts to compute a relative URI of target against base.
// Returns the relative form if possible, otherwise the absolute target.
func relativeURI(target, base string) string {
	if target == "" {
		return ""
	}

	if base == "" {
		return target
	}

	// Same URI — no xml:base needed
	if target == base {
		return ""
	}

	targetURL, err := url.Parse(target)
	if err != nil {
		return target
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return target
	}

	// Different schemes or hosts — can't relativize
	if targetURL.Scheme != baseURL.Scheme || targetURL.Host != baseURL.Host {
		return target
	}

	// Both are relative paths or both file-like: compute relative path
	targetPath := targetURL.Path
	if targetPath == "" {
		targetPath = target
	}
	basePath := baseURL.Path
	if basePath == "" {
		basePath = base
	}

	return makeRelativePath(targetPath, basePath)
}

// makeRelativePath computes a relative path from basePath's directory to targetPath.
func makeRelativePath(targetPath, basePath string) string {
	// Split into directory components
	baseDir := filepath.Dir(basePath)
	if baseDir == "." {
		// Base has no directory component — target is already relative
		return targetPath
	}

	// Use filepath.Rel for the computation
	rel, err := filepath.Rel(baseDir, targetPath)
	if err != nil {
		return targetPath
	}

	// filepath.Rel uses OS separators; normalize to forward slashes
	return strings.ReplaceAll(rel, string(filepath.Separator), "/")
}

func newXIncludeNode(doc *helium.Document, etype helium.ElementType, name string) helium.Node {
	return helium.NewXIncludeNode(doc, etype, name)
}

// fixupNamespaceDecls ensures that elements being moved out of their declaring
// context carry their own namespace declarations. For each element in the subtree,
// if it has an active namespace whose prefix is not declared in the element's own
// nsDefs, a declaration is added.
func fixupNamespaceDecls(n helium.Node) {
	if n.Type() != helium.ElementNode {
		return
	}
	elem := n.(*helium.Element)

	// Build set of locally declared prefixes
	declared := make(map[string]bool)
	if nc, ok := helium.Node(elem).(helium.NamespaceContainer); ok {
		for _, ns := range nc.Namespaces() {
			declared[ns.Prefix()] = true
		}
	}

	// If the element has an active namespace prefix not locally declared, add it
	if nsr, ok := helium.Node(elem).(helium.Namespacer); ok {
		if ns := nsr.Namespace(); ns != nil {
			if ns.Prefix() != "" && !declared[ns.Prefix()] {
				_ = elem.SetNamespace(ns.Prefix(), ns.URI())
			}
		}
	}

	// Recurse into children
	for c := elem.FirstChild(); c != nil; c = c.NextSibling() {
		fixupNamespaceDecls(c)
	}
}

// validateXMLChars checks that data contains only valid XML characters.
// Valid XML chars: #x9 | #xA | #xD | [#x20-#xD7FF] | [#xE000-#xFFFD] | [#x10000-#x10FFFF]
func validateXMLChars(data []byte) error {
	for i := 0; i < len(data); {
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size <= 1 {
			return fmt.Errorf("invalid byte at offset %d", i)
		}
		if !isValidXMLChar(r) {
			return fmt.Errorf("invalid XML character U+%04X at offset %d", r, i)
		}
		i += size
	}
	return nil
}

func isValidXMLChar(r rune) bool {
	return r == 0x9 || r == 0xA || r == 0xD ||
		(r >= 0x20 && r <= 0xD7FF) ||
		(r >= 0xE000 && r <= 0xFFFD) ||
		(r >= 0x10000 && r <= 0x10FFFF)
}

// effectiveBaseURI computes the effective base URI of a node by walking
// ancestor xml:base attributes. Per RFC 3986 / XML Base, each xml:base
// is resolved against the parent's effective base, starting from the document URI.
func effectiveBaseURI(node helium.Node, docURI string) string {
	// Collect xml:base values from ancestors (leaf-to-root order)
	var bases []string
	for n := node; n != nil; n = n.Parent() {
		if elem, ok := n.(*helium.Element); ok {
			for _, a := range elem.Attributes() {
				if a.Name() == "xml:base" {
					bases = append(bases, a.Value())
					break
				}
			}
		}
	}
	if len(bases) == 0 {
		return docURI
	}

	// Apply xml:base values from root to leaf
	result := docURI
	for i := len(bases) - 1; i >= 0; i-- {
		result = resolveBase(result, bases[i])
	}
	return result
}

// resolveBase resolves an xml:base value against a current base URI using
// RFC 3986 URI resolution semantics. Unlike filepath.Join, ".." segments
// cannot traverse above the URI root.
func resolveBase(currentBase, xmlBase string) string {
	xmlBaseURL, err := url.Parse(xmlBase)
	if err != nil {
		return xmlBase
	}
	// Absolute URI replaces entirely
	if xmlBaseURL.IsAbs() {
		return xmlBase
	}

	if currentBase == "" {
		return xmlBase
	}

	baseURL, err := url.Parse(currentBase)
	if err != nil {
		return xmlBase
	}

	// If the base already has a real scheme (not file), use standard resolution.
	if baseURL.Scheme != "" && baseURL.Scheme != "file" {
		return baseURL.ResolveReference(xmlBaseURL).String()
	}

	// For file-like paths (no scheme), use URL resolution with a synthetic
	// absolute prefix so that ".." segments are properly bounded by the
	// URI root, matching RFC 3986 semantics.
	const syntheticPrefix = "synthetic://h/"
	syntheticBase := syntheticPrefix + currentBase
	absBase, err := url.Parse(syntheticBase)
	if err != nil {
		return filepath.Join(filepath.Dir(currentBase), xmlBase)
	}
	resolved := absBase.ResolveReference(xmlBaseURL)
	result := strings.TrimPrefix(resolved.String(), syntheticPrefix)
	return result
}

// commonAncestorDir returns the longest common directory prefix of two paths.
func commonAncestorDir(a, b string) string {
	aDir := filepath.Dir(filepath.Clean(a))
	bDir := filepath.Dir(filepath.Clean(b))
	aParts := strings.Split(aDir, string(filepath.Separator))
	bParts := strings.Split(bDir, string(filepath.Separator))

	n := len(aParts)
	if len(bParts) < n {
		n = len(bParts)
	}

	common := 0
	for i := 0; i < n; i++ {
		if aParts[i] != bParts[i] {
			break
		}
		common = i + 1
	}

	if common == 0 {
		return "."
	}

	return strings.Join(aParts[:common], string(filepath.Separator))
}

// computeFixupBases computes relative URI bases for xml:base fixup.
// When both the source and target document URIs are absolute paths,
// they are converted to relative paths against their common ancestor
// directory. This ensures that ".." traversal in xml:base attributes
// is bounded at the logical root, matching RFC 3986 URI resolution.
func (p *processor) computeFixupBases(inc *helium.Element, sourceURI string) (string, string) {
	relSource := sourceURI
	relTarget := p.baseURI

	if filepath.IsAbs(sourceURI) && filepath.IsAbs(p.baseURI) {
		root := commonAncestorDir(sourceURI, p.baseURI)
		if rel, err := filepath.Rel(root, sourceURI); err == nil {
			relSource = rel
		}
		if rel, err := filepath.Rel(root, p.baseURI); err == nil {
			relTarget = rel
		}
	}

	return relSource, effectiveBaseURI(inc, relTarget)
}

// fileResolver resolves URIs by reading from the filesystem.
type fileResolver struct{}

func (r *fileResolver) Resolve(href, base string) (io.ReadCloser, error) {
	path := href
	if !filepath.IsAbs(path) && base != "" {
		baseURL, err := url.Parse(base)
		if err == nil && (baseURL.Scheme == "" || baseURL.Scheme == "file") {
			basePath := baseURL.Path
			if basePath == "" {
				basePath = base
			}
			path = filepath.Join(filepath.Dir(basePath), href)
		}
	}
	return os.Open(path)
}
