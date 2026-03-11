package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics 监控指标集合
type Metrics struct {
	// 向量数量
	vectorCount prometheus.Gauge

	// 插入操作
	insertCount   prometheus.Counter
	insertLatency prometheus.Histogram

	// 搜索操作
	searchCount   prometheus.Counter
	searchLatency prometheus.Histogram

	// 删除操作
	deleteCount prometheus.Counter

	// 存储指标
	storageSize prometheus.Gauge
	cacheHits   prometheus.Counter
	cacheMisses prometheus.Counter

	// 错误指标
	errorCount prometheus.Counter
}

// NewMetrics 创建监控指标
func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		vectorCount: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "vector_count",
			Help:      "Total number of vectors in the database",
		}),
		insertCount: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "insert_count",
			Help:      "Total number of insert operations",
		}),
		insertLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "insert_latency_seconds",
			Help:      "Insert operation latency",
			Buckets:   prometheus.DefBuckets,
		}),
		searchCount: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "search_count",
			Help:      "Total number of search operations",
		}),
		searchLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "search_latency_seconds",
			Help:      "Search operation latency",
			Buckets:   prometheus.DefBuckets,
		}),
		deleteCount: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "delete_count",
			Help:      "Total number of delete operations",
		}),
		storageSize: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "storage_size_bytes",
			Help:      "Storage size in bytes",
		}),
		cacheHits: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "cache_hits",
			Help:      "Number of cache hits",
		}),
		cacheMisses: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "cache_misses",
			Help:      "Number of cache misses",
		}),
		errorCount: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "error_count",
			Help:      "Total number of errors",
		}),
	}

	return m
}

// IncVectorCount 增加向量计数
func (m *Metrics) IncVectorCount() {
	m.vectorCount.Inc()
}

// DecVectorCount 减少向量计数
func (m *Metrics) DecVectorCount() {
	m.vectorCount.Dec()
}

// SetVectorCount 设置向量计数
func (m *Metrics) SetVectorCount(count float64) {
	m.vectorCount.Set(count)
}

// ObserveInsert 记录插入操作
func (m *Metrics) ObserveInsert(count int, duration float64) {
	m.insertCount.Add(float64(count))
	m.insertLatency.Observe(duration)
}

// ObserveSearch 记录搜索操作
func (m *Metrics) ObserveSearch(count int, duration float64) {
	m.searchCount.Add(float64(count))
	m.searchLatency.Observe(duration)
}

// IncDeleteCount 增加删除计数
func (m *Metrics) IncDeleteCount() {
	m.deleteCount.Inc()
}

// SetStorageSize 设置存储大小
func (m *Metrics) SetStorageSize(size float64) {
	m.storageSize.Set(size)
}

// IncCacheHit 增加缓存命中
func (m *Metrics) IncCacheHit() {
	m.cacheHits.Inc()
}

// IncCacheMiss 增加缓存未命中
func (m *Metrics) IncCacheMiss() {
	m.cacheMisses.Inc()
}

// IncErrorCount 增加错误计数
func (m *Metrics) IncErrorCount() {
	m.errorCount.Inc()
}

// GetMetrics 获取所有指标
func (m *Metrics) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"vector_count": m.vectorCount,
		"insert_count": m.insertCount,
		"search_count": m.searchCount,
		"delete_count": m.deleteCount,
		"cache_hits":   m.cacheHits,
		"cache_misses": m.cacheMisses,
		"error_count":  m.errorCount,
	}
}

// Collector 收集器接口
type Collector interface {
	Collect(ch chan<- prometheus.Metric)
	Describe(ch chan<- *prometheus.Desc)
}

// Register 注册收集器
func Register(c Collector) {
	prometheus.MustRegister(c)
}

// Unregister 注销收集器
func Unregister(c Collector) {
	prometheus.Unregister(c)
}
