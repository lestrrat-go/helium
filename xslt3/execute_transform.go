package xslt3

import (
	"context"
	"maps"
	"strings"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
)

// remapValidationNode resolves a node from the original (validated) source tree to
// the corresponding node in the stripped copy the transform runs on. Strict source
// validation runs on the original tree, so its type annotations and nilled flags are
// keyed by original nodes; the transform navigates the copy, so those keys must be
// remapped. When nodeMap is nil (no strip rules — original and copy are the same
// tree) or the node has no copy (e.g. it was omitted from the copy), the node is
// returned unchanged.
func remapValidationNode(nodeMap map[helium.Node]helium.Node, node helium.Node) helium.Node {
	if nodeMap == nil || node == nil {
		return node
	}
	if mapped, ok := nodeMap[node]; ok {
		return mapped
	}
	return node
}

// remapSelectionToCopy rewrites the node items of an initial match selection so
// that any node belonging to the original source document points instead to the
// corresponding node in the stripped copy. Items that are not nodes, or that
// belong to a different document (e.g. fn:doc()-loaded), are passed through
// unchanged. A node that DID belong to the original source but has no copy (a
// whitespace-only text/CDATA node that strip-space omitted) is DROPPED from the
// selection, so it neither points at the unstripped original nor inflates the
// sequence length that drives position()/last() in the apply loop.
//
// src is the original source document the selection was computed against; it is
// used only to distinguish an omitted source node (drop) from a node in another
// document (pass through).
//
// nodeMap is the original->copy correspondence built by copyAndStrip during the
// single-pass copy. Because whitespace-only nodes are omitted from the copy, the
// two trees no longer share a child shape, so the map (not a parallel walk) is
// the only correct source of correspondence. It already covers elements,
// text/comment/PI leaves, the document node, and each element's attributes (keyed
// by expanded name). Namespace nodes are wrappers off the child spine; a selected
// *helium.NamespaceNodeWrapper is remapped by locating the mapped owner element
// and rebuilding an equivalent wrapper bound to a matching namespace declaration
// in scope on the copy. This ensures every selected node (element, text, comment,
// PI, attribute, namespace) points into the stripped copy, so XPath navigation
// from a matched template observes the same tree as the context node.
func remapSelectionToCopy(sel xpath3.Sequence, src *helium.Document, nodeMap map[helium.Node]helium.Node) xpath3.Sequence {
	items := make(xpath3.ItemSlice, 0, sequence.Len(sel))
	for i := range sequence.Len(sel) {
		item := sel.Get(i)
		ni, ok := item.(xpath3.NodeItem)
		if !ok {
			items = append(items, item)
			continue
		}
		// Namespace nodes are wrappers off the child spine: remap by relocating
		// the owner element onto the copy and rebuilding an equivalent wrapper.
		if nsw, ok := ni.Node.(*helium.NamespaceNodeWrapper); ok {
			if mapped := remapNamespaceWrapper(nodeMap, nsw); mapped != nil {
				items = append(items, xpath3.NodeItem{Node: mapped})
				continue
			}
			items = append(items, item)
			continue
		}
		if mapped, found := nodeMap[ni.Node]; found {
			items = append(items, xpath3.NodeItem{Node: mapped})
			continue
		}
		// No copy exists. If the node belonged to the original source, strip-space
		// omitted it (whitespace-only text/CDATA): drop it so it is absent from the
		// stripped selection and does not skew position()/last(). Nodes from other
		// documents (e.g. fn:doc()-loaded) keep their identity and pass through.
		if ni.Node != nil && ni.Node.OwnerDocument() == src {
			continue
		}
		items = append(items, item)
	}
	return items
}

// dropSelectionItems removes from an initial match selection any NodeItem whose
// node is in the removed set. It is used after an IN-PLACE whitespace strip of
// the (already-copied) source: a selected whitespace-only node the strip unlinked
// must be dropped so it is absent from the selection and does not skew
// position()/last() in the apply-templates loop. Non-node items and retained
// nodes pass through unchanged.
func dropSelectionItems(sel xpath3.Sequence, removed map[helium.Node]struct{}) xpath3.Sequence {
	items := make(xpath3.ItemSlice, 0, sequence.Len(sel))
	for i := range sequence.Len(sel) {
		item := sel.Get(i)
		if ni, ok := item.(xpath3.NodeItem); ok {
			if _, gone := removed[ni.Node]; gone {
				continue
			}
		}
		items = append(items, item)
	}
	return items
}

// mapElementAttributes records the correspondence between the attributes of an
// original element and those of its copy, keyed by expanded (URI, local) name
// (unique within a single element).
func mapElementAttributes(nodeMap map[helium.Node]helium.Node, ea, eb *helium.Element) {
	copyAttrs := make(map[string]*helium.Attribute)
	for _, attr := range eb.Attributes() {
		copyAttrs[helium.ClarkName(attr.URI(), attr.LocalName())] = attr
	}
	for _, attr := range ea.Attributes() {
		if dst, ok := copyAttrs[helium.ClarkName(attr.URI(), attr.LocalName())]; ok {
			nodeMap[attr] = dst
		}
	}
}

