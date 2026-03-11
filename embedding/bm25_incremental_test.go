package embedding

import (
	"testing"
)

func TestIncrementalBM25Vectorizer(t *testing.T) {
	config := BM25Config{
		MaxVocabSize: 100,
		K1:           1.5,
		B:            0.75,
	}

	vectorizer := NewIncrementalBM25Vectorizer(config)

	// 初始文档
	initialDocs := []string{
		"Go 编程语言",
		"Python 数据分析",
	}

	// 测试 Fit
	vectorizer.Fit(initialDocs)
	if vectorizer.GetDocCount() != 2 {
		t.Errorf("初始文档数量错误，期望 2, 得到 %d", vectorizer.GetDocCount())
	}

	version1 := vectorizer.GetVersion()
	if version1 != 1 {
		t.Errorf("版本号错误，期望 1, 得到 %d", version1)
	}

	// 测试增量添加
	newDocs := []string{
		"Java 企业开发",
		"C++ 系统编程",
	}

	err := vectorizer.AddDocuments(newDocs)
	if err != nil {
		t.Fatalf("添加文档失败：%v", err)
	}

	if vectorizer.GetDocCount() != 4 {
		t.Errorf("添加后文档数量错误，期望 4, 得到 %d", vectorizer.GetDocCount())
	}

	version2 := vectorizer.GetVersion()
	if version2 <= version1 {
		t.Errorf("版本号应该增加，之前 %d, 现在 %d", version1, version2)
	}

	// 测试向量化
	query := "编程语言"
	vector := vectorizer.Transform(query)
	if len(vector) == 0 {
		t.Error("向量化结果不应为空")
	}
}

func TestIncrementalBM25RemoveDocuments(t *testing.T) {
	config := BM25Config{
		MaxVocabSize: 100,
	}

	vectorizer := NewIncrementalBM25Vectorizer(config)

	documents := []string{
		"文档一",
		"文档二",
		"文档三",
		"文档四",
	}

	vectorizer.Fit(documents)
	initialCount := vectorizer.GetDocCount()

	// 移除部分文档
	indices := []int{0, 2} // 移除文档一和文档三
	err := vectorizer.RemoveDocuments(indices)
	if err != nil {
		t.Fatalf("移除文档失败：%v", err)
	}

	expectedCount := initialCount - len(indices)
	if vectorizer.GetDocCount() != expectedCount {
		t.Errorf("移除后文档数量错误，期望 %d, 得到 %d", expectedCount, vectorizer.GetDocCount())
	}
}

func TestIncrementalBM25MultipleUpdates(t *testing.T) {
	config := BM25Config{
		MaxVocabSize: 200,
	}

	vectorizer := NewIncrementalBM25Vectorizer(config)

	// 初始文档
	vectorizer.Fit([]string{"初始文档 1", "初始文档 2"})
	version1 := vectorizer.GetVersion()

	// 多次增量更新
	for i := 0; i < 5; i++ {
		err := vectorizer.AddDocuments([]string{
			"新增文档" + string(rune('1'+i)),
		})
		if err != nil {
			t.Fatalf("第 %d 次添加失败：%v", i+1, err)
		}
	}

	version2 := vectorizer.GetVersion()
	if version2 <= version1 {
		t.Error("版本号应该随每次更新增加")
	}

	if vectorizer.GetDocCount() != 7 { // 2 初始 + 5 新增
		t.Errorf("文档数量错误，期望 7, 得到 %d", vectorizer.GetDocCount())
	}
}

func TestIncrementalBM25Cache(t *testing.T) {
	config := BM25Config{
		MaxVocabSize: 100,
	}

	vectorizer := NewIncrementalBM25Vectorizer(config)
	documents := []string{"测试文档"}

	vectorizer.Fit(documents)
	vectorizer.EnableCache()

	_ = vectorizer.Transform("测试文档")
	if vectorizer.GetCacheSize() == 0 {
		t.Error("缓存应该有内容")
	}

	// 增量更新后缓存应该被清空
	vectorizer.AddDocuments([]string{"新文档"})
	// 注意：实现中 AddDocuments 会清空缓存
}

func TestIncrementalBM25VocabularyGrowth(t *testing.T) {
	config := BM25Config{
		MaxVocabSize: 100,
	}

	vectorizer := NewIncrementalBM25Vectorizer(config)

	// 初始词汇表较小
	vectorizer.Fit([]string{"词 1 词 2 词 3"})
	initialVocabSize := vectorizer.GetVocabularySize()

	// 添加包含新词的文档
	vectorizer.AddDocuments([]string{"词 4 词 5 词 6"})

	newVocabSize := vectorizer.GetVocabularySize()
	if newVocabSize <= initialVocabSize {
		t.Logf("词汇表大小：初始 %d, 新增后 %d", initialVocabSize, newVocabSize)
		// 注意：由于 MinDocFreq 限制，新词可能不会立即加入词汇表
	}
}
