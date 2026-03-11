package embedding

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// 生成随机文档用于压力测试
func generateRandomDocs(count, minLen, maxLen int) []string {
	docs := make([]string, count)
	words := []string{"go", "python", "java", "编程", "语言", "开发", "数据", "算法",
		"系统", "性能", "优化", "搜索", "索引", "向量", "数据库", "机器学习", "人工智能"}

	for i := 0; i < count; i++ {
		length := minLen + rand.Intn(maxLen-minLen+1)
		doc := ""
		for j := 0; j < length; j++ {
			if j > 0 {
				doc += " "
			}
			doc += words[rand.Intn(len(words))]
		}
		docs[i] = doc
	}
	return docs
}

// 生成多字段随机文档
func generateRandomMultiFieldDocs(count int) []map[string]string {
	docs := make([]map[string]string, count)
	titles := []string{"Go 编程", "Python 数据分析", "Java 开发", "机器学习", "人工智能", "系统优化"}
	tags := []string{"go,编程", "python,数据", "java,企业", "ml,ai", "system,perf"}

	for i := 0; i < count; i++ {
		docs[i] = map[string]string{
			"title":   titles[rand.Intn(len(titles))],
			"content": generateRandomDocs(1, 20, 100)[0],
			"tags":    tags[rand.Intn(len(tags))],
		}
	}
	return docs
}

// BenchmarkBM25Fit BM25 训练基准测试
func BenchmarkBM25Fit(b *testing.B) {
	config := DefaultBM25Config()
	config.MaxVocabSize = 5000

	docs := generateRandomDocs(1000, 50, 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := NewBM25Vectorizer(config)
		v.Fit(docs)
	}
}

// BenchmarkBM25Transform BM25 向量化基准测试
func BenchmarkBM25Transform(b *testing.B) {
	config := DefaultBM25Config()
	v := NewBM25Vectorizer(config)

	docs := generateRandomDocs(500, 50, 200)
	v.Fit(docs)

	query := "编程 语言 开发"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Transform(query)
	}
}

// BenchmarkBM25BatchTransform 批量向量化基准测试
func BenchmarkBM25BatchTransform(b *testing.B) {
	config := DefaultBM25Config()
	v := NewBM25Vectorizer(config)

	docs := generateRandomDocs(500, 50, 200)
	v.Fit(docs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.BatchTransform(docs)
	}
}

// BenchmarkBM25F_Fit BM25F 训练基准测试
func BenchmarkBM25F_Fit(b *testing.B) {
	config := BM25FConfig{
		BM25Config:   BM25Config{MaxVocabSize: 5000},
		FieldWeights: map[string]float64{"title": 2.0, "content": 1.0, "tags": 1.5},
	}

	docs := generateRandomMultiFieldDocs(500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := NewBM25FVectorizer(config)
		v.Fit(docs)
	}
}

// BenchmarkBM25F_Transform BM25F 向量化基准测试
func BenchmarkBM25F_Transform(b *testing.B) {
	config := BM25FConfig{
		BM25Config:   BM25Config{MaxVocabSize: 5000},
		FieldWeights: map[string]float64{"title": 2.0, "content": 1.0},
	}

	docs := generateRandomMultiFieldDocs(500)
	v := NewBM25FVectorizer(config)
	v.Fit(docs)

	query := map[string]string{"title": "Go", "content": "编程"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Transform(query)
	}
}

// BenchmarkBM25F_BatchTransform BM25F 批量向量化基准测试
func BenchmarkBM25F_BatchTransform(b *testing.B) {
	config := BM25FConfig{
		BM25Config:   BM25Config{MaxVocabSize: 5000},
		FieldWeights: map[string]float64{"title": 2.0, "content": 1.0},
	}

	docs := generateRandomMultiFieldDocs(500)
	v := NewBM25FVectorizer(config)
	v.Fit(docs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.BatchTransform(docs)
	}
}

// BenchmarkIncrementalAdd 增量添加文档基准测试
func BenchmarkIncrementalAdd(b *testing.B) {
	config := BM25Config{MaxVocabSize: 10000}
	v := NewIncrementalBM25Vectorizer(config)

	// 初始文档
	initialDocs := generateRandomDocs(1000, 50, 200)
	v.Fit(initialDocs)

	newDocs := generateRandomDocs(100, 50, 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.AddDocuments(newDocs)
	}
}

