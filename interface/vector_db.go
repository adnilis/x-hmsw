package iface

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/adnilis/x-hmsw/embedding"
	"github.com/adnilis/x-hmsw/storage/badger"
	"github.com/adnilis/x-hmsw/storage/bbolt"
	"github.com/adnilis/x-hmsw/storage/memory"
	"github.com/adnilis/x-hmsw/storage/mmap"
	"github.com/adnilis/x-hmsw/storage/pebble"
)

// VectorDB 向量数据库接口
type VectorDB interface {
	Insert(vectors []Vector) error
	Delete(ids []string) error
	Search(query Vector, opts SearchOptions) ([]SearchResult, error)
	GetByID(id string) (*Vector, error)
	BatchGetByID(ids []string) ([]*Vector, error)
	Save(path string) error
	Load(path string) error
	Close() error
	Count() (int, error)
}

// Storage 存储接口
type Storage interface {
	Get(id string) (*Vector, error)
	BatchGet(ids []string) ([]*Vector, error)
	Put(vec *Vector) error
	BatchPut(vectors []*Vector) error
	Delete(id string) error
	BatchDelete(ids []string) error
	Iterate(fn func(*Vector) bool) error
	Count() (int, error)
	Save(path string) error
	Load(path string) error
	Close() error
}

// PureGoVectorDB 纯 Go 向量数据库实现
type PureGoVectorDB struct {
	config       Config
	index        Index
	storage      Storage
	distanceFunc func(a, b []float32) float32
	logger       Logger
	mu           sync.RWMutex
	closed       bool
	// ID映射：索引数字ID -> 用户字符串ID
	idMap []string
	// 反向映射：用户字符串ID -> 索引数字ID
	idReverseMap map[string]int
	// 自动保存相关字段
	autoSave      bool
	autoSaveTimer *time.Timer
	autoSaveDone  chan struct{}
	savePath      string
	dirty         bool
	// Embedding相关
	embeddingFunc      embedding.EmbeddingFunc
	batchEmbeddingFunc embedding.BatchEmbeddingFunc
	// 统一的向量化器接口
	vectorizer      embedding.Vectorizer
	vectorizerType  EmbeddingType
	vectorizerMutex sync.Mutex
}

// NewPureGoVectorDB 创建向量数据库
func NewPureGoVectorDB(config Config) (*PureGoVectorDB, error) {
	// 初始化日志
	log := NewLogger("vector-db")
	log.Info("initializing vector database", "config", config)

	db := &PureGoVectorDB{
		config:       config,
		logger:       log,
		idMap:        make([]string, 0),
		idReverseMap: make(map[string]int),
	}

	// 初始化距离函数
	db.distanceFunc = createDistanceFunc(config.DistanceMetric)

	// 初始化索引
	indexConfig := IndexConfig{
		Dimension:      config.Dimension,
		MaxVectors:     config.MaxVectors,
		DistanceFunc:   db.distanceFunc,
		M:              config.M,
		EfConstruction: config.EfConstruction,
		NumClusters:    config.NumClusters,
		Nprobe:         config.Nprobe,
	}

	var err error
	db.index, err = NewIndex(config.IndexType, indexConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}

	// 初始化存储
	switch config.StorageType {
	case Badger, "":
		store, err := badger.NewBadgerStorage(config.StoragePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create badger storage: %w", err)
		}
		db.storage = store
	case BBolt:
		store, err := bbolt.NewBBoltStorage(config.StoragePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create bbolt storage: %w", err)
		}
		db.storage = store
	case Pebble:
		store, err := pebble.NewPebbleStorage(config.StoragePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create pebble storage: %w", err)
		}
		db.storage = store
	case MMap:
		store, err := mmap.NewMMapStorage(config.StoragePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create mmap storage: %w", err)
		}
		db.storage = store
	case Memory:
		store := memory.NewMemoryStorage()
		db.storage = store
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", config.StorageType)
	}

	// 从存储中加载已有数据到索引
	log.Info("loading existing vectors from storage")
	count, err := db.loadExistingVectors()
	if err != nil {
		log.Error("failed to load existing vectors", "error", err)
		// 不返回错误，允许空数据库启动
	} else if count > 0 {
		log.Info("loaded existing vectors", "count", count)
	}

	log.Info("vector database initialized successfully")
	return db, nil
}

