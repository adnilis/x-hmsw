# 类型定义

x-hmsw 定义了丰富的数据类型，支持各种向量数据库操作。

## 类型概览

| 类型类别 | 用途 | 复杂度 |
|---------|------|--------|
| **核心类型** | 向量、搜索结果 | 低 |
| **配置类型** | 索引、存储配置 | 中 |
| **枚举类型** | 距离度量、索引类型 | 低 |
| **选项类型** | 搜索选项、过滤条件 | 中 |

## 核心类型

### Vector

**Vector** 表示一个向量及其元数据。

```go
type Vector struct {
    ID        string                 // 向量唯一标识
    Vector    []float32              // 向量数据
    Payload   map[string]interface{} // 附加负载
    Timestamp int64                  // 时间戳
}
```

**字段说明**:
- `ID`: 向量的唯一标识符
- `Vector`: 向量数据，float32 类型
- `Payload`: 附加的元数据，键值对形式
- `Timestamp`: 向量创建或更新时间戳

**使用示例**:

```go
vector := types.Vector{
    ID:     "vec_001",
    Vector: []float32{0.1, 0.2, 0.3, ...},
    Payload: map[string]interface{}{
        "category": "image",
        "label":    "cat",
    },
    Timestamp: time.Now().Unix(),
}
```

### SearchResult

**SearchResult** 表示搜索结果。

```go
type SearchResult struct {
    ID       string                 // 向量 ID
    Distance float32                // 距离
    Payload  map[string]interface{} // 附加负载
}
```

**字段说明**:
- `ID`: 匹配的向量 ID
- `Distance`: 与查询向量的距离
- `Payload`: 向量的附加负载

**使用示例**:

```go
results := []types.SearchResult{
    {
        ID:       "vec_001",
        Distance: 0.123,
        Payload: map[string]interface{}{
            "category": "image",
            "label":    "cat",
        },
    },
}
```

### IndexSearchResult

**IndexSearchResult** 表示索引搜索结果。

```go
type IndexSearchResult struct {
    ID       string   // 向量 ID
    Distance float32  // 距离
    Vector   []float32 // 向量数据
}
```

**字段说明**:
- `ID`: 匹配的向量 ID
- `Distance`: 与查询向量的距离
- `Vector`: 向量数据

### IndexNode

**IndexNode** 表示索引节点。

```go
type IndexNode struct {
    ID     string   // 节点 ID
    Vector []float32 // 向量数据
}
```

## 配置类型

### Config

**Config** 是向量数据库的主配置。

```go
type Config struct {
    Dimension      int           // 向量维度
    IndexType      IndexType     // 索引类型
    StorageType    StorageType   // 存储类型
    DistanceMetric DistanceMetric // 距离度量
    Compression    CompressionType // 压缩类型
    Serialization  SerializationType // 序列化类型
    EnableMetrics  bool          // 启用监控
    AutoSave       bool          // 自动保存
    AutoSaveInterval time.Duration // 自动保存间隔
}
```

**字段说明**:
- `Dimension`: 向量维度
- `IndexType`: 索引类型（HNSW、IVF、Flat）
- `StorageType`: 存储类型（Memory、Badger、BBolt、Pebble、Mmap）
- `DistanceMetric`: 距离度量（Cosine、L2、InnerProduct）
- `Compression`: 压缩类型（PQ、SQ、Binary）
- `Serialization`: 序列化类型（Protobuf、Msgpack、Binary）
- `EnableMetrics`: 是否启用监控
- `AutoSave`: 是否自动保存
- `AutoSaveInterval`: 自动保存间隔

**使用示例**:

```go
config := types.Config{
    Dimension:      128,
    IndexType:      types.HNSW,
    StorageType:    types.Badger,
    DistanceMetric: types.Cosine,
    Compression:    types.SQ,
    Serialization:  types.Protobuf,
    EnableMetrics:  true,
    AutoSave:       true,
    AutoSaveInterval: 30 * time.Second,
}
```

### IndexConfig

**IndexConfig** 是索引配置。