// BenchmarkSparseTransform 稀疏向量化基准测试
func BenchmarkSparseTransform(b *testing.B) {
	config := BM25Config{MaxVocabSize: 5000}
	v := NewSparseBM25Vectorizer(config)
	v.SetThreshold(0.1)

	docs := generateRandomDocs(500, 50, 200)
	v.Fit(docs)

	query := "编程 语言"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.TransformToSparse(query)
	}
}

// BenchmarkQueryOptimizerTopK Top-K 搜索基准测试
func BenchmarkQueryOptimizerTopK(b *testing.B) {
	config := DefaultBM25Config()
	v := NewBM25Vectorizer(config)

	docs := generateRandomDocs(1000, 50, 200)
	v.Fit(docs)

	optimizer := NewQueryOptimizer(v)
	optimizer.SetMaxResults(100)

	// 核心优化：预计算所有文档向量
	optimizer.PrecomputeDocVectors(docs)

	query := "编程 语言 开发"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = optimizer.TopKSearch(query, docs, 10)
	}
}

// BenchmarkQueryOptimizerPruned 剪枝搜索基准测试
func BenchmarkQueryOptimizerPruned(b *testing.B) {
	config := DefaultBM25Config()
	v := NewBM25Vectorizer(config)

	docs := generateRandomDocs(1000, 50, 200)
	v.Fit(docs)

	optimizer := NewQueryOptimizer(v)

	// 核心优化：预计算所有文档向量
	optimizer.PrecomputeDocVectors(docs)

	query := "编程"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = optimizer.PrunedSearch(query, docs, 10, 0.2)
	}
}

// TestBM25Performance 性能测试
func TestBM25Performance(t *testing.T) {
	config := DefaultBM25Config()
	config.MaxVocabSize = 10000

	// 测试不同数据规模
	scales := []struct {
		name     string
		docCount int
		docLen   int
	}{
		{"small", 100, 50},
		{"medium", 1000, 100},
		{"large", 5000, 150},
	}

	for _, scale := range scales {
		t.Run(scale.name, func(t *testing.T) {
			docs := generateRandomDocs(scale.docCount, scale.docLen, scale.docLen*2)

			start := time.Now()
			v := NewBM25Vectorizer(config)
			v.Fit(docs)
			fitTime := time.Since(start)

			start = time.Now()
			for i := 0; i < 100; i++ {
				_ = v.Transform("编程 语言")
			}
			transformTime := time.Since(start)

			t.Logf("规模 %s: 训练时间=%v, 100 次向量化=%v",
				scale.name, fitTime, transformTime)
		})
	}
}

// TestSparseMemoryUsage 稀疏向量内存使用测试
func TestSparseMemoryUsage(t *testing.T) {
	config := BM25Config{MaxVocabSize: 10000}
	v := NewSparseBM25Vectorizer(config)

	docs := generateRandomDocs(1000, 100, 200)
	v.Fit(docs)

	// 测试不同阈值下的内存使用
	thresholds := []float32{0.0, 0.05, 0.1, 0.2, 0.5}

	for _, threshold := range thresholds {
		v.SetThreshold(threshold)
		sparseVec := v.TransformToSparse("编程 语言 开发 系统")

		denseSize := v.GetVocabularySize() * 4 // 4 bytes per float32
		sparseSize := (len(sparseVec.Indices) + len(sparseVec.Values)) * 4

		t.Logf("阈值 %.2f: 稀疏元素=%d/%d, 节省内存=%.2f%%",
			threshold, len(sparseVec.Indices), v.GetVocabularySize(),
			float64(denseSize-sparseSize)/float64(denseSize)*100)
	}
}

// BenchmarkCachePerformance 缓存性能测试
func BenchmarkCachePerformance(b *testing.B) {
	config := DefaultBM25Config()
	v := NewBM25Vectorizer(config)

	docs := generateRandomDocs(500, 50, 200)
	v.Fit(docs)

	query := "编程 语言 开发"

	// 无缓存
	b.Run("NoCache", func(b *testing.B) {
		v.DisableCache()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = v.Transform(query)
		}
	})

	// 有缓存
	b.Run("WithCache", func(b *testing.B) {
		v.EnableCache()
		// 预热缓存
		_ = v.Transform(query)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = v.Transform(query)
		}
	})
}

