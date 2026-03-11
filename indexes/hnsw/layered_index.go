package hnsw

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// LayeredIndex 分层索引管理器
// 将索引分为两层：粗粒度索引（磁盘）和细粒度索引（内存）
type LayeredIndex struct {
	coarseIndex *CoarseIndex // 粗粒度索引（磁盘，按需加载）
	fineIndex   *FineIndex   // 细粒度索引（内存，热数据）
	config      LayeredConfig
	mu          sync.RWMutex
	stats       LayeredStats
}

// LayeredConfig 分层索引配置
type LayeredConfig struct {
	// 粗粒度索引配置
	CoarsePageSize    int    // 每页向量数量（默认 10000）
	CoarseMaxPages    int    // 最大页数
	CoarseUsePQ       bool   // 是否使用 PQ 量化
	CoarsePQSubspaces int    // PQ 子空间数量
	CoarsePQCentroids int    // PQ 每个子空间的质心数
	CoarseStoragePath string // 粗粒度索引存储路径

	// 细粒度索引配置
	FineMaxSize          int // 细粒度索引最大容量（向量数量）
	FineLRUSize          int // LRU 缓存大小
	FinePromoteThreshold int // 热数据提升阈值（访问次数）
	FineEvictThreshold   int // 冷数据驱逐阈值（未访问时间，秒）

	// 搜索配置
	SearchCoarseK int // 粗粒度搜索返回的候选数量
	SearchFineK   int // 最终返回的结果数量
}

// LayeredStats 分层索引统计信息
type LayeredStats struct {
	TotalInserts   int64
	TotalSearches  int64
	CoarseSearches int64
	FineSearches   int64
	Promotions     int64
	Evictions      int64
	CacheHitRate   float64
	LastUpdated    time.Time
}

// DefaultLayeredConfig 默认配置
func DefaultLayeredConfig() LayeredConfig {
	return LayeredConfig{
		CoarsePageSize:    10000,
		CoarseMaxPages:    100,
		CoarseUsePQ:       true,
		CoarsePQSubspaces: 8,
		CoarsePQCentroids: 256,
		CoarseStoragePath: "./data/coarse_index",

		FineMaxSize:          10000,
		FineLRUSize:          5000,
		FinePromoteThreshold: 3,
		FineEvictThreshold:   3600, // 1小时

		SearchCoarseK: 100,
		SearchFineK:   10,
	}
}

