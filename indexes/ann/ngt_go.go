package ann

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// NGTIndex NGT 索引实现
type NGTIndex struct {
	dimension    int
	edgeSize     int
	searchEdges  int
	distanceFunc func(a, b []float32) float32
	vectors      [][]float32
	ids          []int
	graph        map[int]*NGTNode
	mu           sync.RWMutex
}

// NGTNode NGT 节点
type NGTNode struct {
	ID        int
	Vector    []float32
	Neighbors map[int]float32 // neighbor_id -> distance
}

// NewNGTIndex 创建 NGT 索引
func NewNGTIndex(dimension, edgeSize, searchEdges int, distanceFunc func(a, b []float32) float32) *NGTIndex {
	return &NGTIndex{
		dimension:    dimension,
		edgeSize:     edgeSize,
		searchEdges:  searchEdges,
		distanceFunc: distanceFunc,
		vectors:      make([][]float32, 0),
		ids:          make([]int, 0),
		graph:        make(map[int]*NGTNode),
	}
}

// Add 添加向量
func (idx *NGTIndex) Add(id int, vector []float32) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// 添加向量
	idx.vectors = append(idx.vectors, vector)
	idx.ids = append(idx.ids, id)

	// 创建节点
	node := &NGTNode{
		ID:        id,
		Vector:    vector,
		Neighbors: make(map[int]float32),
	}

	// 查找最近邻
	if len(idx.vectors) > 1 {
		neighbors := idx.searchKNN(vector, idx.edgeSize)
		for _, n := range neighbors {
			node.Neighbors[n.ID] = n.Distance
		}
	}

	idx.graph[id] = node

	// 更新其他节点的邻居
	if len(idx.vectors) > 1 {
		idx.updateNeighbors(id, vector)
	}

	return nil
}

// updateNeighbors 更新邻居
func (idx *NGTIndex) updateNeighbors(newID int, newVector []float32) {
	for id, node := range idx.graph {
		if id == newID {
			continue
		}

		dist := idx.distanceFunc(newVector, node.Vector)
		if len(node.Neighbors) < idx.edgeSize {
			node.Neighbors[newID] = dist
		} else {
			// 检查是否应该替换
			maxDist := float32(0)
			maxID := -1
			for nid, ndist := range node.Neighbors {
				if ndist > maxDist {
					maxDist = ndist
					maxID = nid
				}
			}

			if maxID != -1 && dist < maxDist {
				delete(node.Neighbors, maxID)
				node.Neighbors[newID] = dist
			}
		}
	}
}

// searchKNN 搜索 K 近邻
func (idx *NGTIndex) searchKNN(query []float32, k int) []Neighbor {
	distances := make([]Neighbor, 0)

	for id, node := range idx.graph {
		dist := idx.distanceFunc(query, node.Vector)
		distances = append(distances, Neighbor{
			ID:       id,
			Distance: dist,
		})
	}

	// 排序
	sort.Slice(distances, func(i, j int) bool {
		return distances[i].Distance < distances[j].Distance
	})

	if k > len(distances) {
		k = len(distances)
	}

	return distances[:k]
}

// Neighbor 邻居信息
type Neighbor struct {
	ID       int
	Distance float32
}

// Search 搜索
func (idx *NGTIndex) Search(query []float32, k int) ([]int, []float32, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	neighbors := idx.searchKNN(query, k)

	ids := make([]int, len(neighbors))
	dists := make([]float32, len(neighbors))

	for i, n := range neighbors {
		ids[i] = n.ID
		dists[i] = n.Distance
	}

	return ids, dists, nil
}

// NGTSnapshot 用于序列化的快照结构体
type NGTSnapshot struct {
	Dimension   int               `json:"dimension"`
	EdgeSize    int               `json:"edge_size"`
	SearchEdges int               `json:"search_edges"`
	Vectors     [][]float32       `json:"vectors"`
	IDs         []int             `json:"ids"`
	Graph       []NGTNodeSnapshot `json:"graph"`
}

// NGTNodeSnapshot NGT 节点快照
type NGTNodeSnapshot struct {
	ID        int             `json:"id"`
	Vector    []float32       `json:"vector"`
	Neighbors map[int]float32 `json:"neighbors"`
}

// Save 保存索引
func (idx *NGTIndex) Save(path string) error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// 创建快照
	snapshot := NGTSnapshot{
		Dimension:   idx.dimension,
		EdgeSize:    idx.edgeSize,
		SearchEdges: idx.searchEdges,
		Vectors:     idx.vectors,
		IDs:         idx.ids,
		Graph:       make([]NGTNodeSnapshot, len(idx.graph)),
	}

	// 序列化图
	i := 0
	for _, node := range idx.graph {
		snapshot.Graph[i] = NGTNodeSnapshot{
			ID:        node.ID,
			Vector:    node.Vector,
			Neighbors: node.Neighbors,
		}
		i++
	}

	// JSON 序列化
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal NGT index: %w", err)
	}

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write NGT index: %w", err)
	}

	return nil
}

// Load 加载索引
func (idx *NGTIndex) Load(path string) error {
	// 读取文件
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read NGT index: %w", err)
	}

	// JSON 反序列化
	var snapshot NGTSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("failed to unmarshal NGT index: %w", err)
	}

	// 重建索引
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.dimension = snapshot.Dimension
	idx.edgeSize = snapshot.EdgeSize
	idx.searchEdges = snapshot.SearchEdges
	idx.vectors = snapshot.Vectors
	idx.ids = snapshot.IDs
	idx.graph = make(map[int]*NGTNode)

	// 重建图
	for _, nodeSnap := range snapshot.Graph {
		idx.graph[nodeSnap.ID] = &NGTNode{
			ID:        nodeSnap.ID,
			Vector:    nodeSnap.Vector,
			Neighbors: nodeSnap.Neighbors,
		}
	}

	return nil
}
