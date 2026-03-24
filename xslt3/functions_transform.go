package xslt3

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
)

func (ec *execContext) xsltFunctionsNS() map[xpath3.QualifiedName]xpath3.Function {
	if ec.cachedFnsNS != nil {
		return ec.cachedFnsNS
	}
	ec.cachedFnsNS = make(map[xpath3.QualifiedName]xpath3.Function, len(ec.stylesheet.functions)+1)

	// Register XSLT document() in the fn: namespace so fn:document() works.
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "document"}] = &xsltFunc{min: 1, max: 2, fn: ec.fnDocument}

	// Override fn:doc to preserve source document identity and apply
	// xsl:strip-space rules to loaded documents.
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "doc"}] = &xsltFunc{min: 1, max: 1, fn: ec.fnDoc}

	// Register XSLT built-in functions in the fn: namespace so they are
	// discoverable via function-lookup with an explicit namespace.
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "system-property"}] =
		&xsltFunc{min: 1, max: 1, fn: ec.fnSystemProperty}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "available-system-properties"}] =
		&xsltFunc{min: 0, max: 0, fn: ec.fnAvailableSystemProperties}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "current-output-uri"}] =
		&xsltFunc{min: 0, max: 0, fn: ec.fnCurrentOutputURI}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "stream-available"}] =
		&xsltFunc{min: 1, max: 1, fn: ec.fnStreamAvailable}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "unparsed-entity-uri"}] =
		&xsltFunc{min: 1, max: 2, fn: ec.fnUnparsedEntityURI}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "unparsed-entity-public-id"}] =
		&xsltFunc{min: 1, max: 2, fn: ec.fnUnparsedEntityPublicID}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "key"}] =
		&xsltFunc{min: 2, max: 3, fn: ec.fnKey}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "generate-id"}] =
		&xsltFunc{min: 0, max: 1, fn: ec.fnGenerateID}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "element-available"}] =
		&xsltFunc{min: 1, max: 1, fn: ec.fnElementAvailable}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "function-available"}] =
		&xsltFunc{min: 1, max: 2, fn: ec.fnFunctionAvailable}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "type-available"}] =
		&xsltFunc{min: 1, max: 1, fn: ec.fnTypeAvailable}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "current"}] =
		&xsltFunc{min: 0, max: 0, fn: ec.fnCurrent}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "current-group"}] =
		&xsltFunc{min: 0, max: 0, fn: ec.fnCurrentGroup, noDynRef: true, dynRefError: errCodeXTDE1061}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "current-grouping-key"}] =
		&xsltFunc{min: 0, max: 0, fn: ec.fnCurrentGroupingKey, noDynRef: true, dynRefError: errCodeXTDE1071}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "current-merge-group"}] =
		&xsltFunc{min: 0, max: 1, fn: ec.fnCurrentMergeGroup}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "current-merge-key"}] =
		&xsltFunc{min: 0, max: 0, fn: ec.fnCurrentMergeKey}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "regex-group"}] =
		&xsltFunc{min: 1, max: 1, fn: ec.fnRegexGroup}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "accumulator-before"}] =
		&xsltFunc{min: 1, max: 1, fn: func(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
			return ec.accumulatorLookup(ctx, args, "accumulator-before", func() (map[helium.Node]map[string]xpath3.Sequence, map[helium.Node]map[string]error) {
				return ec.accumulatorBeforeByNode, ec.accumulatorBeforeErrorByNode
			})
		}}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "accumulator-after"}] =
		&xsltFunc{min: 1, max: 1, fn: func(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
			return ec.accumulatorLookup(ctx, args, "accumulator-after", func() (map[helium.Node]map[string]xpath3.Sequence, map[helium.Node]map[string]error) {
				return ec.accumulatorAfterByNode, ec.accumulatorAfterErrorByNode
			})
		}}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "copy-of"}] =
		&xsltFunc{min: 0, max: 1, fn: ec.fnCopyOf}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "snapshot"}] =
		&xsltFunc{min: 0, max: 1, fn: ec.fnSnapshot}
	ec.cachedFnsNS[xpath3.QualifiedName{URI: xpath3.NSFn, Name: "transform"}] =
		&xsltFunc{min: 1, max: 1, fn: ec.fnTransform}

	ec.registerSchemaConstructors(ec.cachedFnsNS)

	if ec.currentPackage != nil {
		// Per-package function scope: all functions from the current
		// package (including private), plus public functions from
		// packages it uses.
		for _, def := range ec.currentPackage.functions {
			if def.Visibility == visAbstract {
				continue // abstract functions have no implementation
			}
			ec.registerUserFunc(def)
		}
		for _, usedPkg := range ec.currentPackage.usedPackages {
			for _, def := range usedPkg.functions {
				if def.Visibility == visPublic || def.Visibility == visFinal || def.Visibility == "" {
					ec.registerUserFunc(def)
				}
			}
		}
	} else {
		for _, def := range ec.stylesheet.functions {
			ec.registerUserFunc(def)
		}
	}

	return ec.cachedFnsNS
}

