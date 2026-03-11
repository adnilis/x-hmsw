package hnsw

import (
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// generateRandomVector 生成随机向量
func generateRandomVector(dim int) []float32 {
	vector := make([]float32, dim)
	for i := 0; i < dim; i++ {
		vector[i] = rand.Float32()
	}
	return vector
}

// TestLayeredIndexPerformance 测试分层索引性能
func TestLayeredIndexPerformance(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "perf_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := DefaultLayeredConfig()
	config.CoarsePageSize = 1000
	config.CoarseMaxPages = 100
	config.FineMaxSize = 10000
	config.FinePromoteThreshold = 3
	config.CoarseStoragePath = tempDir

	li, err := NewLayeredIndex(config)
	if err != nil {
		t.Fatalf("Failed to create layered index: %v", err)
	}
	defer li.Close()

	// 测试插入性能
	t.Log("=== Insert Performance ===")
	numVectors := 100000
	start := time.Now()
	for i := 0; i < numVectors; i++ {
		vector := generateRandomVector(128)
		li.Insert(i, vector)
	}
	insertDuration := time.Since(start)
	t.Logf("Inserted %d vectors in %v (%.2f vectors/sec)",
		numVectors, insertDuration, float64(numVectors)/insertDuration.Seconds())

	// 测试搜索性能
	t.Log("\n=== Search Performance ===")
	query := generateRandomVector(128)
	numSearches := 1000
	start = time.Now()
	for i := 0; i < numSearches; i++ {
		li.Search(query, 10)
	}
	searchDuration := time.Since(start)
	t.Logf("Performed %d searches in %v (%.2f searches/sec)",
		numSearches, searchDuration, float64(numSearches)/searchDuration.Seconds())

	// 测试热数据提升
	t.Log("\n=== Hot Data Promotion ===")
	stats := li.GetStats()
	t.Logf("Promotions: %d", stats.Promotions)
	t.Logf("Cache Hit Rate: %.2f%%", stats.CacheHitRate*100)

	// 测试保存和加载性能
	t.Log("\n=== Save/Load Performance ===")
	start = time.Now()
	if err := li.Save(); err != nil {
		t.Fatalf("Failed to save index: %v", err)
	}
	saveDuration := time.Since(start)
	t.Logf("Saved index in %v", saveDuration)

	// 计算文件大小
	var totalSize int64
	filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	t.Logf("Total storage size: %.2f MB", float64(totalSize)/(1024*1024))

	// 测试加载性能
	start = time.Now()
	li2, err := NewLayeredIndex(config)
	if err != nil {
		t.Fatalf("Failed to create layered index: %v", err)
	}
	defer li2.Close()
	if err := li2.Load(); err != nil {
		t.Fatalf("Failed to load index: %v", err)
	}
	loadDuration := time.Since(start)
	t.Logf("Loaded index in %v", loadDuration)

	// 验证加载后的搜索性能
	start = time.Now()
	for i := 0; i < numSearches; i++ {
		li2.Search(query, 10)
	}
	searchAfterLoadDuration := time.Since(start)
	t.Logf("Performed %d searches after load in %v (%.2f searches/sec)",
		numSearches, searchAfterLoadDuration, float64(numSearches)/searchAfterLoadDuration.Seconds())

	// 最终统计
	stats = li2.GetStats()
	fineStats := li2.fineIndex.GetStats()
	coarseStats := li2.coarseIndex.GetStats()
	t.Log("\n=== Final Statistics ===")
	t.Logf("Total Vectors: %d", coarseStats.TotalVectors)
	t.Logf("Total Pages: %d", coarseStats.TotalPages)
	t.Logf("Fine Index Size: %d", fineStats.Size)
	t.Logf("Cache Hit Rate: %.2f%%", stats.CacheHitRate*100)
	t.Logf("Promotions: %d", stats.Promotions)
	t.Logf("Evictions: %d", stats.Evictions)
}

// TestLayeredIndexMemoryReduction 测试内存减少效果
func TestLayeredIndexMemoryReduction(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "memory_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Log("=== Memory Reduction Test ===")

	// 测试不同规模的向量集
	testCases := []struct {
		name       string
		numVectors int
		pageSize   int
		fineSize   int
	}{
		{"Small", 10000, 1000, 1000},
		{"Medium", 50000, 1000, 5000},
		{"Large", 100000, 1000, 10000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := DefaultLayeredConfig()
			config.CoarsePageSize = tc.pageSize
			config.CoarseMaxPages = tc.numVectors/tc.pageSize + 10
			config.FineMaxSize = tc.fineSize
			config.FinePromoteThreshold = 3
			config.CoarseStoragePath = tempDir

			li, err := NewLayeredIndex(config)
			if err != nil {
				t.Fatalf("Failed to create layered index: %v", err)
			}
			defer li.Close()

			// 插入向量
			for i := 0; i < tc.numVectors; i++ {
				vector := generateRandomVector(128)
				li.Insert(i, vector)
			}

			// 执行一些搜索以触发热数据提升
			query := generateRandomVector(128)
			for i := 0; i < 100; i++ {
				li.Search(query, 10)
			}

			// 获取统计信息
			stats := li.GetStats()
			fineStats := li.fineIndex.GetStats()

			// 计算内存使用
			// 假设每个float32占用4字节
			vectorSize := 128 * 4 // 每个向量512字节
			totalMemory := float64(tc.numVectors * vectorSize)
			fineIndexMemory := float64(fineStats.Size * vectorSize)
			memoryReduction := (totalMemory - fineIndexMemory) / totalMemory * 100

			t.Logf("\n%s Test:", tc.name)
			t.Logf("  Total Vectors: %d", tc.numVectors)
			t.Logf("  Fine Index Size: %d", fineStats.Size)
			t.Logf("  Total Memory (if all in memory): %.2f MB", totalMemory/(1024*1024))
			t.Logf("  Fine Index Memory: %.2f MB", fineIndexMemory/(1024*1024))
			t.Logf("  Memory Reduction: %.2f%%", memoryReduction)
			t.Logf("  Cache Hit Rate: %.2f%%", stats.CacheHitRate*100)
			t.Logf("  Promotions: %d", stats.Promotions)
			t.Logf("  Evictions: %d", stats.Evictions)

			// 验证内存减少是否达到目标
			if memoryReduction < 70 {
				t.Logf("Warning: Memory reduction (%.2f%%) is below target (70%%)", memoryReduction)
			}
		})
	}
}

