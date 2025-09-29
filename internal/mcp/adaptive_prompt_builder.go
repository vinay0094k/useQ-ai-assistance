package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/yourusername/useq-ai-assistant/models"
)

// AdaptivePromptBuilder builds intelligent prompts based on context
type AdaptivePromptBuilder struct {
	templates map[IntentType]*PromptTemplate
}

// PromptTemplate represents a template for building prompts
type PromptTemplate struct {
	SystemPrompt    string            `json:"system_prompt"`
	UserTemplate    string            `json:"user_template"`
	ContextTemplate string            `json:"context_template"`
	ExampleTemplate string            `json:"example_template"`
	Variables       map[string]string `json:"variables"`
}

// NewAdaptivePromptBuilder creates a new adaptive prompt builder
func NewAdaptivePromptBuilder() *AdaptivePromptBuilder {
	builder := &AdaptivePromptBuilder{
		templates: make(map[IntentType]*PromptTemplate),
	}
	builder.initializeTemplates()
	return builder
}

// BuildPrompt builds an intelligent prompt with full context
func (apb *AdaptivePromptBuilder) BuildPrompt(ctx context.Context, query *models.Query, intent *ClassifiedIntent, context *FilteredContext) (*AdaptivePrompt, error) {
	template := apb.getTemplate(intent.Primary)
	
	// Build system prompt
	systemPrompt := apb.buildSystemPrompt(template, intent, context)
	
	// Build user prompt with context
	userPrompt := apb.buildUserPrompt(template, query, intent, context)
	
	// Build context section
	contextSection := apb.buildContextSection(template, context)
	
	// Build examples section
	examplesSection := apb.buildExamplesSection(template, context)
	
	return &AdaptivePrompt{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Context:      contextSection,
		Examples:     examplesSection,
	}, nil
}

// initializeTemplates sets up prompt templates for different intents
func (apb *AdaptivePromptBuilder) initializeTemplates() {
	// Explanation template
	apb.templates[IntentExplain] = &PromptTemplate{
		SystemPrompt: `You are an expert software architect analyzing a Go application. 
Provide clear, comprehensive explanations of code architecture and flow.
Focus on:
- High-level architecture and design patterns
- Component interactions and data flow
- Key responsibilities of each module
- Integration points and dependencies

Use the provided project context to give accurate, specific explanations.`,

		UserTemplate: `Explain: {{.Query}}

Project Context:
{{.ProjectContext}}

Please provide a detailed explanation covering:
1. Overall architecture and design
2. Key components and their roles
3. Data flow and interactions
4. Important patterns and conventions`,

		ContextTemplate: `PROJECT STRUCTURE:
{{.ProjectStructure}}

KEY FILES:
{{.KeyFiles}}

SYSTEM INFO:
{{.SystemInfo}}`,

		ExampleTemplate: `RELEVANT CODE EXAMPLES:
{{.CodeExamples}}`,
	}
	
	// Generation template
	apb.templates[IntentGenerate] = &PromptTemplate{
		SystemPrompt: `You are an expert Go developer. Generate clean, idiomatic Go code that follows the project's existing patterns and conventions.

Requirements:
- Follow existing code style and patterns
- Include proper error handling
- Add appropriate comments
- Use project's naming conventions
- Include necessary imports`,

		UserTemplate: `Generate: {{.Query}}

Project Patterns:
{{.ProjectPatterns}}

Similar Examples:
{{.SimilarExamples}}

Generate production-ready Go code that fits seamlessly into this project.`,
	}
	
	// Search template
	apb.templates[IntentSearch] = &PromptTemplate{
		SystemPrompt: `You are a code search assistant. Help users find relevant code in their project.
Provide specific file locations, function names, and brief explanations.`,

		UserTemplate: `Search for: {{.Query}}

Available Files:
{{.FileList}}

Provide specific locations and brief descriptions of relevant code.`,
	}
	
	// System status template
	apb.templates[IntentSystemStatus] = &PromptTemplate{
		SystemPrompt: `You are a system information assistant. Provide clear, formatted system status information.`,

		UserTemplate: `System Query: {{.Query}}

Current System State:
{{.SystemInfo}}

Provide a clear summary of the requested system information.`,
	}
}

// buildSystemPrompt builds the system prompt
func (apb *AdaptivePromptBuilder) buildSystemPrompt(template *PromptTemplate, intent *ClassifiedIntent, context *FilteredContext) string {
	prompt := template.SystemPrompt
	
	// Add quality requirements
	if intent.QualityRequirements.RequireExamples {
		prompt += "\n\nIMPORTANT: Include specific code examples from the project."
	}
	
	if intent.QualityRequirements.RequireContext {
		prompt += "\n\nIMPORTANT: Use the provided project context to ensure accuracy."
	}
	
	return prompt
}

