package embedding

import (
	"math"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/adnilis/x-hmsw/types"
)

// BM25LVectorizer BM25L 向量化器（优化版）
// BM25L 是改进长度归一化的 BM25 变体
// 优化特性：
// - 使用切片代替 map 存储 IDF，提升访问性能
// - 支持查询结果缓存
type BM25LVectorizer struct {
	config       BM25Config
	vocabulary   map[string]int
	idf          []float64
	docCount     int32
	avgDocLength float64
	docLengths   []int

	mu        sync.RWMutex
	tokenizer func(string) []string

	cache        map[string][]float32
	cacheMu      sync.RWMutex
	cacheEnabled bool
	cacheSize    int32
	maxCacheSize int32

	hits   int64
	misses int64
}

// NewBM25LVectorizer 创建 BM25L 向量化器
func NewBM25LVectorizer(config BM25Config) *BM25LVectorizer {
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
	if config.Variant != types.EmbeddingBM25L {
		config.Variant = types.EmbeddingBM25L
	}

	return &BM25LVectorizer{
		config:       config,
		vocabulary:   make(map[string]int),
		idf:          make([]float64, 0),
		cache:        make(map[string][]float32),
		cacheEnabled: false,
		maxCacheSize: 10000,
		tokenizer:    mixedTokenizer,
	}
}

// Fit 训练（并行优化版）
func (v *BM25LVectorizer) Fit(documents []string) {
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

	type localResult struct {
		wordFreq    map[string]int64
		docFreq     map[string]int64
		docLengths  []int
		totalLength int64
	}

	locals := make([]localResult, numWorkers)
	for i := 0; i < numWorkers; i++ {
		locals[i] = localResult{
			wordFreq:   make(map[string]int64),
			docFreq:    make(map[string]int64),
			docLengths: make([]int, 0),
		}
	}

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
			// 复用 seen map 减少分配
			seen := make(map[string]bool, 64)
			for i := s; i < e; i++ {
				for k := range seen {
					delete(seen, k)
				}
				tokens := v.tokenizer(documents[i])
				local.docLengths = append(local.docLengths, len(tokens))
				local.totalLength += int64(len(tokens))

				for _, token := range tokens {
					local.wordFreq[token]++
					if !seen[token] {
						seen[token] = true
						local.docFreq[token]++
					}
				}
			}
			wg.Done()
		}(start, end)
	}

	wg.Wait()

	// 合并结果
	wordFreq := make(map[string]int64, docCount*10)
	docFreq := make(map[string]int64, docCount*5)
	var totalLength int64
	v.docLengths = make([]int, docCount)

	offset := 0
	for _, local := range locals {
		for word, freq := range local.wordFreq {
			wordFreq[word] += freq
		}
		for word, freq := range local.docFreq {
			docFreq[word] += freq
		}
		totalLength += local.totalLength
		for _, length := range local.docLengths {
			v.docLengths[offset] = length
			offset++
		}
	}

	v.docCount = int32(docCount)
	v.avgDocLength = float64(totalLength) / float64(docCount)

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

	v.vocabulary = make(map[string]int, len(words))
	v.idf = make([]float64, len(words))

	for i, w := range words {
		v.vocabulary[w.word] = i
		df := float64(w.df)
		// BM25L IDF 公式
		v.idf[i] = math.Log(float64(docCount)/df + 1)
	}

	v.cacheMu.Lock()
	v.cache = make(map[string][]float32)
	v.cacheMu.Unlock()
	atomic.StoreInt32(&v.cacheSize, 0)
}

// Transform 向量化
func (v *BM25LVectorizer) Transform(document string) []float32 {
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
	avgDocLength := v.avgDocLength
	idf := v.idf
	vocab := v.vocabulary
	v.mu.RUnlock()

	if vocabSize == 0 {
		return []float32{}
	}

	tokens := v.tokenizer(document)
	docLength := len(tokens)

	tf := make(map[string]int, len(tokens))
	for _, token := range tokens {
		if _, exists := vocab[token]; exists {
			tf[token]++
		}
	}

	// BM25L 长度归一化
	lengthNorm := float64(docLength) / (float64(docLength) + avgDocLength)

	vector := make([]float32, vocabSize)
	for word, count := range tf {
		idx := vocab[word]
		numerator := float64(count) * (k1 + 1)
		denominator := float64(count) + k1*lengthNorm
		vector[idx] = float32(idf[idx] * numerator / denominator)
	}

	if v.cacheEnabled {
		v.cacheMu.Lock()
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

// BatchTransform 批量处理（动态worker优化版）
func (v *BM25LVectorizer) BatchTransform(documents []string) [][]float32 {
	n := len(documents)
	if n == 0 {
		return nil
	}

	// 小批量使用串行
	if n < 100 {
		vectors := make([][]float32, n)
		for i := 0; i < n; i++ {
			vectors[i] = v.Transform(documents[i])
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
				vectors[j] = v.Transform(documents[j])
			}
		}(start, end)
	}

	wg.Wait()
	return vectors
}

// EnableCache 启用缓存
func (v *BM25LVectorizer) EnableCache() {
	v.cacheMu.Lock()
	defer v.cacheMu.Unlock()
	v.cacheEnabled = true
}

// DisableCache 禁用缓存
func (v *BM25LVectorizer) DisableCache() {
	v.cacheMu.Lock()
	defer v.cacheMu.Unlock()
	v.cacheEnabled = false
}

// GetCacheStats 获取统计
func (v *BM25LVectorizer) GetCacheStats() (hits, misses, size int64) {
	return atomic.LoadInt64(&v.hits),
		atomic.LoadInt64(&v.misses),
		int64(atomic.LoadInt32(&v.cacheSize))
}

// GetDimension 获取向量维度
func (v *BM25LVectorizer) GetDimension() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.vocabulary)
}

// CreateEmbeddingFunc 创建Embedding函数
func (v *BM25LVectorizer) CreateEmbeddingFunc() EmbeddingFunc {
	return func(text string) ([]float32, error) {
		vector := v.Transform(text)
		if len(vector) == 0 {
			return nil, nil
		}
		return vector, nil
	}
}

// CreateBatchEmbeddingFunc 创建批量Embedding函数
func (v *BM25LVectorizer) CreateBatchEmbeddingFunc() BatchEmbeddingFunc {
	return func(texts []string) ([][]float32, error) {
		vectors := make([][]float32, len(texts))
		for i, text := range texts {
			vectors[i] = v.Transform(text)
		}
		return vectors, nil
	}
}

// GetVocabularySize 获取词汇表大小
func (v *BM25LVectorizer) GetVocabularySize() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.vocabulary)
}

// GetAvgDocLength 获取平均文档长度
func (v *BM25LVectorizer) GetAvgDocLength() float64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.avgDocLength
}
