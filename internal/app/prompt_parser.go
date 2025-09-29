package app

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/yourusername/useq-ai-assistant/models"
)

// PromptParser handles natural language query parsing and intent detection
type PromptParser struct {
	// Intent patterns for different query types
	searchPatterns        []*IntentPattern
	generationPatterns    []*IntentPattern
	explanationPatterns   []*IntentPattern
	debuggingPatterns     []*IntentPattern
	testingPatterns       []*IntentPattern
	reviewPatterns        []*IntentPattern
	documentationPatterns []*IntentPattern

	// Entity extraction patterns
	entityPatterns map[models.EntityType]*regexp.Regexp

	// Keyword extraction
	stopWords    map[string]bool
	techKeywords map[string]bool
}

// IntentPattern represents a pattern for detecting query intent
type IntentPattern struct {
	Pattern     *regexp.Regexp
	Weight      float64
	QueryType   models.QueryType
	Keywords    []string
	Description string
}

// NewPromptParser creates a new prompt parser instance
func NewPromptParser() *PromptParser {
	parser := &PromptParser{
		entityPatterns: make(map[models.EntityType]*regexp.Regexp),
		stopWords:      createStopWords(),
		techKeywords:   createTechKeywords(),
	}

	parser.initializeIntentPatterns()
	parser.initializeEntityPatterns()

	return parser
}

// ParseIntent analyzes user input and determines the intent
func (p *PromptParser) ParseIntent(input string) (*models.QueryIntent, error) {
	if strings.TrimSpace(input) == "" {
		return nil, fmt.Errorf("empty input provided")
	}

	// Normalize input
	normalized := p.normalizeInput(input)

	// Extract keywords first
	keywords := p.extractKeywords(normalized)

	// Extract entities
	entities := p.extractEntities(input)

	// Detect primary intent
	intentScores := p.calculateIntentScores(normalized, keywords)

	primary, confidence := p.selectPrimaryIntent(intentScores)
	secondary := p.selectSecondaryIntents(intentScores, primary)

	// Extract file and function targets
	fileTargets := p.extractFileTargets(input)
	funcTargets := p.extractFunctionTargets(input)

	return &models.QueryIntent{
		Primary:     primary,
		Secondary:   secondary,
		Confidence:  confidence,
		Keywords:    keywords,
		Entities:    entities,
		FileTargets: fileTargets,
		FuncTargets: funcTargets,
	}, nil
}

