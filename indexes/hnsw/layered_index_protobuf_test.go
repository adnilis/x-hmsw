package hnsw

import (
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLayeredIndexV2Performance 测试分层索引 V2 性能
func TestLayeredIndexV2Performance(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 测试配置
	config := LayeredConfig{
		CoarsePageSize:    1000,
		CoarseMaxPages:    100,
		CoarseUsePQ:       false,
		CoarseStoragePath: filepath.Join(tempDir, "coarse_index"),

		FineMaxSize:          10000,
		FineLRUSize:          5000,
		FinePromoteThreshold: 3,
		FineEvictThreshold:   3600,

		SearchCoarseK: 100,
		SearchFineK:   10,
	}

	// 创建分层索引 V2（启用增量保存）
	li, err := NewLayeredIndexV2(config, true)
	if err != nil {
		t.Fatalf("Failed to create layered index: %v", err)
	}

	// 生成测试向量
	vectorDim := 128
	numVectors := 100000

	t.Logf("Generating %d vectors with dimension %d", numVectors, vectorDim)
	vectors := make([][]float32, numVectors)
	for i := 0; i < numVectors; i++ {
		vectors[i] = make([]float32, vectorDim)
		for j := 0; j < vectorDim; j++ {
			vectors[i][j] = rand.Float32()
		}
	}

	// 插入向量
	t.Log("Inserting vectors...")
	start := time.Now()
	for i := 0; i < numVectors; i++ {
		if err := li.Insert(i, vectors[i]); err != nil {
			t.Fatalf("Failed to insert vector %d: %v", i, err)
		}
	}
	insertTime := time.Since(start)
	t.Logf("Inserted %d vectors in %v (%.2f vectors/sec)",
		numVectors, insertTime, float64(numVectors)/insertTime.Seconds())

	// 获取脏页面数量
	dirtyPageCount := li.GetDirtyPageCount()
	t.Logf("Dirty pages: %d", dirtyPageCount)

	// 测试增量保存
	t.Log("Testing delta save...")
	start = time.Now()
	savedCount, err := li.SaveDelta()
	if err != nil {
		t.Fatalf("Failed to save delta: %v", err)
	}
	deltaSaveTime := time.Since(start)
	t.Logf("Delta save: saved %d pages in %v (%.2f pages/sec)",
		savedCount, deltaSaveTime, float64(savedCount)/deltaSaveTime.Seconds())

	// 测试搜索性能
	t.Log("Testing search performance...")
	numSearches := 1000
	start = time.Now()
	for i := 0; i < numSearches; i++ {
		query := vectors[rand.Intn(numVectors)]
		_, err := li.Search(query, 10)
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}
	}
	searchTime := time.Since(start)
	t.Logf("Search: %d searches in %v (%.2f searches/sec)",
		numSearches, searchTime, float64(numSearches)/searchTime.Seconds())

	// 获取统计信息
	stats := li.GetStats()
	t.Logf("Stats: TotalInserts=%d, TotalSearches=%d, CoarseSearches=%d, FineSearches=%d, Promotions=%d, Evictions=%d",
		stats.TotalInserts, stats.TotalSearches, stats.CoarseSearches, stats.FineSearches, stats.Promotions, stats.Evictions)

	// 获取粗粒度索引统计信息
	coarseStats := li.GetCoarseStats()
	t.Logf("Coarse Stats: TotalVectors=%d, TotalPages=%d, PageLoads=%d, CacheHits=%d, CacheMisses=%d",
		coarseStats.TotalVectors, coarseStats.TotalPages, coarseStats.PageLoads, coarseStats.CacheHits, coarseStats.CacheMisses)

	// 获取细粒度索引统计信息
	fineStats := li.GetFineStats()
	t.Logf("Fine Stats: Size=%d, Hits=%d, Misses=%d, Evictions=%d",
		fineStats.Size, fineStats.Hits, fineStats.Misses, fineStats.Evictions)

	// 计算缓存命中率
	cacheHitRate := 0.0
	if coarseStats.CacheHits+coarseStats.CacheMisses > 0 {
		cacheHitRate = float64(coarseStats.CacheHits) / float64(coarseStats.CacheHits+coarseStats.CacheMisses) * 100
	}
	t.Logf("Cache hit rate: %.2f%%", cacheHitRate)

	// 测试完整保存
	t.Log("Testing full save...")
	start = time.Now()
	if err := li.Save(); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}
	fullSaveTime := time.Since(start)
	t.Logf("Full save: completed in %v", fullSaveTime)

	// 计算存储大小
	var totalSize int64
	filepath.Walk(config.CoarseStoragePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	t.Logf("Total storage size: %.2f MB", float64(totalSize)/(1024*1024))

	// 测试加载
	t.Log("Testing load...")
	start = time.Now()
	if err := li.Load(); err != nil {
		t.Fatalf("Failed to load: %v", err)
	}
	loadTime := time.Since(start)
	t.Logf("Load: completed in %v", loadTime)

	// 测试加载后搜索性能
	t.Log("Testing search performance after load...")
	start = time.Now()
	for i := 0; i < numSearches; i++ {
		query := vectors[rand.Intn(numVectors)]
		_, err := li.Search(query, 10)
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}
	}
	searchAfterLoadTime := time.Since(start)
	t.Logf("Search after load: %d searches in %v (%.2f searches/sec)",
		numSearches, searchAfterLoadTime, float64(numSearches)/searchAfterLoadTime.Seconds())

	// 关闭索引
	if err := li.Close(); err != nil {
		t.Fatalf("Failed to close: %v", err)
	}
}

