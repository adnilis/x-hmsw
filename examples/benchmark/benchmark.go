package main

import (
	"fmt"
	"math/rand/v2"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/adnilis/x-hmsw/indexes/hnsw"
	"github.com/adnilis/x-hmsw/utils/math"
)

// BenchmarkConfig 压测配置
type BenchmarkConfig struct {
	Dimension      int
	NumVectors     int
	NumQueries     int
	NumThreads     int
	M              int
	EfConstruction int
	EfSearch       int
}

// BenchmarkResult 压测结果
type BenchmarkResult struct {
	BuildTime        time.Duration
	BuildThroughput  float64 // vectors/sec
	AvgSearchTime    time.Duration
	SearchThroughput float64 // queries/sec
	P99SearchTime    time.Duration
	MaxSearchTime    time.Duration
	TotalInserts     int64
	TotalSearches    int64
	MemoryUsageMB    float64
	Recall           float64 // 召回率
}

func main() {
	fmt.Println("=== HNSW Vector Database Benchmark ===")
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))
	fmt.Printf("CPU cores: %d\n\n", runtime.NumCPU())

	// 测试配置
	configs := []BenchmarkConfig{
		{
			Dimension:      128,
			NumVectors:     10000,
			NumQueries:     1000,
			NumThreads:     4,
			M:              16,
			EfConstruction: 200,
			EfSearch:       100,
		},
		{
			Dimension:      384,
			NumVectors:     50000,
			NumQueries:     1000,
			NumThreads:     4,
			M:              16,
			EfConstruction: 200,
			EfSearch:       100,
		},
		{
			Dimension:      128,
			NumVectors:     100000,
			NumQueries:     1000,
			NumThreads:     8,
			M:              16,
			EfConstruction: 200,
			EfSearch:       100,
		},
	}

	for i, config := range configs {
		fmt.Printf("\n=== Test %d: %d vectors, %d dim, %d threads ===\n",
			i+1, config.NumVectors, config.Dimension, config.NumThreads)
		result := runBenchmark(config)
		printResult(result)
	}

	// 并发测试
	fmt.Println("\n=== Concurrent Insert & Search Test ===")
	testConcurrent()
}

func runBenchmark(config BenchmarkConfig) BenchmarkResult {
	var result BenchmarkResult

	// 创建 HNSW 索引
	index := hnsw.NewHNSW(
		config.Dimension,
		config.M,
		config.EfConstruction,
		config.NumVectors,
		math.CosineDistance,
	)

	// 生成测试数据
	fmt.Printf("Generating %d vectors...\n", config.NumVectors)
	vectors := generateVectors(config.NumVectors, config.Dimension)
	queries := generateVectors(config.NumQueries, config.Dimension)

	// 记录初始内存
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	initialMem := m.Alloc

	// 构建索引
	fmt.Printf("Building index...\n")
	start := time.Now()
	for i, vec := range vectors {
		index.Insert(i, vec)
		if (i+1)%10000 == 0 {
			fmt.Printf("  Inserted %d/%d vectors\n", i+1, config.NumVectors)
		}
	}
	result.BuildTime = time.Since(start)
	result.TotalInserts = int64(config.NumVectors)
	result.BuildThroughput = float64(config.NumVectors) / result.BuildTime.Seconds()

	// 记录构建后内存
	runtime.ReadMemStats(&m)
	result.MemoryUsageMB = float64(m.Alloc-initialMem) / 1024 / 1024

	// 搜索测试
	fmt.Printf("Running %d queries...\n", config.NumQueries)
	searchTimes := make([]time.Duration, config.NumQueries)
	var totalSearchTime time.Duration

	for i, query := range queries {
		start := time.Now()
		results := index.Search(query, 10)
		searchTime := time.Since(start)
		searchTimes[i] = searchTime
		totalSearchTime += searchTime
		result.TotalSearches++

		if (i+1)%100 == 0 {
			fmt.Printf("  Completed %d/%d queries\n", i+1, config.NumQueries)
		}

		// 验证结果不为空
		if len(results) == 0 {
			fmt.Printf("  Warning: Query %d returned no results\n", i)
		}
	}

	// 计算搜索统计
	result.AvgSearchTime = totalSearchTime / time.Duration(config.NumQueries)
	result.SearchThroughput = float64(config.NumQueries) / totalSearchTime.Seconds()

	// 计算 P99 和最大搜索时间
	sortedTimes := make([]time.Duration, len(searchTimes))
	copy(sortedTimes, searchTimes)
	for i := 0; i < len(sortedTimes); i++ {
		for j := i + 1; j < len(sortedTimes); j++ {
			if sortedTimes[i] > sortedTimes[j] {
				sortedTimes[i], sortedTimes[j] = sortedTimes[j], sortedTimes[i]
			}
		}
	}
	p99Index := int(float64(len(sortedTimes)) * 0.99)
	if p99Index >= len(sortedTimes) {
		p99Index = len(sortedTimes) - 1
	}
	result.P99SearchTime = sortedTimes[p99Index]
	result.MaxSearchTime = sortedTimes[len(sortedTimes)-1]

	// 计算召回率（使用前 10 个查询）
	result.Recall = calculateRecall(index, vectors[:10], queries[:10], 10)

	return result
}

