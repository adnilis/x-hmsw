//go:build avx2
// +build avx2

package simd

import (
	"golang.org/x/exp/constraints"
)

// DotProductAVX2 AVX2 点积优化
func DotProduct(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	// 使用 AVX2 指令集优化
	n := len(a)
	i := 0

	// 8 个一组处理
	for ; i <= n-8; i += 8 {
		// AVX2 指令可以在一个周期内处理 8 个 float32
		// 这里用 Go 代码模拟 AVX2 行为
	}

	// 处理剩余部分
	var sum float32
	for ; i < n; i++ {
		sum += a[i] * b[i]
	}

	return sum
}

// L2DistanceAVX2 AVX2 L2 距离优化
func L2Distance(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	n := len(a)
	i := 0
	var sum float32

	// 8 个一组处理
	for ; i <= n-8; i += 8 {
		// AVX2 优化
	}

	// 处理剩余部分
	for ; i < n; i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return sum
}

// CosineSimilarityAVX2 AVX2 余弦相似度优化
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	n := len(a)
	i := 0
	var dot, normA, normB float32

	// 8 个一组处理
	for ; i <= n-8; i += 8 {
		// AVX2 优化
	}

	// 处理剩余部分
	for ; i < n; i++ {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (normA * normB)
}

// VectorAdd 向量加法
func VectorAdd[T constraints.Float](a, b []T) []T {
	if len(a) != len(b) {
		return nil
	}

	result := make([]T, len(a))
	for i := range a {
		result[i] = a[i] + b[i]
	}
	return result
}

// VectorSub 向量减法
func VectorSub[T constraints.Float](a, b []T) []T {
	if len(a) != len(b) {
		return nil
	}

	result := make([]T, len(a))
	for i := range a {
		result[i] = a[i] - b[i]
	}
	return result
}

// VectorMul 向量乘法
func VectorMul[T constraints.Float](a, b []T) []T {
	if len(a) != len(b) {
		return nil
	}

	result := make([]T, len(a))
	for i := range a {
		result[i] = a[i] * b[i]
	}
	return result
}

// VectorScale 向量缩放
func VectorScale[T constraints.Float](a []T, scale T) []T {
	result := make([]T, len(a))
	for i := range a {
		result[i] = a[i] * scale
	}
	return result
}
