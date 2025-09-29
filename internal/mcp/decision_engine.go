package mcp

import (
	"context"
	"strings"

	"github.com/yourusername/useq-ai-assistant/models"
)

// DecisionEngine analyzes queries and determines MCP operations needed
type DecisionEngine struct{}

// NewDecisionEngine creates a new decision engine
func NewDecisionEngine() *DecisionEngine {
	return &DecisionEngine{}
}

// AnalyzeQuery determines what MCP operations are needed for a query
func (de *DecisionEngine) AnalyzeQuery(ctx context.Context, query *models.Query) *models.MCPRequirements {
	requirements := &models.MCPRequirements{}
	
	input := strings.ToLower(query.UserInput)
	
	// File system operations
	if strings.Contains(input, "file") || strings.Contains(input, "search") || 
	   strings.Contains(input, "find") || strings.Contains(input, "code") {
		requirements.NeedsFilesystem = true
		requirements.FilePatterns = []string{"*.go", "*.js", "*.py", "*.md", "*.json"}
	}
	
	// Git operations
	if strings.Contains(input, "git") || strings.Contains(input, "commit") || 
	   strings.Contains(input, "branch") || strings.Contains(input, "history") {
		requirements.NeedsGit = true
	}
	
	// Database operations
	if strings.Contains(input, "database") || strings.Contains(input, "sql") || 
	   strings.Contains(input, "query") {
		requirements.NeedsSQLite = true
	}
	
	return requirements
}
