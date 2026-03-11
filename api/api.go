package api

import (
	"fmt"
	"time"

	"github.com/adnilis/x-hmsw/embedding"
	iface "github.com/adnilis/x-hmsw/interface"
	"github.com/adnilis/x-hmsw/types"
)

// QuickDB 简化的向量数据库接口
type QuickDB struct {
	db *iface.PureGoVectorDB
}

// NewQuick 创建快速向量数据库（使用默认配置）
// 自动启用自动保存，每30秒保存一次
func NewQuick(storagePath string) (*QuickDB, error) {
	return NewQuickWithConfig(storagePath, 30*time.Second)
}

// NewQuickWithConfig 创建快速向量数据库（自定义自动保存间隔）
func NewQuickWithConfig(storagePath string, autoSaveInterval time.Duration) (*QuickDB, error) {
	config := iface.Config{
		Dimension:      384, // 默认维度
		IndexType:      iface.HNSW,
		StorageType:    iface.Badger,
		StoragePath:    storagePath,
		DistanceMetric: iface.Cosine,
		M:              16,
		EfConstruction: 200,
		MaxVectors:     1000000,
		EmbeddingField: "content", // 默认从Payload的content字段提取文本
	}

	db, err := iface.NewPureGoVectorDB(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// 启用自动保存
	savePath := storagePath + "/autosave"
	if err := db.EnableAutoSave(autoSaveInterval, savePath); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable auto save: %w", err)
	}

	return &QuickDB{db: db}, nil
}

// NewQuickWithStoragePath 使用自定义存储路径创建数据库
func NewQuickWithStoragePath(storagePath string) (*QuickDB, error) {
	return NewQuick(storagePath)
}

// Insert 插入向量
func (q *QuickDB) Insert(vectors []types.Vector) error {
	return q.db.Insert(vectors)
}

// InsertOne 插入单个向量
func (q *QuickDB) InsertOne(vector types.Vector) error {
	return q.db.Insert([]types.Vector{vector})
}

// Search 搜索向量（使用默认选项）
func (q *QuickDB) Search(query types.Vector, topK int) ([]iface.SearchResult, error) {
	opts := iface.SearchOptions{
		TopK:        topK,
		MinScore:    0.0,
		WithVector:  false,
		WithPayload: true,
	}
	return q.db.Search(query, opts)
}

// SearchWithFilter 使用Payload过滤搜索向量
func (q *QuickDB) SearchWithFilter(query types.Vector, topK int, filter map[string]interface{}) ([]iface.SearchResult, error) {
	opts := iface.SearchOptions{
		TopK:        topK,
		MinScore:    0.0,
		WithVector:  false,
		WithPayload: true,
		Filter:      filter,
	}
	return q.db.Search(query, opts)
}

// SearchWithOptions 使用自定义选项搜索向量
func (q *QuickDB) SearchWithOptions(query types.Vector, opts iface.SearchOptions) ([]iface.SearchResult, error) {
	return q.db.Search(query, opts)
}

// Delete 删除向量
func (q *QuickDB) Delete(ids []string) error {
	return q.db.Delete(ids)
}

// DeleteOne 删除单个向量
func (q *QuickDB) DeleteOne(id string) error {
	return q.db.Delete([]string{id})
}

// InsertWithText 插入文本，自动生成向量
// id: 向量唯一标识
// text: 文本内容
// payload: 元数据/负载数据（可选）
func (q *QuickDB) InsertWithText(id, text string, payload map[string]interface{}) error {
	if payload == nil {
		payload = make(map[string]interface{})
	}
	payload["content"] = text

	vector := types.Vector{
		ID:      id,
		Payload: payload,
	}
	return q.db.Insert([]types.Vector{vector})
}