// TestLayeredIndexV2VsOriginal 比较 LayeredIndexV2 和原始 LayeredIndex 性能
func TestLayeredIndexV2VsOriginal(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 测试配置
	config := LayeredConfig{
		CoarsePageSize:       1000,
		CoarseMaxPages:       100,
		CoarseUsePQ:          false,
		CoarseStoragePath:    filepath.Join(tempDir, "test"),
		FineMaxSize:          10000,
		FineLRUSize:          5000,
		FinePromoteThreshold: 3,
		FineEvictThreshold:   3600,
		SearchCoarseK:        100,
		SearchFineK:          10,
	}

	// 生成测试向量
	vectorDim := 128
	numVectors := 10000

	t.Logf("Generating %d vectors with dimension %d", numVectors, vectorDim)
	vectors := make([][]float32, numVectors)
	for i := 0; i < numVectors; i++ {
		vectors[i] = make([]float32, vectorDim)
		for j := 0; j < vectorDim; j++ {
			vectors[i][j] = rand.Float32()
		}
	}

	// 测试原始版本
	t.Log("\n=== Testing Original LayeredIndex ===")
	originalDir := filepath.Join(tempDir, "original")
	configOriginal := config
	configOriginal.CoarseStoragePath = originalDir

	liOriginal, err := NewLayeredIndex(configOriginal)
	if err != nil {
		t.Fatalf("Failed to create original layered index: %v", err)
	}

	// 插入向量
	start := time.Now()
	for i := 0; i < numVectors; i++ {
		if err := liOriginal.Insert(i, vectors[i]); err != nil {
			t.Fatalf("Failed to insert vector %d: %v", i, err)
		}
	}
	originalInsertTime := time.Since(start)
	t.Logf("Original Insert: %v (%.2f vectors/sec)",
		originalInsertTime, float64(numVectors)/originalInsertTime.Seconds())

	// 保存
	start = time.Now()
	if err := liOriginal.Save(); err != nil {
		t.Fatalf("Failed to save original: %v", err)
	}
	originalSaveTime := time.Since(start)
	t.Logf("Original Save: %v", originalSaveTime)

	// 测试 V2 版本
	t.Log("\n=== Testing LayeredIndexV2 ===")
	v2Dir := filepath.Join(tempDir, "v2")
	configV2 := config
	configV2.CoarseStoragePath = v2Dir

	liV2, err := NewLayeredIndexV2(configV2, true)
	if err != nil {
		t.Fatalf("Failed to create layered index V2: %v", err)
	}

	// 插入向量
	start = time.Now()
	for i := 0; i < numVectors; i++ {
		if err := liV2.Insert(i, vectors[i]); err != nil {
			t.Fatalf("Failed to insert vector %d: %v", i, err)
		}
	}
	v2InsertTime := time.Since(start)
	t.Logf("V2 Insert: %v (%.2f vectors/sec)",
		v2InsertTime, float64(numVectors)/v2InsertTime.Seconds())

	// 增量保存
	start = time.Now()
	savedCount, err := liV2.SaveDelta()
	if err != nil {
		t.Fatalf("Failed to save delta: %v", err)
	}
	v2DeltaSaveTime := time.Since(start)
	t.Logf("V2 Delta Save: %v (%.2f pages/sec)",
		v2DeltaSaveTime, float64(savedCount)/v2DeltaSaveTime.Seconds())

	// 完整保存
	start = time.Now()
	if err := liV2.Save(); err != nil {
		t.Fatalf("Failed to save V2: %v", err)
	}
	v2SaveTime := time.Since(start)
	t.Logf("V2 Full Save: %v", v2SaveTime)

	// 性能对比
	t.Log("\n=== Performance Comparison ===")
	t.Logf("Insert Speed:")
	t.Logf("  Original: %.2f vectors/sec", float64(numVectors)/originalInsertTime.Seconds())
	t.Logf("  V2:       %.2f vectors/sec", float64(numVectors)/v2InsertTime.Seconds())
	t.Logf("  Speedup:  %.2fx", float64(originalInsertTime.Seconds())/float64(v2InsertTime.Seconds()))

	t.Logf("\nSave Time:")
	t.Logf("  Original: %v", originalSaveTime)
	t.Logf("  V2:       %v", v2SaveTime)
	t.Logf("  Speedup:  %.2fx", float64(originalSaveTime.Seconds())/float64(v2SaveTime.Seconds()))
}

