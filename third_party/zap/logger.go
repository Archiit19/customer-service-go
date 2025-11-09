package zap

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"go.uber.org/zap/zapcore"
)

type Field struct {
	Key   string
	Value any
}

type AtomicLevel struct {
	level zapcore.Level
}

type Config struct {
	Level             AtomicLevel
	Development       bool
	Encoding          string
	OutputPaths       []string
	ErrorOutputPaths  []string
	EncoderConfig     zapcore.EncoderConfig
	DisableCaller     bool
	DisableStacktrace bool
}

type Logger struct {
	mu      sync.Mutex
	level   zapcore.Level
	fields  []Field
	encCfg  zapcore.EncoderConfig
	writers []io.Writer
}

func NewAtomicLevelAt(level zapcore.Level) AtomicLevel {
	return AtomicLevel{level: level}
}

func (a AtomicLevel) Enabled(level zapcore.Level) bool {
	return level >= a.level
}

func (c Config) Build() (*Logger, error) {
	writers := make([]io.Writer, 0, len(c.OutputPaths))
	for _, path := range c.OutputPaths {
		switch path {
		case "stdout":
			writers = append(writers, os.Stdout)
		case "stderr":
			writers = append(writers, os.Stderr)
		default:
			f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				return nil, err
			}
			writers = append(writers, f)
		}
	}
	if len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}
	return &Logger{
		level:   c.Level.level,
		encCfg:  c.EncoderConfig,
		writers: writers,
	}, nil
}

func (l *Logger) With(fields ...Field) *Logger {
	clone := &Logger{
		level:   l.level,
		encCfg:  l.encCfg,
		writers: l.writers,
	}
	clone.fields = append(clone.fields, l.fields...)
	clone.fields = append(clone.fields, fields...)
	return clone
}

func (l *Logger) Debug(msg string, fields ...Field) {
	l.log(zapcore.DebugLevel, msg, fields)
}

func (l *Logger) Info(msg string, fields ...Field) {
	l.log(zapcore.InfoLevel, msg, fields)
}

func (l *Logger) Warn(msg string, fields ...Field) {
	l.log(zapcore.WarnLevel, msg, fields)
}

func (l *Logger) Error(msg string, fields ...Field) {
	l.log(zapcore.ErrorLevel, msg, fields)
}

func (l *Logger) Sync() error {
	return nil
}

func (l *Logger) log(level zapcore.Level, msg string, fields []Field) {
	if level < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	entry := make(map[string]any)
	cfg := l.encCfg
	if cfg.TimeKey != "" {
		if cfg.EncodeTime != nil {
			entry[cfg.TimeKey] = cfg.EncodeTime(time.Now())
		} else {
			entry[cfg.TimeKey] = zapcore.ISO8601TimeEncoder(time.Now())
		}
	}
	if cfg.LevelKey != "" {
		if cfg.EncodeLevel != nil {
			entry[cfg.LevelKey] = cfg.EncodeLevel(level)
		} else {
			entry[cfg.LevelKey] = level.String()
		}
	}
	if cfg.MessageKey != "" {
		entry[cfg.MessageKey] = msg
	}
	for _, f := range l.fields {
		entry[f.Key] = renderValue(f.Value)
	}
	for _, f := range fields {
		entry[f.Key] = renderValue(f.Value)
	}
	data, err := json.Marshal(entry)
	if err != nil {
		data = []byte(fmt.Sprintf(`{"level":"error","message":"logging failure","error":"%v"}`, err))
	}
	for _, w := range l.writers {
		_, _ = w.Write(data)
		_, _ = w.Write([]byte(cfg.LineEnding))
	}
}

func renderValue(v any) any {
	switch val := v.(type) {
	case error:
		return val.Error()
	default:
		return val
	}
}

func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

func Int32(key string, value int32) Field {
	return Field{Key: key, Value: value}
}

func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.String()}
}

func Error(err error) Field {
	return Field{Key: "error", Value: err}
}

func Any(key string, value any) Field {
	return Field{Key: key, Value: value}
}