// buildUserPrompt builds the user prompt with context
func (apb *AdaptivePromptBuilder) buildUserPrompt(template *PromptTemplate, query *models.Query, intent *ClassifiedIntent, context *FilteredContext) string {
	userPrompt := template.UserTemplate
	
	// Replace variables
	userPrompt = strings.ReplaceAll(userPrompt, "{{.Query}}", query.UserInput)
	
	// Add project context
	if context.ProjectInfo != nil {
		projectContext := apb.formatProjectContext(context.ProjectInfo)
		userPrompt = strings.ReplaceAll(userPrompt, "{{.ProjectContext}}", projectContext)
	}
	
	// Add file list
	if len(context.RelevantFiles) > 0 {
		fileList := strings.Join(context.RelevantFiles, "\n- ")
		userPrompt = strings.ReplaceAll(userPrompt, "{{.FileList}}", "- "+fileList)
	}
	
	return userPrompt
}

// buildContextSection builds the context section
func (apb *AdaptivePromptBuilder) buildContextSection(template *PromptTemplate, context *FilteredContext) string {
	if template.ContextTemplate == "" {
		return ""
	}
	
	contextSection := template.ContextTemplate
	
	// Replace project structure
	if context.ProjectInfo != nil {
		if structure, ok := context.ProjectInfo["structure"].(map[string]interface{}); ok {
			structureText := apb.formatStructure(structure, 0)
			contextSection = strings.ReplaceAll(contextSection, "{{.ProjectStructure}}", structureText)
		}
		
		// Replace key files
		if len(context.RelevantFiles) > 0 {
			keyFiles := strings.Join(context.RelevantFiles, "\n- ")
			contextSection = strings.ReplaceAll(contextSection, "{{.KeyFiles}}", "- "+keyFiles)
		}
		
		// Replace system info
		if context.SystemInfo != nil {
			systemInfo := apb.formatSystemInfo(context.SystemInfo)
			contextSection = strings.ReplaceAll(contextSection, "{{.SystemInfo}}", systemInfo)
		}
	}
	
	return contextSection
}

// buildExamplesSection builds the examples section
func (apb *AdaptivePromptBuilder) buildExamplesSection(template *PromptTemplate, context *FilteredContext) string {
	if template.ExampleTemplate == "" || len(context.CodeExamples) == 0 {
		return ""
	}
	
	examplesSection := template.ExampleTemplate
	codeExamples := strings.Join(context.CodeExamples, "\n\n")
	examplesSection = strings.ReplaceAll(examplesSection, "{{.CodeExamples}}", codeExamples)
	
	return examplesSection
}

// getTemplate gets the appropriate template for an intent
func (apb *AdaptivePromptBuilder) getTemplate(intent IntentType) *PromptTemplate {
	if template, exists := apb.templates[intent]; exists {
		return template
	}
	// Default to explanation template
	return apb.templates[IntentExplain]
}

// formatProjectContext formats project context for display
func (apb *AdaptivePromptBuilder) formatProjectContext(projectInfo map[string]interface{}) string {
	var context strings.Builder
	
	if fileCount, ok := projectInfo["file_count"].(int); ok {
		context.WriteString(fmt.Sprintf("- Total Go files: %d\n", fileCount))
	}
	
	if dirs, ok := projectInfo["directories"].([]string); ok {
		context.WriteString(fmt.Sprintf("- Key directories: %s\n", strings.Join(dirs, ", ")))
	}
	
	return context.String()
}

// formatStructure formats project structure for display
func (apb *AdaptivePromptBuilder) formatStructure(structure map[string]interface{}, depth int) string {
	var result strings.Builder
	indent := strings.Repeat("  ", depth)
	
	for key, value := range structure {
		result.WriteString(fmt.Sprintf("%s- %s\n", indent, key))
		if subMap, ok := value.(map[string]interface{}); ok && depth < 2 {
			result.WriteString(apb.formatStructure(subMap, depth+1))
		}
	}
	
	return result.String()
}

// formatSystemInfo formats system information for display
func (apb *AdaptivePromptBuilder) formatSystemInfo(systemInfo map[string]interface{}) string {
	var info strings.Builder
	
	for key, value := range systemInfo {
		info.WriteString(fmt.Sprintf("- %s: %v\n", key, value))
	}
	
	return info.String()
}