// BenchmarkLayeredIndexV2Insert LayeredIndexV2 插入基准测试
func BenchmarkLayeredIndexV2Insert(b *testing.B) {
	tempDir := b.TempDir()

	config := LayeredConfig{
		CoarsePageSize:       1000,
		CoarseMaxPages:       100,
		CoarseUsePQ:          false,
		CoarseStoragePath:    filepath.Join(tempDir, "coarse_index"),
		FineMaxSize:          10000,
		FineLRUSize:          5000,
		FinePromoteThreshold: 3,
		FineEvictThreshold:   3600,
		SearchCoarseK:        100,
		SearchFineK:          10,
	}

	vectorDim := 128

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		li, err := NewLayeredIndexV2(config, true)
		if err != nil {
			b.Fatalf("Failed to create layered index: %v", err)
		}

		// 插入向量
		for j := 0; j < 1000; j++ {
			vector := make([]float32, vectorDim)
			for k := 0; k < vectorDim; k++ {
				vector[k] = rand.Float32()
			}
			if err := li.Insert(j, vector); err != nil {
				b.Fatalf("Failed to insert vector: %v", err)
			}
		}
	}
}

// BenchmarkLayeredIndexV2Search LayeredIndexV2 搜索基准测试
func BenchmarkLayeredIndexV2Search(b *testing.B) {
	tempDir := b.TempDir()

	config := LayeredConfig{
		CoarsePageSize:       1000,
		CoarseMaxPages:       100,
		CoarseUsePQ:          false,
		CoarseStoragePath:    filepath.Join(tempDir, "coarse_index"),
		FineMaxSize:          10000,
		FineLRUSize:          5000,
		FinePromoteThreshold: 3,
		FineEvictThreshold:   3600,
		SearchCoarseK:        100,
		SearchFineK:          10,
	}

	vectorDim := 128
	numVectors := 10000

	// 创建索引并插入向量
	li, err := NewLayeredIndexV2(config, true)
	if err != nil {
		b.Fatalf("Failed to create layered index: %v", err)
	}

	vectors := make([][]float32, numVectors)
	for i := 0; i < numVectors; i++ {
		vectors[i] = make([]float32, vectorDim)
		for j := 0; j < vectorDim; j++ {
			vectors[i][j] = rand.Float32()
		}
		if err := li.Insert(i, vectors[i]); err != nil {
			b.Fatalf("Failed to insert vector: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := vectors[rand.Intn(numVectors)]
		_, err := li.Search(query, 10)
		if err != nil {
			b.Fatalf("Failed to search: %v", err)
		}
	}
}
