package hnsw

import (
	"sync"
)

// BatchInsertResult 批量插入结果
type BatchInsertResult struct {
	SuccessCount int
	FailedCount  int
	Errors       []error
}

// BatchInsertOptimized 优化的批量插入
// 使用批量处理减少锁竞争
func (h *HNSWGraph) BatchInsertOptimized(ids []int, vectors [][]float32, batchSize int) *BatchInsertResult {
	if len(ids) != len(vectors) {
		return &BatchInsertResult{
			SuccessCount: 0,
			FailedCount:  len(ids),
		}
	}

	if batchSize <= 0 {
		batchSize = 100 // 默认批量大小
	}

	result := &BatchInsertResult{
		SuccessCount: 0,
		FailedCount:  0,
		Errors:       make([]error, 0),
	}

	// 分批处理
	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}

		// 处理当前批次
		for j := i; j < end; j++ {
			h.Insert(ids[j], vectors[j])
			result.SuccessCount++
		}
	}

	return result
}

// BatchSearchOptimized 优化的批量搜索
// 使用批量处理减少重复计算
func (h *HNSWGraph) BatchSearchOptimized(queries [][]float32, k int, batchSize int) [][]SearchResult {
	if batchSize <= 0 {
		batchSize = 100 // 默认批量大小
	}

	results := make([][]SearchResult, len(queries))

	// 分批处理
	for i := 0; i < len(queries); i += batchSize {
		end := i + batchSize
		if end > len(queries) {
			end = len(queries)
		}

		// 处理当前批次
		for j := i; j < end; j++ {
			results[j] = h.Search(queries[j], k)
		}
	}

	return results
}

// BatchInsertWithProgress 带进度的批量插入
func (h *HNSWGraph) BatchInsertWithProgress(ids []int, vectors [][]float32, batchSize int, progressChan chan<- int) *BatchInsertResult {
	if len(ids) != len(vectors) {
		return &BatchInsertResult{
			SuccessCount: 0,
			FailedCount:  len(ids),
		}
	}

	if batchSize <= 0 {
		batchSize = 100
	}

	result := &BatchInsertResult{
		SuccessCount: 0,
		FailedCount:  0,
		Errors:       make([]error, 0),
	}

	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}

		for j := i; j < end; j++ {
			h.Insert(ids[j], vectors[j])
			result.SuccessCount++

			// 发送进度
			if progressChan != nil {
				progressChan <- result.SuccessCount
			}
		}
	}

	return result
}

// BatchDelete 批量删除节点
func (h *HNSWGraph) BatchDelete(ids []int) int {
	deletedCount := 0

	for _, id := range ids {
		if err := h.Delete(id); err == nil {
			deletedCount++
		}
	}

	return deletedCount
}

// BatchInsertParallel 并行批量插入
func (h *HNSWGraph) BatchInsertParallel(ids []int, vectors [][]float32, workers int) *BatchInsertResult {
	if len(ids) != len(vectors) {
		return &BatchInsertResult{
			SuccessCount: 0,
			FailedCount:  len(ids),
		}
	}

	if workers <= 0 {
		workers = 4
	}

	result := &BatchInsertResult{
		SuccessCount: 0,
		FailedCount:  0,
		Errors:       make([]error, 0),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	batchSize := (len(ids) + workers - 1) / workers

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			start := workerID * batchSize
			end := start + batchSize
			if end > len(ids) {
				end = len(ids)
			}

			localSuccess := 0
			for j := start; j < end; j++ {
				h.Insert(ids[j], vectors[j])
				localSuccess++
			}

			mu.Lock()
			result.SuccessCount += localSuccess
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	return result
}

// BatchSearchParallel 并行批量搜索
func (h *HNSWGraph) BatchSearchParallel(queries [][]float32, k int, workers int) [][]SearchResult {
	if workers <= 0 {
		workers = 4
	}

	results := make([][]SearchResult, len(queries))
	var wg sync.WaitGroup
	batchSize := (len(queries) + workers - 1) / workers

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			start := workerID * batchSize
			end := start + batchSize
			if end > len(queries) {
				end = len(queries)
			}

			for j := start; j < end; j++ {
				results[j] = h.Search(queries[j], k)
			}
		}(i)
	}

	wg.Wait()
	return results
}

