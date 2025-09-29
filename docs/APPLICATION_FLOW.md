# useQ AI Assistant - Complete Application Flow

## 🏗️ Application Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    CLI Entry Point                          │
│                   (cmd/main.go)                             │
├─────────────────────────────────────────────────────────────┤
│                 Application Layer                           │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │              CLIApplication                             │ │
│  │         (internal/app/cli.go)                           │ │
│  └─────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                Intelligence Layer                           │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │           Intelligent Query Processor                   │ │
│  │        (internal/mcp/intelligent_query_processor.go)    │ │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────┐   │ │
│  │  │ Intent      │ │ Context     │ │ Prompt          │   │ │
│  │  │ Classifier  │ │ Gatherer    │ │ Builder         │   │ │
│  │  └─────────────┘ └─────────────┘ └─────────────────┘   │ │
│  └─────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                   Agent Layer                               │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │ Manager     │ │ Search      │ │ Coding              │   │
│  │ Agent       │ │ Agent       │ │ Agent               │   │
│  └─────────────┘ └─────────────┘ └─────────────────────┘   │
├─────────────────────────────────────────────────────────────┤
│                 MCP System Layer                            │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │              Intelligent Executor                       │ │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────┐   │ │
│  │  │ Command     │ │ Safety      │ │ Filesystem      │   │ │
│  │  │ Registry    │ │ Validator   │ │ Server          │   │ │
│  │  └─────────────┘ └─────────────┘ └─────────────────┘   │ │
│  └─────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                   Core Services                             │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │ LLM         │ │ Vector      │ │ Storage             │   │
│  │ Manager     │ │ Database    │ │ (SQLite)            │   │
│  └─────────────┘ └─────────────┘ └─────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## 🔄 Application Startup Flow

### 1. **Initialization Sequence**
```
main() → Load .env → Initialize Config → Create CLIApplication
  ↓
CLIApplication.NewCLIApplication()
  ├─ Initialize Storage (SQLite)
  ├─ Initialize Vector Database (Qdrant)
  ├─ Initialize LLM Manager (OpenAI/Gemini from .env)
  ├─ Initialize MCP Client
  ├─ Initialize Code Indexer
  ├─ Initialize Agents (Manager, Search, Coding, Intelligence)
  └─ Auto-index if no files found
  ↓
Start Interactive CLI Loop
```

### 2. **Component Dependencies**
```
CLIApplication
├─ SessionManager (user sessions)
├─ PromptParser (intent parsing)
├─ CodeIndexer (file indexing)
├─ VectorDB (Qdrant client)
├─ LLMManager (multi-provider)
├─ ManagerAgent
│   ├─ SearchAgent
│   ├─ CodingAgent
│   ├─ IntelligenceCodingAgent
│   ├─ ContextAwareSearchAgent
│   └─ SystemAgent
└─ MCPClient
    ├─ IntelligentQueryProcessor
    ├─ IntelligentExecutor
    ├─ FilesystemServer
    └─ Learning/Caching systems
```

## 🧠 Complete Query Processing Flow

### **Example Query: "explain the flow of this application"**

