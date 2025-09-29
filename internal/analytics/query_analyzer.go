package analytics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// QueryAnalyzer collects and analyzes real query patterns to validate assumptions
type QueryAnalyzer struct {
	queries     []QueryRecord
	mu          sync.RWMutex
	logFile     *os.File
	startTime   time.Time
	testMode    bool
	comparisons []SearchComparison
}

// QueryRecord tracks every query for pattern analysis
type QueryRecord struct {
	ID                string                 `json:"id"`
	UserInput         string                 `json:"user_input"`
	Timestamp         time.Time              `json:"timestamp"`
	ActualTier        string                 `json:"actual_tier"`
	PredictedTier     string                 `json:"predicted_tier"`
	ProcessingTime    time.Duration          `json:"processing_time"`
	ActualCost        float64                `json:"actual_cost"`
	TokensUsed        int                    `json:"tokens_used"`
	Success           bool                   `json:"success"`
	UserSatisfaction  int                    `json:"user_satisfaction"` // 1-5 rating
	ManualClassification string              `json:"manual_classification"`
	Context           map[string]interface{} `json:"context"`
}

// SearchComparison compares vector search vs keyword search accuracy
type SearchComparison struct {
	Query           string    `json:"query"`
	VectorResults   []string  `json:"vector_results"`
	KeywordResults  []string  `json:"keyword_results"`
	UserPreferred   string    `json:"user_preferred"` // "vector", "keyword", "both", "neither"
	VectorAccuracy  float64   `json:"vector_accuracy"`
	KeywordAccuracy float64   `json:"keyword_accuracy"`
	Timestamp       time.Time `json:"timestamp"`
}

// ValidationReport provides data-driven insights
type ValidationReport struct {
	TotalQueries      int                    `json:"total_queries"`
	ActualDistribution map[string]int        `json:"actual_distribution"`
	PredictedDistribution map[string]int     `json:"predicted_distribution"`
	ClassificationAccuracy float64           `json:"classification_accuracy"`
	ActualCosts       CostBreakdown          `json:"actual_costs"`
	PredictedCosts    CostBreakdown          `json:"predicted_costs"`
	SearchAccuracy    SearchAccuracyReport   `json:"search_accuracy"`
	UserSatisfaction  SatisfactionReport     `json:"user_satisfaction"`
	Recommendations   []string               `json:"recommendations"`
	GeneratedAt       time.Time              `json:"generated_at"`
}

// CostBreakdown tracks actual vs predicted costs
type CostBreakdown struct {
	Tier1Cost   float64 `json:"tier1_cost"`
	Tier2Cost   float64 `json:"tier2_cost"`
	Tier3Cost   float64 `json:"tier3_cost"`
	TotalCost   float64 `json:"total_cost"`
	IndexingCost float64 `json:"indexing_cost"`
}

// SearchAccuracyReport compares search methods
type SearchAccuracyReport struct {
	VectorSearchAccuracy  float64 `json:"vector_search_accuracy"`
	KeywordSearchAccuracy float64 `json:"keyword_search_accuracy"`
	VectorPreferenceRate  float64 `json:"vector_preference_rate"`
	TotalComparisons      int     `json:"total_comparisons"`
}

// SatisfactionReport tracks user satisfaction
type SatisfactionReport struct {
	AverageRating    float64            `json:"average_rating"`
	RatingsByTier    map[string]float64 `json:"ratings_by_tier"`
	SatisfactionRate float64            `json:"satisfaction_rate"` // % of ratings >= 3
}

// NewQueryAnalyzer creates a new query analyzer
func NewQueryAnalyzer() (*QueryAnalyzer, error) {
	// Create analytics directory
	if err := os.MkdirAll("analytics", 0755); err != nil {
		return nil, fmt.Errorf("failed to create analytics directory: %w", err)
	}

	// Create log file for raw query data
	logFile, err := os.OpenFile(
		filepath.Join("analytics", fmt.Sprintf("queries_%s.jsonl", time.Now().Format("2006-01-02"))),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create query log file: %w", err)
	}

	return &QueryAnalyzer{
		queries:     make([]QueryRecord, 0),
		logFile:     logFile,
		startTime:   time.Now(),
		testMode:    false,
		comparisons: make([]SearchComparison, 0),
	}, nil
}