// NewLayeredIndex 创建新的分层索引
func NewLayeredIndex(config LayeredConfig) (*LayeredIndex, error) {
	// 创建粗粒度索引
	coarseIndex, err := NewCoarseIndex(CoarseConfig{
		PageSize:    config.CoarsePageSize,
		MaxPages:    config.CoarseMaxPages,
		UsePQ:       config.CoarseUsePQ,
		PQSubspaces: config.CoarsePQSubspaces,
		PQCentroids: config.CoarsePQCentroids,
		StoragePath: config.CoarseStoragePath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create coarse index: %w", err)
	}

	// 创建细粒度索引
	fineIndex := NewFineIndex(FineConfig{
		MaxSize: config.FineMaxSize,
		LRUSize: config.FineLRUSize,
	})

	return &LayeredIndex{
		coarseIndex: coarseIndex,
		fineIndex:   fineIndex,
		config:      config,
		stats: LayeredStats{
			LastUpdated: time.Now(),
		},
	}, nil
}

// Insert 插入向量到分层索引
func (li *LayeredIndex) Insert(id int, vector []float32) error {
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
func (li *LayeredIndex) Search(query []float32, k int) ([]SearchResult, error) {
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

	// 4. 合并结果，去重
	merged := li.mergeResults(fineResults, coarseResults)

	// 5. 更新访问计数，用于热数据提升
	for i := 0; i < min(len(merged), k); i++ {
		li.fineIndex.RecordAccess(merged[i].Node.ID)
	}

	// 6. 提升热数据到细粒度索引
	li.promoteHotData(merged)

	// 7. 驱逐冷数据
	li.evictColdData()

	// 8. 更新缓存命中率
	li.updateCacheHitRate(len(fineResults), k)

	// 9. 返回前 k 个结果
	if len(merged) > k {
		merged = merged[:k]
	}

	return merged, nil
}

// mergeResults 合并搜索结果，去重
func (li *LayeredIndex) mergeResults(fine, coarse []SearchResult) []SearchResult {
	seen := make(map[int]bool)
	merged := make([]SearchResult, 0, len(fine)+len(coarse))

	// 先添加细粒度索引结果
	for _, result := range fine {
		if !seen[result.Node.ID] {
			seen[result.Node.ID] = true
			merged = append(merged, result)
		}
	}

	// 再添加粗粒度索引结果
	for _, result := range coarse {
		if !seen[result.Node.ID] {
			seen[result.Node.ID] = true
			merged = append(merged, result)
		}
	}

	// 按距离排序
	sortResults(merged)

	return merged
}

// promoteHotData 提升热数据到细粒度索引
func (li *LayeredIndex) promoteHotData(results []SearchResult) {
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

// evictColdData 驱逐冷数据
func (li *LayeredIndex) evictColdData() {
	// 获取需要驱逐的节点
	evicted := li.fineIndex.EvictCold(int64(li.config.FineEvictThreshold))
	for range evicted {
		li.stats.Evictions++
	}
}

// updateCacheHitRate 更新缓存命中率
func (li *LayeredIndex) updateCacheHitRate(hits, total int) {
	if total > 0 {
		// 使用指数移动平均平滑命中率
		currentRate := float64(hits) / float64(total)
		if li.stats.CacheHitRate == 0 {
			li.stats.CacheHitRate = currentRate
		} else {
			li.stats.CacheHitRate = 0.9*li.stats.CacheHitRate + 0.1*currentRate
		}
	}
}

// PromoteToHot 手动提升向量到热数据
func (li *LayeredIndex) PromoteToHot(id int) error {
	li.mu.Lock()
	defer li.mu.Unlock()

	// 检查是否已经在细粒度索引中
	if li.fineIndex.Contains(id) {
		return nil
	}

	// 从粗粒度索引获取向量
	vector, err := li.coarseIndex.GetVector(id)
	if err != nil {
		return fmt.Errorf("failed to get vector from coarse index: %w", err)
	}

	// 插入到细粒度索引
	if li.fineIndex.Size() >= li.config.FineMaxSize {
		// 驱逐最冷的数据
		evicted := li.fineIndex.EvictOne()
		if evicted != -1 {
			li.stats.Evictions++
		}
	}

	if err := li.fineIndex.Insert(id, vector); err != nil {
		return fmt.Errorf("failed to insert into fine index: %w", err)
	}

	li.stats.Promotions++
	return nil
}

// EvictFromHot 手动驱逐向量从热数据
func (li *LayeredIndex) EvictFromHot(id int) error {
	li.mu.Lock()
	defer li.mu.Unlock()

	if err := li.fineIndex.Delete(id); err != nil {
		return fmt.Errorf("failed to delete from fine index: %w", err)
	}

	li.stats.Evictions++
	return nil
}

// GetStats 获取统计信息
func (li *LayeredIndex) GetStats() LayeredStats {
	li.mu.RLock()
	defer li.mu.RUnlock()

	// 更新缓存命中率
	fineStats := li.fineIndex.GetStats()
	li.stats.CacheHitRate = fineStats.HitRate

	return li.stats
}

// Save 保存分层索引
func (li *LayeredIndex) Save() error {
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

// Load 加载分层索引
func (li *LayeredIndex) Load() error {
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
func (li *LayeredIndex) Close() error {
	// 先保存索引（Save()会获取锁）
	if err := li.Save(); err != nil {
		return err
	}

	li.mu.Lock()
	defer li.mu.Unlock()

	// 关闭粗粒度索引
	if err := li.coarseIndex.Close(); err != nil {
		return fmt.Errorf("failed to close coarse index: %w", err)
	}

	// 关闭细粒度索引
	if err := li.fineIndex.Close(); err != nil {
		return fmt.Errorf("failed to close fine index: %w", err)
	}

	return nil
}

// sortResults 按距离排序搜索结果
func sortResults(results []SearchResult) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})
}
