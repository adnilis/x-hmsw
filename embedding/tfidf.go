package embedding

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/adnilis/x-hmsw/utils/stopwords"
)

const (
	// 默认词汇表大小
	defaultMaxVocabSize = 10000
	// 最小文档频率
	defaultMinDocFreq = 1
	// 最大文档频率比例
	defaultMaxDocFreq = 1.0
	// TF 计算的平滑因子
	tfSmoothFactor = 0.5
	// 最小词长度
	minWordLength = 2
)

// TFIDFVectorizer TF-IDF向量化器
type TFIDFVectorizer struct {
	config     TFIDFConfig
	vocabulary map[string]int        // 词汇表：词 -> 索引
	idf        map[string]float64    // 逆文档频率
	docCount   int                   // 文档总数
	mu         sync.RWMutex          // 读写锁
	tokenizer  func(string) []string // 分词函数
}

// NewTFIDFVectorizer 创建TF-IDF向量化器
func NewTFIDFVectorizer(config TFIDFConfig) *TFIDFVectorizer {
	if config.MaxVocabSize <= 0 {
		config.MaxVocabSize = defaultMaxVocabSize
	}
	if config.MinDocFreq < 1 {
		config.MinDocFreq = defaultMinDocFreq
	}
	if config.MaxDocFreq <= 0 || config.MaxDocFreq > 1.0 {
		config.MaxDocFreq = defaultMaxDocFreq
	}

	return &TFIDFVectorizer{
		config:     config,
		vocabulary: make(map[string]int),
		idf:        make(map[string]float64),
		docCount:   0,
		tokenizer:  mixedTokenizer, // 使用混合分词器支持中英文
	}
}

// defaultTokenizer 默认分词函数（英文）
func defaultTokenizer(text string) []string {
	// 转换为小写
	text = strings.ToLower(text)

	// 移除标点符号
	reg := regexp.MustCompile(`[^\p{L}\p{N}\s]`)
	text = reg.ReplaceAllString(text, " ")

	// 分词
	words := strings.Fields(text)

	// 预分配切片容量
	tokens := make([]string, 0, len(words)/2)

	// 过滤空词和短词
	for _, word := range words {
		if len(word) > minWordLength && !isStopWord(word) {
			tokens = append(tokens, word)
		}
	}

	return tokens
}

// isStopWord 检查是否为停用词（支持中英文）
func isStopWord(word string) bool {
	// 检查中文停用词
	if stopwords.IsStopWord("zh", word) {
		return true
	}
	// 检查英文停用词
	if stopwords.IsStopWord("en", word) {
		return true
	}
	return false
}

// Fit 训练TF-IDF模型（构建词汇表和IDF）
func (v *TFIDFVectorizer) Fit(documents []string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	// 统计词频
	docFreq := make(map[string]int)   // 词的文档频率
	wordCount := make(map[string]int) // 词的总出现次数

	for _, doc := range documents {
		tokens := v.tokenizer(doc)
		seen := make(map[string]bool) // 当前文档中已见过的词

		for _, token := range tokens {
			wordCount[token]++
			if !seen[token] {
				docFreq[token]++
				seen[token] = true
			}
		}
	}

	v.docCount = len(documents)

	// 过滤词汇表
	type wordInfo struct {
		word  string
		count int
	}

	var words []wordInfo
	for word, count := range wordCount {
		df := docFreq[word]
		// 应用文档频率过滤
		if df >= v.config.MinDocFreq && float64(df)/float64(v.docCount) <= v.config.MaxDocFreq {
			words = append(words, wordInfo{word: word, count: count})
		}
	}

	// 按词频排序，选择高频词
	sort.Slice(words, func(i, j int) bool {
		return words[i].count > words[j].count
	})

	// 限制词汇表大小
	maxSize := v.config.MaxVocabSize
	if len(words) > maxSize {
		words = words[:maxSize]
	}

	// 构建词汇表
	v.vocabulary = make(map[string]int)
	for i, w := range words {
		v.vocabulary[w.word] = i
	}

	// 计算IDF
	v.idf = make(map[string]float64)
	for word := range v.vocabulary {
		df := docFreq[word]
		// IDF = log((N + 1) / (df + 1)) + 1
		v.idf[word] = math.Log(float64(v.docCount+1)/float64(df+1)) + 1
	}
}

