package display

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"

	"github.com/yourusername/useq-ai-assistant/models"
)

// DisplayRenderer handles all CLI display operations
type DisplayRenderer struct {
	config    DisplayConfig
	colorizer *SyntaxHighlighter
	symbols   SymbolSet
	width     int
	height    int
}

// DisplayConfig holds display configuration
type DisplayConfig struct {
	Theme            string        `json:"theme"`
	ShowLineNumbers  bool          `json:"show_line_numbers"`
	ShowProgress     bool          `json:"show_progress"`
	StreamingEnabled bool          `json:"streaming_enabled"`
	CharDelay        time.Duration `json:"char_delay"`
	LineDelay        time.Duration `json:"line_delay"`
	MaxWidth         int           `json:"max_width"`
	IndentSize       int           `json:"indent_size"`
	EnableBorders    bool          `json:"enable_borders"`
	EnableIcons      bool          `json:"enable_icons"`
	CompactMode      bool          `json:"compact_mode"`
}

// SymbolSet defines icons/symbols for different elements
type SymbolSet struct {
	Bullet     string
	LastBullet string
	Pipe       string
	Success    string
	Error      string
	Warning    string
	Info       string
	Search     string
	Code       string
	Test       string
	Docs       string
	Debug      string
	Loading    string
	Arrow      string
	RightArrow string
}

// ColorScheme defines colors for different elements
type ColorScheme struct {
	Primary    *color.Color
	Secondary  *color.Color
	Success    *color.Color
	Warning    *color.Color
	Error      *color.Color
	Info       *color.Color
	Muted      *color.Color
	Code       CodeColors
	Border     *color.Color
	LineNumber *color.Color
}

// CodeColors defines colors for syntax highlighting
type CodeColors struct {
	Keyword  *color.Color
	String   *color.Color
	Comment  *color.Color
	Function *color.Color
	Variable *color.Color
	Number   *color.Color
	Type     *color.Color
	Operator *color.Color
}

// NewDisplayRenderer creates a new display renderer
func NewDisplayRenderer(config DisplayConfig) *DisplayRenderer {
	dr := &DisplayRenderer{
		config:    config,
		colorizer: NewSyntaxHighlighter(),
		width:     120,
		height:    30,
	}

	dr.initializeSymbols()
	dr.initializeColors()

	return dr
}

// initializeSymbols sets up display symbols
func (dr *DisplayRenderer) initializeSymbols() {
	if dr.config.EnableIcons {
		dr.symbols = SymbolSet{
			Bullet:     "├─",
			LastBullet: "└─",
			Pipe:       "│",
			Success:    "✅",
			Error:      "❌",
			Warning:    "⚠️",
			Info:       "💡",
			Search:     "🔍",
			Code:       "📝",
			Test:       "🧪",
			Docs:       "📚",
			Debug:      "🐛",
			Loading:    "🔄",
			Arrow:      "→",
			RightArrow: "▶",
		}
	} else {
		dr.symbols = SymbolSet{
			Bullet:     "├─",
			LastBullet: "└─",
			Pipe:       "│",
			Success:    "[✓]",
			Error:      "[✗]",
			Warning:    "[!]",
			Info:       "[i]",
			Search:     "[?]",
			Code:       "[C]",
			Test:       "[T]",
			Docs:       "[D]",
			Debug:      "[B]",
			Loading:    "[*]",
			Arrow:      "->",
			RightArrow: ">",
		}
	}
}

// initializeColors sets up color scheme
func (dr *DisplayRenderer) initializeColors() {
	// This would be implemented based on theme
	// For now, using a default dark theme
}

// RenderResponse renders a complete AI response with beautiful formatting
func (dr *DisplayRenderer) RenderResponse(response *models.Response) {
	dr.printHeader(response)

	// Render different content types
	if response.Content.Text != "" {
		dr.renderText(response.Content.Text)
	}

	if response.Content.Code != nil {
		dr.renderCode(response.Content.Code)
	}

	if response.Content.Search != nil {
		dr.renderSearchResults(response.Content.Search)
	}

	if len(response.Content.Files) > 0 {
		dr.renderFileChanges(response.Content.Files)
	}

	if len(response.Content.Suggestions) > 0 {
		dr.renderSuggestions(response.Content.Suggestions)
	}

	dr.printFooter(response)
}

