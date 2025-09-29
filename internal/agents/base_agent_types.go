package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/useq-ai-assistant/internal/llm"
	"github.com/yourusername/useq-ai-assistant/internal/vectordb"
	"github.com/yourusername/useq-ai-assistant/models"
	"github.com/yourusername/useq-ai-assistant/storage"
)

// Agent is the common base struct for all agents
type Agent struct {
	Dependencies *AgentDependencies
	Config       interface{} // each agent can store its config (search, coding, etc.)
	Metrics      *AgentMetrics
}

// --------------------------------------------------------------------
// Common supporting types for all agents
// --------------------------------------------------------------------

// AgentInterface defines the common behavior every agent must implement
type AgentInterface interface {
	// Capability & specialization
	GetCapabilities() AgentCapabilities
	GetSpecialization() AgentSpecialization

	// Query handling
	CanHandle(ctx context.Context, query *models.Query) (bool, float64)
	GetConfidenceScore(ctx context.Context, query *models.Query) float64
	ValidateQuery(query *models.Query) error
	Process(ctx context.Context, query *models.Query) (*models.Response, error)

	// Metrics
	GetMetrics() AgentMetrics
}

// AgentCapabilities describes what an agent can do
type AgentCapabilities struct {
	CanGenerateCode    bool
	CanSearchCode      bool
	CanAnalyzeCode     bool
	CanDebugCode       bool
	CanWriteTests      bool
	CanWriteDocs       bool
	CanReviewCode      bool
	SupportedLanguages []string
	MaxComplexity      int
	RequiresContext    bool
}

// AgentMetrics tracks usage/performance statistics
type AgentMetrics struct {
	QueriesHandled      int
	SuccessRate         float64
	AverageResponseTime time.Duration
	AverageConfidence   float64
	TokensUsed          int64
	TotalCost           float64
	LastUsed            time.Time
	ErrorCount          int
}

// AgentType identifies the type of agent
type AgentType string

const (
	AgentTypeCoding AgentType = "coding"
	AgentTypeSearch AgentType = "search"
	AgentTypeOther  AgentType = "other"
)

// AgentConfig represents common configuration for all agents
type AgentConfig struct {
	MaxTokens          int                    `json:"max_tokens"`
	Temperature        float64                `json:"temperature"`
	MaxRetries         int                    `json:"max_retries"`
	Timeout            time.Duration          `json:"timeout"`
	EnableLogging      bool                   `json:"enable_logging"`
	LogLevel           string                 `json:"log_level"`
	CacheEnabled       bool                   `json:"cache_enabled"`
	CacheTTL           time.Duration          `json:"cache_ttl"`
	PreferencesEnabled bool                   `json:"preferences_enabled"`
	ContextWindow      int                    `json:"context_window"`
	StreamingEnabled   bool                   `json:"streaming_enabled"`
	ParallelProcessing bool                   `json:"parallel_processing"`
	MaxConcurrency     int                    `json:"max_concurrency"`
	CustomPrompts      map[string]string      `json:"custom_prompts"`
	ModelPreferences   map[string]interface{} `json:"model_preferences"`
}

// AgentDependencies holds common dependencies for agents
type AgentDependencies struct {
	LLMManager *llm.Manager               `json:"-"`
	VectorDB   *vectordb.QdrantClient     `json:"-"`
	Storage    *storage.SQLiteDB          `json:"-"`
	Embedder   *vectordb.EmbeddingService `json:"-"`
	Logger     Logger                     `json:"-"`
	Metrics    MetricsCollector           `json:"-"`
	Cache      CacheManager               `json:"-"`
	MCPClient  MCPClientInterface         `json:"-"`
}

// MCPClientInterface defines the interface for MCP client operations
type MCPClientInterface interface {
	ProcessQuery(ctx context.Context, query *models.Query) (*models.MCPContext, error)
	GetCacheStats() map[string]interface{}
	InvalidateCache(projectPath string)
}

// AgentSpecialization represents specific areas of expertise
type AgentSpecialization struct {
	Type        AgentType `json:"type"`
	Languages   []string  `json:"languages"`
	Frameworks  []string  `json:"frameworks"`
	Domains     []string  `json:"domains"`
	Complexity  int       `json:"complexity"`
	Description string    `json:"description"`
}

// Logger interface for agent logging
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})
}