```
┌─────────────────────────────────────────────────────────────┐
│  STEP 1: CLI INPUT PROCESSING                               │
└─────────────────────────────────────────────────────────────┘
User Input: "explain the flow of this application"
  ↓
main.go:runInteractiveCLI()
  ├─ Generate Query ID: query_1758912812199199876
  ├─ Create models.Query object
  ├─ Load environment context
  └─ Call CLIApplication.ProcessQuery()

┌─────────────────────────────────────────────────────────────┐
│  STEP 2: INTELLIGENT INTENT CLASSIFICATION                 │
└─────────────────────────────────────────────────────────────┘
CLIApplication.ProcessQuery() → ManagerAgent.RouteQuery()
  ↓
ManagerAgent.shouldUseIntelligentProcessing()
  ├─ Detects: "explain" + "flow" + "application"
  ├─ Classification: COMPLEX ARCHITECTURAL QUERY
  ├─ Decision: Route to IntelligentQueryProcessor
  └─ Reason: Requires deep context + architectural understanding

┌─────────────────────────────────────────────────────────────┐
│  STEP 3: INTELLIGENT QUERY PROCESSING PIPELINE             │
└─────────────────────────────────────────────────────────────┘
IntelligentQueryProcessor.ProcessQuery()
  ↓
3.1: Intent Classification
  ├─ IntentClassifier.ClassifyIntent()
  ├─ Primary: IntentExplain (confidence: 0.9)
  ├─ Complexity: 8/10 (architectural explanation)
  ├─ Required context: [project_structure, code_examples, architecture]
  ├─ Token budget: 4000 (explanations need more tokens)
  └─ Quality requirements: examples + context + validation

3.2: Execution Plan Creation
  ├─ Required operations: [filesystem_structure, code_analysis, dependency_mapping]
  ├─ Context depth: comprehensive
  ├─ Parallel operations: [filesystem_scan, vector_search, system_info]
  ├─ Cache strategy: 15min TTL, pre-cache enabled
  └─ Fallback strategies: [openai, gemini, enhanced_search]

┌─────────────────────────────────────────────────────────────┐
│  STEP 4: PARALLEL CONTEXT GATHERING                        │
└─────────────────────────────────────────────────────────────┘
ParallelContextGatherer.GatherContext() - 4 parallel operations:

4.1: [Parallel] Project Structure Analysis
  ├─ IntelligentExecutor.executeFileSystemCommand()
  ├─ Commands: find . -name "*.go" -type f
  ├─ Commands: tree -I "vendor|node_modules" -L 3
  ├─ Result: 77 Go files, directory structure
  └─ Cache: project_structure (15min TTL)

4.2: [Parallel] Vector Search (if VectorDB available)
  ├─ Generate embedding for: "application flow architecture"
  ├─ Search similar code in indexed files
  ├─ Result: main.go, cli.go, manager_agent.go (top matches)
  └─ Boost MCP-discovered files (+0.1 relevance score)

4.3: [Parallel] System Information
  ├─ Commands: ps -o pid,%mem,%cpu,comm
  ├─ Commands: du -sh .
  ├─ Result: memory usage, disk usage
  └─ Cache: system_status (5min TTL)

4.4: [Parallel] Code Examples
  ├─ Extract key architectural files
  ├─ Files: cmd/main.go, internal/app/cli.go, internal/agents/manager_agent.go
  └─ Result: relevant code snippets

┌─────────────────────────────────────────────────────────────┐
│  STEP 5: INTELLIGENT CONTEXT FILTERING                     │
└─────────────────────────────────────────────────────────────┘
Filter and prioritize gathered context:
  ├─ Merge: filesystem + vector + system data
  ├─ Remove duplicates and low-relevance items
  ├─ Prioritize by: relevance score + recency + usage patterns
  ├─ Truncate to fit token budget (prevent context overflow)
  └─ Structure hierarchically: summary → details

┌─────────────────────────────────────────────────────────────┐
│  STEP 6: ADAPTIVE PROMPT CONSTRUCTION                      │
└─────────────────────────────────────────────────────────────┘
AdaptivePromptBuilder.BuildPrompt():
  ├─ System prompt: "You are an expert software architect analyzing a Go application..."
  ├─ Project context: "77 Go files, CLI architecture, key directories: internal/, cmd/, models/"
  ├─ Key files: cmd/main.go, internal/app/cli.go, internal/agents/manager_agent.go
  ├─ Structure: internal/ (agents, app, mcp, vectordb), cmd/, models/, config/
  └─ Quality instructions: "Provide detailed architectural explanation with component interactions"

┌─────────────────────────────────────────────────────────────┐
│  STEP 7: LLM GENERATION WITH FALLBACK                      │
└─────────────────────────────────────────────────────────────┘
LLM Generation with automatic fallback:
  ↓
7.1: Primary - OpenAI GPT-4
  ├─ Load API key from .env: OPENAI_API_KEY
  ├─ Model: gpt-4-turbo-preview
  ├─ Send enhanced prompt with full context
  ├─ Stream response token-by-token
  ├─ Track: 1,247 input tokens, 892 output tokens
  ├─ Cost: $0.0374
  └─ Success: Generate comprehensive explanation
  
7.2: Fallback - Gemini Pro (if OpenAI fails)
  ├─ Load API key from .env: GEMINI_API_KEY
  ├─ Adjust prompt format for Gemini
  ├─ Model: gemini-1.5-pro
  └─ Fallback cost calculation
  
7.3: Final Fallback - Enhanced Search Results
  ├─ Return structured search results
  ├─ Include file references and snippets
  └─ No LLM cost

┌─────────────────────────────────────────────────────────────┐
│  STEP 8: RESPONSE POST-PROCESSING                          │
└─────────────────────────────────────────────────────────────┘
ResponseProcessor.EnhanceResponse():
  ├─ Add source attribution: [main.go, cli.go, manager_agent.go]
  ├─ Add execution metadata: 2.3s processing time
  ├─ Add token usage: 1,247 input + 892 output = 2,139 total
  ├─ Add cost tracking: $0.0374
  ├─ Format with file references
  ├─ Confidence score: 0.92
  └─ Add tools used: [filesystem_analysis, semantic_search, llm_generation]

┌─────────────────────────────────────────────────────────────┐
│  STEP 9: LEARNING & FEEDBACK LOOP                          │
└─────────────────────────────────────────────────────────────┘
LearningEngine.RecordSuccess():
  ├─ Pattern: "explain_flow_application" → success
  ├─ Optimal operations: [filesystem_structure, code_analysis]
  ├─ Update cache TTL: extend to 20min (frequently accessed)
  ├─ Store for future prediction
  ├─ Pre-cache related patterns
  └─ Update usage tracker

┌─────────────────────────────────────────────────────────────┐
│  FINAL RESPONSE                                             │
└─────────────────────────────────────────────────────────────┘
Response: "This application follows a CLI-based architecture where the main entry point (cmd/main.go) initializes a CLIApplication that uses a ManagerAgent to intelligently route queries to specialized agents. The flow involves: 1) Intent parsing, 2) MCP integration for filesystem context, 3) Vector database search, 4) LLM generation with project context, and 5) Structured response formatting..."

📁 Key Files Referenced:
- cmd/main.go
- internal/app/cli.go  
- internal/agents/manager_agent.go

📊 System Context:
- Indexed files: 77
- Processing time: 2.3s
- Tokens used: 2,139
- Cost: $0.0374
```

