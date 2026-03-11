package iface

import "github.com/adnilis/x-hmsw/types"

// 从 types 包导入所有类型
type Vector = types.Vector
type SearchResult = types.SearchResult
type SearchOptions = types.SearchOptions
type DistanceMetric = types.DistanceMetric
type IndexType = types.IndexType
type StorageType = types.StorageType
type EmbeddingType = types.EmbeddingType
type Config = types.Config

const (
	Cosine          = types.Cosine
	L2              = types.L2
	InnerProduct    = types.InnerProduct
	HNSW            = types.HNSW
	IVF             = types.IVF
	Flat            = types.Flat
	Memory          = types.Memory
	Badger          = types.Badger
	BBolt           = types.BBolt
	Pebble          = types.Pebble
	MMap            = types.MMap
	EmbeddingOpenAI = types.EmbeddingOpenAI
	EmbeddingTFIDF  = types.EmbeddingTFIDF
	EmbeddingBM25   = types.EmbeddingBM25
	EmbeddingBM25F  = types.EmbeddingBM25F
	EmbeddingBM25L  = types.EmbeddingBM25L
	EmbeddingBM25P  = types.EmbeddingBM25P
)

// DefaultConfig 默认配置
func DefaultConfig() Config {
	return types.DefaultConfig()
}
