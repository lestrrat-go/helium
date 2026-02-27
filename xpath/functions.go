package xpath

import (
	"fmt"
	"math"
	"strings"
	helium "github.com/lestrrat-go/helium"
)

type xpathFunc func(ctx *evalContext, args []Expr) (*Result, error)

var builtinFunctions map[string]xpathFunc

func init() {
	builtinFunctions = map[string]xpathFunc{
		// Node-set functions
		"last":             fnLast,
		"position":         fnPosition,
		"count":            fnCount,
		"id":               fnID,
		"local-name":       fnLocalName,
		"namespace-uri":    fnNamespaceURI,
		"name":             fnName,

		// String functions
		"string":           fnString,
		"concat":           fnConcat,
		"starts-with":      fnStartsWith,
		"contains":         fnContains,
		"substring-before": fnSubstringBefore,
		"substring-after":  fnSubstringAfter,
		"substring":        fnSubstring,
		"string-length":    fnStringLength,
		"normalize-space":  fnNormalizeSpace,
		"translate":        fnTranslate,

		// Boolean functions
		"boolean":          fnBoolean,
		"not":              fnNot,
		"true":             fnTrue,
		"false":            fnFalse,
		"lang":             fnLang,

		// Number functions
		"number":           fnNumber,
		"sum":              fnSum,
		"floor":            fnFloor,
		"ceiling":          fnCeiling,
		"round":            fnRound,
	}
}

func evalFunctionCall(ctx *evalContext, fc FunctionCall) (*Result, error) {
	if err := ctx.countOps(1); err != nil {
		return nil, err
	}
	fn, ok := builtinFunctions[fc.Name]
	if !ok {
		return nil, fmt.Errorf("unknown function: %s", fc.Name)
	}
	return fn(ctx, fc.Args)
}

// --- Node-set functions ---

func fnLast(ctx *evalContext, args []Expr) (*Result, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("last() takes no arguments")
	}
	return &Result{Type: NumberResult, Number: float64(ctx.size)}, nil
}

func fnPosition(ctx *evalContext, args []Expr) (*Result, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("position() takes no arguments")
	}
	return &Result{Type: NumberResult, Number: float64(ctx.position)}, nil
}

func fnCount(ctx *evalContext, args []Expr) (*Result, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("count() takes exactly 1 argument")
	}
	r, err := eval(ctx, args[0])
	if err != nil {
		return nil, err
	}
	if r.Type != NodeSetResult {
		return nil, fmt.Errorf("count() argument must be a node-set")
	}
	return &Result{Type: NumberResult, Number: float64(len(r.NodeSet))}, nil
}

func fnID(ctx *evalContext, args []Expr) (*Result, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("id() takes exactly 1 argument")
	}
	r, err := eval(ctx, args[0])
	if err != nil {
		return nil, err
	}

	idValues := collectIDValues(r)
	if len(idValues) == 0 {
		return &Result{Type: NodeSetResult}, nil
	}

	// Get document and DTD
	root := documentRoot(ctx.node)
	doc, ok := root.(*helium.Document)
	if !ok {
		return &Result{Type: NodeSetResult}, nil
	}
	dtd := doc.IntSubset()
	if dtd == nil {
		return &Result{Type: NodeSetResult}, nil
	}

	// Build set of target IDs for fast lookup
	targets := make(map[string]bool, len(idValues))
	for _, v := range idValues {
		targets[v] = true
	}

	// Walk the document tree looking for elements with ID attributes
	var result []helium.Node
	collectIDNodes(root, dtd, targets, &result)

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

// collectIDNodes recursively searches the subtree rooted at n for elements whose
// DTD-declared ID attribute matches one of the target values.
func collectIDNodes(n helium.Node, dtd *helium.DTD, targets map[string]bool, result *[]helium.Node) {
	if elem, ok := n.(*helium.Element); ok {
		checkElementIDMatch(elem, dtd, targets, result)
	}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		collectIDNodes(c, dtd, targets, result)
	}
}

