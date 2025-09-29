package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type ExecutionTracer struct {
	file      *os.File
	queryID   string
	startTime time.Time
	depth     int
}

type ExecutionStep struct {
	Timestamp time.Time `json:"timestamp"`
	QueryID   string    `json:"query_id"`
	File      string    `json:"file"`
	Function  string    `json:"function"`
	Line      int       `json:"line"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	Depth     int       `json:"depth"`
	Duration  string    `json:"duration,omitempty"`
}

func NewExecutionTracer(queryID string) (*ExecutionTracer, error) {
	logDir := "./logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}

	filename := fmt.Sprintf("steps_%s.log", time.Now().Format("2006-01-02"))
	logFile := filepath.Join(logDir, filename)

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	tracer := &ExecutionTracer{
		file:      file,
		queryID:   queryID,
		startTime: time.Now(),
		depth:     0,
	}

	// Write header if this is a new file
	if stat, err := file.Stat(); err == nil && stat.Size() == 0 {
		header := fmt.Sprintf("=== useQ AI Assistant - Step-by-Step Execution Log ===\n")
		header += fmt.Sprintf("Date: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		header += fmt.Sprintf("Format: [timestamp] depth symbol action | file:line function() | operation | details\n")
		header += fmt.Sprintf("Symbols: ▶=start ◀=end ┌─=enter └─=exit →=step 📁=file 🗄️=db 🔍=search 🤖=llm 💬=response\n")
		header += fmt.Sprintf("%s\n\n", strings.Repeat("=", 80))
		file.WriteString(header)
	}

	// Log query start
	tracer.LogStart("USER_QUERY_RECEIVED")

	return tracer, nil
}

func (et *ExecutionTracer) LogStart(action string) {
	et.logExecution("START", action, "")
}

func (et *ExecutionTracer) LogFileAccess(filePath, operation string) {
	et.logExecution("FILE_ACCESS", operation, fmt.Sprintf("File: %s", filePath))
}

func (et *ExecutionTracer) LogFunctionCall(functionName, details string) {
	et.depth++
	et.logExecution("FUNCTION_ENTER", functionName, details)
}

func (et *ExecutionTracer) LogFunctionExit(functionName, result string) {
	et.logExecution("FUNCTION_EXIT", functionName, result)
	et.depth--
}

func (et *ExecutionTracer) LogStep(stepName, details string) {
	et.logExecution("STEP", stepName, details)
}

func (et *ExecutionTracer) LogFileRead(filePath string, success bool) {
	status := "SUCCESS"
	if !success {
		status = "FAILED"
	}
	et.logExecution("FILE_READ", status, fmt.Sprintf("File: %s", filePath))
}

func (et *ExecutionTracer) LogFileWrite(filePath string, success bool) {
	status := "SUCCESS"
	if !success {
		status = "FAILED"
	}
	et.logExecution("FILE_WRITE", status, fmt.Sprintf("File: %s", filePath))
}

func (et *ExecutionTracer) LogDatabaseQuery(query, table string) {
	et.logExecution("DB_QUERY", "SQL_EXECUTION", fmt.Sprintf("Table: %s, Query: %s", table, query))
}

func (et *ExecutionTracer) LogVectorSearch(query string, results int) {
	et.logExecution("VECTOR_SEARCH", "SEMANTIC_SEARCH", fmt.Sprintf("Query: %s, Results: %d", query, results))
}

func (et *ExecutionTracer) LogLLMCall(provider, model, prompt string) {
	et.logExecution("LLM_CALL", "AI_REQUEST", fmt.Sprintf("Provider: %s, Model: %s, Prompt: %s", provider, model, truncate(prompt, 100)))
}

func (et *ExecutionTracer) LogLLMResponse(provider string, tokens int, cost float64) {
	et.logExecution("LLM_RESPONSE", "AI_RESPONSE", fmt.Sprintf("Provider: %s, Tokens: %d, Cost: $%.4f", provider, tokens, cost))
}

func (et *ExecutionTracer) LogEnd(result string) {
	duration := time.Since(et.startTime)
	et.logExecution("END", "QUERY_COMPLETED", fmt.Sprintf("Result: %s, Total Duration: %v", result, duration))
}

func (et *ExecutionTracer) logExecution(action, operation, details string) {
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	}

	// Get function name
	funcName := "unknown"
	if fn := runtime.FuncForPC(pc); fn != nil {
		funcName = fn.Name()
		// Extract just the function name without package path
		if idx := strings.LastIndex(funcName, "."); idx != -1 {
			funcName = funcName[idx+1:]
		}
	}

	// Get relative file path
	if idx := strings.LastIndex(file, "/useq-ai-assistance/"); idx != -1 {
		file = file[idx+len("/useq-ai-assistance/"):]
	}

	timestamp := time.Now().Format("15:04:05.000")
	indent := strings.Repeat("│  ", et.depth)

	// Different symbols for different actions
	symbol := "•"
	switch action {
	case "START":
		symbol = "▶"
	case "END":
		symbol = "◀"
	case "FUNCTION_ENTER":
		symbol = "┌─"
	case "FUNCTION_EXIT":
		symbol = "└─"
	case "FILE_READ", "FILE_WRITE":
		symbol = "📁"
	case "STEP":
		symbol = "→"
	case "DB_QUERY":
		symbol = "🗄️"
	case "VECTOR_SEARCH":
		symbol = "🔍"
	case "LLM_CALL":
		symbol = "🤖"
	case "LLM_RESPONSE":
		symbol = "💬"
	}

	logLine := fmt.Sprintf("[%s] %s%s %s | %s:%d %s() | %s | %s\n",
		timestamp, indent, symbol, action, file, line, funcName, operation, details)

	if et.file != nil {
		et.file.WriteString(logLine)
		et.file.Sync()
	}
}

func (et *ExecutionTracer) Close() error {
	if et.file != nil {
		return et.file.Close()
	}
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
