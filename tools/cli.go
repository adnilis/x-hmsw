package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/adnilis/x-hmsw/indexes/hnsw"
	"github.com/adnilis/x-hmsw/utils/math"
)

const (
	defaultIndexFile = "vectors.json"
	defaultDimension = 128
	defaultM         = 16
	defaultEfConstr  = 200
	defaultMaxNodes  = 10000
)

func main() {
	fmt.Println("=== Vector DB CLI Tool ===")
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "create":
		createIndex(os.Args[2:])
	case "insert":
		insertVectors(os.Args[2:])
	case "search":
		searchVectors(os.Args[2:])
	case "benchmark":
		runBenchmark()
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Print(`Usage: x-hmsw <command> [options]

Commands:
  create [dimension] [m] [ef_construction] [max_nodes]
          Create a new index (default: 128 16 200 10000)
  insert <vectors_file> [index_file]
          Insert vectors from JSON file
  search <query_vector> [k] [index_file]
          Search k nearest neighbors (default: 5)
  benchmark
          Run benchmark test
`)
}

// IndexConfig 索引配置
type IndexConfig struct {
	Dimension   int    `json:"dimension"`
	M           int    `json:"m"`
	EfConstruct int    `json:"ef_construction"`
	MaxNodes    int    `json:"max_nodes"`
	Distance    string `json:"distance"`
}

// VectorData 向量数据
type VectorData struct {
	ID     int       `json:"id"`
	Vector []float32 `json:"vector"`
}

// IndexData 索引数据
type IndexData struct {
	Config  IndexConfig  `json:"config"`
	Vectors []VectorData `json:"vectors"`
}