// 打印性能报告
func PrintPerformanceReport() {
	fmt.Println("=== BM25 性能报告 ===")

	config := DefaultBM25Config()
	config.MaxVocabSize = 10000

	docs := generateRandomDocs(5000, 100, 150)

	// 训练性能
	start := time.Now()
	v := NewBM25Vectorizer(config)
	v.Fit(docs)
	fitTime := time.Since(start)
	fmt.Printf("训练 5000 个文档 (平均 125 词): %v\n", fitTime)

	// 向量化性能
	start = time.Now()
	for i := 0; i < 1000; i++ {
		_ = v.Transform("编程 语言 开发")
	}
	transformTime := time.Since(start)
	fmt.Printf("1000 次向量化操作：%v (平均 %.2f μs/次)\n",
		transformTime, float64(transformTime.Microseconds())/1000)

	// 批量向量化
	start = time.Now()
	_ = v.BatchTransform(docs[:100])
	batchTime := time.Since(start)
	fmt.Printf("批量向量化 100 个文档：%v\n", batchTime)

	// 搜索性能
	optimizer := NewQueryOptimizer(v)
	optimizer.SetMaxResults(100)

	start = time.Now()
	for i := 0; i < 100; i++ {
		_ = optimizer.TopKSearch("编程 语言", docs, 10)
	}
	searchTime := time.Since(start)
	fmt.Printf("100 次 Top-K 搜索 (1000 文档): %v (平均 %.2f μs/次)\n",
		searchTime, float64(searchTime.Microseconds())/100)

	// 稀疏向量
	sparseV := NewSparseBM25Vectorizer(config)
	sparseV.SetThreshold(0.1)
	sparseV.Fit(docs)

	start = time.Now()
	for i := 0; i < 1000; i++ {
		_ = sparseV.TransformToSparse("编程 语言")
	}
	sparseTime := time.Since(start)
	fmt.Printf("1000 次稀疏向量化：%v (平均 %.2f μs/次)\n",
		sparseTime, float64(sparseTime.Microseconds())/1000)
}

// BenchmarkBM25L_Fit BM25L 训练基准测试
func BenchmarkBM25L_Fit(b *testing.B) {
	config := DefaultBM25Config()
	config.MaxVocabSize = 5000

	docs := generateRandomDocs(1000, 50, 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := NewBM25LVectorizer(config)
		v.Fit(docs)
	}
}

// BenchmarkBM25L_Transform BM25L 向量化基准测试
func BenchmarkBM25L_Transform(b *testing.B) {
	config := DefaultBM25Config()
	v := NewBM25LVectorizer(config)

	docs := generateRandomDocs(500, 50, 200)
	v.Fit(docs)

	query := "编程 语言 开发"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Transform(query)
	}
}

// BenchmarkBM25L_BatchTransform BM25L 批量向量化基准测试
func BenchmarkBM25L_BatchTransform(b *testing.B) {
	config := DefaultBM25Config()
	v := NewBM25LVectorizer(config)

	docs := generateRandomDocs(500, 50, 200)
	v.Fit(docs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.BatchTransform(docs)
	}
}

// BenchmarkBM25P_Fit BM25+ 训练基准测试
func BenchmarkBM25P_Fit(b *testing.B) {
	config := DefaultBM25Config()
	config.MaxVocabSize = 5000

	docs := generateRandomDocs(1000, 50, 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := NewBM25PlusVectorizer(config)
		v.Fit(docs)
	}
}

// BenchmarkBM25P_Transform BM25+ 向量化基准测试
func BenchmarkBM25P_Transform(b *testing.B) {
	config := DefaultBM25Config()
	v := NewBM25PlusVectorizer(config)

	docs := generateRandomDocs(500, 50, 200)
	v.Fit(docs)

	query := "编程 语言 开发"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Transform(query)
	}
}

// BenchmarkBM25P_BatchTransform BM25+ 批量向量化基准测试
func BenchmarkBM25P_BatchTransform(b *testing.B) {
	config := DefaultBM25Config()
	v := NewBM25PlusVectorizer(config)

	docs := generateRandomDocs(500, 50, 200)
	v.Fit(docs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.BatchTransform(docs)
	}
}
