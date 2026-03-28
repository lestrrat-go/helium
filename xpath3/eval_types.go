package xpath3

import (
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xmlchar"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

func evalInstanceOfExpr(evalFn exprEvaluator, goCtx context.Context, ec *evalContext, e InstanceOfExpr) (Sequence, error) {
	seq, err := evalFn(goCtx, ec, e.Expr)
	if err != nil {
		return nil, err
	}
	return SingleBoolean(matchesSequenceType(seq, e.Type, ec)), nil
}

func evalCastExpr(evalFn exprEvaluator, goCtx context.Context, ec *evalContext, e CastExpr) (Sequence, error) {
	targetType := resolveAtomicTypeName(e.Type, ec)
	if isAbstractCastTarget(targetType) {
		return nil, &XPathError{
			Code:    errCodeXPST0080,
			Message: fmt.Sprintf("cannot use abstract type %s as cast target", targetType),
		}
	}
	seq, err := evalFn(goCtx, ec, e.Expr)
	if err != nil {
		return nil, err
	}
	if seqLen(seq) == 0 {
		if e.AllowEmpty {
			return nil, nil //nolint:nilnil
		}
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "cast requires non-empty sequence"}
	}
	if seqLen(seq) > 1 {
		return nil, &XPathError{Code: errCodeXPTY0004, Message: "cast requires singleton"}
	}
	av, err := AtomizeItem(seq.Get(0))
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
		// If CastAtomic fails for a non-built-in type, try resolving via schema declarations.
		if ec.schemaDeclarations != nil {
			ns := ""
			if e.Type.Prefix != "" && ec.namespaces != nil {
				ns = ec.namespaces[e.Type.Prefix]
			}
			if builtinBase := resolveToBuiltinBase(e.Type.Name, ns, ec.schemaDeclarations); builtinBase != "" {
				result, castErr := CastAtomic(av, builtinBase)
				if castErr != nil {
					return nil, castErr
				}
				// Validate facets for user-defined types using Q{ns}local format.
				s, _ := AtomicToString(result)
				annName := QAnnotation(ns, e.Type.Name)
				if facetErr := ec.schemaDeclarations.ValidateCast(goCtx, s, annName); facetErr != nil {
					return nil, &XPathError{Code: errCodeFORG0001, Message: fmt.Sprintf("cannot cast %q to %s: %v", s, targetType, facetErr)}
				}
				result.TypeName = targetType
				return SingleAtomic(result), nil
			}
			// For union types, try each member type.
			if members := ec.schemaDeclarations.UnionMemberTypes(targetType); len(members) > 0 {
				for _, memberType := range members {
					result, castErr := CastAtomic(av, memberType)
					if castErr == nil {
						return SingleAtomic(result), nil
					}
				}
			}
		}
		return nil, err
	}
	return SingleAtomic(result), nil
}

