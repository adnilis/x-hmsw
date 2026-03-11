package hnsw

import (
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestCoarseIndexProtobufPerformance 测试 protobuf 序列化性能
func TestCoarseIndexProtobufPerformance(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 测试配置
	config := CoarseConfig{
		PageSize:    1000,
		MaxPages:    100,
		UsePQ:       false,
		StoragePath: filepath.Join(tempDir, "protobuf"),
	}

	// 创建粗粒度索引 V2（protobuf）
	ci, err := NewCoarseIndexV2(config, true)
	if err != nil {
		t.Fatalf("Failed to create coarse index: %v", err)
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

	// 插入向量
	t.Log("Inserting vectors...")
	start := time.Now()
	for i := 0; i < numVectors; i++ {
		if err := ci.Insert(i, vectors[i]); err != nil {
			t.Fatalf("Failed to insert vector %d: %v", i, err)
		}
	}
	insertTime := time.Since(start)
	t.Logf("Inserted %d vectors in %v (%.2f vectors/sec)",
		numVectors, insertTime, float64(numVectors)/insertTime.Seconds())

	// 获取脏页面数量
	dirtyPageCount := ci.GetDirtyPageCount()
	t.Logf("Dirty pages: %d", dirtyPageCount)

	// 测试增量保存
	t.Log("Testing delta save...")
	start = time.Now()
	savedCount, err := ci.SaveDelta()
	if err != nil {
		t.Fatalf("Failed to save delta: %v", err)
	}
	deltaSaveTime := time.Since(start)
	t.Logf("Delta save: saved %d pages in %v (%.2f pages/sec)",
		savedCount, deltaSaveTime, float64(savedCount)/deltaSaveTime.Seconds())

	// 测试完整保存
	t.Log("Testing full save...")
	start = time.Now()
	if err := ci.Save(); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}
	fullSaveTime := time.Since(start)
	t.Logf("Full save: completed in %v", fullSaveTime)

	// 计算存储大小
	var totalSize int64
	filepath.Walk(config.StoragePath, func(path string, info os.FileInfo, err error) error {
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
	if err := ci.Load(); err != nil {
		t.Fatalf("Failed to load: %v", err)
	}
	loadTime := time.Since(start)
	t.Logf("Load: completed in %v", loadTime)

	// 测试搜索性能
	t.Log("Testing search performance...")
	numSearches := 1000
	start = time.Now()
	for i := 0; i < numSearches; i++ {
		query := vectors[rand.Intn(numVectors)]
		_, err := ci.Search(query, 10)
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}
	}
	searchTime := time.Since(start)
	t.Logf("Search: %d searches in %v (%.2f searches/sec)",
		numSearches, searchTime, float64(numSearches)/searchTime.Seconds())

	// 获取统计信息
	stats := ci.GetStats()
	t.Logf("Stats: TotalVectors=%d, TotalPages=%d, PageLoads=%d, CacheHits=%d, CacheMisses=%d",
		stats.TotalVectors, stats.TotalPages, stats.PageLoads, stats.CacheHits, stats.CacheMisses)

	// 关闭索引
	if err := ci.Close(); err != nil {
		t.Fatalf("Failed to close: %v", err)
	}
}

// TestCoarseIndexJSONVsProtobuf 比较 JSON 和 protobuf 性能
func TestCoarseIndexJSONVsProtobuf(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 测试配置
	config := CoarseConfig{
		PageSize:    1000,
		MaxPages:    100,
		UsePQ:       false,
		StoragePath: filepath.Join(tempDir, "test"),
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

	// 测试 JSON 版本
	t.Log("\n=== Testing JSON Version ===")
	jsonDir := filepath.Join(tempDir, "json")
	configJSON := config
	configJSON.StoragePath = jsonDir

	ciJSON, err := NewCoarseIndex(CoarseConfig{
		PageSize:    configJSON.PageSize,
		MaxPages:    configJSON.MaxPages,
		UsePQ:       configJSON.UsePQ,
		StoragePath: configJSON.StoragePath,
	})
	if err != nil {
		t.Fatalf("Failed to create JSON coarse index: %v", err)
	}

	// 插入向量
	start := time.Now()
	for i := 0; i < numVectors; i++ {
		if err := ciJSON.Insert(i, vectors[i]); err != nil {
			t.Fatalf("Failed to insert vector %d: %v", i, err)
		}
	}
	jsonInsertTime := time.Since(start)
	t.Logf("JSON Insert: %v (%.2f vectors/sec)",
		jsonInsertTime, float64(numVectors)/jsonInsertTime.Seconds())

	// 保存
	start = time.Now()
	if err := ciJSON.Save(); err != nil {
		t.Fatalf("Failed to save JSON: %v", err)
	}
	jsonSaveTime := time.Since(start)
	t.Logf("JSON Save: %v", jsonSaveTime)

	// 计算存储大小
	var jsonSize int64
	filepath.Walk(jsonDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			jsonSize += info.Size()
		}
		return nil
	})
	t.Logf("JSON Storage size: %.2f MB", float64(jsonSize)/(1024*1024))

	// 测试 protobuf 版本
	t.Log("\n=== Testing Protobuf Version ===")
	protobufDir := filepath.Join(tempDir, "protobuf")
	configProtobuf := config
	configProtobuf.StoragePath = protobufDir

	ciProtobuf, err := NewCoarseIndexV2(configProtobuf, true)
	if err != nil {
		t.Fatalf("Failed to create protobuf coarse index: %v", err)
	}

	// 插入向量
	start = time.Now()
	for i := 0; i < numVectors; i++ {
		if err := ciProtobuf.Insert(i, vectors[i]); err != nil {
			t.Fatalf("Failed to insert vector %d: %v", i, err)
		}
	}
	protobufInsertTime := time.Since(start)
	t.Logf("Protobuf Insert: %v (%.2f vectors/sec)",
		protobufInsertTime, float64(numVectors)/protobufInsertTime.Seconds())

	// 增量保存
	start = time.Now()
	savedCount, err := ciProtobuf.SaveDelta()
	if err != nil {
		t.Fatalf("Failed to save delta: %v", err)
	}
	protobufDeltaSaveTime := time.Since(start)
	t.Logf("Protobuf Delta Save: %v (%.2f pages/sec)",
		protobufDeltaSaveTime, float64(savedCount)/protobufDeltaSaveTime.Seconds())

	// 完整保存
	start = time.Now()
	if err := ciProtobuf.Save(); err != nil {
		t.Fatalf("Failed to save protobuf: %v", err)
	}
	protobufSaveTime := time.Since(start)
	t.Logf("Protobuf Full Save: %v", protobufSaveTime)

	// 计算存储大小
	var protobufSize int64
	filepath.Walk(protobufDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			protobufSize += info.Size()
		}
		return nil
	})
	t.Logf("Protobuf Storage size: %.2f MB", float64(protobufSize)/(1024*1024))

	// 性能对比
	t.Log("\n=== Performance Comparison ===")
	t.Logf("Insert Speed:")
	t.Logf("  JSON:      %.2f vectors/sec", float64(numVectors)/jsonInsertTime.Seconds())
	t.Logf("  Protobuf:  %.2f vectors/sec", float64(numVectors)/protobufInsertTime.Seconds())
	t.Logf("  Speedup:   %.2fx", float64(jsonInsertTime.Seconds())/float64(protobufInsertTime.Seconds()))

	t.Logf("\nSave Time:")
	t.Logf("  JSON:      %v", jsonSaveTime)
	t.Logf("  Protobuf:  %v", protobufSaveTime)
	t.Logf("  Speedup:   %.2fx", float64(jsonSaveTime.Seconds())/float64(protobufSaveTime.Seconds()))

	t.Logf("\nStorage Size:")
	t.Logf("  JSON:      %.2f MB", float64(jsonSize)/(1024*1024))
	t.Logf("  Protobuf:  %.2f MB", float64(protobufSize)/(1024*1024))
	t.Logf("  Reduction: %.2f%%", (1.0-float64(protobufSize)/float64(jsonSize))*100)

	// 验证 protobuf 性能更好
	if protobufSaveTime > jsonSaveTime {
		t.Logf("Warning: Protobuf save time (%v) is slower than JSON (%v)", protobufSaveTime, jsonSaveTime)
	}

	if protobufSize > jsonSize {
		t.Logf("Warning: Protobuf storage size (%.2f MB) is larger than JSON (%.2f MB)",
			float64(protobufSize)/(1024*1024), float64(jsonSize)/(1024*1024))
	}
}

