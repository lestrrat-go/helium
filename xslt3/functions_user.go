package xslt3

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
)

type xslMultiArityFunc struct {
	variants []*xslUserFunc
	minArity int
	maxArity int
}

func (f *xslMultiArityFunc) MinArity() int { return f.minArity }
func (f *xslMultiArityFunc) MaxArity() int { return f.maxArity }
func (f *xslMultiArityFunc) Call(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	for _, v := range f.variants {
		if len(args) == len(v.def.Params) {
			return v.Call(ctx, args)
		}
	}
	return nil, fmt.Errorf("xpath3: arity mismatch: no overload accepts %d arguments", len(args))
}

func (f *xslMultiArityFunc) findVariant(arity int) *xslUserFunc {
	for _, v := range f.variants {
		if len(v.def.Params) == arity {
			return v
		}
	}
	return nil
}

func (f *xslMultiArityFunc) FuncParamTypesForArity(arity int) []xpath3.SequenceType {
	if v := f.findVariant(arity); v != nil {
		return v.FuncParamTypes()
	}
	return nil
}

func (f *xslMultiArityFunc) FuncReturnTypeForArity(arity int) *xpath3.SequenceType {
	if v := f.findVariant(arity); v != nil {
		return v.FuncReturnType()
	}
	return nil
}

func (f *xslMultiArityFunc) addVariant(v *xslUserFunc) {
	f.variants = append(f.variants, v)
	arity := len(v.def.Params)
	if arity < f.minArity {
		f.minArity = arity
	}
	if arity > f.maxArity {
		f.maxArity = arity
	}
}

// xslUserFunc wraps an xsl:function for use as an xpath3.Function.
type xslUserFunc struct {
	def *xslFunction
	ec  *execContext
}

func (f *xslUserFunc) MinArity() int { return len(f.def.Params) }
func (f *xslUserFunc) MaxArity() int { return len(f.def.Params) }

// FuncParamTypes returns the parameter types prepareCall parsed at compile
// time. Callers share the slice and must not modify it.
func (f *xslUserFunc) FuncParamTypes() []xpath3.SequenceType { return f.def.paramTypes }

// FuncReturnType returns the return type prepareCall parsed at compile time.
func (f *xslUserFunc) FuncReturnType() *xpath3.SequenceType { return f.def.returnType }

// prepareCall derives the data every call needs from the compiled function, so
// that a call parses no sequence type and inspects no body instruction. It runs
// once, when the function is compiled.
func (fn *xslFunction) prepareCall() {
	fn.paramTypes = make([]xpath3.SequenceType, 0, len(fn.Params))
	fn.paramCheckTypes = make([]sequenceType, len(fn.Params))
	for i, p := range fn.Params {
		if p.As != "" {
			fn.paramCheckTypes[i] = parseSequenceType(p.As)
		}
		if fn.paramTypes == nil {
			continue
		}
		as := p.As
		if as == "" {
			as = "item()*"
		}
		st, err := xpath3.ParseSequenceType(as)
		if err != nil {
			fn.paramTypes = nil
			continue
		}
		fn.paramTypes = append(fn.paramTypes, st)
	}

	fn.returnType = nil
	if fn.As != "" {
		fn.returnCheckType = parseSequenceType(fn.As)
		if st, err := xpath3.ParseSequenceType(fn.As); err == nil {
			fn.returnType = &st
		}
	}

	// A body made only of select-form xsl:sequence instructions never adds a
	// node to the call's output wrapper: execXSLSequence captures each item
	// into the frame's pendingItems because the frame captures items and its
	// insertion point is the wrapper itself. Every other instruction (an LRE,
	// xsl:element, a contained-constructor xsl:sequence, xsl:on-empty,
	// xsl:try, ...) may build nodes under the wrapper. An empty body writes
	// nothing either.
	fn.selectOnly = true
	for _, inst := range fn.Body {
		if _, ok := inst.(*xslSequenceInst); !ok {
			fn.selectOnly = false
			break
		}
	}
}

