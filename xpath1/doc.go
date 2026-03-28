// Package xpath1 implements XPath 1.0 expression parsing and evaluation
// against helium XML document trees.
//
// # Quick Start
//
// For one-off queries, use the convenience functions [Find] and [Evaluate]:
//
//	nodes, err := xpath1.Find(ctx, doc, "//title")
//
// # Compilation and Evaluation
//
// For repeated evaluation, compile once and evaluate many times:
//
//	expr, err := xpath1.Compile("//book[price > 30]")
//	eval := xpath1.NewEvaluator().
//	    Namespaces(map[string]string{"bk": "urn:books"})
//	result, err := eval.Evaluate(ctx, expr, doc)
//
// The [Evaluator] supports namespace bindings, variables, custom functions,
// and an operation limit, all configured via fluent builder methods.
//
// # Results
//
// [*Result] contains a type discriminant ([Result.Type]) and typed fields:
// [Result.NodeSet], [Result.Bool], [Result.Number], [Result.String].
//
// # Examples
//
// Example code for this package lives in the examples/ directory at the
// repository root (files prefixed with xpath_). Because examples are in
// a separate test module they do not appear in the generated documentation.
package xpath1
