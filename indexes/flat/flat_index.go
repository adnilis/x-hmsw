package flat

import (
	"sort"
	"sync"
)

// FlatIndex 暴力搜索索引
type FlatIndex struct {
	vectors      [][]float32
	ids          []int
	distanceFunc func(a, b []float32) float32
	mu           sync.RWMutex
}

// NewFlatIndex 创建暴力搜索索引
func NewFlatIndex(distanceFunc func(a, b []float32) float32) *FlatIndex {
	return &FlatIndex{
		vectors:      make([][]float32, 0),
		ids:          make([]int, 0),
		distanceFunc: distanceFunc,
	}
}

// Add 添加向量
func (idx *FlatIndex) Add(id int, vector []float32) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.vectors = append(idx.vectors, vector)
	idx.ids = append(idx.ids, id)
}

// Search 搜索最近邻
func (idx *FlatIndex) Search(query []float32, k int) ([]int, []float32, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	n := len(idx.vectors)
	if n == 0 {
		return nil, nil, nil
	}

	// 计算所有距离
	type result struct {
		id   int
		dist float32
	}

	results := make([]result, n)
	for i, vec := range idx.vectors {
		results[i] = result{
			id:   idx.ids[i],
			dist: idx.distanceFunc(query, vec),
		}
	}

	// 排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].dist < results[j].dist
	})

	// 返回前 k 个
	resultSize := k
	if resultSize > len(results) {
		resultSize = len(results)
	}

	ids := make([]int, resultSize)
	dists := make([]float32, resultSize)

	for i := 0; i < resultSize; i++ {
		ids[i] = results[i].id
		dists[i] = results[i].dist
	}

	return ids, dists, nil
}

// Delete 删除向量
func (idx *FlatIndex) Delete(id int) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	for i, vid := range idx.ids {
		if vid == id {
			idx.vectors = append(idx.vectors[:i], idx.vectors[i+1:]...)
			idx.ids = append(idx.ids[:i], idx.ids[i+1:]...)
			return
		}
	}
}

// Count 返回向量数量
func (idx *FlatIndex) Count() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.vectors)
}
