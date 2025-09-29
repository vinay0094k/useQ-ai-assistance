package agents

import (
	"time"
)

// =============================================================================
// CONTEXT-AWARE SEARCH AGENT CONFIGURATION
// =============================================================================

// SearchConfig holds search agent specific configuration (shared with regular search agent)
type ContextAwareSearchAgentConfig struct {
	AgentConfig
	MaxResults          int     `json:"max_results"`
	SimilarityThreshold float32 `json:"similarity_threshold"`
	EnableReranking     bool    `json:"enable_reranking"`
	IncludeContext      bool    `json:"include_context"`
	ExpandResults       bool    `json:"expand_results"`
	SemanticSearch      bool    `json:"semantic_search"`
	ExactMatchBonus     float32 `json:"exact_match_bonus"`
	FuzzySearch         bool    `json:"fuzzy_search"`
	RegexSearch         bool    `json:"regex_search"`
	HistoryEnabled      bool    `json:"history_enabled"`
	ResultCaching       bool    `json:"result_caching"`
}

// =============================================================================
// CONTEXT-AWARE SEARCH TYPES
// =============================================================================

// SearchIntent represents intelligent search intent
type ContextAwareSearchAgentIntent struct {
	Query         string                       `json:"query"`
	SearchType    SearchType                   `json:"search_type"`
	Language      string                       `json:"language"`
	Keywords      []string                     `json:"keywords"`
	FunctionNames []string                     `json:"function_names"`
	TypeNames     []string                     `json:"type_names"`
	FilePatterns  []string                     `json:"file_patterns"`
	Filters       map[string]string            `json:"filters"`
	ExactMatch    bool                         `json:"exact_match"`
	CaseSensitive bool                         `json:"case_sensitive"`
	Scope         ContextAwareSearchAgentScope `json:"scope"`
	Context       map[string]interface{}       `json:"context"`
	Precision     float64                      `json:"precision"`
}

// SearchType represents different types of search
type SearchType string

const (
	SearchTypeGeneral   SearchType = "general"
	SearchTypeFunction  SearchType = "function"
	SearchTypeType      SearchType = "type"
	SearchTypeInterface SearchType = "interface"
	SearchTypeFile      SearchType = "file"
	SearchTypePackage   SearchType = "package"
	SearchTypeUsage     SearchType = "usage"
	SearchTypePattern   SearchType = "pattern"
	SearchTypeSemantic  SearchType = "semantic"
	SearchTypeKeyword   SearchType = "keyword"
	SearchTypeRegex     SearchType = "regex"
)

// ContextAwareSearchAgentScope represents the scope of search
type ContextAwareSearchAgentScope struct {
	Files        []string   `json:"files,omitempty"`
	Directories  []string   `json:"directories,omitempty"`
	Packages     []string   `json:"packages,omitempty"`
	Languages    []string   `json:"languages,omitempty"`
	ExcludeFiles []string   `json:"exclude_files,omitempty"`
	IncludeTests bool       `json:"include_tests"`
	IncludeDocs  bool       `json:"include_docs"`
	TimeRange    *TimeRange `json:"time_range,omitempty"`
}

// SearchResult represents an enhanced search result
type SearchResult struct {
	File        string            `json:"file"`
	Function    string            `json:"function,omitempty"`
	Type        string            `json:"type,omitempty"`
	Line        int               `json:"line"`
	Score       float64           `json:"score"`
	Context     string            `json:"context"`
	Explanation string            `json:"explanation,omitempty"`
	Usage       []UsageExample    `json:"usage,omitempty"`
	ChunkType   string            `json:"chunk_type"`
	Language    string            `json:"language"`
	Package     string            `json:"package,omitempty"`
	Metadata    map[string]string `json:"metadata"`
}

