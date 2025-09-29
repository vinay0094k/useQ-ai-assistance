package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type ConsoleLogger struct {
	file *os.File
}

func NewConsoleLogger() (*ConsoleLogger, error) {
	logDir := "./logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}

	filename := fmt.Sprintf("console_logs_%s.log", time.Now().Format("2006-01-02"))
	logFile := filepath.Join(logDir, filename)

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &ConsoleLogger{file: file}, nil
}

func (cl *ConsoleLogger) Log(message string) {
	timestamp := time.Now().Format("15:04:05")
	logLine := fmt.Sprintf("[%s] %s\n", timestamp, message)

	// Write to console
	fmt.Print(logLine)

	// Write to file
	if cl.file != nil {
		cl.file.WriteString(logLine)
	}
}

func (cl *ConsoleLogger) Close() error {
	if cl.file != nil {
		return cl.file.Close()
	}
	return nil
}