func createIndex(args []string) {
	// 解析参数
	dimension := defaultDimension
	m := defaultM
	efConstr := defaultEfConstr
	maxNodes := defaultMaxNodes

	if len(args) >= 1 {
		if d, err := strconv.Atoi(args[0]); err == nil {
			dimension = d
		}
	}
	if len(args) >= 2 {
		if val, err := strconv.Atoi(args[1]); err == nil {
			m = val
		}
	}
	if len(args) >= 3 {
		if val, err := strconv.Atoi(args[2]); err == nil {
			efConstr = val
		}
	}
	if len(args) >= 4 {
		if val, err := strconv.Atoi(args[3]); err == nil {
			maxNodes = val
		}
	}

	// 创建索引
	distanceFunc := math.CosineDistance
	_ = hnsw.NewHNSW(dimension, m, efConstr, maxNodes, distanceFunc)

	// 保存空索引
	data := IndexData{
		Config: IndexConfig{
			Dimension:   dimension,
			M:           m,
			EfConstruct: efConstr,
			MaxNodes:    maxNodes,
			Distance:    "cosine",
		},
		Vectors: []VectorData{},
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("错误：序列化索引失败：%v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(defaultIndexFile, jsonData, 0644); err != nil {
		fmt.Printf("错误：保存索引失败：%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ 创建索引成功\n")
	fmt.Printf("  维度：%d\n", dimension)
	fmt.Printf("  M: %d\n", m)
	fmt.Printf("  EfConstruction: %d\n", efConstr)
	fmt.Printf("  MaxNodes: %d\n", maxNodes)
	fmt.Printf("  距离度量：%s\n", data.Config.Distance)
	fmt.Printf("  文件：%s\n", defaultIndexFile)
}

func insertVectors(args []string) {
	if len(args) < 1 {
		fmt.Println("错误：需要指定向量文件")
		printUsage()
		os.Exit(1)
	}

	vectorsFile := args[0]
	indexFile := defaultIndexFile
	if len(args) > 1 {
		indexFile = args[1]
	}

	// 读取现有索引
	var data IndexData
	if existing, err := os.ReadFile(indexFile); err == nil {
		if err := json.Unmarshal(existing, &data); err != nil {
			fmt.Printf("错误：解析索引文件失败：%v\n", err)
			os.Exit(1)
		}
	}

	// 读取要插入的向量
	vectorData, err := readVectorsFromFile(vectorsFile)
	if err != nil {
		fmt.Printf("错误：读取向量文件失败：%v\n", err)
		os.Exit(1)
	}

	// 添加向量
	data.Vectors = append(data.Vectors, vectorData...)

	// 保存索引
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("错误：序列化索引失败：%v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(indexFile, jsonData, 0644); err != nil {
		fmt.Printf("错误：保存索引失败：%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ 插入 %d 个向量成功\n", len(vectorData))
	fmt.Printf("  总向量数：%d\n", len(data.Vectors))
	fmt.Printf("  文件：%s\n", indexFile)
}

func searchVectors(args []string) {
	if len(args) < 1 {
		fmt.Println("错误：需要指定查询向量")
		printUsage()
		os.Exit(1)
	}

	queryStr := args[0]
	k := 5
	indexFile := defaultIndexFile

	if len(args) > 1 {
		if val, err := strconv.Atoi(args[1]); err == nil {
			k = val
		}
	}
	if len(args) > 2 {
		indexFile = args[2]
	}

	// 解析查询向量
	query := parseVector(queryStr)
	if len(query) == 0 {
		fmt.Println("错误：无效的查询向量格式")
		os.Exit(1)
	}

	// 读取索引
	var data IndexData
	existing, err := os.ReadFile(indexFile)
	if err != nil {
		fmt.Printf("错误：读取索引文件失败：%v\n", err)
		os.Exit(1)
	}

	if err := json.Unmarshal(existing, &data); err != nil {
		fmt.Printf("错误：解析索引文件失败：%v\n", err)
		os.Exit(1)
	}

	// 创建索引并搜索
	distanceFunc := math.CosineDistance
	if data.Config.Distance == "l2" {
		distanceFunc = math.L2Distance
	}

	index := hnsw.NewHNSW(data.Config.Dimension, data.Config.M, data.Config.EfConstruct, data.Config.MaxNodes, distanceFunc)

	// 插入所有向量
	for _, vd := range data.Vectors {
		index.Insert(vd.ID, vd.Vector)
	}

	// 搜索
	results := index.Search(query, k)

	fmt.Printf("✓ 搜索结果 (k=%d)\n", k)
	fmt.Printf("  查询向量：[%s]\n", truncateVector(queryStr, 50))
	fmt.Printf("  索引文件：%s\n", indexFile)
	fmt.Printf("  找到 %d 个结果:\n", len(results))

	for i, r := range results {
		fmt.Printf("  %d. ID: %d, 距离：%.6f\n", i+1, r.Node.ID, r.Distance)
	}
}

func readVectorsFromFile(filename string) ([]VectorData, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var vectors []VectorData
	if err := json.Unmarshal(content, &vectors); err != nil {
		// 尝试解析为单个向量
		var v VectorData
		if err2 := json.Unmarshal(content, &v); err2 == nil {
			return []VectorData{v}, nil
		}
		return nil, err
	}

	return vectors, nil
}

func parseVector(s string) []float32 {
	s = strings.Trim(s, "[]")
	parts := strings.Split(s, ",")
	result := make([]float32, len(parts))
	for i, p := range parts {
		if val, err := strconv.ParseFloat(strings.TrimSpace(p), 32); err == nil {
			result[i] = float32(val)
		}
	}
	return result
}

func truncateVector(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func runBenchmark() {
	fmt.Println("Running benchmark...")
	// 创建测试数据
	dimension := 128
	numVectors := 10000
	vectors := make([][]float32, numVectors)
	for i := 0; i < numVectors; i++ {
		vectors[i] = make([]float32, dimension)
		for j := 0; j < dimension; j++ {
			vectors[i][j] = float32(i+j) / float32(dimension)
		}
	}

	// 创建 HNSW 索引
	index := hnsw.NewHNSW(dimension, 16, 200, 10000, math.CosineDistance)
	fmt.Println("Building index...")
	for i, vec := range vectors {
		index.Insert(i, vec)
	}

	// 测试搜索
	query := vectors[0]
	fmt.Println("Searching...")
	results := index.Search(query, 5)
	fmt.Printf("Found %d results\n", len(results))
	for _, r := range results {
		fmt.Printf("ID: %d, Distance: %f\n", r.Node.ID, r.Distance)
	}

	fmt.Println("\nBenchmark completed")
}
