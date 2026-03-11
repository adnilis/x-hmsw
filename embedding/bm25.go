package embedding

import (
	"math"
	"sort"
	"sync"
	"sync/atomic"
)

const (
	// BM25 默认参数
	defaultBM25K1 = 1.5  // 词频饱和参数
	defaultBM25B  = 0.75 // 长度归一化参数
)

// BM25Vectorizer BM25向量化器（优化版）
// BM25 是一种改进的 TF-IDF 算法，广泛用于信息检索
// 优化特性：
// - 使用切片代替 map 存储 IDF，提升访问性能
// - 支持查询结果缓存
// - 批量查询并发优化
type BM25Vectorizer struct {
	config       BM25Config
	vocabulary   map[string]int
	idf          []float64 // 使用切片代替 map 提高性能
	docCount     int32
	avgDocLength float64
	docLengths   []int
	mu           sync.RWMutex
	tokenizer    func(string) []string

	// 缓存优化
	cache        map[string][]float32
	cacheMu      sync.RWMutex
	cacheEnabled bool
	cacheSize    int32
	maxCacheSize int32

	// 性能统计
	hits   int64
	misses int64
}

// NewBM25Vectorizer 创建 BM25 向量化器
func NewBM25Vectorizer(config BM25Config) *BM25Vectorizer {
	if config.MaxVocabSize <= 0 {
		config.MaxVocabSize = defaultMaxVocabSize
	}

	if config.MaxVocabSize <= 0 {
		config.MaxVocabSize = defaultMaxVocabSize
	}
	if config.MinDocFreq < 1 {
		config.MinDocFreq = defaultMinDocFreq
	}
	if config.MaxDocFreq <= 0 || config.MaxDocFreq > 1.0 {
		config.MaxDocFreq = defaultMaxDocFreq
	}
	if config.K1 <= 0 {
		config.K1 = defaultBM25K1
	}
	if config.B < 0 || config.B > 1 {
		config.B = defaultBM25B
	}

	return &BM25Vectorizer{
		config:       config,
		vocabulary:   make(map[string]int),
		idf:          make([]float64, 0),
		cache:        make(map[string][]float32),
		cacheEnabled: false,
		maxCacheSize: 10000, // 默认缓存 10000 个查询
		tokenizer:    mixedTokenizer,
	}
}

// Fit 训练 BM25 模型
func (v *BM25Vectorizer) Fit(documents []string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	docCount := len(documents)
	if docCount == 0 {
		return
	}

	// 使用并发统计词频
	wordFreq := make(map[string]int64)
	docFreq := make(map[string]int64)
	totalTokens := int64(0)

	// 简单顺序处理（避免并发 map 写入问题）
	for _, doc := range documents {
		tokens := v.tokenizer(doc)
		seen := make(map[string]bool)

		for _, token := range tokens {
			wordFreq[token]++
			if !seen[token] {
				seen[token] = true
				docFreq[token]++
			}
			totalTokens++
		}
	}

	v.docCount = int32(docCount)

	// 计算平均文档长度
	avgLen := float64(totalTokens) / float64(docCount)
	v.avgDocLength = avgLen

	// 过滤词汇表
	type wordInfo struct {
		word string
		df   int64
	}

	words := make([]wordInfo, 0, len(wordFreq))
	for word, df := range docFreq {
		if df >= int64(v.config.MinDocFreq) &&
			float64(df)/float64(docCount) <= v.config.MaxDocFreq {
			words = append(words, wordInfo{word: word, df: df})
		}
	}

	// 按词频排序
	sort.Slice(words, func(i, j int) bool {
		return wordFreq[words[i].word] > wordFreq[words[j].word]
	})

	// 限制词汇表大小
	maxSize := v.config.MaxVocabSize
	if len(words) > maxSize {
		words = words[:maxSize]
	}

	// 构建词汇表和 IDF
	v.vocabulary = make(map[string]int, len(words))
	v.idf = make([]float64, len(words))

	for i, w := range words {
		v.vocabulary[w.word] = i
		// BM25 IDF 公式
		df := float64(w.df)
		v.idf[i] = math.Log((float64(docCount)-df+0.5)/(df+0.5) + 1)
	}

	// 清空缓存
	v.cacheMu.Lock()
	v.cache = make(map[string][]float32)
	v.cacheMu.Unlock()
	atomic.StoreInt32(&v.cacheSize, 0)
}

