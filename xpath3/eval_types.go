package xpath3

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/lestrrat-go/helium/internal/lexicon"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

func evalInstanceOfExpr(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e InstanceOfExpr) (Sequence, error) {
	seq, err := evalFn(ctx, ec, e.Expr)
	if err != nil {
		return nil, err
	}
	return SingleBoolean(matchesSequenceType(seq, e.Type, ec)), nil
}

// qnameCastLexical prepares the (lexical, nsMap) used to re-validate an
// ALREADY-RESOLVED QName VALUE against a schema-aware QName/NOTATION-derived cast
// target. The source value's prefix may be bound only on the instance node and
// absent from the assertion's STATIC namespace map, so re-serializing to
// `prefix:local` and re-resolving against the static map would fail even though
// the value is already valid. Instead it binds the lexical's prefix to the value's
// OWN namespace URI in a COPY of base, so schema validation resolves to the same
// URI. A no-namespace value keeps its bare local name. Used only inside the
// schema-aware cast fallback, so existing xpath3 cast behavior is unchanged.
func qnameCastLexical(qv QNameValue, base map[string]string) (string, map[string]string) {
	if qv.URI == "" {
		return qv.Local, base
	}
	ns := make(map[string]string, len(base)+1)
	maps.Copy(ns, base)
	prefix := qv.Prefix
	if prefix == "" {
		for i := 0; ; i++ {
			cand := fmt.Sprintf("ns%d", i)
			if _, taken := ns[cand]; !taken {
				prefix = cand
				break
			}
		}
	}
	ns[prefix] = qv.URI
	return prefix + ":" + qv.Local, ns
}

func schemaAwareCast(ctx context.Context, ec *evalContext, av AtomicValue, targetType string) (AtomicValue, error) {
	return schemaAwareCastRec(ctx, ec, av, targetType, make(map[string]struct{}))
}

func schemaAwareCastRec(ctx context.Context, ec *evalContext, av AtomicValue, targetType string, seen map[string]struct{}) (AtomicValue, error) {
	result, err := CastAtomic(av, targetType)
	if err == nil {
		return result, nil
	}
	if ec == nil || ec.schemaDeclarations == nil {
		return AtomicValue{}, err
	}
	if _, active := seen[targetType]; active {
		return AtomicValue{}, err
	}
	seen[targetType] = struct{}{}
	defer delete(seen, targetType)

	schemaErr := err
	if local, ns, ok := schemaAnnotationParts(targetType); ok {
		annName := QAnnotation(ns, local)
		if builtinBase := resolveToBuiltinBase(local, ns, ec.schemaDeclarations); builtinBase != "" {
			result, baseErr := schemaAwareCastViaBuiltin(ctx, ec, av, targetType, annName, builtinBase, err)
			if baseErr == nil {
				return result, nil
			}
			schemaErr = baseErr
		}
	}
	if members := ec.schemaDeclarations.UnionMemberTypes(targetType); len(members) > 0 {
		for _, memberType := range members {
			result, castErr := schemaAwareCastRec(ctx, ec, av, memberType, seen)
			if castErr == nil {
				// F&O 3.1 §19.3.5 allows already-typed values to cast to the
				// first castable atomic member of a union. Any facets on the target
				// union then apply to that resulting member value, not to the
				// original typed source lexical; string/untypedAtomic sources keep
				// the lexical casting path from §19.2.
				targetValue := av
				if av.TypeName != TypeString && av.TypeName != TypeUntypedAtomic {
					targetValue = result
				}
				if targetErr := schemaAwareValidateTarget(ctx, ec, targetValue, targetType); targetErr != nil {
					return AtomicValue{}, targetErr
				}
				return result, nil
			}
		}
	}
	return AtomicValue{}, schemaErr
}

func schemaAwareValidateTarget(ctx context.Context, ec *evalContext, av AtomicValue, targetType string) error {
	local, ns, ok := schemaAnnotationParts(targetType)
	if !ok {
		return nil
	}
	s, nsForCast := schemaAwareCastLexical(av, ec.namespaces)
	annName := QAnnotation(ns, local)
	if err := ec.schemaDeclarations.ValidateCastWithNS(ctx, s, annName, nsForCast); err != nil {
		return &XPathError{Code: errCodeFORG0001, Message: fmt.Sprintf("cannot cast %q to %s: %v", s, targetType, err)}
	}
	return nil
}

