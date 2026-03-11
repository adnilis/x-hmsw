package embedding

import (
	"container/heap"
	"math"
	"runtime"
	"sort"
	"sync"
	"time"
)

// SearchResult 搜索结果
type SearchResult struct {
	DocID int
	Score float32
	Doc   string
	Rank  int
}

// PrecomputedVectors 预计算的向量集合
type PrecomputedVectors struct {
	vectors [][]float32
	mu      sync.RWMutex
}

// NewPrecomputedVectors 创建预计算向量容器
func NewPrecomputedVectors(capacity int) *PrecomputedVectors {
	return &PrecomputedVectors{
		vectors: make([][]float32, 0, capacity),
	}
}

// SetVectors 设置预计算的向量
func (p *PrecomputedVectors) SetVectors(vectors [][]float32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.vectors = vectors
}

// Get 获取指定索引的向量
func (p *PrecomputedVectors) Get(idx int) []float32 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if idx >= 0 && idx < len(p.vectors) {
		return p.vectors[idx]
	}
	return nil
}

// Len 返回向量数量
func (p *PrecomputedVectors) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.vectors)
}

// QueryOptimizer 浼樺寲鍚庣殑鏌ヨ浼樺寲鍣?
type QueryOptimizer struct {
	vectorizer  *BM25Vectorizer
	maxResults  int
	minScore    float32
	cache       map[string][]float32 // 鏌ヨ缂撳瓨
	docVecCache map[int][]float32    // 鏂囨。鍚戦噺缂撳瓨
	precomputed *PrecomputedVectors  // 预计算向量（核心优化）
	useCache    bool
	mu          sync.RWMutex
	stats       QueryOptimizerStats
}

// QueryOptimizerStats 浼樺寲鍣ㄧ粺璁′俊鎭?
type QueryOptimizerStats struct {
	HitCount  int64
	MissCount int64
	TotalTime time.Duration
	AvgTime   time.Duration
}

// NewQueryOptimizer 鍒涘缓浼樺寲鐨勬煡璇紭鍖栧櫒
func NewQueryOptimizer(vectorizer *BM25Vectorizer) *QueryOptimizer {
	return &QueryOptimizer{
		vectorizer:  vectorizer,
		maxResults:  100,
		minScore:    0.0,
		cache:       make(map[string][]float32),
		docVecCache: make(map[int][]float32),
		precomputed: NewPrecomputedVectors(0),
		useCache:    true,
	}
}

// PrecomputeDocVectors 预计算所有文档的向量（核心优化）
// 这避免了搜索时重复对每个文档进行向量化
func (o *QueryOptimizer) PrecomputeDocVectors(documents []string) {
	if len(documents) == 0 {
		return
	}
	vectors := o.vectorizer.BatchTransform(documents)
	o.precomputed.SetVectors(vectors)
}

// PrecomputeDocVectorsWithPool 使用对象池预计算向量
func (o *QueryOptimizer) PrecomputeDocVectorsWithPool(documents []string, pool *sync.Pool) {
	if len(documents) == 0 {
		return
	}
	n := len(documents)
	vectors := make([][]float32, n)

	// 使用并发加速
	numWorkers := runtime.NumCPU()
	if n < 100 {
		numWorkers = 1
	}

	chunkSize := (n + numWorkers - 1) / numWorkers
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
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
			for i := s; i < e; i++ {
				vectors[i] = o.vectorizer.Transform(documents[i])
			}
		}(start, end)
	}

	wg.Wait()
	o.precomputed.SetVectors(vectors)
}

// ClearPrecomputedVectors 清除预计算的向量
func (o *QueryOptimizer) ClearPrecomputedVectors() {
	o.precomputed.SetVectors(nil)
}

// SetMaxResults 璁剧疆鏈€澶х粨鏋滄暟
func (o *QueryOptimizer) SetMaxResults(max int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.maxResults = max
}

// EnableCache 鍚敤缂撳瓨
func (o *QueryOptimizer) EnableCache() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.useCache = true
}

// DisableCache 绂佺敤缂撳瓨
func (o *QueryOptimizer) DisableCache() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.useCache = false
}

// ClearCache 娓呯┖缂撳瓨
func (o *QueryOptimizer) ClearCache() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.cache = make(map[string][]float32)
	o.docVecCache = make(map[int][]float32)
}

// GetStats 鑾峰彇缁熻淇℃伅
func (o *QueryOptimizer) GetStats() QueryOptimizerStats {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.stats
}

