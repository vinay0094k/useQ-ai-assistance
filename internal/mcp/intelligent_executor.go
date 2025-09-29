package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// IntelligentExecutor analyzes queries and determines appropriate commands to run
type IntelligentExecutor struct {
	decisionEngine   *DecisionEngine
	commandRegistry  *CommandRegistry
	safetyValidator  *SafetyValidator
	executionHistory []ExecutionRecord
}

// CommandRegistry holds available commands and their metadata
type CommandRegistry struct {
	commands map[string]*CommandDefinition
}

// CommandDefinition defines a command that can be executed
type CommandDefinition struct {
	Name        string            `json:"name"`
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	Safety      SafetyLevel       `json:"safety"`
	Triggers    []string          `json:"triggers"`
	Context     map[string]string `json:"context"`
}

// SafetyLevel defines command safety levels
type SafetyLevel string

const (
	SafetyLevelSafe       SafetyLevel = "safe"       // Read-only operations
	SafetyLevelModerate   SafetyLevel = "moderate"   // File modifications
	SafetyLevelDangerous  SafetyLevel = "dangerous"  // System changes
	SafetyLevelRestricted SafetyLevel = "restricted" // Blocked commands
)

// ExecutionRecord tracks command execution history
type ExecutionRecord struct {
	QueryID     string                 `json:"query_id"`
	Query       string                 `json:"query"`
	Commands    []string               `json:"commands"`
	Results     map[string]interface{} `json:"results"`
	Success     bool                   `json:"success"`
	Duration    time.Duration          `json:"duration"`
	Timestamp   time.Time              `json:"timestamp"`
	SafetyLevel SafetyLevel            `json:"safety_level"`
}

// NewIntelligentExecutor creates a new intelligent command executor
func NewIntelligentExecutor() *IntelligentExecutor {
	executor := &IntelligentExecutor{
		decisionEngine:   NewDecisionEngine(),
		commandRegistry:  NewCommandRegistry(),
		safetyValidator:  NewSafetyValidator(),
		executionHistory: make([]ExecutionRecord, 0),
	}
	
	executor.initializeCommands()
	return executor
}

// AnalyzeAndExecute analyzes query and executes appropriate commands
func (ie *IntelligentExecutor) AnalyzeAndExecute(ctx context.Context, query *models.Query) (*models.MCPContext, error) {
	startTime := time.Now()
	
	// Step 1: Analyze query to determine required commands
	analysis := ie.analyzeQueryForCommands(query)
	
	// Step 2: Select appropriate commands based on analysis
	selectedCommands := ie.selectCommands(analysis)
	
	// Step 3: Validate safety of selected commands
	if err := ie.validateCommandSafety(selectedCommands); err != nil {
		return nil, fmt.Errorf("command safety validation failed: %w", err)
	}
	
	// Step 4: Execute commands and gather results
	results, err := ie.executeCommands(ctx, selectedCommands, query)
	if err != nil {
		return nil, fmt.Errorf("command execution failed: %w", err)
	}
	
	// Step 5: Build MCP context from results
	mcpContext := ie.buildMCPContext(query, selectedCommands, results)
	
	// Step 6: Record execution for learning
	ie.recordExecution(query, selectedCommands, results, err == nil, time.Since(startTime))
	
	return mcpContext, nil
}

// analyzeQueryForCommands determines what type of information is needed
func (ie *IntelligentExecutor) analyzeQueryForCommands(query *models.Query) *QueryAnalysis {
	input := strings.ToLower(query.UserInput)
	
	analysis := &QueryAnalysis{
		Query:            query.UserInput,
		Intent:           ie.determineIntent(input),
		RequiredInfo:     make([]InfoType, 0),
		Scope:            ie.determineScope(input),
		UrgencyLevel:     ie.determineUrgency(input),
		SafetyRequirement: ie.determineSafetyRequirement(input),
	}
	
	// Determine what information is needed based on query content
	if ie.needsFileSystemInfo(input) {
		analysis.RequiredInfo = append(analysis.RequiredInfo, InfoTypeFileSystem)
	}
	if ie.needsSystemInfo(input) {
		analysis.RequiredInfo = append(analysis.RequiredInfo, InfoTypeSystem)
	}
	if ie.needsGitInfo(input) {
		analysis.RequiredInfo = append(analysis.RequiredInfo, InfoTypeGit)
	}
	if ie.needsProcessInfo(input) {
		analysis.RequiredInfo = append(analysis.RequiredInfo, InfoTypeProcess)
	}
	if ie.needsNetworkInfo(input) {
		analysis.RequiredInfo = append(analysis.RequiredInfo, InfoTypeNetwork)
	}
	if ie.needsDatabaseInfo(input) {
		analysis.RequiredInfo = append(analysis.RequiredInfo, InfoTypeDatabase)
	}
	
	return analysis
}