// xsltEvaluateFunctionsNS returns the namespaced function map for use with
// xsl:evaluate. Per XSLT 3.0 §20.3, user-defined stylesheet functions
// (xsl:function) are NOT available in xsl:evaluate unless they are
// explicitly declared as public or final.
func (ec *execContext) xsltEvaluateFunctionsNS() map[xpath3.QualifiedName]xpath3.Function {
	all := ec.xsltFunctionsNS()
	// Collect QNames of user functions that are NOT explicitly public/final
	excluded := make(map[xpath3.QualifiedName]struct{})
	fns := ec.stylesheet.functions
	if ec.currentPackage != nil {
		fns = ec.currentPackage.functions
	}
	for _, def := range fns {
		vis := def.Visibility
		if vis == visPublic || vis == visFinal {
			continue // explicitly public → available in xsl:evaluate
		}
		excluded[def.Name] = struct{}{}
	}
	result := make(map[xpath3.QualifiedName]xpath3.Function, len(all))
	for k, v := range all {
		if _, skip := excluded[k]; skip {
			continue
		}
		result[k] = v
	}
	return result
}

// registerUserFunc registers an XSL user function into cachedFnsNS,
// handling multi-arity overloads by wrapping them in xslMultiArityFunc.
func (ec *execContext) registerUserFunc(def *xslFunction) {
	qn := def.Name
	uf := &xslUserFunc{def: def, ec: ec}
	if existing, ok := ec.cachedFnsNS[qn]; ok {
		if maf, ok := existing.(*xslMultiArityFunc); ok {
			maf.addVariant(uf)
		} else {
			maf := &xslMultiArityFunc{minArity: existing.MinArity(), maxArity: existing.MaxArity()}
			if euf, ok := existing.(*xslUserFunc); ok {
				maf.variants = append(maf.variants, euf)
			}
			maf.addVariant(uf)
			ec.cachedFnsNS[qn] = maf
		}
	} else {
		ec.cachedFnsNS[qn] = uf
	}
}

// findXSLFunction finds an xsl:function by QName and arity (-1 = any).
func (ec *execContext) findXSLFunction(qn xpath3.QualifiedName, arity int) *xslFunction {
	if arity < 0 {
		// Any arity: just check if any overload exists
		for fk, def := range ec.stylesheet.functions {
			if fk.Name == qn {
				return def
			}
		}
		return nil
	}
	fk := funcKey{Name: qn, Arity: arity}
	return ec.stylesheet.functions[fk]
}

// findXSLFunctionByArity finds an xsl:function by QName and exact arity.
func (ec *execContext) findXSLFunctionByArity(qn xpath3.QualifiedName, arity int) *xslFunction {
	fk := funcKey{Name: qn, Arity: arity}
	return ec.stylesheet.functions[fk]
}