// StreamResponse renders a streaming response character by character
func (dr *DisplayRenderer) StreamResponse(responseChan <-chan string, metadata *ResponseMetadata) {
	if !dr.config.StreamingEnabled {
		// Collect all content and render at once
		var fullContent strings.Builder
		for chunk := range responseChan {
			fullContent.WriteString(chunk)
		}
		dr.renderText(fullContent.String())
		return
	}

	// Print header
	dr.printStreamingHeader(metadata)

	// Stream content with typewriter effect
	lineBuffer := strings.Builder{}

	for chunk := range responseChan {
		for _, char := range chunk {
			fmt.Print(string(char))
			lineBuffer.WriteRune(char)

			// Add character delay for typewriter effect
			if dr.config.CharDelay > 0 {
				time.Sleep(dr.config.CharDelay)
			}

			// Handle line breaks
			if char == '\n' {
				if dr.config.LineDelay > 0 {
					time.Sleep(dr.config.LineDelay)
				}
				lineBuffer.Reset()
			}
		}
	}

	fmt.Println() // Ensure we end with a newline
}

// printHeader prints the response header with metadata
func (dr *DisplayRenderer) printHeader(response *models.Response) {
	fmt.Println()

	if dr.config.EnableBorders {
		dr.printBorder("┌", "─", "┐")
	}

	// Title line with provider and agent info
	title := fmt.Sprintf("%s AI Response", dr.symbols.RightArrow)
	providerInfo := fmt.Sprintf("Provider: %s | Agent: %s",
		response.Provider, response.AgentUsed)

	if dr.config.EnableBorders {
		fmt.Printf("│ %s %s\n",
			color.New(color.FgCyan, color.Bold).Sprint(title),
			strings.Repeat(" ", dr.width-len(title)-len(providerInfo)-4))
		fmt.Printf("│ %s %s\n",
			color.New(color.FgYellow).Sprint(providerInfo),
			strings.Repeat(" ", dr.width-len(providerInfo)-4))
	} else {
		color.New(color.FgCyan, color.Bold).Println(title)
		color.New(color.FgYellow).Println(providerInfo)
	}

	// Metadata line
	if !dr.config.CompactMode {
		metadata := fmt.Sprintf("Tokens: %d | Cost: $%.4f | Time: %v | Quality: %.1f%%",
			response.TokenUsage.TotalTokens,
			response.Cost.TotalCost,
			response.Metadata.GenerationTime.Truncate(time.Millisecond),
			response.Quality.Accuracy*100)

		if dr.config.EnableBorders {
			fmt.Printf("│ %s %s\n",
				color.New(color.FgMagenta).Sprint(metadata),
				strings.Repeat(" ", max(0, dr.width-len(metadata)-4)))
			dr.printBorder("├", "─", "┤")
		} else {
			color.New(color.FgMagenta).Println(metadata)
			fmt.Println(strings.Repeat("─", 50))
		}
	}
}

// printStreamingHeader prints header for streaming responses
func (dr *DisplayRenderer) printStreamingHeader(metadata *ResponseMetadata) {
	fmt.Println()
	title := fmt.Sprintf("%s %s Generating Response...", dr.symbols.Loading, dr.symbols.RightArrow)
	color.New(color.FgCyan, color.Bold).Println(title)

	if metadata != nil {
		info := fmt.Sprintf("Provider: %s | Estimated tokens: ~%d",
			metadata.Provider, metadata.EstimatedTokens)
		color.New(color.FgYellow).Println(info)
	}

	fmt.Println(strings.Repeat("─", 50))
	fmt.Println()
}

