package mcp

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// QueryTier represents the 3-tier classification system
type QueryTier string

const (
	TierSimple  QueryTier = "simple"  // MCP Direct - $0, <100ms
	TierMedium  QueryTier = "medium"  // MCP + Vector - $0, <500ms  
	TierComplex QueryTier = "complex" // Full LLM Pipeline - $0.01-0.03, 1-3s
)

// QueryClassifier implements the 3-tier classification system
type QueryClassifier struct {
	simplePatterns  []*ClassificationPattern
	mediumPatterns  []*ClassificationPattern
	complexPatterns []*ClassificationPattern
	stats           *ClassificationStats
}

// ClassificationPattern represents a pattern for query classification
type ClassificationPattern struct {
	Name        string         `json:"name"`
	Regex       *regexp.Regexp `json:"regex"`
	Keywords    []string       `json:"keywords"`
	Weight      float64        `json:"weight"`
	Description string         `json:"description"`
}

// ClassificationResult represents the result of query classification
type ClassificationResult struct {
	Tier                QueryTier              `json:"tier"`
	Confidence          float64                `json:"confidence"`
	MatchedPatterns     []string               `json:"matched_patterns"`
	EstimatedCost       float64                `json:"estimated_cost"`
	EstimatedTime       time.Duration          `json:"estimated_time"`
	RequiredOperations  []string               `json:"required_operations"`
	SkipLLM            bool                   `json:"skip_llm"`
	ProcessingStrategy  ProcessingStrategy     `json:"processing_strategy"`
	Reasoning          string                 `json:"reasoning"`
}

// ProcessingStrategy defines how to process the query
type ProcessingStrategy struct {
	Type        string   `json:"type"`        // "mcp_direct", "mcp_vector", "full_pipeline"
	Operations  []string `json:"operations"`  // Required MCP operations
	UseVector   bool     `json:"use_vector"`  // Whether to use vector search
	UseLLM      bool     `json:"use_llm"`     // Whether to call LLM
	CacheKey    string   `json:"cache_key"`   // Cache key for results
}

// ClassificationStats tracks classification performance
type ClassificationStats struct {
	TotalQueries    int                    `json:"total_queries"`
	TierBreakdown   map[QueryTier]int      `json:"tier_breakdown"`
	CostSavings     float64                `json:"cost_savings"`
	TimeSavings     time.Duration          `json:"time_savings"`
	AccuracyRate    float64                `json:"accuracy_rate"`
	LastUpdated     time.Time              `json:"last_updated"`
}

// NewQueryClassifier creates a new 3-tier query classifier
func NewQueryClassifier() *QueryClassifier {
	classifier := &QueryClassifier{
		stats: &ClassificationStats{
			TierBreakdown: make(map[QueryTier]int),
			LastUpdated:   time.Now(),
		},
	}
	
	classifier.initializePatterns()
	return classifier
}

// ClassifyQuery performs 3-tier classification with decision tree
func (qc *QueryClassifier) ClassifyQuery(ctx context.Context, query *models.Query) (*ClassificationResult, error) {
	input := strings.ToLower(strings.TrimSpace(query.UserInput))
	
	// DECISION TREE: Check in order of specificity
	
	// 1. Check for COMPLEX patterns first (most specific)
	if result := qc.checkComplexPatterns(input, query); result != nil {
		qc.updateStats(TierComplex)
		return result, nil
	}
	
	// 2. Check for SIMPLE patterns (high confidence, specific actions)
	if result := qc.checkSimplePatterns(input, query); result != nil {
		qc.updateStats(TierSimple)
		return result, nil
	}
	
	// 3. Check for MEDIUM patterns (search/lookup without explanation)
	if result := qc.checkMediumPatterns(input, query); result != nil {
		qc.updateStats(TierMedium)
		return result, nil
	}
	
	// 4. DEFAULT: Route to Tier 2 (safer than assuming complex)
	result := &ClassificationResult{
		Tier:               TierMedium,
		Confidence:         0.5,
		MatchedPatterns:    []string{"default_medium"},
		EstimatedCost:      0.0,
		EstimatedTime:      500 * time.Millisecond,
		RequiredOperations: []string{"filesystem_search", "vector_search"},
		SkipLLM:           true,
		ProcessingStrategy: ProcessingStrategy{
			Type:       "mcp_vector",
			Operations: []string{"filesystem_search", "vector_search"},
			UseVector:  true,
			UseLLM:     false,
		},
		Reasoning: "Default routing to medium tier for safety",
	}
	
	qc.updateStats(TierMedium)
	return result, nil
}

