package agents

import (
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// =============================================================================
// MANAGER AGENT ROUTING TYPES
// =============================================================================

// RoutingDecision tracks routing decisions for learning and optimization
type RoutingDecision struct {
	QueryID       string    `json:"query_id"`
	Intent        string    `json:"intent"`
	SelectedAgent string    `json:"selected_agent"`
	Confidence    float64   `json:"confidence"`
	Success       bool      `json:"success"`
	Timestamp     time.Time `json:"timestamp"`
}

// RoutingAnalysis represents query analysis for intelligent agent routing
type RoutingAnalysis struct {
	PrimaryIntent        string   `json:"primary_intent"`
	SecondaryIntents     []string `json:"secondary_intents"`
	Complexity           float64  `json:"complexity"`
	Domain               string   `json:"domain"`
	RequiredCapabilities []string `json:"required_capabilities"`
	ContextNeeds         float64  `json:"context_needs"`
	UrgencyLevel         string   `json:"urgency_level"`
}

// =============================================================================
// QUERY ANALYSIS AND INTENT
// =============================================================================

// QueryIntent represents the parsed intent from a user query
type QueryIntent struct {
	Type         IntentType             `json:"type"`
	Confidence   float64                `json:"confidence"`
	Language     string                 `json:"language"`
	Keywords     []string               `json:"keywords"`
	Entities     []ExtractedEntity      `json:"entities"`
	Context      map[string]interface{} `json:"context"`
	Constraints  []string               `json:"constraints"`
	Requirements []string               `json:"requirements"`
}

// IntentType represents different types of user intents
type IntentType string

const (
	IntentTypeGenerate IntentType = "generate"
	IntentTypeSearch   IntentType = "search"
	IntentTypeExplain  IntentType = "explain"
	IntentTypeDebug    IntentType = "debug"
	IntentTypeTest     IntentType = "test"
	IntentTypeDocument IntentType = "document"
	IntentTypeReview   IntentType = "review"
	IntentTypeRefactor IntentType = "refactor"
	IntentTypeOptimize IntentType = "optimize"
	IntentTypeAnalyze  IntentType = "analyze"
)

// ExtractedEntity represents an entity extracted from the query
type ExtractedEntity struct {
	Type       EntityType `json:"type"`
	Value      string     `json:"value"`
	Confidence float64    `json:"confidence"`
	Context    string     `json:"context"`
}

// EntityType represents different types of entities
type EntityType string

const (
	EntityTypeFunction  EntityType = "function"
	EntityTypeType      EntityType = "type"
	EntityTypeVariable  EntityType = "variable"
	EntityTypeFile      EntityType = "file"
	EntityTypePackage   EntityType = "package"
	EntityTypeKeyword   EntityType = "keyword"
	EntityTypeFramework EntityType = "framework"
	EntityTypeLibrary   EntityType = "library"
)

// =============================================================================
// AGENT CAPABILITIES AND TYPES
// =============================================================================

// =============================================================================
// INTELLIGENT ROUTING STRATEGY
// =============================================================================

// RoutingStrategy defines how the manager routes queries to agents
type RoutingStrategy struct {
	PrimaryStrategy     RoutingMethod   `json:"primary_strategy"`
	FallbackStrategies  []RoutingMethod `json:"fallback_strategies"`
	ConfidenceThreshold float64         `json:"confidence_threshold"`
	LearningEnabled     bool            `json:"learning_enabled"`
	HistoryWeight       float64         `json:"history_weight"`
	Factors             []RoutingFactor `json:"factors"`
	AgentPriorities     map[string]int  `json:"agent_priorities"`
}

// RoutingMethod represents different routing methods
type RoutingMethod string

const (
	RoutingMethodCapability RoutingMethod = "capability_based"
	RoutingMethodIntent     RoutingMethod = "intent_based"
	RoutingMethodLearning   RoutingMethod = "learning_based"
	RoutingMethodHybrid     RoutingMethod = "hybrid"
	RoutingMethodRoundRobin RoutingMethod = "round_robin"
)

// RoutingFactor represents factors that influence routing decisions
type RoutingFactor struct {
	Name        string  `json:"name"`
	Weight      float64 `json:"weight"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
}

// AgentScoring represents scoring of agents for a query
type AgentScoring struct {
	AgentName     string             `json:"agent_name"`
	Score         float64            `json:"score"`
	Factors       map[string]float64 `json:"factors"`
	CanHandle     bool               `json:"can_handle"`
	Confidence    float64            `json:"confidence"`
	Reasoning     string             `json:"reasoning"`
	EstimatedCost float64            `json:"estimated_cost"`
	EstimatedTime time.Duration      `json:"estimated_time"`
}

// RoutingContext holds context for routing decisions
type RoutingContext struct {
	Query             *models.Query       `json:"query"`
	Analysis          *RoutingAnalysis    `json:"analysis"`
	AvailableAgents   []string            `json:"available_agents"`
	AgentScores       []*AgentScoring     `json:"agent_scores"`
	HistoricalData    []RoutingDecision   `json:"historical_data"`
	UserPreferences   *RoutingPreferences `json:"user_preferences"`
	SystemConstraints *SystemConstraints  `json:"system_constraints"`
}

// RoutingPreferences represents user preferences for routing
type RoutingPreferences struct {
	PreferredAgents     []string      `json:"preferred_agents"`
	AvoidAgents         []string      `json:"avoid_agents"`
	PreferSpeed         bool          `json:"prefer_speed"`
	PreferAccuracy      bool          `json:"prefer_accuracy"`
	PreferCostEffective bool          `json:"prefer_cost_effective"`
	MaxWaitTime         time.Duration `json:"max_wait_time"`
	MinConfidence       float64       `json:"min_confidence"`
}

// SystemConstraints represents system-level constraints
type SystemConstraints struct {
	MaxConcurrentRequests int           `json:"max_concurrent_requests"`
	MaxTokensPerRequest   int           `json:"max_tokens_per_request"`
	MaxCostPerRequest     float64       `json:"max_cost_per_request"`
	MaxResponseTime       time.Duration `json:"max_response_time"`
	AvailableProviders    []string      `json:"available_providers"`
	MaintenanceMode       bool          `json:"maintenance_mode"`
}

// =============================================================================
// PERFORMANCE AND MONITORING
// =============================================================================

// RoutingMetrics tracks routing performance
type RoutingMetrics struct {
	TotalQueries        int                      `json:"total_queries"`
	SuccessfulRoutings  int                      `json:"successful_routings"`
	FailedRoutings      int                      `json:"failed_routings"`
	AverageRoutingTime  time.Duration            `json:"average_routing_time"`
	AgentUtilization    map[string]int           `json:"agent_utilization"`
	IntentDistribution  map[string]int           `json:"intent_distribution"`
	SuccessRateByAgent  map[string]float64       `json:"success_rate_by_agent"`
	SuccessRateByIntent map[string]float64       `json:"success_rate_by_intent"`
	CostByAgent         map[string]float64       `json:"cost_by_agent"`
	ResponseTimeByAgent map[string]time.Duration `json:"response_time_by_agent"`
	LastUpdated         time.Time                `json:"last_updated"`
}

// RoutingHealth represents the health of the routing system
type RoutingHealth struct {
	OverallHealth     string             `json:"overall_health"`
	AgentHealth       map[string]string  `json:"agent_health"`
	RecentErrors      []RoutingError     `json:"recent_errors"`
	PerformanceAlerts []PerformanceAlert `json:"performance_alerts"`
	SystemLoad        float64            `json:"system_load"`
	MemoryUsage       float64            `json:"memory_usage"`
	ActiveConnections int                `json:"active_connections"`
	HealthCheckTime   time.Time          `json:"health_check_time"`
}

// RoutingError represents routing errors
type RoutingError struct {
	QueryID     string    `json:"query_id"`
	AgentName   string    `json:"agent_name"`
	Error       string    `json:"error"`
	ErrorType   string    `json:"error_type"`
	Timestamp   time.Time `json:"timestamp"`
	Recoverable bool      `json:"recoverable"`
}

// PerformanceAlert represents performance alerts
type PerformanceAlert struct {
	Type        string    `json:"type"`
	Message     string    `json:"message"`
	Severity    string    `json:"severity"`
	AgentName   string    `json:"agent_name,omitempty"`
	MetricValue float64   `json:"metric_value"`
	Threshold   float64   `json:"threshold"`
	Timestamp   time.Time `json:"timestamp"`
	Resolved    bool      `json:"resolved"`
}

// =============================================================================
// LEARNING AND ADAPTATION
// =============================================================================

// LearningData represents data used for routing optimization
type LearningData struct {
	QueryPatterns      map[string]float64   `json:"query_patterns"`
	AgentPerformance   map[string]float64   `json:"agent_performance"`
	UserSatisfaction   map[string]float64   `json:"user_satisfaction"`
	SeasonalTrends     map[string][]float64 `json:"seasonal_trends"`
	FailurePatterns    map[string]int       `json:"failure_patterns"`
	OptimalRoutings    []OptimalRouting     `json:"optimal_routings"`
	LastLearningUpdate time.Time            `json:"last_learning_update"`
}

// OptimalRouting represents learned optimal routings
type OptimalRouting struct {
	QueryPattern string        `json:"query_pattern"`
	Intent       string        `json:"intent"`
	BestAgent    string        `json:"best_agent"`
	Confidence   float64       `json:"confidence"`
	SuccessRate  float64       `json:"success_rate"`
	AverageTime  time.Duration `json:"average_time"`
	AverageCost  float64       `json:"average_cost"`
	SampleSize   int           `json:"sample_size"`
	LastUpdated  time.Time     `json:"last_updated"`
}

// AdaptationStrategy defines how the system adapts over time
type AdaptationStrategy struct {
	LearningRate        float64       `json:"learning_rate"`
	AdaptationInterval  time.Duration `json:"adaptation_interval"`
	MinSampleSize       int           `json:"min_sample_size"`
	ConfidenceDecay     float64       `json:"confidence_decay"`
	ExplorationRate     float64       `json:"exploration_rate"`
	EnableReinforcement bool          `json:"enable_reinforcement"`
}

// =============================================================================
// SHARED BASE AGENT TYPES
// =============================================================================

// MetricsCollector interface for collecting agent metrics

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// NewAgentConfig creates a new agent configuration with defaults
func NewManagerAgentConfig() *AgentConfig {
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

// NewRoutingStrategy creates a default routing strategy
func NewRoutingStrategy() *RoutingStrategy {
	return &RoutingStrategy{
		PrimaryStrategy:     RoutingMethodHybrid,
		FallbackStrategies:  []RoutingMethod{RoutingMethodCapability, RoutingMethodIntent},
		ConfidenceThreshold: 0.7,
		LearningEnabled:     true,
		HistoryWeight:       0.3,
		Factors: []RoutingFactor{
			{Name: "capability_match", Weight: 0.4, Type: "boolean", Description: "Agent can handle the query"},
			{Name: "intent_confidence", Weight: 0.3, Type: "float", Description: "Confidence in intent parsing"},
			{Name: "historical_success", Weight: 0.2, Type: "float", Description: "Historical success rate"},
			{Name: "load_balancing", Weight: 0.1, Type: "float", Description: "Current agent load"},
		},
		AgentPriorities: make(map[string]int),
	}
}

// NewRoutingMetrics creates new routing metrics
func NewRoutingMetrics() *RoutingMetrics {
	return &RoutingMetrics{
		AgentUtilization:    make(map[string]int),
		IntentDistribution:  make(map[string]int),
		SuccessRateByAgent:  make(map[string]float64),
		SuccessRateByIntent: make(map[string]float64),
		CostByAgent:         make(map[string]float64),
		ResponseTimeByAgent: make(map[string]time.Duration),
		LastUpdated:         time.Now(),
	}
}
