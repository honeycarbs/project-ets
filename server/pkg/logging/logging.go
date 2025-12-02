package logging

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	s *zap.SugaredLogger
}

func New(level string) *Logger {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(parseLevel(level))

	z, err := cfg.Build()
	if err != nil {
		z, _ = zap.NewProduction()
	}

	return &Logger{s: z.Sugar()}
}

func (l *Logger) With(keyvals ...any) *Logger {
	return &Logger{s: l.s.With(keyvals...)}
}

func (l *Logger) Debug(msg string, keyvals ...any) {
	l.s.Debugw(msg, keyvals...)
}

func (l *Logger) Info(msg string, keyvals ...any) {
	l.s.Infow(msg, keyvals...)
}

func (l *Logger) Warn(msg string, keyvals ...any) {
	l.s.Warnw(msg, keyvals...)
}

func (l *Logger) Error(msg string, keyvals ...any) {
	l.s.Errorw(msg, keyvals...)
}

func (l *Logger) Sync() error {
	return l.s.Sync()
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	case "panic":
		return zapcore.PanicLevel
	default:
		return zapcore.InfoLevel
	}
}
