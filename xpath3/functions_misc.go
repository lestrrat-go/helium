package xpath3

import (
	"context"
	"os"
	"time"
)

func init() {
	registerFn("static-base-uri", 0, 0, fnStaticBaseURI)
	registerFn("default-collation", 0, 0, fnDefaultCollation)
	registerFn("available-environment-variables", 0, 0, fnAvailableEnvVars)
	registerFn("environment-variable", 1, 1, fnEnvironmentVariable)
	registerFn("current-dateTime", 0, 0, fnCurrentDateTime)
	registerFn("current-date", 0, 0, fnCurrentDate)
	registerFn("current-time", 0, 0, fnCurrentTime)
	registerFn("implicit-timezone", 0, 0, fnImplicitTimezone)
}

func fnStaticBaseURI(_ context.Context, _ []Sequence) (Sequence, error) {
	return nil, nil // Not available in this implementation
}

func fnDefaultCollation(_ context.Context, _ []Sequence) (Sequence, error) {
	return SingleAtomic(AtomicValue{
		TypeName: TypeAnyURI,
		Value:    "http://www.w3.org/2005/xpath-functions/collation/codepoint",
	}), nil
}

func fnAvailableEnvVars(_ context.Context, _ []Sequence) (Sequence, error) {
	envs := os.Environ()
	result := make(Sequence, len(envs))
	for i, env := range envs {
		// Each entry is "KEY=VALUE"; we want just the key
		for j := 0; j < len(env); j++ {
			if env[j] == '=' {
				result[i] = AtomicValue{TypeName: TypeString, Value: env[:j]}
				break
			}
		}
		if result[i] == nil {
			result[i] = AtomicValue{TypeName: TypeString, Value: env}
		}
	}
	return result, nil
}

func fnEnvironmentVariable(_ context.Context, args []Sequence) (Sequence, error) {
	name := seqToString(args[0])
	val, ok := os.LookupEnv(name)
	if !ok {
		return nil, nil
	}
	return SingleString(val), nil
}

func fnCurrentDateTime(_ context.Context, _ []Sequence) (Sequence, error) {
	return SingleAtomic(AtomicValue{TypeName: TypeDateTime, Value: time.Now()}), nil
}

func fnCurrentDate(_ context.Context, _ []Sequence) (Sequence, error) {
	now := time.Now()
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return SingleAtomic(AtomicValue{TypeName: TypeDate, Value: date}), nil
}

func fnCurrentTime(_ context.Context, _ []Sequence) (Sequence, error) {
	return SingleAtomic(AtomicValue{TypeName: TypeTime, Value: time.Now()}), nil
}

func fnImplicitTimezone(_ context.Context, _ []Sequence) (Sequence, error) {
	_, offset := time.Now().Zone()
	hours := offset / 3600
	minutes := (offset % 3600) / 60
	d := Duration{
		Seconds: float64(hours*3600 + minutes*60),
	}
	return SingleAtomic(AtomicValue{TypeName: TypeDayTimeDuration, Value: d}), nil
}
