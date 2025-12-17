package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Config holds logging configuration
type Config struct {
	// File settings
	LogDir     string // Directory for log files (default: executable directory)
	LogFile    string // Log file name (default: ldapmerge.log)
	MaxSize    int    // Max size in MB before rotation (default: 100)
	MaxBackups int    // Max number of old log files (default: 5)
	MaxAge     int    // Max days to retain old logs (default: 30)
	Compress   bool   // Compress rotated files (default: true)

	// Output settings
	Level      slog.Level // Log level (default: Info)
	JSONFormat bool       // Use JSON format (default: true for file)
	Console    bool       // Also output to console (default: false)
}

// DefaultConfig returns default logging configuration
func DefaultConfig() Config {
	return Config{
		LogDir:     "",
		LogFile:    "ldapmerge.log",
		MaxSize:    100, // 100 MB
		MaxBackups: 5,
		MaxAge:     30, // 30 days
		Compress:   true,
		Level:      slog.LevelInfo,
		JSONFormat: true,
		Console:    false,
	}
}

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	*slog.Logger
	lj *lumberjack.Logger
}

// New creates a new logger with the given configuration
func New(cfg Config) (*Logger, error) {
	logPath := getLogPath(cfg)

	// Ensure log directory exists
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}

	// Setup lumberjack for rotation
	lj := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
		LocalTime:  true,
	}

	var writer io.Writer = lj
	if cfg.Console {
		writer = io.MultiWriter(lj, os.Stdout)
	}

	// Create handler based on format preference
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: cfg.Level,
	}

	if cfg.JSONFormat {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	logger := slog.New(handler)

	return &Logger{
		Logger: logger,
		lj:     lj,
	}, nil
}

// Close closes the underlying log file
func (l *Logger) Close() error {
	if l.lj != nil {
		return l.lj.Close()
	}
	return nil
}

// Rotate forces log rotation
func (l *Logger) Rotate() error {
	if l.lj != nil {
		return l.lj.Rotate()
	}
	return nil
}

// getLogPath determines the log file path
func getLogPath(cfg Config) string {
	logDir := cfg.LogDir

	if logDir == "" {
		// Default: directory of executable
		exe, err := os.Executable()
		if err != nil {
			logDir = "."
		} else {
			logDir = filepath.Dir(exe)
		}
	}

	return filepath.Join(logDir, cfg.LogFile)
}

// Global logger instance
var globalLogger *Logger

// Init initializes the global logger
func Init(cfg Config) error {
	logger, err := New(cfg)
	if err != nil {
		return err
	}
	globalLogger = logger
	slog.SetDefault(logger.Logger)
	return nil
}

// Close closes the global logger
func Close() error {
	if globalLogger != nil {
		return globalLogger.Close()
	}
	return nil
}

// Get returns the global logger
func Get() *Logger {
	return globalLogger
}

// Convenience functions that use the global logger

func Debug(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Debug(msg, args...)
	}
}

func Info(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Info(msg, args...)
	}
}

func Warn(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Warn(msg, args...)
	}
}

func Error(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Error(msg, args...)
	}
}

func With(args ...any) *slog.Logger {
	if globalLogger != nil {
		return globalLogger.With(args...)
	}
	return slog.Default()
}
