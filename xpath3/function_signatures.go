package xpath3

import "strings"

// functionSignature stores the declared parameter and return types for a built-in function.
type functionSignature struct {
	ParamTypes []SequenceType
	ReturnType *SequenceType
}

type sigKey struct {
	URI   string
	Name  string
	Arity int
}

// builtinSignatures maps (namespace, name, arity) → type signature for built-in functions.
// This is used by evalNamedFunctionRef to populate FunctionItem.ParamTypes/ReturnType.
var builtinSignatures = map[sigKey]functionSignature{}

func registerSig(name string, arity int, params []SequenceType, ret SequenceType) {
	builtinSignatures[sigKey{URI: NSFn, Name: name, Arity: arity}] = functionSignature{
		ParamTypes: params,
		ReturnType: &ret,
	}
}

// seqType helpers for concise signature definitions
func stAtomic(typeName string, occ Occurrence) SequenceType {
	// typeName is like "xs:numeric" — split into prefix and name
	prefix := "xs"
	name := typeName
	if idx := strings.IndexByte(typeName, ':'); idx >= 0 {
		prefix = typeName[:idx]
		name = typeName[idx+1:]
	}
	return SequenceType{ItemTest: AtomicOrUnionType{Prefix: prefix, Name: name}, Occurrence: occ}
}

func stNode(occ Occurrence) SequenceType {
	return SequenceType{ItemTest: TypeTest{Kind: NodeKindNode}, Occurrence: occ}
}

func stItem(occ Occurrence) SequenceType {
	return SequenceType{ItemTest: AnyItemTest{}, Occurrence: occ}
}

func stFunc(params []SequenceType, ret SequenceType) SequenceType {
	return SequenceType{
		ItemTest: FunctionTest{
			ParamTypes: params,
			ReturnType: ret,
		},
		Occurrence: OccurrenceExactlyOne,
	}
}

