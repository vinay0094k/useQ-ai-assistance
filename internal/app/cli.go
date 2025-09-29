// internal/app/cli.go
// Enhanced CLI application with comprehensive step-by-step execution tracing
// Provides detailed logging of file processing, function execution, database operations, and pipeline flow
package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/yourusername/useq-ai-assistant/display"
	"github.com/yourusername/useq-ai-assistant/internal/agents"
	"github.com/yourusername/useq-ai-assistant/internal/indexer"
	"github.com/yourusername/useq-ai-assistant/internal/llm"
	"github.com/yourusername/useq-ai-assistant/internal/logger"
	"github.com/yourusername/useq-ai-assistant/internal/mcp"
	"github.com/yourusername/useq-ai-assistant/internal/vectordb"
	"github.com/yourusername/useq-ai-assistant/models"
	"github.com/yourusername/useq-ai-assistant/storage"
)

// CLIApplication represents the main CLI application with enhanced logging
type CLIApplication struct {
	config                  *Config
	stepLogger              *logger.StepLogger
	executionTracer         *logger.ExecutionTracer
	sessionManager          *SessionManager
	promptParser            *PromptParser
	indexer                 *indexer.CodeIndexer
	vectorDB                *vectordb.QdrantClient
	llmManager              *llm.Manager
	codingAgent             *agents.CodingAgentImpl
	searchAgent             agents.SearchAgentImpl
	contextSearchAgent      *agents.ContextAwareSearchAgentImpl
	intelligenceCodingAgent agents.IntelligenceCodingAgentImpl
	managerAgent            *agents.ManagerAgent
	storage                 *storage.SQLiteDB
	mcpClient               agents.MCPClientInterface
	logger                  agents.Logger
	startTime               time.Time
	sessionID               string
	debugMode               bool
}

// Config holds application configuration
type Config struct {
	ProjectRoot       string
	DatabasePath      string
	LogLevel          string
	EnableStepLogging bool
	DebugMode         bool
	IndexedExtensions []string
	ExcludedDirs      []string
	AIProviders       llm.AIProvidersConfig
	Performance       PerformanceConfig
	VectorDB          VectorDBConfig
}

// PerformanceConfig holds performance settings
type PerformanceConfig struct {
	MaxFileSize        int64
	IndexingBatchSize  int
	MaxParallelWorkers int
	CacheEnabled       bool
	CacheTTL           time.Duration
}

// VectorDBConfig holds vector database configuration
type VectorDBConfig struct {
	URL            string
	APIKey         string
	CollectionName string
	Dimension      int
}

// NewCLIApplication creates a new CLI application instance with enhanced logging
func NewCLIApplication() (*CLIApplication, error) {
	return NewCLIApplicationWithLLM(nil)
}

func NewCLIApplicationWithLLM(llmManager *llm.Manager) (*CLIApplication, error) {
	fmt.Printf("🔄 Initializing CLI Application...\n")

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	fmt.Printf("✅ Configuration loaded\n")

	// Generate session ID
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())

	// Initialize step logger with FILE OUTPUT ONLY
	stepLogger, err := logger.NewStepLogger(
		sessionID,
		"", // Query ID will be set per query
		config.LogLevel,
		false, // DISABLE console output - logs go to files only
		config.EnableStepLogging,
	)
	if err != nil {
		fmt.Printf("❌ Failed to create step logger: %v\n", err)
		return nil, fmt.Errorf("failed to create step logger: %w", err)
	}

	// Log to file but show minimal console info
	fmt.Printf("📝 Step logger initialized - logs written to: ./logs/steps_%s.log\n", time.Now().Format("2006-01-02"))

	app := &CLIApplication{
		config:     config,
		stepLogger: stepLogger,
		sessionID:  sessionID,
		startTime:  time.Now(),
		debugMode:  config.DebugMode,
	}

	// Log detailed info to file
	app.logInfo("CLI_INIT", fmt.Sprintf("CLI Application initialization started with session: %s", sessionID))

	// Initialize components with detailed logging
	fmt.Printf("🔄 Initializing components...\n")
	if err := app.initializeComponentsWithLogging(llmManager); err != nil {
		fmt.Printf("❌ Failed to initialize components: %v\n", err)
		app.logError("CLI_INIT", "Failed to initialize components", err)
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	fmt.Printf("✅ CLI Application initialized successfully\n")
	app.logSuccess("CLI_INIT", "CLI Application ready for operation")

	return app, nil
}