// createDistanceFunc 创建距离函数
func createDistanceFunc(metric DistanceMetric) func(a, b []float32) float32 {
	switch metric {
	case Cosine, "":
		return func(a, b []float32) float32 {
			// 余弦相似度转换为距离 (1 - similarity)
			return 1.0 - cosineSimilarity(a, b)
		}
	case L2:
		return l2Distance
	case InnerProduct:
		return innerProductDistance
	default:
		return cosineSimilarity
	}
}

// loadExistingVectors 从存储中加载已有向量到索引
func (db *PureGoVectorDB) loadExistingVectors() (int, error) {
	count := 0
	err := db.storage.Iterate(func(vec *Vector) bool {
		// 检查ID是否已存在
		if existingIndexID, exists := db.idReverseMap[vec.ID]; exists {
			// ID已存在，复用旧的索引ID
			db.index.Insert(existingIndexID, vec.Vector)
			// 更新ID映射
			db.idMap[existingIndexID] = vec.ID
			db.idReverseMap[vec.ID] = existingIndexID
		} else {
			// ID不存在，创建新的索引节点
			indexID := len(db.idMap)
			db.index.Insert(indexID, vec.Vector)
			// 建立ID映射
			db.idMap = append(db.idMap, vec.ID)
			db.idReverseMap[vec.ID] = indexID
			count++
		}
		return true
	})
	return count, err
}

// Insert 插入向量
func (db *PureGoVectorDB) Insert(vectors []Vector) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return fmt.Errorf("database is closed")
	}

	db.logger.Info("inserting vectors", "count", len(vectors))

	// 批量插入存储
	var storageVectors []*Vector
	for i := range vectors {
		vec := vectors[i]
		if vec.Timestamp.IsZero() {
			vec.Timestamp = time.Now()
		}
		storageVectors = append(storageVectors, &vec)
	}

	// 批量生成向量（如果设置了batchEmbeddingFunc）
	if db.batchEmbeddingFunc != nil {
		// 收集需要生成向量的文本
		textsToEmbed := make([]string, 0)
		indicesToEmbed := make([]int, 0)

		for i, vec := range storageVectors {
			if len(vec.Vector) == 0 {
				text, ok := vec.Payload[db.config.EmbeddingField].(string)
				if !ok {
					db.logger.Error("failed to extract text from payload", "id", vec.ID, "field", db.config.EmbeddingField)
					return fmt.Errorf("vector is empty and failed to extract text from payload field '%s'", db.config.EmbeddingField)
				}
				textsToEmbed = append(textsToEmbed, text)
				indicesToEmbed = append(indicesToEmbed, i)
			}
		}

		// 批量生成向量
		if len(textsToEmbed) > 0 {
			embeddings, err := db.batchEmbeddingFunc(textsToEmbed)
			if err != nil {
				db.logger.Error("failed to generate batch embeddings", "error", err)
				return fmt.Errorf("failed to generate batch embeddings: %w", err)
			}

			// 将生成的向量分配给对应的向量对象
			for i, idx := range indicesToEmbed {
				if i < len(embeddings) {
					storageVectors[idx].Vector = embeddings[i]
				}
			}
		}
	} else if db.embeddingFunc != nil {
		// 回退到逐个生成向量（向后兼容）
		for _, vec := range storageVectors {
			if len(vec.Vector) == 0 {
				// 从Payload中提取文本
				text, ok := vec.Payload[db.config.EmbeddingField].(string)
				if !ok {
					db.logger.Error("failed to extract text from payload", "id", vec.ID, "field", db.config.EmbeddingField)
					return fmt.Errorf("vector is empty and failed to extract text from payload field '%s'", db.config.EmbeddingField)
				}

				// 调用EmbeddingFunc生成向量
				embedding, err := db.embeddingFunc(text)
				if err != nil {
					db.logger.Error("failed to generate embedding", "id", vec.ID, "error", err)
					return fmt.Errorf("failed to generate embedding for vector '%s': %w", vec.ID, err)
				}

				vec.Vector = embedding
			}
		}
	} else {
		// 没有设置embedding函数，允许空向量（将在搜索时初始化TF-IDF）
	}

	if err := db.storage.BatchPut(storageVectors); err != nil {
		db.logger.Error("failed to batch put vectors", "error", err)
		return fmt.Errorf("failed to store vectors: %w", err)
	}

	// 插入索引并建立ID映射
	for _, vec := range vectors {
		// 检查ID是否已存在
		if existingIndexID, exists := db.idReverseMap[vec.ID]; exists {
			// ID已存在，复用旧的索引ID（索引会更新向量数据）
			db.index.Insert(existingIndexID, vec.Vector)
			// 更新ID映射
			db.idMap[existingIndexID] = vec.ID
			db.idReverseMap[vec.ID] = existingIndexID
		} else {
			// ID不存在，创建新的索引节点
			indexID := len(db.idMap)
			db.index.Insert(indexID, vec.Vector)
			// 建立ID映射
			db.idMap = append(db.idMap, vec.ID)
			db.idReverseMap[vec.ID] = indexID
		}

	}

	// 标记为脏数据
	db.markDirty()

	db.logger.Info("vectors inserted successfully")
	return nil
}

