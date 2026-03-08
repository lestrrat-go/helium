package xpath3

import (
	"context"
	"fmt"

	helium "github.com/lestrrat-go/helium"
)

// Function is the interface for XPath 3.1 functions, both built-in and user-defined.
type Function interface {
	MinArity() int
	MaxArity() int // -1 = variadic
	Call(ctx context.Context, args []Sequence) (Sequence, error)
}

// FunctionContext provides evaluation context to functions that need it.
type FunctionContext interface {
	Node() helium.Node
	Position() int
	Size() int
	Namespace(prefix string) (string, bool)
	Variable(name string) (Sequence, bool)
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

	// 2. User functions by qualified name
	if ec.fnsNS != nil {
		qn := QualifiedName{URI: uri, Name: name}
		if fn, ok := ec.fnsNS[qn]; ok {
			if err := checkArity(fn, name, arity); err != nil {
				return nil, err
			}
			return fn, nil
		}
	}

	// 3. Built-in functions
	qn := QualifiedName{URI: uri, Name: name}
	if fn, ok := builtinFunctions3[qn]; ok {
		if err := checkArity(fn, name, arity); err != nil {
			return nil, err
		}
		return fn, nil
	}

	// 4. If no prefix, try fn: namespace
	if prefix == "" && uri != NSFn {
		qn = QualifiedName{URI: NSFn, Name: name}
		if fn, ok := builtinFunctions3[qn]; ok {
			if err := checkArity(fn, name, arity); err != nil {
				return nil, err
			}
			return fn, nil
		}
	}

	if prefix != "" {
		return nil, fmt.Errorf("%w: %s:%s#%d", ErrUnknownFunction, prefix, name, arity)
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