// initializePatterns sets up the classification patterns
func (qc *QueryClassifier) initializePatterns() {
	// TIER 1: SIMPLE PATTERNS (80% of traffic)
	qc.simplePatterns = []*ClassificationPattern{
		{
			Name:        "file_operations",
			Regex:       regexp.MustCompile(`^(list|show|display|get|read|cat|open|ls)\s`),
			Keywords:    []string{"list", "show", "display", "get", "read", "cat", "open", "ls"},
			Weight:      0.95,
			Description: "Direct file operations",
		},
		{
			Name:        "directory_operations", 
			Regex:       regexp.MustCompile(`(what files|files in|show directory|directory|folder|tree|pwd)`),
			Keywords:    []string{"what files", "files in", "directory", "folder", "tree", "pwd"},
			Weight:      0.9,
			Description: "Directory listing operations",
		},
		{
			Name:        "system_status",
			Regex:       regexp.MustCompile(`^(memory|status|health|system info|cpu|disk|usage)$`),
			Keywords:    []string{"memory", "status", "health", "system", "cpu", "disk", "usage"},
			Weight:      0.95,
			Description: "System status checks",
		},
		{
			Name:        "direct_file_reads",
			Regex:       regexp.MustCompile(`(show me|read|cat|open)\s+[\w./]+\.(go|yaml|json|md)`),
			Keywords:    []string{"show me", "read", "cat", "open"},
			Weight:      0.9,
			Description: "Direct file content requests",
		},
	}
	
	// TIER 2: MEDIUM PATTERNS (15% of traffic)
	qc.mediumPatterns = []*ClassificationPattern{
		{
			Name:        "search_operations",
			Regex:       regexp.MustCompile(`^(find|search|locate|where is)\s`),
			Keywords:    []string{"find", "search", "locate", "where is"},
			Weight:      0.85,
			Description: "Search operations without explanation",
		},
		{
			Name:        "code_lookups",
			Regex:       regexp.MustCompile(`(show all|find all|all functions|all methods|all structs|all handlers)`),
			Keywords:    []string{"show all", "find all", "all functions", "all methods", "all structs"},
			Weight:      0.8,
			Description: "Code element lookups",
		},
		{
			Name:        "pattern_matching",
			Regex:       regexp.MustCompile(`(functions that|files containing|structs with|methods in)`),
			Keywords:    []string{"functions that", "files containing", "structs with", "methods in"},
			Weight:      0.8,
			Description: "Pattern-based code matching",
		},
		{
			Name:        "simple_counts",
			Regex:       regexp.MustCompile(`^(how many|count|number of)\s`),
			Keywords:    []string{"how many", "count", "number of"},
			Weight:      0.85,
			Description: "Counting operations",
		},
	}
	
	// TIER 3: COMPLEX PATTERNS (5% of traffic)
	qc.complexPatterns = []*ClassificationPattern{
		{
			Name:        "explanation_requests",
			Regex:       regexp.MustCompile(`(explain|describe|how does|what is|tell me about|walk through)`),
			Keywords:    []string{"explain", "describe", "how does", "what is", "tell me about", "walk through"},
			Weight:      0.95,
			Description: "Explanation and understanding requests",
		},
		{
			Name:        "analysis_requests",
			Regex:       regexp.MustCompile(`(analyze|review|improve|refactor|optimize|suggest|audit)`),
			Keywords:    []string{"analyze", "review", "improve", "refactor", "optimize", "suggest", "audit"},
			Weight:      0.9,
			Description: "Code analysis and improvement requests",
		},
		{
			Name:        "generation_requests",
			Regex:       regexp.MustCompile(`(create|generate|write|add|implement|build|make)\s`),
			Keywords:    []string{"create", "generate", "write", "add", "implement", "build", "make"},
			Weight:      0.9,
			Description: "Code generation requests",
		},
		{
			Name:        "architecture_queries",
			Regex:       regexp.MustCompile(`(architecture|design|flow|pattern|structure|component|system)`),
			Keywords:    []string{"architecture", "design", "flow", "pattern", "structure", "component", "system"},
			Weight:      0.85,
			Description: "Architectural understanding requests",
		},
		{
			Name:        "multi_step_queries",
			Regex:       regexp.MustCompile(`\s(and|then|also|plus|additionally)\s`),
			Keywords:    []string{" and ", " then ", " also ", " plus ", " additionally "},
			Weight:      0.8,
			Description: "Multi-step or compound queries",
		},
	}
}