// checkElementIDMatch checks whether elem has a DTD-declared ID attribute
// whose value is one of the target IDs.
func checkElementIDMatch(elem *helium.Element, dtd *helium.DTD, targets map[string]bool, result *[]helium.Node) {
	for _, adecl := range dtd.AttributesForElement(elem.LocalName()) {
		if adecl.AType() != helium.AttrID {
			continue
		}
		for _, attr := range elem.Attributes() {
			if attr.LocalName() == adecl.Name() {
				if targets[attr.Value()] {
					*result = append(*result, elem)
				}
				break
			}
		}
	}
}

func fnLocalName(ctx *evalContext, args []Expr) (*Result, error) {
	n, ok, err := nodeArgOrContext(ctx, args)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &Result{Type: StringResult}, nil
	}
	return &Result{Type: StringResult, String: localNameOf(n)}, nil
}

func fnNamespaceURI(ctx *evalContext, args []Expr) (*Result, error) {
	n, ok, err := nodeArgOrContext(ctx, args)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &Result{Type: StringResult}, nil
	}
	return &Result{Type: StringResult, String: nodeNamespaceURI(n)}, nil
}

func fnName(ctx *evalContext, args []Expr) (*Result, error) {
	n, ok, err := nodeArgOrContext(ctx, args)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &Result{Type: StringResult}, nil
	}
	return &Result{Type: StringResult, String: n.Name()}, nil
}

// nodeArgOrContext returns the first node from an optional node-set argument,
// or the context node if no argument is provided.
// The second return value reports whether a node was found.
func nodeArgOrContext(ctx *evalContext, args []Expr) (helium.Node, bool, error) {
	if len(args) == 0 {
		return ctx.node, true, nil
	}
	if len(args) != 1 {
		return nil, false, fmt.Errorf("expected 0 or 1 arguments")
	}
	r, err := eval(ctx, args[0])
	if err != nil {
		return nil, false, err
	}
	if r.Type != NodeSetResult {
		return nil, false, fmt.Errorf("argument must be a node-set")
	}
	if len(r.NodeSet) == 0 {
		return nil, false, nil
	}
	return r.NodeSet[0], true, nil
}

// --- String functions ---

func fnString(ctx *evalContext, args []Expr) (*Result, error) {
	if len(args) == 0 {
		s := stringValue(ctx.node)
		return &Result{Type: StringResult, String: s}, nil
	}
	if len(args) != 1 {
		return nil, fmt.Errorf("string() takes 0 or 1 arguments")
	}
	r, err := eval(ctx, args[0])
	if err != nil {
		return nil, err
	}
	return &Result{Type: StringResult, String: resultToString(r)}, nil
}

func fnConcat(ctx *evalContext, args []Expr) (*Result, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("concat() requires at least 2 arguments")
	}
	var b strings.Builder
	for _, arg := range args {
		r, err := eval(ctx, arg)
		if err != nil {
			return nil, err
		}
		b.WriteString(resultToString(r))
	}
	return &Result{Type: StringResult, String: b.String()}, nil
}

func fnStartsWith(ctx *evalContext, args []Expr) (*Result, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("starts-with() takes exactly 2 arguments")
	}
	s, err := evalToString(ctx, args[0])
	if err != nil {
		return nil, err
	}
	prefix, err := evalToString(ctx, args[1])
	if err != nil {
		return nil, err
	}
	return &Result{Type: BooleanResult, Boolean: strings.HasPrefix(s, prefix)}, nil
}

func fnContains(ctx *evalContext, args []Expr) (*Result, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("contains() takes exactly 2 arguments")
	}
	s, err := evalToString(ctx, args[0])
	if err != nil {
		return nil, err
	}
	sub, err := evalToString(ctx, args[1])
	if err != nil {
		return nil, err
	}
	return &Result{Type: BooleanResult, Boolean: strings.Contains(s, sub)}, nil
}

