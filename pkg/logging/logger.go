package logging

import (
	"context"
	"log/slog"
	"os"
)

// Logger is a small abstraction over structured logging.
type Logger interface {
	Debug(msg string, attrs ...any)
	Info(msg string, attrs ...any)
	Warn(msg string, attrs ...any)
	Error(msg string, attrs ...any)
}

func New(level string) Logger {
	return newSlogLogger(level)
}

func newSlogLogger(level string) Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}
	h := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(h)
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Discard returns a logger that drops all output.
func Discard() Logger {
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 100})
	return slog.New(h)
}

// ContextWithLogger stores logger in context for optional downstream usage.
func ContextWithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, contextKeyLogger{}, logger)
}

// FromContext returns context-bound logger or fallback.
func FromContext(ctx context.Context, fallback Logger) Logger {
	if v := ctx.Value(contextKeyLogger{}); v != nil {
		if lg, ok := v.(Logger); ok {
			return lg
		}
	}
	return fallback
}

type contextKeyLogger struct{}
