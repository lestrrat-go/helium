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
	def *XSLFunction
	ec  *execContext
}

func (f *xslUserFunc) MinArity() int { return len(f.def.Params) }
func (f *xslUserFunc) MaxArity() int { return len(f.def.Params) }

func (f *xslUserFunc) FuncParamTypes() []xpath3.SequenceType {
	pts := make([]xpath3.SequenceType, 0, len(f.def.Params))
	for _, p := range f.def.Params {
		as := p.As
		if as == "" {
			as = "item()*"
		}
		st, err := xpath3.ParseSequenceType(as)
		if err != nil {
			return nil
		}
		pts = append(pts, st)
	}
	return pts
}

func (f *xslUserFunc) FuncReturnType() *xpath3.SequenceType {
	as := f.def.As
	if as == "" {
		return nil
	}
	st, err := xpath3.ParseSequenceType(as)
	if err != nil {
		return nil
	}
	return &st
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
	savedFnsNS := ec.cachedFnsNS
	savedPackage := ec.currentPackage
	if f.def.OwnerPackage != nil && f.def.OwnerPackage != ec.currentPackage {
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
				st := parseSequenceType(param.As)
				checked, err := checkSequenceType(val, st, errCodeXTTE0790, "param $"+param.Name, ec)
				if err != nil {
					return nil, err
				}
				val = checked
			}
			ec.setVar(param.Name, val)
		} else if param.Select != nil {
			result, err := ec.evalXPath(param.Select, ec.contextNode)
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
	tmpDoc := helium.NewDefaultDocument()
	tmpRoot, _ := tmpDoc.CreateElement("_xsl_fn_result")
	_ = tmpDoc.SetDocumentElement(tmpRoot)

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

	// Return captured items if any, otherwise collect from DOM.
	// For atomic return types, atomize the captured items.
	// Adjacent text nodes are merged (XSLT spec: constructing a temporary
	// tree merges adjacent text nodes in the result).
	var result xpath3.ItemSlice
	if len(frame.pendingItems) > 0 {
		if tmpRoot.FirstChild() != nil {
			var seq xpath3.ItemSlice
			for child := range helium.Children(tmpRoot) {
				seq = append(seq, xpath3.NodeItem{Node: child})
			}
			seq = append(seq, frame.pendingItems...)
			merged := mergeAdjacentTextNodes(seq)
			if atomicReturn {
				result = xpath3.ItemSlice(sequence.Materialize(atomizeSequence(merged)))
			} else {
				result = xpath3.ItemSlice(sequence.Materialize(merged))
			}
		} else if atomicReturn {
			result = xpath3.ItemSlice(sequence.Materialize(atomizeSequence(mergeAdjacentTextNodes(frame.pendingItems))))
		} else {
			result = xpath3.ItemSlice(sequence.Materialize(mergeAdjacentTextNodes(frame.pendingItems)))
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
		st := parseSequenceType(f.def.As)
		checked, err := checkSequenceType(result, st, errCodeXTTE0780, "function "+f.def.Name.Name, ec)
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

func (ec *execContext) functionCacheKey(def *XSLFunction, args []xpath3.Sequence) (string, bool) {
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
		b.WriteString(s)
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
		helium.UnlinkNode(child)
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
	case "string", "integer", "boolean", "double", "float", "decimal",
		"date", "dateTime", "time", "duration", "anyURI":
		return true
	}
	return false
}

// element-available(name) returns true if the named XSLT element is available.
