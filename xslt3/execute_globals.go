package xslt3

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/uripath"
	"github.com/lestrrat-go/helium/internal/xpathstream"
	"github.com/lestrrat-go/helium/xpath3"
)

// checkGlobalVarsStreamingContext raises XPDY0002 if any global variable has
// a select expression that depends on the implicit document context. When the
// initial mode is streamable, the principal source document is not available
// for random access in global variable initialisers.
func checkGlobalVarsStreamingContext(ss *Stylesheet) error {
	for _, v := range ss.globalVars {
		if v.Select == nil {
			continue
		}
		if xpathstream.ExprHasDownwardStep(v.Select) || xpathstream.ExprUsesContextItem(v.Select) || xpathstream.ExprUsesDescendantOrSelf(v.Select) {
			return dynamicError(errCodeXPDY0002,
				"global variable $%s references the document context, which is absent when the initial mode is streamable",
				v.Name)
		}
	}
	return nil
}

// Params with caller-provided values are set immediately; all others are
// evaluated on first access to support arbitrary declaration order.
func (ec *execContext) initGlobalVars(ctx context.Context, cfg *transformConfig) error {
	ec.transformConfig = cfg
	ec.globalVarDefs = make(map[string]*variable, len(ec.stylesheet.globalVars))
	ec.globalParamDefs = make(map[string]*param, len(ec.stylesheet.globalParams))
	ec.globalEvaluating = make(map[string]bool)

	// Register params — set immediately if caller provided a value
	for _, p := range ec.stylesheet.globalParams {
		if cfg != nil {
			if cfg.sequenceParams != nil {
				if seq, ok := cfg.sequenceParams[p.Name]; ok {
					if p.As != "" {
						st := parseSequenceType(p.As)
						checked, err := checkSequenceType(ctx, seq, st, errCodeXTTE0590, "param $"+p.Name, ec)
						if err != nil {
							return err
						}
						seq = checked
					}
					ec.globalVars[p.Name] = seq
					continue
				}
			}
		}
		ec.globalParamDefs[p.Name] = p
	}

	// XTDE0050: required stylesheet parameters that were not supplied by
	// the caller must raise an error immediately, even if never referenced.
	for _, p := range ec.globalParamDefs {
		if p.Required {
			return dynamicError(errCodeXTDE0050, "required stylesheet parameter $%s was not supplied", p.Name)
		}
	}

	// Register variables for lazy evaluation
	for _, v := range ec.stylesheet.globalVars {
		ec.globalVarDefs[v.Name] = v
	}

	return nil
}

// validateGlobalContextItem checks the source document against the
// xsl:global-context-item declaration. Returns XTDE3086 when the declaration
// requires a context item but none is supplied, and XTTE0590 if the supplied
// source doesn't match the declared type.
func (ec *execContext) validateGlobalContextItem(ctx context.Context, source *helium.Document) error {
	if err := validateUsedPackageGlobalContextItem(ec.stylesheet, map[*Stylesheet]struct{}{}); err != nil {
		return err
	}
	gci := ec.stylesheet.globalContextItem
	if gci == nil {
		return nil
	}
	// The item actually used as the global context item: when the global context
	// item is absent (use="absent", or fn:transform supplied neither a source-node
	// nor an explicit item) there is none; otherwise an explicit fn:transform
	// global-context-item (any item() — atomic/map/array/function or a node)
	// overrides the source document node, which is the last-resort default. This
	// is what must be type-checked against @as.
	var ctxItem xpath3.Item
	switch {
	case ec.globalContextAbsent:
		ctxItem = nil
	case ec.globalContextItem != nil:
		ctxItem = ec.globalContextItem
	case source != nil:
		ctxItem = xpath3.NodeItem{Node: source}
	}
	if ctxItem == nil {
		if gci.Use == ctxItemRequired {
			return dynamicError(errCodeXTDE3086, "global-context-item use=\"required\" but no global context item was supplied")
		}
		return nil
	}
	if gci.As == "" {
		return nil
	}
	// Validate the supplied global context item against the declared sequence
	// type using the namespace-aware type
	// machinery, so that prefixed element tests like
	// document-node(element(p:root)) compare both local name and namespace
	// rather than local name alone. Prefixes and the default element namespace
	// in the @as type must resolve against the namespace context of the
	// xsl:global-context-item declaration itself, not the runtime
	// stylesheet-wide context, so install the saved declaration-site context
	// for the duration of the check.
	savedNSOverride := ec.nsOverride
	savedXPathDefaultNS := ec.xpathDefaultNS
	savedHasXPathDefaultNS := ec.hasXPathDefaultNS
	ec.nsOverride = gci.Namespaces
	ec.xpathDefaultNS = gci.XPathDefaultNS
	ec.hasXPathDefaultNS = gci.HasXPathDefaultNS
	defer func() {
		ec.nsOverride = savedNSOverride
		ec.xpathDefaultNS = savedXPathDefaultNS
		ec.hasXPathDefaultNS = savedHasXPathDefaultNS
	}()

	st := parseSequenceType(gci.As)
	seq := xpath3.ItemSlice{ctxItem}
	if _, err := checkSequenceType(ctx, seq, st, errCodeXTTE0590, "global-context-item", ec); err != nil {
		return err
	}
	return nil
}

