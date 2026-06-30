package xpath3

import "reflect"

// catchErrorVarKeys are the implicit error variables the evaluator binds inside a
// try/catch catch clause (XPath/XQuery error namespace, conventional "err" prefix),
// keyed by the SAME lexical "err:local" form a reference site produces. They are
// bound for the scope of each catch body so a reference to one is not reported as a
// free variable. Kept in sync with eval_control.go buildCatchContext.
var catchErrorVarKeys = []string{
	"err:code",
	"err:description",
	"err:value",
	"err:module",
	"err:line-number",
	"err:column-number",
	"err:additional",
}

// TypeNameRef is an atomic-or-union type name referenced by an expression in a
// cast / castable / instance of / treat as construct. Prefix is empty for an
// unprefixed name.
type TypeNameRef struct {
	Prefix string
	Name   string
}

// StaticReferences summarizes the variable and type-name references in a compiled
// expression, for callers that must impose static-context restrictions beyond the
// namespace-prefix validation done by Validate. XSD 1.1 conditional type
// assignment is one such caller: an xs:alternative @test XPath has no in-scope
// variables and may reference only built-in (xs:) atomic types, so the schema
// compiler rejects any FreeVariable and any non-built-in TypeName.
type StaticReferences struct {
	// FreeVariables lists the variable references ("prefix:local" or, when
	// unprefixed, "local") that are NOT bound by an enclosing for/let/quantified
	// binding or inline-function parameter within the expression itself.
	FreeVariables []string
	// TypeNames lists the atomic-or-union type names named by cast / castable /
	// instance of / treat as, AND the type annotations of element()/attribute()/
	// document-node() kind tests, wherever they appear (including nested array/map/
	// function item types and path-step node tests).
	TypeNames []TypeNameRef
	// FunctionNames lists the callee QNames of every static function reference — a
	// FunctionCall (which includes user-defined-type CONSTRUCTOR calls like
	// t:smallInt(...)) or a NamedFunctionRef (f#arity). Each is a (Prefix, local-Name)
	// pair; an unprefixed name resolves to the default FUNCTION namespace (fn), not the
	// default element namespace. CTA uses this to reject any function outside the
	// standard function library / built-in constructor namespaces.
	FunctionNames []TypeNameRef
}

// StaticReferences walks the expression's syntax tree and reports its free
// variable references and atomic type-name references. It is side-effect free and
// intended for schema-compile-time analysis, not the evaluation hot path.
func (e *Expression) StaticReferences() StaticReferences {
	ast := e.astExpr()
	c := &staticRefCollector{bound: map[string]int{}}
	c.walk(ast)
	return StaticReferences{FreeVariables: c.freeVars, TypeNames: c.typeNames, FunctionNames: c.funcNames}
}

type staticRefCollector struct {
	// bound is a REFCOUNT of in-scope variable bindings keyed by lexical name, not a
	// flat present/absent set: an inner for/let/quantified/inline-function binding may
	// SHADOW an outer one of the same name, so binding increments and leaving scope
	// decrements (deleting at zero). Restoring the prior count on exit keeps the outer
	// binding visible to later references instead of unbinding it — a flat set with an
	// unconditional delete would wrongly report the still-bound outer variable as free.
	bound     map[string]int
	freeVars  []string
	typeNames []TypeNameRef
	funcNames []TypeNameRef
	seenFree  map[string]struct{}
}

// bindVar enters a variable binding for the duration of an enclosed scope.
func (c *staticRefCollector) bindVar(key string) {
	c.bound[key]++
}

// unbindVar leaves a variable binding, restoring any shadowed outer binding.
func (c *staticRefCollector) unbindVar(key string) {
	if c.bound[key] <= 1 {
		delete(c.bound, key)
		return
	}
	c.bound[key]--
}

// variableKey returns the lexical key under which a variable is tracked for
// bound/free analysis. VariableExpr.Name already carries the full lexical form
// ("prefix:local" when prefixed, or "local" / "Q{uri}local" otherwise), exactly
// as the for/let/quantified/inline-function binding sites store their bound
// variable (the raw variable token value), so the key is simply that name — the
// prefix must NOT be prepended again, which would mis-key "p:x" as "p:p:x" and
// wrongly report a bound prefixed variable as free.
func variableKey(name string) string {
	return name
}

func (c *staticRefCollector) addFreeVar(key string) {
	if c.bound[key] > 0 {
		return
	}
	if c.seenFree == nil {
		c.seenFree = map[string]struct{}{}
	}
	if _, ok := c.seenFree[key]; ok {
		return
	}
	c.seenFree[key] = struct{}{}
	c.freeVars = append(c.freeVars, key)
}

func (c *staticRefCollector) addAtomicType(prefix, name string) {
	if name == "" {
		return
	}
	c.typeNames = append(c.typeNames, TypeNameRef{Prefix: prefix, Name: name})
}