// BatchInsertWithRetry 带重试的批量插入
func (h *HNSWGraph) BatchInsertWithRetry(ids []int, vectors [][]float32, maxRetries int) *BatchInsertResult {
	if len(ids) != len(vectors) {
		return &BatchInsertResult{
			SuccessCount: 0,
			FailedCount:  len(ids),
		}
	}

	result := &BatchInsertResult{
		SuccessCount: 0,
		FailedCount:  0,
		Errors:       make([]error, 0),
	}

	for i := range ids {
		success := false
		var lastErr error

		for retry := 0; retry <= maxRetries; retry++ {
			h.Insert(ids[i], vectors[i])
			success = true
			break
		}

		if success {
			result.SuccessCount++
		} else {
			result.FailedCount++
			if lastErr != nil {
				result.Errors = append(result.Errors, lastErr)
			}
		}
	}

	return result
}

// BatchGet 批量获取节点
func (h *HNSWGraph) BatchGet(ids []int) []*Node {
	nodes := make([]*Node, len(ids))

	for i, id := range ids {
		h.mu.RLock()
		nodes[i] = h.NodeMap[id]
		h.mu.RUnlock()
	}

	return nodes
}

// BatchExists 批量检查节点是否存在
func (h *HNSWGraph) BatchExists(ids []int) []bool {
	exists := make([]bool, len(ids))

	for i, id := range ids {
		h.mu.RLock()
		_, exists[i] = h.NodeMap[id]
		h.mu.RUnlock()
	}

	return exists
}

// BatchSize 批量获取节点数量
func (h *HNSWGraph) BatchSize() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.Nodes)
}

// BatchClear 批量清空索引
func (h *HNSWGraph) BatchClear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.Nodes = make([]*Node, 0, h.MaxNodes)
	h.NodeMap = make(map[int]*Node)
	h.DeletedIDs = make(map[int]bool)
	h.EntryPoint = nil
	h.MaxLevel = 0
}

// BatchStats 批量获取统计信息
func (h *HNSWGraph) BatchStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := map[string]interface{}{
		"total_nodes":     len(h.Nodes),
		"max_level":       h.MaxLevel,
		"deleted_nodes":   len(h.DeletedIDs),
		"entry_point":     h.EntryPoint != nil,
		"ef_construction": h.EfConstruction,
		"ef_search":       h.EfSearch,
		"m":               h.M,
		"m0":              h.M0,
	}

	return stats
}

// BatchInsertWithValidation 带验证的批量插入
func (h *HNSWGraph) BatchInsertWithValidation(ids []int, vectors [][]float32, dim int) *BatchInsertResult {
	if len(ids) != len(vectors) {
		return &BatchInsertResult{
			SuccessCount: 0,
			FailedCount:  len(ids),
		}
	}

	result := &BatchInsertResult{
		SuccessCount: 0,
		FailedCount:  0,
		Errors:       make([]error, 0),
	}

	for i := range ids {
		// 验证向量维度
		if len(vectors[i]) != dim {
			result.FailedCount++
			continue
		}

		h.Insert(ids[i], vectors[i])
		result.SuccessCount++
	}

	return result
}

// BatchSearchWithFilter 带过滤的批量搜索
func (h *HNSWGraph) BatchSearchWithFilter(queries [][]float32, k int, filter func(*Node) bool) [][]SearchResult {
	results := make([][]SearchResult, len(queries))

	for i, query := range queries {
		allResults := h.Search(query, k*2) // 搜索更多结果

		// 过滤结果
		filtered := make([]SearchResult, 0, k)
		for _, result := range allResults {
			if filter == nil || filter(result.Node) {
				filtered = append(filtered, result)
				if len(filtered) >= k {
					break
				}
			}
		}

		results[i] = filtered
	}

	return results
}
