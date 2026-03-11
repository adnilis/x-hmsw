package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"runtime/debug"
	"time"

	"github.com/adnilis/logger"
	"github.com/adnilis/x-hmsw/api"
	iface "github.com/adnilis/x-hmsw/interface"
	"github.com/adnilis/x-hmsw/types"
)

// ChatMessage 聊天消息结构
type ChatMessage struct {
	ID        string
	Role      string // "user" 或 "assistant"
	Content   string
	Timestamp time.Time
}

// 模拟聊天记录
var chatHistory = []ChatMessage{
	{
		ID:      "msg_001",
		Role:    "user",
		Content: "你好，我想了解机器学习的基本概念",
	},
	{
		ID:      "msg_002",
		Role:    "assistant",
		Content: "机器学习是人工智能的一个分支，它使计算机能够从数据中学习并改进性能，而无需明确编程。主要包括监督学习、无监督学习和强化学习三大类。",
	},
	{
		ID:      "msg_003",
		Role:    "user",
		Content: "什么是监督学习？",
	},
	{
		ID:      "msg_004",
		Role:    "assistant",
		Content: "监督学习是机器学习的一种方法，它使用标记的训练数据来学习从输入到输出的映射。常见的监督学习算法包括线性回归、逻辑回归、决策树、支持向量机和神经网络等。",
	},
	{
		ID:      "msg_005",
		Role:    "user",
		Content: "深度学习和机器学习有什么区别？",
	},
	{
		ID:      "msg_006",
		Role:    "assistant",
		Content: "深度学习是机器学习的一个子集，它使用多层神经网络来学习数据的复杂模式。与传统机器学习相比，深度学习可以自动提取特征，适合处理图像、语音和文本等非结构化数据。",
	},
	{
		ID:      "msg_007",
		Role:    "user",
		Content: "如何选择合适的机器学习算法？",
	},
	{
		ID:      "msg_008",
		Role:    "assistant",
		Content: "选择机器学习算法需要考虑多个因素：数据类型和规模、问题类型（分类、回归、聚类）、可解释性要求、训练时间和计算资源。建议从小规模实验开始，逐步优化。",
	},
	{
		ID:      "msg_009",
		Role:    "user",
		Content: "什么是过拟合？如何避免？",
	},
	{
		ID:      "msg_010",
		Role:    "assistant",
		Content: "过拟合是指模型在训练数据上表现很好，但在新数据上表现很差。避免过拟合的方法包括：增加训练数据、使用正则化、简化模型结构、使用交叉验证、早停法和数据增强等技术。",
	},
	{
		ID:      "msg_011",
		Role:    "user",
		Content: "什么是神经网络？",
	},
	{
		ID:      "msg_012",
		Role:    "assistant",
		Content: "神经网络是一种受生物神经系统启发的计算模型，由多个相互连接的节点（神经元）组成。每个神经元接收输入，通过激活函数处理后输出。神经网络通过调整连接权重来学习数据中的模式。",
	},
	{
		ID:      "msg_013",
		Role:    "user",
		Content: "什么是卷积神经网络（CNN）？",
	},
	{
		ID:      "msg_014",
		Role:    "assistant",
		Content: "卷积神经网络是一种专门用于处理网格状数据（如图像）的深度学习模型。它使用卷积层自动提取局部特征，通过池化层降低维度，广泛应用于图像分类、目标检测和图像分割等任务。",
	},
	{
		ID:      "msg_015",
		Role:    "user",
		Content: "什么是循环神经网络（RNN）？",
	},
	{
		ID:      "msg_016",
		Role:    "assistant",
		Content: "循环神经网络是一种用于处理序列数据的神经网络，它通过隐藏状态在时间步之间传递信息。RNN 适合处理文本、语音和时间序列等数据，但存在梯度消失和梯度爆炸问题。",
	},
	{
		ID:      "msg_017",
		Role:    "user",
		Content: "什么是 Transformer？",
	},
	{
		ID:      "msg_018",
		Role:    "assistant",
		Content: "Transformer 是一种基于自注意力机制的深度学习模型，它能够并行处理序列数据，解决了 RNN 的长距离依赖问题。Transformer 是现代大语言模型（如 GPT、BERT）的基础架构。",
	},
	{
		ID:      "msg_019",
		Role:    "user",
		Content: "如何评估机器学习模型的性能？",
	},
	{
		ID:      "msg_020",
		Role:    "assistant",
		Content: "评估机器学习模型性能需要使用适当的指标：分类任务使用准确率、精确率、召回率、F1 分数和 ROC-AUC；回归任务使用均方误差、均方根误差和 R² 分数。还需要使用交叉验证来确保模型的泛化能力。",
	},
}

