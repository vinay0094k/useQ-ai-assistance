// Why this file: ./models/response_model.go
// This defines comprehensive response structures including code generation, search results, file changes, suggestions, and quality metrics.
// It ensures responses are well-structured with proper metadata for display, cost tracking, and learning.
package models

import (
	"time"
)

// Response represents a structured response from the AI assistant
type Response struct {
	ID         string           `json:"id"`
	QueryID    string           `json:"query_id"`
	Type       ResponseType     `json:"type"`
	Content    ResponseContent  `json:"content"`
	Metadata   ResponseMetadata `json:"metadata"`
	TokenUsage TokenUsage       `json:"token_usage"`
	Cost       Cost             `json:"cost"`
	Timestamp  time.Time        `json:"timestamp"`
	AgentUsed  string           `json:"agent_used"`
	Provider   string           `json:"provider"`
	Quality    QualityMetrics   `json:"quality"`
}

// ResponseType defines different types of responses
type ResponseType string

const (
	ResponseTypeCode          ResponseType = "code"
	ResponseTypeExplanation   ResponseType = "explanation"
	ResponseTypeSearch        ResponseType = "search"
	ResponseTypeDocumentation ResponseType = "documentation"
	ResponseTypeError         ResponseType = "error"
	ResponseTypeDebug         ResponseType = "debug"
	ResponseTypeTest          ResponseType = "test"
	ResponseTypeRefactor      ResponseType = "refactor"
	ResponseTypeSuggestion    ResponseType = "suggestion"
	ResponseTypeSystem        ResponseType = "system"
)

// ResponseContent holds the actual content of the response
type ResponseContent struct {
	Text        string          `json:"text"`
	Code        *CodeResponse   `json:"code,omitempty"`
	Search      *SearchResponse `json:"search,omitempty"`
	Files       []FileChange    `json:"files,omitempty"`
	Suggestions []Suggestion    `json:"suggestions,omitempty"`
	References  []Reference     `json:"references,omitempty"`
	Errors      []ErrorDetail   `json:"errors,omitempty"`
}

// CodeResponse represents generated or modified code
type CodeResponse struct {
	Language     string          `json:"language"`
	Code         string          `json:"code"`
	Explanation  string          `json:"explanation,omitempty"`
	Changes      []CodeChange    `json:"changes,omitempty"`
	Tests        []TestCase      `json:"tests,omitempty"`
	Dependencies []Dependency    `json:"dependencies,omitempty"`
	Validation   *CodeValidation `json:"validation,omitempty"`
	Provider     string          `json:"provider,omitempty"`
	Context      string          `json:"context,omitempty"`
	Intent       interface{}     `json:"intent,omitempty"`
}

// CodeChange represents a specific change to code
type CodeChange struct {
	Type        ChangeType `json:"type"`
	File        string     `json:"file"`
	StartLine   int        `json:"start_line"`
	EndLine     int        `json:"end_line"`
	OldContent  string     `json:"old_content,omitempty"`
	NewContent  string     `json:"new_content"`
	Explanation string     `json:"explanation"`
}

// ChangeType defines types of code changes
type ChangeType string

const (
	ChangeTypeAdd     ChangeType = "add"
	ChangeTypeModify  ChangeType = "modify"
	ChangeTypeDelete  ChangeType = "delete"
	ChangeTypeReplace ChangeType = "replace"
)

// SearchResponse represents search results
type SearchResponse struct {
	Query     string         `json:"query"`
	Results   []SearchResult `json:"results"`
	Total     int            `json:"total"`
	TimeTaken time.Duration  `json:"time_taken"`
}

// SearchResult represents a single search result
type SearchResult struct {
	File        string         `json:"file"`
	Function    string         `json:"function,omitempty"`
	Line        int            `json:"line"`
	Score       float64        `json:"score"`
	Context     string         `json:"context"`
	Explanation string         `json:"explanation,omitempty"`
	Usage       []UsageExample `json:"usage,omitempty"`
}

