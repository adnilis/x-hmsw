# 工具函数

x-hmsw 提供丰富的工具函数，支持各种计算和优化操作。

## 工具函数概览

| 工具类别 | 功能 | 性能影响 |
|---------|------|---------|
| **数学函数** | 距离计算、向量运算 | 高 |
| **位集合** | 高效位操作 | 中 |
| **预取优化** | 内存访问优化 | 中 |
| **并发控制** | 并发安全操作 | 低 |

## 工具函数详解

### 数学函数

**数学函数** 提供向量计算和距离计算功能。

**功能**:
- 距离计算（余弦、L2、内积）
- 向量运算（加、减、乘、除）
- 归一化
- SIMD 加速

**详细文档**: [数学函数.md](数学函数.md)

### 位集合

**位集合** 提供高效的位操作功能。

**功能**:
- 位设置和清除
- 位查询
- 位运算
- 内存优化

**详细文档**: [位集合.md](位集合.md)

### 预取优化

**预取优化** 提供内存访问优化功能。

**功能**:
- 数据预取
- 缓存优化
- 减少缓存未命中

**详细文档**: [预取优化.md](预取优化.md)

## 使用示例

### 数学函数

```go
import "github.com/yourusername/x-hmsw/utils/math"

// 计算余弦距离
distance := math.CosineDistance(vec1, vec2)

// 计算L2距离
distance := math.L2Distance(vec1, vec2)

// 计算内积
product := math.InnerProduct(vec1, vec2)

// 向量归一化
normalized := math.Normalize(vec)
```

### 位集合

```go
import "github.com/yourusername/x-hmsw/utils/bitset"

// 创建位集合
bs := bitset.New(1000)

// 设置位
bs.Set(10)

// 清除位
bs.Clear(10)

// 查询位
if bs.Test(10) {
    // 位已设置
}

// 位运算
bs.And(other)
bs.Or(other)
bs.Xor(other)
```

### 预取优化

```go
import "github.com/yourusername/x-hmsw/utils/prefetch"

// 预取数据
prefetch.Prefetch(data)

// 预取到 L1 缓存
prefetch.PrefetchToL1(data)

// 预取到 L2 缓存
prefetch.PrefetchToL2(data)
```

## 性能优化

### SIMD 加速

数学函数使用 SIMD 指令加速计算：

```go
// 自动使用 SIMD 加速
distance := math.CosineDistance(vec1, vec2)
```

### 对象池

使用对象池减少内存分配：

```go
// 使用对象池
pool := math.NewDistanceCalculatorPool()
calculator := pool.Get()
defer pool.Put(calculator)

distance := calculator.Calculate(vec1, vec2)
```

### 并发安全

工具函数支持并发使用：

```go
// 并发计算
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        distance := math.CosineDistance(vec1, vec2)
    }()
}
wg.Wait()
```

## 最佳实践

### 1. 选择合适的距离度量

| 距离度量 | 适用场景 | 性能 |
|---------|---------|------|
| 余弦距离 | 文本、推荐 | 高 |
| L2 距离 | 图像、音频 | 高 |
| 内积 | 推荐系统 | 最高 |

### 2. 使用 SIMD 加速

确保使用支持 SIMD 的函数：

```go
// 推荐：使用 SIMD 加速
distance := math.CosineDistance(vec1, vec2)

// 不推荐：手动计算
distance := manualCosineDistance(vec1, vec2)
```

### 3. 使用对象池

对于频繁操作，使用对象池：

```go
// 推荐：使用对象池
pool := math.NewDistanceCalculatorPool()
calculator := pool.Get()
defer pool.Put(calculator)

// 不推荐：每次创建新对象
calculator := math.NewDistanceCalculator()
```

### 4. 预取数据

对于大规模数据，使用预取：

```go
// 推荐：预取数据
prefetch.Prefetch(data)
process(data)

// 不推荐：直接处理
process(data)
```

## 性能对比

### 距离计算性能

| 距离度量 | 无 SIMD | 有 SIMD | 提升 |
|---------|---------|---------|------|
| 余弦距离 | 100ns | 75ns | 25% |
| L2 距离 | 120ns | 90ns | 25% |
| 内积 | 80ns | 60ns | 25% |

### 位集合性能

| 操作 | 数组 | 位集合 | 提升 |
|-----|------|--------|------|
| 设置 | 10ns | 5ns | 50% |
| 查询 | 10ns | 5ns | 50% |
| 运算 | 100ns | 50ns | 50% |

## 相关文档

- [数学函数](数学函数.md) - 数学函数详解
- [位集合](位集合.md) - 位集合详解
- [预取优化](预取优化.md) - 预取优化详解
- [性能优化](../09-性能优化/README.md) - 性能优化

---

**最后更新**: 2026年3月10日
