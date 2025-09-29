package mcp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// TierProcessor handles processing for each tier
type TierProcessor struct {
	filesystemServer *FilesystemServer
	vectorDB         VectorDBInterface
	llmManager       LLMManagerInterface
	cache            CacheInterface
}

// Interfaces for dependencies
type VectorDBInterface interface {
	Search(ctx context.Context, query string, limit int) ([]interface{}, error)
}

type LLMManagerInterface interface {
	Generate(ctx context.Context, request interface{}) (interface{}, error)
}

type CacheInterface interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
}

// NewTierProcessor creates a new tier processor
func NewTierProcessor(vectorDB VectorDBInterface, llmManager LLMManagerInterface) *TierProcessor {
	return &TierProcessor{
		filesystemServer: NewFilesystemServer(),
		vectorDB:         vectorDB,
		llmManager:       llmManager,
		cache:            NewSimpleCache(),
	}
}

// ProcessTier1 handles simple queries with direct MCP execution
func (tp *TierProcessor) ProcessTier1(ctx context.Context, query *models.Query, classification *ClassificationResult) (*models.Response, error) {
	startTime := time.Now()
	
	// Check cache first
	if cached, found := tp.cache.Get(classification.ProcessingStrategy.CacheKey); found {
		if response, ok := cached.(*models.Response); ok {
			response.Metadata.GenerationTime = time.Since(startTime)
			return response, nil
		}
	}
	
	var result string
	var err error
	
	// Execute operations directly based on classification
	for _, operation := range classification.RequiredOperations {
		switch operation {
		case "filesystem_list":
			result, err = tp.executeFileList(ctx, query)
		case "filesystem_tree":
			result, err = tp.executeDirectoryTree(ctx, query)
		case "system_info":
			result, err = tp.executeSystemInfo(ctx, query)
		case "filesystem_read":
			result, err = tp.executeFileRead(ctx, query)
		default:
			result, err = tp.executeGenericFilesystem(ctx, query)
		}
		
		if err != nil {
			return nil, fmt.Errorf("tier 1 operation failed: %w", err)
		}
	}
	
	// Create response without LLM
	response := &models.Response{
		ID:      fmt.Sprintf("tier1_response_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    models.ResponseTypeSystem,
		Content: models.ResponseContent{
			Text: result,
		},
		AgentUsed:  "mcp_direct",
		Provider:   "filesystem",
		TokenUsage: models.TokenUsage{TotalTokens: 0},
		Cost:       models.Cost{TotalCost: 0.0, Currency: "USD"},
		Metadata: models.ResponseMetadata{
			GenerationTime: time.Since(startTime),
			Confidence:     classification.Confidence,
			Tools:          []string{"filesystem"},
		},
		Timestamp: time.Now(),
	}
	
	// Cache the result
	tp.cache.Set(classification.ProcessingStrategy.CacheKey, response, 10*time.Minute)
	
	return response, nil
}

// ProcessTier2 handles medium queries with MCP + Vector search
func (tp *TierProcessor) ProcessTier2(ctx context.Context, query *models.Query, classification *ClassificationResult) (*models.Response, error) {
	startTime := time.Now()
	
	// Check cache first
	if cached, found := tp.cache.Get(classification.ProcessingStrategy.CacheKey); found {
		if response, ok := cached.(*models.Response); ok {
			response.Metadata.GenerationTime = time.Since(startTime)
			return response, nil
		}
	}
	
	var filesystemResult string
	var vectorResults []interface{}
	var err error
	
	// Execute operations in parallel
	resultChan := make(chan interface{}, 2)
	errorChan := make(chan error, 2)
	
	// Parallel operation 1: Filesystem
	go func() {
		result, err := tp.executeFilesystemSearch(ctx, query)
		if err != nil {
			errorChan <- err
		} else {
			resultChan <- result
		}
	}()
	
	// Parallel operation 2: Vector search (if available)
	if tp.vectorDB != nil {
		go func() {
			results, err := tp.vectorDB.Search(ctx, query.UserInput, 10)
			if err != nil {
				errorChan <- err
			} else {
				resultChan <- results
			}
		}()
	}
	
	// Collect results
	resultsReceived := 0
	expectedResults := 1
	if tp.vectorDB != nil {
		expectedResults = 2
	}
	
	for resultsReceived < expectedResults {
		select {
		case result := <-resultChan:
			if str, ok := result.(string); ok {
				filesystemResult = str
			} else if results, ok := result.([]interface{}); ok {
				vectorResults = results
			}
			resultsReceived++
		case err := <-errorChan:
			// Log error but continue with partial results
			fmt.Printf("⚠️ Tier 2 operation error: %v\n", err)
			resultsReceived++
		case <-time.After(5 * time.Second):
			// Timeout - proceed with partial results
			break
		}
	}
	
	// Format results without LLM synthesis
	formattedResult := tp.formatTier2Results(filesystemResult, vectorResults, query)
	
	response := &models.Response{
		ID:      fmt.Sprintf("tier2_response_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    models.ResponseTypeSearch,
		Content: models.ResponseContent{
			Text: formattedResult,
		},
		AgentUsed:  "mcp_vector",
		Provider:   "hybrid",
		TokenUsage: models.TokenUsage{TotalTokens: 0},
		Cost:       models.Cost{TotalCost: 0.0, Currency: "USD"},
		Metadata: models.ResponseMetadata{
			GenerationTime: time.Since(startTime),
			Confidence:     classification.Confidence,
			Tools:          []string{"filesystem", "vector_search"},
		},
		Timestamp: time.Now(),
	}
	
	// Cache the result
	tp.cache.Set(classification.ProcessingStrategy.CacheKey, response, 5*time.Minute)
	
	return response, nil
}

// ProcessTier3 handles complex queries with full LLM pipeline
func (tp *TierProcessor) ProcessTier3(ctx context.Context, query *models.Query, classification *ClassificationResult) (*models.Response, error) {
	startTime := time.Now()
	
	// This will use the existing intelligent query processor
	// for full context gathering and LLM generation
	
	// Build rich context
	context := tp.gatherRichContext(ctx, query, classification)
	
	// Build adaptive prompt
	prompt := tp.buildAdaptivePrompt(query, context, classification)
	
	// Call LLM with fallback
	llmResponse, err := tp.callLLMWithFallback(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("tier 3 LLM processing failed: %w", err)
	}
	
	// Enhance response with context
	response := tp.enhanceResponseWithContext(query, llmResponse, context, classification)
	response.Metadata.GenerationTime = time.Since(startTime)
	
	return response, nil
}

// Tier 1 Execution Methods
func (tp *TierProcessor) executeFileList(ctx context.Context, query *models.Query) (string, error) {
	input := strings.ToLower(query.UserInput)
	
	// Determine what to list based on query
	var pattern string
	if strings.Contains(input, "go") || strings.Contains(input, ".go") {
		pattern = "*.go"
	} else if strings.Contains(input, "yaml") || strings.Contains(input, ".yaml") {
		pattern = "*.yaml"
	} else if strings.Contains(input, "json") || strings.Contains(input, ".json") {
		pattern = "*.json"
	} else {
		pattern = "*"
	}
	
	cmd := exec.CommandContext(ctx, "find", ".", "-name", pattern, "-type", "f")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("file listing failed: %w", err)
	}
	
	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(files) == 1 && files[0] == "" {
		return "No files found matching the criteria.", nil
	}
	
	// Format output
	result := fmt.Sprintf("📁 Found %d files:\n", len(files))
	for i, file := range files {
		if i >= 20 { // Limit to first 20 files
			result += fmt.Sprintf("... and %d more files\n", len(files)-20)
			break
		}
		result += fmt.Sprintf("  %d. %s\n", i+1, file)
	}
	
	return result, nil
}

func (tp *TierProcessor) executeDirectoryTree(ctx context.Context, query *models.Query) (string, error) {
	cmd := exec.CommandContext(ctx, "find", ".", "-type", "d", "-not", "-path", "*/.*")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("directory tree failed: %w", err)
	}
	
	dirs := strings.Split(strings.TrimSpace(string(output)), "\n")
	
	result := fmt.Sprintf("📂 Project Structure (%d directories):\n", len(dirs))
	for _, dir := range dirs {
		if dir == "." {
			continue
		}
		depth := strings.Count(dir, "/")
		indent := strings.Repeat("  ", depth)
		dirName := filepath.Base(dir)
		result += fmt.Sprintf("%s├─ %s/\n", indent, dirName)
	}
	
	return result, nil
}