func validateUsedPackageGlobalContextItem(ss *Stylesheet, seen map[*Stylesheet]struct{}) error {
	if ss == nil {
		return nil
	}
	for _, pkg := range ss.usedPackages {
		if pkg == nil {
			continue
		}
		if _, ok := seen[pkg]; ok {
			continue
		}
		seen[pkg] = struct{}{}
		if pkg.globalContextItem != nil && pkg.globalContextItem.Use == ctxItemRequired {
			return dynamicError(errCodeXTTE0590, "library package declares xsl:global-context-item use=\"required\"")
		}
		if err := validateUsedPackageGlobalContextItem(pkg, seen); err != nil {
			return err
		}
	}
	return nil
}

// evaluateAllGlobals eagerly evaluates every pending global variable and
// parameter.  Errors raised here (e.g. FOAR0001 from division by zero in a
// global variable initializer) are returned to the caller so they become
// non-recoverable — they cannot be caught by xsl:try/xsl:catch.
func (ec *execContext) evaluateAllGlobals(ctx context.Context) error {
	for len(ec.globalParamDefs) > 0 || len(ec.globalVarDefs) > 0 {
		progress := false
		// Evaluate in sorted key order for deterministic temporary tree
		// creation order (see collectAllVars for rationale).
		paramNames := make([]string, 0, len(ec.globalParamDefs))
		for name := range ec.globalParamDefs {
			paramNames = append(paramNames, name)
		}
		slices.Sort(paramNames)
		for _, name := range paramNames {
			p, ok := ec.globalParamDefs[name]
			if !ok {
				continue
			}
			if _, err := ec.evaluateGlobalParam(ctx, p); err != nil {
				return err
			}
			progress = true
		}
		varNames := make([]string, 0, len(ec.globalVarDefs))
		for name := range ec.globalVarDefs {
			varNames = append(varNames, name)
		}
		slices.Sort(varNames)
		for _, name := range varNames {
			v, ok := ec.globalVarDefs[name]
			if !ok {
				continue // already evaluated as a dependency
			}
			// Abstract variables have no implementation — skip eager
			// evaluation. They will raise XTDE3052 if actually referenced.
			// Also skip variables from used packages: they may reference
			// abstract components that have not been overridden, and should
			// only be evaluated when actually referenced (lazy semantics).
			if v.Visibility == visAbstract || v.OwnerPackage != nil {
				continue
			}
			if _, err := ec.evaluateGlobalVar(ctx, v); err != nil {
				return err
			}
			progress = true
		}
		if !progress {
			break
		}
	}
	return nil
}

// invokeInitialFunction calls a named function as the transformation entry point.
func (ec *execContext) invokeInitialFunction(ctx context.Context, cfg *transformConfig) (*helium.Document, error) {
	name := resolveQName(cfg.initialFunction, ec.stylesheet.namespaces)

	// Parse the Clark notation {uri}local into a QualifiedName
	var qn xpath3.QualifiedName
	if strings.HasPrefix(name, "{") {
		if idx := strings.IndexByte(name, '}'); idx > 0 {
			qn = xpath3.QualifiedName{URI: name[1:idx], Name: name[idx+1:]}
		}
	}
	if qn.Name == "" {
		qn = xpath3.QualifiedName{Name: name}
	}

	fn := ec.findXSLFunctionByArity(qn, len(cfg.initialFunctionParams))
	if fn == nil {
		return nil, dynamicError(errCodeXTDE0041, "initial function %q not found", cfg.initialFunction)
	}
	if fn.Visibility != visPublic && fn.Visibility != visFinal {
		return nil, dynamicError(errCodeXTDE0041, "initial function %q is not public", cfg.initialFunction)
	}
	if len(cfg.initialFunctionParams) != len(fn.Params) {
		return nil, dynamicError(errCodeXTDE0041, "initial function %q expects %d parameters but %d were supplied", cfg.initialFunction, len(fn.Params), len(cfg.initialFunctionParams))
	}

	// Call the function
	userFunc := &xslUserFunc{def: fn, ec: ec}
	result, fnErr := userFunc.Call(ctx, cfg.initialFunctionParams)
	if fnErr != nil {
		return nil, fnErr
	}

	// Capture the raw XDM result for assert-type/assert-eq/etc.
	ec.rawResultSequence = result

	// Write the result to the output document
	if err := ec.outputSequence(result); err != nil {
		return nil, err
	}
	return ec.resultDoc, nil
}