// TestLayeredIndexScalability 测试分层索引可扩展性
func TestLayeredIndexScalability(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "scalability_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Log("=== Scalability Test ===")

	// 测试不同规模的向量集
	scales := []int{10000, 50000, 100000, 200000}

	for _, numVectors := range scales {
		config := DefaultLayeredConfig()
		config.CoarsePageSize = 1000
		config.CoarseMaxPages = numVectors/1000 + 10
		config.FineMaxSize = 10000
		config.FinePromoteThreshold = 3
		config.CoarseStoragePath = tempDir

		li, err := NewLayeredIndex(config)
		if err != nil {
			t.Fatalf("Failed to create layered index: %v", err)
		}
		defer li.Close()

		// 测试插入性能
		start := time.Now()
		for i := 0; i < numVectors; i++ {
			vector := generateRandomVector(128)
			li.Insert(i, vector)
		}
		insertDuration := time.Since(start)

		// 测试搜索性能
		query := generateRandomVector(128)
		start = time.Now()
		for i := 0; i < 100; i++ {
			li.Search(query, 10)
		}
		searchDuration := time.Since(start)

		// 获取统计信息
		stats := li.GetStats()
		fineStats := li.fineIndex.GetStats()

		t.Logf("\nScale: %d vectors", numVectors)
		t.Logf("  Insert Time: %v (%.2f vectors/sec)", insertDuration, float64(numVectors)/insertDuration.Seconds())
		t.Logf("  Search Time (100 searches): %v (%.2f searches/sec)", searchDuration, 100.0/searchDuration.Seconds())
		t.Logf("  Fine Index Size: %d", fineStats.Size)
		t.Logf("  Cache Hit Rate: %.2f%%", stats.CacheHitRate*100)
	}
}
