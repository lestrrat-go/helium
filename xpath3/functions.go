package xpath3

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/lestrrat-go/helium/internal/catalog"
)

// Function is the interface for XPath 3.1 functions, both built-in and user-defined.
type Function interface {
	MinArity() int
	MaxArity() int // -1 = variadic
	Call(ctx context.Context, args []Sequence) (Sequence, error)
}

// Namespace URIs for standard XPath 3.1 function namespaces.
const (
	NSFn    = catalog.Fn
	NSMath  = catalog.Math
	NSMap   = catalog.Map
	NSArray = catalog.Array
	NSErr   = catalog.Err
	NSXS    = catalog.XSD
)

// Default prefix → URI mappings.
var defaultPrefixNS = map[string]string{
	"fn":    NSFn,
	"math":  NSMath,
	"map":   NSMap,
	"array": NSArray,
	"err":   NSErr,
	"xs":    NSXS,
}

// namespacePrefixFor returns the conventional prefix for a known namespace URI.
func namespacePrefixFor(uri string) string {
	for prefix, ns := range defaultPrefixNS {
		if ns == uri {
			return prefix
		}
	}
	return ""
}

// builtinFunctions3 is the package-level registry of built-in functions.
// Populated in Phase 4 init() calls. Keyed by QualifiedName{URI, Name}.
var builtinFunctions3 = map[QualifiedName]Function{}

// resolveFunction finds a function by prefix, local name, and arity.
// Resolution order:
//  1. User-registered functions by local name (ec.functions)
//  2. User-registered functions by QualifiedName (ec.fnsNS)
//  3. Built-in functions by QualifiedName (builtinFunctions3)
//  4. Default fn: namespace (no prefix = fn:)
func resolveFunction(ec *evalContext, prefix, name string, arity int) (Function, error) {
	// Handle Q{uri}local URIQualifiedName syntax from the lexer
	if prefix == "" && strings.HasPrefix(name, "Q{") {
		if idx := strings.Index(name, "}"); idx >= 0 {
			uri := name[2:idx]
			local := name[idx+1:]
			return resolveFunctionByURI(ec, uri, local, arity)
		}
	}

	// 1. User functions by local name (prefix-less or matching)
	if prefix == "" && ec.functions != nil {
		if fn, ok := ec.functions[name]; ok {
			if err := checkArity(fn, name, arity); err != nil {
				return nil, err
			}
			return fn, nil
		}
	}

	// Resolve prefix to namespace URI
	uri, err := resolvePrefix(ec, prefix)
	if err != nil {
		return nil, err
	}

	return resolveFunctionByURI(ec, uri, name, arity)
}

func resolveFunctionByURI(ec *evalContext, uri, name string, arity int) (Function, error) {
	// User functions by qualified name
	if ec.fnsNS != nil {
		qn := QualifiedName{URI: uri, Name: name}
		if fn, ok := ec.fnsNS[qn]; ok {
			if err := checkArity(fn, name, arity); err != nil {
				return nil, err
			}
			return fn, nil
		}
	}

	// Built-in functions
	qn := QualifiedName{URI: uri, Name: name}
	if fn, ok := builtinFunctions3[qn]; ok {
		if err := checkArity(fn, name, arity); err != nil {
			return nil, err
		}
		return fn, nil
	}

	return nil, fmt.Errorf("%w: %s#%d", ErrUnknownFunction, name, arity)
}

func resolvePrefix(ec *evalContext, prefix string) (string, error) {
	if prefix == "" {
		return NSFn, nil
	}
	// Check user-provided namespace bindings
	if ec.namespaces != nil {
		if uri, ok := ec.namespaces[prefix]; ok {
			return uri, nil
		}
	}
	// Check default prefix mappings
	if uri, ok := defaultPrefixNS[prefix]; ok {
		return uri, nil
	}
	return "", &XPathError{Code: errCodeFONS0004, Message: "undeclared namespace prefix: " + prefix}
}

func checkArity(fn Function, name string, arity int) error {
	min := fn.MinArity()
	max := fn.MaxArity()
	if arity < min {
		return fmt.Errorf("%w: %s requires at least %d arguments, got %d", ErrArityMismatch, name, min, arity)
	}
	if max >= 0 && arity > max {
		return fmt.Errorf("%w: %s accepts at most %d arguments, got %d", ErrArityMismatch, name, max, arity)
	}
	return nil
}

