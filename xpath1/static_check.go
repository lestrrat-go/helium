package xpath1

import (
	"fmt"

	"github.com/lestrrat-go/helium/internal/lexicon"
)

func validateStaticExpr(expr Expr, cfg *evalConfig, depth int) error {
	depth++
	if depth > maxRecursionDepth {
		return ErrRecursionLimit
	}

	switch e := expr.(type) {
	case *LocationPath:
		return validateStaticLocationPath(e, cfg, depth)
	case BinaryExpr:
		if err := validateStaticExpr(e.Left, cfg, depth); err != nil {
			return err
		}
		return validateStaticExpr(e.Right, cfg, depth)
	case UnaryExpr:
		return validateStaticExpr(e.Operand, cfg, depth)
	case LiteralExpr, NumberExpr:
		return nil
	case VariableExpr:
		if cfg != nil && cfg.variables != nil {
			if _, ok := cfg.variables[e.Name]; ok {
				return nil
			}
		}
		return fmt.Errorf("%w: $%s", ErrUndefinedVariable, e.Name)
	case FunctionCall:
		if err := validateStaticFunction(e, cfg); err != nil {
			return err
		}
		for _, arg := range e.Args {
			if err := validateStaticExpr(arg, cfg, depth); err != nil {
				return err
			}
		}
		return nil
	case FilterExpr:
		if err := validateStaticExpr(e.Expr, cfg, depth); err != nil {
			return err
		}
		return validateStaticPredicates(e.Predicates, cfg, depth)
	case UnionExpr:
		if err := validateStaticExpr(e.Left, cfg, depth); err != nil {
			return err
		}
		return validateStaticExpr(e.Right, cfg, depth)
	case PathExpr:
		if err := validateStaticExpr(e.Filter, cfg, depth); err != nil {
			return err
		}
		return validateStaticLocationPath(e.Path, cfg, depth)
	default:
		return fmt.Errorf("%w: %T", ErrUnsupportedExpr, expr)
	}
}

func validateStaticLocationPath(path *LocationPath, cfg *evalConfig, depth int) error {
	if path == nil {
		return nil
	}
	for _, step := range path.Steps {
		if test, ok := step.NodeTest.(NameTest); ok && test.Prefix != "" {
			if !staticNameTestPrefixDefined(test.Prefix, cfg) {
				return fmt.Errorf("%w: %s", ErrUnknownNamespacePrefix, test.Prefix)
			}
		}
		if err := validateStaticPredicates(step.Predicates, cfg, depth); err != nil {
			return err
		}
	}
	return nil
}

func validateStaticPredicates(predicates []Expr, cfg *evalConfig, depth int) error {
	for _, predicate := range predicates {
		if err := validateStaticExpr(predicate, cfg, depth); err != nil {
			return err
		}
	}
	return nil
}

func staticNameTestPrefixDefined(prefix string, cfg *evalConfig) bool {
	if cfg != nil && cfg.namespaces != nil {
		if _, ok := cfg.namespaces[prefix]; ok {
			return true
		}
	}
	return prefix == lexicon.PrefixXML
}

func validateStaticFunction(call FunctionCall, cfg *evalConfig) error {
	if call.Prefix == "" {
		if _, ok := builtinFunctions[call.Name]; ok {
			return nil
		}
		if cfg != nil && cfg.functions != nil && cfg.functions[call.Name] != nil {
			return nil
		}
		return fmt.Errorf("%w: %s", ErrUnknownFunction, call.Name)
	}

	if cfg == nil || cfg.namespaces == nil {
		return fmt.Errorf("%w: %s", ErrUnknownFunctionNamespace, call.Prefix)
	}
	uri, ok := cfg.namespaces[call.Prefix]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownFunctionNamespace, call.Prefix)
	}
	if cfg.functionsNS != nil && cfg.functionsNS[QualifiedName{URI: uri, Name: call.Name}] != nil {
		return nil
	}
	return fmt.Errorf("%w: {%s}%s", ErrUnknownFunction, uri, call.Name)
}
