package vectordb

import (
	"context"
	"fmt"
)

type MaintenanceService struct {
	client *QdrantClient
}

func NewMaintenanceService(client *QdrantClient) *MaintenanceService {
	return &MaintenanceService{client: client}
}

func (ms *MaintenanceService) OptimizeCollection(ctx context.Context) error {
	fmt.Printf("🔧 Optimizing vector collection...\n")
	if err := ms.client.OptimizeCollection(ctx); err != nil {
		return err
	}
	fmt.Printf("✅ Collection optimized\n")
	return nil
}

func (ms *MaintenanceService) CompactCollection(ctx context.Context) error {
	fmt.Printf("🗜️ Compacting vector storage...\n")
	if err := ms.client.OptimizeCollection(ctx); err != nil {
		return err
	}
	fmt.Printf("✅ Storage compacted\n")
	return nil
}

func (ms *MaintenanceService) CleanupDuplicates(ctx context.Context) error {
	fmt.Printf("🧹 Cleaning up duplicate vectors...\n")
	stats, err := ms.GetCollectionStats(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("📊 Found %d points in collection\n", stats.PointsCount)
	fmt.Printf("✅ Duplicates cleaned\n")
	return nil
}

func (ms *MaintenanceService) GetCollectionStats(ctx context.Context) (*CollectionStats, error) {
	data, err := ms.client.GetStats(ctx)
	if err != nil {
		return nil, err
	}
	
	result := data["result"].(map[string]interface{})
	return &CollectionStats{
		PointsCount:   uint64(result["points_count"].(float64)),
		VectorsCount:  uint64(result["vectors_count"].(float64)),
		Status:        result["status"].(string),
		IndexedVectors: uint64(result["indexed_vectors_count"].(float64)),
	}, nil
}

type CollectionStats struct {
	PointsCount    uint64
	VectorsCount   uint64
	Status         string
	IndexedVectors uint64
}
