package xpath3

import (
	"context"
	"fmt"
	"strings"
)

func evalInstanceOfExpr(ec *evalContext, e InstanceOfExpr) (Sequence, error) {
	seq, err := eval(ec, e.Expr)
	if err != nil {
		return nil, err
	}
	return SingleBoolean(matchesSequenceType(seq, e.Type, ec)), nil
}

func evalCastExpr(ec *evalContext, e CastExpr) (Sequence, error) {
	targetType := resolveAtomicTypeName(e.Type, ec)
	if isAbstractCastTarget(targetType) {
		return nil, &XPathError{
			Code:    errCodeXPST0080,
			Message: fmt.Sprintf("cannot use abstract type %s as cast target", targetType),
		}
	}
	seq, err := eval(ec, e.Expr)
	if err != nil {
		return nil, err
	}
	if len(seq) == 0 {
		if e.AllowEmpty {
			return nil, nil
		}
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "cast requires non-empty sequence"}
	}
	if len(seq) > 1 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "cast requires singleton"}
	}
	av, err := AtomizeItem(seq[0])
	if err != nil {
		return nil, err
	}
	// xs:QName cast from string requires namespace context
	if targetType == TypeQName {
		result, err := castToQName(av, ec)
		if err != nil {
			return nil, err
		}
		return SingleAtomic(result), nil
	}
	// xs:numeric is a union type: if already numeric return as-is, else cast to double
	if targetType == TypeNumeric {
		if av.IsNumeric() {
			return SingleAtomic(av), nil
		}
		result, err := CastAtomic(av, TypeDouble)
		if err != nil {
			return nil, err
		}
		return SingleAtomic(result), nil
	}
	result, err := CastAtomic(av, targetType)
	if err != nil {
		return nil, err
	}
	return SingleAtomic(result), nil
}

func evalCastableExpr(ec *evalContext, e CastableExpr) (Sequence, error) {
	targetType := resolveAtomicTypeName(e.Type, ec)
	// Abstract types raise a static error even for castable (XPST0080)
	if isAbstractCastTarget(targetType) {
		return nil, &XPathError{
			Code:    errCodeXPST0080,
			Message: fmt.Sprintf("cannot use abstract type %s as castable target", targetType),
		}
	}
	seq, err := eval(ec, e.Expr)
	if err != nil {
		return nil, err
	}
	if len(seq) == 0 {
		return SingleBoolean(e.AllowEmpty), nil
	}
	if len(seq) > 1 {
		return SingleBoolean(false), nil
	}
	av, err := AtomizeItem(seq[0])
	if err != nil {
		return SingleBoolean(false), nil
	}
	// xs:QName cast from string requires namespace context
	if targetType == TypeQName {
		_, castErr := castToQName(av, ec)
		return SingleBoolean(castErr == nil), nil
	}
	// xs:numeric is a union type
	if targetType == TypeNumeric {
		if av.IsNumeric() {
			return SingleBoolean(true), nil
		}
		_, castErr := CastAtomic(av, TypeDouble)
		return SingleBoolean(castErr == nil), nil
	}
	_, castErr := CastAtomic(av, targetType)
	return SingleBoolean(castErr == nil), nil
}

func evalTreatAsExpr(ec *evalContext, e TreatAsExpr) (Sequence, error) {
	seq, err := eval(ec, e.Expr)
	if err != nil {
		return nil, err
	}
	if !matchesSequenceType(seq, e.Type, ec) {
		return nil, &XPathError{Code: errCodeXPDY0050, Message: fmt.Sprintf("treat as: sequence does not match required type %v (actual length %d)", e.Type, len(seq))}
	}
	return seq, nil
}

// resolveAtomicTypeName maps an AtomicTypeName to an internal xs:-prefixed type string.
// Per XPath 3.1, unprefixed type names in cast/instance-of default to the xs: namespace.
// This is a pragmatic simplification; a strict implementation would require checking
// the default element namespace and raising XPST0081 if it is not XSD.
func resolveAtomicTypeName(tn AtomicTypeName, ec *evalContext) string {
	if tn.Prefix == "" {
		return "xs:" + tn.Name
	}
	if tn.Prefix == "xs" || tn.Prefix == "xsd" {
		return "xs:" + tn.Name
	}
	// Resolve via namespace context
	if ec.namespaces != nil {
		if uri, ok := ec.namespaces[tn.Prefix]; ok {
			if uri == "http://www.w3.org/2001/XMLSchema" {
				return "xs:" + tn.Name
			}
		}
	}
	return tn.Prefix + ":" + tn.Name
}