// EnhancedSearchResult extends SearchResult with context-aware information
type EnhancedSearchResult struct {
	SearchResult    SearchResult           `json:"search_result"`
	ContextualInfo  map[string]interface{} `json:"contextual_info"`
	RelatedPatterns []string               `json:"related_patterns"`
	UsageExamples   []string               `json:"usage_examples"`
	Dependencies    []string               `json:"dependencies"`
	QualityMetrics  map[string]float64     `json:"quality_metrics"`
	RelevanceScore  float64                `json:"relevance_score"`
}

// =============================================================================
// INTELLIGENT SEARCH STRATEGY
// =============================================================================

// SearchStrategy defines intelligent search strategy for context-aware search
type ContextAwareSearchAgentStrategy struct {
	PrimaryMethod    SearchMethod         `json:"primary_method"`
	SecondaryMethods []SearchMethod       `json:"secondary_methods"`
	ContextLayers    []ContextLayer       `json:"context_layers"`
	RankingFactors   []AgentRankingFactor `json:"ranking_factors"`
	FilterChain      []SearchFilter       `json:"filter_chain"`
	MaxResults       int                  `json:"max_results"`
}

// SearchMethod defines search methods
type SearchMethod string

const (
	SearchMethodSemantic   SearchMethod = "semantic"
	SearchMethodStructural SearchMethod = "structural"
	SearchMethodDependency SearchMethod = "dependency"
	SearchMethodUsage      SearchMethod = "usage"
)

// SearchFilter represents search filters
type SearchFilter struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// ContextLayer represents context layers for enhanced search
type ContextLayer struct {
	Name   string                 `json:"name"`
	Weight float64                `json:"weight"`
	Data   map[string]interface{} `json:"data"`
}

// =============================================================================
// CONTEXT-AWARE SEARCH CONTEXT
// =============================================================================

// SearchContext holds context for search operations
type ContextAwareSearchAgentContext struct {
	Query           string                         `json:"query"`
	Intent          *ContextAwareSearchAgentIntent `json:"intent"`
	Filters         map[string]interface{}         `json:"filters"`
	ScopeInfo       *ContextAwareSearchAgentConfig `json:"scope_info"`
	HistoryContext  []SearchHistory                `json:"history_context"`
	UserPreferences *AgentPreferences              `json:"user_preferences"`
}

// SearchHistory represents previous search context
type SearchHistory struct {
	Query     string    `json:"query"`
	Results   int       `json:"results"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
}

// ContextAwareSearchAgentScopeType defines search scope types
type ContextAwareSearchAgentScopeType string

const (
	SearchScopeProject ContextAwareSearchAgentScopeType = "project"
	SearchScopeFile    ContextAwareSearchAgentScopeType = "file"
	SearchScopePackage ContextAwareSearchAgentScopeType = "package"
	SearchScopeGlobal  ContextAwareSearchAgentScopeType = "global"
)

// ContextAwareSearchAgentConfig creates a new search agent configuration
func NewSContextAwareSearchAgentConfig() *ContextAwareSearchAgentConfig {
	base := NewAgentConfig()
	return &ContextAwareSearchAgentConfig{
		AgentConfig:         *base,
		MaxResults:          10,   // Reduced for higher quality results
		SimilarityThreshold: 0.15, // Higher threshold for better relevance
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

// NewAgentConfig creates a new agent configuration with defaults
func NewContextAwareAgentConfig() *AgentConfig {
	return &AgentConfig{
		MaxTokens:          4000,
		Temperature:        0.3,
		MaxRetries:         3,
		Timeout:            30 * time.Second,
		EnableLogging:      true,
		LogLevel:           "info",
		CacheEnabled:       true,
		CacheTTL:           5 * time.Minute,
		PreferencesEnabled: true,
		ContextWindow:      8192,
		StreamingEnabled:   true,
		ParallelProcessing: false,
		MaxConcurrency:     1,
		CustomPrompts:      make(map[string]string),
		ModelPreferences:   make(map[string]interface{}),
	}
}
