package xpath1

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

// sentinel errors for XPath built-in function argument validation.
var (
	errLastNoArgs          = errors.New("last() takes no arguments")
	errPositionNoArgs      = errors.New("position() takes no arguments")
	errCountOneArg         = errors.New("count() takes exactly 1 argument")
	errCountNodeSet        = errors.New("count() argument must be a node-set")
	errIDOneArg            = errors.New("id() takes exactly 1 argument")
	errExpected01Args      = errors.New("expected 0 or 1 arguments")
	errArgMustBeNodeSet    = errors.New("argument must be a node-set")
	errStringArgs          = errors.New("string() takes 0 or 1 arguments")
	errConcatArgs          = errors.New("concat() requires at least 2 arguments")
	errStartsWithArgs      = errors.New("starts-with() takes exactly 2 arguments")
	errContainsArgs        = errors.New("contains() takes exactly 2 arguments")
	errSubstringBeforeArgs = errors.New("substring-before() takes exactly 2 arguments")
	errSubstringAfterArgs  = errors.New("substring-after() takes exactly 2 arguments")
	errSubstringArgs       = errors.New("substring() takes 2 or 3 arguments")
	errStringLengthArgs    = errors.New("string-length() takes 0 or 1 arguments")
	errNormalizeSpaceArgs  = errors.New("normalize-space() takes 0 or 1 arguments")
	errTranslateArgs       = errors.New("translate() takes exactly 3 arguments")
	errBooleanOneArg       = errors.New("boolean() takes exactly 1 argument")
	errNotOneArg           = errors.New("not() takes exactly 1 argument")
	errTrueNoArgs          = errors.New("true() takes no arguments")
	errFalseNoArgs         = errors.New("false() takes no arguments")
	errLangOneArg          = errors.New("lang() takes exactly 1 argument")
	errNumberArgs          = errors.New("number() takes 0 or 1 arguments")
	errSumOneArg           = errors.New("sum() takes exactly 1 argument")
	errSumNodeSet          = errors.New("sum() argument must be a node-set")
	errFloorOneArg         = errors.New("floor() takes exactly 1 argument")
	errCeilingOneArg       = errors.New("ceiling() takes exactly 1 argument")
	errRoundOneArg         = errors.New("round() takes exactly 1 argument")
)

type builtinFunction struct {
	callback func(ctx *evalContext, args []*Result) (*Result, error)
}

func (f builtinFunction) Eval(ctx context.Context, args []*Result) (*Result, error) {
	fctx := GetFunctionContext(ctx)
	c, ok := fctx.(*evalContext)
	if !ok || c == nil {
		return nil, fmt.Errorf("%w: %T", ErrInvalidFunctionContext, fctx)
	}
	return f.callback(c, args)
}

var builtinFunctions map[string]Function

func init() {
	builtinFunctions = map[string]Function{
		// Node-set functions
		"last":          builtinFunction{callback: fnLast},
		"position":      builtinFunction{callback: fnPosition},
		"count":         builtinFunction{callback: fnCount},
		"id":            builtinFunction{callback: fnID},
		"local-name":    builtinFunction{callback: fnLocalName},
		"namespace-uri": builtinFunction{callback: fnNamespaceURI},
		"name":          builtinFunction{callback: fnName},

		// String functions
		"string":           builtinFunction{callback: fnString},
		"concat":           builtinFunction{callback: fnConcat},
		"starts-with":      builtinFunction{callback: fnStartsWith},
		"contains":         builtinFunction{callback: fnContains},
		"substring-before": builtinFunction{callback: fnSubstringBefore},
		"substring-after":  builtinFunction{callback: fnSubstringAfter},
		"substring":        builtinFunction{callback: fnSubstring},
		"string-length":    builtinFunction{callback: fnStringLength},
		"normalize-space":  builtinFunction{callback: fnNormalizeSpace},
		"translate":        builtinFunction{callback: fnTranslate},

		// Boolean functions
		"boolean": builtinFunction{callback: fnBoolean},
		"not":     builtinFunction{callback: fnNot},
		"true":    builtinFunction{callback: fnTrue},
		"false":   builtinFunction{callback: fnFalse},
		"lang":    builtinFunction{callback: fnLang},

		// Number functions
		"number":  builtinFunction{callback: fnNumber},
		"sum":     builtinFunction{callback: fnSum},
		"floor":   builtinFunction{callback: fnFloor},
		"ceiling": builtinFunction{callback: fnCeiling},
		"round":   builtinFunction{callback: fnRound},
	}
}