// renderText renders plain text with proper formatting
func (dr *DisplayRenderer) renderText(text string) {
	if text == "" {
		return
	}

	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if dr.config.ShowLineNumbers && len(lines) > 5 {
			lineNum := color.New(color.FgBlue).Sprintf("%3d │ ", i+1)
			fmt.Print(lineNum)
		} else if dr.config.EnableBorders {
			fmt.Print("│ ")
		}

		// Apply basic formatting
		formatted := dr.formatTextLine(line)
		fmt.Println(formatted)
	}

	fmt.Println()
}

// renderCode renders code with syntax highlighting and line numbers
func (dr *DisplayRenderer) renderCode(codeResp *models.CodeResponse) {
	fmt.Println()

	// Code header
	header := fmt.Sprintf("%s Generated Code (%s)", dr.symbols.Code, codeResp.Language)
	color.New(color.FgGreen, color.Bold).Println(header)

	if codeResp.Explanation != "" {
		color.New(color.FgCyan).Printf("📖 %s\n", codeResp.Explanation)
	}

	fmt.Println(strings.Repeat("─", 50))

	// Render code with syntax highlighting
	dr.renderHighlightedCode(codeResp.Code, codeResp.Language)

	// Show changes if any
	if len(codeResp.Changes) > 0 {
		fmt.Println()
		color.New(color.FgYellow, color.Bold).Println("📝 Code Changes:")

		for i, change := range codeResp.Changes {
			symbol := dr.symbols.Bullet
			if i == len(codeResp.Changes)-1 {
				symbol = dr.symbols.LastBullet
			}

			changeColor := dr.getChangeColor(change.Type)
			fmt.Printf("%s %s %s:%d-%d\n",
				symbol,
				changeColor.Sprint(strings.ToUpper(string(change.Type))),
				change.File,
				change.StartLine,
				change.EndLine)

			if change.Explanation != "" {
				fmt.Printf("   %s\n", color.New(color.FgWhite).Sprint(change.Explanation))
			}
		}
	}

	// Show tests if any
	if len(codeResp.Tests) > 0 {
		fmt.Println()
		color.New(color.FgMagenta, color.Bold).Printf("%s Generated Tests:\n", dr.symbols.Test)

		for _, test := range codeResp.Tests {
			fmt.Printf("%s %s - %s\n",
				dr.symbols.Bullet,
				color.New(color.FgCyan).Sprint(test.Name),
				test.Description)
		}
	}
}

// renderSearchResults renders search results in a structured format
func (dr *DisplayRenderer) renderSearchResults(searchResp *models.SearchResponse) {
	fmt.Println()

	// Search header
	header := fmt.Sprintf("%s Search Results (%d found)", dr.symbols.Search, searchResp.Total)
	color.New(color.FgBlue, color.Bold).Println(header)

	if searchResp.Query != "" {
		fmt.Printf("Query: %s\n", color.New(color.FgWhite).Sprint(searchResp.Query))
	}

	fmt.Printf("Time: %v\n", searchResp.TimeTaken.Truncate(time.Millisecond))
	fmt.Println(strings.Repeat("─", 50))

	if len(searchResp.Results) == 0 {
		color.New(color.FgYellow).Println("No results found.")
		return
	}

	// Render each result
	for i, result := range searchResp.Results {
		symbol := dr.symbols.Bullet
		if i == len(searchResp.Results)-1 {
			symbol = dr.symbols.LastBullet
		}

		// Main result line
		fmt.Printf("%s %s:%d",
			symbol,
			color.New(color.FgCyan).Sprint(result.File),
			result.Line)

		if result.Function != "" {
			fmt.Printf(" - %s", color.New(color.FgGreen).Sprint(result.Function))
		}

		fmt.Printf(" (Score: %.2f)\n", result.Score)

		// Context and explanation
		if result.Context != "" {
			fmt.Printf("   %s\n", color.New(color.FgWhite).Sprint(result.Context))
		}

		if result.Explanation != "" {
			fmt.Printf("   %s %s\n", dr.symbols.Info,
				color.New(color.FgYellow).Sprint(result.Explanation))
		}

		// Usage examples
		if len(result.Usage) > 0 {
			fmt.Printf("   %s Usage:\n", dr.symbols.RightArrow)
			for _, usage := range result.Usage {
				fmt.Printf("     • %s:%d - %s\n",
					usage.File, usage.Line, usage.Description)
			}
		}

		fmt.Println()
	}
}

