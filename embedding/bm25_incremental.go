package embedding

import (
	"math"
	"sort"
	"sync"
)

// IncrementalBM25Vectorizer 支持增量更新的 BM25 向量化器
type IncrementalBM25Vectorizer struct {
	config         BM25Config
	vocabulary     map[string]int
	idf            map[string]float64
	docCount       int
	avgDocLength   float64
	docLengths     []int
	docTokens      [][]string // 存储每个文档的分词结果，用于增量更新
	mu             sync.RWMutex
	tokenizer      func(string) []string
	cache          map[string][]float32
	cacheMu        sync.RWMutex
	cacheEnabled   bool
	lastFitVersion int64 // 版本号，用于追踪更新
}

// NewIncrementalBM25Vectorizer 创建支持增量更新的 BM25 向量化器
func NewIncrementalBM25Vectorizer(config BM25Config) *IncrementalBM25Vectorizer {
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

	return &IncrementalBM25Vectorizer{
		config:         config,
		vocabulary:     make(map[string]int),
		idf:            make(map[string]float64),
		docCount:       0,
		docLengths:     make([]int, 0),
		docTokens:      make([][]string, 0),
		tokenizer:      mixedTokenizer,
		cache:          make(map[string][]float32),
		cacheEnabled:   false,
		lastFitVersion: 0,
	}
}

// Fit 初始训练 BM25 模型
func (v *IncrementalBM25Vectorizer) Fit(documents []string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	docCount := len(documents)
	if docCount == 0 {
		return
	}

	// 统计词频和文档长度
	docFreq := make(map[string]int)
	wordCount := make(map[string]int)
	docLengths := make([]int, docCount)
	docTokens := make([][]string, docCount)

	for i, doc := range documents {
		tokens := v.tokenizer(doc)
		docLengths[i] = len(tokens)
		docTokens[i] = tokens

		seen := make(map[string]bool)
		for _, token := range tokens {
			wordCount[token]++
			if !seen[token] {
				docFreq[token]++
				seen[token] = true
			}
		}
	}

	v.docCount = docCount
	v.docLengths = docLengths
	v.docTokens = docTokens

	// 计算平均文档长度
	totalLength := 0
	for _, length := range docLengths {
		totalLength += length
	}
	v.avgDocLength = float64(totalLength) / float64(docCount)

	// 过滤词汇表
	type wordInfo struct {
		word  string
		count int
	}

	words := make([]wordInfo, 0, len(wordCount))
	for word, count := range wordCount {
		df := docFreq[word]
		if df >= v.config.MinDocFreq && float64(df)/float64(docCount) <= v.config.MaxDocFreq {
			words = append(words, wordInfo{word: word, count: count})
		}
	}

	// 按词频排序
	sort.Slice(words, func(i, j int) bool {
		return words[i].count > words[j].count
	})

	// 限制词汇表大小
	maxSize := v.config.MaxVocabSize
	if len(words) > maxSize {
		words = words[:maxSize]
	}

	// 构建词汇表
	v.vocabulary = make(map[string]int, len(words))
	for i, w := range words {
		v.vocabulary[w.word] = i
	}

	// 计算 IDF
	v.idf = make(map[string]float64, len(v.vocabulary))
	for word := range v.vocabulary {
		df := docFreq[word]
		v.idf[word] = math.Log((float64(docCount-df)+0.5)/(float64(df)+0.5) + 1)
	}

	// 清空缓存
	v.cacheMu.Lock()
	v.cache = make(map[string][]float32)
	v.cacheMu.Unlock()

	v.lastFitVersion++
}

