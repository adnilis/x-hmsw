# 存储引擎

x-hmsw 支持多种存储引擎，可根据场景选择最适合的方案。

## 存储引擎概览

| 存储引擎 | 性能 | 持久化 | 内存占用 | 适用场景 |
|---------|------|--------|---------|---------|
| **BadgerDB** | 高 | 是 | 中 | 通用场景，性能均衡 |
| **BBolt** | 中 | 是 | 低 | 单机应用，简单可靠 |
| **PebbleDB** | 高 | 是 | 中 | 高写入场景 |
| **Memory** | 最高 | 否 | 高 | 需要最快访问速度 |
| **Mmap** | 高 | 是 | 中 | 平衡性能和持久化 |

## 存储引擎详解

### BadgerDB

**BadgerDB** 是基于 LSM 树的高性能键值存储。

**特点**:
- 高性能读写
- 支持事务
- 自动压缩
- 适合通用场景

**适用场景**:
- 通用键值存储
- 需要事务支持
- 读写均衡的场景

**详细文档**: [BadgerDB.md](BadgerDB.md)

### BBolt

**BBolt** 是嵌入式键值存储，基于 BoltDB。

**特点**:
- 简单可靠
- 单文件存储
- ACID 事务
- 适合单机应用

**适用场景**:
- 单机应用
- 配置存储
- 小规模数据

**详细文档**: [BBolt.md](BBolt.md)

### PebbleDB

**PebbleDB** 是 CockroachDB 的存储引擎。

**特点**:
- 优秀的写入性能
- 基于 LSM 树
- 支持并发
- 适合高写入场景

**适用场景**:
- 高写入场景
- 大规模数据
- 需要高吞吐量

**详细文档**: [PebbleDB.md](PebbleDB.md)

### Memory

**Memory** 是纯内存存储。

**特点**:
- 最快的访问速度
- 无持久化
- 数据易丢失
- 适合缓存场景

**适用场景**:
- 缓存
- 临时数据
- 可以接受数据丢失

**详细文档**: [内存存储.md](内存存储.md)

### Mmap

**Mmap** 是内存映射文件存储。

**特点**:
- 接近内存性能
- 支持持久化
- 操作系统管理缓存
- 平衡性能和持久化

**适用场景**:
- 需要持久化
- 希望接近内存性能
- 大规模数据

**详细文档**: [内存映射.md](内存映射.md)

## 其他存储功能

### 写前日志 (WAL)

**WAL (Write-Ahead Logging)** 是写前日志机制。

**特点**:
- 保证数据持久性
- 支持崩溃恢复
- 提高写入性能

**详细文档**: [写前日志.md](写前日志.md)

### 备份恢复

**备份恢复** 功能支持数据备份和恢复。

**特点**:
- 支持全量备份
- 支持增量备份
- 支持数据恢复
- 自动备份功能

**详细文档**: [备份恢复.md](备份恢复.md)

### 存储验证

**存储验证** 功能验证数据完整性。

**特点**:
- 检查数据损坏
- 验证索引一致性
- 支持自动修复

**详细文档**: [存储验证.md](存储验证.md)

## 如何选择存储引擎

### 根据持久化需求

| 持久化需求 | 推荐存储 |
|-----------|---------|
| 需要持久化 | BadgerDB, BBolt, PebbleDB, Mmap |
| 不需要持久化 | Memory |

### 根据性能需求

| 性能需求 | 推荐存储 |
|---------|---------|
| 最高性能 | Memory |
| 高性能 | BadgerDB, PebbleDB, Mmap |
| 中等性能 | BBolt |

### 根据写入场景

| 写入场景 | 推荐存储 |
|---------|---------|
| 高写入 | PebbleDB |
| 读写均衡 | BadgerDB |
| 低写入 | BBolt, Mmap |

### 根据部署环境

| 部署环境 | 推荐存储 |
|---------|---------|
| 单机应用 | BBolt |
| 分布式应用 | BadgerDB, PebbleDB |
| 容器化应用 | Memory, Mmap |

## 性能对比

基于以下配置的性能测试：
- 向量维度: 128
- 数据规模: 10,000 向量
- 索引类型: HNSW

| 存储引擎 | 插入时间 | 平均搜索时间 | QPS |
|---------|---------|-------------|-----|
| Memory | 6.100s | 289µs | 3,457 |
| BadgerDB | 6.424s | 554µs | 1,807 |
| BBolt | 6.324s | 527µs | 1,897 |
| PebbleDB | 6.512s | 541µs | 1,848 |
| Mmap | 6.234s | 312µs | 3,205 |