// selectCommands chooses appropriate commands based on analysis
func (ie *IntelligentExecutor) selectCommands(analysis *QueryAnalysis) []*CommandDefinition {
	var commands []*CommandDefinition
	
	for _, infoType := range analysis.RequiredInfo {
		switch infoType {
		case InfoTypeFileSystem:
			commands = append(commands, ie.getFileSystemCommands(analysis)...)
		case InfoTypeSystem:
			commands = append(commands, ie.getSystemCommands(analysis)...)
		case InfoTypeGit:
			commands = append(commands, ie.getGitCommands(analysis)...)
		case InfoTypeProcess:
			commands = append(commands, ie.getProcessCommands(analysis)...)
		case InfoTypeNetwork:
			commands = append(commands, ie.getNetworkCommands(analysis)...)
		case InfoTypeDatabaseInfo:
			commands = append(commands, ie.getDatabaseCommands(analysis)...)
		}
	}
	
	// Sort by priority and safety
	ie.prioritizeCommands(commands, analysis)
	
	return commands
}

// executeCommands runs the selected commands safely
func (ie *IntelligentExecutor) executeCommands(ctx context.Context, commands []*CommandDefinition, query *models.Query) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	
	for _, cmd := range commands {
		// Create execution context with timeout
		execCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		
		// Execute command
		result, err := ie.executeCommand(execCtx, cmd)
		cancel()
		
		if err != nil {
			// Log error but continue with other commands
			results[cmd.Name+"_error"] = err.Error()
			continue
		}
		
		results[cmd.Name] = result
	}
	
	return results, nil
}

// executeCommand executes a single command safely
func (ie *IntelligentExecutor) executeCommand(ctx context.Context, cmd *CommandDefinition) (interface{}, error) {
	switch cmd.Category {
	case "filesystem":
		return ie.executeFileSystemCommand(ctx, cmd)
	case "system":
		return ie.executeSystemCommand(ctx, cmd)
	case "git":
		return ie.executeGitCommand(ctx, cmd)
	case "process":
		return ie.executeProcessCommand(ctx, cmd)
	case "network":
		return ie.executeNetworkCommand(ctx, cmd)
	case "database":
		return ie.executeDatabaseCommand(ctx, cmd)
	default:
		return nil, fmt.Errorf("unknown command category: %s", cmd.Category)
	}
}

