package xinclude

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/encoding"
	"github.com/lestrrat-go/helium/internal/iofs"
	"github.com/lestrrat-go/helium/internal/iolimit"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/uripath"
	"github.com/lestrrat-go/helium/xpointer"
)

const maxURILength = 2000

// defaultMaxIncludeSize is the per-include byte cap used when none is configured
// via [Processor.MaxIncludeSize]. It guards against a hostile or pathological
// Resolver (e.g. one returning an endless or multi-gigabyte reader) exhausting
// memory: the bytes of every included resource are read fully and cached.
const defaultMaxIncludeSize = 10 << 20 // 10 MiB

// defaultMaxIncludeDepth is the xi:include nesting-depth cap used when none is
// configured via [Processor.MaxIncludeDepth]. It guards against pathological or
// maliciously deep include chains. This bounds nesting depth only; cyclic
// includes are caught separately by circular-inclusion detection.
const defaultMaxIncludeDepth = 40

// maxIncludeAggregateMultiplier bounds the cumulative bytes materialized across
// the entire XInclude expansion as a multiple of the effective per-include cap
// (see effectiveMaxIncludeSize). The per-include cap only stops a single
// oversized resource; without an aggregate bound a document could splice many
// includes — each under the per-include cap, or the same cached resource reused
// repeatedly — and still materialize unbounded total bytes (a memory/
// amplification DoS). Expressing the aggregate as a multiple of the per-include
// cap keeps it proportional: lowering MaxIncludeSize lowers the aggregate too.
// With the 10 MiB default per-include cap the aggregate is 1 GiB.
const maxIncludeAggregateMultiplier = 100

// maxTotalIncludes bounds the aggregate number of resources spliced across the
// entire XInclude expansion, independent of their individual sizes. It guards
// against amplification by a very large number of tiny includes.
const maxTotalIncludes = 1 << 16 // 65536

// ErrIncludeTooLarge is returned when an included resource exceeds the maximum
// allowed size. This covers both the per-include cap (see
// [Processor.MaxIncludeSize]; 10 MiB by default), enforced against the actual
// number of bytes read, and the internal aggregate bound across the whole
// expansion (cumulative bytes / spliced-resource count) that guards against
// amplification by many includes each staying under the per-include cap.
var ErrIncludeTooLarge = errors.New("xi:include: included resource exceeds maximum allowed size")

// Resolver loads content from a URI.
//
// The processor resolves each xi:include href against the include's
// effective base URI (honoring ancestor xml:base attributes) BEFORE
// calling Resolve. The href argument is therefore the fully-resolved
// location to open, and base carries the effective base URI it was
// resolved against for informational/logging use. A Resolver MUST open
// href directly and MUST NOT resolve href against base a second time —
// doing so double-applies the base directory (e.g. opening
// dir/dir/inc.xml instead of dir/inc.xml).
type Resolver interface {
	Resolve(href, base string) (io.ReadCloser, error)
}

// processorCfg holds the configuration for a Processor.
type processorCfg struct {
	noMarkers       bool
	noBaseFixup     bool
	resolver        Resolver
	baseURI         string
	errorHandler    helium.ErrorHandler
	maxIncludeSize  int
	maxIncludeDepth int
	parser          *helium.Parser
}

// Processor configures XInclude processing. It is a value-style wrapper:
// fluent methods return updated copies and the original is never mutated.
type Processor struct {
	cfg *processorCfg
}

// NewProcessor creates a new Processor with default settings.
func NewProcessor() Processor {
	return Processor{cfg: &processorCfg{}}
}

func (p Processor) clone() Processor {
	if p.cfg == nil {
		return Processor{cfg: &processorCfg{}}
	}
	cp := *p.cfg
	return Processor{cfg: &cp}
}

// NoXIncludeMarkers suppresses XIncludeStart/End marker nodes.
func (p Processor) NoXIncludeMarkers() Processor {
	p = p.clone()
	p.cfg.noMarkers = true
	return p
}

// NoBaseFixup disables xml:base fixup on included content.
func (p Processor) NoBaseFixup() Processor {
	p = p.clone()
	p.cfg.noBaseFixup = true
	return p
}

// Resolver sets a custom resource resolver. When unset, the processor is
// secure by default: it uses a deny-all resolver that refuses every
// filesystem access, so untrusted input cannot disclose local files (mirrors
// the deny-all default of [helium.NewParser]). To grant access, supply a
// Resolver backed by a confined [fs.FS] — see [NewFSResolver], e.g.
// NewFSResolver(fsys) with fsys from [os.Root.FS]. To restore the historical
// behavior of opening any OS path, pass NewFSResolver([helium.PermissiveFS]()).
func (p Processor) Resolver(r Resolver) Processor {
	p = p.clone()
	p.cfg.resolver = r
	return p
}

// NewFSResolver returns a [Resolver] that opens hrefs through the given
// [fs.FS]. The processor resolves each xi:include href against the
// include's effective base URI before calling the resolver, so the href
// handed to fsys.Open is already the resolved location — it is opened
// directly and the base argument is ignored (joining it again would
// double-apply the base directory). A nil fsys is treated as the
// permissive default that opens any OS path verbatim.
//
// Names handed to fsys.Open use forward slashes and are canonicalized
// with [path.Clean]. A resolved name that escapes the FS root via ".."
// (e.g. "../foo") leaves a leading ".." in the cleaned result and will
// be rejected by an [fs.ValidPath]-enforcing FS, blocking that form of
// path traversal.
//
// Confinement note: rejecting ".." is NOT a complete sandbox. The level
// of confinement is entirely determined by the fsys you pass:
//
//   - [os.DirFS] is NOT confined. It rejects "../" names per
//     [fs.ValidPath], but it does NOT prevent a symlink inside the root
//     from pointing outside it: if "root/link.txt" is a symlink to
//     "../secret.txt", an xi:include with parse="text" can read that
//     file. Do not treat os.DirFS as a security boundary for untrusted
//     input.
//   - [os.OpenRoot] / [os.Root.FS] (Go 1.24+) IS confined: it refuses to
//     traverse symlinks that escape the root, so it is the appropriate
//     choice when the XInclude input is untrusted.
//   - [testing/fstest.MapFS] is an in-memory map with no host access.
//
// When using an FS that enforces [fs.ValidPath] — such as [os.DirFS],
// [testing/fstest.MapFS], or [os.Root.FS] — the document's base URI must
// be FS-relative: no leading slash and no file:// URL. An absolute base
// URI (e.g. set via [Processor.BaseURI] with "/abs/path/main.xml" or
// "file:///abs/main.xml") produces an absolute resolved name that
// fs.ValidPath rejects, so the open fails, even when the href itself
// does not escape. The exact error is FS-implementation-specific
// ([os.DirFS] and [os.Root.FS] report [fs.ErrInvalid], while
// [testing/fstest.MapFS] reports [fs.ErrNotExist]). This is a
// deliberate fail-loud contract — silently
// trimming the leading slash would re-anchor absolute paths under the FS
// root, masking caller mistakes and risking wrong-file opens. The
// permissive default (NewFSResolver(nil)) does not enforce fs.ValidPath
// and instead opens any OS path verbatim, so it accepts absolute base
// URIs.
func NewFSResolver(fsys fs.FS) Resolver {
	if fsys == nil {
		fsys = iofs.PermissiveRoot{}
	}
	return &fsResolver{fsys: fsys}
}

// BaseURI sets the base URI for resolving relative hrefs.
func (p Processor) BaseURI(uri string) Processor {
	p = p.clone()
	p.cfg.baseURI = uri
	return p
}