// Search 搜索向量
func (db *PureGoVectorDB) Search(query Vector, opts SearchOptions) ([]SearchResult, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, fmt.Errorf("database is closed")
	}

	if opts.TopK <= 0 {
		opts.TopK = 10
	}

	// 如果没有设置embedding函数,根据EmbeddingType初始化默认的向量化器
	if db.batchEmbeddingFunc == nil && db.embeddingFunc == nil {
		db.initializeDefaultEmbedding(nil)
	}

	// 使用索引搜索
	indexResults := db.index.Search(query.Vector, opts.TopK)

	// 转换为统一结果
	results := make([]SearchResult, 0, len(indexResults))
	for _, r := range indexResults {
		score := 1.0 - r.Distance // 转换为相似度分数

		// 应用过滤
		if score < opts.MinScore {
			continue
		}
		if opts.MaxDistance > 0 && r.Distance > opts.MaxDistance {
			continue
		}

		// 使用ID映射获取用户ID
		userID := ""
		if r.Node.ID < len(db.idMap) {
			userID = db.idMap[r.Node.ID]
		} else {
			userID = fmt.Sprintf("%d", r.Node.ID)
		}

		result := SearchResult{
			ID:       userID,
			Score:    score,
			Distance: r.Distance,
		}

		// 如果需要返回向量数据或负载，或者有过滤条件
		if opts.WithVector || opts.WithPayload || len(opts.Filter) > 0 {
			vec, err := db.storage.Get(userID)
			if err != nil {
				// 如果获取失败且需要Payload，跳过该结果
				if opts.WithPayload || len(opts.Filter) > 0 {
					continue
				}
			} else {
				// 应用Payload过滤
				if len(opts.Filter) > 0 {
					if !matchPayload(vec.Payload, opts.Filter) {
						continue
					}
				}

				if opts.WithVector {
					result.Vector = vec.Vector
				}
				if opts.WithPayload {
					result.Payload = vec.Payload
				}
			}
		}

		results = append(results, result)
	}
	return results, nil
}

