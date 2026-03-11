package embedding

import (
	"testing"
)

func TestSparseBM25Vectorizer(t *testing.T) {
	config := BM25Config{
		MaxVocabSize: 100,
		K1:           1.5,
		B:            0.75,
	}

	vectorizer := NewSparseBM25Vectorizer(config)
	vectorizer.SetThreshold(0.1)

	documents := []string{
		"Go 编程语言高效",
		"Python 数据分析",
		"Java 企业开发",
	}

	// 测试 Fit
	vectorizer.Fit(documents)
	if vectorizer.GetVocabularySize() == 0 {
		t.Error("词汇表大小应为正数")
	}

	// 测试稀疏向量化
	query := "编程语言"
	sparseVec := vectorizer.TransformToSparse(query)

	if sparseVec == nil {
		t.Fatal("稀疏向量不应为 nil")
	}

	t.Logf("稀疏向量 - 索引：%v, 值：%v", sparseVec.Indices, sparseVec.Values)
	t.Logf("非零元素数量：%d/%d", len(sparseVec.Indices), vectorizer.GetVocabularySize())
}

func TestSparseVectorThreshold(t *testing.T) {
	config := BM25Config{
		MaxVocabSize: 100,
	}

	vectorizer := NewSparseBM25Vectorizer(config)
	documents := []string{"文档内容测试"}

	vectorizer.Fit(documents)

	// 测试不同阈值
	vectorizer.SetThreshold(0.0)
	sparseVec1 := vectorizer.TransformToSparse("文档")
	count1 := len(sparseVec1.Indices)

	vectorizer.SetThreshold(0.5)
	sparseVec2 := vectorizer.TransformToSparse("文档")
	count2 := len(sparseVec2.Indices)

	t.Logf("阈值 0.0: %d 个非零元素", count1)
	t.Logf("阈值 0.5: %d 个非零元素", count2)

	// 高阈值应该产生更少的非零元素
	if count2 > count1 {
		t.Error("高阈值应该产生更少的非零元素")
	}
}

func TestSparseToDense(t *testing.T) {
	config := BM25Config{
		MaxVocabSize: 100,
	}

	vectorizer := NewSparseBM25Vectorizer(config)
	documents := []string{"测试文档"}

	vectorizer.Fit(documents)

	sparseVec := vectorizer.TransformToSparse("测试")
	denseVec := vectorizer.SparseToDense(sparseVec)

	if len(denseVec) != vectorizer.GetVocabularySize() {
		t.Errorf("稠密向量长度错误，期望 %d, 得到 %d", vectorizer.GetVocabularySize(), len(denseVec))
	}

	// 检查非零位置是否一致
	for i, idx := range sparseVec.Indices {
		if denseVec[idx] != sparseVec.Values[i] {
			t.Errorf("索引 %d 的值不匹配", idx)
		}
	}
}

func TestSparseVectorDotProduct(t *testing.T) {
	// 创建两个稀疏向量
	v1 := &SparseVector{
		Indices: []int{0, 2, 5},
		Values:  []float32{1.0, 2.0, 3.0},
	}

	v2 := &SparseVector{
		Indices: []int{0, 2, 7},
		Values:  []float32{4.0, 5.0, 6.0},
	}

	// 计算点积：1*4 + 2*5 = 14
	result := v1.DotProduct(v2)
	expected := float32(14.0)

	if result != expected {
		t.Errorf("点积错误，期望 %.2f, 得到 %.2f", expected, result)
	}
}

func TestSparseBM25Cache(t *testing.T) {
	config := BM25Config{
		MaxVocabSize: 100,
	}

	vectorizer := NewSparseBM25Vectorizer(config)
	documents := []string{"测试文档"}

	vectorizer.Fit(documents)
	vectorizer.EnableCache()

	_ = vectorizer.TransformToSparse("测试文档")

	if vectorizer.GetCacheSize() == 0 {
		t.Error("缓存应该有内容")
	}

	vectorizer.ClearCache()
	if vectorizer.GetCacheSize() > 0 {
		t.Error("缓存应该为空")
	}
}

func TestSparseBM25FitTransform(t *testing.T) {
	config := BM25Config{
		MaxVocabSize: 100,
	}

	vectorizer := NewSparseBM25Vectorizer(config)
	documents := []string{
		"文档一",
		"文档二",
		"文档三",
	}

	vectors := vectorizer.FitTransform(documents)

	if len(vectors) != len(documents) {
		t.Errorf("向量数量错误，期望 %d, 得到 %d", len(documents), len(vectors))
	}

	for i, vec := range vectors {
		if vec == nil {
			t.Errorf("文档 %d 的向量为 nil", i)
		}
	}
}