// builtinFunc is a simple implementation of Function for built-in functions.
type builtinFunc struct {
	name string
	min  int
	max  int // -1 = variadic
	fn   func(ctx context.Context, args []Sequence) (Sequence, error)
}

func (f *builtinFunc) MinArity() int { return f.min }
func (f *builtinFunc) MaxArity() int { return f.max }
func (f *builtinFunc) Call(ctx context.Context, args []Sequence) (Sequence, error) {
	return f.fn(ctx, args)
}

// IsBuiltinFunction returns true if name is a registered XPath built-in function.
func IsBuiltinFunction(name string) bool {
	_, ok := builtinFunctions3[QualifiedName{URI: NSFn, Name: name}]
	return ok
}

// IsBuiltinFunctionNS returns true if name in the given namespace is a registered built-in function.
func IsBuiltinFunctionNS(uri, name string) bool {
	_, ok := builtinFunctions3[QualifiedName{URI: uri, Name: name}]
	return ok
}

// BuiltinFunctionAcceptsArity returns true if a built-in function accepts
// the given arity.
func BuiltinFunctionAcceptsArity(uri, name string, arity int) bool {
	fn, ok := builtinFunctions3[QualifiedName{URI: uri, Name: name}]
	if !ok {
		return false
	}
	return arity >= fn.MinArity() && (fn.MaxArity() < 0 || arity <= fn.MaxArity())
}

// registerFn is a convenience for registering a built-in function in the fn: namespace.
func registerFn(name string, min, max int, fn func(context.Context, []Sequence) (Sequence, error)) {
	builtinFunctions3[QualifiedName{URI: NSFn, Name: name}] = &builtinFunc{
		name: name, min: min, max: max, fn: fn,
	}
}

// registerNS is a convenience for registering a built-in function in a specific namespace.
func registerNS(uri, name string, min, max int, fn func(context.Context, []Sequence) (Sequence, error)) {
	builtinFunctions3[QualifiedName{URI: uri, Name: name}] = &builtinFunc{
		name: name, min: min, max: max, fn: fn,
	}
}

// seqToStringErr atomizes the first item to a string, propagating errors.
func seqToStringErr(seq Sequence) (string, error) {
	if len(seq) == 0 {
		return "", nil
	}
	a, err := AtomizeItem(seq[0])
	if err != nil {
		return "", err
	}
	return atomicToString(a)
}

// coerceArgToStringRequired applies XPath 3.1 function coercion rules for xs:string params.
// Like coerceArgToString but rejects empty sequences (for non-optional string parameters).
func coerceArgToStringRequired(seq Sequence) (string, error) {
	if len(seq) == 0 {
		return "", &XPathError{Code: errCodeXPTY0004, Message: "expected xs:string, got empty sequence"}
	}
	return coerceArgToString(seq)
}

// coerceArgToString applies XPath 3.1 function coercion rules for xs:string? params.
// Accepts: empty sequence → "", xs:string/xs:anyURI → as-is, xs:untypedAtomic → cast.
// Rejects all other types with XPTY0004.
func coerceArgToString(seq Sequence) (string, error) {
	switch len(seq) {
	case 0:
		return "", nil
	case 1:
	default:
		return "", &XPathError{Code: errCodeXPTY0004, Message: "expected xs:string?, got sequence of length > 1"}
	}
	a, err := AtomizeItem(seq[0])
	if err != nil {
		return "", err
	}
	switch a.TypeName {
	case TypeString, TypeAnyURI, TypeUntypedAtomic,
		TypeNormalizedString, TypeToken, TypeLanguage, TypeName, TypeNCName,
		TypeNMTOKEN, TypeNMTOKENS, TypeENTITY, TypeID, TypeIDREF, TypeIDREFS:
		s, ok := a.Value.(string)
		if !ok {
			return "", fmt.Errorf("xpath3: internal error: expected string for %s", a.TypeName)
		}
		return s, nil
	default:
		// User-defined types: check if the underlying value is a string.
		if s, ok := a.Value.(string); ok {
			return s, nil
		}
		return "", &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("expected xs:string?, got %s", a.TypeName)}
	}
}

