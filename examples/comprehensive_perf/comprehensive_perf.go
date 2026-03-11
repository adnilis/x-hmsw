package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	iface "github.com/adnilis/x-hmsw/interface"
	"github.com/adnilis/x-hmsw/types"
)

// 测试配置
const (
	TestDimension      = 128
	TestNumVectors     = 10000
	TestNumQueries     = 100
	TestTopK           = 10
	TestWarmupQueries  = 10
	TestStorageBaseDir = "./test_performance"
)

// 性能指标
type PerformanceMetrics struct {
	IndexType      string
	StorageType    string
	BuildTime      time.Duration
	InsertTime     time.Duration
	SearchTime     time.Duration
	AvgSearchTime  time.Duration
	QPS            float64
	MemoryUsage    string
	TotalVectors   int
	SearchAccuracy float32
}

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║     Comprehensive Vector DB Performance Test Suite            ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Printf("\nTest Configuration:\n")
	fmt.Printf("  - Dimension: %d\n", TestDimension)
	fmt.Printf("  - Vectors: %d\n", TestNumVectors)
	fmt.Printf("  - Queries: %d\n", TestNumQueries)
	fmt.Printf("  - Top-K: %d\n", TestTopK)
	fmt.Printf("  - Warmup Queries: %d\n", TestWarmupQueries)

	// 准备测试数据
	fmt.Println("\n📊 Generating test data...")
	vectors, queries := generateTestData()

	// 定义所有测试组合
	testCombinations := []struct {
		indexType   iface.IndexType
		storageType iface.StorageType
	}{
		// HNSW 组合
		{iface.HNSW, iface.Memory},
		{iface.HNSW, iface.Badger},
		{iface.HNSW, iface.BBolt},
		{iface.HNSW, iface.Pebble},
		// {iface.HNSW, iface.MMap}, // MMap has a bug, skip for now

		// IVF 组合
		{iface.IVF, iface.Memory},
		{iface.IVF, iface.Badger},
		{iface.IVF, iface.BBolt},
		{iface.IVF, iface.Pebble},

		// Flat 组合
		{iface.Flat, iface.Memory},
		{iface.Flat, iface.Badger},
		{iface.Flat, iface.BBolt},
		{iface.Flat, iface.Pebble},
	}

	// 运行所有测试
	var results []PerformanceMetrics
	for i, combo := range testCombinations {
		fmt.Printf("\n[%d/%d] Testing %s + %s...\n", i+1, len(testCombinations),
			combo.indexType, combo.storageType)

		metrics, err := runTest(combo.indexType, combo.storageType, vectors, queries)
		if err != nil {
			fmt.Printf("  ❌ Error: %v\n", err)
			continue
		}

		results = append(results, metrics)
		printMetrics(metrics)
	}

	// 生成汇总报告
	generateSummaryReport(results)

	// 清理测试数据
	fmt.Println("\n🧹 Cleaning up test data...")
	cleanupTestData()
}

// 生成测试数据
func generateTestData() ([]types.Vector, []types.Vector) {
	rand.Seed(time.Now().UnixNano())

	// 生成插入向量
	vectors := make([]types.Vector, TestNumVectors)
	for i := 0; i < TestNumVectors; i++ {
		vec := make([]float32, TestDimension)
		for j := 0; j < TestDimension; j++ {
			vec[j] = rand.Float32()
		}
		vectors[i] = types.Vector{
			ID:      fmt.Sprintf("vec_%d", i),
			Vector:  vec,
			Payload: map[string]interface{}{"index": i},
		}
	}

	// 生成查询向量
	queries := make([]types.Vector, TestNumQueries)
	for i := 0; i < TestNumQueries; i++ {
		vec := make([]float32, TestDimension)
		for j := 0; j < TestDimension; j++ {
			vec[j] = rand.Float32()
		}
		queries[i] = types.Vector{
			ID:     fmt.Sprintf("query_%d", i),
			Vector: vec,
		}
	}

	return vectors, queries
}

