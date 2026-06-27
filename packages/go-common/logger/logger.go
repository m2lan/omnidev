// Package logger provides structured logging with zap.
package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log is the global logger instance.
var Log *zap.Logger

// Sugared is the sugared logger for convenience.
var Sugared *zap.SugaredLogger

// Config holds logger configuration.
type Config struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // json, console
}

// Init initializes the global logger.
func Init(cfg Config) error {
	level := parseLevel(cfg.Level)

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeDuration = zapcore.MillisDurationEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	var encoder zapcore.Encoder
	if cfg.Format == "console" || cfg.Format == "" {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.Lock(os.Stdout),
		level,
	)

	Log = zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(0),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	Sugared = Log.Sugar()

	return nil
}

// InitDefault initializes a default logger for testing.
func InitDefault() {
	Log = zap.NewNop()
	Sugared = Log.Sugar()
}

// WithFields creates a logger with additional fields.
func WithFields(fields ...zap.Field) *zap.Logger {
	if Log == nil {
		InitDefault()
	}
	return Log.With(fields...)
}

// Sync flushes any buffered log entries.
func Sync() error {
	if Log == nil {
		return nil
	}
	return Log.Sync()
}

func parseLevel(s string) zapcore.Level {
	switch strings.ToLower(s) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
