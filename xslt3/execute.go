package xslt3

import (
	"context"
	"io"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
)

type execContextKey struct{}

// execContext holds XSLT transformation state. Stored inside context.Context.
type execContext struct {
	stylesheet                   *Stylesheet // immutable after construction; do not reassign
	sourceDoc                    *helium.Document
	resultDoc                    *helium.Document
	currentNode                  helium.Node // XSLT current() node
	contextNode                  helium.Node // XPath context node
	contextItem                  xpath3.Item // non-nil when context is an atomic value (for-each over atomics)
	position                     int
	size                         int
	localVars                    *varScope
	globalVars                   map[string]xpath3.Sequence
	globalVarDefs                map[string]*variable // unevaluated global variable definitions (lazy)
	globalParamDefs              map[string]*param    // unevaluated global param definitions (lazy)
	globalEvaluating             map[string]bool      // circular dependency detection
	collectingVars               bool                 // reentrancy guard for collectAllVars
	currentMode                  string
	currentTemplate              *template                  // use setCurrentTemplate(); do not assign directly
	currentTemplateBaseDir       string                     // use baseDir(); computed by setCurrentTemplate()
	currentPackage               *Stylesheet                // owning package of currently executing template/function
	xpathDefaultNS               string                     // current xpath-default-namespace
	hasXPathDefaultNS            bool                       // true when xpathDefaultNS is explicitly set
	defaultValidation            string                     // runtime copy of stylesheet.defaultValidation (save/restore per scope)
	defaultCollation             string                     // current default-collation URI (empty = codepoint)
	tunnelParams                 map[string]xpath3.Sequence // tunnel parameters passed through apply-templates
	currentGroup                 xpath3.Sequence            // current-group() value during for-each-group
	currentGroupKey              xpath3.Sequence            // current-grouping-key() value during for-each-group
	inGroupContext               bool                       // true when inside for-each-group body
	groupHasKey                  bool                       // true when innermost for-each-group uses group-by or group-adjacent
	depth                        int                        // recursion depth
	outputStack                  []*outputFrame
	keyTables                    map[string]*keyTable
	keyBuildingDepth             int // >0 when inside buildKeyTable (use-expr may recurse)
	keyUseExprDepth              int // >0 when evaluating a key's use-expression (self-ref returns empty)
	docCache                     map[string]*helium.Document
	functionResultCache          map[string]xpath3.Sequence
	cachedFns                    map[string]xpath3.Function               // cached xsltFunctions() result
	cachedFnsNS                  map[xpath3.QualifiedName]xpath3.Function // cached xsltFunctionsNS() result
	globalVarsGen                uint64                                   // incremented when globalVars changes
	cachedVarsMap                map[string]xpath3.Sequence               // cached result of collectAllVars (globals only)
	cachedVarsGen                uint64                                   // globalVarsGen at time cachedVarsMap was built
	accumulatorState             map[string]xpath3.Sequence               // accumulator name -> current value
	accumulatorStateError        map[string]error                         // accumulator name -> deferred error
	accumulatorBeforeByNode      map[helium.Node]map[string]xpath3.Sequence
	accumulatorAfterByNode       map[helium.Node]map[string]xpath3.Sequence
	accumulatorBeforeErrorByNode map[helium.Node]map[string]error // deferred errors for accumulator-before
	accumulatorAfterErrorByNode  map[helium.Node]map[string]error // deferred errors for accumulator-after
	accumulatorComputedDocs      map[helium.Node]struct{}         // document roots for which accumulator states have been computed
	activeAccumulators           map[string]struct{}              // accumulator names allowed in current source-document context
	requireStreamableAccums      bool                             // require streamable="yes" for current accumulator access context
	evaluatingAccumulator        bool
	evaluatingMergeKey           bool
	regexGroups                  []string                   // captured groups for regex-group() inside xsl:matching-substring
	breakValue                   xpath3.Sequence            // value produced by xsl:break
	nextIterParams               map[string]xpath3.Sequence // param values from xsl:next-iteration
	inMergeAction                bool                       // true inside xsl:merge-action (for XTDE3480/XTDE3510)
	errSourceLine                int                        // source line of last-executed instruction (for xsl:catch)
	errSourceModule              string                     // source module of last-executed instruction (for xsl:catch)
	msgHandler                   MessageHandler
	transformConfig                 *transformConfig
	transformCtx                 context.Context             // parent context from Transform caller (for cancellation/deadlines)
	currentTime                  time.Time                   // stable fn:current-* value for whole transformation
	schemaRegistry               *schemaRegistry             // merged schema registry for schema-aware processing
	typeAnnotations              map[helium.Node]string      // node → xs:... type annotation (schema-aware)
	preservedIDAnnotations       map[helium.Node]string      // ID/IDREF annotations preserved after input-type-annotations="strip"
	nilledElements               map[*helium.Element]struct{} // elements with xsi:nil="true" confirmed by XSD validation
	validatedDocs                map[*helium.Document]struct{} // documents that have been schema-validated
	resultDocuments              map[string]*helium.Document // secondary result documents keyed by href
	resultDocItems               map[string]xpath3.Sequence  // secondary result document items for json/adaptive serialization
	resultDocOutputDefs          map[string]*OutputDef       // effective output definition per secondary result document
	usedResultURIs               map[string]struct{}         // URIs already written by xsl:result-document (includes "")
	insideResultDocPrimary       bool                        // true while executing result-document body targeting primary URI
	currentResultDocMethod       string                      // effective output method during result-document body execution
	temporaryOutputDepth         int                         // >0 when inside a temporary output state (XTDE1480)
	primaryClaimedImplicitly     bool                        // true when implicit content has been written to primary output
	staticBaseURIOverride        string                      // non-empty when xml:base on an LRE overrides the template's base URI
	currentOutputURI             string                      // current output URI for current-output-uri() function
	inPatternMatch               bool                        // true during pattern matching (current-output-uri() returns empty)
	patternMatchErr              error                       // non-nil if a fatal error occurred during pattern matching
	inSortKeyEval                bool                        // true during sort key evaluation (current-output-uri() returns empty)
	atomicTextNodes              map[helium.Node]struct{}    // text nodes created from atomic item serialization
	nodeMemoIDs                  map[helium.Node]uint64      // stable per-transform node identities for function caching
	nextNodeMemoID               uint64
	paramDocOutputDefs           map[*resultDocumentInst]*OutputDef // per-invocation cache for parameter-document output defs
	primaryCharacterMaps         []string                     // character map names from xsl:result-document targeting primary output
	primaryResolvedCharMap       map[rune]string              // resolved character map from parameter-document for primary output
	primaryOutputOverrides       *OutputDef                   // serialization param overrides from primary xsl:result-document
	rawResultSequence            xpath3.Sequence              // raw XDM result sequence (set when initial template has as="...")
	nsFixupAllowed               map[*helium.Element]struct{} // elements whose prefix NS was auto-generated (fixup eligible)
	overridingTemplate           *template                    // currently executing overriding template (for xsl:original)
	overridingVarDef             *variable                    // currently evaluating overriding variable (for $xsl:original)
	originalFunc                 xpath3.Function              // current xsl:original function (set during overriding function call)
	docOrderCache                *xpath3.DocOrderCache        // shared document-order cache for consistent cross-document ordering
	traceWriter                  io.Writer                    // destination for fn:trace output (nil = os.Stderr)

	// cached base XPath evaluator — rebuilt when invalidation keys change
	cachedBaseEval                  xpath3.Evaluator
	cachedBaseEvalValid             bool
	cachedBaseEvalXPathDefaultNS    string
	cachedBaseEvalHasXPathDefaultNS bool
	cachedBaseEvalBaseURI           string
	cachedBaseEvalPackage           *Stylesheet
}

