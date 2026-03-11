package embedding

import (
	"math"
	"testing"
)

func TestNewTFIDFVectorizer(t *testing.T) {
	config := DefaultTFIDFConfig()
	vectorizer := NewTFIDFVectorizer(config)

	if vectorizer == nil {
		t.Fatal("NewTFIDFVectorizer returned nil")
	}

	if vectorizer.config.MaxVocabSize != 10000 {
		t.Errorf("Expected MaxVocabSize 10000, got %d", vectorizer.config.MaxVocabSize)
	}

	if vectorizer.config.MinDocFreq != 1 {
		t.Errorf("Expected MinDocFreq 1, got %d", vectorizer.config.MinDocFreq)
	}

	if vectorizer.config.MaxDocFreq != 1.0 {
		t.Errorf("Expected MaxDocFreq 1.0, got %f", vectorizer.config.MaxDocFreq)
	}

	if !vectorizer.config.Normalize {
		t.Error("Expected Normalize to be true")
	}
}

func TestTFIDFVectorizer_Fit(t *testing.T) {
	config := DefaultTFIDFConfig()
	vectorizer := NewTFIDFVectorizer(config)

	documents := []string{
		"hello world",
		"hello there",
		"world of code",
	}

	vectorizer.Fit(documents)

	if vectorizer.docCount != 3 {
		t.Errorf("Expected docCount 3, got %d", vectorizer.docCount)
	}

	if len(vectorizer.vocabulary) == 0 {
		t.Error("Expected non-empty vocabulary")
	}

	if len(vectorizer.idf) == 0 {
		t.Error("Expected non-empty idf")
	}
}

func TestTFIDFVectorizer_Transform(t *testing.T) {
	config := DefaultTFIDFConfig()
	vectorizer := NewTFIDFVectorizer(config)

	documents := []string{
		"hello world",
		"hello there",
		"world of code",
	}

	vectorizer.Fit(documents)

	// 测试转换
	vector := vectorizer.Transform("hello world")

	if len(vector) == 0 {
		t.Error("Expected non-empty vector")
	}

	// 测试归一化
	if vectorizer.config.Normalize {
		norm := float32(0)
		for _, val := range vector {
			norm += val * val
		}
		norm = float32(math.Sqrt(float64(norm)))
		if norm > 1.0001 || norm < 0.9999 {
			t.Errorf("Expected normalized vector (norm ~1.0), got %f", norm)
		}
	}
}

func TestTFIDFVectorizer_FitTransform(t *testing.T) {
	config := DefaultTFIDFConfig()
	vectorizer := NewTFIDFVectorizer(config)

	documents := []string{
		"hello world",
		"hello there",
		"world of code",
	}

	vectors := vectorizer.FitTransform(documents)

	if len(vectors) != 3 {
		t.Errorf("Expected 3 vectors, got %d", len(vectors))
	}

	for i, vector := range vectors {
		if len(vector) == 0 {
			t.Errorf("Expected non-empty vector at index %d", i)
		}
	}

	// 所有向量应该有相同的维度
	dim := len(vectors[0])
	for i, vector := range vectors {
		if len(vector) != dim {
			t.Errorf("Vector at index %d has dimension %d, expected %d", i, len(vector), dim)
		}
	}
}

func TestTFIDFVectorizer_CreateEmbeddingFunc(t *testing.T) {
	config := DefaultTFIDFConfig()
	vectorizer := NewTFIDFVectorizer(config)

	documents := []string{
		"hello world",
		"hello there",
		"world of code",
	}

	vectorizer.Fit(documents)

	embeddingFunc := vectorizer.CreateEmbeddingFunc()

	vector, err := embeddingFunc("hello world")
	if err != nil {
		t.Errorf("CreateEmbeddingFunc returned error: %v", err)
	}

	if len(vector) == 0 {
		t.Error("Expected non-empty vector")
	}
}

func TestTFIDFVectorizer_CreateBatchEmbeddingFunc(t *testing.T) {
	config := DefaultTFIDFConfig()
	vectorizer := NewTFIDFVectorizer(config)

	documents := []string{
		"hello world",
		"hello there",
		"world of code",
	}

	vectorizer.Fit(documents)

	batchFunc := vectorizer.CreateBatchEmbeddingFunc()

	texts := []string{"hello world", "hello there"}
	vectors, err := batchFunc(texts)
	if err != nil {
		t.Errorf("CreateBatchEmbeddingFunc returned error: %v", err)
	}

	if len(vectors) != 2 {
		t.Errorf("Expected 2 vectors, got %d", len(vectors))
	}

	for i, vector := range vectors {
		if len(vector) == 0 {
			t.Errorf("Expected non-empty vector at index %d", i)
		}
	}
}

