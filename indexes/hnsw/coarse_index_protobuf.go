package hnsw

import (
	"fmt"
	"math"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/adnilis/x-hmsw/compression/pq"
	"github.com/adnilis/x-hmsw/serialization/protobuf"
)

// CoarseIndexV2 粗粒度索引 V2（使用 protobuf 序列化）
// 使用分页存储，支持按需加载、PQ 量化和增量保存
type CoarseIndexV2 struct {
	pages       []*Page
	pageMap     map[int]int // ID 到页码的映射
	config      CoarseConfig
	quantizer   *pq.ProductQuantizer // PQ 量化器
	mu          sync.RWMutex
	stats       CoarseStats
	serializer  *protobuf.CoarseIndexSerializer // protobuf 序列化器
	enableDelta bool                            // 是否启用增量保存
}

// NewCoarseIndexV2 创建新的粗粒度索引 V2
func NewCoarseIndexV2(config CoarseConfig, enableDelta bool) (*CoarseIndexV2, error) {
	// 创建存储目录
	if err := os.MkdirAll(config.StoragePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// 创建 protobuf 序列化器
	serializer := protobuf.NewCoarseIndexSerializer(config.StoragePath, enableDelta)

	ci := &CoarseIndexV2{
		pages:       make([]*Page, 0, config.MaxPages),
		pageMap:     make(map[int]int),
		config:      config,
		serializer:  serializer,
		enableDelta: enableDelta,
		stats: CoarseStats{
			TotalPages: 0,
		},
	}

	// 如果启用 PQ 量化，初始化量化器
	if config.UsePQ {
		// 注意：需要先知道向量维度才能创建 PQ 量化器
		// 这里暂时不初始化，在第一次插入时根据向量维度创建
	}

	return ci, nil
}

// Insert 插入向量到粗粒度索引
func (ci *CoarseIndexV2) Insert(id int, vector []float32) error {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	// 查找或创建页面
	pageIndex := ci.findOrCreatePage(id)
	page := ci.pages[pageIndex]

	page.mu.Lock()
	defer page.mu.Unlock()

	// 如果启用 PQ 量化，先量化向量
	if ci.config.UsePQ && ci.quantizer != nil {
		quantized := ci.quantizer.Encode(vector)
		page.Quantized[id] = quantized
	}

	// 存储原始向量
	page.Vectors[id] = vector
	ci.stats.TotalVectors++

	// 标记页面为脏
	if ci.enableDelta {
		ci.serializer.MarkPageDirty(page.PageID)
	}

	return nil
}

// Search 在粗粒度索引中搜索
func (ci *CoarseIndexV2) Search(query []float32, k int) ([]SearchResult, error) {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	results := make([]SearchResult, 0, k)

	// 遍历所有页面
	for pageIndex, page := range ci.pages {
		// 如果页面未加载，加载它
		if !page.Loaded {
			ci.mu.RUnlock()
			if err := ci.loadPage(pageIndex); err != nil {
				ci.mu.RLock()
				continue
			}
			ci.mu.RLock()
			page = ci.pages[pageIndex]
		}

		page.mu.RLock()

		// 在页面中搜索
		for id, vector := range page.Vectors {
			distance := cosineDistance(query, vector)
			results = append(results, SearchResult{
				Node: &Node{
					ID:     id,
					Vector: vector,
				},
				Distance: distance,
			})
		}

		page.mu.RUnlock()
	}

	// 按距离排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	// 返回前 k 个结果
	if len(results) > k {
		results = results[:k]
	}

	return results, nil
}

// GetVector 获取向量（按需加载）
func (ci *CoarseIndexV2) GetVector(id int) ([]float32, error) {
	ci.mu.RLock()
	pageIndex, exists := ci.pageMap[id]
	ci.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("vector with ID %d not found", id)
	}

	page := ci.pages[pageIndex]

	// 加载页面（如果未加载）
	if err := ci.loadPage(pageIndex); err != nil {
		return nil, fmt.Errorf("failed to load page: %w", err)
	}

	page.mu.RLock()
	defer page.mu.RUnlock()

	vector, exists := page.Vectors[id]
	if !exists {
		return nil, fmt.Errorf("vector with ID %d not found in page", id)
	}

	return vector, nil
}

