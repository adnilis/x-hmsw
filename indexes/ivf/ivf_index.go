package ivf

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Cluster 聚类
type Cluster struct {
	Centroid []float32
	Vectors  []int
}

// IVFSnapshot 用于序列化的 IVF 快照
type IVFSnapshot struct {
	Dimension   int               `json:"dimension"`
	NumClusters int               `json:"num_clusters"`
	Clusters    []ClusterSnapshot `json:"clusters"`
	Vectors     [][]float32       `json:"vectors"`
	IDs         []int             `json:"ids"`
}

// ClusterSnapshot 聚类快照
type ClusterSnapshot struct {
	Centroid []float32 `json:"centroid"`
	Vectors  []int     `json:"vectors"`
}

// IVFIndex IVF 索引结构
type IVFIndex struct {
	dimension    int
	numClusters  int
	clusters     []Cluster
	vectors      [][]float32
	ids          []int
	distanceFunc func(a, b []float32) float32
	mu           sync.RWMutex
	rng          *rand.Rand
}

// NewIVFIndex 创建 IVF 索引
func NewIVFIndex(dimension, numClusters int, distanceFunc func(a, b []float32) float32) *IVFIndex {
	return &IVFIndex{
		dimension:    dimension,
		numClusters:  numClusters,
		clusters:     make([]Cluster, 0),
		vectors:      make([][]float32, 0),
		ids:          make([]int, 0),
		distanceFunc: distanceFunc,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Add 添加向量
func (idx *IVFIndex) Add(id int, vector []float32) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.vectors = append(idx.vectors, vector)
	idx.ids = append(idx.ids, id)
}

// GetAllVectors 获取所有向量（用于训练）
func (idx *IVFIndex) GetAllVectors() [][]float32 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	vectors := make([][]float32, len(idx.vectors))
	copy(vectors, idx.vectors)
	return vectors
}

// Train 训练索引（K-means 聚类）
func (idx *IVFIndex) Train(vectors [][]float32, maxIterations int) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if len(vectors) < idx.numClusters {
		return fmt.Errorf("not enough vectors for clustering")
	}

	// 保存向量
	idx.vectors = make([][]float32, len(vectors))
	copy(idx.vectors, vectors)
	idx.ids = make([]int, len(vectors))
	for i := range idx.ids {
		idx.ids[i] = i
	}

	// 随机初始化质心
	idx.clusters = make([]Cluster, idx.numClusters)
	for i := 0; i < idx.numClusters; i++ {
		idx.clusters[i].Centroid = make([]float32, idx.dimension)
		copy(idx.clusters[i].Centroid, vectors[i])
	}

	// K-means 迭代
	for iter := 0; iter < maxIterations; iter++ {
		changed := false

		// 清空聚类
		for i := range idx.clusters {
			idx.clusters[i].Vectors = make([]int, 0)
		}

		// 分配向量到最近的质心
		for i, vec := range vectors {
			bestCluster := -1
			bestDist := float32(math.MaxFloat32)

			for j, cluster := range idx.clusters {
				dist := idx.distanceFunc(vec, cluster.Centroid)
				if dist < bestDist {
					bestDist = dist
					bestCluster = j
				}
			}

			if bestCluster >= 0 {
				idx.clusters[bestCluster].Vectors = append(idx.clusters[bestCluster].Vectors, i)
			}
		}

		// 更新质心
		for i, cluster := range idx.clusters {
			if len(cluster.Vectors) == 0 {
				continue
			}

			newCentroid := make([]float32, idx.dimension)
			for _, vecIdx := range cluster.Vectors {
				vec := vectors[vecIdx]
				for j := range newCentroid {
					newCentroid[j] += vec[j]
				}
			}

			for j := range newCentroid {
				newCentroid[j] /= float32(len(cluster.Vectors))
			}

			// 检查是否收敛
			dist := idx.distanceFunc(cluster.Centroid, newCentroid)
			if dist > 1e-6 {
				changed = true
			}

			idx.clusters[i].Centroid = newCentroid
		}

		if !changed {
			break
		}
	}

	return nil
}