// AddDocuments 增量添加新文档
func (v *IncrementalBM25Vectorizer) AddDocuments(newDocuments []string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	newCount := len(newDocuments)
	if newCount == 0 {
		return nil
	}

	// 统计新文档的词频和长度
	newDocFreq := make(map[string]int)
	newWordCount := make(map[string]int)
	newDocLengths := make([]int, newCount)
	newDocTokens := make([][]string, newCount)

	for i, doc := range newDocuments {
		tokens := v.tokenizer(doc)
		newDocLengths[i] = len(tokens)
		newDocTokens[i] = tokens

		seen := make(map[string]bool)
		for _, token := range tokens {
			newWordCount[token]++
			if !seen[token] {
				newDocFreq[token]++
				seen[token] = true
			}
		}
	}

	// 更新文档长度列表
	v.docLengths = append(v.docLengths, newDocLengths...)
	v.docTokens = append(v.docTokens, newDocTokens...)

	// 更新文档总数
	oldDocCount := v.docCount
	v.docCount += newCount

	// 更新平均文档长度
	oldTotalLength := int(v.avgDocLength * float64(oldDocCount))
	newTotalLength := oldTotalLength
	for _, length := range newDocLengths {
		newTotalLength += length
	}
	v.avgDocLength = float64(newTotalLength) / float64(v.docCount)

	// 合并词汇表并更新词频统计
	// 注意：这里不重新过滤词汇表，保持已有词汇表稳定
	for word := range newWordCount {
		if _, exists := v.vocabulary[word]; !exists {
			// 新词：检查是否应该加入词汇表
			df := newDocFreq[word]
			if len(v.vocabulary) < v.config.MaxVocabSize &&
				df >= v.config.MinDocFreq &&
				float64(df)/float64(v.docCount) <= v.config.MaxDocFreq {
				// 分配新的索引
				v.vocabulary[word] = len(v.vocabulary)
				// 计算初始 IDF
				v.idf[word] = math.Log((float64(v.docCount-df)+0.5)/(float64(df)+0.5) + 1)
			}
		}
	}

	// 更新所有词的 IDF (因为文档总数变化了)
	for word := range v.idf {
		df := 0
		// 统计该词在所有文档中的出现次数
		for i, tokens := range v.docTokens {
			if i >= oldDocCount {
				// 只统计新增文档
				for _, token := range tokens {
					if token == word {
						df++
						break
					}
				}
			} else {
				// 已有文档需要重新统计
				for _, token := range tokens {
					if token == word {
						df++
						break
					}
				}
			}
		}
		// 重新计算 IDF
		v.idf[word] = math.Log((float64(v.docCount-df)+0.5)/(float64(df)+0.5) + 1)
	}

	// 清空缓存
	v.cacheMu.Lock()
	v.cache = make(map[string][]float32)
	v.cacheMu.Unlock()

	v.lastFitVersion++
	return nil
}

// RemoveDocuments 移除指定索引的文档 (通过标记实现，不实际删除以保持一致性)
func (v *IncrementalBM25Vectorizer) RemoveDocuments(docIndices []int) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if len(docIndices) == 0 {
		return nil
	}

	// 创建要删除的索引集合
	deleteSet := make(map[int]bool)
	for _, idx := range docIndices {
		if idx >= 0 && idx < len(v.docTokens) {
			deleteSet[idx] = true
		}
	}

	// 重新统计
	newDocFreq := make(map[string]int)
	newWordCount := make(map[string]int)
	newDocLengths := make([]int, 0)
	newDocTokens := make([][]string, 0)

	for i, tokens := range v.docTokens {
		if !deleteSet[i] {
			newDocLengths = append(newDocLengths, len(tokens))
			newDocTokens = append(newDocTokens, tokens)

			seen := make(map[string]bool)
			for _, token := range tokens {
				newWordCount[token]++
				if !seen[token] {
					newDocFreq[token]++
					seen[token] = true
				}
			}
		}
	}

	// 更新状态
	v.docCount = len(newDocTokens)
	v.docLengths = newDocLengths
	v.docTokens = newDocTokens

	// 更新平均文档长度
	totalLength := 0
	for _, length := range newDocLengths {
		totalLength += length
	}
	if v.docCount > 0 {
		v.avgDocLength = float64(totalLength) / float64(v.docCount)
	}

	// 更新 IDF
	for word := range v.idf {
		df := newDocFreq[word]
		if df > 0 {
			v.idf[word] = math.Log((float64(v.docCount-df)+0.5)/(float64(df)+0.5) + 1)
		}
	}

	// 清空缓存
	v.cacheMu.Lock()
	v.cache = make(map[string][]float32)
	v.cacheMu.Unlock()

	v.lastFitVersion++
	return nil
}

