package main

import (
	"fmt"
	"math"

	"github.com/adnilis/x-hmsw/embedding"
	"github.com/adnilis/x-hmsw/indexes/hnsw"
)

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

func main() {
	fmt.Println("=== 调试HNSW距离计算 ===\n")

	// 创建TF-IDF向量化器
	config := embedding.DefaultTFIDFConfig()
	config.Normalize = true
	vectorizer := embedding.NewTFIDFVectorizer(config)

	// 训练文档
	docs := []string{
		"机器学习算法研究",
		"深度神经网络应用",
		"自然语言处理技术",
		"计算机视觉识别",
		"machine learning algorithms",
		"deep neural networks",
		"natural language processing",
		"computer vision recognition",
	}

	fmt.Println("1. 训练TF-IDF模型...")
	vectorizer.Fit(docs)
	fmt.Printf("   词汇表大小: %d\n\n", vectorizer.GetDimension())

	// 转换文档为向量
	fmt.Println("2. 转换文档为向量...")
	vectors := make([][]float32, len(docs))
	for i, doc := range docs {
		vectors[i] = vectorizer.Transform(doc)
		fmt.Printf("   [%d] %s\n", i, doc)
		fmt.Printf("       向量范数: %.6f\n", vectorNorm(vectors[i]))
	}

	// 创建HNSW索引
	fmt.Println("\n3. 创建HNSW索引...")
	distanceFunc := func(a, b []float32) float32 {
		return 1.0 - cosineSimilarity(a, b)
	}
	index := hnsw.NewHNSW(vectorizer.GetDimension(), 16, 200, 10000, distanceFunc)

	// 插入向量
	fmt.Println("4. 插入向量到HNSW...")
	for i, vec := range vectors {
		node := index.Insert(i, vec)
		fmt.Printf("   [%d] 插入成功, 节点ID: %d\n", i, node.ID)
	}

	// 测试查询
	fmt.Println("\n5. 测试查询...")
	queries := []string{
		"机器学习",
		"machine learning",
	}

	for _, queryText := range queries {
		queryVec := vectorizer.Transform(queryText)
		fmt.Printf("\n   查询: '%s'\n", queryText)
		fmt.Printf("   查询向量范数: %.6f\n", vectorNorm(queryVec))

		// HNSW搜索
		results := index.Search(queryVec, 3)
		fmt.Printf("   HNSW搜索结果:\n")
		for i, r := range results {
			fmt.Printf("      [%d] ID=%d, Distance=%.6f, Score=%.6f\n",
				i, r.Node.ID, r.Distance, 1.0-r.Distance)
		}

		// 直接计算相似度
		fmt.Printf("   直接计算相似度:\n")
		for i, doc := range docs {
			sim := cosineSimilarity(queryVec, vectors[i])
			fmt.Printf("      [%d] %s: 相似度=%.6f\n", i, doc, sim)
		}
	}

	// 检查HNSW内部向量
	fmt.Println("\n6. 检查HNSW内部向量...")
	fmt.Println("   (跳过 - GetNode方法不可用)")
}

func vectorNorm(vec []float32) float32 {
	var sum float32
	for _, v := range vec {
		sum += v * v
	}
	return float32(math.Sqrt(float64(sum)))
}