// MetricsCollector interface for collecting agent metrics
type MetricsCollector interface {
	IncrementCounter(name string, tags map[string]string)
	RecordTimer(name string, duration time.Duration, tags map[string]string)
	RecordGauge(name string, value float64, tags map[string]string)
	RecordHistogram(name string, value float64, tags map[string]string)
}

// CacheManager interface for caching
type CacheManager interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration) error
	Delete(key string) error
	Clear() error
	Stats() CacheStats
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits   int64 `json:"hits"`
	Misses int64 `json:"misses"`
	Size   int64 `json:"size"`
}

// UsageExample shows how code is used
type UsageExample struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Context     string `json:"context"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Type        string `json:"type"` // call, instantiation, inheritance, etc.
}

// TimeRange represents a time range for searches
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// RankingFactor is a single ranking signal
type AgentRankingFactor struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
}

// AgentScopeType enumerates scope target types
type AgentScopeAType string

const (
	AgentScopeProject AgentScopeAType = "project"
	AgentScopeFile    AgentScopeAType = "file"
	AgentScopePackage AgentScopeAType = "package"
	AgentScopeGlobal  AgentScopeAType = "global"
)

// AgentPreferences represents user search preferences
type AgentPreferences struct {
	PreferredLanguages  []string `json:"preferred_languages"`
	MaxResults          int      `json:"max_results"`
	SortBy              string   `json:"sort_by"`
	GroupBy             string   `json:"group_by"`
	IncludePrivate      bool     `json:"include_private"`
	ShowLineNumbers     bool     `json:"show_line_numbers"`
	HighlightMatches    bool     `json:"highlight_matches"`
	SimilarityThreshold float64  `json:"similarity_threshold"`
}

// AgentSearchConfig represents search configuration
type AgentSearchConfig struct {
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

// AgentSearchType represents different types of search
type AgentSearchType string

const (
	AgentSearchTypeGeneral   AgentSearchType = "general"
	AgentSearchTypeFunction  AgentSearchType = "function"
	AgentSearchTypeType      AgentSearchType = "type"
	AgentSearchTypeInterface AgentSearchType = "interface"
	AgentSearchTypeFile      AgentSearchType = "file"
	AgentSearchTypePackage   AgentSearchType = "package"
	AgentSearchTypeUsage     AgentSearchType = "usage"
	AgentSearchTypePattern   AgentSearchType = "pattern"
	AgentSearchTypeSemantic  AgentSearchType = "semantic"
	AgentSearchTypeKeyword   AgentSearchType = "keyword"
	AgentSearchTypeRegex     AgentSearchType = "regex"
)

// AgentSearchScope represents the scope of search
type AgentSearchScope struct {
	Files        []string   `json:"files,omitempty"`
	Directories  []string   `json:"directories,omitempty"`
	Packages     []string   `json:"packages,omitempty"`
	Languages    []string   `json:"languages,omitempty"`
	ExcludeFiles []string   `json:"exclude_files,omitempty"`
	IncludeTests bool       `json:"include_tests"`
	IncludeDocs  bool       `json:"include_docs"`
	TimeRange    *TimeRange `json:"time_range,omitempty"`
}

// AgentSearchMethod defines search methods
type AgentSearchMethod string

const (
	AgentSearchMethodSemantic   AgentSearchMethod = "semantic"
	AgentSearchMethodStructural AgentSearchMethod = "structural"
	AgentSearchMethodDependency AgentSearchMethod = "dependency"
	AgentSearchMethodUsage      AgentSearchMethod = "usage"
)

// AgentSearchFilter represents search filters
type AgentSearchFilter struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// AgentContextLayer represents context layers
type AgentContextLayer struct {
	Name   string                 `json:"name"`
	Weight float64                `json:"weight"`
	Data   map[string]interface{} `json:"data"`
}

// AgentSearchResult represents a search result
type AgentSearchResult struct {
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

// AgentSearchIntent represents parsed search intent
type AgentSearchIntent struct {
	Query         string                 `json:"query"`
	SearchType    AgentSearchType        `json:"search_type"`
	Language      string                 `json:"language"`
	Keywords      []string               `json:"keywords"`
	FunctionNames []string               `json:"function_names"`
	TypeNames     []string               `json:"type_names"`
	FilePatterns  []string               `json:"file_patterns"`
	Filters       map[string]string      `json:"filters"`
	ExactMatch    bool                   `json:"exact_match"`
	CaseSensitive bool                   `json:"case_sensitive"`
	Scope         AgentSearchScope       `json:"scope"`
	Context       map[string]interface{} `json:"context"`
	Precision     float64                `json:"precision"`
}

// AgentCodeAnalysis represents code analysis results
type AgentCodeAnalysis struct {
	Language     string                 `json:"language"`
	Complexity   float64                `json:"complexity"`
	QualityScore float64                `json:"quality_score"`
	Issues       []string               `json:"issues"`
	Suggestions  []string               `json:"suggestions"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// AgentSemanticContext represents semantic context
