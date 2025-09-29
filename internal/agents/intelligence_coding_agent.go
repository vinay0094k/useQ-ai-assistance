package agents

import (
	"context"
	"crypto/md5"
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/useq-ai-assistant/internal/llm"
)

// ------------------------------------------------------------------
// Local helper interfaces / wrappers (to avoid changing centralized types)
// ------------------------------------------------------------------

// IntelligenceProcessor defines how a single intelligence layer processes code/context.
type IntelligenceProcessor interface {
	Process(ctx context.Context, code string, ctxObj *IntelligenceCodingAgentDeepAnalysisContext) (*LayerResult, error)
	GetCapabilities() []string
	Configure(config map[string]interface{}) error
}

// LayerResult is a lightweight result from a layer processor.
type LayerResult struct {
	Name        string
	Findings    []string
	Metrics     map[string]float64
	Confidence  float64
	Annotations map[string]interface{}
}

// IntelligenceLayer wraps the centralized IntelligenceCodingAgentLayer with a Processor
// so we can keep processor behavior local to this file.
type IntelligenceLayer struct {
	IntelligenceCodingAgentLayer
	Processor IntelligenceProcessor
}

// Basic interfaces for other agents (simplified to avoid dependencies)
type BasicCodingAgent interface {
	GetCodeContext(ctx context.Context, query *Query) (*BasicCodeContext, error)
}

type BasicSearchAgent interface {
	SearchCode(ctx context.Context, query string) ([]IntelligenceCodingAgentSearchResult, error)
}

// Simplified context types to avoid circular dependencies
type BasicCodeContext struct {
	Query        string
	Language     string
	Framework    string
	ProjectPath  string
	Dependencies []string
}

type IntelligenceCodingAgentSearchResult struct {
	File    string
	Line    int
	Content string
	Score   float64
}

// Query represents a user query (simplified)
type Query struct {
	ID        string
	UserInput string
	Language  string
	Context   map[string]interface{}
}

// Response represents an AI response (simplified)
type Response struct {
	ID         string
	QueryID    string
	Type       string
	Content    ResponseContent
	AgentUsed  string
	Provider   string
	TokenUsage TokenUsage
	Cost       Cost
	Metadata   ResponseMetadata
	Timestamp  time.Time
}

type ResponseContent struct {
	Text string
	Code *CodeResponse
}

type CodeResponse struct {
	Code         string
	Language     string
	Explanation  string
	Tests        []string
	Dependencies []string
	Validation   *CodeValidation
}

type CodeValidation struct {
	IsValid bool
	Errors  []string
}

type TokenUsage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

type Cost struct {
	TotalCost float64
	Currency  string
}

type ResponseMetadata struct {
	GenerationTime time.Duration
	Confidence     float64
}

// ------------------------------------------------------------------
// IntelligenceCodingAgentImpl
// ------------------------------------------------------------------

// IntelligenceCodingAgentImpl provides deep, AI-powered code understanding and generation.
// Uses centralized types defined in agent_types.go and intelligence types.
type IntelligenceCodingAgentImpl struct {
	dependencies       *AgentDependencies
	config             *IntelligenceCodingAgentConfig
	metrics            *AgentMetrics
	searchAgent        BasicSearchAgent
	codingAgent        BasicCodingAgent
	intelligenceLayers []IntelligenceLayer
	analysisCache      map[string]*IntelligenceCodingAgentDeepAnalysisResult
	patternDatabase    *IntelligenceCodingAgentPatternDatabase
}

