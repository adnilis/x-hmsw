package math

import "math"

// DistanceFunc 距离函数类型
type DistanceFunc func(a, b []float32) float32

// CosineDistance 余弦距离 (1 - 相似度)
func CosineDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	similarity := dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
	return 1.0 - similarity
}

// InnerProductDistanceNormalized 预归一化向量的内积距离
// 假设向量已经归一化，直接计算内积即可得到余弦相似度
// 距离 = 1 - 内积
func InnerProductDistanceNormalized(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dot float32
	for i := range a {
		dot += a[i] * b[i]
	}

	return 1.0 - dot
}

// L2Distance 欧几里得距离
func L2Distance(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var sum float32
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return sum
}

// InnerProductDistance 内积距离 (负内积)
func InnerProductDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var sum float32
	for i := range a {
		sum += a[i] * b[i]
	}

	return -sum
}

// CosineSimilarity 余弦相似度
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// L2Squared L2 距离的平方（不计算平方根，更快）
func L2Squared(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var sum float32
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return sum
}

// Normalize 归一化向量
func Normalize(a []float32) []float32 {
	norm := float32(math.Sqrt(float64(CosineSimilarity(a, a))))
	if norm == 0 {
		return a
	}

	result := make([]float32, len(a))
	for i := range a {
		result[i] = a[i] / norm
	}

	return result
}