// coerceToSequenceType applies XPath 3.1 function coercion rules (§3.1.5.2):
// atomization, numeric promotion (integer→float/double, float→double),
// and URI-to-string promotion. Returns the coerced sequence and true if
// coercion succeeded, or the original sequence and false if it did not.
func coerceToSequenceType(seq Sequence, st SequenceType, ec *evalContext) (Sequence, bool) {
	if matchesSequenceType(seq, st, ec) {
		return seq, true
	}
	if len(seq) == 1 {
		if fnTest, ok := st.ItemTest.(FunctionTest); ok {
			adapted, ok := coerceFunctionItem(seq[0], fnTest, ec)
			if ok {
				return Sequence{adapted}, true
			}
		}
	}
	// Try to coerce each item
	var targetType string
	switch t := st.ItemTest.(type) {
	case AtomicOrUnionType:
		targetType = resolveAtomicTypeName(AtomicTypeName(t), ec)
	default:
		return seq, false
	}
	result := make(Sequence, len(seq))
	for i, item := range seq {
		av, ok := item.(AtomicValue)
		if !ok {
			// Try atomization
			atomized, err := AtomizeItem(item)
			if err != nil {
				return seq, false
			}
			av = atomized
		}
		// Numeric promotion
		switch targetType {
		case TypeDouble:
			if av.TypeName == TypeInteger || av.TypeName == TypeDecimal || av.TypeName == TypeFloat ||
				isSubtypeOf(av.TypeName, TypeInteger) {
				promoted, err := castToDouble(av)
				if err != nil {
					return seq, false
				}
				result[i] = promoted
				continue
			}
		case TypeFloat:
			if av.TypeName == TypeInteger || av.TypeName == TypeDecimal ||
				isSubtypeOf(av.TypeName, TypeInteger) {
				promoted, err := castToFloat(av)
				if err != nil {
					return seq, false
				}
				result[i] = promoted
				continue
			}
		case TypeString:
			if av.TypeName == TypeAnyURI {
				result[i] = AtomicValue{TypeName: TypeString, Value: av.Value}
				continue
			}
		}
		// Untypedatomic → target type
		if av.TypeName == TypeUntypedAtomic {
			cast, err := CastAtomic(av, targetType)
			if err != nil {
				return seq, false
			}
			result[i] = cast
			continue
		}
		if isSubtypeOf(av.TypeName, targetType) {
			result[i] = av
			continue
		}
		return seq, false
	}
	// Verify occurrence constraint on the coerced result
	switch st.Occurrence {
	case OccurrenceExactlyOne:
		if len(result) != 1 {
			return seq, false
		}
	case OccurrenceOneOrMore:
		if len(result) == 0 {
			return seq, false
		}
	case OccurrenceZeroOrOne:
		if len(result) > 1 {
			return seq, false
		}
	}
	return result, true
}

func coerceFunctionItem(item Item, target FunctionTest, ec *evalContext) (Item, bool) {
	if target.AnyFunction {
		return nil, false
	}

	actual, ok := item.(FunctionItem)
	if !ok {
		return nil, false
	}
	if actual.Arity >= 0 && actual.Arity != len(target.ParamTypes) {
		return nil, false
	}

	paramTypes := append([]SequenceType(nil), target.ParamTypes...)
	returnType := target.ReturnType

	return FunctionItem{
		Arity:      len(paramTypes),
		Name:       actual.Name,
		Namespace:  actual.Namespace,
		ParamTypes: paramTypes,
		ReturnType: &returnType,
		Invoke: func(ctx context.Context, args []Sequence) (Sequence, error) {
			if len(args) != len(paramTypes) {
				return nil, &XPathError{
					Code:    errCodeXPTY0004,
					Message: fmt.Sprintf("arity mismatch: expected %d arguments, got %d", len(paramTypes), len(args)),
				}
			}

			invokeCtx := ec
			if fnCtx := getFnContext(ctx); fnCtx != nil {
				invokeCtx = fnCtx
			}

			callArgs := make([]Sequence, len(args))
			copy(callArgs, args)
			if len(actual.ParamTypes) > 0 {
				for i, arg := range args {
					coerced, ok := coerceToSequenceType(arg, actual.ParamTypes[i], invokeCtx)
					if !ok {
						return nil, &XPathError{
							Code:    errCodeXPTY0004,
							Message: fmt.Sprintf("function argument %d does not match required type %v", i+1, actual.ParamTypes[i]),
						}
					}
					callArgs[i] = coerced
				}
			}

			result, err := actual.Invoke(ctx, callArgs)
			if err != nil {
				return nil, err
			}

			coercedResult, ok := coerceToSequenceType(result, target.ReturnType, invokeCtx)
			if !ok {
				return nil, &XPathError{
					Code:    errCodeXPTY0004,
					Message: fmt.Sprintf("function result does not match required type %v", target.ReturnType),
				}
			}
			return coercedResult, nil
		},
	}, true
}