// initializeComponentsWithLogging initializes all application components with detailed logging
func (app *CLIApplication) initializeComponentsWithLogging(llmManager *llm.Manager) error {
	app.logInfo("COMPONENT_INIT", "Starting component initialization sequence")
	mainStep := app.stepLogger.StartStep(logger.ComponentCLI, "initializing_all_components", nil)

	// 1. Initialize storage with detailed logging
	fmt.Printf("  🔄 Storage...\n")
	if err := app.initializeStorage(); err != nil {
		app.stepLogger.FailStep(mainStep, err)
		fmt.Printf("  ❌ Storage initialization failed\n")
		return err
	}
	fmt.Printf("  ✅ Storage ready\n")

	// 2. Initialize vector database
	fmt.Printf("  🔄 Vector Database...\n")
	if err := app.initializeVectorDB(); err != nil {
		app.stepLogger.FailStep(mainStep, err)
		fmt.Printf("  ❌ Vector DB initialization failed\n")
		return err
	}
	fmt.Printf("  ✅ Vector Database ready\n")

	// 3. Initialize LLM manager
	fmt.Printf("  🔄 AI Providers...\n")
	if err := app.initializeLLMManagerWithExternal(llmManager); err != nil {
		app.stepLogger.FailStep(mainStep, err)
		fmt.Printf("  ❌ LLM Manager initialization failed\n")
		return err
	}
	fmt.Printf("  ✅ AI Providers ready\n")

	// 4. Initialize MCP client
	fmt.Printf("  🔄 MCP Client...\n")
	app.initializeMCPClient()
	fmt.Printf("  ✅ MCP Client ready\n")

	// 5. Initialize code indexer
	fmt.Printf("  🔄 Code Indexer...\n")
	if err := app.initializeIndexer(); err != nil {
		app.stepLogger.FailStep(mainStep, err)
		fmt.Printf("  ❌ Indexer initialization failed\n")
		return err
	}
	fmt.Printf("  ✅ Code Indexer ready\n")

	// 6. Initialize other components
	app.initializeOtherComponents()
	fmt.Printf("  ✅ Session & Parser ready\n")

	// 7. Check if indexing is needed and run it synchronously
	fmt.Printf("  🔄 Checking indexing status...\n")
	if err := app.checkAndRunIndexing(); err != nil {
		fmt.Printf("  ⚠️ Indexing failed: %v\n", err)
		// Don't fail initialization, just log the error
		app.logError("AUTO_INDEXING", "Automatic indexing failed", err)
	}

	app.stepLogger.CompleteStep(mainStep, "All components initialized successfully")
	app.logSuccess("COMPONENT_INIT", "All components ready for operation")
	return nil
}

// checkAndRunIndexing checks if indexing is needed and runs it synchronously
func (app *CLIApplication) checkAndRunIndexing() error {
	// Check if database has any files
	stats, err := app.storage.GetStats()
	if err != nil {
		fmt.Printf("  ❌ Failed to get database stats: %v\n", err)
		return fmt.Errorf("failed to get database stats: %w", err)
	}

	fileCount := stats.TotalFiles
	fmt.Printf("  📊 Database has %d indexed files\n", fileCount)

	// If no files indexed, run full indexing
	if fileCount == 0 {
		fmt.Printf("  🔄 No files indexed, starting automatic indexing...\n")
		fmt.Printf("  📁 Project root: %s\n", app.indexer.GetProjectRoot())
		ctx := context.Background()

		err := app.indexer.StartFullReindexingWithProgress(ctx, func(progress display.IndexingProgress) {
			if progress.ProcessedFiles%10 == 0 || progress.ProcessedFiles == progress.TotalFiles {
				fmt.Printf("  📈 Indexing: %d/%d files, %d functions\n",
					progress.ProcessedFiles, progress.TotalFiles, progress.FunctionsFound)
			}
		})

		if err != nil {
			return fmt.Errorf("indexing failed: %w", err)
		}

		fmt.Printf("  ✅ Automatic indexing completed\n")
	} else {
		fmt.Printf("  ✅ Files already indexed\n")
	}

	return nil
}

// initializeStorage initializes the SQLite storage with detailed logging
func (app *CLIApplication) initializeStorage() error {
	app.logInfo("STORAGE_INIT", "Initializing SQLite database")
	storageStep := app.stepLogger.StartStep(logger.ComponentCLI, "initializing_storage",
		map[string]interface{}{
			"database_path": app.config.DatabasePath,
			"database_dir":  filepath.Dir(app.config.DatabasePath),
		})

	// Check if database directory exists
	dbDir := filepath.Dir(app.config.DatabasePath)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		app.logInfo("STORAGE_INIT", fmt.Sprintf("Creating database directory: %s", dbDir))
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			app.stepLogger.FailStep(storageStep, err)
			return fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	// Initialize database
	app.logInfo("STORAGE_INIT", "Connecting to SQLite database...")
	var err error
	app.storage, err = storage.NewSQLiteDB(app.config.DatabasePath)
	if err != nil {
		app.logError("STORAGE_INIT", "Database connection failed", err)
		app.stepLogger.FailStep(storageStep, err)
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Test database connection
	app.logInfo("STORAGE_INIT", "Testing database connection...")
	stats, err := app.storage.GetStats()
	if err != nil {
		app.logError("STORAGE_INIT", "Database health check failed", err)
		app.stepLogger.FailStep(storageStep, err)
		return fmt.Errorf("database health check failed: %w", err)
	}

	app.logSuccess("STORAGE_INIT", "SQLite database initialized", map[string]interface{}{
		"files":         stats.TotalFiles,
		"queries":       stats.TotalQueries,
		"responses":     stats.TotalResponses,
		"languages":     len(stats.LanguageBreakdown),
	})
	app.stepLogger.CompleteStep(storageStep, map[string]interface{}{
		"status": "connected",
		"stats":  stats,
	})

	return nil
}

// initializeVectorDB initializes Qdrant vector database
func (app *CLIApplication) initializeVectorDB() error {
	app.logInfo("VECTORDB_INIT", "Initializing Qdrant vector database")
	vectorStep := app.stepLogger.StartStep(logger.ComponentVectorDB, "connecting_qdrant",
		map[string]interface{}{
			"url":        app.config.VectorDB.URL,
			"collection": app.config.VectorDB.CollectionName,
			"dimension":  app.config.VectorDB.Dimension,
		})

	// Parse URL
	url := app.config.VectorDB.URL
	if url == "" {
		url = "localhost:6333" // Default
		app.logInfo("VECTORDB_INIT", "Using default Qdrant URL: localhost:6333")
	}

	// Clean URL format
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")

	parts := strings.Split(url, ":")
	if len(parts) != 2 {
		err := fmt.Errorf("invalid URL format: %s", app.config.VectorDB.URL)
		app.logError("VECTORDB_INIT", "URL parsing failed", err)
		app.stepLogger.FailStep(vectorStep, err)
		return err
	}

	host := parts[0]
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		app.logError("VECTORDB_INIT", "Invalid port number", err)
		app.stepLogger.FailStep(vectorStep, err)
		return fmt.Errorf("invalid port in URL: %s", parts[1])
	}

	app.logInfo("VECTORDB_INIT", fmt.Sprintf("Connecting to Qdrant at %s:%d", host, port))

	// Create Qdrant client
	app.vectorDB, err = vectordb.NewQdrantClient(&vectordb.QdrantConfig{
		Host:              host,
		Port:              port,
		Collection:        app.config.VectorDB.CollectionName,
		VectorSize:        app.config.VectorDB.Dimension,
		MaxRetries:        3,
		RetryDelay:        time.Second,
		ConnectionTimeout: 30 * time.Second,
		BatchSize:         100,
	})
	if err != nil {
		app.logError("VECTORDB_INIT", "Qdrant client creation failed", err)
		app.stepLogger.FailStep(vectorStep, err)
		return fmt.Errorf("failed to initialize vector database: %w", err)
	}

	app.logSuccess("VECTORDB_INIT", "Qdrant client connected successfully")
	app.stepLogger.CompleteStep(vectorStep, "Qdrant client connected")
	return nil
}

