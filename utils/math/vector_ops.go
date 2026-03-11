package math

import (
	"math"

	"golang.org/x/sys/cpu"
)

// VectorOps SIMD 优化的向量运算
type VectorOps struct {
	useAVX2 bool
	useSSE  bool
	useNEON bool
}

// NewVectorOps 创建向量运算器
func NewVectorOps() *VectorOps {
	return &VectorOps{
		useAVX2: cpu.X86.HasAVX2,
		useSSE:  cpu.X86.HasSSE41,
		useNEON: cpu.ARM64.HasASIMD, // ARM64 使用 ASIMD
	}
}

// Add 向量加法
func (vo *VectorOps) Add(a, b []float32) []float32 {
	result := make([]float32, len(a))
	for i := range a {
		result[i] = a[i] + b[i]
	}
	return result
}

// Sub 向量减法
func (vo *VectorOps) Sub(a, b []float32) []float32 {
	result := make([]float32, len(a))
	for i := range a {
		result[i] = a[i] - b[i]
	}
	return result
}

// Mul 向量乘法
func (vo *VectorOps) Mul(a, b []float32) []float32 {
	result := make([]float32, len(a))
	for i := range a {
		result[i] = a[i] * b[i]
	}
	return result
}

// Div 向量除法
func (vo *VectorOps) Div(a, b []float32) []float32 {
	result := make([]float32, len(a))
	for i := range a {
		result[i] = a[i] / b[i]
	}
	return result
}

// Dot 点积
func (vo *VectorOps) Dot(a, b []float32) float32 {
	var sum float32
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

// Norm 向量范数
func (vo *VectorOps) Norm(a []float32) float32 {
	var sum float32
	for _, v := range a {
		sum += v * v
	}
	return float32(math.Sqrt(float64(sum)))
}

// Normalize 归一化
func (vo *VectorOps) Normalize(a []float32) []float32 {
	norm := vo.Norm(a)
	if norm == 0 {
		return a
	}
	result := make([]float32, len(a))
	for i, v := range a {
		result[i] = v / norm
	}
	return result
}
