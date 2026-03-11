package iface

import (
	"fmt"

	"github.com/adnilis/x-hmsw/indexes/flat"
	"github.com/adnilis/x-hmsw/indexes/hnsw"
	"github.com/adnilis/x-hmsw/indexes/ivf"
	"github.com/adnilis/x-hmsw/types"
)

// Index 统一的索引接口
type Index interface {
	// Insert 插入向量到索引
	Insert(id int, vector []float32) error

	// Search 搜索最近邻
	Search(query []float32, k int) []types.IndexSearchResult

	// Delete 从索引中删除向量
	Delete(id int) error

	// Count 返回索引中的向量数量
	Count() int
}

// IndexConfig 索引配置
type IndexConfig struct {
	Dimension    int
	MaxVectors   int
	DistanceFunc func(a, b []float32) float32
	// HNSW 参数
	M              int
	EfConstruction int
	// IVF 参数
	NumClusters int
	Nprobe      int
}

// NewIndex 创建索引
func NewIndex(indexType IndexType, config IndexConfig) (Index, error) {
	switch indexType {
	case HNSW, "":
		return NewHNSWIndex(config), nil
	case IVF:
		return NewIVFIndex(config), nil
	case Flat:
		return NewFlatIndex(config), nil
	default:
		return nil, fmt.Errorf("unsupported index type: %s", indexType)
	}
}

// HNSWIndexAdapter HNSW索引适配器
type HNSWIndexAdapter struct {
	graph *hnsw.HNSWGraph
}

// NewHNSWIndex 创建HNSW索引
func NewHNSWIndex(config IndexConfig) Index {
	graph := hnsw.NewHNSW(
		config.Dimension,
		config.M,
		config.EfConstruction,
		config.MaxVectors,
		config.DistanceFunc,
	)
	return &HNSWIndexAdapter{graph: graph}
}

func (a *HNSWIndexAdapter) Insert(id int, vector []float32) error {
	a.graph.Insert(id, vector)
	return nil
}

func (a *HNSWIndexAdapter) Search(query []float32, k int) []types.IndexSearchResult {
	hnswResults := a.graph.Search(query, k)
	results := make([]types.IndexSearchResult, len(hnswResults))
	for i, r := range hnswResults {
		results[i] = types.IndexSearchResult{
			Node: &types.IndexNode{
				ID:      r.Node.ID,
				Vector:  r.Node.Vector,
				Deleted: r.Node.Deleted,
			},
			Distance: r.Distance,
		}
	}
	return results
}

func (a *HNSWIndexAdapter) Delete(id int) error {
	return a.graph.Delete(id)
}

func (a *HNSWIndexAdapter) Count() int {
	return a.graph.Count()
}

// IVFIndexAdapter IVF索引适配器
type IVFIndexAdapter struct {
	index          *ivf.IVFIndex
	nprobe         int
	trained        bool
	trainThreshold int // 达到这个向量数量时自动训练
}

// NewIVFIndex 创建IVF索引
func NewIVFIndex(config IndexConfig) Index {
	index := ivf.NewIVFIndex(config.Dimension, config.NumClusters, config.DistanceFunc)
	nprobe := config.Nprobe
	if nprobe <= 0 {
		nprobe = 10 // 默认值
	}
	return &IVFIndexAdapter{
		index:          index,
		nprobe:         nprobe,
		trained:        false,
		trainThreshold: config.NumClusters * 10, // 默认为聚类数的10倍
	}
}

func (a *IVFIndexAdapter) Insert(id int, vector []float32) error {
	a.index.Add(id, vector)

	// 检查是否需要训练
	if !a.trained && a.index.Count() >= a.trainThreshold {
		a.train()
	}

	return nil
}

// train 训练IVF索引
func (a *IVFIndexAdapter) train() {
	// 获取所有向量
	vectors := a.index.GetAllVectors()
	if len(vectors) == 0 {
		return
	}

	// 训练聚类
	err := a.index.Train(vectors, 20) // 20次迭代
	if err == nil {
		a.trained = true
	}
}

func (a *IVFIndexAdapter) Search(query []float32, k int) []types.IndexSearchResult {
	// 如果还没有训练，先训练
	if !a.trained {
		a.train()
	}

	// IVF搜索返回ids和distances，需要转换为SearchResult
	ids, distances, err := a.index.Search(query, k, a.nprobe)
	if err != nil {
		return []types.IndexSearchResult{}
	}

	results := make([]types.IndexSearchResult, len(ids))
	for i := range ids {
		results[i] = types.IndexSearchResult{
			Node: &types.IndexNode{
				ID: ids[i],
			},
			Distance: distances[i],
		}
	}
	return results
}

func (a *IVFIndexAdapter) Delete(id int) error {
	// IVF索引不支持删除，返回nil
	return nil
}

func (a *IVFIndexAdapter) Count() int {
	return a.index.Count()
}

// FlatIndexAdapter Flat索引适配器
type FlatIndexAdapter struct {
	index *flat.FlatIndex
}

// NewFlatIndex 创建Flat索引
func NewFlatIndex(config IndexConfig) Index {
	index := flat.NewFlatIndex(config.DistanceFunc)
	return &FlatIndexAdapter{index: index}
}

func (a *FlatIndexAdapter) Insert(id int, vector []float32) error {
	a.index.Add(id, vector)
	return nil
}

func (a *FlatIndexAdapter) Search(query []float32, k int) []types.IndexSearchResult {
	ids, distances, err := a.index.Search(query, k)
	if err != nil {
		return []types.IndexSearchResult{}
	}

	results := make([]types.IndexSearchResult, len(ids))
	for i := range ids {
		results[i] = types.IndexSearchResult{
			Node: &types.IndexNode{
				ID: ids[i],
			},
			Distance: distances[i],
		}
	}
	return results
}

func (a *FlatIndexAdapter) Delete(id int) error {
	a.index.Delete(id)
	return nil
}

func (a *FlatIndexAdapter) Count() int {
	return a.index.Count()
}
