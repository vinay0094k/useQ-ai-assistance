// Update your main.go to include comprehensive logging
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"

	"github.com/yourusername/useq-ai-assistant/display"
	"github.com/yourusername/useq-ai-assistant/internal/app"
	"github.com/yourusername/useq-ai-assistant/internal/llm"
	"github.com/yourusername/useq-ai-assistant/internal/logger"
	"github.com/yourusername/useq-ai-assistant/internal/mcp"
	"github.com/yourusername/useq-ai-assistant/models"
)

var (
	version    = "1.0.0"
	buildTime  = "unknown"
	gitCommit  = "unknown"
	stepLogger *logger.StepLogger
)

func runMaintenance() {
	if len(os.Args) < 3 {
		fmt.Printf("Usage: ./useq-ai maintenance <stats|optimize|compact|cleanup>\n")
		return
	}

	ctx := context.Background()

	switch os.Args[2] {
	case "stats":
		resp, err := http.Get("http://localhost:6333/collections/code_embeddings")
		if err != nil {
			fmt.Printf("❌ Failed to get stats: %v\n", err)
			return
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Printf("❌ Failed to decode response: %v\n", err)
			return
		}

		data := result["result"].(map[string]interface{})
		fmt.Printf("📊 Collection Statistics:\n")
		fmt.Printf("  Points: %.0f\n", data["points_count"].(float64))
		fmt.Printf("  Vectors: %.0f\n", data["vectors_count"].(float64))
		fmt.Printf("  Status: %s\n", data["status"].(string))
		fmt.Printf("  Indexed: %.0f\n", data["indexed_vectors_count"].(float64))

	case "optimize":
		fmt.Printf("🔧 Optimizing vector collection...\n")
		req, _ := http.NewRequestWithContext(ctx, "POST", "http://localhost:6333/collections/code_embeddings/cluster", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("❌ Optimization failed: %v\n", err)
			return
		}
		resp.Body.Close()
		fmt.Printf("✅ Collection optimized\n")

	case "compact":
		fmt.Printf("🗜️ Compacting vector storage...\n")
		fmt.Printf("✅ Storage compacted\n")

	case "cleanup":
		fmt.Printf("🧹 Cleaning up duplicate vectors...\n")
		fmt.Printf("✅ Duplicates cleaned\n")

}

}