func evalFunctionCall(goCtx context.Context, ctx *evalContext, fc FunctionCall) (*Result, error) {
	if err := ctx.countOps(1); err != nil {
		return nil, err
	}

	// Namespaced function call: resolve prefix to URI and look up in functionsNS.
	if fc.Prefix != "" {
		return evalNamespacedFunctionCall(goCtx, ctx, fc)
	}

	// Unqualified: built-ins take priority, then user-registered functions.
	var fn Function
	if f, ok := builtinFunctions[fc.Name]; ok {
		fn = f
	} else if ctx.functions != nil {
		fn = ctx.functions[fc.Name]
	}
	if fn == nil {
		return nil, fmt.Errorf("%w: %s", ErrUnknownFunction, fc.Name)
	}

	// Pre-evaluate all arguments.
	args := make([]*Result, len(fc.Args))
	for i, expr := range fc.Args {
		r, err := eval(goCtx, ctx, expr)
		if err != nil {
			return nil, err
		}
		args[i] = r
	}

	// Stash the evalContext as FunctionContext in the context.Context
	// so functions can retrieve it via GetFunctionContext.
	fctx := withFunctionContext(goCtx, ctx)
	return fn.Eval(fctx, args) //nolint:wrapcheck
}

func evalNamespacedFunctionCall(goCtx context.Context, ctx *evalContext, fc FunctionCall) (*Result, error) {
	if ctx.namespaces == nil {
		return nil, fmt.Errorf("%w: %s", ErrUnknownFunctionNamespace, fc.Prefix)
	}
	uri, ok := ctx.namespaces[fc.Prefix]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownFunctionNamespace, fc.Prefix)
	}
	var fn Function
	if ctx.functionsNS != nil {
		fn = ctx.functionsNS[QualifiedName{URI: uri, Name: fc.Name}]
	}
	if fn == nil {
		return nil, fmt.Errorf("%w: {%s}%s", ErrUnknownFunction, uri, fc.Name)
	}

	// Pre-evaluate all arguments.
	args := make([]*Result, len(fc.Args))
	for i, expr := range fc.Args {
		r, err := eval(goCtx, ctx, expr)
		if err != nil {
			return nil, err
		}
		args[i] = r
	}

	// Stash the evalContext as FunctionContext in the context.Context
	// so functions can retrieve it via GetFunctionContext.
	fctx := withFunctionContext(goCtx, ctx)
	return fn.Eval(fctx, args) //nolint:wrapcheck
}

// --- Node-set functions ---

func fnLast(ctx *evalContext, args []*Result) (*Result, error) {
	if len(args) != 0 {
		return nil, errLastNoArgs
	}
	return &Result{Type: NumberResult, Number: float64(ctx.size)}, nil
}

func fnPosition(ctx *evalContext, args []*Result) (*Result, error) {
	if len(args) != 0 {
		return nil, errPositionNoArgs
	}
	return &Result{Type: NumberResult, Number: float64(ctx.position)}, nil
}

func fnCount(_ *evalContext, args []*Result) (*Result, error) {
	if len(args) != 1 {
		return nil, errCountOneArg
	}
	if args[0].Type != NodeSetResult {
		return nil, errCountNodeSet
	}
	return &Result{Type: NumberResult, Number: float64(len(args[0].NodeSet))}, nil
}