func init() {
	// Numeric functions — (xs:numeric?) as xs:numeric?
	numQQ := []SequenceType{stAtomic(TypeNumeric, OccurrenceZeroOrOne)}
	retNumQ := stAtomic(TypeNumeric, OccurrenceZeroOrOne)
	for _, name := range []string{"abs", "ceiling", "floor"} {
		registerSig(name, 1, numQQ, retNumQ)
	}
	registerSig("round", 1, numQQ, retNumQ)
	registerSig("round", 2, []SequenceType{stAtomic(TypeNumeric, OccurrenceZeroOrOne), stAtomic(TypeInteger, OccurrenceExactlyOne)}, retNumQ)
	registerSig("round-half-to-even", 1, numQQ, retNumQ)
	registerSig("round-half-to-even", 2, []SequenceType{stAtomic(TypeNumeric, OccurrenceZeroOrOne), stAtomic(TypeInteger, OccurrenceExactlyOne)}, retNumQ)

	// Node functions
	registerSig("name", 0, nil, stAtomic(TypeString, OccurrenceExactlyOne))
	registerSig("name", 1, []SequenceType{stNode(OccurrenceZeroOrOne)}, stAtomic(TypeString, OccurrenceExactlyOne))
	registerSig("local-name", 0, nil, stAtomic(TypeString, OccurrenceExactlyOne))
	registerSig("local-name", 1, []SequenceType{stNode(OccurrenceZeroOrOne)}, stAtomic(TypeString, OccurrenceExactlyOne))
	registerSig("namespace-uri", 0, nil, stAtomic(TypeAnyURI, OccurrenceExactlyOne))
	registerSig("namespace-uri", 1, []SequenceType{stNode(OccurrenceZeroOrOne)}, stAtomic(TypeAnyURI, OccurrenceExactlyOne))
	registerSig("node-name", 0, nil, stAtomic(TypeQName, OccurrenceZeroOrOne))
	registerSig("node-name", 1, []SequenceType{stNode(OccurrenceZeroOrOne)}, stAtomic(TypeQName, OccurrenceZeroOrOne))

	// String functions
	registerSig("string", 0, nil, stAtomic(TypeString, OccurrenceExactlyOne))
	registerSig("string", 1, []SequenceType{stItem(OccurrenceZeroOrOne)}, stAtomic(TypeString, OccurrenceExactlyOne))
	registerSig("string-length", 0, nil, stAtomic(TypeInteger, OccurrenceExactlyOne))
	registerSig("string-length", 1, []SequenceType{stAtomic(TypeString, OccurrenceZeroOrOne)}, stAtomic(TypeInteger, OccurrenceExactlyOne))
	registerSig("concat", 2, []SequenceType{stItem(OccurrenceZeroOrMore), stItem(OccurrenceZeroOrMore)}, stAtomic(TypeString, OccurrenceExactlyOne))
	registerSig("contains", 2, []SequenceType{stAtomic(TypeString, OccurrenceZeroOrOne), stAtomic(TypeString, OccurrenceZeroOrOne)}, stAtomic(TypeBoolean, OccurrenceExactlyOne))

	// Boolean functions
	registerSig("boolean", 1, []SequenceType{stItem(OccurrenceZeroOrMore)}, stAtomic(TypeBoolean, OccurrenceExactlyOne))
	registerSig("not", 1, []SequenceType{stItem(OccurrenceZeroOrMore)}, stAtomic(TypeBoolean, OccurrenceExactlyOne))
	registerSig("true", 0, nil, stAtomic(TypeBoolean, OccurrenceExactlyOne))
	registerSig("false", 0, nil, stAtomic(TypeBoolean, OccurrenceExactlyOne))

	// Aggregate functions
	registerSig("count", 1, []SequenceType{stItem(OccurrenceZeroOrMore)}, stAtomic(TypeInteger, OccurrenceExactlyOne))
	registerSig("sum", 1, []SequenceType{stItem(OccurrenceZeroOrMore)}, stItem(OccurrenceExactlyOne))
	registerSig("sum", 2, []SequenceType{stItem(OccurrenceZeroOrMore), stItem(OccurrenceZeroOrMore)}, stItem(OccurrenceZeroOrMore))

	// Sequence functions
	registerSig("empty", 1, []SequenceType{stItem(OccurrenceZeroOrMore)}, stAtomic(TypeBoolean, OccurrenceExactlyOne))
	registerSig("exists", 1, []SequenceType{stItem(OccurrenceZeroOrMore)}, stAtomic(TypeBoolean, OccurrenceExactlyOne))

	// Higher-order functions
	registerSig("filter", 2, []SequenceType{
		stItem(OccurrenceZeroOrMore),
		stFunc([]SequenceType{stItem(OccurrenceExactlyOne)}, stAtomic(TypeBoolean, OccurrenceExactlyOne)),
	}, stItem(OccurrenceZeroOrMore))
	registerSig("for-each", 2, []SequenceType{
		stItem(OccurrenceZeroOrMore),
		stFunc([]SequenceType{stItem(OccurrenceExactlyOne)}, stItem(OccurrenceZeroOrMore)),
	}, stItem(OccurrenceZeroOrMore))
	registerSig("for-each-pair", 3, []SequenceType{
		stItem(OccurrenceZeroOrMore),
		stItem(OccurrenceZeroOrMore),
		stFunc([]SequenceType{stItem(OccurrenceExactlyOne), stItem(OccurrenceExactlyOne)}, stItem(OccurrenceZeroOrMore)),
	}, stItem(OccurrenceZeroOrMore))
	registerSig("fold-left", 3, []SequenceType{
		stItem(OccurrenceZeroOrMore),
		stItem(OccurrenceZeroOrMore),
		stFunc([]SequenceType{stItem(OccurrenceZeroOrMore), stItem(OccurrenceExactlyOne)}, stItem(OccurrenceZeroOrMore)),
	}, stItem(OccurrenceZeroOrMore))
	registerSig("fold-right", 3, []SequenceType{
		stItem(OccurrenceZeroOrMore),
		stItem(OccurrenceZeroOrMore),
		stFunc([]SequenceType{stItem(OccurrenceExactlyOne), stItem(OccurrenceZeroOrMore)}, stItem(OccurrenceZeroOrMore)),
	}, stItem(OccurrenceZeroOrMore))
	registerSig("sort", 2, []SequenceType{
		stItem(OccurrenceZeroOrMore),
		stFunc([]SequenceType{stItem(OccurrenceExactlyOne)}, stItem(OccurrenceZeroOrMore)),
	}, stItem(OccurrenceZeroOrMore))
}

// lookupFunctionSignature returns the type signature for a built-in function.
func lookupFunctionSignature(uri, name string, arity int) *functionSignature {
	sig, ok := builtinSignatures[sigKey{URI: uri, Name: name, Arity: arity}]
	if !ok {
		return nil
	}
	return &sig
}