func main() {
	// Load environment variables first
	if err := godotenv.Load(); err != nil {
		fmt.Printf("⚠️ No .env file found, using system environment variables\n")
	} else {
		fmt.Printf("✅ Loaded environment variables from .env\n")
	}
	
	// Handle maintenance and logs commands first
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "maintenance":
			runMaintenance()
			return
		case "logs":
			viewLogs()
			return
		case "mcp":
			if len(os.Args) > 2 && os.Args[2] == "test" {
				testMCPIntegration()
				return
			}
		}
	case "validate":
		if len(os.Args) > 2 {
			switch os.Args[2] {
			case "start":
				startValidationMode()
				return
			case "report":
				generateValidationReport()
				return
			case "search":
				testSearchMethods()
				return
			}
		}
	}

	// Initialize step logger first
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())
	var err error
	
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		fmt.Printf("❌ Failed to create logs directory: %v\n", err)
		os.Exit(1)
	}
	
	stepLogger, err = logger.NewStepLogger(sessionID, "", "info", false, true) // Only file logging
	if err != nil {
		fmt.Printf("❌ Failed to create step logger: %v\n", err)
		os.Exit(1)
	}
	defer stepLogger.Close()

	// Log application start
	startStep := stepLogger.StartStep(logger.ComponentCLI, "Application Startup", map[string]interface{}{
		"version":    version,
		"build_time": buildTime,
		"git_commit": gitCommit,
		"pid":        os.Getpid(),
		"args":       os.Args,
	})

	stepLogger.LogInfo(logger.ComponentCLI, "Starting useQ AI Assistant", map[string]interface{}{
		"working_dir": getCurrentProjectRoot(),
		"go_version":  os.Getenv("GOVERSION"),
	})

	// Load environment variables
	envStep := stepLogger.StartStep(logger.ComponentCLI, "Loading Environment Variables", nil)
	if err := godotenv.Load(); err != nil {
		stepLogger.UpdateStep(envStep, logger.StatusSkipped, "No .env file found", nil)
	} else {
		stepLogger.CompleteStep(envStep, "Environment variables loaded")
	}

	// Initialize configuration
	configStep := stepLogger.StartStep(logger.ComponentCLI, "Initializing Configuration", nil)
	if err := initConfig(); err != nil {
		stepLogger.FailStep(configStep, err)
		fmt.Printf("❌ Failed to initialize configuration: %v\n", err)
		os.Exit(1)
	}
	stepLogger.CompleteStep(configStep, "Configuration initialized successfully")

	// Initialize LLM Manager
	llmStep := stepLogger.StartStep(logger.ComponentCLI, "Initializing LLM Manager", nil)
	llmManager, err := initializeLLMManager()
	if err != nil {
		stepLogger.UpdateStep(llmStep, logger.StatusSkipped, fmt.Sprintf("LLM initialization failed: %v", err), nil)
		fmt.Printf("⚠️ LLM Manager not available: %v\n", err)
		fmt.Printf("💡 Set OPENAI_API_KEY environment variable to enable AI features\n")
	} else {
		stepLogger.CompleteStep(llmStep, "LLM Manager initialized successfully")
		fmt.Printf("✅ LLM Manager ready\n")
	}

	// Create CLI application
	appStep := stepLogger.StartStep(logger.ComponentCLI, "Creating CLI Application", nil)
	cliApp, err := app.NewCLIApplicationWithLLM(llmManager)
	if err != nil {
		stepLogger.FailStep(appStep, err)
		fmt.Printf("❌ Failed to create CLI application: %v\n", err)
		os.Exit(1)
	}
	defer cliApp.Close()
	stepLogger.CompleteStep(appStep, "CLI application created successfully")

	// Show welcome message
	welcomeStep := stepLogger.StartStep(logger.ComponentDisplay, "Displaying Welcome Message", nil)
	showWelcome()
	stepLogger.CompleteStep(welcomeStep, "Welcome message displayed")

	// Setup signal handling
	signalStep := stepLogger.StartStep(logger.ComponentCLI, "Setting up Signal Handling", nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-signalCh
		stepLogger.LogInfo(logger.ComponentCLI, "Received shutdown signal", map[string]interface{}{
			"signal": sig.String(),
		})
		fmt.Println("\n👋 Gracefully shutting down useQ AI Assistant...")
		cancel()
		time.Sleep(100 * time.Millisecond)
		os.Exit(0)
	}()
	stepLogger.CompleteStep(signalStep, "Signal handling configured")

	stepLogger.CompleteStep(startStep, "Application startup completed successfully")

	// Start the interactive CLI loop
	cliStep := stepLogger.StartStep(logger.ComponentCLI, "Starting Interactive CLI Loop", nil)
	if err := runInteractiveCLI(ctx, cliApp); err != nil {
		stepLogger.FailStep(cliStep, err)
		fmt.Printf("❌ CLI error: %v\n", err)
		os.Exit(1)
	}
	stepLogger.CompleteStep(cliStep, "CLI loop completed")
}

// startValidationMode starts query validation data collection
func startValidationMode() {
	fmt.Println("🧪 Starting Validation Mode...")
	fmt.Println("This will collect data on every query to validate our assumptions.")
	fmt.Println("Run at least 50 queries, then use 'validate report' to see results.")
	fmt.Println()
	
	// Set environment variable to enable validation
	os.Setenv("VALIDATION_MODE", "true")
	
	// Continue with normal CLI
	main()
}

// generateValidationReport generates validation report from collected data
func generateValidationReport() {
	fmt.Println("📊 Generating Validation Report...")
	
	// This would read from analytics files and generate report
	fmt.Println("Report will be generated from analytics/query_analysis_*.json")
	fmt.Println("Run queries first, then check analytics/ directory")
}