func (ec *execContext) setCurrentTemplate(tmpl *template) {
	ec.currentTemplate = tmpl
	if tmpl != nil && tmpl.BaseURI != "" {
		ec.currentTemplateBaseDir = filepath.Dir(tmpl.BaseURI)
	} else if ec.stylesheet.baseURI != "" {
		ec.currentTemplateBaseDir = filepath.Dir(ec.stylesheet.baseURI)
	} else {
		ec.currentTemplateBaseDir = ""
	}
}

// baseDir returns the base directory for resolving relative URIs.
// The value is computed by setCurrentTemplate from the current template's
// base URI, falling back to the stylesheet's base URI.
func (ec *execContext) baseDir() string {
	return ec.currentTemplateBaseDir
}

func withExecContext(ctx context.Context, ec *execContext) context.Context {
	return context.WithValue(ctx, execContextKey{}, ec)
}

func getExecContext(ctx context.Context) *execContext {
	v, _ := ctx.Value(execContextKey{}).(*execContext)
	return v
}

func normalizeNode(node helium.Node) helium.Node {
	if node == nil {
		return nil
	}
	v := reflect.ValueOf(node)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if v.IsNil() {
			return nil
		}
	}
	return node
}

func (ec *execContext) markAtomicTextNode(node helium.Node) {
	if node == nil {
		return
	}
	if ec.atomicTextNodes == nil {
		ec.atomicTextNodes = make(map[helium.Node]struct{})
	}
	ec.atomicTextNodes[node] = struct{}{}
}

func (ec *execContext) isAtomicTextNode(node helium.Node) bool {
	if node == nil || ec.atomicTextNodes == nil {
		return false
	}
	_, ok := ec.atomicTextNodes[node]
	return ok
}

func (ec *execContext) clearAtomicTextNode(node helium.Node) {
	if node == nil || ec.atomicTextNodes == nil {
		return
	}
	delete(ec.atomicTextNodes, node)
}

// annotateAttr applies a type annotation to a just-set attribute on an element.
// If typeName is empty, this is a no-op.
func (ec *execContext) annotateAttr(elem *helium.Element, typeName, localName, nsURI, value string) {
	if typeName == "" {
		return
	}
	if nsURI != "" {
		if attr := elem.GetAttributeNodeNS(localName, nsURI); attr != nil {
			ec.annotateNode(attr, typeName)
		}
	} else {
		for _, attr := range elem.Attributes() {
			if attr.Name() == localName {
				ec.annotateNode(attr, typeName)
				break
			}
		}
	}
	if ec.annotationIsID(typeName) {
		// Register on the document that owns this element (may be a
		// temporary document during variable/function body evaluation).
		out := ec.currentOutput()
		out.doc.RegisterID(value, elem)
	}
}

func (ec *execContext) annotationIsID(typeName string) bool {
	if typeName == "" {
		return false
	}
	if xpath3.BuiltinIsSubtypeOf(typeName, xpath3.TypeID) {
		return true
	}
	if ec.schemaRegistry == nil {
		return false
	}
	return ec.schemaRegistry.IsSubtypeOf(typeName, xpath3.TypeID)
}

// annotateNode records a type annotation for a result-tree node.
func (ec *execContext) annotateNode(node helium.Node, typeName string) {
	if typeName == "" {
		return
	}
	if ec.typeAnnotations == nil {
		ec.typeAnnotations = make(map[helium.Node]string)
	}
	ec.typeAnnotations[node] = typeName
}

// markNilled records that an element was confirmed nilled by XSD validation.
func (ec *execContext) markNilled(elem *helium.Element) {
	if ec.nilledElements == nil {
		ec.nilledElements = make(map[*helium.Element]struct{})
	}
	ec.nilledElements[elem] = struct{}{}
}

