package agents

import (
	"time"
)

// =============================================================================
// INTELLIGENCE CODING AGENT CONFIGURATION
// =============================================================================

// IntelligenceCodingAgentConfig extends base agent configuration with advanced coding features
type IntelligenceCodingAgentConfig struct {
	AgentConfig
	// Intelligence Settings
	IntelligenceLevel      int  `json:"intelligence_level"` // 1-10 scale
	MaxLayers              int  `json:"max_layers"`         // Max intelligence layers
	EnablePatternDetection bool `json:"enable_pattern_detection"`

	// Deep Analysis Settings
	EnableDeepAnalysis   bool `json:"enable_deep_analysis"`
	AnalysisDepth        int  `json:"analysis_depth"` // 1-10 scale
	CrossFileAnalysis    bool `json:"cross_file_analysis"`
	ArchitectureAnalysis bool `json:"architecture_analysis"`
	PerformanceAnalysis  bool `json:"performance_analysis"`

	// Code Generation Settings
	MaxCodeLength        int     `json:"max_code_length"`
	CodeQualityThreshold float64 `json:"code_quality_threshold"`
	EnableOptimization   bool    `json:"enable_optimization"`
	EnableTesting        bool    `json:"enable_testing"`
	EnableDocumentation  bool    `json:"enable_documentation"`

	// Intelligence Features
	PatternRecognition bool `json:"pattern_recognition"`
	AutoRefactoring    bool `json:"auto_refactoring"`
	SmartCompletion    bool `json:"smart_completion"`
	ContextAwareness   bool `json:"context_awareness"`
	LearningEnabled    bool `json:"learning_enabled"`
}

// =============================================================================
// INTELLIGENCE CODING AGENT TYPES
// =============================================================================

// IntelligenceCodingAgentType represents different types of coding intelligence
type IntelligenceCodingAgentType string

const (
	IntelligenceCodingAgentTypeGeneration    IntelligenceCodingAgentType = "generation"
	IntelligenceCodingAgentTypeRefactoring   IntelligenceCodingAgentType = "refactoring"
	IntelligenceCodingAgentTypeOptimization  IntelligenceCodingAgentType = "optimization"
	IntelligenceCodingAgentTypeAnalysis      IntelligenceCodingAgentType = "analysis"
	IntelligenceCodingAgentTypeDebugging     IntelligenceCodingAgentType = "debugging"
	IntelligenceCodingAgentTypeTesting       IntelligenceCodingAgentType = "testing"
	IntelligenceCodingAgentTypeDocumentation IntelligenceCodingAgentType = "documentation"
)

// IntelligenceCodingAgentIntent represents coding intent with intelligence
type IntelligenceCodingAgentIntent struct {
	Query                string                            `json:"query"`
	Type                 IntelligenceCodingAgentType       `json:"type"`
	Language             string                            `json:"language"`
	Framework            string                            `json:"framework,omitempty"`
	Context              map[string]interface{}            `json:"context"`
	Requirements         []string                          `json:"requirements"`
	Constraints          []string                          `json:"constraints"`
	QualityLevel         int                               `json:"quality_level"` // 1-10
	OptimizeFor          string                            `json:"optimize_for"`  // performance, readability, maintainability
	IncludeTests         bool                              `json:"include_tests"`
	IncludeDocs          bool                              `json:"include_docs"`
	Complexity           IntelligenceCodingAgentComplexity `json:"complexity"`
	ComplexityLevel      int                               `json:"complexity_level"`
	RequiresAnalysis     bool                              `json:"requires_analysis"`
	RequiresGeneration   bool                              `json:"requires_generation"`
	DomainSpecific       bool                              `json:"domain_specific"`
	GenerationType       string                            `json:"generation_type"`
	RequiresOptimization bool                              `json:"requires_optimization"`
	CrossFileScope       bool                              `json:"cross_file_scope"`
	ArchitecturalScope   bool                              `json:"architecture_scope"`
	AnalysisType         string                            `json:"analysis_type"`
	OptimizationTarget   string                            `json:"optimization_target"`
	QualityFocus         []string                          `json:"quality_focus"`
}

// IntelligenceCodingAgentComplexity represents code complexity levels
type IntelligenceCodingAgentComplexity string

const (
	IntelligenceCodingAgentComplexitySimple   IntelligenceCodingAgentComplexity = "simple"
	IntelligenceCodingAgentComplexityModerate IntelligenceCodingAgentComplexity = "moderate"
	IntelligenceCodingAgentComplexityComplex  IntelligenceCodingAgentComplexity = "complex"
	IntelligenceCodingAgentComplexityAdvanced IntelligenceCodingAgentComplexity = "advanced"
)