func matchesSequenceType(seq Sequence, st SequenceType, ec *evalContext) bool {
	if st.Void {
		return len(seq) == 0
	}
	switch st.Occurrence {
	case OccurrenceExactlyOne:
		if len(seq) != 1 {
			return false
		}
	case OccurrenceZeroOrOne:
		if len(seq) > 1 {
			return false
		}
	case OccurrenceOneOrMore:
		if len(seq) == 0 {
			return false
		}
	case OccurrenceZeroOrMore:
		// any length ok
	}
	for _, item := range seq {
		if !matchesItemType(item, st.ItemTest, ec) {
			return false
		}
	}
	return true
}

func matchesItemType(item Item, test NodeTest, ec *evalContext) bool {
	if test == nil {
		return true
	}
	switch t := test.(type) {
	case AnyItemTest:
		return true
	case TypeTest:
		ni, ok := item.(NodeItem)
		if !ok {
			return false
		}
		return matchTypeTest(t, ni.Node)
	case NameTest:
		// XPath 3.1: bare NameTest in SequenceType context matches elements only,
		// so AxisChild is correct (matchNameTest default branch requires ElementNode).
		ni, ok := item.(NodeItem)
		if !ok {
			return false
		}
		return matchNameTest(t, ni.Node, AxisChild, ec)
	case ElementTest:
		ni, ok := item.(NodeItem)
		if !ok {
			return false
		}
		if t.TypeName != "" {
			ann := ni.TypeAnnotation
			if ann == "" {
				ann = TypeUntyped // elements default to xs:untyped
			}
			target := resolveTestTypeName(t.TypeName, ec)
			if !isSubtypeOf(ann, target) {
				return false
			}
		}
		return matchNodeTest(t, ni.Node, AxisChild, ec)
	case AttributeTest:
		ni, ok := item.(NodeItem)
		if !ok {
			return false
		}
		if t.TypeName != "" {
			ann := ni.TypeAnnotation
			if ann == "" {
				ann = TypeUntypedAtomic // attributes default to xs:untypedAtomic
			}
			target := resolveTestTypeName(t.TypeName, ec)
			if !isSubtypeOf(ann, target) {
				return false
			}
		}
		return matchNodeTest(t, ni.Node, AxisAttribute, ec)
	case DocumentTest:
		ni, ok := item.(NodeItem)
		if !ok {
			return false
		}
		return matchNodeTest(t, ni.Node, AxisChild, ec)
	case AtomicOrUnionType:
		av, ok := item.(AtomicValue)
		if !ok {
			return false
		}
		targetType := resolveAtomicTypeName(AtomicTypeName(t), ec)
		if targetType == TypeAnyAtomicType {
			return true
		}
		return isSubtypeOf(av.TypeName, targetType)
	case FunctionTest:
		// Maps and arrays are functions per XPath 3.1
		if t.AnyFunction {
			switch item.(type) {
			case FunctionItem, MapItem, ArrayItem:
				return true
			}
			return false
		}
		// Typed function test: function(ParamTypes...) as ReturnType
		switch v := item.(type) {
		case FunctionItem:
			// Check arity matches
			if v.Arity >= 0 && v.Arity != len(t.ParamTypes) {
				return false
			}
			// If the function has typed parameters, check subtype relationship
			// Function subtyping: contravariant in parameters, covariant in return type
			if len(v.ParamTypes) > 0 && len(t.ParamTypes) > 0 {
				for i, testParam := range t.ParamTypes {
					// Contravariant: the function's declared param type must be a supertype
					// of (or same as) the test's param type. In practice, this means the
					// test's param type must be a subtype of the function's param type.
					if !isSequenceSubtype(testParam, v.ParamTypes[i], ec) {
						return false
					}
				}
			}
			// Return type: covariant - the function's return must be a subtype of test's return
			if v.ReturnType != nil && t.ReturnType.ItemTest != nil {
				if !isSequenceSubtype(*v.ReturnType, t.ReturnType, ec) {
					return false
				}
			}
			return true
		case MapItem:
			// Maps are function(xs:anyAtomicType) as item()*
			// Match function(K) as V:
			// - If V requires at least one item (ExactlyOne or OneOrMore), the map can't
			//   guarantee returning non-empty for all possible K inputs → false
			// - Otherwise, check that all existing values for keys matching K satisfy V
			if len(t.ParamTypes) != 1 {
				return false
			}
			rt := t.ReturnType
			// Maps may return empty for missing keys, so V must allow empty
			if !rt.Void && (rt.Occurrence == OccurrenceExactlyOne || rt.Occurrence == OccurrenceOneOrMore) {
				return false
			}
			allMatch := true
			m := v
			_ = m.ForEach(func(k AtomicValue, val Sequence) error {
				if matchesItemType(k, t.ParamTypes[0].ItemTest, ec) {
					if !matchesSequenceType(val, rt, ec) {
						allMatch = false
					}
				}
				return nil
			})
			return allMatch
		case ArrayItem:
			// Arrays are function(xs:integer) as item()*
			if len(t.ParamTypes) != 1 {
				return false
			}
			// Check all members match return type
			for _, member := range v.members0() {
				if !matchesSequenceType(member, t.ReturnType, ec) {
					return false
				}
			}
			return true
		}
		return false
	case MapTest:
		m, ok := item.(MapItem)
		if !ok {
			return false
		}
		if t.AnyType {
			return true
		}
		// Typed map test: check all keys match key type and all values match value type
		allMatch := true
		_ = m.ForEach(func(k AtomicValue, v Sequence) error {
			if !matchesItemType(k, t.KeyType, ec) {
				allMatch = false
			}
			if allMatch && !matchesSequenceType(v, t.ValType, ec) {
				allMatch = false
			}
			return nil
		})
		return allMatch
	case ArrayTest:
		a, ok := item.(ArrayItem)
		if !ok {
			return false
		}
		if t.AnyType {
			return true
		}
		// Typed array test: check all members match member type
		for _, member := range a.members0() {
			if !matchesSequenceType(member, t.MemberType, ec) {
				return false
			}
		}
		return true
	}
	return false
}