// findOrCreatePage 查找或创建页面
func (ci *CoarseIndexV2) findOrCreatePage(id int) int {
	// 计算向量应该属于哪个页面
	pageIndex := id / ci.config.PageSize

	// 检查页面是否已存在
	if pageIndex < len(ci.pages) && ci.pages[pageIndex] != nil {
		ci.pageMap[id] = pageIndex
		return pageIndex
	}

	// 创建新页面
	if pageIndex >= ci.config.MaxPages {
		// 超过最大页数，使用 LRU 驱逐
		pageIndex = ci.evictPage()
	}

	page := &Page{
		PageID:     pageIndex,
		Vectors:    make(map[int][]float32),
		Quantized:  make(map[int][]byte),
		Loaded:     true,
		LastAccess: getCurrentTimestamp(),
	}

	// 确保pages数组足够大
	for len(ci.pages) <= pageIndex {
		ci.pages = append(ci.pages, nil)
	}
	ci.pages[pageIndex] = page
	ci.pageMap[id] = pageIndex
	ci.stats.TotalPages++

	return pageIndex
}

// loadPage 加载页面（懒加载）
func (ci *CoarseIndexV2) loadPage(pageIndex int) error {
	page := ci.pages[pageIndex]

	// 检查是否已加载（不持有锁）
	page.mu.RLock()
	loaded := page.Loaded
	page.mu.RUnlock()

	if loaded {
		ci.stats.CacheHits++
		page.mu.Lock()
		page.LastAccess = getCurrentTimestamp()
		page.mu.Unlock()
		return nil
	}

	ci.stats.CacheMisses++
	ci.stats.PageLoads++

	// 从磁盘加载页面
	if err := ci.loadPageFromDisk(pageIndex); err != nil {
		return fmt.Errorf("failed to load page from disk: %w", err)
	}

	return nil
}

// loadPageFromDisk 从磁盘加载页面
func (ci *CoarseIndexV2) loadPageFromDisk(pageIndex int) error {
	page := ci.pages[pageIndex]

	// 使用 protobuf 加载页面
	pageData, err := ci.serializer.LoadPage(pageIndex)
	if err != nil {
		return err
	}

	// 复制数据到page
	page.mu.Lock()
	defer page.mu.Unlock()

	page.PageID = int(pageData.PageId)
	page.Vectors = make(map[int][]float32)
	page.Quantized = make(map[int][]byte)

	for id, vectorData := range pageData.Vectors {
		page.Vectors[int(id)] = vectorData.Vector
	}

	for id, quantized := range pageData.Quantized {
		page.Quantized[int(id)] = quantized
	}

	page.Loaded = true
	page.LastAccess = getCurrentTimestamp()

	return nil
}

// evictPage 驱逐页面（LRU）
func (ci *CoarseIndexV2) evictPage() int {
	// 找到最久未使用的页面
	lruIndex := 0
	lruTime := int64(math.MaxInt64)

	for i, page := range ci.pages {
		if page.LastAccess < lruTime {
			lruTime = page.LastAccess
			lruIndex = i
		}
	}

	// 保存页面到磁盘
	if err := ci.savePageToDisk(lruIndex); err != nil {
		// 保存失败，继续使用该页面
		return lruIndex
	}

	// 清空页面内容（但不删除pageMap中的映射）
	ci.pages[lruIndex].mu.Lock()
	ci.pages[lruIndex].Vectors = make(map[int][]float32)
	ci.pages[lruIndex].Quantized = make(map[int][]byte)
	ci.pages[lruIndex].Loaded = false
	ci.pages[lruIndex].mu.Unlock()

	return lruIndex
}

// savePageToDisk 保存页面到磁盘
func (ci *CoarseIndexV2) savePageToDisk(pageIndex int) error {
	page := ci.pages[pageIndex]

	page.mu.RLock()
	defer page.mu.RUnlock()

	// 转换为 protobuf 格式
	pageData := &protobuf.PageDataPB{
		PageId:     int32(page.PageID),
		Vectors:    make(map[int32]*protobuf.VectorDataPB),
		Quantized:  make(map[int32][]byte),
		LastAccess: page.LastAccess,
		Loaded:     page.Loaded,
	}

	for id, vector := range page.Vectors {
		pageData.Vectors[int32(id)] = &protobuf.VectorDataPB{
			Vector: vector,
		}
	}

	for id, quantized := range page.Quantized {
		pageData.Quantized[int32(id)] = quantized
	}

	// 使用 protobuf 保存页面
	return ci.serializer.SavePage(pageData)
}

