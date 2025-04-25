package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger provides a structured logging interface
type Logger struct {
	*zap.SugaredLogger
}

// New creates a new logger with the given name
func New(name string) *Logger {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		fmt.Printf("Error creating logs directory: %v\n", err)
	}

	// Get current timestamp for log file
	timestamp := time.Now().Format("20060102")
	logFile := filepath.Join("logs", fmt.Sprintf("%s_%s.log", name, timestamp))

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create file core
	fileWriter, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening log file: %v\n", err)
	}

	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(fileWriter),
		zap.InfoLevel,
	)

	// Create console core
	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		zap.InfoLevel,
	)

	// Combine cores
	core := zapcore.NewTee(fileCore, consoleCore)

	// Create logger
	logger := zap.New(core).Named(name)

	return &Logger{
		SugaredLogger: logger.Sugar(),
	}
}