// initializeCommands sets up the command registry
func (ie *IntelligentExecutor) initializeCommands() {
	// File system commands
	ie.commandRegistry.Register(&CommandDefinition{
		Name:        "list_files",
		Command:     "find",
		Args:        []string{".", "-name", "*.go", "-type", "f"},
		Description: "List Go files in project",
		Category:    "filesystem",
		Safety:      SafetyLevelSafe,
		Triggers:    []string{"files", "list", "show", "find"},
	})
	
	ie.commandRegistry.Register(&CommandDefinition{
		Name:        "file_count",
		Command:     "find",
		Args:        []string{".", "-name", "*.go", "-type", "f", "|", "wc", "-l"},
		Description: "Count Go files",
		Category:    "filesystem",
		Safety:      SafetyLevelSafe,
		Triggers:    []string{"count", "how many", "number"},
	})
	
	ie.commandRegistry.Register(&CommandDefinition{
		Name:        "project_structure",
		Command:     "tree",
		Args:        []string{"-I", "vendor|node_modules|.git", "-L", "3"},
		Description: "Show project structure",
		Category:    "filesystem",
		Safety:      SafetyLevelSafe,
		Triggers:    []string{"structure", "tree", "organization"},
	})
	
	// System commands
	ie.commandRegistry.Register(&CommandDefinition{
		Name:        "memory_usage",
		Command:     "ps",
		Args:        []string{"-o", "pid,ppid,%mem,%cpu,comm", "-p", "$$"},
		Description: "Show memory usage",
		Category:    "system",
		Safety:      SafetyLevelSafe,
		Triggers:    []string{"memory", "cpu", "usage", "performance"},
	})
	
	ie.commandRegistry.Register(&CommandDefinition{
		Name:        "disk_usage",
		Command:     "du",
		Args:        []string{"-sh", "."},
		Description: "Show disk usage",
		Category:    "system",
		Safety:      SafetyLevelSafe,
		Triggers:    []string{"disk", "space", "size"},
	})
	
	// Git commands
	ie.commandRegistry.Register(&CommandDefinition{
		Name:        "git_status",
		Command:     "git",
		Args:        []string{"status", "--porcelain"},
		Description: "Get git status",
		Category:    "git",
		Safety:      SafetyLevelSafe,
		Triggers:    []string{"git", "status", "changes", "modified"},
	})
	
	ie.commandRegistry.Register(&CommandDefinition{
		Name:        "git_log",
		Command:     "git",
		Args:        []string{"log", "--oneline", "-10"},
		Description: "Get recent commits",
		Category:    "git",
		Safety:      SafetyLevelSafe,
		Triggers:    []string{"commits", "history", "log"},
	})
	
	// Process commands
	ie.commandRegistry.Register(&CommandDefinition{
		Name:        "running_processes",
		Command:     "ps",
		Args:        []string{"aux"},
		Description: "List running processes",
		Category:    "process",
		Safety:      SafetyLevelSafe,
		Triggers:    []string{"processes", "running", "ps"},
	})
}

// Query analysis helper methods
func (ie *IntelligentExecutor) needsFileSystemInfo(input string) bool {
	keywords := []string{"files", "directories", "structure", "tree", "find", "list", "count"}
	return ie.containsAny(input, keywords)
}

func (ie *IntelligentExecutor) needsSystemInfo(input string) bool {
	keywords := []string{"memory", "cpu", "disk", "performance", "usage", "system"}
	return ie.containsAny(input, keywords)
}

func (ie *IntelligentExecutor) needsGitInfo(input string) bool {
	keywords := []string{"git", "commit", "branch", "status", "history", "changes"}
	return ie.containsAny(input, keywords)
}

func (ie *IntelligentExecutor) needsProcessInfo(input string) bool {
	keywords := []string{"process", "running", "ps", "kill", "pid"}
	return ie.containsAny(input, keywords)
}

func (ie *IntelligentExecutor) needsNetworkInfo(input string) bool {
	keywords := []string{"network", "port", "connection", "netstat", "ping"}
	return ie.containsAny(input, keywords)
}

func (ie *IntelligentExecutor) needsDatabaseInfo(input string) bool {
	keywords := []string{"database", "db", "sql", "table", "query"}
	return ie.containsAny(input, keywords)
}

func (ie *IntelligentExecutor) containsAny(input string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(input, keyword) {
			return true
		}
	}
	return false
}

// Command execution methods
func (ie *IntelligentExecutor) executeFileSystemCommand(ctx context.Context, cmd *CommandDefinition) (interface{}, error) {
	switch cmd.Name {
	case "list_files":
		return ie.listGoFiles()
	case "file_count":
		return ie.countGoFiles()
	case "project_structure":
		return ie.getProjectStructure()
	default:
		return ie.executeShellCommand(ctx, cmd)
	}
}

func (ie *IntelligentExecutor) executeSystemCommand(ctx context.Context, cmd *CommandDefinition) (interface{}, error) {
	switch cmd.Name {
	case "memory_usage":
		return ie.getMemoryUsage()
	case "disk_usage":
		return ie.getDiskUsage()
	default:
		return ie.executeShellCommand(ctx, cmd)
	}
}

func (ie *IntelligentExecutor) executeGitCommand(ctx context.Context, cmd *CommandDefinition) (interface{}, error) {
	return ie.executeShellCommand(ctx, cmd)
}

func (ie *IntelligentExecutor) executeProcessCommand(ctx context.Context, cmd *CommandDefinition) (interface{}, error) {
	return ie.executeShellCommand(ctx, cmd)
}

func (ie *IntelligentExecutor) executeNetworkCommand(ctx context.Context, cmd *CommandDefinition) (interface{}, error) {
	return ie.executeShellCommand(ctx, cmd)
}