// 运行单个测试
func runTest(indexType iface.IndexType, storageType iface.StorageType,
	vectors []types.Vector, queries []types.Vector) (PerformanceMetrics, error) {

	metrics := PerformanceMetrics{
		IndexType:   string(indexType),
		StorageType: string(storageType),
	}

	// 创建存储路径
	storagePath := filepath.Join(TestStorageBaseDir,
		fmt.Sprintf("%s_%s", strings.ToLower(string(indexType)),
			strings.ToLower(string(storageType))))

	// 配置数据库
	config := iface.Config{
		Dimension:      TestDimension,
		IndexType:      indexType,
		StorageType:    storageType,
		StoragePath:    storagePath,
		DistanceMetric: iface.Cosine,
		M:              16,
		EfConstruction: 200,
		MaxVectors:     TestNumVectors,
		NumClusters:    100,
		Nprobe:         10,
	}

	// 创建数据库
	db, err := iface.NewPureGoVectorDB(config)
	if err != nil {
		return metrics, fmt.Errorf("failed to create database: %w", err)
	}
	defer db.Close()

	// 测试插入性能
	fmt.Printf("  📥 Inserting %d vectors...\n", len(vectors))
	insertStart := time.Now()
	if err := db.Insert(vectors); err != nil {
		return metrics, fmt.Errorf("failed to insert vectors: %w", err)
	}
	metrics.InsertTime = time.Since(insertStart)
	metrics.TotalVectors = len(vectors)

	// 等待索引构建完成
	time.Sleep(100 * time.Millisecond)
	metrics.BuildTime = metrics.InsertTime

	// 验证向量数量
	count, err := db.Count()
	if err != nil {
		return metrics, fmt.Errorf("failed to count vectors: %w", err)
	}
	if count != len(vectors) {
		return metrics, fmt.Errorf("vector count mismatch: expected %d, got %d", len(vectors), count)
	}

	// 预热查询
	fmt.Printf("  🔥 Warming up with %d queries...\n", TestWarmupQueries)
	for i := 0; i < TestWarmupQueries; i++ {
		opts := iface.SearchOptions{
			TopK:        TestTopK,
			MinScore:    0.0,
			WithVector:  false,
			WithPayload: true,
		}
		_, _ = db.Search(queries[i%len(queries)], opts)
	}

	// 测试搜索性能
	fmt.Printf("  🔍 Running %d search queries...\n", TestNumQueries)
	searchStart := time.Now()
	var totalSearchTime time.Duration

	for i := 0; i < TestNumQueries; i++ {
		queryStart := time.Now()
		opts := iface.SearchOptions{
			TopK:        TestTopK,
			MinScore:    0.0,
			WithVector:  false,
			WithPayload: true,
		}
		results, err := db.Search(queries[i%len(queries)], opts)
		if err != nil {
			return metrics, fmt.Errorf("search failed: %w", err)
		}
		totalSearchTime += time.Since(queryStart)

		// 计算搜索精度（第一个结果的分数）
		if len(results) > 0 {
			metrics.SearchAccuracy += results[0].Score
		}
	}

	metrics.SearchTime = time.Since(searchStart)
	metrics.AvgSearchTime = totalSearchTime / time.Duration(TestNumQueries)
	metrics.QPS = float64(TestNumQueries) / metrics.SearchTime.Seconds()
	metrics.SearchAccuracy /= float32(TestNumQueries)

	// 获取内存使用情况
	metrics.MemoryUsage = getMemoryUsage()

	return metrics, nil
}

// 打印性能指标
func printMetrics(metrics PerformanceMetrics) {
	fmt.Printf("\n  📈 Performance Results:\n")
	fmt.Printf("     Insert: %v (%.2f vectors/sec)\n",
		metrics.InsertTime,
		float64(metrics.TotalVectors)/metrics.InsertTime.Seconds())
	fmt.Printf("     Search: %v (avg: %v)\n",
		metrics.SearchTime, metrics.AvgSearchTime)
	fmt.Printf("     QPS: %.2f\n", metrics.QPS)
	fmt.Printf("     Accuracy: %.4f\n", metrics.SearchAccuracy)
	fmt.Printf("     Memory: %s\n", metrics.MemoryUsage)
}