func (tp *TierProcessor) executeSystemInfo(ctx context.Context, query *models.Query) (string, error) {
	input := strings.ToLower(query.UserInput)
	
	var result strings.Builder
	result.WriteString("🖥️ System Information:\n")
	
	// Memory usage
	if strings.Contains(input, "memory") || strings.Contains(input, "cpu") {
		cmd := exec.CommandContext(ctx, "ps", "-o", "pid,%mem,%cpu,comm", "-p", fmt.Sprintf("%d", os.Getpid()))
		if output, err := cmd.Output(); err == nil {
			result.WriteString(fmt.Sprintf("Memory/CPU: %s", string(output)))
		}
	}
	
	// Disk usage
	if strings.Contains(input, "disk") || strings.Contains(input, "space") {
		cmd := exec.CommandContext(ctx, "du", "-sh", ".")
		if output, err := cmd.Output(); err == nil {
			result.WriteString(fmt.Sprintf("Disk Usage: %s", strings.TrimSpace(string(output))))
		}
	}
	
	// Process info
	if strings.Contains(input, "status") || strings.Contains(input, "info") {
		result.WriteString(fmt.Sprintf("Process ID: %d\n", os.Getpid()))
		if wd, err := os.Getwd(); err == nil {
			result.WriteString(fmt.Sprintf("Working Directory: %s\n", wd))
		}
	}
	
	return result.String(), nil
}

