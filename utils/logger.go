package utils

import (
	"fmt"
	"log"
	"os"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// String returns the string representation of the log level
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
	default:
		return "UNKNOWN"
	}
}

// Logger is a custom logger with level support
type Logger struct {
	*log.Logger
	level  LogLevel
	fields map[string]interface{}
}

// NewLogger creates a new logger with the specified level
func NewLogger(level LogLevel) *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "", 0),
		level:  level,
		fields: make(map[string]interface{}),
	}
}

// shouldLog checks if a message at the given level should be logged
func (l *Logger) shouldLog(level LogLevel) bool {
	return level >= l.level
}

// formatMessage formats a log message with timestamp and level
func (l *Logger) formatMessage(level LogLevel, format string, v ...interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, v...)
	
	// Add fields if present
	if len(l.fields) > 0 {
		fieldsStr := " ["
		first := true
		for key, val := range l.fields {
			if !first {
				fieldsStr += ", "
			}
			fieldsStr += fmt.Sprintf("%s=%v", key, val)
			first = false
		}
		fieldsStr += "]"
		message += fieldsStr
	}
	
	return fmt.Sprintf("[%s] [%s] %s", timestamp, level.String(), message)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, v ...interface{}) {
	if l.shouldLog(DEBUG) {
		l.Println(l.formatMessage(DEBUG, format, v...))
	}
}

// Info logs an info message
func (l *Logger) Info(format string, v ...interface{}) {
	if l.shouldLog(INFO) {
		l.Println(l.formatMessage(INFO, format, v...))
	}
}

// Warn logs a warning message
func (l *Logger) Warn(format string, v ...interface{}) {
	if l.shouldLog(WARN) {
		l.Println(l.formatMessage(WARN, format, v...))
	}
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	if l.shouldLog(ERROR) {
		l.Println(l.formatMessage(ERROR, format, v...))
	}
}

// WithFields returns a new logger with the specified fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newLogger := &Logger{
		Logger: l.Logger,
		level:  l.level,
		fields: make(map[string]interface{}),
	}
	
	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	
	// Add new fields
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	
	return newLogger
}

// WithField returns a new logger with a single field added
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return l.WithFields(map[string]interface{}{key: value})
}

// SetLevel changes the log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// Global logger instance
var Log = NewLogger(INFO)