// renderFileChanges renders file changes to be made
func (dr *DisplayRenderer) renderFileChanges(changes []models.FileChange) {
	fmt.Println()

	header := fmt.Sprintf("%s File Changes", dr.symbols.Code)
	color.New(color.FgMagenta, color.Bold).Println(header)
	fmt.Println(strings.Repeat("─", 50))

	for i, change := range changes {
		symbol := dr.symbols.Bullet
		if i == len(changes)-1 {
			symbol = dr.symbols.LastBullet
		}

		actionColor := dr.getActionColor(change.Action)
		fmt.Printf("%s %s %s\n",
			symbol,
			actionColor.Sprint(strings.ToUpper(string(change.Action))),
			color.New(color.FgCyan).Sprint(change.Path))

		// Show specific changes
		if len(change.Changes) > 0 {
			for _, codeChange := range change.Changes {
				fmt.Printf("   %s Line %d-%d: %s\n",
					dr.symbols.RightArrow,
					codeChange.StartLine,
					codeChange.EndLine,
					codeChange.Explanation)
			}
		}

		fmt.Println()
	}
}

// renderSuggestions renders AI suggestions
func (dr *DisplayRenderer) renderSuggestions(suggestions []models.Suggestion) {
	fmt.Println()

	header := fmt.Sprintf("%s Suggestions", dr.symbols.Info)
	color.New(color.FgYellow, color.Bold).Println(header)
	fmt.Println(strings.Repeat("─", 50))

	for i, suggestion := range suggestions {
		symbol := dr.symbols.Bullet
		if i == len(suggestions)-1 {
			symbol = dr.symbols.LastBullet
		}

		typeColor := dr.getSuggestionColor(suggestion.Type)
		fmt.Printf("%s %s %s (%.0f%% confidence)\n",
			symbol,
			typeColor.Sprint(strings.ToUpper(string(suggestion.Type))),
			suggestion.Title,
			suggestion.Confidence*100)

		fmt.Printf("   %s\n", suggestion.Description)

		if suggestion.Code != "" {
			fmt.Println("   Code:")
			dr.renderCodeSnippet(suggestion.Code, "go")
		}

		fmt.Println()
	}
}

// renderHighlightedCode renders code with syntax highlighting
func (dr *DisplayRenderer) renderHighlightedCode(code, language string) {
	lines := strings.Split(code, "\n")

	for i, line := range lines {
		// Line number
		lineNum := color.New(color.FgBlue).Sprintf("%3d │ ", i+1)
		fmt.Print(lineNum)

		// Highlighted code
		highlighted := dr.colorizer.Highlight(line, language)
		fmt.Println(highlighted)
	}
}

// renderCodeSnippet renders a small code snippet with highlighting
func (dr *DisplayRenderer) renderCodeSnippet(code, language string) {
	lines := strings.Split(code, "\n")

	for _, line := range lines {
		fmt.Printf("     %s\n", dr.colorizer.Highlight(line, language))
	}
}

// printFooter prints the response footer
func (dr *DisplayRenderer) printFooter(response *models.Response) {
	if dr.config.EnableBorders {
		dr.printBorder("└", "─", "┘")
	} else {
		fmt.Println(strings.Repeat("─", 50))
	}

	// Footer with feedback prompt
	if !dr.config.CompactMode {
		fmt.Println()
		feedbackText := "Rate this response (1-5) or press Enter to continue:"
		color.New(color.FgMagenta).Print(feedbackText)
		fmt.Print(" ")
	}

	fmt.Println()
}

