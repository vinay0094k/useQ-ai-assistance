package agents

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// SystemAgent handles system/runtime queries
type SystemAgent struct {
	dependencies *AgentDependencies
	config       SystemAgentConfig
}

// SystemAgentConfig holds configuration for system agent
type SystemAgentConfig struct {
	MonitoringEnabled bool `json:"monitoring_enabled"`
	MetricsInterval   time.Duration `json:"metrics_interval"`
}

// NewSystemAgent creates a new system agent
func NewSystemAgent(deps *AgentDependencies) *SystemAgent {
	return &SystemAgent{
		dependencies: deps,
		config: SystemAgentConfig{
			MonitoringEnabled: true,
			MetricsInterval:   30 * time.Second,
		},
	}
}

// Process handles system/runtime queries
func (sa *SystemAgent) Process(ctx context.Context, query *models.Query) (*models.Response, error) {
	switch query.Type {
	case models.QueryTypeSystem, models.QueryTypeRuntime, models.QueryTypeMonitoring:
		return sa.handleSystemQuery(ctx, query)
	default:
		return nil, fmt.Errorf("unsupported query type: %s", query.Type)
	}
}

// handleSystemQuery processes system-related queries
func (sa *SystemAgent) handleSystemQuery(ctx context.Context, query *models.Query) (*models.Response, error) {
	systemInfo := sa.gatherSystemInfo()
	
	response := &models.Response{
		ID:        "system-" + query.ID,
		QueryID:   query.ID,
		Type:      models.ResponseTypeSystem,
		Content: models.ResponseContent{
			Text: sa.formatSystemInfo(systemInfo),
		},
		Metadata: models.ResponseMetadata{
			GenerationTime: time.Since(time.Now()),
			Confidence:     1.0,
		},
		AgentUsed: "system",
		Timestamp: time.Now(),
	}
	
	return response, nil
}

// gatherSystemInfo collects current system information
func (sa *SystemAgent) gatherSystemInfo() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return map[string]interface{}{
		"memory": map[string]interface{}{
			"allocated":     m.Alloc,
			"total_alloc":   m.TotalAlloc,
			"heap_objects":  m.HeapObjects,
			"gc_cycles":     m.NumGC,
		},
		"runtime": map[string]interface{}{
			"goroutines":    runtime.NumGoroutine(),
			"go_version":    runtime.Version(),
			"os":           runtime.GOOS,
			"arch":         runtime.GOARCH,
		},
		"process": map[string]interface{}{
			"pid":          os.Getpid(),
			"working_dir":  sa.getWorkingDir(),
		},
		"timestamp": time.Now(),
	}
}

// formatSystemInfo formats system information for display
func (sa *SystemAgent) formatSystemInfo(info map[string]interface{}) string {
	result := "🖥️  **System Information**\n\n"
	
	if memory, ok := info["memory"].(map[string]interface{}); ok {
		result += "**Memory:**\n"
		result += fmt.Sprintf("- Allocated: %d bytes\n", memory["allocated"])
		result += fmt.Sprintf("- Heap Objects: %d\n", memory["heap_objects"])
		result += fmt.Sprintf("- GC Cycles: %d\n", memory["gc_cycles"])
		result += "\n"
	}
	
	if runtime, ok := info["runtime"].(map[string]interface{}); ok {
		result += "**Runtime:**\n"
		result += fmt.Sprintf("- Goroutines: %d\n", runtime["goroutines"])
		result += fmt.Sprintf("- Go Version: %s\n", runtime["go_version"])
		result += fmt.Sprintf("- OS: %s\n", runtime["os"])
		result += fmt.Sprintf("- Architecture: %s\n", runtime["arch"])
		result += "\n"
	}
	
	if process, ok := info["process"].(map[string]interface{}); ok {
		result += "**Process:**\n"
		result += fmt.Sprintf("- PID: %d\n", process["pid"])
		result += fmt.Sprintf("- Working Directory: %s\n", process["working_dir"])
	}
	
	return result
}

func (sa *SystemAgent) getWorkingDir() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "unknown"
}
