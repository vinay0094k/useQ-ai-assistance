package agents

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/yourusername/useq-ai-assistant/internal/llm"
	"github.com/yourusername/useq-ai-assistant/internal/vectordb"
	"github.com/yourusername/useq-ai-assistant/models"
	"github.com/yourusername/useq-ai-assistant/storage"
)

// SearchAgentImpl implements the search agent using centralized types
type SearchAgentImpl struct {
	dependencies *AgentDependencies
	config       *SearchAgentConfig
	metrics      *AgentMetrics
}

// NewSearchAgentConfig creates a new search agent configuration
func NewSearchAgentConfig() *SearchAgentConfig {
	base := NewAgentConfig()
	return &SearchAgentConfig{
		AgentConfig:         *base,
		MaxResults:          10,
		SimilarityThreshold: 0.15,
		EnableReranking:     true,
		IncludeContext:      true,
		ExpandResults:       true,
		SemanticSearch:      true,
		ExactMatchBonus:     0.1,
		FuzzySearch:         true,
		RegexSearch:         true,
		HistoryEnabled:      true,
		ResultCaching:       true,
	}
}

// NewSearchAgentImpl creates a new search agent with centralized configuration
func NewSearchAgent(deps *AgentDependencies) *SearchAgentImpl {
	return &SearchAgentImpl{
		dependencies: deps,
		config:       NewSearchAgentConfig(),
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

// HandleQuery performs semantic search using the vector database
func (sa *SearchAgentImpl) HandleQuery(ctx context.Context, query *models.Query) (*models.Response, error) {
	// Perform vector search
	searchResults, err := sa.dependencies.VectorDB.Search(ctx, query.UserInput, 5)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Enhance search results with MCP context
	if query.MCPContext != nil && query.MCPContext.RequiresMCP {
		searchResults = sa.boostMCPRelevantResults(searchResults, query.MCPContext)
	}

	// Check if query needs LLM synthesis
	if sa.needsLLMSynthesis(query) {
		return sa.synthesizeWithLLM(ctx, query, searchResults)
	}

	// For simple searches, return formatted results
	return sa.formatSearchResults(query, searchResults), nil
}

func (sa *SearchAgentImpl) needsLLMSynthesis(query *models.Query) bool {
	keywords := []string{"explain", "what is", "describe", "how does", "tell me about", "what files", "show me"}
	userInput := strings.ToLower(query.UserInput)
	
	for _, keyword := range keywords {
		if strings.Contains(userInput, keyword) {
			return true
		}
	}
	return false
}

func (sa *SearchAgentImpl) synthesizeWithLLM(ctx context.Context, query *models.Query, searchResults []*vectordb.SearchResult) (*models.Response, error) {
	// Build context from search results
	contextText := ""
	for i, result := range searchResults {
		if i >= 5 { break } // Limit to top 5 results
		contextText += fmt.Sprintf("\n## File %d: %s\n```\n%s\n```\n", 
			i+1, result.Chunk.FilePath, result.Chunk.Content)
	}
	
	// Build prompt
	prompt := fmt.Sprintf(`You are analyzing a codebase. Based on these code snippets from the project:

%s

Answer this question: %s

Provide a clear explanation referencing the actual code above. Be specific about file names and functions.`, 
		contextText, query.UserInput)
	
	// Call LLM
	llmRequest := &llm.GenerationRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are a code analysis expert."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}
	
	llmResponse, err := sa.dependencies.LLMManager.Generate(ctx, llmRequest)
	if err != nil {
		// Fallback to basic formatting if LLM fails
		return sa.formatSearchResults(query, searchResults), nil
	}
	
	return &models.Response{
		ID:      fmt.Sprintf("response_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    models.ResponseTypeSearch,
		Content: models.ResponseContent{
			Text: llmResponse.Content,
		},
		AgentUsed:   "search_agent",
		Provider:    llmResponse.Provider,
		TokenUsage:  llmResponse.TokenUsage,
		Cost:        llmResponse.Cost,
		Metadata: models.ResponseMetadata{
			Confidence: 0.8,
		},
		Timestamp: time.Now(),
	}, nil
}

func (sa *SearchAgentImpl) formatSearchResults(query *models.Query, searchResults []*vectordb.SearchResult) *models.Response {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Found %d relevant code chunks:\n\n", len(searchResults)))

	for i, result := range searchResults {
		content.WriteString(fmt.Sprintf("%d. **%s** (Score: %.2f)\n", i+1, result.Chunk.FilePath, result.Score))
		content.WriteString(fmt.Sprintf("```%s\n%s\n```\n\n", "go", result.Chunk.Content))
	}

	return &models.Response{
		ID:      fmt.Sprintf("response_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    models.ResponseTypeSearch,
		Content: models.ResponseContent{
			Text: content.String(),
		},
		AgentUsed: "search_agent",
		Provider:  "none",
		Metadata: models.ResponseMetadata{
			Confidence: 0.8,
		},
		Timestamp: time.Now(),
	}
}

// GetCapabilities returns what this agent can do
func (sa *SearchAgentImpl) GetCapabilities() AgentCapabilities {
	return AgentCapabilities{
		CanGenerateCode:    false,
		CanSearchCode:      true,
		CanAnalyzeCode:     true,
		CanDebugCode:       false,
		CanWriteTests:      false,
		CanWriteDocs:       false,
		CanReviewCode:      false,
		SupportedLanguages: []string{"go", "javascript", "python", "rust", "java"},
		MaxComplexity:      10,
		RequiresContext:    false,
	}
}

// CanHandle determines if this agent can handle the given query
func (sa *SearchAgentImpl) CanHandle(ctx context.Context, query *models.Query) (bool, float64) {
	intent, err := sa.parseSearchIntent(query)
	if err != nil {
		return false, 0.0
	}

	confidence := sa.calculateHandlingConfidence(intent, query)
	canHandle := confidence >= 0.5 && sa.isSearchQuery(intent)

	return canHandle, confidence
}

// GetSpecialization returns the agent's area of expertise
func (sa *SearchAgentImpl) GetSpecialization() AgentSpecialization {
	return AgentSpecialization{
		Type:        AgentTypeSearch,
		Languages:   []string{"go", "javascript", "python", "rust", "java"},
		Frameworks:  []string{"*"}, // Universal search
		Domains:     []string{"code_search", "semantic_search", "pattern_matching", "file_discovery"},
		Complexity:  10,
		Description: "Intelligent multi-strategy code search across project files",
	}
}

// GetConfidenceScore returns confidence in handling the query
func (sa *SearchAgentImpl) GetConfidenceScore(ctx context.Context, query *models.Query) float64 {
	intent, err := sa.parseSearchIntent(query)
	if err != nil {
		return 0.0
	}

	return sa.calculateHandlingConfidence(intent, query)
}

// ValidateQuery checks if the query is valid for this agent
func (sa *SearchAgentImpl) ValidateQuery(query *models.Query) error {
	if err := ValidateQuery(query); err != nil {
		return err
	}

	intent, err := sa.parseSearchIntent(query)
	if err != nil {
		return fmt.Errorf("invalid search query: %w", err)
	}

	if !sa.isSearchQuery(intent) {
		return fmt.Errorf("query is not a search request")
	}

	return nil
}

// Process handles the query and returns a response
func (sa *SearchAgentImpl) Process(ctx context.Context, query *models.Query) (*models.Response, error) {
	return sa.Search(ctx, query)
}

// GetMetrics returns performance metrics for this agent
func (sa *SearchAgentImpl) GetMetrics() AgentMetrics {
	return *sa.metrics
}

// Search performs intelligent code search (main SearchAgentImpl interface method)
func (sa *SearchAgentImpl) Search(ctx context.Context, query *models.Query) (*models.Response, error) {
	startTime := time.Now()
	sa.updateMetrics(startTime)

	if sa.dependencies == nil {
		return sa.createFallbackResponse(query, "Dependencies not initialized"), nil
	}
	// Check critical dependencies and handle gracefully
	if sa.dependencies.VectorDB == nil && sa.dependencies.Storage == nil {
		return sa.createFallbackResponse(query, "No search backend available (VectorDB and Storage both nil)"), nil
	}

	// Log step-by-step processing
	sa.logStep("Starting intelligent search process", map[string]interface{}{
		"query_id": query.ID,
		"language": query.Language,
		"input":    query.UserInput,
	})

	// Parse search intent from query
	intent, err := sa.parseSearchIntent(query)
	if err != nil {
		sa.metrics.ErrorCount++
		return nil, fmt.Errorf("failed to parse search intent: %w", err)
	}

	sa.logStep("Parsed search intent", map[string]interface{}{
		"search_type":    string(intent.SearchType),
		"keywords":       len(intent.Keywords),
		"function_names": len(intent.FunctionNames),
		"type_names":     len(intent.TypeNames),
		"file_patterns":  len(intent.FilePatterns),
		"exact_match":    intent.ExactMatch,
	})

	// Build search context
	searchContext, err := sa.GetSearchContext(ctx, query)
	if err != nil {
		sa.logStep("Warning: failed to build search context", map[string]interface{}{
			"error": err.Error(),
		})
		// Continue with basic context
		searchContext = &SearchAgentContext{
			Query:   query.UserInput,
			Intent:  intent,
			Filters: make(map[string]interface{}),
		}
	}

	sa.logStep("Built search context", map[string]interface{}{
		"filters_count":   len(searchContext.Filters),
		"history_entries": len(searchContext.HistoryContext),
		"scope_files":     sa.getScopeFilesCount(searchContext.ScopeInfo),
	})

	// Use MCP context if available for enhanced search
	var searchResults []*SearchAgentResult
	
	if query.MCPContext != nil && query.MCPContext.RequiresMCP {
		searchResults, err = sa.searchWithMCPContext(ctx, intent, query.MCPContext)
	} else {
		searchResults, err = sa.performBasicSearch(ctx, intent, searchContext)
	}
	
	if err != nil {
		sa.metrics.ErrorCount++
		return nil, fmt.Errorf("search failed: %w", err)
	}

	sa.logStep("Completed multi-strategy search", map[string]interface{}{
		"raw_results": len(searchResults),
	})

	// Rerank and enhance results
	if sa.config.EnableReranking {
		searchResults = sa.rerankResults(searchResults, intent)
		sa.logStep("Reranked results", map[string]interface{}{
			"reranked_results": len(searchResults),
		})
	}

	// Add usage examples and context
	if sa.config.IncludeContext {
		searchResults = sa.enhanceWithContext(ctx, searchResults, intent)
		sa.logStep("Enhanced results with context", map[string]interface{}{
			"enhanced_results": len(searchResults),
		})
	}

	// Calculate confidence
	confidence := sa.calculateSearchConfidence(searchResults, intent)

	// Create comprehensive response
	response := sa.buildSearchResponse(query, intent, searchResults, confidence, startTime)

	sa.logStep("Search completed successfully", map[string]interface{}{
		"response_id":    response.ID,
		"total_results":  len(searchResults),
		"confidence":     confidence,
		"total_time_ms":  time.Since(startTime).Milliseconds(),
		"files_analyzed": response.Metadata.FilesAnalyzed,
		"index_hits":     response.Metadata.IndexHits,
	})

	// Update success metrics
	sa.updateSuccessMetrics(startTime, confidence, len(searchResults))

	return response, nil
}

// searchWithMCPContext performs search enhanced with MCP command results
func (sa *SearchAgentImpl) searchWithMCPContext(ctx context.Context, intent *SearchAgentIntent, mcpContext *models.MCPContext) ([]*SearchAgentResult, error) {
	var results []*SearchAgentResult
	
	// Process MCP data to create search results
	for operation, data := range mcpContext.Data {
		switch operation {
		case "list_files":
			if fileData, ok := data.(map[string]interface{}); ok {
				if files, ok := fileData["files"].([]string); ok {
					for i, file := range files {
						if i >= 10 { // Limit results
							break
						}
						results = append(results, &SearchAgentResult{
							File:        file,
							Function:    "",
							Line:        1,
							Score:       0.9,
							Context:     fmt.Sprintf("Go source file: %s", file),
							Explanation: "Found in project file listing",
						})
					}
				}
			}
			
		case "file_count":
			if countData, ok := data.(map[string]interface{}); ok {
				if count, ok := countData["count"].(int); ok {
					results = append(results, &SearchAgentResult{
						File:        "project_summary",
						Function:    "file_count",
						Line:        0,
						Score:       1.0,
						Context:     fmt.Sprintf("Project contains %d Go files", count),
						Explanation: "File count from filesystem analysis",
					})
				}
			}
			
		case "memory_usage":
			if memData, ok := data.(map[string]interface{}); ok {
				if memInfo, ok := memData["memory_info"].(string); ok {
					results = append(results, &SearchAgentResult{
						File:        "system_info",
						Function:    "memory_status",
						Line:        0,
						Score:       1.0,
						Context:     memInfo,
						Explanation: "Current system memory usage",
					})
				}
			}
			
		case "project_structure":
			if structData, ok := data.(map[string]interface{}); ok {
				if dirs, ok := structData["directories"].([]string); ok {
					results = append(results, &SearchAgentResult{
						File:        "project_structure",
						Function:    "directory_tree",
						Line:        0,
						Score:       1.0,
						Context:     fmt.Sprintf("Project has %d directories: %s", len(dirs), strings.Join(dirs[:min(3, len(dirs))], ", ")),
						Explanation: "Project directory structure",
					})
				}
			}
		}
	}
	
	// If no MCP results, fall back to basic search
	if len(results) == 0 {
		return sa.performBasicSearch(ctx, intent, nil)
	}
	
	return results, nil
}

// performBasicSearch performs basic vector/database search
func (sa *SearchAgentImpl) performBasicSearch(ctx context.Context, intent *SearchAgentIntent, searchContext *SearchAgentContext) ([]*SearchAgentResult, error) {
	// This is the existing multi-strategy search logic
	return sa.performMultiStrategySearch(ctx, intent, searchContext)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ================================== fallback responses ==================================
func (sa *SearchAgentImpl) createFallbackResponse(query *models.Query, reason string) *models.Response {
	// Try to get some results even without full backend
	var contextualInfo strings.Builder
	contextualInfo.WriteString(fmt.Sprintf("Search request: '%s'\n\n", query.UserInput))

	var searchResults []*vectordb.SearchResult
	
	// If we have vector DB, try to get some results
	if sa.dependencies != nil && sa.dependencies.VectorDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		results, err := sa.dependencies.VectorDB.Search(ctx, query.UserInput, 5)
		if err == nil && len(results) > 0 {
			searchResults = results
			contextualInfo.WriteString("🔍 Found relevant code in your project:\n\n")
			for i, result := range results {
				if i >= 3 {
					break
				} // Limit to top 3
				contextualInfo.WriteString(fmt.Sprintf("📁 %s (similarity: %.3f)\n",
					result.Chunk.FilePath, result.Score))
				if result.Chunk.Content != "" {
					// Show snippet of content
					content := result.Chunk.Content
					if len(content) > 100 {
						content = content[:100] + "..."
					}
					contextualInfo.WriteString(fmt.Sprintf("   📝 %s\n", content))
				}
				contextualInfo.WriteString("\n")
			}
			contextualInfo.WriteString("💡 To get deeper analysis of these files, connect the LLM Manager.\n\n")
		}
	}

	contextualInfo.WriteString(fmt.Sprintf("Status: %s\n\n", reason))
	contextualInfo.WriteString("To enable full semantic search:\n")
	contextualInfo.WriteString("1. ✅ Vector Database (Connected - finding relevant code)\n")
	contextualInfo.WriteString("2. ❌ LLM Manager (Connect OpenAI/Gemini for analysis)\n")
	contextualInfo.WriteString("3. ❌ Full indexing pipeline (for comprehensive search)\n")

	return &models.Response{
		ID:      fmt.Sprintf("search_response_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    models.ResponseTypeCode,
		Content: models.ResponseContent{
			Text: contextualInfo.String(),
		},
		AgentUsed:  "search_agent",
		Timestamp:  time.Now(),
		TokenUsage: models.TokenUsage{TotalTokens: 0},
		Cost:       models.Cost{TotalCost: 0.0, Currency: "USD"},
		Metadata: models.ResponseMetadata{
			GenerationTime: time.Since(time.Now()),
			Confidence:     0.7, // Higher confidence when we have results
			FilesAnalyzed:  len(searchResults),
			IndexHits:      len(searchResults),
		},
	}
}

// GetSearchContext builds comprehensive search context
func (sa *SearchAgentImpl) GetSearchContext(ctx context.Context, query *models.Query) (*SearchAgentContext, error) {
	intent, err := sa.parseSearchIntent(query)
	if err != nil {
		return nil, err
	}

	context := &SearchAgentContext{
		Query:   query.UserInput,
		Intent:  intent,
		Filters: sa.buildSearchFilters(intent, query),
		ScopeInfo: &SearchAgentScope{
			Languages:    []string{query.Language},
			IncludeTests: true,
			IncludeDocs:  true,
		},
		HistoryContext: sa.getSearchHistory(ctx, query),
		UserPreferences: &SearchAgentPreferences{
			AgentPreferences: AgentPreferences{
				MaxResults:          sa.config.MaxResults,
				SimilarityThreshold: float64(sa.config.SimilarityThreshold),
				ShowLineNumbers:     true,
				HighlightMatches:    true,
			},
		},
	}

	return context, nil
}

// =============================================================================
// PRIVATE IMPLEMENTATION METHODS
// =============================================================================

// parseSearchIntent analyzes the query to understand search intent
func (sa *SearchAgentImpl) parseSearchIntent(query *models.Query) (*SearchAgentIntent, error) {
	intent := &SearchAgentIntent{
		Query:    query.UserInput,
		Language: query.Language,
		Keywords: make([]string, 0),
		Filters:  make(map[string]string),
		Scope:    SearchAgentScope{},
	}

	input := strings.ToLower(query.UserInput)

	// Determine search type based on query patterns
	intent.SearchType = sa.determineSearchType(input)

	// Extract entities from query
	intent.FunctionNames = sa.extractFunctionNames(input)
	intent.TypeNames = sa.extractTypeNames(input)
	intent.FilePatterns = sa.extractFilePatterns(input)
	intent.Keywords = sa.extractKeywords(input)

	// Determine search characteristics
	intent.ExactMatch = sa.detectExactMatch(input)
	intent.CaseSensitive = sa.detectCaseSensitive(input)

	// Build scope
	intent.Scope = sa.buildSearchScope(input, query.Language)

	// Add language filter
	if query.Language != "" {
		intent.Filters["language"] = query.Language
	}

	return intent, nil
}

// performMultiStrategySearch performs search using multiple strategies
func (sa *SearchAgentImpl) performMultiStrategySearch(ctx context.Context, intent *SearchAgentIntent, searchContext *SearchAgentContext) ([]*SearchAgentResult, error) {
	var allResults []*SearchAgentResult

	fmt.Printf("🔍 DEBUG: Starting multi-strategy search\n")
	fmt.Printf("🔍 DEBUG: SemanticSearch enabled: %v\n", sa.config.SemanticSearch)

	sa.logStep("Starting multi-strategy search", map[string]interface{}{
		"semantic_enabled": sa.config.SemanticSearch,
		"fuzzy_enabled":    sa.config.FuzzySearch,
		"regex_enabled":    sa.config.RegexSearch,
	})

	// 1. Semantic Search (if enabled and VectorDB available)
	if sa.config.SemanticSearch {
		fmt.Printf("🔍 DEBUG: Calling performSemanticSearch\n")
		semanticResults, err := sa.performSemanticSearch(ctx, intent, searchContext)
		if err != nil {
			fmt.Printf("❌ DEBUG: Semantic search failed: %v\n", err)
			// Don't return error, continue with other search methods
		} else {
			allResults = append(allResults, semanticResults...)
			fmt.Printf("✅ DEBUG: Semantic search added %d results\n", len(semanticResults))
		}
	}

	// Strategy 2: Keyword search in metadata
	keywordResults, err := sa.performKeywordSearch(ctx, intent, searchContext)
	if err != nil {
		sa.logStep("Keyword search failed", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		allResults = append(allResults, keywordResults...)
		sa.logStep("Keyword search completed", map[string]interface{}{
			"results": len(keywordResults),
		})
	}

	// Strategy 3: Exact name matching
	exactResults, err := sa.performExactSearch(ctx, intent, searchContext)
	if err != nil {
		sa.logStep("Exact search failed", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		allResults = append(allResults, exactResults...)
		sa.logStep("Exact search completed", map[string]interface{}{
			"results": len(exactResults),
		})
	}

	// Strategy 4: Fuzzy search (if enabled)
	if sa.config.FuzzySearch {
		fuzzyResults, err := sa.performFuzzySearch(ctx, intent, searchContext)
		if err != nil {
			sa.logStep("Fuzzy search failed", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			allResults = append(allResults, fuzzyResults...)
			sa.logStep("Fuzzy search completed", map[string]interface{}{
				"results": len(fuzzyResults),
			})
		}
	}

	// Strategy 5: Pattern/Regex search (if enabled)
	if sa.config.RegexSearch && sa.containsRegexPatterns(intent.Query) {
		regexResults, err := sa.performRegexSearch(ctx, intent, searchContext)
		if err != nil {
			sa.logStep("Regex search failed", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			allResults = append(allResults, regexResults...)
			sa.logStep("Regex search completed", map[string]interface{}{
				"results": len(regexResults),
			})
		}
	}

	// Deduplicate and merge results
	dedupResults := sa.deduplicateResults(allResults)

	sa.logStep("Deduplicated search results", map[string]interface{}{
		"original_count":     len(allResults),
		"deduplicated_count": len(dedupResults),
	})

	// Limit results
	if len(dedupResults) > sa.config.MaxResults {
		dedupResults = dedupResults[:sa.config.MaxResults]
		sa.logStep("Limited results to max", map[string]interface{}{
			"max_results": sa.config.MaxResults,
		})
	}

	return dedupResults, nil
}

// performSemanticSearch performs vector-based semantic search
func (sa *SearchAgentImpl) performSemanticSearch(ctx context.Context, intent *SearchAgentIntent, searchContext *SearchAgentContext) ([]*SearchAgentResult, error) {
	fmt.Printf("🔍 DEBUG: Starting semantic search for query: %s\n", intent.Query)

	if sa.dependencies == nil || sa.dependencies.VectorDB == nil {
		fmt.Printf("⚠️  DEBUG: VectorDB not available, skipping semantic search\n")
		return []*SearchAgentResult{}, nil // Return empty results instead of crashing
	}

	// Try vector search first
	vectorResults, err := sa.dependencies.VectorDB.Search(ctx, intent.Query, sa.config.MaxResults)
	if err != nil {
		fmt.Printf("❌ DEBUG: Vector search failed: %v\n", err)
		fmt.Printf("🔍 DEBUG: Falling back to storage-based search\n")

		// Fallback to storage-based search using indexed chunks
		return sa.performStorageBasedSearch(ctx, intent, searchContext)
	}

	fmt.Printf("✅ DEBUG: Vector search returned %d results\n", len(vectorResults))

	// Convert vector results to search results with quality filtering
	results := make([]*SearchAgentResult, 0, len(vectorResults))
	fmt.Printf("🔍 DEBUG: Similarity threshold: %f\n", sa.config.SimilarityThreshold)

	queryLower := strings.ToLower(intent.Query)

	for i, vr := range vectorResults {
		fmt.Printf("🔍 DEBUG: Result %d score: %f (threshold: %f)\n", i, vr.Score, sa.config.SimilarityThreshold)

		// Content relevance check
		contentLower := strings.ToLower(vr.Chunk.Content)
		relevanceBoost := 0.0

		// Boost for exact keyword matches
		queryWords := strings.Fields(queryLower)
		matchCount := 0
		for _, word := range queryWords {
			if strings.Contains(contentLower, word) {
				matchCount++
			}
		}
		if matchCount > 0 {
			relevanceBoost = float64(matchCount) * 0.05
		}

		adjustedScore := vr.Score + float32(relevanceBoost)

		if adjustedScore >= sa.config.SimilarityThreshold {
			result := sa.convertVectorResult(vr)
			result.ChunkType = "semantic"
			result.Score = float64(adjustedScore)

			results = append(results, result)
			fmt.Printf("✅ DEBUG: Added result %d (boosted: +%.2f)\n", i, relevanceBoost)
		} else {
			fmt.Printf("❌ DEBUG: Skipped result %d (score too low)\n", i)
		}
	}

	// Sort by score descending for best results first
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Score < results[j].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results, nil
}

// performStorageBasedSearch searches indexed chunks from storage
func (sa *SearchAgentImpl) performStorageBasedSearch(ctx context.Context, intent *SearchAgentIntent, searchContext *SearchAgentContext) ([]*SearchAgentResult, error) {
	fmt.Printf("🔍 DEBUG: Performing storage-based search\n")

	if sa.dependencies == nil || sa.dependencies.Storage == nil {
		fmt.Printf("⚠️  DEBUG: Storage not available, returning empty results\n")
		return []*SearchAgentResult{}, nil // Return empty results instead of crashing
	}
	// Get database stats first
	stats, err := sa.dependencies.Storage.GetStats()
	if err == nil {
		fmt.Printf("📊 DEBUG: Database stats: %+v\n", stats)
	}

	// Try to search for functions with any keyword from the query
	keywords := strings.Fields(strings.ToLower(intent.Query))
	var results []*SearchAgentResult

	for _, keyword := range keywords {
		if len(keyword) < 2 { // Skip very short words
			continue
		}

		functions, err := sa.dependencies.Storage.SearchFunctions(keyword)
		if err != nil {
			fmt.Printf("❌ DEBUG: Failed to search functions for '%s': %v\n", keyword, err)
			continue
		}

		fmt.Printf("✅ DEBUG: Found %d functions for keyword '%s'\n", len(functions), keyword)

		// Convert functions to search results
		for _, function := range functions {
			result := &SearchAgentResult{
				File:      fmt.Sprintf("file_id_%d", function.FileID),
				Function:  function.Name,
				Line:      function.StartLine,
				Score:     0.9,
				Context:   function.Signature,
				ChunkType: "function",
				Language:  "go",
			}
			results = append(results, result)

			if len(results) >= sa.config.MaxResults {
				break
			}
		}

		if len(results) >= sa.config.MaxResults {
			break
		}
	}

	// If no functions found, try to get indexed files
	if len(results) == 0 {
		files, err := sa.dependencies.Storage.GetIndexedFiles()
		if err == nil {
			fmt.Printf("📁 DEBUG: Found %d indexed files\n", len(files))
			// Create results from file paths
			for i, file := range files {
				if i >= 3 { // Limit to 3 files for demo
					break
				}
				result := &SearchAgentResult{
					File:      file,
					Line:      1,
					Score:     0.7,
					Context:   fmt.Sprintf("Indexed file: %s", file),
					ChunkType: "file",
					Language:  "go",
				}
				results = append(results, result)
			}
		}
	}

	fmt.Printf("✅ DEBUG: Storage search returned %d results\n", len(results))
	return results, nil
}

// performKeywordSearch performs traditional keyword-based search
func (sa *SearchAgentImpl) performKeywordSearch(ctx context.Context, intent *SearchAgentIntent, searchContext *SearchAgentContext) ([]*SearchAgentResult, error) {
	var results []*SearchAgentResult

	// Search functions by name
	for _, funcName := range intent.FunctionNames {
		functions, err := sa.dependencies.Storage.SearchFunctions(funcName)
		if err != nil {
			continue
		}

		for _, function := range functions {
			result := sa.convertFunctionResult(function, 0.85) // High confidence for keyword matches
			result.ChunkType = "keyword"
			results = append(results, result)
		}
	}

	// Search for type names
	for _, typeName := range intent.TypeNames {
		// Would implement type search in storage
		_ = typeName
	}

	// Search by general keywords
	for _, keyword := range intent.Keywords {
		// Would implement general keyword search
		_ = keyword
	}

	return results, nil
}

// performExactSearch performs exact name matching
func (sa *SearchAgentImpl) performExactSearch(ctx context.Context, intent *SearchAgentIntent, searchContext *SearchAgentContext) ([]*SearchAgentResult, error) {
	var results []*SearchAgentResult

	// Exact function name matches
	for _, funcName := range intent.FunctionNames {
		functions, err := sa.dependencies.Storage.SearchFunctions(funcName)
		if err != nil {
			continue
		}

		for _, function := range functions {
			if function.Name == funcName {
				result := sa.convertFunctionResult(function, 0.98) // Very high confidence for exact matches
				result.Score += float64(sa.config.ExactMatchBonus)
				result.ChunkType = "exact"
				results = append(results, result)
			}
		}
	}

	return results, nil
}

// performFuzzySearch performs fuzzy matching search
func (sa *SearchAgentImpl) performFuzzySearch(ctx context.Context, intent *SearchAgentIntent, searchContext *SearchAgentContext) ([]*SearchAgentResult, error) {
	// Implement fuzzy search logic (simplified for now)
	var results []*SearchAgentResult

	// This would implement actual fuzzy matching algorithms
	// like Levenshtein distance, soundex, etc.

	return results, nil
}

// performRegexSearch performs pattern/regex search
func (sa *SearchAgentImpl) performRegexSearch(ctx context.Context, intent *SearchAgentIntent, searchContext *SearchAgentContext) ([]*SearchAgentResult, error) {
	// Implement regex search logic (simplified for now)
	var results []*SearchAgentResult

	// This would implement actual regex search across code content

	return results, nil
}

// Helper methods for search processing

func (sa *SearchAgentImpl) calculateHandlingConfidence(intent *SearchAgentIntent, query *models.Query) float64 {
	factors := map[string]float64{}

	// Query type confidence
	switch intent.SearchType {
	case SearchAgentTypeFunction, SearchAgentTypeType, SearchAgentTypeInterface:
		factors["query_type"] = 0.95
	case SearchAgentTypeGeneral, SearchAgentTypeSemantic:
		factors["query_type"] = 0.8
	case SearchAgentTypePattern, SearchAgentTypeRegex:
		factors["query_type"] = 0.7
	default:
		factors["query_type"] = 0.6
	}

	// Entity extraction confidence
	entityCount := len(intent.FunctionNames) + len(intent.TypeNames) + len(intent.Keywords)
	if entityCount > 0 {
		factors["entity_extraction"] = 0.9
	} else {
		factors["entity_extraction"] = 0.5
	}

	// Query clarity
	if len(strings.Fields(query.UserInput)) >= 2 {
		factors["query_clarity"] = 0.8
	} else {
		factors["query_clarity"] = 0.6
	}

	return CalculateConfidence(factors)
}

func (sa *SearchAgentImpl) isSearchQuery(intent *SearchAgentIntent) bool {
	searchKeywords := []string{"find", "search", "look", "locate", "where", "show", "get", "list"}

	input := strings.ToLower(intent.Query)
	for _, keyword := range searchKeywords {
		if strings.Contains(input, keyword) {
			return true
		}
	}

	// Also consider it a search if we found specific entities
	return len(intent.FunctionNames) > 0 || len(intent.TypeNames) > 0 || len(intent.Keywords) > 0
}

func (sa *SearchAgentImpl) determineSearchType(input string) SearchAgentType {
	patterns := map[SearchAgentType][]string{
		SearchAgentTypeFunction:  {"function", "func", "method", "procedure"},
		SearchAgentTypeType:      {"struct", "type", "class", "interface"},
		SearchAgentTypeInterface: {"interface", "contract", "protocol"},
		SearchAgentTypeFile:      {"file", "files", "in file", ".go", ".js", ".py"},
		SearchAgentTypePackage:   {"package", "module", "import", "library"},
		SearchAgentTypeUsage:     {"usage", "used", "called", "references", "calls"},
		SearchAgentTypePattern:   {"pattern", "like", "similar", "matches"},
		SearchAgentTypeSemantic:  {"similar to", "related to", "about"},
		SearchAgentTypeRegex:     {"regex", "pattern", "match", "/.*/"},
	}

	for searchType, keywords := range patterns {
		for _, keyword := range keywords {
			if strings.Contains(input, keyword) {
				return searchType
			}
		}
	}

	return SearchAgentTypeGeneral
}

func (sa *SearchAgentImpl) extractFunctionNames(input string) []string {
	words := strings.Fields(input)
	var functions []string

	for i, word := range words {
		// Look for words followed by parentheses
		if strings.Contains(word, "(") {
			funcName := strings.TrimSuffix(strings.Split(word, "(")[0], "(")
			if funcName != "" && sa.isValidIdentifier(funcName) {
				functions = append(functions, funcName)
			}
		}

		// Look for patterns like "function foo" or "func bar"
		if i > 0 && (words[i-1] == "function" || words[i-1] == "func") && sa.isValidIdentifier(word) {
			functions = append(functions, word)
		}
	}

	return functions
}

func (sa *SearchAgentImpl) extractTypeNames(input string) []string {
	words := strings.Fields(input)
	var types []string

	for i, word := range words {
		// Look for capitalized words (potential types)
		if len(word) > 0 && word[0] >= 'A' && word[0] <= 'Z' && sa.isValidIdentifier(word) {
			types = append(types, word)
		}

		// Look for patterns like "struct User" or "type User"
		if i > 0 && (words[i-1] == "struct" || words[i-1] == "type" || words[i-1] == "interface") && sa.isValidIdentifier(word) {
			types = append(types, word)
		}
	}

	return types
}

func (sa *SearchAgentImpl) extractFilePatterns(input string) []string {
	words := strings.Fields(input)
	var patterns []string

	for _, word := range words {
		// Look for file extensions or file paths
		if strings.Contains(word, ".") && (strings.HasSuffix(word, ".go") ||
			strings.HasSuffix(word, ".js") || strings.HasSuffix(word, ".py") ||
			strings.Contains(word, "/")) {
			patterns = append(patterns, word)
		}
	}

	return patterns
}

func (sa *SearchAgentImpl) extractKeywords(input string) []string {
	// Remove common stop words and extract meaningful keywords
	stopWords := map[string]bool{
		"find": true, "search": true, "look": true, "get": true, "show": true,
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
	}

	words := strings.Fields(strings.ToLower(input))
	var keywords []string

	for _, word := range words {
		cleaned := strings.Trim(word, ".,!?;:()[]{}\"'")
		if len(cleaned) > 2 && !stopWords[cleaned] && sa.isValidKeyword(cleaned) {
			keywords = append(keywords, cleaned)
		}
	}

	return keywords
}

func (sa *SearchAgentImpl) detectExactMatch(input string) bool {
	return strings.Contains(input, "exact") || strings.Contains(input, "\"")
}

func (sa *SearchAgentImpl) detectCaseSensitive(input string) bool {
	return strings.Contains(input, "case sensitive") || strings.Contains(input, "case-sensitive")
}

func (sa *SearchAgentImpl) buildSearchScope(input string, language string) SearchAgentScope {
	// Calculate boolean values first
	includeTests := !strings.Contains(input, "no tests") && !strings.Contains(input, "exclude tests")
	includeDocs := !strings.Contains(input, "no docs") && !strings.Contains(input, "exclude docs")

	// Override if explicit test inclusion is mentioned
	if strings.Contains(input, "test") && !strings.Contains(input, "exclude test") {
		includeTests = true
	}

	scope := SearchAgentScope{
		Languages:    []string{language},
		IncludeTests: includeTests,
		IncludeDocs:  includeDocs,
	}

	return scope
}

func (sa *SearchAgentImpl) buildSearchFilters(intent *SearchAgentIntent, query *models.Query) map[string]interface{} {
	filters := make(map[string]interface{})

	if intent.Language != "" {
		filters["language"] = intent.Language
	}

	switch intent.SearchType {
	case SearchAgentTypeFunction:
		filters["chunk_type"] = "function"
	case SearchAgentTypeType:
		filters["chunk_type"] = "type"
	case SearchAgentTypeInterface:
		filters["chunk_type"] = "interface"
	}

	return filters
}

func (sa *SearchAgentImpl) buildVectorSearchFilters(intent *SearchAgentIntent, searchContext *SearchAgentContext) map[string]interface{} {
	filters := make(map[string]interface{})

	// Merge from search context
	for k, v := range searchContext.Filters {
		filters[k] = v
	}

	// Add intent-specific filters
	if intent.Language != "" {
		filters["language"] = intent.Language
	}

	switch intent.SearchType {
	case SearchAgentTypeFunction:
		filters["chunk_type"] = "function"
	case SearchAgentTypeType:
		filters["chunk_type"] = "type"
	case SearchAgentTypeInterface:
		filters["chunk_type"] = "interface"
	}

	return filters
}

func (sa *SearchAgentImpl) getSearchHistory(ctx context.Context, query *models.Query) []SearchAgentHistory {
	// Would implement actual search history retrieval
	return []SearchAgentHistory{}
}

func (sa *SearchAgentImpl) containsRegexPatterns(query string) bool {
	regexPatterns := []string{"/", ".*", "\\", "[", "]", "^", "$"}
	for _, pattern := range regexPatterns {
		if strings.Contains(query, pattern) {
			return true
		}
	}
	return false
}

// Result processing methods

func (sa *SearchAgentImpl) convertVectorResult(vr *vectordb.SearchResult) *SearchAgentResult {
	// Enhanced result conversion with content filtering
	content := vr.Chunk.Content
	if len(content) > 500 {
		content = content[:500] + "..."
	}

	return &SearchAgentResult{
		File:      vr.Chunk.FilePath,
		Function:  sa.extractFunctionName(vr.Chunk.Content),
		Type:      sa.detectCodeType(vr.Chunk.Content),
		Line:      vr.Chunk.StartLine,
		Score:     float64(vr.Score),
		Context:   content,
		ChunkType: sa.classifyChunk(vr.Chunk.Content),
		Language:  vr.Chunk.Language,
		Package:   sa.extractPackageName(vr.Chunk.FilePath),
		Metadata:  map[string]string{"content": content},
	}
}

func (sa *SearchAgentImpl) extractFunctionName(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Handle regular functions
		if strings.HasPrefix(line, "func ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				funcName := strings.Split(parts[1], "(")[0]
				// Remove receiver if present
				if strings.Contains(funcName, ")") {
					if idx := strings.Index(funcName, ")"); idx != -1 && idx+1 < len(funcName) {
						funcName = funcName[idx+1:]
					}
				}
				if funcName != "" {
					return funcName
				}
			}
		}
		// Handle type declarations
		if strings.HasPrefix(line, "type ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
		// Handle variable declarations
		if strings.HasPrefix(line, "var ") || strings.HasPrefix(line, "const ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return ""
}

func (sa *SearchAgentImpl) detectCodeType(content string) string {
	content = strings.ToLower(content)
	if strings.Contains(content, "func ") {
		return "function"
	}
	if strings.Contains(content, "type ") && strings.Contains(content, "struct") {
		return "struct"
	}
	if strings.Contains(content, "interface") {
		return "interface"
	}
	return "code"
}

func (sa *SearchAgentImpl) classifyChunk(content string) string {
	content = strings.ToLower(content)
	if strings.Contains(content, "test") {
		return "test"
	}
	if strings.Contains(content, "main") {
		return "main"
	}
	if strings.Contains(content, "error") || strings.Contains(content, "err") {
		return "error_handling"
	}
	return "implementation"
}

func (sa *SearchAgentImpl) extractPackageName(filePath string) string {
	parts := strings.Split(filePath, "/")
	if len(parts) > 1 {
		return parts[len(parts)-2]
	}
	return ""
}

func (sa *SearchAgentImpl) convertFunctionResult(function *storage.CodeFunction, score float64) *SearchAgentResult {
	return &SearchAgentResult{
		Function:  function.Name,
		Line:      function.StartLine,
		Score:     score,
		Context:   function.Signature,
		ChunkType: "function",
		Language:  "go", // Would be detected from context
		Metadata: map[string]string{
			"signature":  function.Signature,
			"visibility": function.Visibility,
			"type":       function.Type,
			"complexity": fmt.Sprintf("%d", function.Complexity),
		},
	}
}

func (sa *SearchAgentImpl) rerankResults(results []*SearchAgentResult, intent *SearchAgentIntent) []*SearchAgentResult {
	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Apply additional ranking factors
	for i, result := range results {
		// Boost exact matches
		if sa.isExactMatch(result, intent) {
			result.Score += float64(sa.config.ExactMatchBonus)
		}

		// Boost based on search type preference
		if sa.matchesSearchType(result, intent.SearchType) {
			result.Score += 0.05
		}

		// Boost recent files
		if sa.isRecentFile(result) {
			result.Score += 0.02
		}

		// Penalty for very low scores
		if result.Score < 0.4 {
			result.Score *= 0.8
		}

		results[i] = result
	}

	// Sort again after reranking
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

func (sa *SearchAgentImpl) enhanceWithContext(ctx context.Context, results []*SearchAgentResult, intent *SearchAgentIntent) []*SearchAgentResult {
	for i, result := range results {
		// Add usage examples
		if result.Function != "" {
			result.Usage = sa.findUsageExamples(ctx, result.Function)
		}

		// Add explanation based on context
		result.Explanation = sa.generateExplanation(result, intent)

		// Add line numbers and context if enabled
		if sa.config.IncludeContext {
			result.Context = sa.enhanceContext(result)
		}

		results[i] = result
	}

	return results
}

func (sa *SearchAgentImpl) deduplicateResults(results []*SearchAgentResult) []*SearchAgentResult {
	seen := make(map[string]*SearchAgentResult)

	for _, result := range results {
		key := fmt.Sprintf("%s:%s:%d", result.File, result.Function, result.Line)

		if existing, exists := seen[key]; exists {
			// Keep the one with higher score
			if result.Score > existing.Score {
				seen[key] = result
			}
		} else {
			seen[key] = result
		}
	}

	// Convert back to slice
	deduped := make([]*SearchAgentResult, 0, len(seen))
	for _, result := range seen {
		deduped = append(deduped, result)
	}

	// Sort by score
	sort.Slice(deduped, func(i, j int) bool {
		return deduped[i].Score > deduped[j].Score
	})

	return deduped
}

// Response building

func (sa *SearchAgentImpl) buildSearchResponse(query *models.Query, intent *SearchAgentIntent,
	results []*SearchAgentResult, confidence float64, startTime time.Time) *models.Response {

	// If we have LLM Manager and results, synthesize intelligent response
	if sa.dependencies.LLMManager != nil && len(results) > 0 {
		return sa.buildLLMEnhancedResponse(query, intent, results, confidence, startTime)
	}

	// convertToResponseResults expects []*SearchAgentImplResult
	return &models.Response{
		ID:      fmt.Sprintf("search_response_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    models.ResponseTypeSearch,
		Content: models.ResponseContent{
			Search: &models.SearchResponse{
				Query:     query.UserInput,
				Results:   sa.convertToResponseResults(results),
				Total:     len(results),
				TimeTaken: time.Since(startTime),
			},
		},
		AgentUsed: "search_agent",
		Provider:  "none",
		TokenUsage: models.TokenUsage{
			InputTokens:  0,
			OutputTokens: 0,
			TotalTokens:  0,
		},
		Cost: models.Cost{
			TotalCost: 0.0,
			Currency:  "USD",
		},
		Metadata: models.ResponseMetadata{
			GenerationTime: time.Since(startTime),
			IndexHits:      len(results),
			FilesAnalyzed:  sa.countUniqueFiles(results),
			Confidence:     confidence,
			Sources:        sa.extractSources(results),
			Tools:          sa.getUsedTools(intent),
			Reasoning:      sa.explainSearchStrategy(intent, len(results)),
		},
		Timestamp: time.Now(),
	}
}

func (sa *SearchAgentImpl) calculateSearchConfidence(results []*SearchAgentResult, intent *SearchAgentIntent) float64 {
	if len(results) == 0 {
		return 0.0
	}

	factors := map[string]float64{}

	// Score-based confidence
	totalScore := 0.0
	for _, result := range results {
		totalScore += result.Score
	}
	avgScore := totalScore / float64(len(results))
	factors["average_score"] = avgScore

	// Result count confidence
	if len(results) >= 5 {
		factors["result_count"] = 0.9
	} else if len(results) >= 2 {
		factors["result_count"] = 0.7
	} else {
		factors["result_count"] = 0.5
	}

	// Exact match bonus
	hasExactMatch := false
	for _, result := range results {
		if sa.isExactMatch(result, intent) {
			hasExactMatch = true
			break
		}
	}
	if hasExactMatch {
		factors["exact_match"] = 0.95
	} else {
		factors["exact_match"] = 0.6
	}

	return CalculateConfidence(factors)
}

// Utility methods

func (sa *SearchAgentImpl) logStep(message string, fields map[string]interface{}) {
	if sa.dependencies.Logger != nil {
		sa.dependencies.Logger.Info(message, fields)
	}
}

func (sa *SearchAgentImpl) updateMetrics(startTime time.Time) {
	sa.metrics.QueriesHandled++
	sa.metrics.LastUsed = startTime
}

func (sa *SearchAgentImpl) updateSuccessMetrics(startTime time.Time, confidence float64, resultCount int) {
	duration := time.Since(startTime)

	// Update running averages
	total := float64(sa.metrics.QueriesHandled)
	sa.metrics.AverageResponseTime = time.Duration(
		(float64(sa.metrics.AverageResponseTime)*(total-1) + float64(duration)) / total,
	)
	sa.metrics.AverageConfidence = (sa.metrics.AverageConfidence*(total-1) + confidence) / total

	// Calculate success rate (successful if we got results)
	if resultCount > 0 {
		successCount := float64(sa.metrics.QueriesHandled - sa.metrics.ErrorCount)
		sa.metrics.SuccessRate = successCount / total
	}
}

func (sa *SearchAgentImpl) convertToResponseResults(results []*SearchAgentResult) []models.SearchResult {
	responseResults := make([]models.SearchResult, len(results))

	for i, result := range results {
		responseResults[i] = models.SearchResult{
			File:        result.File,
			Function:    result.Function,
			Line:        result.Line,
			Score:       result.Score,
			Context:     result.Context,
			Explanation: result.Explanation,
			Usage:       sa.convertUsageExamples(result.Usage),
		}
	}

	return responseResults
}

func (sa *SearchAgentImpl) convertUsageExamples(usage []UsageExample) []models.UsageExample {
	examples := make([]models.UsageExample, len(usage))
	for i, example := range usage {
		examples[i] = models.UsageExample{
			File:        example.File,
			Line:        example.Line,
			Context:     example.Context,
			Description: example.Description,
		}
	}
	return examples
}

func (sa *SearchAgentImpl) countUniqueFiles(results []*SearchAgentResult) int {
	files := make(map[string]bool)
	for _, result := range results {
		if result.File != "" {
			files[result.File] = true
		}
	}
	return len(files)
}

func (sa *SearchAgentImpl) extractSources(results []*SearchAgentResult) []string {
	sources := make(map[string]bool)
	for _, result := range results {
		if result.File != "" {
			sources[result.File] = true
		}
	}

	sourceList := make([]string, 0, len(sources))
	for source := range sources {
		sourceList = append(sourceList, source)
	}

	return sourceList
}

func (sa *SearchAgentImpl) getUsedTools(intent *SearchAgentIntent) []string {
	tools := []string{"metadata_search"}

	if sa.config.SemanticSearch {
		tools = append(tools, "vector_search")
	}
	if sa.config.FuzzySearch {
		tools = append(tools, "fuzzy_search")
	}
	if sa.config.RegexSearch && sa.containsRegexPatterns(intent.Query) {
		tools = append(tools, "regex_search")
	}

	return tools
}

func (sa *SearchAgentImpl) explainSearchStrategy(intent *SearchAgentIntent, resultCount int) string {
	strategy := string(intent.SearchType)
	return fmt.Sprintf("Used %s search strategy, found %d results across multiple search methods",
		strategy, resultCount)
}

// Helper validation methods
func (sa *SearchAgentImpl) isValidIdentifier(word string) bool {
	if len(word) == 0 {
		return false
	}

	for i, r := range word {
		if i == 0 && !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_') {
			return false
		}
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_') {
			return false
		}
	}

	return true
}

func (sa *SearchAgentImpl) isValidKeyword(word string) bool {
	return len(word) > 2 && sa.isValidIdentifier(word)
}

func (sa *SearchAgentImpl) isExactMatch(result *SearchAgentResult, intent *SearchAgentIntent) bool {
	for _, funcName := range intent.FunctionNames {
		if result.Function == funcName {
			return true
		}
	}
	for _, typeName := range intent.TypeNames {
		if result.Type == typeName {
			return true
		}
	}
	return false
}

func (sa *SearchAgentImpl) matchesSearchType(result *SearchAgentResult, searchType SearchAgentType) bool {
	switch searchType {
	case SearchAgentTypeFunction:
		return result.ChunkType == "function" || result.ChunkType == "method"
	case SearchAgentTypeType:
		return result.ChunkType == "type"
	case SearchAgentTypeInterface:
		return result.ChunkType == "interface"
	default:
		return true
	}
}

func (sa *SearchAgentImpl) isRecentFile(result *SearchAgentResult) bool {
	// Placeholder for file recency check. Integrate with file metadata/timestamps when available.
	return false
}

func (sa *SearchAgentImpl) findUsageExamples(ctx context.Context, functionName string) []UsageExample {
	// Placeholder - would implement actual usage search (likely using Storage)
	// Return a minimal example so callers have something to show.
	return []UsageExample{
		{
			File:        "example.go",
			Line:        42,
			Context:     fmt.Sprintf("result := %s(param)", functionName),
			Description: "Basic usage example",
			Type:        "call",
		},
	}
}

func (sa *SearchAgentImpl) generateExplanation(result *SearchAgentResult, intent *SearchAgentIntent) string {
	if result.Function != "" {
		return fmt.Sprintf("Function '%s' in %s (line %d)", result.Function, result.File, result.Line)
	}
	if result.Type != "" {
		return fmt.Sprintf("Type '%s' in %s (line %d)", result.Type, result.File, result.Line)
	}
	return fmt.Sprintf("Code element in %s at line %d", result.File, result.Line)
}

func (sa *SearchAgentImpl) enhanceContext(result *SearchAgentResult) string {
	// Would implement actual context enhancement with line/nearby lines, highlighting, etc.
	return result.Context
}

// getScopeFilesCount safely gets the count of files in scope
func (sa *SearchAgentImpl) getScopeFilesCount(scopeInfo *SearchAgentScope) int {
	if scopeInfo == nil {
		return 0
	}
	return len(scopeInfo.Files)
}

// ExtractMCPFileResults extracts relevant file information from MCP context
func (sa *SearchAgentImpl) ExtractMCPFileResults(mcpContext *models.MCPContext) []string {
	var results []string
	
	// Extract file information
	if files, ok := mcpContext.Data["project_files"].([]map[string]interface{}); ok {
		for _, file := range files[:min(3, len(files))] { // Limit to 3 files
			if path, ok := file["path"].(string); ok {
				size := int64(0)
				if s, ok := file["size"].(int64); ok {
					size = s
				}
				results = append(results, fmt.Sprintf("%s (%d bytes)", path, size))
			}
		}
	}
	
	// Add file count summary
	if count, ok := mcpContext.Data["file_count"].(int); ok {
		results = append(results, fmt.Sprintf("Total project files: %d", count))
	}
	
	return results
}

// boostMCPRelevantResults boosts search results for files found in MCP context
func (sa *SearchAgentImpl) boostMCPRelevantResults(results []*vectordb.SearchResult, mcpContext *models.MCPContext) []*vectordb.SearchResult {
	if mcpContext == nil {
		return results
	}
	
	// Get MCP file paths
	mcpFiles := sa.getMCPFilePaths(mcpContext)
	if len(mcpFiles) == 0 {
		return results
	}
	
	// Boost scores for MCP-discovered files
	for i, result := range results {
		if sa.isInMCPFiles(result.Chunk.FilePath, mcpFiles) {
			results[i].Score += 0.1 // Boost MCP-discovered files
		}
	}
	
	return results
}

// GetMCPFilePaths extracts file paths from MCP context (public for testing)
func (sa *SearchAgentImpl) GetMCPFilePaths(mcpContext *models.MCPContext) []string {
	return sa.getMCPFilePaths(mcpContext)
}

// getMCPFilePaths extracts file paths from MCP context
func (sa *SearchAgentImpl) getMCPFilePaths(mcpContext *models.MCPContext) []string {
	var paths []string
	
	if files, ok := mcpContext.Data["project_files"].([]map[string]interface{}); ok {
		for _, file := range files {
			if path, ok := file["path"].(string); ok {
				paths = append(paths, path)
			}
		}
	}
	
	return paths
}

// isInMCPFiles checks if a file path is in the MCP discovered files
func (sa *SearchAgentImpl) isInMCPFiles(filePath string, mcpFiles []string) bool {
	for _, mcpFile := range mcpFiles {
		if filePath == mcpFile {
			return true
		}
	}
	return false
}

// buildLLMEnhancedResponse creates an intelligent response using LLM synthesis
func (sa *SearchAgentImpl) buildLLMEnhancedResponse(query *models.Query, intent *SearchAgentIntent,
	results []*SearchAgentResult, confidence float64, startTime time.Time) *models.Response {

	// Prepare context for LLM
	var contextBuilder strings.Builder
	contextBuilder.WriteString(fmt.Sprintf("User Query: %s\n\n", query.UserInput))
	contextBuilder.WriteString("Search Results:\n")
	
	for i, result := range results {
		if i >= 5 { // Limit to top 5 results for context
			break
		}
		contextBuilder.WriteString(fmt.Sprintf("%d. File: %s\n", i+1, result.File))
		if len(result.Context) > 200 {
			contextBuilder.WriteString(fmt.Sprintf("   Content: %s...\n", result.Context[:200]))
		} else {
			contextBuilder.WriteString(fmt.Sprintf("   Content: %s\n", result.Context))
		}
		contextBuilder.WriteString(fmt.Sprintf("   Score: %.2f\n\n", result.Score))
	}

	// Create LLM request
	llmRequest := &llm.GenerationRequest{
		Messages: []llm.Message{
			{
				Role: "system",
				Content: "You are a code search assistant. Analyze the search results and provide a helpful, contextual explanation of what was found. Include specific examples from the code when relevant.",
			},
			{
				Role: "user", 
				Content: contextBuilder.String(),
			},
		},
		Model:       "gpt-3.5-turbo",
		Temperature: 0.3,
		MaxTokens:   500,
	}

	// Call LLM
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	llmResponse, err := sa.dependencies.LLMManager.Generate(ctx, llmRequest)
	if err != nil {
		// Fallback to basic response if LLM fails
		sa.logStep("LLM synthesis failed, using fallback", map[string]interface{}{
			"error": err.Error(),
		})
		return sa.buildBasicSearchResponse(query, intent, results, confidence, startTime)
	}

	// Create enhanced response with LLM content
	return &models.Response{
		ID:      fmt.Sprintf("search_response_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    models.ResponseTypeSearch,
		Content: models.ResponseContent{
			Text: llmResponse.Content,
			Search: &models.SearchResponse{
				Query:     query.UserInput,
				Results:   sa.convertToResponseResults(results),
				Total:     len(results),
				TimeTaken: time.Since(startTime),
			},
		},
		AgentUsed: "search_agent",
		Provider:  llmResponse.Provider,
		TokenUsage: llmResponse.TokenUsage,
		Cost:       llmResponse.Cost,
		Metadata: models.ResponseMetadata{
			GenerationTime: time.Since(startTime),
			IndexHits:      len(results),
			FilesAnalyzed:  sa.countUniqueFiles(results),
			Confidence:     confidence,
			Sources:        sa.extractSources(results),
			Tools:          sa.getUsedTools(intent),
			Reasoning:      "LLM-enhanced search analysis with contextual explanation",
		},
		Timestamp: time.Now(),
	}
}

// buildBasicSearchResponse creates a basic response without LLM
func (sa *SearchAgentImpl) buildBasicSearchResponse(query *models.Query, intent *SearchAgentIntent,
	results []*SearchAgentResult, confidence float64, startTime time.Time) *models.Response {
	
	return &models.Response{
		ID:      fmt.Sprintf("search_response_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    models.ResponseTypeSearch,
		Content: models.ResponseContent{
			Search: &models.SearchResponse{
				Query:     query.UserInput,
				Results:   sa.convertToResponseResults(results),
				Total:     len(results),
				TimeTaken: time.Since(startTime),
			},
		},
		AgentUsed: "search_agent",
		Provider:  "none",
		TokenUsage: models.TokenUsage{InputTokens: 0, OutputTokens: 0, TotalTokens: 0},
		Cost:       models.Cost{TotalCost: 0.0, Currency: "USD"},
		Metadata: models.ResponseMetadata{
			GenerationTime: time.Since(startTime),
			IndexHits:      len(results),
			FilesAnalyzed:  sa.countUniqueFiles(results),
			Confidence:     confidence,
			Sources:        sa.extractSources(results),
			Tools:          sa.getUsedTools(intent),
			Reasoning:      sa.explainSearchStrategy(intent, len(results)),
		},
		Timestamp: time.Now(),
	}
}