// initializeLLMManagerWithExternal uses external LLM manager or falls back to internal
func (app *CLIApplication) initializeLLMManagerWithExternal(externalLLM *llm.Manager) error {
	if externalLLM != nil {
		app.logInfo("LLM_INIT", "Using external LLM manager")
		app.llmManager = externalLLM
		return nil
	}
	
	// Fallback to internal initialization
	return app.initializeLLMManager()
}

// initializeLLMManager initializes the AI provider manager
func (app *CLIApplication) initializeLLMManager() error {
	app.logInfo("LLM_INIT", "Initializing AI provider manager")
	llmStep := app.stepLogger.StartStep(logger.ComponentLLM, "initializing_providers",
		map[string]interface{}{
			"primary":   app.config.AIProviders.Primary,
			"fallbacks": app.config.AIProviders.FallbackOrder,
		})

	// Check API keys
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		app.logWarning("LLM_INIT", "OPENAI_API_KEY not set - OpenAI provider will be unavailable")
	} else {
		app.logInfo("LLM_INIT", "OpenAI API key found")
	}

	var err error
	app.llmManager, err = llm.NewManager(app.config.AIProviders)
	if err != nil {
		app.logError("LLM_INIT", "LLM manager creation failed", err)
		app.stepLogger.FailStep(llmStep, err)
		return fmt.Errorf("failed to initialize LLM manager: %w", err)
	}

	app.logSuccess("LLM_INIT", "AI provider manager initialized")
	app.stepLogger.CompleteStep(llmStep, "AI providers initialized")
	return nil
}

// initializeIndexer initializes the code indexer
func (app *CLIApplication) initializeIndexer() error {
	app.logInfo("INDEXER_INIT", "Initializing code indexer")
	indexerStep := app.stepLogger.StartStep(logger.ComponentIndexer, "initializing_indexer",
		map[string]interface{}{
			"project_root": app.config.ProjectRoot,
			"extensions":   app.config.IndexedExtensions,
			"excluded":     app.config.ExcludedDirs,
		})

	// Verify project root exists
	if _, err := os.Stat(app.config.ProjectRoot); os.IsNotExist(err) {
		app.logError("INDEXER_INIT", "Project root does not exist", err)
		app.stepLogger.FailStep(indexerStep, err)
		return fmt.Errorf("project root does not exist: %s", app.config.ProjectRoot)
	}

	app.logInfo("INDEXER_INIT", fmt.Sprintf("Project root verified: %s", app.config.ProjectRoot))

	var err error
	app.indexer, err = indexer.NewCodeIndexer(
		app.config.ProjectRoot,
		app.config.IndexedExtensions,
		app.config.ExcludedDirs,
		app.vectorDB,
		app.storage,
	)
	if err != nil {
		app.logError("INDEXER_INIT", "Code indexer creation failed", err)
		app.stepLogger.FailStep(indexerStep, err)
		return fmt.Errorf("failed to initialize code indexer: %w", err)
	}

	app.logSuccess("INDEXER_INIT", "Code indexer initialized successfully")
	app.stepLogger.CompleteStep(indexerStep, "Code indexer initialized")
	return nil
}

// initializeOtherComponents initializes remaining components
func (app *CLIApplication) initializeOtherComponents() {
	app.logInfo("OTHER_INIT", "Initializing session manager and prompt parser")

	// Initialize session manager
	app.sessionManager = NewSessionManager(app.storage)
	app.logInfo("OTHER_INIT", "Session manager initialized")

	// Initialize prompt parser
	app.promptParser = NewPromptParser()
	app.logInfo("OTHER_INIT", "Prompt parser initialized")

	// Initialize agents
	app.initializeAgents()
}

