package logging

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKeyType uint8

const (
	contextKey contextKeyType = iota
)

var nop = zap.NewNop()

// New creates a new Logger from a Core and options, and injects it into a Context
func New(ctx context.Context, core zapcore.Core, opts ...zap.Option) context.Context {
	return WithLogger(ctx, zap.New(core, opts...))
}

// WithLogger adds an existing Logger to a Context's values
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, contextKey, logger)
}

// FromContext attempts to retrieve a Logger from a Context's values
func FromContext(ctx context.Context) *zap.Logger {
	if logger, is := ctx.Value(contextKey).(*zap.Logger); is {
		return logger
	}

	return nop
}

// With adds fields to a Logger and re-injects it into a child Context
func With(ctx context.Context, fields ...zap.Field) (context.Context, *zap.Logger) {
	if logger, is := ctx.Value(contextKey).(*zap.Logger); is {
		logger = logger.With(fields...)

		return context.WithValue(ctx, contextKey, logger), logger
	}

	return ctx, nop
}

// Named appends a name to a Logger and re-injects it into a child Context
func Named(ctx context.Context, name string, fields ...zap.Field) (context.Context, *zap.Logger) {
	if logger, is := ctx.Value(contextKey).(*zap.Logger); is {
		logger = logger.Named(name)

		if len(fields) > 0 {
			logger = logger.With(fields...)
		}

		return context.WithValue(ctx, contextKey, logger), logger
	}

	return ctx, nop
}

// Info is a helper to log a single info-level message to a Context logger
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Info(msg, fields...)
}

// Error is a helper to log a single error-level message to a Context logger
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Error(msg, fields...)
}
