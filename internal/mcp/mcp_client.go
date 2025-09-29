package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// MCPClient coordinates with MCP servers
type MCPClient struct {
	decisionEngine   *DecisionEngine
	executor         *Executor
	filesystemServer *FilesystemServer
	contextCache     *MCPContextCache
	fileWatcher      *FileWatcher
	usageTracker     *UsageTracker
	predictiveCache  *PredictiveCache
}

// NewMCPClient creates a new MCP client
func NewMCPClient() *MCPClient {
	cache := NewMCPContextCache(5 * time.Minute) // 5 minute TTL
	watcher, _ := NewFileWatcher(cache)
	usageTracker := NewUsageTracker()
	
	client := &MCPClient{
		decisionEngine:   NewDecisionEngine(),
		executor:         NewExecutor(),
		filesystemServer: NewFilesystemServer(),
		contextCache:     cache,
		fileWatcher:      watcher,
		usageTracker:     usageTracker,
	}
	
	// Initialize predictive cache
	client.predictiveCache = NewPredictiveCache(cache, usageTracker, client)
	
	return client
}

// ProcessQuery processes a query through MCP pipeline
func (mc *MCPClient) ProcessQuery(ctx context.Context, query *models.Query) (*models.MCPContext, error) {
	// Use predictive cache for intelligent caching and learning
	return mc.predictiveCache.GetOrPredict(ctx, query)
}

// getProjectPath extracts project path from query context
func (mc *MCPClient) getProjectPath(query *models.Query) string {
	// Use ProjectRoot if available
	if query.ProjectRoot != "" {
		return query.ProjectRoot
	}
	
	// Check environment for project path
	if query.Context.Environment != nil {
		if path, ok := query.Context.Environment["project_path"]; ok {
			return path
		}
	}
	
	// Default to current directory
	return "."
}

// generateContextHash creates a hash for context comparison
func (mc *MCPClient) generateContextHash(context *models.MCPContext) string {
	// Simple hash based on operations and file count
	hash := ""
	for _, op := range context.Operations {
		hash += op + ":"
	}
	if count, ok := context.Data["file_count"].(int); ok {
		hash += fmt.Sprintf("files:%d", count)
	}
	return hash
}

// GetCacheStats returns cache statistics
func (mc *MCPClient) GetCacheStats() map[string]interface{} {
	return mc.contextCache.GetStats()
}

// InvalidateCache manually invalidates cache for a project
func (mc *MCPClient) InvalidateCache(projectPath string) {
	mc.contextCache.Invalidate(projectPath)
}

// GetUsageStats returns usage pattern statistics
func (mc *MCPClient) GetUsageStats() map[string]interface{} {
	return mc.usageTracker.GetStats()
}

// GetLearningInsights returns insights from usage patterns
func (mc *MCPClient) GetLearningInsights() map[string]interface{} {
	usageStats := mc.usageTracker.GetStats()
	cacheStats := mc.contextCache.GetStats()
	
	return map[string]interface{}{
		"usage_patterns": usageStats,
		"cache_performance": cacheStats,
		"learning_enabled": true,
		"predictive_caching": true,
	}
}