## 🔄 Detailed Query Processing Flow

### **Phase 1: Query Reception & Preprocessing**
```
User Input → CLI Reader → Input Validation → Query Object Creation
  ↓
models.Query{
  ID: "query_1758912812199199876"
  UserInput: "explain the flow of this application"
  Language: "go"
  Context: {Environment, ProjectRoot}
  Timestamp: time.Now()
}
```

### **Phase 2: Intelligent Routing Decision**
```
CLIApplication.ProcessQuery() → ManagerAgent.RouteQuery()
  ↓
Routing Analysis:
  ├─ shouldUseIntelligentProcessing() → TRUE
  │   ├─ Detects: "explain" + "flow" + "application"
  │   ├─ Classification: ARCHITECTURAL_EXPLANATION
  │   └─ Complexity: HIGH (requires deep context)
  │
  ├─ Route Decision: IntelligentQueryProcessor
  └─ Fallback: Traditional agent routing
```

### **Phase 3: Intelligent Processing Pipeline**
```
IntelligentQueryProcessor.ProcessQuery()
  ↓
3.1: Intent Classification
  ├─ IntentClassifier analyzes query patterns
  ├─ Primary: IntentExplain (confidence: 0.9)
  ├─ Secondary: [IntentAnalyze] (confidence: 0.6)
  ├─ Complexity: 8/10 (architectural explanation)
  ├─ Required context: [project_structure, code_examples, architecture]
  ├─ Token budget: 4000
  └─ Quality requirements: {examples: true, context: true, validation: true}

3.2: Execution Plan Creation
  ├─ Required operations: [filesystem_structure, code_analysis, dependency_mapping]
  ├─ Context depth: comprehensive
  ├─ Parallel operations: [filesystem_scan, vector_search, system_info]
  ├─ Cache strategy: {TTL: 15min, pre_cache: true}
  └─ Fallback strategies: [openai, gemini, enhanced_search]
```

