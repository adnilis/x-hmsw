package hnsw

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLayeredIndexBasic(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建配置
	config := DefaultLayeredConfig()
	config.CoarseStoragePath = tempDir
	config.FineMaxSize = 100
	config.FinePromoteThreshold = 2

	// 创建分层索引
	li, err := NewLayeredIndex(config)
	if err != nil {
		t.Fatalf("Failed to create layered index: %v", err)
	}
	defer li.Close()

	// 插入测试数据
	vectors := testGenerateTestVectors(1000, 128)
	for i, vector := range vectors {
		if err := li.Insert(i, vector); err != nil {
			t.Fatalf("Failed to insert vector %d: %v", i, err)
		}
	}

	// 搜索测试
	query := vectors[0]
	results, err := li.Search(query, 10)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	// 验证返回了正确数量的结果
	if len(results) < 10 {
		t.Errorf("Expected at least 10 results, got %d", len(results))
	}

	// 验证结果按距离排序
	for i := 1; i < len(results); i++ {
		if results[i].Distance < results[i-1].Distance {
			t.Errorf("Results not sorted by distance: results[%d].Distance=%f < results[%d].Distance=%f",
				i, results[i].Distance, i-1, results[i-1].Distance)
		}
	}

	// 验证第一个结果的距离应该很小（因为查询向量本身在索引中）
	if results[0].Distance > 0.1 {
		t.Errorf("Expected first result distance to be small, got %f", results[0].Distance)
	}
}

func TestLayeredIndexHotDataPromotion(t *testing.T) {
	tempDir := t.TempDir()

	config := DefaultLayeredConfig()
	config.CoarseStoragePath = tempDir
	config.FineMaxSize = 10
	config.FinePromoteThreshold = 3

	li, err := NewLayeredIndex(config)
	if err != nil {
		t.Fatalf("Failed to create layered index: %v", err)
	}
	defer li.Close()

	// 插入测试数据
	vectors := testGenerateTestVectors(100, 128)
	for i, vector := range vectors {
		li.Insert(i, vector)
	}

	// 多次搜索相同查询以触发热数据提升
	query := vectors[0]
	for i := 0; i < 5; i++ {
		_, err := li.Search(query, 5)
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}
	}

	// 检查统计信息
	stats := li.GetStats()
	if stats.Promotions == 0 {
		t.Error("Expected at least one promotion")
	}
}

func TestLayeredIndexColdDataEviction(t *testing.T) {
	tempDir := t.TempDir()

	config := DefaultLayeredConfig()
	config.CoarseStoragePath = tempDir
	config.FineMaxSize = 10
	config.FineEvictThreshold = 1 // 1秒后驱逐

	li, err := NewLayeredIndex(config)
	if err != nil {
		t.Fatalf("Failed to create layered index: %v", err)
	}
	defer li.Close()

	// 插入测试数据
	vectors := testGenerateTestVectors(100, 128)
	for i, vector := range vectors {
		li.Insert(i, vector)
	}

	// 搜索一些向量
	for i := 0; i < 5; i++ {
		li.Search(vectors[i], 5)
	}

	// 等待驱逐阈值
	time.Sleep(2 * time.Second)

	// 再次搜索以触发驱逐
	li.Search(vectors[10], 5)

	// 检查统计信息
	stats := li.GetStats()
	if stats.Evictions == 0 {
		t.Error("Expected at least one eviction")
	}
}

func TestLayeredIndexSaveLoad(t *testing.T) {
	tempDir := t.TempDir()

	config := DefaultLayeredConfig()
	config.CoarseStoragePath = tempDir
	config.FineMaxSize = 100

	// 创建并填充索引
	li1, err := NewLayeredIndex(config)
	if err != nil {
		t.Fatalf("Failed to create layered index: %v", err)
	}

	vectors := testGenerateTestVectors(1000, 128)
	for i, vector := range vectors {
		li1.Insert(i, vector)
	}

	// 保存索引
	if err := li1.Save(); err != nil {
		t.Fatalf("Failed to save index: %v", err)
	}
	li1.Close()

	// 加载索引
	li2, err := NewLayeredIndex(config)
	if err != nil {
		t.Fatalf("Failed to create layered index: %v", err)
	}
	defer li2.Close()

	if err := li2.Load(); err != nil {
		t.Fatalf("Failed to load index: %v", err)
	}

	// 验证搜索结果
	query := vectors[0]
	results2, err := li2.Search(query, 10)
	if err != nil {
		t.Fatalf("Failed to search in loaded index: %v", err)
	}

	// 验证返回了正确数量的结果
	if len(results2) < 10 {
		t.Errorf("Expected at least 10 results, got %d", len(results2))
	}

	// 验证结果按距离排序
	for i := 1; i < len(results2); i++ {
		if results2[i].Distance < results2[i-1].Distance {
			t.Errorf("Results not sorted by distance: results[%d].Distance=%f < results[%d].Distance=%f",
				i, results2[i].Distance, i-1, results2[i-1].Distance)
		}
	}

	// 验证第一个结果的距离应该很小
	if results2[0].Distance > 0.1 {
		t.Errorf("Expected first result distance to be small, got %f", results2[0].Distance)
	}
}