// addFunctionName records a static function-reference callee QName. Prefix is
// empty for an unprefixed name (default function namespace).
func (c *staticRefCollector) addFunctionName(prefix, name string) {
	if name == "" {
		return
	}
	c.funcNames = append(c.funcNames, TypeNameRef{Prefix: prefix, Name: name})
}

// addTypeNameLexical records a type name held as a raw lexical "prefix:local"
// string (as element()/attribute() kind tests store their type annotation),
// splitting it the same way the parser splits AtomicOrUnionType/AtomicTypeName.
func (c *staticRefCollector) addTypeNameLexical(lexical string) {
	if lexical == "" {
		return
	}
	prefix, local := splitQName(lexical)
	c.addAtomicType(prefix, local)
}

// walkSequenceType collects every atomic-or-union type name named anywhere in a
// sequence type, INCLUDING the user type names nested inside array(T), map(K,V),
// and function(P...) as R item types. A purely top-level scan would miss a
// forbidden user-defined type hidden inside such a parameterized item type (e.g.
// "instance of array(t:T)"), so the walk descends recursively into the item test.
func (c *staticRefCollector) walkSequenceType(st SequenceType) {
	c.walkItemTest(st.ItemTest)
}

// walkItemTest collects atomic-or-union type names reachable from an item test,
// recursing through the parameterized array/map/function item types and the
// type-annotated kind tests (element()/attribute()/document-node()).
func (c *staticRefCollector) walkItemTest(it NodeTest) {
	it, ok := derefNodeTest(it)
	if !ok {
		return
	}
	switch t := it.(type) {
	case AtomicOrUnionType:
		c.addAtomicType(t.Prefix, t.Name)
	case ElementTest:
		// element(name?, type?) — the optional second arg is a user/built-in type
		// annotation. The name arg is an element NAME test, not a type, so it is not
		// recorded. TypeName is the raw lexical "prefix:local" (parser scanQName).
		c.addTypeNameLexical(t.TypeName)
	case AttributeTest:
		c.addTypeNameLexical(t.TypeName)
	case DocumentTest:
		c.walkItemTest(t.Inner)
	case ArrayTest:
		if !t.AnyType {
			c.walkSequenceType(t.MemberType)
		}
	case MapTest:
		if !t.AnyType {
			c.walkItemTest(t.KeyType)
			c.walkSequenceType(t.ValType)
		}
	case FunctionTest:
		if !t.AnyFunction {
			for _, pt := range t.ParamTypes {
				c.walkSequenceType(pt)
			}
			c.walkSequenceType(t.ReturnType)
		}
	}
}

// derefExprNode normalizes any pointer-form Expr to its value form via reflection.
// CompileExpr stores caller-built ASTs unchanged and the VM lowerer accepts both
// value and pointer node forms (e.g. &CastExpr{...}), so StaticReferences must too.
// (The package's other derefExpr in streamability.go only covers a fixed handful of
// pointer types, so it is not reused here.) ok is false for a nil node (including a
// typed nil pointer).
func derefExprNode(node Expr) (Expr, bool) {
	rv := reflect.ValueOf(node)
	if rv.Kind() != reflect.Pointer {
		return node, node != nil
	}
	if rv.IsNil() {
		return nil, false
	}
	if e, ok := rv.Elem().Interface().(Expr); ok {
		return e, true
	}
	return node, true
}

// derefNodeTest is the NodeTest analogue of derefExpr, normalizing a pointer-form
// item test (e.g. &ArrayTest{...}) to its value form.
func derefNodeTest(it NodeTest) (NodeTest, bool) {
	rv := reflect.ValueOf(it)
	if rv.Kind() != reflect.Pointer {
		return it, it != nil
	}
	if rv.IsNil() {
		return nil, false
	}
	if nt, ok := rv.Elem().Interface().(NodeTest); ok {
		return nt, true
	}
	return it, true
}

