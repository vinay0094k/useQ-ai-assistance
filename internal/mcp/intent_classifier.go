package mcp

import (
	"context"
	"regexp"
	"strings"

	"github.com/yourusername/useq-ai-assistant/models"
)

// IntentClassifier performs intelligent intent classification
type IntentClassifier struct {
	patterns map[IntentType][]*IntentPattern
}

// IntentPattern represents a pattern for intent detection
type IntentPattern struct {
	Regex       *regexp.Regexp `json:"regex"`
	Weight      float64        `json:"weight"`
	Keywords    []string       `json:"keywords"`
	Context     []string       `json:"context"`
	Complexity  int            `json:"complexity"`
}

// NewIntentClassifier creates a new intent classifier
func NewIntentClassifier() *IntentClassifier {
	classifier := &IntentClassifier{
		patterns: make(map[IntentType][]*IntentPattern),
	}
	classifier.initializePatterns()
	return classifier
}

// ClassifyIntent performs intelligent intent classification with confidence scoring
func (ic *IntentClassifier) ClassifyIntent(ctx context.Context, query *models.Query) (*ClassifiedIntent, error) {
	input := strings.ToLower(query.UserInput)
	
	// Extract keywords and entities
	keywords := ic.extractKeywords(input)
	entities := ic.extractEntities(input)
	
	// Calculate intent scores
	intentScores := ic.calculateIntentScores(input, keywords)
	
	// Determine primary and secondary intents
	primary, confidence := ic.selectPrimaryIntent(intentScores)
	secondary := ic.selectSecondaryIntents(intentScores, primary)
	
	// Assess complexity
	complexity := ic.assessComplexity(input, keywords, entities)
	
	// Determine required context
	requiredContext := ic.determineRequiredContext(primary, input, complexity)
	
	// Determine expected output type
	outputType := ic.determineOutputType(primary, input)
	
	// Set quality requirements
	qualityReqs := ic.determineQualityRequirements(primary, complexity)
	
	return &ClassifiedIntent{
		Primary:             primary,
		Secondary:           secondary,
		Confidence:          confidence,
		ComplexityLevel:     complexity,
		RequiredContext:     requiredContext,
		ExpectedOutputType:  outputType,
		QualityRequirements: qualityReqs,
		Keywords:            keywords,
		Entities:            entities,
	}, nil
}

// initializePatterns sets up intent detection patterns
func (ic *IntentClassifier) initializePatterns() {
	// Explanation patterns
	ic.patterns[IntentExplain] = []*IntentPattern{
		{
			Regex:      regexp.MustCompile(`(?i)\b(explain|describe|what\s+is|how\s+does|walk\s+through|flow)\b`),
			Weight:     0.9,
			Keywords:   []string{"explain", "describe", "flow", "architecture"},
			Complexity: 7,
		},
		{
			Regex:      regexp.MustCompile(`(?i)\b(understand|meaning|purpose|overview)\b`),
			Weight:     0.8,
			Keywords:   []string{"understand", "overview"},
			Complexity: 6,
		},
	}
	
	// Generation patterns
	ic.patterns[IntentGenerate] = []*IntentPattern{
		{
			Regex:      regexp.MustCompile(`(?i)\b(create|generate|make|build|implement|write)\b`),
			Weight:     0.9,
			Keywords:   []string{"create", "generate", "implement"},
			Complexity: 6,
		},
		{
			Regex:      regexp.MustCompile(`(?i)\b(new|add)\s+.*(function|service|handler|api)\b`),
			Weight:     0.8,
			Keywords:   []string{"new", "add"},
			Complexity: 7,
		},
	}
	
	// Search patterns
	ic.patterns[IntentSearch] = []*IntentPattern{
		{
			Regex:      regexp.MustCompile(`(?i)\b(search|find|look\s+for|locate|show|list)\b`),
			Weight:     0.9,
			Keywords:   []string{"search", "find", "show"},
			Complexity: 3,
		},
		{
			Regex:      regexp.MustCompile(`(?i)\b(where\s+is|how\s+many|count)\b`),
			Weight:     0.8,
			Keywords:   []string{"where", "count"},
			Complexity: 2,
		},
	}
	
	// System status patterns
	ic.patterns[IntentSystemStatus] = []*IntentPattern{
		{
			Regex:      regexp.MustCompile(`(?i)\b(status|info|statistics|current|usage|memory|cpu)\b`),
			Weight:     0.9,
			Keywords:   []string{"status", "info", "usage"},
			Complexity: 2,
		},
	}
	
	// Analysis patterns
	ic.patterns[IntentAnalyze] = []*IntentPattern{
		{
			Regex:      regexp.MustCompile(`(?i)\b(analyze|review|check|examine|audit)\b`),
			Weight:     0.9,
			Keywords:   []string{"analyze", "review"},
			Complexity: 8,
		},
	}
}

// calculateIntentScores calculates confidence scores for each intent
func (ic *IntentClassifier) calculateIntentScores(input string, keywords []string) map[IntentType]float64 {
	scores := make(map[IntentType]float64)
	
	for intentType, patterns := range ic.patterns {
		maxScore := 0.0
		
		for _, pattern := range patterns {
			score := 0.0
			
			// Pattern match score
			if pattern.Regex.MatchString(input) {
				score += pattern.Weight
			}
			
			// Keyword match bonus
			keywordMatches := 0
			for _, keyword := range pattern.Keywords {
				for _, inputKeyword := range keywords {
					if strings.Contains(inputKeyword, keyword) || strings.Contains(keyword, inputKeyword) {
						keywordMatches++
					}
				}
			}
			
			if len(pattern.Keywords) > 0 {
				keywordBonus := float64(keywordMatches) / float64(len(pattern.Keywords)) * 0.3
				score += keywordBonus
			}
			
			if score > maxScore {
				maxScore = score
			}
		}
		
		scores[intentType] = maxScore
	}
	
	return scores
}