```go
type IndexConfig struct {
    Dimension      int           // 向量维度
    DistanceMetric DistanceMetric // 距离度量
    M              int           // HNSW: 每层最大连接数
    EfConstruction int           // HNSW: 构建时的搜索宽度
    EfSearch       int           // HNSW: 搜索时的搜索宽度
    NumClusters    int           // IVF: 聚类数量
    Nprobe         int           // IVF: 搜索时探测的聚类数
}
```

## 枚举类型

### DistanceMetric

**DistanceMetric** 定义距离度量类型。

```go
type DistanceMetric int

const (
    Cosine       DistanceMetric = iota // 余弦距离
    L2                                  // L2 距离（欧氏距离）
    InnerProduct                        // 内积
)
```

**使用示例**:

```go
metric := types.Cosine
```

### IndexType

**IndexType** 定义索引类型。

```go
type IndexType int

const (
    HNSW IndexType = iota // HNSW 索引
    IVF                   // IVF 索引
    Flat                  // Flat 索引
    ANN                   // ANN 索引
)
```

**使用示例**:

```go
indexType := types.HNSW
```

### StorageType

**StorageType** 定义存储类型。

```go
type StorageType int

const (
    Memory StorageType = iota // 内存存储
    Badger                    // BadgerDB
    BBolt                     // BBolt
    Pebble                    // PebbleDB
    Mmap                      // 内存映射
)
```

**使用示例**:

```go
storageType := types.Badger
```

### CompressionType

**CompressionType** 定义压缩类型。

```go
type CompressionType int

const (
    PQ CompressionType = iota // 乘积量化
    SQ                        // 标量量化
    Binary                    // 二进制量化
)
```

**使用示例**:

```go
compressionType := types.SQ
```

### SerializationType

**SerializationType** 定义序列化类型。

```go
type SerializationType int

const (
    Protobuf SerializationType = iota // Protobuf
    Msgpack                           // MessagePack
    Binary                            // Binary
)
```

**使用示例**:

```go
serializationType := types.Protobuf
```

## 选项类型

### SearchOptions

**SearchOptions** 定义搜索选项。

```go
type SearchOptions struct {
    TopK int // 返回的 Top-K 结果
}
```

**使用示例**:

```go
options := types.SearchOptions{
    TopK: 10,
}
```

### FilterCondition

**FilterCondition** 定义过滤条件。

```go
type FilterCondition struct {
    Field    string      // 字段名
    Operator string      // 操作符（=, !=, >, <, >=, <=）
    Value    interface{} // 值
}
```

**使用示例**:

```go
filter := types.FilterCondition{
    Field:    "category",
    Operator: "=",
    Value:    "image",
}
```

## 类型转换

### Vector 到 IndexNode

```go
func VectorToIndexNode(v Vector) IndexNode {
    return IndexNode{
        ID:     v.ID,
        Vector: v.Vector,
    }
}
```

### IndexSearchResult 到 SearchResult

```go
func IndexSearchResultToSearchResult(r IndexSearchResult, payload map[string]interface{}) SearchResult {
    return SearchResult{
        ID:       r.ID,
        Distance: r.Distance,
        Payload:  payload,
    }
}
```

## 最佳实践

### 1. 使用合适的距离度量

| 距离度量 | 适用场景 |
|---------|---------|
| Cosine | 文本、推荐 |
| L2 | 图像、音频 |
| InnerProduct | 推荐系统 |

### 2. 使用合适的索引类型

| 索引类型 | 适用场景 |
|---------|---------|
| HNSW | 通用场景 |
| IVF | 大规模数据 |
| Flat | 小数据集 |
| ANN | 近似搜索 |

### 3. 使用合适的存储类型

| 存储类型 | 适用场景 |
|---------|---------|
| Memory | 高性能 |
| Badger | 通用场景 |
| BBolt | 单机应用 |
| Pebble | 高写入 |
| Mmap | 只读 |

### 4. 使用合适的压缩类型

| 压缩类型 | 适用场景 |
|---------|---------|
| PQ | 高压缩比 |
| SQ | 平衡 |
| Binary | 二进制向量 |

## 相关文档

- [核心类型](核心类型.md) - 核心类型详解
- [架构设计](../03-架构设计.md) - 系统架构
- [API接口](../07-API接口/README.md) - API 接口

---

**最后更新**: 2026年3月10日