// xsltEvaluateFunctions returns XSLT built-in functions available in
// xsl:evaluate dynamic context. Per XSLT 3.0 section 20.3, current()
// is excluded.
func (ec *execContext) xsltEvaluateFunctions() map[string]xpath3.Function {
	fns := ec.xsltFunctions()
	evalFns := make(map[string]xpath3.Function, len(fns))
	for k, v := range fns {
		switch k {
		case "current", "system-property", "current-output-uri", "available-system-properties":
			continue
		}
		evalFns[k] = v
	}
	return evalFns
}

type transformDepthKey struct{}

const maxTransformDepth = 10

// resultDocCollector implements ResultDocumentReceiver for fn:transform,
// collecting secondary result documents into a map.
type resultDocCollector struct {
	results map[string]*helium.Document
}

func (c resultDocCollector) HandleResultDocument(href string, doc *helium.Document) error {
	c.results[href] = doc
	return nil
}

// newNestedCompiler creates a Compiler pre-configured with the same
// resolver, package resolver, and import schemas that were used to
// compile this stylesheet, so that fn:transform nested compiles
// behave consistently with top-level compilation.
func (ss *Stylesheet) newNestedCompiler() Compiler {
	c := NewCompiler()
	if ss.uriResolver != nil {
		c = c.URIResolver(ss.uriResolver)
	}
	if ss.packageResolver != nil {
		c = c.PackageResolver(ss.packageResolver)
	}
	if len(ss.compilerImportSchemas) > 0 {
		c = c.ImportSchemas(ss.compilerImportSchemas...)
	}
	return c
}

