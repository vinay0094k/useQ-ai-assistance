package agents

import (
	"time"
)

// =============================================================================
// CODING AGENT CONFIGURATION
// =============================================================================
// Note: Base AgentConfig and AgentDependencies are imported from base_agent_types.go

// CodingAgentConfig extends base agent configuration with coding-specific settings
type CodingAgentConfig struct {
	AgentConfig                 // Embedded from base_agent_types.go
	MaxContextFiles     int     `json:"max_context_files"`
	MaxContextLines     int     `json:"max_context_lines"`
	SimilarityThreshold float32 `json:"similarity_threshold"`
	IncludeTests        bool    `json:"include_tests"`
	IncludeDocs         bool    `json:"include_docs"`
	UseProjectPatterns  bool    `json:"use_project_patterns"`
	MaxExamples         int     `json:"max_examples"`
	GenerateComments    bool    `json:"generate_comments"`
	GenerateTests       bool    `json:"generate_tests"`
	ValidateGenerated   bool    `json:"validate_generated"`
	OptimizeCode        bool    `json:"optimize_code"`
}

// =============================================================================
// CODE GENERATION INTENT
// =============================================================================

// CodeIntent represents parsed code generation intent
type CodingAgentIntent struct {
	Type              CodingAgentIntentType `json:"type"`
	Description       string                `json:"description"`
	FunctionName      string                `json:"function_name,omitempty"`
	Parameters        []AgentParameter      `json:"parameters,omitempty"` // Parameter imported from base
	ReturnType        string                `json:"return_type,omitempty"`
	TargetFile        string                `json:"target_file,omitempty"`
	Framework         string                `json:"framework,omitempty"`
	Libraries         []string              `json:"libraries,omitempty"`
	Constraints       []string              `json:"constraints,omitempty"`
	RequiredFeatures  []string              `json:"required_features,omitempty"`
	IntegrationPoints []string              `json:"integration_points,omitempty"`
	Complexity        int                   `json:"complexity,omitempty"`
	Priority          int                   `json:"priority,omitempty"`
	Context           string                `json:"context,omitempty"`
}

// CodingAgentIntentType represents different types of code generation
type CodingAgentIntentType string

const (
	CodeIntentFunction   CodingAgentIntentType = "function"
	CodeIntentMethod     CodingAgentIntentType = "method"
	CodeIntentStruct     CodingAgentIntentType = "struct"
	CodeIntentInterface  CodingAgentIntentType = "interface"
	CodeIntentHandler    CodingAgentIntentType = "handler"
	CodeIntentService    CodingAgentIntentType = "service"
	CodeIntentRepository CodingAgentIntentType = "repository"
	CodeIntentMiddleware CodingAgentIntentType = "middleware"
	CodeIntentTest       CodingAgentIntentType = "test"
	CodeIntentScript     CodingAgentIntentType = "script"
	CodeIntentConfig     CodingAgentIntentType = "config"
	CodeIntentModel      CodingAgentIntentType = "model"
	CodeIntentController CodingAgentIntentType = "controller"
	CodeIntentValidator  CodingAgentIntentType = "validator"
	CodeIntentUtility    CodingAgentIntentType = "utility"
)

// =============================================================================
// CODE ANALYSIS AND CONTEXT (Full Definitions)
// =============================================================================

// CodeContext holds comprehensive code context for agents
type CodeContext struct {
	ProjectInfo       *ProjectInfo        `json:"project_info"`
	SimilarCode       []CodeExample       `json:"similar_code"`
	RelevantTypes     []TypeDefinition    `json:"relevant_types"`
	RelevantFunctions []FunctionDef       `json:"relevant_functions"`
	Dependencies      []Dependency        `json:"dependencies"`
	Patterns          []ProjectPattern    `json:"patterns"`
	ImportSuggestions []ImportSuggestion  `json:"import_suggestions"`
	UsageExamples     []UsageExample      `json:"usage_examples"` // UsageExample imported from base
	FileStructure     map[string]FileInfo `json:"file_structure"`
	ArchitectureInfo  *ArchitectureInfo   `json:"architecture_info"`
}