// globalSourceNode returns the source document context node used when
// evaluating global variables and params, or nil when the global context
// item is absent (XPDY0002).
func (ec *execContext) globalSourceNode() helium.Node {
	if ec.globalContextAbsent {
		return nil
	}
	if ni, ok := ec.globalContextItem.(xpath3.NodeItem); ok {
		return normalizeNode(ni.Node)
	}
	if ec.globalContextItem != nil {
		// A non-node global context item (atomic/map/array/function) has no
		// context node; it is exposed via ec.contextItem instead.
		return nil
	}
	return normalizeNode(ec.sourceDoc)
}

// evaluateGlobalVar evaluates a global variable on first access.
func (ec *execContext) evaluateGlobalVar(ctx context.Context, v *variable) (xpath3.Sequence, error) {
	if ec.globalEvaluating[v.Name] {
		return nil, fmt.Errorf("%w: global variable %q", ErrCircularRef, v.Name)
	}
	ec.globalEvaluating[v.Name] = true
	defer delete(ec.globalEvaluating, v.Name)

	var val xpath3.Sequence
	ctx = withExecContext(ctx, ec)

	// XSLT 2.0 erratum XT.E19: mode="#current" in a global variable
	// must use the default mode, not whatever mode happens to be active
	// when the variable is lazily evaluated.
	savedMode := ec.currentMode
	ec.currentMode = ec.stylesheet.defaultMode
	defer func() { ec.currentMode = savedMode }()

	// Switch to the variable's owning package so that private functions
	// from that package are visible during evaluation. Invalidate the
	// cached function maps so they are rebuilt for the new package scope.
	savedPackage := ec.currentPackage
	if v.OwnerPackage != nil && v.OwnerPackage != ec.currentPackage {
		ec.currentPackage = v.OwnerPackage
		ec.cachedFns = nil
		ec.cachedFnsNS = nil
	}
	defer func() {
		if ec.currentPackage != savedPackage {
			ec.currentPackage = savedPackage
			ec.cachedFns = nil
			ec.cachedFnsNS = nil
		}
	}()

	// XPDY0002: when a variable belongs to a library package that does not
	// declare xsl:global-context-item (or declares use="absent"), the global
	// context item is absent during evaluation. Temporarily clear the context
	// node so XPath expressions that depend on it raise XPDY0002.
	savedContextNode := ec.contextNode
	savedContextItem := ec.contextItem
	savedSourceDoc := ec.sourceDoc
	if v.OwnerPackage != nil && v.OwnerPackage != ec.stylesheet {
		gci := v.OwnerPackage.globalContextItem
		if gci == nil || gci.Use == ctxItemAbsent {
			ec.contextNode = nil
			ec.contextItem = nil
			ec.sourceDoc = nil
		}
	}
	defer func() {
		ec.contextNode = savedContextNode
		ec.contextItem = savedContextItem
		ec.sourceDoc = savedSourceDoc
	}()

	// Track overriding variable for $xsl:original support
	savedOverridingVarDef := ec.overridingVarDef
	if v.OriginalVar != nil {
		ec.overridingVarDef = v
	}
	defer func() { ec.overridingVarDef = savedOverridingVarDef }()

	// Restore xpath-default-namespace from the variable's definition site
	// so that XPath expressions in the body (e.g. xsl:sequence select)
	// resolve unprefixed element names against the correct namespace.
	savedXPathDefaultNS := ec.xpathDefaultNS
	savedHasXPathDefaultNS := ec.hasXPathDefaultNS
	if v.XPathDefaultNS != "" {
		ec.xpathDefaultNS = v.XPathDefaultNS
		ec.hasXPathDefaultNS = true
	}
	defer func() {
		ec.xpathDefaultNS = savedXPathDefaultNS
		ec.hasXPathDefaultNS = savedHasXPathDefaultNS
	}()

	// Abstract variables have no implementation — raise XTDE3052.
	if v.Visibility == visAbstract {
		return nil, dynamicError(errCodeXTDE3052,
			"abstract variable $%s was invoked without being overridden", v.Name)
	}

	// Pin the static base URI to the variable's declaration site so that the
	// body (including any xsl:evaluate / unparsed-text within it) resolves
	// resources against the global's module base, never the caller's template
	// base — even when this global is first referenced lazily from a template
	// that carries its own xml:base. The override holds the declaration base
	// (module base + any xml:base override); the pinned flag prevents a fall-
	// through to currentTemplate when that base is empty.
	savedBaseOverride := ec.staticBaseURIOverride
	savedBasePinned := ec.staticBaseURIPinned
	ec.staticBaseURIOverride = v.StaticBaseURI
	ec.staticBaseURIPinned = true
	defer func() {
		ec.staticBaseURIOverride = savedBaseOverride
		ec.staticBaseURIPinned = savedBasePinned
	}()

	// Static variables use their pre-computed compile-time value.
	if v.StaticValue != nil {
		val = v.StaticValue
		delete(ec.globalVarDefs, v.Name)
		ec.globalVars[v.Name] = val
		ec.globalVarsGen++
		return val, nil
	}

	if v.Select != nil {
		result, err := ec.evalXPath(ctx, v.Select, ec.globalSourceNode())
		if err != nil {
			return nil, fmt.Errorf("error evaluating global variable %q: %w", v.Name, err)
		}
		val = result.Sequence()
	} else if len(v.Body) > 0 {
		// Global variable body must evaluate with the source document as
		// context node, not whatever the current template context is. Save
		// and restore ec.contextNode so that XPath expressions inside the
		// body (e.g. value-of select="doc/a") resolve relative to "/".
		sourceNode := ec.globalSourceNode()
		savedCtx := ec.contextNode
		ec.contextNode = sourceNode
		ec.temporaryOutputDepth++
		var err error
		if strings.HasPrefix(v.As, "document-node") && !strings.HasSuffix(v.As, "*") && !strings.HasSuffix(v.As, "+") {
			// document-node() or document-node()?: wrap body in document node
			val, err = ec.evaluateBodyAsDocument(ctx, v.Body)
		} else if v.As != "" {
			// With as attribute: evaluate as sequence constructor,
			// keeping each node as a separate item
			val, err = ec.evaluateBodyAsSequence(ctx, v.Body)
		} else {
			// No as: wrap in document node (temporary tree)
			val, err = ec.evaluateBodyAsDocument(ctx, v.Body)
		}
		ec.temporaryOutputDepth--
		ec.contextNode = savedCtx
		if err != nil {
			return nil, fmt.Errorf("error evaluating global variable %q body: %w", v.Name, err)
		}
	} else {
		// No select, no body (or empty body after whitespace stripping).
		// XSLT 3.0 §9.3: if as specifies a sequence type whose occurrence
		// indicator is ? or *, the effective value is an empty sequence.
		if v.As != "" && (strings.HasSuffix(v.As, "?") || strings.HasSuffix(v.As, "*")) {
			val = nil
		} else {
			val = xpath3.SingleString("")
		}
	}

	// Type check against the declared as type
	if v.As != "" {
		st := parseSequenceType(v.As)
		checked, err := checkSequenceType(ctx, val, st, errCodeXTTE0570, "variable $"+v.Name, ec)
		if err != nil {
			return nil, err
		}
		val = checked
	}

	ec.globalVars[v.Name] = val
	ec.globalVarsGen++
	delete(ec.globalVarDefs, v.Name)
	return val, nil
}

