// Package xpath3 implements XPath 3.1 expression parsing and evaluation
// against helium XML document trees.
//
// The evaluation pipeline has two phases: compilation and execution.
//
// # Compilation
//
// Use [NewCompiler] to obtain a [Compiler], then call [Compiler.Compile] to
// produce an [*Expression]. Compiled expressions are safe for concurrent use
// and may be evaluated many times.
//
//	c := xpath3.NewCompiler()
//	expr, err := c.Compile("//book[price > 30]")
//
// # Evaluation
//
// Use [NewEvaluator] to obtain an [Evaluator], configure it with namespace
// bindings, variables, custom functions, and other dynamic-context settings
// via fluent builder methods, then call [Evaluator.Evaluate]:
//
//	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
//	    Namespaces(map[string]string{"bk": "urn:books"})
//	result, err := eval.Evaluate(ctx, expr, doc)
//
// # Results
//
// [*Result] wraps a [Sequence] of [Item] values. Inspect the result with
// type-checking helpers ([Result.IsNodeSet], [Result.IsBoolean], etc.) and
// extract values with [Result.Nodes], [Result.Atomics], or [Result.Sequence].
//
// # Features
//
// The evaluator supports the full XPath 3.1 language: FLWOR expressions,
// quantified expressions, if/then/else, try/catch, maps, arrays, inline
// functions, higher-order functions, the arrow operator, simple map, string
// concatenation, and all comparison forms (value, general, node).
//
// Over 100 built-in functions are provided across the fn:, math:, map:,
// array: namespaces. Custom functions can be registered via
// [Evaluator.Functions] or [Evaluator.FunctionResolver].
package xpath3
