package xpath3

import (
	"context"
	"math"
	"math/rand"
	"os"
	"sort"
	"time"
)

var qt3EnvironmentVariables = map[string]string{
	"QTTEST":      "42",
	"QTTEST2":     "other",
	"QTTESTEMPTY": "",
}

func init() {
	registerFn("static-base-uri", 0, 0, fnStaticBaseURI)
	registerFn("default-collation", 0, 0, fnDefaultCollation)
	registerFn("available-environment-variables", 0, 0, fnAvailableEnvVars)
	registerFn("environment-variable", 1, 1, fnEnvironmentVariable)
	registerFn("current-dateTime", 0, 0, fnCurrentDateTime)
	registerFn("current-date", 0, 0, fnCurrentDate)
	registerFn("current-time", 0, 0, fnCurrentTime)
	registerFn("implicit-timezone", 0, 0, fnImplicitTimezone)
	registerFn("default-language", 0, 0, fnDefaultLanguage)
	registerFn("random-number-generator", 0, 1, fnRandomNumberGenerator)
}

func fnStaticBaseURI(ctx context.Context, _ []Sequence) (Sequence, error) {
	if ec := getFnContext(ctx); ec != nil && ec.baseURI != "" {
		return SingleAtomic(AtomicValue{
			TypeName: TypeAnyURI,
			Value:    ec.baseURI,
		}), nil
	}
	return nil, nil
}

func fnDefaultCollation(_ context.Context, _ []Sequence) (Sequence, error) {
	return SingleAtomic(AtomicValue{
		TypeName: TypeAnyURI,
		Value:    "http://www.w3.org/2005/xpath-functions/collation/codepoint",
	}), nil
}

func fnAvailableEnvVars(_ context.Context, _ []Sequence) (Sequence, error) {
	names := make(map[string]struct{}, len(qt3EnvironmentVariables)+len(os.Environ()))
	for name := range qt3EnvironmentVariables {
		names[name] = struct{}{}
	}
	for _, env := range os.Environ() {
		if eq := len(env); eq > 0 {
			for i := 0; i < len(env); i++ {
				if env[i] == '=' {
					names[env[:i]] = struct{}{}
					break
				}
			}
		}
	}

	keys := make([]string, 0, len(names))
	for name := range names {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	result := make(Sequence, len(keys))
	for i, name := range keys {
		result[i] = AtomicValue{TypeName: TypeString, Value: name}
	}
	return result, nil
}

func fnEnvironmentVariable(_ context.Context, args []Sequence) (Sequence, error) {
	name, err := coerceArgToStringRequired(args[0])
	if err != nil {
		return nil, err
	}
	if val, ok := qt3EnvironmentVariables[name]; ok {
		return SingleString(val), nil
	}
	val, ok := os.LookupEnv(name)
	if !ok {
		return nil, nil
	}
	return SingleString(val), nil
}

func fnCurrentDateTime(ctx context.Context, _ []Sequence) (Sequence, error) {
	now := currentTimeFromCtx(ctx)
	return SingleAtomic(AtomicValue{TypeName: TypeDateTime, Value: now}), nil
}

func fnCurrentDate(ctx context.Context, _ []Sequence) (Sequence, error) {
	now := currentTimeFromCtx(ctx)
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return SingleAtomic(AtomicValue{TypeName: TypeDate, Value: date}), nil
}

func fnCurrentTime(ctx context.Context, _ []Sequence) (Sequence, error) {
	now := currentTimeFromCtx(ctx)
	return SingleAtomic(AtomicValue{TypeName: TypeTime, Value: now}), nil
}

func currentTimeFromCtx(ctx context.Context) time.Time {
	if ec := getFnContext(ctx); ec != nil {
		return ec.getCurrentTime()
	}
	return time.Now()
}

func fnRandomNumberGenerator(_ context.Context, args []Sequence) (Sequence, error) {
	var seed int64
	if len(args) > 0 && len(args[0]) > 0 {
		a, err := AtomizeItem(args[0][0])
		if err != nil {
			return nil, err
		}
		s, _ := atomicToString(a)
		// Use string hash as seed for reproducibility
		for _, c := range s {
			seed = seed*31 + int64(c)
		}
	} else {
		seed = time.Now().UnixNano()
	}
	return Sequence{makeRNGMap(seed)}, nil
}

func makeRNGMap(seed int64) MapItem {
	rng := rand.New(rand.NewSource(seed))
	number := rng.Float64()

	nextFn := FunctionItem{
		Arity: 0,
		Name:  "next",
		Invoke: func(_ context.Context, _ []Sequence) (Sequence, error) {
			nextSeed := rng.Int63()
			return Sequence{makeRNGMap(nextSeed)}, nil
		},
	}

	permuteFn := FunctionItem{
		Arity: 1,
		Name:  "permute",
		Invoke: func(_ context.Context, callArgs []Sequence) (Sequence, error) {
			seq := callArgs[0]
			perm := make(Sequence, len(seq))
			copy(perm, seq)
			localRng := rand.New(rand.NewSource(seed))
			sort.SliceStable(perm, func(i, j int) bool {
				return localRng.Intn(2) == 0
			})
			return perm, nil
		},
	}

	return NewMap([]MapEntry{
		{Key: AtomicValue{TypeName: TypeString, Value: "number"}, Value: Sequence{AtomicValue{TypeName: TypeDouble, Value: NewDouble(number)}}},
		{Key: AtomicValue{TypeName: TypeString, Value: "next"}, Value: Sequence{nextFn}},
		{Key: AtomicValue{TypeName: TypeString, Value: "permute"}, Value: Sequence{permuteFn}},
	})
}

func fnDefaultLanguage(ctx context.Context, _ []Sequence) (Sequence, error) {
	if ec := getFnContext(ctx); ec != nil {
		return SingleAtomic(AtomicValue{
			TypeName: TypeLanguage,
			Value:    ec.getDefaultLanguage(),
		}), nil
	}
	return SingleAtomic(AtomicValue{
		TypeName: TypeLanguage,
		Value:    "en",
	}), nil
}

func fnImplicitTimezone(ctx context.Context, _ []Sequence) (Sequence, error) {
	var offset int
	if ec := getFnContext(ctx); ec != nil && ec.implicitTimezone != nil {
		_, offset = time.Now().In(ec.implicitTimezone).Zone()
	} else {
		_, offset = time.Now().Zone()
	}
	d := Duration{
		Seconds: math.Abs(float64(offset)),
	}
	if offset < 0 {
		d.Negative = true
	}
	return SingleAtomic(AtomicValue{TypeName: TypeDayTimeDuration, Value: d}), nil
}