// Transform 优化的向量化方法
func (v *BM25Vectorizer) Transform(document string) []float32 {
	// 检查缓存
	if v.cacheEnabled {
		v.cacheMu.RLock()
		if cached, ok := v.cache[document]; ok {
			v.cacheMu.RUnlock()
			atomic.AddInt64(&v.hits, 1)
			return cached
		}
		v.cacheMu.RUnlock()
	}

	v.mu.RLock()
	vocabSize := len(v.vocabulary)
	k1 := v.config.K1
	b := v.config.B
	avgDocLength := v.avgDocLength
	idf := v.idf
	vocab := v.vocabulary
	v.mu.RUnlock()

	if vocabSize == 0 {
		return []float32{}
	}

	// 分词
	tokens := v.tokenizer(document)
	docLength := len(tokens)

	// 计算词频（只统计词汇表中的词）
	tf := make(map[string]int, len(tokens))
	for _, token := range tokens {
		if _, exists := vocab[token]; exists {
			tf[token]++
		}
	}

	// 预计算长度归一化
	lengthNorm := 1.0 - b
	if avgDocLength > 0 {
		lengthNorm += b * float64(docLength) / avgDocLength
	}

	// 创建向量（使用零值初始化）
	vector := make([]float32, vocabSize)

	// 计算 BM25 分数
	for word, count := range tf {
		idx := vocab[word]
		// 内联 BM25 计算
		numerator := float64(count) * (k1 + 1)
		denominator := float64(count) + k1*lengthNorm
		vector[idx] = float32(idf[idx] * numerator / denominator)
	}

	// 缓存结果
	if v.cacheEnabled {
		v.cacheMu.Lock()
		// LRU 简单实现：如果缓存满了，清空
		if atomic.LoadInt32(&v.cacheSize) >= v.maxCacheSize {
			v.cache = make(map[string][]float32)
			atomic.StoreInt32(&v.cacheSize, 0)
		}
		v.cache[document] = vector
		atomic.AddInt32(&v.cacheSize, 1)
		v.cacheMu.Unlock()
		atomic.AddInt64(&v.misses, 1)
	}

	return vector
}

// BatchTransform 批量向量化（并发优化）
func (v *BM25Vectorizer) BatchTransform(documents []string) [][]float32 {
	vectors := make([][]float32, len(documents))

	// 简单并发实现
	n := len(documents)
	chunkSize := (n + 3) / 4 // 4 个 worker
	var wg sync.WaitGroup

	for i := 0; i < 4; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > n {
			end = n
		}
		if start >= n {
			break
		}

		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			for j := s; j < e; j++ {
				vectors[j] = v.Transform(documents[j])
			}
		}(start, end)
	}

	wg.Wait()
	return vectors
}

// GetCacheStats 获取缓存统计
func (v *BM25Vectorizer) GetCacheStats() (hits, misses, size int64) {
	return atomic.LoadInt64(&v.hits),
		atomic.LoadInt64(&v.misses),
		int64(atomic.LoadInt32(&v.cacheSize))
}

// EnableCache 启用缓存
func (v *BM25Vectorizer) EnableCache() {
	v.cacheMu.Lock()
	defer v.cacheMu.Unlock()
	v.cacheEnabled = true
}

// DisableCache 禁用缓存
func (v *BM25Vectorizer) DisableCache() {
	v.cacheMu.Lock()
	defer v.cacheMu.Unlock()
	v.cacheEnabled = false
}

// GetVocabularySize 获取词汇表大小
func (v *BM25Vectorizer) GetVocabularySize() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.vocabulary)
}

// GetAvgDocLength 获取平均文档长度
func (v *BM25Vectorizer) GetAvgDocLength() float64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.avgDocLength
}

// GetDimension 获取向量维度（词汇表大小）
func (v *BM25Vectorizer) GetDimension() int {
	return v.GetVocabularySize()
}

// CreateEmbeddingFunc 创建Embedding函数
func (v *BM25Vectorizer) CreateEmbeddingFunc() EmbeddingFunc {
	return func(text string) ([]float32, error) {
		vector := v.Transform(text)
		if len(vector) == 0 {
			return nil, nil
		}
		return vector, nil
	}
}

// CreateBatchEmbeddingFunc 创建批量Embedding函数
func (v *BM25Vectorizer) CreateBatchEmbeddingFunc() BatchEmbeddingFunc {
	return func(texts []string) ([][]float32, error) {
		vectors := make([][]float32, len(texts))
		for i, text := range texts {
			vectors[i] = v.Transform(text)
		}
		return vectors, nil
	}
}
