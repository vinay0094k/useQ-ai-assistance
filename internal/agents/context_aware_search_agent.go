package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// ContextAwareSearchAgentImpl provides intelligent, context-aware search capabilities
type ContextAwareSearchAgentImpl struct {
	dependencies *AgentDependencies
	config       *ContextAwareSearchAgentConfig
	metrics      *AgentMetrics
}

// NewContextAwareSearchAgentConfig creates a new ContextAwareSearchAgentConfig with sensible defaults.
func NewContextAwareSearchAgentConfig() *ContextAwareSearchAgentConfig {
	base := NewAgentConfig() // assumes NewAgentConfig() exists and returns *AgentConfig
	return &ContextAwareSearchAgentConfig{
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

// NewContextAwareSearchAgentImpl creates a new context-aware search agent
func NewContextAwareSearchAgentImpl(deps *AgentDependencies) *ContextAwareSearchAgentImpl {
	return &ContextAwareSearchAgentImpl{
		dependencies: deps,
		config:       NewContextAwareSearchAgentConfig(),
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

// CanHandle determines if this agent can handle the query
func (casa *ContextAwareSearchAgentImpl) CanHandle(ctx context.Context, query *models.Query) (bool, float64) {
	confidence := casa.calculateSearchConfidence(query)
	return confidence >= 0.7, confidence
}

// Process performs intelligent context-aware search
func (casa *ContextAwareSearchAgentImpl) Process(ctx context.Context, query *models.Query) (*models.Response, error) {
	startTime := time.Now()
	casa.updateMetrics(startTime)

	// Parse search intent with context awareness
	searchIntent := casa.parseSearchIntent(query)

	// Enhance search intent with MCP context if available
	if query.MCPContext != nil && query.MCPContext.RequiresMCP {
		casa.enhanceSearchIntentWithMCP(searchIntent, query.MCPContext)
	}

	// Build contextual search strategy
	strategy := casa.buildSearchStrategy(ctx, searchIntent, query)

	// Execute multi-layered search
	results, err := casa.executeContextualSearch(ctx, strategy, query)
	if err != nil {
		casa.metrics.ErrorCount++
		return nil, fmt.Errorf("contextual search failed: %w", err)
	}

	// Enhance results with context
	enhancedResults := casa.enhanceResultsWithContext(ctx, results, searchIntent)

	// Build intelligent response
	response := casa.buildContextualResponse(query, searchIntent, enhancedResults)

	casa.updateSuccessMetrics(startTime, 0.85, &models.TokenUsage{})
	return response, nil
}

// parseSearchIntent extracts intelligent search intent
func (casa *ContextAwareSearchAgentImpl) parseSearchIntent(query *models.Query) *ContextAwareSearchAgentIntent {
	input := strings.ToLower(query.UserInput)

	intent := &ContextAwareSearchAgentIntent{
		Query:      query.UserInput,
		SearchType: casa.determineSearchType(input),
		Keywords:   casa.extractSmartKeywords(input),
		Context:    casa.inferContext(input),
		Scope:      ContextAwareSearchAgentScope{}, // matches the type in your types file
		Filters:    casa.convertFiltersToStringMap(casa.extractFilters(input)),
		Precision:  casa.calculateRequiredPrecision(input),
	}

	return intent
}

// buildSearchStrategy creates intelligent search strategy
func (casa *ContextAwareSearchAgentImpl) buildSearchStrategy(ctx context.Context, intent *ContextAwareSearchAgentIntent, query *models.Query) *ContextAwareSearchAgentStrategy {
	return &ContextAwareSearchAgentStrategy{
		PrimaryMethod:    casa.selectPrimarySearchMethod(intent),
		SecondaryMethods: casa.selectSecondaryMethods(intent),
		ContextLayers:    casa.buildContextLayers(ctx, intent),
		RankingFactors:   casa.defineRankingFactors(intent),
		FilterChain:      casa.buildFilterChain(intent),
		MaxResults:       casa.calculateOptimalResultCount(intent),
	}
}

// executeContextualSearch performs multi-layered intelligent search
func (casa *ContextAwareSearchAgentImpl) executeContextualSearch(ctx context.Context, strategy *ContextAwareSearchAgentStrategy, query *models.Query) ([]*SearchResult, error) {
	var allResults []*SearchResult

	// Layer 1: Semantic search with context
	semanticResults, err := casa.performSemanticSearchWithContext(ctx, strategy, query)
	if err == nil && len(semanticResults) > 0 {
		allResults = append(allResults, semanticResults...)
	}

	// Layer 2: Structural pattern search
	structuralResults, err := casa.performStructuralSearch(ctx, strategy, query)
	if err == nil && len(structuralResults) > 0 {
		allResults = append(allResults, structuralResults...)
	}

	// Layer 3: Dependency-aware search
	dependencyResults, err := casa.performDependencyAwareSearch(ctx, strategy, query)
	if err == nil && len(dependencyResults) > 0 {
		allResults = append(allResults, dependencyResults...)
	}

	// Layer 4: Usage pattern search
	usageResults, err := casa.performUsagePatternSearch(ctx, strategy, query)
	if err == nil && len(usageResults) > 0 {
		allResults = append(allResults, usageResults...)
	}

	// Intelligent result fusion and ranking
	return casa.fuseAndRankResults(allResults, strategy), nil
}

// enhanceResultsWithContext adds intelligent context to results
func (casa *ContextAwareSearchAgentImpl) enhanceResultsWithContext(ctx context.Context, results []*SearchResult, intent *ContextAwareSearchAgentIntent) []*EnhancedSearchResult {
	enhanced := make([]*EnhancedSearchResult, len(results))

	for i, result := range results {
		enhanced[i] = &EnhancedSearchResult{
			SearchResult: *result,
			ContextualInfo: map[string]interface{}{
				"file_type": "go",
				"relevance": result.Score,
			},
			RelatedPatterns: []string{},
			UsageExamples:   []string{},
			Dependencies:    []string{},
			QualityMetrics:  map[string]float64{"relevance": result.Score},
			RelevanceScore:  result.Score,
		}
	}

	return enhanced
}

// Helper methods for search intelligence

func (casa *ContextAwareSearchAgentImpl) determineSearchType(input string) SearchType {
	patterns := map[SearchType][]string{
		SearchTypeFunction:  {"function", "func", "method", "procedure"},
		SearchTypeType:      {"struct", "type", "model", "entity"},
		SearchTypeInterface: {"interface", "contract", "protocol"},
		SearchTypePattern:   {"pattern", "example", "how to", "usage"},
		SearchTypeGeneral:   {"error", "bug", "issue", "problem", "test", "testing", "spec", "verify", "config", "setting", "option", "parameter"},
	}

	for searchType, keywords := range patterns {
		for _, keyword := range keywords {
			if strings.Contains(input, keyword) {
				return searchType
			}
		}
	}

	return SearchTypeGeneral
}

func (casa *ContextAwareSearchAgentImpl) extractSmartKeywords(input string) []string {
	// Advanced keyword extraction with context awareness
	words := strings.Fields(input)
	var keywords []string

	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "how": true,
		"what": true, "where": true, "when": true, "why": true,
	}

	for _, word := range words {
		word = strings.ToLower(strings.Trim(word, ".,!?;:"))
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

func (casa *ContextAwareSearchAgentImpl) calculateSearchConfidence(query *models.Query) float64 {
	confidence := 0.5 // Base confidence

	input := strings.ToLower(query.UserInput)

	// Boost for search-related terms
	searchTerms := []string{"search", "find", "look", "locate", "show", "list", "get"}
	for _, term := range searchTerms {
		if strings.Contains(input, term) {
			confidence += 0.2
			break
		}
	}

	// Boost for specific technical terms
	if strings.Contains(input, "function") || strings.Contains(input, "method") {
		confidence += 0.15
	}

	// Boost for code-related queries
	codeTerms := []string{"struct", "interface", "type", "package", "import"}
	for _, term := range codeTerms {
		if strings.Contains(input, term) {
			confidence += 0.1
		}
	}

	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// Placeholder implementations for complex methods
func (casa *ContextAwareSearchAgentImpl) inferContext(input string) map[string]interface{} {
	return map[string]interface{}{
		"domain":     casa.inferDomain(input),
		"complexity": casa.inferComplexity(input),
		"urgency":    casa.inferUrgency(input),
	}
}

func (casa *ContextAwareSearchAgentImpl) performSemanticSearchWithContext(ctx context.Context, strategy *ContextAwareSearchAgentStrategy, query *models.Query) ([]*SearchResult, error) {
	if casa.dependencies == nil || casa.dependencies.VectorDB == nil {
		return []*SearchResult{}, nil
	}

	// Use existing vector search with enhanced query
	searchQuery := fmt.Sprintf("find similar %s examples", query.UserInput)
	results, err := casa.dependencies.VectorDB.Search(ctx, searchQuery, 10)
	if err != nil {
		return []*SearchResult{}, err
	}

	// Convert to SearchResult format
	searchResults := make([]*SearchResult, len(results))
	for i, result := range results {
		// Try to extract function name from content
		functionName := extractFunctionName(result.Chunk.Content)
		if functionName == "" {
			functionName = fmt.Sprintf("lines_%d-%d", result.Chunk.StartLine, result.Chunk.EndLine)
		}

		searchResults[i] = &SearchResult{
			File:     result.Chunk.FilePath,
			Function: functionName,
			Line:     result.Chunk.StartLine,
			Score:    float64(result.Score),
			Context:  result.Chunk.Content,
			Language: result.Chunk.Language,
		}
	}

	return searchResults, nil
}

// extractFunctionName tries to extract function name from code content
func extractFunctionName(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for Go function declarations
		if strings.HasPrefix(line, "func ") {
			// Extract function name
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				funcName := parts[1]
				// Remove receiver if present
				if strings.Contains(funcName, ")") {
					if idx := strings.Index(funcName, ")"); idx != -1 && idx+1 < len(funcName) {
						funcName = funcName[idx+1:]
					}
				}
				// Remove parameters
				if idx := strings.Index(funcName, "("); idx != -1 {
					funcName = funcName[:idx]
				}
				if funcName != "" {
					return funcName
				}
			}
		}
		// Look for type declarations
		if strings.HasPrefix(line, "type ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
		// Look for method declarations
		if strings.Contains(line, "func (") && strings.Contains(line, ")") {
			if idx := strings.Index(line, ")"); idx != -1 {
				remaining := line[idx+1:]
				parts := strings.Fields(strings.TrimSpace(remaining))
				if len(parts) > 0 {
					funcName := parts[0]
					if idx := strings.Index(funcName, "("); idx != -1 {
						funcName = funcName[:idx]
					}
					if funcName != "" {
						return funcName
					}
				}
			}
		}
	}
	return ""
}

func (casa *ContextAwareSearchAgentImpl) buildContextualResponse(query *models.Query, intent *ContextAwareSearchAgentIntent, results []*EnhancedSearchResult) *models.Response {
	return &models.Response{
		ID:      fmt.Sprintf("search_response_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Content: models.ResponseContent{
			Text: casa.formatSearchResults(results),
		},
		Type:      models.ResponseTypeSearch,
		AgentUsed: "context_aware_search_agent",
		Provider:  "internal",
		TokenUsage: models.TokenUsage{
			InputTokens:  0,
			OutputTokens: 0,
			TotalTokens:  0,
		},
		Cost:      models.Cost{TotalCost: 0.0, Currency: "USD"},
		Timestamp: time.Now(),
	}
}

func (casa *ContextAwareSearchAgentImpl) formatSearchResults(results []*EnhancedSearchResult) string {
	if len(results) == 0 {
		return "No results found matching your search criteria."
	}

	// Limit to top 5 results
	maxResults := 5
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("🔍 Found %d relevant matches:\n\n", len(results)))

	for i, result := range results {
		filePath := result.SearchResult.File
		functionName := result.SearchResult.Function
		
		content.WriteString(fmt.Sprintf("%d. 📁 %s", i+1, filePath))
		if functionName != "" && !strings.HasPrefix(functionName, "lines_") {
			content.WriteString(fmt.Sprintf(" → %s()", functionName))
		}
		content.WriteString(fmt.Sprintf(" (%.3f)\n", result.RelevanceScore))
		
		// Format contextual info as readable text
		if result.ContextualInfo != nil {
			contextText := formatContextInfo(result.ContextualInfo)
			if contextText != "" {
				content.WriteString(fmt.Sprintf("   💡 %s\n", contextText))
			}
		}
		
		// Show clean code snippet
		if result.SearchResult.Context != "" {
			snippet := extractCleanCodeSnippet(result.SearchResult.Context)
			content.WriteString(fmt.Sprintf("   📝 %s\n", snippet))
		}
		content.WriteString("\n")
	}

	return content.String()
}

// formatContextInfo converts context map to readable text
func formatContextInfo(contextInfo interface{}) string {
	if contextMap, ok := contextInfo.(map[string]interface{}); ok {
		var parts []string
		if relatedFiles, ok := contextMap["related_files"].([]string); ok && len(relatedFiles) > 0 {
			parts = append(parts, fmt.Sprintf("Related: %s", strings.Join(relatedFiles[:min(2, len(relatedFiles))], ", ")))
		}
		if patterns, ok := contextMap["patterns"].([]string); ok && len(patterns) > 0 {
			parts = append(parts, fmt.Sprintf("Pattern: %s", patterns[0]))
		}
		if category, ok := contextMap["category"].(string); ok && category != "" {
			parts = append(parts, fmt.Sprintf("Type: %s", category))
		}
		return strings.Join(parts, " | ")
	}
	return ""
}

// extractCleanCodeSnippet gets a meaningful code snippet
func extractCleanCodeSnippet(content string) string {
	lines := strings.Split(content, "\n")
	var cleanLines []string
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*") {
			cleanLines = append(cleanLines, trimmed)
			if len(cleanLines) >= 2 { // Show max 2 meaningful lines
				break
			}
		}
	}
	
	if len(cleanLines) == 0 {
		return "Code snippet available"
	}
	
	snippet := strings.Join(cleanLines, " | ")
	if len(snippet) > 80 {
		snippet = snippet[:77] + "..."
	}
	return snippet
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Metric and utility methods
func (casa *ContextAwareSearchAgentImpl) updateMetrics(startTime time.Time) {
	casa.metrics.QueriesHandled++
	casa.metrics.LastUsed = time.Now()
}

func (casa *ContextAwareSearchAgentImpl) updateSuccessMetrics(startTime time.Time, confidence float64, tokenUsage *models.TokenUsage) {
	duration := time.Since(startTime)
	casa.metrics.AverageResponseTime = (casa.metrics.AverageResponseTime + duration) / 2
	casa.metrics.AverageConfidence = (casa.metrics.AverageConfidence + confidence) / 2
	if casa.metrics.QueriesHandled > 0 {
		casa.metrics.SuccessRate = float64(casa.metrics.QueriesHandled-casa.metrics.ErrorCount) / float64(casa.metrics.QueriesHandled)
	}
}

func (casa *ContextAwareSearchAgentImpl) convertFiltersToStringMap(filters []SearchFilter) map[string]string {
	result := make(map[string]string)
	for _, filter := range filters {
		if str, ok := filter.Value.(string); ok {
			result[filter.Type] = str
		}
	}
	return result
}

// Placeholder methods for advanced functionality
func (casa *ContextAwareSearchAgentImpl) determineScopeFromQuery(input string) ContextAwareSearchAgentScope {
	return ContextAwareSearchAgentScope{
		IncludeTests: true,
		IncludeDocs:  true,
	}
}

func (casa *ContextAwareSearchAgentImpl) extractFilters(input string) []SearchFilter {
	return []SearchFilter{}
}

func (casa *ContextAwareSearchAgentImpl) calculateRequiredPrecision(input string) float64 { return 0.8 }

func (casa *ContextAwareSearchAgentImpl) selectPrimarySearchMethod(intent *ContextAwareSearchAgentIntent) SearchMethod {
	return SearchMethodSemantic
}

func (casa *ContextAwareSearchAgentImpl) selectSecondaryMethods(intent *ContextAwareSearchAgentIntent) []SearchMethod {
	return []SearchMethod{}
}

func (casa *ContextAwareSearchAgentImpl) buildContextLayers(ctx context.Context, intent *ContextAwareSearchAgentIntent) []ContextLayer {
	return []ContextLayer{}
}

func (casa *ContextAwareSearchAgentImpl) defineRankingFactors(intent *ContextAwareSearchAgentIntent) []AgentRankingFactor {
	return []AgentRankingFactor{}
}

func (casa *ContextAwareSearchAgentImpl) buildFilterChain(intent *ContextAwareSearchAgentIntent) []SearchFilter {
	return []SearchFilter{}
}

func (casa *ContextAwareSearchAgentImpl) calculateOptimalResultCount(intent *ContextAwareSearchAgentIntent) int {
	return 10
}

func (casa *ContextAwareSearchAgentImpl) performStructuralSearch(ctx context.Context, strategy *ContextAwareSearchAgentStrategy, query *models.Query) ([]*SearchResult, error) {
	return []*SearchResult{}, nil
}

func (casa *ContextAwareSearchAgentImpl) performDependencyAwareSearch(ctx context.Context, strategy *ContextAwareSearchAgentStrategy, query *models.Query) ([]*SearchResult, error) {
	return []*SearchResult{}, nil
}

func (casa *ContextAwareSearchAgentImpl) performUsagePatternSearch(ctx context.Context, strategy *ContextAwareSearchAgentStrategy, query *models.Query) ([]*SearchResult, error) {
	return []*SearchResult{}, nil
}

func (casa *ContextAwareSearchAgentImpl) fuseAndRankResults(results []*SearchResult, strategy *ContextAwareSearchAgentStrategy) []*SearchResult {
	return results
}

func (casa *ContextAwareSearchAgentImpl) gatherContextualInfo(ctx context.Context, result *SearchResult) map[string]interface{} {
	return map[string]interface{}{}
}

func (casa *ContextAwareSearchAgentImpl) findRelatedPatterns(ctx context.Context, result *SearchResult) []string {
	return []string{}
}

func (casa *ContextAwareSearchAgentImpl) findUsageExamples(ctx context.Context, result *SearchResult) []string {
	return []string{}
}

func (casa *ContextAwareSearchAgentImpl) analyzeDependencies(ctx context.Context, result *SearchResult) []string {
	return []string{}
}

func (casa *ContextAwareSearchAgentImpl) calculateQualityMetrics(result *SearchResult) map[string]float64 {
	return map[string]float64{}
}

func (casa *ContextAwareSearchAgentImpl) calculateContextualRelevance(result *SearchResult, intent *ContextAwareSearchAgentIntent) float64 {
	return 0.8
}

func (casa *ContextAwareSearchAgentImpl) calculateOverallConfidence(results []*EnhancedSearchResult) float64 {
	return 0.85
}

func (casa *ContextAwareSearchAgentImpl) inferDomain(input string) string     { return "general" }
func (casa *ContextAwareSearchAgentImpl) inferComplexity(input string) string { return "medium" }
func (casa *ContextAwareSearchAgentImpl) inferUrgency(input string) string    { return "normal" }

// enhanceSearchIntentWithMCP enhances search intent with MCP filesystem context
func (casa *ContextAwareSearchAgentImpl) enhanceSearchIntentWithMCP(intent *ContextAwareSearchAgentIntent, mcpContext *models.MCPContext) {
	// Initialize context map if nil
	if intent.Context == nil {
		intent.Context = make(map[string]interface{})
	}
	
	// Add file paths to file patterns
	if files, ok := mcpContext.Data["project_files"].([]map[string]interface{}); ok {
		for _, file := range files[:min(5, len(files))] { // Limit to 5 files
			if path, ok := file["path"].(string); ok {
				intent.FilePatterns = append(intent.FilePatterns, path)
			}
		}
	}
	
	// Add project structure context
	if structure, ok := mcpContext.Data["project_structure"].(map[string]interface{}); ok {
		intent.Context["mcp_structure"] = casa.extractStructureHints(structure)
	}
	
	// Add file count as context
	if count, ok := mcpContext.Data["file_count"].(int); ok {
		intent.Context["mcp_file_count"] = count
	}
}

// extractStructureHints extracts contextual hints from project structure
func (casa *ContextAwareSearchAgentImpl) extractStructureHints(structure map[string]interface{}) []string {
	hints := []string{}
	
	if _, hasInternal := structure["internal"]; hasInternal {
		hints = append(hints, "has_internal_architecture")
	}
	if _, hasCmd := structure["cmd"]; hasCmd {
		hints = append(hints, "has_cmd_structure")
	}
	if _, hasModels := structure["models"]; hasModels {
		hints = append(hints, "has_models_layer")
	}
	if _, hasMCP := structure["internal"].(map[string]interface{})["mcp"]; hasMCP {
		hints = append(hints, "has_mcp_integration")
	}
	
	return hints
}