// Save 保存粗粒度索引（完整保存）
func (ci *CoarseIndexV2) Save() error {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	// 保存所有已加载的页面
	for _, page := range ci.pages {
		if page != nil && page.Loaded {
			if err := ci.savePageToDisk(page.PageID); err != nil {
				return fmt.Errorf("failed to save page %d: %w", page.PageID, err)
			}
		}
	}

	// 保存元数据
	if err := ci.saveMetadata(); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	// 清空脏页面标记
	if ci.enableDelta {
		ci.serializer.ClearDirtyPages()
	}

	return nil
}

// SaveDelta 增量保存（只保存脏页面）
func (ci *CoarseIndexV2) SaveDelta() (int, error) {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	if !ci.enableDelta {
		return 0, fmt.Errorf("delta save is not enabled")
	}

	// 获取脏页面列表
	dirtyPages := ci.serializer.GetDirtyPages()
	if len(dirtyPages) == 0 {
		return 0, nil
	}

	// 保存脏页面
	savedCount := 0
	for _, pageID := range dirtyPages {
		if pageID < len(ci.pages) && ci.pages[pageID] != nil && ci.pages[pageID].Loaded {
			if err := ci.savePageToDisk(pageID); err != nil {
				return savedCount, fmt.Errorf("failed to save dirty page %d: %w", pageID, err)
			}
			savedCount++
		}
	}

	// 保存脏页面映射
	if _, err := ci.serializer.SaveDirtyPages(); err != nil {
		return savedCount, fmt.Errorf("failed to save dirty map: %w", err)
	}

	// 清空脏页面标记
	ci.serializer.ClearDirtyPages()

	return savedCount, nil
}

// Load 加载粗粒度索引
func (ci *CoarseIndexV2) Load() error {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	// 加载元数据
	if err := ci.loadMetadata(); err != nil {
		return fmt.Errorf("failed to load metadata: %w", err)
	}

	// 页面按需加载，不在这里加载
	return nil
}

// saveMetadata 保存元数据
func (ci *CoarseIndexV2) saveMetadata() error {
	// 转换 pageMap 为 protobuf 格式
	pageMapPB := make(map[int32]int32)
	for id, pageIndex := range ci.pageMap {
		pageMapPB[int32(id)] = int32(pageIndex)
	}

	metadata := &protobuf.CoarseIndexMetadataPB{
		TotalVectors: int32(ci.stats.TotalVectors),
		TotalPages:   int32(ci.stats.TotalPages),
		PageSize:     int32(ci.config.PageSize),
		MaxPages:     int32(ci.config.MaxPages),
		UsePq:        ci.config.UsePQ,
		PqSubspaces:  int32(ci.config.PQSubspaces),
		PqCentroids:  int32(ci.config.PQCentroids),
		PageMap:      pageMapPB,
		LastModified: time.Now().UnixNano(),
	}

	return ci.serializer.SaveMetadata(metadata)
}

// loadMetadata 加载元数据
func (ci *CoarseIndexV2) loadMetadata() error {
	metadata, err := ci.serializer.LoadMetadata()
	if err != nil {
		return err
	}

	// 恢复统计信息
	ci.stats.TotalVectors = int(metadata.TotalVectors)
	ci.stats.TotalPages = int(metadata.TotalPages)

	// 恢复 pageMap
	ci.pageMap = make(map[int]int)
	for id, pageIndex := range metadata.PageMap {
		ci.pageMap[int(id)] = int(pageIndex)
	}

	// 重建页面结构
	ci.pages = make([]*Page, ci.stats.TotalPages)
	for i := 0; i < ci.stats.TotalPages; i++ {
		ci.pages[i] = &Page{
			PageID:    i,
			Vectors:   make(map[int][]float32),
			Quantized: make(map[int][]byte),
			Loaded:    false,
		}
	}

	return nil
}

// Close 关闭粗粒度索引
func (ci *CoarseIndexV2) Close() error {
	return ci.Save()
}

// GetStats 获取统计信息
func (ci *CoarseIndexV2) GetStats() CoarseStats {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	return ci.stats
}

// GetDirtyPageCount 获取脏页面数量
func (ci *CoarseIndexV2) GetDirtyPageCount() int {
	if !ci.enableDelta {
		return 0
	}
	return ci.serializer.GetDirtyPageCount()
}