func TestTFIDFVectorizer_GetVocabularySize(t *testing.T) {
	config := DefaultTFIDFConfig()
	vectorizer := NewTFIDFVectorizer(config)

	documents := []string{
		"hello world",
		"hello there",
		"world of code",
	}

	vectorizer.Fit(documents)

	size := vectorizer.GetVocabularySize()
	if size == 0 {
		t.Error("Expected non-zero vocabulary size")
	}
}

func TestTFIDFVectorizer_GetDimension(t *testing.T) {
	config := DefaultTFIDFConfig()
	vectorizer := NewTFIDFVectorizer(config)

	documents := []string{
		"hello world",
		"hello there",
		"world of code",
	}

	vectorizer.Fit(documents)

	dim := vectorizer.GetDimension()
	if dim == 0 {
		t.Error("Expected non-zero dimension")
	}

	if dim != vectorizer.GetVocabularySize() {
		t.Errorf("Dimension %d should equal vocabulary size %d", dim, vectorizer.GetVocabularySize())
	}
}

func TestTFIDFVectorizer_SetTokenizer(t *testing.T) {
	config := DefaultTFIDFConfig()
	vectorizer := NewTFIDFVectorizer(config)

	// 设置自定义分词器
	vectorizer.SetTokenizer(chineseTokenizer)

	documents := []string{
		"你好世界",
		"你好代码",
		"世界代码",
	}

	vectorizer.Fit(documents)

	if vectorizer.docCount != 3 {
		t.Errorf("Expected docCount 3, got %d", vectorizer.docCount)
	}

	vector := vectorizer.Transform("你好世界")
	if len(vector) == 0 {
		t.Error("Expected non-empty vector for Chinese text")
	}
}

func TestTFIDFVectorizer_ChineseTokenizer(t *testing.T) {
	config := DefaultTFIDFConfig()
	vectorizer := NewTFIDFVectorizer(config)

	vectorizer.SetTokenizer(chineseTokenizer)

	documents := []string{
		"你好世界",
		"你好代码",
		"世界代码",
	}

	vectorizer.Fit(documents)

	vector := vectorizer.Transform("你好世界")
	if len(vector) == 0 {
		t.Error("Expected non-empty vector for Chinese text")
	}
}

func TestTFIDFVectorizer_MixedTokenizer(t *testing.T) {
	config := DefaultTFIDFConfig()
	vectorizer := NewTFIDFVectorizer(config)

	vectorizer.SetTokenizer(mixedTokenizer)

	documents := []string{
		"hello world 你好世界",
		"hello there 你好代码",
		"world of code 世界代码",
	}

	vectorizer.Fit(documents)

	vector := vectorizer.Transform("hello world 你好世界")
	if len(vector) == 0 {
		t.Error("Expected non-empty vector for mixed text")
	}
}

func TestTFIDFVectorizer_StopWordFiltering(t *testing.T) {
	config := DefaultTFIDFConfig()
	vectorizer := NewTFIDFVectorizer(config)

	documents := []string{
		"the quick brown fox",
		"jumps over the lazy dog",
		"the dog is lazy",
	}

	vectorizer.Fit(documents)

	// 检查停用词是否被过滤
	if _, exists := vectorizer.vocabulary["the"]; exists {
		t.Error("Stop word 'the' should be filtered out")
	}

	if _, exists := vectorizer.vocabulary["is"]; exists {
		t.Error("Stop word 'is' should be filtered out")
	}

	// 检查有效词是否保留
	if _, exists := vectorizer.vocabulary["quick"]; !exists {
		t.Error("Word 'quick' should be in vocabulary")
	}

	if _, exists := vectorizer.vocabulary["fox"]; !exists {
		t.Error("Word 'fox' should be in vocabulary")
	}
}