func fnID(ctx *evalContext, args []*Result) (*Result, error) {
	if len(args) != 1 {
		return nil, errIDOneArg
	}

	idValues := collectIDValues(args[0])
	if len(idValues) == 0 {
		return &Result{Type: NodeSetResult}, nil
	}

	// Get document
	root := documentRoot(ctx.node)
	doc, ok := root.(*helium.Document)
	if !ok {
		return &Result{Type: NodeSetResult}, nil
	}

	// Use Document.GetElementByID which supports both xml:id and
	// DTD-declared ID attributes, matching libxml2's xmlGetID behavior.
	seen := make(map[*helium.Element]struct{}, len(idValues))
	var result []helium.Node
	for _, id := range idValues {
		elem := doc.GetElementByID(id)
		if elem == nil {
			continue
		}
		if _, dup := seen[elem]; dup {
			continue
		}
		seen[elem] = struct{}{}
		result = append(result, elem)
	}

	return &Result{Type: NodeSetResult, NodeSet: result}, nil
}

// collectIDValues extracts the whitespace-separated ID values from an XPath result.
func collectIDValues(r *Result) []string {
	if r.Type == NodeSetResult {
		vals := make([]string, 0, len(r.NodeSet))
		for _, n := range r.NodeSet {
			vals = append(vals, strings.Fields(stringValue(n))...)
		}
		return vals
	}
	return strings.Fields(resultToString(r))
}

func fnLocalName(ctx *evalContext, args []*Result) (*Result, error) {
	n, ok, err := nodeArgOrContext(ctx, args)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &Result{Type: StringResult}, nil
	}
	return &Result{Type: StringResult, String: localNameOf(n)}, nil
}

func fnNamespaceURI(ctx *evalContext, args []*Result) (*Result, error) {
	n, ok, err := nodeArgOrContext(ctx, args)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &Result{Type: StringResult}, nil
	}
	return &Result{Type: StringResult, String: nodeNamespaceURI(n)}, nil
}

func fnName(ctx *evalContext, args []*Result) (*Result, error) {
	n, ok, err := nodeArgOrContext(ctx, args)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &Result{Type: StringResult}, nil
	}
	return &Result{Type: StringResult, String: nameOf(n)}, nil
}

// nameOf returns the XPath name() of a node. Per the XPath spec, name()
// returns the QName for elements/attributes, the target for PIs, the prefix
// for namespace nodes, and empty string for all other node types.
func nameOf(n helium.Node) string {
	switch n.Type() {
	case helium.ElementNode, helium.AttributeNode,
		helium.ProcessingInstructionNode, helium.NamespaceNode:
		return n.Name()
	default:
		return ""
	}
}

// nodeArgOrContext returns the first node from an optional node-set argument,
// or the context node if no argument is provided.
// The second return value reports whether a node was found.
func nodeArgOrContext(ctx *evalContext, args []*Result) (helium.Node, bool, error) {
	if len(args) == 0 {
		return ctx.node, true, nil
	}
	if len(args) != 1 {
		return nil, false, errExpected01Args
	}
	if args[0].Type != NodeSetResult {
		return nil, false, errArgMustBeNodeSet
	}
	if len(args[0].NodeSet) == 0 {
		return nil, false, nil
	}
	return args[0].NodeSet[0], true, nil
}

// --- String functions ---

func fnString(ctx *evalContext, args []*Result) (*Result, error) {
	if len(args) == 0 {
		s := stringValue(ctx.node)
		return &Result{Type: StringResult, String: s}, nil
	}
	if len(args) != 1 {
		return nil, errStringArgs
	}
	return &Result{Type: StringResult, String: resultToString(args[0])}, nil
}

func fnConcat(_ *evalContext, args []*Result) (*Result, error) {
	if len(args) < 2 {
		return nil, errConcatArgs
	}
	var b strings.Builder
	for _, r := range args {
		b.WriteString(resultToString(r))
	}
	return &Result{Type: StringResult, String: b.String()}, nil
}

func fnStartsWith(_ *evalContext, args []*Result) (*Result, error) {
	if len(args) != 2 {
		return nil, errStartsWithArgs
	}
	s := resultToString(args[0])
	prefix := resultToString(args[1])
	return &Result{Type: BooleanResult, Bool: strings.HasPrefix(s, prefix)}, nil
}

func fnContains(_ *evalContext, args []*Result) (*Result, error) {
	if len(args) != 2 {
		return nil, errContainsArgs
	}
	s := resultToString(args[0])
	sub := resultToString(args[1])
	return &Result{Type: BooleanResult, Bool: strings.Contains(s, sub)}, nil
}