// preserveIDAnnotations populates the document-level ID index and sets
// attribute AType for IDREF/IDREFS attributes based on schema type annotations.
// This ensures fn:id() and fn:idref() still work after input-type-annotations="strip".
func (ec *execContext) preserveIDAnnotations() {
	if ec.typeAnnotations == nil {
		return
	}
	for node, ann := range ec.typeAnnotations {
		isID := isIDType(ann, ec.schemaRegistry)
		isIDRef := isIDRefType(ann, ec.schemaRegistry)
		isIDRefs := isIDRefsType(ann, ec.schemaRegistry)
		if !isID && !isIDRef && !isIDRefs {
			continue
		}

		switch typed := node.(type) {
		case *helium.Attribute:
			if isID {
				typed.SetAType(enum.AttrID)
				parent, ok := typed.Parent().(*helium.Element)
				if ok {
					if doc, ok := documentRoot(parent).(*helium.Document); ok {
						doc.RegisterID(strings.TrimSpace(typed.Value()), parent)
					}
				}
			}
			if isIDRef {
				typed.SetAType(enum.AttrIDRef)
			}
			if isIDRefs {
				typed.SetAType(enum.AttrIDRefs)
			}
		case *helium.Element:
			if isID {
				if doc, ok := documentRoot(typed).(*helium.Document); ok {
					doc.RegisterID(strings.TrimSpace(string(typed.Content())), typed)
				}
			}
		}
	}
}

func isIDType(ann string, reg *schemaRegistry) bool {
	if ann == "xs:ID" {
		return true
	}
	if reg == nil {
		return false
	}
	return reg.IsSubtypeOf(ann, "xs:ID")
}

func isIDRefType(ann string, reg *schemaRegistry) bool {
	if ann == "xs:IDREF" {
		return true
	}
	if reg == nil {
		return false
	}
	return reg.IsSubtypeOf(ann, "xs:IDREF")
}

func isIDRefsType(ann string, reg *schemaRegistry) bool {
	if ann == "xs:IDREFS" {
		return true
	}
	if reg == nil {
		return false
	}
	return reg.IsSubtypeOf(ann, "xs:IDREFS")
}

// transferNilledStatus transfers the nilled property from a source subtree to
// a destination subtree (used when copy-of/snapshot creates a deep copy).
func (ec *execContext) transferNilledStatus(src, dst helium.Node) {
	if ec.nilledElements == nil {
		return
	}
	if srcElem, ok := src.(*helium.Element); ok {
		if dstElem, ok := dst.(*helium.Element); ok {
			if _, nilled := ec.nilledElements[srcElem]; nilled {
				ec.markNilled(dstElem)
			}
			// Recursively transfer for child elements.
			srcChild := srcElem.FirstChild()
			dstChild := dstElem.FirstChild()
			for srcChild != nil && dstChild != nil {
				ec.transferNilledStatus(srcChild, dstChild)
				srcChild = srcChild.NextSibling()
				dstChild = dstChild.NextSibling()
			}
		}
	}
	// For document nodes, transfer nilled status for all children.
	if srcDoc, ok := src.(*helium.Document); ok {
		if dstDoc, ok := dst.(*helium.Document); ok {
			srcChild := srcDoc.FirstChild()
			dstChild := dstDoc.FirstChild()
			for srcChild != nil && dstChild != nil {
				ec.transferNilledStatus(srcChild, dstChild)
				srcChild = srcChild.NextSibling()
				dstChild = dstChild.NextSibling()
			}
		}
	}
}

// isNilled returns true if the element was confirmed nilled during XSD validation.
func (ec *execContext) isNilled(elem *helium.Element) bool {
	if ec.nilledElements == nil {
		return false
	}
	_, ok := ec.nilledElements[elem]
	return ok
}

// deepTransferAnnotations recursively copies type annotations from a source
// subtree to a destination subtree. The trees must have the same structure.
func (ec *execContext) deepTransferAnnotations(src, dst helium.Node) {
	if ec.typeAnnotations == nil {
		return
	}
	if ann, ok := ec.typeAnnotations[src]; ok {
		ec.annotateNode(dst, ann)
	}
	// Transfer attribute annotations
	if srcElem, ok := src.(*helium.Element); ok {
		if dstElem, ok := dst.(*helium.Element); ok {
			for _, srcAttr := range srcElem.Attributes() {
				if ann, ok := ec.typeAnnotations[srcAttr]; ok {
					for _, dstAttr := range dstElem.Attributes() {
						if srcAttr.LocalName() == dstAttr.LocalName() && srcAttr.URI() == dstAttr.URI() {
							ec.annotateNode(dstAttr, ann)
							break
						}
					}
				}
			}
		}
	}
	// Transfer child annotations recursively
	srcChild := src.FirstChild()
	dstChild := dst.FirstChild()
	for srcChild != nil && dstChild != nil {
		ec.deepTransferAnnotations(srcChild, dstChild)
		srcChild = srcChild.NextSibling()
		dstChild = dstChild.NextSibling()
	}
}

// transferAnnotationsForCopy transfers type annotations from a source node to
// newly-copied children of parent. lastBefore is the last child of parent
// before the copy (nil if parent was empty). For document-node sources, the
// children of the document are matched pairwise with the newly-added children.
// For non-document sources, the single new child (parent's last child) is
// matched with the source.
func (ec *execContext) transferAnnotationsForCopy(src helium.Node, parent helium.Node, lastBefore helium.Node) {
	if ec.typeAnnotations == nil {
		return
	}
	if src.Type() == helium.DocumentNode {
		// Document node: children were copied individually.
		// Walk src children and match them to the newly-added children.
		var firstNew helium.Node
		if lastBefore == nil {
			firstNew = parent.FirstChild()
		} else {
			firstNew = lastBefore.NextSibling()
		}
		srcChild := src.FirstChild()
		dstChild := firstNew
		for srcChild != nil && dstChild != nil {
			ec.deepTransferAnnotations(srcChild, dstChild)
			srcChild = srcChild.NextSibling()
			dstChild = dstChild.NextSibling()
		}
	} else {
		// Non-document: a single node was copied as the last child.
		last := parent.LastChild()
		if last != nil {
			ec.deepTransferAnnotations(src, last)
		}
	}
}