// Transform 转换文档
func (v *IncrementalBM25Vectorizer) Transform(document string) []float32 {
	v.mu.RLock()
	vocabSize := len(v.vocabulary)
	v.mu.RUnlock()

	if vocabSize == 0 {
		return []float32{}
	}

	// 检查缓存
	if v.cacheEnabled {
		v.cacheMu.RLock()
		if cached, ok := v.cache[document]; ok {
			v.cacheMu.RUnlock()
			return cached
		}
		v.cacheMu.RUnlock()
	}

	vector := make([]float32, vocabSize)
	tokens := v.tokenizer(document)
	docLength := len(tokens)

	// 计算词频
	tf := make(map[string]int)
	for _, token := range tokens {
		v.mu.RLock()
		_, exists := v.vocabulary[token]
		v.mu.RUnlock()
		if exists {
			tf[token]++
		}
	}

	// 获取配置参数
	v.mu.RLock()
	k1 := v.config.K1
	b := v.config.B
	avgDocLength := v.avgDocLength
	v.mu.RUnlock()

	// 预计算长度归一化因子
	lengthNorm := 1.0 - b
	if avgDocLength > 0 {
		lengthNorm += b * float64(docLength) / avgDocLength
	}

	// 计算 BM25 分数
	v.mu.RLock()
	for word, count := range tf {
		idx, exists := v.vocabulary[word]
		if exists {
			numerator := float64(count) * (k1 + 1)
			denominator := float64(count) + k1*lengthNorm
			bm25Score := v.idf[word] * numerator / denominator
			vector[idx] = float32(bm25Score)
		}
	}
	v.mu.RUnlock()

	// 缓存结果
	if v.cacheEnabled {
		v.cacheMu.Lock()
		v.cache[document] = vector
		v.cacheMu.Unlock()
	}

	return vector
}

// GetVersion 获取当前版本号
func (v *IncrementalBM25Vectorizer) GetVersion() int64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.lastFitVersion
}

// GetDocCount 获取文档数量
func (v *IncrementalBM25Vectorizer) GetDocCount() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.docCount
}

// FitTransform 训练并转换文档
func (v *IncrementalBM25Vectorizer) FitTransform(documents []string) [][]float32 {
	v.Fit(documents)
	vectors := make([][]float32, len(documents))
	for i, doc := range documents {
		vectors[i] = v.Transform(doc)
	}
	return vectors
}

// GetVocabularySize 获取词汇表大小
func (v *IncrementalBM25Vectorizer) GetVocabularySize() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.vocabulary)
}

// GetDimension 获取向量维度
func (v *IncrementalBM25Vectorizer) GetDimension() int {
	return v.GetVocabularySize()
}

// EnableCache 启用缓存
func (v *IncrementalBM25Vectorizer) EnableCache() {
	v.cacheMu.Lock()
	defer v.cacheMu.Unlock()
	v.cacheEnabled = true
}

// DisableCache 禁用缓存
func (v *IncrementalBM25Vectorizer) DisableCache() {
	v.cacheMu.Lock()
	defer v.cacheMu.Unlock()
	v.cacheEnabled = false
	v.cache = make(map[string][]float32)
}

// ClearCache 清空缓存
func (v *IncrementalBM25Vectorizer) ClearCache() {
	v.cacheMu.Lock()
	defer v.cacheMu.Unlock()
	v.cache = make(map[string][]float32)
}

// GetCacheSize 获取缓存大小
func (v *IncrementalBM25Vectorizer) GetCacheSize() int {
	v.cacheMu.RLock()
	defer v.cacheMu.RUnlock()
	return len(v.cache)
}

// IsCacheEnabled 检查缓存是否启用
func (v *IncrementalBM25Vectorizer) IsCacheEnabled() bool {
	v.cacheMu.RLock()
	defer v.cacheMu.RUnlock()
	return v.cacheEnabled
}