// TopKSearch 浼樺寲鐨?Top-K 鎼滅储
// 优化：优先使用预计算的向量，避免重复向量化
func (o *QueryOptimizer) TopKSearch(query string, documents []string, k int) []SearchResult {
	start := time.Now()

	o.mu.RLock()
	maxK := o.maxResults
	minScore := o.minScore
	useCache := o.useCache
	precomputed := o.precomputed
	o.mu.RUnlock()

	if k > maxK {
		k = maxK
	}

	// 浠庣紦瀛樿幏鍙栨煡璇㈠悜参阅?
	var queryVec []float32
	if useCache {
		o.mu.RLock()
		if cached, ok := o.cache[query]; ok {
			queryVec = cached
			o.stats.HitCount++
			o.mu.RUnlock()
		} else {
			o.mu.RUnlock()
			queryVec = o.vectorizer.Transform(query)
			o.mu.Lock()
			o.cache[query] = queryVec
			o.stats.MissCount++
			o.mu.Unlock()
		}
	} else {
		queryVec = o.vectorizer.Transform(query)
	}

	if len(queryVec) == 0 {
		return []SearchResult{}
	}

	docCount := len(documents)
	scores := make([]struct {
		id    int
		score float32
	}, docCount)

	// 核心优化：优先使用预计算的向量
	precomputedLen := precomputed.Len()

	if precomputedLen > 0 && precomputedLen == docCount {
		// 使用预计算的向量（最快路径）
		for i := 0; i < docCount; i++ {
			docVec := precomputed.Get(i)
			if docVec == nil || len(docVec) == 0 {
				continue
			}
			score := o.cosineSimilarity(queryVec, docVec)
			scores[i] = struct {
				id    int
				score float32
			}{i, score}
		}
	} else {
		// 回退到原有逻辑
		for i, doc := range documents {
			var docVec []float32

			// 浣跨敤鏂囨。鍚戦噺缂撳瓨
			if useCache {
				o.mu.RLock()
				if cached, ok := o.docVecCache[i]; ok {
					docVec = cached
					o.mu.RUnlock()
				} else {
					o.mu.RUnlock()
					docVec = o.vectorizer.Transform(doc)
					o.mu.Lock()
					o.docVecCache[i] = docVec
					o.mu.Unlock()
				}
			} else {
				docVec = o.vectorizer.Transform(doc)
			}

			if len(docVec) == 0 {
				continue
			}

			score := o.cosineSimilarity(queryVec, docVec)
			scores[i] = struct {
				id    int
				score float32
			}{i, score}
		}
	}

	// 蹇骞€閽夋嫨 Top-K
	o.mu.RLock()
	result := o.quickSelectTopK(scores, k, minScore)
	o.mu.RUnlock()

	o.mu.Lock()
	o.stats.TotalTime += time.Since(start)
	if o.stats.HitCount+o.stats.MissCount > 0 {
		o.stats.AvgTime = o.stats.TotalTime / time.Duration(o.stats.HitCount+o.stats.MissCount)
	}
	o.mu.Unlock()

	return result
}

// quickSelectTopK 浣跨敤蹇€熼€夋嫨绠楁硶鑾峰彇 Top-K
func (o *QueryOptimizer) quickSelectTopK(scores []struct {
	id    int
	score float32
}, k int, minScore float32) []SearchResult {
	// 杩囨护浣庡垎
	valid := make([]struct {
		id    int
		score float32
	}, 0, len(scores))
	for _, s := range scores {
		if s.score >= minScore {
			valid = append(valid, s)
		}
	}

	if len(valid) == 0 {
		return []SearchResult{}
	}

	if k >= len(valid) {
		// 鍏ㄩ儴鎺掑簭
		sort.Slice(valid, func(i, j int) bool {
			return valid[i].score > valid[j].score
		})
		result := make([]SearchResult, len(valid))
		for i, s := range valid {
			result[i] = SearchResult{DocID: s.id, Score: s.score, Rank: i + 1}
		}
		return result
	}

	// 蹇€熼€夋嫨
	left, right := 0, len(valid)-1
	for left < right {
		pivotIdx := partition(valid, left, right)
		if pivotIdx == k-1 {
			break
		} else if pivotIdx < k-1 {
			left = pivotIdx + 1
		} else {
			right = pivotIdx - 1
		}
	}

	// 瀵瑰墠 k 涓帓搴?
	sort.Slice(valid[:k], func(i, j int) bool {
		return valid[i].score > valid[j].score
	})

	result := make([]SearchResult, k)
	for i := 0; i < k; i++ {
		result[i] = SearchResult{DocID: valid[i].id, Score: valid[i].score, Rank: i + 1}
	}
	return result
}