// Delete 删除向量
func (db *PureGoVectorDB) Delete(id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	// 从存储删除
	if err := db.storage.BatchDelete([]string{id}); err != nil {
		return fmt.Errorf("failed to delete from storage: %w", err)
	}
	if indexID, exists := db.idReverseMap[id]; exists {
		delete(db.idReverseMap, id)
		// 标记ID映射中的位置为空（不删除以保持索引一致性）
		if indexID < len(db.idMap) {
			db.idMap[indexID] = ""
		}

		// 从索引中删除节点
		if err := db.index.Delete(indexID); err != nil {
			db.logger.Warn("failed to delete node from index", "id", id, "index_id", indexID, "error", err)
			return err
		}
	}
	return nil
}

// Deletes 删除向量
func (db *PureGoVectorDB) Deletes(ids []string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return fmt.Errorf("database is closed")
	}

	db.logger.Info("deleting vectors", "count", len(ids))

	// 从存储删除
	if err := db.storage.BatchDelete(ids); err != nil {
		return fmt.Errorf("failed to delete from storage: %w", err)
	}

	// 从ID映射中删除
	for _, id := range ids {
		if indexID, exists := db.idReverseMap[id]; exists {
			delete(db.idReverseMap, id)
			// 标记ID映射中的位置为空（不删除以保持索引一致性）
			if indexID < len(db.idMap) {
				db.idMap[indexID] = ""
			}

			// 从索引中删除节点
			if err := db.index.Delete(indexID); err != nil {
				db.logger.Warn("failed to delete node from index", "id", id, "index_id", indexID, "error", err)
				// 不返回错误，继续删除其他节点
			}
		}
	}

	// 标记为脏数据
	db.markDirty()

	db.logger.Info("vectors deleted successfully")
	return nil
}

// Save 保存数据库
func (db *PureGoVectorDB) Save(path string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return fmt.Errorf("database is closed")
	}

	db.logger.Info("saving database", "path", path)
	return db.storage.Save(path)
}

// Load 加载数据库
func (db *PureGoVectorDB) Load(path string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return fmt.Errorf("database is closed")
	}

	db.logger.Info("loading database", "path", path)
	return db.storage.Load(path)
}

// Count 获取向量数量
func (db *PureGoVectorDB) Count() (int, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return 0, fmt.Errorf("database is closed")
	}

	return db.storage.Count()
}

// GetByID 根据ID获取向量
func (db *PureGoVectorDB) GetByID(id string) (*Vector, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, fmt.Errorf("database is closed")
	}

	return db.storage.Get(id)
}

// BatchGetByID 根据ID列表批量获取向量
func (db *PureGoVectorDB) BatchGetByID(ids []string) ([]*Vector, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, fmt.Errorf("database is closed")
	}

	return db.storage.BatchGet(ids)
}

// Close 关闭数据库
func (db *PureGoVectorDB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return nil
	}

	db.closed = true
	db.logger.Info("closing database")

	// 停止自动保存
	if db.autoSave {
		db.disableAutoSave()
	}

	// 如果有脏数据，强制保存
	if db.dirty && db.savePath != "" {
		db.logger.Info("saving dirty data before close")
		if err := db.storage.Save(db.savePath); err != nil {
			db.logger.Error("failed to save dirty data", "error", err)
		}
	}

	if db.storage != nil {
		return db.storage.Close()
	}

	return nil
}

// EnableAutoSave 启用自动保存
func (db *PureGoVectorDB) EnableAutoSave(interval time.Duration, savePath string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return fmt.Errorf("database is closed")
	}

	if db.autoSave {
		return fmt.Errorf("auto save already enabled")
	}

	db.autoSave = true
	db.savePath = savePath
	db.autoSaveDone = make(chan struct{})

	// 启动自动保存循环
	go db.autoSaveLoop(interval)

	return nil
}

// DisableAutoSave 禁用自动保存
func (db *PureGoVectorDB) DisableAutoSave() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return fmt.Errorf("database is closed")
	}

	if !db.autoSave {
		return nil
	}

	db.disableAutoSave()

	return nil
}

