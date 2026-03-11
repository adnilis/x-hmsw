package embedding

import (
	"math"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
)

// BM25FConfig BM25F 多字段配置
type BM25FConfig struct {
	BM25Config
	FieldWeights map[string]float64 // 字段权重，如 {"title": 2.0, "content": 1.0}
}

// BM25FVectorizer BM25F 多字段向量化器（优化版）
// BM25F 支持多字段文档，每个字段可以有不同的权重
// 优化特性：
// - 使用切片代替 map 存储 IDF，提升访问性能
// - 支持查询结果缓存
type BM25FVectorizer struct {
	config          BM25FConfig
	vocabulary      map[string]int
	idf             []float64 // 切片优化
	docCount        int32
	avgDocLength    float64
	avgFieldLengths map[string]float64
	fieldDocLengths []map[string]int

	mu        sync.RWMutex
	tokenizer func(string) []string

	// 缓存优化
	cache        map[string][]float32
	cacheMu      sync.RWMutex
	cacheEnabled bool
	cacheSize    int32
	maxCacheSize int32

	// 统计
	hits   int64
	misses int64
}

// NewBM25FVectorizer 创建 BM25F 向量化器
func NewBM25FVectorizer(config BM25FConfig) *BM25FVectorizer {
	if config.MaxVocabSize <= 0 {
		config.MaxVocabSize = defaultMaxVocabSize
	}
	if config.FieldWeights == nil {
		config.FieldWeights = map[string]float64{"title": 2.0, "content": 1.0}
	}

	return &BM25FVectorizer{
		config:          config,
		vocabulary:      make(map[string]int),
		idf:             make([]float64, 0),
		avgFieldLengths: make(map[string]float64),
		cache:           make(map[string][]float32),
		cacheEnabled:    false,
		maxCacheSize:    10000,
		tokenizer:       mixedTokenizer,
	}
}

// Fit 优化的训练方法（并行化版）
func (v *BM25FVectorizer) Fit(documents []map[string]string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	docCount := len(documents)
	if docCount == 0 {
		return
	}

	// 并行化词频统计
	numWorkers := runtime.NumCPU()
	if docCount < 100 {
		numWorkers = 1
	} else if docCount < 1000 {
		numWorkers = 4
	}

	// 每个 worker 的局部结果（预分配内存减少GC压力）
	avgTokensPerDoc := 100 // 估算平均值
	estimatedVocabSize := docCount * avgTokensPerDoc / 2

	type localResult struct {
		wordFreq     map[string]int64
		docFreq      map[string]int64
		fieldLengths map[string]int64
		fieldDocs    map[string]int
		totalTokens  int64
	}

	locals := make([]localResult, numWorkers)
	for i := 0; i < numWorkers; i++ {
		localVocab := estimatedVocabSize / numWorkers
		locals[i] = localResult{
			wordFreq:     make(map[string]int64, localVocab),
			docFreq:      make(map[string]int64, localVocab/2),
			fieldLengths: make(map[string]int64, 8),
			fieldDocs:    make(map[string]int, 8),
		}
	}

	// 预分配文档长度数组
	v.fieldDocLengths = make([]map[string]int, docCount)

	// 并行处理文档
	chunkSize := (docCount + numWorkers - 1) / numWorkers
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > docCount {
			end = docCount
		}
		if start >= docCount {
			break
		}

		localIdx := w
		wg.Add(1)
		go func(s, e int) {
			local := &locals[localIdx]
			for i := s; i < e; i++ {
				doc := documents[i]
				fieldLengthsMap := make(map[string]int)
				for field, text := range doc {
					tokens := v.tokenizer(text)
					fieldLengthsMap[field] = len(tokens)
					local.fieldLengths[field] += int64(len(tokens))
					local.totalTokens += int64(len(tokens))
					local.fieldDocs[field]++

					seen := make(map[string]bool)
					for _, token := range tokens {
						local.wordFreq[token]++
						if !seen[token] {
							seen[token] = true
							local.docFreq[token]++
						}
					}
				}
				v.fieldDocLengths[i] = fieldLengthsMap
			}
			wg.Done()
		}(start, end)
	}

	wg.Wait()

	// 合并所有 worker 的结果
	wordFreq := make(map[string]int64, docCount*10)
	docFreq := make(map[string]int64, docCount*5)
	fieldLengths := make(map[string]int64)
	var totalTokens int64

	for _, local := range locals {
		for word, freq := range local.wordFreq {
			wordFreq[word] += freq
		}
		for word, freq := range local.docFreq {
			docFreq[word] += freq
		}
		for field, length := range local.fieldLengths {
			fieldLengths[field] += length
		}
		totalTokens += local.totalTokens
	}

	v.docCount = int32(docCount)
	v.avgDocLength = float64(totalTokens) / float64(docCount)

	// 计算字段平均长度
	for field, length := range fieldLengths {
		v.avgFieldLengths[field] = float64(length) / float64(docCount)
	}

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

	sort.Slice(words, func(i, j int) bool {
		return wordFreq[words[i].word] > wordFreq[words[j].word]
	})

	maxSize := v.config.MaxVocabSize
	if len(words) > maxSize {
		words = words[:maxSize]
	}

	// 构建词汇表和 IDF
	v.vocabulary = make(map[string]int, len(words))
	v.idf = make([]float64, len(words))

	for i, w := range words {
		v.vocabulary[w.word] = i
		df := float64(w.df)
		v.idf[i] = math.Log((float64(docCount)-df+0.5)/(df+0.5) + 1)
	}

	// 清空缓存
	v.cacheMu.Lock()
	v.cache = make(map[string][]float32)
	v.cacheMu.Unlock()
	atomic.StoreInt32(&v.cacheSize, 0)
}

