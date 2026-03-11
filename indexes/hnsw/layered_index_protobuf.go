package hnsw

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// LayeredIndexV2 分层索引管理器 V2（使用 protobuf 序列化）
// 将索引分为两层：粗粒度索引（磁盘）和细粒度索引（内存）
type LayeredIndexV2 struct {
	coarseIndex *CoarseIndexV2 // 粗粒度索引（磁盘，按需加载，protobuf 序列化）
	fineIndex   *FineIndex     // 细粒度索引（内存，热数据）
	config      LayeredConfig
	mu          sync.RWMutex
	stats       LayeredStats
}

// NewLayeredIndexV2 创建新的分层索引 V2
func NewLayeredIndexV2(config LayeredConfig, enableDelta bool) (*LayeredIndexV2, error) {
	// 创建粗粒度索引 V2（protobuf 序列化）
	coarseIndex, err := NewCoarseIndexV2(CoarseConfig{
		PageSize:    config.CoarsePageSize,
		MaxPages:    config.CoarseMaxPages,
		UsePQ:       config.CoarseUsePQ,
		PQSubspaces: config.CoarsePQSubspaces,
		PQCentroids: config.CoarsePQCentroids,
		StoragePath: config.CoarseStoragePath,
	}, enableDelta)
	if err != nil {
		return nil, fmt.Errorf("failed to create coarse index: %w", err)
	}

	// 创建细粒度索引
	fineIndex := NewFineIndex(FineConfig{
		MaxSize: config.FineMaxSize,
		LRUSize: config.FineLRUSize,
	})

	return &LayeredIndexV2{
		coarseIndex: coarseIndex,
		fineIndex:   fineIndex,
		config:      config,
		stats: LayeredStats{
			LastUpdated: time.Now(),
		},
	}, nil
}

// Insert 插入向量到分层索引
func (li *LayeredIndexV2) Insert(id int, vector []float32) error {
	li.mu.Lock()
	defer li.mu.Unlock()

	// 只插入到粗粒度索引（磁盘）
	// 热数据会通过搜索时自动提升到细粒度索引
	if err := li.coarseIndex.Insert(id, vector); err != nil {
		return fmt.Errorf("failed to insert into coarse index: %w", err)
	}

	li.stats.TotalInserts++
	li.stats.LastUpdated = time.Now()

	return nil
}

// Search 在分层索引中搜索
func (li *LayeredIndexV2) Search(query []float32, k int) ([]SearchResult, error) {
	li.mu.Lock()
	defer li.mu.Unlock()

	li.stats.TotalSearches++
	li.stats.LastUpdated = time.Now()

	// 1. 先在细粒度索引中搜索（热数据）
	fineResults, err := li.fineIndex.Search(query, k)
	if err != nil {
		return nil, fmt.Errorf("failed to search in fine index: %w", err)
	}
	li.stats.FineSearches++

	// 2. 如果细粒度索引结果足够，直接返回
	if len(fineResults) >= k {
		return fineResults, nil
	}

	// 3. 在粗粒度索引中搜索（冷数据）
	coarseK := li.config.SearchCoarseK
	if coarseK < k {
		coarseK = k
	}
	coarseResults, err := li.coarseIndex.Search(query, coarseK)
	if err != nil {
		return nil, fmt.Errorf("failed to search in coarse index: %w", err)
	}
	li.stats.CoarseSearches++

	// 4. 合并结果并去重
	mergedResults := mergeResults(fineResults, coarseResults)

	// 5. 提升热数据到细粒度索引
	li.promoteHotData(mergedResults)

	// 6. 返回前 k 个结果
	if len(mergedResults) > k {
		mergedResults = mergedResults[:k]
	}

	return mergedResults, nil
}

// promoteHotData 提升热数据到细粒度索引
func (li *LayeredIndexV2) promoteHotData(results []SearchResult) {
	for _, result := range results {
		accessCount := li.fineIndex.GetAccessCount(result.Node.ID)
		if accessCount >= li.config.FinePromoteThreshold {
			// 检查是否已经在细粒度索引中
			if !li.fineIndex.Contains(result.Node.ID) {
				// 从粗粒度索引获取向量
				vector, err := li.coarseIndex.GetVector(result.Node.ID)
				if err != nil {
					continue
				}

				// 插入到细粒度索引
				if li.fineIndex.Size() < li.config.FineMaxSize {
					li.fineIndex.Insert(result.Node.ID, vector)
					li.stats.Promotions++
				}
			}
		}
	}
}