// varScope is a variable scope chain.
type varScope struct {
	vars           map[string]xpath3.Sequence
	deferredErrors map[string]error // errors deferred until variable is actually used
	parent         *varScope
}

var varScopePool = sync.Pool{
	New: func() any { return &varScope{} },
}

func (vs *varScope) lookup(name string) (xpath3.Sequence, bool) {
	for s := vs; s != nil; s = s.parent {
		if v, ok := s.vars[name]; ok {
			return v, true
		}
	}
	return nil, false
}

// lookupDeferred returns a deferred error for a variable, if one exists.
func (vs *varScope) lookupDeferred(name string) error {
	for s := vs; s != nil; s = s.parent {
		if s.deferredErrors != nil {
			if err, ok := s.deferredErrors[name]; ok {
				return err
			}
		}
	}
	return nil
}

func (ec *execContext) pushVarScope() {
	vs := varScopePool.Get().(*varScope)
	vs.parent = ec.localVars
	ec.localVars = vs
}

func (ec *execContext) popVarScope() {
	if ec.localVars == nil {
		return
	}
	old := ec.localVars
	ec.localVars = old.parent
	old.parent = nil
	clear(old.vars)
	clear(old.deferredErrors)
	varScopePool.Put(old)
}

func (ec *execContext) setVar(name string, value xpath3.Sequence) {
	if ec.localVars == nil {
		ec.localVars = varScopePool.Get().(*varScope)
	}
	if ec.localVars.vars == nil {
		ec.localVars.vars = make(map[string]xpath3.Sequence, 4)
	}
	ec.localVars.vars[name] = value
}

// setVarDeferred stores a variable with a deferred error. The variable
// is set to an empty sequence; the error is raised only when the variable
// is actually referenced via ResolveVariable.
func (ec *execContext) setVarDeferred(name string, err error) {
	if ec.localVars == nil {
		ec.localVars = varScopePool.Get().(*varScope)
	}
	if ec.localVars.vars == nil {
		ec.localVars.vars = make(map[string]xpath3.Sequence, 4)
	}
	ec.localVars.vars[name] = xpath3.EmptySequence()
	if ec.localVars.deferredErrors == nil {
		ec.localVars.deferredErrors = make(map[string]error, 2)
	}
	ec.localVars.deferredErrors[name] = err
}

func (ec *execContext) resolvePrefix(prefix string) string {
	if uri, ok := ec.stylesheet.namespaces[prefix]; ok {
		return uri
	}
	// The xml prefix is universally predeclared per XML Namespaces spec.
	if prefix == lexicon.PrefixXML {
		return lexicon.NamespaceXML
	}
	return ""
}

// currentOutput returns the current output frame.
func (ec *execContext) currentOutput() *outputFrame {
	return ec.outputStack[len(ec.outputStack)-1]
}

func (ec *execContext) addNodeUntracked(node helium.Node) error {
	out := ec.currentOutput()
	if err := out.current.AddChild(node); err != nil {
		return err
	}
	// Record text nodes in the active conditional scope so that separator
	// artifacts can be removed when on-empty fires (separators between
	// zero-length strings must not make content non-empty per XSLT 3.0).
	// Comment nodes (used as on-empty/on-non-empty placeholders) are
	// excluded — they are managed by resolveConditionalScope directly.
	if node.Type() == helium.TextNode {
		if n := len(out.conditionalScopes); n > 0 {
			out.conditionalScopes[n-1].untrackedNodes = append(out.conditionalScopes[n-1].untrackedNodes, node)
		}
	}
	return nil
}

// addNode adds a node to the current output insertion point.
func (ec *execContext) addNode(node helium.Node) error {
	out := ec.currentOutput()
	// When sequenceMode is set, capture all nodes as separate items in
	// the pending list. This prevents DOM text-node merging and keeps
	// attributes, comments, PIs, and elements as distinct sequence items.
	// Used by variable/param/with-param with an as attribute.
	if out.sequenceMode {
		out.pendingItems = append(out.pendingItems, xpath3.NodeItem{Node: node})
		out.noteOutput()
		return nil
	}

	nodeType := node.Type()
	isText := nodeType == helium.TextNode

	// When separateTextNodes is set, capture each text node as a separate
	// string item to avoid DOM text-node merging.  This is needed by
	// xsl:value-of with separator + body content so that each produced
	// text value remains a distinct item for separator insertion.
	if out.separateTextNodes && isText {
		// Keep as NodeItem so that mergeAdjacentTextNodes can merge them.
		out.pendingItems = append(out.pendingItems, xpath3.NodeItem{Node: node})
		out.noteOutput()
		return nil
	}
	// Reset atomic adjacency tracking when any node (text or element)
	// is added via addNode. Text nodes from xsl:value-of or xsl:text
	// break the "consecutive atomic" chain from xsl:sequence outputs.
	out.prevWasAtomic = false

	// For text nodes, compute whitespace-only once; used by multiple checks below.
	isNonWhitespaceText := isText && strings.TrimSpace(string(node.Content())) != ""

	// When item-separator is explicitly set, insert it between any pair
	// of adjacent items in the result sequence (XSLT 3.0 serialization).
	// This covers nodes produced by instructions like xsl:comment, xsl:element,
	// xsl:processing-instruction that don't go through execXSLSequence.
	// Only non-text nodes trigger this; text nodes from xsl:text/xsl:value-of
	// are not separate "items" in the serialization sense.
	if out.itemSeparator != nil && out.prevHadOutput && !isText && nodeType != helium.AttributeNode {
		sepStr := *out.itemSeparator
		if sepStr != "" {
			sep, tErr := ec.resultDoc.CreateText([]byte(sepStr))
			if tErr != nil {
				return tErr
			}
			if err := out.current.AddChild(sep); err != nil {
				return err
			}
		}
	}
	// XTRE1495: if the primary output was claimed by an xsl:result-document
	// and we are writing implicit content to the base (primary) frame, error.
	if len(ec.outputStack) == 1 && !ec.insideResultDocPrimary {
		if _, claimed := ec.usedResultURIs[""]; claimed {
			if isText && !isNonWhitespaceText {
				return nil
			}
			return dynamicError(errCodeXTRE1495, "cannot write to primary output: URI already used by xsl:result-document")
		}
		if !isText || isNonWhitespaceText {
			ec.primaryClaimedImplicitly = true
		}
	}
	// Zero-length text nodes are discarded from the result tree
	// (XSLT 3.0 §11.4.1: "If the result ... is a zero-length string,
	// then no text node is created").
	if isText && len(node.Content()) == 0 {
		return nil
	}
	// When a text node is about to be merged (via addChild) with an
	// existing text node that was marked as atomic (from xsl:on-empty /
	// xsl:sequence splice), clear the atomic marker.  The DOM's addChild
	// merges adjacent text nodes, so the combined text is no longer purely
	// an atomic serialization and must not trigger inter-atomic space
	// separators in subsequent splice operations.
	if isText {
		if last := out.current.LastChild(); last != nil && last.Type() == helium.TextNode {
			ec.clearAtomicTextNode(last)
		}
	}
	if err := out.current.AddChild(node); err != nil {
		return err
	}
	out.noteOutput()
	// Track that output was produced for item-separator insertion
	// between non-atomic items (e.g., comment→atomic, element→atomic).
	// Whitespace-only text nodes between instructions don't count.
	if !isText || isNonWhitespaceText {
		out.prevHadOutput = true
	}
	return nil
}

