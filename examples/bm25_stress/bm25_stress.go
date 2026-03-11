package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"sort"
	"time"

	"github.com/adnilis/x-hmsw/embedding"
)

// 测试配置
type TestConfig struct {
	Name       string
	DocCount   int
	DocLength  int
	QueryCount int
}

// 测试结果
type TestResult struct {
	Name          string
	FitTime       time.Duration
	TransformTime time.Duration
	BatchTime     time.Duration
	MemoryMB      float64
	VocabSize     int
	AvgQueryTime  time.Duration
	QPS           float64
}

var (
	// 词库
	words = []string{
		"go", "python", "java", "javascript", "rust", "c++", "csharp",
		"编程", "语言", "开发", "数据", "算法", "系统", "性能", "优化",
		"搜索", "索引", "向量", "数据库", "机器学习", "人工智能", "深度学习",
		"神经网络", "自然语言", "计算机视觉", "推荐系统", "分布式", "微服务",
		"云计算", "大数据", "区块链", "网络安全", "前端", "后端", "全栈",
		"测试", "部署", "运维", "监控", "日志", "缓存", "消息队列",
		"api", "rest", "graphql", "grpc", "websocket", "http", "https",
		"docker", "kubernetes", "容器", "虚拟化", "服务器", "客户端",
		"框架", "库", "工具", "平台", "架构", "设计", "模式", "最佳实践",
	}

	// 测试配置
	testConfigs = []TestConfig{
		{"小规模", 1000, 50, 1000},
		{"中规模", 5000, 100, 2000},
		{"大规模", 10000, 150, 5000},
		{"超大规模", 20000, 200, 10000},
	}

	// 查询词
	queries = []string{
		"编程语言开发",
		"机器学习算法",
		"系统性能优化",
		"数据库索引",
		"人工智能应用",
		"分布式架构设计",
		"云计算平台",
		"网络安全防护",
		"前端框架开发",
		"后端服务优化",
	}
)

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║           BM25 全变体性能压测系统                              ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 打印系统信息
	printSystemInfo()

	// 测试所有BM25变体
	variants := []string{"bm25", "bm25f", "bm25l", "bm25plus"}

	for _, variant := range variants {
		fmt.Printf("\n╔════════════════════════════════════════════════════════════════╗\n")
		fmt.Printf("║  %s 变体测试（优化版）\n", variant)
		fmt.Printf("╚════════════════════════════════════════════════════════════════╝\n")

		results := make([]TestResult, 0, len(testConfigs))

		for _, config := range testConfigs {
			result := runTest(variant, config)
			results = append(results, result)
		}

		// 打印汇总
		printSummary(variant, results)
	}

	// 对比所有变体
	fmt.Printf("\n╔════════════════════════════════════════════════════════════════╗\n")
	fmt.Println("║  所有变体性能对比")
	fmt.Printf("╚════════════════════════════════════════════════════════════════╝\n")
	compareAllVariants()
}

func printSystemInfo() {
	fmt.Println("【系统信息】")
	fmt.Printf("  Go 版本: %s\n", runtime.Version())
	fmt.Printf("  CPU 核心数: %d\n", runtime.NumCPU())
	fmt.Printf("  操作系统: %s\n", runtime.GOOS)
	fmt.Printf("  架构: %s\n", runtime.GOARCH)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("  初始内存: %.2f MB\n", float64(m.HeapAlloc)/1024/1024)
	fmt.Println()
}

