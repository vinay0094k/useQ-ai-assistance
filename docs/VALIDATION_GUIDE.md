# Query Classification Validation Guide

## 🧪 How to Validate Your Assumptions

Before claiming 97% cost reduction, you need REAL data. Here's how to collect it:

## Step 1: Start Data Collection

```bash
# Enable validation mode
./useq validate start

# This will:
# ✅ Log every query with timing and costs
# ✅ Track classification accuracy
# ✅ Record user satisfaction
# ✅ Compare search methods
```

## Step 2: Collect Real Usage Data

**Run at least 50 diverse queries:**

```bash
# Simple queries (should be Tier 1)
useQ> list files
useQ> show directory
useQ> memory usage
useQ> what files are in internal/

# Medium queries (should be Tier 2)  
useQ> find authentication code
useQ> search for error handling
useQ> where are the test files
useQ> how many Go files

# Complex queries (should be Tier 3)
useQ> explain the architecture
useQ> create a microservice
useQ> analyze the codebase
useQ> how does authentication work
```

## Step 3: Manual Classification Check

For each query, the system will ask:
```
🔍 MANUAL CLASSIFICATION NEEDED:
Query: "find authentication patterns"
How would you classify this? (1)Simple (2)Medium (3)Complex: 
```

**Be honest!** This validates if the automatic classification matches human intuition.

## Step 4: Search Method Comparison

```bash
# Compare vector vs keyword search
./useq validate search

# For each test query:
🔬 SEARCH METHOD COMPARISON
Query: "find authentication code"
Vector results: [auth.go, login.go, jwt.go]
Keyword results: [auth.go, user.go, config.go]
Which is more relevant? (v)ector, (k)eyword, (b)oth, (n)either:
```

## Step 5: Generate Validation Report

```bash
# After 50+ queries
./useq validate report
```

**Expected Output:**
```
📊 VALIDATION RESULTS (73 queries)
========================================

DISTRIBUTION ANALYSIS:
                 Predicted  Actual   Difference
Tier 1 (Simple):    80%      67%     -13%
Tier 2 (Medium):    15%      23%     +8%
Tier 3 (Complex):    5%      10%     +5%

COST ANALYSIS:
Actual total cost: $0.47
Predicted cost: $0.10
Difference: +$0.37 (users ask more complex questions)

CLASSIFICATION ACCURACY: 78%

SEARCH METHOD COMPARISON:
Vector preferred: 45%
Keyword preferred: 35%
Both equally good: 20%

RECOMMENDATIONS:
• ⚠️ More Tier 3 queries than expected - adjust patterns
• 💡 Vector search only marginally better - consider SQLite FTS
• ✅ Classification accuracy acceptable but could improve
```

## What This Tells You

### **If Distribution is Wrong:**
```
Actual: 60% Tier 1, 30% Tier 2, 10% Tier 3
→ Users ask more complex questions than assumed
→ Cost savings: 85% instead of 97%
→ Still good, but adjust expectations
```

### **If Vector Search Isn't Better:**
```
User preference: 60% keyword, 40% vector
→ Vector search not worth embedding costs
→ Recommendation: Use SQLite FTS instead
→ Save $0.50/month in embedding costs
```

### **If Classification is Inaccurate:**
```
Accuracy: 65%
→ Too many misclassifications
→ Tune patterns in query_classifier.go
→ Add more training examples
```

## Alternative Paths Based on Data

### **Path 1: Vector Search Not Worth It**
```
Data shows: Keyword search 90% as good
Action: Remove VectorDB, use SQLite FTS
Result: Zero external dependencies, $0 search costs
```

### **Path 2: Different Distribution**
```
Data shows: 50% Tier 1, 40% Tier 2, 10% Tier 3
Action: Adjust cost estimates and patterns
Result: Still 90% cost reduction (not 97%)
```

### **Path 3: Classification Needs Work**
```
Data shows: 60% accuracy
Action: Improve patterns, add machine learning
Result: Better routing, more accurate costs
```

## Honest Questions to Answer

1. **Do you actually need semantic search?**
   - Test: Run same queries with SQLite FTS
   - Measure: Accuracy difference
   - Decide: Is 10% accuracy worth $4/month?

2. **Are your query patterns realistic?**
   - Test: Log 100 real queries from actual usage
   - Measure: Actual distribution vs predicted
   - Adjust: Cost estimates based on real data

3. **Is the complexity justified?**
   - Alternative: Start with SQLite FTS + MCP only
   - Measure: User satisfaction without vector search
   - Add: VectorDB only if FTS isn't good enough

## Success Criteria

**Before claiming success, validate:**
- [ ] Classification accuracy >80%
- [ ] Actual distribution within 10% of predicted
- [ ] Vector search preferred >70% of time
- [ ] Total monthly cost <$10 for 100 queries/day
- [ ] User satisfaction >4/5 stars

**If any criteria fail:**
- Adjust implementation
- Revise cost estimates  
- Consider simpler alternatives

This validation approach ensures you're building the right system based on real data, not assumptions.