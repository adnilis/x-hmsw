package hnsw

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/adnilis/x-hmsw/compression/pq"
)

// CoarseIndex 粗粒度索引
// 使用分页存储，支持按需加载和 PQ 量化
type CoarseIndex struct {
	pages     []*Page
	pageMap   map[int]int // ID 到页码的映射
	config    CoarseConfig
	quantizer *pq.ProductQuantizer // PQ 量化器
	mu        sync.RWMutex
	stats     CoarseStats
}

// CoarseConfig 粗粒度索引配置
type CoarseConfig struct {
	PageSize    int    // 每页向量数量
	MaxPages    int    // 最大页数
	UsePQ       bool   // 是否使用 PQ 量化
	PQSubspaces int    // PQ 子空间数量
	PQCentroids int    // PQ 每个子空间的质心数
	StoragePath string // 存储路径
}

// CoarseStats 粗粒度索引统计信息
type CoarseStats struct {
	TotalVectors int
	TotalPages   int
	PageLoads    int64
	CacheHits    int64
	CacheMisses  int64
}

// Page 页面
type Page struct {
	PageID     int
	Vectors    map[int][]float32 // ID -> 向量
	Quantized  map[int][]byte    // ID -> 量化后的向量（如果使用 PQ）
	Loaded     bool              // 是否已加载到内存
	LastAccess int64             // 最后访问时间（用于 LRU）
	mu         sync.RWMutex
}

// PageStorage 页面存储
type PageStorage struct {
	basePath string
	mu       sync.Mutex
}

// NewCoarseIndex 创建新的粗粒度索引
func NewCoarseIndex(config CoarseConfig) (*CoarseIndex, error) {
	// 创建存储目录
	if err := os.MkdirAll(config.StoragePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	ci := &CoarseIndex{
		pages:   make([]*Page, 0, config.MaxPages),
		pageMap: make(map[int]int),
		config:  config,
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
func (ci *CoarseIndex) Insert(id int, vector []float32) error {
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

	return nil
}

// Search 在粗粒度索引中搜索
func (ci *CoarseIndex) Search(query []float32, k int) ([]SearchResult, error) {
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
func (ci *CoarseIndex) GetVector(id int) ([]float32, error) {
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
func (ci *CoarseIndex) findOrCreatePage(id int) int {
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
func (ci *CoarseIndex) loadPage(pageIndex int) error {
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
	storage := &PageStorage{basePath: ci.config.StoragePath}
	if err := storage.LoadPage(page); err != nil {
		return fmt.Errorf("failed to load page from disk: %w", err)
	}

	return nil
}

// evictPage 驱逐页面（LRU）
func (ci *CoarseIndex) evictPage() int {
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
	storage := &PageStorage{basePath: ci.config.StoragePath}
	if err := storage.SavePage(ci.pages[lruIndex]); err != nil {
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

// Save 保存粗粒度索引
func (ci *CoarseIndex) Save() error {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	storage := &PageStorage{basePath: ci.config.StoragePath}

	// 保存所有页面
	for _, page := range ci.pages {
		if page.Loaded {
			if err := storage.SavePage(page); err != nil {
				return fmt.Errorf("failed to save page %d: %w", page.PageID, err)
			}
		}
	}

	// 保存元数据
	if err := ci.saveMetadata(); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	return nil
}

// Load 加载粗粒度索引
func (ci *CoarseIndex) Load() error {
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
func (ci *CoarseIndex) saveMetadata() error {
	metadata := map[string]interface{}{
		"totalVectors": ci.stats.TotalVectors,
		"totalPages":   ci.stats.TotalPages,
		"config":       ci.config,
		"pageMap":      ci.pageMap,
	}

	metadataPath := filepath.Join(ci.config.StoragePath, "metadata.json")
	file, err := os.Create(metadataPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(metadata)
}

// loadMetadata 加载元数据
func (ci *CoarseIndex) loadMetadata() error {
	metadataPath := filepath.Join(ci.config.StoragePath, "metadata.json")
	file, err := os.Open(metadataPath)
	if err != nil {
		return err
	}
	defer file.Close()

	var metadata map[string]interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&metadata); err != nil {
		return err
	}

	// 恢复统计信息
	if totalVectors, ok := metadata["totalVectors"].(float64); ok {
		ci.stats.TotalVectors = int(totalVectors)
	}
	if totalPages, ok := metadata["totalPages"].(float64); ok {
		ci.stats.TotalPages = int(totalPages)
	}

	// 恢复 pageMap
	if pageMapData, ok := metadata["pageMap"].(map[string]interface{}); ok {
		ci.pageMap = make(map[int]int)
		for idStr, pageIndex := range pageMapData {
			var id int
			fmt.Sscanf(idStr, "%d", &id)
			if pageIndexFloat, ok := pageIndex.(float64); ok {
				ci.pageMap[id] = int(pageIndexFloat)
			}
		}
	} else if pageMapData, ok := metadata["pageMap"].(map[string]float64); ok {
		// 处理直接反序列化为map[string]float64的情况
		ci.pageMap = make(map[int]int)
		for idStr, pageIndex := range pageMapData {
			var id int
			fmt.Sscanf(idStr, "%d", &id)
			ci.pageMap[id] = int(pageIndex)
		}
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
func (ci *CoarseIndex) Close() error {
	return ci.Save()
}

// GetStats 获取统计信息
func (ci *CoarseIndex) GetStats() CoarseStats {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	return ci.stats
}

// SavePage 保存页面到磁盘
func (ps *PageStorage) SavePage(page *Page) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	page.mu.RLock()
	defer page.mu.RUnlock()

	pagePath := filepath.Join(ps.basePath, fmt.Sprintf("page_%d.json", page.PageID))
	file, err := os.Create(pagePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(page)
}

// LoadPage 从磁盘加载页面
func (ps *PageStorage) LoadPage(page *Page) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	pagePath := filepath.Join(ps.basePath, fmt.Sprintf("page_%d.json", page.PageID))
	file, err := os.Open(pagePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 创建临时结构来解码（避免覆盖mu字段）
	type PageData struct {
		PageID    int
		Vectors   map[int][]float32
		Quantized map[int][]byte
		Loaded    bool
	}

	var data PageData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return err
	}

	// 复制数据到page
	page.mu.Lock()
	defer page.mu.Unlock()
	page.PageID = data.PageID
	page.Vectors = data.Vectors
	page.Quantized = data.Quantized
	page.Loaded = data.Loaded

	return nil
}

// cosineDistance 计算余弦距离
func cosineDistance(a, b []float32) float32 {
	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 1.0
	}
	return 1.0 - dotProduct/(float32(math.Sqrt(float64(normA)))*float32(math.Sqrt(float64(normB))))
}

// getCurrentTimestamp 获取当前时间戳
func getCurrentTimestamp() int64 {
	return 0 // 简化实现，实际应该使用 time.Now().Unix()
}