// ProjectInfo holds comprehensive project information
type ProjectInfo struct {
	Name               string                 `json:"name"`
	Language           string                 `json:"language"`
	Framework          string                 `json:"framework,omitempty"`
	Version            string                 `json:"version,omitempty"`
	PackageName        string                 `json:"package_name,omitempty"`
	Architecture       ArchitectureType       `json:"architecture"`
	CodingStyle        CodingStyle            `json:"coding_style"`
	Dependencies       []string               `json:"dependencies"`
	DevDependencies    []string               `json:"dev_dependencies"`
	FileStructure      map[string]string      `json:"file_structure"`
	Configuration      map[string]interface{} `json:"configuration"`
	BuildSystem        string                 `json:"build_system,omitempty"`
	TestFrameworks     []string               `json:"test_frameworks"`
	DocumentationStyle string                 `json:"documentation_style,omitempty"`
}

// ArchitectureType represents different architecture patterns
type ArchitectureType string

const (
	ArchitectureMonolith     ArchitectureType = "monolith"
	ArchitectureMicroservice ArchitectureType = "microservice"
	ArchitectureLayered      ArchitectureType = "layered"
	ArchitectureHexagonal    ArchitectureType = "hexagonal"
	ArchitectureMVC          ArchitectureType = "mvc"
	ArchitectureMVVM         ArchitectureType = "mvvm"
	ArchitectureCleanArch    ArchitectureType = "clean_architecture"
)

// ArchitectureInfo holds detailed architecture information
type ArchitectureInfo struct {
	Type        ArchitectureType       `json:"type"`
	Patterns    []string               `json:"patterns"`
	Layers      []Layer                `json:"layers"`
	Components  []Component            `json:"components"`
	Services    []Service              `json:"services"`
	Boundaries  []ArchitectureBoundary `json:"boundaries"`
	Description string                 `json:"description"`
}

// Layer represents an architectural layer
type Layer struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Dependencies []string `json:"dependencies"`
	Components   []string `json:"components"`
}

// Component represents an architectural component
type Component struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Layer        string   `json:"layer"`
	Dependencies []string `json:"dependencies"`
	Description  string   `json:"description"`
}

// Service represents a service in the architecture
type Service struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Endpoints    []string `json:"endpoints"`
	Dependencies []string `json:"dependencies"`
	Description  string   `json:"description"`
}

// ArchitectureBoundary represents boundaries between components
type ArchitectureBoundary struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// CodingStyle represents project coding patterns and conventions
type CodingStyle struct {
	NamingConvention   NamingConvention `json:"naming_convention"`
	ErrorHandlingStyle string           `json:"error_handling"`
	LoggingPattern     string           `json:"logging_pattern"`
	TestingFramework   string           `json:"testing_framework"`
	CommonPatterns     []string         `json:"common_patterns"`
	PreferredLibraries []string         `json:"preferred_libraries"`
	CodeFormatting     CodeFormatting   `json:"code_formatting"`
	DocumentationStyle string           `json:"documentation_style"`
	CommentStyle       string           `json:"comment_style"`
	ImportOrganization string           `json:"import_organization"`
}

// NamingConvention represents naming conventions
type NamingConvention struct {
	Functions  string `json:"functions"` // camelCase, snake_case, etc.
	Variables  string `json:"variables"`
	Constants  string `json:"constants"`
	Types      string `json:"types"`
	Packages   string `json:"packages"`
	Files      string `json:"files"`
	Interfaces string `json:"interfaces"`
}

// CodeFormatting represents code formatting preferences
type CodeFormatting struct {
	IndentStyle    string `json:"indent_style"` // spaces, tabs
	IndentSize     int    `json:"indent_size"`
	LineLength     int    `json:"line_length"`
	BraceStyle     string `json:"brace_style"`
	SpacingRules   string `json:"spacing_rules"`
	AlignmentRules string `json:"alignment_rules"`
}

// CodeExample represents a code example from the project
type CodeExample struct {
	ID            string            `json:"id"`
	Function      string            `json:"function"`
	File          string            `json:"file"`
	StartLine     int               `json:"start_line"`
	EndLine       int               `json:"end_line"`
	Code          string            `json:"code"`
	Context       string            `json:"context"`
	Similarity    float64           `json:"similarity"`
	Usage         []UsageExample    `json:"usage,omitempty"` // UsageExample from base
	Documentation string            `json:"documentation,omitempty"`
	Metadata      map[string]string `json:"metadata"`
	Language      string            `json:"language"`
	Complexity    int               `json:"complexity"`
}

