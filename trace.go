//go:build !notrace

package helium

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"
)

type traceLoggerKey struct{}
type spanIDKey struct{}

// Span interface provides the upgrade path for future OpenTelemetry compatibility
type Span interface {
	End()
}

// internalSpan implements the Span interface
type internalSpan struct {
	finishFunc func()
}

func (s *internalSpan) End() {
	s.finishFunc()
}

// SpanInfo holds information about a tracing span
type SpanInfo struct {
	ID       string
	ParentID string
	Name     string
	Start    time.Time
	Tags     map[string]string
}

// TracingEnabled allows runtime control of tracing overhead
var TracingEnabled = true

// the null logger is a logger that does nothing
var nullLogger = slog.New(slog.DiscardHandler)

// WithTraceLogger adds a trace logger to the context
func WithTraceLogger(ctx context.Context, tlog *slog.Logger) context.Context {
	// If the context already has a trace logger, return the context as is
	if _, ok := ctx.Value(traceLoggerKey{}).(*slog.Logger); ok {
		return ctx
	}

	// Otherwise, create a new context with the trace logger
	return context.WithValue(ctx, traceLoggerKey{}, tlog)
}

// WithSpan creates a new span context with a generated span ID
func WithSpan(ctx context.Context, name string) (context.Context, *SpanInfo) {
	spanID := generateSpanID()
	parentID := ""

	// Get parent span ID if it exists
	if parent, ok := ctx.Value(spanIDKey{}).(*SpanInfo); ok {
		parentID = parent.ID
	}

	span := &SpanInfo{
		ID:       spanID,
		ParentID: parentID,
		Name:     name,
		Start:    time.Now(),
	}

	return context.WithValue(ctx, spanIDKey{}, span), span
}

// generateSpanID creates a random 16-character hex string for span identification
func generateSpanID() string {
	if !TracingEnabled {
		return ""
	}
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return hex.EncodeToString([]byte(time.Now().Format("15040500")))
	}
	return hex.EncodeToString(bytes)
}

func getTraceLogFromContext(ctx context.Context) *slog.Logger {
	if !TracingEnabled {
		return nullLogger
	}
	// If the context has a trace logger, use that
	if tlog, ok := ctx.Value(traceLoggerKey{}).(*slog.Logger); ok {
		// Add span information if available
		if span, ok := ctx.Value(spanIDKey{}).(*SpanInfo); ok {
			logger := tlog.With(
				slog.String("span_id", span.ID),
				slog.String("span_name", span.Name),
			)
			if span.ParentID != "" {
				logger = logger.With(slog.String("parent_span_id", span.ParentID))
			}
			return logger
		}
		return tlog
	}

	// Otherwise, return a null logger
	return nullLogger
}

// StartSpan creates a new span for tracing function execution and returns a Span object with End() method
// Modern pattern: ctx, span := StartSpan(ctx, name); defer span.End()
func StartSpan(ctx context.Context, spanName string) (context.Context, Span) {
	if !TracingEnabled {
		return ctx, &internalSpan{finishFunc: func() {}}
	}
	ctx, span := WithSpan(ctx, spanName)
	tlog := getTraceLogFromContext(ctx)
	tlog.Debug("START", slog.String("operation", "function_entry"))

	finishFunc := func() {
		if !TracingEnabled {
			return
		}
		duration := time.Since(span.Start)
		tlog.Debug("END",
			slog.String("operation", "function_exit"),
			slog.Duration("duration", duration),
		)
	}

	return ctx, &internalSpan{finishFunc: finishFunc}
}

// TraceEvent logs a structured event with span context
func TraceEvent(ctx context.Context, msg string, attrs ...slog.Attr) {
	if !TracingEnabled {
		return
	}
	tlog := getTraceLogFromContext(ctx)
	if len(attrs) == 0 {
		tlog.Debug(msg)
		return
	}

	// Simple conversion without pre-allocation optimization
	var args []any
	for _, attr := range attrs {
		args = append(args, attr.Key, attr.Value)
	}
	tlog.Debug(msg, args...)
}

// TraceError logs an error with span context
func TraceError(ctx context.Context, err error, msg string, attrs ...slog.Attr) {
	if !TracingEnabled {
		return
	}
	tlog := getTraceLogFromContext(ctx)
	// Simple conversion without pre-allocation optimization
	args := []any{"error", err.Error()}
	for _, attr := range attrs {
		args = append(args, attr.Key, attr.Value)
	}
	tlog.Error(msg, args...)
}

// SetTracingEnabled allows runtime control of tracing overhead
func SetTracingEnabled(enabled bool) {
	TracingEnabled = enabled
}