// MaxIncludeSize sets the maximum number of bytes read from a single included
// resource (XML or text). The cap is enforced against the actual number of
// bytes read, guarding against a hostile or pathological Resolver (e.g. an
// endless or multi-gigabyte reader) exhausting memory before the bytes are
// cached. A value less than or equal to zero (the default) means 10 MiB is
// used. Exceeding the cap fails the include with [ErrIncludeTooLarge].
func (p Processor) MaxIncludeSize(n int) Processor {
	p = p.clone()
	p.cfg.maxIncludeSize = n
	return p
}

// MaxIncludeDepth sets the maximum nesting depth of xi:include directives — how
// deeply an included document may itself include further documents before
// processing fails with "maximum include depth exceeded". It bounds nesting
// depth only; cyclic includes are caught separately by circular-inclusion
// detection. A value less than or equal to zero (the default) means 40 is used.
func (p Processor) MaxIncludeDepth(n int) Processor {
	p = p.clone()
	p.cfg.maxIncludeDepth = n
	return p
}

// Parser sets the [helium.Parser] used to parse included documents. The
// injected parser supplies the inner parse's resource limits (element depth,
// name length, entity amplification, content-model depth) and is used as the
// base; XInclude still forces its own loading policy on top — external DTD
// loading is enabled and the filesystem is confined to the configured
// [Resolver]'s sandbox (see [Processor.Resolver]), regardless of the injected
// parser's FS. When unset, a default [helium.NewParser] is used as the base.
func (p Processor) Parser(parser helium.Parser) Processor {
	p = p.clone()
	p.cfg.parser = &parser
	return p
}

// ErrorHandler sets a handler for non-fatal warnings such as
// entity definition mismatches during XInclude entity merging.
// Errors delivered to the handler have ErrorLevelWarning.
func (p Processor) ErrorHandler(h helium.ErrorHandler) Processor {
	p = p.clone()
	p.cfg.errorHandler = h
	return p
}

type docCacheEntry struct {
	data []byte // raw bytes for re-parsing
	err  error
}

type txtCacheEntry struct {
	data []byte
	err  error
}

type processor struct {
	noMarkers       bool
	noBaseFixup     bool
	resolver        Resolver
	baseURI         string
	expanding       map[string]bool          // circular inclusion detection (set during recursive expansion)
	docCache        map[string]docCacheEntry // cached raw bytes for XML documents
	txtCache        map[string]txtCacheEntry // cached text inclusions
	errorHandler    helium.ErrorHandler
	maxIncludeSize  int
	maxIncludeDepth int
	parser          *helium.Parser
	snapshot        *helium.Document // in-memory copy of the entry document for same-document XPointers
	snapshotURI     string           // base URI the snapshot corresponds to
	depth           int
	count           int
	totalBytes      int64 // aggregate bytes materialized across all includes (bounds amplification)
}

// Process performs XInclude processing on the document.
// Returns the number of substitutions made, or an error.
func (proc Processor) Process(ctx context.Context, doc *helium.Document) (int, error) { //nolint:contextcheck
	if ctx == nil {
		ctx = context.Background()
	}
	count, err := proc.ProcessTree(ctx, doc)
	if count > 0 {
		doc.SetProperties(doc.Properties() | helium.DocXInclude)
	}
	return count, err
}

// ProcessTree performs XInclude processing starting from any node in the tree.
// When called with a *Document, it processes the entire document.
// Returns the number of substitutions made, or an error.
func (proc Processor) ProcessTree(ctx context.Context, node helium.Node) (int, error) { //nolint:contextcheck
	if ctx == nil {
		ctx = context.Background()
	}
	cfg := proc.cfg
	if cfg == nil {
		cfg = &processorCfg{}
	}
	p := &processor{
		noMarkers:       cfg.noMarkers,
		noBaseFixup:     cfg.noBaseFixup,
		resolver:        cfg.resolver,
		baseURI:         cfg.baseURI,
		errorHandler:    cfg.errorHandler,
		maxIncludeSize:  cfg.maxIncludeSize,
		maxIncludeDepth: cfg.maxIncludeDepth,
		parser:          cfg.parser,
		expanding:       make(map[string]bool),
		docCache:        make(map[string]docCacheEntry),
		txtCache:        make(map[string]txtCacheEntry),
	}
	if p.resolver == nil {
		// Secure by default: an unset resolver denies all filesystem access
		// (mirroring helium.NewParser()'s deny-all FS). Untrusted input cannot
		// disclose local files via <xi:include href="/etc/passwd"/>. Callers
		// opt back into host access with Resolver(NewFSResolver(helium.PermissiveFS()))
		// or, preferably, a confined fs.FS (os.Root.FS).
		p.resolver = NewFSResolver(iofs.DenyAll{})
	}

	// Capture a resolver-free snapshot of the entry document so top-level
	// same-document XPointer references can be evaluated against the original
	// infoset (before inclusions mutate the tree) without re-reading the base
	// URI through the resolver. Only taken when such a reference is actually
	// present, so documents without same-document XPointers pay no copy cost.
	if doc := ownerDocument(node); doc != nil && hasSameDocumentXPointer(node) {
		if snap := snapshotForXPointer(doc); snap != nil {
			p.snapshot = snap
			p.snapshotURI = p.baseURI
		}
	}

	if err := p.processNode(ctx, node); err != nil {
		return p.count, err
	}
	return p.count, nil
}

