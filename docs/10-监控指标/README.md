# 监控指标

x-hmsw 提供完善的监控指标，帮助了解系统运行状态和性能。

## 监控概览

x-hmsw 使用 Prometheus 作为监控指标收集系统，提供以下监控能力：

- **操作计数**: 插入、搜索、删除操作计数
- **性能指标**: 操作延迟、吞吐量
- **资源指标**: 向量数量、存储大小
- **缓存指标**: 缓存命中率
- **错误指标**: 错误计数

## Prometheus 指标

### 核心指标

| 指标名称 | 类型 | 描述 |
|---------|------|------|
| `vector_count` | Gauge | 当前向量数量 |
| `insert_count` | Counter | 插入操作总次数 |
| `insert_latency_seconds` | Histogram | 插入操作延迟分布 |
| `search_count` | Counter | 搜索操作总次数 |
| `search_latency_seconds` | Histogram | 搜索操作延迟分布 |
| `delete_count` | Counter | 删除操作总次数 |
| `storage_size_bytes` | Gauge | 存储占用字节数 |
| `cache_hits` | Counter | 缓存命中次数 |
| `cache_misses` | Counter | 缓存未命中次数 |
| `error_count` | Counter | 错误总次数 |

### 指标详解

#### vector_count

**类型**: Gauge

**描述**: 当前数据库中的向量总数

**示例**:
```
vector_count 10000
```

#### insert_count

**类型**: Counter

**描述**: 插入操作的总次数

**示例**:
```
insert_count 50000
```

#### insert_latency_seconds

**类型**: Histogram

**描述**: 插入操作的延迟分布（秒）

**标签**:
- `quantile`: 分位数 (0.5, 0.9, 0.99)

**示例**:
```
insert_latency_seconds{quantile="0.5"} 0.0001
insert_latency_seconds{quantile="0.9"} 0.0002
insert_latency_seconds{quantile="0.99"} 0.0005
insert_latency_seconds_sum 5.0
insert_latency_seconds_count 50000
```

#### search_count

**类型**: Counter

**描述**: 搜索操作的总次数

**示例**:
```
search_count 100000
```

#### search_latency_seconds

**类型**: Histogram

**描述**: 搜索操作的延迟分布（秒）

**标签**:
- `quantile`: 分位数 (0.5, 0.9, 0.99)

**示例**:
```
search_latency_seconds{quantile="0.5"} 0.0003
search_latency_seconds{quantile="0.9"} 0.0005
search_latency_seconds{quantile="0.99"} 0.001
search_latency_seconds_sum 30.0
search_latency_seconds_count 100000
```

#### delete_count

**类型**: Counter

**描述**: 删除操作的总次数

**示例**:
```
delete_count 1000
```

#### storage_size_bytes

**类型**: Gauge

**描述**: 存储占用的字节数

**示例**:
```
storage_size_bytes 52428800
```

#### cache_hits

**类型**: Counter

**描述**: 缓存命中的总次数

**示例**:
```
cache_hits 80000
```

#### cache_misses

**类型**: Counter

**描述**: 缓存未命中的总次数

**示例**:
```
cache_misses 20000
```

#### error_count

**类型**: Counter

**描述**: 错误的总次数

**示例**:
```
error_count 50
```

## 使用 Prometheus

### 启用监控

```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

// 启用监控
config := iface.Config{
    Dimension: 128,
    EnableMetrics: true,  // 启用监控
}
db, _ := iface.NewPureGoVectorDB(config)

// 暴露监控端点
http.Handle("/metrics", promhttp.Handler())
http.ListenAndServe(":9090", nil)
```

### 查询指标

访问 `http://localhost:9090/metrics` 查看所有指标。

### Prometheus 配置

创建 `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'x-hmsw'
    static_configs:
      - targets: ['localhost:9090']
```

启动 Prometheus:

```bash
prometheus --config.file=prometheus.yml
```

## Grafana 仪表板

### 创建仪表板

1. 添加 Prometheus 数据源
2. 创建新仪表板
3. 添加面板

### 示例面板

#### 向量数量

```promql
vector_count
```

#### 插入速率

```promql
rate(insert_count[5m])
```

#### 搜索速率

```promql
rate(search_count[5m])
```

#### 搜索延迟 P99

```promql
histogram_quantile(0.99, rate(search_latency_seconds_bucket[5m]))
```

#### 缓存命中率

```promql
rate(cache_hits[5m]) / (rate(cache_hits[5m]) + rate(cache_misses[5m]))
```

#### 错误率

```promql
rate(error_count[5m])
```

## 告警规则

### 创建告警规则

创建 `alerts.yml`:

```yaml
groups:
  - name: x-hmsw
    rules:
      # 高错误率告警
      - alert: HighErrorRate
        expr: rate(error_count[5m]) > 10
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "高错误率"
          description: "错误率超过 10 次/秒"

      # 高延迟告警
      - alert: HighSearchLatency
        expr: histogram_quantile(0.99, rate(search_latency_seconds_bucket[5m])) > 0.01
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "高搜索延迟"
          description: "搜索延迟 P99 超过 10ms"

      # 低缓存命中率告警
      - alert: LowCacheHitRate
        expr: rate(cache_hits[5m]) / (rate(cache_hits[5m]) + rate(cache_misses[5m])) < 0.8
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "低缓存命中率"
          description: "缓存命中率低于 80%"
```

### 配置 Prometheus 使用告警规则

更新 `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s

rule_files:
  - "alerts.yml"

scrape_configs:
  - job_name: 'x-hmsw'
    static_configs:
      - targets: ['localhost:9090']

alerting:
  alertmanagers:
    - static_configs:
        - targets: ['localhost:9093']
```

## 监控最佳实践

### 1. 设置合理的采样间隔

```yaml
scrape_interval: 15s  # 15秒采样一次
```

### 2. 使用合适的聚合窗口

```promql
# 5分钟窗口
rate(insert_count[5m])

# 1小时窗口
rate(insert_count[1h])
```

### 3. 关注关键指标

- **性能**: 搜索延迟、插入延迟
- **吞吐量**: 搜索速率、插入速率
- **资源**: 向量数量、存储大小
- **稳定性**: 错误率、缓存命中率

### 4. 设置合理的告警阈值

根据实际业务需求设置告警阈值，避免告警风暴。

### 5. 定期审查告警规则

定期审查告警规则，确保告警的有效性。

## 相关文档

- [Prometheus](Prometheus.md) - Prometheus 详细配置
- [性能优化](../09-性能优化/README.md) - 性能优化
- [架构设计](../03-架构设计.md) - 系统架构

---

**最后更新**: 2026年3月10日
