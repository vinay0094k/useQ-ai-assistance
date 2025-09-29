package agents

import (
	"time"
)

// =============================================================================
// SEARCH AGENT STRUCT (uses base types)
// =============================================================================

// SearchAgentStruct implements SearchAgent interface using base types
type SearchAgentStruct struct {
	Dependencies *AgentDependencies // From base
	Config       *SearchAgentConfig // Search-specific
	Metrics      *AgentMetrics      // From base
}

// =============================================================================
// SEARCH AGENT CONFIGURATION
// =============================================================================

// SearchAgentConfig holds search agent specific configuration
type SearchAgentConfig struct {
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
// SEARCH AGENT INTENT AND QUERY ANALYSIS
// =============================================================================

// SearchAgentIntent represents parsed search intent
type SearchAgentIntent struct {
	Query         string                 `json:"query"`
	SearchType    SearchAgentType        `json:"search_type"`
	Language      string                 `json:"language"`
	Keywords      []string               `json:"keywords"`
	FunctionNames []string               `json:"function_names"`
	TypeNames     []string               `json:"type_names"`
	FilePatterns  []string               `json:"file_patterns"`
	Filters       map[string]string      `json:"filters"`
	ExactMatch    bool                   `json:"exact_match"`
	CaseSensitive bool                   `json:"case_sensitive"`
	Scope         SearchAgentScope       `json:"scope"`
	Context       map[string]interface{} `json:"context"`
	Precision     float64                `json:"precision"`
}

// SearchAgentType represents different types of search
type SearchAgentType string

const (
	SearchAgentTypeGeneral   SearchAgentType = "general"
	SearchAgentTypeFunction  SearchAgentType = "function"
	SearchAgentTypeType      SearchAgentType = "type"
	SearchAgentTypeInterface SearchAgentType = "interface"
	SearchAgentTypeFile      SearchAgentType = "file"
	SearchAgentTypePackage   SearchAgentType = "package"
	SearchAgentTypeUsage     SearchAgentType = "usage"
	SearchAgentTypePattern   SearchAgentType = "pattern"
	SearchAgentTypeSemantic  SearchAgentType = "semantic"
	SearchAgentTypeKeyword   SearchAgentType = "keyword"
	SearchAgentTypeRegex     SearchAgentType = "regex"
)

// SearchAgentScope represents the scope of search
type SearchAgentScope struct {
	Files        []string   `json:"files,omitempty"`
	Directories  []string   `json:"directories,omitempty"`
	Packages     []string   `json:"packages,omitempty"`
	Languages    []string   `json:"languages,omitempty"`
	ExcludeFiles []string   `json:"exclude_files,omitempty"`
	IncludeTests bool       `json:"include_tests"`
	IncludeDocs  bool       `json:"include_docs"`
	TimeRange    *TimeRange `json:"time_range,omitempty"`
}

// =============================================================================
// SEARCH AGENT CONTEXT AND RESULTS
// =============================================================================

// SearchAgentContext holds context for search operations
type SearchAgentContext struct {
	Query           string                  `json:"query"`
	Intent          *SearchAgentIntent      `json:"intent"`
	Filters         map[string]interface{}  `json:"filters"`
	ScopeInfo       *SearchAgentScope       `json:"scope_info"`
	HistoryContext  []SearchAgentHistory    `json:"history_context"`
	UserPreferences *SearchAgentPreferences `json:"user_preferences"`
}

// SearchAgentHistory represents previous search context
type SearchAgentHistory struct {
	Query     string    `json:"query"`
	Results   int       `json:"results"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
}

// SearchAgentPreferences represents user search preferences
type SearchAgentPreferences struct {
	AgentPreferences
}

// SearchAgentResult represents an enhanced search result
type SearchAgentResult struct {
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

// SearchAgentEnhancedResult extends SearchAgentResult with context
type SearchAgentEnhancedResult struct {
	SearchResult    SearchAgentResult      `json:"search_result"`
	ContextualInfo  map[string]interface{} `json:"contextual_info"`
	RelatedPatterns []string               `json:"related_patterns"`
	UsageExamples   []string               `json:"usage_examples"`
	Dependencies    []string               `json:"dependencies"`
	QualityMetrics  map[string]float64     `json:"quality_metrics"`
	RelevanceScore  float64                `json:"relevance_score"`
}

// =============================================================================
// SEARCH AGENT STRATEGY
// =============================================================================

// SearchAgentStrategy defines intelligent search strategy
type SearchAgentStrategy struct {
	PrimaryMethod    SearchAgentMethod   `json:"primary_method"`
	SecondaryMethods []SearchAgentMethod `json:"secondary_methods"`
	ContextLayers    []SearchAgentLayer  `json:"context_layers"`
	RankingFactors   []SearchAgentFactor `json:"ranking_factors"`
	FilterChain      []SearchAgentFilter `json:"filter_chain"`
	MaxResults       int                 `json:"max_results"`
}

// SearchAgentMethod defines search methods
type SearchAgentMethod string

const (
	SearchAgentMethodSemantic   SearchAgentMethod = "semantic"
	SearchAgentMethodStructural SearchAgentMethod = "structural"
	SearchAgentMethodDependency SearchAgentMethod = "dependency"
	SearchAgentMethodUsage      SearchAgentMethod = "usage"
)

// SearchAgentFilter represents search filters
type SearchAgentFilter struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// SearchAgentLayer represents context layers
type SearchAgentLayer struct {
	Name   string                 `json:"name"`
	Weight float64                `json:"weight"`
	Data   map[string]interface{} `json:"data"`
}

// SearchAgentFactor represents ranking factors
type SearchAgentFactor struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
}

// SearchAgentScopeType defines search scope types
type SearchAgentScopeType string

const (
	SearchAgentScopeProject SearchAgentScopeType = "project"
	SearchAgentScopeFile    SearchAgentScopeType = "file"
	SearchAgentScopePackage SearchAgentScopeType = "package"
	SearchAgentScopeGlobal  SearchAgentScopeType = "global"
)

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================
