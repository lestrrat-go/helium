//go:build notrace

package helium

import (
	"context"
	"log/slog"
	"time"
)

// No-op implementations when built with -tags notrace for production performance

type traceLoggerKey struct{}
type spanIDKey struct{}

// Span interface provides the upgrade path for future OpenTelemetry compatibility - no-op version
type Span interface {
	End()
}

// noOpSpan implements the Span interface for no-trace builds
type noOpSpan struct{}

func (s *noOpSpan) End() {
	// no-op
}

// SpanInfo holds information about a tracing span
type SpanInfo struct {
	ID       string
	ParentID string
	Name     string
	Start    time.Time
	Tags     map[string]string
}

// Performance optimization: tracing is completely disabled
var TracingEnabled = false

// WithTraceLogger adds a trace logger to the context - no-op version
func WithTraceLogger(ctx context.Context, tlog *slog.Logger) context.Context {
	return ctx
}

// WithSpan creates a new span context - no-op version
func WithSpan(ctx context.Context, name string) (context.Context, *SpanInfo) {
	return ctx, nil
}

// StartSpan creates a new span - no-op version for maximum performance
func StartSpan(ctx context.Context, spanName string) (context.Context, Span) {
	return ctx, &noOpSpan{}
}

// TraceEvent logs a structured event - no-op version
func TraceEvent(ctx context.Context, msg string, attrs ...slog.Attr) {
	// no-op
}

// TraceError logs an error - no-op version
func TraceError(ctx context.Context, err error, msg string, attrs ...slog.Attr) {
	// no-op
}

// SetTracingEnabled allows runtime control - no-op version
func SetTracingEnabled(enabled bool) {
	// no-op
}

// getTraceLogFromContext returns null logger - no-op version
func getTraceLogFromContext(ctx context.Context) *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// generateSpanID creates empty span ID - no-op version
func generateSpanID() string {
	return ""
}