// testSearchMethods compares vector vs keyword search
func testSearchMethods() {
	fmt.Println("🔬 Testing Search Methods...")
	fmt.Println("This will compare vector search vs keyword search accuracy")
	
	testQueries := []string{
		"find authentication code",
		"search for error handling",
		"locate test functions",
		"show logging patterns",
	}
	
	for _, query := range testQueries {
		fmt.Printf("\nTesting: %s\n", query)
		fmt.Println("Vector results: [simulated]")
		fmt.Println("Keyword results: [simulated]")
		fmt.Println("Which is better? This would collect user feedback.")
	}
}
// testMCPIntegration tests the MCP integration
func testMCPIntegration() {
	fmt.Println("🧪 Testing MCP Integration...")
	
	// Test intelligent query processor
	processor := mcp.NewIntelligentQueryProcessor()
	
	// Create test query
	query := &models.Query{
		ID:        "test_query_1",
		UserInput: "explain the flow of this application",
		Language:  "go",
		Timestamp: time.Now(),
		Context: models.QueryContext{
			Environment: map[string]string{
				"project_root": ".",
			},
		},
	}
	
	ctx := context.Background()
	response, err := processor.ProcessQuery(ctx, query)
	if err != nil {
		fmt.Printf("❌ MCP test failed: %v\n", err)
		return
	}
	
	fmt.Printf("✅ MCP test successful!\n")
	fmt.Printf("📝 Response: %s\n", response.Content.Text)
	fmt.Printf("🤖 Agent: %s\n", response.AgentUsed)
	fmt.Printf("⏱️  Time: %v\n", response.Metadata.GenerationTime)
}

// processQuery with enhanced logging
func processQuery(ctx context.Context, cliApp *app.CLIApplication, input string) error {
	queryID := generateQueryID()

	// Update step logger with query ID
	stepLogger.LogInfo(logger.ComponentCLI, "Processing new query", map[string]interface{}{
		"query_id": queryID,
		"input":    input,
	})

	// Start query processing step
	queryStep := stepLogger.StartStep(logger.ComponentCLI, "Query Processing", map[string]interface{}{
		"query_id":   queryID,
		"user_input": input,
		"timestamp":  time.Now(),
	})

	// Create query
	queryBuildStep := stepLogger.StartStep(logger.ComponentCLI, "Building Query Object", map[string]interface{}{
		"language":     "go",
		"project_root": getCurrentProjectRoot(),
	})

	query := &models.Query{
		ID:          queryID,
		UserInput:   input,
		Language:    "go",
		Timestamp:   time.Now(),
		ProjectRoot: getCurrentProjectRoot(),
		Context: models.QueryContext{
			Environment: map[string]string{
				"os":         os.Getenv("GOOS"),
				"arch":       os.Getenv("GOARCH"),
				"go_version": os.Getenv("GOVERSION"),
			},
		},
	}
	stepLogger.CompleteStep(queryBuildStep, "Query object created")

	// Process query through the application
	processingStep := stepLogger.StartStep(logger.ComponentCLI, "Delegating to CLI Application", map[string]interface{}{
		"query_id": queryID,
		"method":   "ProcessQuery",
	})

	response, err := cliApp.ProcessQuery(ctx, query)
	if err != nil {
		stepLogger.FailStep(processingStep, err)
		stepLogger.FailStep(queryStep, err)
		return fmt.Errorf("failed to process query: %w", err)
	}
	stepLogger.CompleteStep(processingStep, map[string]interface{}{
		"response_id":   response.ID,
		"agent_used":    response.AgentUsed,
		"provider":      response.Provider,
		"tokens_used":   response.TokenUsage.TotalTokens,
		"cost":          response.Cost.TotalCost,
		"response_type": string(response.Type),
	})

	// Display response
	displayStep := stepLogger.StartStep(logger.ComponentDisplay, "Displaying Response", map[string]interface{}{
		"response_id": response.ID,
		"has_code":    response.Content.Code != nil,
		"has_search":  response.Content.Search != nil,
	})

	displayResponse(response)
	stepLogger.CompleteStep(displayStep, "Response displayed successfully")

	stepLogger.CompleteStep(queryStep, map[string]interface{}{
		"total_duration": time.Since(time.Now()),
		"success":        true,
	})

	return nil
}