// Transform 将文档转换为TF-IDF向量
func (v *TFIDFVectorizer) Transform(document string) []float32 {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if len(v.vocabulary) == 0 {
		return []float32{}
	}

	vector := make([]float32, len(v.vocabulary))
	tokens := v.tokenizer(document)

	// 计算词频
	tf := make(map[string]int)
	for _, token := range tokens {
		if _, exists := v.vocabulary[token]; exists {
			tf[token]++
		}
	}

	// 计算TF-IDF
	maxTF := 1
	for _, count := range tf {
		if count > maxTF {
			maxTF = count
		}
	}

	// 预计算 TF 归一化因子
	tfNormFactor := tfSmoothFactor / float64(maxTF)

	for word, count := range tf {
		idx, exists := v.vocabulary[word]
		if exists {
			// TF = 0.5 + 0.5 * (count / maxTF)
			tfValue := tfSmoothFactor + tfNormFactor*float64(count)
			vector[idx] = float32(tfValue * v.idf[word])
		}
	}

	// 归一化
	if v.config.Normalize {
		norm := float32(0)
		for _, val := range vector {
			norm += val * val
		}
		if norm > 0 {
			norm = float32(math.Sqrt(float64(norm)))
			for i := range vector {
				vector[i] /= norm
			}
		}
	}

	return vector
}

// FitTransform 训练并转换文档
func (v *TFIDFVectorizer) FitTransform(documents []string) [][]float32 {
	v.Fit(documents)

	vectors := make([][]float32, len(documents))
	for i, doc := range documents {
		vectors[i] = v.Transform(doc)
	}

	return vectors
}

// BatchTransform 批量转换文档
func (v *TFIDFVectorizer) BatchTransform(documents []string) [][]float32 {
	vectors := make([][]float32, len(documents))
	for i, doc := range documents {
		vectors[i] = v.Transform(doc)
	}
	return vectors
}

// GetVocabularySize 获取词汇表大小
func (v *TFIDFVectorizer) GetVocabularySize() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.vocabulary)
}

// GetDimension 获取向量维度
func (v *TFIDFVectorizer) GetDimension() int {
	return v.GetVocabularySize()
}

// CreateEmbeddingFunc 创建Embedding函数
func (v *TFIDFVectorizer) CreateEmbeddingFunc() EmbeddingFunc {
	return func(text string) ([]float32, error) {
		vector := v.Transform(text)
		if len(vector) == 0 {
			return nil, nil
		}
		return vector, nil
	}
}

// CreateBatchEmbeddingFunc 创建批量Embedding函数
func (v *TFIDFVectorizer) CreateBatchEmbeddingFunc() BatchEmbeddingFunc {
	return func(texts []string) ([][]float32, error) {
		vectors := make([][]float32, len(texts))
		for i, text := range texts {
			vectors[i] = v.Transform(text)
		}
		return vectors, nil
	}
}

// SetTokenizer 设置自定义分词函数
func (v *TFIDFVectorizer) SetTokenizer(tokenizer func(string) []string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.tokenizer = tokenizer
}

// chineseTokenizer 中文分词函数（改进版）
func chineseTokenizer(text string) []string {
	// 预分配切片容量
	tokens := make([]string, 0, len(text)/2)

	// 简单的中文分词：按字符分词，但保留所有中文字符
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			// 中文字符单独成词
			tokens = append(tokens, string(r))
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			// 英文和数字
			tokens = append(tokens, strings.ToLower(string(r)))
		}
	}

	// 过滤停用词和短词（中文不过滤停用词）
	filtered := make([]string, 0, len(tokens))
	for _, token := range tokens {
		// 判断是否为中文（只检查第一个字符）
		isChinese := len(token) > 0 && unicode.Is(unicode.Han, []rune(token)[0])

		// 中文不过滤停用词，英文过滤停用词和短词
		if len(token) > 0 {
			if isChinese {
				// 中文只过滤标点符号（已经在前面处理了）
				filtered = append(filtered, token)
			} else {
				// 英文过滤停用词和短词
				if !isStopWord(token) && len(token) > minWordLength {
					filtered = append(filtered, token)
				}
			}
		}
	}

	return filtered
}

// mixedTokenizer 混合分词函数（支持中英文）
func mixedTokenizer(text string) []string {
	// 检测是否包含中文
	hasChinese := false
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			hasChinese = true
			break
		}
	}

	if hasChinese {
		return chineseTokenizer(text)
	}
	return defaultTokenizer(text)
}