func TestCoarseIndexBasic(t *testing.T) {
	tempDir := t.TempDir()

	config := CoarseConfig{
		PageSize:    100,
		MaxPages:    10,
		UsePQ:       false,
		StoragePath: tempDir,
	}

	ci, err := NewCoarseIndex(config)
	if err != nil {
		t.Fatalf("Failed to create coarse index: %v", err)
	}
	defer ci.Close()

	// 插入测试数据
	vectors := testGenerateTestVectors(500, 128)
	for i, vector := range vectors {
		if err := ci.Insert(i, vector); err != nil {
			t.Fatalf("Failed to insert vector %d: %v", i, err)
		}
	}

	// 搜索测试
	query := vectors[0]
	results, err := ci.Search(query, 10)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	// 验证返回了正确数量的结果
	if len(results) < 10 {
		t.Errorf("Expected at least 10 results, got %d", len(results))
	}

	// 验证结果按距离排序
	for i := 1; i < len(results); i++ {
		if results[i].Distance < results[i-1].Distance {
			t.Errorf("Results not sorted by distance: results[%d].Distance=%f < results[%d].Distance=%f",
				i, results[i].Distance, i-1, results[i-1].Distance)
		}
	}

	// 验证第一个结果的距离应该很小
	if results[0].Distance > 0.1 {
		t.Errorf("Expected first result distance to be small, got %f", results[0].Distance)
	}
}

func TestCoarseIndexLazyLoading(t *testing.T) {
	tempDir := t.TempDir()

	config := CoarseConfig{
		PageSize:    100,
		MaxPages:    10,
		UsePQ:       false,
		StoragePath: tempDir,
	}

	ci, err := NewCoarseIndex(config)
	if err != nil {
		t.Fatalf("Failed to create coarse index: %v", err)
	}

	// 插入测试数据
	vectors := testGenerateTestVectors(500, 128)
	for i, vector := range vectors {
		ci.Insert(i, vector)
	}

	// 保存索引
	if err := ci.Save(); err != nil {
		t.Fatalf("Failed to save index: %v", err)
	}

	// 创建新索引并加载
	ci2, err := NewCoarseIndex(config)
	if err != nil {
		t.Fatalf("Failed to create coarse index: %v", err)
	}
	defer ci2.Close()

	if err := ci2.Load(); err != nil {
		t.Fatalf("Failed to load index: %v", err)
	}

	// 验证页面未加载
	for _, page := range ci2.pages {
		if page.Loaded {
			t.Error("Expected pages to be unloaded after Load()")
		}
	}

	// 获取向量应该触发懒加载
	vector, err := ci2.GetVector(0)
	if err != nil {
		t.Fatalf("Failed to get vector: %v", err)
	}

	if len(vector) != 128 {
		t.Errorf("Expected vector length 128, got %d", len(vector))
	}

	// 验证统计信息
	stats := ci2.GetStats()
	if stats.PageLoads == 0 {
		t.Error("Expected at least one page load")
	}
}

func TestFineIndexBasic(t *testing.T) {
	config := FineConfig{
		MaxSize: 100,
		LRUSize: 50,
	}

	fi := NewFineIndex(config)

	// 插入测试数据
	vectors := testGenerateTestVectors(50, 128)
	for i, vector := range vectors {
		if err := fi.Insert(i, vector); err != nil {
			t.Fatalf("Failed to insert vector %d: %v", i, err)
		}
	}

	// 搜索测试
	query := vectors[0]
	results, err := fi.Search(query, 10)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	if results[0].Node.ID != 0 {
		t.Errorf("Expected first result ID to be 0, got %d", results[0].Node.ID)
	}
}

func TestFineIndexLRU(t *testing.T) {
	config := FineConfig{
		MaxSize: 10,
		LRUSize: 5,
	}

	fi := NewFineIndex(config)

	// 插入测试数据
	vectors := testGenerateTestVectors(10, 128)
	for i, vector := range vectors {
		fi.Insert(i, vector)
	}

	// 访问一些向量
	for i := 0; i < 5; i++ {
		fi.RecordAccess(i)
	}

	// 驱逐一个向量
	evictedID := fi.EvictOne()
	if evictedID == -1 {
		t.Error("Expected to evict a vector")
	}

	// 验证被驱逐的向量不存在
	if fi.Contains(evictedID) {
		t.Errorf("Expected vector %d to be evicted", evictedID)
	}
}