// RecordQuery records a query for analysis
func (qa *QueryAnalyzer) RecordQuery(query *models.Query, response *models.Response, predictedTier, actualTier string) {
	qa.mu.Lock()
	defer qa.mu.Unlock()

	record := QueryRecord{
		ID:                query.ID,
		UserInput:         query.UserInput,
		Timestamp:         time.Now(),
		ActualTier:        actualTier,
		PredictedTier:     predictedTier,
		ProcessingTime:    response.Metadata.GenerationTime,
		ActualCost:        response.Cost.TotalCost,
		TokensUsed:        response.TokenUsage.TotalTokens,
		Success:           response.Type != models.ResponseTypeError,
		UserSatisfaction:  0, // Will be filled by user feedback
		ManualClassification: "", // Will be filled by manual review
	}

	qa.queries = append(qa.queries, record)

	// Write to log file immediately
	if qa.logFile != nil {
		jsonData, _ := json.Marshal(record)
		qa.logFile.WriteString(string(jsonData) + "\n")
		qa.logFile.Sync()
	}

	// Print real-time validation info
	qa.printRealTimeValidation(record)
}

// printRealTimeValidation shows validation info in real-time
func (qa *QueryAnalyzer) printRealTimeValidation(record QueryRecord) {
	fmt.Printf("\n📊 VALIDATION DATA:\n")
	fmt.Printf("├─ Query: %s\n", record.UserInput)
	fmt.Printf("├─ Predicted: %s | Actual: %s", record.PredictedTier, record.ActualTier)
	
	if record.PredictedTier == record.ActualTier {
		fmt.Printf(" ✅\n")
	} else {
		fmt.Printf(" ❌ MISCLASSIFICATION\n")
	}
	
	fmt.Printf("├─ Cost: $%.6f | Time: %v\n", record.ActualCost, record.ProcessingTime.Truncate(time.Millisecond))
	fmt.Printf("└─ Tokens: %d\n", record.TokensUsed)
}

// CompareSearchMethods compares vector search vs keyword search
func (qa *QueryAnalyzer) CompareSearchMethods(query string, vectorResults, keywordResults []string) {
	fmt.Printf("\n🔬 SEARCH COMPARISON:\n")
	fmt.Printf("Query: %s\n", query)
	fmt.Printf("Vector results: %v\n", vectorResults)
	fmt.Printf("Keyword results: %v\n", keywordResults)
	fmt.Printf("Which is more relevant? (v)ector, (k)eyword, (b)oth, (n)either: ")

	// In a real implementation, you'd collect user input here
	// For now, just record the comparison
	comparison := SearchComparison{
		Query:          query,
		VectorResults:  vectorResults,
		KeywordResults: keywordResults,
		Timestamp:      time.Now(),
	}

	qa.mu.Lock()
	qa.comparisons = append(qa.comparisons, comparison)
	qa.mu.Unlock()
}

// GenerateValidationReport creates a comprehensive validation report
func (qa *QueryAnalyzer) GenerateValidationReport() *ValidationReport {
	qa.mu.RLock()
	defer qa.mu.RUnlock()

	if len(qa.queries) == 0 {
		return &ValidationReport{
			TotalQueries: 0,
			Recommendations: []string{
				"❌ NO DATA COLLECTED YET",
				"Run at least 50 queries to get meaningful insights",
				"Use 'useQ> validate start' to begin data collection",
			},
			GeneratedAt: time.Now(),
		}
	}

	// Calculate actual distribution
	actualDist := make(map[string]int)
	predictedDist := make(map[string]int)
	correctPredictions := 0
	totalCosts := CostBreakdown{}
	totalSatisfaction := 0.0
	ratedQueries := 0

	for _, record := range qa.queries {
		actualDist[record.ActualTier]++
		predictedDist[record.PredictedTier]++
		
		if record.ActualTier == record.PredictedTier {
			correctPredictions++
		}

		// Accumulate costs by actual tier
		switch record.ActualTier {
		case "tier1":
			totalCosts.Tier1Cost += record.ActualCost
		case "tier2":
			totalCosts.Tier2Cost += record.ActualCost
		case "tier3":
			totalCosts.Tier3Cost += record.ActualCost
		}
		totalCosts.TotalCost += record.ActualCost

		if record.UserSatisfaction > 0 {
			totalSatisfaction += float64(record.UserSatisfaction)
			ratedQueries++
		}
	}

	// Calculate accuracy
	accuracy := float64(correctPredictions) / float64(len(qa.queries))

	// Generate recommendations
	recommendations := qa.generateRecommendations(actualDist, totalCosts, accuracy)

	return &ValidationReport{
		TotalQueries:           len(qa.queries),
		ActualDistribution:     actualDist,
		PredictedDistribution:  predictedDist,
		ClassificationAccuracy: accuracy,
		ActualCosts:           totalCosts,
		PredictedCosts:        qa.calculatePredictedCosts(len(qa.queries)),
		UserSatisfaction: SatisfactionReport{
			AverageRating:    totalSatisfaction / float64(max(ratedQueries, 1)),
			SatisfactionRate: qa.calculateSatisfactionRate(),
		},
		Recommendations: recommendations,
		GeneratedAt:     time.Now(),
	}
}

