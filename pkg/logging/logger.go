package logging

import (
	"context"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is a small abstraction over structured logging.
type Logger interface {
	Debug(msg string, attrs ...any)
	Info(msg string, attrs ...any)
	Warn(msg string, attrs ...any)
	Error(msg string, attrs ...any)
}

func New(level string) Logger {
	return newZapLogger(level)
}

func newZapLogger(level string) Logger {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"
	cfg.Level = zap.NewAtomicLevelAt(parseLevel(level))
	core, err := cfg.Build()
	if err != nil {
		return &zapLogger{base: zap.NewNop()}
	}
	return &zapLogger{base: core}
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// Discard returns a logger that drops all output.
func Discard() Logger {
	return &zapLogger{base: zap.NewNop()}
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

type zapLogger struct {
	base *zap.Logger
}

func (l *zapLogger) Debug(msg string, attrs ...any) {
	if l == nil || l.base == nil {
		return
	}
	l.base.Debug(msg, toFields(attrs...)...)
}

func (l *zapLogger) Info(msg string, attrs ...any) {
	if l == nil || l.base == nil {
		return
	}
	l.base.Info(msg, toFields(attrs...)...)
}

func (l *zapLogger) Warn(msg string, attrs ...any) {
	if l == nil || l.base == nil {
		return
	}
	l.base.Warn(msg, toFields(attrs...)...)
}

func (l *zapLogger) Error(msg string, attrs ...any) {
	if l == nil || l.base == nil {
		return
	}
	l.base.Error(msg, toFields(attrs...)...)
}

func toFields(attrs ...any) []zap.Field {
	fields := make([]zap.Field, 0, len(attrs)/2+1)
	for i := 0; i < len(attrs); i += 2 {
		key, ok := attrs[i].(string)
		if !ok || strings.TrimSpace(key) == "" {
			continue
		}
		if i+1 >= len(attrs) {
			fields = append(fields, zap.Any(key, nil))
			continue
		}
		fields = append(fields, zap.Any(key, attrs[i+1]))
	}
	return fields
}

type contextKeyLogger struct{}