// evaluateGlobalParam evaluates a global param on first access.
func (ec *execContext) evaluateGlobalParam(ctx context.Context, p *param) (xpath3.Sequence, error) {
	if ec.globalEvaluating[p.Name] {
		return nil, fmt.Errorf("%w: global param %q", ErrCircularRef, p.Name)
	}
	ec.globalEvaluating[p.Name] = true
	defer delete(ec.globalEvaluating, p.Name)

	var val xpath3.Sequence
	ctx = withExecContext(ctx, ec)

	// XSLT 2.0 erratum XT.E19: mode="#current" in a global param
	// must use the default mode, not whatever mode happens to be active
	// when the param is lazily evaluated.
	savedMode := ec.currentMode
	ec.currentMode = ec.stylesheet.defaultMode
	defer func() { ec.currentMode = savedMode }()

	// Pin the static base URI to the param's declaration site so that
	// static-base-uri() (and any xsl:evaluate / unparsed-text in the body)
	// resolves against the param's module base, never the caller's template
	// base. The override holds the declaration base (module base + any xml:base
	// override); the pinned flag prevents a fall-through to currentTemplate when
	// that base is empty.
	savedBaseOverride := ec.staticBaseURIOverride
	savedBasePinned := ec.staticBaseURIPinned
	ec.staticBaseURIOverride = p.StaticBaseURI
	ec.staticBaseURIPinned = true
	defer func() {
		ec.staticBaseURIOverride = savedBaseOverride
		ec.staticBaseURIPinned = savedBasePinned
	}()

	// XTDE0050: required stylesheet parameter not supplied by the caller.
	// If we reach here (not set in initGlobalVars), no external value was given.
	if p.Required {
		return nil, dynamicError(errCodeXTDE0050, "required stylesheet parameter $%s was not supplied", p.Name)
	}

	if p.Select != nil {
		result, err := ec.evalXPath(ctx, p.Select, ec.globalSourceNode())
		if err != nil {
			return nil, fmt.Errorf("error evaluating global param %q: %w", p.Name, err)
		}
		val = result.Sequence()
	} else if len(p.Body) > 0 {
		// Global param body must evaluate with the source document as
		// context node (same as the Select path above).
		sourceNode := ec.globalSourceNode()
		savedCtx := ec.contextNode
		ec.contextNode = sourceNode
		ec.temporaryOutputDepth++
		var err error
		if p.As != "" {
			// With as attribute: evaluate as sequence constructor,
			// keeping each node as a separate item
			val, err = ec.evaluateBodyAsSequence(ctx, p.Body)
		} else {
			// No as: wrap in document node (temporary tree)
			val, err = ec.evaluateBodyAsDocument(ctx, p.Body)
		}
		ec.temporaryOutputDepth--
		ec.contextNode = savedCtx
		if err != nil {
			return nil, fmt.Errorf("error evaluating global param %q body: %w", p.Name, err)
		}
	} else if p.As == "" {
		// XSLT 3.0: param with no select, no body, and no as attribute
		// defaults to a zero-length string.
		val = xpath3.SingleString("")
	}
	// When as is specified but no select/body, val remains nil (empty sequence).
	// This is correct for as="T?" and as="T*" occurrence indicators.

	// Type check against the declared as type
	if p.As != "" {
		st := parseSequenceType(p.As)
		checked, err := checkSequenceType(ctx, val, st, errCodeXTTE0570, "param $"+p.Name, ec)
		if err != nil {
			return nil, err
		}
		val = checked
	}

	ec.globalVars[p.Name] = val
	ec.globalVarsGen++
	delete(ec.globalParamDefs, p.Name)
	return val, nil
}

