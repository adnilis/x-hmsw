package embedding

import (
	"math"
	"sort"
	"sync"
)

// SparseVector 绋€鐤忓悜閲忚〃绀?
type SparseVector struct {
	Indices []int     // 闈為浂鍏冪礌鐨勭储寮?
	Values  []float32 // 瀵瑰簲鐨勫€?
}

// SparseBM25Vectorizer 鏀寔绋€鐤忓悜閲忎紭鍖栫殑 BM25 鍚戦噺鍖栧櫒
type SparseBM25Vectorizer struct {
	config       BM25Config
	vocabulary   map[string]int
	idf          map[string]float64
	docCount     int
	avgDocLength float64
	docLengths   []int
	mu           sync.RWMutex
	tokenizer    func(string) []string
	sparseCache  map[string]*SparseVector
	cacheMu      sync.RWMutex
	cacheEnabled bool
	threshold    float32 // 闃堝€硷紝浣庝簬姝ゅ€肩殑鍏冪礌涓嶅瓨鍌?
}

// NewSparseBM25Vectorizer 鍒涘缓绋€鐤忓悜閲?BM25 鍚戦噺鍖栧櫒
func NewSparseBM25Vectorizer(config BM25Config) *SparseBM25Vectorizer {
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

	return &SparseBM25Vectorizer{
		config:       config,
		vocabulary:   make(map[string]int),
		idf:          make(map[string]float64),
		docCount:     0,
		docLengths:   make([]int, 0),
		tokenizer:    mixedTokenizer,
		sparseCache:  make(map[string]*SparseVector),
		cacheEnabled: false,
		threshold:    0.0, // 榛樿瀛樺偍鎵€鏈夐潪闆跺€?
	}
}

// SetThreshold 璁剧疆绋€鐤忓寲闃堝€?
func (v *SparseBM25Vectorizer) SetThreshold(threshold float32) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.threshold = threshold
}