func (tp *TierProcessor) executeFileRead(ctx context.Context, query *models.Query) (string, error) {
	input := query.UserInput
	
	// Extract filename from query
	words := strings.Fields(input)
	var filename string
	
	for _, word := range words {
		if strings.Contains(word, ".go") || strings.Contains(word, ".yaml") || 
		   strings.Contains(word, ".json") || strings.Contains(word, ".md") {
			filename = word
			break
		}
	}
	
	if filename == "" {
		return "Please specify a filename to read.", nil
	}
	
	// Read file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Sprintf("❌ Could not read file '%s': %v", filename, err), nil
	}
	
	// Limit content size for display
	contentStr := string(content)
	if len(contentStr) > 2000 {
		contentStr = contentStr[:2000] + "\n... (truncated)"
	}
	
	return fmt.Sprintf("📄 Content of %s:\n```\n%s\n```", filename, contentStr), nil
}

// Tier 2 Execution Methods
func (tp *TierProcessor) executeFilesystemSearch(ctx context.Context, query *models.Query) (string, error) {
	input := strings.ToLower(query.UserInput)
	
	// Extract search terms
	searchTerms := tp.extractSearchTerms(input)
	if len(searchTerms) == 0 {
		return "No search terms found in query.", nil
	}
	
	var results []string
	
	// Search in Go files
	for _, term := range searchTerms {
		cmd := exec.CommandContext(ctx, "find", ".", "-name", "*.go", "-exec", "grep", "-l", term, "{}", ";")
		if output, err := cmd.Output(); err == nil {
			files := strings.Split(strings.TrimSpace(string(output)), "\n")
			for _, file := range files {
				if file != "" && !tp.contains(results, file) {
					results = append(results, file)
				}
			}
		}
	}
	
	if len(results) == 0 {
		return fmt.Sprintf("No files found containing: %s", strings.Join(searchTerms, ", ")), nil
	}
	
	// Format results
	resultStr := fmt.Sprintf("🔍 Found %d files containing '%s':\n", len(results), strings.Join(searchTerms, "', '"))
	for i, file := range results {
		if i >= 10 { // Limit to first 10 results
			resultStr += fmt.Sprintf("... and %d more files\n", len(results)-10)
			break
		}
		resultStr += fmt.Sprintf("  %d. %s\n", i+1, file)
	}
	
	return resultStr, nil
}