// initializeMCPClient initializes the MCP client for enhanced context
func (app *CLIApplication) initializeMCPClient() {
	app.logInfo("MCP_INIT", "Initializing MCP client")
	app.mcpClient = mcp.NewMCPClient()
	
	// Create logger adapter for agents
	app.logger = &LoggerAdapter{stepLogger: app.stepLogger}
	app.logInfo("MCP_INIT", "MCP client and logger initialized")
}

func (app *CLIApplication) initializeAgents() {
	app.logInfo("AGENT_INIT", "Initializing AI agents")

	// Create embedder for search functionality
	embeddingConfig := &vectordb.EmbeddingConfig{
		APIKey:   app.config.AIProviders.OpenAI.APIKey,
		Endpoint: "https://api.openai.com/v1/embeddings",
		Model:    "text-embedding-3-small",
	}
	embedder := vectordb.NewEmbeddingService(embeddingConfig)

	//Create agent dependencies
	deps := &agents.AgentDependencies{
		LLMManager: app.llmManager,
		VectorDB:   app.vectorDB,
		Storage:    app.storage,
		Embedder:   embedder,
		Logger:     app.logger,
		MCPClient:  app.mcpClient,
	}
	// Initialize manager agent (handles all routing)
	app.managerAgent = agents.NewManagerAgent(deps)
	app.logInfo("AGENT_INIT", "Manager agent initialized")
	app.logInfo("AGENT_INIT", "All agents initialized via manager")

	// Get references to specialized agents from manager
	app.searchAgent = *app.managerAgent.SearchAgent
	app.codingAgent = app.managerAgent.CodingAgent
	app.contextSearchAgent = app.managerAgent.ContextAwareSearchAgent
	app.intelligenceCodingAgent = *app.managerAgent.IntelligenceCodingAgent
	app.logInfo("AGENT_INIT", "All agents initialized via manager")
}

// ProcessQuery processes a user query with comprehensive logging
func (app *CLIApplication) ProcessQuery(ctx context.Context, query *models.Query) (*models.Response, error) {
	app.logInfo("QUERY_PROC", fmt.Sprintf("Processing query: %s", query.UserInput))

	// Create execution tracer for detailed flow tracking
	tracer, err := logger.NewExecutionTracer(query.ID)
	if err != nil {
		app.logError("TRACER_INIT", "Failed to create execution tracer", err)
	}
	defer func() {
		if tracer != nil {
			tracer.Close()
		}
	}()

	if tracer != nil {
		tracer.LogFunctionCall("ProcessQuery", fmt.Sprintf("Input: %s", query.UserInput))
	}

	queryStep := app.stepLogger.StartStep(logger.ComponentCLI, "processing_query",
		map[string]interface{}{
			"query_id":     query.ID,
			"input":        query.UserInput,
			"input_length": len(query.UserInput),
			"language":     query.Language,
		})

	// Update logger with query ID
	queryLogger, err := logger.NewStepLogger(
		app.sessionID,
		query.ID,
		app.config.LogLevel,
		true, // Keep console output enabled for debugging
		app.config.EnableStepLogging,
	)
	if err == nil {
		app.stepLogger = queryLogger
	}

	// Parse query intent with detailed logging
	intent, err := app.parseQueryWithLogging(query, tracer)
	if err != nil {
		if tracer != nil {
			tracer.LogFunctionExit("ProcessQuery", fmt.Sprintf("ERROR: %v", err))
		}

		app.stepLogger.FailStep(queryStep, err)
		return nil, err
	}

	// Route to appropriate handler with logging
	response, err := app.routeQueryWithLogging(ctx, query, intent, tracer)
	if err != nil {
		if tracer != nil {
			tracer.LogFunctionExit("ProcessQuery", fmt.Sprintf("ERROR: %v", err))
		}
		app.stepLogger.FailStep(queryStep, err)
		return nil, err
	}

	// Save session data with logging
	app.saveSessionWithLogging(query, response, tracer)
	if tracer != nil {
		tracer.LogFunctionExit("ProcessQuery", fmt.Sprintf("SUCCESS: %s response generated", response.Type))
		tracer.LogEnd(fmt.Sprintf("Query completed successfully - %s", response.Type))
	}

	app.stepLogger.CompleteStep(queryStep, map[string]interface{}{
		"agent":       response.AgentUsed,
		"provider":    response.Provider,
		"tokens":      response.TokenUsage.TotalTokens,
		"cost":        response.Cost.TotalCost,
		"duration_ms": response.Metadata.GenerationTime.Milliseconds(),
	})

	app.logSuccess("QUERY_PROC", "Query processed successfully", map[string]interface{}{
		"response_type": response.Type,
		"agent":         response.AgentUsed,
		"tokens":        response.TokenUsage.TotalTokens,
	})

	return response, nil
}