// IntelligenceCodingAgentResult represents enhanced coding results
type IntelligenceCodingAgentResult struct {
	Code          string                 `json:"code"`
	Language      string                 `json:"language"`
	Framework     string                 `json:"framework,omitempty"`
	Explanation   string                 `json:"explanation"`
	QualityScore  float64                `json:"quality_score"`
	Performance   map[string]interface{} `json:"performance"`
	Tests         []string               `json:"tests,omitempty"`
	Documentation string                 `json:"documentation,omitempty"`
	Dependencies  []string               `json:"dependencies"`
	Suggestions   []string               `json:"suggestions"`
	Warnings      []string               `json:"warnings"`
	Metadata      map[string]string      `json:"metadata"`
	UsageExamples []UsageExample         `json:"usage_examples"`
}

// IntelligenceCodingAgentContext holds context for coding operations
type IntelligenceCodingAgentContext struct {
	Query                string                              `json:"query"`
	Intent               *IntelligenceCodingAgentIntent      `json:"intent"`
	ProjectContext       map[string]interface{}              `json:"project_context"`
	FileContext          []string                            `json:"file_context"`
	HistoryContext       []IntelligenceCodingAgentHistory    `json:"history_context"`
	UserPreferences      *IntelligenceCodingAgentPreferences `json:"user_preferences"`
	ArchitecturalContext *AgentArchitecturalContext          `json:"architectural_context"`
}

// IntelligenceCodingAgentHistory represents coding history
type IntelligenceCodingAgentHistory struct {
	Query        string    `json:"query"`
	Language     string    `json:"language"`
	Success      bool      `json:"success"`
	QualityScore float64   `json:"quality_score"`
	Timestamp    time.Time `json:"timestamp"`
}

// IntelligenceCodingAgentPreferences represents user coding preferences
type IntelligenceCodingAgentPreferences struct {
	AgentPreferences
	PreferredStyle      string   `json:"preferred_style"`
	PreferredFrameworks []string `json:"preferred_frameworks"`
	CodeStandards       string   `json:"code_standards"`
	TestingFramework    string   `json:"testing_framework"`
}

// IntelligenceCodingAgentStrategy defines coding strategy
type IntelligenceCodingAgentStrategy struct {
	Approach          string                          `json:"approach"`
	Methods           []IntelligenceCodingAgentMethod `json:"methods"`
	QualityChecks     []string                        `json:"quality_checks"`
	OptimizationLevel int                             `json:"optimization_level"`
}

// IntelligenceCodingAgentMethod defines coding methods
type IntelligenceCodingAgentMethod string

const (
	IntelligenceCodingAgentMethodTDD          IntelligenceCodingAgentMethod = "tdd"
	IntelligenceCodingAgentMethodBDD          IntelligenceCodingAgentMethod = "bdd"
	IntelligenceCodingAgentMethodRefactoring  IntelligenceCodingAgentMethod = "refactoring"
	IntelligenceCodingAgentMethodOptimization IntelligenceCodingAgentMethod = "optimization"
)

// IntelligenceCodingAgentLayer represents intelligence layers
type IntelligenceCodingAgentLayer struct {
	Name    string                 `json:"name"`
	Type    string                 `json:"type"`
	Weight  float64                `json:"weight"`
	Enabled bool                   `json:"enabled"`
	Config  map[string]interface{} `json:"config"`
}

// IntelligenceCodingAgentAnalysisResult represents deep analysis results
type IntelligenceCodingAgentAnalysisResult struct {
	Code         string                 `json:"code"`
	Language     string                 `json:"language"`
	Complexity   float64                `json:"complexity"`
	QualityScore float64                `json:"quality_score"`
	Patterns     []string               `json:"patterns"`
	Suggestions  []string               `json:"suggestions"`
	Issues       []string               `json:"issues"`
	Dependencies []string               `json:"dependencies"`
	Performance  map[string]interface{} `json:"performance"`
	Architecture map[string]interface{} `json:"architecture"`
	Timestamp    time.Time              `json:"timestamp"`
}

// IntelligenceCodingAgentPatternDatabase represents pattern database
type IntelligenceCodingAgentPatternDatabase struct {
	Patterns    map[string]IntelligenceCodingAgentPattern `json:"patterns"`
	Categories  []string                                  `json:"categories"`
	LastUpdated time.Time                                 `json:"last_updated"`
}

