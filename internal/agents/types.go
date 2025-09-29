package agents

import (
	"context"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// AgentInterface defines the common behavior every agent must implement
type AgentInterface interface {
	// Core capabilities
	CanHandle(ctx context.Context, query *models.Query) (bool, float64)
	Process(ctx context.Context, query *models.Query) (*models.Response, error)
	GetCapabilities() Capabilities
	GetMetrics() Metrics
}

// Capabilities describes what an agent can do
type Capabilities struct {
	CanGenerateCode    bool
	CanSearchCode      bool
	CanAnalyzeCode     bool
	SupportedLanguages []string
	MaxComplexity      int
	RequiresContext    bool
}

// Metrics tracks agent performance
type Metrics struct {
	QueriesHandled      int
	SuccessRate         float64
	AverageResponseTime time.Duration
	AverageConfidence   float64
	TokensUsed          int64
	TotalCost           float64
	LastUsed            time.Time
	ErrorCount          int
}

// Dependencies holds common dependencies for agents
type Dependencies struct {
	Storage    Storage
	VectorDB   VectorDB
	LLMManager LLMManager
	Logger     Logger
}

// Simplified interfaces to break circular dependencies
type Storage interface {
	GetFile(path string) (interface{}, error)
	SaveFile(file interface{}) error
	GetStats() (interface{}, error)
}

type VectorDB interface {
	Search(ctx context.Context, query string, limit int) ([]interface{}, error)
}

type LLMManager interface {
	Generate(ctx context.Context, request interface{}) (interface{}, error)
}

type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
}