### **Phase 4: Parallel Context Gathering**
```
ParallelContextGatherer.GatherContext() - 4 concurrent operations:

[Goroutine 1] Project Structure Analysis
  ├─ IntelligentExecutor.executeFileSystemCommand()
  ├─ Commands executed:
  │   ├─ find . -name "*.go" -type f
  │   ├─ tree -I "vendor|node_modules" -L 3
  │   └─ du -sh .
  ├─ Results: 77 Go files, directory tree, size info
  └─ Cache: project_structure (15min TTL)

[Goroutine 2] Vector Search (if available)
  ├─ Generate embedding: "application flow architecture"
  ├─ Search indexed codebase
  ├─ Results: main.go, cli.go, manager_agent.go (top matches)
  ├─ Relevance scores: [0.89, 0.76, 0.71]
  └─ Boost MCP-discovered files (+0.1 score)

[Goroutine 3] System Information
  ├─ Commands executed:
  │   ├─ ps -o pid,%mem,%cpu,comm
  │   └─ whoami && pwd
  ├─ Results: process info, memory usage, current user/directory
  └─ Cache: system_status (5min TTL)

[Goroutine 4] Code Examples
  ├─ Extract from key architectural files
  ├─ Files: cmd/main.go, internal/app/cli.go, internal/agents/manager_agent.go
  ├─ Parse function signatures and comments
  └─ Results: relevant code snippets with context
```

### **Phase 5: Context Filtering & Optimization**
```
Filter and prioritize gathered context:
  ├─ Merge: filesystem + vector + system + code data
  ├─ Remove duplicates and low-relevance items (<0.3 score)
  ├─ Prioritize by: relevance_score × recency × usage_frequency
  ├─ Truncate to fit token budget (prevent context overflow)
  ├─ Structure hierarchically: summary → details → examples
  └─ Result: Optimized context within token limits
```

### **Phase 6: Adaptive Prompt Construction**
```
AdaptivePromptBuilder.BuildPrompt():
  ↓
System Prompt:
"You are an expert software architect analyzing a Go application. 
Provide clear, comprehensive explanations of code architecture and flow.
Focus on: high-level architecture, component interactions, data flow, integration points.
Use the provided project context to give accurate, specific explanations."

User Prompt:
"Explain: explain the flow of this application

Project Context:
- Total Go files: 77
- Key directories: internal, cmd, models, config
- Architecture: CLI application with agent-based routing

PROJECT STRUCTURE:
- cmd/
  - main.go
- internal/
  - app/
  - agents/
  - mcp/
  - vectordb/
- models/
- config/

KEY FILES:
- cmd/main.go
- internal/app/cli.go
- internal/agents/manager_agent.go

Please provide a detailed explanation covering:
1. Overall architecture and design
2. Key components and their roles  
3. Data flow and interactions
4. Important patterns and conventions"
```

### **Phase 7: LLM Generation with Fallback Chain**
```
LLM Generation with automatic provider fallback:
  ↓
7.1: Primary Provider - OpenAI GPT-4
  ├─ Load from .env: OPENAI_API_KEY
  ├─ Model: gpt-4-turbo-preview
  ├─ Request: {messages, max_tokens: 4000, temperature: 0.1}
  ├─ Send enhanced prompt with full context
  ├─ Stream response token-by-token
  ├─ Track usage: 1,247 input + 892 output = 2,139 tokens
  ├─ Calculate cost: $0.0374
  └─ SUCCESS → Continue to post-processing
  
7.2: Fallback Provider - Gemini Pro (if OpenAI fails)
  ├─ Load from .env: GEMINI_API_KEY  
  ├─ Model: gemini-1.5-pro
  ├─ Adjust prompt format for Gemini
  ├─ Different cost calculation
  └─ If SUCCESS → Continue, else next fallback
  
7.3: Fallback Provider - Cohere (if Gemini fails)
  ├─ Load from .env: COHERE_API_KEY
  ├─ Model: command-r-plus
  └─ If SUCCESS → Continue, else final fallback
  
7.4: Final Fallback - Enhanced Search Results
  ├─ Return structured search results from vector DB
  ├─ Include file references and code snippets
  ├─ Add context from MCP operations
  ├─ No LLM cost
  └─ Confidence: 0.6 (lower than LLM response)
```

