package xpath3

import (
	"context"
	"fmt"
	"strings"
)

// Function is the interface for XPath 3.1 functions, both built-in and user-defined.
type Function interface {
	MinArity() int
	MaxArity() int // -1 = variadic
	Call(ctx context.Context, args []Sequence) (Sequence, error)
}

// Namespace URIs for standard XPath 3.1 function namespaces.
const (
	NSFn    = "http://www.w3.org/2005/xpath-functions"
	NSMath  = "http://www.w3.org/2005/xpath-functions/math"
	NSMap   = "http://www.w3.org/2005/xpath-functions/map"
	NSArray = "http://www.w3.org/2005/xpath-functions/array"
	NSErr   = "http://www.w3.org/2005/xqt-errors"
	NSXS    = "http://www.w3.org/2001/XMLSchema"
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
	uri := resolvePrefix(ec, prefix)

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

	// If no explicit URI matched, try fn: namespace as default
	if uri != NSFn {
		qn = QualifiedName{URI: NSFn, Name: name}
		if fn, ok := builtinFunctions3[qn]; ok {
			if err := checkArity(fn, name, arity); err != nil {
				return nil, err
			}
			return fn, nil
		}
	}

	return nil, fmt.Errorf("%w: %s#%d", ErrUnknownFunction, name, arity)
}

func resolvePrefix(ec *evalContext, prefix string) string {
	if prefix == "" {
		return NSFn
	}
	// Check user-provided namespace bindings
	if ec.namespaces != nil {
		if uri, ok := ec.namespaces[prefix]; ok {
			return uri
		}
	}
	// Check default prefix mappings
	if uri, ok := defaultPrefixNS[prefix]; ok {
		return uri
	}
	return prefix
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

// seqToString atomizes the first item to a string, or returns "".
func seqToString(seq Sequence) string {
	if len(seq) == 0 {
		return ""
	}
	a, err := AtomizeItem(seq[0])
	if err != nil {
		return ""
	}
	s, _ := atomicToString(a)
	return s
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

// coerceArgToString applies XPath 3.1 function coercion rules for xs:string? params.
// Accepts: empty sequence → "", xs:string/xs:anyURI → as-is, xs:untypedAtomic → cast.
// Rejects all other types with XPTY0004.
func coerceArgToString(seq Sequence) (string, error) {
	if len(seq) == 0 {
		return "", nil
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
		return "", &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("expected xs:string?, got %s", a.TypeName)}
	}
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
