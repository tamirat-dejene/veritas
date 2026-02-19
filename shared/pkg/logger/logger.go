package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger() (*zap.Logger, error) {
	format := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT")))
	level := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL")))

	var cfg zap.Config
	if format == "console" || format == "pretty" {
		cfg = zap.NewDevelopmentConfig()
		cfg.Encoding = "console"
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.EncoderConfig.EncodeCaller = nil // Disable caller for cleaner output
		cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05")
		cfg.EncoderConfig.ConsoleSeparator = " "
		cfg.EncoderConfig.LineEnding = zapcore.DefaultLineEnding
	} else {
		cfg = zap.NewProductionConfig()
		cfg.Encoding = "json"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	cfg.Level = zap.NewAtomicLevelAt(parseLevel(level))
	return cfg.Build()
}

func IsConsoleFormat() bool {
	format := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT")))
	return format == "console" || format == "pretty"
}

func GetLogStyle() string {
	style := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_STYLE")))
	if style == "" {
		return "compact" // default
	}
	return style
}

func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}