// parseQueryWithLogging parses query intent with detailed logging
func (app *CLIApplication) parseQueryWithLogging(query *models.Query, tracer *logger.ExecutionTracer) (*models.QueryIntent, error) {
	if tracer != nil {
		tracer.LogFunctionCall("parseQueryWithLogging", fmt.Sprintf("Parsing intent for: %s", query.UserInput))
		tracer.LogStep("STEP_1", "Starting query intent parsing")
	}
	app.logInfo("PARSE_INTENT", "Parsing query intent")
	parseStep := app.stepLogger.StartStep(logger.ComponentParser, "parsing_intent", query.UserInput)
	if tracer != nil {
		tracer.LogFileAccess("internal/app/prompt_parser.go", "ParseIntent")
		tracer.LogStep("STEP_2", "Accessing prompt parser module")
	}

	intent, err := app.promptParser.ParseIntent(query.UserInput)
	if err != nil {
		app.logError("PARSE_INTENT", "Intent parsing failed", err)
		app.stepLogger.FailStep(parseStep, err)
		if tracer != nil {
			tracer.LogStep("STEP_ERROR", fmt.Sprintf("Parser failed: %v", err))
			tracer.LogFunctionExit("parseQueryWithLogging", fmt.Sprintf("ERROR: %v", err))
		}
		return nil, fmt.Errorf("failed to parse query intent: %w", err)
	}

	if tracer != nil {
		tracer.LogStep("STEP_3", fmt.Sprintf("Intent parsed successfully: %s (confidence: %.2f)", intent.Primary, intent.Confidence))
	}

	app.logSuccess("PARSE_INTENT", "Intent parsed successfully", map[string]interface{}{
		"primary_intent": intent.Primary,
		"confidence":     intent.Confidence,
		"keywords_count": len(intent.Keywords),
		"keywords":       intent.Keywords,
	})

	app.stepLogger.CompleteStep(parseStep, map[string]interface{}{
		"primary_intent": intent.Primary,
		"confidence":     intent.Confidence,
		"keywords":       intent.Keywords,
	})

	if tracer != nil {
		tracer.LogStep("STEP_4", "Intent parsing completed successfully")
		tracer.LogFunctionExit("parseQueryWithLogging", fmt.Sprintf("SUCCESS: Intent=%s, Confidence=%.2f", intent.Primary, intent.Confidence))
	}

	return intent, nil
}

// routeQueryWithLogging routes query to appropriate handler with logging
func (app *CLIApplication) routeQueryWithLogging(ctx context.Context, query *models.Query, intent *models.QueryIntent, tracer *logger.ExecutionTracer) (*models.Response, error) {
	if tracer != nil {
		tracer.LogFunctionCall("routeQueryWithLogging", fmt.Sprintf("Routing to handler for intent: %s", intent.Primary))
	}
	app.logInfo("ROUTE_QUERY", fmt.Sprintf("Routing query to handler for intent: %s (confidence: %.2f)", intent.Primary, intent.Confidence))

	routeStep := app.stepLogger.StartStep(logger.ComponentAgent, "routing_query", map[string]interface{}{
		"intent":     intent.Primary,
		"confidence": intent.Confidence,
		"keywords":   intent.Keywords,
	})

	var response *models.Response
	var err error

	// switch intent.Primary {

	// Use ManagerAgent for intelligent centralized routing
	if app.managerAgent != nil {
		app.logInfo("ROUTE_QUERY", "Using ManagerAgent for intelligent routing")
		response, err = app.managerAgent.RouteQuery(ctx, query)
		if err == nil {
			app.logSuccess("ROUTE_QUERY", "ManagerAgent completed successfully", map[string]interface{}{
				"agent":    response.AgentUsed,
				"provider": response.Provider,
				"tokens":   response.TokenUsage.TotalTokens,
			})
			app.stepLogger.CompleteStep(routeStep, map[string]interface{}{
				"agent":    response.AgentUsed,
				"provider": response.Provider,
				"tokens":   response.TokenUsage.TotalTokens,
			})
			if tracer != nil {
				tracer.LogFunctionExit("routeQueryWithLogging", fmt.Sprintf("SUCCESS: %s response from %s via ManagerAgent", response.Type, response.Provider))

			}
			return response, nil

		}
		app.logError("ROUTE_QUERY", "ManagerAgent failed, falling back to manual routing", err)

	}
	// Fallback to manual routing if ManagerAgent fails or is not available
	app.logInfo("ROUTE_QUERY", "Using manual routing fallback")

	switch intent.Primary {
	case models.QueryTypeSearch:
		app.logInfo("ROUTE_QUERY", "Routing to Search handler")
		if tracer != nil {
			tracer.LogFileAccess("internal/app/cli.go", "handleSearchQueryWithLogging")
		}
		response, err = app.handleSearchQueryWithLogging(ctx, query, intent, tracer)
	case models.QueryTypeGeneration:
		app.logInfo("ROUTE_QUERY", "Routing to Generation handler")
		if tracer != nil {
			tracer.LogFileAccess("internal/app/cli.go", "handleGenerationQueryWithLogging")
		}
		response, err = app.handleGenerationQueryWithLogging(ctx, query, intent, tracer)

	case models.QueryTypeExplanation:
		app.logInfo("ROUTE_QUERY", "Routing to Explanation handler")
		if tracer != nil {
			tracer.LogFileAccess("internal/app/cli.go", "handleExplanationQueryWithLogging")
		}
		response, err = app.handleExplanationQueryWithLogging(ctx, query, intent, tracer)
	case models.QueryTypeDebugging:
		app.logInfo("ROUTE_QUERY", "Routing to Debugging handler")
		if tracer != nil {
			tracer.LogFileAccess("internal/app/cli.go", "handleDebuggingQueryWithLogging")
		}
		response, err = app.handleDebuggingQueryWithLogging(ctx, query, intent, tracer)
	case models.QueryTypeTesting:
		app.logInfo("ROUTE_QUERY", "Routing to Testing handler")
		if tracer != nil {
			tracer.LogFileAccess("internal/app/cli.go", "handleTestingQueryWithLogging")
		}
		response, err = app.handleTestingQueryWithLogging(ctx, query, intent, tracer)
	default:
		app.logInfo("ROUTE_QUERY", "Routing to General handler")
		if tracer != nil {
			tracer.LogFileAccess("internal/app/cli.go", "handleGeneralQueryWithLogging")
		}
		response, err = app.handleGeneralQueryWithLogging(ctx, query, intent, tracer)
	}

	if err != nil {
		app.logError("ROUTE_QUERY", "Handler execution failed", err)
		app.stepLogger.FailStep(routeStep, err)
		return nil, fmt.Errorf("failed to process query: %w", err)
	}

	app.logSuccess("ROUTE_QUERY", "Handler completed successfully", map[string]interface{}{
		"agent":    response.AgentUsed,
		"provider": response.Provider,
		"tokens":   response.TokenUsage.TotalTokens,
	})

	app.stepLogger.CompleteStep(routeStep, map[string]interface{}{
		"agent":    response.AgentUsed,
		"provider": response.Provider,
		"tokens":   response.TokenUsage.TotalTokens,
	})

	if tracer != nil {
		tracer.LogFunctionExit("routeQueryWithLogging", fmt.Sprintf("SUCCESS: %s response from %s", response.Type, response.Provider))
	}

	return response, nil
}

