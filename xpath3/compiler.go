package xpath3

// Compiler configures XPath 3.1 expression compilation.
// It is a value-style wrapper: fluent methods return updated copies
// and the original is never mutated.
//
// Compiler exists for symmetry with xslt3.Compiler and future growth.
// Currently the config is empty, but compile-time knobs may be added later.
type Compiler struct {
	cfg *compilerCfg
}

type compilerCfg struct {
	// empty initially
}

// NewCompiler creates a new Compiler with default settings.
func NewCompiler() Compiler {
	return Compiler{cfg: &compilerCfg{}}
}

// Compile parses an XPath 3.1 expression string into a reusable Expression.
func (c Compiler) Compile(expr string) (*Expression, error) {
	l, err := newLexer(expr)
	if err != nil {
		return nil, err
	}
	program, prefixPlan, err := compileFromLexer(l)
	if err != nil {
		return nil, err
	}
	return &Expression{
		source:     expr,
		program:    program,
		prefixPlan: prefixPlan,
	}, nil
}

// MustCompile is like Compile but panics on error.
func (c Compiler) MustCompile(expr string) *Expression {
	e, err := c.Compile(expr)
	if err != nil {
		panic("xpath3: Compile(" + expr + "): " + err.Error())
	}
	return e
}

// CompileExpr compiles a pre-parsed AST Expr into an Expression.
func (c Compiler) CompileExpr(ast Expr) (*Expression, error) {
	program, prefixPlan, err := compileVMProgram(ast)
	if err != nil {
		return nil, err
	}
	return &Expression{
		ast:        ast,
		program:    program,
		prefixPlan: prefixPlan,
	}, nil
}
