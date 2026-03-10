package xpath3

import "context"

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