func fnSubstringBefore(ctx *evalContext, args []Expr) (*Result, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("substring-before() takes exactly 2 arguments")
	}
	s, err := evalToString(ctx, args[0])
	if err != nil {
		return nil, err
	}
	sep, err := evalToString(ctx, args[1])
	if err != nil {
		return nil, err
	}
	idx := strings.Index(s, sep)
	if idx < 0 {
		return &Result{Type: StringResult}, nil
	}
	return &Result{Type: StringResult, String: s[:idx]}, nil
}

func fnSubstringAfter(ctx *evalContext, args []Expr) (*Result, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("substring-after() takes exactly 2 arguments")
	}
	s, err := evalToString(ctx, args[0])
	if err != nil {
		return nil, err
	}
	sep, err := evalToString(ctx, args[1])
	if err != nil {
		return nil, err
	}
	idx := strings.Index(s, sep)
	if idx < 0 {
		return &Result{Type: StringResult}, nil
	}
	return &Result{Type: StringResult, String: s[idx+len(sep):]}, nil
}

func fnSubstring(ctx *evalContext, args []Expr) (*Result, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("substring() takes 2 or 3 arguments")
	}
	s, err := evalToString(ctx, args[0])
	if err != nil {
		return nil, err
	}
	startPos, err := evalToNumber(ctx, args[1])
	if err != nil {
		return nil, err
	}

	// XPath spec: character at position p is included iff
	//   p >= round(startPos) AND p < round(startPos) + round(length)
	// where positions are 1-based. round() independently on each arg.
	rStart := math.Floor(startPos + 0.5) // XPath round

	if len(args) == 3 {
		return fnSubstring3(ctx, args[2], s, rStart)
	}
	return fnSubstring2(s, rStart), nil
}

// fnSubstring3 handles the 3-argument form of substring().
func fnSubstring3(ctx *evalContext, lengthArg Expr, s string, rStart float64) (*Result, error) {
	length, err := evalToNumber(ctx, lengthArg)
	if err != nil {
		return nil, err
	}
	rLength := math.Floor(length + 0.5)
	var b strings.Builder
	for i, r := range []rune(s) {
		p := float64(i + 1) // 1-based position
		if p >= rStart && p < rStart+rLength {
			b.WriteRune(r)
		}
	}
	return &Result{Type: StringResult, String: b.String()}, nil
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

func fnStringLength(ctx *evalContext, args []Expr) (*Result, error) {
	var s string
	switch len(args) {
	case 0:
		s = stringValue(ctx.node)
	case 1:
		var err error
		s, err = evalToString(ctx, args[0])
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("string-length() takes 0 or 1 arguments")
	}
	return &Result{Type: NumberResult, Number: float64(len([]rune(s)))}, nil
}

func fnNormalizeSpace(ctx *evalContext, args []Expr) (*Result, error) {
	var s string
	switch len(args) {
	case 0:
		s = stringValue(ctx.node)
	case 1:
		var err error
		s, err = evalToString(ctx, args[0])
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("normalize-space() takes 0 or 1 arguments")
	}

	// Strip leading/trailing whitespace and collapse internal whitespace
	fields := strings.Fields(s)
	return &Result{Type: StringResult, String: strings.Join(fields, " ")}, nil
}

func fnTranslate(ctx *evalContext, args []Expr) (*Result, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("translate() takes exactly 3 arguments")
	}
	s, err := evalToString(ctx, args[0])
	if err != nil {
		return nil, err
	}
	from, err := evalToString(ctx, args[1])
	if err != nil {
		return nil, err
	}
	to, err := evalToString(ctx, args[2])
	if err != nil {
		return nil, err
	}

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

func fnBoolean(ctx *evalContext, args []Expr) (*Result, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("boolean() takes exactly 1 argument")
	}
	r, err := eval(ctx, args[0])
	if err != nil {
		return nil, err
	}
	return &Result{Type: BooleanResult, Boolean: resultToBoolean(r)}, nil
}

