package ivf

import (
	"math"
	"math/rand"
	"time"
)

// KMeans K-means 聚类实现
type KMeans struct {
	dimension     int
	numClusters   int
	maxIterations int
	distanceFunc  func(a, b []float32) float32
	rng           *rand.Rand
}

// NewKMeans 创建 K-means 聚类器
func NewKMeans(dimension, numClusters int, distanceFunc func(a, b []float32) float32) *KMeans {
	return &KMeans{
		dimension:     dimension,
		numClusters:   numClusters,
		maxIterations: 100,
		distanceFunc:  distanceFunc,
		rng:           rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Fit 训练聚类模型
func (km *KMeans) Fit(vectors [][]float32) ([]Cluster, error) {
	n := len(vectors)
	if n < km.numClusters {
		return nil, nil
	}

	// 初始化质心（随机选择）
	centroids := make([][]float32, km.numClusters)
	indices := km.rng.Perm(n)
	for i := 0; i < km.numClusters; i++ {
		centroids[i] = make([]float32, km.dimension)
		copy(centroids[i], vectors[indices[i]])
	}

	// 分配向量到聚类
	labels := make([]int, n)
	for iter := 0; iter < km.maxIterations; iter++ {
		changed := false

		// 分配步骤
		for i, vec := range vectors {
			bestCluster := -1
			bestDist := float32(math.MaxFloat32)

			for j, centroid := range centroids {
				dist := km.distanceFunc(vec, centroid)
				if dist < bestDist {
					bestDist = dist
					bestCluster = j
				}
			}

			if bestCluster != labels[i] {
				changed = true
				labels[i] = bestCluster
			}
		}

		// 更新质心
		for j := range centroids {
			count := 0
			for i, label := range labels {
				if label == j {
					for k := range centroids[j] {
						centroids[j][k] += vectors[i][k]
					}
					count++
				}
			}

			if count > 0 {
				for k := range centroids[j] {
					centroids[j][k] /= float32(count)
				}
			}
		}

		if !changed {
			break
		}
	}

	// 构建聚类结果
	clusters := make([]Cluster, km.numClusters)
	for i := range clusters {
		clusters[i].Centroid = centroids[i]
		clusters[i].Vectors = make([]int, 0)
	}

	for i, label := range labels {
		clusters[label].Vectors = append(clusters[label].Vectors, i)
	}

	return clusters, nil
}
