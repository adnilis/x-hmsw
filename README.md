# x-hmsw

<div align="center">

**高性能纯 Go 向量数据库**

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![QPS](https://img.shields.io/badge/QPS-3,457-green.svg)](docs/09-性能优化/README.md)
[![Vectors](https://img.shields.io/badge/Vectors-1M+-orange.svg)](docs/01-项目概述.md)

快速 · 轻量 · 高性能

[快速开始](#快速开始) · [文档](docs/) · [示例](examples/) · [性能](#性能)

</div>

---

## 简介

**x-hmsw** 是一个纯 Go 实现的高性能向量数据库，专为大规模向量相似度搜索而设计。它提供了完整的向量数据库解决方案，包括多种索引算法、灵活的存储引擎、先进的向量压缩技术以及完善的监控系统。

### 核心特性

- 🚀 **高性能**: QPS 达到 3,457，支持 SIMD 加速、对象池优化
- 📦 **多索引支持**: HNSW、IVF、Flat、ANN 四种索引算法
- 💾 **灵活存储**: Memory、BadgerDB、BBolt、PebbleDB、Mmap 五种存储引擎
- 🗜️ **向量压缩**: PQ、SQ、Binary 三种压缩技术
- 📝 **文本向量化**: TF-IDF、BM25（含多种变体）、OpenAI Embeddings
- 🎯 **易用性**: QuickDB 简化 API，开箱即用
- 📊 **监控**: 内置 Prometheus 指标支持
- 🔧 **纯 Go**: 无外部依赖，易于集成

### 性能指标

| 指标 | 数值 |
|-----|------|
| **QPS** | 3,457 |
| **平均搜索时间** | 289µs |
| **插入性能** | 1,639 vectors/s |
| **内存占用** | 4.6MB/10K 向量 |
| **精度** | 0.8368 (HNSW) |

---

## 快速开始

### 安装

```bash
go get github.com/adnilis/x-hmsw
```

### 基础使用

```go
package main

import (
    "fmt"
    "github.com/adnilis/x-hmsw/api"
    "github.com/adnilis/x-hmsw/types"
)

func main() {
    // 创建向量数据库
    db, err := api.NewQuick("./data")
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // 插入向量
    vectors := []types.Vector{
        {
            ID:     "vec_001",
            Vector: []float32{0.1, 0.2, 0.3, 0.4},
            Payload: map[string]interface{}{
                "category": "tech",
            },
        },
        {
            ID:     "vec_002",
            Vector: []float32{0.5, 0.6, 0.7, 0.8},
            Payload: map[string]interface{}{
                "category": "science",
            },
        },
    }
    err = db.Insert(vectors)
    if err != nil {
        panic(err)
    }

    // 搜索向量
    query := types.Vector{Vector: []float32{0.1, 0.2, 0.3, 0.4}}
    results, err := db.Search(query, 5)
    if err != nil {
        panic(err)
    }

    fmt.Println("搜索结果:", results)
}
```

### 文本向量化

```go
package main

import (
    "github.com/adnilis/x-hmsw/embedding"
)

func main() {
    // 创建 TF-IDF 向量化器
    tfidf := embedding.NewTFIDF()

    // 添加文档
    documents := []string{
        "机器学习是人工智能的一个分支",
        "深度学习是机器学习的一种方法",
    }
    tfidf.AddDocuments(documents)

    // 训练模型
    tfidf.Train()

    // 向量化文本
    vector := tfidf.Vectorize("什么是机器学习")
    fmt.Println("向量:", vector)
}
```

---

## 架构设计

```
┌─────────────────────────────────────────────────────────┐
│                    Application Layer                     │
├─────────────────────────────────────────────────────────┤
│                      API Layer                          │
│                   (QuickDB 接口)                         │
├─────────────────────────────────────────────────────────┤
│                   Interface Layer                       │
│              (VectorDB, Index 接口)                      │
├─────────────────────────────────────────────────────────┤
│                 Embedding Layer                         │
│         (TF-IDF, BM25, OpenAI Embeddings)               │
├─────────────────────────────────────────────────────────┤
│                    Index Layer                          │
│         (HNSW, IVF, Flat, ANN 索引实现)                 │
├─────────────────────────────────────────────────────────┤
│                 Compression Layer                       │
│            (PQ, SQ, Binary 压缩)                        │
├─────────────────────────────────────────────────────────┤
│                   Storage Layer                         │
│      (Badger, BBolt, Pebble, Memory, Mmap)             │
├─────────────────────────────────────────────────────────┤
│                    Utils Layer                          │
│         (SIMD, Pool, Math, Concurrency)                 │
├─────────────────────────────────────────────────────────┤
│                  Infrastructure                         │
│              (Metrics, Logging, Serialization)          │
└─────────────────────────────────────────────────────────┘
```

---

## 索引算法

### HNSW (Hierarchical Navigable Small World)

基于分层小世界图的近似最近邻搜索，提供高精度和高性能的平衡。

```go
config := iface.Config{
    Dimension:      128,
    IndexType:      iface.HNSW,
    M:              16,
    EfConstruction: 200,
    EfSearch:       100,
}
```

**性能**: QPS 3,457, 精度 0.8368

### IVF (Inverted File)

基于聚类的倒排索引，适合大规模数据集。

```go
config := iface.Config{
    Dimension:   128,
    IndexType:   iface.IVF,
    NumClusters: 100,
    Nprobe:      10,
}
```

**性能**: QPS 3,205, 精度 0.8123

### Flat

暴力搜索，提供精确结果，适合小规模数据集。

```go
config := iface.Config{
    Dimension: 128,
    IndexType: iface.Flat,
}
```

**性能**: QPS 803, 精度 1.0000

---

## 存储引擎

| 存储引擎 | 搜索时间 | QPS | 特点 |
|---------|---------|-----|------|
| **Memory** | 289µs | 3,457 | 最快，易失性 |
| **Mmap** | 312µs | 3,205 | 快速，持久化 |
| **BBolt** | 527µs | 1,897 | 稳定，单文件 |
| **PebbleDB** | 541µs | 1,848 | 高写入 |
| **BadgerDB** | 554µs | 1,807 | 分布式 |

---

## 性能优化

### SIMD 加速

使用 AVX2、SSE、NEON 指令集加速距离计算，性能提升 20-30%。

### 对象池

复用临时对象，减少内存分配和 GC 压力，性能提升 10-20%。

### 并发控制

细粒度锁和无锁数据结构，充分利用多核，性能提升 2-4x。

### 预取优化

提前加载数据，减少缓存未命中，性能提升 5-10%。

详细性能报告请参考 [性能优化文档](docs/09-性能优化/README.md)。

---

## 文档

- [项目概述](docs/01-项目概述.md)
- [快速开始](docs/02-快速开始.md)
- [架构设计](docs/03-架构设计.md)
- [索引算法](docs/04-索引算法/README.md)
- [存储引擎](docs/05-存储引擎/README.md)
- [向量压缩](docs/06-向量压缩/README.md)
- [API 接口](docs/07-API接口/README.md)
- [序列化](docs/08-序列化/README.md)
- [性能优化](docs/09-性能优化/README.md)
- [监控指标](docs/10-监控指标/README.md)
- [工具函数](docs/11-工具函数/README.md)
- [类型定义](docs/12-类型定义/README.md)
- [文本向量化](docs/13-文本向量化/README.md)

---

## 示例

- [基础示例](examples/)
- [性能测试](examples/benchmark/)
- [综合性能测试](examples/comprehensive_perf/)
- [索引类型演示](examples/index_types_demo/)
- [BM25 压力测试](examples/bm25_stress/)

---

## 性能对比

### 与主流向量库对比

| 向量库 | QPS | 部署复杂度 | 资源占用 | 语言 | 适用场景 |
|-------|-----|-----------|---------|------|---------|
| **x-hmsw** | 3,457 | ⭐ 极简 | ⭐ 低 | Go | 中小规模、边缘、嵌入式 |
| **Milvus** | 5,000-10,000 | ⭐⭐⭐⭐ 复杂 | ⭐⭐⭐ 高 | Go/Python | 大规模分布式 |
| **Faiss** | 4,000-8,000 | ⭐⭐ 中等 | ⭐⭐ 中 | C++/Python | 高性能计算 |
| **Weaviate** | 1,000-2,000 | ⭐⭐⭐ 中等 | ⭐⭐⭐⭐ 高 | Go | 语义搜索、知识图谱 |
| **Qdrant** | 2,000-3,000 | ⭐⭐⭐ 中等 | ⭐⭐⭐ 中 | Rust | 生产环境、过滤需求 |
| **pgvector** | 500-1,000 | ⭐⭐ 中等 | ⭐⭐⭐ 高 | C/SQL | SQL 集成场景 |
| **Chroma** | 1,000-1,500 | ⭐ 极简 | ⭐⭐ 中 | Python | Python 应用、快速原型 |

### 适用场景

✅ **选择 x-hmsw 的场景**:
- 中小规模数据（< 1M 向量）
- Go 语言项目
- 边缘计算/嵌入式设备
- 需要快速部署
- 资源受限环境
- 需要多种存储选项

❌ **选择其他向量库的场景**:
- 超大规模数据（> 10M 向量）→ Milvus
- 需要复杂过滤 → Qdrant
- SQL 集成需求 → pgvector
- Python 生态优先 → Chroma/Faiss
- 需要丰富功能（GraphQL 等）→ Weaviate

---

## 贡献

欢迎贡献代码！请遵循以下步骤：

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

---

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

---

## 联系方式

- 项目主页: [https://github.com/adnilis/x-hmsw](https://github.com/adnilis/x-hmsw)
- 问题反馈: [Issues](https://github.com/adnilis/x-hmsw/issues)

---

## 致谢

感谢所有为本项目做出贡献的开发者！

---

<div align="center">

**如果这个项目对你有帮助，请给一个 ⭐️ Star！**

Made with ❤️ by x-hmsw Contributors

</div>
