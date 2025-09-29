// Why this file: ./internal/logger/step_logger.go
// This implements the step-by-step logging system for debugging.
// It provides real-time console output with icons (🔄, ✅, ❌), file logging, execution summaries, and detailed tracking of each component's actions - essential for debugging the flow.
package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// StepLogger provides detailed step-by-step logging for debugging
type StepLogger struct {
	logger        *zap.Logger
	stepCounter   int
	sessionID     string
	queryID       string
	startTime     time.Time
	steps         []LogStep
	mu            sync.RWMutex
	enableConsole bool
	enableFile    bool
	logLevel      zapcore.Level
}

// LogStep represents a single step in the execution flow
type LogStep struct {
	StepNumber int                    `json:"step_number"`
	Component  string                 `json:"component"`
	Action     string                 `json:"action"`
	Status     StepStatus             `json:"status"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    *time.Time             `json:"end_time,omitempty"`
	Duration   time.Duration          `json:"duration,omitempty"`
	Details    interface{}            `json:"details,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// StepStatus represents the status of a step
type StepStatus string

const (
	StatusStarted    StepStatus = "started"
	StatusInProgress StepStatus = "in_progress"
	StatusCompleted  StepStatus = "completed"
	StatusFailed     StepStatus = "failed"
	StatusSkipped    StepStatus = "skipped"
)

// Component represents different system components
type Component string

const (
	ComponentCLI      Component = "cli"
	ComponentParser   Component = "parser"
	ComponentIndexer  Component = "indexer"
	ComponentVectorDB Component = "vectordb"
	ComponentLLM      Component = "llm"
	ComponentAgent    Component = "agent"
	ComponentMCP      Component = "mcp"
	ComponentDisplay  Component = "display"
	ComponentFeedback Component = "feedback"
	ComponentCache    Component = "cache"
)

// NewStepLogger creates a new step logger instance
func NewStepLogger(sessionID, queryID string, logLevel string, enableConsole, enableFile bool) (*StepLogger, error) {
	level := zapcore.InfoLevel
	switch strings.ToLower(logLevel) {
	case "debug":
		level = zapcore.DebugLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	}

	config := zap.NewProductionConfig()
	config.Level.SetLevel(level)
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	var outputs []string
	if enableConsole {
		outputs = append(outputs, "stdout")
	}
	if enableFile {
		logDir := "./logs"
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
		logFile := filepath.Join(logDir, fmt.Sprintf("steps_%s.log", time.Now().Format("2006-01-02")))
		outputs = append(outputs, logFile)
	}
	config.OutputPaths = outputs

	logger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return &StepLogger{
		logger:        logger,
		sessionID:     sessionID,
		queryID:       queryID,
		startTime:     time.Now(),
		steps:         make([]LogStep, 0),
		enableConsole: enableConsole,
		enableFile:    enableFile,
		logLevel:      level,
	}, nil
}

// StartStep begins a new step in the execution flow
func (sl *StepLogger) StartStep(component Component, action string, details interface{}) int {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	sl.stepCounter++
	step := LogStep{
		StepNumber: sl.stepCounter,
		Component:  string(component),
		Action:     action,
		Status:     StatusStarted,
		StartTime:  time.Now(),
		Details:    details,
		Metadata:   make(map[string]interface{}),
	}

	sl.steps = append(sl.steps, step)

	// Log to console/file
	// JSON logs disabled for console - only file logging

	// Console output disabled - logs go to file only

	return sl.stepCounter
}

// UpdateStep updates an existing step with progress information
func (sl *StepLogger) UpdateStep(stepNumber int, status StepStatus, details interface{}, metadata map[string]interface{}) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	if stepNumber <= 0 || stepNumber > len(sl.steps) {
		return
	}

	step := &sl.steps[stepNumber-1]
	step.Status = status
	if details != nil {
		step.Details = details
	}
	if metadata != nil {
		for k, v := range metadata {
			step.Metadata[k] = v
		}
	}

	sl.logger.Info("Step updated",
		zap.String("session_id", sl.sessionID),
		zap.String("query_id", sl.queryID),
		zap.Int("step", stepNumber),
		zap.String("component", step.Component),
		zap.String("action", step.Action),
		zap.String("status", string(status)),
		zap.Any("details", details),
		zap.Any("metadata", metadata),
	)

	// Console output disabled - logs go to file only
}

