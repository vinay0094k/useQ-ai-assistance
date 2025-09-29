package indexer

import (
	"context"
	"sync"
)

// GraphBuilder builds knowledge graphs from parsed code
type GraphBuilder struct {
	nodes map[string]*GraphNode
	edges map[string][]*GraphEdge
	mu    sync.RWMutex
}

// NewGraphBuilder creates a new graph builder
func NewGraphBuilder() *GraphBuilder {
	return &GraphBuilder{
		nodes: make(map[string]*GraphNode),
		edges: make(map[string][]*GraphEdge),
	}
}

// BuildFromChunks builds graph from code chunks
func (gb *GraphBuilder) BuildFromChunks(ctx context.Context, chunks []*CodeChunk) error {
	for _, chunk := range chunks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			gb.processChunk(chunk)
		}
	}
	return nil
}

func (gb *GraphBuilder) processChunk(chunk *CodeChunk) {
	gb.mu.Lock()
	defer gb.mu.Unlock()
	
	node := &GraphNode{
		ID:       chunk.ID,
		Type:     string(chunk.Type),
		Name:     chunk.Context.FunctionName,
		FilePath: chunk.FilePath,
		Package:  chunk.Context.PackageName,
		Metadata: chunk.Metadata,
	}
	gb.nodes[node.ID] = node

	for _, dep := range chunk.Context.Dependencies {
		edge := &GraphEdge{
			From:         chunk.ID,
			To:           dep,
			Relationship: "uses",
			Weight:       1.0,
		}
		gb.edges[edge.From] = append(gb.edges[edge.From], edge)
	}
}

// GetRelatedNodes finds nodes related to a given node
func (gb *GraphBuilder) GetRelatedNodes(nodeID string, maxDepth int) []*GraphNode {
	gb.mu.RLock()
	defer gb.mu.RUnlock()

	visited := make(map[string]bool)
	var result []*GraphNode
	gb.traverseGraph(nodeID, maxDepth, 0, visited, &result)
	return result
}

func (gb *GraphBuilder) traverseGraph(nodeID string, maxDepth, currentDepth int, visited map[string]bool, result *[]*GraphNode) {
	if currentDepth >= maxDepth || visited[nodeID] {
		return
	}
	
	visited[nodeID] = true
	if node, exists := gb.nodes[nodeID]; exists {
		*result = append(*result, node)
	}
	
	for _, edge := range gb.edges[nodeID] {
		gb.traverseGraph(edge.To, maxDepth, currentDepth+1, visited, result)
	}
}
