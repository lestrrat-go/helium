package xslt3

import (
	"context"
	"errors"
	"maps"
	"strings"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
)

// remapSelectionToCopy rewrites the node items of an initial match selection so
// that any node belonging to the original source document points instead to the
// corresponding node in the stripped copy. Items that are not nodes, or that
// belong to a different document (e.g. fn:doc()-loaded), are passed through
// unchanged.
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
func remapSelectionToCopy(sel xpath3.Sequence, nodeMap map[helium.Node]helium.Node) xpath3.Sequence {
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
	effectiveSource := source
	if ss.globalContextItem != nil && ss.globalContextItem.Use == ctxItemAbsent {
		effectiveSource = nil
	}

	// xsl:strip-space removes whitespace-only text nodes from the source tree.
	// The source document is owned by the caller, so it must never be mutated in
	// place: doing so destroys node identity, corrupts a tree the caller may
	// reuse (e.g. for a later XPath query or a second transform), and is unsafe
	// under concurrent reuse. Deep-copy the source first and run the transform
	// against the copy so strip-space (applied below) only ever touches our
	// private copy. The copy becomes the exec context's source, so the initial
	// context node and all node identity stay consistent throughout the
	// transform. Only copy when strip-space rules exist (it is otherwise pure
	// overhead).
	var matchSelection xpath3.Sequence
	if cfg != nil {
		matchSelection = cfg.initialMatchSelection
	}
	if len(ss.stripSpace) > 0 && effectiveSource != nil {
		// copyAndStrip deep-copies the source, omits the whitespace-only text
		// nodes strip-space would remove, declares namespaces without
		// over-declaration, and preserves the URI/DTD — all in a single
		// traversal. This replaces the former three-pass approach
		// (CopyDoc + pruneRedundantNamespaceDecls + stripWhitespaceFromDoc) and
		// is byte-for-byte equivalent. At this point the effective strip/preserve
		// rules are the stylesheet's own (no package scope is active yet).
		// Only build the original->copy node map when there is a selection to
		// remap; the common case (no selection) skips that bookkeeping entirely.
		needMap := matchSelection != nil && sequence.Len(matchSelection) > 0
		srcCopy, nodeMap, copyErr := copyAndStrip(effectiveSource, ss.stripSpace, ss.preserveSpace, needMap)
		if copyErr != nil {
			return nil, copyErr
		}
		// The initial match selection (if any) was computed against the original
		// source tree, so its node items still point into the original. Remap any
		// selected node that lives in the original source into the corresponding
		// node of the copy so template matching runs over the same (stripped)
		// tree as the context node. Selected nodes from other documents (e.g.
		// fn:doc()-loaded) are left untouched.
		if needMap {
			matchSelection = remapSelectionToCopy(matchSelection, nodeMap)
		}
		effectiveSource = srcCopy
	}

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
		usedResultURIs:      make(map[string]struct{}),
		defaultValidation:   ss.defaultValidation,
		defaultCollation:    ss.defaultCollation,
		docOrderCache:       xpath3.NewDocOrderCache(),
	}
	ec.setCurrentTemplate(nil) // initialize currentTemplateBaseDir from stylesheet
	// Set the principal output method so that isItemOutputMethod() returns
	// true when the default xsl:output method is json or adaptive.  Without
	// this, maps/arrays/functions produced during the principal result tree
	// construction would raise XTDE0450 even though the serialization method
	// allows non-node items.
	if defOut := ss.outputs[""]; defOut != nil && defOut.Method != "" {
		ec.currentResultDocMethod = defOut.Method
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
	}
	// Make the transform config (and thus its URIResolver) available before
	// any runtime resource loading. initGlobalVars also assigns this; setting
	// it here ensures schema-location loads below route through the resolver.
	ec.transformConfig = cfg

	// Build the runtime schema registry from both xsl:import-schema and any
	// xsi:schemaLocation declarations on the source document so typed source
	// trees remain available even when the stylesheet itself has no imports.
	runtimeSchemas := append([]*xsd.Schema(nil), ss.schemas...)
	if effectiveSource != nil {
		sourceSchemas, schemaErr := ec.loadSchemasFromSchemaLocation(ctx, effectiveSource)
		// A schema-location load failure is normally non-fatal under lax
		// validation (the source is simply not validated). A resource-limit
		// breach must NOT be demoted that way, or the cap is defeated for a
		// schema referenced via xsi:schemaLocation; propagate it regardless of
		// the validation mode, preserving ErrResourceTooLarge for callers.
		if schemaErr != nil && (ss.defaultValidation == validationStrict || errors.Is(schemaErr, ErrResourceTooLarge)) {
			return nil, schemaErr
		}
		runtimeSchemas = mergeRuntimeSchemas(runtimeSchemas, sourceSchemas)
	}
	// Merge externally-provided source schemas (e.g. from test catalog
	// environments that associate a schema with the source document).
	if cfg != nil && len(cfg.sourceSchemas) > 0 {
		runtimeSchemas = mergeRuntimeSchemas(runtimeSchemas, cfg.sourceSchemas)
	}
	if ss.schemaAware || len(runtimeSchemas) > 0 {
		ec.schemaRegistry = &schemaRegistry{schemas: runtimeSchemas}
		ec.typeAnnotations = make(map[helium.Node]string)
		if len(runtimeSchemas) > 0 && effectiveSource != nil {
			vr, valErr := ec.schemaRegistry.ValidateDoc(ctx, effectiveSource)
			if valErr != nil && ss.defaultValidation == validationStrict {
				return nil, valErr
			}
			for node, typeName := range vr.Annotations {
				ec.annotateNode(node, typeName)
			}
			for elem := range vr.NilledElements {
				ec.markNilled(elem)
			}
			if len(vr.Annotations) > 0 {
				if ec.validatedDocs == nil {
					ec.validatedDocs = make(map[*helium.Document]struct{})
				}
				ec.validatedDocs[effectiveSource] = struct{}{}
			}
		}
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
			result, evalErr := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, expr, effectiveSource)
			if evalErr != nil || sequence.Len(result.Sequence()) == 0 {
				ec.globalContextAbsent = true
			}
		}
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
		// XTDE0040: no source document supplied and no initial template identified
		if effectiveSource == nil && (matchSelection == nil || sequence.Len(matchSelection) == 0) {
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
		if matchSelection != nil && sequence.Len(matchSelection) > 0 {
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
						if err := ec.executeAtomicTemplate(ctx, tmpl, v, resolvedMode); err != nil {
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
	for href, items := range ec.resultDocItems {
		if items == nil || sequence.Len(items) == 0 {
			continue
		}
		doc := ec.resultDocuments[href]
		if doc == nil {
			doc = helium.NewDefaultDocument()
			ec.resultDocuments[href] = doc
		}
		// Serialize items into a text node in the document.
		var buf strings.Builder
		outDef := ec.resultDocOutputDefs[href]
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

	// Deliver secondary result documents to the receiver.
	if cfg != nil && cfg.resultDocHandler != nil {
		for href, doc := range ec.resultDocuments {
			outDef := ec.resultDocOutputDefs[href]
			if err := cfg.resultDocHandler.HandleResultDocument(href, doc, outDef); err != nil {
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
				allowDupes := false
				if defOut := ss.outputs[""]; defOut != nil {
					allowDupes = defOut.AllowDuplicateNames
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
			outDef.MethodExplicit = true
		}
		if ov.Standalone != "" {
			outDef.Standalone = ov.Standalone
		}
		if ov.Encoding != "" {
			outDef.Encoding = ov.Encoding
		}
		outDef.Indent = ov.Indent || outDef.Indent
		outDef.OmitDeclaration = ov.OmitDeclaration || outDef.OmitDeclaration
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
		if ov.IncludeContentType != nil {
			outDef.IncludeContentType = ov.IncludeContentType
		}
		if ov.ByteOrderMark {
			outDef.ByteOrderMark = true
		}
		if len(ov.CDATASections) > 0 {
			outDef.CDATASections = ov.CDATASections
		}
		if ov.NormalizationForm != "" {
			outDef.NormalizationForm = ov.NormalizationForm
		}
		if ov.Version != "" {
			outDef.Version = ov.Version
		}
		if ov.EscapeURIAttributes != nil {
			outDef.EscapeURIAttributes = ov.EscapeURIAttributes
		}
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
