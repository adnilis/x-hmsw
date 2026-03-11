# x-hmsw 向量数据库文档

> 纯 Go 实现的高性能向量数据库

## 📖 文档目录

### 快速开始
- [项目概述](01-项目概述.md) - 了解 x-hmsw 的功能特性和技术架构
- [快速开始](02-快速开始.md) - 5分钟快速上手指南
- [架构设计](03-架构设计.md) - 系统架构和设计理念

### 核心功能
- [索引算法](04-索引算法/README.md) - HNSW、IVF、Flat、ANN 等索引算法详解
- [存储引擎](05-存储引擎/README.md) - 多种存储引擎的实现和选择
- [向量压缩](06-向量压缩/README.md) - PQ、SQ、Binary 等压缩技术
- [文本向量化](13-文本向量化/README.md) - TF-IDF、BM25、OpenAI Embeddings 等向量化方法

### API 接口
- [API 接口](07-API接口/README.md) - QuickDB 和核心接口使用指南

### 高级特性
- [序列化](08-序列化/README.md) - 数据序列化方案
- [性能优化](09-性能优化/README.md) - 性能优化技巧和最佳实践
- [监控指标](10-监控指标/README.md) - Prometheus 监控集成

### 工具和类型
- [工具函数](11-工具函数/README.md) - 数学工具、位集合等
- [类型定义](12-类型定义/README.md) - 核心数据类型定义

## 🚀 快速链接

### 核心接口
- [QuickDB API](07-API接口/QuickDB.md) - 简化的向量数据库接口
- [VectorDB 接口](07-API接口/VectorDB接口.md) - 核心数据库接口
- [Index 接口](07-API接口/Index接口.md) - 索引接口定义

### 索引算法
- [HNSW 索引](04-索引算法/HNSW索引.md) - 分层小世界图索引
- [IVF 索引](04-索引算法/IVF索引.md) - 倒排文件索引
- [Flat 索引](04-索引算法/Flat索引.md) - 暴力搜索索引

### 存储引擎
- [BadgerDB](05-存储引擎/BadgerDB.md) - 基于 LSM 的键值存储
- [BBolt](05-存储引擎/BBolt.md) - 嵌入式键值存储
- [PebbleDB](05-存储引擎/PebbleDB.md) - CockroachDB 的存储引擎

### 性能报告
- [性能优化报告](09-性能优化/性能报告.md) - 详细的性能测试和优化结果

## 📊 项目特性

- **多索引支持**: HNSW、IVF、Flat、ANN 等多种索引算法
- **灵活存储**: 支持 Badger、BBolt、Pebble、内存、Mmap 等存储引擎
- **文本向量化**: 内置 TF-IDF、BM25、OpenAI Embeddings 等向量化方法
- **向量压缩**: PQ、SQ、Binary 等压缩技术，降低内存占用
- **高性能**: SIMD 加速、对象池优化、并发控制
- **易用性**: QuickDB 提供开箱即用的简化接口
- **监控**: 内置 Prometheus 指标支持

## 🛠 技术栈

- **语言**: Go 1.26.0
- **索引**: HNSW、IVF、Flat
- **存储**: BadgerDB、BBolt、PebbleDB
- **压缩**: Product Quantization、Scalar Quantization、Binary Quantization
- **序列化**: Protobuf、MessagePack、Binary
- **监控**: Prometheus
- **日志**: uber/zap

## 📝 文档说明

本文档采用标准 Markdown 格式，包含：
- 详细的 API 说明和代码示例
- 架构设计和实现原理
- 性能优化技巧和最佳实践
- 完整的目录导航和交叉链接

## 🔗 相关资源

- [GitHub 仓库](https://github.com/adnilis/x-hmsw)
- [示例代码](../examples/)

---

**最后更新**: 2026年3月11日