// disableAutoSave 内部方法：禁用自动保存（不加锁）
func (db *PureGoVectorDB) disableAutoSave() {
	db.autoSave = false

	if db.autoSaveTimer != nil {
		db.autoSaveTimer.Stop()
	}

	if db.autoSaveDone != nil {
		close(db.autoSaveDone)
		db.autoSaveDone = nil
	}
}

// SetEmbeddingFunc 设置Embedding函数
func (db *PureGoVectorDB) SetEmbeddingFunc(fn embedding.EmbeddingFunc) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.embeddingFunc = fn
}

// GetEmbeddingFunc 获取Embedding函数
func (db *PureGoVectorDB) GetEmbeddingFunc() embedding.EmbeddingFunc {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.embeddingFunc
}

// SetBatchEmbeddingFunc 设置批量Embedding函数
func (db *PureGoVectorDB) SetBatchEmbeddingFunc(fn embedding.BatchEmbeddingFunc) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.batchEmbeddingFunc = fn
}

// GetBatchEmbeddingFunc 获取批量Embedding函数
func (db *PureGoVectorDB) GetBatchEmbeddingFunc() embedding.BatchEmbeddingFunc {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.batchEmbeddingFunc
}

// GetConfig 获取数据库配置
func (db *PureGoVectorDB) GetConfig() Config {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.config
}

// initializeDefaultEmbedding 根据EmbeddingType初始化默认的向量化器
// 在第一次搜索时调用,从storage中获取所有已有文档训练
func (db *PureGoVectorDB) initializeDefaultEmbedding(vectors []*Vector) {
	db.vectorizerMutex.Lock()
	defer db.vectorizerMutex.Unlock()

	// 如果已经初始化过,直接返回
	if db.vectorizer != nil {
		return
	}

	db.logger.Info("initializing default vectorizer", "type", db.config.EmbeddingType)

	// 从storage中收集所有文档用于训练
	var allVectors []*Vector
	err := db.storage.Iterate(func(vec *Vector) bool {
		allVectors = append(allVectors, vec)
		return true
	})
	if err != nil {
		db.logger.Error("failed to iterate vectors for training", "error", err)
		return
	}

	// 收集所有文档文本用于训练
	trainingDocs := make([]string, 0, len(allVectors))
	for _, vec := range allVectors {
		if text, ok := vec.Payload[db.config.EmbeddingField].(string); ok && text != "" {
			trainingDocs = append(trainingDocs, text)
		}
	}

	if len(trainingDocs) == 0 {
		db.logger.Warn("no training documents found for vectorizer initialization")
		return
	}

	// 根据EmbeddingType创建对应的向量化器
	_, vectorizer_ := embedding.NewEmbeddingVectorizer(db.config.EmbeddingType)
	db.vectorizer = vectorizer_

	// 训练向量化器
	db.vectorizer.Fit(trainingDocs)

	// 设置批量embedding函数
	db.batchEmbeddingFunc = db.vectorizer.CreateBatchEmbeddingFunc()

	// 更新向量维度为词汇表大小
	newDimension := db.vectorizer.GetDimension()
	if newDimension != db.config.Dimension {
		db.logger.Info("updating vector dimension",
			"old_dimension", db.config.Dimension,
			"new_dimension", newDimension)
		db.config.Dimension = newDimension

		// 重建索引以适应新的维度
		newIndexConfig := IndexConfig{
			Dimension:      newDimension,
			MaxVectors:     db.config.MaxVectors,
			DistanceFunc:   db.distanceFunc,
			M:              db.config.M,
			EfConstruction: db.config.EfConstruction,
			NumClusters:    db.config.NumClusters,
			Nprobe:         db.config.Nprobe,
		}

		newIndex, err := NewIndex(db.config.IndexType, newIndexConfig)
		if err != nil {
			db.logger.Error("failed to recreate index with new dimension", "error", err)
			return
		}

		// 将旧索引中的向量迁移到新索引
		oldIndex := db.index
		db.index = newIndex

		// 清空ID映射，因为索引已重建
		db.idMap = make([]string, 0)
		db.idReverseMap = make(map[string]int)

		// 重新生成所有向量并插入到新索引
		for _, vec := range allVectors {
			if len(vec.Vector) == 0 {
				// 生成向量
				if text, ok := vec.Payload[db.config.EmbeddingField].(string); ok && text != "" {
					embeddings, err := db.batchEmbeddingFunc([]string{text})
					if err != nil {
						db.logger.Error("failed to generate embedding", "id", vec.ID, "error", err)
						continue
					}
					if len(embeddings) > 0 {
						vec.Vector = embeddings[0]
					}
				}
			}

			// 插入到新索引
			if len(vec.Vector) > 0 {
				// 分配新的int ID
				intID := len(db.idMap)
				db.index.Insert(intID, vec.Vector)
				db.idMap = append(db.idMap, vec.ID)
				db.idReverseMap[vec.ID] = intID
			}
		}

		db.logger.Info("reloaded vectors into new index", "count", len(allVectors))

		// 清理旧索引
		if oldIndex != nil {
			_ = oldIndex.Delete(0) // 尝试清理
		}
	}

	db.vectorizerType = db.config.EmbeddingType
	db.logger.Info("vectorizer initialized successfully",
		"type", db.vectorizerType,
		"vocabulary_size", newDimension,
		"training_docs", len(trainingDocs))
}

