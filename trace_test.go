package helium

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithTraceLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	ctx := context.Background()

	// Test adding trace logger
	ctx = WithTraceLogger(ctx, logger)

	// Test that the logger is retrievable
	tlog := getTraceLogFromContext(ctx)
	if !TracingEnabled {
		// In no-trace builds, the logger might be nil - that's expected
		if tlog == nil {
			t.Skip("Tracing disabled - skipping trace logger test")
			return
		}
	}
	require.NotNil(t, tlog)

	// Test logging
	tlog.Debug("test message")

	output := buf.String()
	if TracingEnabled {
		require.Contains(t, output, "test message")
	}
}

func TestWithSpan(t *testing.T) {
	if !TracingEnabled {
		t.Skip("Tracing disabled - skipping span test")
		return
	}

	ctx := context.Background()

	// Test creating a span
	ctx, span := WithSpan(ctx, "test_operation")

	require.NotEmpty(t, span.ID)
	require.Equal(t, "test_operation", span.Name)
	require.Empty(t, span.ParentID)
	require.False(t, span.Start.IsZero())

	// Test nested span
	_, span2 := WithSpan(ctx, "nested_operation")

	require.NotEmpty(t, span2.ID)
	require.Equal(t, "nested_operation", span2.Name)
	require.Equal(t, span.ID, span2.ParentID)
	require.NotEqual(t, span.ID, span2.ID)
}

func TestStartSpan(t *testing.T) {
	if !TracingEnabled {
		t.Skip("Tracing disabled - skipping StartSpan test")
		return
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	ctx := WithTraceLogger(context.Background(), logger)

	// Test StartSpan helper
	ctx, span := StartSpan(ctx, "test_function")

	// Simulate some work
	time.Sleep(time.Millisecond)

	span.End()

	output := buf.String()
	require.Contains(t, output, "START")
	require.Contains(t, output, "END")
	require.Contains(t, output, "span_id")
	require.Contains(t, output, "span_name")
	require.Contains(t, output, "test_function")
	require.Contains(t, output, "duration")
}

func TestTraceEvent(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	ctx := WithTraceLogger(context.Background(), logger)
	ctx, _ = WithSpan(ctx, "test_span")

	TraceEvent(ctx, "processing data",
		slog.String("data_type", "xml"),
		slog.Int("size", 1024),
	)

	output := buf.String()
	if TracingEnabled {
		require.Contains(t, output, "processing data")
		require.Contains(t, output, "data_type")
		require.Contains(t, output, "xml")
		require.Contains(t, output, "size")
		require.Contains(t, output, "1024")
		require.Contains(t, output, "span_id")
	} else {
		// In no-trace mode, no output should be generated
		require.Empty(t, output)
	}
}

func TestTraceError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	ctx := WithTraceLogger(context.Background(), logger)
	ctx, _ = WithSpan(ctx, "error_span")

	testErr := errors.New("test error")
	TraceError(ctx, testErr, "error occurred", slog.String("component", "parser"))

	output := buf.String()
	if TracingEnabled {
		require.Contains(t, output, "error occurred")
		require.Contains(t, output, "test error")
		require.Contains(t, output, "component")
		require.Contains(t, output, "parser")
		require.Contains(t, output, "span_id")
	} else {
		// In no-trace mode, no output should be generated
		require.Empty(t, output)
	}

	if TracingEnabled {
		require.Contains(t, output, "ERROR")
	}
}

func TestNullLogger(t *testing.T) {
	ctx := context.Background()

	// Test getting logger from context without trace logger
	tlog := getTraceLogFromContext(ctx)
	require.NotNil(t, tlog)

	// Should not panic when logging to null logger
	require.NotPanics(t, func() {
		tlog.Debug("this should not output anything")
		TraceEvent(ctx, "test event")
		TraceError(ctx, errors.New("test"), "test error")
	})
}

func TestSpanIDGeneration(t *testing.T) {
	if !TracingEnabled {
		t.Skip("Tracing disabled - skipping span ID generation test")
		return
	}

	// Test that span IDs are unique
	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		id := generateSpanID()
		require.NotEmpty(t, id)
		require.Len(t, id, 16) // 8 bytes = 16 hex chars
		require.False(t, ids[id], "Span ID collision detected: %s", id)
		ids[id] = true
	}
}
