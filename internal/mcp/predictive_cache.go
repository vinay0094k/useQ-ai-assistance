package mcp

import (
	"context"
	"sync"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// PredictiveCache manages intelligent pre-caching based on usage patterns
type PredictiveCache struct {
	contextCache  *MCPContextCache
	usageTracker  *UsageTracker
	mcpClient     *MCPClient
	preCacheQueue chan string
	mu            sync.RWMutex
}

// NewPredictiveCache creates a new predictive cache
func NewPredictiveCache(contextCache *MCPContextCache, usageTracker *UsageTracker, mcpClient *MCPClient) *PredictiveCache {
	pc := &PredictiveCache{
		contextCache:  contextCache,
		usageTracker:  usageTracker,
		mcpClient:     mcpClient,
		preCacheQueue: make(chan string, 10),
	}
	
	go pc.preCacheWorker()
	
	return pc
}

// GetOrPredict gets cached context or predicts and creates it
func (pc *PredictiveCache) GetOrPredict(ctx context.Context, query *models.Query) (*models.MCPContext, error) {
	projectPath := pc.getProjectPath(query)
	
	// Try cache first
	if cached, found := pc.contextCache.Get(projectPath); found {
		return cached, nil
	}
	
	// Use predicted operations for faster context creation
	predictedOps := pc.usageTracker.PredictOperations(query)
	
	start := time.Now()
	mcpContext, err := pc.createPredictiveContext(ctx, query, predictedOps)
	responseTime := time.Since(start)
	
	if err != nil {
		return nil, err
	}
	
	// Record usage for learning
	pc.usageTracker.RecordUsage(query, mcpContext.Operations, responseTime)
	
	// Cache with adaptive TTL
	adaptiveTTL := pc.usageTracker.GetAdaptiveTTL(projectPath)
	pc.cacheWithTTL(projectPath, mcpContext, adaptiveTTL)
	
	// Queue for pre-caching if frequently used
	if pc.usageTracker.ShouldPreCache(projectPath) {
		select {
		case pc.preCacheQueue <- projectPath:
		default: // Queue full, skip
		}
	}
	
	return mcpContext, nil
}

// createPredictiveContext creates context using predicted operations
func (pc *PredictiveCache) createPredictiveContext(ctx context.Context, query *models.Query, predictedOps []string) (*models.MCPContext, error) {
	mcpContext := &models.MCPContext{
		RequiresMCP: true,
		Operations:  []string{},
		Data:        make(map[string]interface{}),
	}
	
	// Execute only predicted operations for faster response
	for _, op := range predictedOps {
		switch op {
		case "filesystem_search":
			if files, err := pc.mcpClient.filesystemServer.SearchFiles([]string{"*.go"}, ""); err == nil {
				mcpContext.Operations = append(mcpContext.Operations, "filesystem_search")
				mcpContext.Data["project_files"] = files
				mcpContext.Data["file_count"] = len(files)
			}
			
		case "project_structure":
			if structure, err := pc.mcpClient.filesystemServer.GetProjectStructure(3); err == nil {
				mcpContext.Operations = append(mcpContext.Operations, "project_structure")
				mcpContext.Data["project_structure"] = structure
			}
			
		case "git_context":
			gitInfo := pc.mcpClient.executor.getGitInfo()
			mcpContext.Operations = append(mcpContext.Operations, "git_context")
			mcpContext.Data["git_info"] = gitInfo
		}
	}
	
	return mcpContext, nil
}

// preCacheWorker runs background pre-caching
func (pc *PredictiveCache) preCacheWorker() {
	for projectPath := range pc.preCacheQueue {
		pc.preCacheProject(projectPath)
	}
}

// preCacheProject pre-caches a frequently used project
func (pc *PredictiveCache) preCacheProject(projectPath string) {
	// Check if already cached
	if _, found := pc.contextCache.Get(projectPath); found {
		return
	}
	
	// Create dummy query for pre-caching
	query := &models.Query{
		ID:          "precache-" + projectPath,
		UserInput:   "precache",
		Type:        models.QueryTypeGeneration,
		ProjectRoot: projectPath,
		Timestamp:   time.Now(),
	}
	
	ctx := context.Background()
	predictedOps := pc.usageTracker.PredictOperations(query)
	
	if mcpContext, err := pc.createPredictiveContext(ctx, query, predictedOps); err == nil {
		adaptiveTTL := pc.usageTracker.GetAdaptiveTTL(projectPath)
		pc.cacheWithTTL(projectPath, mcpContext, adaptiveTTL)
	}
}

// cacheWithTTL caches context with custom TTL
func (pc *PredictiveCache) cacheWithTTL(projectPath string, context *models.MCPContext, ttl time.Duration) {
	fileCount := 0
	if count, ok := context.Data["file_count"].(int); ok {
		fileCount = count
	}
	
	hash := pc.generateContextHash(context)
	pc.contextCache.Set(projectPath, context, fileCount, hash)
}

// generateContextHash creates a hash for context comparison
func (pc *PredictiveCache) generateContextHash(context *models.MCPContext) string {
	hash := ""
	for _, op := range context.Operations {
		hash += op + ":"
	}
	if count, ok := context.Data["file_count"].(int); ok {
		hash += string(rune(count))
	}
	return hash
}

func (pc *PredictiveCache) getProjectPath(query *models.Query) string {
	if query.ProjectRoot != "" {
		return query.ProjectRoot
	}
	return "."
}