// autoSaveLoop 自动保存循环
func (db *PureGoVectorDB) autoSaveLoop(interval time.Duration) {
	db.autoSaveTimer = time.NewTimer(interval)

	for {
		select {
		case <-db.autoSaveTimer.C:
			db.mu.Lock()
			if !db.autoSave || db.closed {
				db.mu.Unlock()
				return
			}

			// 只有脏数据才保存
			if db.dirty {
				if err := db.storage.Save(db.savePath); err != nil {
					db.logger.Error("auto save failed", "error", err)
				} else {
					db.dirty = false
				}
			}

			db.mu.Unlock()
			db.autoSaveTimer.Reset(interval)

		case <-db.autoSaveDone:
			return
		}
	}
}

// markDirty 标记数据为脏（需要保存）
func (db *PureGoVectorDB) markDirty() {
	if db.autoSave {
		db.dirty = true
	}
}

// ForceSave 强制立即保存
func (db *PureGoVectorDB) ForceSave(path string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return fmt.Errorf("database is closed")
	}

	db.logger.Info("force saving database", "path", path)
	if err := db.storage.Save(path); err != nil {
		return fmt.Errorf("failed to force save: %w", err)
	}

	db.dirty = false
	return nil
}

// IsDirty 检查是否有未保存的数据
func (db *PureGoVectorDB) IsDirty() bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.dirty
}

// 辅助函数：余弦相似度
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// 辅助函数：L2 距离
func l2Distance(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var sum float32
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return sum
}

// 辅助函数：内积距离
func innerProductDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var sum float32
	for i := range a {
		sum += a[i] * b[i]
	}

	return -sum // 返回负值，因为内积越大越好
}