func TestTFIDFVectorizer_VocabSizeLimit(t *testing.T) {
	config := TFIDFConfig{
		MaxVocabSize: 5,
		MinDocFreq:   1,
		MaxDocFreq:   1.0,
		Normalize:    true,
	}

	vectorizer := NewTFIDFVectorizer(config)

	// 创建超过词汇表限制的文档
	documents := []string{
		"word1 word2 word3 word4 word5 word6",
		"word7 word8 word9 word10 word11 word12",
		"word13 word14 word15 word16 word17 word18",
	}

	vectorizer.Fit(documents)

	size := vectorizer.GetVocabularySize()
	if size > 5 {
		t.Errorf("Expected vocabulary size <= 5, got %d", size)
	}
}

func TestTFIDFVectorizer_DocFreqFiltering(t *testing.T) {
	config := TFIDFConfig{
		MaxVocabSize: 10000,
		MinDocFreq:   2, // 至少出现在2个文档中
		MaxDocFreq:   1.0,
		Normalize:    true,
	}

	vectorizer := NewTFIDFVectorizer(config)

	documents := []string{
		"common word1 word2",
		"common word3 word4",
		"rare word5 word6",
	}

	vectorizer.Fit(documents)

	// "common" 出现在2个文档中，应该保留
	if _, exists := vectorizer.vocabulary["common"]; !exists {
		t.Error("Word 'common' should be in vocabulary (appears in 2 docs)")
	}

	// "rare" 只出现在1个文档中，应该被过滤
	if _, exists := vectorizer.vocabulary["rare"]; exists {
		t.Error("Word 'rare' should be filtered out (appears in only 1 doc)")
	}
}

func TestTFIDFVectorizer_NoNormalization(t *testing.T) {
	config := TFIDFConfig{
		MaxVocabSize: 10000,
		MinDocFreq:   1,
		MaxDocFreq:   1.0,
		Normalize:    false, // 不归一化
	}

	vectorizer := NewTFIDFVectorizer(config)

	documents := []string{
		"hello world",
		"hello there",
		"world of code",
	}

	vectorizer.Fit(documents)

	vector := vectorizer.Transform("hello world")

	// 检查向量是否未归一化
	norm := float32(0)
	for _, val := range vector {
		norm += val * val
	}
	norm = float32(math.Sqrt(float64(norm)))

	// 未归一化的向量范数应该不等于1
	if norm == 1.0 {
		t.Error("Expected non-normalized vector (norm != 1.0)")
	}
}

func TestTFIDFVectorizer_EmptyDocument(t *testing.T) {
	config := DefaultTFIDFConfig()
	vectorizer := NewTFIDFVectorizer(config)

	documents := []string{
		"hello world",
		"hello there",
	}

	vectorizer.Fit(documents)

	// 转换空文档
	vector := vectorizer.Transform("")
	if len(vector) != vectorizer.GetDimension() {
		t.Errorf("Expected vector dimension %d, got %d", vectorizer.GetDimension(), len(vector))
	}

	// 空文档的向量应该全为0
	for _, val := range vector {
		if val != 0 {
			t.Error("Expected zero vector for empty document")
		}
	}
}

func TestTFIDFVectorizer_EmptyDocuments(t *testing.T) {
	config := DefaultTFIDFConfig()
	vectorizer := NewTFIDFVectorizer(config)

	// 训练空文档列表
	vectorizer.Fit([]string{})

	if vectorizer.docCount != 0 {
		t.Errorf("Expected docCount 0, got %d", vectorizer.docCount)
	}

	if len(vectorizer.vocabulary) != 0 {
		t.Errorf("Expected empty vocabulary, got %d words", len(vectorizer.vocabulary))
	}
}

func TestTFIDFVectorizer_Similarity(t *testing.T) {
	config := DefaultTFIDFConfig()
	vectorizer := NewTFIDFVectorizer(config)

	documents := []string{
		"machine learning is awesome",
		"deep learning is powerful",
		"cooking recipes are great",
	}

	vectorizer.Fit(documents)

	// 相似的文档应该有更高的相似度
	vec1 := vectorizer.Transform("machine learning")
	vec2 := vectorizer.Transform("deep learning")
	vec3 := vectorizer.Transform("cooking recipes")

	sim12 := cosineSimilarity(vec1, vec2)
	sim13 := cosineSimilarity(vec1, vec3)

	if sim12 <= sim13 {
		t.Errorf("Expected sim12 (%f) > sim13 (%f)", sim12, sim13)
	}
}

// cosineSimilarity 计算余弦相似度
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

	return dot / float32(math.Sqrt(float64(normA))*math.Sqrt(float64(normB)))
}
