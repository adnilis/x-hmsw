package hnsw

import (
	"container/list"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

// FineIndex 细粒度索引
// 使用 LRU 缓存管理热数据
type FineIndex struct {
	vectors    map[int][]float32 // ID -> 向量
	accessMap  map[int]int       // ID -> 访问次数
	accessTime map[int]int64     // ID -> 最后访问时间
	lru        *LRUCache         // LRU 缓存
	config     FineConfig
	mu         sync.RWMutex
	stats      FineStats
}

// FineConfig 细粒度索引配置
type FineConfig struct {
	MaxSize int // 最大容量（向量数量）
	LRUSize int // LRU 缓存大小
}

// FineStats 细粒度索引统计信息
type FineStats struct {
	Size      int
	Hits      int64
	Misses    int64
	HitRate   float64
	Evictions int64
}

// LRUCache LRU 缓存实现
type LRUCache struct {
	capacity int
	list     *list.List
	items    map[int]*list.Element
	mu       sync.RWMutex
}

// lruItem LRU 缓存项
type lruItem struct {
	key   int
	value []float32
}

// NewFineIndex 创建新的细粒度索引
func NewFineIndex(config FineConfig) *FineIndex {
	return &FineIndex{
		vectors:    make(map[int][]float32),
		accessMap:  make(map[int]int),
		accessTime: make(map[int]int64),
		lru:        NewLRUCache(config.LRUSize),
		config:     config,
		stats: FineStats{
			Size: 0,
		},
	}
}

// Insert 插入向量到细粒度索引
func (fi *FineIndex) Insert(id int, vector []float32) error {
	fi.mu.Lock()
	defer fi.mu.Unlock()

	// 检查容量
	if len(fi.vectors) >= fi.config.MaxSize {
		return fmt.Errorf("fine index is full (max size: %d)", fi.config.MaxSize)
	}

	// 存储向量
	fi.vectors[id] = vector
	fi.accessMap[id] = 0
	fi.accessTime[id] = time.Now().Unix()

	// 添加到 LRU 缓存
	fi.lru.Put(id, vector)

	fi.stats.Size = len(fi.vectors)

	return nil
}

// Search 在细粒度索引中搜索
func (fi *FineIndex) Search(query []float32, k int) ([]SearchResult, error) {
	fi.mu.RLock()
	defer fi.mu.RUnlock()

	results := make([]SearchResult, 0, k)

	// 遍历所有向量
	for id, vector := range fi.vectors {
		distance := cosineDistance(query, vector)
		results = append(results, SearchResult{
			Node: &Node{
				ID:     id,
				Vector: vector,
			},
			Distance: distance,
		})
	}

	// 按距离排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	// 返回前 k 个结果
	if len(results) > k {
		results = results[:k]
	}

	// 更新统计信息
	if len(results) > 0 {
		fi.stats.Hits++
	} else {
		fi.stats.Misses++
	}
	fi.updateHitRate()

	return results, nil
}

// Delete 从细粒度索引中删除向量
func (fi *FineIndex) Delete(id int) error {
	fi.mu.Lock()
	defer fi.mu.Unlock()

	if _, exists := fi.vectors[id]; !exists {
		return fmt.Errorf("vector with ID %d not found", id)
	}

	delete(fi.vectors, id)
	delete(fi.accessMap, id)
	delete(fi.accessTime, id)

	// 从 LRU 缓存中删除
	fi.lru.Remove(id)

	fi.stats.Size = len(fi.vectors)

	return nil
}

// Contains 检查向量是否存在
func (fi *FineIndex) Contains(id int) bool {
	fi.mu.RLock()
	defer fi.mu.RUnlock()

	_, exists := fi.vectors[id]
	return exists
}

// GetVector 获取向量
func (fi *FineIndex) GetVector(id int) ([]float32, error) {
	fi.mu.RLock()
	defer fi.mu.RUnlock()

	vector, exists := fi.vectors[id]
	if !exists {
		return nil, fmt.Errorf("vector with ID %d not found", id)
	}

	return vector, nil
}

// RecordAccess 记录访问
func (fi *FineIndex) RecordAccess(id int) {
	fi.mu.Lock()
	defer fi.mu.Unlock()

	// 记录访问次数和时间（即使向量不在细粒度索引中）
	fi.accessMap[id]++
	fi.accessTime[id] = time.Now().Unix()

	// 如果向量在细粒度索引中，更新 LRU 缓存
	if vector, ok := fi.vectors[id]; ok {
		fi.lru.Put(id, vector)
	}
}

// GetAccessCount 获取访问次数
func (fi *FineIndex) GetAccessCount(id int) int {
	fi.mu.RLock()
	defer fi.mu.RUnlock()

	return fi.accessMap[id]
}

// GetAccessTime 获取最后访问时间
func (fi *FineIndex) GetAccessTime(id int) int64 {
	fi.mu.RLock()
	defer fi.mu.RUnlock()

	return fi.accessTime[id]
}

// Size 获取当前大小
func (fi *FineIndex) Size() int {
	fi.mu.RLock()
	defer fi.mu.RUnlock()

	return len(fi.vectors)
}

// EvictOne 驱逐一个最久未使用的向量
func (fi *FineIndex) EvictOne() int {
	fi.mu.Lock()
	defer fi.mu.Unlock()

	// 从 LRU 缓存中获取最久未使用的项
	id, _, ok := fi.lru.GetOldest()
	if !ok {
		return -1
	}

	// 删除向量
	delete(fi.vectors, id)
	delete(fi.accessMap, id)
	delete(fi.accessTime, id)

	fi.stats.Size = len(fi.vectors)
	fi.stats.Evictions++

	return id
}

// EvictCold 驱逐冷数据
func (fi *FineIndex) EvictCold(threshold int64) []int {
	fi.mu.Lock()
	defer fi.mu.Unlock()

	now := time.Now().Unix()
	evicted := make([]int, 0)

	for id, accessTime := range fi.accessTime {
		if now-accessTime > threshold {
			delete(fi.vectors, id)
			delete(fi.accessMap, id)
			delete(fi.accessTime, id)
			fi.lru.Remove(id)
			evicted = append(evicted, id)
		}
	}

	fi.stats.Size = len(fi.vectors)
	fi.stats.Evictions += int64(len(evicted))

	return evicted
}

// GetStats 获取统计信息
func (fi *FineIndex) GetStats() FineStats {
	fi.mu.RLock()
	defer fi.mu.RUnlock()

	return fi.stats
}

// updateHitRate 更新命中率
func (fi *FineIndex) updateHitRate() {
	total := fi.stats.Hits + fi.stats.Misses
	if total > 0 {
		fi.stats.HitRate = float64(fi.stats.Hits) / float64(total)
	}
}

// Save 保存细粒度索引
func (fi *FineIndex) Save(path string) error {
	fi.mu.RLock()
	defer fi.mu.RUnlock()

	data := map[string]interface{}{
		"vectors":    fi.vectors,
		"accessMap":  fi.accessMap,
		"accessTime": fi.accessTime,
		"config":     fi.config,
		"stats":      fi.stats,
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// Load 加载细粒度索引
func (fi *FineIndex) Load(path string) error {
	fi.mu.Lock()
	defer fi.mu.Unlock()

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var data map[string]interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return err
	}

	// 恢复 vectors
	if vectorsData, ok := data["vectors"].(map[string]interface{}); ok {
		fi.vectors = make(map[int][]float32)
		for idStr, vector := range vectorsData {
			var id int
			fmt.Sscanf(idStr, "%d", &id)
			if vectorSlice, ok := vector.([]interface{}); ok {
				vec := make([]float32, len(vectorSlice))
				for i, v := range vectorSlice {
					if f, ok := v.(float64); ok {
						vec[i] = float32(f)
					}
				}
				fi.vectors[id] = vec
			}
		}
	}

	// 恢复 accessMap
	if accessMapData, ok := data["accessMap"].(map[string]interface{}); ok {
		fi.accessMap = make(map[int]int)
		for idStr, count := range accessMapData {
			var id int
			fmt.Sscanf(idStr, "%d", &id)
			if countFloat, ok := count.(float64); ok {
				fi.accessMap[id] = int(countFloat)
			}
		}
	}

	// 恢复 accessTime
	if accessTimeData, ok := data["accessTime"].(map[string]interface{}); ok {
		fi.accessTime = make(map[int]int64)
		for idStr, timeVal := range accessTimeData {
			var id int
			fmt.Sscanf(idStr, "%d", &id)
			if timeFloat, ok := timeVal.(float64); ok {
				fi.accessTime[id] = int64(timeFloat)
			}
		}
	}

	// 恢复统计信息
	if statsData, ok := data["stats"].(map[string]interface{}); ok {
		if size, ok := statsData["Size"].(float64); ok {
			fi.stats.Size = int(size)
		}
		if hits, ok := statsData["Hits"].(float64); ok {
			fi.stats.Hits = int64(hits)
		}
		if misses, ok := statsData["Misses"].(float64); ok {
			fi.stats.Misses = int64(misses)
		}
		if hitRate, ok := statsData["HitRate"].(float64); ok {
			fi.stats.HitRate = hitRate
		}
		if evictions, ok := statsData["Evictions"].(float64); ok {
			fi.stats.Evictions = int64(evictions)
		}
	}

	// 重建 LRU 缓存
	fi.lru = NewLRUCache(fi.config.LRUSize)
	for id, vector := range fi.vectors {
		fi.lru.Put(id, vector)
	}

	return nil
}

// Close 关闭细粒度索引
func (fi *FineIndex) Close() error {
	return nil
}

// NewLRUCache 创建新的 LRU 缓存
func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		list:     list.New(),
		items:    make(map[int]*list.Element),
	}
}

