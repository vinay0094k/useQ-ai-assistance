// Why this file: ./models/query_model.go
// This defines the data structures for user queries, including types (search, generation, debugging), context (current files, git info), and intent parsing.
// This structured approach enables intelligent query routing to appropriate agents.

package models

import (
	"time"
)

// QueryType represents different types of queries the system can handle
type QueryType string

const (
	QueryTypeSearch        QueryType = "search"
	QueryTypeGeneration    QueryType = "generation"
	QueryTypeExplanation   QueryType = "explanation"
	QueryTypeDebugging     QueryType = "debugging"
	QueryTypeTesting       QueryType = "testing"
	QueryTypeReview        QueryType = "review"
	QueryTypeRefactoring   QueryType = "refactoring"
	QueryTypeDocumentation QueryType = "documentation"
	QueryTypeSystem        QueryType = "system"
	QueryTypeRuntime       QueryType = "runtime"
	QueryTypeMonitoring    QueryType = "monitoring"
)

// Query represents a user query with context and metadata
type Query struct {
	ID          string            `json:"id"`
	UserInput   string            `json:"user_input"`
	Type        QueryType         `json:"type"`
	Language    string            `json:"language"`
	Context     QueryContext      `json:"context"`
	Intent      QueryIntent       `json:"intent"`
	Metadata    map[string]string `json:"metadata"`
	Timestamp   time.Time         `json:"timestamp"`
	SessionID   string            `json:"session_id"`
	ProjectRoot string            `json:"project_root"`
	MCPContext  *MCPContext       `json:"mcp_context,omitempty"`
}

// QueryContext holds contextual information for the query
type QueryContext struct {
	CurrentFile  string            `json:"current_file,omitempty"`
	CurrentLine  int               `json:"current_line,omitempty"`
	Selection    *TextSelection    `json:"selection,omitempty"`
	RecentFiles  []string          `json:"recent_files,omitempty"`
	ProjectFiles []string          `json:"project_files,omitempty"`
	Dependencies []string          `json:"dependencies,omitempty"`
	GitBranch    string            `json:"git_branch,omitempty"`
	GitCommit    string            `json:"git_commit,omitempty"`
	Environment  map[string]string `json:"environment,omitempty"`
}

// TextSelection represents selected text with position information
type TextSelection struct {
	Text      string `json:"text"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	StartCol  int    `json:"start_col"`
	EndCol    int    `json:"end_col"`
}

// QueryIntent represents the parsed intent of a user query
type QueryIntent struct {
	Primary     QueryType   `json:"primary"`
	Secondary   []QueryType `json:"secondary,omitempty"`
	Confidence  float64     `json:"confidence"`
	Keywords    []string    `json:"keywords"`
	Entities    []Entity    `json:"entities"`
	FileTargets []string    `json:"file_targets,omitempty"`
	FuncTargets []string    `json:"func_targets,omitempty"`
}

// Entity represents named entities found in the query
type Entity struct {
	Type  EntityType `json:"type"`
	Value string     `json:"value"`
	Start int        `json:"start"`
	End   int        `json:"end"`
}

// EntityType represents different types of entities
type EntityType string

const (
	EntityTypeFunction EntityType = "function"
	EntityTypeVariable EntityType = "variable"
	EntityTypeType     EntityType = "type"
	EntityTypePackage  EntityType = "package"
	EntityTypeFile     EntityType = "file"
	EntityTypeError    EntityType = "error"
	EntityTypeFeature  EntityType = "feature"
	EntityTypeLibrary  EntityType = "library"
)

// QueryRoutingResult represents how a query should be routed
type QueryRoutingResult struct {
	TargetAgent   string             `json:"target_agent"`
	Confidence    float64            `json:"confidence"`
	Alternatives  []AgentAlternative `json:"alternatives,omitempty"`
	RequiredTools []string           `json:"required_tools,omitempty"`
	EstimatedCost float64            `json:"estimated_cost"`
	EstimatedTime time.Duration      `json:"estimated_time"`
}

// AgentAlternative represents alternative agents that could handle the query
type AgentAlternative struct {
	AgentName  string  `json:"agent_name"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

// MCPContext holds MCP-related data for enhanced query processing
type MCPContext struct {
	RequiresMCP bool                   `json:"requires_mcp"`
	Operations  []string               `json:"operations"`
	Data        map[string]interface{} `json:"data"`
}

// MCPRequirements defines what MCP capabilities are needed
type MCPRequirements struct {
	NeedsFilesystem bool     `json:"needs_filesystem"`
	NeedsGit        bool     `json:"needs_git"`
	NeedsSQLite     bool     `json:"needs_sqlite"`
	FilePatterns    []string `json:"file_patterns"`
}