// UsageExample shows how the found code is used
type UsageExample struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Context     string `json:"context"`
	Description string `json:"description"`
}

// FileChange represents changes to be made to files
type FileChange struct {
	Path    string       `json:"path"`
	Action  FileAction   `json:"action"`
	Content string       `json:"content,omitempty"`
	Changes []CodeChange `json:"changes,omitempty"`
	Backup  bool         `json:"backup"`
}

// FileAction defines what action to take on a file
type FileAction string

const (
	FileActionCreate FileAction = "create"
	FileActionModify FileAction = "modify"
	FileActionDelete FileAction = "delete"
	FileActionRename FileAction = "rename"
)

// Suggestion represents actionable suggestions
type Suggestion struct {
	Type        SuggestionType `json:"type"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Code        string         `json:"code,omitempty"`
	File        string         `json:"file,omitempty"`
	Line        int            `json:"line,omitempty"`
	Confidence  float64        `json:"confidence"`
}

// SuggestionType defines types of suggestions
type SuggestionType string

const (
	SuggestionTypeImprovement  SuggestionType = "improvement"
	SuggestionTypeOptimization SuggestionType = "optimization"
	SuggestionTypeBugFix       SuggestionType = "bugfix"
	SuggestionTypeSecurity     SuggestionType = "security"
	SuggestionTypeStyle        SuggestionType = "style"
	SuggestionTypePerformance  SuggestionType = "performance"
)

// Reference represents references to external resources
type Reference struct {
	Type        ReferenceType `json:"type"`
	Title       string        `json:"title"`
	URL         string        `json:"url,omitempty"`
	File        string        `json:"file,omitempty"`
	Line        int           `json:"line,omitempty"`
	Description string        `json:"description"`
}

// ReferenceType defines types of references
type ReferenceType string

const (
	ReferenceTypeDocumentation ReferenceType = "documentation"
	ReferenceTypeExample       ReferenceType = "example"
	ReferenceTypeLibrary       ReferenceType = "library"
	ReferenceTypeInternal      ReferenceType = "internal"
)

// ResponseMetadata holds metadata about the response
type ResponseMetadata struct {
	GenerationTime time.Duration `json:"generation_time"`
	IndexHits      int           `json:"index_hits"`
	FilesAnalyzed  int           `json:"files_analyzed"`
	Confidence     float64       `json:"confidence"`
	Sources        []string      `json:"sources"`
	Tools          []string      `json:"tools_used"`
	Reasoning      string        `json:"reasoning,omitempty"`
}

// QualityMetrics tracks response quality
type QualityMetrics struct {
	Accuracy     float64 `json:"accuracy"`
	Relevance    float64 `json:"relevance"`
	Completeness float64 `json:"completeness"`
	Clarity      float64 `json:"clarity"`
}

// TestCase represents generated test cases
type TestCase struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Type        string `json:"type"` // unit, integration, benchmark
}

// Dependency represents code dependencies
type Dependency struct {
	Name     string `json:"name"`
	Version  string `json:"version,omitempty"`
	Type     string `json:"type"` // import, module, library
	Required bool   `json:"required"`
}

// CodeValidation represents code validation results
type CodeValidation struct {
	IsValid  bool              `json:"is_valid"`
	Issues   []ValidationIssue `json:"issues"`
	Warnings []ValidationIssue `json:"warnings"`
	Score    float64           `json:"score"`
}

// ValidationIssue represents a single validation issue
type ValidationIssue struct {
	Type       string `json:"type"`
	Message    string `json:"message"`
	Line       int    `json:"line"`
	Severity   string `json:"severity"`
	Suggestion string `json:"suggestion"`
}

// ErrorDetail represents detailed error information
type ErrorDetail struct {
	Type       string `json:"type"`
	Message    string `json:"message"`
	File       string `json:"file,omitempty"`
	Line       int    `json:"line,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}