// Fit 璁粌 BM25 妯″瀷
func (v *SparseBM25Vectorizer) Fit(documents []string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	docCount := len(documents)
	if docCount == 0 {
		return
	}

	// 缁熻璇嶉鍜屾枃妗ｉ暱搴?
	docFreq := make(map[string]int)
	wordCount := make(map[string]int)
	docLengths := make([]int, docCount)

	for i, doc := range documents {
		tokens := v.tokenizer(doc)
		docLengths[i] = len(tokens)

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

	// 璁＄畻骞冲潎鏂囨。闀垮害
	totalLength := 0
	for _, length := range docLengths {
		totalLength += length
	}
	v.avgDocLength = float64(totalLength) / float64(docCount)

	// 杩囨护璇嶆眹琛?
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

	sort.Slice(words, func(i, j int) bool {
		return words[i].count > words[j].count
	})

	maxSize := v.config.MaxVocabSize
	if len(words) > maxSize {
		words = words[:maxSize]
	}

	// 鏋勫缓璇嶆眹琛?
	v.vocabulary = make(map[string]int, len(words))
	for i, w := range words {
		v.vocabulary[w.word] = i
	}

	// 璁＄畻 IDF
	v.idf = make(map[string]float64, len(v.vocabulary))
	for word := range v.vocabulary {
		df := docFreq[word]
		v.idf[word] = math.Log((float64(docCount-df)+0.5)/(float64(df)+0.5) + 1)
	}

	// 娓呯┖缂撳瓨
	v.cacheMu.Lock()
	v.sparseCache = make(map[string]*SparseVector)
	v.cacheMu.Unlock()
}

// TransformToSparse 灏嗘枃妗ｈ浆鎹负绋€鐤忓悜閲?
func (v *SparseBM25Vectorizer) TransformToSparse(document string) *SparseVector {
	v.mu.RLock()
	vocabSize := len(v.vocabulary)
	v.mu.RUnlock()

	if vocabSize == 0 {
		return &SparseVector{Indices: []int{}, Values: []float32{}}
	}

	// 妫€鏌ョ紦瀛?
	if v.cacheEnabled {
		v.cacheMu.RLock()
		if cached, ok := v.sparseCache[document]; ok {
			v.cacheMu.RUnlock()
			return cached
		}
		v.cacheMu.RUnlock()
	}

	tokens := v.tokenizer(document)
	docLength := len(tokens)

	// 璁＄畻璇嶉
	tf := make(map[string]int)
	for _, token := range tokens {
		v.mu.RLock()
		_, exists := v.vocabulary[token]
		v.mu.RUnlock()
		if exists {
			tf[token]++
		}
	}

	// 鑾峰彇閰嶇疆鍙傛暟
	v.mu.RLock()
	k1 := v.config.K1
	b := v.config.B
	avgDocLength := v.avgDocLength
	threshold := v.threshold
	v.mu.RUnlock()

	// 棰勮绠楅暱搴﹀綊涓€鍖栧洜瀛?
	lengthNorm := 1.0 - b
	if avgDocLength > 0 {
		lengthNorm += b * float64(docLength) / avgDocLength
	}

	// 璁＄畻 BM25 鍒嗘暟骞舵瀯寤虹█鐤忓悜閲?
	indices := make([]int, 0, len(tf))
	values := make([]float32, 0, len(tf))

	v.mu.RLock()
	for word, count := range tf {
		idx, exists := v.vocabulary[word]
		if exists {
			numerator := float64(count) * (k1 + 1)
			denominator := float64(count) + k1*lengthNorm
			bm25Score := float32(v.idf[word] * numerator / denominator)

			// 鍙瓨鍌ㄨ秴杩囬槇鍊肩殑鍊?
			if bm25Score >= threshold {
				indices = append(indices, idx)
				values = append(values, bm25Score)
			}
		}
	}
	v.mu.RUnlock()

	// 鎸夌储寮曟帓搴?
	sparseVec := &SparseVector{
		Indices: indices,
		Values:  values,
	}

	// 缂撳瓨缁撴灉
	if v.cacheEnabled {
		v.cacheMu.Lock()
		v.sparseCache[document] = sparseVec
		v.cacheMu.Unlock()
	}

	return sparseVec
}

// SparseToDense 灏嗙█鐤忓悜閲忚浆鎹负绋犲瘑鍚戦噺
func (v *SparseBM25Vectorizer) SparseToDense(sparse *SparseVector) []float32 {
	v.mu.RLock()
	vocabSize := len(v.vocabulary)
	v.mu.RUnlock()

	if vocabSize == 0 {
		return []float32{}
	}

	dense := make([]float32, vocabSize)
	for i, idx := range sparse.Indices {
		if idx < vocabSize {
			dense[idx] = sparse.Values[i]
		}
	}
	return dense
}

// DotProduct 璁＄畻涓や釜绋€鐤忓悜閲忕殑鐐圭Н
func (v1 *SparseVector) DotProduct(v2 *SparseVector) float32 {
	result := float32(0.0)
	i, j := 0, 0

	for i < len(v1.Indices) && j < len(v2.Indices) {
		if v1.Indices[i] == v2.Indices[j] {
			result += v1.Values[i] * v2.Values[j]
			i++
			j++
		} else if v1.Indices[i] < v2.Indices[j] {
			i++
		} else {
			j++
		}
	}

	return result
}

// FitTransform 璁粌骞惰浆鎹㈡枃妗?
func (v *SparseBM25Vectorizer) FitTransform(documents []string) []*SparseVector {
	v.Fit(documents)
	vectors := make([]*SparseVector, len(documents))
	for i, doc := range documents {
		vectors[i] = v.TransformToSparse(doc)
	}
	return vectors
}

// GetVocabularySize 鑾峰彇璇嶆眹琛ㄥぇ灏?
func (v *SparseBM25Vectorizer) GetVocabularySize() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.vocabulary)
}

// GetDimension 鑾峰彇鍚戦噺缁村害
func (v *SparseBM25Vectorizer) GetDimension() int {
	return v.GetVocabularySize()
}

// EnableCache 鍚敤缂撳瓨
func (v *SparseBM25Vectorizer) EnableCache() {
	v.cacheMu.Lock()
	defer v.cacheMu.Unlock()
	v.cacheEnabled = true
}

// DisableCache 绂佺敤缂撳瓨
func (v *SparseBM25Vectorizer) DisableCache() {
	v.cacheMu.Lock()
	defer v.cacheMu.Unlock()
	v.cacheEnabled = false
	v.sparseCache = make(map[string]*SparseVector)
}

// ClearCache 娓呯┖缂撳瓨
func (v *SparseBM25Vectorizer) ClearCache() {
	v.cacheMu.Lock()
	defer v.cacheMu.Unlock()
	v.sparseCache = make(map[string]*SparseVector)
}

// GetCacheSize 鑾峰彇缂撳瓨澶у皬
func (v *SparseBM25Vectorizer) GetCacheSize() int {
	v.cacheMu.RLock()
	defer v.cacheMu.RUnlock()
	return len(v.sparseCache)
}

// IsCacheEnabled 妫€鏌ョ紦瀛樻槸鍚﹀惎鐢?
func (v *SparseBM25Vectorizer) IsCacheEnabled() bool {
	v.cacheMu.RLock()
	defer v.cacheMu.RUnlock()
	return v.cacheEnabled
}

