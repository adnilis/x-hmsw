package embedding

import (
	"math"
	"math/rand"
	"time"
)

// MockEmbeddingFunc 创建一个基于文本哈希的Mock Embedding函数
// 这个函数为相同的文本生成相同的向量，便于测试
func MockEmbeddingFunc(dim int) EmbeddingFunc {
	return func(text string) ([]float32, error) {
		return generateMockVector(dim, text), nil
	}
}

// generateMockVector 生成模拟向量
// 使用文本内容生成确定性的伪随机向量
// 这样相同的文本会生成相同的向量，便于测试
func generateMockVector(dim int, text string) []float32 {
	vector := make([]float32, dim)

	// 使用文本内容生成确定性的伪随机向量
	seed := hashString(text)
	r := rand.New(rand.NewSource(int64(seed)))

	for i := 0; i < dim; i++ {
		vector[i] = r.Float32()*2 - 1 // [-1, 1]
	}

	// 归一化向量
	norm := float32(0)
	for _, v := range vector {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))
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

// RandomEmbeddingFunc 创建一个生成随机向量的Embedding函数
// 每次调用都会生成不同的向量
func RandomEmbeddingFunc(dim int) EmbeddingFunc {
	// 使用当前时间作为种子
	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))

	return func(text string) ([]float32, error) {
		vector := make([]float32, dim)
		for i := 0; i < dim; i++ {
			vector[i] = r.Float32()*2 - 1 // [-1, 1]
		}

		// 归一化向量
		norm := float32(0)
		for _, v := range vector {
			norm += v * v
		}
		norm = float32(math.Sqrt(float64(norm)))
		if norm > 0 {
			for i := range vector {
				vector[i] /= norm
			}
		}

		return vector, nil
	}
}

// MockBatchEmbeddingFunc 创建一个批量Mock Embedding函数
func MockBatchEmbeddingFunc(dim int) BatchEmbeddingFunc {
	return func(texts []string) ([][]float32, error) {
		embeddings := make([][]float32, len(texts))
		for i, text := range texts {
			embeddings[i] = generateMockVector(dim, text)
		}
		return embeddings, nil
	}
}

// RandomBatchEmbeddingFunc 创建一个批量随机Embedding函数
func RandomBatchEmbeddingFunc(dim int) BatchEmbeddingFunc {
	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))

	return func(texts []string) ([][]float32, error) {
		embeddings := make([][]float32, len(texts))
		for i := range texts {
			vector := make([]float32, dim)
			for j := 0; j < dim; j++ {
				vector[j] = r.Float32()*2 - 1 // [-1, 1]
			}

			// 归一化向量
			norm := float32(0)
			for _, v := range vector {
				norm += v * v
			}
			norm = float32(math.Sqrt(float64(norm)))
			if norm > 0 {
				for j := range vector {
					vector[j] /= norm
				}
			}

			embeddings[i] = vector
		}
		return embeddings, nil
	}
}