// functionOutputRoot returns the document and wrapper element a call of def
// writes its body's output into.
//
// A select-only body (see prepareCall) never appends to the wrapper, so every
// such call on ec shares one scratch wrapper, nested and recursive calls
// included: each call keeps its items in its own outputFrame, and the wrapper
// stays childless. Any other body gets a fresh document. That document is not
// freed after the call: the nodes the body builds are allocated from it and
// are returned to the caller, and ec keeps per-node state (nsFixupAllowed,
// typeAnnotations, ...) keyed by node identity, which a recycled slab would
// alias.
func (ec *execContext) functionOutputRoot(def *xslFunction) (*helium.Document, *helium.Element, error) {
	if def.selectOnly && ec.fnScratchRoot != nil {
		return ec.fnScratchDoc, ec.fnScratchRoot, nil
	}
	doc := helium.NewDefaultDocument()
	root, err := doc.CreateElement("_xsl_fn_result")
	if err != nil {
		return nil, nil, err
	}
	_ = doc.SetDocumentElement(root)
	if def.selectOnly {
		ec.fnScratchDoc = doc
		ec.fnScratchRoot = root
	}
	return doc, root, nil
}

func (f *xslUserFunc) Call(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	// Retrieve the XSLT exec context from the context.Context
	ec := f.ec
	if ecFromCtx := getExecContext(ctx); ecFromCtx != nil {
		ec = ecFromCtx
	}

	cacheKey, cacheable := ec.functionCacheKey(f.def, args)
	if cacheable {
		if result, ok := ec.functionResultCache[cacheKey]; ok {
			return cloneXPathSequence(result), nil
		}
	}

	// Recursion depth check
	ec.depth++
	if ec.depth > maxRecursionDepth {
		ec.depth--
		return nil, dynamicError(errCodeXTDE0820, "recursion depth exceeded in xsl:function %s", f.def.Name.Name)
	}
	defer func() { ec.depth-- }()

	// If the function belongs to a package, switch function scope.
	// Override functions always run in the main stylesheet context
	// (currentPackage=nil) because their body is defined in the using
	// stylesheet and may reference functions from that scope.
	savedFnsNS := ec.cachedFnsNS
	savedPackage := ec.currentPackage
	if f.def.IsOverride && ec.currentPackage != nil {
		ec.cachedFnsNS = nil
		ec.currentPackage = nil
	} else if f.def.OwnerPackage != nil && f.def.OwnerPackage != ec.currentPackage {
		ec.cachedFnsNS = nil
		ec.currentPackage = f.def.OwnerPackage
	}

	// Save and restore execution state.
	// xsl:function creates a new scope — tunnel params and current mode
	// are NOT inherited (XSLT 2.0 erratum XT.E19).
	savedContext := ec.contextNode
	savedCurrent := ec.currentNode
	savedPos := ec.position
	savedSize := ec.size
	savedTunnel := ec.tunnelParams
	savedMode := ec.currentMode
	savedGroups := ec.regexGroups
	savedInMerge := ec.inMergeAction
	ec.contextNode = nil
	ec.currentNode = nil
	ec.tunnelParams = nil
	ec.currentMode = ec.stylesheet.defaultMode
	ec.regexGroups = nil     // regex-group() returns empty inside xsl:function
	ec.inMergeAction = false // XTDE3480/XTDE3510: merge context not available in functions
	defer func() {
		ec.contextNode = savedContext
		ec.currentNode = savedCurrent
		ec.position = savedPos
		ec.size = savedSize
		ec.tunnelParams = savedTunnel
		ec.currentMode = savedMode
		ec.regexGroups = savedGroups
		ec.inMergeAction = savedInMerge
		ec.cachedFnsNS = savedFnsNS
		ec.currentPackage = savedPackage
	}()

	// Track the original function for xsl:original() support.
	// Stored on execContext so FunctionResolver can find it without
	// polluting fnsNS (which would make it visible to function-lookup).
	savedOriginalFunc := ec.originalFunc
	if f.def.OriginalFunc != nil {
		ec.originalFunc = &xslUserFunc{def: f.def.OriginalFunc, ec: ec}
	}
	defer func() { ec.originalFunc = savedOriginalFunc }()

	// Push new variable scope for parameters
	ec.pushVarScope()
	defer ec.popVarScope()

	// Bind parameters with type checking/coercion (XTTE0790)
	for i, param := range f.def.Params {
		if i < len(args) {
			val := args[i]
			if param.As != "" {
				st := f.def.paramCheckTypes[i]
				checked, err := checkSequenceType(ctx, val, st, errCodeXTTE0790, "param $"+param.Name, ec)
				if err != nil {
					return nil, err
				}
				val = checked
			}
			ec.setVar(param.Name, val)
		} else if param.Select != nil {
			result, err := ec.evalXPath(ctx, param.Select, ec.contextNode)
			if err != nil {
				return nil, err
			}
			ec.setVar(param.Name, result.Sequence())
		} else {
			ec.setVar(param.Name, xpath3.EmptySequence())
		}
	}

	// Execute the function body, collecting result into a temporary document.
	// For functions returning atomic types, use captureItems mode so that
	// attribute nodes returned by xsl:sequence are preserved directly
	// (writing them to a DOM tree loses them as attributes of the wrapper).
	tmpDoc, tmpRoot, err := ec.functionOutputRoot(f.def)
	if err != nil {
		return nil, err
	}

	atomicReturn := f.def.As != "" && isAtomicTypeName(f.def.As)
	frame := &outputFrame{current: tmpRoot, doc: tmpDoc, captureItems: true, sequenceMode: true}
	ec.outputStack = append(ec.outputStack, frame)
	ec.temporaryOutputDepth++
	defer func() {
		ec.temporaryOutputDepth--
		ec.outputStack = ec.outputStack[:len(ec.outputStack)-1]
	}()

	for _, inst := range f.def.Body {
		if err := ec.executeInstruction(ctx, inst); err != nil {
			return nil, err
		}
	}
	if f.def.selectOnly && tmpRoot.FirstChild() != nil {
		// prepareCall rules this out. Should a node ever land on the shared
		// wrapper, detach it, so no other call sharing the wrapper sees it,
		// and return it ahead of the captured items.
		frame.pendingItems = append(ec.collectNodeChildren(tmpRoot), frame.pendingItems...)
	}

	// Return captured items if any, otherwise collect from DOM.
	// For atomic return types, atomize the captured items.
	// Per XSLT 3.0 §5.7.2, adjacent text nodes in a sequence are NOT
	// merged (merging only applies to tree construction). Zero-length
	// text nodes are preserved as distinct items.
	var result xpath3.ItemSlice
	if len(frame.pendingItems) > 0 {
		if tmpRoot.FirstChild() != nil {
			var seq xpath3.ItemSlice
			for child := range helium.Children(tmpRoot) {
				seq = append(seq, xpath3.NodeItem{Node: child})
			}
			seq = append(seq, frame.pendingItems...)
			if atomicReturn {
				result = xpath3.ItemSlice(sequence.Materialize(atomizeSequence(seq)))
			} else {
				result = xpath3.ItemSlice(sequence.Materialize(seq))
			}
		} else if atomicReturn {
			result = xpath3.ItemSlice(sequence.Materialize(atomizeSequence(frame.pendingItems)))
		} else {
			result = xpath3.ItemSlice(sequence.Materialize(frame.pendingItems))
		}
	} else {
		result = ec.collectNodeChildren(tmpRoot)
		if atomicReturn {
			result = xpath3.ItemSlice(sequence.Materialize(atomizeSequence(result)))
		}
	}

	// Strip DOE marker PIs — DOE is ignored in function return values
	// (temporary output state per XSLT 3.0 §20.1).
	stripped, _ := stripDOEMarkers(result)
	result = xpath3.ItemSlice(sequence.Materialize(stripped))

	// Type check against the declared as type
	if f.def.As != "" {
		checked, err := checkSequenceType(ctx, result, f.def.returnCheckType, errCodeXTTE0780, "function "+f.def.Name.Name, ec)
		if err != nil {
			return nil, err
		}
		result = xpath3.ItemSlice(sequence.Materialize(checked))
	}

	if cacheable {
		ec.functionResultCache[cacheKey] = cloneXPathSequence(result)
	}

	return result, nil
}