## 使用示例

### 使用 BadgerDB

```go
config := iface.Config{
    Dimension:   128,
    StorageType: iface.Badger,
    StoragePath: "./data/badger",
}

db, _ := iface.NewPureGoVectorDB(config)
```

### 使用 BBolt

```go
config := iface.Config{
    Dimension:   128,
    StorageType: iface.BBolt,
    StoragePath: "./data/bbolt.db",
}

db, _ := iface.NewPureGoVectorDB(config)
```

### 使用 PebbleDB

```go
config := iface.Config{
    Dimension:   128,
    StorageType: iface.Pebble,
    StoragePath: "./data/pebble",
}

db, _ := iface.NewPureGoVectorDB(config)
```

### 使用 Memory

```go
config := iface.Config{
    Dimension:   128,
    StorageType: iface.Memory,
}

db, _ := iface.NewPureGoVectorDB(config)
```

### 使用 Mmap

```go
config := iface.Config{
    Dimension:   128,
    StorageType: iface.MMap,
    StoragePath: "./data/mmap",
}

db, _ := iface.NewPureGoVectorDB(config)
```

## 自动保存功能

x-hmsw 提供自动保存功能，定期将数据持久化到磁盘。

### 启用自动保存

```go
// 使用 QuickDB 启用自动保存（默认 30 秒间隔）
db, _ := api.NewQuick("./data")

// 自定义自动保存间隔（1 分钟）
db, _ := api.NewQuickWithConfig("./data", 1*time.Minute)
```

### 自动保存特点

- **定时保存**: 按指定间隔自动保存数据
- **增量保存**: 只保存变更的数据
- **崩溃恢复**: 支持从自动保存点恢复
- **性能优化**: 异步保存，不影响主流程

## 存储引擎配置

### BadgerDB 配置

```go
// BadgerDB 特定配置
config.StorageOptions = map[string]interface{}{
    "ValueLogFileSize": 1 << 30,  // 1GB
    "MemTableSize":     64 << 20, // 64MB
}
```

### BBolt 配置

```go
// BBolt 特定配置
config.StorageOptions = map[string]interface{}{
    "NoSync": false,  // 是否禁用 fsync
    "Timeout": 30,    // 超时时间（秒）
}
```

### PebbleDB 配置

```go
// PebbleDB 特定配置
config.StorageOptions = map[string]interface{}{
    "CacheSize": 256 << 20,  // 256MB 缓存
    "MemTableSize": 64 << 20, // 64MB MemTable
}
```

## 最佳实践

1. **通用场景**: 使用 BadgerDB，性能均衡
2. **单机应用**: 使用 BBolt，简单可靠
3. **高写入场景**: 使用 PebbleDB，高吞吐量
4. **缓存场景**: 使用 Memory，最快速度
5. **需要持久化**: 使用 Mmap，平衡性能和持久化

## 存储引擎对比

### BadgerDB vs BBolt

| 特性 | BadgerDB | BBolt |
|-----|----------|-------|
| 性能 | 高 | 中 |
| 事务 | 支持 | 支持 |
| 并发 | 高 | 低 |
| 文件数 | 多 | 单文件 |
| 适用场景 | 通用 | 单机 |

### BadgerDB vs PebbleDB

| 特性 | BadgerDB | PebbleDB |
|-----|----------|----------|
| 性能 | 高 | 高 |
| 写入性能 | 高 | 更高 |
| 成熟度 | 高 | 中 |
| 适用场景 | 通用 | 高写入 |

### Memory vs Mmap

| 特性 | Memory | Mmap |
|-----|--------|------|
| 性能 | 最高 | 高 |
| 持久化 | 否 | 是 |
| 内存占用 | 高 | 中 |
| 适用场景 | 缓存 | 持久化 |

## 相关文档

- [BadgerDB](BadgerDB.md) - BadgerDB 详解
- [BBolt](BBolt.md) - BBolt 详解
- [PebbleDB](PebbleDB.md) - PebbleDB 详解
- [内存存储](内存存储.md) - 内存存储详解
- [内存映射](内存映射.md) - 内存映射详解
- [写前日志](写前日志.md) - WAL 机制
- [备份恢复](备份恢复.md) - 备份功能
- [存储验证](存储验证.md) - 存储验证
- [架构设计](../03-架构设计.md) - 系统架构

---

**最后更新**: 2026年3月10日