func (ec *execContext) executeSequenceConstructor(ctx context.Context, body []instruction) error {
	if len(body) == 0 {
		return nil
	}
	out := ec.currentOutput()
	out.seqConstructorGen++
	scopeIdx := len(out.conditionalScopes)
	out.conditionalScopes = append(out.conditionalScopes, conditionalScope{})
	for _, inst := range body {
		_, isOnEmpty := inst.(*onEmptyInst)
		_, isOnNonEmpty := inst.(*onNonEmptyInst)
		snap := out.outputSerial
		if err := ec.executeInstruction(ctx, inst); err != nil {
			out.conditionalScopes = out.conditionalScopes[:scopeIdx]
			return err
		}
		if !isOnEmpty && !isOnNonEmpty && out.outputSerial != snap {
			out.conditionalScopes[scopeIdx].hasOutput = true
		}
	}
	scope := out.conditionalScopes[scopeIdx]
	out.conditionalScopes = out.conditionalScopes[:scopeIdx]
	return ec.resolveConditionalScope(scope)
}

func (ec *execContext) resolveConditionalScope(scope conditionalScope) error {
	// When on-empty fires (!scope.hasOutput), remove nodes that were added
	// via addNodeUntracked. These are separator artifacts inserted between
	// zero-length atomic values and must not appear in the output (XSLT 3.0:
	// "Separators between zero-length strings do not make the content
	// non-empty").
	if !scope.hasOutput {
		for _, n := range scope.untrackedNodes {
			helium.UnlinkNode(n)
		}
	}

	for _, action := range scope.actions {
		shouldRun := (action.kind == conditionalOnEmpty && !scope.hasOutput) ||
			(action.kind == conditionalOnNonEmpty && scope.hasOutput)
		if !shouldRun {
			if action.placeholder != nil {
				helium.UnlinkNode(action.placeholder)
			}
			continue
		}

		if err := ec.spliceConditionalSequence(action.placeholder, action.content, action.prevWasAtomic); err != nil {
			return err
		}
	}
	return nil
}

func (ec *execContext) spliceConditionalSequence(placeholder helium.Node, seq xpath3.Sequence, prevWasAtomic bool) error {
	if placeholder == nil {
		return ec.outputSequence(seq)
	}

	parent := placeholder.Parent()
	out := ec.currentOutput()
	savedCurrent := out.current
	if parent != nil {
		out.current = parent
		defer func() {
			out.current = savedCurrent
		}()
	}

	seq = flattenArraysInSequence(seq)
	var nodes []helium.Node
	// Determine whether the content preceding this splice was an atomic value.
	// We use two signals:
	//   (a) prevWasAtomic captured at registration time (reflects the
	//       xsl:sequence / xsl:value-of state at the point on-empty was seen)
	//   (b) isAtomicTextNode on the placeholder's previous DOM sibling
	//       (reflects spliced atomic text that hasn't been merged away)
	// Either being true means we need an inter-atomic space separator.
	prevAtomic := prevWasAtomic || ec.isAtomicTextNode(placeholder.PrevSibling())
	for item := range sequence.Items(seq) {
		switch v := item.(type) {
		case xpath3.NodeItem:
			prevAtomic = false
			switch v.Node.Type() {
			case helium.DocumentNode:
				for child := v.Node.FirstChild(); child != nil; child = child.NextSibling() {
					copied, err := helium.CopyNode(child, ec.resultDoc)
					if err != nil {
						return err
					}
					nodes = append(nodes, copied)
				}
			case helium.AttributeNode:
				elem, ok := parent.(*helium.Element)
				if !ok {
					return dynamicError(errCodeXTDE0410, "cannot add attribute to a non-element node")
				}
				copyAttributeToElement(elem, v.Node.(*helium.Attribute))
				out.noteOutput()
			default:
				copied, err := helium.CopyNode(v.Node, ec.resultDoc)
				if err != nil {
					return err
				}
				nodes = append(nodes, copied)
			}
		case xpath3.AtomicValue:
			s, err := xpath3.AtomicToString(v)
			if err != nil {
				return err
			}
			if s == "" {
				// Empty-string atomics produce no text, but if preceded by
				// another atomic, insert an inter-atomic separator so that
				// on-non-empty select="''" properly contributes to the
				// inter-atomic space chain.
				if prevAtomic {
					sep, sErr := ec.resultDoc.CreateText([]byte(" "))
					if sErr != nil {
						return sErr
					}
					nodes = append(nodes, sep)
				}
				prevAtomic = true
				continue
			}
			if prevAtomic {
				sep, err := ec.resultDoc.CreateText([]byte(" "))
				if err != nil {
					return err
				}
				nodes = append(nodes, sep)
			}
			text, err := ec.resultDoc.CreateText([]byte(s))
			if err != nil {
				return err
			}
			nodes = append(nodes, text)
			ec.markAtomicTextNode(text)
			prevAtomic = true
		}
	}

	if len(nodes) == 0 {
		// Even though no DOM nodes were produced, update prevWasAtomic
		// if an empty-string atomic was seen. This ensures that
		// subsequent xsl:sequence or on-empty instructions see the
		// correct atomic adjacency state (e.g., on-non-empty with
		// select="''" should contribute to the inter-atomic chain).
		if prevAtomic {
			out.prevWasAtomic = true
		}
		helium.UnlinkNode(placeholder)
		return nil
	}

	spliceReplace(placeholder, nodes)
	// Fix namespace declarations on spliced elements that moved into a
	// context with a different default namespace (e.g. a no-namespace
	// element placed under a parent with xmlns="...").
	for _, n := range nodes {
		if elem, ok := n.(*helium.Element); ok {
			ec.fixNamespacesAfterCopy(elem)
		}
	}
	// Update prevWasAtomic to reflect the spliced content so that
	// subsequent instructions see the correct atomic adjacency state.
	out.prevWasAtomic = prevAtomic
	out.noteOutput()
	return nil
}

