# MCP (Model Context Protocol) Integration Guide

## Overview

The useQ AI Assistant now includes intelligent MCP integration that automatically determines and executes appropriate system commands based on user queries.

## How It Works

### 1. Query Analysis
When you ask a question like "show me current CPU usage", the system:

1. **Analyzes Intent**: Determines you want system information
2. **Selects Commands**: Chooses appropriate commands (`ps`, `top`, etc.)
3. **Validates Safety**: Ensures commands are safe to execute
4. **Executes Commands**: Runs commands and captures output
5. **Enhances Response**: Uses command results to provide better answers

### 2. Intelligent Command Selection

The system maps query types to appropriate commands:

```
"how many files" → find . -name "*.go" -type f | wc -l
"show CPU usage" → ps -o pid,ppid,%mem,%cpu,comm
"git status"     → git status --porcelain
"project tree"   → tree -I "vendor|node_modules" -L 3
```

### 3. Safety Validation

Commands are categorized by safety level:
- **Safe**: Read-only operations (find, ls, cat, grep)
- **Moderate**: File modifications (touch, mkdir)
- **Dangerous**: System changes (rm, chmod, kill)
- **Restricted**: Blocked commands (sudo, su)

## Integration Points

### 1. Manager Agent (`internal/agents/manager_agent.go`)
- Routes queries through MCP processing
- Enhances query context with command results
- Logs MCP operations for debugging

### 2. Search Agent (`internal/agents/search_agent.go`)
- Uses MCP results to enhance search responses
- Provides system information when requested
- Falls back to vector search when appropriate

### 3. MCP Client (`internal/mcp/mcp_client.go`)
- Coordinates intelligent command execution
- Manages caching and performance optimization
- Handles multiple information sources

## Example Queries and Commands

| User Query | Detected Intent | Commands Executed | Result |
|------------|----------------|-------------------|---------|
| "show me current CPU usage" | system_info | `ps -o %cpu,comm` | Memory/CPU statistics |
| "how many Go files are there" | file_count | `find . -name "*.go" \| wc -l` | File count |
| "show project structure" | filesystem | `tree -L 3` | Directory tree |
| "git status" | git_info | `git status --porcelain` | Repository status |
| "list authentication functions" | code_search | Vector search + file analysis | Function listings |

## Testing MCP Integration

Use the built-in test command:

```bash
useQ> mcp test
```

This will run through various query types and show how MCP commands are selected and executed.

## Configuration

MCP behavior is configured in `config/config.yaml`:

```yaml
mcp:
  enabled: true
  max_execution_time: 10s
  safety_level: safe
  allowed_commands:
    - find
    - ls
    - cat
    - grep
    - git
    - ps
    - du
    - tree
```

## Benefits

1. **Contextual Responses**: Answers include real-time system information
2. **Intelligent Automation**: System determines what commands to run
3. **Safety First**: All commands are validated before execution
4. **Performance**: Results are cached for repeated queries
5. **Learning**: System learns from successful command patterns

## Future Enhancements

- **Machine Learning**: Learn optimal command selection from usage patterns
- **Custom Commands**: Allow users to register custom command mappings
- **Advanced Safety**: More sophisticated safety analysis
- **Parallel Execution**: Run multiple commands concurrently
- **Result Fusion**: Combine multiple command outputs intelligently