func TestFineIndexSaveLoad(t *testing.T) {
	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "fine_index.json")

	config := FineConfig{
		MaxSize: 100,
		LRUSize: 50,
	}

	// 创建并填充索引
	fi1 := NewFineIndex(config)
	vectors := testGenerateTestVectors(50, 128)
	for i, vector := range vectors {
		fi1.Insert(i, vector)
	}

	// 保存索引
	if err := fi1.Save(savePath); err != nil {
		t.Fatalf("Failed to save index: %v", err)
	}

	// 加载索引
	fi2 := NewFineIndex(config)
	if err := fi2.Load(savePath); err != nil {
		t.Fatalf("Failed to load index: %v", err)
	}

	// 验证搜索结果
	query := vectors[0]
	results1, _ := fi1.Search(query, 10)
	results2, err := fi2.Search(query, 10)
	if err != nil {
		t.Fatalf("Failed to search in loaded index: %v", err)
	}

	if len(results1) != len(results2) {
		t.Errorf("Expected %d results, got %d", len(results1), len(results2))
	}
}

func TestLRUCache(t *testing.T) {
	cache := NewLRUCache(3)

	// 添加项
	cache.Put(1, []float32{1.0, 2.0})
	cache.Put(2, []float32{3.0, 4.0})
	cache.Put(3, []float32{5.0, 6.0})

	// 验证所有项都存在
	if _, ok := cache.Get(1); !ok {
		t.Error("Expected key 1 to exist")
	}
	if _, ok := cache.Get(2); !ok {
		t.Error("Expected key 2 to exist")
	}
	if _, ok := cache.Get(3); !ok {
		t.Error("Expected key 3 to exist")
	}

	// 添加第4项，应该驱逐第1项
	cache.Put(4, []float32{7.0, 8.0})

	if _, ok := cache.Get(1); ok {
		t.Error("Expected key 1 to be evicted")
	}
	if _, ok := cache.Get(4); !ok {
		t.Error("Expected key 4 to exist")
	}

	// 验证缓存大小
	if cache.Size() != 3 {
		t.Errorf("Expected cache size 3, got %d", cache.Size())
	}
}

func BenchmarkLayeredIndexInsert(b *testing.B) {
	tempDir := b.TempDir()

	config := DefaultLayeredConfig()
	config.CoarseStoragePath = tempDir
	config.FineMaxSize = 10000

	li, _ := NewLayeredIndex(config)
	defer li.Close()

	vectors := testGenerateTestVectors(b.N, 128)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		li.Insert(i, vectors[i])
	}
}

func BenchmarkLayeredIndexSearch(b *testing.B) {
	tempDir := b.TempDir()

	config := DefaultLayeredConfig()
	config.CoarseStoragePath = tempDir
	config.FineMaxSize = 10000

	li, _ := NewLayeredIndex(config)
	defer li.Close()

	// 插入测试数据
	vectors := testGenerateTestVectors(10000, 128)
	for i, vector := range vectors {
		li.Insert(i, vector)
	}

	// 预热：搜索一些查询以建立热数据
	for i := 0; i < 100; i++ {
		li.Search(vectors[i], 10)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := vectors[i%10000]
		li.Search(query, 10)
	}
}

func BenchmarkCoarseIndexSearch(b *testing.B) {
	tempDir := b.TempDir()

	config := CoarseConfig{
		PageSize:    1000,
		MaxPages:    100,
		UsePQ:       false,
		StoragePath: tempDir,
	}

	ci, _ := NewCoarseIndex(config)
	defer ci.Close()

	// 插入测试数据
	vectors := testGenerateTestVectors(10000, 128)
	for i, vector := range vectors {
		ci.Insert(i, vector)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := vectors[i%10000]
		ci.Search(query, 10)
	}
}

func BenchmarkFineIndexSearch(b *testing.B) {
	config := FineConfig{
		MaxSize: 10000,
		LRUSize: 5000,
	}

	fi := NewFineIndex(config)

	// 插入测试数据
	vectors := testGenerateTestVectors(10000, 128)
	for i, vector := range vectors {
		fi.Insert(i, vector)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := vectors[i%10000]
		fi.Search(query, 10)
	}
}

// 测试主函数
func TestMain(m *testing.M) {
	// 设置测试环境
	fmt.Println("Running layered index tests...")
	os.Exit(m.Run())
}

// 测试辅助函数：生成测试向量
func testGenerateTestVectors(count, dim int) [][]float32 {
	vectors := make([][]float32, count)
	for i := 0; i < count; i++ {
		vectors[i] = testGenerateTestVector(dim, i)
	}
	return vectors
}

func testGenerateTestVector(dim int, seed int) []float32 {
	vector := make([]float32, dim)
	for i := 0; i < dim; i++ {
		// 使用简单的伪随机数生成
		vector[i] = float32((seed*31+i*17)%1000) / 1000.0
	}
	// 归一化
	norm := float32(0)
	for _, v := range vector {
		norm += v * v
	}
	if norm > 0 {
		norm = float32(1.0) / float32(1.0+norm)
		for i := range vector {
			vector[i] *= norm
		}
	}
	return vector
}