// Enhanced query handlers with logging
func (app *CLIApplication) handleSearchQueryWithLogging(ctx context.Context, query *models.Query, intent *models.QueryIntent, tracer *logger.ExecutionTracer) (*models.Response, error) {
	app.logInfo("SEARCH_HANDLER", fmt.Sprintf("Executing search for keywords: %v", intent.Keywords))
	searchStep := app.stepLogger.StartStep(logger.ComponentAgent, "executing_search", map[string]interface{}{
		"keywords": intent.Keywords,
		"query":    query.UserInput,
	})

	// Use the search agent to perform actual search
	embeddingConfig := &vectordb.EmbeddingConfig{
		APIKey:   "", // Will be loaded from environment
		Endpoint: "https://api.openai.com/v1/embeddings",
		Model:    "text-embedding-3-small",
	}
	embedder := vectordb.NewEmbeddingService(embeddingConfig)

	searchAgent := agents.NewSearchAgent(&agents.AgentDependencies{
		VectorDB: app.vectorDB,
		Storage:  app.storage,
		Embedder: embedder,
		Logger:   nil, // TODO: Implement proper logger interface
	})

	response, err := searchAgent.Search(ctx, query)
	if err != nil {
		app.stepLogger.FailStep(searchStep, err)
		return nil, fmt.Errorf("search failed: %w", err)
	}

	app.stepLogger.CompleteStep(searchStep, "Search completed")
	app.logSuccess("SEARCH_HANDLER", "Search completed successfully")
	return response, nil
}

func (app *CLIApplication) handleGenerationQueryWithLogging(ctx context.Context, query *models.Query, intent *models.QueryIntent, tracer *logger.ExecutionTracer) (*models.Response, error) {
	app.logInfo("GEN_HANDLER", "Code generation handler called")

	// Check if CodingAgent can handle this query
	if app.codingAgent != nil {
		canHandle, confidence := app.codingAgent.CanHandle(ctx, query)
		app.logInfo("GEN_HANDLER", fmt.Sprintf("CodingAgent can handle: %v (confidence: %.2f)", canHandle, confidence))

		if canHandle && confidence >= 0.6 {
			app.logInfo("GEN_HANDLER", "Using CodingAgent for code generation")
			response, err := app.codingAgent.Process(ctx, query)
			if err != nil {
				// app.logError("GEN_HANDLER", fmt.Sprintf("CodingAgent failed: %v", err))
				app.logError("GEN_HANDLER", "CodingAgent failed", err)
				// Fallback to general handler
				return app.handleGeneralQueryWithLogging(ctx, query, intent, tracer)
			}
			app.logSuccess("GEN_HANDLER", "CodingAgent completed successfully")
			return response, nil
		}
	}

	app.logInfo("GEN_HANDLER", "Falling back to general handler")
	return app.handleGeneralQueryWithLogging(ctx, query, intent, tracer)
}

func (app *CLIApplication) handleExplanationQueryWithLogging(ctx context.Context, query *models.Query, intent *models.QueryIntent, tracer *logger.ExecutionTracer) (*models.Response, error) {
	app.logInfo("EXPLAIN_HANDLER", "Explanation handler called")
	return app.handleGeneralQueryWithLogging(ctx, query, intent, tracer)
}

func (app *CLIApplication) handleDebuggingQueryWithLogging(ctx context.Context, query *models.Query, intent *models.QueryIntent, tracer *logger.ExecutionTracer) (*models.Response, error) {
	app.logInfo("DEBUG_HANDLER", "Debugging handler called")
	return app.handleGeneralQueryWithLogging(ctx, query, intent, tracer)
}

