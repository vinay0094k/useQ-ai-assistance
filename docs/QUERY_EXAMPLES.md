# Query Processing Examples

## 🔍 Search Queries

### Simple Search
```
Query: "find authentication functions"
  ↓
Intent: IntentSearch (confidence: 0.9)
Context: minimal
Operations: [semantic_search, keyword_search]
Agent: SearchAgent
Commands: find . -name "*.go" -exec grep -l "auth\|Auth" {} \;
Response: List of files with authentication code
```

### Pattern Search  
```
Query: "show me similar patterns to our authentication"
  ↓
Intent: IntentSearch (confidence: 0.8)
Context: moderate  
Operations: [pattern_search, similar_code_examples]
Agent: ContextAwareSearchAgent
Commands: [filesystem_scan, vector_search]
Response: Similar authentication patterns with usage examples
```

## 🛠️ Generation Queries

### Simple Generation
```
Query: "create a hello world function"
  ↓
Intent: IntentGenerate (confidence: 0.9)
Context: minimal
Operations: [pattern_search]
Agent: CodingAgent
Commands: find . -name "*.go" -exec grep -l "func.*main\|Hello" {} \;
Response: Generated Go function following project patterns
```

### Complex Generation
```
Query: "create a microservice for user authentication with logging and monitoring"
  ↓
Intent: IntentGenerate (confidence: 0.9)
Context: comprehensive
Operations: [pattern_search, similar_code_examples, dependency_analysis, project_conventions]
Agent: IntelligenceCodingAgent
Commands: [
  find . -name "*.go" -exec grep -l "auth\|service\|handler" {} \;,
  find . -name "*.go" -exec grep -l "log\|monitor" {} \;,
  tree -I "vendor" -L 3
]
Response: Complete microservice with authentication, logging, monitoring
```

## 📊 System Queries

### System Status
```
Query: "show me current CPU usage"
  ↓
Intent: IntentSystemStatus (confidence: 0.95)
Context: minimal
Operations: [system_info]
Agent: SystemAgent
Commands: ps -o pid,ppid,%mem,%cpu,comm
Response: Current process information and resource usage
```

### File Statistics
```
Query: "how many files are indexed"
  ↓
Intent: IntentSystemStatus (confidence: 0.8)
Context: minimal
Operations: [file_count, index_status]
Agent: SearchAgent
Commands: [
  find . -name "*.go" -type f | wc -l,
  sqlite3 storage/useq.db "SELECT COUNT(*) FROM files"
]
Response: "77 Go files found, 45 indexed in database"
```

## 🧠 Explanation Queries

### Architecture Explanation
```
Query: "explain the flow of this application"
  ↓
Intent: IntentExplain (confidence: 0.9)
Context: comprehensive
Operations: [filesystem_structure, code_analysis, dependency_mapping, architecture_analysis]
Agent: IntelligentQueryProcessor
Commands: [
  find . -name "*.go" -type f,
  tree -I "vendor|node_modules" -L 3,
  find . -name "*.go" -exec grep -l "main\|CLI\|Agent" {} \;
]
LLM: Enhanced prompt with full project context
Response: Detailed architectural explanation with component interactions
```

### Code Flow Explanation
```
Query: "how does error handling work in this project"
  ↓
Intent: IntentExplain (confidence: 0.85)
Context: moderate
Operations: [code_analysis, pattern_search, usage_examples]
Agent: IntelligenceCodingAgent
Commands: [
  find . -name "*.go" -exec grep -l "error\|Error\|err" {} \;,
  grep -n "if err != nil" internal/**/*.go
]
Response: Error handling patterns with specific examples from codebase
```

## 🔧 Debug/Analysis Queries

### Code Analysis
```
Query: "analyze the performance of the indexer"
  ↓
Intent: IntentAnalyze (confidence: 0.9)
Context: comprehensive
Operations: [code_analysis, performance_analysis, dependency_mapping]
Agent: IntelligenceCodingAgent
Commands: [
  find . -path "*/indexer/*" -name "*.go",
  grep -n "time\|performance\|benchmark" internal/indexer/*.go
]
Response: Performance analysis with bottlenecks and optimization suggestions
```

## 🔄 Fallback Examples

### LLM Provider Fallback
```
Query: "create a REST API handler"
  ↓
Primary: OpenAI GPT-4
  ├─ API Call: SUCCESS
  ├─ Tokens: 1,200
  ├─ Cost: $0.036
  └─ Response: Generated REST handler code

Alternative Flow (if OpenAI fails):
  ↓
Fallback: Gemini Pro
  ├─ API Call: SUCCESS
  ├─ Tokens: 1,200  
  ├─ Cost: $0.0126 (cheaper)
  └─ Response: Generated REST handler code

Final Fallback (if all LLMs fail):
  ↓
Enhanced Search: 
  ├─ Vector search: "REST API handler examples"
  ├─ Results: Existing handler patterns from codebase
  ├─ Cost: $0.00
  └─ Response: "Found 3 similar handlers in your project: [files...]"
```

### Agent Routing Fallback
```
Query: "complex architectural refactoring request"
  ↓
Primary: IntelligenceCodingAgent
  ├─ CanHandle(): false (LLM not available)
  └─ Confidence: 0.3

Fallback: CodingAgent
  ├─ CanHandle(): true
  ├─ Confidence: 0.6
  └─ Process(): Basic code generation

Final Fallback: SearchAgent
  ├─ CanHandle(): true (always)
  ├─ Confidence: 0.5
  └─ Process(): Search for similar patterns
```

## 📈 Performance Optimization Flow

### Cache Hit Scenario
```
Query: "explain the flow of this application" (repeated)
  ↓
Cache Check: HIT (pattern: "explain_flow_application")
  ├─ Cached context: project_structure (valid for 20min)
  ├─ Cached operations: [filesystem_structure, code_analysis]
  ├─ Skip: MCP command execution
  ├─ Processing time: 200ms (vs 2.3s cold)
  └─ Cost savings: 80% reduction
```

### Learning Optimization
```
After 5 similar queries:
  ↓
LearningEngine optimizations:
  ├─ Learned pattern: "explain_*_application" → optimal_ops
  ├─ Pre-cache: project structure on startup
  ├─ Predict: next likely query types
  ├─ Optimize: command execution order
  └─ Result: 3x faster processing for learned patterns
```