func (p *processor) processNode(ctx context.Context, n helium.Node) error {
	// Repeatedly collect and process xi:include elements at this level.
	// Fallback processing may insert new xi:include elements as siblings,
	// so we loop until no more are found.
	for {
		var includes []*helium.Element
		for c := range helium.Children(n) {
			if isXInclude(c) {
				elem, ok := helium.AsNode[*helium.Element](c)
				if !ok {
					continue
				}
				includes = append(includes, elem)
			}
		}
		if len(includes) == 0 {
			break
		}
		for _, inc := range includes {
			if err := p.processInclude(ctx, inc); err != nil {
				return err
			}
		}
	}

	// Recurse into remaining children (including newly inserted content)
	for c := range helium.Children(n) {
		if c.Type() == helium.ElementNode {
			if isFallback(c) {
				return fmt.Errorf("xi:fallback is not the child of an 'include'")
			}
			if err := p.processNode(ctx, c); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *processor) processInclude(ctx context.Context, inc *helium.Element) error {
	// Reject before fetching/splicing the resource so an over-limit include
	// never retrieves or processes its target. Expanding this include nests its
	// content one level deeper (p.depth+1), so maxDepth is the actual ceiling on
	// nesting depth.
	maxDepth := p.maxIncludeDepth
	if maxDepth <= 0 {
		maxDepth = defaultMaxIncludeDepth
	}
	if p.depth+1 > maxDepth {
		return fmt.Errorf("xi:include: maximum include depth (%d) exceeded", maxDepth)
	}

	// Aggregate-count guard: bound the total number of resources spliced across
	// the whole expansion. p.count is the running total of substitutions made so
	// far; reject before processing one more so repeated (cached) includes cannot
	// amplify into unbounded subtree materialization.
	if p.count >= maxTotalIncludes {
		return fmt.Errorf("xi:include: aggregate of %d included resources: %w", maxTotalIncludes, ErrIncludeTooLarge)
	}

	if err := validateIncludeChildren(inc); err != nil {
		return err
	}

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

	// URI length limit (matches libxml2's XML_MAX_URI_LENGTH)
	if len(href) > maxURILength {
		return fmt.Errorf("xi:include: URI too long")
	}

	// 2003 namespace with fragment in href is an error per spec
	if getNamespaceURI(inc) == lexicon.NamespaceXInclude11 && fragment != "" {
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
	}
	// else: the href had no document part (a pure fragment such as href="#a", or
	// the href attribute was absent and only an xpointer was given) → this is a
	// same-document reference. Leave resolved empty so includeXMLWithXPointer
	// takes the in-memory snapshot path rather than re-loading the base URI
	// through the resolver — which, under the deny-all default resolver, would
	// fail even though the target is the current document.

	// Circular inclusion check key includes xpointer expression. For a
	// same-document reference (resolved == "") the key must also carry the
	// current document identity (p.baseURI); otherwise every fragment-only
	// include collapses to the literal "#xptr" and an included document's own
	// same-document include would collide with the includer's.
	circularKey := resolved
	if xptrExpr != "" {
		if resolved != "" {
			circularKey = resolved + "#" + xptrExpr
		} else {
			circularKey = p.baseURI + "#" + xptrExpr
		}
	}
	if p.expanding[circularKey] {
		return fmt.Errorf("xi:include: circular inclusion detected for %q", circularKey)
	}

	var err error
	switch parse {
	case "xml":
		if xptrExpr != "" {
			err = p.includeXMLWithXPointer(ctx, inc, resolved, xptrExpr, incBase)
		} else {
			err = p.includeXML(ctx, inc, resolved, incBase)
		}
	case "text":
		if resolved == "" {
			err = fmt.Errorf("xi:include: text inclusion requires href")
		} else {
			err = p.includeText(inc, resolved, incBase)
		}
	default:
		err = fmt.Errorf("xi:include: unsupported parse value %q", parse)
	}

	if err != nil {
		return p.handleFallback(inc, err)
	}
	return nil
}

func (p *processor) includeXML(ctx context.Context, inc *helium.Element, uri string, incBase string) error {
	doc, err := p.loadXMLDoc(ctx, uri, incBase, false)
	if err != nil {
		return err
	}

	p.mergeEntities(ctx, doc, inc.OwnerDocument())

	// Collect top-level children from included document, skipping DTD nodes
	var nodes []helium.Node
	for c := range helium.Children(doc) {
		if c.Type() == helium.DTDNode {
			continue
		}
		nodes = append(nodes, c)
	}

	if len(nodes) == 0 {
		helium.UnlinkNode(inc)
		p.count++
		return nil
	}

	if err := checkMultiRootInclusion(inc, nodes); err != nil {
		return err
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
	return p.recurseIncluded(ctx, nodes, uri, uri)
}

// recurseIncluded processes included nodes for nested xi:include while a
// temporary expansion state is in effect: it records key in p.expanding for
// circular detection, increments depth, and (when base != "") sets the base
// URI so relative hrefs in the included content resolve correctly. A base of
// "" leaves p.baseURI unchanged. All three pieces of state are restored on
// return via defer; the fields are independent so the reverse-order restore
// is observably equivalent to restoring them in any order.
func (p *processor) recurseIncluded(ctx context.Context, nodes []helium.Node, key, base string) error {
	savedBase := p.baseURI
	if base != "" {
		p.baseURI = base
	}
	p.expanding[key] = true
	p.depth++
	defer func() {
		p.depth--
		delete(p.expanding, key)
		p.baseURI = savedBase
	}()

	for _, n := range nodes {
		if n.Type() == helium.ElementNode {
			if err := p.processNode(ctx, n); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *processor) includeXMLWithXPointer(ctx context.Context, inc *helium.Element, uri string, xptrExpr string, incBase string) error {
	var doc *helium.Document
	var err error

	if uri == "" {
		// Same-document reference. It must be evaluated against the original
		// document infoset (before any nodes were inserted by earlier XInclude
		// processing in this pass), but it needs no filesystem access, so it
		// must NOT go through the resolver — doing so would re-load the base
		// URI and fail under the deny-all default resolver.
		//
		// For a top-level same-document reference we use the in-memory snapshot
		// of the entry document taken before processing began. Inside an
		// included document (p.baseURI has been switched to that document's URI
		// during recursion) the original bytes are available via loadXMLDoc's
		// cache, so re-parse from there; this only reaches the resolver when an
		// FS-backed resolver was already used to load that included document.
		switch {
		case p.snapshot != nil && p.baseURI == p.snapshotURI:
			doc = p.snapshot
		case p.baseURI != "":
			doc, err = p.loadXMLDoc(ctx, p.baseURI, p.baseURI, true)
			if err != nil {
				return err
			}
		default:
			doc = inc.OwnerDocument()
		}
	} else {
		doc, err = p.loadXMLDoc(ctx, uri, incBase, true)
		if err != nil {
			return err
		}
	}

	p.mergeEntities(ctx, doc, inc.OwnerDocument())

	// Evaluate XPointer expression against the document
	nodes, err := xpointer.Evaluate(ctx, doc, xptrExpr)
	if err != nil {
		return fmt.Errorf("xi:include: XPointer evaluation failed: %w", err)
	}

	if len(nodes) == 0 {
		helium.UnlinkNode(inc)
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

	// Deep-copy result nodes into the target document. Each selected node is
	// deep-copied, and an xpointer can select up to the xpath node-set limit,
	// so over a source with many nested same-name elements (e.g. xpointer(//a))
	// the copies are O(n^2) the source size — far larger than the bytes READ
	// from the source. Charge the estimated copy footprint against the aggregate
	// bound BEFORE each copy so the same guard that bounds bytes read also bounds
	// nodes copied, failing fast instead of materializing unboundedly.
	targetDoc := inc.OwnerDocument()
	var copies []helium.Node
	for _, n := range nodes {
		cost, costErr := subtreeCopyCost(n)
		if costErr != nil {
			return fmt.Errorf("xi:include: %w", costErr)
		}
		if err := p.accountIncludedBytes(uri, cost); err != nil {
			return err
		}
		c, copyErr := helium.CopyNode(n, targetDoc)
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

	if err := checkMultiRootInclusion(inc, copies); err != nil {
		return err
	}

	p.replaceWithNodes(inc, copies)
	p.count++

	// Circular detection key. For a same-document reference (uri == "") the key
	// carries the current document identity (p.baseURI) so it matches the key
	// computed in processNode and does not collide across documents.
	circularKey := p.baseURI + "#" + xptrExpr
	if uri != "" {
		circularKey = uri + "#" + xptrExpr
	}

	// Recursively process included content for nested xi:include.
	// A same-document reference (uri == "") must leave the base URI
	// unchanged, so pass an empty base in that case.
	return p.recurseIncluded(ctx, copies, circularKey, uri)
}

// mergeEntities merges general entities from the included document's DTD
// subsets into the target document's internal subset, matching libxml2's
// xmlXIncludeMergeEntities behavior. Creates a minimal internal subset on
// the target if it doesn't have one and the source does.
func (p *processor) mergeEntities(ctx context.Context, src, dst *helium.Document) {
	srcInt := src.IntSubset()
	srcExt := src.ExtSubset()
	if srcInt == nil && srcExt == nil {
		return
	}

	// Ensure target has an internal subset
	dstInt := dst.IntSubset()
	if dstInt == nil {
		for c := range helium.Children(dst) {
			if elem, ok := helium.AsNode[*helium.Element](c); ok {
				var err error
				dstInt, err = dst.CreateInternalSubset(elem.LocalName(), "", "")
				if err != nil {
					return
				}
				break
			}
		}
		if dstInt == nil {
			return
		}
	}

	merge := func(srcDTD *helium.DTD) {
		if srcDTD == nil {
			return
		}
		srcDTD.ForEachEntity(func(name string, srcEnt *helium.Entity) {
			existing, _ := dstInt.AddEntity(name, srcEnt.EntityType(), srcEnt.ExternalID(), srcEnt.SystemID(), string(srcEnt.Content()))
			if existing == nil {
				return
			}
			// Check for definition mismatch (first-definition-wins, warn on conflict)
			if p.errorHandler == nil {
				return
			}
			mismatch := false
			if existing.EntityType() != srcEnt.EntityType() {
				mismatch = true
			} else if srcEnt.SystemID() != "" && existing.SystemID() != "" && existing.SystemID() != srcEnt.SystemID() {
				mismatch = true
			} else if srcEnt.ExternalID() != "" && existing.ExternalID() != "" && existing.ExternalID() != srcEnt.ExternalID() {
				mismatch = true
			} else if len(srcEnt.Content()) > 0 && len(existing.Content()) > 0 && string(existing.Content()) != string(srcEnt.Content()) {
				mismatch = true
			}
			if mismatch {
				p.errorHandler.Handle(ctx, helium.NewLeveledError(
					fmt.Sprintf("xi:include: entity '%s' definition mismatch", name),
					helium.ErrorLevelWarning,
				))
			}
		})
	}

	merge(srcInt)
	merge(srcExt)
}

// fetch resolves uri (against base) through the configured Resolver and reads
// its bytes, capped at the include-size limit. Errors are wrapped with the URI
// for context ("failed to resolve %q" / "error reading %q"); the caller is
// responsible for negative-caching the returned (already wrapped) error.
// readCapped fully drains the reader before returning, so closing it on return
// is safe.
func (p *processor) fetch(uri, base string) ([]byte, error) {
	rc, err := p.resolve(uri, base)
	if err != nil {
		return nil, fmt.Errorf("xi:include: failed to resolve %q: %w", uri, err)
	}
	defer func() { _ = rc.Close() }()

	data, err := p.readCapped(rc)
	if err != nil {
		return nil, fmt.Errorf("xi:include: error reading %q: %w", uri, err)
	}
	return data, nil
}

func (p *processor) loadXMLDoc(ctx context.Context, uri string, base string, substituteEntities bool) (*helium.Document, error) {
	cacheKey := uri
	if substituteEntities {
		cacheKey = uri + "\x00noent"
	}

	if entry, ok := p.docCache[cacheKey]; ok {
		if entry.err != nil {
			return nil, entry.err
		}
		// A cached resource still materializes a fresh subtree on every reuse, so
		// it counts toward the aggregate bound just like a freshly fetched one.
		if err := p.accountIncludedBytes(uri, len(entry.data)); err != nil {
			return nil, err
		}
		// Re-parse from cached bytes: each inclusion needs independent
		// nodes since they get moved into the target document tree.
		return p.parseXMLData(ctx, entry.data, uri, substituteEntities)
	}

	data, err := p.fetch(uri, base)
	if err != nil {
		p.docCache[cacheKey] = docCacheEntry{err: err}
		return nil, err
	}
	if err := p.accountIncludedBytes(uri, len(data)); err != nil {
		// Cache the bytes anyway so a later same-URI include hits the same
		// aggregate guard rather than re-fetching.
		p.docCache[cacheKey] = docCacheEntry{data: data}
		return nil, err
	}

	doc, err := p.parseXMLData(ctx, data, uri, substituteEntities)
	if err != nil {
		p.docCache[cacheKey] = docCacheEntry{err: err}
		return nil, err
	}

	// Cache raw bytes for subsequent includes of the same URI
	p.docCache[cacheKey] = docCacheEntry{data: data}
	return doc, nil
}

// resolve opens an already-resolved URI through the configured Resolver.
// uri has already been resolved against base (the include's effective base
// URI, honoring ancestor xml:base) by processInclude/resolveURI, so it is
// passed as the href to open while base is passed only as informational
// context (per the Resolver contract): resolvers MUST open href directly
// and MUST NOT resolve it against base again, which would double-apply the
// base directory (e.g. open dir/dir/inc.xml instead of dir/inc.xml).
func (p *processor) resolve(uri, base string) (io.ReadCloser, error) {
	return p.resolver.Resolve(uri, base) //nolint:wrapcheck // callers wrap with the URI for context
}

// readCapped reads all bytes from r but no more than the configured include-size
// cap, returning [ErrIncludeTooLarge] when the resource exceeds it. The cap is
// enforced against the bytes actually read so a Resolver that returns an
// endless or oversized reader cannot exhaust memory.
func (p *processor) readCapped(r io.Reader) ([]byte, error) {
	limit := p.effectiveMaxIncludeSize()

	data, exceeded, err := iolimit.ReadAll(r, int64(limit))
	if exceeded {
		return nil, ErrIncludeTooLarge
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // caller wraps with the URI for context
	}
	return data, nil
}

// effectiveMaxIncludeSize returns the per-include byte cap in effect: the
// configured [Processor.MaxIncludeSize] or defaultMaxIncludeSize when unset.
func (p *processor) effectiveMaxIncludeSize() int {
	if p.maxIncludeSize > 0 {
		return p.maxIncludeSize
	}
	return defaultMaxIncludeSize
}

// accountIncludedBytes adds the bytes materialized by one included resource to
// the running aggregate and fails once the cumulative total exceeds the
// aggregate bound (maxIncludeAggregateMultiplier × the per-include cap). It is
// called for every spliced occurrence — including repeated cache hits — so the
// bound covers both many distinct includes and one cached resource reused many
// times. uri is included for diagnostic context.
func (p *processor) accountIncludedBytes(uri string, n int) error {
	p.totalBytes += int64(n)
	if p.totalBytes > int64(p.effectiveMaxIncludeSize())*maxIncludeAggregateMultiplier {
		return fmt.Errorf("xi:include: aggregate size including %q exceeds maximum: %w", uri, ErrIncludeTooLarge)
	}
	return nil
}

// copiedNodeOverhead is a conservative per-node byte estimate of a DOM node's
// in-memory footprint (struct fields and pointers), added on top of its textual
// content so that the byte-denominated aggregate bound also meaningfully limits
// the number of nodes a deep copy materializes.
const copiedNodeOverhead = 64

// subtreeCopyCost estimates, in bytes, the memory a deep copy of n materializes:
// copiedNodeOverhead plus the textual length of every node in the subtree
// (descendants and an element's attributes) AND the namespace objects that
// helium.CopyNode allocates. It walks the SOURCE subtree — reading
// already-resident memory — so the estimate is charged BEFORE the copy is
// allocated. Container nodes' Content() is NOT used (it aggregates all
// descendants, which would make the walk itself O(n^2)); only the leaf
// text-bearing nodes contribute content length.
//
// Namespaces must be counted because CopyNode's over-declare path
// (deepCopier.bindNamespacesOverDeclare) allocates a fresh Namespace object for
// every nsDefs declaration, one more for the element's active namespace, and one
// additional over-declared object when that active namespace's prefix was not
// among the declarations; copyAttributes allocates one per namespaced attribute.
// Without this, a namespace-heavy nested source could stay under the per-include
// byte cap while xpointer(//a) multiplied copied namespace objects across
// overlapping subtrees, bypassing the aggregate materialization bound.
//
// A tree cycle in the source (ErrWalkCycle) is propagated so the caller fails
// the include rather than charging a partial cost and copying a corrupt tree.
func subtreeCopyCost(n helium.Node) (int, error) {
	var total int
	if err := helium.Walk(n, helium.NodeWalkerFunc(func(node helium.Node) error {
		total += copiedNodeOverhead
		switch node.Type() {
		case helium.ElementNode:
			elem, ok := node.(*helium.Element)
			if !ok {
				return nil
			}
			total += len(elem.Name())
			decls := elem.Namespaces()
			for _, ns := range decls {
				total += namespaceCopyCost(ns)
			}
			if ns := elem.Namespace(); ns != nil {
				// SetActiveNamespace always allocates the active namespace object.
				total += namespaceCopyCost(ns)
				// CopyNode over-declares the active namespace too when its prefix
				// is not already declared and its URI is non-empty.
				if ns.URI() != "" && !nsPrefixDeclared(decls, ns.Prefix()) {
					total += namespaceCopyCost(ns)
				}
			}
			for _, a := range elem.Attributes() {
				total += copiedNodeOverhead + len(a.Name()) + len(a.Value())
				if a.URI() != "" {
					// copyAttributes allocates a Namespace object per namespaced attr.
					total += copiedNodeOverhead + len(a.Prefix()) + len(a.URI())
				}
			}
		case helium.TextNode, helium.CDATASectionNode, helium.CommentNode, helium.ProcessingInstructionNode:
			total += len(node.Content())
		}
		return nil
	})); err != nil {
		return 0, err
	}
	return total, nil
}

// namespaceCopyCost is the estimated footprint of one copied Namespace object:
// the per-node overhead plus its prefix and URI strings.
func namespaceCopyCost(ns *helium.Namespace) int {
	return copiedNodeOverhead + len(ns.Prefix()) + len(ns.URI())
}

// nsPrefixDeclared reports whether prefix appears among the given namespace
// declarations, mirroring the declaredPrefixes set in
// deepCopier.bindNamespacesOverDeclare.
func nsPrefixDeclared(decls []*helium.Namespace, prefix string) bool {
	for _, ns := range decls {
		if ns.Prefix() == prefix {
			return true
		}
	}
	return false
}

func (p *processor) parseXMLData(ctx context.Context, data []byte, uri string, substituteEntities bool) (*helium.Document, error) {
	// Start from the caller-injected parser (for its resource limits) or a
	// default, then force XInclude's own loading policy on top: external DTD
	// loading and the per-include base URI. The FS is set below to the
	// resolver's sandbox (NOT the injected parser's FS) — see Processor.Parser.
	base := helium.NewParser()
	if p.parser != nil {
		base = *p.parser
	}
	parser := base.LoadExternalDTD(true).BaseURI(uri)
	// Thread the resolver's filesystem into the inner parser so external
	// entities and external DTDs declared inside the included document
	// resolve through the SAME sandbox as XInclude itself, not the parser's
	// default permissive filesystem. Otherwise an attacker-supplied included
	// document could expand a SYSTEM entity (e.g. "/etc/passwd") off the host
	// filesystem, bypassing a strict Resolver (XXE). The default resolver is
	// deny-all, so inner SYSTEM references are blocked unless the caller opts
	// into an FS-backed resolver; a custom (non-FS) resolver gets a deny-all FS
	// so inner SYSTEM references cannot reach the host.
	//
	// Detection uses the fsBacked capability interface rather than a concrete
	// *fsResolver assertion so callers can wrap NewFSResolver (e.g. for
	// logging/metrics) without silently losing the in-sandbox loader. The FS
	// is wrapped with normalizingFS because the parser builds the names it
	// hands to Open via filepath.Join/BuildURI — OS-native separators on
	// Windows and possibly absolute — whereas an fs.FS expects slash-separated,
	// path.Clean'd names. Without normalization a sandbox that accepts
	// XInclude hrefs would spuriously reject the document's own external
	// entities/DTDs.
	if fr, ok := p.resolver.(fsBacked); ok {
		// NewParser now blocks external entity/DTD loading by default; lift that
		// block so the included document's own external references resolve, but
		// keep them confined to the resolver's sandbox FS (set below).
		parser = parser.BlockXXE(false).FS(normalizingFS{fsys: fr.FS()})
	} else {
		// Custom (non-FS) resolver: keep the default XXE block AND deny the FS so
		// inner SYSTEM references cannot reach the host (defense in depth).
		parser = parser.FS(denyAllFS{})
	}
	if substituteEntities {
		parser = parser.SubstituteEntities(true)
	}
	doc, err := parser.Parse(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("xi:include: error parsing %q: %w", uri, err)
	}
	return doc, nil
}

func (p *processor) includeText(inc *helium.Element, uri string, incBase string) error {
	data, err := p.loadText(uri, incBase)
	if err != nil {
		return err
	}

	// Handle encoding attribute. Distinguish a missing encoding attribute
	// (process the bytes as UTF-8) from one that is present but names an
	// encoding we cannot honour. A present-but-empty encoding="" is treated
	// as an unsupported encoding rather than as absent, so the raw bytes are
	// not silently consumed as UTF-8.
	if encName, ok := findAttr(inc, "encoding"); ok {
		enc := encoding.Load(encName)
		// An unsupported requested encoding is a resource error: the raw bytes
		// must not be silently treated as UTF-8. Returning an error here lets
		// the caller's fallback handling apply (or fail the inclusion).
		if enc == nil {
			return fmt.Errorf("xi:include: unsupported encoding %q for %q", encName, uri)
		}
		decoded, decErr := enc.NewDecoder().Bytes(data)
		if decErr != nil {
			return fmt.Errorf("xi:include: error decoding %q with encoding %q: %w", uri, encName, decErr)
		}
		data = decoded
	}

	// Validate that text contains only valid XML characters
	if err := validateXMLChars(data); err != nil {
		return fmt.Errorf("xi:include: %s contains invalid char", uri)
	}

	doc := inc.OwnerDocument()
	text := doc.CreateText(data)

	p.replaceWithNodes(inc, []helium.Node{text})
	p.count++
	return nil
}

func (p *processor) loadText(uri string, base string) ([]byte, error) {
	if entry, ok := p.txtCache[uri]; ok {
		if entry.err != nil {
			return nil, entry.err
		}
		// A cached text resource is re-materialized on every reuse, so it counts
		// toward the aggregate bound just like a freshly fetched one.
		if err := p.accountIncludedBytes(uri, len(entry.data)); err != nil {
			return nil, err
		}
		return entry.data, nil
	}

	data, err := p.fetch(uri, base)
	if err != nil {
		p.txtCache[uri] = txtCacheEntry{err: err}
		return nil, err
	}

	p.txtCache[uri] = txtCacheEntry{data: data}
	if err := p.accountIncludedBytes(uri, len(data)); err != nil {
		return nil, err
	}
	return data, nil
}

func (p *processor) handleFallback(inc *helium.Element, origErr error) error {
	nsURI := getNamespaceURI(inc)
	for c := range helium.Children(inc) {
		if c.Type() == helium.ElementNode {
			if elem, ok := c.(*helium.Element); ok {
				if elem.LocalName() == lexicon.XSLTElementFallback && getNamespaceURI(elem) == nsURI {
					return p.processFallback(inc, elem)
				}
			}
		}
	}
	return origErr
}

func (p *processor) processFallback(inc *helium.Element, fb *helium.Element) error { //nolint:unparam // always nil but callers check for future-proofing
	var nodes []helium.Node
	for c := range helium.Children(fb) {
		nodes = append(nodes, c)
	}

	if len(nodes) == 0 {
		helium.UnlinkNode(inc)
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
		helium.UnlinkNode(target)
		return
	}

	// Detach nodes from their original parents
	for _, n := range nodes {
		if n.Parent() != nil {
			helium.UnlinkNode(n.(helium.MutableNode)) //nolint:forcetypeassert
		}
	}

	if !p.noMarkers {
		doc := target.OwnerDocument()
		startMarker := newXIncludeMarker(doc, helium.XIncludeStartNode, target.Name())
		endMarker := newXIncludeMarker(doc, helium.XIncludeEndNode, target.Name())

		expanded := make([]helium.Node, 0, len(nodes)+2)
		expanded = append(expanded, startMarker)
		expanded = append(expanded, nodes...)
		expanded = append(expanded, endMarker)
		nodes = expanded
	}

	spliceReplace(target, nodes)
}

func spliceReplace(target helium.MutableNode, nodes []helium.Node) {
	if len(nodes) == 0 {
		helium.UnlinkNode(target)
		return
	}
	_ = target.Replace(nodes...)
}

func isXINamespace(ns string) bool {
	return ns == lexicon.NamespaceXInclude || ns == lexicon.NamespaceXInclude11
}

func isFallback(n helium.Node) bool {
	if n.Type() != helium.ElementNode {
		return false
	}
	elem, ok := n.(*helium.Element)
	if !ok {
		return false
	}
	return elem.LocalName() == "fallback" && isXINamespace(getNamespaceURI(elem))
}

func validateIncludeChildren(inc *helium.Element) error {
	nsURI := getNamespaceURI(inc)
	var fallbackCount int
	for c := range helium.Children(inc) {
		if c.Type() != helium.ElementNode {
			continue
		}
		elem, ok := c.(*helium.Element)
		if !ok {
			continue
		}
		cNS := getNamespaceURI(elem)
		if elem.LocalName() == "include" && isXINamespace(cNS) {
			return fmt.Errorf("xi:include has an 'include' child")
		}
		if elem.LocalName() == "fallback" && cNS == nsURI {
			fallbackCount++
			if fallbackCount > 1 {
				return fmt.Errorf("xi:include has multiple fallback children")
			}
		}
	}
	return nil
}

func checkMultiRootInclusion(inc *helium.Element, nodes []helium.Node) error {
	parent := inc.Parent()
	if parent == nil || parent.Type() == helium.ElementNode {
		return nil
	}
	// Parent is a Document node — count element nodes in replacement
	var elemCount int
	for _, n := range nodes {
		if n.Type() == helium.ElementNode {
			elemCount++
		}
	}
	if elemCount > 1 {
		return fmt.Errorf("xi:include: would result in multiple root nodes")
	}
	if elemCount == 0 {
		return fmt.Errorf("xi:include: would result in no root node")
	}
	return nil
}

// ownerDocument returns the document owning n, or n itself when n is a Document.
func ownerDocument(n helium.Node) *helium.Document {
	if n == nil {
		return nil
	}
	if doc, ok := helium.AsNode[*helium.Document](n); ok {
		return doc
	}
	return n.OwnerDocument()
}

// snapshotForXPointer builds the resolver-free, ID-preserving snapshot of the
// entry document used to evaluate top-level same-document XPointer references
// against the original infoset (before inclusions mutate the tree).
//
// helium.CopyDoc alone is insufficient: it reproduces the tree and the internal
// DTD subset but drops the ID-resolution state that XPointer shorthand pointers
// depend on (they resolve via Document.GetElementByID). Without carrying that
// state a source parsed with SkipIDs(true) would wrongly START resolving
// xml:id/ID attributes in the copy, and a source whose ids were declared in an
// external DTD subset would resolve FEWER ids than the original. So the snapshot
// additionally:
//
//   - carries the source's SkipIDs state (authoritative for GetElementByID), so
//     a SkipIDs source yields a snapshot that resolves NO ids;
//   - carries the source's external DTD subset (CopyDoc copies only the internal
//     subset), which GetElementByID's lazy fallback consults for ID-typed
//     attribute declarations;
//   - rebuilds the copy's interned ID table by translating each source ID entry's
//     element through the source->copy element correspondence, so a parsed
//     source's ids resolve to the copy's elements rather than the source's.
//
// Returns nil when the copy fails; the caller then falls back to evaluating
// against the live (possibly mutated) document.
func snapshotForXPointer(doc *helium.Document) *helium.Document {
	snap, err := helium.CopyDoc(doc)
	if err != nil {
		return nil
	}

	// Authoritative ID-skip state first: a SkipIDs source must resolve no ids in
	// the snapshot either.
	snap.SetSkipIDs(doc.SkipIDs())

	// CopyDoc copies only the internal subset; carry the external subset too so
	// the lazy GetElementByID fallback sees the same ID-typed ATTLIST decls.
	helium.CopyExtSubset(doc, snap)

	// Rebuild the interned ID table by element correspondence. Skip when the
	// source skips ids (nothing resolves) or has no interned table (an API-built
	// source relies on the lazy fallback, already reproduced via the DTD subsets).
	srcIDs := doc.IDTable()
	if doc.SkipIDs() || len(srcIDs) == 0 {
		return snap
	}

	// CopyDoc reproduces the element spine 1:1 in document order, so a parallel
	// pre-order walk of both trees yields the source->copy element correspondence.
	var srcElems, cpElems []*helium.Element
	collectElementsPreorder(doc, &srcElems)
	collectElementsPreorder(snap, &cpElems)
	if len(srcElems) != len(cpElems) {
		// Defensive: shapes diverged unexpectedly; skip the rebuild rather than
		// risk mismapping ids onto the wrong elements.
		return snap
	}
	idx := make(map[*helium.Element]*helium.Element, len(srcElems))
	for i := range srcElems {
		idx[srcElems[i]] = cpElems[i]
	}
	for id, srcElem := range srcIDs {
		if cp := idx[srcElem]; cp != nil {
			snap.RegisterID(id, cp)
		}
	}
	return snap
}

// collectElementsPreorder appends every element descendant of n (document order,
// pre-order) to out. It descends only through element nodes — matching
// helium.CopyDoc, which does not copy elements nested inside entity-reference
// expansions — so the source and copy walks stay aligned.
func collectElementsPreorder(n helium.Node, out *[]*helium.Element) {
	for c := range helium.Children(n) {
		if elem, ok := helium.AsNode[*helium.Element](c); ok {
			*out = append(*out, elem)
			collectElementsPreorder(elem, out)
		}
	}
}

// hasSameDocumentXPointer reports whether n or any descendant is an xi:include
// that references the current document via an XPointer (no href, or an href
// that is only a fragment). Used to decide whether a snapshot of the entry
// document is needed before processing begins.
func hasSameDocumentXPointer(n helium.Node) bool {
	if isXInclude(n) {
		if elem, ok := helium.AsNode[*helium.Element](n); ok && isSameDocumentInclude(elem) {
			return true
		}
	}
	for c := range helium.Children(n) {
		if hasSameDocumentXPointer(c) {
			return true
		}
	}
	return false
}

// isSameDocumentInclude reports whether elem is an xi:include whose target is
// the current document: it has an XPointer (via the xpointer attribute or an
// href fragment) and no document-selecting href.
func isSameDocumentInclude(elem *helium.Element) bool {
	href := getAttr(elem, "href")
	xptr := getAttr(elem, "xpointer")
	var fragment string
	if idx := strings.IndexByte(href, '#'); idx >= 0 {
		fragment = href[idx+1:]
		href = href[:idx]
	}
	if href != "" {
		return false
	}
	return xptr != "" || fragment != ""
}

func isXInclude(n helium.Node) bool {
	if n.Type() != helium.ElementNode {
		return false
	}
	elem, ok := n.(*helium.Element)
	if !ok {
		return false
	}
	return elem.LocalName() == "include" && isXINamespace(getNamespaceURI(elem))
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

// getAttr performs a 3-pass attribute lookup matching libxml2's xmlXIncludeGetProp:
// 1. Try attribute with the element's XInclude namespace URI
// 2. Try attribute with the other XInclude namespace URI
// 3. Try unqualified attribute (no namespace)
func getAttr(elem *helium.Element, name string) string {
	val, _ := findAttr(elem, name)
	return val
}

// findAttr performs the same 3-pass lookup as getAttr but also reports whether
// the attribute was present. This distinguishes a missing attribute from one
// that is present with an empty value, which getAttr alone cannot do.
func findAttr(elem *helium.Element, name string) (string, bool) {
	elemNS := getNamespaceURI(elem)
	var otherNS string
	if elemNS == lexicon.NamespaceXInclude {
		otherNS = lexicon.NamespaceXInclude11
	} else {
		otherNS = lexicon.NamespaceXInclude
	}

	attrs := elem.Attributes()

	// Pass 1: element's own XInclude namespace
	for _, a := range attrs {
		if a.LocalName() == name && a.URI() == elemNS {
			return a.Value(), true
		}
	}
	// Pass 2: the other XInclude namespace
	for _, a := range attrs {
		if a.LocalName() == name && a.URI() == otherNS {
			return a.Value(), true
		}
	}
	// Pass 3: unqualified (no namespace)
	for _, a := range attrs {
		if a.LocalName() == name && a.URI() == "" {
			return a.Value(), true
		}
	}
	return "", false
}

func resolveURI(href, base string) (string, error) {
	// A native Windows base ("D:\\dir\\main.xml", "D:/dir/main.xml", or a UNC
	// "\\host\\share") is a local filesystem path, not a URI. url.Parse would
	// read its drive letter "D" as a URI scheme and emit garbage like
	// "d:///fragment.xml", so resolve it with local-path (forward-slash)
	// semantics BEFORE the URI machinery. The shape is detected from the string
	// alone (uripath), so this branch is exercised on POSIX too.
	if uripath.IsWindowsAbsolute(base) {
		hrefURL, perr := url.Parse(href)
		if perr != nil {
			return "", fmt.Errorf("xi:include: invalid href %q: %w", href, perr)
		}
		if hrefURL.IsAbs() {
			return href, nil
		}
		slashBase := uripath.ToSlash(base)
		return path.Join(path.Dir(slashBase), href), nil
	}

	hrefURL, err := url.Parse(href)
	if err != nil {
		return "", fmt.Errorf("xi:include: invalid href %q: %w", href, err)
	}

	if hrefURL.IsAbs() {
		return href, nil
	}

	if base == "" {
		return href, nil
	}

	// For file-like paths (no scheme), use slash-based resolution
	// to avoid Go's url.ResolveReference quirk that adds leading '/'
	// to purely relative paths.
	// If base fails to parse as a URL, fall back to returning href unresolved.
	baseURL, parseErr := url.Parse(base)
	if parseErr != nil {
		return href, nil //nolint:nilerr // intentional fallback: unparseable base is treated as absent
	}
	if baseURL.Scheme == "" || baseURL.Scheme == lexicon.SchemeFile {
		basePath := baseURL.Path
		if basePath == "" {
			basePath = base
		}
		// Join with forward-slash (path) semantics so the result uses '/' on
		// every OS; on Windows filepath.Dir/Join would emit '\'.
		return path.Join(path.Dir(uripath.ToSlash(basePath)), href), nil
	}

	return baseURL.ResolveReference(hrefURL).String(), nil
}

// existingXMLBase returns the value of elem's xml:base attribute and whether
// it is present.
func existingXMLBase(elem *helium.Element) (string, bool) {
	for _, a := range elem.Attributes() {
		if a.Name() == lexicon.QNameXMLBase {
			return a.Value(), true
		}
	}
	return "", false
}

// setXMLBase sets elem's xml:base attribute to base, but only when base is
// non-empty; an empty base is a no-op so callers can pass the result of
// relativeURI directly.
func setXMLBase(elem *helium.Element, base string) {
	if base == "" {
		return
	}
	xmlBaseNS := helium.NewNamespace(lexicon.PrefixXML, lexicon.NamespaceXML)
	_ = elem.SetLiteralAttributeNS("base", base, xmlBaseNS)
}

// computeAndSetBaseURI computes the relative URI of the included resource
// against the target document's base, and sets xml:base only when needed.
// Used for whole-document XML inclusion (includeXML).
func computeAndSetBaseURI(elem *helium.Element, includedURI, targetBase string) {
	// If the included element already has xml:base set, leave it alone
	if _, ok := existingXMLBase(elem); ok {
		return
	}

	// Compute relative URI if possible
	setXMLBase(elem, relativeURI(includedURI, targetBase))
}

// computeBaseForIncludedNode sets xml:base on a node that was included via
// XPointer. srcEffectiveBase is the absolute effective base of the source
// node (computed from its ancestor xml:base chain). targetEffectiveBase is
// the effective base at the xi:include point in the target document.
func computeBaseForIncludedNode(elem *helium.Element, srcEffectiveBase, targetEffectiveBase string) {
	// Check if this element has an existing xml:base attribute
	if existingBase, ok := existingXMLBase(elem); ok && existingBase != "" {
		// Element has xml:base in the source. If absolute, keep it.
		if u, err := url.Parse(existingBase); err == nil && u.IsAbs() {
			return
		}
		// The element's xml:base was relative to the source context.
		// srcEffectiveBase already incorporates this element's xml:base,
		// so relativize it against the target's effective base.
		setXMLBase(elem, relativeURI(srcEffectiveBase, targetEffectiveBase))
		return
	}

	// No xml:base — set one relative to the target's effective base.
	setXMLBase(elem, relativeURI(srcEffectiveBase, targetEffectiveBase))
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

// makeRelativePath computes a relative path from basePath's directory to
// targetPath. The computation is pure forward-slash (RFC 3986 dot-segment)
// semantics via [uripath.SlashRel], NOT filepath.Rel: the result is an xml:base
// relative reference that MUST be byte-identical on POSIX and Windows (where
// filepath.Rel splits on '\', mishandles '/'-bearing inputs, and applies
// drive-letter rules, producing wrong "../" sequences).
func makeRelativePath(targetPath, basePath string) string {
	baseDir := uripath.SlashDir(basePath)
	if baseDir == "." {
		// Base has no directory component — target is already relative.
		return uripath.ToSlash(targetPath)
	}
	return uripath.SlashRel(baseDir, targetPath)
}

func newXIncludeMarker(doc *helium.Document, etype helium.ElementType, name string) helium.Node {
	return helium.NewXIncludeMarker(doc, etype, name)
}

// fixupNamespaceDecls ensures that elements being moved out of their declaring
// context carry their own namespace declarations. For each element in the subtree,
// if it has an active namespace whose prefix is not declared in the element's own
// nsDefs, a declaration is added.
func fixupNamespaceDecls(n helium.Node) {
	if n.Type() != helium.ElementNode {
		return
	}
	elem, ok := helium.AsNode[*helium.Element](n)
	if !ok {
		return
	}

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
				_ = elem.DeclareNamespace(ns.Prefix(), ns.URI())
			}
		}
	}

	// Recurse into children
	for c := range helium.Children(elem) {
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
				if a.Name() == lexicon.QNameXMLBase {
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
	for _, v := range slices.Backward(bases) {
		result = resolveBase(result, v)
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

	// A native Windows base ("D:\\dir\\doc.xml", "D:/dir/doc.xml", or a UNC path)
	// is a local filesystem path, not a URI. url.Parse would read its drive
	// letter as a scheme and emit garbage like "d:///one/two", so resolve it with
	// local-path (forward-slash) semantics, matching resolveURI's Windows branch.
	// xmlBase is relative here (absolute was handled above), so its directory is
	// dropped and the new base segment is appended. The shape is detected from
	// the string alone, so the branch runs on POSIX too.
	if uripath.IsWindowsAbsolute(currentBase) {
		slashBase := uripath.ToSlash(currentBase)
		const syntheticPrefix = "synthetic://h/"
		absBase, perr := url.Parse(syntheticPrefix + slashBase)
		if perr != nil {
			return path.Join(path.Dir(slashBase), xmlBase)
		}
		resolved := absBase.ResolveReference(xmlBaseURL)
		return strings.TrimPrefix(resolved.String(), syntheticPrefix)
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
		// Forward-slash fallback (never filepath.Join, which emits '\' on
		// Windows): drop the base's last segment and append the relative base.
		return uripath.JoinLocalBaseDir(uripath.SlashDir(currentBase), xmlBase)
	}
	resolved := absBase.ResolveReference(xmlBaseURL)
	result := strings.TrimPrefix(resolved.String(), syntheticPrefix)
	return result
}

// computeFixupBases computes relative URI bases for xml:base fixup.
// When both the source and target document URIs are absolute paths,
// they are converted to relative paths against their common ancestor
// directory. This ensures that ".." traversal in xml:base attributes
// is bounded at the logical root, matching RFC 3986 URI resolution.
//
// The relativization is pure forward-slash ([uripath.SlashCommonDir] +
// [uripath.SlashRel]), NOT filepath.*: sourceURI arrives in forward-slash form
// (resolveURI normalizes it) while p.baseURI is a native OS path, so on Windows
// they mix '/' and '\'. filepath.Clean/Dir/Rel then split on '\', fail to find
// the common ancestor, and emit wrong "../" sequences. uripath normalizes both
// to '/' first so the output is byte-identical on POSIX and Windows.
func (p *processor) computeFixupBases(inc *helium.Element, sourceURI string) (string, string) {
	relSource := uripath.ToSlash(sourceURI)
	relTarget := uripath.ToSlash(p.baseURI)

	if uripath.IsAbsolutePath(sourceURI) && uripath.IsAbsolutePath(p.baseURI) {
		root := uripath.SlashCommonDir(sourceURI, p.baseURI)
		relSource = uripath.SlashRel(root, sourceURI)
		relTarget = uripath.SlashRel(root, p.baseURI)
	}

	return relSource, effectiveBaseURI(inc, relTarget)
}

// isFileURI reports whether href is an absolute "file:" URI (e.g.
// "file:///tmp/inc.xml"). Such hrefs must be converted to a local path before
// being handed to an [fs.FS]; plain OS paths and other schemes are not.
func isFileURI(href string) bool {
	u, err := url.Parse(href)
	if err != nil {
		return false
	}
	return u.Scheme == lexicon.SchemeFile
}

// fsResolver resolves URIs by reading from an [fs.FS]. Passing a nil fsys to
// [NewFSResolver] yields [iofs.PermissiveRoot], which opens any OS path
// verbatim; callers handling untrusted input should construct one with a
// stricter fs.FS via [NewFSResolver]. Note that a Processor with no resolver
// configured does NOT use this permissive form — it denies all access (see
// [Processor.Resolver]).
type fsResolver struct {
	fsys fs.FS
}

// fsBacked is an optional capability interface that a [Resolver] may
// implement to expose the [fs.FS] it reads from. When a resolver is
// fsBacked, XInclude threads that same FS into the inner parser so external
// entities and DTDs declared inside an included document load through the
// SAME sandbox (rather than the host filesystem behind the resolver's back).
// Resolvers that wrap [NewFSResolver] (e.g. for logging or metrics) should
// also implement this so the in-sandbox loader is preserved through the wrap.
type fsBacked interface {
	FS() fs.FS
}

// FS implements [fsBacked], exposing the underlying filesystem so XInclude
// can reuse it for the inner parser's external entity/DTD resolution.
func (r *fsResolver) FS() fs.FS { return r.fsys }

// normalizingFS adapts an [fs.FS] so it tolerates the OS-native, possibly
// absolute names the helium parser builds via [filepath.Join]/BuildURI for
// external entities and DTDs. It converts separators to forward slashes and
// canonicalizes with [path.Clean] before delegating to the wrapped FS, so an
// FS (os.DirFS, fstest.MapFS, os.OpenRoot) that already accepts XInclude
// hrefs sees the document's own external references in the same shape. A
// remaining ".." prefix still surfaces to the wrapped FS, preserving the
// ".."-rejection backstop. Confinement against escaping symlinks is the
// wrapped FS's responsibility — os.DirFS does not provide it (see
// [NewFSResolver]).
type normalizingFS struct {
	fsys fs.FS
}

// Open implements [fs.FS].
func (n normalizingFS) Open(name string) (fs.File, error) {
	// An included document parsed with BaseURI("file:///...") resolves its own
	// external DTDs/entities against that base, yielding "file:" URIs (e.g.
	// "file:///dir/decl.dtd") rather than plain paths. Convert those to a local
	// path here, mirroring fsResolver.Resolve, so nested external-DTD/entity
	// resolution succeeds uniformly. Non-local-host file URIs are rejected.
	if isFileURI(name) {
		p, err := iofs.FileURIToPath(name)
		if err != nil {
			return nil, fmt.Errorf("xinclude: %w", err)
		}
		return n.fsys.Open(filepath.ToSlash(p)) //nolint:wrapcheck // passthrough; underlying FS errors propagate verbatim
	}
	return n.fsys.Open(path.Clean(filepath.ToSlash(name))) //nolint:wrapcheck // passthrough; underlying FS errors propagate verbatim
}

// denyAllFS is an fs.FS that refuses every open. It is threaded into the inner
// parser used for included documents when the XInclude resolver is a custom
// (non-fsResolver) implementation, so external entities/DTDs in the included
// document cannot reach the host filesystem behind the resolver's back.
type denyAllFS struct{}

func (denyAllFS) Open(string) (fs.File, error) { return nil, fs.ErrNotExist }

func (r *fsResolver) Resolve(href, _ string) (io.ReadCloser, error) {
	// href is already fully resolved against the include's effective base
	// by the processor (see the Resolver contract), so it is opened
	// directly — base is intentionally ignored. Joining base here would
	// double-apply the base directory (e.g. dir/dir/inc.xml).
	//
	// An absolute "file:" href (e.g. from <xi:include href="file:///tmp/x"/>)
	// is a URI, not a path; handing it to fs.FS verbatim would clean it to
	// "file:/tmp/x" and fail. Convert it to a local filesystem path first,
	// matching the catalog loader (PR #602). Non-local-host file URIs are
	// rejected by the helper.
	if isFileURI(href) {
		p, err := iofs.FileURIToPath(href)
		if err != nil {
			return nil, fmt.Errorf("xi:include: %w", err)
		}
		return r.fsys.Open(filepath.ToSlash(p)) //nolint:wrapcheck // resolver errors propagate to caller verbatim
	}

	// Upstream resolution (resolveURI) may emit OS-specific separators via
	// filepath.Join on Windows, so normalize to slashes for fs.FS, which
	// requires slash-separated names.
	p := filepath.ToSlash(href)
	// fs.FS uses slash-separated paths; using path (not filepath) keeps
	// names valid on Windows. path.Clean canonicalizes so the configured
	// fs.FS (os.DirFS, fstest.MapFS, os.OpenRoot) sees consistent input,
	// and any remaining ".." prefix means the resolved name ascended above
	// the FS root and will be rejected by an fs.ValidPath-enforcing FS.
	// Note this only blocks ".." traversal; symlink confinement depends on
	// the FS (os.DirFS follows escaping symlinks, os.OpenRoot does not) —
	// see [NewFSResolver].
	return r.fsys.Open(path.Clean(p)) //nolint:wrapcheck // resolver errors propagate to caller verbatim
}