// isSequenceSubtype checks if type A is a subtype of type B.
// This is used for function subtyping checks.
func isSequenceSubtype(a, b SequenceType, ec *evalContext) bool {
	// Void (empty-sequence()) is subtype of anything with ZeroOrMore or ZeroOrOne occurrence
	if a.Void {
		return b.Void || b.Occurrence == OccurrenceZeroOrMore || b.Occurrence == OccurrenceZeroOrOne
	}
	if b.Void {
		return a.Void
	}
	// Check occurrence compatibility: A's occurrence must be at least as restrictive as B's
	if !occurrenceSubtype(a.Occurrence, b.Occurrence) {
		return false
	}
	// Check item type subtype
	return isItemTypeSubtype(a.ItemTest, b.ItemTest, ec)
}

// occurrenceSubtype checks if occurrence a is at least as restrictive as b.
func occurrenceSubtype(a, b Occurrence) bool {
	switch b {
	case OccurrenceZeroOrMore:
		return true // anything is subtype of *
	case OccurrenceOneOrMore:
		return a == OccurrenceExactlyOne || a == OccurrenceOneOrMore
	case OccurrenceZeroOrOne:
		return a == OccurrenceExactlyOne || a == OccurrenceZeroOrOne
	case OccurrenceExactlyOne:
		return a == OccurrenceExactlyOne
	}
	return false
}