func main() {
	// 设置 panic 恢复
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Panic recovered: %v\n", r)
			fmt.Fprintf(os.Stderr, "Stack trace: %s\n", string(debug.Stack()))
			os.Exit(1)
		}
	}()

	logger.Debug("========================================")
	logger.Debug("   向量数据库 - 聊天记录管理演示")
	logger.Debug("========================================\n")

	// 1. 数据库初始化
	logger.Debug("【步骤 1】初始化向量数据库...")
	db, err := initializeDatabase()
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer db.Close()
	logger.Debug("✓ 数据库初始化成功\n")

	// 2. 保存聊天记录
	logger.Debug("【步骤 2】保存聊天记录到向量数据库...")
	err = saveChatHistory(db)
	if err != nil {
		log.Fatalf("保存聊天记录失败: %v", err)
	}
	logger.Debug("✓ 聊天记录保存成功\n")

	// 3. 搜索测试
	logger.Debug("【步骤 3】执行搜索测试...")
	testSearch(db)
	logger.Debug("✓ 搜索测试完成\n")

	// 4. 高级过滤测试
	testAdvancedFilters(db)

	logger.Debug("========================================")
	logger.Debug("   演示完成！")
	logger.Debug("========================================")
}

// initializeDatabase 初始化向量数据库
func initializeDatabase() (*api.QuickDB, error) {
	// 使用构建器模式创建数据库
	db, err := api.NewBuilder().
		WithDimension(384).
		WithStoragePath("./data/chat_vectors").
		WithHNSWParams(16, 200, 1000000).
		WithAutoSave(true, 30*time.Second).
		Build()

	if err != nil {
		return nil, err
	}

	logger.Debug("  配置参数:\n")
	logger.Debug("    - 维度: 384\n")
	logger.Debug("    - 索引类型: HNSW\n")
	logger.Debug("    - 距离度量: Cosine\n")
	logger.Debug("    - 存储类型: Badger\n")
	logger.Debug("    - 存储路径: ./data/chat_vectors\n")
	logger.Debug("    - 最大向量数: 1000000\n")
	logger.Debug("    - M 参数: 16\n")
	logger.Debug("    - EfConstruction: 200\n")
	logger.Debug("    - 自动保存: 30秒\n")

	return db, nil
}

// saveChatHistory 保存聊天记录到向量数据库
func saveChatHistory(db *api.QuickDB) error {
	logger.Debug("  准备保存 %d 条聊天记录...\n", len(chatHistory))

	// 将聊天记录转换为向量（使用默认TF-IDF，不生成向量）
	vectors := make([]types.Vector, 0, len(chatHistory))
	for i, msg := range chatHistory {
		// 使用数字 ID（HNSW 索引需要）
		vec := types.Vector{
			ID: fmt.Sprintf("%d", i), // 使用索引作为 ID
			// 不设置Vector，让TF-IDF在第一次搜索时自动生成
			Payload: map[string]interface{}{
				"msg_id":    msg.ID, // 保存原始消息 ID
				"role":      msg.Role,
				"content":   msg.Content, // TF-IDF会使用这个字段
				"timestamp": time.Now(),
				"index":     i,
			},
		}
		vectors = append(vectors, vec)
	}

	// 插入向量
	start := time.Now()
	if err := db.Insert(vectors); err != nil {
		return fmt.Errorf("插入向量失败: %w", err)
	}
	elapsed := time.Since(start)

	logger.Debug("  ✓ 成功保存 %d 条聊天记录\n", len(vectors))
	logger.Debug("  ✓ 耗时: %v\n", elapsed)
	logger.Debug("  ✓ 平均速度: %.2f 条/秒\n", float64(len(vectors))/elapsed.Seconds())

	// 显示统计信息
	count, err := db.Count()
	if err == nil {
		logger.Debug("  ✓ 数据库当前向量总数: %d\n", count)
	}

	return nil
}

