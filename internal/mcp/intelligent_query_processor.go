package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// IntelligentQueryProcessor orchestrates the complete query processing pipeline
type IntelligentQueryProcessor struct {
	intentClassifier    *IntentClassifier
	contextGatherer     *ParallelContextGatherer
	promptBuilder       *AdaptivePromptBuilder
	commandExecutor     *IntelligentExecutor
	responseProcessor   *ResponseProcessor
	learningEngine      *LearningEngine
	performanceOptimizer *PerformanceOptimizer
}

// QueryProcessingPlan represents the execution plan for a query
type QueryProcessingPlan struct {
	QueryID             string                 `json:"query_id"`
	Intent              *ClassifiedIntent      `json:"intent"`
	RequiredOperations  []string               `json:"required_operations"`
	ContextDepth        ContextDepth           `json:"context_depth"`
	TokenBudget         int                    `json:"token_budget"`
	QualityThreshold    float64                `json:"quality_threshold"`
	FallbackStrategies  []string               `json:"fallback_strategies"`
	ParallelOperations  []ParallelOperation    `json:"parallel_operations"`
	EstimatedDuration   time.Duration          `json:"estimated_duration"`
	CacheStrategy       CacheStrategy          `json:"cache_strategy"`
}

// ClassifiedIntent represents intelligent intent classification
type ClassifiedIntent struct {
	Primary             IntentType             `json:"primary"`
	Secondary           []IntentType           `json:"secondary"`
	Confidence          float64                `json:"confidence"`
	ComplexityLevel     int                    `json:"complexity_level"` // 1-10
	RequiredContext     []ContextType          `json:"required_context"`
	ExpectedOutputType  OutputType             `json:"expected_output_type"`
	QualityRequirements QualityRequirements    `json:"quality_requirements"`
	Keywords            []string               `json:"keywords"`
	Entities            []ExtractedEntity      `json:"entities"`
}

// IntentType represents different types of user intents
type IntentType string

const (
	IntentExplain      IntentType = "explain"
	IntentGenerate     IntentType = "generate"
	IntentSearch       IntentType = "search"
	IntentAnalyze      IntentType = "analyze"
	IntentOptimize     IntentType = "optimize"
	IntentRefactor     IntentType = "refactor"
	IntentDebug        IntentType = "debug"
	IntentTest         IntentType = "test"
	IntentDocument     IntentType = "document"
	IntentSystemStatus IntentType = "system_status"
)

// ContextType represents types of context needed
type ContextType string

const (
	ContextProjectStructure ContextType = "project_structure"
	ContextCodeExamples     ContextType = "code_examples"
	ContextDependencies     ContextType = "dependencies"
	ContextArchitecture     ContextType = "architecture"
	ContextSystemInfo       ContextType = "system_info"
	ContextGitInfo          ContextType = "git_info"
	ContextUsagePatterns    ContextType = "usage_patterns"
)

// ContextDepth represents how much context to gather
type ContextDepth string

const (
	ContextMinimal      ContextDepth = "minimal"      // Basic info only
	ContextModerate     ContextDepth = "moderate"     // Standard context
	ContextComprehensive ContextDepth = "comprehensive" // Full context
)

// OutputType represents expected output types
type OutputType string

const (
	OutputExplanation OutputType = "explanation"
	OutputCode        OutputType = "code"
	OutputAnalysis    OutputType = "analysis"
	OutputList        OutputType = "list"
	OutputStatus      OutputType = "status"
)

// QualityRequirements represents quality requirements
type QualityRequirements struct {
	MinConfidence    float64 `json:"min_confidence"`
	RequireExamples  bool    `json:"require_examples"`
	RequireContext   bool    `json:"require_context"`
	RequireValidation bool   `json:"require_validation"`
}

// ParallelOperation represents operations that can run in parallel
type ParallelOperation struct {
	Type        string                 `json:"type"`
	Priority    int                    `json:"priority"`
	Timeout     time.Duration          `json:"timeout"`
	CacheKey    string                 `json:"cache_key"`
	Fallback    string                 `json:"fallback"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// CacheStrategy represents caching strategy
type CacheStrategy struct {
	UseCache     bool          `json:"use_cache"`
	TTL          time.Duration `json:"ttl"`
	PreCache     bool          `json:"pre_cache"`
	InvalidateOn []string      `json:"invalidate_on"`
}

// ExtractedEntity represents entities extracted from query
type ExtractedEntity struct {
	Type       string  `json:"type"`
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence"`
	Context    string  `json:"context"`
}