// evaluateBody executes instructions and captures the result as a sequence.
// When instructions produce nodes, they are wrapped as a temporary tree.
func (ec *execContext) evaluateBody(ctx context.Context, body []instruction) (xpath3.Sequence, error) {
	// Create a temporary document to capture output
	tmpDoc := helium.NewDefaultDocument()
	tmpRoot := tmpDoc.CreateElement("_tmp")
	if err := tmpDoc.AddChild(tmpRoot); err != nil {
		return nil, err
	}

	// Push a new output frame with capture mode enabled
	frame := &outputFrame{doc: tmpDoc, current: tmpRoot, captureItems: true}
	ec.outputStack = append(ec.outputStack, frame)
	defer func() {
		ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
	}()

	if err := ec.executeSequenceConstructor(ctx, body); err != nil {
		return nil, err
	}

	// If we captured atomic items via xsl:sequence, return them directly
	if len(frame.pendingItems) > 0 {
		if tmpRoot.FirstChild() != nil {
			var seq xpath3.ItemSlice
			for child := range helium.Children(tmpRoot) {
				seq = append(seq, xpath3.NodeItem{Node: child})
			}
			seq = append(seq, frame.pendingItems...)
			return seq, nil
		}
		return frame.pendingItems, nil
	}

	// Return all children as node items
	var seq xpath3.ItemSlice
	for child := range helium.Children(tmpRoot) {
		seq = append(seq, xpath3.NodeItem{Node: child})
	}
	// Also collect attributes that were set on the temporary root
	// (e.g., from xsl:attribute with as="attribute()")
	for _, attr := range tmpRoot.Attributes() {
		seq = append(seq, xpath3.NodeItem{Node: attr})
	}
	if len(seq) > 0 {
		return seq, nil
	}

	return xpath3.EmptySequence(), nil
}

// evaluateBodyForAttr evaluates the body capturing each text fragment and
// atomic value as separate items in pendingItems. Consecutive atomics are
// NOT space-separated (itemSeparator=""); the caller controls the join
// separator via stringifySequenceWithSep. This preserves interleaving
// order between xsl:text and xsl:sequence output.
func (ec *execContext) evaluateBodyForAttr(ctx context.Context, body []instruction) (xpath3.Sequence, error) {
	tmpDoc := helium.NewDefaultDocument()
	tmpRoot := tmpDoc.CreateElement("_tmp")
	if err := tmpDoc.AddChild(tmpRoot); err != nil {
		return nil, err
	}

	emptySep := ""
	frame := &outputFrame{
		doc:               tmpDoc,
		current:           tmpRoot,
		captureItems:      true,
		sequenceMode:      true,
		separateTextNodes: true,
		itemSeparator:     &emptySep,
	}
	ec.outputStack = append(ec.outputStack, frame)
	defer func() {
		ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
	}()

	if err := ec.executeSequenceConstructor(ctx, body); err != nil {
		return nil, err
	}

	if len(frame.pendingItems) > 0 {
		return frame.pendingItems, nil
	}
	return xpath3.EmptySequence(), nil
}