func spliceReplace(target helium.Node, nodes []helium.Node) {
	if len(nodes) == 0 {
		helium.UnlinkNode(target)
		return
	}

	afterTarget := target.NextSibling()
	_ = target.Replace(nodes[0])

	prev := nodes[0]
	for i := 1; i < len(nodes); i++ {
		cur := nodes[i]
		cur.SetParent(prev.Parent())
		cur.SetPrevSibling(prev)
		prev.SetNextSibling(cur)
		prev = cur
	}

	last := nodes[len(nodes)-1]
	last.SetNextSibling(afterTarget)
	if afterTarget != nil {
		afterTarget.SetPrevSibling(last)
	}
	// Update parent's LastChild if the target was the last child.
	// Replace only updates LastChild for nodes[0]; when additional
	// nodes were spliced after it, the parent's LastChild must point
	// to the true last node.
	if afterTarget == nil && len(nodes) > 1 {
		if parent := last.Parent(); parent != nil {
			helium.SetLastChild(parent, last)
		}
	}
}

// collectPackageNamespaces recursively collects all namespace bindings from
// used packages into the provided map. Main stylesheet bindings take precedence.
func collectPackageNamespaces(ss *Stylesheet, ns map[string]string) {
	for _, pkg := range ss.usedPackages {
		collectPackageNamespaces(pkg, ns)
		for k, v := range pkg.namespaces {
			if k == "" {
				continue
			}
			if _, exists := ns[k]; !exists {
				ns[k] = v
			}
		}
	}
}

func (ec *execContext) collectionResolver() xpath3.CollectionResolver {
	if ec.transformConfig == nil || ec.transformConfig.collectionResolver == nil {
		return nil
	}
	if len(ec.stylesheet.stripSpace) == 0 && len(ec.stylesheet.preserveSpace) == 0 {
		return ec.transformConfig.collectionResolver
	}
	return strippingCollectionResolver{
		base: ec.transformConfig.collectionResolver,
		ec:   ec,
	}
}

type strippingCollectionResolver struct {
	base xpath3.CollectionResolver
	ec   *execContext
}

func (r strippingCollectionResolver) ResolveCollection(uri string) (xpath3.Sequence, error) {
	seq, err := r.base.ResolveCollection(uri)
	if err != nil {
		return nil, err
	}

	out := make(xpath3.ItemSlice, 0, sequence.Len(seq))
	for item := range sequence.Items(seq) {
		ni, ok := item.(xpath3.NodeItem)
		if !ok {
			out = append(out, item)
			continue
		}

		copied, err := r.ec.copyCollectionNode(ni.Node)
		if err != nil {
			return nil, err
		}
		out = append(out, xpath3.NodeItem{Node: copied})
	}
	return out, nil
}

func (r strippingCollectionResolver) ResolveURICollection(uri string) ([]string, error) {
	return r.base.ResolveURICollection(uri)
}

func (ec *execContext) copyCollectionNode(node helium.Node) (helium.Node, error) {
	if node == nil {
		return nil, nil
	}

	if doc, ok := node.(*helium.Document); ok {
		copied, err := helium.CopyDoc(doc)
		if err != nil {
			return nil, err
		}
		copied.SetURL(doc.URL())
		ec.stripWhitespaceFromDoc(copied)
		return copied, nil
	}

	dst := helium.NewDefaultDocument()
	copied, err := helium.CopyNode(node, dst)
	if err != nil {
		return nil, err
	}
	if err := dst.AddChild(copied); err != nil {
		return nil, err
	}
	if owner := node.OwnerDocument(); owner != nil {
		dst.SetURL(owner.URL())
	}
	ec.stripWhitespaceFromDoc(dst)
	return copied, nil
}

// sortXPathEvalState creates a reusable xpath3.EvalState from the base context.
// Used by sort to avoid per-item newEvalContext allocations.
func (ec *execContext) sortXPathEvalState() *xpath3.EvalState {
	return ec.xpathEvaluator().NewEvalState(ec.xpathContext(), nil)
}

// xpathContext returns a context.Context suitable for XPath evaluation.
// It carries cancellation/deadlines and the xslt3 exec context, but
// NOT XPath config (that comes from the Evaluator).
func (ec *execContext) xpathContext() context.Context {
	ctx := ec.transformCtx
	if ctx == nil {
		ctx = context.Background()
	}
	return withExecContext(ctx, ec)
}