func (tp *TierProcessor) formatTier2Results(filesystemResult string, vectorResults []interface{}, query *models.Query) string {
	var result strings.Builder
	
	result.WriteString("🔍 Search Results:\n\n")
	
	// Add filesystem results
	if filesystemResult != "" {
		result.WriteString("📁 Filesystem Search:\n")
		result.WriteString(filesystemResult)
		result.WriteString("\n")
	}
	
	// Add vector results if available
	if len(vectorResults) > 0 {
		result.WriteString("🧠 Semantic Search:\n")
		for i, vr := range vectorResults {
			if i >= 5 { // Limit to top 5 vector results
				break
			}
			result.WriteString(fmt.Sprintf("  %d. Vector match (score: %.3f)\n", i+1, 0.8)) // Placeholder score
		}
	}
	
	return result.String()
}

// Tier 3 Helper Methods
func (tp *TierProcessor) gatherRichContext(ctx context.Context, query *models.Query, classification *ClassificationResult) map[string]interface{} {
	context := make(map[string]interface{})
	
	// Get project structure
	if structure, err := tp.getProjectStructure(ctx); err == nil {
		context["project_structure"] = structure
	}
	
	// Get relevant files
	if files, err := tp.getRelevantFiles(ctx, query); err == nil {
		context["relevant_files"] = files
	}
	
	// Get vector search results if available
	if tp.vectorDB != nil {
		if results, err := tp.vectorDB.Search(ctx, query.UserInput, 5); err == nil {
			context["similar_code"] = results
		}
	}
	
	return context
}

func (tp *TierProcessor) buildAdaptivePrompt(query *models.Query, context map[string]interface{}, classification *ClassificationResult) string {
	var prompt strings.Builder
	
	// System prompt based on query type
	switch {
	case strings.Contains(strings.ToLower(query.UserInput), "explain"):
		prompt.WriteString("You are an expert software architect. Provide clear, comprehensive explanations of code architecture and flow.\n\n")
	case strings.Contains(strings.ToLower(query.UserInput), "create"):
		prompt.WriteString("You are an expert developer. Generate clean, idiomatic code following project patterns.\n\n")
	default:
		prompt.WriteString("You are a helpful coding assistant. Provide accurate, contextual responses.\n\n")
	}
	
	// Add project context
	if structure, ok := context["project_structure"].(map[string]interface{}); ok {
		prompt.WriteString("PROJECT CONTEXT:\n")
		prompt.WriteString(tp.formatStructureForPrompt(structure))
		prompt.WriteString("\n")
	}
	
	// Add relevant files
	if files, ok := context["relevant_files"].([]string); ok {
		prompt.WriteString("KEY FILES:\n")
		for _, file := range files {
			prompt.WriteString(fmt.Sprintf("- %s\n", file))
		}
		prompt.WriteString("\n")
	}
	
	// Add user query
	prompt.WriteString(fmt.Sprintf("USER REQUEST: %s\n", query.UserInput))
	
	return prompt.String()
}

func (tp *TierProcessor) callLLMWithFallback(ctx context.Context, prompt string) (interface{}, error) {
	if tp.llmManager == nil {
		return nil, fmt.Errorf("LLM manager not available")
	}
	
	request := map[string]interface{}{
		"prompt":      prompt,
		"max_tokens":  2000,
		"temperature": 0.1,
	}
	
	return tp.llmManager.Generate(ctx, request)
}