// evaluateBodySeparateText is like evaluateBody but keeps each produced
// text node as a separate string item instead of letting the DOM merge
// adjacent text nodes.  This is needed by xsl:value-of with separator
// so that each text value is a distinct item for separator insertion.
func (ec *execContext) evaluateBodySeparateText(ctx context.Context, body []instruction) (xpath3.Sequence, error) {
	tmpDoc := helium.NewDefaultDocument()
	tmpRoot := tmpDoc.CreateElement("_tmp")
	if err := tmpDoc.AddChild(tmpRoot); err != nil {
		return nil, err
	}

	frame := &outputFrame{doc: tmpDoc, current: tmpRoot, captureItems: true, separateTextNodes: true}
	ec.outputStack = append(ec.outputStack, frame)
	defer func() {
		ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
	}()

	if err := ec.executeSequenceConstructor(ctx, body); err != nil {
		return nil, err
	}

	// Collect DOM children (non-text) and pending items in order
	if tmpRoot.FirstChild() != nil {
		var seq xpath3.ItemSlice
		for child := range helium.Children(tmpRoot) {
			seq = append(seq, xpath3.NodeItem{Node: child})
		}
		seq = append(seq, frame.pendingItems...)
		return seq, nil
	}
	if len(frame.pendingItems) > 0 {
		return frame.pendingItems, nil
	}
	return xpath3.EmptySequence(), nil
}

// ensureFileURI converts an absolute file system path to a file:// URI.
// Paths that already contain a scheme (e.g. "http://", "file://") are
// returned unchanged. Both POSIX-absolute ("/a/b" -> "file:///a/b") and
// Windows-absolute ("D:\\a\\b" -> "file:///D:/a/b", "\\\\host\\share" ->
// "file://host/share") paths are normalized so a later url.Parse does not read
// a Windows drive letter as a URI scheme. The Windows shapes are detected by
// string pattern (uripath), so this conversion is exercised on Linux CI too.
func ensureFileURI(path string) string {
	if path == "" {
		return path
	}
	if strings.Contains(path, "://") {
		return path
	}
	if uripath.IsWindowsAbsolute(path) {
		return uripath.WindowsToFileURI(path)
	}
	if strings.HasPrefix(path, "/") {
		return "file://" + path
	}
	return path
}

// evaluateBodyAsDocument executes instructions and wraps the result in a
// document node (temporary tree), as required by the XSLT spec for variables
// and params with content body and no select/as attributes.
// effectiveStaticBaseURI returns the static base URI for the current
// execution context, considering xml:base overrides, template base URIs,
// and the stylesheet base URI.
func (ec *execContext) effectiveStaticBaseURI() string {
	if ec.staticBaseURIOverride != "" {
		return ec.staticBaseURIOverride
	}
	if ec.staticBaseURIPinned {
		// Evaluating a global variable/param body: the static base URI is
		// pinned to the declaration site (its module base) and must never fall
		// through to the currently-executing template's base URI NOR the using
		// stylesheet's base URI. The override already carries the declaration
		// base whenever it is known; an empty pinned base is authoritative (e.g.
		// a used package whose PackageResolver returns an empty base), so we
		// return "" rather than falling back to ec.stylesheet.baseURI — falling
		// back would resolve the global's xsl:evaluate / unparsed-text against
		// the USING stylesheet, the very bug this pinning prevents.
		return ec.staticBaseURIOverride
	}
	if ec.currentTemplate != nil && ec.currentTemplate.BaseURI != "" {
		return ec.currentTemplate.BaseURI
	}
	return ec.stylesheet.baseURI
}