// BenchmarkCoarseIndexSaveJSON JSON 保存基准测试
func BenchmarkCoarseIndexSaveJSON(b *testing.B) {
	tempDir := b.TempDir()

	config := CoarseConfig{
		PageSize:    1000,
		MaxPages:    100,
		UsePQ:       false,
		StoragePath: filepath.Join(tempDir, "json"),
	}

	vectorDim := 128
	numVectors := 10000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ci, err := NewCoarseIndex(config)
		if err != nil {
			b.Fatalf("Failed to create coarse index: %v", err)
		}

		// 插入向量
		for j := 0; j < numVectors; j++ {
			vector := make([]float32, vectorDim)
			for k := 0; k < vectorDim; k++ {
				vector[k] = rand.Float32()
			}
			if err := ci.Insert(j, vector); err != nil {
				b.Fatalf("Failed to insert vector: %v", err)
			}
		}

		// 保存
		if err := ci.Save(); err != nil {
			b.Fatalf("Failed to save: %v", err)
		}
	}
}

// BenchmarkCoarseIndexSaveProtobuf Protobuf 保存基准测试
func BenchmarkCoarseIndexSaveProtobuf(b *testing.B) {
	tempDir := b.TempDir()

	config := CoarseConfig{
		PageSize:    1000,
		MaxPages:    100,
		UsePQ:       false,
		StoragePath: filepath.Join(tempDir, "protobuf"),
	}

	vectorDim := 128
	numVectors := 10000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ci, err := NewCoarseIndexV2(config, true)
		if err != nil {
			b.Fatalf("Failed to create coarse index: %v", err)
		}

		// 插入向量
		for j := 0; j < numVectors; j++ {
			vector := make([]float32, vectorDim)
			for k := 0; k < vectorDim; k++ {
				vector[k] = rand.Float32()
			}
			if err := ci.Insert(j, vector); err != nil {
				b.Fatalf("Failed to insert vector: %v", err)
			}
		}

		// 保存
		if err := ci.Save(); err != nil {
			b.Fatalf("Failed to save: %v", err)
		}
	}
}

// BenchmarkCoarseIndexSaveDeltaProtobuf Protobuf 增量保存基准测试
func BenchmarkCoarseIndexSaveDeltaProtobuf(b *testing.B) {
	tempDir := b.TempDir()

	config := CoarseConfig{
		PageSize:    1000,
		MaxPages:    100,
		UsePQ:       false,
		StoragePath: filepath.Join(tempDir, "protobuf"),
	}

	vectorDim := 128
	numVectors := 10000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ci, err := NewCoarseIndexV2(config, true)
		if err != nil {
			b.Fatalf("Failed to create coarse index: %v", err)
		}

		// 插入向量
		for j := 0; j < numVectors; j++ {
			vector := make([]float32, vectorDim)
			for k := 0; k < vectorDim; k++ {
				vector[k] = rand.Float32()
			}
			if err := ci.Insert(j, vector); err != nil {
				b.Fatalf("Failed to insert vector: %v", err)
			}
		}

		// 增量保存
		if _, err := ci.SaveDelta(); err != nil {
			b.Fatalf("Failed to save delta: %v", err)
		}
	}
}
