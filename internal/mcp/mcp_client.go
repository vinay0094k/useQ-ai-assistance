package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// MCPClient coordinates with MCP servers
type MCPClient struct {
	queryClassifier     *QueryClassifier
	tierProcessor       *TierProcessor
	decisionEngine   *DecisionEngine
	executor         *Executor
	intelligentExecutor *IntelligentExecutor
	filesystemServer *FilesystemServer
	contextCache     *MCPContextCache
	fileWatcher      *FileWatcher
	usageTracker     *UsageTracker
	predictiveCache  *PredictiveCache
}

// NewMCPClient creates a new MCP client
func NewMCPClient() *MCPClient {
	// Initialize the 3-tier classification system
	classifier := NewQueryClassifier()
	
	cache := NewMCPContextCache(5 * time.Minute) // 5 minute TTL
	watcher, _ := NewFileWatcher(cache)
	usageTracker := NewUsageTracker()
	
	client := &MCPClient{
		queryClassifier:     classifier,
		tierProcessor:       NewTierProcessor(nil, nil), // Will be set by dependencies
		intelligentExecutor: NewIntelligentExecutor(),
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

// GetQueryClassifier returns the query classifier for external access
func (mc *MCPClient) GetQueryClassifier() *QueryClassifier {
	return mc.queryClassifier
}

// ProcessQuery processes a query through MCP pipeline
func (mc *MCPClient) ProcessQuery(ctx context.Context, query *models.Query) (*models.MCPContext, error) {
	// STEP 1: Classify query into 3 tiers
	classification, err := mc.queryClassifier.ClassifyQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query classification failed: %w", err)
	}
	
	// STEP 2: Process based on tier
	switch classification.Tier {
	case TierSimple:
		// Direct MCP execution - no LLM needed
		return mc.processTier1Query(ctx, query, classification)
	case TierMedium:
		// MCP + Vector search - no LLM needed
		return mc.processTier2Query(ctx, query, classification)
	case TierComplex:
		// Full pipeline with LLM
		return mc.processTier3Query(ctx, query, classification)
	default:
		return mc.intelligentExecutor.AnalyzeAndExecute(ctx, query)
	}
}

// processTier1Query handles simple queries with direct MCP
func (mc *MCPClient) processTier1Query(ctx context.Context, query *models.Query, classification *ClassificationResult) (*models.MCPContext, error) {
	// Execute filesystem operations directly
	operations := classification.RequiredOperations
	data := make(map[string]interface{})
	
	for _, operation := range operations {
		switch operation {
		case "filesystem_list":
			if files, err := mc.filesystemServer.SearchFiles([]string{"*.go"}, ""); err == nil {
				data["files"] = files
				data["file_count"] = len(files)
			}
		case "filesystem_tree":
			if structure, err := mc.filesystemServer.GetProjectStructure(3); err == nil {
				data["project_structure"] = structure
			}
		case "system_info":
			data["system_info"] = mc.getSystemInfo()
		}
	}
	
	return &models.MCPContext{
		RequiresMCP: true,
		Operations:  operations,
		Data:        data,
	}, nil
}

// processTier2Query handles medium queries with MCP + Vector
func (mc *MCPClient) processTier2Query(ctx context.Context, query *models.Query, classification *ClassificationResult) (*models.MCPContext, error) {
	// Similar to Tier 1 but add vector search results
	mcpContext, err := mc.processTier1Query(ctx, query, classification)
	if err != nil {
		return nil, err
	}
	
	// Add vector search placeholder (would integrate with actual vector DB)
	mcpContext.Data["vector_search"] = map[string]interface{}{
		"query": query.UserInput,
		"note":  "Vector search results would be added here",
	}
	
	return mcpContext, nil
}

// processTier3Query handles complex queries with full pipeline
func (mc *MCPClient) processTier3Query(ctx context.Context, query *models.Query, classification *ClassificationResult) (*models.MCPContext, error) {
	// Use existing intelligent executor for complex processing
	return mc.intelligentExecutor.AnalyzeAndExecute(ctx, query)
}

// getSystemInfo gets basic system information
func (mc *MCPClient) getSystemInfo() map[string]interface{} {
	return map[string]interface{}{
		"timestamp": time.Now(),
		"status":    "running",
	}
}

// SetDependencies allows setting vector DB and LLM manager
func (mc *MCPClient) SetDependencies(vectorDB VectorDBInterface, llmManager LLMManagerInterface) {
	if mc.tierProcessor != nil {
		mc.tierProcessor.vectorDB = vectorDB
		mc.tierProcessor.llmManager = llmManager
	}
}

// GetClassificationStats returns classification statistics
func (mc *MCPClient) GetClassificationStats() *ClassificationStats {
	return mc.queryClassifier.GetStats()
}

// PrintClassificationStats prints classification statistics
func (mc *MCPClient) PrintClassificationStats() {
	mc.queryClassifier.PrintStats()
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
