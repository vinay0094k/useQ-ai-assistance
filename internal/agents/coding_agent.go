package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/useq-ai-assistant/internal/llm"
	"github.com/yourusername/useq-ai-assistant/models"
)

// CodingAgentImpl handles code generation queries using centralized types
type CodingAgentImpl struct {
	dependencies *AgentDependencies
	config       *CodingAgentConfig
	metrics      *AgentMetrics
}

// NewCodingAgentConfig creates a new coding agent configuration with sensible defaults
func NewCodingAgentConfig() *CodingAgentConfig {
	base := NewAgentConfig()
	return &CodingAgentConfig{
		AgentConfig:         *base,
		MaxContextFiles:     12,    // Reasonable number of context files
		MaxContextLines:     800,   // Sufficient context without overwhelming
		SimilarityThreshold: 0.75,  // Good balance for code similarity matching
		IncludeTests:        true,  // Encourage test inclusion by default
		IncludeDocs:         true,  // Include documentation by default
		UseProjectPatterns:  true,  // Leverage existing project patterns
		MaxExamples:         8,     // Reasonable number of usage examples
		GenerateComments:    true,  // Generate comments for better code quality
		GenerateTests:       true,  // Generate tests by default
		ValidateGenerated:   true,  // Validate generated code for quality
		OptimizeCode:        false, // Optimization can be enabled when specifically needed
	}
}

// NewCodingAgent creates a new coding agent with centralized configuration
func NewCodingAgent(deps *AgentDependencies) *CodingAgentImpl {
	return &CodingAgentImpl{
		dependencies: deps,
		config:       NewCodingAgentConfig(),
		metrics: &AgentMetrics{
			QueriesHandled:      0,
			SuccessRate:         0.0,
			AverageResponseTime: 0,
			AverageConfidence:   0.0,
			TokensUsed:          0,
			TotalCost:           0.0,
			LastUsed:            time.Now(),
			ErrorCount:          0,
		},
	}
}

// GetCapabilities returns what this agent can do
func (ca *CodingAgentImpl) GetCapabilities() AgentCapabilities {
	return AgentCapabilities{
		CanGenerateCode:    true,
		CanSearchCode:      false, // Delegates to search agent
		CanAnalyzeCode:     true,
		CanDebugCode:       false,
		CanWriteTests:      ca.config.GenerateTests,
		CanWriteDocs:       ca.config.GenerateComments,
		CanReviewCode:      false,
		SupportedLanguages: []string{"go"},
		MaxComplexity:      8,
		RequiresContext:    true,
	}
}

// CanHandle determines if this agent can handle the given query
func (ca *CodingAgentImpl) CanHandle(ctx context.Context, query *models.Query) (bool, float64) {
	intent, err := ca.parseCodeIntent(query)
	if err != nil {
		return false, 0.0
	}

	// Calculate confidence based on query characteristics
	confidence := ca.calculateHandlingConfidence(intent, query)

	// Can handle if confidence is above threshold and it's a code generation query
	canHandle := confidence >= 0.6 && ca.isCodeGenerationQuery(intent)

	return canHandle, confidence
}

// GetSpecialization returns the agent's area of expertise
func (ca *CodingAgentImpl) GetSpecialization() AgentSpecialization {
	return AgentSpecialization{
		Type:        AgentTypeCoding,
		Languages:   []string{"go"},
		Frameworks:  []string{"gin", "fiber", "echo", "stdlib"},
		Domains:     []string{"web_development", "microservices", "cli_tools", "apis"},
		Complexity:  8,
		Description: "Generates Go code based on project patterns and context",
	}
}

// GetConfidenceScore returns confidence in handling the query
func (ca *CodingAgentImpl) GetConfidenceScore(ctx context.Context, query *models.Query) float64 {
	intent, err := ca.parseCodeIntent(query)
	if err != nil {
		return 0.0
	}
	return ca.calculateHandlingConfidence(intent, query)
}

// ValidateQuery checks if the query is valid for this agent
func (ca *CodingAgentImpl) ValidateQuery(query *models.Query) error {
	if err := ValidateQuery(query); err != nil {
		return err
	}

	intent, err := ca.parseCodeIntent(query)
	if err != nil {
		return fmt.Errorf("failed to parse intent: %w", err)
	}

	if !ca.isCodeGenerationQuery(intent) {
		return fmt.Errorf("query is not a code generation request")
	}

	return nil
}

