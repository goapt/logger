package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync/atomic"

	"github.com/goapt/logger/rolling"
)

// ---------------------------------------------------------------------------
// Handler options
// ---------------------------------------------------------------------------

// HandlerOption configures a handler created by NewJSONHandler.
type HandlerOption func(*handlerConfig)

type handlerConfig struct {
	level       slog.Level
	addSource   bool
	replaceAttr func([]string, slog.Attr) slog.Attr
}

// WithLevel sets the minimum log level for the handler.
// Only log records at or above this level will be written.
func WithLevel(level slog.Level) HandlerOption {
	return func(c *handlerConfig) { c.level = level }
}

// WithSource enables source file and line number in log output.
func WithSource() HandlerOption {
	return func(c *handlerConfig) { c.addSource = true }
}

// WithReplaceAttr sets a custom attribute replacement function.
// It is called after the default time formatting, so it can override
// the formatted time or transform any other attribute.
func WithReplaceAttr(fn func(groups []string, a slog.Attr) slog.Attr) HandlerOption {
	return func(c *handlerConfig) { c.replaceAttr = fn }
}

// ---------------------------------------------------------------------------
// NewJSONHandler
// ---------------------------------------------------------------------------

// NewJSONHandler creates a slog.Handler that writes JSON-formatted logs to w.
// By default, the handler logs at slog.LevelInfo and formats timestamps as
// "2006-01-02 15:04:05.000". Use HandlerOption functions to customize behavior.
//
// Example:
//
//	handler := logger.NewJSONHandler(os.Stdout, logger.WithLevel(slog.LevelDebug))
func NewJSONHandler(w io.Writer, opts ...HandlerOption) slog.Handler {
	cfg := handlerConfig{level: slog.LevelInfo}
	for _, o := range opts {
		o(&cfg)
	}

	replaceAttr := func(groups []string, a slog.Attr) slog.Attr {
		// Default: format timestamps.
		if a.Value.Kind() == slog.KindTime {
			return slog.String(a.Key, a.Value.Time().Format("2006-01-02 15:04:05.000"))
		}
		// User-provided replacement (called after default).
		if cfg.replaceAttr != nil {
			return cfg.replaceAttr(groups, a)
		}
		return a
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		AddSource:   cfg.addSource,
		Level:       cfg.level,
		ReplaceAttr: replaceAttr,
	})

	return handler
}

// ---------------------------------------------------------------------------
// New — create a logger from one or more handlers
// ---------------------------------------------------------------------------

// New creates a *slog.Logger that broadcasts log records to all given handlers.
// Each handler can have its own minimum level (set via WithLevel).
// If no handlers are provided, a default JSON handler writing to os.Stdout at
// slog.LevelInfo is used.
//
// Example — debug to stdout, info and above to file:
//
//	l := logger.New(
//	    logger.NewJSONHandler(os.Stdout, logger.WithLevel(slog.LevelDebug)),
//	    logger.NewJSONHandler(logger.NewFileWriter("app.log"), logger.WithLevel(slog.LevelInfo)),
//	)
//	l.Debug("debug msg")  // stdout only
//	l.Info("info msg")    // stdout + file
func New(handlers ...slog.Handler) *slog.Logger {
	switch len(handlers) {
	case 0:
		return slog.New(NewJSONHandler(os.Stdout))
	case 1:
		return slog.New(handlers[0])
	default:
		return slog.New(slog.NewMultiHandler(handlers...))
	}
}

// ---------------------------------------------------------------------------
// Default logger management
// ---------------------------------------------------------------------------

var defaultLogger atomic.Pointer[slog.Logger]

func init() {
	defaultLogger.Store(New())
}

// Default returns the standard logger used by the package-level output functions.
func Default() *slog.Logger { return defaultLogger.Load() }

// SetDefault sets the default logger used by the package-level output functions.
func SetDefault(logger *slog.Logger) { defaultLogger.Store(logger) }

// ---------------------------------------------------------------------------
// File writer
// ---------------------------------------------------------------------------

// FileOption configures a file writer created by NewFileWriter.
type FileOption func(*fileConfig)

type fileConfig struct {
	maxSize  int64
	maxFiles int
	maxAge   int
}

// WithMaxSize sets the maximum size in bytes for a single log file before rotation.
// Default is 200 MB.
func WithMaxSize(size int64) FileOption {
	return func(c *fileConfig) { c.maxSize = size }
}

// WithMaxFiles sets the maximum number of old log files to retain.
// Default is 3.
func WithMaxFiles(n int) FileOption {
	return func(c *fileConfig) { c.maxFiles = n }
}

// WithMaxAge sets the maximum number of days to retain old log files.
// Default is 3.
func WithMaxAge(days int) FileOption {
	return func(c *fileConfig) { c.maxAge = days }
}

// NewFileWriter creates an io.WriteCloser that writes to a rotating log file.
// It uses the rolling package for file rotation based on size, file count, and age.
//
// Example:
//
//	w := logger.NewFileWriter("app.log", logger.WithMaxSize(100<<20), logger.WithMaxFiles(5))
//	defer w.Close()
func NewFileWriter(filename string, opts ...FileOption) io.WriteCloser {
	cfg := fileConfig{
		maxSize:  200 * 1024 * 1024, // 200 MB
		maxFiles: 3,
		maxAge:   3,
	}
	for _, o := range opts {
		o(&cfg)
	}

	roller, err := rolling.NewRoller(
		filename,
		cfg.maxSize,
		rolling.WithMaxBackups(cfg.maxFiles),
		rolling.WithMaxAge(cfg.maxAge),
	)
	if err != nil {
		panic(fmt.Sprintf("logger: new file writer error: %s", err))
	}
	return roller
}

// Debug logs at [slog.LevelDebug] using the default logger.
func Debug(msg string, args ...any) {
	Default().Debug(msg, args...)
}

// DebugContext logs at [slog.LevelDebug] using the default logger with the given context.
func DebugContext(ctx context.Context, msg string, args ...any) {
	Default().DebugContext(ctx, msg, args...)
}

// Info logs at [slog.LevelInfo] using the default logger.
func Info(msg string, args ...any) {
	Default().Info(msg, args...)
}

// InfoContext logs at [slog.LevelInfo] using the default logger with the given context.
func InfoContext(ctx context.Context, msg string, args ...any) {
	Default().InfoContext(ctx, msg, args...)
}

// Warn logs at [slog.LevelWarn] using the default logger.
func Warn(msg string, args ...any) {
	Default().Warn(msg, args...)
}

// WarnContext logs at [slog.LevelWarn] using the default logger with the given context.
func WarnContext(ctx context.Context, msg string, args ...any) {
	Default().WarnContext(ctx, msg, args...)
}

// Error logs at [slog.LevelError] using the default logger.
func Error(msg string, args ...any) {
	Default().Error(msg, args...)
}

// ErrorContext logs at [slog.LevelError] using the default logger with the given context.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	Default().ErrorContext(ctx, msg, args...)
}