// Search 搜索最近邻
func (idx *IVFIndex) Search(query []float32, k int, nprobe int) ([]int, []float32, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if len(idx.clusters) == 0 {
		return nil, nil, fmt.Errorf("index not trained")
	}

	// 找到最近的 nprobe 个聚类
	type clusterDist struct {
		idx  int
		dist float32
	}

	clusterDists := make([]clusterDist, len(idx.clusters))
	for i, cluster := range idx.clusters {
		dist := idx.distanceFunc(query, cluster.Centroid)
		clusterDists[i] = clusterDist{i, dist}
	}

	sort.Slice(clusterDists, func(i, j int) bool {
		return clusterDists[i].dist < clusterDists[j].dist
	})

	// 在选中的聚类中搜索
	probeClusters := nprobe
	if probeClusters > len(idx.clusters) {
		probeClusters = len(idx.clusters)
	}

	// 收集候选向量
	candidates := make([]struct {
		id   int
		dist float32
	}, 0)

	for i := 0; i < probeClusters; i++ {
		clusterIdx := clusterDists[i].idx
		cluster := idx.clusters[clusterIdx]

		for _, vecIdx := range cluster.Vectors {
			if vecIdx < len(idx.vectors) {
				dist := idx.distanceFunc(query, idx.vectors[vecIdx])
				candidates = append(candidates, struct {
					id   int
					dist float32
				}{
					id:   idx.ids[vecIdx],
					dist: dist,
				})
			}
		}
	}

	// 排序并返回前 k 个
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].dist < candidates[j].dist
	})

	resultSize := k
	if resultSize > len(candidates) {
		resultSize = len(candidates)
	}

	ids := make([]int, resultSize)
	dists := make([]float32, resultSize)

	for i := 0; i < resultSize; i++ {
		ids[i] = candidates[i].id
		dists[i] = candidates[i].dist
	}

	return ids, dists, nil
}

// Count 返回向量数量
func (idx *IVFIndex) Count() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.vectors)
}

// Save 保存索引
func (idx *IVFIndex) Save(path string) error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// 创建目录
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建快照
	snapshot := IVFSnapshot{
		Dimension:   idx.dimension,
		NumClusters: idx.numClusters,
		Clusters:    make([]ClusterSnapshot, len(idx.clusters)),
		Vectors:     idx.vectors,
		IDs:         idx.ids,
	}

	// 序列化聚类
	for i, cluster := range idx.clusters {
		snapshot.Clusters[i] = ClusterSnapshot{
			Centroid: cluster.Centroid,
			Vectors:  cluster.Vectors,
		}
	}

	// 序列化
	jsonData, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal IVF snapshot: %w", err)
	}

	// 写入文件
	savePath := filepath.Join(path, "ivf_index.json")
	if err := ioutil.WriteFile(savePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Load 加载索引
func (idx *IVFIndex) Load(path string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// 读取文件
	loadPath := filepath.Join(path, "ivf_index.json")
	jsonData, err := ioutil.ReadFile(loadPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", loadPath)
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	// 反序列化
	var snapshot IVFSnapshot
	if err := json.Unmarshal(jsonData, &snapshot); err != nil {
		return fmt.Errorf("failed to unmarshal IVF snapshot: %w", err)
	}

	// 重建索引
	idx.dimension = snapshot.Dimension
	idx.numClusters = snapshot.NumClusters
	idx.vectors = snapshot.Vectors
	idx.ids = snapshot.IDs

	// 重建聚类
	idx.clusters = make([]Cluster, len(snapshot.Clusters))
	for i, clusterSnap := range snapshot.Clusters {
		idx.clusters[i] = Cluster{
			Centroid: clusterSnap.Centroid,
			Vectors:  clusterSnap.Vectors,
		}
	}

	return nil
}