func evalCastableExpr(evalFn exprEvaluator, goCtx context.Context, ec *evalContext, e CastableExpr) (Sequence, error) {
	targetType := resolveAtomicTypeName(e.Type, ec)
	// Abstract types raise a static error even for castable (XPST0080)
	if isAbstractCastTarget(targetType) {
		return nil, &XPathError{
			Code:    errCodeXPST0080,
			Message: fmt.Sprintf("cannot use abstract type %s as castable target", targetType),
		}
	}
	seq, err := evalFn(goCtx, ec, e.Expr)
	if err != nil {
		return nil, err
	}
	if seqLen(seq) == 0 {
		return SingleBoolean(e.AllowEmpty), nil
	}
	if seqLen(seq) > 1 {
		return SingleBoolean(false), nil
	}
	av, err := AtomizeItem(seq.Get(0))
	if err != nil {
		return SingleBoolean(false), nil //nolint:nilerr // castable returns false on atomization failure
	}
	// xs:QName cast from string requires namespace context
	if targetType == TypeQName {
		_, castErr := castToQName(av, ec)
		return SingleBoolean(castErr == nil), nil
	}
	// xs:ENTITIES, xs:IDREFS, xs:NMTOKENS are list types:
	// castable returns true if the string is a whitespace-separated list of
	// at least one valid member (NCName for ENTITIES/IDREFS, NMTOKEN for NMTOKENS).
	if targetType == TypeENTITIES || targetType == TypeIDREFS || targetType == TypeNMTOKENS {
		s, err := AtomicToString(av)
		if err != nil {
			return SingleBoolean(false), nil //nolint:nilerr // castable returns false on conversion failure
		}
		return SingleBoolean(isValidListCastable(s, targetType)), nil
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
	if castErr != nil && ec.schemaDeclarations != nil {
		ns := ""
		if e.Type.Prefix != "" && ec.namespaces != nil {
			ns = ec.namespaces[e.Type.Prefix]
		}
		annName := QAnnotation(ns, e.Type.Name)
		if builtinBase := resolveToBuiltinBase(e.Type.Name, ns, ec.schemaDeclarations); builtinBase != "" {
			// NOTATION and QName derived types require namespace context
			// for resolution. The abstract base type (xs:NOTATION, xs:QName)
			// cannot be used as a cast target directly, so validate with
			// namespace context instead.
			if builtinBase == TypeNOTATION || builtinBase == TypeQName {
				// Only string, untypedAtomic, QName, and NOTATION types
				// (or schema-derived variants holding QNameValue) can be
				// cast to QName/NOTATION derived types.
				_, isQV := av.Value.(QNameValue)
				srcOK := av.TypeName == TypeString || av.TypeName == TypeUntypedAtomic ||
					av.TypeName == TypeQName || av.TypeName == TypeNOTATION || isQV
				if srcOK {
					s, _ := AtomicToString(av)
					castErr = ec.schemaDeclarations.ValidateCastWithNS(goCtx, s, annName, ec.namespaces)
				}
			} else {
				result, baseErr := CastAtomic(av, builtinBase)
				if baseErr == nil {
					s, _ := AtomicToString(result)
					castErr = ec.schemaDeclarations.ValidateCast(goCtx, s, annName)
				} else {
					castErr = baseErr
				}
			}
		}
	}
	return SingleBoolean(castErr == nil), nil
}

// isValidListCastable checks whether a string value is castable to
// a list type (xs:ENTITIES, xs:IDREFS, xs:NMTOKENS). The string must
// contain at least one whitespace-separated token, and each token must
// be valid for the list's member type (NCName for ENTITIES/IDREFS,
// NMTOKEN for NMTOKENS).
func isValidListCastable(s string, listType string) bool {
	tokens := strings.Fields(s)
	if len(tokens) == 0 {
		return false
	}
	for _, tok := range tokens {
		switch listType {
		case TypeENTITIES, TypeIDREFS:
			if !reNCName.MatchString(tok) {
				return false
			}
		case TypeNMTOKENS:
			if !reNMTOKEN.MatchString(tok) {
				return false
			}
		}
	}
	return true
}

func evalTreatAsExpr(evalFn exprEvaluator, goCtx context.Context, ec *evalContext, e TreatAsExpr) (Sequence, error) {
	seq, err := evalFn(goCtx, ec, e.Expr)
	if err != nil {
		return nil, err
	}
	if !matchesSequenceType(seq, e.Type, ec) {
		return nil, &XPathError{Code: errCodeXPDY0050, Message: fmt.Sprintf("treat as: sequence does not match required type %v (actual length %d)", e.Type, seqLen(seq))}
	}
	return seq, nil
}

// resolveToBuiltinBase walks the schema type hierarchy from a user-defined type
// to find the ultimate built-in XSD base type (e.g., xs:integer). Returns ""
// if the type is not found in schema declarations.
func resolveToBuiltinBase(local, ns string, decls SchemaDeclarations) string {
	current := local
	currentNS := ns
	for range 32 {
		baseType, ok := decls.LookupSchemaType(current, currentNS)
		if !ok {
			return ""
		}
		if IsKnownXSDType(baseType) {
			return baseType
		}
		// Parse the Q{ns}local format for the next iteration.
		newLocal, newNS, parsed := schemaAnnotationParts(baseType)
		if !parsed {
			return ""
		}
		current = newLocal
		currentNS = newNS
	}
	return ""
}

// resolveAtomicTypeName maps an AtomicTypeName to the internal type name format.
// Unprefixed names first try an imported no-namespace schema type, then fall
// back to the existing xs: shorthand behavior when no schema declaration exists.
func resolveAtomicTypeName(tn AtomicTypeName, ec *evalContext) string {
	if tn.Prefix == "" {
		// Check if a default element namespace is set.
		if ec != nil && ec.namespaces != nil {
			if defNS, ok := ec.namespaces[""]; ok && defNS != "" {
				if defNS == lexicon.NamespaceXSD {
					return "xs:" + tn.Name
				}
				return QAnnotation(defNS, tn.Name)
			}
		}
		if ec != nil && ec.schemaDeclarations != nil {
			if _, ok := ec.schemaDeclarations.LookupSchemaType(tn.Name, ""); ok {
				return QAnnotation("", tn.Name)
			}
		}
		// No default namespace: assume xs: namespace for known types.
		// For unknown names, only use Q{} form when schema declarations
		// are available (XSLT schema-aware mode); otherwise keep xs:
		// prefix so static type checking can report XPST0051.
		return "xs:" + tn.Name
	}
	if tn.Prefix == "xs" || tn.Prefix == "xsd" {
		return "xs:" + tn.Name
	}
	// Resolve via namespace context
	if ec.namespaces != nil {
		if uri, ok := ec.namespaces[tn.Prefix]; ok {
			if uri == lexicon.NamespaceXSD {
				return "xs:" + tn.Name
			}
			// Non-XSD namespace: use Q{ns}local annotation format
			return QAnnotation(uri, tn.Name)
		}
	}
	return tn.Prefix + ":" + tn.Name
}

// CoerceToSequenceType applies XPath 3.1 function coercion rules (§3.1.5.2):
// atomization, numeric promotion (integer→float/double, float→double),
// URI-to-string promotion, and function coercion.  Returns the coerced
// sequence and true on success, or the original sequence and false on failure.
func CoerceToSequenceType(seq Sequence, st SequenceType) (Sequence, bool) {
	return coerceToSequenceType(seq, st, nil)
}

// coerceToSequenceType is the internal version with an evalContext.
func coerceToSequenceType(seq Sequence, st SequenceType, ec *evalContext) (Sequence, bool) {
	if matchesSequenceType(seq, st, ec) {
		return seq, true
	}
	if seqLen(seq) == 1 {
		if fnTest, ok := st.ItemTest.(FunctionTest); ok {
			adapted, ok := coerceFunctionItem(seq.Get(0), fnTest, ec)
			if ok {
				return ItemSlice{adapted}, true
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
	result := make(ItemSlice, seqLen(seq))
	i := 0
	for item := range seqItems(seq) {
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
				i++
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
				i++
				continue
			}
		case TypeString:
			if av.TypeName == TypeAnyURI {
				result[i] = AtomicValue{TypeName: TypeString, Value: av.Value}
				i++
				continue
			}
		}
		// Untypedatomic → target type
		if av.TypeName == TypeUntypedAtomic {
			// xs:numeric is a union type — cast untypedAtomic to xs:double
			castTarget := targetType
			if castTarget == TypeNumeric {
				castTarget = TypeDouble
			}
			cast, err := CastAtomic(av, castTarget)
			if err != nil {
				return seq, false
			}
			result[i] = cast
			i++
			continue
		}
		if isSubtypeOf(av.TypeName, targetType) {
			result[i] = av
			i++
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

// MatchesSequenceType checks if a sequence matches a SequenceType using
// the XPath 3.1 SequenceType matching rules (no coercion).
func MatchesSequenceType(seq Sequence, st SequenceType) bool {
	return matchesSequenceType(seq, st, nil)
}

// CheckFunctionParamCompat checks whether a FunctionItem's declared parameter
// types are compatible with a target FunctionTest's parameter types using
// contravariance. Returns true if each target param type is a subtype of the
// corresponding function param type (i.e., the function can accept the target's
// argument types). If the function has no declared param types, returns true
// (unknown types are assumed compatible).
func CheckFunctionParamCompat(fi FunctionItem, st SequenceType) bool {
	ft, ok := st.ItemTest.(FunctionTest)
	if !ok || ft.AnyFunction {
		return true
	}
	if len(fi.ParamTypes) == 0 || len(ft.ParamTypes) == 0 {
		return true // no type info → assume compatible
	}
	if len(fi.ParamTypes) != len(ft.ParamTypes) {
		return false
	}
	for i, testParam := range ft.ParamTypes {
		// Contravariant: the function's param must be a supertype of the target's param
		if !isSequenceSubtype(testParam, fi.ParamTypes[i], nil) {
			return false
		}
	}
	return true
}

func matchesSequenceType(seq Sequence, st SequenceType, ec *evalContext) bool {
	if st.Void {
		return seqLen(seq) == 0
	}
	switch st.Occurrence {
	case OccurrenceExactlyOne:
		if seqLen(seq) != 1 {
			return false
		}
	case OccurrenceZeroOrOne:
		if seqLen(seq) > 1 {
			return false
		}
	case OccurrenceOneOrMore:
		if seqLen(seq) == 0 {
			return false
		}
	case OccurrenceZeroOrMore:
		// any length ok
	}
	for item := range seqItems(seq) {
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
				if ec == nil || ec.schemaDeclarations == nil || !ec.schemaDeclarations.IsSubtypeOf(ann, target) {
					return false
				}
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
				if ec == nil || ec.schemaDeclarations == nil || !ec.schemaDeclarations.IsSubtypeOf(ann, target) {
					return false
				}
			}
		}
		return matchNodeTest(t, ni.Node, AxisAttribute, ec)
	case SchemaElementTest:
		ni, ok := item.(NodeItem)
		if !ok || ni.Node.Type() != helium.ElementNode {
			return false
		}
		if ec == nil || ec.schemaDeclarations == nil {
			return false
		}
		local, ns := resolveSchemaTestName(t.Name, ec)
		// The node must have the same name as the declared element
		// (or a substitution group member — checked separately below).
		nameMatch := ixpath.LocalNameOf(ni.Node) == local && ixpath.NodeNamespaceURI(ni.Node) == ns
		if !nameMatch {
			// Check substitution group membership.
			if ec.schemaDeclarations == nil {
				return false
			}
			headType, headFound := ec.schemaDeclarations.LookupSchemaElement(local, ns)
			if !headFound {
				return false
			}
			// The node's type must be a subtype of the head's type.
			ann := ni.TypeAnnotation
			if ann == "" {
				ann = TypeUntyped
			}
			// Untyped elements do NOT match schema-element().
			if ann == TypeUntyped {
				return false
			}
			if isSubtypeOf(ann, headType) || ec.schemaDeclarations.IsSubtypeOf(ann, headType) {
				return true
			}
			return false
		}
		typeName, found := ec.schemaDeclarations.LookupSchemaElement(local, ns)
		if !found {
			return false
		}
		ann := ni.TypeAnnotation
		if ann == "" {
			ann = TypeUntyped
		}
		// Untyped elements have not been validated — they do NOT match schema-element().
		if ann == TypeUntyped {
			return false
		}
		if !isSubtypeOf(ann, typeName) {
			if ec != nil && ec.schemaDeclarations != nil {
				return ec.schemaDeclarations.IsSubtypeOf(ann, typeName)
			}
			return false
		}
		return true
	case SchemaAttributeTest:
		ni, ok := item.(NodeItem)
		if !ok || ni.Node.Type() != helium.AttributeNode {
			return false
		}
		if ec == nil || ec.schemaDeclarations == nil {
			return false
		}
		local, ns := resolveSchemaTestName(t.Name, ec)
		// The node must have the same name as the declared attribute.
		if ixpath.LocalNameOf(ni.Node) != local || ixpath.NodeNamespaceURI(ni.Node) != ns {
			return false
		}
		typeName, found := ec.schemaDeclarations.LookupSchemaAttribute(local, ns)
		if !found {
			return false
		}
		ann := ni.TypeAnnotation
		if ann == "" {
			ann = TypeUntypedAtomic
		}
		// For instance-of checks, untyped attributes match schema-attribute()
		// when name + declaration exist. This handles constructed elements with
		// xsl:type that don't yet propagate attribute type annotations.
		if ann == TypeUntypedAtomic {
			return true
		}
		if !isSubtypeOf(ann, typeName) {
			if ec != nil && ec.schemaDeclarations != nil {
				return ec.schemaDeclarations.IsSubtypeOf(ann, typeName)
			}
			return false
		}
		return true
	case DocumentTest:
		ni, ok := item.(NodeItem)
		if !ok {
			return false
		}
		return matchNodeTest(t, ni.Node, AxisChild, ec)
	case PITest:
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
		if isSubtypeOf(av.TypeName, targetType) {
			return true
		}
		// Fall back to schema declarations for user-defined types.
		if ec != nil && ec.schemaDeclarations != nil {
			return ec.schemaDeclarations.IsSubtypeOf(av.TypeName, targetType)
		}
		return false
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
	case NamespaceNodeTest:
		ni, ok := item.(NodeItem)
		if !ok {
			return false
		}
		return ni.Node.Type() == helium.NamespaceNode
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
		if bt.Name == "" && bt.TypeName == "" {
			return true // element() matches any element
		}
		// Check name: if bt has a specific name, at must match
		if bt.Name != "" && bt.Name != "*" && at.Name != bt.Name {
			return false
		}
		// Check type annotation: if bt specifies a type, at must have a
		// compatible type annotation. element(N) without TypeName
		// is equivalent to element(N, xs:anyType?) (nillable), which
		// is NOT a subtype of element(N, T) (non-nillable).
		if bt.TypeName != "" {
			if at.TypeName == "" {
				return false // untyped element(N) is not subtype of typed element(N, T)
			}
			bType := resolveTestTypeName(bt.TypeName, ec)
			if !isSubtypeOf(at.TypeName, bType) {
				return false
			}
			// Nillable: element(N, T?) is not subtype of element(N, T)
			if at.Nillable && !bt.Nillable {
				return false
			}
		}
		return true
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

	// NOTATION → QName: NOTATION values are QName-like.
	if v.TypeName == TypeNOTATION {
		return AtomicValue{TypeName: TypeQName, Value: v.Value}, nil
	}
	if _, ok := v.Value.(QNameValue); ok {
		return AtomicValue{TypeName: TypeQName, Value: v.Value}, nil
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
	if p, l, ok := strings.Cut(s, ":"); ok {
		prefix = p
		local = l
	}

	if !xmlchar.IsValidNCName(local) || (prefix != "" && !xmlchar.IsValidNCName(prefix)) {
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
