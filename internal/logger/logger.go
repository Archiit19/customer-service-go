package logger

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Field = zap.Field

type Logger interface {
	Debug(ctx context.Context, msg string, fields ...Field)
	Info(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, fields ...Field)
	Sync() error
}

type contextKey string

const requestIDKey contextKey = "request_id"

type zapLogger struct {
	base *zap.Logger
}

func New(level string) (Logger, error) {
	cfg := zap.Config{
		Level:             zap.NewAtomicLevelAt(parseLevel(level)),
		Development:       false,
		Encoding:          "json",
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stderr"},
		EncoderConfig:     encoderConfig(),
		DisableCaller:     true,
		DisableStacktrace: true,
	}
	lg, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return &zapLogger{base: lg}, nil
}

func (l *zapLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	l.withContext(ctx).Debug(msg, fields...)
}

func (l *zapLogger) Info(ctx context.Context, msg string, fields ...Field) {
	l.withContext(ctx).Info(msg, fields...)
}

func (l *zapLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	l.withContext(ctx).Warn(msg, fields...)
}

func (l *zapLogger) Error(ctx context.Context, msg string, fields ...Field) {
	l.withContext(ctx).Error(msg, fields...)
}

func (l *zapLogger) Sync() error {
	return l.base.Sync()
}

func (l *zapLogger) withContext(ctx context.Context) *zap.Logger {
	requestID := RequestIDFromContext(ctx)
	if requestID == "" {
		requestID = "system"
	}
	return l.base.With(zap.String(string(requestIDKey), requestID))
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return zapcore.DebugLevel
	case "WARN":
		return zapcore.WarnLevel
	case "ERROR":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func encoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value := ctx.Value(requestIDKey)
	if value == nil {
		return ""
	}
	id, _ := value.(string)
	return id
}

func String(key, value string) Field {
	return zap.String(key, value)
}

func Int(key string, value int) Field {
	return zap.Int(key, value)
}

func Int32(key string, value int32) Field {
	return zap.Int32(key, value)
}

func Int64(key string, value int64) Field {
	return zap.Int64(key, value)
}

func Bool(key string, value bool) Field {
	return zap.Bool(key, value)
}

func Duration(key string, value time.Duration) Field {
	return zap.Duration(key, value)
}

func Err(err error) Field {
	return zap.Error(err)
}

func Any(key string, value any) Field {
	return zap.Any(key, value)
}