// coerceArgToInteger applies XPath 3.1 function coercion rules for xs:integer params.
// Accepts: xs:integer (and subtypes), xs:untypedAtomic (cast to integer).
// Rejects all other types with XPTY0004.
func coerceArgToInteger(seq Sequence) (int64, error) {
	switch len(seq) {
	case 0:
		return 0, &XPathError{Code: errCodeXPTY0004, Message: "expected xs:integer, got empty sequence"}
	case 1:
	default:
		return 0, &XPathError{Code: errCodeXPTY0004, Message: "expected xs:integer, got sequence of length > 1"}
	}
	a, err := AtomizeItem(seq[0])
	if err != nil {
		return 0, err
	}
	if a.TypeName == TypeUntypedAtomic {
		casted, cerr := castToInteger(a)
		if cerr != nil {
			return 0, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("cannot cast %s to xs:integer", a.TypeName)}
		}
		a = casted
	}
	if !isSubtypeOf(a.TypeName, TypeInteger) {
		return 0, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("expected xs:integer, got %s", a.TypeName)}
	}
	n, ok := a.Value.(*big.Int)
	if !ok {
		return 0, fmt.Errorf("xpath3: internal error: expected *big.Int for %s", a.TypeName)
	}
	return n.Int64(), nil
}

// coerceArgToDoubleRequired applies XPath 3.1 function coercion rules for xs:double params.
// Accepts any numeric type and xs:untypedAtomic after casting. Rejects empty and multi-item sequences.
func coerceArgToDoubleRequired(seq Sequence) (float64, error) {
	a, err := extractSingleAtomicArg(seq, "xs:double")
	if err != nil {
		if xpErr, ok := err.(*XPathError); ok {
			switch {
			case strings.Contains(xpErr.Message, "empty sequence"):
				return 0, &XPathError{Code: errCodeXPTY0004, Message: "expected xs:double, got empty sequence"}
			case strings.Contains(xpErr.Message, "length > 1"):
				return 0, &XPathError{Code: errCodeXPTY0004, Message: "expected xs:double, got sequence of length > 1"}
			}
		}
		return 0, err
	}
	if a.TypeName == TypeUntypedAtomic {
		casted, cerr := CastAtomic(a, TypeDouble)
		if cerr != nil {
			return 0, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("cannot cast %s to xs:double", a.TypeName)}
		}
		a = casted
	}
	if !isSubtypeOf(a.TypeName, TypeNumeric) {
		return 0, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("expected xs:double, got %s", a.TypeName)}
	}
	return a.ToFloat64(), nil
}

// extractSingleAtomicArg enforces that seq contains exactly one item and atomizes it.
// Used for function parameters typed as xs:anyAtomicType (not optional).
func extractSingleAtomicArg(seq Sequence, fnName string) (AtomicValue, error) {
	switch len(seq) {
	case 0:
		return AtomicValue{}, &XPathError{Code: errCodeXPTY0004, Message: fnName + ": expected single atomic value, got empty sequence"}
	case 1:
		return AtomizeItem(seq[0])
	default:
		return AtomicValue{}, &XPathError{Code: errCodeXPTY0004, Message: fnName + ": expected single atomic value, got sequence of length > 1"}
	}
}

// coerceToInteger applies function coercion rules for xs:integer to a single AtomicValue.
// Accepts: xs:integer (and subtypes), xs:untypedAtomic (cast to integer).
// Rejects double, float, decimal, and all other types with XPTY0004.
func coerceToInteger(a AtomicValue) (AtomicValue, error) {
	if a.TypeName == TypeUntypedAtomic {
		casted, cerr := castToInteger(a)
		if cerr != nil {
			return AtomicValue{}, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("cannot cast %s to xs:integer", a.TypeName)}
		}
		return casted, nil
	}
	if !isSubtypeOf(a.TypeName, TypeInteger) {
		return AtomicValue{}, &XPathError{Code: errCodeXPTY0004, Message: fmt.Sprintf("expected xs:integer, got %s", a.TypeName)}
	}
	return a, nil
}

// seqToDouble atomizes the first item to a float64.
func seqToDouble(seq Sequence) float64 {
	if len(seq) == 0 {
		return 0
	}
	a, err := AtomizeItem(seq[0])
	if err != nil {
		return 0
	}
	return a.ToFloat64()
}