// Put 添加或更新缓存项
func (lc *LRUCache) Put(key int, value []float32) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	// 如果已存在，更新并移到前面
	if elem, exists := lc.items[key]; exists {
		lc.list.MoveToFront(elem)
		elem.Value.(*lruItem).value = value
		return
	}

	// 如果缓存已满，删除最久未使用的项
	if lc.list.Len() >= lc.capacity {
		if elem := lc.list.Back(); elem != nil {
			lc.list.Remove(elem)
			delete(lc.items, elem.Value.(*lruItem).key)
		}
	}

	// 添加新项
	item := &lruItem{key: key, value: value}
	elem := lc.list.PushFront(item)
	lc.items[key] = elem
}

// Get 获取缓存项
func (lc *LRUCache) Get(key int) ([]float32, bool) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if elem, exists := lc.items[key]; exists {
		lc.list.MoveToFront(elem)
		return elem.Value.(*lruItem).value, true
	}

	return nil, false
}

// Remove 删除缓存项
func (lc *LRUCache) Remove(key int) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if elem, exists := lc.items[key]; exists {
		lc.list.Remove(elem)
		delete(lc.items, key)
	}
}

// GetOldest 获取最久未使用的项
func (lc *LRUCache) GetOldest() (int, []float32, bool) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if elem := lc.list.Back(); elem != nil {
		item := elem.Value.(*lruItem)
		return item.key, item.value, true
	}

	return 0, nil, false
}

// Size 获取缓存大小
func (lc *LRUCache) Size() int {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	return lc.list.Len()
}

// Clear 清空缓存
func (lc *LRUCache) Clear() {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.list.Init()
	lc.items = make(map[int]*list.Element)
}