// TypeDefinition represents a type in the project
type TypeDefinition struct {
	Name          string              `json:"name"`
	Type          TypeKind            `json:"type"`
	Fields        map[string]Field    `json:"fields,omitempty"`
	Methods       []MethodDef         `json:"methods,omitempty"`
	File          string              `json:"file"`
	Line          int                 `json:"line"`
	Package       string              `json:"package"`
	Documentation string              `json:"documentation,omitempty"`
	Usage         []UsageExample      `json:"usage,omitempty"` // UsageExample from base
	Implements    []string            `json:"implements,omitempty"`
	Extends       string              `json:"extends,omitempty"`
	Generics      []AgentGenericParam `json:"generics,omitempty"` // GenericParam from base
	Visibility    AgentVisibility     `json:"visibility"`         // Visibility from base
}

// TypeKind represents different kinds of types
type TypeKind string

const (
	TypeKindStruct    TypeKind = "struct"
	TypeKindInterface TypeKind = "interface"
	TypeKindEnum      TypeKind = "enum"
	TypeKindUnion     TypeKind = "union"
	TypeKindAlias     TypeKind = "alias"
	TypeKindGeneric   TypeKind = "generic"
)

// Field represents a field in a type
type Field struct {
	Name          string          `json:"name"`
	Type          string          `json:"type"`
	Tags          string          `json:"tags,omitempty"`
	Documentation string          `json:"documentation,omitempty"`
	Visibility    AgentVisibility `json:"visibility"` // Visibility from base
	IsRequired    bool            `json:"is_required"`
	DefaultValue  string          `json:"default_value,omitempty"`
}

// FunctionDef represents a function definition
type FunctionDef struct {
	Name          string           `json:"name"`
	Signature     string           `json:"signature"`
	Parameters    []AgentParameter `json:"parameters"` // Parameter from base
	ReturnType    string           `json:"return_type"`
	File          string           `json:"file"`
	StartLine     int              `json:"start_line"`
	EndLine       int              `json:"end_line"`
	Package       string           `json:"package"`
	Receiver      *AgentReceiver   `json:"receiver,omitempty"` // Receiver from base
	Documentation string           `json:"documentation,omitempty"`
	Complexity    int              `json:"complexity"`
	Visibility    AgentVisibility  `json:"visibility"` // Visibility from base
	IsTest        bool             `json:"is_test"`
	IsAsync       bool             `json:"is_async"`
	Tags          []string         `json:"tags"`
}

// MethodDef represents a method definition
type MethodDef struct {
	Name          string           `json:"name"`
	Signature     string           `json:"signature"`
	Parameters    []AgentParameter `json:"parameters"` // Parameter from base
	ReturnType    string           `json:"return_type"`
	Receiver      *AgentReceiver   `json:"receiver"` // Receiver from base
	Documentation string           `json:"documentation,omitempty"`
	Visibility    AgentVisibility  `json:"visibility"` // Visibility from base
	IsStatic      bool             `json:"is_static"`
	IsAbstract    bool             `json:"is_abstract"`
}

// Dependency represents a project dependency
type Dependency struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Type        DependencyType    `json:"type"`
	Source      string            `json:"source"`
	Usage       []string          `json:"usage"`
	IsDevOnly   bool              `json:"is_dev_only"`
	License     string            `json:"license,omitempty"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata"`
}

// DependencyType represents types of dependencies
type DependencyType string

const (
	DependencyTypeProduction  DependencyType = "production"
	DependencyTypeDevelopment DependencyType = "development"
	DependencyTypeTest        DependencyType = "test"
	DependencyTypeOptional    DependencyType = "optional"
	DependencyTypePeer        DependencyType = "peer"
)

// ProjectPattern represents common patterns in the project
type ProjectPattern struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Pattern     string   `json:"pattern"`
	Type        string   `json:"type"`
	Context     string   `json:"context"`
	Frequency   int      `json:"frequency"`
	Confidence  float64  `json:"confidence"`
	Example     string   `json:"example,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags"`
}

