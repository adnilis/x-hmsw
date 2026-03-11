package embedding

import (
	"os"

	"github.com/adnilis/logger"
	"github.com/adnilis/x-hmsw/types"
)

// EmbeddingFunc 定义文本到向量的转换函数
type EmbeddingFunc func(text string) ([]float32, error)

// BatchEmbeddingFunc 定义批量文本到向量的转换函数
type BatchEmbeddingFunc func(texts []string) ([][]float32, error)

// Vectorizer 向量化器接口
// 统一的向量化器接口，支持 TF-IDF、BM25 等不同实现
type Vectorizer interface {
	// Fit 训练向量化器
	Fit(documents []string)
	// Transform 将单个文档转换为向量
	Transform(document string) []float32
	// BatchTransform 批量将文档转换为向量
	BatchTransform(documents []string) [][]float32
	// GetDimension 获取向量维度
	GetDimension() int
	// CreateEmbeddingFunc 创建单个文本的 Embedding 函数
	CreateEmbeddingFunc() EmbeddingFunc
	// CreateBatchEmbeddingFunc 创建批量文本的 Embedding 函数
	CreateBatchEmbeddingFunc() BatchEmbeddingFunc
}

// Config Embedding配置
type Config struct {
	BaseURL string // API基础URL
	APIKey  string // API密钥
	Model   string // 模型名称
}

// DefaultConfig 返回默认配置
// 优先使用环境变量，否则使用默认值
func DefaultConfig() Config {
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = ""
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "text-embedding-3-small"
	}

	return Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
	}
}

// TFIDFConfig TF-IDF配置
type TFIDFConfig struct {
	MaxVocabSize int     // 最大词汇表大小（默认10000）
	MinDocFreq   int     // 最小文档频率（默认1）
	MaxDocFreq   float64 // 最大文档频率比例（默认1.0，即不限制）
	Normalize    bool    // 是否归一化向量（默认true）
}

// DefaultTFIDFConfig 返回默认TF-IDF配置
func DefaultTFIDFConfig() TFIDFConfig {
	return TFIDFConfig{
		MaxVocabSize: 10000,
		MinDocFreq:   1,
		MaxDocFreq:   1.0,
		Normalize:    true,
	}
}

// BM25Config BM25配置
type BM25Config struct {
	MaxVocabSize int                 // 最大词汇表大小（默认10000）
	MinDocFreq   int                 // 最小文档频率（默认1）
	MaxDocFreq   float64             // 最大文档频率比例（默认1.0，即不限制）
	K1           float64             // 词频饱和参数（默认1.5，通常1.2-2.0）
	B            float64             // 长度归一化参数（默认0.75，通常0.5-1.0）
	Variant      types.EmbeddingType // BM25变体: "bm25", "bm25l", "bm25+", "bm25f"（默认"bm25"）
	Delta        float64             // BM25+的delta参数（默认1.0）
}

// DefaultBM25Config 返回默认BM25配置
func DefaultBM25Config() BM25Config {
	return BM25Config{
		MaxVocabSize: 10000,
		MinDocFreq:   1,
		MaxDocFreq:   1.0,
		K1:           1.5,
		B:            0.75,
		Variant:      types.EmbeddingBM25,
		Delta:        1.0,
	}
}

// BM25FVectorizerAdapter BM25F 向量化器适配器，将多字段向量器适配为单字段接口
type BM25FVectorizerAdapter struct {
	vectorizer *BM25FVectorizer
}

func NewBM25FVectorizerAdapter(vectorizer *BM25FVectorizer) *BM25FVectorizerAdapter {
	return &BM25FVectorizerAdapter{vectorizer: vectorizer}
}

func (a *BM25FVectorizerAdapter) Fit(documents []string) {
	// 将单字符串文档转换为多字段格式
	multiFieldDocs := make([]map[string]string, len(documents))
	for i, doc := range documents {
		multiFieldDocs[i] = map[string]string{"content": doc}
	}
	a.vectorizer.Fit(multiFieldDocs)
}

func (a *BM25FVectorizerAdapter) Transform(document string) []float32 {
	doc := map[string]string{"content": document}
	return a.vectorizer.Transform(doc)
}

func (a *BM25FVectorizerAdapter) BatchTransform(documents []string) [][]float32 {
	return a.vectorizer.BatchTransformStrings(documents)
}

func (a *BM25FVectorizerAdapter) GetDimension() int {
	return a.vectorizer.GetDimension()
}

func (a *BM25FVectorizerAdapter) CreateEmbeddingFunc() EmbeddingFunc {
	return a.vectorizer.CreateEmbeddingFunc()
}

func (a *BM25FVectorizerAdapter) CreateBatchEmbeddingFunc() BatchEmbeddingFunc {
	return a.vectorizer.CreateBatchEmbeddingFunc()
}

// NewEmbeddingVectorizer 创建向量化器
func NewEmbeddingVectorizer(embedType types.EmbeddingType) (defaultConfig Config, vectorizer Vectorizer) {
	switch embedType {
	case types.EmbeddingBM25:
		bm25Config := DefaultBM25Config()
		vectorizer = NewBM25Vectorizer(bm25Config)
	case types.EmbeddingBM25L:
		bm25Config := DefaultBM25Config()
		bm25Config.Variant = "bm25l"
		vectorizer = NewBM25LVectorizer(bm25Config)
	case types.EmbeddingBM25P:
		bm25Config := DefaultBM25Config()
		bm25Config.Variant = "bm25+"
		vectorizer = NewBM25PlusVectorizer(bm25Config)
	case types.EmbeddingBM25F:
		bm25fConfig := BM25FConfig{
			BM25Config:   DefaultBM25Config(),
			FieldWeights: map[string]float64{"title": 2.0, "content": 1.0},
		}
		bm25fVectorizer := NewBM25FVectorizer(bm25fConfig)
		vectorizer = NewBM25FVectorizerAdapter(bm25fVectorizer)
	case types.EmbeddingTFIDF:
		tfidfConfig := DefaultTFIDFConfig()
		vectorizer = NewTFIDFVectorizer(tfidfConfig)
	default:
		logger.Warn("unknown embedding type, falling back to BM25: %v", string(embedType))
		bm25Config := DefaultBM25Config()
		vectorizer = NewBM25Vectorizer(bm25Config)
	}
	return defaultConfig, vectorizer
}