func cloneXPathSequence(seq xpath3.Sequence) xpath3.Sequence {
	if seq == nil {
		return nil
	}
	return append(xpath3.ItemSlice(nil), sequence.Materialize(seq)...)
}

func (ec *execContext) functionCacheKey(def *xslFunction, args []xpath3.Sequence) (string, bool) {
	if ec == nil || def == nil {
		return "", false
	}
	// Cache when cache="yes" or new-each-time="no" (deterministic function).
	if !def.Cache && def.NewEachTime != lexicon.ValueNo {
		return "", false
	}

	var b strings.Builder
	if def.OwnerPackage != nil {
		fmt.Fprintf(&b, "pkg:%p|", def.OwnerPackage)
	}
	b.WriteString(def.Name.URI)
	b.WriteByte('|')
	b.WriteString(def.Name.Name)
	b.WriteByte('#')
	b.WriteString(strconv.Itoa(len(args)))
	for _, arg := range args {
		b.WriteByte('|')
		if !ec.writeFunctionCacheSequence(&b, arg) {
			return "", false
		}
	}
	return b.String(), true
}

func (ec *execContext) writeFunctionCacheSequence(b *strings.Builder, seq xpath3.Sequence) bool {
	b.WriteByte('[')
	b.WriteString(strconv.Itoa(sequence.Len(seq)))
	for item := range sequence.Items(seq) {
		b.WriteByte(';')
		if !ec.writeFunctionCacheItem(b, item) {
			return false
		}
	}
	b.WriteByte(']')
	return true
}