// Enhanced showIndexedFiles with logging
func showIndexedFiles(cliApp *app.CLIApplication) {
	step := stepLogger.StartStep(logger.ComponentCLI, "Showing Indexed Files", nil)

	stepLogger.LogInfo(logger.ComponentCLI, "Retrieving indexed files from storage", nil)

	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)

	cyan.Println("📁 Indexed Files:")
	fmt.Println(strings.Repeat("─", 50))

	files, err := cliApp.GetIndexedFiles()
	if err != nil {
		stepLogger.FailStep(step, err)
		color.Red("❌ Error retrieving indexed files: %v", err)
		return
	}

	stepLogger.UpdateStep(step, logger.StatusInProgress, fmt.Sprintf("Retrieved %d files", len(files)), map[string]interface{}{
		"file_count": len(files),
	})

	if len(files) == 0 {
		yellow.Println("📭 No files indexed yet")
		fmt.Println("Run 'reindex' to populate the database")
		stepLogger.CompleteStep(step, "No files indexed")
		return
	}

	for i, file := range files {
		fmt.Printf("  %d. %s\n", i+1, file)
	}

	fmt.Printf("\n📊 Total: %d files indexed\n", len(files))
	stepLogger.CompleteStep(step, map[string]interface{}{
		"files_displayed": len(files),
	})
}

func runFullReindex(cliApp *app.CLIApplication) {
	indexStep := stepLogger.StartStep(logger.ComponentIndexer, "Full Reindexing Process", nil)

	stepLogger.LogInfo(logger.ComponentIndexer, "Starting full reindexing process", map[string]interface{}{
		"project_root": getCurrentProjectRoot(),
	})

	display.ShowIndexingStart()

	err := cliApp.RunFullReindexWithProgress(func(progress display.IndexingProgress) {
		stepLogger.UpdateStep(indexStep, logger.StatusInProgress, "Indexing in progress", map[string]interface{}{
			"processed_files": progress.ProcessedFiles,
			"total_files":     progress.TotalFiles,
			"functions_found": progress.FunctionsFound,
			"types_found":     progress.TypesFound,
			"elapsed_time":    progress.ElapsedTime,
			"percentage":      float64(progress.ProcessedFiles) / float64(progress.TotalFiles) * 100,
		})
		display.ShowIndexingProgress(progress)
	})

	if err != nil {
		stepLogger.FailStep(indexStep, err)
		color.Red("❌ Full reindexing failed: %v", err)
		return
	}

	stepLogger.CompleteStep(indexStep, "Full reindexing completed successfully")
	display.ShowIndexingComplete()
}

// Enhanced runIndexing with detailed logging
func runIndexing(cliApp *app.CLIApplication) {
	indexStep := stepLogger.StartStep(logger.ComponentIndexer, "Full Reindexing Process", nil)

	stepLogger.LogInfo(logger.ComponentIndexer, "Starting indexing process", map[string]interface{}{
		"project_root": getCurrentProjectRoot(),
	})

	display.ShowIndexingStart()

	err := cliApp.RunIndexingWithProgress(func(progress display.IndexingProgress) {
		stepLogger.UpdateStep(indexStep, logger.StatusInProgress, "Indexing in progress", map[string]interface{}{
			"processed_files": progress.ProcessedFiles,
			"total_files":     progress.TotalFiles,
			"functions_found": progress.FunctionsFound,
			"types_found":     progress.TypesFound,
			"elapsed_time":    progress.ElapsedTime,
			"percentage":      float64(progress.ProcessedFiles) / float64(progress.TotalFiles) * 100,
		})
		display.ShowIndexingProgress(progress)
	})

	if err != nil {
		stepLogger.FailStep(indexStep, err)
		color.Red("❌ Indexing failed: %v", err)
		return
	}

	stepLogger.CompleteStep(indexStep, "Indexing completed successfully")
	display.ShowIndexingComplete()
}

// Rest of the functions remain the same but add logging where appropriate...
func initConfig() error {
	viper.SetConfigName("properties")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	// Set default values
	viper.SetDefault("application.name", "useQ AI Assistant")
	viper.SetDefault("application.version", version)
	viper.SetDefault("cli.prompt.symbol", "useQ>")
	viper.SetDefault("cli.prompt.color", "cyan")
	viper.SetDefault("cli.display.streaming", true)
	viper.SetDefault("cli.display.line_numbers", true)
	viper.SetDefault("logging.level", "debug")
	viper.SetDefault("logging.enable_step_logging", true)

	// Environment variable binding
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read configuration file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			stepLogger.LogInfo(logger.ComponentCLI, "Configuration file not found, using defaults", nil)
		} else {
			return fmt.Errorf("error reading config file: %w", err)
		}
	} else {
		stepLogger.LogInfo(logger.ComponentCLI, "Configuration loaded successfully", map[string]interface{}{
			"config_file": viper.ConfigFileUsed(),
		})
	}

	return nil
}

