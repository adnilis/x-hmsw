# 文本向量化

x-hmsw 内置了多种文本向量化方法，无需外部依赖即可将文本转换为向量。

## 向量化方法概览

| 向量化方法 | 特点 | 适用场景 | 外部依赖 |
|-----------|------|---------|---------|
| **TF-IDF** | 经典、简单、快速 | 通用场景、快速原型 | 无 |
| **BM25** | 改进的 TF-IDF、多种变体 | 搜索引擎、文档检索 | 无 |
| **OpenAI Embeddings** | 高质量、语义丰富 | 需要高质量语义表示 | OpenAI API |

## 向量化方法详解

### TF-IDF

**TF-IDF (Term Frequency-Inverse Document Frequency)** 是经典的文本向量化方法。

**特点**:
- 简单快速
- 无需训练
- 适合快速原型开发
- 对词频敏感

**核心参数**:
- `MaxVocabSize`: 最大词汇表大小（默认 10000）
- `MinDocFreq`: 最小文档频率（默认 1）
- `MaxDocFreq`: 最大文档频率比例（默认 1.0）
- `Normalize`: 是否归一化向量（默认 true）

**适用场景**:
- 快速原型开发
- 通用文本检索
- 资源受限环境

**详细文档**: [TF-IDF.md](TF-IDF.md)

### BM25

**BM25** 是改进的 TF-IDF 方法，广泛用于搜索引擎。

**特点**:
- 改进的词频饱和
- 长度归一化
- 多种变体可选
- 支持增量更新

**核心参数**:
- `K1`: 词频饱和参数（默认 1.5，通常 1.2-2.0）
- `B`: 长度归一化参数（默认 0.75，通常 0.5-1.0）
- `Variant`: BM25 变体（bm25、bm25l、bm25+、bm25f）
- `Delta`: BM25+ 的 delta 参数（默认 1.0）

**BM25 变体**:
- **BM25**: 标准版本
- **BM25L**: 长度归一化改进版本
- **BM25+**: 添加 delta 参数改进
- **BM25F**: 支持字段加权

**适用场景**:
- 搜索引擎
- 文档检索
- 聊天记录检索
- 需要增量更新的场景

**详细文档**: [BM25.md](BM25.md)

### OpenAI Embeddings

**OpenAI Embeddings** 调用 OpenAI API 获取高质量的文本向量。

**特点**:
- 高质量语义表示
- 支持多种模型
- 批量处理
- 需要网络连接

**核心参数**:
- `BaseURL`: API 基础 URL（默认 https://api.openai.com/v1）
- `APIKey`: API 密钥
- `Model`: 模型名称（默认 text-embedding-3-small）

**支持模型**:
- `text-embedding-3-small`: 1536 维，快速且经济
- `text-embedding-3-large`: 3072 维，更高精度
- `text-embedding-ada-002`: 1536 维，经典模型

**适用场景**:
- 需要高质量语义表示
- 有网络连接
- 可以接受 API 调用成本
- 复杂语义理解任务

**详细文档**: [OpenAI Embeddings.md](OpenAI Embeddings.md)

## 如何选择向量化方法

### 根据场景

| 场景 | 推荐方法 |
|-----|---------|
| 快速原型 | TF-IDF |
| 搜索引擎 | BM25 |
| 聊天记录检索 | BM25 |
| 复杂语义理解 | OpenAI Embeddings |
| 离线环境 | TF-IDF 或 BM25 |

### 根据性能要求

| 性能要求 | 推荐方法 |
|---------|---------|
| 最高性能 | TF-IDF |
| 平衡性能和质量 | BM25 |
| 最高质量 | OpenAI Embeddings |

### 根据资源限制

| 资源限制 | 推荐方法 |
|---------|---------|
| 无外部依赖 | TF-IDF、BM25 |
| 有网络连接 | OpenAI Embeddings |
| 需要增量更新 | BM25 |

## 使用示例

### TF-IDF 基础使用

```go
package main

import (
    "fmt"
    "github.com/adnilis/x-hmsw/embedding"
)

func main() {
    // 准备文档
    documents := []string{
        "机器学习是人工智能的一个分支",
        "深度学习使用多层神经网络",
        "自然语言处理处理文本数据",
    }

    // 创建 TF-IDF 向量化器
    config := embedding.DefaultTFIDFConfig()
    vectorizer := embedding.NewTFIDF(config)

    // 训练向量化器
    vectorizer.Fit(documents)

    // 转换单个文档
    vec := vectorizer.Transform("神经网络学习")
    fmt.Printf("向量维度: %d\n", len(vec))

    // 批量转换
    vectors := vectorizer.BatchTransform(documents)
    fmt.Printf("批量转换数量: %d\n", len(vectors))
}
```