// ShowProgress displays a progress bar for long operations
func (dr *DisplayRenderer) ShowProgress(description string, total int) *progressbar.ProgressBar {
	if !dr.config.ShowProgress {
		return nil
	}

	bar := progressbar.NewOptions(total,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWidth(50),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionOnCompletion(func() {
			fmt.Printf(" %s Complete!\n", dr.symbols.Success)
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
	)

	return bar
}

// ShowSpinner shows a spinner for indeterminate progress
func (dr *DisplayRenderer) ShowSpinner(description string) *Spinner {
	if !dr.config.ShowProgress {
		return nil
	}

	spinner := NewSpinner(description)
	spinner.Start()
	return spinner
}

// Helper methods

// printBorder prints a border line
func (dr *DisplayRenderer) printBorder(left, middle, right string) {
	fmt.Printf("%s%s%s\n",
		left,
		strings.Repeat(middle, dr.width-2),
		right)
}

// formatTextLine applies basic text formatting
func (dr *DisplayRenderer) formatTextLine(line string) string {
	// Bold text between **
	line = strings.ReplaceAll(line, "**", "")

	// Code blocks between `
	parts := strings.Split(line, "`")
	for i := 1; i < len(parts); i += 2 {
		if i < len(parts) {
			parts[i] = color.New(color.FgYellow, color.BgBlack).Sprint(parts[i])
		}
	}

	return strings.Join(parts, "")
}

// Color helper methods
func (dr *DisplayRenderer) getChangeColor(changeType models.ChangeType) *color.Color {
	switch changeType {
	case models.ChangeTypeAdd:
		return color.New(color.FgGreen)
	case models.ChangeTypeModify:
		return color.New(color.FgYellow)
	case models.ChangeTypeDelete:
		return color.New(color.FgRed)
	case models.ChangeTypeReplace:
		return color.New(color.FgMagenta)
	default:
		return color.New(color.FgWhite)
	}
}

func (dr *DisplayRenderer) getActionColor(action models.FileAction) *color.Color {
	switch action {
	case models.FileActionCreate:
		return color.New(color.FgGreen)
	case models.FileActionModify:
		return color.New(color.FgYellow)
	case models.FileActionDelete:
		return color.New(color.FgRed)
	case models.FileActionRename:
		return color.New(color.FgCyan)
	default:
		return color.New(color.FgWhite)
	}
}

func (dr *DisplayRenderer) getSuggestionColor(suggestionType models.SuggestionType) *color.Color {
	switch suggestionType {
	case models.SuggestionTypeImprovement:
		return color.New(color.FgGreen)
	case models.SuggestionTypeOptimization:
		return color.New(color.FgYellow)
	case models.SuggestionTypeBugFix:
		return color.New(color.FgRed)
	case models.SuggestionTypeSecurity:
		return color.New(color.FgMagenta)
	case models.SuggestionTypeStyle:
		return color.New(color.FgCyan)
	case models.SuggestionTypePerformance:
		return color.New(color.FgBlue)
	default:
		return color.New(color.FgWhite)
	}
}

// Utility functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ResponseMetadata holds streaming response metadata
type ResponseMetadata struct {
	Provider        string
	Agent           string
	EstimatedTokens int
}

// Spinner represents a loading spinner
type Spinner struct {
	description string
	running     bool
	chars       []string
	current     int
}

// NewSpinner creates a new spinner
func NewSpinner(description string) *Spinner {
	return &Spinner{
		description: description,
		chars:       []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	}
}

// Start starts the spinner
func (s *Spinner) Start() {
	s.running = true
	go func() {
		for s.running {
			fmt.Printf("\r%s %s", s.chars[s.current], s.description)
			s.current = (s.current + 1) % len(s.chars)
			time.Sleep(100 * time.Millisecond)
		}
		fmt.Print("\r")
	}()
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	s.running = false
	time.Sleep(150 * time.Millisecond) // Allow final print to complete
}