// isItemTypeSubtype checks if item type A is a subtype of item type B.
func isItemTypeSubtype(a, b NodeTest, ec *evalContext) bool {
	if b == nil || isAnyItemTest(b) {
		return true // item() is supertype of everything
	}
	if a == nil || isAnyItemTest(a) {
		return false // item() is not subtype of anything more specific
	}
	// Same type category checks
	switch bt := b.(type) {
	case AtomicOrUnionType:
		at, ok := a.(AtomicOrUnionType)
		if !ok {
			return false
		}
		aType := resolveAtomicTypeName(AtomicTypeName(at), ec)
		bType := resolveAtomicTypeName(AtomicTypeName(bt), ec)
		return isSubtypeOf(aType, bType)
	case MapTest:
		at, ok := a.(MapTest)
		if !ok {
			// FunctionTest might be supertype of MapTest
			return false
		}
		if bt.AnyType {
			return true // map(*) is supertype of all maps
		}
		if at.AnyType {
			return false // map(*) is not subtype of typed map
		}
		// Check key and value types
		return isItemTypeSubtype(at.KeyType, bt.KeyType, ec) &&
			isSequenceSubtype(at.ValType, bt.ValType, ec)
	case ArrayTest:
		at, ok := a.(ArrayTest)
		if !ok {
			return false
		}
		if bt.AnyType {
			return true
		}
		if at.AnyType {
			return false
		}
		return isSequenceSubtype(at.MemberType, bt.MemberType, ec)
	case TypeTest:
		// node() and its kind specializations
		if bt.Kind == NodeKindNode {
			// node() is supertype of all node-related tests
			switch a.(type) {
			case TypeTest, ElementTest, AttributeTest, DocumentTest, NameTest:
				return true
			}
			return false
		}
		at, ok := a.(TypeTest)
		if !ok {
			return false
		}
		// Same kind or at is more specific
		return at.Kind == bt.Kind
	case NameTest:
		// element/attribute name tests — A must be the same or more specific name
		at, ok := a.(NameTest)
		if !ok {
			return false
		}
		if bt.Local == "*" {
			return true
		}
		return at.Local == bt.Local && at.Prefix == bt.Prefix
	case ElementTest:
		at, ok := a.(ElementTest)
		if !ok {
			return false
		}
		if bt.Name == "" {
			return true // element() matches any element
		}
		return at.Name == bt.Name
	case AttributeTest:
		at, ok := a.(AttributeTest)
		if !ok {
			return false
		}
		if bt.Name == "" {
			return true
		}
		return at.Name == bt.Name
	case FunctionTest:
		// MapTest and ArrayTest are subtypes of FunctionTest
		if bt.AnyFunction {
			switch a.(type) {
			case FunctionTest, MapTest, ArrayTest:
				return true
			}
			return false
		}
		// Typed function test: check contravariant params and covariant return
		switch at := a.(type) {
		case FunctionTest:
			if at.AnyFunction {
				return false
			}
			if len(at.ParamTypes) != len(bt.ParamTypes) {
				return false
			}
			for i := range bt.ParamTypes {
				// Contravariant: B's param must be subtype of A's param
				if !isSequenceSubtype(bt.ParamTypes[i], at.ParamTypes[i], ec) {
					return false
				}
			}
			return isSequenceSubtype(at.ReturnType, bt.ReturnType, ec)
		case MapTest:
			// Map is function(xs:anyAtomicType) as item()*
			if len(bt.ParamTypes) != 1 {
				return false
			}
			return true // Maps can be subtype of compatible function tests
		case ArrayTest:
			if len(bt.ParamTypes) != 1 {
				return false
			}
			return true
		}
		return false
	}
	return false
}

func isAnyItemTest(t NodeTest) bool {
	_, ok := t.(AnyItemTest)
	return ok
}

// castToQName casts an atomic value to xs:QName using the evaluation context's
// namespace bindings to resolve prefixes. Per XPath 3.1, casting from xs:string
// (or xs:untypedAtomic) to xs:QName requires namespace context.
func castToQName(v AtomicValue, ec *evalContext) (AtomicValue, error) {
	// QName → QName: identity
	if v.TypeName == TypeQName {
		return v, nil
	}

	// Only string and untypedAtomic can be cast to QName
	if v.TypeName != TypeString && v.TypeName != TypeUntypedAtomic {
		return AtomicValue{}, &XPathError{
			Code:    errCodeXPTY0004,
			Message: fmt.Sprintf("cannot cast %s to %s", v.TypeName, TypeQName),
		}
	}

	s := strings.TrimSpace(v.StringVal())
	prefix := ""
	local := s
	if idx := strings.IndexByte(s, ':'); idx >= 0 {
		prefix = s[:idx]
		local = s[idx+1:]
	}

	if !isValidNCName(local) || (prefix != "" && !isValidNCName(prefix)) {
		return AtomicValue{}, &XPathError{
			Code:    errCodeFORG0001,
			Message: fmt.Sprintf("invalid QName: %q", s),
		}
	}

	uri := ""
	if prefix != "" {
		resolved := false
		if ec != nil && ec.namespaces != nil {
			if ns, ok := ec.namespaces[prefix]; ok {
				uri = ns
				resolved = true
			}
		}
		if !resolved {
			// Fall back to default prefix mappings (fn, xs, math, map, array, err)
			if ns, ok := defaultPrefixNS[prefix]; ok {
				uri = ns
				resolved = true
			}
		}
		if !resolved {
			return AtomicValue{}, &XPathError{
				Code:    errCodeFONS0004,
				Message: fmt.Sprintf("no namespace binding for prefix %q", prefix),
			}
		}
	} else {
		// No prefix: check default namespace in context
		if ec != nil && ec.namespaces != nil {
			if ns, ok := ec.namespaces[""]; ok {
				uri = ns
			}
		}
	}

	return AtomicValue{
		TypeName: TypeQName,
		Value:    QNameValue{Prefix: prefix, Local: local, URI: uri},
	}, nil
}
