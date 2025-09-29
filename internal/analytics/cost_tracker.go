package analytics

import (
	"fmt"
	"sync"
	"time"
)

// CostTracker tracks REAL costs vs predictions
type CostTracker struct {
	dailyCosts    map[string]float64 // date -> cost
	embeddingCosts float64
	llmCosts      float64
	totalQueries  int
	mu            sync.RWMutex
}

// NewCostTracker creates a new cost tracker
func NewCostTracker() *CostTracker {
	return &CostTracker{
		dailyCosts: make(map[string]float64),
	}
}

// RecordEmbeddingCost records actual embedding API costs
func (ct *CostTracker) RecordEmbeddingCost(tokens int, cost float64) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.embeddingCosts += cost
	today := time.Now().Format("2006-01-02")
	ct.dailyCosts[today] += cost

	fmt.Printf("💰 Embedding: $%.6f (%d tokens) | Daily total: $%.4f\n", 
		cost, tokens, ct.dailyCosts[today])
}

// RecordLLMCost records actual LLM costs
func (ct *CostTracker) RecordLLMCost(tokens int, cost float64) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.llmCosts += cost
	today := time.Now().Format("2006-01-02")
	ct.dailyCosts[today] += cost

	fmt.Printf("💰 LLM: $%.4f (%d tokens) | Daily total: $%.4f\n", 
		cost, tokens, ct.dailyCosts[today])
}

// GetDailyCost returns today's actual cost
func (ct *CostTracker) GetDailyCost() float64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	today := time.Now().Format("2006-01-02")
	return ct.dailyCosts[today]
}

// GetTotalCost returns total accumulated cost
func (ct *CostTracker) GetTotalCost() float64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.embeddingCosts + ct.llmCosts
}

// PrintCostSummary prints cost breakdown
func (ct *CostTracker) PrintCostSummary() {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	fmt.Printf("\n💰 ACTUAL COST SUMMARY:\n")
	fmt.Printf("├─ Embedding costs: $%.4f\n", ct.embeddingCosts)
	fmt.Printf("├─ LLM costs: $%.4f\n", ct.llmCosts)
	fmt.Printf("├─ Total cost: $%.4f\n", ct.embeddingCosts + ct.llmCosts)
	fmt.Printf("└─ Today's cost: $%.4f\n", ct.GetDailyCost())
}