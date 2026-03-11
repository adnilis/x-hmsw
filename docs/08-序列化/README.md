# 序列化

x-hmsw 支持多种数据序列化格式，满足不同场景的需求。

## 序列化格式概览

| 序列化格式 | 性能 | 大小 | 兼容性 | 适用场景 |
|-----------|------|------|--------|---------|
| **Protobuf** | 高 | 小 | 高 | 跨语言、高性能 |
| **MessagePack** | 高 | 小 | 中 | 轻量级、快速 |
| **Binary** | 最高 | 最小 | 低 | 纯 Go 环境 |

## 序列化格式详解

### Protobuf

**Protocol Buffers** 是 Google 开发的二进制序列化格式。

**特点**:
- 高性能
- 小体积
- 跨语言支持
- 向后兼容

**适用场景**:
- 跨语言通信
- 需要高性能
- 需要小体积

**详细文档**: [Protobuf.md](Protobuf.md)

### MessagePack

**MessagePack** 是高效的二进制序列化格式。

**特点**:
- 高性能
- 小体积
- 简单易用
- 支持多种语言

**适用场景**:
- 轻量级应用
- 需要快速序列化
- JSON 替代方案

**详细文档**: [MessagePack.md](MessagePack.md)

### 二进制序列化

**Binary** 是纯 Go 的二进制序列化格式。

**特点**:
- 最高性能
- 最小体积
- 仅支持 Go
- 简单直接

**适用场景**:
- 纯 Go 环境
- 需要最高性能
- 不需要跨语言

**详细文档**: [二进制序列化.md](二进制序列化.md)

## 如何选择序列化格式

### 根据性能需求

| 性能需求 | 推荐格式 |
|---------|---------|
| 最高性能 | Binary |
| 高性能 | Protobuf, MessagePack |

### 根据跨语言需求

| 跨语言需求 | 推荐格式 |
|-----------|---------|
| 需要跨语言 | Protobuf, MessagePack |
| 纯 Go 环境 | Binary |

### 根据体积需求

| 体积需求 | 推荐格式 |
|---------|---------|
| 最小体积 | Binary |
| 小体积 | Protobuf, MessagePack |

## 性能对比

基于以下配置的性能测试：
- 数据规模: 10,000 个向量
- 向量维度: 128

| 序列化格式 | 序列化时间 | 反序列化时间 | 数据大小 |
|-----------|-----------|-------------|---------|
| Binary | 12ms | 8ms | 5.12 MB |
| MessagePack | 18ms | 15ms | 5.45 MB |
| Protobuf | 22ms | 19ms | 5.28 MB |

## 使用示例

### 使用 Protobuf

```go
// 序列化
data, err := serialization.EncodeProtobuf(vector)

// 反序列化
vector, err := serialization.DecodeProtobuf(data)
```

### 使用 MessagePack

```go
// 序列化
data, err := serialization.EncodeMsgpack(vector)

// 反序列化
vector, err := serialization.DecodeMsgpack(data)
```

### 使用 Binary

```go
// 序列化
data, err := serialization.EncodeBinary(vector)

// 反序列化
vector, err := serialization.DecodeBinary(data)
```

## 序列化与存储的结合

序列化格式可以与存储引擎结合使用：

```go
config := iface.Config{
    Dimension:      128,
    StorageType:    iface.Badger,
    Serialization:  iface.Protobuf,  // 使用 Protobuf 序列化
}
```

## 最佳实践

1. **跨语言场景**: 使用 Protobuf，兼容性好
2. **纯 Go 环境**: 使用 Binary，性能最高
3. **轻量级应用**: 使用 MessagePack，简单高效
4. **需要兼容性**: 使用 Protobuf，向后兼容

## 序列化格式对比

### Protobuf vs MessagePack

| 特性 | Protobuf | MessagePack |
|-----|----------|------------|
| 性能 | 高 | 高 |
| 大小 | 小 | 小 |
| 兼容性 | 高 | 中 |
| Schema | 需要 | 不需要 |
| 适用场景 | 跨语言 | 轻量级 |

### Protobuf vs Binary

| 特性 | Protobuf | Binary |
|-----|----------|--------|
| 性能 | 高 | 最高 |
| 大小 | 小 | 最小 |
| 兼容性 | 高 | 低 |
| Schema | 需要 | 不需要 |
| 适用场景 | 跨语言 | 纯 Go |

### MessagePack vs Binary

| 特性 | MessagePack | Binary |
|-----|------------|--------|
| 性能 | 高 | 最高 |
| 大小 | 小 | 最小 |
| 兼容性 | 中 | 低 |
| Schema | 不需要 | 不需要 |
| 适用场景 | 轻量级 | 纯 Go |

## 相关文档

- [Protobuf](Protobuf.md) - Protobuf 序列化详解
- [MessagePack](MessagePack.md) - MessagePack 序列化详解
- [二进制序列化](二进制序列化.md) - Binary 序列化详解
- [架构设计](../03-架构设计.md) - 系统架构

---

**最后更新**: 2026年3月10日