// fnTransform implements fn:transform() — dynamically compile and execute
// an XSLT stylesheet.
func (ec *execContext) fnTransform(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	// Check recursion depth
	depth := 0
	if d, ok := ctx.Value(transformDepthKey{}).(int); ok {
		depth = d
	}
	if depth >= maxTransformDepth {
		return nil, dynamicError(errCodeFOXT0004, "fn:transform: maximum recursion depth (%d) exceeded", maxTransformDepth)
	}
	ctx = context.WithValue(ctx, transformDepthKey{}, depth+1)
	if len(args) != 1 || (args[0] == nil || sequence.Len(args[0]) != 1) {
		return nil, dynamicError(errCodeFOXT0001, "fn:transform requires a single map argument")
	}
	m, ok := args[0].Get(0).(xpath3.MapItem)
	if !ok {
		return nil, dynamicError(errCodeFOXT0001, "fn:transform argument must be a map")
	}

	// Extract option values from the map
	getStr := func(key string) string {
		k := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: key}
		seq, ok := m.Get(k)
		if !ok || seq == nil || sequence.Len(seq) == 0 {
			return ""
		}
		av, err := xpath3.AtomizeItem(seq.Get(0))
		if err != nil {
			return ""
		}
		s, err := xpath3.AtomicToString(av)
		if err != nil {
			return ""
		}
		return s
	}

	getSeq := func(key string) xpath3.Sequence {
		k := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: key}
		seq, ok := m.Get(k)
		if !ok {
			return nil
		}
		return seq
	}

	stylesheetLoc := getStr("stylesheet-location")
	packageName := getStr("package-name")
	packageVersion := getStr("package-version")
	initialTemplate := getStr("initial-template")
	initialMode := getStr("initial-mode")
	deliveryFormat := getStr("delivery-format")
	initialMatchSel := getSeq("initial-match-selection")
	sourceNode := getSeq("source-node")
	stylesheetParamsSeq := getSeq("stylesheet-params")
	staticParamsSeq := getSeq("static-params")

	// Build a compiler that inherits the outer stylesheet's configuration.
	nestedCompiler := ec.stylesheet.newNestedCompiler()

	// Apply static-params from the options map to the nested compiler.
	if staticParamsSeq != nil && sequence.Len(staticParamsSeq) > 0 {
		if sm, ok := staticParamsSeq.Get(0).(xpath3.MapItem); ok {
			_ = sm.ForEach(func(key xpath3.AtomicValue, value xpath3.Sequence) error {
				name, sErr := xpath3.AtomicToString(key)
				if sErr != nil {
					return nil
				}
				nestedCompiler = nestedCompiler.SetStaticParameter(name, value)
				return nil
			})
		}
	}

	// Compile the stylesheet
	var ss *Stylesheet
	if stylesheetLoc != "" {
		// Resolve relative to the current stylesheet base URI
		loc := stylesheetLoc
		if ec.stylesheet.baseURI != "" && !filepath.IsAbs(loc) {
			loc = filepath.Join(filepath.Dir(ec.stylesheet.baseURI), loc)
		}
		absPath, absErr := filepath.Abs(loc)
		if absErr != nil {
			absPath = loc
		}
		data, readErr := os.ReadFile(loc)
		if readErr != nil {
			return nil, dynamicError(errCodeFOXT0003, "fn:transform: cannot read stylesheet %q: %v", stylesheetLoc, readErr)
		}
		doc, parseErr := parseStylesheetDocument(ctx, data, absPath)
		if parseErr != nil {
			return nil, dynamicError(errCodeFOXT0003, "fn:transform: cannot parse stylesheet %q: %v", stylesheetLoc, parseErr)
		}
		var compileErr error
		ss, compileErr = nestedCompiler.BaseURI(absPath).Compile(ctx, doc)
		if compileErr != nil {
			return nil, dynamicError(errCodeFOXT0003, "fn:transform: cannot compile stylesheet %q: %v", stylesheetLoc, compileErr)
		}
	} else if packageName != "" {
		// Resolve via package-name / package-version using the PackageResolver
		// stored on the compiled stylesheet (set at compile time).
		resolver := ec.stylesheet.packageResolver
		if resolver == nil {
			return nil, dynamicError(errCodeFOXT0002, "fn:transform: package-name specified but no PackageResolver available")
		}
		rc, location, resolveErr := resolver.ResolvePackage(packageName, packageVersion)
		if resolveErr != nil {
			return nil, dynamicError(errCodeFOXT0003, "fn:transform: cannot resolve package %q (version %q): %v", packageName, packageVersion, resolveErr)
		}
		data, readErr := io.ReadAll(rc)
		_ = rc.Close()
		if readErr != nil {
			return nil, dynamicError(errCodeFOXT0003, "fn:transform: cannot read package %q: %v", packageName, readErr)
		}
		doc, parseErr := parseStylesheetDocument(ctx, data, location)
		if parseErr != nil {
			return nil, dynamicError(errCodeFOXT0003, "fn:transform: cannot parse package %q: %v", packageName, parseErr)
		}
		compiler := nestedCompiler
		if location != "" {
			compiler = compiler.BaseURI(location)
		}
		var compileErr error
		ss, compileErr = compiler.Compile(ctx, doc)
		if compileErr != nil {
			return nil, dynamicError(errCodeFOXT0003, "fn:transform: cannot compile package %q: %v", packageName, compileErr)
		}
	} else {
		// Check for stylesheet-node
		ssNodeSeq := getSeq("stylesheet-node")
		if ssNodeSeq != nil && sequence.Len(ssNodeSeq) > 0 {
			if ni, ok := ssNodeSeq.Get(0).(xpath3.NodeItem); ok {
				// Find the document containing this node
				var doc *helium.Document
				n := ni.Node
				for n != nil {
					if d, ok := n.(*helium.Document); ok {
						doc = d
						break
					}
					n = n.Parent()
				}
				if doc == nil {
					return nil, dynamicError(errCodeFOXT0003, "fn:transform: stylesheet-node is not part of a document")
				}
				var compileErr error
				ss, compileErr = nestedCompiler.Compile(ctx, doc)
				if compileErr != nil {
					return nil, dynamicError(errCodeFOXT0003, "fn:transform: cannot compile stylesheet: %v", compileErr)
				}
			}
		}
	}

	if ss == nil {
		return nil, dynamicError(errCodeFOXT0002, "fn:transform: no stylesheet specified (stylesheet-location, stylesheet-node, or package-name required)")
	}

	// Determine the source document
	var sourceDoc *helium.Document
	if sourceNode != nil && sequence.Len(sourceNode) > 0 {
		if ni, ok := sourceNode.Get(0).(xpath3.NodeItem); ok {
			n := ni.Node
			for n != nil {
				if d, ok := n.(*helium.Document); ok {
					sourceDoc = d
					break
				}
				n = n.Parent()
			}
		}
	}

	// Build a fresh transform config for the inner fn:transform call.
	secondaryResults := make(map[string]*helium.Document)
	fnTransformCfg := &transformConfig{
		initialTemplate:   initialTemplate,
		initialMode:       initialMode,
		resultDocReceiver: resultDocCollector{results: secondaryResults},
	}

	// Apply stylesheet-params from the options map as runtime parameters.
	if stylesheetParamsSeq != nil && sequence.Len(stylesheetParamsSeq) > 0 {
		if sm, ok := stylesheetParamsSeq.Get(0).(xpath3.MapItem); ok {
			params := make(map[string]xpath3.Sequence, sm.Size())
			_ = sm.ForEach(func(key xpath3.AtomicValue, value xpath3.Sequence) error {
				name, sErr := xpath3.AtomicToString(key)
				if sErr != nil {
					return nil
				}
				params[name] = value
				return nil
			})
			fnTransformCfg.sequenceParams = params
		}
	}

	// Execute the transform
	var resultDoc *helium.Document
	var capturedItems xpath3.Sequence
	if initialMatchSel != nil && sequence.Len(initialMatchSel) > 0 {
		// initial-match-selection: create a document wrapper for non-node items
		// or apply templates to the selection
		if sequence.Len(initialMatchSel) == 1 {
			if ni, ok := initialMatchSel.Get(0).(xpath3.NodeItem); ok {
				n := ni.Node
				for n != nil {
					if d, ok := n.(*helium.Document); ok {
						sourceDoc = d
						break
					}
					n = n.Parent()
				}
			}
		}
		// For atomic values, we need to create a temporary document and
		// use a different execution path. For now, wrap in a document.
		if sourceDoc == nil {
			sourceDoc = helium.NewDefaultDocument()
		}

		// Set up the initial match selection on the exec context.
		// Enable raw capture when delivery-format is "raw" so function
		// items and other non-node XDM values are preserved.
		rawCapture := deliveryFormat == "raw"
		var execErr error
		resultDoc, capturedItems, execErr = executeTransformWithSelection(ctx, sourceDoc, ss, fnTransformCfg, initialMatchSel, rawCapture)
		if execErr != nil {
			return nil, execErr
		}
	} else {
		if sourceDoc == nil {
			sourceDoc = helium.NewDefaultDocument()
		}
		if deliveryFormat == "raw" {
			fnTransformCfg.rawCapture = true
			var execErr error
			resultDoc, execErr = executeTransform(ctx, sourceDoc, ss, fnTransformCfg)
			if execErr != nil {
				return nil, execErr
			}
			capturedItems = fnTransformCfg.rawCapturedItems
		} else {
			var execErr error
			resultDoc, execErr = executeTransform(ctx, sourceDoc, ss, fnTransformCfg)
			if execErr != nil {
				return nil, execErr
			}
		}
	}

	// Build result map
	outputKey := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "output"}
	result := xpath3.MapItem{}

	switch deliveryFormat {
	case "raw":
		// Raw delivery: return the XDM items from the transformation.
		// When captured items are available (from raw capture mode), use
		// those directly — they may contain function items, maps, etc.
		// that cannot be represented as DOM children. Otherwise fall
		// back to extracting DOM children for backward compatibility.
		if capturedItems != nil && sequence.Len(capturedItems) > 0 {
			// Merge DOM children and captured non-node items.
			var seq xpath3.ItemSlice
			for child := range helium.Children(resultDoc) {
				seq = append(seq, xpath3.NodeItem{Node: child})
			}
			seq = append(seq, sequence.Materialize(capturedItems)...)
			result = result.Put(outputKey, seq)
		} else if resultDoc != nil {
			var seq xpath3.ItemSlice
			for child := range helium.Children(resultDoc) {
				seq = append(seq, xpath3.NodeItem{Node: child})
			}
			result = result.Put(outputKey, seq)
		}
		// Secondary results are returned as document nodes in raw mode.
		for href, doc := range secondaryResults {
			hrefKey := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: href}
			result = result.Put(hrefKey, xpath3.ItemSlice{xpath3.NodeItem{Node: doc}})
		}
	case "serialized":
		// Serialized delivery: serialize the result document to a string.
		if resultDoc != nil {
			outDef := fnTransformCfg.resolvedOutputDef
			var buf bytes.Buffer
			if err := SerializeResult(&buf, resultDoc, outDef); err != nil {
				return nil, dynamicError(errCodeFOXT0003, "fn:transform: serialization error: %v", err)
			}
			result = result.Put(outputKey, xpath3.SingleString(buf.String()))
		}
		// Serialize secondary results too.
		for href, doc := range secondaryResults {
			hrefKey := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: href}
			outDef := ss.outputs[href]
			if outDef == nil {
				outDef = ss.outputs[""]
			}
			var buf bytes.Buffer
			if err := SerializeResult(&buf, doc, outDef); err != nil {
				result = result.Put(hrefKey, xpath3.SingleString(""))
			} else {
				result = result.Put(hrefKey, xpath3.SingleString(buf.String()))
			}
		}
	default:
		// Default: return the result document
		if resultDoc != nil {
			result = result.Put(outputKey, xpath3.ItemSlice{xpath3.NodeItem{Node: resultDoc}})
		}
		// Add secondary results as document nodes.
		for href, doc := range secondaryResults {
			hrefKey := xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: href}
			result = result.Put(hrefKey, xpath3.ItemSlice{xpath3.NodeItem{Node: doc}})
		}
	}

	return xpath3.ItemSlice{result}, nil
}