func schemaAwareCastLexical(av AtomicValue, base map[string]string) (string, map[string]string) {
	s, _ := AtomicToString(av)
	if qv, isQV := av.Value.(QNameValue); isQV {
		return qnameCastLexical(qv, base)
	}
	return s, base
}

func schemaAwareFacetLexical(src, cast AtomicValue, base map[string]string) (string, map[string]string) {
	if src.TypeName == TypeString || src.TypeName == TypeUntypedAtomic {
		return schemaAwareCastLexical(src, base)
	}
	return schemaAwareCastLexical(cast, base)
}

func schemaAwareCastViaBuiltin(ctx context.Context, ec *evalContext, av AtomicValue, targetType, annName, builtinBase string, fallback error) (AtomicValue, error) {
	if builtinBase == TypeNOTATION || builtinBase == TypeQName {
		_, isQV := av.Value.(QNameValue)
		srcOK := av.TypeName == TypeString || av.TypeName == TypeUntypedAtomic ||
			av.TypeName == TypeQName || av.TypeName == TypeNOTATION || isQV
		if !srcOK {
			return AtomicValue{}, fallback
		}
		s, nsForCast := schemaAwareCastLexical(av, ec.namespaces)
		if vErr := ec.schemaDeclarations.ValidateCastWithNS(ctx, s, annName, nsForCast); vErr != nil {
			return AtomicValue{}, &XPathError{Code: errCodeFORG0001, Message: fmt.Sprintf("cannot cast %q to %s: %v", s, targetType, vErr)}
		}
		qv, qErr := castToQName(av, ec)
		if qErr != nil {
			return AtomicValue{}, qErr
		}
		qv.BaseType = builtinBase
		qv.TypeName = targetType
		return qv, nil
	}
	result, castErr := CastAtomic(av, builtinBase)
	if castErr != nil {
		return AtomicValue{}, castErr
	}
	s, nsForCast := schemaAwareFacetLexical(av, result, ec.namespaces)
	if facetErr := ec.schemaDeclarations.ValidateCastWithNS(ctx, s, annName, nsForCast); facetErr != nil {
		return AtomicValue{}, &XPathError{Code: errCodeFORG0001, Message: fmt.Sprintf("cannot cast %q to %s: %v", s, targetType, facetErr)}
	}
	result.BaseType = builtinBase
	result.TypeName = targetType
	return result, nil
}

func evalCastExpr(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e CastExpr) (Sequence, error) {
	targetType := resolveAtomicTypeName(e.Type, ec)
	if isAbstractCastTarget(targetType) {
		return nil, &XPathError{
			Code:    errCodeXPST0080,
			Message: fmt.Sprintf("cannot use abstract type %s as cast target", targetType),
		}
	}
	seq, err := evalFn(ctx, ec, e.Expr)
	if err != nil {
		return nil, err
	}
	// Atomize THROUGH the stream (atomizeSingletonOperand) so a schema-typed node
	// whose typed value is a list/union expands to its atoms; the cast cardinality
	// (singleton, or empty when `as T?`) is then applied to the ATOMIZED result, not
	// the raw item count — a single node atomizing to >1 value is a cardinality error.
	atoms, err := atomizeSingletonOperand(seq)
	if err != nil {
		return nil, err
	}
	if len(atoms) == 0 {
		if e.AllowEmpty {
			return validNilSequence, nil
		}
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "cast requires non-empty sequence"}
	}
	if len(atoms) > 1 {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "cast requires singleton"}
	}
	av := atoms[0]
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
		if result, err = schemaAwareCast(ctx, ec, av, targetType); err != nil {
			return nil, err
		}
	}
	return SingleAtomic(result), nil
}