// ImportSuggestion suggests imports based on project patterns
type ImportSuggestion struct {
	Import     string  `json:"import"`
	Alias      string  `json:"alias,omitempty"`
	Usage      string  `json:"usage"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
	Type       string  `json:"type"` // standard, third-party, local
	IsUsed     bool    `json:"is_used"`
}

// FileInfo represents information about a file
type FileInfo struct {
	Path         string            `json:"path"`
	Name         string            `json:"name"`
	Extension    string            `json:"extension"`
	Size         int64             `json:"size"`
	Language     string            `json:"language"`
	LineCount    int               `json:"line_count"`
	Functions    []string          `json:"functions"`
	Types        []string          `json:"types"`
	Imports      []string          `json:"imports"`
	Package      string            `json:"package"`
	LastModified time.Time         `json:"last_modified"`
	Metadata     map[string]string `json:"metadata"`
}

// =============================================================================
// CODE ANALYSIS STRUCTURES
// =============================================================================

// CodeAnalysis represents comprehensive code analysis results
type CodeAnalysis struct {
	Language      string                `json:"language"`
	Complexity    *ComplexityAnalysis   `json:"complexity"`
	Dependencies  []Dependency          `json:"dependencies"`
	Architecture  *ArchitectureAnalysis `json:"architecture"`
	Quality       *QualityMetrics       `json:"quality"`
	Patterns      []DetectedPattern     `json:"patterns"`
	Suggestions   []CodeSuggestion      `json:"suggestions"`
	Issues        []CodeIssue           `json:"issues"`
	Documentation *DocAnalysis          `json:"documentation"`
	TestCoverage  *TestCoverage         `json:"test_coverage"`
}

// ComplexityAnalysis represents code complexity metrics
type ComplexityAnalysis struct {
	Cyclomatic      int     `json:"cyclomatic"`
	Cognitive       int     `json:"cognitive"`
	Halstead        int     `json:"halstead"`
	Maintainability float64 `json:"maintainability"`
	TechnicalDebt   int     `json:"technical_debt"`
}

// ArchitectureAnalysis represents architecture analysis
type ArchitectureAnalysis struct {
	Type         ArchitectureType `json:"type"`
	Layers       []Layer          `json:"layers"`
	Components   []Component      `json:"components"`
	Dependencies []ArchDependency `json:"dependencies"`
	Violations   []ArchViolation  `json:"violations"`
}

// ArchDependency represents an architectural dependency
type ArchDependency struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Type   string `json:"type"`
	Weight int    `json:"weight"`
}

// ArchViolation represents an architectural violation
type ArchViolation struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Location    string `json:"location"`
}

// QualityMetrics represents code quality metrics
type QualityMetrics struct {
	Readability     float64 `json:"readability"`
	Maintainability float64 `json:"maintainability"`
	Testability     float64 `json:"testability"`
	Reusability     float64 `json:"reusability"`
	Overall         float64 `json:"overall"`
}

// DetectedPattern represents a detected code pattern
type DetectedPattern struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Confidence  float64 `json:"confidence"`
	Location    string  `json:"location"`
	Description string  `json:"description"`
}

// CodeSuggestion represents a code improvement suggestion
type CodeSuggestion struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Code        string `json:"code,omitempty"`
	Impact      string `json:"impact"`
	Priority    int    `json:"priority"`
}

// CodeIssue represents a code issue
type CodeIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	Rule        string `json:"rule,omitempty"`
}

// DocAnalysis represents documentation analysis
type DocAnalysis struct {
	Coverage    float64  `json:"coverage"`
	Quality     float64  `json:"quality"`
	Missing     []string `json:"missing"`
	Outdated    []string `json:"outdated"`
	Suggestions []string `json:"suggestions"`
}

// TestCoverage represents test coverage information
type TestCoverage struct {
	Percentage   float64  `json:"percentage"`
	Lines        int      `json:"lines"`
	CoveredLines int      `json:"covered_lines"`
	Missing      []string `json:"missing"`
	Flaky        []string `json:"flaky"`
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// NewCodingConfig creates a new coding agent configuration
func NewCodingConfig() *CodingAgentConfig {
	base := NewAgentConfig() // Function from base_agent_types.go
	return &CodingAgentConfig{
		AgentConfig:         *base,
		MaxContextFiles:     10,
		MaxContextLines:     500,
		SimilarityThreshold: 0.6,
		IncludeTests:        true,
		IncludeDocs:         true,
		UseProjectPatterns:  true,
		MaxExamples:         5,
		GenerateComments:    true,
		GenerateTests:       false,
		ValidateGenerated:   true,
		OptimizeCode:        false,
	}
}
