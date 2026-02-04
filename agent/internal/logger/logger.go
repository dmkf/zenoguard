package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logger *Logger
	once   sync.Once
	mu     sync.Mutex
)

// LogLevel represents log level
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger represents the logger
type Logger struct {
	mu     sync.Mutex
	level  LogLevel
	writer *lumberjack.Logger
}

// Init initializes the logger
func Init(logPath string, level LogLevel) error {
	var initErr error
	once.Do(func() {
		// Ensure directory exists
		dir := filepath.Dir(logPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			initErr = fmt.Errorf("failed to create log directory: %w", err)
			return
		}

		// Create rotating file logger
		rotateLogger := &lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    5, // MB
			MaxBackups: 5,
			MaxAge:     30, // days
			Compress:   true,
			LocalTime:  true,
		}

		logger = &Logger{
			level:  level,
			writer: rotateLogger,
		}

		Info("Logger initialized: " + logPath)
	})

	return initErr
}

// Get returns the logger instance
func Get() *Logger {
	if logger == nil {
		// Default initialization
		_ = Init("/var/log/zenoguard/agent.log", INFO)
	}
	return logger
}

// SetLevel sets the log level
func SetLevel(level LogLevel) {
	if logger != nil {
		logger.mu.Lock()
		logger.level = level
		logger.mu.Unlock()
	}
}

// log logs a message
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("{\"time\":\"%s\",\"level\":\"%s\",\"message\":\"%s\"}\n",
		timestamp, level.String(), message)

	l.mu.Lock()
	defer l.mu.Unlock()

	// Write to rotating file
	if l.writer != nil {
		l.writer.Write([]byte(logLine))
	}

	// Also write to stderr for fatal errors
	if level >= ERROR {
		os.Stderr.Write([]byte(logLine))
	}
}

// Debug logs a debug message
func Debug(format string, args ...interface{}) {
	Get().log(DEBUG, format, args...)
}

// Info logs an info message
func Info(format string, args ...interface{}) {
	Get().log(INFO, format, args...)
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	Get().log(WARN, format, args...)
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	Get().log(ERROR, format, args...)
}

// Fatal logs a fatal message and exits
func Fatal(format string, args ...interface{}) {
	Get().log(FATAL, format, args...)
	os.Exit(1)
}

// Close closes the logger
func Close() error {
	if logger != nil && logger.writer != nil {
		return logger.writer.Close()
	}
	return nil
}

// Rotate forces a log rotation
func Rotate() error {
	if logger != nil && logger.writer != nil {
		return logger.writer.Rotate()
	}
	return nil
}