func evalCastableExpr(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e CastableExpr) (Sequence, error) {
	targetType := resolveAtomicTypeName(e.Type, ec)
	// Abstract types raise a static error even for castable (XPST0080)
	if isAbstractCastTarget(targetType) {
		return nil, &XPathError{
			Code:    errCodeXPST0080,
			Message: fmt.Sprintf("cannot use abstract type %s as castable target", targetType),
		}
	}
	seq, err := evalFn(ctx, ec, e.Expr)
	if err != nil {
		return nil, err
	}
	// Atomize THROUGH the stream so a schema-typed node whose typed value is a
	// list/union expands; castable's cardinality (singleton, or empty when `as T?`)
	// is applied to the ATOMIZED result — a node atomizing to >1 value is NOT
	// castable to a single atomic type.
	atoms, err := atomizeSingletonOperand(seq)
	if err != nil {
		return SingleBoolean(false), nil //nolint:nilerr // castable returns false on atomization failure
	}
	if len(atoms) == 0 {
		return SingleBoolean(e.AllowEmpty), nil
	}
	if len(atoms) > 1 {
		return SingleBoolean(false), nil
	}
	av := atoms[0]
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
	_, castErr := schemaAwareCast(ctx, ec, av, targetType)
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

func evalTreatAsExpr(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e TreatAsExpr) (Sequence, error) {
	seq, err := evalFn(ctx, ec, e.Expr)
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
	// Resolve via namespace context. ec may be nil when a function item's
	// coercion runs without a captured eval context; in that case the custom
	// prefix cannot be resolved, so fall through to the unresolved-type form
	// (Prefix:Name) which fails coercion with XPTY0004 rather than panicking.
	if ec != nil && ec.namespaces != nil {
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

// errCoerceMismatch is a sentinel returned by coerceToSequenceTypeE for a plain
// type or cardinality mismatch — one with no more-specific typed error. Callers
// map it to XPTY0004. A more-specific typed error (FOTY0013 from atomizing a
// function/map item, FORG0001 from an invalid cast, …) is returned directly so
// it can be surfaced to try/catch instead of being collapsed into XPTY0004.
var errCoerceMismatch = errors.New("xpath3: sequence type mismatch")

// CoerceToSequenceType applies XPath 3.1 function coercion rules (§3.1.5.2):
// atomization, numeric promotion (integer→float/double, float→double),
// URI-to-string promotion, and function coercion.  Returns the coerced
// sequence and true on success, or the original sequence and false on failure.
func CoerceToSequenceType(seq Sequence, st SequenceType) (Sequence, bool) {
	return coerceToSequenceType(seq, st, nil)
}

// coerceToSequenceType is the boolean-returning wrapper retained for callers that
// only need success/failure. It discards the specific error code; callers that
// must surface FOTY0013/FORG0001 should use coerceToSequenceTypeE.
func coerceToSequenceType(seq Sequence, st SequenceType, ec *evalContext) (Sequence, bool) {
	out, err := coerceToSequenceTypeE(seq, st, ec)
	if err != nil {
		return seq, false
	}
	return out, true
}

// coerceToSequenceTypeE is the error-propagating coercion. On a plain type or
// cardinality mismatch it returns errCoerceMismatch; on a typed atomization/cast
// failure it returns that underlying error (FOTY0013, FORG0001, …). Atomization
// errors take precedence over cardinality rejection, matching the spec ordering
// (atomization happens before the occurrence check).
func coerceToSequenceTypeE(seq Sequence, st SequenceType, ec *evalContext) (Sequence, error) {
	// XPath 1.0 compatibility mode: apply the 1.0 function-conversion rules
	// (first-item truncation; fn:string / fn:number coercion) before the ordinary
	// path. Reached only under XSLT backwards-compatible processing.
	if ec != nil && ec.xpath10Compat && !st.Void {
		newSeq, done, err := coerceXPath10Compat(seq, st, ec)
		if done {
			return newSeq, err
		}
		seq = newSeq
	}
	// Fast path: an item() item type matches any item, so per-item coercion is a
	// no-op. Only the cardinality (occurrence) constraint can reject. Check it
	// using seqLen — which is O(1) for every Sequence implementation (slice and
	// lazy Range alike) — instead of iterating the sequence through
	// matchesSequenceType. This preserves laziness for hot functions like
	// fn:count / fn:exists (item()* / item()+ parameters) so a large lazy range
	// (1 to N) is never materialized just to satisfy the signature gate.
	if !st.Void {
		if _, anyItem := st.ItemTest.(AnyItemTest); anyItem {
			n := seqLen(seq)
			switch st.Occurrence {
			case OccurrenceExactlyOne:
				if n != 1 {
					return seq, errCoerceMismatch
				}
			case OccurrenceZeroOrOne:
				if n > 1 {
					return seq, errCoerceMismatch
				}
			case OccurrenceOneOrMore:
				if n == 0 {
					return seq, errCoerceMismatch
				}
			case OccurrenceZeroOrMore:
				// any length ok
			}
			return seq, nil
		}
	}
	if matchesSequenceType(seq, st, ec) {
		return seq, nil
	}
	if seqLen(seq) == 1 {
		if fnTest, ok := st.ItemTest.(FunctionTest); ok {
			adapted, ok := coerceFunctionItem(seq.Get(0), fnTest, ec)
			if ok {
				return ItemSlice{adapted}, nil
			}
		}
	}
	// Try to coerce each item
	var targetType string
	switch t := st.ItemTest.(type) {
	case AtomicOrUnionType:
		targetType = resolveAtomicTypeName(AtomicTypeName(t), ec)
	default:
		return seq, errCoerceMismatch
	}
	// For a singleton/optional target occurrence the result may hold at most one
	// item, so cap atomization: stop as soon as a second atom appears rather than
	// materializing the whole (possibly huge) sequence before rejecting on
	// cardinality. A typed atomization error encountered before the cap still
	// propagates (atomization precedes the occurrence check).
	maxAtoms := 0
	switch st.Occurrence {
	case OccurrenceExactlyOne, OccurrenceZeroOrOne:
		maxAtoms = 1
	}
	// Atomize the sequence. atomizeStream correctly flattens arrays and
	// expands list-typed nodes (e.g. xs:list of xs:decimal) into multiple
	// atomic values — a per-item AtomizeItem would collapse those incorrectly.
	var atoms []AtomicValue
	tooMany := false
	err := atomizeStream(seq, func(av AtomicValue) (bool, error) {
		atoms = append(atoms, av)
		if maxAtoms > 0 && len(atoms) > maxAtoms {
			tooMany = true
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return seq, err
	}
	if tooMany {
		// More atoms than the occurrence indicator permits: cardinality mismatch.
		return seq, errCoerceMismatch
	}
	result := make(ItemSlice, len(atoms))
	i := 0
	for _, av := range atoms {
		// xs:anyAtomicType is the generalized atomic supertype: once a node has
		// been atomized (to xs:untypedAtomic), any atomic value already matches.
		// It is abstract, so it has no concrete cast — accept the atom as-is
		// rather than attempting an (impossible) cast to it.
		if targetType == TypeAnyAtomicType {
			result[i] = av
			i++
			continue
		}
		// Numeric promotion. The acceptance condition mirrors atomicMatchesTargetType
		// so the cast that follows always agrees with the gate: any numeric the gate
		// admits (built-in numerics, integer-derived subtypes, and schema-derived
		// values whose built-in base/ancestry is numeric) is normalized via
		// PromoteSchemaType before the cast, so castToDouble/castToFloat operate on a
		// built-in-typed value instead of failing on an unrecognized schema type.
		switch targetType {
		case TypeDouble:
			if atomicMatchesTargetType(av, TypeDouble, ec) || atomicMatchesTargetType(av, TypeFloat, ec) ||
				atomicMatchesTargetType(av, TypeInteger, ec) || atomicMatchesTargetType(av, TypeDecimal, ec) {
				promoted, err := castToDouble(PromoteSchemaType(av))
				if err != nil {
					return seq, err
				}
				result[i] = promoted
				i++
				continue
			}
		case TypeFloat:
			if atomicMatchesTargetType(av, TypeInteger, ec) || atomicMatchesTargetType(av, TypeDecimal, ec) {
				promoted, err := castToFloat(PromoteSchemaType(av))
				if err != nil {
					return seq, err
				}
				result[i] = promoted
				i++
				continue
			}
		case TypeString:
			// xs:anyURI (and its subtypes) promote to xs:string per the
			// function-conversion rules. Match via atomicMatchesTargetType so a
			// schema-DERIVED anyURI (carrying BaseType xs:anyURI, or whose
			// ancestry the schema knows) is admitted, not only the exact
			// xs:anyURI type. PromoteSchemaType normalizes the schema-derived
			// value to a built-in-typed anyURI so its string value is read the
			// same way the builtins read it.
			if atomicMatchesTargetType(av, TypeAnyURI, ec) {
				promoted := PromoteSchemaType(av)
				result[i] = AtomicValue{TypeName: TypeString, Value: promoted.Value}
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
				return seq, err
			}
			result[i] = cast
			i++
			continue
		}
		// Accept when the value matches the target by built-in subtype hierarchy,
		// its built-in BaseType (schema-derived values the builtins promote via
		// PromoteSchemaType), or schema-declaration ancestry (incl. the xs:numeric
		// union). atomicMatchesTargetType is the shared gate matchesItemType uses.
		if atomicMatchesTargetType(av, targetType, ec) {
			result[i] = av
			i++
			continue
		}
		return seq, errCoerceMismatch
	}
	// Verify occurrence constraint on the coerced result
	switch st.Occurrence {
	case OccurrenceExactlyOne:
		if len(result) != 1 {
			return seq, errCoerceMismatch
		}
	case OccurrenceOneOrMore:
		if len(result) == 0 {
			return seq, errCoerceMismatch
		}
	case OccurrenceZeroOrOne:
		if len(result) > 1 {
			return seq, errCoerceMismatch
		}
	}
	return result, nil
}

func coerceFunctionItem(item Item, target FunctionTest, ec *evalContext) (Item, bool) {
	if target.AnyFunction {
		return nil, false
	}

	// Maps and arrays are function items of arity 1 per XPath 3.1, so they
	// participate in function coercion just like an inline/named function.
	actual, ok := asFunctionItem(item)
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
					Code:    lexicon.ErrXPTY0004,
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
							Code:    lexicon.ErrXPTY0004,
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
					Code:    lexicon.ErrXPTY0004,
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

// atomicMatchesTargetType reports whether an atomic value satisfies an
// AtomicOrUnionType item test whose resolved name is targetType. Beyond the
// built-in subtype hierarchy and schema-declaration ancestry, it honors the
// value's own BaseType: a schema-derived value whose built-in BaseType is a
// subtype of (or promotes to) the target must match, because the numeric (and
// other) builtins accept it via PromoteSchemaType. Without this the static
// signature gate would reject, e.g., abs($v) for $v of a custom type derived
// from xs:decimal even though fnAbs promotes and handles it.
func atomicMatchesTargetType(av AtomicValue, targetType string, ec *evalContext) bool {
	if targetType == TypeAnyAtomicType {
		return true
	}
	if isSubtypeOf(av.TypeName, targetType) {
		return true
	}
	// Schema-derived value carrying a built-in BaseType: match using the base
	// type, mirroring PromoteSchemaType (which the builtins use). Only built-in
	// base types are trusted here so acceptance is not broadened for unrelated
	// user types.
	if av.BaseType != "" && IsKnownXSDType(av.BaseType) && isSubtypeOf(av.BaseType, targetType) {
		return true
	}
	// Fall back to schema declarations for user-defined types whose ancestry is
	// only known to the compiled schema.
	if ec != nil && ec.schemaDeclarations != nil {
		// xs:numeric is a synthetic union the schema layer does not model, so
		// resolve it against its member built-in types.
		if targetType == TypeNumeric {
			return ec.schemaDeclarations.IsSubtypeOf(av.TypeName, TypeDecimal) ||
				ec.schemaDeclarations.IsSubtypeOf(av.TypeName, TypeFloat) ||
				ec.schemaDeclarations.IsSubtypeOf(av.TypeName, TypeDouble)
		}
		return ec.schemaDeclarations.IsSubtypeOf(av.TypeName, targetType)
	}
	return false
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
		local, ns := resolveSchemaTestName(t.Name, ec, false)
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
		local, ns := resolveSchemaTestName(t.Name, ec, true)
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
		return atomicMatchesTargetType(av, targetType, ec)
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
			_ = m.forEach0(func(k AtomicValue, val Sequence) error {
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
		_ = m.forEach0(func(k AtomicValue, v Sequence) error {
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
			Code:    lexicon.ErrXPTY0004,
			Message: fmt.Sprintf("cannot cast %s to %s", v.TypeName, TypeQName),
		}
	}

	prefix, local, _, validNC := domutil.SplitLexicalQName(v.StringVal())
	if !validNC {
		return AtomicValue{}, &XPathError{
			Code:    errCodeFORG0001,
			Message: fmt.Sprintf("invalid QName: %q", strings.TrimSpace(v.StringVal())),
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
		// No prefix: check default namespace in context. Under XSD value-space
		// semantics (qnameValueNoDefaultNS), an unprefixed QName/NOTATION VALUE has
		// no namespace — the XPath default element namespace is NOT applied (opt-in;
		// default xpath3 behavior is unchanged).
		if ec != nil && ec.namespaces != nil && !ec.qnameValueNoDefaultNS {
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
