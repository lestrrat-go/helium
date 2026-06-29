package xpath3

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
	// instance of / treat as. Kind-test type names (element()/attribute()) are not
	// included.
	TypeNames []TypeNameRef
}

// StaticReferences walks the expression's syntax tree and reports its free
// variable references and atomic type-name references. It is side-effect free and
// intended for schema-compile-time analysis, not the evaluation hot path.
func (e *Expression) StaticReferences() StaticReferences {
	ast := e.astExpr()
	c := &staticRefCollector{bound: map[string]struct{}{}}
	c.walk(ast)
	return StaticReferences{FreeVariables: c.freeVars, TypeNames: c.typeNames}
}

type staticRefCollector struct {
	bound     map[string]struct{}
	freeVars  []string
	typeNames []TypeNameRef
	seenFree  map[string]struct{}
}

func variableKey(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + ":" + name
}

func (c *staticRefCollector) addFreeVar(key string) {
	if _, ok := c.bound[key]; ok {
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

func (c *staticRefCollector) walkSequenceType(st SequenceType) {
	if aut, ok := st.ItemTest.(AtomicOrUnionType); ok {
		c.addAtomicType(aut.Prefix, aut.Name)
	}
}

func (c *staticRefCollector) walk(node Expr) { //nolint:gocyclo
	if node == nil {
		return
	}
	switch n := node.(type) {
	case VariableExpr:
		c.addFreeVar(variableKey(n.Prefix, n.Name))
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
		for _, arg := range n.Args {
			c.walk(arg)
		}
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
		c.walk(n.Try)
		for _, ct := range n.Catches {
			c.walk(ct.Expr)
		}
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
			key := cl.Var
			c.bound[key] = struct{}{}
			added = append(added, key)
			if cl.PosVar != "" {
				pk := cl.PosVar
				c.bound[pk] = struct{}{}
				added = append(added, pk)
			}
		case LetClause:
			c.walk(cl.Expr)
			key := cl.Var
			c.bound[key] = struct{}{}
			added = append(added, key)
		}
	}
	c.walk(n.Return)
	for _, k := range added {
		delete(c.bound, k)
	}
}

// walkQuantified binds every quantifier variable for the satisfies expression;
// each binding's domain is in the scope of the prior bindings.
func (c *staticRefCollector) walkQuantified(n QuantifiedExpr) {
	var added []string
	for _, b := range n.Bindings {
		c.walk(b.Domain)
		key := b.Var
		c.bound[key] = struct{}{}
		added = append(added, key)
	}
	c.walk(n.Satisfies)
	for _, k := range added {
		delete(c.bound, k)
	}
}

func (c *staticRefCollector) walkInlineFunction(n InlineFunctionExpr) {
	var added []string
	for _, p := range n.Params {
		key := p.Name
		c.bound[key] = struct{}{}
		added = append(added, key)
	}
	c.walk(n.Body)
	for _, k := range added {
		delete(c.bound, k)
	}
}