func testConcurrent() {
	dimension := 128
	numVectors := 10000
	numQueries := 1000
	numThreads := 8

	index := hnsw.NewHNSW(dimension, 16, 200, numVectors, math.CosineDistance)
	vectors := generateVectors(numVectors, dimension)
	queries := generateVectors(numQueries, dimension)

	// 并发插入测试
	fmt.Printf("Concurrent insert test: %d threads, %d vectors\n", numThreads, numVectors)
	start := time.Now()
	var wg sync.WaitGroup
	vectorsPerThread := numVectors / numThreads

	for t := 0; t < numThreads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()
			startIdx := threadID * vectorsPerThread
			endIdx := startIdx + vectorsPerThread
			if endIdx > numVectors {
				endIdx = numVectors
			}

			for i := startIdx; i < endIdx; i++ {
				index.Insert(i, vectors[i])
			}
		}(t)
	}
	wg.Wait()
	insertTime := time.Since(start)
	fmt.Printf("  Insert time: %v (%.2f vectors/sec)\n", insertTime,
		float64(numVectors)/insertTime.Seconds())

	// 并发搜索测试
	fmt.Printf("Concurrent search test: %d threads, %d queries\n", numThreads, numQueries)
	start = time.Now()
	var totalSearches atomic.Int64
	var failedSearches atomic.Int64

	for t := 0; t < numThreads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()
			queriesPerThread := numQueries / numThreads
			startIdx := threadID * queriesPerThread
			endIdx := startIdx + queriesPerThread
			if endIdx > numQueries {
				endIdx = numQueries
			}

			for i := startIdx; i < endIdx; i++ {
				results := index.Search(queries[i], 10)
				totalSearches.Add(1)
				if len(results) == 0 {
					failedSearches.Add(1)
				}
			}
		}(t)
	}
	wg.Wait()
	searchTime := time.Since(start)
	fmt.Printf("  Search time: %v (%.2f queries/sec)\n", searchTime,
		float64(numQueries)/searchTime.Seconds())
	fmt.Printf("  Total searches: %d, Failed: %d (%.2f%%)\n",
		totalSearches.Load(), failedSearches.Load(),
		float64(failedSearches.Load())/float64(totalSearches.Load())*100)
}

func generateVectors(count, dim int) [][]float32 {
	vectors := make([][]float32, count)
	for i := 0; i < count; i++ {
		vectors[i] = make([]float32, dim)
		for j := 0; j < dim; j++ {
			vectors[i][j] = rand.Float32()*2 - 1 // [-1, 1]
		}
	}
	return vectors
}

func calculateRecall(index *hnsw.HNSWGraph, vectors, queries [][]float32, k int) float64 {
	if len(queries) == 0 {
		return 0
	}

	var totalRecall float64
	for _, query := range queries {
		// 使用暴力搜索找到真实的最近邻
		trueNeighbors := bruteForceSearch(vectors, query, k)

		// 使用 HNSW 搜索
		hnswResults := index.Search(query, k)

		// 计算召回率
		recall := 0.0
		for _, hnswResult := range hnswResults {
			for _, trueID := range trueNeighbors {
				if hnswResult.Node.ID == trueID {
					recall++
					break
				}
			}
		}
		totalRecall += recall / float64(k)
	}

	return totalRecall / float64(len(queries))
}

func bruteForceSearch(vectors [][]float32, query []float32, k int) []int {
	type neighbor struct {
		id       int
		distance float32
	}

	neighbors := make([]neighbor, len(vectors))
	for i, vec := range vectors {
		neighbors[i] = neighbor{
			id:       i,
			distance: math.CosineDistance(vec, query),
		}
	}

	// 排序
	for i := 0; i < len(neighbors); i++ {
		for j := i + 1; j < len(neighbors); j++ {
			if neighbors[i].distance > neighbors[j].distance {
				neighbors[i], neighbors[j] = neighbors[j], neighbors[i]
			}
		}
	}

	// 返回前 k 个 ID
	result := make([]int, k)
	for i := 0; i < k && i < len(neighbors); i++ {
		result[i] = neighbors[i].id
	}
	return result
}

func printResult(result BenchmarkResult) {
	fmt.Println("\n--- Results ---")
	fmt.Printf("Build time:          %v\n", result.BuildTime)
	fmt.Printf("Build throughput:    %.2f vectors/sec\n", result.BuildThroughput)
	fmt.Printf("Avg search time:     %v\n", result.AvgSearchTime)
	fmt.Printf("P99 search time:     %v\n", result.P99SearchTime)
	fmt.Printf("Max search time:     %v\n", result.MaxSearchTime)
	fmt.Printf("Search throughput:   %.2f queries/sec\n", result.SearchThroughput)
	fmt.Printf("Total inserts:       %d\n", result.TotalInserts)
	fmt.Printf("Total searches:      %d\n", result.TotalSearches)
	fmt.Printf("Memory usage:        %.2f MB\n", result.MemoryUsageMB)
	fmt.Printf("Recall@10:           %.2f%%\n", result.Recall*100)
}
