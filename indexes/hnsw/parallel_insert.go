package hnsw

import (
	"sync"
)

// BatchInsert 批量插入节点（并行优化）
// 使用worker pool并行处理多个插入操作
func (h *HNSWGraph) BatchInsert(ids []int, vectors [][]float32, workers int) error {
	if len(ids) != len(vectors) {
		return nil
	}

	if workers <= 0 {
		workers = 4 // 默认4个worker
	}

	// 创建任务队列
	taskQueue := make(chan insertTask, len(ids))
	results := make(chan error, len(ids))

	// 启动worker pool
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskQueue {
				h.Insert(task.id, task.vector)
				results <- nil
			}
		}()
	}

	// 分发任务
	go func() {
		for i := range ids {
			taskQueue <- insertTask{
				id:     ids[i],
				vector: vectors[i],
			}
		}
		close(taskQueue)
	}()

	// 等待所有worker完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	for range results {
		// 忽略错误，继续处理
	}

	return nil
}

// insertTask 插入任务
type insertTask struct {
	id     int
	vector []float32
}

// BatchSearch 批量搜索（并行优化）
// 使用worker pool并行处理多个搜索操作
func (h *HNSWGraph) BatchSearch(queries [][]float32, k int, workers int) [][]SearchResult {
	if workers <= 0 {
		workers = 4 // 默认4个worker
	}

	// 创建任务队列
	taskQueue := make(chan searchTask, len(queries))
	results := make(chan searchResult, len(queries))

	// 启动worker pool
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskQueue {
				result := h.Search(task.query, task.k)
				results <- searchResult{
					index:   task.index,
					results: result,
				}
			}
		}()
	}

	// 分发任务
	go func() {
		for i, query := range queries {
			taskQueue <- searchTask{
				index: i,
				query: query,
				k:     k,
			}
		}
		close(taskQueue)
	}()

	// 等待所有worker完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	output := make([][]SearchResult, len(queries))
	for result := range results {
		output[result.index] = result.results
	}

	return output
}

// searchTask 搜索任务
type searchTask struct {
	index int
	query []float32
	k     int
}

// searchResult 搜索结果
type searchResult struct {
	index   int
	results []SearchResult
}

// ParallelInsertWithBarrier 带屏障的并行插入
// 确保所有插入操作完成后再继续
func (h *HNSWGraph) ParallelInsertWithBarrier(ids []int, vectors [][]float32, workers int) error {
	if len(ids) != len(vectors) {
		return nil
	}

	if workers <= 0 {
		workers = 4
	}

	var wg sync.WaitGroup
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

			for j := start; j < end; j++ {
				h.Insert(ids[j], vectors[j])
			}
		}(i)
	}

	wg.Wait()
	return nil
}

// ParallelSearchWithBarrier 带屏障的并行搜索
func (h *HNSWGraph) ParallelSearchWithBarrier(queries [][]float32, k int, workers int) [][]SearchResult {
	if workers <= 0 {
		workers = 4
	}

	var wg sync.WaitGroup
	results := make([][]SearchResult, len(queries))
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