func fnNot(ctx *evalContext, args []Expr) (*Result, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("not() takes exactly 1 argument")
	}
	r, err := eval(ctx, args[0])
	if err != nil {
		return nil, err
	}
	return &Result{Type: BooleanResult, Boolean: !resultToBoolean(r)}, nil
}

func fnTrue(_ *evalContext, args []Expr) (*Result, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("true() takes no arguments")
	}
	return &Result{Type: BooleanResult, Boolean: true}, nil
}

func fnFalse(_ *evalContext, args []Expr) (*Result, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("false() takes no arguments")
	}
	return &Result{Type: BooleanResult, Boolean: false}, nil
}

func fnLang(ctx *evalContext, args []Expr) (*Result, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("lang() takes exactly 1 argument")
	}
	langArg, err := evalToString(ctx, args[0])
	if err != nil {
		return nil, err
	}
	langArg = strings.ToLower(langArg)

	// Walk up the tree looking for xml:lang
	for n := ctx.node; n != nil; n = n.Parent() {
		elem, ok := n.(*helium.Element)
		if !ok {
			continue
		}
		for _, attr := range elem.Attributes() {
			if attr.Name() == "xml:lang" || attr.LocalName() == "lang" {
				val := strings.ToLower(attr.Value())
				if val == langArg || strings.HasPrefix(val, langArg+"-") {
					return &Result{Type: BooleanResult, Boolean: true}, nil
				}
				return &Result{Type: BooleanResult, Boolean: false}, nil
			}
		}
	}
	return &Result{Type: BooleanResult, Boolean: false}, nil
}

// --- Number functions ---

func fnNumber(ctx *evalContext, args []Expr) (*Result, error) {
	if len(args) == 0 {
		s := stringValue(ctx.node)
		return &Result{Type: NumberResult, Number: stringToNumber(s)}, nil
	}
	if len(args) != 1 {
		return nil, fmt.Errorf("number() takes 0 or 1 arguments")
	}
	r, err := eval(ctx, args[0])
	if err != nil {
		return nil, err
	}
	return &Result{Type: NumberResult, Number: resultToNumber(r)}, nil
}

func fnSum(ctx *evalContext, args []Expr) (*Result, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("sum() takes exactly 1 argument")
	}
	r, err := eval(ctx, args[0])
	if err != nil {
		return nil, err
	}
	if r.Type != NodeSetResult {
		return nil, fmt.Errorf("sum() argument must be a node-set")
	}
	var total float64
	for _, n := range r.NodeSet {
		total += stringToNumber(stringValue(n))
	}
	return &Result{Type: NumberResult, Number: total}, nil
}

func fnFloor(ctx *evalContext, args []Expr) (*Result, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("floor() takes exactly 1 argument")
	}
	n, err := evalToNumber(ctx, args[0])
	if err != nil {
		return nil, err
	}
	return &Result{Type: NumberResult, Number: math.Floor(n)}, nil
}

func fnCeiling(ctx *evalContext, args []Expr) (*Result, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("ceiling() takes exactly 1 argument")
	}
	n, err := evalToNumber(ctx, args[0])
	if err != nil {
		return nil, err
	}
	return &Result{Type: NumberResult, Number: math.Ceil(n)}, nil
}

func fnRound(ctx *evalContext, args []Expr) (*Result, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("round() takes exactly 1 argument")
	}
	n, err := evalToNumber(ctx, args[0])
	if err != nil {
		return nil, err
	}
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

// --- Helpers ---

func evalToString(ctx *evalContext, expr Expr) (string, error) {
	r, err := eval(ctx, expr)
	if err != nil {
		return "", err
	}
	return resultToString(r), nil
}

func evalToNumber(ctx *evalContext, expr Expr) (float64, error) {
	r, err := eval(ctx, expr)
	if err != nil {
		return 0, err
	}
	return resultToNumber(r), nil
}