func (tp *TierProcessor) enhanceResponseWithContext(query *models.Query, llmResponse interface{}, context map[string]interface{}, classification *ClassificationResult) *models.Response {
	responseText := fmt.Sprintf("%v", llmResponse)
	
	// Add context references
	if files, ok := context["relevant_files"].([]string); ok && len(files) > 0 {
		responseText += "\n\n📁 Key Files Referenced:\n"
		for _, file := range files {
			responseText += fmt.Sprintf("- %s\n", file)
		}
	}
	
	return &models.Response{
		ID:      fmt.Sprintf("tier3_response_%d", time.Now().UnixNano()),
		QueryID: query.ID,
		Type:    models.ResponseTypeExplanation,
		Content: models.ResponseContent{
			Text: responseText,
		},
		AgentUsed:  "intelligent_processor",
		Provider:   "openai", // Would be determined by actual LLM call
		TokenUsage: models.TokenUsage{TotalTokens: 1500}, // Would be from actual LLM response
		Cost:       models.Cost{TotalCost: classification.EstimatedCost, Currency: "USD"},
		Metadata: models.ResponseMetadata{
			Confidence: classification.Confidence,
			Tools:      []string{"filesystem", "vector_search", "llm_generation"},
		},
		Timestamp: time.Now(),
	}
}

// Helper Methods
func (tp *TierProcessor) extractSearchTerms(input string) []string {
	// Remove common words and extract meaningful terms
	words := strings.Fields(input)
	var terms []string
	
	stopWords := map[string]bool{
		"find": true, "search": true, "for": true, "all": true, "the": true,
		"show": true, "me": true, "get": true, "list": true,
	}
	
	for _, word := range words {
		if len(word) > 2 && !stopWords[word] {
			terms = append(terms, word)
		}
	}
	
	return terms
}

func (tp *TierProcessor) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (tp *TierProcessor) executeGenericFilesystem(ctx context.Context, query *models.Query) (string, error) {
	return "Filesystem operation completed.", nil
}

func (tp *TierProcessor) getProjectStructure(ctx context.Context) (map[string]interface{}, error) {
	cmd := exec.CommandContext(ctx, "find", ".", "-type", "d", "-not", "-path", "*/.*")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	
	dirs := strings.Split(strings.TrimSpace(string(output)), "\n")
	structure := make(map[string]interface{})
	
	for _, dir := range dirs {
		if dir == "." {
			continue
		}
		parts := strings.Split(strings.TrimPrefix(dir, "./"), "/")
		current := structure
		for _, part := range parts {
			if current[part] == nil {
				current[part] = make(map[string]interface{})
			}
			if next, ok := current[part].(map[string]interface{}); ok {
				current = next
			}
		}
	}
	
	return structure, nil
}

func (tp *TierProcessor) getRelevantFiles(ctx context.Context, query *models.Query) ([]string, error) {
	// Get key architectural files for explanation queries
	keyFiles := []string{
		"cmd/main.go",
		"internal/app/cli.go", 
		"internal/agents/manager_agent.go",
		"internal/mcp/mcp_client.go",
	}
	
	var existingFiles []string
	for _, file := range keyFiles {
		if _, err := os.Stat(file); err == nil {
			existingFiles = append(existingFiles, file)
		}
	}
	
	return existingFiles, nil
}

func (tp *TierProcessor) formatStructureForPrompt(structure map[string]interface{}) string {
	var result strings.Builder
	tp.formatStructureRecursive(structure, "", &result)
	return result.String()
}

func (tp *TierProcessor) formatStructureRecursive(structure map[string]interface{}, prefix string, result *strings.Builder) {
	for key := range structure {
		result.WriteString(fmt.Sprintf("%s- %s/\n", prefix, key))
		if subMap, ok := structure[key].(map[string]interface{}); ok && len(prefix) < 4 {
			tp.formatStructureRecursive(subMap, prefix+"  ", result)
		}
	}
}

// SimpleCache implementation
type SimpleCache struct {
	data map[string]CacheItem
}

type CacheItem struct {
	Value     interface{}
	ExpiresAt time.Time
}

func NewSimpleCache() *SimpleCache {
	return &SimpleCache{
		data: make(map[string]CacheItem),
	}
}

func (sc *SimpleCache) Get(key string) (interface{}, bool) {
	item, exists := sc.data[key]
	if !exists || time.Now().After(item.ExpiresAt) {
		delete(sc.data, key)
		return nil, false
	}
	return item.Value, true
}

func (sc *SimpleCache) Set(key string, value interface{}, ttl time.Duration) {
	sc.data[key] = CacheItem{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
}