// testSearch 执行搜索测试
func testSearch(db *api.QuickDB) {
	// 定义测试查询
	testQueries := []struct {
		name   string
		query  string
		topK   int
		filter map[string]interface{} // Payload过滤条件
	}{
		{
			name:  "机器学习基础（无过滤）",
			query: "什么是机器学习？",
			topK:  13,
		},
		{
			name:  "机器学习基础（仅用户消息）",
			query: "什么是机器学习？",
			topK:  13,
			filter: map[string]interface{}{
				"role": "user",
			},
		},
		{
			name:  "机器学习基础（仅助手回复）",
			query: "什么是机器学习？",
			topK:  13,
			filter: map[string]interface{}{
				"role": "assistant",
			},
		},
		{
			name:  "深度学习（无过滤）",
			query: "深度学习和神经网络的关系",
			topK:  13,
		},
		{
			name:  "深度学习（仅用户消息）",
			query: "深度学习和神经网络的关系",
			topK:  13,
			filter: map[string]interface{}{
				"role": "user",
			},
		},
		{
			name:  "模型评估（无过滤）",
			query: "如何评估机器学习模型",
			topK:  13,
		},
		{
			name:  "模型评估（仅用户消息）",
			query: "如何评估机器学习模型",
			topK:  13,
			filter: map[string]interface{}{
				"role": "user",
			},
		},
		{
			name:  "过拟合问题（无过滤）",
			query: "什么是过拟合，如何解决",
			topK:  13,
		},
		{
			name:  "过拟合问题（仅助手回复）",
			query: "什么是过拟合，如何解决",
			topK:  13,
			filter: map[string]interface{}{
				"role": "assistant",
			},
		},
	}

	for _, test := range testQueries {
		logger.Debug("\n  【测试】%s\n", test.name)
		logger.Debug("  查询: \"%s\"\n", test.query)

		if test.filter != nil {
			logger.Debug("  过滤条件: %v\n", test.filter)
		}

		// 执行搜索（使用SearchByText，TF-IDF会自动生成向量）
		start := time.Now()
		var results []iface.SearchResult
		var err error

		if test.filter != nil {
			// 使用Payload过滤
			results, err = db.SearchByTextWithFilter(test.query, test.topK, test.filter)
		} else {
			// 普通搜索（使用SearchByText）
			results, err = db.SearchByText(test.query, test.topK)
		}

		if err != nil {
			logger.Debug("  ✗ 搜索失败: %v\n", err)
			continue
		}
		elapsed := time.Since(start)

		logger.Debug("  搜索耗时: %v\n", elapsed)
		logger.Debug("  找到 %d 个结果:\n", len(results))

		for i, result := range results {
			// 检查 Payload 是否存在
			if result.Payload == nil {
				logger.Debug("    %d. [%.4f] ID=%s (无负载信息)\n", i+1, result.Score, result.ID)
				continue
			}

			role := "unknown"
			if r, ok := result.Payload["role"].(string); ok {
				role = r
			}

			content := "无内容"
			if c, ok := result.Payload["content"].(string); ok {
				content = c
				// 截断过长的内容
				if len(content) > 80 {
					content = content[:80] + "..."
				}
			}

			msgID := result.ID
			if mid, ok := result.Payload["msg_id"].(string); ok {
				msgID = mid
			}

			logger.Debug("    %d. [%.4f] [%s] %s: %s\n", i+1, result.Score, msgID, role, content)
		}
	}
}

