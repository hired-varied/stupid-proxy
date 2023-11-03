package utils

import (
	"fmt"
	"log"
	"os"
)

const (
	// DebugLevel log level
	DebugLevel = 0
	// InfoLevel log level
	InfoLevel = 1
	// WarningLevel log level
	WarningLevel = 2
	// ErrorLevel log level
	ErrorLevel = 3
)

// Logger is a custom logger
type Logger struct {
	level int
}

// NewLogger a new Logger
func NewLogger(level int) *Logger {
	return &Logger{level}
}

var std = log.New(os.Stderr, "", log.LstdFlags)

// Debug log
func (l *Logger) Debug(format string, v ...interface{}) {
	if l.level > DebugLevel {
		return
	}
	std.Output(2, fmt.Sprintf(format, v...))
}

// Info log
func (l *Logger) Info(format string, v ...interface{}) {
	if l.level > InfoLevel {
		return
	}
	std.Output(2, fmt.Sprintf(format, v...))
}

// Warning log
func (l *Logger) Warning(format string, v ...interface{}) {
	if l.level > WarningLevel {
		return
	}
	std.Output(2, fmt.Sprintf(format, v...))
}

// Error log
func (l *Logger) Error(format string, v ...interface{}) {
	if l.level > ErrorLevel {
		return
	}
	std.Output(2, fmt.Sprintf(format, v...))
}