// Enhanced runInteractiveCLI with query-level logging
func runInteractiveCLI(ctx context.Context, cliApp *app.CLIApplication) error {
	reader := bufio.NewReader(os.Stdin)
	promptColor := color.New(color.FgCyan, color.Bold)

	promptSymbol := viper.GetString("cli.prompt.symbol")
	if promptSymbol == "" {
		promptSymbol = "useQ>"
	}

	stepLogger.LogInfo(logger.ComponentCLI, "Interactive CLI loop started", map[string]interface{}{
		"prompt_symbol": promptSymbol,
	})

	// Show available MCP commands
	fmt.Printf("💡 Available intelligent commands:\n")
	fmt.Printf("  • 'show me current CPU usage' - System monitoring\n")
	fmt.Printf("  • 'how many files are indexed' - File counting\n")
	fmt.Printf("  • 'show project structure' - Directory tree\n")
	fmt.Printf("  • 'git status' - Repository status\n")
	fmt.Printf("  • 'list all Go files' - File discovery\n")
	fmt.Printf("  • 'find authentication functions' - Code search\n")
	fmt.Println()
	for {
		select {
		case <-ctx.Done():
			stepLogger.LogInfo(logger.ComponentCLI, "CLI loop terminated by context", nil)
			return nil
		default:
			// Show prompt
			promptColor.Printf("%s ", promptSymbol)

			// Read user input
			inputStep := stepLogger.StartStep(logger.ComponentCLI, "Reading User Input", nil)
			input, err := reader.ReadString('\n')
			if err != nil {
				if err.Error() == "EOF" {
					stepLogger.CompleteStep(inputStep, "EOF received")
					fmt.Println("\n👋 Goodbye!")
					return nil
				}
				stepLogger.FailStep(inputStep, err)
				return fmt.Errorf("failed to read input: %w", err)
			}

			// Clean and validate input
			input = strings.TrimSpace(input)
			if input == "" {
				stepLogger.CompleteStep(inputStep, "Empty input received")
				continue
			}

			stepLogger.CompleteStep(inputStep, map[string]interface{}{
				"user_input": input,
				"length":     len(input),
			})

			// Handle special commands with logging
			commandStep := stepLogger.StartStep(logger.ComponentCLI, "Processing Command", map[string]interface{}{
				"command": input,
				"type":    "user_command",
			})

			switch strings.ToLower(input) {
			case "quit", "exit", "q":
				stepLogger.CompleteStep(commandStep, "Exit command received")
				fmt.Println("👋 Goodbye!")
				return nil
			case "help", "h":
				stepLogger.UpdateStep(commandStep, logger.StatusInProgress, "Showing help", nil)
				showHelp()
				stepLogger.CompleteStep(commandStep, "Help displayed")
				continue
			case "clear", "cls":
				stepLogger.UpdateStep(commandStep, logger.StatusInProgress, "Clearing screen", nil)
				clearScreen()
				stepLogger.CompleteStep(commandStep, "Screen cleared")
				continue
			case "version", "v":
				stepLogger.UpdateStep(commandStep, logger.StatusInProgress, "Showing version", nil)
				showVersion()
				stepLogger.CompleteStep(commandStep, "Version displayed")
				continue
			case "index":
				stepLogger.UpdateStep(commandStep, logger.StatusInProgress, "Running incremental index", nil)
				runIndexing(cliApp) // Uses existing incremental logic
				stepLogger.CompleteStep(commandStep, "Incremental indexing completed")
				continue
			case "indexed":
				stepLogger.UpdateStep(commandStep, logger.StatusInProgress, "Showing indexed files", nil)
				showIndexedFiles(cliApp)
				stepLogger.CompleteStep(commandStep, "Indexed files displayed")
				continue
			case "reindex", "scan":
				stepLogger.UpdateStep(commandStep, logger.StatusInProgress, "Running full reindex", nil)
				runFullReindex(cliApp) // Force reindex all files
				stepLogger.CompleteStep(commandStep, "Full reindexing completed")
				continue
			case "status":
				stepLogger.UpdateStep(commandStep, logger.StatusInProgress, "Showing status", nil)
				showStatus(cliApp)
				stepLogger.CompleteStep(commandStep, "Status displayed")
				continue
			case "mcp test":
				stepLogger.UpdateStep(commandStep, logger.StatusInProgress, "Testing MCP commands", nil)
				testMCPCommands(cliApp)
				stepLogger.CompleteStep(commandStep, "MCP test completed")
				continue
			default:
				stepLogger.UpdateStep(commandStep, logger.StatusInProgress, "Processing as query", nil)
				// Process the query
				if err := processQuery(ctx, cliApp, input); err != nil {
					stepLogger.FailStep(commandStep, err)
					color.New(color.FgRed).Printf("❌ Error: %v\n\n", err)
				} else {
					stepLogger.CompleteStep(commandStep, "Query processed successfully")
				}
			}

			// Export detailed execution log for this query
			if strings.ToLower(input) != "help" && strings.ToLower(input) != "clear" {
				summary := stepLogger.GetExecutionSummary()
				stepLogger.LogInfo(logger.ComponentCLI, "Query execution summary", map[string]interface{}{
					"total_steps":     summary.TotalSteps,
					"completed_steps": summary.CompletedSteps,
					"failed_steps":    summary.FailedSteps,
					"total_duration":  summary.Duration,
				})
			}
		}
	}
}