func (ec *execContext) evaluateBodyAsDocument(ctx context.Context, body []instruction) (xpath3.Sequence, error) {
	tmpDoc := helium.NewDefaultDocument()
	// Mark as internal so document-uri() returns empty for temporary trees.
	tmpDoc.SetProperties(tmpDoc.Properties() | helium.DocInternal)
	// Set the document URL to the static base URI so that base-uri()
	// on constructed nodes returns the correct value.
	if baseURI := ec.effectiveStaticBaseURI(); baseURI != "" {
		tmpDoc.SetURL(baseURI)
	}

	frame := &outputFrame{doc: tmpDoc, current: tmpDoc, captureItems: true, documentConstructor: true}
	ec.outputStack = append(ec.outputStack, frame)

	// Temporarily set resultDoc to the temp document so that nodes
	// created by LREs and xsl:element belong to the correct document.
	// This ensures root() on those nodes returns tmpDoc, not the main result.
	savedResultDoc := ec.resultDoc
	ec.resultDoc = tmpDoc
	defer func() {
		ec.resultDoc = savedResultDoc
		ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
	}()

	if err := ec.executeSequenceConstructor(ctx, body); err != nil {
		return nil, err
	}

	// Per XSLT spec: in document-node context (variable/param without as),
	// node items from xsl:sequence are added as children of the document
	// node, while atomic items are converted to text nodes and added as
	// space-separated text. xsl:sequence output was marked in the DOM with
	// placeholder PIs (see execXSLSequence); resolve each placeholder in
	// document order so the constructed text/nodes interleave correctly with
	// the literal result elements already built into the tree.
	if len(frame.seqPlaceholders) > 0 {
		// If the body produced exactly one document node (e.g. via xsl:copy
		// or xsl:copy-of of a document) and there is no other content, use
		// that document directly. This preserves DTD information (unparsed
		// entities, notations) that would be lost if we atomized it.
		if first := tmpDoc.FirstChild(); first != nil && first.NextSibling() == nil {
			if items, ok := frame.seqPlaceholders[first]; ok && len(items) == 1 {
				if ni, ok := items[0].(xpath3.NodeItem); ok {
					if capturedDoc, ok := ni.Node.(*helium.Document); ok {
						capturedDoc.SetProperties(capturedDoc.Properties() | helium.DocInternal)
						return xpath3.ItemSlice{xpath3.NodeItem{Node: capturedDoc}}, nil
					}
				}
			}
		}

		if err := ec.resolveSequencePlaceholders(tmpDoc, frame.seqPlaceholders); err != nil {
			return nil, err
		}
	}

	// Items captured outside xsl:sequence (e.g. xsl:copy / xsl:copy-of of a
	// document node, or attributes) are buffered in pendingItems and appended
	// after the DOM content, as before.
	if len(frame.pendingItems) > 0 {
		// If the body produced exactly one document node (e.g. via xsl:copy
		// or xsl:copy-of of a document) and the temporary doc has no DOM
		// children of its own, use that document directly. This preserves
		// DTD information (unparsed entities, notations) that would be lost
		// if we atomized the document into text.
		if len(frame.pendingItems) == 1 && tmpDoc.FirstChild() == nil {
			if ni, ok := frame.pendingItems[0].(xpath3.NodeItem); ok {
				if capturedDoc, ok := ni.Node.(*helium.Document); ok {
					capturedDoc.SetProperties(capturedDoc.Properties() | helium.DocInternal)
					return xpath3.ItemSlice{xpath3.NodeItem{Node: capturedDoc}}, nil
				}
			}
		}

		var sb strings.Builder
		prevAtomic := false
		flushText := func() error {
			if sb.Len() == 0 {
				return nil
			}
			text := tmpDoc.CreateText([]byte(sb.String()))
			sb.Reset()
			return tmpDoc.AddChild(text)
		}
		for _, item := range frame.pendingItems {
			if ni, ok := item.(xpath3.NodeItem); ok {
				if err := flushText(); err != nil {
					return nil, err
				}
				prevAtomic = false
				copied, copyErr := helium.CopyNode(ni.Node, tmpDoc)
				if copyErr != nil {
					return nil, copyErr
				}
				if err := tmpDoc.AddChild(copied); err != nil {
					return nil, err
				}
				continue
			}
			if prevAtomic {
				sb.WriteByte(' ')
			}
			av, err := xpath3.AtomizeItem(item)
			if err != nil {
				_, _ = fmt.Fprint(&sb, item)
			} else {
				s, serr := xpath3.AtomicToString(av)
				if serr != nil {
					_, _ = fmt.Fprint(&sb, item)
				} else {
					sb.WriteString(s)
				}
			}
			prevAtomic = true
		}
		if err := flushText(); err != nil {
			return nil, err
		}
	}

	// XSLT spec §5.7.2: "After the result tree is constructed, any text node
	// in the tree whose string-value is zero-length is deleted."
	for child := tmpDoc.FirstChild(); child != nil; {
		next := child.NextSibling()
		if child.Type() == helium.TextNode && len(child.Content()) == 0 {
			helium.UnlinkNode(child.(helium.MutableNode)) //nolint:forcetypeassert
		}
		child = next
	}

	return xpath3.ItemSlice{xpath3.NodeItem{Node: tmpDoc}}, nil
}

