package mcp

import (
	"sync"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// UsagePattern represents learned usage patterns
type UsagePattern struct {
	ProjectPath     string            `json:"project_path"`
	QueryTypes      map[string]int    `json:"query_types"`
	Operations      map[string]int    `json:"operations"`
	TimePatterns    map[int]int       `json:"time_patterns"` // hour -> count
	LastAccessed    time.Time         `json:"last_accessed"`
	AccessCount     int               `json:"access_count"`
	AvgResponseTime time.Duration     `json:"avg_response_time"`
}

// UsageTracker learns from MCP usage patterns
type UsageTracker struct {
	patterns map[string]*UsagePattern
	mu       sync.RWMutex
}

// NewUsageTracker creates a new usage tracker
func NewUsageTracker() *UsageTracker {
	return &UsageTracker{
		patterns: make(map[string]*UsagePattern),
	}
}

// RecordUsage records a usage event
func (ut *UsageTracker) RecordUsage(query *models.Query, operations []string, responseTime time.Duration) {
	ut.mu.Lock()
	defer ut.mu.Unlock()
	
	projectPath := ut.getProjectPath(query)
	pattern, exists := ut.patterns[projectPath]
	
	if !exists {
		pattern = &UsagePattern{
			ProjectPath:  projectPath,
			QueryTypes:   make(map[string]int),
			Operations:   make(map[string]int),
			TimePatterns: make(map[int]int),
		}
		ut.patterns[projectPath] = pattern
	}
	
	// Update patterns
	pattern.QueryTypes[string(query.Type)]++
	pattern.TimePatterns[time.Now().Hour()]++
	pattern.LastAccessed = time.Now()
	pattern.AccessCount++
	
	// Update average response time
	if pattern.AvgResponseTime == 0 {
		pattern.AvgResponseTime = responseTime
	} else {
		pattern.AvgResponseTime = (pattern.AvgResponseTime + responseTime) / 2
	}
	
	// Record operations
	for _, op := range operations {
		pattern.Operations[op]++
	}
}

// PredictOperations predicts likely operations for a query
func (ut *UsageTracker) PredictOperations(query *models.Query) []string {
	ut.mu.RLock()
	defer ut.mu.RUnlock()
	
	projectPath := ut.getProjectPath(query)
	pattern, exists := ut.patterns[projectPath]
	
	if !exists {
		return []string{"filesystem_search"} // Default
	}
	
	// Return most common operations for this query type
	var predicted []string
	queryType := string(query.Type)
	
	if pattern.QueryTypes[queryType] > 0 {
		for op, count := range pattern.Operations {
			if count >= 2 { // Threshold for prediction
				predicted = append(predicted, op)
			}
		}
	}
	
	if len(predicted) == 0 {
		predicted = []string{"filesystem_search"}
	}
	
	return predicted
}

// ShouldPreCache determines if a project should be pre-cached
func (ut *UsageTracker) ShouldPreCache(projectPath string) bool {
	ut.mu.RLock()
	defer ut.mu.RUnlock()
	
	pattern, exists := ut.patterns[projectPath]
	if !exists {
		return false
	}
	
	// Pre-cache if accessed recently and frequently
	return pattern.AccessCount >= 3 && 
		   time.Since(pattern.LastAccessed) < 1*time.Hour
}

// GetAdaptiveTTL returns adaptive TTL based on usage patterns
func (ut *UsageTracker) GetAdaptiveTTL(projectPath string) time.Duration {
	ut.mu.RLock()
	defer ut.mu.RUnlock()
	
	pattern, exists := ut.patterns[projectPath]
	if !exists {
		return 5 * time.Minute // Default TTL
	}
	
	// More frequent access = longer TTL
	if pattern.AccessCount >= 10 {
		return 15 * time.Minute
	} else if pattern.AccessCount >= 5 {
		return 10 * time.Minute
	}
	
	return 5 * time.Minute
}

// GetStats returns usage statistics
func (ut *UsageTracker) GetStats() map[string]interface{} {
	ut.mu.RLock()
	defer ut.mu.RUnlock()
	
	totalAccess := 0
	activeProjects := 0
	
	for _, pattern := range ut.patterns {
		totalAccess += pattern.AccessCount
		if time.Since(pattern.LastAccessed) < 24*time.Hour {
			activeProjects++
		}
	}
	
	return map[string]interface{}{
		"total_projects":   len(ut.patterns),
		"active_projects":  activeProjects,
		"total_accesses":   totalAccess,
	}
}

func (ut *UsageTracker) getProjectPath(query *models.Query) string {
	if query.ProjectRoot != "" {
		return query.ProjectRoot
	}
	return "."
}
