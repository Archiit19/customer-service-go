package zapcore

import (
	"strings"
	"time"
)

type Level int8

const (
	DebugLevel Level = -1
	InfoLevel  Level = 0
	WarnLevel  Level = 1
	ErrorLevel Level = 2
)

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	default:
		return "info"
	}
}

type LevelEncoder func(Level) string

type TimeEncoder func(time.Time) string

type DurationEncoder func(time.Duration) string

type CallerEncoder func(string) string

type EncoderConfig struct {
	TimeKey        string
	LevelKey       string
	NameKey        string
	CallerKey      string
	MessageKey     string
	StacktraceKey  string
	LineEnding     string
	EncodeLevel    LevelEncoder
	EncodeTime     TimeEncoder
	EncodeDuration DurationEncoder
	EncodeCaller   CallerEncoder
}

const DefaultLineEnding = "\n"

func CapitalLevelEncoder(level Level) string {
	return strings.ToUpper(level.String())
}

func ISO8601TimeEncoder(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func StringDurationEncoder(d time.Duration) string {
	return d.String()
}

func ShortCallerEncoder(caller string) string {
	return caller
}
