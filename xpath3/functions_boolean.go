package xpath3

import "context"

func init() {
	registerFn("boolean", 1, 1, fnBoolean)
	registerFn("not", 1, 1, fnNot)
	registerFn("true", 0, 0, fnTrue)
	registerFn("false", 0, 0, fnFalse)
}

func fnBoolean(_ context.Context, args []Sequence) (Sequence, error) {
	b, err := EBV(args[0])
	if err != nil {
		return nil, err
	}
	return SingleBoolean(b), nil
}

func fnNot(_ context.Context, args []Sequence) (Sequence, error) {
	b, err := EBV(args[0])
	if err != nil {
		return nil, err
	}
	return SingleBoolean(!b), nil
}

func fnTrue(_ context.Context, _ []Sequence) (Sequence, error) {
	return SingleBoolean(true), nil
}

func fnFalse(_ context.Context, _ []Sequence) (Sequence, error) {
	return SingleBoolean(false), nil
}