### BM25 基础使用

```go
package main

import (
    "fmt"
    "github.com/adnilis/x-hmsw/embedding"
)

func main() {
    // 准备文档
    documents := []string{
        "机器学习是人工智能的一个分支",
        "深度学习使用多层神经网络",
        "自然语言处理处理文本数据",
    }

    // 创建 BM25 向量化器
    config := embedding.DefaultBM25Config()
    vectorizer := embedding.NewBM25(config)

    // 训练向量化器
    vectorizer.Fit(documents)

    // 转换单个文档
    vec := vectorizer.Transform("神经网络学习")
    fmt.Printf("向量维度: %d\n", len(vec))

    // 批量转换
    vectors := vectorizer.BatchTransform(documents)
    fmt.Printf("批量转换数量: %d\n", len(vectors))
}
```

### BM25 增量更新

```go
package main

import (
    "fmt"
    "github.com/adnilis/x-hmsw/embedding"
)

func main() {
    // 初始文档
    documents := []string{
        "机器学习是人工智能的一个分支",
        "深度学习使用多层神经网络",
    }

    // 创建 BM25 向量化器
    config := embedding.DefaultBM25Config()
    vectorizer := embedding.NewBM25(config)

    // 训练向量化器
    vectorizer.Fit(documents)

    // 增量添加新文档
    newDocuments := []string{
        "自然语言处理处理文本数据",
        "计算机视觉处理图像数据",
    }

    // 增量更新
    vectorizer.IncrementalFit(newDocuments)

    // 转换文档
    vec := vectorizer.Transform("神经网络学习")
    fmt.Printf("向量维度: %d\n", len(vec))
}
```

### OpenAI Embeddings 基础使用

```go
package main

import (
    "fmt"
    "os"
    "github.com/adnilis/x-hmsw/embedding"
)

func main() {
    // 设置环境变量
    os.Setenv("OPENAI_API_KEY", "your-api-key")
    os.Setenv("OPENAI_BASE_URL", "https://api.openai.com/v1")
    os.Setenv("OPENAI_MODEL", "text-embedding-3-small")

    // 创建 OpenAI Embedding 客户端
    config := embedding.DefaultConfig()
    client := embedding.NewOpenAIEmbedding(config)

    // 单个文本向量化
    text := "神经网络学习"
    vec, err := client.CreateEmbedding(text)
    if err != nil {
        panic(err)
    }
    fmt.Printf("向量维度: %d\n", len(vec))

    // 批量向量化
    texts := []string{
        "机器学习是人工智能的一个分支",
        "深度学习使用多层神经网络",
        "自然语言处理处理文本数据",
    }
    vectors, err := client.CreateBatchEmbedding(texts)
    if err != nil {
        panic(err)
    }
    fmt.Printf("批量转换数量: %d\n", len(vectors))
}
```

## 性能对比

基于以下配置的性能测试：

- 文档数量: 10,000
- 平均文档长度: 100 词
- 测试环境: Intel i7, 16GB RAM

| 方法 | 训练时间 | 转换时间 | 向量维度 | 内存占用 |
|-----|---------|---------|---------|---------|
| TF-IDF | 0.5s | 0.1ms | 10,000 | 80MB |
| BM25 | 0.6s | 0.15ms | 10,000 | 85MB |
| OpenAI Embeddings | N/A | 100ms | 1536 | 12MB |

## 最佳实践

### 1. 选择合适的向量化方法

- **快速原型**: 使用 TF-IDF
- **搜索引擎**: 使用 BM25
- **高质量语义**: 使用 OpenAI Embeddings

### 2. 参数调优

- **TF-IDF**: 调整 `MaxVocabSize` 控制向量维度
- **BM25**: 调整 `K1` 和 `B` 参数优化检索效果
- **OpenAI**: 选择合适的模型平衡性能和成本

### 3. 增量更新

- 对于动态数据集，使用 BM25 的增量更新功能
- 避免频繁重新训练

### 4. 批量处理

- 使用批量转换 API 提高效率
- OpenAI Embeddings 支持批量处理，减少 API 调用次数

## 相关资源

- [TF-IDF 文档](TF-IDF.md)
- [BM25 文档](BM25.md)
- [OpenAI Embeddings 文档](OpenAI Embeddings.md)
- [API 接口文档](../07-API接口/README.md)
- [示例代码](../../examples/)

---

**最后更新**: 2026年3月11日
