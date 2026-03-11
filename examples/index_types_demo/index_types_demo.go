package main

import (
	"fmt"
	"time"

	iface "github.com/adnilis/x-hmsw/interface"
	"github.com/adnilis/x-hmsw/types"
)

// 测试不同索引类型
func main() {
	fmt.Println("=== 索引类型测试 ===\n")

	// 测试数据
	dimension := 128
	numVectors := 1000
	vectors := generateTestVectors(numVectors, dimension)

	// 测试 HNSW 索引
	fmt.Println("【1】测试 HNSW 索引")
	testHNSWIndex(vectors, dimension)

	// 测试 IVF 索引
	fmt.Println("\n【2】测试 IVF 索引")
	testIVFIndex(vectors, dimension)

	// 测试 Flat 索引
	fmt.Println("\n【3】测试 Flat 索引")
	testFlatIndex(vectors, dimension)

	fmt.Println("\n=== 所有索引类型测试完成 ===")
}

func generateTestVectors(count, dim int) []types.Vector {
	vectors := make([]types.Vector, count)
	for i := 0; i < count; i++ {
		vectors[i] = types.Vector{
			ID:     fmt.Sprintf("vec_%04d", i),
			Vector: make([]float32, dim),
			Payload: map[string]interface{}{
				"category": fmt.Sprintf("cat_%d", i%5),
				"index":    i,
			},
		}
		for j := 0; j < dim; j++ {
			vectors[i].Vector[j] = float32(i+j) / float32(dim)
		}
	}
	return vectors
}

func testHNSWIndex(vectors []types.Vector, dimension int) {
	config := types.Config{
		Dimension:      dimension,
		IndexType:      types.HNSW,
		DistanceMetric: types.Cosine,
		StorageType:    types.Memory,
		MaxVectors:     len(vectors),
		M:              16,
		EfConstruction: 200,
	}

	db, err := iface.NewPureGoVectorDB(config)
	if err != nil {
		fmt.Printf("  ✗ 创建数据库失败: %v\n", err)
		return
	}
	defer db.Close()

	// 插入向量
	start := time.Now()
	if err := db.Insert(vectors); err != nil {
		fmt.Printf("  ✗ 插入向量失败: %v\n", err)
		return
	}
	insertTime := time.Since(start)

	// 搜索测试
	query := vectors[0]
	start = time.Now()
	results, err := db.Search(query, iface.SearchOptions{TopK: 10})
	if err != nil {
		fmt.Printf("  ✗ 搜索失败: %v\n", err)
		return
	}
	searchTime := time.Since(start)

	// 统计
	count, _ := db.Count()

	fmt.Printf("  ✓ 插入时间: %v (%.2f vectors/sec)\n", insertTime, float64(len(vectors))/insertTime.Seconds())
	fmt.Printf("  ✓ 搜索时间: %v\n", searchTime)
	fmt.Printf("  ✓ 向量数量: %d\n", count)
	fmt.Printf("  ✓ 搜索结果: %d 个\n", len(results))
	if len(results) > 0 {
		fmt.Printf("  ✓ 第一个结果: ID=%s, Score=%.4f\n", results[0].ID, results[0].Score)
	}
}

func testIVFIndex(vectors []types.Vector, dimension int) {
	config := types.Config{
		Dimension:      dimension,
		IndexType:      types.IVF,
		DistanceMetric: types.Cosine,
		StorageType:    types.Memory,
		MaxVectors:     len(vectors),
		NumClusters:    50,
		Nprobe:         10,
	}

	db, err := iface.NewPureGoVectorDB(config)
	if err != nil {
		fmt.Printf("  ✗ 创建数据库失败: %v\n", err)
		return
	}
	defer db.Close()

	// 插入向量
	start := time.Now()
	if err := db.Insert(vectors); err != nil {
		fmt.Printf("  ✗ 插入向量失败: %v\n", err)
		return
	}
	insertTime := time.Since(start)

	// 搜索测试
	query := vectors[0]
	start = time.Now()
	results, err := db.Search(query, iface.SearchOptions{TopK: 10})
	if err != nil {
		fmt.Printf("  ✗ 搜索失败: %v\n", err)
		return
	}
	searchTime := time.Since(start)

	// 统计
	count, _ := db.Count()

	fmt.Printf("  ✓ 插入时间: %v (%.2f vectors/sec)\n", insertTime, float64(len(vectors))/insertTime.Seconds())
	fmt.Printf("  ✓ 搜索时间: %v\n", searchTime)
	fmt.Printf("  ✓ 向量数量: %d\n", count)
	fmt.Printf("  ✓ 搜索结果: %d 个\n", len(results))
	if len(results) > 0 {
		fmt.Printf("  ✓ 第一个结果: ID=%s, Score=%.4f\n", results[0].ID, results[0].Score)
	}
}

func testFlatIndex(vectors []types.Vector, dimension int) {
	config := types.Config{
		Dimension:      dimension,
		IndexType:      types.Flat,
		DistanceMetric: types.Cosine,
		StorageType:    types.Memory,
		MaxVectors:     len(vectors),
	}

	db, err := iface.NewPureGoVectorDB(config)
	if err != nil {
		fmt.Printf("  ✗ 创建数据库失败: %v\n", err)
		return
	}
	defer db.Close()

	// 插入向量
	start := time.Now()
	if err := db.Insert(vectors); err != nil {
		fmt.Printf("  ✗ 插入向量失败: %v\n", err)
		return
	}
	insertTime := time.Since(start)

	// 搜索测试
	query := vectors[0]
	start = time.Now()
	results, err := db.Search(query, iface.SearchOptions{TopK: 10})
	if err != nil {
		fmt.Printf("  ✗ 搜索失败: %v\n", err)
		return
	}
	searchTime := time.Since(start)

	// 统计
	count, _ := db.Count()

	fmt.Printf("  ✓ 插入时间: %v (%.2f vectors/sec)\n", insertTime, float64(len(vectors))/insertTime.Seconds())
	fmt.Printf("  ✓ 搜索时间: %v\n", searchTime)
	fmt.Printf("  ✓ 向量数量: %d\n", count)
	fmt.Printf("  ✓ 搜索结果: %d 个\n", len(results))
	if len(results) > 0 {
		fmt.Printf("  ✓ 第一个结果: ID=%s, Score=%.4f\n", results[0].ID, results[0].Score)
	}
}
