package agent

/*
	Provide a basic logger for the agent. The logger will have the following features:
	- Log levels: INFO, WARNING, ERROR, DEBUG
	- Log to stdout
	- Log to file
*/

import (
	"fmt"
	"os"
	"time"
)

// Logger is the logger for the agent
type Logger struct {
	logFile *os.File
}

// the global logger
var logger *Logger

func GetLogger() *Logger {
	if logger == nil {
		logger, _ = NewLogger("")
	}
	return logger
}

// NewLogger creates a new logger
func NewLogger(logFile string) (*Logger, error) {
	// Create the log file if provided, otherwise log to stdout

	if logFile == "" {
		return &Logger{}, nil
	}

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	return &Logger{logFile: f}, nil
}

// Close closes the logger
func (l *Logger) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...interface{}) {
	l.log("INFO", msg, args...)
}

// Warning logs a warning message
func (l *Logger) Warning(msg string, args ...interface{}) {
	l.log("WARNING", msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	l.log("ERROR", msg, args...)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.log("DEBUG", msg, args...)
}

func (l *Logger) log(level string, msg string, args ...interface{}) {
	now := time.Now().Format(time.RFC3339)

	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	if l.logFile != nil {
		fmt.Fprintf(l.logFile, "[%s] [%s] %s\n", now, level, msg)
	} else {
		fmt.Fprintf(os.Stdout, "[%s] [%s] %s\n", now, level, msg)
	}
}