// NewIntelligenceCodingAgent creates a new intelligence coding agent.
// supply pointers to the search and coding agents (or nil if not available).
func NewIntelligenceCodingAgent(deps *AgentDependencies, searchAgent BasicSearchAgent, codingAgent BasicCodingAgent) *IntelligenceCodingAgentImpl {
	agent := &IntelligenceCodingAgentImpl{
		dependencies:    deps,
		config:          NewIntelligenceCodingAgentConfig(),
		searchAgent:     searchAgent,
		codingAgent:     codingAgent,
		analysisCache:   make(map[string]*IntelligenceCodingAgentDeepAnalysisResult),
		patternDatabase: NewIntelligenceCodingAgentPatternDatabase(),
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

	// Initialize intelligence layers (and their local processors)
	agent.initializeIntelligenceLayers()

	return agent
}

// GetCapabilities returns enhanced capabilities
func (ica *IntelligenceCodingAgentImpl) GetCapabilities() AgentCapabilities {
	return AgentCapabilities{
		CanGenerateCode:    true,
		CanSearchCode:      true,
		CanAnalyzeCode:     true,
		CanDebugCode:       true,
		CanWriteTests:      true,
		CanWriteDocs:       true,
		CanReviewCode:      true,
		SupportedLanguages: []string{"go", "javascript", "python", "rust", "java", "typescript"},
		MaxComplexity:      10,
		RequiresContext:    true,
	}
}

// GetSpecialization returns the agent's specialization
func (ica *IntelligenceCodingAgentImpl) GetSpecialization() AgentSpecialization {
	return AgentSpecialization{
		Type:        AgentTypeCoding,
		Languages:   []string{"go", "javascript", "python", "rust", "java", "typescript"},
		Frameworks:  []string{"*"},
		Domains:     []string{"architecture", "performance", "security", "quality", "patterns"},
		Complexity:  10,
		Description: "Advanced AI-powered code analysis, generation, and optimization with deep architectural understanding",
	}
}

// CanHandle determines if this agent can handle the query with intelligence assessment
func (ica *IntelligenceCodingAgentImpl) CanHandle(ctx context.Context, query *Query) (bool, float64) {
	deepIntent, err := ica.parseDeepIntent(query)
	if err != nil {
		return false, 0.0
	}
	confidence := ica.calculateIntelligenceConfidence(deepIntent, query)
	canHandle := confidence >= 0.7 && ica.requiresIntelligentProcessing(deepIntent)
	return canHandle, confidence
}

// GetConfidenceScore returns detailed confidence
func (ica *IntelligenceCodingAgentImpl) GetConfidenceScore(ctx context.Context, query *Query) float64 {
	deepIntent, err := ica.parseDeepIntent(query)
	if err != nil {
		return 0.0
	}
	return ica.calculateIntelligenceConfidence(deepIntent, query)
}

// ValidateQuery performs advanced query validation
func (ica *IntelligenceCodingAgentImpl) ValidateQuery(query *Query) error {
	if query == nil {
		return fmt.Errorf("query cannot be nil")
	}
	if strings.TrimSpace(query.UserInput) == "" {
		return fmt.Errorf("query input cannot be empty")
	}
	if query.Language == "" {
		query.Language = "go" // Default to Go
	}

	deepIntent, err := ica.parseDeepIntent(query)
	if err != nil {
		return fmt.Errorf("failed to parse deep intent: %w", err)
	}
	if !ica.requiresIntelligentProcessing(deepIntent) {
		return fmt.Errorf("query does not require intelligent processing")
	}
	return nil
}

// Process handles the query with full intelligence processing
func (ica *IntelligenceCodingAgentImpl) Process(ctx context.Context, query *Query) (*Response, error) {
	start := time.Now()
	ica.updateMetrics(start)

	// Check if LLM Manager is available
	if ica.dependencies.LLMManager == nil {
		return ica.createFallbackResponse(query, "LLM Manager not available for intelligent processing"), nil
	}

	ica.logStep("Starting intelligent code processing", map[string]interface{}{
		"query_id":   query.ID,
		"language":   query.Language,
		"input_type": ica.classifyInputType(query.UserInput),
		"complexity": ica.estimateComplexity(query.UserInput),
	})

	deepIntent, err := ica.parseDeepIntent(query)
	if err != nil {
		ica.metrics.ErrorCount++
		return nil, fmt.Errorf("failed to parse deep intent: %w", err)
	}

	ica.logStep("Parsed deep intent", map[string]interface{}{
		"intent_type":         deepIntent.Type,
		"complexity_level":    deepIntent.ComplexityLevel,
		"requires_analysis":   deepIntent.RequiresAnalysis,
		"requires_generation": deepIntent.RequiresGeneration,
		"domain_specific":     deepIntent.DomainSpecific,
	})

	// Build context for intelligence processing
	deepContext, err := ica.buildIntelligenceCodingAgentContext(ctx, deepIntent, query)
	if err != nil {
		ica.metrics.ErrorCount++
		return nil, fmt.Errorf("failed to build deep context: %w", err)
	}

	ica.logStep("Built deep code context", map[string]interface{}{
		"context_layers":      len(ica.intelligenceLayers),
		"patterns_detected":   len(deepContext.DetectedPatterns),
		"cross_file_refs":     ica.countCrossFileReferences(deepContext.Code, deepContext.Language),
		"architectural_depth": ica.getArchitecturalDepth(deepContext.Code, deepContext.Language),
	})

	// perform the multi-layer processing
	response, err := ica.processWithIntelligence(ctx, deepIntent, deepContext, query)
	if err != nil {
		ica.metrics.ErrorCount++
		return nil, fmt.Errorf("intelligent processing failed: %w", err)
	}

	// log completion and update metrics
	ica.logStep("Intelligence processing completed", map[string]interface{}{
		"response_id":        response.ID,
		"processing_time_ms": time.Since(start).Milliseconds(),
		"confidence":         response.Metadata.Confidence,
		"intelligence_score": ica.calculateIntelligenceScore(response),
	})

	ica.updateSuccessMetrics(start, response.Metadata.Confidence, &response.TokenUsage)
	return response, nil
}

// GetMetrics returns metrics snapshot
func (ica *IntelligenceCodingAgentImpl) GetMetrics() AgentMetrics {
	return *ica.metrics
}

// AnalyzeCode — wrapper that uses performDeepAnalysis
func (ica *IntelligenceCodingAgentImpl) AnalyzeCode(ctx context.Context, code string, language string) (*AgentCodeAnalysis, error) {
	ica.logStep("Starting deep code analysis", map[string]interface{}{
		"language":    language,
		"code_length": len(code),
	})

	// Build request
	req := &IntelligenceCodingAgentDeepAnalysisRequest{
		Code:         code,
		Language:     language,
		AnalysisType: "deep_intelligence",
		Depth:        ica.config.AnalysisDepth,
	}

	result, err := ica.performDeepAnalysis(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("deep analysis failed: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("no analysis result")
	}

	// result embeds AgentCodeAnalysis in the types file, so return that
	if result.AgentCodeAnalysis.Language == "" {
		result.AgentCodeAnalysis.Language = language
	}
	return &result.AgentCodeAnalysis, nil
}

// GetCodeContext builds enhanced code context
func (ica *IntelligenceCodingAgentImpl) GetCodeContext(ctx context.Context, query *Query) (*IntelligenceCodingAgentContext, error) {
	if ica.codingAgent == nil {
		return &IntelligenceCodingAgentContext{}, nil
	}
	codeCtx, err := ica.codingAgent.GetCodeContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return &IntelligenceCodingAgentContext{
		Query: codeCtx.Query,
		ProjectContext: map[string]interface{}{
			"language":     codeCtx.Language,
			"framework":    codeCtx.Framework,
			"project_path": codeCtx.ProjectPath,
		},
		FileContext: codeCtx.Dependencies,
	}, nil
}

// ------------------------------------------------------------------
// CORE INTELLIGENCE PROCESSING
// ------------------------------------------------------------------

// processWithIntelligence orchestrates the chosen strategy
func (ica *IntelligenceCodingAgentImpl) processWithIntelligence(ctx context.Context, intent *IntelligenceCodingAgentIntent,
	deepContext *IntelligenceCodingAgentDeepAnalysisContext, query *Query) (*Response, error) {

	strategy := ica.determineProcessingStrategy(intent, deepContext)
	ica.logStep("Determined processing strategy", map[string]interface{}{
		"strategy_type":   strategy.Type,
		"layers_involved": len(ica.intelligenceLayers),
	})

	var response *Response
	var err error

	switch strategy.Type {
	case StrategyDeepAnalysis:
		response, err = ica.performIntelligentAnalysis(ctx, intent, deepContext, query)
	case StrategyIntelligentGeneration:
		response, err = ica.performIntelligentGeneration(ctx, intent, deepContext, query)
	case StrategyOptimization:
		response, err = ica.performIntelligentOptimization(ctx, intent, deepContext, query)
	case StrategyRefactoring:
		response, err = ica.performIntelligentRefactoring(ctx, intent, deepContext, query)
	case StrategyArchitecturalDesign:
		response, err = ica.performArchitecturalDesign(ctx, intent, deepContext, query)
	case StrategyHybrid:
		response, err = ica.performHybridProcessing(ctx, intent, deepContext, query)
	default:
		return nil, fmt.Errorf("unknown processing strategy: %s", strategy.Type)
	}

	if err != nil {
		return nil, err
	}

	// annotate with intelligence metadata
	response = ica.enhanceResponseWithIntelligence(response, intent, deepContext, strategy)
	return response, nil
}

// performIntelligentGeneration — generate code with intelligence
func (ica *IntelligenceCodingAgentImpl) performIntelligentGeneration(ctx context.Context, intent *IntelligenceCodingAgentIntent,
	deepContext *IntelligenceCodingAgentDeepAnalysisContext, query *Query) (*Response, error) {

	ica.logStep("Starting intelligent code generation", map[string]interface{}{
		"generation_type": intent.GenerationType,
		"complexity":      intent.ComplexityLevel,
	})

	prompts := ica.buildIntelligentPrompts(intent, deepContext, query)
	codeResp, tokenUsage, err := ica.generateWithIntelligence(ctx, prompts, intent, deepContext)
	if err != nil {
		return nil, fmt.Errorf("intelligent generation failed: %w", err)
	}

	codeResp, err = ica.postProcessWithIntelligence(ctx, codeResp, intent, deepContext)
	if err != nil {
		return nil, fmt.Errorf("post-processing failed: %w", err)
	}

	// optional advanced validation
	if ica.config != nil && ica.config.EnableTesting {
		if validation, err := ica.performAdvancedValidation(ctx, codeResp, intent, deepContext); err == nil {
			codeResp.Validation = validation
		}
	}

	resp := ica.buildIntelligentResponse(query, intent, deepContext, codeResp, tokenUsage)
	return resp, nil
}

// performIntelligentAnalysis — deep code analysis
func (ica *IntelligenceCodingAgentImpl) performIntelligentAnalysis(ctx context.Context, intent *IntelligenceCodingAgentIntent,
	deepContext *IntelligenceCodingAgentDeepAnalysisContext, query *Query) (*Response, error) {

	ica.logStep("Starting intelligent code analysis", map[string]interface{}{
		"analysis_depth": ica.config.AnalysisDepth,
		"code_length":    len(query.UserInput),
		"cross_file":     ica.config.CrossFileAnalysis,
	})

	code := ica.extractCodeFromQuery(query)
	req := &IntelligenceCodingAgentDeepAnalysisRequest{
		Code:         code,
		Language:     query.Language,
		AnalysisType: "deep_intelligence",
		Depth:        ica.config.AnalysisDepth,
		Layers:       ica.convertLayersToCodingLayers(),
	}

	deepResult, err := ica.performDeepAnalysis(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("deep analysis failed: %w", err)
	}

	return ica.buildAnalysisResponse(query, intent, deepContext, deepResult), nil
}

// ------------------------------------------------------------------
// DEEP ANALYSIS ENGINE
// ------------------------------------------------------------------

// performDeepAnalysis executes each enabled layer and synthesizes results
func (ica *IntelligenceCodingAgentImpl) performDeepAnalysis(ctx context.Context, request *IntelligenceCodingAgentDeepAnalysisRequest) (*IntelligenceCodingAgentDeepAnalysisResult, error) {
	start := time.Now()

	cacheKey := ica.generateCacheKey(request)
	if cached, ok := ica.analysisCache[cacheKey]; ok {
		ica.logStep("Retrieved analysis from cache", map[string]interface{}{"cache_key": cacheKey})
		return cached, nil
	}

	// initialize a lightweight aggregator
	layerResults := make(map[string]*LayerResult)

	// build a IntelligenceCodingAgentDeepAnalysisContext
	dCtx := &IntelligenceCodingAgentDeepAnalysisContext{
		Code:           request.Code,
		Language:       request.Language,
		ProjectContext: request.Context,
	}

	// iterate over the configured layers (local wrapper)
	for _, layer := range ica.intelligenceLayers {
		if !layer.Enabled || layer.Processor == nil {
			continue
		}
		ica.logStep("Processing intelligence layer", map[string]interface{}{
			"layer_name": layer.Name,
			"layer_type": layer.Type,
			"weight":     layer.Weight,
		})

		lr, err := layer.Processor.Process(ctx, request.Code, dCtx)
		if err != nil {
			ica.logStep("Layer processing failed", map[string]interface{}{"layer_name": layer.Name, "error": err.Error()})
			continue
		}
		layerResults[layer.Name] = lr
	}

	// synthesize results (placeholder synthesis)
	result := &IntelligenceCodingAgentDeepAnalysisResult{
		AgentCodeAnalysis: AgentCodeAnalysis{
			Language:     request.Language,
			Complexity:   0.0,
			QualityScore: 0.0,
			Issues:       []string{},
			Suggestions:  []string{},
			Metadata:     map[string]interface{}{},
		},
		DeepInsights: []string{},
		Patterns:     []string{},
		Architecture: map[string]interface{}{},
		Performance:  map[string]interface{}{},
	}

	// simple aggregation logic
	totalConf := 0.0
	count := 0
	for _, lr := range layerResults {
		totalConf += lr.Confidence
		count++
		// merge findings
		for _, f := range lr.Findings {
			result.DeepInsights = append(result.DeepInsights, f)
			result.Patterns = append(result.Patterns, f)
		}
	}
	if count > 0 {
		result.ProcessingTime = time.Since(start)
		result.Confidence = totalConf / float64(count)
	} else {
		result.ProcessingTime = time.Since(start)
		result.Confidence = 0.0
	}

	// cache
	ica.analysisCache[cacheKey] = result

	ica.logStep("Deep analysis completed", map[string]interface{}{
		"processing_time_ms": result.ProcessingTime.Milliseconds(),
		"confidence":         result.Confidence,
		"layers_processed":   len(layerResults),
	})

	return result, nil
}

// ------------------------------------------------------------------
// LAYER INITIALIZATION
// ------------------------------------------------------------------

// initializeIntelligenceLayers sets up intelligence layers and their processors.
// Processors are minimal mocks by default; replace with real implementations.
func (ica *IntelligenceCodingAgentImpl) initializeIntelligenceLayers() {
	// local small helper to add a layer with a mock processor
	add := func(name, typ string, weight float64, enabled bool, cfg map[string]interface{}, proc IntelligenceProcessor) {
		ica.intelligenceLayers = append(ica.intelligenceLayers, IntelligenceLayer{
			IntelligenceCodingAgentLayer: IntelligenceCodingAgentLayer{
				Name:    name,
				Type:    typ,
				Weight:  weight,
				Enabled: enabled,
				Config:  cfg,
			},
			Processor: proc,
		})
	}

	// add layers (weights / enabled flags guided by config)
	add("syntactic_analysis", "syntactic", 0.20, true, map[string]interface{}{"depth": 5}, NewMockProcessor())
	add("semantic_analysis", "semantic", 0.25, true, map[string]interface{}{"llm_enhanced": true}, NewMockProcessor())
	add("architecture_analysis", "architecture", 0.20, ica.config.ArchitectureAnalysis, map[string]interface{}{"cross_file": ica.config.CrossFileAnalysis}, NewMockProcessor())
	add("performance_analysis", "performance", 0.15, ica.config.PerformanceAnalysis, map[string]interface{}{"optimization_focus": true}, NewMockProcessor())
	add("security_analysis", "security", 0.10, true, map[string]interface{}{"vulnerability_scan": true}, NewMockProcessor())
	add("quality_analysis", "quality", 0.10, true, map[string]interface{}{"maintainability_focus": true}, NewMockProcessor())

	ica.logStep("Initialized intelligence layers", map[string]interface{}{
		"total_layers":   len(ica.intelligenceLayers),
		"enabled_layers": ica.countEnabledLayers(),
	})
}

// ------------------------------------------------------------------
// HELPERS, PARSING, UTILITIES
// ------------------------------------------------------------------

// parseDeepIntent extracts deep intent from query
func (ica *IntelligenceCodingAgentImpl) parseDeepIntent(query *Query) (*IntelligenceCodingAgentIntent, error) {
	if query == nil {
		return nil, fmt.Errorf("query nil")
	}
	input := strings.ToLower(query.UserInput)

	intent := &IntelligenceCodingAgentIntent{
		Query:                query.UserInput,
		Type:                 IntelligenceCodingAgentType(ica.classifyDeepIntentType(input)),
		Language:             query.Language,
		ComplexityLevel:      ica.estimateComplexity(input),
		RequiresAnalysis:     ica.requiresAnalysis(input),
		RequiresGeneration:   ica.requiresGeneration(input),
		RequiresOptimization: ica.requiresOptimization(input),
		DomainSpecific:       ica.isDomainSpecific(input),
		CrossFileScope:       ica.requiresCrossFileAnalysis(input),
		ArchitecturalScope:   ica.requiresArchitecturalAnalysis(input),
	}

	intent.GenerationType = ica.determineGenerationType(input)
	intent.AnalysisType = ica.determineAnalysisType(input)
	intent.OptimizationTarget = ica.determineOptimizationTarget(input)
	intent.QualityFocus = ica.determineQualityFocus(input)

	return intent, nil
}

// buildIntelligenceCodingAgentContext creates comprehensive context
func (ica *IntelligenceCodingAgentImpl) buildIntelligenceCodingAgentContext(ctx context.Context, intent *IntelligenceCodingAgentIntent, query *Query) (*IntelligenceCodingAgentDeepAnalysisContext, error) {
	// try to get base context from codingAgent if present
	var base *IntelligenceCodingAgentContext
	if ica.codingAgent != nil {
		bctx, err := ica.codingAgent.GetCodeContext(ctx, query)
		if err == nil {
			base = &IntelligenceCodingAgentContext{
				Query: bctx.Query,
				ProjectContext: map[string]interface{}{
					"language":  bctx.Language,
					"framework": bctx.Framework,
				},
			}
		}
	}
	if base == nil {
		base = &IntelligenceCodingAgentContext{}
	}

	deep := &IntelligenceCodingAgentDeepAnalysisContext{
		Code:                 query.UserInput, // Fixed: use query.UserInput instead of undefined request.Code
		Language:             query.Language,  // Fixed: use query.Language instead of undefined request.Language
		ProjectContext:       query.Context,   // Fixed: use query.Context instead of undefined request.Context
		DetectedPatterns:     []string{},
		SemanticContext:      &AgentSemanticContext{},
		ArchitecturalContext: &AgentArchitecturalContext{},
	}

	// semantic context (placeholder)
	if intent.RequiresAnalysis || intent.ComplexityLevel > 5 {
		deep.SemanticContext = &AgentSemanticContext{}
	}

	// architectural context
	if intent.ArchitecturalScope || ica.config.ArchitectureAnalysis {
		deep.ArchitecturalContext = &AgentArchitecturalContext{}
	}

	// detect patterns when enabled
	if ica.config.PatternRecognition {
		patterns, _ := ica.detectIntelligentPatterns(ctx, query.UserInput, deep)
		deep.DetectedPatterns = patterns
	}

	return deep, nil
}

// calculateIntelligenceConfidence — aggregate many signals
func (ica *IntelligenceCodingAgentImpl) calculateIntelligenceConfidence(intent *IntelligenceCodingAgentIntent, query *Query) float64 {
	factors := map[string]float64{}

	if intent.ComplexityLevel >= 7 {
		factors["high_complexity"] = 0.9
	} else if intent.ComplexityLevel >= 4 {
		factors["medium_complexity"] = 0.7
	} else {
		factors["low_complexity"] = 0.4
	}

	if intent.RequiresAnalysis {
		factors["analysis_required"] = 0.8
	}
	if intent.RequiresGeneration && intent.ComplexityLevel > 5 {
		factors["complex_generation"] = 0.85
	}
	if intent.ArchitecturalScope {
		factors["architectural_scope"] = 0.9
	}
	if intent.DomainSpecific {
		factors["domain_specific"] = 0.75
	}

	langs := []string{"go", "javascript", "python", "rust", "java", "typescript"}
	for _, l := range langs {
		if query.Language == l {
			factors["language_support"] = 0.9
			break
		}
	}

	wordCount := len(strings.Fields(query.UserInput))
	if wordCount >= 5 {
		factors["query_clarity"] = 0.8
	} else if wordCount >= 3 {
		factors["query_clarity"] = 0.6
	} else {
		factors["query_clarity"] = 0.4
	}

	return CalculateConfidence(factors)
}

// Utility logging helper
func (ica *IntelligenceCodingAgentImpl) logStep(message string, fields map[string]interface{}) {
	if ica.dependencies != nil && ica.dependencies.Logger != nil {
		ica.dependencies.Logger.Info(message, fields)
	}
}

func (ica *IntelligenceCodingAgentImpl) updateMetrics(startTime time.Time) {
	ica.metrics.QueriesHandled++
	ica.metrics.LastUsed = startTime
}

func (ica *IntelligenceCodingAgentImpl) updateSuccessMetrics(startTime time.Time, confidence float64, tokenUsage *TokenUsage) {
	duration := time.Since(startTime)
	total := float64(ica.metrics.QueriesHandled)
	if total <= 0 {
		total = 1
	}
	ica.metrics.AverageResponseTime = time.Duration(
		(float64(ica.metrics.AverageResponseTime)*(total-1) + float64(duration)) / total,
	)
	ica.metrics.AverageConfidence = (ica.metrics.AverageConfidence*(total-1) + confidence) / total
	ica.metrics.TokensUsed += int64(tokenUsage.TotalTokens)
	successCount := float64(ica.metrics.QueriesHandled - ica.metrics.ErrorCount)
	ica.metrics.SuccessRate = successCount / float64(ica.metrics.QueriesHandled)
}

// convertLayersToCodingLayers returns a list of intelligence layers in the type expected by deep analysis request.
func (ica *IntelligenceCodingAgentImpl) convertLayersToCodingLayers() []IntelligenceCodingAgentLayer {
	layers := make([]IntelligenceCodingAgentLayer, 0, len(ica.intelligenceLayers))
	for _, l := range ica.intelligenceLayers {
		layers = append(layers, l.IntelligenceCodingAgentLayer)
	}
	return layers
}

// countEnabledLayers returns the number of enabled layers
func (ica *IntelligenceCodingAgentImpl) countEnabledLayers() int {
	count := 0
	for _, l := range ica.intelligenceLayers {
		if l.Enabled {
			count++
		}
	}
	return count
}

// calculateIntelligenceScore is placeholder for intelligence scoring
func (ica *IntelligenceCodingAgentImpl) calculateIntelligenceScore(response *Response) float64 {
	return 0.85
}

// ------------------------------------------------------------------
// Small helper / placeholder implementations (mocks)
// ------------------------------------------------------------------

// NewIntelligenceCodingAgentPatternDatabase returns a simple empty database object.
func NewIntelligenceCodingAgentPatternDatabase() *IntelligenceCodingAgentPatternDatabase {
	return &IntelligenceCodingAgentPatternDatabase{
		Patterns:    map[string]IntelligenceCodingAgentPattern{},
		Categories:  []string{},
		LastUpdated: time.Now(),
	}
}

// MockProcessor is a trivial processor used by default; replace with real implementations.
type MockProcessor struct{}

func NewMockProcessor() IntelligenceProcessor { return &MockProcessor{} }

func (mp *MockProcessor) Process(ctx context.Context, code string, ctxObj *IntelligenceCodingAgentDeepAnalysisContext) (*LayerResult, error) {
	// trivial analysis: split code into words and return first few as findings
	findings := []string{}
	words := strings.Fields(code)
	for i, w := range words {
		if i >= 5 {
			break
		}
		findings = append(findings, w)
	}
	return &LayerResult{
		Name:       "mock",
		Findings:   findings,
		Metrics:    map[string]float64{"len": float64(len(code))},
		Confidence: 0.6,
	}, nil
}

func (mp *MockProcessor) GetCapabilities() []string                     { return []string{"mock"} }
func (mp *MockProcessor) Configure(config map[string]interface{}) error { return nil }

// determineProcessingStrategy chooses processing strategy — placeholder.
func (ica *IntelligenceCodingAgentImpl) determineProcessingStrategy(intent *IntelligenceCodingAgentIntent, ctx *IntelligenceCodingAgentDeepAnalysisContext) *IntelligenceCodingAgentProcessingStrategy {
	return &IntelligenceCodingAgentProcessingStrategy{
		Name: "intelligent_generation",
		Type: StrategyIntelligentGeneration,
	}
}

// ------------------------------------------------------------------
// Placeholder implementations for missing methods
// ------------------------------------------------------------------

func (ica *IntelligenceCodingAgentImpl) performIntelligentOptimization(ctx context.Context, intent *IntelligenceCodingAgentIntent, deepContext *IntelligenceCodingAgentDeepAnalysisContext, query *Query) (*Response, error) {
	return &Response{
		ID:        fmt.Sprintf("optimization_response_%d", time.Now().UnixNano()),
		QueryID:   query.ID,
		Type:      "optimization",
		Content:   ResponseContent{Text: "Code optimization completed"},
		AgentUsed: "intelligence_coding_agent",
		Timestamp: time.Now(),
	}, nil
}

func (ica *IntelligenceCodingAgentImpl) performIntelligentRefactoring(ctx context.Context, intent *IntelligenceCodingAgentIntent, deepContext *IntelligenceCodingAgentDeepAnalysisContext, query *Query) (*Response, error) {
	return &Response{
		ID:        fmt.Sprintf("refactoring_response_%d", time.Now().UnixNano()),
		QueryID:   query.ID,
		Type:      "refactoring",
		Content:   ResponseContent{Text: "Code refactoring completed"},
		AgentUsed: "intelligence_coding_agent",
		Timestamp: time.Now(),
	}, nil
}

func (ica *IntelligenceCodingAgentImpl) performArchitecturalDesign(ctx context.Context, intent *IntelligenceCodingAgentIntent, deepContext *IntelligenceCodingAgentDeepAnalysisContext, query *Query) (*Response, error) {
	return &Response{
		ID:        fmt.Sprintf("architectural_response_%d", time.Now().UnixNano()),
		QueryID:   query.ID,
		Type:      "architectural_design",
		Content:   ResponseContent{Text: "Architectural design completed"},
		AgentUsed: "intelligence_coding_agent",
		Timestamp: time.Now(),
	}, nil
}

func (ica *IntelligenceCodingAgentImpl) performHybridProcessing(ctx context.Context, intent *IntelligenceCodingAgentIntent, deepContext *IntelligenceCodingAgentDeepAnalysisContext, query *Query) (*Response, error) {
	return &Response{
		ID:        fmt.Sprintf("hybrid_response_%d", time.Now().UnixNano()),
		QueryID:   query.ID,
		Type:      "hybrid",
		Content:   ResponseContent{Text: "Hybrid processing completed"},
		AgentUsed: "intelligence_coding_agent",
		Timestamp: time.Now(),
	}, nil
}

func (ica *IntelligenceCodingAgentImpl) enhanceResponseWithIntelligence(response *Response, intent *IntelligenceCodingAgentIntent, deepContext *IntelligenceCodingAgentDeepAnalysisContext, strategy *IntelligenceCodingAgentProcessingStrategy) *Response {
	if response == nil {
		response = &Response{}
	}
	// Enhance response with intelligence metadata
	response.Metadata.Confidence = 0.85
	return response
}

func (ica *IntelligenceCodingAgentImpl) buildIntelligentPrompts(intent *IntelligenceCodingAgentIntent, deepContext *IntelligenceCodingAgentDeepAnalysisContext, query *Query) *IntelligenceCodingAgentGenerationPrompts {
	return &IntelligenceCodingAgentGenerationPrompts{
		SystemPrompt: "You are an intelligent code generation assistant",
		UserPrompt:   query.UserInput,
		Temperature:  0.3,
		MaxTokens:    4000,
	}
}

func (ica *IntelligenceCodingAgentImpl) generateWithIntelligence(ctx context.Context, prompts *IntelligenceCodingAgentGenerationPrompts, intent *IntelligenceCodingAgentIntent, deepContext *IntelligenceCodingAgentDeepAnalysisContext) (*CodeResponse, *TokenUsage, error) {
	
	// Create intelligent prompt for LLM
	systemPrompt := "You are an expert software engineer with deep knowledge of code architecture, patterns, and best practices. Provide intelligent, well-structured code solutions with detailed explanations."
	
	userPrompt := fmt.Sprintf(`
Task: %s
Language: %s
Context: %s
Requirements:
- Provide clean, well-documented code
- Include error handling where appropriate
- Follow best practices for %s
- Explain the solution approach

Please generate the requested code with a comprehensive explanation.
`, intent.Type, deepContext.Language, deepContext.ProjectContext, deepContext.Language)

	// Create LLM request
	llmRequest := &llm.GenerationRequest{
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Model:       "gpt-3.5-turbo",
		Temperature: 0.3,
		MaxTokens:   1000,
	}

	// Call LLM
	llmResponse, err := ica.dependencies.LLMManager.Generate(ctx, llmRequest)
	if err != nil {
		return &CodeResponse{
			Code:        "// Error generating code",
			Language:    deepContext.Language,
			Explanation: fmt.Sprintf("LLM generation failed: %v", err),
		}, &TokenUsage{InputTokens: 0, OutputTokens: 0, TotalTokens: 0}, err
	}

	// Parse response to extract code and explanation
	content := llmResponse.Content
	var code, explanation string
	
	// Simple parsing - look for code blocks
	if strings.Contains(content, "```") {
		parts := strings.Split(content, "```")
		if len(parts) >= 3 {
			code = strings.TrimSpace(parts[1])
			// Remove language identifier if present
			if lines := strings.Split(code, "\n"); len(lines) > 0 {
				if strings.Contains(lines[0], deepContext.Language) {
					code = strings.Join(lines[1:], "\n")
				}
			}
			explanation = strings.TrimSpace(parts[0] + parts[2])
		} else {
			explanation = content
			code = "// Code extraction failed"
		}
	} else {
		explanation = content
		code = "// No code block found in response"
	}

	return &CodeResponse{
		Code:        code,
		Language:    deepContext.Language,
		Explanation: explanation,
	}, &TokenUsage{
		InputTokens:  llmResponse.TokenUsage.InputTokens,
		OutputTokens: llmResponse.TokenUsage.OutputTokens,
		TotalTokens:  llmResponse.TokenUsage.TotalTokens,
	}, nil
}

func (ica *IntelligenceCodingAgentImpl) postProcessWithIntelligence(ctx context.Context, response *CodeResponse, intent *IntelligenceCodingAgentIntent, deepContext *IntelligenceCodingAgentDeepAnalysisContext) (*CodeResponse, error) {
	return response, nil
}

func (ica *IntelligenceCodingAgentImpl) countCrossFileReferences(code string, language string) int {
	count := 0
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "import") || strings.Contains(line, "require") || strings.Contains(line, "include") {
			count++
		}
	}
	return count
}