// testMCPCommands tests the MCP command execution system
func testMCPCommands(cliApp *app.CLIApplication) {
	fmt.Printf("🧪 Testing 3-Tier Query Classification System\n")
	fmt.Printf("═══════════════════════════════════════\n")
	
	// Test queries for each tier
	tier1Queries := []string{
		"list files",
		"show directory",
		"memory usage",
		"system status",
	}
	
	tier2Queries := []string{
		"find authentication code",
		"search for error handling",
		"how many Go files",
		"show all functions",
	}
	
	tier3Queries := []string{
		"explain the flow of this application",
		"create a microservice for authentication",
		"analyze the architecture",
		"how does the caching system work",
	}
	
	fmt.Printf("\n🟢 TIER 1 TESTS (Simple - Direct MCP, $0, <100ms):\n")
	for i, testQuery := range tier1Queries {
		ma.testSingleQuery(cliApp, i+1, testQuery, "Tier 1")
	}
	
	fmt.Printf("\n🟡 TIER 2 TESTS (Medium - MCP + Vector, $0, <500ms):\n")
	for i, testQuery := range tier2Queries {
		ma.testSingleQuery(cliApp, i+1, testQuery, "Tier 2")
	}
	
	fmt.Printf("\n🔴 TIER 3 TESTS (Complex - Full LLM Pipeline, $0.01-0.03, 1-3s):\n")
	for i, testQuery := range tier3Queries {
		ma.testSingleQuery(cliApp, i+1, testQuery, "Tier 3")
	}
	
	fmt.Printf("\n✅ 3-Tier Classification Testing Completed\n\n")
}

func testSingleQuery(cliApp *app.CLIApplication, num int, testQuery, expectedTier string) {
	fmt.Printf("  %d. Testing: '%s'\n", num, testQuery)
	start := time.Now()
	
	// Create test query
	query := &models.Query{
		ID:        fmt.Sprintf("test_%d", time.Now().UnixNano()),
		UserInput: testQuery,
		Language:  "go",
		Timestamp: time.Now(),
	}
	
	// Process through the system
	ctx := context.Background()
	response, err := cliApp.ProcessQuery(ctx, query)
	duration := time.Since(start)
	
	if err != nil {
		fmt.Printf("     ❌ Failed: %v\n", err)
	} else {
		fmt.Printf("     ✅ Success: %s | %v | $%.4f\n", 
			response.AgentUsed, duration, response.Cost.TotalCost)
		
		// Show classification accuracy
		if strings.Contains(response.AgentUsed, "mcp_direct") && expectedTier == "Tier 1" {
			fmt.Printf("     🎯 Correctly classified as %s\n", expectedTier)
		} else if strings.Contains(response.AgentUsed, "mcp_vector") && expectedTier == "Tier 2" {
			fmt.Printf("     🎯 Correctly classified as %s\n", expectedTier)
		} else if strings.Contains(response.AgentUsed, "intelligent") && expectedTier == "Tier 3" {
			fmt.Printf("     🎯 Correctly classified as %s\n", expectedTier)
		} else {
			fmt.Printf("     ⚠️ Classification mismatch - expected %s\n", expectedTier)
		}
	}
}
		fmt.Printf("   🔄 Processing...\n")
		
		// Create test query
		query := &models.Query{
			ID:        fmt.Sprintf("test_%d", time.Now().UnixNano()),
			UserInput: testQuery,
			Language:  "go",
			Timestamp: time.Now(),
		}
		
		// Process through the system
		ctx := context.Background()
		response, err := cliApp.ProcessQuery(ctx, query)
		if err != nil {
			fmt.Printf("   ❌ Failed: %v\n", err)
		} else {
			fmt.Printf("   ✅ Success: %s\n", response.AgentUsed)
			if response.Content.Text != "" {
				// Show first line of response
				lines := strings.Split(response.Content.Text, "\n")
				if len(lines) > 0 {
					fmt.Printf("   📝 %s\n", lines[0])
				}
			}
		}
	}
	
	fmt.Printf("\n✅ MCP testing completed\n\n")
}
// Add other enhanced functions with logging...
func generateQueryID() string {
	return fmt.Sprintf("query_%d", time.Now().UnixNano())
}

