# 3-Tier Query Classification System

## Overview

The useQ AI Assistant now implements a 3-tier classification system that dramatically reduces costs and improves performance by routing most queries away from expensive LLM calls.

## Classification Tiers

### **Tier 1: Simple Queries (80% of traffic)**
- **Route to**: MCP Filesystem Direct
- **Cost**: $0 | **Time**: <100ms | **No LLM**
- **Processing**: Direct filesystem operations, formatted output

**Patterns Detected:**
- Action verbs: `list`, `show`, `display`, `get`, `read`, `cat`, `open`
- File operations: `what files`, `files in`, `show directory`, `ls`, `tree`, `pwd`
- Status checks: `memory`, `status`, `health`, `system info`
- Direct reads: `show me main.go`, `read config.yaml`

**Examples:**
```
✓ "list files in agents folder"        → find ./agents -type f
✓ "show me main.go"                     → cat cmd/main.go  
✓ "what files are in internal/"         → ls internal/
✓ "memory usage"                        → ps -o %mem,%cpu
✓ "system status"                       → system info display
```

### **Tier 2: Medium Queries (15% of traffic)**
- **Route to**: MCP + Vector Search  
- **Cost**: $0 | **Time**: <500ms | **No LLM**
- **Processing**: Filesystem + vector search, structured results

**Patterns Detected:**
- Search operations: `find`, `search`, `locate`, `where is`
- Code lookups: `show all functions`, `find handlers`, `locate tests`
- Pattern matching: `functions that`, `files containing`, `structs with`
- Counting: `how many`, `count`, `number of`

**Examples:**
```
✓ "find all authentication code"       → grep + vector search
✓ "search for error handling patterns" → semantic search
✓ "where are the agent implementations" → filesystem + vector
✓ "show me all test files"             → find *_test.go
✓ "how many Go files"                  → find + count
```

### **Tier 3: Complex Queries (5% of traffic)**
- **Route to**: Full Pipeline with LLM
- **Cost**: $0.01-0.03 | **Time**: 1-3s | **Uses LLM**
- **Processing**: Full context gathering + LLM synthesis

**Patterns Detected:**
- Explanations: `explain`, `describe`, `how does`, `what is`, `tell me about`
- Analysis: `analyze`, `review`, `improve`, `refactor`, `optimize`, `suggest`
- Generation: `create`, `generate`, `write`, `add`, `implement`, `build`
- Architecture: `architecture`, `design`, `flow`, `pattern`, `structure`
- Multi-step: contains `and`, `then`, `also`, `plus`, `additionally`

**Examples:**
```
✓ "explain the flow of this application"              → Full LLM pipeline
✓ "analyze the authentication system and suggest improvements" → LLM analysis
✓ "create a new agent following existing patterns"    → LLM generation
✓ "how does the caching system work"                  → LLM explanation
✓ "what is the overall architecture"                  → LLM architecture analysis
```

## Decision Tree Implementation

```
Query received
    ↓
Does it match Complex patterns?
    YES → Route to Tier 3 (Full LLM Pipeline)
    NO ↓
    
Does it match Simple patterns?  
    YES → Route to Tier 1 (MCP Direct)
    NO ↓
    
Does it match Medium patterns?
    YES → Route to Tier 2 (MCP + Vector)
    NO ↓
    
Default → Route to Tier 2 (safer than assuming complex)
```

## Expected Performance Improvements

### **Before Classification System:**
- 100 queries = $3.74 (all routed to LLM)
- Average time: 2s per query
- Total cost: $3.74

### **After Classification System:**
- 80 simple queries × $0.00 = $0.00
- 15 medium queries × $0.00 = $0.00  
- 5 complex queries × $0.02 = $0.10
- **Total cost: $0.10 (97% cost reduction)**
- **Average time: 350ms per query (82% faster)**

## Testing the System

```bash
# Test all tiers
useQ> mcp test

# Test Tier 1 (Simple)
useQ> list files
useQ> show directory  
useQ> memory usage

# Test Tier 2 (Medium)
useQ> find authentication code
useQ> how many Go files
useQ> search for error handling

# Test Tier 3 (Complex)  
useQ> explain the flow of this application
useQ> create a microservice for authentication
useQ> analyze the architecture
```

## Monitoring Classification

```bash
# View classification statistics
useQ> classification stats

# Expected output:
📊 Query Classification Statistics:
├─ Total Queries: 100
├─ Simple (Tier 1): 80 (80.0%)
├─ Medium (Tier 2): 15 (15.0%)  
├─ Complex (Tier 3): 5 (5.0%)
└─ Cost Savings: $3.64 (97.3% reduction)
```

This system ensures that only truly complex queries that require AI reasoning use expensive LLM calls, while simple filesystem operations and searches are handled efficiently without any AI costs.