// Transform 优化的多字段向量化
func (v *BM25FVectorizer) Transform(doc map[string]string) []float32 {
	// 构建缓存键
	cacheKey := ""
	for field, text := range doc {
		cacheKey += field + ":" + text + ";"
	}

	// 检查缓存
	if v.cacheEnabled {
		v.cacheMu.RLock()
		if cached, ok := v.cache[cacheKey]; ok {
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
	fieldWeights := v.config.FieldWeights
	avgFieldLengths := v.avgFieldLengths
	idf := v.idf
	vocab := v.vocabulary
	v.mu.RUnlock()

	if vocabSize == 0 {
		return []float32{}
	}

	// 计算词频（按字段加权）
	tf := make(map[string]float64)

	for field, text := range doc {
		tokens := v.tokenizer(text)
		fieldLen := float64(len(tokens))
		avgLen := avgFieldLengths[field]

		// 字段长度归一化
		lengthNorm := 1.0 - b
		if avgLen > 0 {
			lengthNorm += b * fieldLen / avgLen
		}

		fieldWeight := fieldWeights[field]
		if fieldWeight == 0 {
			fieldWeight = 1.0
		}

		seen := make(map[string]bool)
		for _, token := range tokens {
			if _, exists := vocab[token]; exists {
				if !seen[token] {
					tf[token] = 0
					seen[token] = true
				}
				// BM25 公式 + 字段权重
				count := float64(1)
				numerator := count * (k1 + 1)
				denominator := count + k1*lengthNorm
				tf[token] += fieldWeight * (idf[vocab[token]] * numerator / denominator)
			}
		}
	}

	// 创建向量
	vector := make([]float32, vocabSize)
	for word, score := range tf {
		idx := vocab[word]
		vector[idx] = float32(score)
	}

	// 缓存
	if v.cacheEnabled {
		v.cacheMu.Lock()
		if atomic.LoadInt32(&v.cacheSize) >= v.maxCacheSize {
			v.cache = make(map[string][]float32)
			atomic.StoreInt32(&v.cacheSize, 0)
		}
		v.cache[cacheKey] = vector
		atomic.AddInt32(&v.cacheSize, 1)
		v.cacheMu.Unlock()
		atomic.AddInt64(&v.misses, 1)
	}

	return vector
}

// BatchTransform 批量向量化（动态worker优化版）
func (v *BM25FVectorizer) BatchTransform(docs []map[string]string) [][]float32 {
	n := len(docs)
	if n == 0 {
		return nil
	}

	// 小批量使用串行
	if n < 100 {
		vectors := make([][]float32, n)
		for i := 0; i < n; i++ {
			vectors[i] = v.Transform(docs[i])
		}
		return vectors
	}

	vectors := make([][]float32, n)

	// 动态计算worker数量
	numWorkers := runtime.NumCPU()
	if n < 1000 {
		numWorkers = runtime.NumCPU()
	} else {
		numWorkers = runtime.NumCPU() * 2
	}
	if numWorkers > n {
		numWorkers = n
	}

	chunkSize := (n + numWorkers - 1) / numWorkers
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
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
				vectors[j] = v.Transform(docs[j])
			}
		}(start, end)
	}

	wg.Wait()
	return vectors
}

// BatchTransformStrings 批量向量化（单字符串版本，实现 Vectorizer 接口）
func (v *BM25FVectorizer) BatchTransformStrings(documents []string) [][]float32 {
	vectors := make([][]float32, len(documents))

	n := len(documents)
	chunkSize := (n + 3) / 4
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
				// 将单字符串转换为多字段格式
				doc := map[string]string{"content": documents[j]}
				vectors[j] = v.Transform(doc)
			}
		}(start, end)
	}

	wg.Wait()
	return vectors
}

// EnableCache 启用缓存
func (v *BM25FVectorizer) EnableCache() {
	v.cacheMu.Lock()
	defer v.cacheMu.Unlock()
	v.cacheEnabled = true
}

// DisableCache 禁用缓存
func (v *BM25FVectorizer) DisableCache() {
	v.cacheMu.Lock()
	defer v.cacheMu.Unlock()
	v.cacheEnabled = false
}

// GetCacheStats 获取缓存统计
func (v *BM25FVectorizer) GetCacheStats() (hits, misses, size int64) {
	return atomic.LoadInt64(&v.hits),
		atomic.LoadInt64(&v.misses),
		int64(atomic.LoadInt32(&v.cacheSize))
}

// GetVocabularySize 获取词汇表大小
func (v *BM25FVectorizer) GetVocabularySize() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.vocabulary)
}

// GetAvgDocLength 获取平均文档长度
func (v *BM25FVectorizer) GetAvgDocLength() float64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.avgDocLength
}

// GetDimension 获取向量维度
func (v *BM25FVectorizer) GetDimension() int {
	return v.GetVocabularySize()
}

// CreateEmbeddingFunc 创建Embedding函数
func (v *BM25FVectorizer) CreateEmbeddingFunc() EmbeddingFunc {
	return func(text string) ([]float32, error) {
		// 将单字符串转换为多字段格式
		doc := map[string]string{"content": text}
		vector := v.Transform(doc)
		if len(vector) == 0 {
			return nil, nil
		}
		return vector, nil
	}
}

// CreateBatchEmbeddingFunc 创建批量Embedding函数
func (v *BM25FVectorizer) CreateBatchEmbeddingFunc() BatchEmbeddingFunc {
	return func(texts []string) ([][]float32, error) {
		// 将字符串数组转换为多字段格式
		docs := make([]map[string]string, len(texts))
		for i, text := range texts {
			docs[i] = map[string]string{"content": text}
		}
		return v.BatchTransform(docs), nil
	}
}
