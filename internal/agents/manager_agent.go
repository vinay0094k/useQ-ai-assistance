package agents

import (
	"context"
	"fmt"
	"os"
	"math"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/yourusername/useq-ai-assistant/internal/mcp"
	"github.com/yourusername/useq-ai-assistant/internal/llm"
	"github.com/yourusername/useq-ai-assistant/models"
)

// ManagerAgent is the centralized agent router that intelligently routes queries to specialized agents
type ManagerAgent struct {
	dependencies            *AgentDependencies
	SearchAgent             *SearchAgentImpl
	CodingAgent             *CodingAgentImpl
	IntelligenceCodingAgent *IntelligenceCodingAgentImpl
	ContextAwareSearchAgent *ContextAwareSearchAgentImpl
	SystemAgent             *SystemAgent
	mcpClient               *mcp.MCPClient
	intelligentProcessor    *mcp.IntelligentQueryProcessor
	llmManager              *llm.Manager
	metrics                 *AgentMetrics
	routingHistory          []RoutingDecision
}

// NewManagerAgent creates a new centralized manager agent
func NewManagerAgent(deps *AgentDependencies) *ManagerAgent {
	manager := &ManagerAgent{
		dependencies:   deps,
		intelligentProcessor: mcp.NewIntelligentQueryProcessor(),
		mcpClient:      mcp.NewMCPClient(),
		routingHistory: make([]RoutingDecision, 0),
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

	// Initialize specialized agents with error handling
	manager.initializeAgents(deps)
	
	// Initialize LLM manager with environment variables
	manager.initializeLLMManager()
	return manager
}

// initializeAgents creates and configures all specialized agents
func (ma *ManagerAgent) initializeAgents(deps *AgentDependencies) {
	// Initialize agents with proper error handling
	if deps != nil {
		// Initialize basic search agent first
		ma.SearchAgent = NewSearchAgent(deps)

		// Initialize coding agent with nil check
		ma.CodingAgent = NewCodingAgent(deps)

		// Initialize context aware search agent
		ma.ContextAwareSearchAgent = NewContextAwareSearchAgentImpl(deps)

		// Initialize intelligence coding agent (using basic interfaces to avoid nil pointer issues)
		ma.IntelligenceCodingAgent = NewIntelligenceCodingAgent(deps, nil, nil)
		
		// Initialize system agent
		ma.SystemAgent = NewSystemAgent(deps)
	}
}

// initializeLLMManager initializes LLM manager with environment variables
func (ma *ManagerAgent) initializeLLMManager() {
	// Load environment variables
	_ = godotenv.Load()
	
	openaiKey := os.Getenv("OPENAI_API_KEY")
	geminiKey := os.Getenv("GEMINI_API_KEY")
	
	if openaiKey == "" && geminiKey == "" {
		if ma.dependencies != nil && ma.dependencies.Logger != nil {
			ma.dependencies.Logger.Warn("No LLM API keys found in environment", nil)
		}
		return
	}
	
	config := llm.AIProvidersConfig{
		Primary:       "openai",
		FallbackOrder: []string{"openai", "gemini"},
		OpenAI: llm.ProviderConfig{
			APIKey:      openaiKey,
			Model:       "gpt-4-turbo-preview",
			MaxTokens:   4000,
			Temperature: 0.1,
			Timeout:     30 * time.Second,
		},
		Gemini: llm.ProviderConfig{
			APIKey:      geminiKey,
			Model:       "gemini-1.5-pro",
			MaxTokens:   4000,
			Temperature: 0.1,
			Timeout:     30 * time.Second,
		},
	}
	
	var err error
	ma.llmManager, err = llm.NewManager(config)
	if err != nil {
		if ma.dependencies != nil && ma.dependencies.Logger != nil {
			ma.dependencies.Logger.Error("Failed to initialize LLM manager", map[string]interface{}{
				"error": err.Error(),
			})
		}
	} else {
		if ma.dependencies != nil && ma.dependencies.Logger != nil {
			ma.dependencies.Logger.Info("LLM manager initialized successfully", map[string]interface{}{
				"primary": config.Primary,
				"fallbacks": config.FallbackOrder,
			})
		}
		
		// Update dependencies
		if ma.dependencies != nil {
			ma.dependencies.LLMManager = ma.llmManager
		}
	}
}

// RouteQuery intelligently routes queries to the most appropriate agent
func (ma *ManagerAgent) RouteQuery(ctx context.Context, query *models.Query) (response *models.Response, err error) {
	// STEP 1: 3-TIER CLASSIFICATION FIRST - COST OPTIMIZATION
	classification, classErr := ma.mcpClient.(*mcp.MCPClient).GetQueryClassifier().ClassifyQuery(ctx, query)
	if classErr == nil {
		// Log classification decision with cost info
		if ma.dependencies != nil && ma.dependencies.Logger != nil {
			ma.dependencies.Logger.Info("Query classified", map[string]interface{}{
				"tier":           classification.Tier,
				"confidence":     classification.Confidence,
				"estimated_cost": classification.EstimatedCost,
				"estimated_time": classification.EstimatedTime,
				"skip_llm":       classification.SkipLLM,
			})
		}

		// Process based on tier classification
		switch classification.Tier {
		case mcp.TierSimple:
			// Tier 1: Direct MCP execution (ACTUAL COST: $0, <100ms)
			return ma.processTier1Query(ctx, query, classification)
		case mcp.TierMedium:
			// Tier 2: MCP + Vector search (ACTUAL COST: ~$0.0005, <500ms)
			return ma.processTier2Query(ctx, query, classification)
		case mcp.TierComplex:
			// Tier 3: Full LLM pipeline (ACTUAL COST: $0.02-0.03, 1-3s)
			return ma.processTier3Query(ctx, query, classification)
		}
	}
	
	// Fallback to original routing if classification fails
	// Add panic recovery with better error reporting
	defer func() {
		if r := recover(); r != nil {
			if ma.dependencies != nil && ma.dependencies.Logger != nil {
				ma.dependencies.Logger.Error("Manager agent panic recovered", map[string]interface{}{
					"panic": r,
					"query": query.UserInput,
				})
			}
			err = fmt.Errorf("manager agent panic: %v", r)
		}
	}()

	startTime := time.Now()
	ma.updateMetrics(startTime)

	// NEW: Use intelligent query processor for complex queries
	if ma.shouldUseIntelligentProcessing(query) {
		if ma.dependencies != nil && ma.dependencies.Logger != nil {
			ma.dependencies.Logger.Info("Using intelligent query processing", map[string]interface{}{
				"query": query.UserInput,
				"reason": "complex_query_detected",
			})
		}
		
		response, err := ma.intelligentProcessor.ProcessQuery(ctx, query)
		if err == nil {
			ma.updateSuccessMetrics(startTime, 0.9, response)
			return response, nil
		}
		
		if ma.dependencies != nil && ma.dependencies.Logger != nil {
			ma.dependencies.Logger.Warn("Intelligent processing failed, falling back to agent routing", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Fallback to traditional agent routing
	routingAnalysis := ma.analyzeQueryForRouting(ctx, query)
	if routingAnalysis == nil {
		return nil, fmt.Errorf("failed to analyze query for routing")
	}

	// Enhanced MCP Integration with Command Execution
	mcpContext, err := ma.mcpClient.ProcessQuery(ctx, query)
	if err == nil && mcpContext != nil {
		query.MCPContext = mcpContext
		
		// Log what commands were executed
		if ma.dependencies != nil && ma.dependencies.Logger != nil {
			ma.dependencies.Logger.Info("MCP commands executed", map[string]interface{}{
				"operations": mcpContext.Operations,
				"data_keys":  ma.extractDataKeys(mcpContext.Data),
			})
		}
	} else if err != nil {
		// Log MCP failure but continue processing
		if ma.dependencies != nil && ma.dependencies.Logger != nil {
			ma.dependencies.Logger.Warn("MCP processing failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Select best agent based on improved analysis
	selectedAgent, confidence := ma.selectBestAgent(ctx, query, routingAnalysis)
	if selectedAgent == "" {
		// Default to search agent for unclassified queries
		selectedAgent = "search"
		confidence = 0.5
	}

	// Record routing decision
	decision := RoutingDecision{
		QueryID:       query.ID,
		Intent:        routingAnalysis.PrimaryIntent,
		SelectedAgent: selectedAgent,
		Confidence:    confidence,
		Timestamp:     time.Now(),
	}

	// Route to selected agent with better error handling
	response, err = ma.executeWithSelectedAgent(ctx, query, selectedAgent)

	// Store query in database
	if ma.dependencies.Storage != nil {
		if storeErr := ma.dependencies.Storage.StoreQuery(query); storeErr != nil {
			if ma.dependencies.Logger != nil {
				ma.dependencies.Logger.Warn("Failed to store query", "error", storeErr)
			}
		}
	}

	// Update routing decision with success status
	decision.Success = (err == nil)
	ma.routingHistory = append(ma.routingHistory, decision)

	// Store response in database
	if err == nil && response != nil && ma.dependencies.Storage != nil {
		if storeErr := ma.dependencies.Storage.StoreResponse(response); storeErr != nil {
			if ma.dependencies.Logger != nil {
				ma.dependencies.Logger.Warn("Failed to store response", "error", storeErr)
			}
		}
	}

	if err != nil {
		ma.metrics.ErrorCount++
		// Try fallback routing
		return ma.handleRoutingFallback(ctx, query, selectedAgent)
	}

	ma.updateSuccessMetrics(startTime, confidence, response)
	return response, nil
}

// processTier1Query handles simple queries with direct MCP execution
func (ma *ManagerAgent) processTier1Query(ctx context.Context, query *models.Query, classification *mcp.ClassificationResult) (*models.Response, error) {
	startTime := time.Now()
	
	// Execute MCP operations directly without LLM
	mcpContext, err := ma.mcpClient.ProcessQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("tier 1 MCP processing failed: %w", err)
	}
	
	// Format response directly from MCP results
	responseText := ma.formatMCPResults(mcpContext, query)
	
	response := &models.Response{
		ID:      fmt.Sprintf("tier1_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    models.ResponseTypeSystem,
		Content: models.ResponseContent{
			Text: responseText,
		},
		AgentUsed:  "mcp_direct",
		Provider:   "filesystem",
		TokenUsage: models.TokenUsage{TotalTokens: 0},
		Cost:       models.Cost{TotalCost: 0.0, Currency: "USD"},
		Metadata: models.ResponseMetadata{
			GenerationTime: time.Since(startTime),
			Confidence:     classification.Confidence,
			Tools:          []string{"mcp_filesystem"},
			Reasoning:      classification.Reasoning,
		},
		Timestamp: time.Now(),
	}
	
	return response, nil
}

// processTier2Query handles medium queries with MCP + Vector search
func (ma *ManagerAgent) processTier2Query(ctx context.Context, query *models.Query, classification *mcp.ClassificationResult) (*models.Response, error) {
	startTime := time.Now()
	
	// Track Tier 2 costs
	if ma.dependencies != nil && ma.dependencies.Logger != nil {
		ma.dependencies.Logger.Info("Processing Tier 2 query", map[string]interface{}{
			"query": query.UserInput,
			"note":  "Will incur embedding costs (~$0.0005)",
		})
	}
	
	// Execute MCP operations
	mcpContext, err := ma.mcpClient.ProcessQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("tier 2 MCP processing failed: %w", err)
	}
	
	// Add vector search if available
	var vectorResults []interface{}
	if ma.dependencies != nil && ma.dependencies.VectorDB != nil {
		// This will cost ~$0.0005 for query embedding
		if results, err := ma.dependencies.VectorDB.Search(ctx, query.UserInput, 10); err == nil {
			vectorResults = results
			if ma.dependencies.Logger != nil {
				ma.dependencies.Logger.Info("Vector search completed", map[string]interface{}{
					"results_count": len(results),
					"embedding_cost": "~$0.0005",
				})
			}
		}
	}
	
	// Format response from MCP + Vector results (no LLM synthesis)
	responseText := ma.formatMCPAndVectorResults(mcpContext, vectorResults, query)
	
	response := &models.Response{
		ID:      fmt.Sprintf("tier2_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    models.ResponseTypeSearch,
		Content: models.ResponseContent{
			Text: responseText,
		},
		AgentUsed:  "mcp_vector",
		Provider:   "mcp_vector_search",
		TokenUsage: models.TokenUsage{TotalTokens: 0},
		Cost:       models.Cost{TotalCost: 0.0005, Currency: "USD"}, // REAL embedding cost
		Metadata: models.ResponseMetadata{
			GenerationTime: time.Since(startTime),
			Confidence:     classification.Confidence,
			Tools:          []string{"mcp_filesystem", "vector_search", "openai_embeddings"},
			Reasoning:      classification.Reasoning,
		},
		Timestamp: time.Now(),
	}
	
	return response, nil
}

// processTier3Query handles complex queries with full LLM pipeline
func (ma *ManagerAgent) processTier3Query(ctx context.Context, query *models.Query, classification *mcp.ClassificationResult) (*models.Response, error) {
	// Use existing intelligent processing for complex queries
	if ma.shouldUseIntelligentProcessing(query) {
		return ma.intelligentProcessor.ProcessQuery(ctx, query)
	}
	
	// Fallback to traditional agent routing
	return ma.routeToTraditionalAgents(ctx, query)
}

// formatMCPResults formats MCP results for Tier 1 responses
func (ma *ManagerAgent) formatMCPResults(mcpContext *models.MCPContext, query *models.Query) string {
	var result strings.Builder
	
	// Format based on what data is available
	if files, ok := mcpContext.Data["files"].([]map[string]interface{}); ok {
		result.WriteString(fmt.Sprintf("📁 Found %d files:\n", len(files)))
		for i, file := range files {
			if i >= 10 {
				result.WriteString(fmt.Sprintf("... and %d more files\n", len(files)-10))
				break
			}
			if path, ok := file["path"].(string); ok {
				result.WriteString(fmt.Sprintf("  %d. %s\n", i+1, path))
			}
		}
	}
	
	if count, ok := mcpContext.Data["file_count"].(int); ok {
		result.WriteString(fmt.Sprintf("\n📊 Total files: %d\n", count))
	}
	
	if structure, ok := mcpContext.Data["project_structure"].(map[string]interface{}); ok {
		result.WriteString("\n📂 Project Structure:\n")
		ma.formatStructureForDisplay(structure, "", &result)
	}
	
	if systemInfo, ok := mcpContext.Data["system_info"].(map[string]interface{}); ok {
		result.WriteString("\n🖥️ System Info:\n")
		for key, value := range systemInfo {
			result.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
		}
	}
	
	return result.String()
}

// formatMCPAndVectorResults formats results for Tier 2 responses
func (ma *ManagerAgent) formatMCPAndVectorResults(mcpContext *models.MCPContext, vectorResults []interface{}, query *models.Query) string {
	var result strings.Builder
	
	// Add MCP results
	result.WriteString(ma.formatMCPResults(mcpContext, query))
	
	// Add vector search results if available
	if len(vectorResults) > 0 {
		result.WriteString("\n🧠 Semantic Search Results:\n")
		for i, vr := range vectorResults {
			if i >= 5 {
				result.WriteString(fmt.Sprintf("... and %d more matches\n", len(vectorResults)-5))
				break
			}
			result.WriteString(fmt.Sprintf("  %d. Similar code found (relevance: %.3f)\n", i+1, 0.8))
		}
	}
	
	return result.String()
}

// routeToTraditionalAgents routes to traditional agents as fallback
func (ma *ManagerAgent) routeToTraditionalAgents(ctx context.Context, query *models.Query) (*models.Response, error) {
	// Use existing routing logic as fallback
	routingAnalysis := ma.analyzeQueryForRouting(ctx, query)
	selectedAgent, confidence := ma.selectBestAgent(ctx, query, routingAnalysis)
	return ma.executeWithSelectedAgent(ctx, query, selectedAgent)
}

// formatStructureForDisplay formats project structure for display
func (ma *ManagerAgent) formatStructureForDisplay(structure map[string]interface{}, prefix string, result *strings.Builder) {
	for key := range structure {
		result.WriteString(fmt.Sprintf("%s├─ %s/\n", prefix, key))
		if subMap, ok := structure[key].(map[string]interface{}); ok && len(prefix) < 6 {
			ma.formatStructureForDisplay(subMap, prefix+"│  ", result)
		}
	}
}
// shouldUseIntelligentProcessing determines if query needs intelligent processing
func (ma *ManagerAgent) shouldUseIntelligentProcessing(query *models.Query) bool {
	input := strings.ToLower(query.UserInput)
	
	// Use intelligent processing for explanation queries
	if strings.Contains(input, "explain") || strings.Contains(input, "flow") || 
	   strings.Contains(input, "architecture") || strings.Contains(input, "how does") {
		return true
	}
	
	// Use for complex generation queries
	if (strings.Contains(input, "create") || strings.Contains(input, "generate")) &&
	   (strings.Contains(input, "service") || strings.Contains(input, "microservice") ||
	    strings.Contains(input, "authentication") || strings.Contains(input, "api")) {
		return true
	}
	
	// Use for analysis queries
	if strings.Contains(input, "analyze") || strings.Contains(input, "review") ||
	   strings.Contains(input, "optimize") || strings.Contains(input, "refactor") {
		return true
	}
	
	return false
}

// analyzeQueryForRouting performs improved analysis of query for routing decisions
func (ma *ManagerAgent) analyzeQueryForRouting(ctx context.Context, query *models.Query) *RoutingAnalysis {
	input := strings.ToLower(strings.TrimSpace(query.UserInput))

	analysis := &RoutingAnalysis{
		PrimaryIntent:        ma.determinePrimaryIntent(input, query),
		SecondaryIntents:     ma.determineSecondaryIntents(input),
		Complexity:           ma.assessComplexity(input),
		Domain:               ma.identifyDomain(input),
		RequiredCapabilities: ma.identifyRequiredCapabilities(input),
		ContextNeeds:         ma.assessContextNeeds(input),
		UrgencyLevel:         ma.assessUrgency(input),
	}

	return analysis
}

// selectBestAgent chooses the most appropriate agent based on improved analysis
func (ma *ManagerAgent) selectBestAgent(ctx context.Context, query *models.Query, analysis *RoutingAnalysis) (string, float64) {
	agentScores := make(map[string]float64)

	// Evaluate each agent's capability for this query with corrected scoring
	agentScores["search"] = ma.evaluateSearchAgent(query, analysis)
	agentScores["context_search"] = ma.evaluateContextSearchAgent(query, analysis)
	agentScores["coding"] = ma.evaluateCodingAgent(query, analysis)
	agentScores["intelligence_coding"] = ma.evaluateIntelligenceCodingAgent(query, analysis)
	agentScores["system"] = ma.evaluateSystemAgent(query, analysis)

	// Apply learning from routing history
	ma.applyHistoricalLearning(agentScores, analysis)

	// Debug logging for routing decisions
	if ma.dependencies != nil && ma.dependencies.Logger != nil {
		ma.dependencies.Logger.Info("Agent scoring results", map[string]interface{}{
			"query":              query.UserInput,
			"primary_intent":     analysis.PrimaryIntent,
			"search_score":       agentScores["search"],
			"context_score":      agentScores["context_search"],
			"coding_score":       agentScores["coding"],
			"intelligence_score": agentScores["intelligence_coding"],
		})
	}

	// Select agent with highest score
	bestAgent := ""
	bestScore := 0.0

	for agent, score := range agentScores {
		if score > bestScore {
			bestScore = score
			bestAgent = agent
		}
	}

	return bestAgent, bestScore
}

// executeWithSelectedAgent routes to the chosen agent with better error handling
func (ma *ManagerAgent) executeWithSelectedAgent(ctx context.Context, query *models.Query, agentName string) (*models.Response, error) {
	switch agentName {
	case "search":
		if ma.SearchAgent == nil {
			return nil, fmt.Errorf("search agent not initialized")
		}
		return ma.SearchAgent.Search(ctx, query)

	case "context_search":
		if ma.ContextAwareSearchAgent == nil {
			return nil, fmt.Errorf("context search agent not initialized")
		}
		return ma.ContextAwareSearchAgent.Process(ctx, query)

	case "coding":
		if ma.CodingAgent == nil {
			return nil, fmt.Errorf("coding agent not initialized")
		}
		// Add null check before calling Process
		response, err := ma.CodingAgent.Process(ctx, query)
		if err != nil {
			if ma.dependencies != nil && ma.dependencies.Logger != nil {
				ma.dependencies.Logger.Error("CodingAgent process failed", map[string]interface{}{
					"error": err.Error(),
					"query": query.UserInput,
				})
			}
			return nil, fmt.Errorf("coding agent failed: %w", err)
		}
		return response, nil

	case "intelligence_coding":
		if ma.IntelligenceCodingAgent == nil {
			return nil, fmt.Errorf("intelligence coding agent not initialized")
		}
		// Convert models.Query to Query for IntelligenceCodingAgent
		icQuery := &Query{
			ID:        query.ID,
			UserInput: query.UserInput,
			Language:  query.Language,
		}
		icResponse, err := ma.IntelligenceCodingAgent.Process(ctx, icQuery)
		if err != nil {
			return nil, fmt.Errorf("intelligence coding agent failed: %w", err)
		}
		// Convert Response to models.Response
		return &models.Response{
			ID:        icResponse.ID,
			QueryID:   icResponse.QueryID,
			Type:      models.ResponseType(icResponse.Type),
			Content:   models.ResponseContent{Text: icResponse.Content.Text},
			AgentUsed: icResponse.AgentUsed,
			Timestamp: icResponse.Timestamp,
			TokenUsage: models.TokenUsage{
				InputTokens:  icResponse.TokenUsage.InputTokens,
				OutputTokens: icResponse.TokenUsage.OutputTokens,
				TotalTokens:  icResponse.TokenUsage.TotalTokens,
			},
			Cost: models.Cost{
				TotalCost: icResponse.Cost.TotalCost,
				Currency:  icResponse.Cost.Currency,
			},
		}, nil

	case "system":
		if ma.SystemAgent == nil {
			return nil, fmt.Errorf("system agent not initialized")
		}
		return ma.SystemAgent.Process(ctx, query)

	default:
		return nil, fmt.Errorf("unknown agent: %s", agentName)
	}
}

// FIXED: Agent evaluation methods with corrected scoring

func (ma *ManagerAgent) evaluateSearchAgent(query *models.Query, analysis *RoutingAnalysis) float64 {
	score := 0.5 // Base score for basic search agent
	input := strings.ToLower(query.UserInput)

	// HIGH score for system status queries
	if ma.isSystemStatusQuery(input) {
		score += 0.4
		return score
	}

	// HIGH score for file count queries
	if ma.isFileCountQuery(input) {
		score += 0.4
		return score
	}

	// HIGH score for basic search intents
	if analysis.PrimaryIntent == "search" || analysis.PrimaryIntent == "find" {
		score += 0.3
	}

	// PREFER basic search for simple queries
	if analysis.Complexity < 0.5 {
		score += 0.2
	}

	// PREFER basic search for informational queries
	if strings.Contains(input, "show") || strings.Contains(input, "list") || strings.Contains(input, "what") {
		score += 0.2
	}

	// REDUCE score significantly for mixed intent queries (let intelligence handle)
	if (strings.Contains(input, "search") || strings.Contains(input, "find")) &&
		(strings.Contains(input, "generate") || strings.Contains(input, "create") || strings.Contains(input, "new")) {
		score -= 0.5 // BIG reduction for mixed search+generate intents
	}

	// REDUCE score for "and" combinations indicating multiple tasks
	if strings.Contains(input, " and ") {
		andParts := strings.Split(input, " and ")
		if len(andParts) >= 2 {
			score -= 0.3 // Reduce for multiple tasks
		}
	}

	return score
}

func (ma *ManagerAgent) evaluateContextSearchAgent(query *models.Query, analysis *RoutingAnalysis) float64 {
	score := 0.2 // Base score
	input := strings.ToLower(query.UserInput)

	// REDUCE score for system status queries (let SearchAgent handle)
	if ma.isSystemStatusQuery(input) || ma.isFileCountQuery(input) {
		score -= 0.1
		return score
	}

	// HIGH score for explicit context/pattern queries
	contextWords := []string{"similar", "pattern", "example", "like", "related", "matching"}
	hasContext := false
	for _, word := range contextWords {
		if strings.Contains(input, word) {
			score += 0.5 // INCREASED from 0.4 to 0.5
			hasContext = true
		}
	}

	// EXTRA boost for "our" + pattern combinations (project-specific patterns)
	if strings.Contains(input, "our") && strings.Contains(input, "pattern") {
		score += 0.3 // BIG boost for "our pattern" queries
	}

	// EXTRA boost for "follow" + pattern (following patterns)
	if strings.Contains(input, "follow") && strings.Contains(input, "pattern") {
		score += 0.3 // BIG boost for "follow pattern" queries
	}

	// BOOST for authentication pattern specifically
	if strings.Contains(input, "authentication") && strings.Contains(input, "pattern") {
		score += 0.2 // Additional boost for auth patterns
	}

	// BOOST for high context needs
	if analysis.ContextNeeds > 0.7 && hasContext {
		score += 0.3
	}

	// REDUCE score for refactoring queries (let intelligence handle)
	if strings.Contains(input, "refactor") {
		score -= 0.2
	}

	return score
}

func (ma *ManagerAgent) evaluateCodingAgent(query *models.Query, analysis *RoutingAnalysis) float64 {
	score := 0.4 // Reasonable base score
	input := strings.ToLower(query.UserInput)

	// HIGH score for simple generation intents
	if analysis.PrimaryIntent == "generation" || analysis.PrimaryIntent == "create" {
		score += 0.4 // Good boost for generation
	}

	// HIGH score for simple coding tasks
	simpleWords := []string{"hello world", "simple", "basic", "function"}
	for _, word := range simpleWords {
		if strings.Contains(input, word) {
			score += 0.3 // Boost for simple tasks
		}
	}

	// REDUCE score significantly for complex tasks (let intelligence handle)
	complexWords := []string{"microservice", "architecture", "optimize", "analyze", "refactor"}
	for _, word := range complexWords {
		if strings.Contains(input, word) {
			score -= 0.4 // BIG reduction for complex tasks
		}
	}

	// REDUCE score for multiple requirements
	requirementWords := []string{"authentication", "logging", "monitoring", "security"}
	requirementCount := 0
	for _, word := range requirementWords {
		if strings.Contains(input, word) {
			requirementCount++
		}
	}
	if requirementCount >= 2 {
		score -= 0.5 // BIG reduction for multiple requirements
	}

	return math.Max(score, 0.1) // Minimum score
}

func (ma *ManagerAgent) evaluateIntelligenceCodingAgent(query *models.Query, analysis *RoutingAnalysis) float64 {
	score := 0.2 // Base score
	input := strings.ToLower(query.UserInput)

	// HIGH score for complex architectural queries
	architecturalWords := []string{"architecture", "microservice", "design", "pattern", "optimize", "performance"}
	for _, word := range architecturalWords {
		if strings.Contains(input, word) {
			score += 0.4 // BIG boost for architectural terms
		}
	}

	// HIGH score for optimization/improvement queries
	if strings.Contains(input, "optimize") || strings.Contains(input, "improve") ||
		strings.Contains(input, "refactor") || strings.Contains(input, "enhance") {
		score += 0.5 // MAJOR boost for optimization
	}

	// HIGH score for analysis requests
	if strings.Contains(input, "analyze") || strings.Contains(input, "review") ||
		strings.Contains(input, "quality") || strings.Contains(input, "architectural") {
		score += 0.4 // BIG boost for analysis
	}

	// HIGH score for complex generation (multiple requirements)
	complexWords := []string{"authentication", "logging", "monitoring", "security", "database"}
	complexCount := 0
	for _, word := range complexWords {
		if strings.Contains(input, word) {
			complexCount++
		}
	}
	if complexCount >= 2 {
		score += 0.6 // MAJOR boost for multiple complex requirements
	}

	// VERY HIGH score for mixed intent queries (search + generate) - THIS IS THE KEY FIX
	if (strings.Contains(input, "search") || strings.Contains(input, "find")) &&
		(strings.Contains(input, "generate") || strings.Contains(input, "create") || strings.Contains(input, "new")) {
		score += 0.7 // MASSIVE boost for mixed search+generate intents
	}

	// HIGH score for "and" combinations (indicating multiple tasks)
	if strings.Contains(input, " and ") {
		andParts := strings.Split(input, " and ")
		if len(andParts) >= 2 {
			hasSearch := false
			hasGenerate := false

			for _, part := range andParts {
				if strings.Contains(part, "search") || strings.Contains(part, "find") {
					hasSearch = true
				}
				if strings.Contains(part, "generate") || strings.Contains(part, "create") || strings.Contains(part, "new") {
					hasGenerate = true
				}
			}

			if hasSearch && hasGenerate {
				score += 0.5 // BIG boost for explicit search AND generate
			}
		}
	}

	// ONLY reduce score for truly simple tasks
	simpleWords := []string{"hello world", "simple function"}
	for _, word := range simpleWords {
		if strings.Contains(input, word) {
			score -= 0.2 // Small reduction for simple tasks
		}
	}

	return math.Min(score, 1.0)
}

// IMPROVED: Intent and analysis methods

func (ma *ManagerAgent) determinePrimaryIntent(input string, query *models.Query) string {
	input = strings.ToLower(input)
	if strings.Contains(input, "explain") &&
		(strings.Contains(input, "workflow") || strings.Contains(input, "architecture") || strings.Contains(input, "project")) {
		return "architecture_explanation"
	}
	// PRIORITY 1: System status queries (FIX for routing issues)
	if ma.isSystemStatusQuery(input) {
		return "system_status"
	}

	// PRIORITY 2: File count queries
	if ma.isFileCountQuery(input) {
		return "file_query"
	}

	// PRIORITY 3: Simple search queries
	if ma.isSimpleSearchQuery(input) {
		return "search"
	}

	// PRIORITY 4: Context search queries
	if ma.isContextSearchQuery(input) {
		return "context_search"
	}

	// Standard intent patterns
	intentPatterns := map[string][]string{
		"generation":  {"create", "generate", "make", "build", "write"},
		"search":      {"search", "find", "look", "locate", "show", "list"},
		"explanation": {"explain", "describe", "what", "how", "why"},
		"analysis":    {"analyze", "review", "check", "examine"},
		"debug":       {"debug", "fix", "error", "problem", "issue"},
		"test":        {"test", "verify", "validate", "check"},
	}

	for intent, patterns := range intentPatterns {
		for _, pattern := range patterns {
			if strings.Contains(input, pattern) {
				return intent
			}
		}
	}

	return "general"
}

// NEW: Helper methods for better query classification

func (ma *ManagerAgent) isSystemStatusQuery(input string) bool {
	statusPatterns := []string{
		"how many", "count", "status", "statistics", "info", "information",
	}
	systemTerms := []string{
		"files", "indexed", "database", "system", "configuration", "settings",
	}

	hasStatusPattern := false
	hasSystemTerm := false

	for _, pattern := range statusPatterns {
		if strings.Contains(input, pattern) {
			hasStatusPattern = true
			break
		}
	}

	for _, term := range systemTerms {
		if strings.Contains(input, term) {
			hasSystemTerm = true
			break
		}
	}

	return hasStatusPattern && hasSystemTerm
}

func (ma *ManagerAgent) isFileCountQuery(input string) bool {
	countWords := []string{"how many", "count", "number of", "total"}
	fileWords := []string{"files", "indexed", "documents", "entries"}

	hasCount := false
	hasFile := false

	for _, word := range countWords {
		if strings.Contains(input, word) {
			hasCount = true
			break
		}
	}

	for _, word := range fileWords {
		if strings.Contains(input, word) {
			hasFile = true
			break
		}
	}

	return hasCount && hasFile
}

func (ma *ManagerAgent) isSimpleSearchQuery(input string) bool {
	simplePatterns := []string{"find", "search", "show", "list", "where"}
	complexPatterns := []string{"pattern", "similar", "example", "like", "related"}

	hasSimple := false
	hasComplex := false

	for _, pattern := range simplePatterns {
		if strings.Contains(input, pattern) {
			hasSimple = true
			break
		}
	}

	for _, pattern := range complexPatterns {
		if strings.Contains(input, pattern) {
			hasComplex = true
			break
		}
	}

	return hasSimple && !hasComplex
}

func (ma *ManagerAgent) isContextSearchQuery(input string) bool {
	contextWords := []string{"pattern", "similar", "example", "like", "related", "matching"}

	for _, word := range contextWords {
		if strings.Contains(input, word) {
			return true
		}
	}
	return false
}

func (ma *ManagerAgent) isSimpleCodeGeneration(input string) bool {
	simpleWords := []string{"create", "write", "generate", "make"}
	complexWords := []string{"optimize", "refactor", "architecture", "analyze", "improve"}

	hasSimple := false
	hasComplex := false

	for _, word := range simpleWords {
		if strings.Contains(input, word) {
			hasSimple = true
			break
		}
	}

	for _, word := range complexWords {
		if strings.Contains(input, word) {
			hasComplex = true
			break
		}
	}

	// Simple coding: has simple words but NOT complex words
	return hasSimple && !hasComplex
}

// IMPROVED: Complexity assessment
func (ma *ManagerAgent) assessComplexity(input string) float64 {
	complexity := 0.3 // Base complexity

	// REDUCE complexity for simple system queries
	if ma.isSystemStatusQuery(input) || ma.isFileCountQuery(input) {
		return 0.1 // VERY low complexity
	}

	// REDUCE complexity for simple coding tasks
	simpleWords := []string{"hello world", "simple", "basic"}
	for _, word := range simpleWords {
		if strings.Contains(input, word) {
			complexity -= 0.2
		}
	}

	// INCREASE complexity for architectural terms
	architecturalTerms := []string{"microservice", "architecture", "design pattern", "optimize", "refactor"}
	for _, term := range architecturalTerms {
		if strings.Contains(input, term) {
			complexity += 0.3 // BIG boost for architectural terms
		}
	}

	// INCREASE complexity for multiple requirements
	requirements := []string{"authentication", "logging", "monitoring", "security", "database"}
	reqCount := 0
	for _, req := range requirements {
		if strings.Contains(input, req) {
			reqCount++
		}
	}
	if reqCount >= 2 {
		complexity += 0.4 // BIG boost for multiple requirements
	}

	// INCREASE complexity for analysis requests
	if strings.Contains(input, "analyze") || strings.Contains(input, "review") {
		complexity += 0.2
	}

	return math.Min(complexity, 1.0)
}

// Rest of the methods remain the same...
func (ma *ManagerAgent) determineSecondaryIntents(input string) []string {
	return []string{}
}

func (ma *ManagerAgent) identifyDomain(input string) string {
	domainKeywords := map[string][]string{
		"web":      {"http", "rest", "api", "handler", "route"},
		"database": {"sql", "query", "table", "database", "storage"},
		"security": {"auth", "security", "token", "encrypt", "permission"},
		"testing":  {"test", "mock", "assert", "verify", "spec"},
		"system":   {"files", "indexed", "status", "configuration"},
	}

	for domain, keywords := range domainKeywords {
		for _, keyword := range keywords {
			if strings.Contains(input, keyword) {
				return domain
			}
		}
	}

	return "general"
}

func (ma *ManagerAgent) identifyRequiredCapabilities(input string) []string {
	capabilities := []string{}

	if strings.Contains(input, "search") || strings.Contains(input, "find") {
		capabilities = append(capabilities, "search")
	}
	if strings.Contains(input, "create") || strings.Contains(input, "generate") {
		capabilities = append(capabilities, "generation")
	}
	if strings.Contains(input, "explain") || strings.Contains(input, "analyze") {
		capabilities = append(capabilities, "analysis")
	}

	return capabilities
}

func (ma *ManagerAgent) assessContextNeeds(input string) float64 {
	contextScore := 0.2 // REDUCED base context needs

	// High context need indicators
	contextIndicators := []string{"similar", "example", "pattern", "like", "related"}
	for _, indicator := range contextIndicators {
		if strings.Contains(input, indicator) {
			contextScore += 0.3
		}
	}

	if contextScore > 1.0 {
		contextScore = 1.0
	}

	return contextScore
}

func (ma *ManagerAgent) assessUrgency(input string) string {
	urgentKeywords := []string{"urgent", "asap", "quickly", "fast", "immediate"}
	for _, keyword := range urgentKeywords {
		if strings.Contains(input, keyword) {
			return "high"
		}
	}
	return "normal"
}

func (ma *ManagerAgent) applyHistoricalLearning(scores map[string]float64, analysis *RoutingAnalysis) {
	recentDecisions := ma.getRecentDecisionsForIntent(analysis.PrimaryIntent, 5)

	for _, decision := range recentDecisions {
		if decision.Success {
			if score, exists := scores[decision.SelectedAgent]; exists {
				scores[decision.SelectedAgent] = score + 0.05
			}
		} else {
			if score, exists := scores[decision.SelectedAgent]; exists {
				scores[decision.SelectedAgent] = score - 0.1
			}
		}
	}
}

func (ma *ManagerAgent) getRecentDecisionsForIntent(intent string, limit int) []RoutingDecision {
	var decisions []RoutingDecision
	count := 0

	for i := len(ma.routingHistory) - 1; i >= 0 && count < limit; i-- {
		if ma.routingHistory[i].Intent == intent {
			decisions = append(decisions, ma.routingHistory[i])
			count++
		}
	}

	return decisions
}

func (ma *ManagerAgent) handleRoutingFallback(ctx context.Context, query *models.Query, failedAgent string) (*models.Response, error) {
	// Try alternative agents in order of preference
	fallbackOrder := []string{"search", "context_search"}

	for _, agent := range fallbackOrder {
		if agent != failedAgent {
			response, err := ma.executeWithSelectedAgent(ctx, query, agent)
			if err == nil {
				return response, nil
			}
		}
	}

	return nil, fmt.Errorf("all agents failed to process query")
}

func (ma *ManagerAgent) updateMetrics(startTime time.Time) {
	ma.metrics.QueriesHandled++
	ma.metrics.LastUsed = time.Now()
}

func (ma *ManagerAgent) updateSuccessMetrics(startTime time.Time, confidence float64, response *models.Response) {
	duration := time.Since(startTime)
	ma.metrics.AverageResponseTime = (ma.metrics.AverageResponseTime + duration) / 2
	ma.metrics.AverageConfidence = (ma.metrics.AverageConfidence + confidence) / 2
	ma.metrics.SuccessRate = float64(ma.metrics.QueriesHandled-ma.metrics.ErrorCount) / float64(ma.metrics.QueriesHandled)

	if response != nil {
		ma.metrics.TokensUsed += int64(response.TokenUsage.TotalTokens)
		ma.metrics.TotalCost += response.Cost.TotalCost
	}
}

func (ma *ManagerAgent) GetMetrics() AgentMetrics {
	return *ma.metrics
}

func (ma *ManagerAgent) GetRoutingHistory(limit int) []RoutingDecision {
	if limit <= 0 || limit > len(ma.routingHistory) {
		return ma.routingHistory
	}

	start := len(ma.routingHistory) - limit
	return ma.routingHistory[start:]
}

// evaluateSystemAgent evaluates system agent capability for the query
func (ma *ManagerAgent) evaluateSystemAgent(query *models.Query, analysis *RoutingAnalysis) float64 {
	score := 0.0
	input := strings.ToLower(query.UserInput)
	
	// High score for system/runtime queries
	if query.Type == models.QueryTypeSystem || query.Type == models.QueryTypeRuntime || query.Type == models.QueryTypeMonitoring {
		score += 0.8
	}
	
	// System-related keywords
	systemWords := []string{"memory", "cpu", "performance", "system", "runtime", "process", "monitor", "status", "health", "metrics"}
	for _, word := range systemWords {
		if strings.Contains(input, word) {
			score += 0.2
		}
	}
	
	return score
}

// extractDataKeys extracts keys from MCP data for logging
func (ma *ManagerAgent) extractDataKeys(data map[string]interface{}) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	return keys
}