// IntelligenceCodingAgentPattern represents code patterns
type IntelligenceCodingAgentPattern struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Language    string    `json:"language"`
	Pattern     string    `json:"pattern"`
	Description string    `json:"description"`
	Examples    []string  `json:"examples"`
	Usage       int       `json:"usage"`
	Quality     float64   `json:"quality"`
	CreatedAt   time.Time `json:"created_at"`
}

// IntelligenceCodingAgentDeepAnalysisRequest represents deep analysis request
type IntelligenceCodingAgentDeepAnalysisRequest struct {
	Code         string                         `json:"code"`
	Language     string                         `json:"language"`
	Context      map[string]interface{}         `json:"context"`
	AnalysisType string                         `json:"analysis_type"`
	Depth        int                            `json:"depth"`
	Layers       []IntelligenceCodingAgentLayer `json:"layers"`
}

// IntelligenceCodingAgentDeepAnalysisResult represents deep analysis result
type IntelligenceCodingAgentDeepAnalysisResult struct {
	AgentCodeAnalysis
	DeepInsights   []string               `json:"deep_insights"`
	Patterns       []string               `json:"patterns"`
	Architecture   map[string]interface{} `json:"architecture"`
	Performance    map[string]interface{} `json:"performance"`
	ProcessingTime time.Duration          `json:"processing_time"`
	Confidence     float64                `json:"confidence"`
}

// IntelligenceCodingAgentDeepIntentType represents deep intent types
type IntelligenceCodingAgentDeepIntentType string

const (
	IntelligenceCodingAgentDeepIntentAnalysis     IntelligenceCodingAgentDeepIntentType = "analysis"
	IntelligenceCodingAgentDeepIntentGeneration   IntelligenceCodingAgentDeepIntentType = "generation"
	IntelligenceCodingAgentDeepIntentOptimization IntelligenceCodingAgentDeepIntentType = "optimization"
	IntelligenceCodingAgentDeepIntentRefactoring  IntelligenceCodingAgentDeepIntentType = "refactoring"
)

// IntelligenceCodingAgentProcessingStrategy represents processing strategy
type IntelligenceCodingAgentProcessingStrategy struct {
	Name     string                 `json:"name"`
	Type     ProcessingStrategyType `json:"type"`
	Steps    []string               `json:"steps"`
	Priority int                    `json:"priority"`
	Config   map[string]interface{} `json:"config"`
}

// IntelligenceCodingAgentGenerationPrompts represents generation prompts
type IntelligenceCodingAgentGenerationPrompts struct {
	SystemPrompt string            `json:"system_prompt"`
	UserPrompt   string            `json:"user_prompt"`
	Examples     []string          `json:"examples"`
	Context      map[string]string `json:"context"`
	Temperature  float64           `json:"temperature"`
	MaxTokens    int               `json:"max_tokens"`
}

// IntelligenceCodingAgentDeepAnalysisContext represents deep analysis context
type IntelligenceCodingAgentDeepAnalysisContext struct {
	Code                 string                     `json:"code"`
	Language             string                     `json:"language"`
	ProjectContext       map[string]interface{}     `json:"project_context"`
	FileContext          []string                   `json:"file_context"`
	Dependencies         []string                   `json:"dependencies"`
	Architecture         map[string]interface{}     `json:"architecture"`
	Complexity           float64                    `json:"complexity"`
	QualityMetrics       map[string]float64         `json:"quality_metrics"`
	DetectedPatterns     []string                   `json:"detected_patterns"`
	SemanticContext      *AgentSemanticContext      `json:"semantic_context"`
	ArchitecturalContext *AgentArchitecturalContext `json:"architectural_context"`
}

// NewIntelligenceCodingAgentConfig creates a new intelligence coding agent configuration
func NewIntelligenceCodingAgentConfig() *IntelligenceCodingAgentConfig {
	base := NewAgentConfig()
	return &IntelligenceCodingAgentConfig{
		AgentConfig:            *base,
		IntelligenceLevel:      5,
		MaxLayers:              3,
		EnablePatternDetection: true,
		EnableDeepAnalysis:     true,
		AnalysisDepth:          5,
		CrossFileAnalysis:      false,
		ArchitectureAnalysis:   true,
		PerformanceAnalysis:    true,
		MaxCodeLength:          10000,
		CodeQualityThreshold:   0.8,
		EnableOptimization:     true,
		EnableTesting:          true,
		EnableDocumentation:    true,
		PatternRecognition:     true,
		AutoRefactoring:        false,
		SmartCompletion:        true,
		ContextAwareness:       true,
		LearningEnabled:        true,
	}
}