func (ec *execContext) writeFunctionCacheItem(b *strings.Builder, item xpath3.Item) bool {
	switch v := item.(type) {
	case xpath3.AtomicValue:
		s, err := xpath3.AtomicToString(v)
		if err != nil {
			return false
		}
		b.WriteString("a:")
		b.WriteString(v.TypeName)
		b.WriteByte('=')
		// AtomicToString for QName returns prefix:local, which omits the
		// namespace URI. Two QNames can share a local name but differ in URI,
		// so we must include the URI in the cache key.
		if q, ok := v.Value.(xpath3.QNameValue); ok {
			b.WriteString("{")
			b.WriteString(q.URI)
			b.WriteString("}")
			b.WriteString(q.Local)
		} else {
			b.WriteString(s)
		}
		return true
	case xpath3.NodeItem:
		b.WriteString("n:")
		b.WriteString(strconv.FormatUint(ec.memoNodeID(v.Node), 10))
		if v.TypeAnnotation != "" {
			b.WriteByte('@')
			b.WriteString(v.TypeAnnotation)
		}
		if v.AtomizedType != "" {
			b.WriteByte('!')
			b.WriteString(v.AtomizedType)
		}
		return true
	case xpath3.MapItem:
		b.WriteString("m{")
		for _, key := range v.Keys() {
			if !ec.writeFunctionCacheItem(b, key) {
				return false
			}
			b.WriteByte(':')
			val, _ := v.Get(key)
			if !ec.writeFunctionCacheSequence(b, val) {
				return false
			}
			b.WriteByte(',')
		}
		b.WriteByte('}')
		return true
	case xpath3.ArrayItem:
		b.WriteString("r[")
		for _, member := range v.Members() {
			if !ec.writeFunctionCacheSequence(b, member) {
				return false
			}
			b.WriteByte(',')
		}
		b.WriteByte(']')
		return true
	case xpath3.FunctionItem:
		return false
	default:
		return false
	}
}

func (ec *execContext) memoNodeID(node helium.Node) uint64 {
	if node == nil {
		return 0
	}
	if ec.nodeMemoIDs == nil {
		ec.nodeMemoIDs = make(map[helium.Node]uint64)
	}
	if id, ok := ec.nodeMemoIDs[node]; ok {
		return id
	}
	ec.nextNodeMemoID++
	ec.nodeMemoIDs[node] = ec.nextNodeMemoID
	return ec.nextNodeMemoID
}

// collectNodeChildren returns all children of a node as NodeItem values.
func (ec *execContext) collectNodeChildren(node helium.Node) xpath3.ItemSlice {
	var seq xpath3.ItemSlice
	var children []helium.Node
	for child := range helium.Children(node) {
		children = append(children, child)
	}
	for _, child := range children {
		helium.UnlinkNode(child.(helium.MutableNode)) //nolint:forcetypeassert
		seq = append(seq, xpath3.NodeItem{Node: child})
	}
	return seq
}

// isAtomicTypeName returns true if the given type name (from an as="" attribute)
// represents an atomic/simple type (not a node type or function type).
// Handles occurrence indicators (?, *, +) and xs: prefixed types.
func isAtomicTypeName(as string) bool {
	// Strip occurrence indicator
	name := strings.TrimRight(as, "?*+")
	name = strings.TrimSpace(name)
	// xs:string, xs:integer, xs:boolean, xs:double, etc.
	if strings.HasPrefix(name, "xs:") {
		return true
	}
	// unprefixed atomic types (rare but possible)
	switch name {
	case lexicon.TypeString, lexicon.TypeInteger, lexicon.TypeBoolean, lexicon.TypeDouble, lexicon.TypeFloat, "decimal",
		lexicon.TypeDate, "dateTime", lexicon.TypeTime, lexicon.TypeDuration, lexicon.TypeAnyURI:
		return true
	}
	return false
}

// element-available(name) returns true if the named XSLT element is available.