// matchPayload 检查Payload是否匹配过滤条件
func matchPayload(payload map[string]interface{}, filter map[string]interface{}) bool {
	// 检查逻辑操作符
	if orConditions, ok := filter["$or"].([]interface{}); ok {
		// $or: 任一条件满足即可
		for _, cond := range orConditions {
			if condMap, ok := cond.(map[string]interface{}); ok {
				if matchPayload(payload, condMap) {
					return true
				}
			}
		}
		return false
	}

	if andConditions, ok := filter["$and"].([]interface{}); ok {
		// $and: 所有条件都必须满足
		for _, cond := range andConditions {
			if condMap, ok := cond.(map[string]interface{}); ok {
				if !matchPayload(payload, condMap) {
					return false
				}
			}
		}
		return true
	}

	// 检查字段条件
	for key, filterValue := range filter {
		// 跳过逻辑操作符（已处理）
		if key == "$or" || key == "$and" {
			continue
		}

		payloadValue, exists := payload[key]
		if !exists {
			return false
		}

		// 检查是否是比较操作符
		if filterMap, ok := filterValue.(map[string]interface{}); ok {
			if !matchWithOperators(payloadValue, filterMap) {
				return false
			}
		} else {
			// 简单值匹配（等同于 $eq）
			if !compareValues(payloadValue, filterValue) {
				return false
			}
		}
	}
	return true
}

// matchWithOperators 使用操作符匹配值
func matchWithOperators(value interface{}, operators map[string]interface{}) bool {
	for op, opValue := range operators {
		switch op {
		case "$eq": // 等于
			if !compareValues(value, opValue) {
				return false
			}
		case "$ne": // 不等于
			if compareValues(value, opValue) {
				return false
			}
		case "$gt": // 大于
			if !compareNumbers(value, opValue, ">") {
				return false
			}
		case "$gte": // 大于等于
			if !compareNumbers(value, opValue, ">=") {
				return false
			}
		case "$lt": // 小于
			if !compareNumbers(value, opValue, "<") {
				return false
			}
		case "$lte": // 小于等于
			if !compareNumbers(value, opValue, "<=") {
				return false
			}
		case "$in": // 在列表中
			if !isInList(value, opValue) {
				return false
			}
		case "$nin": // 不在列表中
			if isInList(value, opValue) {
				return false
			}
		}
	}
	return true
}

// compareNumbers 比较数值
func compareNumbers(a, b interface{}, op string) bool {
	// 尝试转换为float64进行比较
	var fa, fb float64

	switch va := a.(type) {
	case int:
		fa = float64(va)
	case int64:
		fa = float64(va)
	case float32:
		fa = float64(va)
	case float64:
		fa = va
	default:
		// 尝试解析时间
		if ta, ok := a.(time.Time); ok {
			fa = float64(ta.Unix())
		} else {
			return false
		}
	}

	switch vb := b.(type) {
	case int:
		fb = float64(vb)
	case int64:
		fb = float64(vb)
	case float32:
		fb = float64(vb)
	case float64:
		fb = vb
	default:
		// 尝试解析时间
		if tb, ok := b.(time.Time); ok {
			fb = float64(tb.Unix())
		} else {
			return false
		}
	}

	switch op {
	case ">":
		return fa > fb
	case ">=":
		return fa >= fb
	case "<":
		return fa < fb
	case "<=":
		return fa <= fb
	}
	return false
}

// isInList 检查值是否在列表中
func isInList(value interface{}, list interface{}) bool {
	listSlice, ok := list.([]interface{})
	if !ok {
		return false
	}

	for _, item := range listSlice {
		if compareValues(value, item) {
			return true
		}
	}
	return false
}

// compareValues 比较两个值是否相等
func compareValues(a, b interface{}) bool {
	// 尝试转换为相同类型进行比较
	switch va := a.(type) {
	case string:
		if vb, ok := b.(string); ok {
			return va == vb
		}
	case int:
		if vb, ok := b.(int); ok {
			return va == vb
		}
		if vb, ok := b.(float64); ok {
			return float64(va) == vb
		}
	case float64:
		if vb, ok := b.(float64); ok {
			return va == vb
		}
		if vb, ok := b.(int); ok {
			return va == float64(vb)
		}
	case bool:
		if vb, ok := b.(bool); ok {
			return va == vb
		}
	case time.Time:
		if vb, ok := b.(time.Time); ok {
			return va.Equal(vb)
		}
	}

	// 如果类型不匹配，尝试字符串比较
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