// GetVector 获取向量
func (li *LayeredIndexV2) GetVector(id int) ([]float32, error) {
	li.mu.RLock()
	defer li.mu.RUnlock()

	// 先尝试从细粒度索引获取
	if vector, err := li.fineIndex.GetVector(id); err == nil {
		return vector, nil
	}

	// 从粗粒度索引获取
	return li.coarseIndex.GetVector(id)
}

// Save 保存分层索引
func (li *LayeredIndexV2) Save() error {
	li.mu.Lock()
	defer li.mu.Unlock()

	// 保存粗粒度索引
	if err := li.coarseIndex.Save(); err != nil {
		return fmt.Errorf("failed to save coarse index: %w", err)
	}

	// 保存细粒度索引
	if err := li.fineIndex.Save(filepath.Join(li.config.CoarseStoragePath, "fine_index.json")); err != nil {
		return fmt.Errorf("failed to save fine index: %w", err)
	}

	return nil
}

// SaveDelta 增量保存分层索引
func (li *LayeredIndexV2) SaveDelta() (int, error) {
	li.mu.Lock()
	defer li.mu.Unlock()

	// 增量保存粗粒度索引
	savedCount, err := li.coarseIndex.SaveDelta()
	if err != nil {
		return savedCount, fmt.Errorf("failed to save delta coarse index: %w", err)
	}

	// 保存细粒度索引
	if err := li.fineIndex.Save(filepath.Join(li.config.CoarseStoragePath, "fine_index.json")); err != nil {
		return savedCount, fmt.Errorf("failed to save fine index: %w", err)
	}

	return savedCount, nil
}

// Load 加载分层索引
func (li *LayeredIndexV2) Load() error {
	li.mu.Lock()
	defer li.mu.Unlock()

	// 加载粗粒度索引
	if err := li.coarseIndex.Load(); err != nil {
		return fmt.Errorf("failed to load coarse index: %w", err)
	}

	// 加载细粒度索引
	fineIndexPath := filepath.Join(li.config.CoarseStoragePath, "fine_index.json")
	if _, err := os.Stat(fineIndexPath); err == nil {
		if err := li.fineIndex.Load(fineIndexPath); err != nil {
			return fmt.Errorf("failed to load fine index: %w", err)
		}
	}

	return nil
}

// Close 关闭分层索引
func (li *LayeredIndexV2) Close() error {
	return li.Save()
}

// GetStats 获取统计信息
func (li *LayeredIndexV2) GetStats() LayeredStats {
	li.mu.RLock()
	defer li.mu.RUnlock()
	return li.stats
}

// GetCoarseStats 获取粗粒度索引统计信息
func (li *LayeredIndexV2) GetCoarseStats() CoarseStats {
	return li.coarseIndex.GetStats()
}

// GetFineStats 获取细粒度索引统计信息
func (li *LayeredIndexV2) GetFineStats() FineStats {
	return li.fineIndex.GetStats()
}

// GetDirtyPageCount 获取脏页面数量
func (li *LayeredIndexV2) GetDirtyPageCount() int {
	return li.coarseIndex.GetDirtyPageCount()
}

// mergeResults 合并搜索结果并去重
func mergeResults(fineResults, coarseResults []SearchResult) []SearchResult {
	seen := make(map[int]bool)
	merged := make([]SearchResult, 0, len(fineResults)+len(coarseResults))

	// 添加细粒度索引结果
	for _, result := range fineResults {
		if !seen[result.Node.ID] {
			seen[result.Node.ID] = true
			merged = append(merged, result)
		}
	}

	// 添加粗粒度索引结果
	for _, result := range coarseResults {
		if !seen[result.Node.ID] {
			seen[result.Node.ID] = true
			merged = append(merged, result)
		}
	}

	// 按距离排序
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Distance < merged[j].Distance
	})

	return merged
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
