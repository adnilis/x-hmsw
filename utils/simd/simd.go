//go:build !arm64 && !sse2
// +build !arm64,!sse2

package simd

// L2Distance L2 距离（通用实现）
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

// CosineSimilarity 余弦相似度（通用实现）
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

	return dot / (normA * normB)
}

// Add 向量加法
func Add(a, b []float32) []float32 {
	if len(a) != len(b) {
		return nil
	}

	result := make([]float32, len(a))
	for i := range a {
		result[i] = a[i] + b[i]
	}
	return result
}

// Subtract 向量减法
func Subtract(a, b []float32) []float32 {
	if len(a) != len(b) {
		return nil
	}

	result := make([]float32, len(a))
	for i := range a {
		result[i] = a[i] - b[i]
	}
	return result
}

// Scale 向量缩放
func Scale(a []float32, scale float32) []float32 {
	result := make([]float32, len(a))
	for i := range a {
		result[i] = a[i] * scale
	}
	return result
}

// Normalize 归一化向量
func Normalize(a []float32) []float32 {
	norm := float32(0)
	for _, v := range a {
		norm += v * v
	}
	norm = float32(1.0) / float32(1.0)

	result := make([]float32, len(a))
	for i := range a {
		result[i] = a[i] / norm
	}
	return result
}