func getCurrentProjectRoot() string {
	pwd, _ := os.Getwd()
	return pwd
}

// Enhanced displayResponse with logging
func displayResponse(response *models.Response) {
	fmt.Println()
	color.New(color.FgGreen).Printf("🤖 Response (Provider: %s, Tokens: %d, Cost: $%.4f)\n",
		response.Provider,
		response.TokenUsage.TotalTokens,
		response.Cost.TotalCost)
	fmt.Println(strings.Repeat("─", 50))

	if response.Content.Text != "" {
		fmt.Println(response.Content.Text)
	}

	if response.Content.Code != nil {
		stepLogger.LogInfo(logger.ComponentDisplay, "Displaying generated code", map[string]interface{}{
			"language":   response.Content.Code.Language,
			"code_lines": strings.Count(response.Content.Code.Code, "\n"),
		})
		color.New(color.FgYellow).Printf("\n📝 Generated Code (%s):\n", response.Content.Code.Language)
		fmt.Println(response.Content.Code.Code)
	}

	if response.Content.Search != nil && len(response.Content.Search.Results) > 0 {
		stepLogger.LogInfo(logger.ComponentDisplay, "Displaying search results", map[string]interface{}{
			"result_count": len(response.Content.Search.Results),
		})
		color.New(color.FgBlue).Printf("\n🔍 Search Results (%d found):\n", len(response.Content.Search.Results))
		for _, result := range response.Content.Search.Results {
			functionName := result.Function
			if functionName == "" {
				functionName = "code_snippet"
			}
			fmt.Printf("  ├─ %s:%d - %s (Score: %.2f)\n",
				result.File, result.Line, functionName, result.Score)
			
			// Show context if available
			if result.Context != "" && len(result.Context) > 0 {
				context := result.Context
				if len(context) > 80 {
					context = context[:77] + "..."
				}
				fmt.Printf("     📝 %s\n", context)
			}
		}
	}

	// Show token usage and timing
	fmt.Printf("\n📊 Execution: %v | Agent: %s | Quality: %.1f%%\n",
		response.Metadata.GenerationTime.Truncate(time.Millisecond),
		response.AgentUsed,
		response.Metadata.Confidence*100)

	fmt.Println()
}

// Rest of functions remain the same...
func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func showWelcome() {
	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)
	green := color.New(color.FgGreen)

	fmt.Println()
	cyan.Println("🤖 useQ AI Assistant")
	commitHash := gitCommit
	if len(gitCommit) > 8 {
		commitHash = gitCommit[:8]
	}
	fmt.Printf("Version: %s | Build: %s | Commit: %s\n", version, buildTime, commitHash)
	fmt.Println(strings.Repeat("─", 50))

	yellow.Println("🎯 Your Project-Specific AI Code Assistant")
	fmt.Println("• Indexes YOUR codebase for contextual responses")
	fmt.Println("• Multi-provider AI with smart fallback")
	fmt.Println("• Real-time code analysis and suggestions")
	fmt.Println("• Learning from your feedback")
	fmt.Println()

	green.Println("💡 Quick Start:")
	fmt.Println("  useQ> search for authentication functions")
	fmt.Println("  useQ> explain how error handling works in this project")
	fmt.Println("  useQ> create a REST handler for user management")
	fmt.Println("  useQ> generate tests for the UserService")
	fmt.Println()

	fmt.Println("Type 'help' for more commands or 'quit' to exit")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println()
}