// initializeIntentPatterns sets up patterns for different intent types
func (p *PromptParser) initializeIntentPatterns() {
	// Search patterns
	p.searchPatterns = []*IntentPattern{
		{
			Pattern:     regexp.MustCompile(`(?i)\b(find|search|locate|look\s+for|where\s+is)\b`),
			Weight:      0.9,
			QueryType:   models.QueryTypeSearch,
			Keywords:    []string{"find", "search", "locate"},
			Description: "Direct search commands",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)\b(show\s+me|list|get)\s+.*(function|method|struct|interface|type|file|package)\b`),
			Weight:      0.8,
			QueryType:   models.QueryTypeSearch,
			Keywords:    []string{"show", "list", "get"},
			Description: "Show/list code elements",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)\b(how\s+many|count|all\s+the)\b`),
			Weight:      0.7,
			QueryType:   models.QueryTypeSearch,
			Keywords:    []string{"count", "how many", "all"},
			Description: "Counting and enumeration queries",
		},
	}

	// Generation patterns
	p.generationPatterns = []*IntentPattern{
		{
			Pattern:     regexp.MustCompile(`(?i)\b(create|generate|make|build|implement|write)\b`),
			Weight:      0.9,
			QueryType:   models.QueryTypeGeneration,
			Keywords:    []string{"create", "generate", "implement", "write"},
			Description: "Code generation commands",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)\b(add\s+a|new)\s+.*(function|method|struct|handler|endpoint|api)\b`),
			Weight:      0.8,
			QueryType:   models.QueryTypeGeneration,
			Keywords:    []string{"add", "new"},
			Description: "Adding new code elements",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)\b(scaffold|template|boilerplate)\b`),
			Weight:      0.7,
			QueryType:   models.QueryTypeGeneration,
			Keywords:    []string{"scaffold", "template"},
			Description: "Template generation",
		},
	}

	// Explanation patterns
	p.explanationPatterns = []*IntentPattern{
		{
			Pattern:     regexp.MustCompile(`(?i)\b(explain|what\s+(is|does)|how\s+(does|is|to))\b`),
			Weight:      0.9,
			QueryType:   models.QueryTypeExplanation,
			Keywords:    []string{"explain", "what", "how"},
			Description: "Explanation requests",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)\b(describe|tell\s+me\s+about|walk\s+through)\b`),
			Weight:      0.8,
			QueryType:   models.QueryTypeExplanation,
			Keywords:    []string{"describe", "tell me"},
			Description: "Description requests",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)\b(understand|meaning|purpose|why)\b`),
			Weight:      0.7,
			QueryType:   models.QueryTypeExplanation,
			Keywords:    []string{"understand", "why", "purpose"},
			Description: "Understanding queries",
		},
	}

	// Debugging patterns
	p.debuggingPatterns = []*IntentPattern{
		{
			Pattern:     regexp.MustCompile(`(?i)\b(debug|fix|error|issue|problem|bug|broken)\b`),
			Weight:      0.9,
			QueryType:   models.QueryTypeDebugging,
			Keywords:    []string{"debug", "fix", "error", "bug"},
			Description: "Debugging requests",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)\b(not\s+working|failing|crash|panic)\b`),
			Weight:      0.8,
			QueryType:   models.QueryTypeDebugging,
			Keywords:    []string{"not working", "failing", "crash"},
			Description: "Failure descriptions",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)\b(troubleshoot|diagnose|trace)\b`),
			Weight:      0.7,
			QueryType:   models.QueryTypeDebugging,
			Keywords:    []string{"troubleshoot", "diagnose"},
			Description: "Diagnostic requests",
		},
	}

	// Testing patterns
	p.testingPatterns = []*IntentPattern{
		{
			Pattern:     regexp.MustCompile(`(?i)\b(test|tests|testing|unit\s+test|integration\s+test)\b`),
			Weight:      0.9,
			QueryType:   models.QueryTypeTesting,
			Keywords:    []string{"test", "testing", "unit test"},
			Description: "Test-related commands",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)\b(mock|benchmark|coverage)\b`),
			Weight:      0.8,
			QueryType:   models.QueryTypeTesting,
			Keywords:    []string{"mock", "benchmark", "coverage"},
			Description: "Testing tools and metrics",
		},
	}

	// Review patterns
	p.reviewPatterns = []*IntentPattern{
		{
			Pattern:     regexp.MustCompile(`(?i)\b(review|check|analyze|audit|inspect)\b`),
			Weight:      0.9,
			QueryType:   models.QueryTypeReview,
			Keywords:    []string{"review", "check", "analyze"},
			Description: "Code review requests",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)\b(optimize|improve|refactor|clean\s+up)\b`),
			Weight:      0.8,
			QueryType:   models.QueryTypeRefactoring,
			Keywords:    []string{"optimize", "improve", "refactor"},
			Description: "Code improvement requests",
		},
	}

	// Documentation patterns
	p.documentationPatterns = []*IntentPattern{
		{
			Pattern:     regexp.MustCompile(`(?i)\b(document|docs|documentation|comment|comments)\b`),
			Weight:      0.9,
			QueryType:   models.QueryTypeDocumentation,
			Keywords:    []string{"document", "docs", "comment"},
			Description: "Documentation requests",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)\b(readme|godoc|api\s+doc)\b`),
			Weight:      0.8,
			QueryType:   models.QueryTypeDocumentation,
			Keywords:    []string{"readme", "godoc", "api doc"},
			Description: "Specific documentation types",
		},
	}
}

// initializeEntityPatterns sets up regex patterns for entity extraction
func (p *PromptParser) initializeEntityPatterns() {
	p.entityPatterns[models.EntityTypeFunction] = regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s*\(\s*\)|\b([a-zA-Z_][a-zA-Z0-9_]*)\s*function\b`)
	p.entityPatterns[models.EntityTypeVariable] = regexp.MustCompile(`\b(var|variable|field)\s+([a-zA-Z_][a-zA-Z0-9_]*)\b`)
	p.entityPatterns[models.EntityTypeType] = regexp.MustCompile(`\b(type|struct|interface)\s+([a-zA-Z_][a-zA-Z0-9_]*)\b`)
	p.entityPatterns[models.EntityTypePackage] = regexp.MustCompile(`\b(package|import)\s+([a-zA-Z_][a-zA-Z0-9_/]*)\b`)
	p.entityPatterns[models.EntityTypeFile] = regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_/]*\.(go|mod|sum|yaml|yml|json))\b`)
	p.entityPatterns[models.EntityTypeError] = regexp.MustCompile(`\b(error|Error|err|panic|fatal)\b`)
}

// normalizeInput preprocesses the input for better pattern matching
func (p *PromptParser) normalizeInput(input string) string {
	// Convert to lowercase
	normalized := strings.ToLower(input)

	// Replace common contractions
	contractions := map[string]string{
		"won't":   "will not",
		"can't":   "cannot",
		"don't":   "do not",
		"doesn't": "does not",
		"isn't":   "is not",
		"aren't":  "are not",
		"what's":  "what is",
		"how's":   "how is",
		"where's": "where is",
	}

	for contraction, expansion := range contractions {
		normalized = strings.ReplaceAll(normalized, contraction, expansion)
	}

	// Clean up extra whitespace
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

	return strings.TrimSpace(normalized)
}

// extractKeywords extracts meaningful keywords from the input
func (p *PromptParser) extractKeywords(input string) []string {
	words := strings.Fields(input)
	var keywords []string

	for _, word := range words {
		// Clean the word
		cleaned := strings.ToLower(regexp.MustCompile(`[^\w]`).ReplaceAllString(word, ""))

		// Skip empty words, stop words, and very short words
		if len(cleaned) < 3 || p.stopWords[cleaned] {
			continue
		}

		// Prioritize technical keywords
		if p.techKeywords[cleaned] {
			keywords = append(keywords, cleaned)
			continue
		}

		// Include if it looks like a meaningful word
		if p.isLikelyKeyword(cleaned) {
			keywords = append(keywords, cleaned)
		}
	}

	return p.deduplicateKeywords(keywords)
}

// extractEntities finds named entities in the input
func (p *PromptParser) extractEntities(input string) []models.Entity {
	var entities []models.Entity

	for entityType, pattern := range p.entityPatterns {
		matches := pattern.FindAllStringSubmatch(input, -1)
		for _, match := range matches {
			if len(match) > 1 {
				// Find the actual captured group
				value := ""
				for i := 1; i < len(match); i++ {
					if match[i] != "" {
						value = match[i]
						break
					}
				}

				if value != "" {
					// Find position in original string
					start := strings.Index(input, value)
					if start != -1 {
						entities = append(entities, models.Entity{
							Type:  entityType,
							Value: value,
							Start: start,
							End:   start + len(value),
						})
					}
				}
			}
		}
	}

	return entities
}

// calculateIntentScores calculates confidence scores for each intent type
func (p *PromptParser) calculateIntentScores(input string, keywords []string) map[models.QueryType]float64 {
	scores := make(map[models.QueryType]float64)

	// Evaluate each pattern type
	p.evaluatePatterns(p.searchPatterns, input, keywords, scores)
	p.evaluatePatterns(p.generationPatterns, input, keywords, scores)
	p.evaluatePatterns(p.explanationPatterns, input, keywords, scores)
	p.evaluatePatterns(p.debuggingPatterns, input, keywords, scores)
	p.evaluatePatterns(p.testingPatterns, input, keywords, scores)
	p.evaluatePatterns(p.reviewPatterns, input, keywords, scores)
	p.evaluatePatterns(p.documentationPatterns, input, keywords, scores)

	// Normalize scores
	return p.normalizeScores(scores)
}

// evaluatePatterns evaluates a set of patterns against the input
func (p *PromptParser) evaluatePatterns(patterns []*IntentPattern, input string, keywords []string, scores map[models.QueryType]float64) {
	for _, pattern := range patterns {
		if pattern.Pattern.MatchString(input) {
			// Base score from pattern weight
			score := pattern.Weight

			// Boost score if keywords match
			keywordBonus := p.calculateKeywordBonus(pattern.Keywords, keywords)
			score += keywordBonus

			// Update score (take maximum if intent already has a score)
			if existing, exists := scores[pattern.QueryType]; exists {
				scores[pattern.QueryType] = max(existing, score)
			} else {
				scores[pattern.QueryType] = score
			}
		}
	}
}

// calculateKeywordBonus calculates bonus score based on keyword matches
func (p *PromptParser) calculateKeywordBonus(patternKeywords, inputKeywords []string) float64 {
	if len(patternKeywords) == 0 || len(inputKeywords) == 0 {
		return 0.0
	}

	matches := 0
	for _, pk := range patternKeywords {
		for _, ik := range inputKeywords {
			if strings.Contains(ik, pk) || strings.Contains(pk, ik) {
				matches++
				break
			}
		}
	}

	return float64(matches) / float64(len(patternKeywords)) * 0.2
}

// selectPrimaryIntent chooses the intent with the highest confidence
func (p *PromptParser) selectPrimaryIntent(scores map[models.QueryType]float64) (models.QueryType, float64) {
	var bestIntent models.QueryType
	var bestScore float64

	for intent, score := range scores {
		if score > bestScore {
			bestScore = score
			bestIntent = intent
		}
	}

	// Default to search if no clear intent
	if bestIntent == "" || bestScore < 0.3 {
		return models.QueryTypeSearch, 0.5
	}

	return bestIntent, min(bestScore, 1.0)
}

// selectSecondaryIntents chooses alternative intents
func (p *PromptParser) selectSecondaryIntents(scores map[models.QueryType]float64, primary models.QueryType) []models.QueryType {
	var secondary []models.QueryType
	threshold := 0.4

	for intent, score := range scores {
		if intent != primary && score >= threshold {
			secondary = append(secondary, intent)
		}
	}

	return secondary
}

// extractFileTargets finds file references in the input
func (p *PromptParser) extractFileTargets(input string) []string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_/]*\.(go|mod|sum|yaml|yml|json))\b`),
		regexp.MustCompile(`\b(in|from|file)\s+([a-zA-Z_][a-zA-Z0-9_/]*)\b`),
	}

	var targets []string
	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(input, -1)
		for _, match := range matches {
			if len(match) > 1 {
				targets = append(targets, match[1])
			}
		}
	}

	return p.deduplicateStrings(targets)
}

