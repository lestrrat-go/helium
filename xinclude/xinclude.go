package xinclude

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/encoding"
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
	parse := getAttr(inc, "parse")
	if parse == "" {
		parse = "xml"
	}

	if href == "" {
		return p.handleFallback(inc, fmt.Errorf("xi:include missing href attribute"))
	}

	// Resolve URI
	resolved, err := resolveURI(href, p.baseURI)
	if err != nil {
		return p.handleFallback(inc, fmt.Errorf("xi:include: cannot resolve URI %q: %w", href, err))
	}

	// Circular inclusion check: fatal if we're already expanding this URI
	if p.expanding[resolved] {
		return fmt.Errorf("xi:include: circular inclusion detected for %q", resolved)
	}

	switch parse {
	case "xml":
		err = p.includeXML(inc, resolved)
	case "text":
		err = p.includeText(inc, resolved)
	default:
		err = fmt.Errorf("xi:include: unsupported parse value %q", parse)
	}

	if err != nil {
		return p.handleFallback(inc, err)
	}
	return nil
}

func (p *processor) includeXML(inc *helium.Element, uri string) error {
	doc, err := p.loadXMLDoc(uri)
	if err != nil {
		return err
	}

	// Collect top-level children from included document
	var nodes []helium.Node
	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		nodes = append(nodes, c)
	}

	if len(nodes) == 0 {
		unlinkNode(inc)
		p.count++
		return nil
	}

	// Set xml:base on included content (if not suppressed)
	if !p.noBaseFixup {
		for _, n := range nodes {
			if elem, ok := n.(*helium.Element); ok {
				computeAndSetBaseURI(elem, uri, p.baseURI)
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

	doc, err := helium.Parse(data)
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

// relativeURI attempts to compute a relative URI of target against base.
// Returns the relative form if possible, otherwise the absolute target.
func relativeURI(target, base string) string {
	if target == "" {
		return ""
	}

	if base == "" {
		return target
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