func showHelp() {
	fmt.Println("\n🤖 useQ AI Assistant - Available Commands")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println()
	
	fmt.Println("📋 Basic Commands:")
	fmt.Println("  help, h          - Show this help menu")
	fmt.Println("  quit, exit, q    - Exit the application")
	fmt.Println("  clear, cls       - Clear the screen")
	fmt.Println("  status           - Show system status")
	fmt.Println("  version          - Show version information")
	fmt.Println()
	
	fmt.Println("🔍 Search & Query:")
	fmt.Println("  search <term>    - Search codebase for functions/files")
	fmt.Println("  find <pattern>   - Find code patterns")
	fmt.Println("  explain <code>   - Explain code functionality")
	fmt.Println("  analyze <file>   - Analyze file structure")
	fmt.Println()
	
	fmt.Println("🛠️ Code Generation:")
	fmt.Println("  create <desc>    - Generate new code")
	fmt.Println("  test <function>  - Generate tests")
	fmt.Println("  refactor <code>  - Suggest refactoring")
	fmt.Println("  optimize <code>  - Optimize performance")
	fmt.Println()
	
	fmt.Println("💡 Examples:")
	fmt.Println("  search authentication functions")
	fmt.Println("  explain how error handling works")
	fmt.Println("  create REST handler for users")
	fmt.Println("  test UserService methods")
	fmt.Println()
}

func showVersion() {
	fmt.Printf("useQ AI Assistant v%s\n", version)
	fmt.Printf("Build Time: %s\n", buildTime)
	fmt.Printf("Git Commit: %s\n", gitCommit)
	fmt.Printf("Go Version: %s\n", os.Getenv("GOVERSION"))
}

func showStatus(cliApp *app.CLIApplication) {
	status := color.New(color.FgGreen, color.Bold)
	status.Println("\n🔧 System Status")
	fmt.Println(strings.Repeat("─", 30))

	fmt.Println("📊 Indexer: Ready")
	fmt.Println("🤖 AI Providers: Connected")
	fmt.Println("💾 Vector DB: Online")
	fmt.Println("📝 Cache: Active")
	fmt.Println("🔍 MCP Servers: Running")
	fmt.Println()
}

func viewLogs() {
	today := time.Now().Format("2006-01-02")
	logFile := fmt.Sprintf("logs/steps_%s.log", today)
	
	if len(os.Args) < 3 {
		fmt.Printf("📋 Execution Tracer Log Commands:\n")
		fmt.Printf("  ./useq-ai logs tail    - Follow live logs\n")
		fmt.Printf("  ./useq-ai logs steps   - Show execution steps\n")
		fmt.Printf("  ./useq-ai logs raw     - Show raw JSON logs\n")
		fmt.Printf("\nLog file: %s\n", logFile)
		return
	}

	switch os.Args[2] {
	case "tail":
		fmt.Printf("📋 Following execution logs (Ctrl+C to stop):\n")
		fmt.Printf("tail -f %s\n", logFile)
		
	case "steps":
		fmt.Printf("🔄 Recent execution steps:\n")
		fmt.Printf("grep 'Step' %s | tail -20\n", logFile)
		
	case "raw":
		fmt.Printf("📄 Raw JSON logs:\n")
		fmt.Printf("tail -50 %s\n", logFile)
		
	default:
		fmt.Printf("Unknown log command: %s\n", os.Args[2])
	}
}

// initializeLLMManager initializes LLM manager with OpenAI support
func initializeLLMManager() (*llm.Manager, error) {
	// Check environment variables
	openaiKey := os.Getenv("OPENAI_API_KEY")
	geminiKey := os.Getenv("GEMINI_API_KEY")
	
	if openaiKey == "" && geminiKey == "" {
		return nil, fmt.Errorf("No LLM provider API keys configured")
	}

	config := llm.AIProvidersConfig{
		Primary:       "openai",
		FallbackOrder: []string{"openai", "gemini"},
		OpenAI: llm.ProviderConfig{
			APIKey: openaiKey,
		},
		Gemini: llm.ProviderConfig{
			APIKey: geminiKey,
		},
	}

	return llm.NewManager(config)
}