// remapNamespaceWrapper rebuilds an initial-selection namespace node so that it
// belongs to the copied tree. It maps the wrapper's owner element onto the copy,
// then returns a fresh wrapper bound to a matching namespace, with the COPIED
// owner as parent. Returns nil only if the owner element could not be mapped.
//
// The wrapper is re-bound by preference to an in-scope declaration on the copied
// owner (or an ancestor) with the same prefix and URI, since namespace nodes can
// be reported on a descendant of the element that declares them. When no such
// declaration exists, the namespace is one the XPath axis SYNTHESIZES rather than
// one stored in Namespaces() — most notably the implicit `xml` binding, which the
// namespace axis fabricates and which never appears in any element's declarations
// (see internal/xpath/axes.go CollectNamespaceNodes). For those, synthesize an
// equivalent Namespace bound to the copied owner, mirroring the axis, so the
// wrapper still points into the stripped copy instead of falling back to the
// unstripped original.
func remapNamespaceWrapper(nodeMap map[helium.Node]helium.Node, nsw *helium.NamespaceNodeWrapper) helium.Node {
	owner := nsw.Parent()
	if owner == nil {
		return nil
	}
	mappedOwner, ok := nodeMap[owner]
	if !ok {
		return nil
	}
	prefix := nsw.Name()
	uri := string(nsw.Content())
	// Prefer an actual in-scope declaration on the copied owner or an ancestor
	// that matches the original wrapper's (prefix, URI).
	for n := mappedOwner; n != nil; n = n.Parent() {
		nc, ok := n.(helium.NamespaceContainer)
		if !ok {
			continue
		}
		for _, ns := range nc.Namespaces() {
			if ns.Prefix() == prefix && ns.URI() == uri {
				return helium.NewNamespaceNodeWrapper(ns, mappedOwner)
			}
		}
	}
	// No stored declaration matches: the namespace is synthesized by the axis
	// (e.g. the implicit `xml` binding). Re-synthesize it against the copied
	// owner so the wrapper still belongs to the stripped copy.
	return helium.NewNamespaceNodeWrapper(helium.NewNamespace(prefix, uri), mappedOwner)
}

// cloneOutputDef returns a deep copy of an OutputDef. All pointer, slice, and
// map fields are freshly allocated so that mutating the clone (or the original)
// never affects the other. Returns nil if src is nil.
func cloneOutputDef(src *OutputDef) *OutputDef {
	if src == nil {
		return nil
	}
	cp := *src
	if src.ResolvedCharMap != nil {
		cp.ResolvedCharMap = make(map[rune]string, len(src.ResolvedCharMap))
		maps.Copy(cp.ResolvedCharMap, src.ResolvedCharMap)
	}
	if src.UseCharacterMaps != nil {
		cp.UseCharacterMaps = append([]string(nil), src.UseCharacterMaps...)
	}
	if src.CDATASections != nil {
		cp.CDATASections = append([]string(nil), src.CDATASections...)
	}
	if src.SuppressIndentation != nil {
		cp.SuppressIndentation = append([]string(nil), src.SuppressIndentation...)
	}
	if src.IncludeContentType != nil {
		v := *src.IncludeContentType
		cp.IncludeContentType = &v
	}
	if src.ItemSeparator != nil {
		v := *src.ItemSeparator
		cp.ItemSeparator = &v
	}
	if src.EscapeURIAttributes != nil {
		v := *src.EscapeURIAttributes
		cp.EscapeURIAttributes = &v
	}
	if src.BuildTree != nil {
		v := *src.BuildTree
		cp.BuildTree = &v
	}
	return &cp
}