// selectPrimaryIntent selects the intent with highest confidence
func (ic *IntentClassifier) selectPrimaryIntent(scores map[IntentType]float64) (IntentType, float64) {
	var bestIntent IntentType
	var bestScore float64
	
	for intent, score := range scores {
		if score > bestScore {
			bestScore = score
			bestIntent = intent
		}
	}
	
	// Default to search if no clear intent
	if bestIntent == "" || bestScore < 0.3 {
		return IntentSearch, 0.5
	}
	
	return bestIntent, bestScore
}

// selectSecondaryIntents selects alternative intents
func (ic *IntentClassifier) selectSecondaryIntents(scores map[IntentType]float64, primary IntentType) []IntentType {
	var secondary []IntentType
	threshold := 0.4
	
	for intent, score := range scores {
		if intent != primary && score >= threshold {
			secondary = append(secondary, intent)
		}
	}
	
	return secondary
}

// extractKeywords extracts meaningful keywords from input
func (ic *IntentClassifier) extractKeywords(input string) []string {
	words := strings.Fields(input)
	var keywords []string
	
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true,
	}
	
	for _, word := range words {
		cleaned := strings.ToLower(regexp.MustCompile(`[^\w]`).ReplaceAllString(word, ""))
		if len(cleaned) >= 3 && !stopWords[cleaned] {
			keywords = append(keywords, cleaned)
		}
	}
	
	return keywords
}

// extractEntities extracts entities from the query
func (ic *IntentClassifier) extractEntities(input string) []ExtractedEntity {
	entities := []ExtractedEntity{}
	
	// Extract function names
	funcPattern := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s*\(\s*\)`)
	matches := funcPattern.FindAllStringSubmatch(input, -1)
	for _, match := range matches {
		if len(match) > 1 {
			entities = append(entities, ExtractedEntity{
				Type:       "function",
				Value:      match[1],
				Confidence: 0.8,
				Context:    "function_call",
			})
		}
	}
	
	// Extract file names
	filePattern := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_/]*\.(go|js|py|md))\b`)
	matches = filePattern.FindAllStringSubmatch(input, -1)
	for _, match := range matches {
		if len(match) > 1 {
			entities = append(entities, ExtractedEntity{
				Type:       "file",
				Value:      match[1],
				Confidence: 0.9,
				Context:    "file_reference",
			})
		}
	}
	
	return entities
}

// assessComplexity assesses query complexity
func (ic *IntentClassifier) assessComplexity(input string, keywords []string, entities []ExtractedEntity) int {
	complexity := 3 // Base complexity
	
	// Increase for architectural terms
	architecturalTerms := []string{"architecture", "design", "pattern", "flow", "structure"}
	for _, term := range architecturalTerms {
		if strings.Contains(input, term) {
			complexity += 2
		}
	}
	
	// Increase for multiple requirements
	requirements := []string{"authentication", "logging", "monitoring", "security", "database"}
	reqCount := 0
	for _, req := range requirements {
		if strings.Contains(input, req) {
			reqCount++
		}
	}
	complexity += reqCount
	
	// Increase for multiple entities
	complexity += len(entities) / 2
	
	// Cap at 10
	if complexity > 10 {
		complexity = 10
	}
	
	return complexity
}

// determineRequiredContext determines what context is needed
func (ic *IntentClassifier) determineRequiredContext(primary IntentType, input string, complexity int) []ContextType {
	context := []ContextType{}
	
	switch primary {
	case IntentExplain:
		context = append(context, ContextProjectStructure, ContextCodeExamples)
		if complexity >= 7 {
			context = append(context, ContextArchitecture, ContextDependencies)
		}
		
	case IntentGenerate:
		context = append(context, ContextCodeExamples, ContextUsagePatterns)
		if complexity >= 6 {
			context = append(context, ContextProjectStructure, ContextDependencies)
		}
		
	case IntentSearch:
		context = append(context, ContextCodeExamples)
		if strings.Contains(input, "similar") || strings.Contains(input, "pattern") {
			context = append(context, ContextUsagePatterns)
		}
		
	case IntentSystemStatus:
		context = append(context, ContextSystemInfo)
		
	default:
		context = append(context, ContextProjectStructure)
	}
	
	return context
}

// determineOutputType determines expected output type
func (ic *IntentClassifier) determineOutputType(primary IntentType, input string) OutputType {
	switch primary {
	case IntentExplain:
		return OutputExplanation
	case IntentGenerate:
		return OutputCode
	case IntentSearch:
		return OutputList
	case IntentAnalyze:
		return OutputAnalysis
	case IntentSystemStatus:
		return OutputStatus
	default:
		return OutputExplanation
	}
}

// determineQualityRequirements sets quality requirements
func (ic *IntentClassifier) determineQualityRequirements(primary IntentType, complexity int) QualityRequirements {
	return QualityRequirements{
		MinConfidence:     0.7,
		RequireExamples:   primary == IntentGenerate || primary == IntentExplain,
		RequireContext:    complexity >= 5,
		RequireValidation: primary == IntentGenerate,
	}
}