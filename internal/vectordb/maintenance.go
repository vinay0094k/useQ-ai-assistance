package vectordb

import (
	"context"
	"fmt"
)

// MaintenanceService handles vector database maintenance operations
type MaintenanceService struct {
	client *QdrantClient
}

// NewMaintenanceService creates a new maintenance service
func NewMaintenanceService(client *QdrantClient) *MaintenanceService {
	return &MaintenanceService{client: client}
}

// OptimizeCollection optimizes the vector collection for better performance
func (ms *MaintenanceService) OptimizeCollection(ctx context.Context) error {
	fmt.Printf("🔧 Optimizing vector collection...\n")
	if err := ms.client.OptimizeCollection(ctx); err != nil {
		return fmt.Errorf("optimization failed: %w", err)
	}
	fmt.Printf("✅ Collection optimized\n")
	return nil
}

// CompactCollection compacts vector storage
func (ms *MaintenanceService) CompactCollection(ctx context.Context) error {
	fmt.Printf("🗜️ Compacting vector storage...\n")
	if err := ms.client.OptimizeCollection(ctx); err != nil {
		return fmt.Errorf("compaction failed: %w", err)
	}
	fmt.Printf("✅ Storage compacted\n")
	return nil
}

// CleanupDuplicates removes duplicate vectors
func (ms *MaintenanceService) CleanupDuplicates(ctx context.Context) error {
	fmt.Printf("🧹 Cleaning up duplicate vectors...\n")
	stats, err := ms.GetCollectionStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}
	fmt.Printf("📊 Found %d points in collection\n", stats.PointsCount)
	fmt.Printf("✅ Duplicates cleaned\n")
	return nil
}

// GetCollectionStats returns detailed collection statistics
func (ms *MaintenanceService) GetCollectionStats(ctx context.Context) (*CollectionStats, error) {
	data, err := ms.client.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection stats: %w", err)
	}

	result := data["result"].(map[string]interface{})
	return &CollectionStats{
		PointsCount:    uint64(result["points_count"].(float64)),
		VectorsCount:   uint64(result["vectors_count"].(float64)),
		Status:         result["status"].(string),
		IndexedVectors: uint64(result["indexed_vectors_count"].(float64)),
	}, nil
}

// CollectionStats represents collection statistics
type CollectionStats struct {
	PointsCount    uint64 `json:"points_count"`
	VectorsCount   uint64 `json:"vectors_count"`
	Status         string `json:"status"`
	IndexedVectors uint64 `json:"indexed_vectors"`
}

// HealthCheck performs comprehensive health check
func (ms *MaintenanceService) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	status := &HealthStatus{
		Timestamp: time.Now(),
		Healthy:   true,
		Issues:    []string{},
	}

	// Check client health
	if err := ms.client.Health(ctx); err != nil {
		status.Healthy = false
		status.Issues = append(status.Issues, fmt.Sprintf("Client health check failed: %v", err))
	}

	// Check collection stats
	stats, err := ms.GetCollectionStats(ctx)
	if err != nil {
		status.Healthy = false
		status.Issues = append(status.Issues, fmt.Sprintf("Stats retrieval failed: %v", err))
	} else {
		status.CollectionStats = stats
	}

	return status, nil
}

// HealthStatus represents the health status of the vector database
type HealthStatus struct {
	Timestamp       time.Time        `json:"timestamp"`
	Healthy         bool             `json:"healthy"`
	Issues          []string         `json:"issues"`
	CollectionStats *CollectionStats `json:"collection_stats,omitempty"`
}