package mcp

import (
	"sync"
	"time"

	"github.com/yourusername/useq-ai-assistant/models"
)

// MCPContextCache manages cached MCP contexts with TTL
type MCPContextCache struct {
	cache    map[string]*CachedContext
	ttl      time.Duration
	mu       sync.RWMutex
}

// CachedContext represents a cached MCP context with metadata
type CachedContext struct {
	Context   *models.MCPContext
	CreatedAt time.Time
	FileCount int
	LastHash  string
}

// NewMCPContextCache creates a new context cache
func NewMCPContextCache(ttl time.Duration) *MCPContextCache {
	return &MCPContextCache{
		cache: make(map[string]*CachedContext),
		ttl:   ttl,
	}
}

// Get retrieves cached context if valid
func (c *MCPContextCache) Get(projectPath string) (*models.MCPContext, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	cached, exists := c.cache[projectPath]
	if !exists {
		return nil, false
	}
	
	// Check TTL
	if time.Since(cached.CreatedAt) > c.ttl {
		return nil, false
	}
	
	return cached.Context, true
}

// Set stores context in cache
func (c *MCPContextCache) Set(projectPath string, context *models.MCPContext, fileCount int, hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.cache[projectPath] = &CachedContext{
		Context:   context,
		CreatedAt: time.Now(),
		FileCount: fileCount,
		LastHash:  hash,
	}
}

// Invalidate removes cached context
func (c *MCPContextCache) Invalidate(projectPath string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.cache, projectPath)
}

// GetStats returns cache statistics
func (c *MCPContextCache) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	valid := 0
	expired := 0
	
	for _, cached := range c.cache {
		if time.Since(cached.CreatedAt) <= c.ttl {
			valid++
		} else {
			expired++
		}
	}
	
	return map[string]interface{}{
		"total_entries": len(c.cache),
		"valid_entries": valid,
		"expired_entries": expired,
		"ttl_seconds": c.ttl.Seconds(),
	}
}