// checkSimplePatterns checks for Tier 1 patterns
func (qc *QueryClassifier) checkSimplePatterns(input string, query *models.Query) *ClassificationResult {
	for _, pattern := range qc.simplePatterns {
		if qc.matchesPattern(input, pattern) {
			return &ClassificationResult{
				Tier:               TierSimple,
				Confidence:         pattern.Weight,
				MatchedPatterns:    []string{pattern.Name},
				EstimatedCost:      0.0,
				EstimatedTime:      100 * time.Millisecond,
				RequiredOperations: qc.getSimpleOperations(pattern.Name),
				SkipLLM:           true,
				ProcessingStrategy: ProcessingStrategy{
					Type:       "mcp_direct",
					Operations: qc.getSimpleOperations(pattern.Name),
					UseVector:  false,
					UseLLM:     false,
					CacheKey:   qc.generateCacheKey(input, "simple"),
				},
				Reasoning: fmt.Sprintf("Simple %s operation - direct MCP execution", pattern.Name),
			}
		}
	}
	return nil
}

// checkMediumPatterns checks for Tier 2 patterns
func (qc *QueryClassifier) checkMediumPatterns(input string, query *models.Query) *ClassificationResult {
	for _, pattern := range qc.mediumPatterns {
		if qc.matchesPattern(input, pattern) {
			return &ClassificationResult{
				Tier:               TierMedium,
				Confidence:         pattern.Weight,
				MatchedPatterns:    []string{pattern.Name},
				EstimatedCost:      0.0,
				EstimatedTime:      500 * time.Millisecond,
				RequiredOperations: qc.getMediumOperations(pattern.Name),
				SkipLLM:           true,
				ProcessingStrategy: ProcessingStrategy{
					Type:       "mcp_vector",
					Operations: qc.getMediumOperations(pattern.Name),
					UseVector:  true,
					UseLLM:     false,
					CacheKey:   qc.generateCacheKey(input, "medium"),
				},
				Reasoning: fmt.Sprintf("Search operation - MCP + vector search without LLM synthesis"),
			}
		}
	}
	return nil
}

// checkComplexPatterns checks for Tier 3 patterns
func (qc *QueryClassifier) checkComplexPatterns(input string, query *models.Query) *ClassificationResult {
	for _, pattern := range qc.complexPatterns {
		if qc.matchesPattern(input, pattern) {
			return &ClassificationResult{
				Tier:               TierComplex,
				Confidence:         pattern.Weight,
				MatchedPatterns:    []string{pattern.Name},
				EstimatedCost:      qc.estimateLLMCost(input),
				EstimatedTime:      qc.estimateProcessingTime(input),
				RequiredOperations: qc.getComplexOperations(pattern.Name),
				SkipLLM:           false,
				ProcessingStrategy: ProcessingStrategy{
					Type:       "full_pipeline",
					Operations: qc.getComplexOperations(pattern.Name),
					UseVector:  true,
					UseLLM:     true,
					CacheKey:   qc.generateCacheKey(input, "complex"),
				},
				Reasoning: fmt.Sprintf("Complex %s request - requires LLM synthesis", pattern.Name),
			}
		}
	}
	return nil
}

// matchesPattern checks if input matches a classification pattern
func (qc *QueryClassifier) matchesPattern(input string, pattern *ClassificationPattern) bool {
	// Check regex match
	if pattern.Regex != nil && pattern.Regex.MatchString(input) {
		return true
	}
	
	// Check keyword matches
	matchCount := 0
	for _, keyword := range pattern.Keywords {
		if strings.Contains(input, keyword) {
			matchCount++
		}
	}
	
	// Require at least one keyword match
	return matchCount > 0
}

// getSimpleOperations returns operations for simple queries
func (qc *QueryClassifier) getSimpleOperations(patternName string) []string {
	switch patternName {
	case "file_operations":
		return []string{"filesystem_list"}
	case "directory_operations":
		return []string{"filesystem_tree"}
	case "system_status":
		return []string{"system_info"}
	case "direct_file_reads":
		return []string{"filesystem_read"}
	default:
		return []string{"filesystem_list"}
	}
}

// getMediumOperations returns operations for medium queries
func (qc *QueryClassifier) getMediumOperations(patternName string) []string {
	switch patternName {
	case "search_operations":
		return []string{"filesystem_search", "vector_search"}
	case "code_lookups":
		return []string{"filesystem_search", "vector_search", "code_analysis"}
	case "pattern_matching":
		return []string{"vector_search", "pattern_analysis"}
	case "simple_counts":
		return []string{"filesystem_count"}
	default:
		return []string{"filesystem_search", "vector_search"}
	}
}

