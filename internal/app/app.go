package app

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/yourusername/useq-ai-assistant/config"
	"github.com/yourusername/useq-ai-assistant/internal/agents"
	"github.com/yourusername/useq-ai-assistant/internal/indexer"
	"github.com/yourusername/useq-ai-assistant/internal/llm"
	"github.com/yourusername/useq-ai-assistant/internal/vectordb"
	"github.com/yourusername/useq-ai-assistant/models"
	"github.com/yourusername/useq-ai-assistant/storage"
)

// Application represents the main application
type Application struct {
	config       *config.Config
	storage      *storage.SQLiteDB
	vectorDB     *vectordb.QdrantClient
	llmManager   *llm.Manager
	agentManager *agents.Manager
	indexer      *indexer.CodeIndexer
}

// New creates a new application instance
func New() (*Application, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	app := &Application{config: cfg}

	// Initialize components
	if err := app.initializeStorage(); err != nil {
		return nil, fmt.Errorf("storage init failed: %w", err)
	}

	if err := app.initializeVectorDB(); err != nil {
		return nil, fmt.Errorf("vector db init failed: %w", err)
	}

	if err := app.initializeLLM(); err != nil {
		fmt.Printf("⚠️ LLM initialization failed: %v\n", err)
		// Continue without LLM - some features will be limited
	}

	if err := app.initializeAgents(); err != nil {
		return nil, fmt.Errorf("agents init failed: %w", err)
	}

	if err := app.initializeIndexer(); err != nil {
		return nil, fmt.Errorf("indexer init failed: %w", err)
	}

	return app, nil
}

// ProcessQuery processes a user query
func (app *Application) ProcessQuery(ctx context.Context, query *models.Query) (*models.Response, error) {
	if app.agentManager == nil {
		return nil, fmt.Errorf("agent manager not initialized")
	}

	return app.agentManager.RouteQuery(ctx, query)
}

// RunIndexing runs the indexing process
func (app *Application) RunIndexing(ctx context.Context) error {
	if app.indexer == nil {
		return fmt.Errorf("indexer not initialized")
	}

	return app.indexer.StartIndexing(ctx)
}

// GetIndexedFiles returns list of indexed files
func (app *Application) GetIndexedFiles() ([]string, error) {
	if app.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}

	return app.storage.GetIndexedFiles()
}

// Close gracefully shuts down the application
func (app *Application) Close() error {
	if app.storage != nil {
		app.storage.Close()
	}
	if app.vectorDB != nil {
		app.vectorDB.Close()
	}
	return nil
}

// Private initialization methods
func (app *Application) initializeStorage() error {
	var err error
	app.storage, err = storage.NewSQLiteDB(app.config.Database.Path)
	return err
}

func (app *Application) initializeVectorDB() error {
	var err error
	app.vectorDB, err = vectordb.NewQdrantClient(&vectordb.QdrantConfig{
		Host:              app.config.Vector.Host,
		Port:              app.config.Vector.Port,
		Collection:        app.config.Vector.Collection,
		VectorSize:        app.config.Vector.Dimension,
		MaxRetries:        3,
		RetryDelay:        time.Second,
		ConnectionTimeout: 30 * time.Second,
		BatchSize:         100,
	})
	return err
}

func (app *Application) initializeLLM() error {
	llmConfig := llm.AIProvidersConfig{
		Primary:       app.config.AI.Primary,
		FallbackOrder: app.config.AI.Fallbacks,
		OpenAI: llm.ProviderConfig{
			APIKey:      app.config.AI.OpenAI.APIKey,
			Model:       app.config.AI.OpenAI.Model,
			MaxTokens:   app.config.AI.OpenAI.MaxTokens,
			Temperature: app.config.AI.OpenAI.Temperature,
			Timeout:     30 * time.Second,
		},
	}

	var err error
	app.llmManager, err = llm.NewManager(llmConfig)
	return err
}

func (app *Application) initializeAgents() error {
	// Create simple logger
	logger := &SimpleLogger{}

	// Create dependencies
	deps := &agents.Dependencies{
		Storage:    app.storage,
		VectorDB:   app.vectorDB,
		LLMManager: app.llmManager,
		Logger:     logger,
	}

	app.agentManager = agents.NewManager(deps)
	return nil
}

func (app *Application) initializeIndexer() error {
	var err error
	app.indexer, err = indexer.NewCodeIndexer(
		app.config.App.ProjectRoot,
		app.config.App.Extensions,
		app.config.App.ExcludeDirs,
		app.vectorDB,
		app.storage,
	)
	return err
}

// SimpleLogger implements the Logger interface
type SimpleLogger struct{}

func (l *SimpleLogger) Info(msg string, fields ...interface{}) {
	fmt.Printf("ℹ️ %s\n", msg)
}

func (l *SimpleLogger) Error(msg string, fields ...interface{}) {
	fmt.Printf("❌ %s\n", msg)
}

func (l *SimpleLogger) Debug(msg string, fields ...interface{}) {
	fmt.Printf("🔍 %s\n", msg)
}