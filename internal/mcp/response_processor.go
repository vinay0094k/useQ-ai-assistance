package mcp

import (
	"fmt"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// ResponseProcessor enhances responses with metadata and context
type ResponseProcessor struct{}

// NewResponseProcessor creates a new response processor
func NewResponseProcessor() *ResponseProcessor {
	return &ResponseProcessor{}
}

// EnhanceResponse enhances the response with context and metadata
func (rp *ResponseProcessor) EnhanceResponse(response *models.Response, contextData *GatheredContext, plan *QueryProcessingPlan) *models.Response {
	if response == nil {
		return nil
	}
	
	// Add source attribution
	response.Metadata.Sources = rp.extractSources(contextData)
	
	// Add context metadata
	response.Metadata.FilesAnalyzed = len(contextData.RelevantFiles)
	response.Metadata.IndexHits = len(contextData.CodeExamples)
	
	// Add tools used
	response.Metadata.Tools = rp.extractToolsUsed(plan)
	
	// Add reasoning
	response.Metadata.Reasoning = rp.buildReasoning(plan, contextData)
	
	// Enhance content with context references
	if response.Content.Text != "" {
		response.Content.Text = rp.addContextReferences(response.Content.Text, contextData)
	}
	
	return response
}

// extractSources extracts source files from context
func (rp *ResponseProcessor) extractSources(contextData *GatheredContext) []string {
	sources := make(map[string]bool)
	
	// Add relevant files
	for _, file := range contextData.RelevantFiles {
		sources[file] = true
	}
	
	// Convert to slice
	var sourceList []string
	for source := range sources {
		sourceList = append(sourceList, source)
	}
	
	return sourceList
}

// extractToolsUsed extracts tools used from the plan
func (rp *ResponseProcessor) extractToolsUsed(plan *QueryProcessingPlan) []string {
	tools := []string{}
	
	for _, operation := range plan.RequiredOperations {
		switch operation {
		case "filesystem_structure":
			tools = append(tools, "filesystem_analysis")
		case "code_analysis":
			tools = append(tools, "code_parser")
		case "vector_search":
			tools = append(tools, "semantic_search")
		case "system_info":
			tools = append(tools, "system_monitor")
		}
	}
	
	return tools
}

// buildReasoning builds reasoning explanation
func (rp *ResponseProcessor) buildReasoning(plan *QueryProcessingPlan, contextData *GatheredContext) string {
	return fmt.Sprintf("Analyzed %d files using %s approach with %s context depth",
		len(contextData.RelevantFiles),
		plan.Intent.Primary,
		plan.ContextDepth)
}

// addContextReferences adds context references to response text
func (rp *ResponseProcessor) addContextReferences(text string, contextData *GatheredContext) string {
	if len(contextData.RelevantFiles) > 0 {
		text += fmt.Sprintf("\n\n📁 **Key Files Referenced:**\n")
		for _, file := range contextData.RelevantFiles {
			text += fmt.Sprintf("- %s\n", file)
		}
	}
	
	if contextData.SystemInfo != nil && len(contextData.SystemInfo) > 0 {
		text += fmt.Sprintf("\n\n📊 **System Context:**\n")
		if fileCount, ok := contextData.SystemInfo["file_count"].(int); ok {
			text += fmt.Sprintf("- Indexed files: %d\n", fileCount)
		}
	}
	
	return text
}