func (ie *IntelligentExecutor) executeDatabaseCommand(ctx context.Context, cmd *CommandDefinition) (interface{}, error) {
	return ie.executeShellCommand(ctx, cmd)
}

// executeShellCommand safely executes shell commands
func (ie *IntelligentExecutor) executeShellCommand(ctx context.Context, cmd *CommandDefinition) (interface{}, error) {
	// Build command
	execCmd := exec.CommandContext(ctx, cmd.Command, cmd.Args...)
	
	// Execute with timeout
	output, err := execCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	}
	
	return map[string]interface{}{
		"command": cmd.Command + " " + strings.Join(cmd.Args, " "),
		"output":  string(output),
		"success": true,
	}, nil
}

// Specific command implementations
func (ie *IntelligentExecutor) listGoFiles() (interface{}, error) {
	cmd := exec.Command("find", ".", "-name", "*.go", "-type", "f")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	return map[string]interface{}{
		"files": files,
		"count": len(files),
	}, nil
}

func (ie *IntelligentExecutor) countGoFiles() (interface{}, error) {
	result, err := ie.listGoFiles()
	if err != nil {
		return nil, err
	}
	
	if data, ok := result.(map[string]interface{}); ok {
		return map[string]interface{}{
			"count": data["count"],
		}, nil
	}
	
	return map[string]interface{}{"count": 0}, nil
}

func (ie *IntelligentExecutor) getProjectStructure() (interface{}, error) {
	cmd := exec.Command("find", ".", "-type", "d", "-not", "-path", "*/.*")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	dirs := strings.Split(strings.TrimSpace(string(output)), "\n")
	return map[string]interface{}{
		"directories": dirs,
		"structure":   ie.buildStructureMap(dirs),
	}, nil
}

func (ie *IntelligentExecutor) getMemoryUsage() (interface{}, error) {
	cmd := exec.Command("ps", "-o", "pid,ppid,%mem,%cpu,comm")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"memory_info": string(output),
		"timestamp":   time.Now(),
	}, nil
}

func (ie *IntelligentExecutor) getDiskUsage() (interface{}, error) {
	cmd := exec.Command("du", "-sh", ".")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"disk_usage": strings.TrimSpace(string(output)),
		"timestamp":  time.Now(),
	}, nil
}

// Helper methods
func (ie *IntelligentExecutor) buildStructureMap(dirs []string) map[string]interface{} {
	structure := make(map[string]interface{})
	
	for _, dir := range dirs {
		if dir == "." {
			continue
		}
		
		parts := strings.Split(strings.TrimPrefix(dir, "./"), "/")
		current := structure
		
		for _, part := range parts {
			if part == "" {
				continue
			}
			if current[part] == nil {
				current[part] = make(map[string]interface{})
			}
			if next, ok := current[part].(map[string]interface{}); ok {
				current = next
			}
		}
	}
	
	return structure
}

func (ie *IntelligentExecutor) buildMCPContext(query *models.Query, commands []*CommandDefinition, results map[string]interface{}) *models.MCPContext {
	operations := make([]string, len(commands))
	for i, cmd := range commands {
		operations[i] = cmd.Name
	}
	
	return &models.MCPContext{
		RequiresMCP: true,
		Operations:  operations,
		Data:        results,
	}
}

func (ie *IntelligentExecutor) recordExecution(query *models.Query, commands []*CommandDefinition, results map[string]interface{}, success bool, duration time.Duration) {
	commandNames := make([]string, len(commands))
	for i, cmd := range commands {
		commandNames[i] = cmd.Name
	}
	
	record := ExecutionRecord{
		QueryID:   query.ID,
		Query:     query.UserInput,
		Commands:  commandNames,
		Results:   results,
		Success:   success,
		Duration:  duration,
		Timestamp: time.Now(),
	}
	
	ie.executionHistory = append(ie.executionHistory, record)
}

// Supporting types and methods
type QueryAnalysis struct {
	Query             string      `json:"query"`
	Intent            string      `json:"intent"`
	RequiredInfo      []InfoType  `json:"required_info"`
	Scope             string      `json:"scope"`
	UrgencyLevel      string      `json:"urgency_level"`
	SafetyRequirement SafetyLevel `json:"safety_requirement"`
}

type InfoType string