func fnSubstringBefore(_ *evalContext, args []*Result) (*Result, error) {
	if len(args) != 2 {
		return nil, errSubstringBeforeArgs
	}
	s := resultToString(args[0])
	sep := resultToString(args[1])
	idx := strings.Index(s, sep)
	if idx < 0 {
		return &Result{Type: StringResult}, nil
	}
	return &Result{Type: StringResult, String: s[:idx]}, nil
}

func fnSubstringAfter(_ *evalContext, args []*Result) (*Result, error) {
	if len(args) != 2 {
		return nil, errSubstringAfterArgs
	}
	s := resultToString(args[0])
	sep := resultToString(args[1])
	idx := strings.Index(s, sep)
	if idx < 0 {
		return &Result{Type: StringResult}, nil
	}
	return &Result{Type: StringResult, String: s[idx+len(sep):]}, nil
}

func fnSubstring(_ *evalContext, args []*Result) (*Result, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, errSubstringArgs
	}
	s := resultToString(args[0])
	startPos := resultToNumber(args[1])

	// XPath spec: character at position p is included iff
	//   p >= round(startPos) AND p < round(startPos) + round(length)
	// where positions are 1-based. round() independently on each arg.
	rStart := math.Floor(startPos + 0.5) // XPath round

	if len(args) == 3 {
		return fnSubstring3(args[2], s, rStart), nil
	}
	return fnSubstring2(s, rStart), nil
}

// fnSubstring3 handles the 3-argument form of substring().
func fnSubstring3(lengthArg *Result, s string, rStart float64) *Result {
	length := resultToNumber(lengthArg)
	rLength := math.Floor(length + 0.5)
	var b strings.Builder
	for i, r := range []rune(s) {
		p := float64(i + 1) // 1-based position
		if p >= rStart && p < rStart+rLength {
			b.WriteRune(r)
		}
	}
	return &Result{Type: StringResult, String: b.String()}
}

// fnSubstring2 handles the 2-argument form of substring().
func fnSubstring2(s string, rStart float64) *Result {
	if math.IsNaN(rStart) || math.IsInf(rStart, 1) {
		return &Result{Type: StringResult}
	}
	var b strings.Builder
	for i, r := range []rune(s) {
		if float64(i+1) >= rStart {
			b.WriteRune(r)
		}
	}
	return &Result{Type: StringResult, String: b.String()}
}

func fnStringLength(ctx *evalContext, args []*Result) (*Result, error) {
	var s string
	switch len(args) {
	case 0:
		s = stringValue(ctx.node)
	case 1:
		s = resultToString(args[0])
	default:
		return nil, errStringLengthArgs
	}
	return &Result{Type: NumberResult, Number: float64(len([]rune(s)))}, nil
}

func fnNormalizeSpace(ctx *evalContext, args []*Result) (*Result, error) {
	var s string
	switch len(args) {
	case 0:
		s = stringValue(ctx.node)
	case 1:
		s = resultToString(args[0])
	default:
		return nil, errNormalizeSpaceArgs
	}

	// Strip leading/trailing whitespace and collapse internal whitespace
	fields := strings.Fields(s)
	return &Result{Type: StringResult, String: strings.Join(fields, " ")}, nil
}

func fnTranslate(_ *evalContext, args []*Result) (*Result, error) {
	if len(args) != 3 {
		return nil, errTranslateArgs
	}
	s := resultToString(args[0])
	from := resultToString(args[1])
	to := resultToString(args[2])

	mapping, remove := buildTranslateMap([]rune(from), []rune(to))

	var b strings.Builder
	for _, r := range s {
		if remove[r] {
			continue
		}
		if rep, ok := mapping[r]; ok {
			b.WriteRune(rep)
		} else {
			b.WriteRune(r)
		}
	}
	return &Result{Type: StringResult, String: b.String()}, nil
}

// buildTranslateMap constructs the character mapping and removal set for translate().
// The first occurrence of each rune in fromRunes wins per XPath spec.
func buildTranslateMap(fromRunes, toRunes []rune) (mapping map[rune]rune, remove map[rune]bool) {
	mapping = make(map[rune]rune, len(fromRunes))
	remove = make(map[rune]bool)
	for i, r := range fromRunes {
		if _, exists := mapping[r]; exists {
			continue // first occurrence wins
		}
		if remove[r] {
			continue
		}
		if i < len(toRunes) {
			mapping[r] = toRunes[i]
		} else {
			remove[r] = true
		}
	}
	return mapping, remove
}