// executeTransformWithSelection runs a transform where the initial match
// selection is an explicit sequence (not derived from a source document root).
// When rawCapture is true, the output frame captures non-node items (function
// items, maps, arrays) so they can be returned in raw delivery format.
func executeTransformWithSelection(ctx context.Context, source *helium.Document, ss *Stylesheet, cfg *transformConfig, selection xpath3.Sequence, rawCapture bool) (*helium.Document, xpath3.Sequence, error) {
	resultDoc := helium.NewDefaultDocument()

	outFrame := &outputFrame{doc: resultDoc, current: resultDoc, captureItems: rawCapture}
	ec := &execContext{
		stylesheet:          ss,
		sourceDoc:           source,
		resultDoc:           resultDoc,
		currentNode:         source,
		contextNode:         source,
		position:            1,
		size:                1,
		globalVars:          make(map[string]xpath3.Sequence),
		currentMode:         "",
		outputStack:         []*outputFrame{outFrame},
		keyTables:           make(map[string]*keyTable),
		docCache:            make(map[string]*helium.Document),
		functionResultCache: make(map[string]xpath3.Sequence),
		accumulatorState:    make(map[string]xpath3.Sequence),
		transformCtx:        ctx,
		resultDocuments:     make(map[string]*helium.Document),
		usedResultURIs:      make(map[string]struct{}),
		defaultValidation:   ss.defaultValidation,
		docOrderCache:       xpath3.NewDocOrderCache(),
	}
	ec.setCurrentTemplate(nil) // initialize currentTemplateBaseDir from stylesheet

	if cfg != nil && cfg.msgReceiver != nil {
		ec.msgReceiver = cfg.msgReceiver
	}

	// Schema-aware: build schema registry and validate source document.
	if ss.schemaAware {
		ec.schemaRegistry = &schemaRegistry{schemas: ss.schemas}
		if len(ss.schemas) > 0 && source != nil {
			vr, valErr := ec.schemaRegistry.ValidateDoc(ctx, source)
			if valErr != nil && ss.defaultValidation == validationStrict {
				return nil, nil, valErr
			}
			for node, typeName := range vr.Annotations {
				ec.annotateNode(node, typeName)
			}
			for elem := range vr.NilledElements {
				ec.markNilled(elem)
			}
		}
	}

	if len(ss.stripSpace) > 0 && source != nil {
		ec.stripWhitespaceFromDoc(source)
	}

	ctx = withExecContext(ctx, ec)

	if err := ec.initGlobalVars(ctx, cfg); err != nil {
		return nil, nil, err
	}

	// Check for initial template
	initialTemplateName := ""
	if cfg != nil && cfg.initialTemplate != "" {
		initialTemplateName = cfg.initialTemplate
	}

	if initialTemplateName != "" {
		tmpl := ss.namedTemplates[initialTemplateName]
		if tmpl == nil {
			return nil, nil, dynamicError(errCodeXTDE0820, "initial template %q not found", initialTemplateName)
		}
		if err := ec.executeTemplate(ctx, tmpl, source, ""); err != nil {
			return nil, nil, err
		}
	} else {
		// Apply templates to the initial match selection
		selLen := sequence.Len(selection)
		for i := range selLen {
			item := selection.Get(i)
			switch v := item.(type) {
			case xpath3.NodeItem:
				ec.contextNode = v.Node
				ec.currentNode = v.Node
				ec.position = i + 1
				ec.size = selLen
				if err := ec.applyTemplates(ctx, v.Node, ""); err != nil {
					return nil, nil, err
				}
			case xpath3.AtomicValue:
				// For atomic values, use atomic template matching
				ec.contextItem = v
				ec.position = i + 1
				ec.size = selLen
				tmpl, tErr := ec.findAtomicTemplate(v, "")
				if tErr != nil {
					return nil, nil, tErr
				}
				if tmpl != nil {
					if err := ec.executeAtomicTemplate(ctx, tmpl, v, ""); err != nil {
						return nil, nil, err
					}
				} else {
					// Built-in: output string value of atomic item
					av, aErr := xpath3.AtomizeItem(v)
					if aErr == nil {
						s, sErr := xpath3.AtomicToString(av)
						if sErr == nil {
							text, tErr := ec.resultDoc.CreateText([]byte(s))
							if tErr != nil {
								return nil, nil, tErr
							}
							if err := ec.addNode(text); err != nil {
								return nil, nil, err
							}
						}
					}
				}
			}
		}
	}

	if cfg != nil && cfg.resultDocReceiver != nil {
		for href, doc := range ec.resultDocuments {
			if err := cfg.resultDocReceiver.HandleResultDocument(href, doc); err != nil {
				return nil, nil, err
			}
		}
	}

	if cfg != nil && cfg.resultDocOutputReceiver != nil {
		for href, outDef := range ec.resultDocOutputDefs {
			if err := cfg.resultDocOutputReceiver.HandleResultDocumentOutput(href, outDef); err != nil {
				return nil, nil, err
			}
		}
	}

	// Collect captured items from the output frame (raw delivery mode).
	var capturedItems xpath3.Sequence
	if rawCapture {
		capturedItems = outFrame.pendingItems
	}

	return resultDoc, capturedItems, nil
}
