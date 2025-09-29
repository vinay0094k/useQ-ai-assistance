# VectorDB Integration Guide

## Overview

The VectorDB package provides intelligent semantic search capabilities that integrate seamlessly with the 3-tier query classification system.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    VectorDB Package                         │
├─────────────────────────────────────────────────────────────┤
│  QdrantClient (gRPC + HTTP fallback)                       │
│  ├─ Connection Management                                   │
│  ├─ Collection Management                                   │
│  ├─ Vector Storage & Retrieval                             │
│  └─ Health Monitoring                                       │
├─────────────────────────────────────────────────────────────┤
│  EmbeddingService (OpenAI + Caching)                       │
│  ├─ Text-to-Vector Conversion                              │
│  ├─ Batch Processing                                        │
│  ├─ Intelligent Caching                                    │
│  └─ Fallback Strategies                                     │
├─────────────────────────────────────────────────────────────┤
│  SearchService (Tier-Aware Search)                         │
│  ├─ Semantic Search                                         │
│  ├─ Result Ranking                                          │
│  ├─ Context Retrieval                                       │
│  └─ Performance Optimization                                │
├─────────────────────────────────────────────────────────────┤
│  Supporting Services                                        │
│  ├─ RankingService (Multi-factor ranking)                  │
│  ├─ ContextRetrieval (Surrounding code)                    │
│  ├─ VectorOptimizer (Performance tuning)                   │
│  └─ MaintenanceService (Health & cleanup)                  │
└─────────────────────────────────────────────────────────────┘
```

## Integration with 3-Tier System

### **Tier 1 (Simple Queries)**
- **VectorDB Role**: Not used (direct MCP only)
- **Performance**: 0ms vector processing
- **Cost**: $0

### **Tier 2 (Medium Queries)**
- **VectorDB Role**: Semantic search without LLM synthesis
- **Performance**: <500ms including embedding generation
- **Cost**: $0 (no LLM, only embedding API)

```go
// Example Tier 2 usage
results, err := searchService.Search(ctx, "find authentication code", 10, nil)
// Returns structured results without LLM explanation
```

### **Tier 3 (Complex Queries)**
- **VectorDB Role**: Comprehensive context retrieval for LLM
- **Performance**: 1-3s including context gathering
- **Cost**: $0.01-0.03 (LLM processing)

```go
// Example Tier 3 usage
contextResults, err := contextRetrieval.RetrieveForTier3(ctx, "explain authentication flow", 5)
// Returns rich context for LLM prompt enhancement
```

## Key Features

### **1. Intelligent Connection Management**
- **Primary**: gRPC connection for performance
- **Fallback**: HTTP API if gRPC fails
- **Auto-retry**: Automatic connection recovery

### **2. Smart Embedding Generation**
- **Primary**: OpenAI embeddings (from .env OPENAI_API_KEY)
- **Caching**: Intelligent caching to reduce API calls
- **Fallback**: Hash-based embeddings for testing

### **3. Multi-Factor Ranking**
```go
RankingWeights{
    Similarity:    0.6,  // Vector similarity (primary)
    TextMatch:     0.2,  // Keyword matching
    FileRelevance: 0.1,  // File importance
    Recency:       0.05, // Code freshness
    Frequency:     0.05, // Access patterns
}
```

### **4. Context-Aware Retrieval**
- **Surrounding Code**: Gets code before/after matches
- **File Context**: Includes related functions in same file
- **Package Context**: Finds related code in same package
- **Usage Examples**: Discovers how code is used

## Performance Optimizations

### **1. Batch Processing**
```go
// Efficient batch operations
optimizer.BatchUpsert(ctx, points, 100)
```

### **2. Intelligent Caching**
```go
// Embedding cache reduces API calls
cache.Set(text, embedding) // Cache for reuse
```

### **3. Connection Fallback**
```go
// Automatic fallback chain
gRPC → HTTP → Error (with detailed logging)
```

### **4. Result Optimization**
```go
// Smart result limiting based on tier
Tier2: limit=10, threshold=0.7  // High confidence only
Tier3: limit=20, threshold=0.3  // More results for LLM context
```

## Usage Examples

### **Basic Search (Tier 2)**
```go
searchService := vectordb.NewSearchService(client, embedder)
results, err := searchService.Search(ctx, "authentication functions", 10, nil)
```

### **Context Retrieval (Tier 3)**
```go
contextRetrieval := vectordb.NewContextRetrieval(searchService, rankingService)
contextResults, err := contextRetrieval.RetrieveForTier3(ctx, "explain auth flow", 5)
```

### **Maintenance Operations**
```go
maintenance := vectordb.NewMaintenanceService(client)
err := maintenance.OptimizeCollection(ctx)
```

## Configuration

```yaml
vectordb:
  host: "localhost"
  port: 6333
  collection: "code_embeddings"
  vector_size: 1536
  batch_size: 100
  max_retries: 3
  connection_timeout: "30s"
```

## Environment Variables

```bash
# Required for embeddings
OPENAI_API_KEY=your_openai_api_key_here

# Optional Qdrant configuration
QDRANT_URL=localhost:6333
QDRANT_API_KEY=your_qdrant_api_key_if_needed
```

## Health Monitoring

```go
// Check VectorDB health
healthStatus, err := maintenance.HealthCheck(ctx)
if healthStatus.Healthy {
    fmt.Printf("✅ VectorDB healthy: %d vectors indexed\n", 
        healthStatus.CollectionStats.PointsCount)
}
```

## Error Handling & Fallbacks

1. **Connection Errors**: gRPC → HTTP → Graceful degradation
2. **Embedding Errors**: OpenAI → Hash-based → Continue without embeddings
3. **Search Errors**: Vector search → Keyword search → Return empty results
4. **Storage Errors**: Retry → Log error → Continue processing

The VectorDB package now provides robust, tier-aware semantic search that enhances the 3-tier classification system while maintaining performance and cost efficiency.