// extractFunctionTargets finds function references in the input
func (p *PromptParser) extractFunctionTargets(input string) []string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s*\(\s*\)`),
		regexp.MustCompile(`\b(function|method)\s+([a-zA-Z_][a-zA-Z0-9_]*)\b`),
	}

	var targets []string
	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(input, -1)
		for _, match := range matches {
			if len(match) > 1 {
				targets = append(targets, match[len(match)-1])
			}
		}
	}

	return p.deduplicateStrings(targets)
}

// Helper functions
func (p *PromptParser) isLikelyKeyword(word string) bool {
	// Must be at least 3 characters
	if len(word) < 3 {
		return false
	}

	// Should contain mostly letters
	letterCount := 0
	for _, r := range word {
		if unicode.IsLetter(r) {
			letterCount++
		}
	}

	return float64(letterCount)/float64(len(word)) >= 0.7
}

func (p *PromptParser) deduplicateKeywords(keywords []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, keyword := range keywords {
		if !seen[keyword] {
			seen[keyword] = true
			result = append(result, keyword)
		}
	}

	return result
}

func (p *PromptParser) deduplicateStrings(strs []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, str := range strs {
		if str != "" && !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}

	return result
}

func (p *PromptParser) normalizeScores(scores map[models.QueryType]float64) map[models.QueryType]float64 {
	// Find maximum score
	maxScore := 0.0
	for _, score := range scores {
		if score > maxScore {
			maxScore = score
		}
	}

	// Normalize if needed
	if maxScore > 1.0 {
		for intent, score := range scores {
			scores[intent] = score / maxScore
		}
	}

	return scores
}

// createStopWords returns common stop words to filter out
func createStopWords() map[string]bool {
	words := []string{
		"a", "an", "and", "are", "as", "at", "be", "by", "for", "from", "has", "he", "in", "is", "it",
		"its", "of", "on", "that", "the", "to", "was", "will", "with", "the", "this", "but", "they",
		"have", "had", "what", "said", "each", "which", "their", "time", "will", "about", "if", "up",
		"out", "many", "then", "them", "these", "so", "some", "her", "would", "make", "like", "into",
		"him", "has", "two", "more", "very", "after", "words", "long", "than", "first", "been", "call",
		"who", "oil", "sit", "now", "find", "down", "day", "did", "get", "come", "made", "may", "part",
	}

	stopWords := make(map[string]bool)
	for _, word := range words {
		stopWords[word] = true
	}
	return stopWords
}

// createTechKeywords returns common technical keywords
func createTechKeywords() map[string]bool {
	words := []string{
		"function", "method", "struct", "interface", "type", "package", "import", "variable", "const",
		"error", "panic", "defer", "goroutine", "channel", "mutex", "sync", "context", "http", "json",
		"database", "sql", "api", "rest", "grpc", "test", "mock", "benchmark", "debug", "log", "config",
		"server", "client", "handler", "middleware", "router", "endpoint", "service", "repository",
		"model", "controller", "authentication", "authorization", "jwt", "oauth", "cors", "tls", "ssl",
		"docker", "kubernetes", "redis", "postgres", "mongodb", "elasticsearch", "kafka", "rabbitmq",
	}

	techWords := make(map[string]bool)
	for _, word := range words {
		techWords[word] = true
	}
	return techWords
}

// Helper functions for Go 1.21+ compatibility
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
