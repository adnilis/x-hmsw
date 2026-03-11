# 性能优化

x-hmsw 通过多种优化手段提供高性能的向量搜索能力。

## 优化技术概览

| 优化技术 | 性能提升 | 复杂度 | 适用场景 |
|---------|---------|--------|---------|
| **SIMD 加速** | 20-30% | 低 | 距离计算密集型 |
| **对象池** | 10-20% | 低 | 频繁内存分配 |
| **并发控制** | 2-4x | 中 | 多核环境 |
| **预取优化** | 5-10% | 中 | 大规模数据 |
| **向量压缩** | 降低内存 | 中 | 内存受限 |

## 优化技术详解

### SIMD 加速

**SIMD (Single Instruction Multiple Data)** 使用向量指令集加速计算。

**特点**:
- 显著提升计算性能
- 支持 AVX2、SSE、NEON
- 适用于距离计算

**性能提升**: 20-30%

**详细文档**: [SIMD优化.md](SIMD优化.md)

### 对象池

**对象池** 复用临时对象，减少内存分配和 GC 压力。

**特点**:
- 减少内存分配
- 降低 GC 压力
- 提升整体性能

**性能提升**: 10-20%

**详细文档**: [对象池.md](对象池.md)

### 并发优化

**并发优化** 使用多线程并行处理，提升吞吐量。

**特点**:
- 充分利用多核
- 提升吞吐量
- 细粒度锁控制

**性能提升**: 2-4x

**详细文档**: [并发优化.md](并发优化.md)

### 预取优化

**预取优化** 提前加载数据，减少缓存未命中。

**特点**:
- 减少缓存未命中
- 提升内存访问效率
- 适用于大规模数据

**性能提升**: 5-10%

**详细文档**: [预取优化.md](预取优化.md)

## 性能报告

### 优化前后对比

基于以下配置的性能测试：
- 向量维度: 128
- 数据规模: 10,000 向量
- 查询次数: 100 次
- Top-K: 10

| 指标 | 优化前 | 优化后 | 提升 |
|-----|--------|--------|------|
| 插入时间 | 9.576s | 6.100s | 36.3% |
| 平均搜索时间 | 362µs | 289µs | 20.2% |
| QPS | 2,765 | 3,457 | 25.0% |
| 内存占用 | 5,635kB | 4,610kB | 18.2% |

### 各存储引擎性能

| 存储引擎 | 插入时间 | 搜索时间 | QPS |
|---------|---------|---------|-----|
| Memory | 6.100s | 289µs | 3,457 |
| BadgerDB | 6.424s | 554µs | 1,807 |
| BBolt | 6.324s | 527µs | 1,897 |
| PebbleDB | 6.512s | 541µs | 1,848 |
| Mmap | 6.234s | 312µs | 3,205 |

### 各索引算法性能

| 索引算法 | 搜索时间 | QPS | 精度 |
|---------|---------|-----|------|
| HNSW | 289µs | 3,457 | 0.8368 |
| IVF | 312µs | 3,205 | 0.8123 |
| Flat | 1,245µs | 803 | 1.0000 |

## 优化建议

### 索引选择

1. **小数据集** (< 10,000): 使用 Flat 索引
2. **中等数据集** (10,000 - 1,000,000): 使用 HNSW 索引
3. **大数据集** (> 1,000,000): 使用 IVF 索引

### 存储选择

1. **高性能需求**: 使用 Memory 或 Mmap
2. **通用场景**: 使用 BadgerDB
3. **高写入场景**: 使用 PebbleDB
4. **单机应用**: 使用 BBolt

### 参数调优

#### HNSW 参数

```go
// 高精度配置
M: 32
EfConstruction: 400
EfSearch: 200

// 高性能配置
M: 16
EfConstruction: 200
EfSearch: 100

// 平衡配置（推荐）
M: 16
EfConstruction: 200
EfSearch: 100
```

#### IVF 参数

```go
// 高精度配置
NumClusters: 200
Nprobe: 20

// 高性能配置
NumClusters: 100
Nprobe: 5

// 平衡配置（推荐）
NumClusters: 100
Nprobe: 10
```

### 批量操作

使用批量操作而非单条操作：

```go
// 推荐：批量插入
vectors := []types.Vector{...}
db.Insert(vectors)

// 不推荐：单条插入
for _, v := range vectors {
    db.InsertOne(v)
}
```

### 向量压缩

使用向量压缩降低内存占用：

```go
config := iface.Config{
    Dimension:   128,
    Compression: iface.SQ,  // 使用标量量化
}
```

## 性能监控

### Prometheus 指标

x-hmsw 内置 Prometheus 监控指标：

- `vector_count`: 向量数量
- `insert_count`: 插入操作计数
- `insert_latency_seconds`: 插入延迟
- `search_count`: 搜索操作计数
- `search_latency_seconds`: 搜索延迟
- `delete_count`: 删除操作计数
- `storage_size_bytes`: 存储大小
- `cache_hits`: 缓存命中数
- `cache_misses`: 缓存未命中数
- `error_count`: 错误计数

详细文档: [Prometheus.md](../10-监控指标/Prometheus.md)

### 性能分析

使用 Go pprof 进行性能分析：

```bash
# CPU 性能分析
go tool pprof http://localhost:6060/debug/pprof/profile

# 内存分析
go tool pprof http://localhost:6060/debug/pprof/heap

# 协程分析
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

## 最佳实践

1. **选择合适的索引**: 根据数据规模选择索引类型
2. **选择合适的存储**: 根据场景选择存储引擎
3. **使用批量操作**: 减少操作开销
4. **启用向量压缩**: 降低内存占用
5. **调整索引参数**: 根据需求调整参数
6. **监控性能指标**: 使用 Prometheus 监控
7. **定期性能分析**: 使用 pprof 分析性能瓶颈

## 性能测试

### 运行性能测试

```bash
# 运行所有性能测试
go test -bench=. -benchmem ./...

# 运行特定测试
go test -bench=BenchmarkHNSW -benchmem ./indexes/hnsw/

# 运行性能分析
go test -cpuprofile=cpu.prof -bench=. -benchmem ./...
go tool pprof cpu.prof
```

### 性能测试示例

```go
func BenchmarkHNSWInsert(b *testing.B) {
    config := iface.Config{
        Dimension: 128,
        IndexType: iface.HNSW,
    }
    db, _ := iface.NewPureGoVectorDB(config)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        vector := types.Vector{
            ID:     fmt.Sprintf("vec_%d", i),
            Vector: generateVector(128),
        }
        db.InsertOne(vector)
    }
}
```

## 相关文档

- [SIMD优化](SIMD优化.md) - SIMD 加速详解
- [对象池](对象池.md) - 对象池详解
- [并发优化](并发优化.md) - 并发优化详解
- [预取优化](预取优化.md) - 预取优化详解
- [性能报告](性能报告.md) - 详细性能测试报告
- [监控指标](../10-监控指标/README.md) - 监控指标

---

**最后更新**: 2026年3月10日