// executeTransform performs the XSLT transformation.
func executeTransform(ctx context.Context, source *helium.Document, ss *Stylesheet, cfg *transformConfig) (*helium.Document, error) {
	if ss == nil {
		return nil, errNilStylesheet
	}
	resultDoc := helium.NewDefaultDocument()
	// The global context item and the initial match selection are SEPARATE
	// (XSLT 3.0 §5.4): xsl:global-context-item use="absent" makes only the GLOBAL
	// CONTEXT ITEM absent — a global "." reference raises XPDY0002, handled via
	// ec.globalContextAbsent below. It must NOT discard the source tree, which is
	// still the initial match selection for apply-templates (otherwise a transform
	// with a source-node and no initial-template would wrongly raise XTDE0040).
	effectiveSource := source

	// The caller's source document is owned by the caller and must never be
	// mutated in place: doing so destroys node identity, corrupts a tree the
	// caller may reuse (e.g. for a later XPath query or a second transform), and
	// is unsafe under concurrent reuse. Two operations would otherwise mutate it:
	// xsl:strip-space / schema-aware whitespace stripping (which removes
	// whitespace-only text nodes), and source-schema validation (which inserts
	// default/fixed attributes). Both therefore run against a private deep copy
	// that becomes the exec context's source, so the initial context node and all
	// node identity stay consistent. A copy is built only when validation or
	// stripping actually has work to do.
	var matchSelection xpath3.Sequence
	if cfg != nil {
		matchSelection = cfg.initialMatchSelection
	}
	// selectionSupplied records whether the caller supplied an initial match
	// selection at all, independent of how many nodes it currently holds. After
	// strip-space remaps the selection (remapSelectionToCopy), an all-whitespace
	// selection can shrink to length 0. In that case the apply-templates step
	// must run against the (empty) selection — producing no output — rather than
	// falling through to the source document.
	selectionSupplied := matchSelection != nil

	captureItems := cfg != nil && cfg.rawCapture
	if defOut := ss.outputs[""]; defOut != nil && isItemSerializationMethod(defOut.Method) {
		captureItems = true
	}
	ec := &execContext{
		stylesheet:          ss,
		sourceDoc:           effectiveSource,
		resultDoc:           resultDoc,
		globalVars:          make(map[string]xpath3.Sequence),
		currentMode:         "",
		outputStack:         []*outputFrame{{doc: resultDoc, current: resultDoc, itemSeparator: ss.defaultItemSeparator(), captureItems: captureItems}},
		keyTables:           make(map[string]*keyTable),
		docCache:            make(map[string]*helium.Document),
		functionResultCache: make(map[string]xpath3.Sequence),
		accumulatorState:    make(map[string]xpath3.Sequence),
		currentTime:         time.Now().UTC(),
		resultDocuments:     make(map[string]*helium.Document),
		resultDocItems:      make(map[string]xpath3.Sequence),
		resultDocOutputDefs: make(map[string]*OutputDef),
		resultDocHrefs:      make(map[string]string),
		usedResultURIs:      make(map[string]struct{}),
		defaultValidation:   ss.defaultValidation,
		defaultCollation:    ss.defaultCollation,
		docOrderCache:       xpath3.NewDocOrderCache(),
	}
	ec.setCurrentTemplate(nil) // start with no current template; baseDir() falls back to the stylesheet base
	// Set the principal output method so that isItemOutputMethod() returns
	// true when the default xsl:output method is json or adaptive.  Without
	// this, maps/arrays/functions produced during the principal result tree
	// construction would raise XTDE0450 even though the serialization method
	// allows non-node items.
	if defOut := ss.outputs[""]; defOut != nil && defOut.Method != "" {
		ec.currentResultDocMethod = defOut.Method
	}
	// Record the output XML version on the result document so any serializer —
	// including a plain DOM writer that does not consult the output definition —
	// serializes XML 1.1 restricted control characters as character references.
	if defOut := ss.outputs[""]; defOut != nil && defOut.Version != "" {
		resultDoc.SetVersion(defOut.Version)
	}

	if effectiveSource != nil {
		ec.currentNode = effectiveSource
		ec.contextNode = effectiveSource
		ec.position = 1
		ec.size = 1
	}

	if cfg != nil && cfg.msgHandler != nil {
		ec.msgHandler = cfg.msgHandler
	}
	if cfg != nil && cfg.traceWriter != nil {
		ec.traceWriter = cfg.traceWriter
	}
	if cfg != nil && cfg.baseOutputURI != "" {
		ec.currentOutputURI = cfg.baseOutputURI
		// The principal result tree always exists and its URI is the base output
		// URI, so reserve the canonical form of that URI up front. A secondary
		// xsl:result-document whose href resolves to it denotes the SAME final
		// result tree and must raise XTDE1490. Canonicalize with the SAME resolver
		// the secondary duplicate key uses (canonicalResultURIKey) so a relative
		// secondary href and the equivalent absolute one both match this seed.
		// The primary output itself keys on the "" sentinel, so this distinct
		// canonical key never makes the primary collide with itself.
		ec.canonicalPrimaryURI = canonicalResultURIKey(cfg.baseOutputURI, "")
		ec.usedResultURIs[ec.canonicalPrimaryURI] = struct{}{}
	} else if cfg != nil && cfg.outputBaseURI != "" {
		// No base-output-uri was supplied, so the principal output has no declared
		// URI (no canonicalPrimaryURI reservation; the principal result-map key
		// stays "output"). Secondary xsl:result-document output URIs still resolve
		// against the best available base — the call's effective static base URI —
		// so their result-map keys are absolute whenever any base exists.
		ec.currentOutputURI = cfg.outputBaseURI
	}
	// Make the transform config (and thus its URIResolver) available before
	// any runtime resource loading. initGlobalVars also assigns this; setting
	// it here ensures schema-location loads below route through the resolver.
	ec.transformConfig = cfg

	// Build the runtime schema registry from both xsl:import-schema and any
	// xsi:schemaLocation declarations on the source document so typed source
	// trees remain available even when the stylesheet itself has no imports.
	// Include schemas imported by every used package (transitively) too: a
	// component reached across an xsl:use-package boundary — e.g. the original
	// function invoked by xsl:original — carries a declared type (as="...") that
	// only the DEFINING package's schema can resolve, so its type names must be
	// available in the same registry. Deduped by schema-object identity (not by
	// target namespace) so two packages that each declare a no-namespace type
	// both remain resolvable.
	runtimeSchemas := collectPackageSchemas(ss)
	if effectiveSource != nil {
		sourceSchemas, schemaErr := ec.loadSchemasFromSchemaLocation(ctx, effectiveSource)
		// A source schema-location load applies the SAME positive-tag discipline as
		// the top-level import-schema path (compileImportSchema) and the nested xsd
		// loaders. Under lax/default validation the transform proceeds ONLY when the
		// load error is a CONFIRMED benign resolution miss ([isDemotableSchemaMiss] —
		// an unresolvable schemaLocation / HTTP 404, best-effort: the source is
		// simply not validated). EVERYTHING else is fatal even under lax — a CONTENT
		// error (fetched but malformed XML / invalid XSD), a post-open read failure,
		// an HTTP 401/403/5xx, or a FATAL load (policy/no-resolver denial,
		// resource-cap breach, path escape, import-depth overflow,
		// permission/multi-error) — since masking a broken or policy-denied
		// authoritative schema would silently skip validation the instance
		// requested. Strict validation stays fatal on any load failure.
		if schemaErr != nil && (ss.defaultValidation == validationStrict ||
			!isDemotableSchemaMiss(schemaErr)) {
			return nil, schemaErr
		}
		runtimeSchemas = mergeRuntimeSchemas(runtimeSchemas, sourceSchemas)
	}
	// Merge externally-provided source schemas (e.g. from test catalog
	// environments that associate a schema with the source document).
	if cfg != nil && len(cfg.sourceSchemas) > 0 {
		runtimeSchemas = mergeRuntimeSchemas(runtimeSchemas, cfg.sourceSchemas)
	}
	sourceValidated := false
	// sourceCopied records whether effectiveSource has been replaced by a private
	// deep copy (made for schema validation). When set, the strip block below
	// strips that copy IN PLACE rather than making a second deep copy.
	sourceCopied := false
	if ss.schemaAware || len(runtimeSchemas) > 0 {
		ec.schemaRegistry = &schemaRegistry{schemas: runtimeSchemas}
		ec.typeAnnotations = make(map[helium.Node]string)
		if len(runtimeSchemas) > 0 && effectiveSource != nil {
			// Source-schema validation inserts default/fixed attributes into the
			// tree it validates, so it must never touch the caller's source. Work
			// on a private, byte-faithful deep copy: validate + annotate + navigate
			// the copy, leaving the caller's source read-only. The copy reuses
			// copyAndStrip with no strip rules and no schema classifier (a pure
			// copy), so the original->copy node map needed to remap an initial-match
			// selection comes for free; the strip block below strips this SAME copy
			// in place (no second copy). The type annotations produced by validating
			// the copy key on the copy's nodes, which is also what drives the
			// XSLT 3.0 §4.4.2 whitespace-strip decision below.
			needMap := selectionSupplied && sequence.Len(matchSelection) > 0
			srcCopy, upfrontMap, copyErr := copyAndStrip(effectiveSource, nil, nil, needMap, nil)
			if copyErr != nil {
				return nil, copyErr
			}
			// The initial match selection (if any) was computed against the caller's
			// original tree; remap each selected node onto the copy the transform
			// navigates. The pure copy omits nothing, so every node has a mapping.
			if needMap {
				matchSelection = remapSelectionToCopy(matchSelection, effectiveSource, upfrontMap)
			}
			effectiveSource = srcCopy
			ec.sourceDoc = srcCopy
			ec.currentNode = srcCopy
			ec.contextNode = srcCopy
			sourceCopied = true

			vr, valErr := ec.schemaRegistry.ValidateDoc(ctx, effectiveSource)
			if valErr != nil && ss.defaultValidation == validationStrict {
				return nil, valErr
			}
			// Annotations and nilled flags key on the copy's nodes (validation ran
			// on the copy), so no remap is needed.
			for node, typeName := range vr.Annotations {
				ec.annotateNode(node, typeName)
			}
			for elem := range vr.NilledElements {
				ec.markNilled(elem)
			}
			sourceValidated = len(vr.Annotations) > 0
		}
	}

	// xsl:strip-space, plus the schema-aware whitespace rules (XSLT 3.0 §4.4.2),
	// remove or preserve whitespace-only text nodes in the source tree. The source
	// document is owned by the caller and must never be mutated in place. A copy is
	// built only when there is actual work — explicit xsl:strip-space rules, or an
	// element-only content type whose whitespace the schema strips — so the common
	// no-strip case keeps the caller's source (and its node identity) untouched.
	if effectiveSource != nil {
		schemaClass := newSchemaWSClassifier(ec.typeAnnotations, ec.schemaRegistry)
		doStrip := len(ss.stripSpace) > 0 ||
			(schemaClass != nil && sourceNeedsSchemaStrip(effectiveSource, schemaClass))
		switch {
		case doStrip && sourceCopied:
			// effectiveSource is already the private copy made for schema
			// validation, so strip it IN PLACE to avoid a second deep copy. The
			// type annotations already key on this tree (validation ran on it), so
			// no remap is needed. Any initial-match-selection node the strip removes
			// is dropped so it does not skew position()/last().
			var removed map[helium.Node]struct{}
			if selectionSupplied && sequence.Len(matchSelection) > 0 {
				removed = make(map[helium.Node]struct{})
			}
			ec.stripWhitespaceFromNodeInto(effectiveSource, removed)
			if len(removed) > 0 {
				matchSelection = dropSelectionItems(matchSelection, removed)
			}
		case doStrip:
			// Non-schema path: the caller's source has not been copied, so copy and
			// strip it in a single pass (omitting whitespace-only nodes from the
			// copy), and remap the selection onto the copy.
			needSelectionMap := matchSelection != nil && sequence.Len(matchSelection) > 0
			// Build the original->copy node map whenever the initial match selection
			// or the gathered type annotations must be carried over to the copy.
			needMap := needSelectionMap || len(ec.typeAnnotations) > 0
			srcCopy, nodeMap, copyErr := copyAndStrip(effectiveSource, ss.stripSpace, ss.preserveSpace, needMap, schemaClass)
			if copyErr != nil {
				return nil, copyErr
			}
			// The initial match selection (if any) was computed against the original
			// source tree; remap any selected node into its copy so template matching
			// runs over the same (stripped) tree as the context node.
			if needSelectionMap {
				matchSelection = remapSelectionToCopy(matchSelection, effectiveSource, nodeMap)
			}
			// Remap the type annotations and nilled flags gathered on the original
			// tree onto the copy the transform navigates.
			if nodeMap != nil {
				ec.remapAnnotationsToCopy(nodeMap)
			}
			effectiveSource = srcCopy
			ec.sourceDoc = srcCopy
			ec.currentNode = srcCopy
			ec.contextNode = srcCopy
		}
	}

	if sourceValidated && effectiveSource != nil {
		if ec.validatedDocs == nil {
			ec.validatedDocs = make(map[*helium.Document]struct{})
		}
		ec.validatedDocs[effectiveSource] = struct{}{}
	}

	// Apply input-type-annotations="strip": remove all type annotations from
	// the source document so elements are xs:untyped and attributes xs:untypedAtomic.
	// The is-id and is-idref properties are preserved (per XSLT 3.0 §3.12)
	// so that fn:id() and fn:idref() still work.
	if ss.inputTypeAnnotations == validationStrip && ec.typeAnnotations != nil {
		ec.preserveIDAnnotations()
		for k := range ec.typeAnnotations {
			delete(ec.typeAnnotations, k)
		}
	}

	// xsl:strip-space whitespace removal already happened during copyAndStrip
	// above, so the source document seen here is the stripped copy.

	// When a globalContextSelect expression is provided, evaluate it against
	// the (possibly stripped) source document.  If the result is empty, the
	// global context item is absent and global variables referencing "."
	// will raise XPDY0002.
	if cfg != nil && cfg.globalContextSelect != "" && effectiveSource != nil {
		expr, compErr := xpath3.NewCompiler().Compile(cfg.globalContextSelect)
		if compErr == nil {
			gcEval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
			if cfg.parser != nil {
				gcEval = gcEval.Parser(*cfg.parser)
			}
			result, evalErr := gcEval.Evaluate(ctx, expr, effectiveSource)
			if evalErr != nil || sequence.Len(result.Sequence()) == 0 {
				ec.globalContextAbsent = true
			}
		}
	}

	// Establish the global context item used when evaluating global
	// variables/parameters, per F&O 3.1 §14.8 (the global-context-item option is
	// an item()) and XSLT 3.0 §5.4.3.1 (xsl:global-context-item).
	//   - use="absent": the global context item is absent regardless of any
	//     supplied option, so a supplied global-context-item is NOT installed and
	//     a global "." reference raises XPDY0002.
	//   - an explicit global-context-item overrides the default (the source
	//     document node). When it is a node, global initialisers see it as "."
	//     (globalSourceNode) while the initial match selection still drives
	//     template matching independently; a non-node item (atomic/map/array/
	//     function) has no context node, so it is exposed through ec.contextItem
	//     for the duration of global evaluation only.
	//   - no source-node and no explicit item (cfg.globalContextAbsent): the
	//     global context item is absent (XPDY0002 on "."; XTDE3086 if required).
	restoreGlobalContextItem := false
	switch {
	case ss.globalContextItem != nil && ss.globalContextItem.Use == ctxItemAbsent:
		ec.globalContextAbsent = true
	case cfg != nil && cfg.globalContextItem != nil:
		ec.globalContextItem = cfg.globalContextItem
		if _, isNode := cfg.globalContextItem.(xpath3.NodeItem); !isNode {
			ec.contextItem = cfg.globalContextItem
			restoreGlobalContextItem = true
		}
	case cfg != nil && cfg.globalContextAbsent:
		ec.globalContextAbsent = true
	}

	// Store exec context in Go context for avt evaluation
	ctx = withExecContext(ctx, ec)

	// Initialize global variables
	if err := ec.initGlobalVars(ctx, cfg); err != nil {
		return nil, err
	}

	// XTTE0590: validate global context item against declared type.
	if err := ec.validateGlobalContextItem(ctx, effectiveSource); err != nil {
		return nil, err
	}

	// XPDY0002: when the initial mode is streamable and the principal source
	// document is provided, global variables whose select expressions depend
	// on the document context are not allowed — the streamed document is not
	// available for random access.
	if effectiveSource != nil && cfg != nil && cfg.initialMode != "" {
		if md := ss.modeDefs[cfg.initialMode]; md != nil && md.Streamable {
			if err := checkGlobalVarsStreamingContext(ss); err != nil {
				return nil, err
			}
		}
	}

	// Eagerly evaluate all global variables and parameters so that errors
	// (e.g. FOAR0001 from division by zero) are raised before any template
	// execution begins.  This ensures that global-variable evaluation errors
	// are non-recoverable and cannot be caught by xsl:try/xsl:catch.
	if err := ec.evaluateAllGlobals(ctx); err != nil {
		return nil, err
	}
	// The non-node global-context-item was exposed as ec.contextItem only for
	// global evaluation; clear it so it does not leak into template execution.
	if restoreGlobalContextItem {
		ec.contextItem = nil
	}

	// Pre-compute accumulator states for the main source document so that
	// accumulator-before() / accumulator-after() work during template execution.
	// This mirrors what xsl:source-document does via prepareSourceDocumentAccumulators.
	if len(ss.accumulators) > 0 && effectiveSource != nil {
		names := append([]string(nil), ss.accumulatorOrder...)
		if err := ec.computeAccumulatorStates(ctx, effectiveSource, names); err != nil {
			return nil, err
		}
		// Mark this document as computed so lazy computation doesn't redo it.
		if ec.accumulatorComputedDocs == nil {
			ec.accumulatorComputedDocs = make(map[helium.Node]struct{})
		}
		ec.accumulatorComputedDocs[documentRoot(effectiveSource)] = struct{}{}
	}

	// Initial function invocation (XSLT 3.0 §2.4)
	if cfg != nil && cfg.initialFunction != "" {
		cfg.resolvedOutputDef = cloneOutputDef(ss.outputs[""])
		doc, err := ec.invokeInitialFunction(ctx, cfg)
		if err != nil {
			return nil, err
		}
		// Deliver the raw XDM result sequence to the receiver.
		if cfg.rawResultHandler != nil && ec.rawResultSequence != nil {
			if err := cfg.rawResultHandler.HandleRawResult(ec.rawResultSequence); err != nil {
				return nil, err
			}
		}
		// Collect captured items for raw delivery format.
		out := ec.currentOutput()
		if out != nil && len(out.pendingItems) > 0 {
			if cfg.rawCapture {
				cfg.rawCapturedItems = out.pendingItems
			}
			cfg.primaryItems = out.pendingItems
		}
		return doc, nil
	}

	// Either call the initial-template or apply templates to the document root
	initialTemplateName := ""
	if cfg != nil && cfg.initialTemplate != "" {
		initialTemplateName = resolveQName(cfg.initialTemplate, ec.stylesheet.namespaces)
	}

	// XSLT 3.0: if no explicit initial template, check for xsl:initial-template
	if initialTemplateName == "" {
		xsltInitial := helium.ClarkName(lexicon.NamespaceXSLT, "initial-template")
		if _, ok := ec.stylesheet.namedTemplates[xsltInitial]; ok {
			initialTemplateName = xsltInitial
		} else if _, ok := ec.stylesheet.namedTemplates["xsl:initial-template"]; ok {
			initialTemplateName = "xsl:initial-template"
		}
	}

	// Determine initial mode
	initialMode := ""
	if cfg != nil && cfg.initialMode != "" {
		initialMode = cfg.initialMode
	}

	if initialTemplateName != "" {
		tmpl := ec.stylesheet.namedTemplates[initialTemplateName]
		if tmpl == nil {
			return nil, dynamicError(errCodeXTDE0820, "initial template %q not found", initialTemplateName)
		}
		// XTDE0040: initial template must be public in a package context
		if ec.stylesheet.isPackage && tmpl.Visibility != "" && tmpl.Visibility != visPublic && tmpl.Visibility != visFinal {
			return nil, dynamicError(errCodeXTDE0040, "initial template %q is not public (visibility=%q)", initialTemplateName, tmpl.Visibility)
		}
		// Also check for templates with no explicit visibility in packages (default=private)
		if ec.stylesheet.isPackage && tmpl.Visibility == "" {
			return nil, dynamicError(errCodeXTDE0040, "initial template %q is not public", initialTemplateName)
		}
		// XTDE0060: initial template must not have any required parameters
		// that are not supplied by the caller.
		for _, p := range tmpl.Params {
			if !p.Required {
				continue
			}
			supplied := false
			if cfg != nil {
				if _, ok := cfg.initialTemplateParams[p.Name]; ok {
					supplied = true
				}
			}
			if !supplied {
				return nil, dynamicError(errCodeXTDE0060, "initial template %q has required parameter $%s", initialTemplateName, p.Name)
			}
		}
		// Pass caller-supplied initial-template params.
		var initialParams map[string]xpath3.Sequence
		if cfg != nil && len(cfg.initialTemplateParams) > 0 {
			initialParams = cloneSequenceMap(cfg.initialTemplateParams)
		}
		// Set tunnel params for initial template (allowed in XSLT 3.0)
		if cfg != nil && len(cfg.initialTemplateTunnel) > 0 {
			ec.tunnelParams = cloneSequenceMap(cfg.initialTemplateTunnel)
		}
		if err := ec.executeTemplate(ctx, tmpl, effectiveSource, initialMode, initialParams); err != nil {
			return nil, err
		}
	} else {
		if initialMode != "" {
			if err := ec.checkInitialMode(initialMode); err != nil {
				return nil, err
			}
		}
		// XTDE0040: no source document supplied and no initial template identified.
		// A supplied initial match selection counts as input even when it remaps to
		// empty (all nodes stripped); only error when no selection was supplied.
		if effectiveSource == nil && !selectionSupplied {
			return nil, dynamicError(errCodeXTDE0040, "no source document and no initial template")
		}
		var initialModeParams map[string]xpath3.Sequence
		if cfg != nil && len(cfg.initialModeParams) > 0 {
			initialModeParams = cloneSequenceMap(cfg.initialModeParams)
		}
		savedTunnel := ec.tunnelParams
		if cfg != nil && len(cfg.initialModeTunnel) > 0 {
			ec.tunnelParams = cloneSequenceMap(cfg.initialModeTunnel)
		}
		if selectionSupplied {
			// An initial match selection was supplied. Apply templates to each of
			// its items. If the selection is empty (e.g. every node was removed by
			// strip-space during remapSelectionToCopy), this loop runs zero times
			// and produces no output — we must NOT fall through to the source
			// document.
			// Resolve mode name for atomic template matching
			resolvedMode := initialMode
			switch resolvedMode {
			case modeCurrent:
				resolvedMode = ec.currentMode
			case modeUnnamed:
				resolvedMode = ""
			case modeDefault, "":
				resolvedMode = ec.stylesheet.defaultMode
			}
			// Further resolve #unnamed if defaultMode points to it
			if resolvedMode == modeUnnamed {
				resolvedMode = ""
			}
			selLen := sequence.Len(matchSelection)
			// Apply templates to the initial match selection items
			for i := range selLen {
				item := matchSelection.Get(i)
				switch v := item.(type) {
				case xpath3.NodeItem:
					ec.contextNode = v.Node
					ec.currentNode = v.Node
					ec.position = i + 1
					ec.size = selLen
					if err := ec.applyTemplates(ctx, v.Node, initialMode, initialModeParams); err != nil {
						ec.tunnelParams = savedTunnel
						return nil, err
					}
				case xpath3.AtomicValue:
					ec.contextItem = v
					ec.position = i + 1
					ec.size = selLen
					tmpl, tErr := ec.findAtomicTemplate(ctx, v, resolvedMode)
					if tErr != nil {
						ec.tunnelParams = savedTunnel
						return nil, tErr
					}
					if tmpl != nil {
						if err := ec.executeAtomicTemplate(ctx, tmpl, v, resolvedMode, initialModeParams); err != nil {
							ec.tunnelParams = savedTunnel
							return nil, err
						}
					} else {
						// Built-in: output string value of atomic item
						av, aErr := xpath3.AtomizeItem(v)
						if aErr == nil {
							s, sErr := xpath3.AtomicToString(av)
							if sErr == nil {
								text := ec.resultDoc.CreateText([]byte(s))
								if err := ec.addNode(text); err != nil {
									ec.tunnelParams = savedTunnel
									return nil, err
								}
							}
						}
					}
				}
			}
		} else {
			if err := ec.applyTemplates(ctx, effectiveSource, initialMode, initialModeParams); err != nil {
				ec.tunnelParams = savedTunnel
				return nil, err
			}
		}
		ec.tunnelParams = savedTunnel
	}

	// For secondary result documents with json/adaptive method, serialize
	// captured items into the document so the handler receives the complete output.
	// Keys here are the RESOLVED absolute output URI (the same key used across
	// resultDocItems/resultDocuments/resultDocOutputDefs).
	for uri, items := range ec.resultDocItems {
		if items == nil || sequence.Len(items) == 0 {
			continue
		}
		doc := ec.resultDocuments[uri]
		if doc == nil {
			doc = helium.NewDefaultDocument()
			doc.SetURL(uri)
			ec.resultDocuments[uri] = doc
		}
		// Serialize items into a text node in the document.
		var buf strings.Builder
		outDef := ec.resultDocOutputDefs[uri]
		if outDef == nil {
			outDef = ss.outputs[""] // fallback to default
		}
		if err := SerializeItems(&buf, items, doc, outDef); err == nil {
			text := doc.CreateText([]byte(buf.String()))
			if text != nil {
				_ = doc.AddChild(text)
			}
		}
	}

	// Deliver secondary result documents to the receiver. The map is keyed by the
	// resolved absolute output URI (which is also doc.URL()), while the public
	// handler receives the raw href as written on xsl:result-document.
	if cfg != nil && cfg.resultDocHandler != nil {
		for uri, doc := range ec.resultDocuments {
			outDef := ec.resultDocOutputDefs[uri]
			if err := cfg.resultDocHandler.HandleResultDocument(ec.resultDocHrefs[uri], doc, outDef); err != nil {
				return nil, err
			}
		}
	}

	// Deliver the raw XDM result sequence to the receiver.
	if cfg != nil && cfg.rawResultHandler != nil && ec.rawResultSequence != nil {
		if err := cfg.rawResultHandler.HandleRawResult(ec.rawResultSequence); err != nil {
			return nil, err
		}
	}

	// Capture non-node items for raw delivery format or json/adaptive output.
	{
		out := ec.currentOutput()
		if cfg != nil && out != nil && len(out.pendingItems) > 0 {
			// SERE0022: validate JSON duplicate keys when allow-duplicate-names is not "yes"
			// for the primary output method.
			primaryMethod := ec.currentResultDocMethod
			if primaryMethod == methodJSON {
				// Derive allow-duplicate-names from the effective primary output
				// definition: the default xsl:output, overridden by any primary
				// xsl:result-document serialization params (which already fold in
				// the default base via evalResultDocOutputDef).
				allowDupes := false
				if defOut := ss.outputs[""]; defOut != nil {
					allowDupes = defOut.AllowDuplicateNames
				}
				if ec.primaryOutputOverrides != nil {
					allowDupes = ec.primaryOutputOverrides.AllowDuplicateNames
				}
				if !allowDupes {
					if err := validateJSONItems(out.pendingItems); err != nil {
						return nil, err
					}
				}
			}
			if cfg.rawCapture {
				cfg.rawCapturedItems = out.pendingItems
			}
			cfg.primaryItems = out.pendingItems
			if cfg.primaryItemsHandler != nil {
				if err := cfg.primaryItemsHandler.HandlePrimaryItems(out.pendingItems); err != nil {
					return nil, err
				}
			}
		}
	}

	// Build the per-invocation output definition for the primary result.
	// Clone before mutating so the compiled stylesheet is never modified.
	outDef := ss.outputs[""]
	if outDef != nil {
		outDef = cloneOutputDef(outDef)
		allMapNames := outDef.UseCharacterMaps
		allMapNames = append(allMapNames, ec.primaryCharacterMaps...)
		if len(allMapNames) > 0 {
			// If we already have package-resolved character maps, use those
			// instead of re-resolving from the main stylesheet (package
			// isolation: character maps are scoped per-package).
			if len(ec.primaryResolvedCharMap) > 0 {
				outDef.ResolvedCharMap = ec.primaryResolvedCharMap
			} else {
				outDef.ResolvedCharMap = resolveCharacterMaps(ss, allMapNames)
			}
		}
	} else if len(ec.primaryCharacterMaps) > 0 {
		outDef = &OutputDef{Method: methodXML, Encoding: lexicon.EncodingUTF8U}
		if len(ec.primaryResolvedCharMap) > 0 {
			outDef.ResolvedCharMap = ec.primaryResolvedCharMap
		} else {
			outDef.ResolvedCharMap = resolveCharacterMaps(ss, ec.primaryCharacterMaps)
		}
	} else if len(ec.primaryResolvedCharMap) > 0 {
		outDef = &OutputDef{Method: methodXML, Encoding: lexicon.EncodingUTF8U}
		outDef.ResolvedCharMap = ec.primaryResolvedCharMap
	}

	// Apply serialization parameter overrides from primary xsl:result-document.
	if ec.primaryOutputOverrides != nil {
		if outDef == nil {
			outDef = &OutputDef{Method: methodXML, Encoding: lexicon.EncodingUTF8U}
		}
		// Clone the overrides so the pointer/slice/map fields merged below are
		// independent allocations; the terminal outDef (exposed via
		// ResolvedOutputDef and handlers) must not alias shared override state.
		ov := cloneOutputDef(ec.primaryOutputOverrides)
		if ov.Method != "" {
			outDef.Method = ov.Method
			// Carry the override's explicitness rather than forcing it true:
			// an override built solely from AVT-only attributes (media-type,
			// html-version, etc.) inherits the base method without making it
			// explicit, so forcing MethodExplicit=true here would wrongly
			// disable html/xhtml auto-detection in serializeResult.
			outDef.MethodExplicit = ov.MethodExplicit
		}
		if ov.Standalone != "" {
			outDef.Standalone = ov.Standalone
		}
		if ov.Encoding != "" {
			outDef.Encoding = ov.Encoding
		}
		// ov folds in the default xsl:output base via evalResultDocOutputDef, so
		// it is the effective output definition. Assign the boolean serialization
		// fields directly rather than OR-ing with outDef: an explicit false on
		// the result-document (e.g. indent="{false()}") must override an inherited
		// true. OR-ing would wrongly keep the inherited true on.
		outDef.Indent = ov.Indent
		outDef.OmitDeclaration = ov.OmitDeclaration
		// Carry the omit-xml-declaration explicitness so the xhtml/html5
		// serializer respects an explicit value rather than defaulting it to
		// "yes" (output_html.go keys on OmitDeclarationExplicit).
		outDef.OmitDeclarationExplicit = ov.OmitDeclarationExplicit
		if ov.DoctypeSystem != "" {
			outDef.DoctypeSystem = ov.DoctypeSystem
		}
		if ov.DoctypePublic != "" {
			outDef.DoctypePublic = ov.DoctypePublic
		}
		if ov.MediaType != "" {
			outDef.MediaType = ov.MediaType
		}
		if ov.HTMLVersion != "" {
			outDef.HTMLVersion = ov.HTMLVersion
		}
		// Effective (base-folded) values; assign directly so an explicit false
		// overrides an inherited true.
		outDef.IncludeContentType = ov.IncludeContentType
		outDef.ByteOrderMark = ov.ByteOrderMark
		if len(ov.CDATASections) > 0 {
			outDef.CDATASections = ov.CDATASections
		}
		if ov.NormalizationForm != "" {
			outDef.NormalizationForm = ov.NormalizationForm
		}
		if ov.Version != "" {
			outDef.Version = ov.Version
		}
		outDef.EscapeURIAttributes = ov.EscapeURIAttributes
		if len(ov.SuppressIndentation) > 0 {
			outDef.SuppressIndentation = ov.SuppressIndentation
		}
		if len(ov.ResolvedCharMap) > 0 {
			outDef.ResolvedCharMap = ov.ResolvedCharMap
		}
		if ov.JSONNodeOutputMethod != "" {
			outDef.JSONNodeOutputMethod = ov.JSONNodeOutputMethod
		}
		if ov.ItemSeparator != nil {
			outDef.ItemSeparator = ov.ItemSeparator
		} else if ov.ItemSeparatorAbsent {
			outDef.ItemSeparator = nil
			outDef.ItemSeparatorAbsent = true
		}
		if ov.BuildTree != nil {
			outDef.BuildTree = ov.BuildTree
		}
		// ov folds in the default xsl:output base via evalResultDocOutputDef, so
		// this is the effective allow-duplicate-names for the primary output.
		outDef.AllowDuplicateNames = ov.AllowDuplicateNames
		// Likewise, ov folds in the default xsl:output base, so this is the
		// effective undeclare-prefixes for the primary output.
		outDef.UndeclarePrefixes = ov.UndeclarePrefixes
	}

	if cfg != nil {
		cfg.resolvedOutputDef = outDef
	}

	// Deliver type annotations and schema declarations to the caller
	// so that post-transform XPath assertions can use them.
	if cfg != nil && cfg.annotationHandler != nil && ec.typeAnnotations != nil {
		var sd xpath3.SchemaDeclarations
		if ec.schemaRegistry != nil {
			sd = ec.schemaRegistry
		}
		if err := cfg.annotationHandler.HandleAnnotations(ec.typeAnnotations, sd); err != nil {
			return nil, err
		}
	}

	return resultDoc, nil
}