func (app *CLIApplication) handleTestingQueryWithLogging(ctx context.Context, query *models.Query, intent *models.QueryIntent, tracer *logger.ExecutionTracer) (*models.Response, error) {
	app.logInfo("TEST_HANDLER", "Testing handler called")
	return app.handleGeneralQueryWithLogging(ctx, query, intent, tracer)
}

func (app *CLIApplication) handleGeneralQueryWithLogging(ctx context.Context, query *models.Query, intent *models.QueryIntent, tracer *logger.ExecutionTracer) (*models.Response, error) {
	app.logInfo("GENERAL_HANDLER", "Processing general query with LLM")
	llmStep := app.stepLogger.StartStep(logger.ComponentLLM, "generating_response", map[string]interface{}{
		"input":       query.UserInput,
		"max_tokens":  1000,
		"temperature": 0.1,
	})

	// Create LLM request
	request := &llm.GenerationRequest{
		Messages: []llm.Message{
			{Role: "user", Content: query.UserInput},
		},
		SystemPrompt: "You are a helpful AI assistant that explains code and applications.",
		MaxTokens:    1000,
		Temperature:  0.1,
	}

	app.logInfo("GENERAL_HANDLER", "Sending request to LLM manager")

	// Generate response using LLM manager
	llmResponse, err := app.llmManager.Generate(ctx, request)
	if err != nil {
		app.logError("GENERAL_HANDLER", "LLM generation failed", err)
		app.stepLogger.FailStep(llmStep, err)
		return nil, fmt.Errorf("failed to generate LLM response: %w", err)
	}

	app.logSuccess("GENERAL_HANDLER", "LLM response generated", map[string]interface{}{
		"provider": llmResponse.Provider,
		"tokens":   llmResponse.TokenUsage.TotalTokens,
		"cost":     llmResponse.Cost.TotalCost,
		"latency":  llmResponse.Latency,
	})

	app.stepLogger.CompleteStep(llmStep, map[string]interface{}{
		"provider": llmResponse.Provider,
		"tokens":   llmResponse.TokenUsage.TotalTokens,
		"cost":     llmResponse.Cost.TotalCost,
	})

	response := &models.Response{
		ID:      fmt.Sprintf("response_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    models.ResponseTypeExplanation,
		Content: models.ResponseContent{
			Text: llmResponse.Content,
		},
		AgentUsed:  "general_agent",
		Provider:   llmResponse.Provider,
		TokenUsage: llmResponse.TokenUsage,
		Cost:       llmResponse.Cost,
		Metadata: models.ResponseMetadata{
			GenerationTime: llmResponse.Latency,
			Confidence:     0.9,
		},
		Timestamp: time.Now(),
	}

	return response, nil
}

// saveSessionWithLogging saves session data with logging
func (app *CLIApplication) saveSessionWithLogging(query *models.Query, response *models.Response, tracer *logger.ExecutionTracer) {
	app.logInfo("SESSION_SAVE", "Saving session data")
	saveStep := app.stepLogger.StartStep(logger.ComponentCLI, "saving_session", map[string]interface{}{
		"query_id":    query.ID,
		"response_id": response.ID,
	})

	if err := app.sessionManager.SaveQuery(query, response); err != nil {
		app.logError("SESSION_SAVE", "Failed to save session data", err)
		app.stepLogger.FailStep(saveStep, err)
	} else {
		app.logSuccess("SESSION_SAVE", "Session data saved successfully")
		app.stepLogger.CompleteStep(saveStep, "Session data saved")
	}
}

// RunFullReindexWithProgress runs full reindexing with comprehensive progress logging
func (app *CLIApplication) RunFullReindexWithProgress(progressCallback func(display.IndexingProgress)) error {
	app.logInfo("FULL_REINDEXING", "Starting full reindexing with progress tracking")

	ctx := context.Background()
	return app.indexer.StartFullReindexingWithProgress(ctx, func(progress display.IndexingProgress) {
		app.logInfo("REINDEXING_PROGRESS", fmt.Sprintf("Progress: %d/%d files, %d functions, %d types",
			progress.ProcessedFiles, progress.TotalFiles, progress.FunctionsFound, progress.TypesFound))
		progressCallback(progress)
	})
}

// RunIndexingWithProgress runs indexing with comprehensive progress logging
func (app *CLIApplication) RunIndexingWithProgress(progressCallback func(display.IndexingProgress)) error {
	app.logInfo("INDEXING", "Starting code indexing with progress tracking")

	ctx := context.Background()
	return app.indexer.StartIndexingWithProgress(ctx, func(progress display.IndexingProgress) {
		app.logInfo("INDEXING_PROGRESS", fmt.Sprintf("Progress: %d/%d files, %d functions, %d types",
			progress.ProcessedFiles, progress.TotalFiles, progress.FunctionsFound, progress.TypesFound))
		progressCallback(progress)
	})
}

// GetIndexedFiles returns list of indexed files with logging
func (app *CLIApplication) GetIndexedFiles() ([]string, error) {
	app.logInfo("GET_FILES", "Retrieving indexed files from storage")

	files, err := app.indexer.GetIndexedFiles()
	if err != nil {
		app.logError("GET_FILES", "Failed to retrieve indexed files", err)
		return nil, err
	}

	app.logSuccess("GET_FILES", fmt.Sprintf("Retrieved %d indexed files", len(files)))
	return files, nil
}

// Close gracefully shuts down the application
func (app *CLIApplication) Close() error {
	app.logInfo("CLI_SHUTDOWN", "Shutting down CLI application")

	if app.stepLogger != nil {
		app.stepLogger.LogInfo(logger.ComponentCLI, "Application shutdown initiated")
		app.stepLogger.Close()
	}

	if app.storage != nil {
		app.storage.Close()
	}

	app.logSuccess("CLI_SHUTDOWN", "Application shutdown completed")
	return nil
}

// Enhanced configuration loading with logging
func loadConfig() (*Config, error) {
	fmt.Printf("📋 Loading application configuration...\n")

	// Set defaults
	viper.SetDefault("project_root", ".")
	viper.SetDefault("sqlite_db_path", "storage/useq.db")
	viper.SetDefault("log_level", "debug")
	viper.SetDefault("enable_step_logging", true)
	viper.SetDefault("debug_mode", true)

	config := &Config{
		ProjectRoot:       viper.GetString("project_root"),
		DatabasePath:      viper.GetString("sqlite_db_path"),
		LogLevel:          viper.GetString("log_level"),
		EnableStepLogging: viper.GetBool("enable_step_logging"),
		DebugMode:         viper.GetBool("debug_mode"),
		IndexedExtensions: []string{".go", ".mod", ".sum"},
		ExcludedDirs:      []string{"vendor", "node_modules", ".git", "bin", "build", "dist"},
		AIProviders: llm.AIProvidersConfig{
			Primary:       "openai",
			FallbackOrder: []string{"gemini", "cohere", "claude"},
			OpenAI: llm.ProviderConfig{
				APIKey:      os.Getenv("OPENAI_API_KEY"),
				Model:       "gpt-4-turbo-preview",
				MaxTokens:   4000,
				Temperature: 0.1,
				Timeout:     30 * time.Second,
			},
		},
		Performance: PerformanceConfig{
			MaxFileSize:        10 * 1024 * 1024, // 10MB
			IndexingBatchSize:  100,
			MaxParallelWorkers: 4,
			CacheEnabled:       true,
			CacheTTL:           time.Hour,
		},
		VectorDB: VectorDBConfig{
			URL:            getEnvOrDefault("QDRANT_URL", "localhost:6333"),
			APIKey:         os.Getenv("QDRANT_API_KEY"),
			CollectionName: "code_embeddings",
			Dimension:      1536,
		},
	}

	return config, nil
}

// Utility function
func getEnvOrDefault(envVar, defaultValue string) string {
	if value := os.Getenv(envVar); value != "" {
		return value
	}
	return defaultValue
}

// File-based logging functions (write to step logger files)
func (app *CLIApplication) logStep(component, message string) {
	if app.stepLogger != nil {
		app.stepLogger.LogInfo(logger.Component(strings.ToUpper(component)), message)
	}
}

func (app *CLIApplication) logInfo(component, message string) {
	if app.stepLogger != nil {
		app.stepLogger.LogInfo(logger.Component(strings.ToUpper(component)), message)
	}
}

func (app *CLIApplication) logSuccess(component, message string, details ...interface{}) {
	if app.stepLogger != nil {
		if len(details) > 0 && details[0] != nil {
			app.stepLogger.LogInfo(logger.Component(strings.ToUpper(component)),
				fmt.Sprintf("✅ %s - Details: %+v", message, details[0]))
		} else {
			app.stepLogger.LogInfo(logger.Component(strings.ToUpper(component)),
				fmt.Sprintf("✅ %s", message))
		}
	}
}

func (app *CLIApplication) logError(component, message string, err error) {
	if app.stepLogger != nil {
		app.stepLogger.LogError(logger.Component(strings.ToUpper(component)), message, err)
	}
}

func (app *CLIApplication) logWarning(component, message string) {
	if app.stepLogger != nil {
		app.stepLogger.LogInfo(logger.Component(strings.ToUpper(component)),
			fmt.Sprintf("⚠️ %s", message))
	}
}

// Keep minimal console output for user experience
func (app *CLIApplication) showProgress(component, message string) {
	fmt.Printf("🔄 [%s] %s\n", component, message)
}

func (app *CLIApplication) showSuccess(component, message string) {
	fmt.Printf("✅ [%s] %s\n", component, message)
}

func (app *CLIApplication) showError(component, message string) {
	fmt.Printf("❌ [%s] %s\n", component, message)
}

// LoggerAdapter adapts StepLogger to agents.Logger interface
type LoggerAdapter struct {
	stepLogger *logger.StepLogger
}

func (l *LoggerAdapter) Info(message string, fields ...interface{}) {
	l.stepLogger.LogInfo(logger.ComponentAgent, message, nil)
}

func (l *LoggerAdapter) Error(message string, fields ...interface{}) {
	l.stepLogger.LogError(logger.ComponentAgent, message, fmt.Errorf("%v", fields))
}

func (l *LoggerAdapter) Debug(message string, fields ...interface{}) {
	l.stepLogger.LogInfo(logger.ComponentAgent, "[DEBUG] "+message, nil)
}

func (l *LoggerAdapter) Warn(message string, fields ...interface{}) {
	l.stepLogger.LogInfo(logger.ComponentAgent, "[WARN] "+message, nil)
}

func (l *LoggerAdapter) Fatal(message string, fields ...interface{}) {
	l.stepLogger.LogError(logger.ComponentAgent, "[FATAL] "+message, fmt.Errorf("%v", fields))
}