// PrunedSearch 鍓灊鎼滅储 - 鎻愬墠缁堟
func (o *QueryOptimizer) PrunedSearch(query string, documents []string, k int, pruneThreshold float32) []SearchResult {
	start := time.Now()

	o.mu.RLock()
	maxK := o.maxResults
	minScore := o.minScore
	o.mu.RUnlock()

	if k > maxK {
		k = maxK
	}

	queryVec := o.vectorizer.Transform(query)
	if len(queryVec) == 0 {
		return []SearchResult{}
	}

	// 浣跨敤鍫嗙淮鎶?Top-K
	h := &MinHeap{}
	heap.Init(h)

	maxPossibleScore := float32(0)

	for i, doc := range documents {
		docVec := o.vectorizer.Transform(doc)
		if len(docVec) == 0 {
			continue
		}

		score := o.cosineSimilarity(queryVec, docVec)

		if score < minScore {
			continue
		}

		if score > maxPossibleScore {
			maxPossibleScore = score
		}

		// 鍓灊锛氬鏋滃綋鍓嶆渶楂樺垎杩滃ぇ浜庡墿浣欏彲鑳界殑鍒嗘暟锛屾彁鍓嶇粓姝?
		if h.Len() == k && score < float32(maxPossibleScore)*pruneThreshold {
			// 鍙互缁х画浼樺寲锛岃繖閲岀畝鍗曞疄鐜?
		}

		if h.Len() < k {
			heap.Push(h, SearchResult{DocID: i, Score: score})
		} else if score > (*h)[0].Score {
			heap.Pop(h)
			heap.Push(h, SearchResult{DocID: i, Score: score})
		}
	}

	result := make([]SearchResult, h.Len())
	for i := h.Len() - 1; i >= 0; i-- {
		result[i] = heap.Pop(h).(SearchResult)
		result[i].Rank = h.Len() + 1
	}

	o.mu.Lock()
	o.stats.TotalTime += time.Since(start)
	o.mu.Unlock()

	return result
}

// BatchTopKSearch 鎵归噺 Top-K 鎼滅储 - 涓€娆℃€у鐞嗗涓煡璇?
func (o *QueryOptimizer) BatchTopKSearch(queries []string, documents []string, k int) [][]SearchResult {
	results := make([][]SearchResult, len(queries))

	// 鎵归噺鍚戦噺鍖栨煡璇?
	queryVecs := make([][]float32, len(queries))
	for i, query := range queries {
		queryVecs[i] = o.vectorizer.Transform(query)
	}

	// 鎵归噺澶勭悊
	for qIdx, queryVec := range queryVecs {
		scores := make([]struct {
			id    int
			score float32
		}, len(documents))

		for docIdx, doc := range documents {
			docVec := o.vectorizer.Transform(doc)
			if len(docVec) == 0 {
				continue
			}
			scores[docIdx] = struct {
				id    int
				score float32
			}{docIdx, o.cosineSimilarity(queryVec, docVec)}
		}

		o.mu.RLock()
		results[qIdx] = o.quickSelectTopK(scores, k, o.minScore)
		o.mu.RUnlock()
	}

	return results
}

// cosineSimilarity 璁＄畻浣欏鸡鐩镐技搴︼紙鍐呰仈浼樺寲锛?
func (o *QueryOptimizer) cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float32
	var normA float32
	var normB float32

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / float32(math.Sqrt(float64(normA*normB))+1e-8)
}

// partition 蹇€熼€夋嫨鐨勫垎鍖哄嚱鏁?
func partition(arr []struct {
	id    int
	score float32
}, left, right int) int {
	pivot := arr[right].score
	i := left - 1

	for j := left; j < right; j++ {
		if arr[j].score > pivot {
			i++
			arr[i], arr[j] = arr[j], arr[i]
		}
	}

	arr[i+1], arr[right] = arr[right], arr[i+1]
	return i + 1
}

// MinHeap 最小堆实现
type MinHeap []SearchResult

func (h MinHeap) Len() int           { return len(h) }
func (h MinHeap) Less(i, j int) bool { return h[i].Score < h[j].Score }
func (h MinHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *MinHeap) Push(x interface{}) {
	*h = append(*h, x.(SearchResult))
}

func (h *MinHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