type AgentSemanticContext struct {
	Keywords   []string               `json:"keywords"`
	Concepts   []string               `json:"concepts"`
	Relations  map[string]interface{} `json:"relations"`
	Confidence float64                `json:"confidence"`
}

// AgentArchitecturalContext represents architectural context
type AgentArchitecturalContext struct {
	Patterns     []string               `json:"patterns"`
	Components   []string               `json:"components"`
	Dependencies []string               `json:"dependencies"`
	Structure    map[string]interface{} `json:"structure"`
}

// AgentParameter represents function/method parameters
type AgentParameter struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Optional bool   `json:"optional"`
	Default  string `json:"default,omitempty"`
}

// AgentGenericParam represents generic parameters
type AgentGenericParam struct {
	Name        string   `json:"name"`
	Constraints []string `json:"constraints,omitempty"`
}

// AgentVisibility represents visibility levels
type AgentVisibility string

const (
	AgentVisibilityPublic    AgentVisibility = "public"
	AgentVisibilityPrivate   AgentVisibility = "private"
	AgentVisibilityProtected AgentVisibility = "protected"
	AgentVisibilityInternal  AgentVisibility = "internal"
)

// AgentReceiver represents method receivers
type AgentReceiver struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ProcessingStrategy defines intelligent processing strategy
type ProcessingStrategy struct {
	Type               ProcessingStrategyType `json:"type"`
	IntelligenceLayers []string               `json:"intelligence_layers"`
	GenerationMode     string                 `json:"generation_mode"`
	AnalysisMode       string                 `json:"analysis_mode"`
	QualityThreshold   float64                `json:"quality_threshold"`
	PerformanceFocus   bool                   `json:"performance_focus"`
	SecurityFocus      bool                   `json:"security_focus"`
}

// ProcessingStrategyType defines types of processing strategies
type ProcessingStrategyType string

const (
	StrategyDeepAnalysis          ProcessingStrategyType = "deep_analysis"
	StrategyIntelligentGeneration ProcessingStrategyType = "intelligent_generation"
	StrategyOptimization          ProcessingStrategyType = "optimization"
	StrategyRefactoring           ProcessingStrategyType = "refactoring"
	StrategyArchitecturalDesign   ProcessingStrategyType = "architectural_design"
	StrategyHybrid                ProcessingStrategyType = "hybrid"
)

// NewAgentConfig creates a new agent configuration with defaults
func NewAgentConfig() *AgentConfig {
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

// --------------------------------------------------------------------
// Helper methods to be used across different agents
// --------------------------------------------------------------------
// CalculateConfidence calculates confidence based on multiple factors
func CalculateConfidence(factors map[string]float64) float64 {
	if len(factors) == 0 {
		return 0.0
	}

	total := 0.0
	for _, value := range factors {
		total += value
	}

	return total / float64(len(factors))
}

// ValidateQuery performs basic validation on a query
func ValidateQuery(query *models.Query) error {
	if query == nil {
		return fmt.Errorf("query cannot be nil")
	}
	if strings.TrimSpace(query.UserInput) == "" {
		return fmt.Errorf("query input cannot be empty")
	}
	if query.Language == "" {
		query.Language = "go" // Default to Go
	}
	return nil
}

// Helper function to safely extract string from map
func GetString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// Helper function to safely extract int from map
func GetInt(m map[string]interface{}, key string) int {
	if val, ok := m[key].(int); ok {
		return val
	}
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	return 0
}

// Helper function to safely extract float64 from map
func GetFloat64(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	if val, ok := m[key].(int); ok {
		return float64(val)
	}
	return 0.0
}

// Helper function to safely extract bool from map
func GetBool(m map[string]interface{}, key string) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return false
}

// ConvertToStringMap converts a map[string]interface{} to map[string]string
func ConvertToStringMap(m map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		if str, ok := v.(string); ok {
			result[k] = str
		} else {
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result
}