// Process handles the query and returns a response
func (ca *CodingAgentImpl) Process(ctx context.Context, query *models.Query) (*models.Response, error) {
	startTime := time.Now()
	ca.updateMetrics(startTime)

	if ca.dependencies == nil {
		return ca.createFallbackResponse(query, "Dependencies not initialized"), nil
	}

	if ca.dependencies.LLMManager == nil {
		return ca.createFallbackResponse(query, "LLM Manager not available"), nil
	}

	// Log step-by-step processing
	ca.logStep("Starting code generation process", map[string]interface{}{
		"query_id": query.ID,
		"language": query.Language,
		"input":    query.UserInput,
	})

	// Parse code generation intent
	intent, err := ca.parseCodeIntent(query)
	if err != nil {
		ca.metrics.ErrorCount++
		return nil, fmt.Errorf("failed to parse code intent: %w", err)
	}

	ca.logStep("Parsed code generation intent", map[string]interface{}{
		"intent_type":   string(intent.Type),
		"function_name": intent.FunctionName,
		"constraints":   len(intent.Constraints),
	})

	// Enhance with MCP context if available
	if query.MCPContext != nil && query.MCPContext.RequiresMCP {
		ca.enhanceIntentWithMCP(intent, query.MCPContext)
		ca.logStep("Enhanced intent with MCP context", map[string]interface{}{
			"mcp_operations": query.MCPContext.Operations,
			"mcp_data_keys": ca.getMCPDataKeys(query.MCPContext),
		})
	}

	// Gather comprehensive code context
	codeContext, err := ca.gatherCodeContext(ctx, intent, query)
	if err != nil {
		ca.metrics.ErrorCount++
		return nil, fmt.Errorf("failed to gather code context: %w", err)
	}

	ca.logStep("Gathered code context", map[string]interface{}{
		"similar_code_examples": len(codeContext.SimilarCode),
		"relevant_types":        len(codeContext.RelevantTypes),
		"relevant_functions":    len(codeContext.RelevantFunctions),
		"patterns":              len(codeContext.Patterns),
		"imports":               len(codeContext.ImportSuggestions),
	})

	// Generate code using LLM with context
	codeResponse, tokenUsage, err := ca.generateContextualCode(ctx, intent, codeContext, query)
	if err != nil {
		ca.metrics.ErrorCount++
		return nil, fmt.Errorf("failed to generate code: %w", err)
	}

	ca.logStep("Generated code", map[string]interface{}{
		"lines_generated": strings.Count(codeResponse.Code, "\n"),
		"tokens_used":     tokenUsage.TotalTokens,
		"provider":        codeResponse.Provider,
	})

	// Validate generated code if enabled
	if ca.config.ValidateGenerated {
		validation, err := ca.validateGeneratedCode(codeResponse, intent)
		if err != nil {
			ca.logStep("Code validation failed", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			ca.logStep("Code validation passed", map[string]interface{}{
				"issues": len(validation.Issues),
			})
			codeResponse.Validation = validation
		}
	}

	// Calculate final confidence
	confidence := ca.calculateCodeConfidence(codeContext, codeResponse)

	// Create comprehensive response
	response := ca.buildResponse(query, intent, codeContext, codeResponse, tokenUsage, confidence, startTime)

	ca.logStep("Code generation completed", map[string]interface{}{
		"response_id":    response.ID,
		"confidence":     confidence,
		"total_time_ms":  time.Since(startTime).Milliseconds(),
		"files_analyzed": response.Metadata.FilesAnalyzed,
	})

	// Update metrics
	ca.updateSuccessMetrics(startTime, confidence, tokenUsage)

	return response, nil
}

// ================================= fallback responses =================================
func (ca *CodingAgentImpl) createFallbackResponse(query *models.Query, reason string) *models.Response {
	var contextualInfo strings.Builder
	contextualInfo.WriteString(fmt.Sprintf("Code generation request: '%s'\n\n", query.UserInput))

	// Try to find relevant code examples first
	if ca.dependencies != nil && ca.dependencies.VectorDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Search for similar code patterns
		searchQuery := fmt.Sprintf("%s code example pattern", query.UserInput)
		results, err := ca.dependencies.VectorDB.Search(ctx, searchQuery, 3)
		if err == nil && len(results) > 0 {
			contextualInfo.WriteString("🔍 Found similar patterns in your project:\n\n")
			for i, result := range results {
				if i >= 2 {
					break
				} // Limit to top 2
				contextualInfo.WriteString(fmt.Sprintf("📁 %s (similarity: %.3f)\n",
					result.Chunk.FilePath, result.Score))
				contextualInfo.WriteString(fmt.Sprintf("   🏷️  %s\n", result.Chunk.Language))
				if result.Chunk.Content != "" {
					// Show relevant code snippet
					content := result.Chunk.Content
					if len(content) > 150 {
						content = content[:150] + "..."
					}
					contextualInfo.WriteString(fmt.Sprintf("   📝 %s\n", content))
				}
				contextualInfo.WriteString("\n")
			}
			contextualInfo.WriteString("💡 Connect LLM Manager to generate code following these patterns.\n\n")
		}
	}

	contextualInfo.WriteString(fmt.Sprintf("Status: %s\n\n", reason))
	contextualInfo.WriteString("To enable AI code generation based on YOUR patterns:\n")
	contextualInfo.WriteString("1. ✅ Pattern Detection (Finding similar code in your project)\n")
	contextualInfo.WriteString("2. ❌ LLM Integration (Connect for intelligent generation)\n")
	contextualInfo.WriteString("3. ❌ Code Analysis (Connect for pattern-following generation)\n")

	return &models.Response{
		ID:      fmt.Sprintf("coding_response_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    models.ResponseTypeCode,
		Content: models.ResponseContent{
			Text: contextualInfo.String(),
		},
		AgentUsed:  "coding_agent",
		Timestamp:  time.Now(),
		TokenUsage: models.TokenUsage{TotalTokens: 0},
		Cost:       models.Cost{TotalCost: 0.0, Currency: "USD"},
		Metadata: models.ResponseMetadata{
			GenerationTime: time.Since(time.Now()),
			Confidence:     0.6,
		},
	}
}

// GetMetrics returns performance metrics for this agent
func (ca *CodingAgentImpl) GetMetrics() AgentMetrics {
	if ca.metrics == nil {
		return AgentMetrics{}
	}
	return *ca.metrics
}

// AnalyzeCode analyzes code structure and patterns
func (ca *CodingAgentImpl) AnalyzeCode(ctx context.Context, code string, language string) (*CodeAnalysis, error) {
	analysis := &CodeAnalysis{
		Language: language,
		Complexity: &ComplexityAnalysis{
			Cyclomatic:      ca.calculateCyclomaticComplexity(code),
			Cognitive:       ca.calculateCognitiveComplexity(code),
			Maintainability: 0.8, // Placeholder
		},
		Quality: &QualityMetrics{
			Readability:     0.8,
			Maintainability: 0.8,
			Testability:     0.7,
			Reusability:     0.6,
			Overall:         0.725,
		},
		Patterns:    ca.detectCodePatterns(code),
		Suggestions: ca.generateCodeSuggestions(code),
		Issues:      ca.detectCodeIssues(code),
	}

	return analysis, nil
}

// GetCodeContext gathers relevant code context
func (ca *CodingAgentImpl) GetCodeContext(ctx context.Context, query *models.Query) (*CodeContext, error) {
	intent, err := ca.parseCodeIntent(query)
	if err != nil {
		return nil, err
	}
	return ca.gatherCodeContext(ctx, intent, query)
}

// =============================================================================
// PRIVATE IMPLEMENTATION METHODS
// =============================================================================

func (ca *CodingAgentImpl) parseCodeIntent(query *models.Query) (*CodingAgentIntent, error) {
	if query == nil {
		return nil, fmt.Errorf("query is nil")
	}

	input := strings.ToLower(strings.TrimSpace(query.UserInput))

	intent := &CodingAgentIntent{
		Description:       query.UserInput,
		RequiredFeatures:  make([]string, 0),
		Constraints:       make([]string, 0),
		Libraries:         make([]string, 0),
		IntegrationPoints: make([]string, 0),
	}

	// Determine code intent type
	intent.Type = ca.determineCodeIntentType(input)

	// Extract specific details (these are lightweight placeholder extractors)
	intent.FunctionName = ca.extractFunctionName(input)
	intent.Parameters = ca.extractParameters(input)
	intent.ReturnType = ca.extractReturnType(input)
	intent.TargetFile = ca.extractTargetFile(input)
	intent.Framework = ca.extractFramework(input)
	intent.Libraries = ca.extractLibraries(input)
	intent.Constraints = ca.extractConstraints(input)
	intent.RequiredFeatures = ca.extractRequiredFeatures(input)

	return intent, nil
}

func (ca *CodingAgentImpl) gatherCodeContext(ctx context.Context, intent *CodingAgentIntent, query *models.Query) (*CodeContext, error) {
	if intent == nil {
		return nil, fmt.Errorf("intent is nil")
	}
	context := &CodeContext{
		SimilarCode:       make([]CodeExample, 0),
		RelevantTypes:     make([]TypeDefinition, 0),
		RelevantFunctions: make([]FunctionDef, 0),
		Dependencies:      make([]Dependency, 0),
		Patterns:          make([]ProjectPattern, 0),
		ImportSuggestions: make([]ImportSuggestion, 0),
		UsageExamples:     make([]UsageExample, 0),
		FileStructure:     make(map[string]FileInfo),
	}

	// Gather project information
	projectInfo, err := ca.analyzeProjectInfo(ctx, query.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze project info: %w", err)
	}
	context.ProjectInfo = projectInfo

	// Skip similar code examples for now
	// TODO: Implement similar code search using dependencies.SearchService

	// Find relevant types and functions
	if types, err := ca.findRelevantTypes(ctx, intent, query); err == nil {
		context.RelevantTypes = types
	} else {
		ca.logStep("Warning: failed to find relevant types", map[string]interface{}{
			"error": err.Error(),
		})
	}
	if funcs, err := ca.findRelevantFunctions(ctx, intent, query); err == nil {
		context.RelevantFunctions = funcs
	} else {
		ca.logStep("Warning: failed to find relevant functions", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Analyze project patterns
	if ca.config.UseProjectPatterns {
		if patterns, err := ca.analyzeProjectPatterns(ctx, intent); err == nil {
			context.Patterns = patterns
		} else {
			ca.logStep("Warning: failed to analyze patterns", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Generate import suggestions
	if imports, err := ca.generateImportSuggestions(ctx, intent, context); err == nil {
		context.ImportSuggestions = imports
	} else {
		ca.logStep("Warning: failed to generate import suggestions", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Find usage examples
	if examples, err := ca.findUsageExamples(ctx, intent); err == nil {
		context.UsageExamples = examples
	} else {
		ca.logStep("Warning: failed to find usage examples", map[string]interface{}{
			"error": err.Error(),
		})
	}

	return context, nil
}

func (ca *CodingAgentImpl) generateContextualCode(ctx context.Context, intent *CodingAgentIntent,
	context *CodeContext, query *models.Query) (*models.CodeResponse, *models.TokenUsage, error) {

	// Build comprehensive prompt with MCP enhancement
	systemPrompt := ca.buildMCPEnhancedSystemPrompt(context, query.MCPContext)
	userPrompt := ca.buildCodeGenerationPrompt(intent, context, query)

	ca.logStep("Built generation prompts", map[string]interface{}{
		"system_prompt_length": len(systemPrompt),
		"user_prompt_length":   len(userPrompt),
		"mcp_enhanced":         query.MCPContext != nil,
	})

	// Create LLM request with MCP context
	request := &llm.GenerationRequest{
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		MaxTokens:   ca.config.MaxTokens,
		Temperature: ca.config.Temperature,
		Model:       "", // Use default model
		Stream:      ca.config.StreamingEnabled,
		MCPContext:  query.MCPContext, // Pass MCP context to LLM
	}

	// Generate response with LLM manager
	llmResponse, err := ca.dependencies.LLMManager.Generate(ctx, request)
	if err != nil {
		return nil, nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	ca.logStep("LLM generation completed", map[string]interface{}{
		"provider":      llmResponse.Provider,
		"input_tokens":  llmResponse.TokenUsage.InputTokens,
		"output_tokens": llmResponse.TokenUsage.OutputTokens,
		"total_cost":    llmResponse.Cost.TotalCost,
	})

	// Parse LLM response into structured code response
	codeResponse, err := ca.parseCodeResponse(llmResponse.Content, intent, query.Language)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse code response: %w", err)
	}

	// Add metadata to code response
	codeResponse.Provider = llmResponse.Provider
	codeResponse.Context = fmt.Sprintf("%+v", context) // Convert CodeContext to string representation
	codeResponse.Intent = intent

	tokenUsage := &models.TokenUsage{
		InputTokens:  llmResponse.TokenUsage.InputTokens,
		OutputTokens: llmResponse.TokenUsage.OutputTokens,
		TotalTokens:  llmResponse.TokenUsage.TotalTokens,
	}

	return codeResponse, tokenUsage, nil
}

// Helper methods for implementation

func (ca *CodingAgentImpl) calculateHandlingConfidence(intent *CodingAgentIntent, query *models.Query) float64 {
	factors := map[string]float64{}

	// Language support
	if query != nil && query.Language == "go" {
		factors["language_support"] = 1.0
	} else {
		factors["language_support"] = 0.0
	}

	// Intent type support
	switch intent.Type {
	case CodeIntentFunction, CodeIntentMethod, CodeIntentHandler, CodeIntentService:
		factors["intent_support"] = 0.9
	case CodeIntentStruct, CodeIntentInterface:
		factors["intent_support"] = 0.8
	case CodeIntentTest:
		factors["intent_support"] = 0.7
	default:
		factors["intent_support"] = 0.6
	}

	// Query clarity
	if query != nil && len(strings.Fields(query.UserInput)) >= 3 {
		factors["query_clarity"] = 0.8
	} else {
		factors["query_clarity"] = 0.5
	}

	return CalculateConfidence(factors)
}

func (ca *CodingAgentImpl) isCodeGenerationQuery(intent *CodingAgentIntent) bool {
	if intent == nil {
		return true // optimistic default
	}

	// Check if it's explicitly a code generation intent
	generationTypes := []CodingAgentIntentType{
		CodeIntentFunction, CodeIntentMethod, CodeIntentStruct,
		CodeIntentInterface, CodeIntentHandler, CodeIntentService,
		CodeIntentRepository, CodeIntentMiddleware, CodeIntentTest,
		CodeIntentScript, CodeIntentConfig,
	}

	for _, genType := range generationTypes {
		if intent.Type == genType {
			return true
		}
	}

	return false
}

func (ca *CodingAgentImpl) determineCodeIntentType(input string) CodingAgentIntentType {
	// Enhanced intent detection with more patterns
	patterns := map[CodingAgentIntentType][]string{
		CodeIntentHandler:    {"handler", "endpoint", "route", "controller", "api"},
		CodeIntentService:    {"service", "business logic", "use case"},
		CodeIntentRepository: {"repository", "repo", "storage", "database", "persistence"},
		CodeIntentMiddleware: {"middleware", "interceptor", "filter"},
		CodeIntentStruct:     {"struct", "type", "model", "entity", "data structure"},
		CodeIntentInterface:  {"interface", "contract", "protocol"},
		CodeIntentTest:       {"test", "testing", "unit test", "integration test"},
		CodeIntentMethod:     {"method", "receiver"},
		CodeIntentFunction:   {"function", "func", "procedure"},
	}

	for intentType, keywords := range patterns {
		for _, keyword := range keywords {
			if strings.Contains(input, keyword) {
				return intentType
			}
		}
	}

	return CodeIntentFunction // Default
}

func (ca *CodingAgentImpl) findSimilarCodeExamples(ctx context.Context, intent *CodingAgentIntent, query *models.Query) ([]CodeExample, error) {
	// TODO: Implement proper search integration
	examples := make([]CodeExample, 0)

	// For now, return empty examples until search integration is complete
	return examples, nil
}

func (ca *CodingAgentImpl) analyzeProjectInfo(ctx context.Context, language string) (*ProjectInfo, error) {
	// Minimal but useful project info; real implementation should inspect project files
	return &ProjectInfo{
		Language:     language,
		PackageName:  "main",
		Architecture: ArchitectureLayered,
		CodingStyle: CodingStyle{
			NamingConvention: NamingConvention{
				Functions:  "camelCase",
				Variables:  "camelCase",
				Constants:  "UPPER_SNAKE",
				Types:      "PascalCase",
				Packages:   "lowercase",
				Files:      "snake_case",
				Interfaces: "PascalCase",
			},
			ErrorHandlingStyle: "explicit error returns",
			LoggingPattern:     "structured logging",
			CommonPatterns:     []string{"error handling", "dependency injection"},
			CodeFormatting: CodeFormatting{
				IndentStyle: "tabs",
				IndentSize:  4,
				LineLength:  100,
				BraceStyle:  "K&R",
			},
		},
		Dependencies:   []string{},
		TestFrameworks: []string{"testing", "testify"},
		BuildSystem:    "go build",
	}, nil
}

// Additional helper methods (simplified implementations)

func (ca *CodingAgentImpl) findRelevantTypes(ctx context.Context, intent *CodingAgentIntent, query *models.Query) ([]TypeDefinition, error) {
	return []TypeDefinition{}, nil
}

func (ca *CodingAgentImpl) findRelevantFunctions(ctx context.Context, intent *CodingAgentIntent, query *models.Query) ([]FunctionDef, error) {
	return []FunctionDef{}, nil
}

func (ca *CodingAgentImpl) analyzeProjectPatterns(ctx context.Context, intent *CodingAgentIntent) ([]ProjectPattern, error) {
	return []ProjectPattern{
		{
			Name:       "error handling",
			Pattern:    "if err != nil { return err }",
			Type:       "error_handling",
			Context:    "function error returns",
			Frequency:  50,
			Confidence: 0.95,
		},
	}, nil
}

func (ca *CodingAgentImpl) generateImportSuggestions(ctx context.Context, intent *CodingAgentIntent, context *CodeContext) ([]ImportSuggestion, error) {
	suggestions := []ImportSuggestion{
		{
			Import:     "fmt",
			Usage:      "string formatting",
			Confidence: 0.9,
			Reason:     "commonly used in Go projects",
			Type:       "standard",
			IsUsed:     true,
		},
	}
	return suggestions, nil
}

func (ca *CodingAgentImpl) findUsageExamples(ctx context.Context, intent *CodingAgentIntent) ([]UsageExample, error) {
	return []UsageExample{}, nil
}

func (ca *CodingAgentImpl) buildSystemPrompt(context *CodeContext) string {
	var prompt strings.Builder

	prompt.WriteString("You are an expert Go developer working on a specific codebase. ")
	prompt.WriteString("Generate code that follows the existing project patterns and conventions.\n\n")

	if context != nil && context.ProjectInfo != nil {
		prompt.WriteString(fmt.Sprintf("Project Language: %s\n", context.ProjectInfo.Language))
		if context.ProjectInfo.Framework != "" {
			prompt.WriteString(fmt.Sprintf("Framework: %s\n", context.ProjectInfo.Framework))
		}
		prompt.WriteString(fmt.Sprintf("Architecture: %s\n", string(context.ProjectInfo.Architecture)))

		// Enhanced coding style information
		style := context.ProjectInfo.CodingStyle
		prompt.WriteString("\nCoding Style Guidelines:\n")
		prompt.WriteString(fmt.Sprintf("- Functions: %s\n", style.NamingConvention.Functions))
		prompt.WriteString(fmt.Sprintf("- Types: %s\n", style.NamingConvention.Types))
		prompt.WriteString(fmt.Sprintf("- Error Handling: %s\n", style.ErrorHandlingStyle))
		if style.LoggingPattern != "" {
			prompt.WriteString(fmt.Sprintf("- Logging: %s\n", style.LoggingPattern))
		}
	}

	prompt.WriteString("\nIMPORTANT: Generate clean, idiomatic Go code that matches the existing codebase style.\n")
	return prompt.String()
}

func (ca *CodingAgentImpl) buildCodeGenerationPrompt(intent *CodingAgentIntent, context *CodeContext, query *models.Query) string {
	var prompt strings.Builder

	lang := "Go"
	if query != nil && query.Language != "" {
		lang = query.Language
	}
	prompt.WriteString(fmt.Sprintf("Generate %s code for: %s\n\n", lang, intent.Description))

	// Include intent details
	if intent.FunctionName != "" {
		prompt.WriteString(fmt.Sprintf("Function Name: %s\n", intent.FunctionName))
	}

	// Include similar code examples
	if context != nil && len(context.SimilarCode) > 0 {
		prompt.WriteString("\nSimilar patterns from your codebase:\n")
		for i, example := range context.SimilarCode {
			if i >= ca.config.MaxExamples {
				break
			}
			prompt.WriteString(fmt.Sprintf("\nExample from %s:\n", example.File))
			prompt.WriteString("```go\n")
			prompt.WriteString(example.Code)
			prompt.WriteString("\n```\n")
		}
	}

	// Include project patterns
	if context != nil && len(context.Patterns) > 0 {
		prompt.WriteString("\nCommon patterns in your project:\n")
		for _, pattern := range context.Patterns {
			prompt.WriteString(fmt.Sprintf("- %s: %s\n", pattern.Name, pattern.Pattern))
		}
	}

	prompt.WriteString("\nGenerate production-ready code with proper error handling and documentation.")
	return prompt.String()
}

func (ca *CodingAgentImpl) parseCodeResponse(content string, intent *CodingAgentIntent, language string) (*models.CodeResponse, error) {
	// This should ideally parse structured LLM output; currently returns raw content as Code field.
	return &models.CodeResponse{
		Language:    language,
		Code:        content,
		Explanation: "Generated code based on your project patterns",
		Changes:     []models.CodeChange{},
		Tests:       []models.TestCase{},
		Intent:      intent,
	}, nil
}

func (ca *CodingAgentImpl) validateGeneratedCode(response *models.CodeResponse, intent *CodingAgentIntent) (*models.CodeValidation, error) {
	// Implement code validation logic
	return &models.CodeValidation{
		IsValid:  true,
		Issues:   []models.ValidationIssue{},
		Warnings: []models.ValidationIssue{},
		Score:    0.9,
	}, nil
}

func (ca *CodingAgentImpl) calculateCodeConfidence(context *CodeContext, response *models.CodeResponse) float64 {
	factors := map[string]float64{}

	if context != nil && len(context.SimilarCode) > 0 {
		factors["similar_examples"] = 0.8
	} else {
		factors["similar_examples"] = 0.4
	}

	if context != nil && len(context.Patterns) > 0 {
		factors["project_patterns"] = 0.9
	} else {
		factors["project_patterns"] = 0.5
	}

	if response != nil && response.Validation != nil && response.Validation.IsValid {
		factors["validation"] = 0.95
	} else {
		factors["validation"] = 0.7
	}

	return CalculateConfidence(factors)
}

func (ca *CodingAgentImpl) buildResponse(query *models.Query, intent *CodingAgentIntent, context *CodeContext,
	codeResponse *models.CodeResponse, tokenUsage *models.TokenUsage, confidence float64, startTime time.Time) *models.Response {

	metadata := models.ResponseMetadata{
		GenerationTime: time.Since(startTime),
		IndexHits:      0,
		FilesAnalyzed:  ca.countContextFiles(context),
		Confidence:     confidence,
		Sources:        ca.extractContextSources(context),
		Tools:          []string{"vector_search", "pattern_analysis", "llm_generation"},
		Reasoning:      ca.explainCodeGeneration(intent, context),
	}

	if tokenUsage == nil {
		tokenUsage = &models.TokenUsage{}
	}

	return &models.Response{
		ID:      fmt.Sprintf("coding_response_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    models.ResponseTypeCode,
		Content: models.ResponseContent{
			Text: fmt.Sprintf("Generated %s code based on your project patterns:", query.Language),
			Code: codeResponse,
		},
		AgentUsed:  "coding_agent",
		Provider:   codeResponse.Provider,
		TokenUsage: *tokenUsage,
		Cost: models.Cost{
			TotalCost: 0.0,
			Currency:  "USD",
		},
		Metadata:  metadata,
		Timestamp: time.Now(),
	}
}

// Utility methods

func (ca *CodingAgentImpl) logStep(message string, fields map[string]interface{}) {
	if ca.dependencies != nil && ca.dependencies.Logger != nil {
		ca.dependencies.Logger.Info(message, fields)
	}
}

func (ca *CodingAgentImpl) updateMetrics(startTime time.Time) {
	if ca.metrics == nil {
		ca.metrics = &AgentMetrics{}
	}
	ca.metrics.QueriesHandled++
	ca.metrics.LastUsed = startTime
}

func (ca *CodingAgentImpl) updateSuccessMetrics(startTime time.Time, confidence float64, tokenUsage *models.TokenUsage) {
	if ca.metrics == nil {
		ca.metrics = &AgentMetrics{}
	}

	duration := time.Since(startTime)
	total := float64(ca.metrics.QueriesHandled)
	if total <= 0 {
		total = 1
	}
	ca.metrics.AverageResponseTime = time.Duration(
		(float64(ca.metrics.AverageResponseTime)*(total-1) + float64(duration)) / total,
	)
	ca.metrics.AverageConfidence = (ca.metrics.AverageConfidence*(total-1) + confidence) / total
	if tokenUsage != nil {
		ca.metrics.TokensUsed += int64(tokenUsage.TotalTokens)
	}

	successCount := float64(ca.metrics.QueriesHandled - ca.metrics.ErrorCount)
	ca.metrics.SuccessRate = successCount / total
}

func (ca *CodingAgentImpl) countContextFiles(context *CodeContext) int {
	files := make(map[string]bool)
	if context != nil {
		for _, example := range context.SimilarCode {
			files[example.File] = true
		}
		for _, typedef := range context.RelevantTypes {
			files[typedef.File] = true
		}
	}
	return len(files)
}

func (ca *CodingAgentImpl) extractContextSources(context *CodeContext) []string {
	sources := make(map[string]bool)
	if context != nil {
		for _, example := range context.SimilarCode {
			sources[example.File] = true
		}
		for _, typedef := range context.RelevantTypes {
			sources[typedef.File] = true
		}
	}
	sourceList := make([]string, 0, len(sources))
	for source := range sources {
		sourceList = append(sourceList, source)
	}
	return sourceList
}

func (ca *CodingAgentImpl) explainCodeGeneration(intent *CodingAgentIntent, context *CodeContext) string {
	simCount := 0
	patCount := 0
	if context != nil {
		simCount = len(context.SimilarCode)
		patCount = len(context.Patterns)
	}
	return fmt.Sprintf("Generated %s code using %d similar examples and %d patterns from your project",
		string(intent.Type), simCount, patCount)
}

// Simplified extraction methods (placeholders for future enhancement)
func (ca *CodingAgentImpl) extractFunctionName(input string) string { return "" }
func (ca *CodingAgentImpl) extractParameters(input string) []AgentParameter {
	return []AgentParameter{}
}
func (ca *CodingAgentImpl) extractReturnType(input string) string         { return "" }
func (ca *CodingAgentImpl) extractTargetFile(input string) string         { return "" }
func (ca *CodingAgentImpl) extractFramework(input string) string          { return "" }
func (ca *CodingAgentImpl) extractLibraries(input string) []string        { return []string{} }
func (ca *CodingAgentImpl) extractConstraints(input string) []string      { return []string{} }
func (ca *CodingAgentImpl) extractRequiredFeatures(input string) []string { return []string{} }

// Code analysis methods (simplified implementations)
func (ca *CodingAgentImpl) calculateCyclomaticComplexity(code string) int { return 1 }
func (ca *CodingAgentImpl) calculateCognitiveComplexity(code string) int  { return 1 }
func (ca *CodingAgentImpl) detectCodePatterns(code string) []DetectedPattern {
	return []DetectedPattern{}
}
func (ca *CodingAgentImpl) generateCodeSuggestions(code string) []CodeSuggestion {
	return []CodeSuggestion{}
}
func (ca *CodingAgentImpl) detectCodeIssues(code string) []CodeIssue { return []CodeIssue{} }

// MCP Context Enhancement Methods
func (ca *CodingAgentImpl) enhanceIntentWithMCP(intent *CodingAgentIntent, mcpContext *models.MCPContext) {
	// Add file context from MCP to Libraries (as reference files)
	if files, ok := mcpContext.Data["project_files"].([]map[string]interface{}); ok {
		for _, file := range files[:min(5, len(files))] { // Limit to 5 files
			if path, ok := file["path"].(string); ok {
				intent.Libraries = append(intent.Libraries, path)
			}
		}
	}
	
	// Add project structure insights to Context
	if structure, ok := mcpContext.Data["project_structure"].(map[string]interface{}); ok {
		intent.Context = ca.extractProjectPatterns(structure)
	}
	
	// Add file count as constraint
	if count, ok := mcpContext.Data["file_count"].(int); ok {
		intent.Constraints = append(intent.Constraints, fmt.Sprintf("project_has_%d_files", count))
	}
}

func (ca *CodingAgentImpl) getMCPDataKeys(mcpContext *models.MCPContext) []string {
	keys := make([]string, 0, len(mcpContext.Data))
	for k := range mcpContext.Data {
		keys = append(keys, k)
	}
	return keys
}

func (ca *CodingAgentImpl) extractProjectPatterns(structure map[string]interface{}) string {
	patterns := []string{}
	if _, hasInternal := structure["internal"]; hasInternal {
		patterns = append(patterns, "internal_structure")
	}
	if _, hasCmd := structure["cmd"]; hasCmd {
		patterns = append(patterns, "cmd_pattern")
	}
	if _, hasModels := structure["models"]; hasModels {
		patterns = append(patterns, "models_layer")
	}
	return strings.Join(patterns, ",")
}

// buildMCPEnhancedSystemPrompt builds system prompt enhanced with MCP context
func (ca *CodingAgentImpl) buildMCPEnhancedSystemPrompt(context *CodeContext, mcpContext *models.MCPContext) string {
	basePrompt := ca.buildSystemPrompt(context)
	
	if mcpContext == nil || !mcpContext.RequiresMCP {
		return basePrompt
	}
	
	mcpInfo := ca.extractMCPPromptInfo(mcpContext)
	if mcpInfo == "" {
		return basePrompt
	}
	
	return fmt.Sprintf(`%s

PROJECT CONTEXT FROM FILESYSTEM ANALYSIS:
%s

Generate code that follows the existing project patterns and structure.`, basePrompt, mcpInfo)
}

// extractMCPPromptInfo extracts relevant MCP information for prompts
func (ca *CodingAgentImpl) extractMCPPromptInfo(mcpContext *models.MCPContext) string {
	var info []string
	
	// Add file count context
	if count, ok := mcpContext.Data["file_count"].(int); ok {
		info = append(info, fmt.Sprintf("- Project contains %d files", count))
	}
	
	// Add key files for reference
	if files, ok := mcpContext.Data["project_files"].([]map[string]interface{}); ok {
		if len(files) > 0 {
			info = append(info, "- Key project files:")
			for _, file := range files[:min(3, len(files))] {
				if path, ok := file["path"].(string); ok {
					info = append(info, fmt.Sprintf("  * %s", path))
				}
			}
		}
	}
	
	// Add architectural patterns
	if structure, ok := mcpContext.Data["project_structure"].(map[string]interface{}); ok {
		patterns := ca.extractArchPatterns(structure)
		if len(patterns) > 0 {
			info = append(info, fmt.Sprintf("- Architecture: %s", strings.Join(patterns, ", ")))
		}
	}
	
	return strings.Join(info, "\n")
}

// extractArchPatterns extracts architectural patterns for prompts
func (ca *CodingAgentImpl) extractArchPatterns(structure map[string]interface{}) []string {
	patterns := []string{}
	
	if _, hasInternal := structure["internal"]; hasInternal {
		patterns = append(patterns, "layered architecture")
	}
	if _, hasCmd := structure["cmd"]; hasCmd {
		patterns = append(patterns, "CLI application")
	}
	if _, hasModels := structure["models"]; hasModels {
		patterns = append(patterns, "domain models")
	}
	
	return patterns
}