// baseXPathEvaluator returns the cached base XPath evaluator, rebuilding
// it only when the invalidation keys change (variable generation, xpath
// default namespace, or effective base URI).
func (ec *execContext) baseXPathEvaluator() xpath3.Evaluator {
	baseURI := ec.effectiveStaticBaseURI()
	if ec.cachedBaseEvalValid &&
		ec.cachedBaseEvalXPathDefaultNS == ec.xpathDefaultNS &&
		ec.cachedBaseEvalHasXPathDefaultNS == ec.hasXPathDefaultNS &&
		ec.cachedBaseEvalBaseURI == baseURI &&
		ec.cachedBaseEvalPackage == ec.currentPackage {
		return ec.cachedBaseEval
	}

	eval := ec.buildBaseXPathEvaluator(baseURI)

	ec.cachedBaseEval = eval
	ec.cachedBaseEvalValid = true
	ec.cachedBaseEvalXPathDefaultNS = ec.xpathDefaultNS
	ec.cachedBaseEvalHasXPathDefaultNS = ec.hasXPathDefaultNS
	ec.cachedBaseEvalBaseURI = baseURI
	ec.cachedBaseEvalPackage = ec.currentPackage
	return eval
}

// buildBaseXPathEvaluator constructs the base evaluator from scratch.
// Variables and functions are NOT included — they change per-scope
// and are applied in xpathEvaluator() as per-call overlays.
func (ec *execContext) buildBaseXPathEvaluator(baseURI string) xpath3.Evaluator {
	eval := xpath3.NewEvaluator(xpath3.EvalBorrowing).
		VariableResolver(ec).
		FunctionResolver(ec).
		CurrentTime(ec.currentTime).
		ImplicitTimezone(ec.currentTime.Location()).
		AllowXML11Chars().
		TraceWriter(ec.traceWriter)

	if len(ec.stylesheet.namespaces) > 0 || ec.hasXPathDefaultNS {
		ns := make(map[string]string, len(ec.stylesheet.namespaces)+1)
		collectPackageNamespaces(ec.stylesheet, ns)
		for k, v := range ec.stylesheet.namespaces {
			if k == "" && !ec.hasXPathDefaultNS {
				continue
			}
			ns[k] = v
		}
		if ec.hasXPathDefaultNS {
			ns[""] = ec.xpathDefaultNS
		}
		eval = eval.Namespaces(ns).StrictPrefixes()
	}
	if baseURI != "" {
		eval = eval.BaseURI(ensureFileURI(baseURI))
	}
	dfmts := ec.effectiveDecimalFormats()
	if len(dfmts) > 0 {
		for qn, df := range dfmts {
			if qn == (xpath3.QualifiedName{}) {
				eval = eval.DefaultDecimalFormat(df)
			}
		}
		eval = eval.NamedDecimalFormats(dfmts)
	}
	if resolver := ec.collectionResolver(); resolver != nil {
		eval = eval.CollectionResolver(resolver)
	}
	return eval
}

// effectiveDecimalFormats returns the decimal formats for the current
// execution scope. When executing code from a used package, the
// package's own decimal formats are used (package-scoped isolation).
func (ec *execContext) effectiveDecimalFormats() map[xpath3.QualifiedName]xpath3.DecimalFormat {
	if ec.currentPackage != nil && ec.currentPackage != ec.stylesheet {
		return ec.currentPackage.decimalFormats
	}
	return ec.stylesheet.decimalFormats
}

// effectiveKeys returns the key definitions for the current execution
// scope. When executing code from a used package, the package's own
// keys are used (package-scoped isolation).
func (ec *execContext) effectiveKeys() map[string][]*keyDef {
	if ec.currentPackage != nil && ec.currentPackage != ec.stylesheet {
		return ec.currentPackage.keys
	}
	return ec.stylesheet.keys
}

// effectiveCharacterMaps returns the character maps for the current
// execution scope. When executing code from a used package, the
// package's own character maps are used (package-scoped isolation).
func (ec *execContext) effectiveCharacterMaps() map[string]*characterMapDef {
	if ec.currentPackage != nil && ec.currentPackage != ec.stylesheet {
		return ec.currentPackage.characterMaps
	}
	return ec.stylesheet.characterMaps
}

// effectiveNamespaceAliases returns the namespace aliases for the
// current execution scope. When executing code from a used package,
// the package's own aliases are used (package-scoped isolation).
func (ec *execContext) effectiveNamespaceAliases() []namespaceAlias {
	if ec.currentPackage != nil && ec.currentPackage != ec.stylesheet {
		return ec.currentPackage.namespaceAliases
	}
	return ec.stylesheet.namespaceAliases
}

// effectiveOutputs returns the output definitions for the current
// execution scope. When executing code from a used package, the
// package's own outputs are used (package-scoped isolation).
func (ec *execContext) effectiveOutputs() map[string]*OutputDef {
	if ec.currentPackage != nil && ec.currentPackage != ec.stylesheet {
		return ec.currentPackage.outputs
	}
	return ec.stylesheet.outputs
}

// effectiveStylesheet returns the stylesheet for the current execution
// scope. When executing code from a used package, the package is returned.
func (ec *execContext) effectiveStylesheet() *Stylesheet {
	if ec.currentPackage != nil && ec.currentPackage != ec.stylesheet {
		return ec.currentPackage
	}
	return ec.stylesheet
}