// NewIntelligentQueryProcessor creates a new intelligent query processor
func NewIntelligentQueryProcessor() *IntelligentQueryProcessor {
	return &IntelligentQueryProcessor{
		intentClassifier:     NewIntentClassifier(),
		contextGatherer:      NewParallelContextGatherer(),
		promptBuilder:        NewAdaptivePromptBuilder(),
		commandExecutor:      NewIntelligentExecutor(),
		responseProcessor:    NewResponseProcessor(),
		learningEngine:       NewLearningEngine(),
		performanceOptimizer: NewPerformanceOptimizer(),
	}
}

// ProcessQuery executes the complete intelligent query processing pipeline
func (iqp *IntelligentQueryProcessor) ProcessQuery(ctx context.Context, query *models.Query) (*models.Response, error) {
	startTime := time.Now()
	
	// Step 1: INTELLIGENT INTENT CLASSIFICATION
	intent, err := iqp.intentClassifier.ClassifyIntent(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("intent classification failed: %w", err)
	}
	
	// Step 2: CREATE EXECUTION PLAN
	plan, err := iqp.createExecutionPlan(ctx, query, intent)
	if err != nil {
		return nil, fmt.Errorf("execution plan creation failed: %w", err)
	}
	
	// Step 3: PARALLEL CONTEXT GATHERING
	contextData, err := iqp.contextGatherer.GatherContext(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("context gathering failed: %w", err)
	}
	
	// Step 4: INTELLIGENT CONTEXT FILTERING
	filteredContext := iqp.filterAndPrioritizeContext(contextData, plan)
	
	// Step 5: ADAPTIVE PROMPT CONSTRUCTION
	prompt, err := iqp.promptBuilder.BuildPrompt(ctx, query, intent, filteredContext)
	if err != nil {
		return nil, fmt.Errorf("prompt construction failed: %w", err)
	}
	
	// Step 6: LLM GENERATION WITH FALLBACK
	response, err := iqp.generateWithFallback(ctx, prompt, plan)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}
	
	// Step 7: RESPONSE POST-PROCESSING
	enhancedResponse := iqp.responseProcessor.EnhanceResponse(response, contextData, plan)
	
	// Step 8: LEARNING & FEEDBACK LOOP
	iqp.learningEngine.RecordSuccess(query, intent, plan, time.Since(startTime))
	
	return enhancedResponse, nil
}

// createExecutionPlan creates an intelligent execution plan
func (iqp *IntelligentQueryProcessor) createExecutionPlan(ctx context.Context, query *models.Query, intent *ClassifiedIntent) (*QueryProcessingPlan, error) {
	plan := &QueryProcessingPlan{
		QueryID:            query.ID,
		Intent:             intent,
		RequiredOperations: iqp.determineRequiredOperations(intent),
		ContextDepth:       iqp.determineContextDepth(intent),
		TokenBudget:        iqp.calculateTokenBudget(intent),
		QualityThreshold:   iqp.determineQualityThreshold(intent),
		FallbackStrategies: iqp.determineFallbackStrategies(intent),
		ParallelOperations: iqp.planParallelOperations(intent),
		CacheStrategy:      iqp.determineCacheStrategy(intent),
	}
	
	return plan, nil
}

// determineRequiredOperations intelligently determines what operations are needed
func (iqp *IntelligentQueryProcessor) determineRequiredOperations(intent *ClassifiedIntent) []string {
	operations := []string{}
	
	switch intent.Primary {
	case IntentExplain:
		operations = append(operations, "filesystem_structure", "code_analysis", "dependency_mapping")
		if iqp.needsArchitecturalContext(intent) {
			operations = append(operations, "architecture_analysis", "component_mapping")
		}
		
	case IntentGenerate:
		operations = append(operations, "pattern_search", "similar_code_examples", "dependency_analysis")
		if iqp.needsProjectContext(intent) {
			operations = append(operations, "project_conventions", "coding_standards")
		}
		
	case IntentSearch:
		operations = append(operations, "semantic_search", "keyword_search")
		if iqp.needsUsageExamples(intent) {
			operations = append(operations, "usage_pattern_search", "related_code_search")
		}
		
	case IntentSystemStatus:
		operations = append(operations, "system_info", "file_count", "index_status")
		
	default:
		operations = append(operations, "general_context", "basic_search")
	}
	
	return operations
}

// determineContextDepth determines how much context to gather
func (iqp *IntelligentQueryProcessor) determineContextDepth(intent *ClassifiedIntent) ContextDepth {
	if intent.ComplexityLevel >= 8 {
		return ContextComprehensive
	} else if intent.ComplexityLevel >= 5 {
		return ContextModerate
	}
	return ContextMinimal
}

// calculateTokenBudget calculates appropriate token budget
func (iqp *IntelligentQueryProcessor) calculateTokenBudget(intent *ClassifiedIntent) int {
	baseBudget := 2000
	
	switch intent.Primary {
	case IntentExplain:
		return baseBudget + 2000 // Explanations need more tokens
	case IntentGenerate:
		return baseBudget + 1500 // Code generation needs more tokens
	case IntentAnalyze:
		return baseBudget + 1000 // Analysis needs moderate tokens
	default:
		return baseBudget
	}
}