const (
	InfoTypeFileSystem InfoType = "filesystem"
	InfoTypeSystem     InfoType = "system"
	InfoTypeGit        InfoType = "git"
	InfoTypeProcess    InfoType = "process"
	InfoTypeNetwork    InfoType = "network"
	InfoTypeDatabase   InfoType = "database"
	InfoTypeDatabaseInfo InfoType = "database_info"
)

func (ie *IntelligentExecutor) determineIntent(input string) string {
	if strings.Contains(input, "show") || strings.Contains(input, "list") {
		return "information"
	}
	if strings.Contains(input, "count") || strings.Contains(input, "how many") {
		return "count"
	}
	if strings.Contains(input, "status") || strings.Contains(input, "check") {
		return "status"
	}
	return "general"
}

func (ie *IntelligentExecutor) determineScope(input string) string {
	if strings.Contains(input, "project") || strings.Contains(input, "all") {
		return "project"
	}
	if strings.Contains(input, "current") || strings.Contains(input, "this") {
		return "current"
	}
	return "default"
}

func (ie *IntelligentExecutor) determineUrgency(input string) string {
	if strings.Contains(input, "urgent") || strings.Contains(input, "quickly") {
		return "high"
	}
	return "normal"
}

func (ie *IntelligentExecutor) determineSafetyRequirement(input string) SafetyLevel {
	if strings.Contains(input, "delete") || strings.Contains(input, "remove") {
		return SafetyLevelDangerous
	}
	if strings.Contains(input, "modify") || strings.Contains(input, "change") {
		return SafetyLevelModerate
	}
	return SafetyLevelSafe
}

// Command selection helpers
func (ie *IntelligentExecutor) getFileSystemCommands(analysis *QueryAnalysis) []*CommandDefinition {
	var commands []*CommandDefinition
	
	if strings.Contains(analysis.Query, "count") || strings.Contains(analysis.Query, "how many") {
		commands = append(commands, ie.commandRegistry.Get("file_count"))
	} else if strings.Contains(analysis.Query, "structure") || strings.Contains(analysis.Query, "tree") {
		commands = append(commands, ie.commandRegistry.Get("project_structure"))
	} else {
		commands = append(commands, ie.commandRegistry.Get("list_files"))
	}
	
	return commands
}

func (ie *IntelligentExecutor) getSystemCommands(analysis *QueryAnalysis) []*CommandDefinition {
	var commands []*CommandDefinition
	
	if strings.Contains(analysis.Query, "memory") || strings.Contains(analysis.Query, "cpu") {
		commands = append(commands, ie.commandRegistry.Get("memory_usage"))
	}
	if strings.Contains(analysis.Query, "disk") || strings.Contains(analysis.Query, "space") {
		commands = append(commands, ie.commandRegistry.Get("disk_usage"))
	}
	
	return commands
}

func (ie *IntelligentExecutor) getGitCommands(analysis *QueryAnalysis) []*CommandDefinition {
	var commands []*CommandDefinition
	
	if strings.Contains(analysis.Query, "status") || strings.Contains(analysis.Query, "changes") {
		commands = append(commands, ie.commandRegistry.Get("git_status"))
	}
	if strings.Contains(analysis.Query, "history") || strings.Contains(analysis.Query, "commits") {
		commands = append(commands, ie.commandRegistry.Get("git_log"))
	}
	
	return commands
}

func (ie *IntelligentExecutor) getProcessCommands(analysis *QueryAnalysis) []*CommandDefinition {
	return []*CommandDefinition{ie.commandRegistry.Get("running_processes")}
}

func (ie *IntelligentExecutor) getNetworkCommands(analysis *QueryAnalysis) []*CommandDefinition {
	return []*CommandDefinition{}
}

func (ie *IntelligentExecutor) getDatabaseCommands(analysis *QueryAnalysis) []*CommandDefinition {
	return []*CommandDefinition{}
}

func (ie *IntelligentExecutor) prioritizeCommands(commands []*CommandDefinition, analysis *QueryAnalysis) {
	// Sort by safety level (safer commands first)
	// This is a simple implementation - could be more sophisticated
}

func (ie *IntelligentExecutor) validateCommandSafety(commands []*CommandDefinition) error {
	for _, cmd := range commands {
		if cmd.Safety == SafetyLevelRestricted {
			return fmt.Errorf("command %s is restricted", cmd.Name)
		}
	}
	return nil
}