package models

import "time"

// MCPOperationType defines the type of MCP operation
type MCPOperationType string

const (
	MCPOperationTypeFileRead     MCPOperationType = "file_read"
	MCPOperationTypeFileWrite    MCPOperationType = "file_write"
	MCPOperationTypeFileSearch   MCPOperationType = "file_search"
	MCPOperationTypeDBQuery      MCPOperationType = "db_query"
	MCPOperationTypeDBSchema     MCPOperationType = "db_schema"
	MCPOperationTypeCodeAnalysis MCPOperationType = "code_analysis"
	
	// Filesystem operations
	MCPOperationFilesystemSearch MCPOperationType = "filesystem_search"
	MCPOperationFilesystemRead   MCPOperationType = "filesystem_read"
	MCPOperationFilesystemWrite  MCPOperationType = "filesystem_write"
	MCPOperationFilesystemList   MCPOperationType = "filesystem_list"
	
	// SQLite operations
	MCPOperationSQLiteQuery      MCPOperationType = "sqlite_query"
	MCPOperationSQLiteSchema     MCPOperationType = "sqlite_schema"
	MCPOperationSQLiteTables     MCPOperationType = "sqlite_tables"
	
	// Git operations
	MCPOperationGitSearch        MCPOperationType = "git_search"
	MCPOperationGitLog           MCPOperationType = "git_log"
)

// MCPOperation represents a single MCP operation to be executed
type MCPOperation struct {
	ID         string                 `json:"id"`
	Type       MCPOperationType       `json:"type"`
	ServerName string                 `json:"server_name"`
	Method     string                 `json:"method"`
	Parameters map[string]interface{} `json:"parameters"`
	Priority   int                    `json:"priority"`
	Timeout    time.Duration          `json:"timeout"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// MCPResult represents the result of an MCP operation
type MCPResult struct {
	OperationID string                 `json:"operation_id"`
	Success     bool                   `json:"success"`
	Data        interface{}            `json:"data"`
	Error       string                 `json:"error,omitempty"`
	Duration    time.Duration          `json:"duration"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Context represents execution context for MCP operations
type Context struct {
	UserID      string                 `json:"user_id"`
	SessionID   string                 `json:"session_id"`
	RequestID   string                 `json:"request_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Environment string                 `json:"environment"`
	Query       QueryContext           `json:"query_context"`
	Metadata    map[string]interface{} `json:"metadata"`
}