### **Phase 8: Response Post-Processing**
```
ResponseProcessor.EnhanceResponse():
  ├─ Add source attribution: [main.go, cli.go, manager_agent.go]
  ├─ Add execution metadata: 2.3s total processing time
  ├─ Add token usage: 2,139 tokens
  ├─ Add cost tracking: $0.0374
  ├─ Format with file references
  ├─ Calculate confidence score: 0.92
  ├─ Add tools used: [filesystem_analysis, semantic_search, llm_generation]
  └─ Add reasoning: "Analyzed 77 files using explain approach with comprehensive context depth"
```

### **Phase 9: Learning & Optimization**
```
LearningEngine.RecordSuccess():
  ├─ Pattern learned: "explain_flow_application" → success
  ├─ Optimal operations: [filesystem_structure, code_analysis]
  ├─ Success rate: 95% for this pattern type
  ├─ Average time: 2.1s
  ├─ Update cache TTL: extend to 20min (frequently accessed)
  ├─ Store for future prediction
  ├─ Pre-cache related patterns: ["explain_architecture", "flow_analysis"]
  └─ Update usage tracker for performance optimization
```

## 🔄 Fallback Strategies

### **1. LLM Provider Fallback Chain**
```
Primary: OpenAI GPT-4
  ├─ API Key: .env OPENAI_API_KEY
  ├─ Model: gpt-4-turbo-preview
  ├─ Cost: $0.01/$0.03 per 1K tokens
  └─ If fails → Gemini Pro
      ├─ API Key: .env GEMINI_API_KEY
      ├─ Model: gemini-1.5-pro
      ├─ Cost: $0.0035/$0.0105 per 1K tokens
      └─ If fails → Cohere
          ├─ API Key: .env COHERE_API_KEY
          ├─ Model: command-r-plus
          ├─ Cost: $0.003/$0.015 per 1K tokens
          └─ If fails → Enhanced Search Results
```

### **2. Agent Routing Fallback**
```
Intelligent Processing
  ├─ If complex query → IntelligentQueryProcessor
  └─ If fails → Traditional Agent Routing
      ├─ ManagerAgent.selectBestAgent()
      ├─ Score each agent's capability
      ├─ Route to highest scoring agent
      └─ If agent fails → Search Agent (default fallback)
```

### **3. Context Gathering Fallback**
```
Parallel Context Gathering
  ├─ MCP Operations (filesystem, git, system)
  │   ├─ If command fails → Skip that operation
  │   └─ Continue with available context
  ├─ Vector Search
  │   ├─ If VectorDB unavailable → Skip semantic search
  │   └─ Use keyword search instead
  └─ Code Examples
      ├─ If file read fails → Use cached examples
      └─ Generate synthetic examples if needed
```

### **4. Response Generation Fallback**
```
Response Generation
  ├─ If LLM generation succeeds → Enhanced response
  ├─ If LLM fails → Structured search results
  ├─ If search fails → Basic file listing
  └─ If all fails → Error response with helpful guidance
```

## 🎯 Key Improvements Implemented

1. **Environment Variable Auto-Loading**: API keys loaded from `.env` automatically
2. **Intelligent Command Execution**: System determines what commands to run based on query
3. **Parallel Processing**: Multiple operations run simultaneously for performance
4. **Adaptive Caching**: Smart caching based on usage patterns and query types
5. **Multi-Provider Fallback**: Automatic fallback between OpenAI, Gemini, Cohere
6. **Learning Engine**: System learns optimal processing patterns over time
7. **Safety Validation**: All commands validated before execution
8. **Context Optimization**: Intelligent filtering to stay within token limits

## 🚀 Performance Characteristics

- **Cold Start**: ~3-5 seconds (first query, no cache)
- **Warm Cache**: ~500ms-1s (cached context)
- **Learned Patterns**: ~200-500ms (optimized operations)
- **Parallel Speedup**: 2-3x faster than sequential processing
- **Memory Usage**: ~50-100MB (depending on indexed files)
- **Token Efficiency**: 90%+ relevant context (smart filtering)