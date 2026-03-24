package xslt3

import (
	"context"
	"strings"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
)

// cloneOutputDef returns a shallow copy of an OutputDef with deep-copied map
// fields. Returns nil if src is nil.
func cloneOutputDef(src *OutputDef) *OutputDef {
	if src == nil {
		return nil
	}
	cp := *src
	if src.ResolvedCharMap != nil {
		cp.ResolvedCharMap = make(map[rune]string, len(src.ResolvedCharMap))
		for k, v := range src.ResolvedCharMap {
			cp.ResolvedCharMap[k] = v
		}
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
		transformCtx:        ctx,
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

	if cfg != nil && cfg.msgReceiver != nil {
		ec.msgReceiver = cfg.msgReceiver
	}
	if cfg != nil && cfg.traceWriter != nil {
		ec.traceWriter = cfg.traceWriter
	}
	if cfg != nil && cfg.baseOutputURI != "" {
		ec.currentOutputURI = cfg.baseOutputURI
	}

	// Build the runtime schema registry from both xsl:import-schema and any
	// xsi:schemaLocation declarations on the source document so typed source
	// trees remain available even when the stylesheet itself has no imports.
	runtimeSchemas := append([]*xsd.Schema(nil), ss.schemas...)
	if effectiveSource != nil {
		sourceSchemas, schemaErr := loadSchemasFromSchemaLocation(ctx, effectiveSource)
		if schemaErr != nil && ss.defaultValidation == validationStrict {
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

	// Apply xsl:strip-space to the source document so that whitespace-only
	// text nodes are removed before template matching and XPath evaluation.
	if len(ss.stripSpace) > 0 && effectiveSource != nil {
		ec.stripWhitespaceFromDoc(effectiveSource)
	}

	// Store exec context in Go context for avt evaluation
	ctx = withExecContext(ctx, ec)

	// Initialize global variables
	if err := ec.initGlobalVars(ctx, cfg); err != nil {
		return nil, err
	}

	// XTTE0590: validate global context item against declared type.
	if err := ec.validateGlobalContextItem(effectiveSource); err != nil {
		return nil, err
	}

	// Eagerly evaluate all global variables and parameters so that errors
	// (e.g. FOAR0001 from division by zero) are raised before any template
	// execution begins.  This ensures that global-variable evaluation errors
	// are non-recoverable and cannot be caught by xsl:try/xsl:catch.
	if err := ec.evaluateAllGlobals(); err != nil {
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
		return ec.invokeInitialFunction(ctx, cfg)
	}

	// Either call the initial-template or apply templates to the document root
	initialTemplateName := ""
	if cfg != nil && cfg.initialTemplate != "" {
		initialTemplateName = resolveQName(cfg.initialTemplate, ec.stylesheet.namespaces)
	}

	// XSLT 3.0: if no explicit initial template, check for xsl:initial-template
	if initialTemplateName == "" {
		xsltInitial := "{" + lexicon.NamespaceXSLT + "}initial-template"
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
		if ec.stylesheet.isPackage && tmpl.Visibility != "" && tmpl.Visibility != visPublic {
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
		if effectiveSource == nil && (cfg == nil || cfg.initialMatchSelection == nil || sequence.Len(cfg.initialMatchSelection) == 0) {
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
		if cfg != nil && cfg.initialMatchSelection != nil && sequence.Len(cfg.initialMatchSelection) > 0 {
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
			selLen := sequence.Len(cfg.initialMatchSelection)
			// Apply templates to the initial match selection items
			for i := range selLen {
				item := cfg.initialMatchSelection.Get(i)
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
					tmpl, tErr := ec.findAtomicTemplate(v, resolvedMode)
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
								text, tErr := ec.resultDoc.CreateText([]byte(s))
								if tErr != nil {
									ec.tunnelParams = savedTunnel
									return nil, tErr
								}
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
			text, _ := doc.CreateText([]byte(buf.String()))
			if text != nil {
				_ = doc.AddChild(text)
			}
		}
	}

	// Deliver secondary result documents to the receiver.
	if cfg != nil && cfg.resultDocReceiver != nil {
		for href, doc := range ec.resultDocuments {
			outDef := ec.resultDocOutputDefs[href]
			if err := cfg.resultDocReceiver.HandleResultDocument(href, doc, outDef); err != nil {
				return nil, err
			}
		}
	}

	// Deliver the raw XDM result sequence to the receiver.
	if cfg != nil && cfg.rawResultReceiver != nil && ec.rawResultSequence != nil {
		if err := cfg.rawResultReceiver.HandleRawResult(ec.rawResultSequence); err != nil {
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
			if cfg.primaryItemsReceiver != nil {
				if err := cfg.primaryItemsReceiver.HandlePrimaryItems(out.pendingItems); err != nil {
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
			outDef.ResolvedCharMap = resolveCharacterMaps(ss, allMapNames)
		}
	} else if len(ec.primaryCharacterMaps) > 0 {
		outDef = &OutputDef{Method: methodXML, Encoding: "UTF-8"}
		outDef.ResolvedCharMap = resolveCharacterMaps(ss, ec.primaryCharacterMaps)
	}
	// Merge resolved character maps from parameter-document (xsl:output or format).
	if len(ec.primaryResolvedCharMap) > 0 {
		if outDef == nil {
			outDef = &OutputDef{Method: methodXML, Encoding: "UTF-8"}
		}
		if outDef.ResolvedCharMap == nil {
			outDef.ResolvedCharMap = make(map[rune]string)
		}
		for k, v := range ec.primaryResolvedCharMap {
			if _, exists := outDef.ResolvedCharMap[k]; !exists {
				outDef.ResolvedCharMap[k] = v
			}
		}
	}

	// Apply serialization parameter overrides from primary xsl:result-document.
	if ec.primaryOutputOverrides != nil {
		if outDef == nil {
			outDef = &OutputDef{Method: methodXML, Encoding: "UTF-8"}
		}
		ov := ec.primaryOutputOverrides
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
	if cfg != nil && cfg.annotationReceiver != nil && ec.typeAnnotations != nil {
		var sd xpath3.SchemaDeclarations
		if ec.schemaRegistry != nil {
			sd = ec.schemaRegistry
		}
		if err := cfg.annotationReceiver.HandleAnnotations(ec.typeAnnotations, sd); err != nil {
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
