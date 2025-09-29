// Update your main.go to include comprehensive logging
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"

	"github.com/yourusername/useq-ai-assistant/internal/app"
	"github.com/yourusername/useq-ai-assistant/models"
)

var (
	version    = "1.0.0"
	buildTime  = "unknown"
	gitCommit  = "unknown"
)

func main() {
	// Initialize application
	fmt.Printf("🤖 useQ AI Assistant v%s\n", version)
	fmt.Printf("🔄 Initializing...\n")
	
	application, err := app.New()
	if err != nil {
		fmt.Printf("❌ Failed to initialize application: %v\n", err)
		os.Exit(1)
	}
	defer application.Close()

	fmt.Printf("✅ Application ready\n")
	showWelcome()

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalCh
		fmt.Println("\n👋 Gracefully shutting down useQ AI Assistant...")
		cancel()
		os.Exit(0)
	}()

	// Start interactive CLI
	runInteractiveCLI(ctx, application)
}

func runInteractiveCLI(ctx context.Context, app *app.Application) {
	promptColor := color.New(color.FgCyan, color.Bold)
	
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Show prompt
			promptColor.Print("useQ> ")
			
			// Read input
			var input string
			fmt.Scanln(&input)
			
			if input == "" {
				continue
			}
			
			// Handle special commands
			switch strings.ToLower(input) {
			case "quit", "exit", "q":
				fmt.Println("👋 Goodbye!")
				return
			case "help", "h":
				showHelp()
				continue
			case "index":
				runIndexing(app)
				continue
			case "files":
				showIndexedFiles(app)
				continue
			}
			
			// Process query
			processQuery(ctx, app, input)
		}
	}
}

func processQuery(ctx context.Context, app *app.Application, input string) {
	queryID := fmt.Sprintf("query_%d", time.Now().UnixNano())

	query := &models.Query{
		ID:          queryID,
		UserInput:   input,
		Language:    "go",
		Timestamp:   time.Now(),
		ProjectRoot: ".",
		Type:        determineQueryType(input),
	}

	response, err := app.ProcessQuery(ctx, query)
	if err != nil {
		color.Red("❌ Error: %v\n", err)
		return
	}

	displayResponse(response)
}

func showIndexedFiles(app *app.Application) {
	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)

	cyan.Println("📁 Indexed Files:")
	fmt.Println(strings.Repeat("─", 50))

	files, err := app.GetIndexedFiles()
	if err != nil {
		color.Red("❌ Error retrieving indexed files: %v", err)
		return
	}

	if len(files) == 0 {
		yellow.Println("📭 No files indexed yet")
		fmt.Println("Run 'index' to populate the database")
		return
	}

	for i, file := range files {
		fmt.Printf("  %d. %s\n", i+1, file)
	}

	fmt.Printf("\n📊 Total: %d files indexed\n", len(files))
}

func runIndexing(app *app.Application) {
	fmt.Println("🔄 Starting indexing...")
	err := app.RunIndexing(context.Background())
	if err != nil {
		color.Red("❌ Indexing failed: %v", err)
		return
	}
	fmt.Println("✅ Indexing completed")
}

func determineQueryType(input string) models.QueryType {
	input = strings.ToLower(input)
	
	if strings.Contains(input, "search") || strings.Contains(input, "find") {
		return models.QueryTypeSearch
	}
	if strings.Contains(input, "create") || strings.Contains(input, "generate") {
		return models.QueryTypeGeneration
	}
	if strings.Contains(input, "explain") || strings.Contains(input, "what") {
		return models.QueryTypeExplanation
	}
	
	return models.QueryTypeSearch // Default
}

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
		color.New(color.FgYellow).Printf("\n📝 Generated Code (%s):\n", response.Content.Code.Language)
		fmt.Println(response.Content.Code.Code)
	}

	// Show token usage and timing
	fmt.Printf("\n📊 Execution: %v | Agent: %s | Quality: %.1f%%\n",
		response.Metadata.GenerationTime.Truncate(time.Millisecond),
		response.AgentUsed,
		response.Metadata.Confidence*100)

	fmt.Println()
}

func showWelcome() {
	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)

	fmt.Println()
	cyan.Println("🤖 useQ AI Assistant")
	fmt.Printf("Version: %s\n", version)
	fmt.Println(strings.Repeat("─", 50))

	yellow.Println("🎯 Your Project-Specific AI Code Assistant")
	fmt.Println("• Search your codebase")
	fmt.Println("• Generate code with AI")
	fmt.Println("• Get explanations and help")
	fmt.Println()

	fmt.Println("💡 Commands:")
	fmt.Println("  help    - Show help")
	fmt.Println("  index   - Index project files")
	fmt.Println("  files   - Show indexed files")
	fmt.Println("  quit    - Exit")
	fmt.Println()
}

func showHelp() {
	fmt.Println("\n🤖 useQ AI Assistant - Commands")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println("  help     - Show this help")
	fmt.Println("  index    - Index project files")
	fmt.Println("  files    - Show indexed files")
	fmt.Println("  quit     - Exit application")
	fmt.Println()
}