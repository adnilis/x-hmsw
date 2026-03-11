package types

import "time"

// Vector 向量数据结构
type Vector struct {
	ID        string                 // 向量唯一标识
	Vector    []float32              // 向量数据
	Payload   map[string]interface{} // 元数据/负载数据
	Timestamp time.Time              // 创建时间戳
}

// SearchResult 搜索结果
type SearchResult struct {
	ID       string                 // 向量 ID
	Score    float32                // 相似度分数 (0-1，越大越相似)
	Distance float32                // 原始距离值
	Vector   []float32              // 可选：返回向量数据
	Payload  map[string]interface{} // 可选：返回负载数据
}

// IndexSearchResult 索引内部搜索结果
type IndexSearchResult struct {
	Node     *IndexNode // 命中的节点
	Distance float32    // 距离值
}

// IndexNode 索引节点
type IndexNode struct {
	ID      int
	Vector  []float32
	Deleted bool // 软删除标记
}

// SearchOptions 搜索选项
type SearchOptions struct {
	TopK        int                    // 返回结果数量
	MinScore    float32                // 最小相似度阈值
	MaxDistance float32                // 最大距离阈值
	WithVector  bool                   // 是否返回向量数据
	WithPayload bool                   // 是否返回负载数据
	Filter      map[string]interface{} // Payload过滤条件
	// 支持的过滤条件格式：
	// 1. 简单值匹配: {"role": "user"}
	// 2. 比较操作: {"timestamp": {"$gt": time1, "$lt": time2}}
	//    支持的操作符: $eq, $ne, $gt, $gte, $lt, $lte, $in, $nin
	// 3. 逻辑操作: {"$or": [{"role": "user"}, {"role": "admin"}], "$and": [...]}
}

// DistanceMetric 距离度量类型
type DistanceMetric string

const (
	Cosine       DistanceMetric = "cosine" // 余弦相似度
	L2           DistanceMetric = "l2"     // 欧几里得距离
	InnerProduct DistanceMetric = "ip"     // 内积
)

// IndexType 索引类型
type IndexType string

const (
	HNSW IndexType = "hnsw" // HNSW 索引
	IVF  IndexType = "ivf"  // IVF 索引
	Flat IndexType = "flat" // 暴力搜索
)

// StorageType 存储类型
type StorageType string

const (
	Memory StorageType = "memory" // 内存存储
	Badger StorageType = "badger" // BadgerDB
	BBolt  StorageType = "bbolt"  // BBolt
	Pebble StorageType = "pebble" // PebbleDB
	MMap   StorageType = "mmap"   // 内存映射
)

// EmbeddingType Embedding类型
type EmbeddingType string

const (
	EmbeddingOpenAI EmbeddingType = "openai" // OpenAI Embedding
	EmbeddingTFIDF  EmbeddingType = "tfidf"  // TF-IDF
	EmbeddingBM25   EmbeddingType = "bm25"   // BM25
	EmbeddingBM25F  EmbeddingType = "bm25f"  // BM25F
	EmbeddingBM25L  EmbeddingType = "bm25l"  // BM25L
	EmbeddingBM25P  EmbeddingType = "bm25+"  // BM25+
)

// Config 数据库配置
type Config struct {
	// 基础配置
	Dimension      int
	IndexType      IndexType
	DistanceMetric DistanceMetric
	StorageType    StorageType
	StoragePath    string

	// 容量配置
	MaxVectors int
	CacheSize  int

	// HNSW 参数
	M              int // 每层最大连接数
	EfConstruction int
	EfSearch       int

	// IVF 参数
	NumClusters int // 聚类数量
	Nprobe      int // 搜索时检查的聚类数

	// 量化配置
	Quantize      bool
	QuantizerType string

	// 性能参数
	NumThreads int

	// Embedding配置
	EmbeddingType  EmbeddingType // Embedding类型（默认bm25）
	EmbeddingField string        // 从Payload中提取文本的字段名（默认"content"）
}

// DefaultConfig 默认配置
func DefaultConfig() Config {
	return Config{
		Dimension:      384,
		IndexType:      HNSW,
		DistanceMetric: Cosine,
		StorageType:    Badger,
		StoragePath:    "./data/vectors",
		MaxVectors:     1000000,
		CacheSize:      10000,
		M:              16,
		EfConstruction: 200,
		EfSearch:       100,
		NumClusters:    100, // IVF默认聚类数
		Nprobe:         10,  // IVF默认搜索聚类数
		Quantize:       false,
		NumThreads:     4,
		EmbeddingType:  EmbeddingBM25, // 默认使用BM25
		EmbeddingField: "content",     // 默认从Payload的content字段提取文本
	}
}