func (ica *IntelligenceCodingAgentImpl) getArchitecturalDepth(code string, language string) int {
	depth := 1
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "class") || strings.Contains(line, "interface") || strings.Contains(line, "struct") {
			depth++
		}
		if strings.Contains(line, "func") || strings.Contains(line, "function") || strings.Contains(line, "method") {
			depth++
		}
	}
	return depth
}

func (ica *IntelligenceCodingAgentImpl) performAdvancedValidation(ctx context.Context, response *CodeResponse, intent *IntelligenceCodingAgentIntent, deepContext *IntelligenceCodingAgentDeepAnalysisContext) (*CodeValidation, error) {
	return &CodeValidation{IsValid: true}, nil
}

func (ica *IntelligenceCodingAgentImpl) buildIntelligentResponse(query *Query, intent *IntelligenceCodingAgentIntent, deepContext *IntelligenceCodingAgentDeepAnalysisContext, codeResponse *CodeResponse, tokenUsage *TokenUsage) *Response {
	return &Response{
		ID:      fmt.Sprintf("intelligence_response_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    "code_generation",
		Content: ResponseContent{
			Text: "Generated with intelligence processing",
			Code: codeResponse,
		},
		AgentUsed:  "intelligence_coding_agent",
		Provider:   "multi_llm",
		TokenUsage: *tokenUsage,
		Cost:       Cost{TotalCost: 0.0, Currency: "USD"},
		Metadata: ResponseMetadata{
			GenerationTime: time.Second,
			Confidence:     0.9,
		},
		Timestamp: time.Now(),
	}
}

func (ica *IntelligenceCodingAgentImpl) extractCodeFromQuery(query *Query) string {
	return query.UserInput
}

func (ica *IntelligenceCodingAgentImpl) buildAnalysisResponse(query *Query, intent *IntelligenceCodingAgentIntent, deepContext *IntelligenceCodingAgentDeepAnalysisContext, result *IntelligenceCodingAgentDeepAnalysisResult) *Response {
	return &Response{
		ID:      fmt.Sprintf("analysis_response_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    "analysis",
		Content: ResponseContent{
			Text: fmt.Sprintf("Deep analysis completed. Confidence: %.2f, Processing time: %v",
				result.Confidence, result.ProcessingTime),
		},
		AgentUsed: "intelligence_coding_agent",
		Metadata: ResponseMetadata{
			Confidence:     result.Confidence,
			GenerationTime: result.ProcessingTime,
		},
		Timestamp: time.Now(),
	}
}

func (ica *IntelligenceCodingAgentImpl) generateCacheKey(request *IntelligenceCodingAgentDeepAnalysisRequest) string {
	hash := md5.Sum([]byte(request.Code + request.Language + request.AnalysisType))
	return fmt.Sprintf("analysis_%x", hash)
}

// ------------------------------------------------------------------
// Small helper detection / heuristics (placeholders)
// ------------------------------------------------------------------
func (ica *IntelligenceCodingAgentImpl) requiresIntelligentProcessing(intent *IntelligenceCodingAgentIntent) bool {
	return intent.ComplexityLevel >= 5 || intent.RequiresAnalysis || intent.ArchitecturalScope
}

func (ica *IntelligenceCodingAgentImpl) classifyInputType(input string) string {
	if strings.Contains(input, "analysis") || strings.Contains(input, "analyze") {
		return "analysis"
	}
	if strings.Contains(input, "generate") || strings.Contains(input, "create") {
		return "generation"
	}
	return "general"
}

func (ica *IntelligenceCodingAgentImpl) estimateComplexity(input string) int {
	complexity := 3
	indicators := []string{"architecture", "design", "pattern", "optimize", "refactor", "security", "performance"}
	for _, indicator := range indicators {
		if strings.Contains(strings.ToLower(input), indicator) {
			complexity += 2
		}
	}
	if complexity > 10 {
		complexity = 10
	}
	return complexity
}

func (ica *IntelligenceCodingAgentImpl) classifyDeepIntentType(input string) string {
	if strings.Contains(input, "analyze") || strings.Contains(input, "analysis") {
		return string(IntelligenceCodingAgentDeepIntentAnalysis)
	}
	if strings.Contains(input, "generate") || strings.Contains(input, "create") {
		return string(IntelligenceCodingAgentDeepIntentGeneration)
	}
	if strings.Contains(input, "optimize") || strings.Contains(input, "performance") {
		return string(IntelligenceCodingAgentDeepIntentOptimization)
	}
	if strings.Contains(input, "refactor") || strings.Contains(input, "improve") {
		return string(IntelligenceCodingAgentDeepIntentRefactoring)
	}
	return string(IntelligenceCodingAgentDeepIntentGeneration)
}

func (ica *IntelligenceCodingAgentImpl) requiresAnalysis(input string) bool {
	return strings.Contains(input, "analyz") || strings.Contains(input, "understand") || strings.Contains(input, "explain")
}

func (ica *IntelligenceCodingAgentImpl) requiresGeneration(input string) bool {
	return strings.Contains(input, "generat") || strings.Contains(input, "creat") || strings.Contains(input, "build")
}

func (ica *IntelligenceCodingAgentImpl) requiresOptimization(input string) bool {
	return strings.Contains(input, "optim") || strings.Contains(input, "perform") || strings.Contains(input, "faster")
}

func (ica *IntelligenceCodingAgentImpl) isDomainSpecific(input string) bool {
	domains := []string{"web", "api", "database", "security", "ml", "ai"}
	for _, domain := range domains {
		if strings.Contains(input, domain) {
			return true
		}
	}
	return false
}

func (ica *IntelligenceCodingAgentImpl) requiresCrossFileAnalysis(input string) bool {
	return strings.Contains(input, "project") || strings.Contains(input, "system") || strings.Contains(input, "architecture")
}

func (ica *IntelligenceCodingAgentImpl) requiresArchitecturalAnalysis(input string) bool {
	return strings.Contains(input, "architecture") || strings.Contains(input, "design") || strings.Contains(input, "pattern")
}

func (ica *IntelligenceCodingAgentImpl) determineGenerationType(input string) string {
	if strings.Contains(input, "function") || strings.Contains(input, "method") {
		return "function"
	}
	return "general"
}

func (ica *IntelligenceCodingAgentImpl) determineAnalysisType(input string) string {
	if strings.Contains(input, "performance") {
		return "performance"
	}
	return "general"
}

func (ica *IntelligenceCodingAgentImpl) determineOptimizationTarget(input string) string {
	if strings.Contains(input, "performance") {
		return "performance"
	}
	return "readability"
}

func (ica *IntelligenceCodingAgentImpl) determineQualityFocus(input string) []string {
	var focus []string
	if strings.Contains(input, "test") {
		focus = append(focus, "testing")
	}
	if strings.Contains(input, "document") {
		focus = append(focus, "documentation")
	}
	if len(focus) == 0 {
		focus = append(focus, "maintainability")
	}
	return focus
}

func (ica *IntelligenceCodingAgentImpl) detectIntelligentPatterns(ctx context.Context, input string, deep *IntelligenceCodingAgentDeepAnalysisContext) ([]string, error) {
	// Simple pattern detection based on keywords
	patterns := []string{}
	if strings.Contains(strings.ToLower(input), "singleton") {
		patterns = append(patterns, "singleton_pattern")
	}
	if strings.Contains(strings.ToLower(input), "factory") {
		patterns = append(patterns, "factory_pattern")
	}
	return patterns, nil
}

// createFallbackResponse creates a fallback response when LLM is not available
func (ica *IntelligenceCodingAgentImpl) createFallbackResponse(query *Query, reason string) *Response {
	return &Response{
		ID:      fmt.Sprintf("intelligence_fallback_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    "code_generation",
		Content: ResponseContent{
			Text: fmt.Sprintf("Intelligence coding request: '%s'\n\nStatus: %s\n\nTo enable intelligent code generation:\n1. ✅ MCP Context (Available)\n2. ❌ LLM Manager (Connect OpenAI/Gemini)\n3. ✅ Intelligence Layers (Ready)", query.UserInput, reason),
		},
		AgentUsed:  "intelligence_coding_agent",
		Provider:   "none",
		TokenUsage: TokenUsage{InputTokens: 0, OutputTokens: 0, TotalTokens: 0},
		Cost:       Cost{TotalCost: 0.0, Currency: "USD"},
		Metadata: ResponseMetadata{
			Confidence:     0.6,
			GenerationTime: time.Since(time.Now()),
		},
		Timestamp: time.Now(),
	}
}