// CompleteStep marks a step as completed
func (sl *StepLogger) CompleteStep(stepNumber int, result interface{}) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	if stepNumber <= 0 || stepNumber > len(sl.steps) {
		return
	}

	step := &sl.steps[stepNumber-1]
	now := time.Now()
	step.Status = StatusCompleted
	step.EndTime = &now
	step.Duration = now.Sub(step.StartTime)
	if result != nil {
		step.Details = result
	}

	sl.logger.Info("Step completed",
		zap.String("session_id", sl.sessionID),
		zap.String("query_id", sl.queryID),
		zap.Int("step", stepNumber),
		zap.String("component", step.Component),
		zap.String("action", step.Action),
		zap.Duration("duration", step.Duration),
		zap.Any("result", result),
	)

	// Completion output disabled - logs go to file only
}

// FailStep marks a step as failed
func (sl *StepLogger) FailStep(stepNumber int, err error) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	if stepNumber <= 0 || stepNumber > len(sl.steps) {
		return
	}

	step := &sl.steps[stepNumber-1]
	now := time.Now()
	step.Status = StatusFailed
	step.EndTime = &now
	step.Duration = now.Sub(step.StartTime)
	if err != nil {
		step.Error = err.Error()
	}

	sl.logger.Error("Step failed",
		zap.String("session_id", sl.sessionID),
		zap.String("query_id", sl.queryID),
		zap.Int("step", stepNumber),
		zap.String("component", step.Component),
		zap.String("action", step.Action),
		zap.Duration("duration", step.Duration),
		zap.Error(err),
	)

	// Failure output disabled - logs go to file only
}

// LogInfo logs an informational message
func (sl *StepLogger) LogInfo(component Component, message string, fields ...interface{}) {
	sl.logger.Info(message,
		zap.String("session_id", sl.sessionID),
		zap.String("query_id", sl.queryID),
		zap.String("component", string(component)),
		zap.Any("data", fields),
	)

	// Info output disabled - logs go to file only
}

// LogError logs an error message
func (sl *StepLogger) LogError(component Component, message string, err error, fields ...interface{}) {
	sl.logger.Error(message,
		zap.String("session_id", sl.sessionID),
		zap.String("query_id", sl.queryID),
		zap.String("component", string(component)),
		zap.Error(err),
		zap.Any("data", fields),
	)

	if sl.enableConsole {
		fmt.Printf("🚨 [%s] %s: %v\n", component, message, err)
	}
}

// GetExecutionSummary returns a summary of all executed steps
func (sl *StepLogger) GetExecutionSummary() ExecutionSummary {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	summary := ExecutionSummary{
		SessionID:  sl.sessionID,
		QueryID:    sl.queryID,
		StartTime:  sl.startTime,
		EndTime:    time.Now(),
		TotalSteps: len(sl.steps),
		Steps:      make([]LogStep, len(sl.steps)),
	}

	copy(summary.Steps, sl.steps)
	summary.Duration = summary.EndTime.Sub(summary.StartTime)

	// Calculate statistics
	for _, step := range sl.steps {
		switch step.Status {
		case StatusCompleted:
			summary.CompletedSteps++
		case StatusFailed:
			summary.FailedSteps++
		case StatusSkipped:
			summary.SkippedSteps++
		}
	}

	return summary
}

// ExportSteps exports all steps to JSON format
func (sl *StepLogger) ExportSteps(filename string) error {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	summary := sl.GetExecutionSummary()
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal steps: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write steps file: %w", err)
	}

	return nil
}

// Close closes the logger
func (sl *StepLogger) Close() error {
	return sl.logger.Sync()
}

// ExecutionSummary provides a summary of the execution flow
type ExecutionSummary struct {
	SessionID      string        `json:"session_id"`
	QueryID        string        `json:"query_id"`
	StartTime      time.Time     `json:"start_time"`
	EndTime        time.Time     `json:"end_time"`
	Duration       time.Duration `json:"duration"`
	TotalSteps     int           `json:"total_steps"`
	CompletedSteps int           `json:"completed_steps"`
	FailedSteps    int           `json:"failed_steps"`
	SkippedSteps   int           `json:"skipped_steps"`
	Steps          []LogStep     `json:"steps"`
}
