package mcp

import (
	"context"
	"sync"
	"time"
)

// ParallelContextGatherer gathers context from multiple sources in parallel
type ParallelContextGatherer struct {
	filesystemServer *FilesystemServer
	commandExecutor  *IntelligentExecutor
	cache           *MCPContextCache
	usageTracker    *UsageTracker
}

// NewParallelContextGatherer creates a new parallel context gatherer
func NewParallelContextGatherer() *ParallelContextGatherer {
	return &ParallelContextGatherer{
		filesystemServer: NewFilesystemServer(),
		commandExecutor:  NewIntelligentExecutor(),
		cache:           NewMCPContextCache(15 * time.Minute),
		usageTracker:    NewUsageTracker(),
	}
}

// GatherContext gathers context from multiple sources in parallel
func (pcg *ParallelContextGatherer) GatherContext(ctx context.Context, plan *QueryProcessingPlan) (*GatheredContext, error) {
	// Create context for parallel operations
	gatherCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	// Create channels for results
	projectInfoChan := make(chan map[string]interface{}, 1)
	filesChan := make(chan []string, 1)
	systemInfoChan := make(chan map[string]interface{}, 1)
	codeExamplesChan := make(chan []string, 1)
	
	var wg sync.WaitGroup
	
	// Parallel operation 1: Get project structure
	wg.Add(1)
	go func() {
		defer wg.Done()
		info := pcg.getProjectInfo(gatherCtx, plan)
		select {
		case projectInfoChan <- info:
		case <-gatherCtx.Done():
		}
	}()
	
	// Parallel operation 2: Get relevant files
	wg.Add(1)
	go func() {
		defer wg.Done()
		files := pcg.getRelevantFiles(gatherCtx, plan)
		select {
		case filesChan <- files:
		case <-gatherCtx.Done():
		}
	}()
	
	// Parallel operation 3: Get system info (if needed)
	if pcg.needsSystemInfo(plan) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			info := pcg.getSystemInfo(gatherCtx, plan)
			select {
			case systemInfoChan <- info:
			case <-gatherCtx.Done():
			}
		}()
	} else {
		systemInfoChan <- map[string]interface{}{}
	}
	
	// Parallel operation 4: Get code examples (if needed)
	if pcg.needsCodeExamples(plan) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			examples := pcg.getCodeExamples(gatherCtx, plan)
			select {
			case codeExamplesChan <- examples:
			case <-gatherCtx.Done():
			}
		}()
	} else {
		codeExamplesChan <- []string{}
	}
	
	// Wait for all operations to complete
	wg.Wait()
	
	// Collect results
	return &GatheredContext{
		ProjectInfo:   <-projectInfoChan,
		RelevantFiles: <-filesChan,
		SystemInfo:    <-systemInfoChan,
		CodeExamples:  <-codeExamplesChan,
	}, nil
}

// getProjectInfo gathers project structure information
func (pcg *ParallelContextGatherer) getProjectInfo(ctx context.Context, plan *QueryProcessingPlan) map[string]interface{} {
	// Check cache first
	cacheKey := "project_structure"
	if cached, found := pcg.cache.Get(cacheKey); found {
		if data, ok := cached.Data["project_info"].(map[string]interface{}); ok {
			return data
		}
	}
	
	// Execute filesystem commands to get project info
	info := map[string]interface{}{}
	
	// Get file count
	if files, err := pcg.filesystemServer.SearchFiles([]string{"*.go"}, ""); err == nil {
		info["file_count"] = len(files)
		info["go_files"] = files
	}
	
	// Get project structure
	if structure, err := pcg.filesystemServer.GetProjectStructure(3); err == nil {
		info["structure"] = structure
		info["directories"] = pcg.extractDirectories(structure)
	}
	
	// Cache the result
	mcpContext := &models.MCPContext{
		RequiresMCP: true,
		Operations:  []string{"filesystem_scan"},
		Data:        map[string]interface{}{"project_info": info},
	}
	pcg.cache.Set(cacheKey, mcpContext, len(info), "project_hash")
	
	return info
}

// getRelevantFiles gets files relevant to the query
func (pcg *ParallelContextGatherer) getRelevantFiles(ctx context.Context, plan *QueryProcessingPlan) []string {
	// For explanation queries, get key architectural files
	if plan.Intent.Primary == IntentExplain {
		keyFiles := []string{
			"cmd/main.go",
			"internal/app/cli.go",
			"internal/agents/manager_agent.go",
			"internal/mcp/mcp_client.go",
			"internal/vectordb/qdrant_client.go",
		}
		
		// Filter to existing files
		var existingFiles []string
		for _, file := range keyFiles {
			if pcg.fileExists(file) {
				existingFiles = append(existingFiles, file)
			}
		}
		return existingFiles
	}
	
	// For other queries, search based on keywords
	return pcg.searchFilesByKeywords(plan.Intent.Keywords)
}

// getSystemInfo gathers system information
func (pcg *ParallelContextGatherer) getSystemInfo(ctx context.Context, plan *QueryProcessingPlan) map[string]interface{} {
	info := map[string]interface{}{}
	
	// Execute system commands based on query
	for _, keyword := range plan.Intent.Keywords {
		switch keyword {
		case "cpu", "memory", "usage":
			if result, err := pcg.commandExecutor.executeMemoryCommand(ctx); err == nil {
				info["memory_usage"] = result
			}
		case "files", "count", "indexed":
			if result, err := pcg.commandExecutor.executeFileCountCommand(ctx); err == nil {
				info["file_stats"] = result
			}
		case "status", "health":
			info["status"] = "running"
			info["timestamp"] = time.Now()
		}
	}
	
	return info
}

// getCodeExamples gets relevant code examples
func (pcg *ParallelContextGatherer) getCodeExamples(ctx context.Context, plan *QueryProcessingPlan) []string {
	examples := []string{}
	
	// This would integrate with vector search to find relevant code
	// For now, return empty - will be implemented when vector search is connected
	
	return examples
}

// Helper methods
func (pcg *ParallelContextGatherer) needsSystemInfo(plan *QueryProcessingPlan) bool {
	return plan.Intent.Primary == IntentSystemStatus ||
		pcg.containsAny(plan.Intent.Keywords, []string{"cpu", "memory", "usage", "status"})
}

func (pcg *ParallelContextGatherer) needsCodeExamples(plan *QueryProcessingPlan) bool {
	return plan.Intent.Primary == IntentGenerate ||
		plan.Intent.Primary == IntentExplain ||
		pcg.containsAny(plan.Intent.Keywords, []string{"example", "pattern", "similar"})
}

func (pcg *ParallelContextGatherer) containsAny(slice []string, items []string) bool {
	for _, item := range items {
		for _, s := range slice {
			if strings.Contains(s, item) {
				return true
			}
		}
	}
	return false
}

func (pcg *ParallelContextGatherer) extractDirectories(structure map[string]interface{}) []string {
	var dirs []string
	for key := range structure {
		dirs = append(dirs, key)
	}
	return dirs
}

func (pcg *ParallelContextGatherer) fileExists(path string) bool {
	// Simple check - in real implementation, would check filesystem
	keyFiles := map[string]bool{
		"cmd/main.go":                        true,
		"internal/app/cli.go":                true,
		"internal/agents/manager_agent.go":   true,
		"internal/mcp/mcp_client.go":         true,
		"internal/vectordb/qdrant_client.go": true,
	}
	return keyFiles[path]
}

func (pcg *ParallelContextGatherer) searchFilesByKeywords(keywords []string) []string {
	// Simple implementation - would use actual file search
	return []string{}
}