func runTest(variant string, config TestConfig) TestResult {
	fmt.Printf("\n▶ [%s] %d 文档, 平均长度 %d, %d 次查询\n",
		config.Name, config.DocCount, config.DocLength, config.QueryCount)

	result := TestResult{Name: config.Name}

	// 生成测试数据
	docs := generateDocs(config.DocCount, config.DocLength)
	runtime.GC()

	switch variant {
	case "bm25":
		cfg := embedding.DefaultBM25Config()
		cfg.MaxVocabSize = 50000
		vectorizer := embedding.NewBM25Vectorizer(cfg)

		// 训练
		runtime.GC()
		start := time.Now()
		vectorizer.Fit(docs)
		result.FitTime = time.Since(start)
		result.VocabSize = vectorizer.GetVocabularySize()

		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		result.MemoryMB = float64(m.HeapAlloc) / 1024 / 1024

		fmt.Printf("  ✓ 训练完成: %v (词汇量: %d, 内存: %.2f MB)\n",
			result.FitTime, result.VocabSize, result.MemoryMB)

		// 单次查询测试
		runtime.GC()
		totalQueryTime := time.Duration(0)
		for i := 0; i < config.QueryCount; i++ {
			query := queries[i%len(queries)]
			start := time.Now()
			_ = vectorizer.Transform(query)
			totalQueryTime += time.Since(start)
		}
		result.TransformTime = totalQueryTime
		result.AvgQueryTime = totalQueryTime / time.Duration(config.QueryCount)
		result.QPS = float64(config.QueryCount) / totalQueryTime.Seconds()

		fmt.Printf("  ✓ %d 次查询: %v (平均: %v, QPS: %.2f)\n",
			config.QueryCount, result.TransformTime, result.AvgQueryTime, result.QPS)

		// 批量查询测试
		batchSize := min(100, len(docs))
		runtime.GC()
		start = time.Now()
		_ = vectorizer.BatchTransform(docs[:batchSize])
		result.BatchTime = time.Since(start)

		fmt.Printf("  ✓ 批量 %d 文档: %v\n", batchSize, result.BatchTime)

	case "bm25f":
		cfg := embedding.BM25FConfig{
			BM25Config:   embedding.BM25Config{MaxVocabSize: 50000},
			FieldWeights: map[string]float64{"title": 2.0, "content": 1.0, "tags": 1.5},
		}
		vectorizer := embedding.NewBM25FVectorizer(cfg)
		multiDocs := convertToMultiField(docs)

		// 训练
		runtime.GC()
		start := time.Now()
		vectorizer.Fit(multiDocs)
		result.FitTime = time.Since(start)
		result.VocabSize = vectorizer.GetVocabularySize()

		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		result.MemoryMB = float64(m.HeapAlloc) / 1024 / 1024

		fmt.Printf("  ✓ 训练完成: %v (词汇量: %d, 内存: %.2f MB)\n",
			result.FitTime, result.VocabSize, result.MemoryMB)

		// 单次查询测试 - BM25F需要多字段查询
		runtime.GC()
		totalQueryTime := time.Duration(0)
		for i := 0; i < config.QueryCount; i++ {
			query := queries[i%len(queries)]
			queryDoc := map[string]string{
				"title":   query,
				"content": query,
				"tags":    query,
			}
			start := time.Now()
			_ = vectorizer.Transform(queryDoc)
			totalQueryTime += time.Since(start)
		}
		result.TransformTime = totalQueryTime
		result.AvgQueryTime = totalQueryTime / time.Duration(config.QueryCount)
		result.QPS = float64(config.QueryCount) / totalQueryTime.Seconds()

		fmt.Printf("  ✓ %d 次查询: %v (平均: %v, QPS: %.2f)\n",
			config.QueryCount, result.TransformTime, result.AvgQueryTime, result.QPS)

		// 批量查询测试 (BM25F不支持批量，跳过)
		result.BatchTime = 0
		fmt.Printf("  ✓ 批量查询: 不支持\n")

	case "bm25l":
		cfg := embedding.DefaultBM25Config()
		cfg.MaxVocabSize = 50000
		cfg.Variant = "bm25l"
		vectorizer := embedding.NewBM25LVectorizer(cfg)

		// 训练
		runtime.GC()
		start := time.Now()
		vectorizer.Fit(docs)
		result.FitTime = time.Since(start)
		result.VocabSize = vectorizer.GetVocabularySize()

		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		result.MemoryMB = float64(m.HeapAlloc) / 1024 / 1024

		fmt.Printf("  ✓ 训练完成: %v (词汇量: %d, 内存: %.2f MB)\n",
			result.FitTime, result.VocabSize, result.MemoryMB)

		// 单次查询测试
		runtime.GC()
		totalQueryTime := time.Duration(0)
		for i := 0; i < config.QueryCount; i++ {
			query := queries[i%len(queries)]
			start := time.Now()
			_ = vectorizer.Transform(query)
			totalQueryTime += time.Since(start)
		}
		result.TransformTime = totalQueryTime
		result.AvgQueryTime = totalQueryTime / time.Duration(config.QueryCount)
		result.QPS = float64(config.QueryCount) / totalQueryTime.Seconds()

		fmt.Printf("  ✓ %d 次查询: %v (平均: %v, QPS: %.2f)\n",
			config.QueryCount, result.TransformTime, result.AvgQueryTime, result.QPS)

		// 批量查询测试 (BM25L不支持批量，跳过)
		result.BatchTime = 0
		fmt.Printf("  ✓ 批量查询: 不支持\n")

	case "bm25plus":
		cfg := embedding.DefaultBM25Config()
		cfg.MaxVocabSize = 50000
		cfg.Variant = "bm25+"
		cfg.Delta = 1.0
		vectorizer := embedding.NewBM25PlusVectorizer(cfg)

		// 训练
		runtime.GC()
		start := time.Now()
		vectorizer.Fit(docs)
		result.FitTime = time.Since(start)
		result.VocabSize = vectorizer.GetVocabularySize()

		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		result.MemoryMB = float64(m.HeapAlloc) / 1024 / 1024

		fmt.Printf("  ✓ 训练完成: %v (词汇量: %d, 内存: %.2f MB)\n",
			result.FitTime, result.VocabSize, result.MemoryMB)

		// 单次查询测试
		runtime.GC()
		totalQueryTime := time.Duration(0)
		for i := 0; i < config.QueryCount; i++ {
			query := queries[i%len(queries)]
			start := time.Now()
			_ = vectorizer.Transform(query)
			totalQueryTime += time.Since(start)
		}
		result.TransformTime = totalQueryTime
		result.AvgQueryTime = totalQueryTime / time.Duration(config.QueryCount)
		result.QPS = float64(config.QueryCount) / totalQueryTime.Seconds()

		fmt.Printf("  ✓ %d 次查询: %v (平均: %v, QPS: %.2f)\n",
			config.QueryCount, result.TransformTime, result.AvgQueryTime, result.QPS)

		// 批量查询测试 (BM25Plus不支持批量，跳过)
		result.BatchTime = 0
		fmt.Printf("  ✓ 批量查询: 不支持\n")
	}

	return result
}