// SearchByText 搜索文本，自动生成查询向量
// query: 查询文本
// topK: 返回结果数量
func (q *QuickDB) SearchByText(query string, topK int) ([]iface.SearchResult, error) {
	// 获取Embedding函数
	embeddingFunc := q.db.GetEmbeddingFunc()
	if embeddingFunc == nil {
		// 如果没有设置embeddingFunc，尝试使用batchEmbeddingFunc
		batchEmbeddingFunc := q.db.GetBatchEmbeddingFunc()
		if batchEmbeddingFunc == nil {
			// 触发一次搜索来初始化默认的TF-IDF
			_, _ = q.Search(types.Vector{}, 1)

			// 再次获取batchEmbeddingFunc
			batchEmbeddingFunc = q.db.GetBatchEmbeddingFunc()
			if batchEmbeddingFunc == nil {
				return nil, fmt.Errorf("embedding function is not set")
			}
		}
		// 使用batchEmbeddingFunc生成单个查询向量
		embeddings, err := batchEmbeddingFunc([]string{query})
		if err != nil {
			return nil, fmt.Errorf("failed to generate query embedding: %w", err)
		}
		if len(embeddings) == 0 {
			return nil, fmt.Errorf("no embedding generated for query")
		}
		queryVector := embeddings[0]
		searchQuery := types.Vector{
			Vector: queryVector,
		}
		return q.Search(searchQuery, topK)
	}

	// 生成查询向量
	queryVector, err := embeddingFunc(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// 执行搜索
	searchQuery := types.Vector{
		Vector: queryVector,
	}
	return q.Search(searchQuery, topK)
}

// SearchByTextWithFilter 使用文本搜索并应用Payload过滤
func (q *QuickDB) SearchByTextWithFilter(query string, topK int, filter map[string]interface{}) ([]iface.SearchResult, error) {
	// 获取Embedding函数
	embeddingFunc := q.db.GetEmbeddingFunc()
	if embeddingFunc == nil {
		// 如果没有设置embeddingFunc，尝试使用batchEmbeddingFunc
		batchEmbeddingFunc := q.db.GetBatchEmbeddingFunc()
		if batchEmbeddingFunc == nil {
			return nil, fmt.Errorf("embedding function is not set")
		}
		// 使用batchEmbeddingFunc生成单个查询向量
		embeddings, err := batchEmbeddingFunc([]string{query})
		if err != nil {
			return nil, fmt.Errorf("failed to generate query embedding: %w", err)
		}
		if len(embeddings) == 0 {
			return nil, fmt.Errorf("no embedding generated for query")
		}
		queryVector := embeddings[0]
		searchQuery := types.Vector{
			Vector: queryVector,
		}
		return q.SearchWithFilter(searchQuery, topK, filter)
	}

	// 生成查询向量
	queryVector, err := embeddingFunc(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// 执行搜索
	searchQuery := types.Vector{
		Vector: queryVector,
	}
	return q.SearchWithFilter(searchQuery, topK, filter)
}

// SetEmbeddingFunc 设置Embedding函数
func (q *QuickDB) SetEmbeddingFunc(fn embedding.EmbeddingFunc) {
	q.db.SetEmbeddingFunc(fn)
}

// SetBatchEmbeddingFunc 设置批量Embedding函数
func (q *QuickDB) SetBatchEmbeddingFunc(fn embedding.BatchEmbeddingFunc) {
	q.db.SetBatchEmbeddingFunc(fn)
}

// GetEmbeddingFunc 获取Embedding函数
func (q *QuickDB) GetEmbeddingFunc() embedding.EmbeddingFunc {
	return q.db.GetEmbeddingFunc()
}

// GetBatchEmbeddingFunc 获取批量Embedding函数
func (q *QuickDB) GetBatchEmbeddingFunc() embedding.BatchEmbeddingFunc {
	return q.db.GetBatchEmbeddingFunc()
}

// Count 获取向量数量
func (q *QuickDB) Count() (int, error) {
	return q.db.Count()
}

// GetByID 根据ID获取向量
func (q *QuickDB) GetByID(id string) (*types.Vector, error) {
	return q.db.GetByID(id)
}

// BatchGetByID 根据ID列表批量获取向量
func (q *QuickDB) BatchGetByID(ids []string) ([]*types.Vector, error) {
	return q.db.BatchGetByID(ids)
}

// Save 手动保存数据库
func (q *QuickDB) Save(path string) error {
	return q.db.ForceSave(path)
}

// Close 关闭数据库
func (q *QuickDB) Close() error {
	return q.db.Close()
}

// ForceSave 强制立即保存
func (q *QuickDB) ForceSave(path string) error {
	return q.db.ForceSave(path)
}

// IsDirty 检查是否有未保存的数据
func (q *QuickDB) IsDirty() bool {
	return q.db.IsDirty()
}

// Builder 构建器模式
type Builder struct {
	config             iface.Config
	autoSave           bool
	autoSaveInterval   time.Duration
	savePath           string
	embeddingFunc      embedding.EmbeddingFunc
	batchEmbeddingFunc embedding.BatchEmbeddingFunc
}

// NewBuilder 创建构建器
func NewBuilder() *Builder {
	return &Builder{
		config: iface.Config{
			Dimension:      384,
			IndexType:      iface.HNSW,
			StorageType:    iface.Badger,
			DistanceMetric: iface.Cosine,
			M:              16,
			EfConstruction: 200,
			MaxVectors:     1000000,
			EmbeddingType:  types.EmbeddingBM25, // 默认使用BM25
			EmbeddingField: "content",           // 默认从Payload的content字段提取文本
		},
		autoSave:         true,
		autoSaveInterval: 30 * time.Second,
	}
}

// WithDimension 设置向量维度
func (b *Builder) WithDimension(dim int) *Builder {
	b.config.Dimension = dim
	return b
}

// WithStoragePath 设置存储路径
func (b *Builder) WithStoragePath(path string) *Builder {
	b.config.StoragePath = path
	return b
}

// WithIndexType 设置索引类型
func (b *Builder) WithIndexType(indexType iface.IndexType) *Builder {
	b.config.IndexType = indexType
	return b
}

// WithStorageType 设置存储类型
func (b *Builder) WithStorageType(storageType iface.StorageType) *Builder {
	b.config.StorageType = storageType
	return b
}

// WithDistanceMetric 设置距离度量
func (b *Builder) WithDistanceMetric(metric iface.DistanceMetric) *Builder {
	b.config.DistanceMetric = metric
	return b
}

// WithHNSWParams 设置HNSW参数
func (b *Builder) WithHNSWParams(m, efConstruction int, maxVectors int) *Builder {
	b.config.M = m
	b.config.EfConstruction = efConstruction
	b.config.MaxVectors = maxVectors
	return b
}

// WithAutoSave 设置自动保存
func (b *Builder) WithAutoSave(enabled bool, interval time.Duration) *Builder {
	b.autoSave = enabled
	b.autoSaveInterval = interval
	return b
}

// WithSavePath 设置保存路径
func (b *Builder) WithSavePath(path string) *Builder {
	b.savePath = path
	return b
}

// WithEmbeddingFunc 设置自定义Embedding函数
func (b *Builder) WithEmbeddingFunc(fn embedding.EmbeddingFunc) *Builder {
	b.embeddingFunc = fn
	return b
}

// WithBatchEmbeddingFunc 设置批量Embedding函数
func (b *Builder) WithBatchEmbeddingFunc(fn embedding.BatchEmbeddingFunc) *Builder {
	b.batchEmbeddingFunc = fn
	return b
}

// WithOpenAIEmbedding 设置OpenAI Embedding（便捷方法）
// baseURL: API基础URL，默认为 https://api.openai.com/v1
// apiKey: OpenAI API密钥
// model: 使用的模型，默认为 text-embedding-3-small
func (b *Builder) WithOpenAIEmbedding(baseURL, apiKey, model string) *Builder {
	client := embedding.NewOpenAIClient(baseURL, apiKey, model)
	b.embeddingFunc = client.CreateEmbeddingFunc()
	return b
}

// WithOpenAIBatchEmbedding 设置OpenAI批量Embedding（推荐使用，性能更好）
// baseURL: API基础URL，默认为 https://api.openai.com/v1
// apiKey: OpenAI API密钥
// model: 使用的模型，默认为 text-embedding-3-small
func (b *Builder) WithOpenAIBatchEmbedding(baseURL, apiKey, model string) *Builder {
	client := embedding.NewOpenAIClient(baseURL, apiKey, model)
	b.batchEmbeddingFunc = client.CreateBatchEmbeddingFunc()
	return b
}

// WithTFIDFEmbedding 设置TF-IDF Embedding（无需外部API，适合离线场景）
// documents: 训练文档列表，用于构建词汇表和IDF
// config: TF-IDF配置，如果为nil则使用默认配置
func (b *Builder) WithTFIDFEmbedding(documents []string, config *embedding.TFIDFConfig) *Builder {
	if config == nil {
		defaultConfig := embedding.DefaultTFIDFConfig()
		config = &defaultConfig
	}

	vectorizer := embedding.NewTFIDFVectorizer(*config)
	vectorizer.Fit(documents)
	b.batchEmbeddingFunc = vectorizer.CreateBatchEmbeddingFunc()

	// 更新向量维度为词汇表大小
	b.config.Dimension = vectorizer.GetDimension()

	return b
}

// WithBM25Embedding 设置BM25 Embedding（无需外部API，适合离线场景）
// documents: 训练文档列表，用于构建词汇表和IDF
// config: BM25配置，如果为nil则使用默认配置
func (b *Builder) WithBM25Embedding(documents []string, config *embedding.BM25Config) *Builder {
	if config == nil {
		defaultConfig := embedding.DefaultBM25Config()
		config = &defaultConfig
	}
	var vectorizer embedding.Vectorizer
	switch config.Variant {
	case types.EmbeddingBM25:
		// 已经是BM25配置，无需修改
	case types.EmbeddingBM25L:
		vectorizer := embedding.NewBM25LVectorizer(*config)
		vectorizer.Fit(documents)
		b.batchEmbeddingFunc = vectorizer.CreateBatchEmbeddingFunc()
	case types.EmbeddingBM25P:
		vectorizer := embedding.NewBM25PlusVectorizer(*config)
		vectorizer.Fit(documents)
		b.batchEmbeddingFunc = vectorizer.CreateBatchEmbeddingFunc()
	case types.EmbeddingBM25F:
		bm25fConfig := embedding.BM25FConfig{
			BM25Config:   embedding.DefaultBM25Config(),
			FieldWeights: map[string]float64{"title": 2.0, "content": 1.0},
		}
		vectorizer := embedding.NewBM25FVectorizer(bm25fConfig)
		b.batchEmbeddingFunc = vectorizer.CreateBatchEmbeddingFunc()
	default:
		// 默认使用BM25
		vectorizer := embedding.NewBM25Vectorizer(*config)
		vectorizer.Fit(documents)
		b.batchEmbeddingFunc = vectorizer.CreateBatchEmbeddingFunc()
	}

	// 更新向量维度为词汇表大小
	b.config.Dimension = vectorizer.GetDimension()

	return b
}

// WithEmbeddingField 设置从Payload中提取文本的字段名（默认"content"）
func (b *Builder) WithEmbeddingField(field string) *Builder {
	b.config.EmbeddingField = field
	return b
}

// Build 构建数据库
func (b *Builder) Build() (*QuickDB, error) {
	if b.config.StoragePath == "" {
		return nil, fmt.Errorf("storage path is required")
	}

	db, err := iface.NewPureGoVectorDB(b.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// 优先使用显式设置的Embedding函数
	if b.embeddingFunc != nil {
		db.SetEmbeddingFunc(b.embeddingFunc)
	}
	if b.batchEmbeddingFunc != nil {
		db.SetBatchEmbeddingFunc(b.batchEmbeddingFunc)
	}

	// 启用自动保存
	if b.autoSave {
		savePath := b.savePath
		if savePath == "" {
			savePath = b.config.StoragePath + "/autosave"
		}
		if err := db.EnableAutoSave(b.autoSaveInterval, savePath); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to enable auto save: %w", err)
		}
	}

	return &QuickDB{db: db}, nil
}