// xpathEvaluator returns the base evaluator with per-call overlays
// (variables, position, size, context item, collation, doc-order cache).
func (ec *execContext) xpathEvaluator() xpath3.Evaluator {
	eval := ec.baseXPathEvaluator().
		Variables(xpath3.VariablesFromMap(ec.collectAllVars())).
		Functions(xpath3.FunctionLibraryFromMaps(ec.xsltFunctions(), ec.xsltFunctionsNS()))
	if ec.typeAnnotations != nil {
		eval = eval.TypeAnnotations(ec.typeAnnotations)
	}
	if ec.preservedIDAnnotations != nil {
		eval = eval.PreservedIDAnnotations(ec.preservedIDAnnotations)
	}
	if ec.schemaRegistry != nil {
		eval = eval.SchemaDeclarations(ec.schemaRegistry)
	}
	if ec.position > 0 {
		eval = eval.Position(ec.position)
	}
	if ec.size > 0 {
		eval = eval.Size(ec.size)
	}
	if ec.contextItem != nil {
		eval = eval.ContextItem(ec.contextItem)
	}
	if ec.defaultCollation != "" {
		eval = eval.DefaultCollation(ec.defaultCollation)
	}
	if ec.docOrderCache != nil {
		eval = eval.DocOrderCache(ec.docOrderCache)
	}
	return eval
}

// evalXPath evaluates an XPath expression using the Evaluator-based path.
func (ec *execContext) evalXPath(expr *xpath3.Expression, node helium.Node) (*xpath3.Result, error) {
	return ec.xpathEvaluator().Evaluate(ec.xpathContext(), expr, node)
}

func (ec *execContext) collectAllVars() map[string]xpath3.Sequence {
	// Eagerly evaluate all pending global vars/params, but only at the
	// top level. Nested calls (from within evaluateGlobalVar/param →
	// newXPathContext → collectAllVars) just snapshot what's available;
	// the lazy lookupVar path handles remaining references on demand.
	if !ec.collectingVars && (len(ec.globalVarDefs) > 0 || len(ec.globalParamDefs) > 0) {
		ec.collectingVars = true
		defer func() { ec.collectingVars = false }()
		for len(ec.globalVarDefs) > 0 || len(ec.globalParamDefs) > 0 {
			progress := false
			// Iterate in sorted key order so that temporary trees created
			// during variable evaluation are registered in the DocOrderCache
			// deterministically. Without this, cross-document ordering in
			// unions (e.g. $v1 | $v2) depends on Go's randomized map iteration.
			varNames := make([]string, 0, len(ec.globalVarDefs))
			for name := range ec.globalVarDefs {
				varNames = append(varNames, name)
			}
			slices.Sort(varNames)
			for _, name := range varNames {
				v, ok := ec.globalVarDefs[name]
				if !ok {
					continue
				}
				if _, err := ec.evaluateGlobalVar(v); err == nil {
					progress = true
				}
			}
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
				if _, err := ec.evaluateGlobalParam(p); err == nil {
					progress = true
				}
			}
			if !progress {
				break // avoid infinite loop on circular deps
			}
		}
	}

	// Fast path: when there are no local variables and globals haven't
	// changed since the last call, return the cached map directly.
	if ec.localVars == nil && ec.cachedVarsGen == ec.globalVarsGen && ec.cachedVarsMap != nil {
		return ec.cachedVarsMap
	}

	vars := make(map[string]xpath3.Sequence, len(ec.globalVars))
	// Start with globals
	for k, v := range ec.globalVars {
		vars[k] = v
	}
	// Walk from outermost to innermost scope so inner scopes override
	var scopes []*varScope
	for s := ec.localVars; s != nil; s = s.parent {
		scopes = append(scopes, s)
	}
	for i := len(scopes) - 1; i >= 0; i-- {
		for k, v := range scopes[i].vars {
			vars[k] = v
		}
	}

	// Cache the result when it's globals-only (no local scopes)
	if ec.localVars == nil {
		ec.cachedVarsMap = vars
		ec.cachedVarsGen = ec.globalVarsGen
	}

	return vars
}

// ResolveFunction implements xpath3.FunctionResolver. It resolves
// xsl:original() calls without exposing the function to function-lookup.
func (ec *execContext) ResolveFunction(_ context.Context, uri, name string, arity int) (xpath3.Function, bool, error) {
	if uri == lexicon.NamespaceXSLT && name == "original" && ec.originalFunc != nil {
		return ec.originalFunc, true, nil
	}
	return nil, false, nil
}

// ResolveVariable implements xpath3.VariableResolver. It lazily evaluates
// global variables and parameters that have not yet been evaluated, allowing
// XPath expressions to reference globals that aren't in the snapshot built
// by collectAllVars (e.g. during circular-but-unused variable scenarios).
func (ec *execContext) ResolveVariable(_ context.Context, name string) (xpath3.Sequence, bool, error) {
	// Handle $xsl:original — resolve to the original overridden variable's value
	if name == "{"+lexicon.NamespaceXSLT+"}original" && ec.overridingVarDef != nil && ec.overridingVarDef.OriginalVar != nil {
		val, err := ec.evaluateGlobalVar(ec.overridingVarDef.OriginalVar)
		if err != nil {
			return nil, false, err
		}
		return val, true, nil
	}
	// Check if the variable is already evaluated as a global
	if v, ok := ec.globalVars[name]; ok {
		return v, true, nil
	}
	// Check local vars (in case the scope was pushed after the snapshot)
	if ec.localVars != nil {
		if v, ok := ec.localVars.lookup(name); ok {
			// Check for deferred errors — raised only on actual use
			if err := ec.localVars.lookupDeferred(name); err != nil {
				return nil, false, err
			}
			return v, true, nil
		}
	}
	// Try to lazily evaluate a pending global variable
	if def, ok := ec.globalVarDefs[name]; ok {
		val, err := ec.evaluateGlobalVar(def)
		if err != nil {
			return nil, false, err
		}
		return val, true, nil
	}
	// Try to lazily evaluate a pending global parameter
	if def, ok := ec.globalParamDefs[name]; ok {
		val, err := ec.evaluateGlobalParam(def)
		if err != nil {
			return nil, false, err
		}
		return val, true, nil
	}
	return nil, false, nil
}