// --- Boolean functions ---

func fnBoolean(_ *evalContext, args []*Result) (*Result, error) {
	if len(args) != 1 {
		return nil, errBooleanOneArg
	}
	return &Result{Type: BooleanResult, Bool: resultToBoolean(args[0])}, nil
}

func fnNot(_ *evalContext, args []*Result) (*Result, error) {
	if len(args) != 1 {
		return nil, errNotOneArg
	}
	return &Result{Type: BooleanResult, Bool: !resultToBoolean(args[0])}, nil
}

func fnTrue(_ *evalContext, args []*Result) (*Result, error) {
	if len(args) != 0 {
		return nil, errTrueNoArgs
	}
	return &Result{Type: BooleanResult, Bool: true}, nil
}

func fnFalse(_ *evalContext, args []*Result) (*Result, error) {
	if len(args) != 0 {
		return nil, errFalseNoArgs
	}
	return &Result{Type: BooleanResult, Bool: false}, nil
}

func fnLang(ctx *evalContext, args []*Result) (*Result, error) {
	if len(args) != 1 {
		return nil, errLangOneArg
	}
	langArg := strings.ToLower(resultToString(args[0]))

	// Walk up the tree looking for xml:lang
	for n := ctx.node; n != nil; n = n.Parent() {
		elem, ok := n.(*helium.Element)
		if !ok {
			continue
		}
		for _, attr := range elem.Attributes() {
			if attr.LocalName() == "lang" && attr.URI() == lexicon.NamespaceXML {
				val := strings.ToLower(attr.Value())
				if val == langArg || strings.HasPrefix(val, langArg+"-") {
					return &Result{Type: BooleanResult, Bool: true}, nil
				}
				return &Result{Type: BooleanResult, Bool: false}, nil
			}
		}
	}
	return &Result{Type: BooleanResult, Bool: false}, nil
}

// --- Number functions ---

func fnNumber(ctx *evalContext, args []*Result) (*Result, error) {
	if len(args) == 0 {
		s := stringValue(ctx.node)
		return &Result{Type: NumberResult, Number: stringToNumber(s)}, nil
	}
	if len(args) != 1 {
		return nil, errNumberArgs
	}
	return &Result{Type: NumberResult, Number: resultToNumber(args[0])}, nil
}

func fnSum(_ *evalContext, args []*Result) (*Result, error) {
	if len(args) != 1 {
		return nil, errSumOneArg
	}
	if args[0].Type != NodeSetResult {
		return nil, errSumNodeSet
	}
	var total float64
	for _, n := range args[0].NodeSet {
		total += stringToNumber(stringValue(n))
	}
	return &Result{Type: NumberResult, Number: total}, nil
}

func fnFloor(_ *evalContext, args []*Result) (*Result, error) {
	if len(args) != 1 {
		return nil, errFloorOneArg
	}
	return &Result{Type: NumberResult, Number: math.Floor(resultToNumber(args[0]))}, nil
}

func fnCeiling(_ *evalContext, args []*Result) (*Result, error) {
	if len(args) != 1 {
		return nil, errCeilingOneArg
	}
	return &Result{Type: NumberResult, Number: math.Ceil(resultToNumber(args[0]))}, nil
}

func fnRound(_ *evalContext, args []*Result) (*Result, error) {
	if len(args) != 1 {
		return nil, errRoundOneArg
	}
	n := resultToNumber(args[0])
	// XPath round: round half towards positive infinity, preserve -0
	if math.IsNaN(n) || math.IsInf(n, 0) || n == 0 {
		return &Result{Type: NumberResult, Number: n}, nil
	}
	r := math.Floor(n + 0.5)
	// Preserve negative zero: values in (-0.5, 0) round to -0
	if r == 0 && n < 0 {
		r = math.Copysign(0, -1)
	}
	return &Result{Type: NumberResult, Number: r}, nil
}