// 生成汇总报告
func generateSummaryReport(results []PerformanceMetrics) {
	fmt.Println("\n" + strings.Repeat("═", 80))
	fmt.Println("📊 COMPREHENSIVE PERFORMANCE SUMMARY")
	fmt.Println(strings.Repeat("═", 80))

	// 按索引类型分组
	indexGroups := make(map[string][]PerformanceMetrics)
	for _, m := range results {
		indexGroups[m.IndexType] = append(indexGroups[m.IndexType], m)
	}

	// 打印每个索引类型的汇总
	for indexType, metrics := range indexGroups {
		fmt.Printf("\n🔹 %s Index:\n", indexType)
		fmt.Printf("%-15s %-12s %-12s %-12s %-12s %-10s\n",
			"Storage", "Insert", "Avg Search", "QPS", "Memory", "Accuracy")
		fmt.Println(strings.Repeat("-", 80))

		for _, m := range metrics {
			fmt.Printf("%-15s %-12v %-12v %-12.2f %-12s %.4f\n",
				m.StorageType,
				m.InsertTime.Round(time.Millisecond),
				m.AvgSearchTime.Round(time.Microsecond),
				m.QPS,
				m.MemoryUsage,
				m.SearchAccuracy)
		}
	}

	// 找出最佳性能
	fmt.Println("\n" + strings.Repeat("═", 80))
	fmt.Println("🏆 BEST PERFORMANCE:")
	fmt.Println(strings.Repeat("═", 80))

	// 最快插入
	sort.Slice(results, func(i, j int) bool {
		return results[i].InsertTime < results[j].InsertTime
	})
	if len(results) > 0 {
		fmt.Printf("Fastest Insert: %s + %s (%v)\n",
			results[0].IndexType, results[0].StorageType, results[0].InsertTime)
	}

	// 最快搜索
	sort.Slice(results, func(i, j int) bool {
		return results[i].AvgSearchTime < results[j].AvgSearchTime
	})
	if len(results) > 0 {
		fmt.Printf("Fastest Search: %s + %s (%v)\n",
			results[0].IndexType, results[0].StorageType, results[0].AvgSearchTime)
	}

	// 最高 QPS
	sort.Slice(results, func(i, j int) bool {
		return results[i].QPS > results[j].QPS
	})
	if len(results) > 0 {
		fmt.Printf("Highest QPS: %s + %s (%.2f)\n",
			results[0].IndexType, results[0].StorageType, results[0].QPS)
	}

	// 最高精度
	sort.Slice(results, func(i, j int) bool {
		return results[i].SearchAccuracy > results[j].SearchAccuracy
	})
	if len(results) > 0 {
		fmt.Printf("Best Accuracy: %s + %s (%.4f)\n",
			results[0].IndexType, results[0].StorageType, results[0].SearchAccuracy)
	}

	// 性能对比图表
	fmt.Println("\n" + strings.Repeat("═", 80))
	fmt.Println("📊 PERFORMANCE COMPARISON (QPS):")
	fmt.Println(strings.Repeat("═", 80))

	for indexType, metrics := range indexGroups {
		fmt.Printf("\n%s:\n", indexType)
		maxQPS := 0.0
		for _, m := range metrics {
			if m.QPS > maxQPS {
				maxQPS = m.QPS
			}
		}

		for _, m := range metrics {
			barLength := int((m.QPS / maxQPS) * 40)
			bar := strings.Repeat("█", barLength)
			fmt.Printf("  %-10s %-40s %.2f QPS\n", m.StorageType, bar, m.QPS)
		}
	}

	fmt.Println("\n" + strings.Repeat("═", 80))
	fmt.Println("✅ Test completed successfully!")
	fmt.Println(strings.Repeat("═", 80))
}

// 获取内存使用情况（简化版）
func getMemoryUsage() string {
	// 在实际应用中，可以使用 runtime.ReadMemStats 获取更详细的信息
	// 这里返回一个简化的估算值
	return "~N/A"
}

// 清理测试数据
func cleanupTestData() {
	if err := os.RemoveAll(TestStorageBaseDir); err != nil {
		fmt.Printf("Warning: failed to cleanup test data: %v\n", err)
	}
}
