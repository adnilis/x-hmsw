package main

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/adnilis/x-hmsw/indexes/hnsw"
)

func main() {
	fmt.Println("=== 高级优化性能测试 ===")
	fmt.Println("测试时间:", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println()

	// 测试配置
	dim := 128
	numVectors := 10000
	numQueries := 100
	k := 10

	fmt.Printf("测试配置:\n")
	fmt.Printf("- 向量维度: %d\n", dim)
	fmt.Printf("- 向量数量: %d\n", numVectors)
	fmt.Printf("- 查询数量: %d\n", numQueries)
	fmt.Printf("- Top-K: %d\n", k)
	fmt.Println()

	// 生成测试数据
	fmt.Println("生成测试数据...")
	vectors := generateVectors(numVectors, dim)
	queries := generateVectors(numQueries, dim)
	fmt.Println("测试数据生成完成")
	fmt.Println()

	// 测试1: 基础HNSW性能
	fmt.Println("=== 测试1: 基础HNSW性能 ===")
	testBasicHNSW(vectors, queries, k)
	fmt.Println()

	// 测试2: 批量插入性能
	fmt.Println("=== 测试2: 批量插入性能 ===")
	testBatchInsert(vectors, queries, k)
	fmt.Println()

	// 测试3: 并行插入性能
	fmt.Println("=== 测试3: 并行插入性能 ===")
	testParallelInsert(vectors, queries, k)
	fmt.Println()

	// 测试4: 批量搜索性能
	fmt.Println("=== 测试4: 批量搜索性能 ===")
	testBatchSearch(vectors, queries, k)
	fmt.Println()

	// 测试5: 并行搜索性能
	fmt.Println("=== 测试5: 并行搜索性能 ===")
	testParallelSearch(vectors, queries, k)
	fmt.Println()

	// 测试6: 综合性能对比
	fmt.Println("=== 测试6: 综合性能对比 ===")
	testComprehensivePerformance(vectors, queries, k)
	fmt.Println()

	fmt.Println("=== 测试完成 ===")
}

func generateVectors(count, dim int) [][]float32 {
	vectors := make([][]float32, count)
	for i := 0; i < count; i++ {
		vectors[i] = make([]float32, dim)
		for j := 0; j < dim; j++ {
			vectors[i][j] = rand.Float32()
		}
	}
	return vectors
}

func testBasicHNSW(vectors [][]float32, queries [][]float32, k int) {
	index := hnsw.NewHNSW(128, 16, 200, 10000, innerProductDistance)

	// 插入性能测试
	fmt.Println("插入测试...")
	start := time.Now()
	for i, vec := range vectors {
		index.Insert(i, vec)
	}
	insertTime := time.Since(start)
	fmt.Printf("插入时间: %v\n", insertTime)
	fmt.Printf("平均插入时间: %v\n", insertTime/time.Duration(len(vectors)))

	// 搜索性能测试
	fmt.Println("\n搜索测试...")
	totalSearchTime := time.Duration(0)
	for _, query := range queries {
		start := time.Now()
		index.Search(query, k)
		totalSearchTime += time.Since(start)
	}
	avgSearchTime := totalSearchTime / time.Duration(len(queries))
	fmt.Printf("总搜索时间: %v\n", totalSearchTime)
	fmt.Printf("平均搜索时间: %v\n", avgSearchTime)
	fmt.Printf("QPS: %.2f\n", float64(len(queries))/totalSearchTime.Seconds())
}

func testBatchInsert(vectors [][]float32, queries [][]float32, k int) {
	index := hnsw.NewHNSW(128, 16, 200, 10000, innerProductDistance)

	// 准备ID和向量
	ids := make([]int, len(vectors))
	for i := range ids {
		ids[i] = i
	}

	// 批量插入性能测试
	fmt.Println("批量插入测试...")
	start := time.Now()
	result := index.BatchInsertOptimized(ids, vectors, 100)
	batchInsertTime := time.Since(start)
	fmt.Printf("批量插入时间: %v\n", batchInsertTime)
	fmt.Printf("成功插入: %d\n", result.SuccessCount)
	fmt.Printf("失败插入: %d\n", result.FailedCount)
	fmt.Printf("平均插入时间: %v\n", batchInsertTime/time.Duration(len(vectors)))

	// 搜索性能测试
	fmt.Println("\n搜索测试...")
	totalSearchTime := time.Duration(0)
	for _, query := range queries {
		start := time.Now()
		index.Search(query, k)
		totalSearchTime += time.Since(start)
	}
	avgSearchTime := totalSearchTime / time.Duration(len(queries))
	fmt.Printf("总搜索时间: %v\n", totalSearchTime)
	fmt.Printf("平均搜索时间: %v\n", avgSearchTime)
	fmt.Printf("QPS: %.2f\n", float64(len(queries))/totalSearchTime.Seconds())
}

func testParallelInsert(vectors [][]float32, queries [][]float32, k int) {
	// 准备ID和向量
	ids := make([]int, len(vectors))
	for i := range ids {
		ids[i] = i
	}

	// 并行插入性能测试
	fmt.Println("并行插入测试...")
	for workers := 2; workers <= 8; workers *= 2 {
		// 创建新的索引
		index := hnsw.NewHNSW(128, 16, 200, 10000, innerProductDistance)

		start := time.Now()
		result := index.BatchInsertParallel(ids, vectors, workers)
		parallelInsertTime := time.Since(start)
		fmt.Printf("Workers=%d: 插入时间=%v, 成功=%d, 平均=%v\n",
			workers, parallelInsertTime, result.SuccessCount,
			parallelInsertTime/time.Duration(len(vectors)))
	}

	// 搜索性能测试
	fmt.Println("\n搜索测试...")
	index := hnsw.NewHNSW(128, 16, 200, 10000, innerProductDistance)
	for i, vec := range vectors {
		index.Insert(i, vec)
	}

	totalSearchTime := time.Duration(0)
	for _, query := range queries {
		start := time.Now()
		index.Search(query, k)
		totalSearchTime += time.Since(start)
	}
	avgSearchTime := totalSearchTime / time.Duration(len(queries))
	fmt.Printf("总搜索时间: %v\n", totalSearchTime)
	fmt.Printf("平均搜索时间: %v\n", avgSearchTime)
	fmt.Printf("QPS: %.2f\n", float64(len(queries))/totalSearchTime.Seconds())
}

func testBatchSearch(vectors [][]float32, queries [][]float32, k int) {
	index := hnsw.NewHNSW(128, 16, 200, 10000, innerProductDistance)

	// 插入数据
	fmt.Println("插入数据...")
	for i, vec := range vectors {
		index.Insert(i, vec)
	}
	fmt.Println("数据插入完成")

	// 批量搜索性能测试
	fmt.Println("\n批量搜索测试...")
	start := time.Now()
	results := index.BatchSearchOptimized(queries, k, 100)
	batchSearchTime := time.Since(start)
	fmt.Printf("批量搜索时间: %v\n", batchSearchTime)
	fmt.Printf("搜索结果数: %d\n", len(results))
	fmt.Printf("平均搜索时间: %v\n", batchSearchTime/time.Duration(len(queries)))
	fmt.Printf("QPS: %.2f\n", float64(len(queries))/batchSearchTime.Seconds())
}

func testParallelSearch(vectors [][]float32, queries [][]float32, k int) {
	index := hnsw.NewHNSW(128, 16, 200, 10000, innerProductDistance)

	// 插入数据
	fmt.Println("插入数据...")
	for i, vec := range vectors {
		index.Insert(i, vec)
	}
	fmt.Println("数据插入完成")

	// 并行搜索性能测试
	fmt.Println("\n并行搜索测试...")
	for workers := 2; workers <= 8; workers *= 2 {
		start := time.Now()
		results := index.BatchSearchParallel(queries, k, workers)
		parallelSearchTime := time.Since(start)
		fmt.Printf("Workers=%d: 搜索时间=%v, 结果数=%d, 平均=%v, QPS=%.2f\n",
			workers, parallelSearchTime, len(results),
			parallelSearchTime/time.Duration(len(queries)),
			float64(len(queries))/parallelSearchTime.Seconds())
	}
}

func testComprehensivePerformance(vectors [][]float32, queries [][]float32, k int) {
	fmt.Println("对比不同优化策略的性能...")

	// 准备ID和向量
	ids := make([]int, len(vectors))
	for i := range ids {
		ids[i] = i
	}

	// 测试1: 顺序插入 + 顺序搜索
	fmt.Println("\n1. 顺序插入 + 顺序搜索")
	index1 := hnsw.NewHNSW(128, 16, 200, 10000, innerProductDistance)

	start := time.Now()
	for i, vec := range vectors {
		index1.Insert(i, vec)
	}
	insertTime1 := time.Since(start)

	start = time.Now()
	for _, query := range queries {
		index1.Search(query, k)
	}
	searchTime1 := time.Since(start)

	fmt.Printf("  插入: %v, 搜索: %v, 总计: %v\n", insertTime1, searchTime1, insertTime1+searchTime1)

	// 测试2: 批量插入 + 顺序搜索
	fmt.Println("\n2. 批量插入 + 顺序搜索")
	index2 := hnsw.NewHNSW(128, 16, 200, 10000, innerProductDistance)

	start = time.Now()
	index2.BatchInsertOptimized(ids, vectors, 100)
	insertTime2 := time.Since(start)

	start = time.Now()
	for _, query := range queries {
		index2.Search(query, k)
	}
	searchTime2 := time.Since(start)

	fmt.Printf("  插入: %v, 搜索: %v, 总计: %v\n", insertTime2, searchTime2, insertTime2+searchTime2)

	// 测试3: 并行插入 + 顺序搜索
	fmt.Println("\n3. 并行插入(4 workers) + 顺序搜索")
	index3 := hnsw.NewHNSW(128, 16, 200, 10000, innerProductDistance)

	start = time.Now()
	index3.BatchInsertParallel(ids, vectors, 4)
	insertTime3 := time.Since(start)

	start = time.Now()
	for _, query := range queries {
		index3.Search(query, k)
	}
	searchTime3 := time.Since(start)

	fmt.Printf("  插入: %v, 搜索: %v, 总计: %v\n", insertTime3, searchTime3, insertTime3+searchTime3)

	// 测试4: 顺序插入 + 批量搜索
	fmt.Println("\n4. 顺序插入 + 批量搜索")
	index4 := hnsw.NewHNSW(128, 16, 200, 10000, innerProductDistance)

	start = time.Now()
	for i, vec := range vectors {
		index4.Insert(i, vec)
	}
	insertTime4 := time.Since(start)

	start = time.Now()
	index4.BatchSearchOptimized(queries, k, 100)
	searchTime4 := time.Since(start)

	fmt.Printf("  插入: %v, 搜索: %v, 总计: %v\n", insertTime4, searchTime4, insertTime4+searchTime4)

	// 测试5: 顺序插入 + 并行搜索
	fmt.Println("\n5. 顺序插入 + 并行搜索(4 workers)")
	index5 := hnsw.NewHNSW(128, 16, 200, 10000, innerProductDistance)

	start = time.Now()
	for i, vec := range vectors {
		index5.Insert(i, vec)
	}
	insertTime5 := time.Since(start)

	start = time.Now()
	index5.BatchSearchParallel(queries, k, 4)
	searchTime5 := time.Since(start)

	fmt.Printf("  插入: %v, 搜索: %v, 总计: %v\n", insertTime5, searchTime5, insertTime5+searchTime5)

	// 测试6: 并行插入 + 并行搜索
	fmt.Println("\n6. 并行插入(4 workers) + 并行搜索(4 workers)")
	index6 := hnsw.NewHNSW(128, 16, 200, 10000, innerProductDistance)

	start = time.Now()
	index6.BatchInsertParallel(ids, vectors, 4)
	insertTime6 := time.Since(start)

	start = time.Now()
	index6.BatchSearchParallel(queries, k, 4)
	searchTime6 := time.Since(start)

	fmt.Printf("  插入: %v, 搜索: %v, 总计: %v\n", insertTime6, searchTime6, insertTime6+searchTime6)

	// 性能提升总结
	fmt.Println("\n=== 性能提升总结 ===")
	fmt.Printf("批量插入 vs 顺序插入: %.2fx\n", float64(insertTime1)/float64(insertTime2))
	fmt.Printf("并行插入 vs 顺序插入: %.2fx\n", float64(insertTime1)/float64(insertTime3))
	fmt.Printf("批量搜索 vs 顺序搜索: %.2fx\n", float64(searchTime1)/float64(searchTime4))
	fmt.Printf("并行搜索 vs 顺序搜索: %.2fx\n", float64(searchTime1)/float64(searchTime5))
	fmt.Printf("并行插入+搜索 vs 顺序插入+搜索: %.2fx\n",
		float64(insertTime1+searchTime1)/float64(insertTime6+searchTime6))
}

func innerProductDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dot float32
	for i := range a {
		dot += a[i] * b[i]
	}
	return 1.0 - dot
}