func printSummary(variant string, results []TestResult) {
	fmt.Printf("\n【%s 性能汇总】\n", variant)
	fmt.Println("┌────────────┬──────────┬──────────┬──────────┬──────────┬──────────┬──────────┐")
	fmt.Println("│   规模     │  训练时间 │ 查询时间 │ 平均查询 │   QPS    │  内存MB  │ 词汇量   │")
	fmt.Println("├────────────┼──────────┼──────────┼──────────┼──────────┼──────────┼──────────┤")

	for _, r := range results {
		fmt.Printf("│ %-10s │ %8v │ %8v │ %8v │ %8.2f │ %8.2f │ %8d │\n",
			r.Name,
			r.FitTime.Round(time.Millisecond),
			r.TransformTime.Round(time.Millisecond),
			r.AvgQueryTime.Round(time.Microsecond),
			r.QPS,
			r.MemoryMB,
			r.VocabSize)
	}

	fmt.Println("└────────────┴──────────┴──────────┴──────────┴──────────┴──────────┴──────────┘")
}

func compareAllVariants() {
	// 使用中等规模数据进行对比
	config := testConfigs[1] // 中规模
	fmt.Printf("\n【对比基准: %s - %d 文档】\n", config.Name, config.DocCount)

	// 收集所有变体的结果
	allResults := make(map[string]TestResult)
	variants := []string{"bm25", "bm25f", "bm25l", "bm25plus"}

	for _, variant := range variants {
		result := runTest(variant, config)
		allResults[variant] = result
	}

	// 打印对比表格
	fmt.Println("\n┌──────────┬──────────┬──────────┬──────────┬──────────┬──────────┐")
	fmt.Println("│  变体    │  训练时间 │ 查询时间 │   QPS    │  内存MB  │ 词汇量   │")
	fmt.Println("├──────────┼──────────┼──────────┼──────────┼──────────┼──────────┤")

	for _, variant := range variants {
		r := allResults[variant]
		fmt.Printf("│ %-8s │ %8v │ %8v │ %8.2f │ %8.2f │ %8d │\n",
			variant,
			r.FitTime.Round(time.Millisecond),
			r.TransformTime.Round(time.Millisecond),
			r.QPS,
			r.MemoryMB,
			r.VocabSize)
	}

	fmt.Println("└──────────┴──────────┴──────────┴──────────┴──────────┴──────────┘")

	// 找出最佳性能
	fmt.Println("\n【性能排名】")

	// 按训练时间排序
	byFitTime := make([]string, 0, len(variants))
	for _, v := range variants {
		byFitTime = append(byFitTime, v)
	}
	sort.Slice(byFitTime, func(i, j int) bool {
		return allResults[byFitTime[i]].FitTime < allResults[byFitTime[j]].FitTime
	})

	fmt.Println("  训练速度:")
	for i, v := range byFitTime {
		fmt.Printf("    %d. %s: %v\n", i+1, v, allResults[v].FitTime.Round(time.Millisecond))
	}

	// 按QPS排序
	byQPS := make([]string, 0, len(variants))
	for _, v := range variants {
		byQPS = append(byQPS, v)
	}
	sort.Slice(byQPS, func(i, j int) bool {
		return allResults[byQPS[i]].QPS > allResults[byQPS[j]].QPS
	})

	fmt.Println("  查询速度 (QPS):")
	for i, v := range byQPS {
		fmt.Printf("    %d. %s: %.2f\n", i+1, v, allResults[v].QPS)
	}

	// 按内存排序
	byMemory := make([]string, 0, len(variants))
	for _, v := range variants {
		byMemory = append(byMemory, v)
	}
	sort.Slice(byMemory, func(i, j int) bool {
		return allResults[byMemory[i]].MemoryMB < allResults[byMemory[j]].MemoryMB
	})

	fmt.Println("  内存占用:")
	for i, v := range byMemory {
		fmt.Printf("    %d. %s: %.2f MB\n", i+1, v, allResults[v].MemoryMB)
	}
}

func generateDocs(count, length int) []string {
	docs := make([]string, count)
	for i := 0; i < count; i++ {
		doc := ""
		docLen := length + rand.Intn(length/2)
		for j := 0; j < docLen; j++ {
			if j > 0 {
				doc += " "
			}
			doc += words[rand.Intn(len(words))]
		}
		docs[i] = doc
	}
	return docs
}

func convertToMultiField(docs []string) []map[string]string {
	multiDocs := make([]map[string]string, len(docs))
	titles := []string{"Go 编程", "Python 数据分析", "Java 开发", "机器学习", "人工智能",
		"系统优化", "网络安全", "云计算", "大数据", "深度学习"}
	tags := []string{"go,编程", "python,数据", "java,企业", "ml,ai", "system,perf",
		"security", "cloud", "bigdata", "dl", "neural"}

	for i, doc := range docs {
		multiDocs[i] = map[string]string{
			"title":   titles[rand.Intn(len(titles))],
			"content": doc,
			"tags":    tags[rand.Intn(len(tags))],
		}
	}
	return multiDocs
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