// generateRecommendations provides data-driven recommendations
func (qa *QueryAnalyzer) generateRecommendations(actualDist map[string]int, costs CostBreakdown, accuracy float64) []string {
	var recommendations []string
	total := float64(len(qa.queries))

	// Check distribution assumptions
	tier1Pct := float64(actualDist["tier1"]) / total * 100
	tier2Pct := float64(actualDist["tier2"]) / total * 100
	tier3Pct := float64(actualDist["tier3"]) / total * 100

	if tier1Pct < 70 {
		recommendations = append(recommendations, 
			fmt.Sprintf("⚠️ Tier 1 only %.1f%% (expected 80%%) - tune simple patterns", tier1Pct))
	}

	if tier3Pct > 10 {
		recommendations = append(recommendations, 
			fmt.Sprintf("⚠️ Tier 3 is %.1f%% (expected 5%%) - too many complex queries", tier3Pct))
	}

	// Check classification accuracy
	if accuracy < 0.8 {
		recommendations = append(recommendations, 
			fmt.Sprintf("❌ Classification accuracy %.1f%% - improve patterns", accuracy*100))
	}

	// Check if costs match predictions
	if costs.TotalCost > 0.10 && len(qa.queries) >= 100 {
		recommendations = append(recommendations, 
			fmt.Sprintf("💰 Actual cost $%.4f higher than predicted - check tier distribution", costs.TotalCost))
	}

	// Check if VectorDB is worth it
	if tier2Pct < 5 {
		recommendations = append(recommendations, 
			"🤔 Very few Tier 2 queries - consider SQLite FTS instead of VectorDB")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "✅ System performing as expected")
	}

	return recommendations
}

// PrintValidationSummary prints current validation status
func (qa *QueryAnalyzer) PrintValidationSummary() {
	qa.mu.RLock()
	defer qa.mu.RUnlock()

	fmt.Printf("\n📊 VALIDATION STATUS:\n")
	fmt.Printf("├─ Queries collected: %d\n", len(qa.queries))
	fmt.Printf("├─ Time period: %v\n", time.Since(qa.startTime).Truncate(time.Second))
	
	if len(qa.queries) >= 10 {
		// Show preliminary distribution
		dist := make(map[string]int)
		for _, q := range qa.queries {
			dist[q.ActualTier]++
		}
		
		total := float64(len(qa.queries))
		fmt.Printf("├─ Preliminary distribution:\n")
		fmt.Printf("│  ├─ Tier 1: %d (%.1f%%)\n", dist["tier1"], float64(dist["tier1"])/total*100)
		fmt.Printf("│  ├─ Tier 2: %d (%.1f%%)\n", dist["tier2"], float64(dist["tier2"])/total*100)
		fmt.Printf("│  └─ Tier 3: %d (%.1f%%)\n", dist["tier3"], float64(dist["tier3"])/total*100)
	}
	
	if len(qa.queries) < 50 {
		fmt.Printf("└─ ⚠️ Need %d more queries for reliable analysis\n", 50-len(qa.queries))
	} else {
		fmt.Printf("└─ ✅ Sufficient data for analysis\n")
	}
}

// StartValidationMode enables detailed query tracking
func (qa *QueryAnalyzer) StartValidationMode() {
	qa.mu.Lock()
	defer qa.mu.Unlock()
	
	qa.testMode = true
	fmt.Printf("🧪 VALIDATION MODE STARTED\n")
	fmt.Printf("├─ Every query will be logged and analyzed\n")
	fmt.Printf("├─ Manual classification prompts enabled\n")
	fmt.Printf("├─ Search method comparisons enabled\n")
	fmt.Printf("└─ Run 50+ queries to get meaningful data\n\n")
}

// ExportValidationData exports all collected data for analysis
func (qa *QueryAnalyzer) ExportValidationData() error {
	qa.mu.RLock()
	defer qa.mu.RUnlock()

	// Export query records
	queryData, err := json.MarshalIndent(qa.queries, "", "  ")
	if err != nil {
		return err
	}

	queryFile := filepath.Join("analytics", fmt.Sprintf("query_analysis_%s.json", time.Now().Format("2006-01-02")))
	if err := os.WriteFile(queryFile, queryData, 0644); err != nil {
		return err
	}

	// Export search comparisons
	comparisonData, err := json.MarshalIndent(qa.comparisons, "", "  ")
	if err != nil {
		return err
	}

	comparisonFile := filepath.Join("analytics", fmt.Sprintf("search_comparisons_%s.json", time.Now().Format("2006-01-02")))
	if err := os.WriteFile(comparisonFile, comparisonData, 0644); err != nil {
		return err
	}

	fmt.Printf("📊 Validation data exported:\n")
	fmt.Printf("├─ Query data: %s\n", queryFile)
	fmt.Printf("└─ Search comparisons: %s\n", comparisonFile)

	return nil
}

