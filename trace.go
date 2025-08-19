package helium

import (
	"context"
	"log/slog"
	"runtime"
)

type traceLoggerKey struct{}

// the null logger is a logger that does nothing
var nullLogger = slog.New(slog.DiscardHandler)

func WithTraceLogger(ctx context.Context, tlog *slog.Logger) context.Context {
	// If the context already has a trace logger, return the context as is
	if _, ok := ctx.Value(traceLoggerKey{}).(*slog.Logger); ok {
		return ctx
	}

	// Otherwise, create a new context with the trace logger
	return context.WithValue(ctx, traceLoggerKey{}, tlog)
}

func getTraceLogFromContext(ctx context.Context) *slog.Logger {
	// If the context has a trace logger, use that
	if tlog, ok := ctx.Value(traceLoggerKey{}).(*slog.Logger); ok {
		// Retrieve the function name of the caller for tracing
		pc, _, _, ok := runtime.Caller(2)
		if ok {
			fn := runtime.FuncForPC(pc)
			if fn != nil {
				tlog = tlog.With(slog.String("fn", fn.Name()))
			}
		}

		return tlog
	}

	// Otherwise, return a null logger
	return nullLogger
}