// resolveSequencePlaceholders replaces the xsl:sequence placeholder PIs in the
// document tree (created by execXSLSequence in document-constructor mode) with
// the nodes/text produced by their buffered items, in document order. Atomic
// items are converted to text and consecutive atomics — whether from the same
// or adjacent xsl:sequence outputs — are separated by a single space, while any
// intervening node from a literal result element resets that adjacency.
func (ec *execContext) resolveSequencePlaceholders(tmpDoc *helium.Document, placeholders map[helium.Node]xpath3.ItemSlice) error {
	var sb strings.Builder
	prevAtomic := false

	// makeText builds a text node from the accumulated atomic run, if any.
	flushText := func() helium.Node {
		if sb.Len() == 0 {
			return nil
		}
		text := tmpDoc.CreateText([]byte(sb.String()))
		sb.Reset()
		return text
	}

	for child := tmpDoc.FirstChild(); child != nil; {
		next := child.NextSibling()

		items, isPlaceholder := placeholders[child]
		if !isPlaceholder {
			// A literal result node breaks the atomic adjacency run.
			prevAtomic = false
			child = next
			continue
		}

		var replacement []helium.Node
		for _, item := range items {
			if ni, ok := item.(xpath3.NodeItem); ok {
				if t := flushText(); t != nil {
					replacement = append(replacement, t)
				}
				prevAtomic = false
				if ni.Node.Type() == helium.DocumentNode {
					// A document node in a sequence constructor is replaced by
					// copies of its children, spliced in document order (mirrors
					// the document-node branch in execXSLSequence). Copying the
					// document node itself would nest a document inside the temp
					// tree instead of splicing the source root element.
					for child := ni.Node.FirstChild(); child != nil; child = child.NextSibling() {
						copied, copyErr := helium.CopyNode(child, tmpDoc)
						if copyErr != nil {
							return copyErr
						}
						replacement = append(replacement, copied)
					}
					continue
				}
				copied, copyErr := helium.CopyNode(ni.Node, tmpDoc)
				if copyErr != nil {
					return copyErr
				}
				replacement = append(replacement, copied)
				continue
			}
			// Atomic item: atomize and join consecutive atomics with a space.
			if prevAtomic {
				sb.WriteByte(' ')
			}
			av, err := xpath3.AtomizeItem(item)
			if err != nil {
				_, _ = fmt.Fprint(&sb, item)
			} else {
				s, serr := xpath3.AtomicToString(av)
				if serr != nil {
					_, _ = fmt.Fprint(&sb, item)
				} else {
					sb.WriteString(s)
				}
			}
			prevAtomic = true
		}

		// If this placeholder ends an atomic run and the following sibling is
		// not another placeholder, the run is complete: flush the text now so
		// it is spliced in at this position.
		if _, nextIsPlaceholder := placeholders[next]; !nextIsPlaceholder {
			if t := flushText(); t != nil {
				replacement = append(replacement, t)
			}
			prevAtomic = false
		}

		if len(replacement) == 0 {
			helium.UnlinkNode(child.(helium.MutableNode)) //nolint:forcetypeassert
			child = next
			continue
		}
		if err := child.(helium.MutableNode).Replace(replacement...); err != nil { //nolint:forcetypeassert
			return err
		}
		child = next
	}

	// XSLT result-tree construction merges adjacent text nodes into one. The
	// Replace splices above do not merge text across the splice boundary
	// (e.g. xsl:sequence text next to xsl:text output), so coalesce adjacent
	// text-node siblings now.
	for child := tmpDoc.FirstChild(); child != nil; {
		next := child.NextSibling()
		if child.Type() == helium.TextNode && next != nil && next.Type() == helium.TextNode {
			if err := child.(helium.MutableNode).AppendText(next.Content()); err != nil { //nolint:forcetypeassert
				return err
			}
			helium.UnlinkNode(next.(helium.MutableNode)) //nolint:forcetypeassert
			continue
		}
		child = next
	}

	return nil
}

// evaluateBodyAsSequence executes instructions and captures the result as a
// flat sequence of items. Unlike evaluateBody, this keeps each produced node
// (text, element, attribute, comment, PI) as a separate item without DOM
// merging. This is needed for variables/params with an "as" attribute, where
// the body is a sequence constructor per the XSLT spec.
func (ec *execContext) evaluateBodyAsSequence(ctx context.Context, body []instruction) (xpath3.Sequence, error) {
	tmpDoc := helium.NewDefaultDocument()
	// Set the document URL to the static base URI so that element nodes
	// with xml:base can resolve against it (even when orphaned).
	if baseURI := ec.effectiveStaticBaseURI(); baseURI != "" {
		tmpDoc.SetURL(baseURI)
	}
	tmpRoot := tmpDoc.CreateElement("_tmp")
	if err := tmpDoc.AddChild(tmpRoot); err != nil {
		return nil, err
	}

	// Use sequenceMode to capture all nodes as separate items
	frame := &outputFrame{doc: tmpDoc, current: tmpRoot, captureItems: true, sequenceMode: true}
	ec.outputStack = append(ec.outputStack, frame)

	// Temporarily set resultDoc to the temp document so that nodes
	// created by copy-of belong to the correct document tree.
	savedResultDoc := ec.resultDoc
	ec.resultDoc = tmpDoc
	defer func() {
		ec.resultDoc = savedResultDoc
		ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
	}()

	if err := ec.executeSequenceConstructor(ctx, body); err != nil {
		return nil, err
	}

	if len(frame.pendingItems) > 0 {
		return frame.pendingItems, nil
	}
	return xpath3.EmptySequence(), nil
}
