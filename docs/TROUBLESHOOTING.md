# Troubleshooting Guide

## 🔧 Common Issues and Solutions

### 0. **3-Tier Classification Issues**

**Problem**: All queries going to Tier 3 (expensive LLM calls)

**Solution**:
```bash
# Check classification stats
useQ> mcp test

# Should show distribution like:
# Simple (Tier 1): 80%
# Medium (Tier 2): 15% 
# Complex (Tier 3): 5%
```

**Problem**: High embedding costs in Tier 2

**Solution**:
```bash
# Check embedding cost tracking
useQ> status

# Monitor costs
grep "Actual cost" logs/steps_$(date +%Y-%m-%d).log
```

### 1. **LLM Provider Issues**

**Problem**: `❌ LLM Manager not available: primary provider 'openai' not available`

**Solution**:
```bash
# Check .env file exists
ls -la .env

# Add your API key
echo "OPENAI_API_KEY=your-key-here" >> .env

# Verify it's loaded
grep OPENAI_API_KEY .env
```

**Test**:
```bash
useQ> create a simple function
# Should now use OpenAI for generation
```

### 2. **Vector Database Connection Issues**

**Problem**: `❌ Qdrant connection failed`

**Solution**:
```bash
# Start Qdrant locally
docker run -p 6333:6333 qdrant/qdrant

# Test connection
curl http://localhost:6333/collections

# Check if collection exists
curl http://localhost:6333/collections/code_embeddings
```

**Test**:
```bash
useQ> find authentication code
# Should perform Tier 2 vector search
# Cost: ~$0.0005 for query embedding
```

### 2.1. **High Embedding Costs**

**Problem**: Unexpected high costs from embeddings

**Diagnosis**:
```bash
# Check actual costs
grep "Actual cost" logs/steps_*.log | tail -10

# Expected costs:
# Query embedding: $0.000005-0.0005
# If seeing $0.01+ per query, something's wrong
```

**Solution**:
```bash
# Check if queries are being classified correctly
useQ> mcp test

# Verify Tier 1 queries skip embeddings entirely
useQ> list files
# Should show: Cost: $0.00 (no embedding generation)

# Verify Tier 2 queries use embeddings
useQ> find auth code  
# Should show: Cost: ~$0.0005 (query embedding only)
```

### 3. **No Files Indexed**

**Problem**: `📭 No files indexed yet`

**Solution**:
```bash
# Run full indexing
useQ> reindex

# Or incremental indexing
useQ> index
```

**Test**:
```bash
useQ> indexed
# Should show list of indexed files
```

### 4. **MCP Commands Not Working**

**Problem**: Commands not executing or returning empty results

**Solution**:
```bash
# Test MCP integration
useQ> mcp test

# Check command permissions
which find tree ps git

# Verify project structure
useQ> show project structure
```

### 5. **Poor Response Quality**

**Problem**: Tier 1/2 responses lack context compared to Tier 3

**This is expected behavior**:
- **Tier 1**: Direct filesystem results (no AI synthesis)
- **Tier 2**: Search results without explanation (no LLM)
- **Tier 3**: Full AI-powered responses with context

**If you need richer responses**:
```bash
# Force Tier 3 processing for specific queries
useQ> explain how authentication works
# This will use LLM for rich explanations
```

### 6. **Classification Accuracy Issues**

**Problem**: Wrong tier classification

**Debug**:
```bash
# Test specific patterns
useQ> list files          # Should be Tier 1
useQ> find auth code      # Should be Tier 2  
useQ> explain architecture # Should be Tier 3

# Check classification logs
grep "Query classified" logs/steps_$(date +%Y-%m-%d).log
```

**Tune classification**:
```go
// Adjust patterns in internal/mcp/query_classifier.go
// if classification is wrong
```
### 7. **Performance Issues**

**Problem**: Slow response times

**Check tier distribution first**:
```bash
# Most queries should be Tier 1/2 (fast)
useQ> classification stats

# Expected:
# Tier 1: 80% (45ms avg)
# Tier 2: 15% (320ms avg)
# Tier 3: 5% (2.1s avg)
```

**Optimization**:
```bash
# Check cache performance
useQ> status

# Check embedding cache hit rate
grep "Cache hit" logs/steps_*.log | wc -l

# Optimize vector collection (if using Qdrant)
useQ> maintenance optimize
```

### 8. **Budget Exceeded**

**Problem**: Monthly costs higher than expected

**Analysis**:
```bash
# Check actual tier distribution
grep "tier.*cost" logs/steps_*.log | sort | uniq -c

# Expected distribution:
# 80% Tier 1 ($0.00)
# 15% Tier 2 ($0.0005)  
# 5% Tier 3 ($0.025)

# If seeing too many Tier 3, tune classification
```

**Emergency cost control**:
```bash
# Disable LLM temporarily (force all to Tier 1/2)
unset OPENAI_API_KEY
./useq
# Still functional with MCP + vector search
```

## 🐛 Debug Mode

Enable detailed logging:
```bash
export DEBUG_MODE=true
export LOG_LEVEL=debug

# View real-time logs
tail -f logs/steps_$(date +%Y-%m-%d).log

# Monitor costs in real-time
tail -f logs/steps_$(date +%Y-%m-%d).log | grep -E "cost|Cost"
```

## 📊 Health Checks

```bash
# Check all components and costs
useQ> status

# Test tier classification
useQ> mcp test

# Test each tier
useQ> list files           # Tier 1: $0.00
useQ> find auth code       # Tier 2: ~$0.0005
useQ> explain architecture # Tier 3: ~$0.025

# Check actual costs
useQ> cost stats
```

## 💰 Cost Monitoring Commands

```bash
# Daily cost summary
grep "Actual.*cost" logs/steps_$(date +%Y-%m-%d).log | \
  awk '{sum += $4} END {printf "Today: $%.4f\n", sum}'

# Query distribution
grep "tier.*confidence" logs/steps_*.log | \
  cut -d'"' -f4 | sort | uniq -c

# Embedding cache hit rate
grep -c "Cache hit" logs/steps_$(date +%Y-%m-%d).log
```

## 🔄 Reset Everything

If things get corrupted:
```bash
# Clean databases
rm -f storage/useq.db agents_test.db*

# Clear caches and logs
rm -rf logs/

# Reset Qdrant collection
curl -X DELETE http://localhost:6333/collections/code_embeddings

# Rebuild
go build -o useq cmd/main.go

# Fresh start
./useq
useQ> reindex

# Monitor indexing costs
tail -f logs/steps_$(date +%Y-%m-%d).log | grep -E "cost|Cost"
```

## 🎯 Expected Performance After Fixes

### **Cost Distribution (100 queries/day)**
```
80 Tier 1 queries × $0.00 = $0.00
15 Tier 2 queries × $0.000005 = $0.000075  
5 Tier 3 queries × $0.045 = $0.225

Daily total: $0.225
Monthly total: $6.75

Without classification: $135/month
Savings: $128.25/month (95% reduction)
```

### **Response Time Distribution**
```
80% queries: 45ms (Tier 1)
15% queries: 320ms (Tier 2)
5% queries: 2.1s (Tier 3)

Average: 0.35s (vs 2s without classification)
Improvement: 82% faster
```

This realistic analysis shows the true costs and benefits of the 3-tier system.