// getComplexOperations returns operations for complex queries
func (qc *QueryClassifier) getComplexOperations(patternName string) []string {
	switch patternName {
	case "explanation_requests":
		return []string{"filesystem_structure", "vector_search", "code_analysis", "dependency_mapping"}
	case "analysis_requests":
		return []string{"code_analysis", "vector_search", "pattern_analysis", "quality_analysis"}
	case "generation_requests":
		return []string{"pattern_search", "vector_search", "code_examples", "project_conventions"}
	case "architecture_queries":
		return []string{"filesystem_structure", "dependency_mapping", "architecture_analysis"}
	case "multi_step_queries":
		return []string{"intent_decomposition", "multi_step_planning", "context_synthesis"}
	default:
		return []string{"filesystem_structure", "vector_search", "llm_generation"}
	}
}

// estimateLLMCost estimates the cost for LLM processing
func (qc *QueryClassifier) estimateLLMCost(input string) float64 {
	// Estimate based on input length and expected response
	inputTokens := len(input) / 4  // Rough token estimation
	outputTokens := 500            // Average response length
	
	// OpenAI GPT-4 pricing: $0.01 input, $0.03 output per 1K tokens
	inputCost := float64(inputTokens) / 1000.0 * 0.01
	outputCost := float64(outputTokens) / 1000.0 * 0.03
	
	return inputCost + outputCost
}

// estimateProcessingTime estimates processing time
func (qc *QueryClassifier) estimateProcessingTime(input string) time.Duration {
	baseTime := 1 * time.Second
	
	// Add time for complexity
	if strings.Contains(input, "architecture") || strings.Contains(input, "explain") {
		baseTime += 1 * time.Second
	}
	
	if strings.Contains(input, "analyze") || strings.Contains(input, "review") {
		baseTime += 500 * time.Millisecond
	}
	
	return baseTime
}

// generateCacheKey generates cache key for results
func (qc *QueryClassifier) generateCacheKey(input, tier string) string {
	// Simple hash-based cache key
	words := strings.Fields(input)
	if len(words) > 3 {
		words = words[:3]
	}
	return fmt.Sprintf("%s_%s_%s", tier, strings.Join(words, "_"), "cache")
}

// updateStats updates classification statistics
func (qc *QueryClassifier) updateStats(tier QueryTier) {
	qc.stats.TotalQueries++
	qc.stats.TierBreakdown[tier]++
	qc.stats.LastUpdated = time.Now()
	
	// Calculate cost savings (compared to routing everything to LLM)
	simpleCount := qc.stats.TierBreakdown[TierSimple]
	mediumCount := qc.stats.TierBreakdown[TierMedium]
	complexCount := qc.stats.TierBreakdown[TierComplex]
	
	// Cost if everything went to LLM: $0.02 average per query
	totalCostIfAllLLM := float64(qc.stats.TotalQueries) * 0.02
	
	// Actual cost: only complex queries use LLM
	actualCost := float64(complexCount) * 0.02
	
	qc.stats.CostSavings = totalCostIfAllLLM - actualCost
}

// GetStats returns current classification statistics
func (qc *QueryClassifier) GetStats() *ClassificationStats {
	return qc.stats
}

// PrintStats prints classification statistics
func (qc *QueryClassifier) PrintStats() {
	fmt.Printf("\n📊 Query Classification Statistics:\n")
	fmt.Printf("├─ Total Queries: %d\n", qc.stats.TotalQueries)
	fmt.Printf("├─ Simple (Tier 1): %d (%.1f%%)\n", 
		qc.stats.TierBreakdown[TierSimple],
		float64(qc.stats.TierBreakdown[TierSimple])/float64(qc.stats.TotalQueries)*100)
	fmt.Printf("├─ Medium (Tier 2): %d (%.1f%%)\n",
		qc.stats.TierBreakdown[TierMedium], 
		float64(qc.stats.TierBreakdown[TierMedium])/float64(qc.stats.TotalQueries)*100)
	fmt.Printf("├─ Complex (Tier 3): %d (%.1f%%)\n",
		qc.stats.TierBreakdown[TierComplex],
		float64(qc.stats.TierBreakdown[TierComplex])/float64(qc.stats.TotalQueries)*100)
	fmt.Printf("└─ Cost Savings: $%.4f (%.1f%% reduction)\n",
		qc.stats.CostSavings,
		qc.stats.CostSavings/(float64(qc.stats.TotalQueries)*0.02)*100)
}