// Helper methods for context determination
func (iqp *IntelligentQueryProcessor) needsArchitecturalContext(intent *ClassifiedIntent) bool {
	architecturalKeywords := []string{"architecture", "flow", "structure", "design", "components"}
	for _, keyword := range architecturalKeywords {
		for _, entityKeyword := range intent.Keywords {
			if strings.Contains(entityKeyword, keyword) {
				return true
			}
		}
	}
	return false
}

func (iqp *IntelligentQueryProcessor) needsProjectContext(intent *ClassifiedIntent) bool {
	return intent.ComplexityLevel >= 6 || len(intent.RequiredContext) > 2
}

func (iqp *IntelligentQueryProcessor) needsUsageExamples(intent *ClassifiedIntent) bool {
	usageKeywords := []string{"example", "usage", "how to", "pattern"}
	for _, keyword := range usageKeywords {
		for _, entityKeyword := range intent.Keywords {
			if strings.Contains(entityKeyword, keyword) {
				return true
			}
		}
	}
	return false
}

func (iqp *IntelligentQueryProcessor) planParallelOperations(intent *ClassifiedIntent) []ParallelOperation {
	operations := []ParallelOperation{}
	
	// Always include basic filesystem operation
	operations = append(operations, ParallelOperation{
		Type:     "filesystem_scan",
		Priority: 1,
		Timeout:  5 * time.Second,
		CacheKey: "project_structure",
	})
	
	// Add vector search if needed
	if intent.Primary == IntentSearch || intent.Primary == IntentExplain {
		operations = append(operations, ParallelOperation{
			Type:     "vector_search",
			Priority: 2,
			Timeout:  10 * time.Second,
			CacheKey: fmt.Sprintf("search_%s", strings.Join(intent.Keywords, "_")),
		})
	}
	
	// Add system info if needed
	if intent.Primary == IntentSystemStatus {
		operations = append(operations, ParallelOperation{
			Type:     "system_info",
			Priority: 1,
			Timeout:  3 * time.Second,
			CacheKey: "system_status",
		})
	}
	
	return operations
}

func (iqp *IntelligentQueryProcessor) determineFallbackStrategies(intent *ClassifiedIntent) []string {
	strategies := []string{"openai", "gemini"}
	
	if intent.Primary == IntentSearch {
		strategies = append(strategies, "vector_search_only")
	}
	
	return strategies
}

func (iqp *IntelligentQueryProcessor) determineCacheStrategy(intent *ClassifiedIntent) CacheStrategy {
	return CacheStrategy{
		UseCache:     true,
		TTL:          15 * time.Minute,
		PreCache:     intent.ComplexityLevel >= 7,
		InvalidateOn: []string{"file_change", "config_change"},
	}
}

func (iqp *IntelligentQueryProcessor) filterAndPrioritizeContext(contextData *GatheredContext, plan *QueryProcessingPlan) *FilteredContext {
	// Implement intelligent context filtering
	return &FilteredContext{
		ProjectInfo:   contextData.ProjectInfo,
		RelevantFiles: contextData.RelevantFiles[:min(10, len(contextData.RelevantFiles))],
		CodeExamples:  contextData.CodeExamples[:min(5, len(contextData.CodeExamples))],
		SystemInfo:    contextData.SystemInfo,
	}
}

func (iqp *IntelligentQueryProcessor) generateWithFallback(ctx context.Context, prompt *AdaptivePrompt, plan *QueryProcessingPlan) (*models.Response, error) {
	// This will be implemented to call LLM with fallback
	return &models.Response{
		ID:      fmt.Sprintf("intelligent_response_%d", time.Now().UnixNano()),
		QueryID: plan.QueryID,
		Type:    models.ResponseTypeExplanation,
		Content: models.ResponseContent{
			Text: "Intelligent response generated with full context",
		},
		AgentUsed: "intelligent_processor",
		Timestamp: time.Now(),
	}, nil
}

// Supporting types
type GatheredContext struct {
	ProjectInfo   map[string]interface{} `json:"project_info"`
	RelevantFiles []string               `json:"relevant_files"`
	CodeExamples  []string               `json:"code_examples"`
	SystemInfo    map[string]interface{} `json:"system_info"`
}

type FilteredContext struct {
	ProjectInfo   map[string]interface{} `json:"project_info"`
	RelevantFiles []string               `json:"relevant_files"`
	CodeExamples  []string               `json:"code_examples"`
	SystemInfo    map[string]interface{} `json:"system_info"`
}

type AdaptivePrompt struct {
	SystemPrompt string `json:"system_prompt"`
	UserPrompt   string `json:"user_prompt"`
	Context      string `json:"context"`
	Examples     string `json:"examples"`
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}