// testAdvancedFilters 测试高级过滤功能
func testAdvancedFilters(db *api.QuickDB) {
	logger.Debug("\n========================================")
	logger.Debug("   高级过滤功能测试")
	logger.Debug("========================================\n")

	// 获取当前时间
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)

	// 定义高级过滤测试
	advancedTests := []struct {
		name   string
		query  string
		topK   int
		filter map[string]interface{}
	}{
		{
			name:  "时间范围过滤（最近1小时）",
			query: "机器学习",
			topK:  10,
			filter: map[string]interface{}{
				"timestamp": map[string]interface{}{
					"$gte": oneHourAgo,
				},
			},
		},
		{
			name:  "索引范围过滤（index >= 5）",
			query: "神经网络",
			topK:  10,
			filter: map[string]interface{}{
				"index": map[string]interface{}{
					"$gte": 5,
				},
			},
		},
		{
			name:  "索引范围过滤（index < 10）",
			query: "学习",
			topK:  10,
			filter: map[string]interface{}{
				"index": map[string]interface{}{
					"$lt": 10,
				},
			},
		},
		{
			name:  "索引范围过滤（5 <= index < 15）",
			query: "模型",
			topK:  10,
			filter: map[string]interface{}{
				"index": map[string]interface{}{
					"$gte": 5,
					"$lt":  15,
				},
			},
		},
		{
			name:  "OR逻辑（用户或助手）",
			query: "学习",
			topK:  10,
			filter: map[string]interface{}{
				"$or": []interface{}{
					map[string]interface{}{"role": "user"},
					map[string]interface{}{"role": "assistant"},
				},
			},
		},
		{
			name:  "AND逻辑（用户且索引>=10）",
			query: "评估",
			topK:  10,
			filter: map[string]interface{}{
				"$and": []interface{}{
					map[string]interface{}{"role": "user"},
					map[string]interface{}{
						"index": map[string]interface{}{"$gte": 10},
					},
				},
			},
		},
		{
			name:  "IN操作（特定索引）",
			query: "网络",
			topK:  10,
			filter: map[string]interface{}{
				"index": map[string]interface{}{
					"$in": []interface{}{1, 5, 10, 15},
				},
			},
		},
		{
			name:  "NOT IN操作（排除特定索引）",
			query: "学习",
			topK:  10,
			filter: map[string]interface{}{
				"index": map[string]interface{}{
					"$nin": []interface{}{0, 1, 2},
				},
			},
		},
		{
			name:  "不等于操作（role != user）",
			query: "模型",
			topK:  10,
			filter: map[string]interface{}{
				"role": map[string]interface{}{
					"$ne": "user",
				},
			},
		},
	}

	for _, test := range advancedTests {
		logger.Debug("\n  【测试】%s\n", test.name)
		logger.Debug("  查询: \"%s\"\n", test.query)
		logger.Debug("  过滤条件: %v\n", test.filter)

		// 执行搜索（使用SearchByTextWithFilter，TF-IDF会自动生成向量）
		start := time.Now()
		results, err := db.SearchByTextWithFilter(test.query, test.topK, test.filter)
		if err != nil {
			logger.Debug("  ✗ 搜索失败: %v\n", err)
			continue
		}
		elapsed := time.Since(start)

		logger.Debug("  搜索耗时: %v\n", elapsed)
		logger.Debug("  找到 %d 个结果:\n", len(results))

		for i, result := range results {
			if result.Payload == nil {
				logger.Debug("    %d. [%.4f] ID=%s (无负载信息)\n", i+1, result.Score, result.ID)
				continue
			}

			role := "unknown"
			if r, ok := result.Payload["role"].(string); ok {
				role = r
			}

			content := "无内容"
			if c, ok := result.Payload["content"].(string); ok {
				content = c
				if len(content) > 60 {
					content = content[:60] + "..."
				}
			}

			msgID := result.ID
			if mid, ok := result.Payload["msg_id"].(string); ok {
				msgID = mid
			}

			index := 0
			if idx, ok := result.Payload["index"].(int); ok {
				index = idx
			}

			logger.Debug("    %d. [%.4f] [%s] index=%d %s: %s\n", i+1, result.Score, msgID, index, role, content)
		}
	}
}

// generateMockVector 生成模拟向量
// 实际应用中应该使用真实的嵌入模型（如 sentence-transformers）
func generateMockVector(dim int, text string) []float32 {
	vector := make([]float32, dim)

	// 使用文本内容生成确定性的伪随机向量
	// 这样相同的文本会生成相同的向量，便于测试
	seed := hashString(text)
	r := rand.New(rand.NewPCG(0, uint64(seed)))

	for i := 0; i < dim; i++ {
		vector[i] = r.Float32()*2 - 1 // [-1, 1]
	}

	// 归一化向量
	norm := float32(0)
	for _, v := range vector {
		norm += v * v
	}
	norm = float32(1.0) / (float32(1.0) + float32(0.0000001))
	if norm > 0 {
		for i := range vector {
			vector[i] /= norm
		}
	}

	return vector
}

// hashString 将字符串转换为哈希值
func hashString(s string) uint64 {
	h := uint64(14695981039346656037)
	for _, c := range s {
		h ^= uint64(c)
		h *= 1099511628211
	}
	return h
}