// checkInitialMode verifies the initial mode is available and returns an error
// if it is not. Returns XTDE0044 if the mode exists but is not public in a
// package, or XTDE0045 if the mode is not found.
func (ec *execContext) checkInitialMode(mode string) error {
	// Resolve #default to the actual default mode name
	resolvedMode := mode
	switch mode {
	case "", modeDefault:
		resolvedMode = ec.stylesheet.defaultMode
	case modeUnnamed:
		resolvedMode = ""
	}

	// The stylesheet's default-mode is always eligible as an initial mode,
	// even if private (per XSLT 3.0 spec).
	if resolvedMode == ec.stylesheet.defaultMode {
		return nil
	}

	// The unnamed mode (#unnamed) is always eligible as an initial mode
	// when explicitly requested, even if private in a package (XSLT 3.0 §2.4).
	if mode == modeUnnamed || resolvedMode == "" {
		return nil
	}

	if ec.stylesheet.isPackage {
		// In packages, check if the resolved mode is public
		if md := ec.stylesheet.modeDefs[resolvedMode]; md != nil {
			if md.Visibility == visPrivate || md.Visibility == "" {
				return dynamicError(errCodeXTDE0044, "initial mode %q is not public in the package", mode)
			}
			return nil
		}
		// For packages with declared-modes="true" (default), undeclared modes
		// are implicitly private.
		if ec.stylesheet.declaredModes {
			return dynamicError(errCodeXTDE0044, "initial mode %q is not declared in the package", mode)
		}
	} else {
		// In non-package stylesheets, special modes are always available
		switch mode {
		case "", modeDefault, modeUnnamed:
			return nil
		}
	}

	// Check explicit mode visibility
	if md := ec.stylesheet.modeDefs[mode]; md != nil {
		if md.Visibility == visPrivate {
			return dynamicError(errCodeXTDE0044, "initial mode %q is not public", mode)
		}
		if ec.stylesheet.isPackage && md.Visibility == "" {
			return dynamicError(errCodeXTDE0044, "initial mode %q is not public in the package", mode)
		}
		return nil
	}
	// The initial mode must have at least one template rule (XTDE0045).
	if len(ec.stylesheet.modeTemplates[mode]) > 0 {
		return nil
	}
	return dynamicError(errCodeXTDE0045, "initial mode %q not found", mode)
}

func cloneSequenceMap(in map[string]xpath3.Sequence) map[string]xpath3.Sequence {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]xpath3.Sequence, len(in))
	for k, v := range in {
		out[k] = append(xpath3.ItemSlice(nil), sequence.Materialize(v)...)
	}
	return out
}

// initGlobalVars registers global variables and parameters for lazy evaluation.
