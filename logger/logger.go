package logger

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

var Logger *slog.Logger

func Init() {
	Logger = slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}),
	)
}

// WithTrace adds trace context to log attributes
func WithTrace(ctx context.Context, attrs ...any) []any {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		attrs = append(attrs,
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}
	return attrs
}

// Info logs info message with trace correlation
func Info(ctx context.Context, msg string, attrs ...any) {
	Logger.InfoContext(ctx, msg, WithTrace(ctx, attrs...)...)
}

// Error logs error message with trace correlation
func Error(ctx context.Context, msg string, attrs ...any) {
	Logger.ErrorContext(ctx, msg, WithTrace(ctx, attrs...)...)
}

// Debug logs debug message with trace correlation
func Debug(ctx context.Context, msg string, attrs ...any) {
	Logger.DebugContext(ctx, msg, WithTrace(ctx, attrs...)...)
}