// PromptManualClassification asks user to manually classify a query
func (qa *QueryAnalyzer) PromptManualClassification(query string) string {
	if !qa.testMode {
		return ""
	}

	fmt.Printf("\n🔍 MANUAL CLASSIFICATION NEEDED:\n")
	fmt.Printf("Query: \"%s\"\n", query)
	fmt.Printf("How would you classify this? (1)Simple (2)Medium (3)Complex: ")
	
	// In a real implementation, you'd read user input here
	// For now, return empty - this would be filled by user interaction
	return ""
}

// Close closes the analyzer and saves final report
func (qa *QueryAnalyzer) Close() error {
	if qa.logFile != nil {
		qa.logFile.Close()
	}

	// Generate final report
	report := qa.GenerateValidationReport()
	reportData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	reportFile := filepath.Join("analytics", fmt.Sprintf("validation_report_%s.json", time.Now().Format("2006-01-02")))
	if err := os.WriteFile(reportFile, reportData, 0644); err != nil {
		return err
	}

	fmt.Printf("\n📋 FINAL VALIDATION REPORT: %s\n", reportFile)
	qa.printFinalSummary(report)

	return nil
}

// printFinalSummary prints the final validation summary
func (qa *QueryAnalyzer) printFinalSummary(report *ValidationReport) {
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("📊 VALIDATION RESULTS (%d queries)\n", report.TotalQueries)
	fmt.Printf(strings.Repeat("=", 60) + "\n")

	// Distribution comparison
	fmt.Printf("\nDISTRIBUTION ANALYSIS:\n")
	fmt.Printf("                 Predicted  Actual   Difference\n")
	fmt.Printf("Tier 1 (Simple):    80%%      %.1f%%     %+.1f%%\n", 
		float64(report.ActualDistribution["tier1"])/float64(report.TotalQueries)*100,
		float64(report.ActualDistribution["tier1"])/float64(report.TotalQueries)*100 - 80)
	fmt.Printf("Tier 2 (Medium):    15%%      %.1f%%     %+.1f%%\n",
		float64(report.ActualDistribution["tier2"])/float64(report.TotalQueries)*100,
		float64(report.ActualDistribution["tier2"])/float64(report.TotalQueries)*100 - 15)
	fmt.Printf("Tier 3 (Complex):    5%%      %.1f%%     %+.1f%%\n",
		float64(report.ActualDistribution["tier3"])/float64(report.TotalQueries)*100,
		float64(report.ActualDistribution["tier3"])/float64(report.TotalQueries)*100 - 5)

	// Cost analysis
	fmt.Printf("\nCOST ANALYSIS:\n")
	fmt.Printf("Actual total cost: $%.4f\n", report.ActualCosts.TotalCost)
	fmt.Printf("Predicted cost: $%.4f\n", report.PredictedCosts.TotalCost)
	fmt.Printf("Difference: $%.4f\n", report.ActualCosts.TotalCost - report.PredictedCosts.TotalCost)

	// Classification accuracy
	fmt.Printf("\nCLASSIFICATION ACCURACY: %.1f%%\n", report.ClassificationAccuracy*100)

	// Recommendations
	fmt.Printf("\nRECOMMENDATIONS:\n")
	for _, rec := range report.Recommendations {
		fmt.Printf("• %s\n", rec)
	}

	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
}

// Helper methods
func (qa *QueryAnalyzer) calculatePredictedCosts(totalQueries int) CostBreakdown {
	// Based on assumed 80/15/5 distribution
	tier1Count := int(float64(totalQueries) * 0.8)
	tier2Count := int(float64(totalQueries) * 0.15)
	tier3Count := int(float64(totalQueries) * 0.05)

	return CostBreakdown{
		Tier1Cost: float64(tier1Count) * 0.0,
		Tier2Cost: float64(tier2Count) * 0.0005,
		Tier3Cost: float64(tier3Count) * 0.025,
		TotalCost: float64(tier2Count)*0.0005 + float64(tier3Count)*0.025,
	}
}

func (qa *QueryAnalyzer) calculateSatisfactionRate() float64 {
	if len(qa.queries) == 0 {
		return 0.0
	}

	satisfied := 0
	rated := 0
	for _, q := range qa.queries {
		if q.UserSatisfaction > 0 {
			rated++
			if q.UserSatisfaction >= 3 {
				satisfied++
			}
		}
	}

	if rated == 0 {
		return 0.0
	}
	return float64(satisfied) / float64(rated)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}