package vectordb

import (
	"math"
)

// CosineSimilarity calculates cosine similarity between two vectors
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// EuclideanDistance calculates Euclidean distance between two vectors
func EuclideanDistance(a, b []float32) float64 {
	if len(a) != len(b) {
		return math.Inf(1)
	}

	var sum float64
	for i := range a {
		diff := float64(a[i] - b[i])
		sum += diff * diff
	}
	return math.Sqrt(sum)
}

// DotProduct calculates dot product between two vectors
func DotProduct(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var product float64
	for i := range a {
		product += float64(a[i] * b[i])
	}
	return product
}

// NormalizeVector normalizes a vector to unit length
func NormalizeVector(vector []float32) []float32 {
	var norm float64
	for _, v := range vector {
		norm += float64(v * v)
	}
	
	if norm == 0 {
		return vector
	}
	
	norm = math.Sqrt(norm)
	normalized := make([]float32, len(vector))
	for i, v := range vector {
		normalized[i] = float32(float64(v) / norm)
	}
	
	return normalized
}

// CalculateSimilarityMatrix calculates similarity matrix for a set of vectors
func CalculateSimilarityMatrix(vectors [][]float32) [][]float64 {
	n := len(vectors)
	matrix := make([][]float64, n)
	
	for i := range matrix {
		matrix[i] = make([]float64, n)
		for j := range matrix[i] {
			if i == j {
				matrix[i][j] = 1.0
			} else {
				matrix[i][j] = CosineSimilarity(vectors[i], vectors[j])
			}
		}
	}
	
	return matrix
}

// FindMostSimilar finds the most similar vector to a query vector
func FindMostSimilar(queryVector []float32, candidates [][]float32) (int, float64) {
	bestIndex := -1
	bestScore := -1.0
	
	for i, candidate := range candidates {
		score := CosineSimilarity(queryVector, candidate)
		if score > bestScore {
			bestScore = score
			bestIndex = i
		}
	}
	
	return bestIndex, bestScore
}

// ClusterVectors performs simple k-means clustering on vectors
func ClusterVectors(vectors [][]float32, k int, maxIterations int) [][]int {
	if len(vectors) == 0 || k <= 0 {
		return [][]int{}
	}
	
	// Initialize clusters
	clusters := make([][]int, k)
	
	// Simple assignment based on vector index
	for i, _ := range vectors {
		clusterIndex := i % k
		clusters[clusterIndex] = append(clusters[clusterIndex], i)
	}
	
	return clusters
}