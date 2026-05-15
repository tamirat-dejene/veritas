package logger

import (
	"context"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey string

const (
	RequestIDKey   contextKey = "requestID"
	UserIDKey      contextKey = "userID"
	RoleKey        contextKey = "role"
	EnterpriseIDKey contextKey = "enterpriseID"
)

// NewLogger initializes a zap logger with configuration from environment variables.
func NewLogger(serviceName string) (*zap.Logger, error) {
	format := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT")))
	level := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL")))

	var cfg zap.Config
	if format == "console" || format == "pretty" {
		cfg = zap.NewDevelopmentConfig()
		cfg.Encoding = "console"
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
		cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05")
		cfg.EncoderConfig.ConsoleSeparator = " "
		cfg.EncoderConfig.LineEnding = zapcore.DefaultLineEnding
		cfg.Development = false
	} else {
		cfg = zap.NewProductionConfig()
		cfg.Encoding = "json"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	cfg.Level = zap.NewAtomicLevelAt(parseLevel(level))
	l, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	if serviceName != "" {
		l = l.With(zap.String("service", serviceName))
	}

	return l, nil
}

// WithContext returns a logger with fields extracted from the context.
func WithContext(ctx context.Context, log *zap.Logger) *zap.Logger {
	if ctx == nil {
		return log
	}

	fields := []zap.Field{}
	if rid, ok := ctx.Value(RequestIDKey).(string); ok && rid != "" {
		fields = append(fields, zap.String("request_id", rid))
	}
	if uid, ok := ctx.Value(UserIDKey).(string); ok && uid != "" {
		fields = append(fields, zap.String("user_id", uid))
	}
	if role, ok := ctx.Value(RoleKey).(string); ok && role != "" {
		fields = append(fields, zap.String("role", role))
	}
	if eid, ok := ctx.Value(EnterpriseIDKey).(string); ok && eid != "" {
		fields = append(fields, zap.String("enterprise_id", eid))
	}

	if len(fields) > 0 {
		return log.With(fields...)
	}
	return log
}

// IsConsoleFormat returns true if the LOG_FORMAT is set to console or pretty.
func IsConsoleFormat() bool {
	format := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT")))
	return format == "console" || format == "pretty"
}

// GetLogStyle returns the value of LOG_STYLE environment variable.
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

// SetRequestID injects a request ID into the context.
func SetRequestID(ctx context.Context, rid string) context.Context {
	return context.WithValue(ctx, RequestIDKey, rid)
}

// SetUserID injects a user ID into the context.
func SetUserID(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, UserIDKey, uid)
}

// SetRole injects a user role into the context.
func SetRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, RoleKey, role)
}

// SetEnterpriseID injects an enterprise ID into the context.
func SetEnterpriseID(ctx context.Context, eid string) context.Context {
	return context.WithValue(ctx, EnterpriseIDKey, eid)
}
