package observability

import (
	"log/slog"
	"os"
)

// Logger wraps slog.Logger with additional context.
type Logger struct {
	*slog.Logger
}

// NewLogger creates a new structured JSON logger.
func NewLogger(level string) *Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &Logger{Logger: logger}
}

// WithRequestID returns a logger with request_id field.
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{Logger: l.Logger.With("request_id", requestID)}
}

// WithTraceID returns a logger with trace_id field.
func (l *Logger) WithTraceID(traceID string) *Logger {
	return &Logger{Logger: l.Logger.With("trace_id", traceID)}
}