func (c *staticRefCollector) walk(node Expr) { //nolint:gocyclo
	node, ok := derefExprNode(node)
	if !ok {
		return
	}
	switch n := node.(type) {
	case VariableExpr:
		c.addFreeVar(variableKey(n.Name))
	case CastExpr:
		c.addAtomicType(n.Type.Prefix, n.Type.Name)
		c.walk(n.Expr)
	case CastableExpr:
		c.addAtomicType(n.Type.Prefix, n.Type.Name)
		c.walk(n.Expr)
	case InstanceOfExpr:
		c.walkSequenceType(n.Type)
		c.walk(n.Expr)
	case TreatAsExpr:
		c.walkSequenceType(n.Type)
		c.walk(n.Expr)
	case FunctionCall:
		c.addFunctionName(n.Prefix, n.Name)
		for _, arg := range n.Args {
			c.walk(arg)
		}
	case NamedFunctionRef:
		c.addFunctionName(n.Prefix, n.Name)
	case DynamicFunctionCall:
		c.walk(n.Func)
		for _, arg := range n.Args {
			c.walk(arg)
		}
	case PathExpr:
		c.walk(n.Filter)
		c.walkLocationPath(n.Path)
	case *LocationPath:
		c.walkLocationPath(n)
	case LocationPath:
		c.walkLocationPath(&n)
	case PathStepExpr:
		c.walk(n.Left)
		c.walk(n.Right)
	case FilterExpr:
		c.walk(n.Expr)
		for _, p := range n.Predicates {
			c.walk(p)
		}
	case BinaryExpr:
		c.walk(n.Left)
		c.walk(n.Right)
	case UnaryExpr:
		c.walk(n.Operand)
	case ConcatExpr:
		c.walk(n.Left)
		c.walk(n.Right)
	case SimpleMapExpr:
		c.walk(n.Left)
		c.walk(n.Right)
	case RangeExpr:
		c.walk(n.Start)
		c.walk(n.End)
	case UnionExpr:
		c.walk(n.Left)
		c.walk(n.Right)
	case IntersectExceptExpr:
		c.walk(n.Left)
		c.walk(n.Right)
	case IfExpr:
		c.walk(n.Cond)
		c.walk(n.Then)
		c.walk(n.Else)
	case FLWORExpr:
		c.walkFLWOR(n)
	case QuantifiedExpr:
		c.walkQuantified(n)
	case TryCatchExpr:
		c.walkTryCatch(n)
	case InlineFunctionExpr:
		c.walkInlineFunction(n)
	case LookupExpr:
		c.walk(n.Expr)
		c.walk(n.Key)
	case UnaryLookupExpr:
		c.walk(n.Key)
	case MapConstructorExpr:
		for _, p := range n.Pairs {
			c.walk(p.Key)
			c.walk(p.Value)
		}
	case ArrayConstructorExpr:
		for _, item := range n.Items {
			c.walk(item)
		}
	case SequenceExpr:
		for _, item := range n.Items {
			c.walk(item)
		}
	}
}

func (c *staticRefCollector) walkLocationPath(lp *LocationPath) {
	if lp == nil {
		return
	}
	for i := range lp.Steps {
		// A step's node test may be a type-annotated kind test (e.g.
		// self::element(*, t:T)), which carries a user/built-in type name.
		c.walkItemTest(lp.Steps[i].NodeTest)
		for _, p := range lp.Steps[i].Predicates {
			c.walk(p)
		}
	}
}

// walkFLWOR evaluates for/let clauses left to right: each clause's expression is
// in the scope established by the PRIOR clauses, and the clause's own variable
// becomes bound for the clauses that follow and the return expression.
func (c *staticRefCollector) walkFLWOR(n FLWORExpr) {
	var added []string
	for _, clause := range n.Clauses {
		switch cl := clause.(type) {
		case ForClause:
			c.walk(cl.Expr)
			c.bindVar(cl.Var)
			added = append(added, cl.Var)
			if cl.PosVar != "" {
				c.bindVar(cl.PosVar)
				added = append(added, cl.PosVar)
			}
		case LetClause:
			c.walk(cl.Expr)
			c.bindVar(cl.Var)
			added = append(added, cl.Var)
		}
	}
	c.walk(n.Return)
	for _, k := range added {
		c.unbindVar(k)
	}
}

// walkTryCatch walks a try/catch expression. Each catch body runs with the
// implicit error variables ($err:code, …) bound by the evaluator, so they are
// bound (refcounted, restoring any shadowed outer binding) for the duration of the
// body and a reference to one is not reported as a free variable.
func (c *staticRefCollector) walkTryCatch(n TryCatchExpr) {
	c.walk(n.Try)
	for _, ct := range n.Catches {
		for _, k := range catchErrorVarKeys {
			c.bindVar(k)
		}
		c.walk(ct.Expr)
		for _, k := range catchErrorVarKeys {
			c.unbindVar(k)
		}
	}
}

// walkQuantified binds every quantifier variable for the satisfies expression;
// each binding's domain is in the scope of the prior bindings.
func (c *staticRefCollector) walkQuantified(n QuantifiedExpr) {
	var added []string
	for _, b := range n.Bindings {
		c.walk(b.Domain)
		c.bindVar(b.Var)
		added = append(added, b.Var)
	}
	c.walk(n.Satisfies)
	for _, k := range added {
		c.unbindVar(k)
	}
}

// walkInlineFunction collects references from an inline function LITERAL
// (function($v as T) as R { body }): its parameter and return SEQUENCE TYPES may
// name user-defined atomic types (which the CTA static check forbids), so they are
// walked alongside the body, mirroring the namespace validator's traversal. The
// parameters are bound for the duration of the body.
func (c *staticRefCollector) walkInlineFunction(n InlineFunctionExpr) {
	var added []string
	for _, p := range n.Params {
		if p.TypeHint != nil {
			c.walkSequenceType(*p.TypeHint)
		}
		c.bindVar(p.Name)
		added = append(added, p.Name)
	}
	if n.ReturnType != nil {
		c.walkSequenceType(*n.ReturnType)
	}
	c.walk(n.Body)
	for _, k := range added {
		c.unbindVar(k)
	}
}
