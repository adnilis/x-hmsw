package concurrency

import "sync"

// WorkerPool 工作池
type WorkerPool struct {
	workers int
	tasks   chan func()
	wg      sync.WaitGroup
}

// NewWorkerPool 创建工作池
func NewWorkerPool(workers int) *WorkerPool {
	if workers <= 0 {
		workers = 4
	}

	pool := &WorkerPool{
		workers: workers,
		tasks:   make(chan func(), workers*2),
	}

	// 启动工作线程
	for i := 0; i < workers; i++ {
		go func() {
			for task := range pool.tasks {
				task()
			}
		}()
	}

	return pool
}

// Submit 提交任务
func (wp *WorkerPool) Submit(task func()) {
	wp.wg.Add(1)
	wp.tasks <- func() {
		defer wp.wg.Done()
		task()
	}
}

// Wait 等待所有任务完成
func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}

// Close 关闭工作池
func (wp *WorkerPool) Close() {
	close(wp.tasks)
}

// WorkerPool 批量点积（并行版本）
func (wp *WorkerPool) BatchDot(vectors [][]float32, query []float32) []float32 {
	results := make([]float32, len(vectors))
	var wg sync.WaitGroup

	// 分块处理
	chunkSize := (len(vectors) + wp.workers - 1) / wp.workers
	for i := 0; i < len(vectors); i += chunkSize {
		end := i + chunkSize
		if end > len(vectors) {
			end = len(vectors)
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for j := start; j < end; j++ {
				results[j] = DotProduct(vectors[j], query)
			}
		}(i, end)
	}

	wg.Wait()
	return results
}

// DotProduct 点积
func DotProduct(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var sum float32
	for i := range a {
		sum += a[i] * b[i]
	}

	return sum
}

// GetWorkerCount 获取工作线程数
func GetWorkerCount() int {
	return 4
}

// ParallelSlice 并发处理切片
func ParallelSlice[T any](input []T, process func(T, int)) {
	n := len(input)
	if n == 0 {
		return
	}

	chunkSize := (n + GetWorkerCount() - 1) / GetWorkerCount()
	var wg sync.WaitGroup

	for i := 0; i < GetWorkerCount(); i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > n {
			end = n
		}
		if start >= n {
			break
		}

		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			for j := s; j < e; j++ {
				process(input[j], j)
			}
		}(start, end)
	}

	wg.Wait()
}
