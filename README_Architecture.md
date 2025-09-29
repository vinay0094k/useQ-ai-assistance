### **🏗️ High-Level Architecture**

┌─────────────────────────────────────────────────────────────┐
│                    AI Assistant System                      │
├─────────────────────────────────────────────────────────────┤
│  CLI Interface (cmd/main.go)                               │
├─────────────────────────────────────────────────────────────┤
│                   Agent Layer                               │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │ Manager     │ │ Coding      │ │ Search              │   │
│  │ Agent       │ │ Agent       │ │ Agent               │   │
│  └─────────────┘ └─────────────┘ └─────────────────────┘   │
├─────────────────────────────────────────────────────────────┤
│                 Intelligence Layer                          │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │              MCP System (Enhanced)                      │ │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────┐   │ │
│  │  │ Usage       │ │ Predictive  │ │ File            │   │ │
│  │  │ Tracker     │ │ Cache       │ │ Watcher         │   │ │
│  │  └─────────────┘ └─────────────┘ └─────────────────┘   │ │
│  └─────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                   Core Services                             │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │ LLM         │ │ Vector      │ │ Storage             │   │
│  │ Manager     │ │ Database    │ │ (SQLite)            │   │
│  └─────────────┘ └─────────────┘ └─────────────────────┘   │
└─────────────────────────────────────────────────────────────┘


### **🧠 MCP Intelligence System Architecture**

┌─────────────────────────────────────────────────────────────┐
│                    MCP Client                               │
├─────────────────────────────────────────────────────────────┤
│  Learning & Prediction Layer                               │
│  ┌─────────────────┐ ┌─────────────────────────────────┐   │
│  │ Usage Tracker   │ │ Predictive Cache Manager        │   │
│  │ - Pattern Learn │ │ - Operation Prediction          │   │
│  │ - Access Count  │ │ - Adaptive TTL                  │   │
│  │ - Time Analysis │ │ - Background Pre-caching        │   │
│  └─────────────────┘ └─────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────┤
│  Caching & Monitoring Layer                               │
│  ┌─────────────────┐ ┌─────────────────────────────────┐   │
│  │ Context Cache   │ │ File Watcher                    │   │
│  │ - 5min TTL      │ │ - Real-time Invalidation        │   │
│  │ - Hash Validate │ │ - .go File Monitoring           │   │
│  └─────────────────┘ └─────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────┤
│  Core MCP Operations                                       │
│  ┌─────────────────┐ ┌─────────────────────────────────┐   │
│  │ Filesystem      │ │ Decision Engine                 │   │
│  │ Server          │ │ - Query Analysis                │   │
│  │ - File Search   │ │ - Operation Selection           │   │
│  │ - Structure     │ │ - Requirement Detection         │   │
│  └─────────────────┘ └─────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘


### **📊 Data Flow Architecture**

User Query → Manager Agent → MCP Client → Intelligence Pipeline

Intelligence Pipeline:
1. Usage Tracker (Learn patterns)
2. Predictive Cache (Check cache/predict operations)  
3. Context Cache (Fast retrieval)
4. File Watcher (Real-time updates)
5. Filesystem Server (Actual operations)
6. Enhanced Context → Agents → LLM → Response


### **🔄 Learning Feedback Loop**

Query → Process → Learn → Predict → Optimize → Faster Response
  ↑                                                      ↓
  └──────────── Continuous Improvement ←─────────────────┘
