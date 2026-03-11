# API 接口

x-hmsw 提供了简洁易用的 API 接口，支持快速开发和灵活配置。

## API 概览

| API | 用途 | 复杂度 | 适用场景 |
|-----|------|--------|---------|
| **QuickDB** | 简化接口 | 低 | 快速开发、原型验证 |
| **VectorDB** | 核心接口 | 中 | 通用场景、需要更多控制 |
| **Index** | 索引接口 | 高 | 自定义索引实现 |

## API 详解

### QuickDB

**QuickDB** 是简化的向量数据库接口，提供开箱即用的功能。

**特点**:
- 简单易用
- 自动保存
- 默认配置
- 适合快速开发

**核心方法**:
- `NewQuick()`: 创建数据库（默认配置）
- `NewQuickWithConfig()`: 创建数据库（自定义配置）
- `Insert()`: 插入向量
- `InsertOne()`: 插入单个向量
- `Search()`: 搜索向量
- `SearchWithFilter()`: 使用过滤条件搜索
- `SearchWithOptions()`: 使用自定义选项搜索
- `Delete()`: 删除向量
- `DeleteOne()`: 删除单个向量
- `Close()`: 关闭数据库

**详细文档**: [QuickDB.md](QuickDB.md)

### VectorDB 接口

**VectorDB** 是核心数据库接口，提供完整的向量数据库功能。

**特点**:
- 功能完整
- 灵活配置
- 支持所有特性
- 适合生产环境

**核心方法**:
- `Insert()`: 插入向量
- `Delete()`: 删除向量
- `Search()`: 搜索向量
- `Save()`: 保存数据
- `Load()`: 加载数据
- `Close()`: 关闭数据库
- `Count()`: 获取向量数量

**详细文档**: [VectorDB接口.md](VectorDB接口.md)

### Index 接口

**Index** 是索引接口，定义了索引的基本操作。

**特点**:
- 统一接口
- 可扩展
- 支持多种索引实现
- 适合自定义索引

**核心方法**:
- `Insert()`: 插入向量到索引
- `Search()`: 搜索最近邻
- `Delete()`: 从索引中删除向量
- `Count()`: 获取索引中的向量数量

**详细文档**: [Index接口.md](Index接口.md)

## 使用示例

### QuickDB 基础使用

```go
package main

import (
    "fmt"
    "github.com/adnilis/x-hmsw/api"
    "github.com/adnilis/x-hmsw/types"
)

func main() {
    // 创建数据库
    db, err := api.NewQuick("./data")
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // 插入向量
    vectors := []types.Vector{
        {ID: "1", Vector: []float32{0.1, 0.2, 0.3}},
        {ID: "2", Vector: []float32{0.4, 0.5, 0.6}},
    }
    db.Insert(vectors)

    // 搜索向量
    query := types.Vector{Vector: []float32{0.1, 0.2, 0.3}}
    results, _ := db.Search(query, 5)

    fmt.Println("搜索结果:", results)
}
```

### VectorDB 接口使用

```go
package main

import (
    "github.com/adnilis/x-hmsw/interface"
    "github.com/adnilis/x-hmsw/types"
)

func main() {
    config := iface.Config{
        Dimension:      128,
        IndexType:      iface.HNSW,
        StorageType:    iface.Badger,
        StoragePath:    "./data",
        DistanceMetric: iface.Cosine,
    }

    db, err := iface.NewPureGoVectorDB(config)
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // 使用数据库...
}
```

### Index 接口使用

```go
package main

import (
    "github.com/adnilis/x-hmsw/interface"
    "github.com/adnilis/x-hmsw/types"
)

func main() {
    config := iface.IndexConfig{
        Dimension:    128,
        DistanceFunc: cosineDistance,
    }

    index := iface.NewIndex(iface.HNSW, config)

    // 使用索引...
}
```

## API 对比

### QuickDB vs VectorDB

| 特性 | QuickDB | VectorDB |
|-----|---------|----------|
| 易用性 | 高 | 中 |
| 功能 | 基础 | 完整 |
| 配置 | 默认 | 灵活 |
| 适用场景 | 快速开发 | 生产环境 |

### VectorDB vs Index

| 特性 | VectorDB | Index |
|-----|----------|-------|
| 功能 | 完整 | 索引操作 |
| 存储 | 支持 | 不支持 |
| 适用场景 | 数据库 | 索引 |

## 配置选项

### QuickDB 配置

```go
// 默认配置
db, _ := api.NewQuick("./data")

// 自定义自动保存间隔
db, _ := api.NewQuickWithConfig("./data", 1*time.Minute)
```

### VectorDB 配置

```go
config := iface.Config{
    // 基础配置
    Dimension:      128,           // 向量维度
    IndexType:      iface.HNSW,    // 索引类型
    StorageType:    iface.Badger,  // 存储类型
    StoragePath:    "./data",      // 存储路径
    DistanceMetric: iface.Cosine,  // 距离度量

    // 容量配置
    MaxVectors: 1000000,  // 最大向量数
    CacheSize:  10000,    // 缓存大小

    // HNSW 参数
    M:              16,   // 每层最大连接数
    EfConstruction: 200,  // 构建时的搜索宽度
    EfSearch:       100,  // 搜索时的搜索宽度

    // IVF 参数
    NumClusters: 100,  // 聚类数量
    Nprobe:      10,   // 搜索时检查的聚类数
}
```

### Index 配置

```go
config := iface.IndexConfig{
    Dimension:    128,           // 向量维度
    MaxVectors:   1000000,       // 最大向量数
    DistanceFunc: cosineDistance, // 距离函数

    // HNSW 参数
    M:              16,   // 每层最大连接数
    EfConstruction: 200,  // 构建时的搜索宽度

    // IVF 参数
    NumClusters: 100,  // 聚类数量
    Nprobe:      10,   // 搜索时检查的聚类数
}
```

## 搜索选项

### 基础搜索

```go
// 基础搜索
results, _ := db.Search(query, 5)
```

### 使用过滤条件

```go
// 使用 Payload 过滤
filter := map[string]interface{}{
    "category": "tech",
}
results, _ := db.SearchWithFilter(query, 5, filter)
```

### 使用自定义选项

```go
// 使用自定义选项
opts := iface.SearchOptions{
    TopK:        10,           // 返回结果数量
    MinScore:    0.8,          // 最小相似度
    MaxDistance: 0.2,          // 最大距离
    WithVector:  true,         // 返回向量数据
    WithPayload: true,         // 返回负载数据
    Filter: map[string]interface{}{
        "category": "tech",
    },
}
results, _ := db.SearchWithOptions(query, opts)
```

## 错误处理

```go
// 插入错误处理
err := db.Insert(vectors)
if err != nil {
    // 处理错误
    log.Printf("插入失败: %v", err)
}

// 搜索错误处理
results, err := db.Search(query, 5)
if err != nil {
    // 处理错误
    log.Printf("搜索失败: %v", err)
}
```

## 最佳实践

1. **快速开发**: 使用 QuickDB，简单易用
2. **生产环境**: 使用 VectorDB，功能完整
3. **自定义索引**: 使用 Index 接口
4. **资源管理**: 使用 defer 确保关闭数据库
5. **错误处理**: 始终检查错误

## 相关文档

- [QuickDB](QuickDB.md) - QuickDB API 详解
- [VectorDB接口](VectorDB接口.md) - VectorDB 接口详解
- [Index接口](Index接口.md) - Index 接口详解
- [快速开始](../02-快速开始.md) - 快速上手指南
- [架构设计](../03-架构设计.md) - 系统架构

---

**最后更新**: 2026年3月10日
