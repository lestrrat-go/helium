package xpath3

import (
	"context"
	"math"
	"math/rand"
	"os"
	"sort"
	"time"

	"github.com/lestrrat-go/helium/internal/lexicon"
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
	registerFn("load-xquery-module", 1, 2, fnLoadXQueryModule)
	registerFn("transform", 1, 1, fnTransform)
}

func fnStaticBaseURI(ctx context.Context, _ []Sequence) (Sequence, error) {
	if ec := getFnContext(ctx); ec != nil && ec.baseURI != "" {
		return SingleAtomic(AtomicValue{
			TypeName: TypeAnyURI,
			Value:    ec.baseURI,
		}), nil
	}
	return nil, nil //nolint:nilnil
}

func fnDefaultCollation(ctx context.Context, _ []Sequence) (Sequence, error) {
	uri := lexicon.CollationCodepoint
	if ec := getFnContext(ctx); ec != nil && ec.defaultCollation != "" {
		uri = ec.defaultCollation
	}
	return SingleAtomic(AtomicValue{
		TypeName: TypeAnyURI,
		Value:    uri,
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

	result := make(ItemSlice, len(keys))
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
		return nil, nil //nolint:nilnil
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
		return ec.getCurrentTime().In(ec.getImplicitTimezone())
	}
	return time.Now()
}

func fnRandomNumberGenerator(ctx context.Context, args []Sequence) (Sequence, error) {
	var seed int64
	if len(args) > 0 && seqLen(args[0]) > 0 {
		a, err := AtomizeItem(args[0].Get(0))
		if err != nil {
			return nil, err
		}
		s, _ := atomicToString(a)
		// Use string hash as seed for reproducibility
		for _, c := range s {
			seed = seed*31 + int64(c)
		}
	} else {
		seed = currentTimeFromCtx(ctx).UnixNano()
	}
	return ItemSlice{makeRNGMap(seed)}, nil
}

func makeRNGMap(seed int64) MapItem {
	rng := rand.New(rand.NewSource(seed))
	number := rng.Float64()

	nextFn := FunctionItem{
		Arity: 0,
		Name:  "next",
		Invoke: func(_ context.Context, _ []Sequence) (Sequence, error) {
			nextSeed := rng.Int63()
			return ItemSlice{makeRNGMap(nextSeed)}, nil
		},
	}

	permuteFn := FunctionItem{
		Arity: 1,
		Name:  "permute",
		Invoke: func(_ context.Context, callArgs []Sequence) (Sequence, error) {
			seq := callArgs[0]
			perm := make(ItemSlice, seqLen(seq))
			copy(perm, seqMaterialize(seq))
			localRng := rand.New(rand.NewSource(seed))
			sort.SliceStable(perm, func(i, j int) bool {
				return localRng.Intn(2) == 0
			})
			return perm, nil
		},
	}

	return NewMap([]MapEntry{
		{Key: AtomicValue{TypeName: TypeString, Value: "number"}, Value: ItemSlice{AtomicValue{TypeName: TypeDouble, Value: NewDouble(number)}}},
		{Key: AtomicValue{TypeName: TypeString, Value: "next"}, Value: ItemSlice{nextFn}},
		{Key: AtomicValue{TypeName: TypeString, Value: "permute"}, Value: ItemSlice{permuteFn}},
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
	if ec := getFnContext(ctx); ec != nil {
		_, offset = ec.getCurrentTime().In(ec.getImplicitTimezone()).Zone()
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

func fnLoadXQueryModule(_ context.Context, _ []Sequence) (Sequence, error) {
	return nil, &XPathError{Code: errCodeFOER0000, Message: "fn:load-xquery-module is not implemented"}
}

func fnTransform(_ context.Context, _ []Sequence) (Sequence, error) {
	return nil, &XPathError{Code: errCodeFOER0000, Message: "fn:transform is not implemented"}
}
