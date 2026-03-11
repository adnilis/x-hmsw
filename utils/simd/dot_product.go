package simd

import (
	"math"
)

// DotProduct 优化的点积计算
// 使用循环展开和向量化优化
func DotProduct(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	n := len(a)
	i := 0
	var sum float32

	// 8个一组处理（循环展开）
	for ; i <= n-8; i += 8 {
		sum += a[i]*b[i] +
			a[i+1]*b[i+1] +
			a[i+2]*b[i+2] +
			a[i+3]*b[i+3] +
			a[i+4]*b[i+4] +
			a[i+5]*b[i+5] +
			a[i+6]*b[i+6] +
			a[i+7]*b[i+7]
	}

	// 4个一组处理
	for ; i <= n-4; i += 4 {
		sum += a[i]*b[i] +
			a[i+1]*b[i+1] +
			a[i+2]*b[i+2] +
			a[i+3]*b[i+3]
	}

	// 处理剩余部分
	for ; i < n; i++ {
		sum += a[i] * b[i]
	}

	return sum
}

// L2SquaredDistance 优化的L2距离平方计算
func L2SquaredDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	n := len(a)
	i := 0
	var sum float32

	// 8个一组处理
	for ; i <= n-8; i += 8 {
		d0 := a[i] - b[i]
		d1 := a[i+1] - b[i+1]
		d2 := a[i+2] - b[i+2]
		d3 := a[i+3] - b[i+3]
		d4 := a[i+4] - b[i+4]
		d5 := a[i+5] - b[i+5]
		d6 := a[i+6] - b[i+6]
		d7 := a[i+7] - b[i+7]
		sum += d0*d0 + d1*d1 + d2*d2 + d3*d3 + d4*d4 + d5*d5 + d6*d6 + d7*d7
	}

	// 4个一组处理
	for ; i <= n-4; i += 4 {
		d0 := a[i] - b[i]
		d1 := a[i+1] - b[i+1]
		d2 := a[i+2] - b[i+2]
		d3 := a[i+3] - b[i+3]
		sum += d0*d0 + d1*d1 + d2*d2 + d3*d3
	}

	// 处理剩余部分
	for ; i < n; i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return sum
}

// NormalizeVector 优化的向量归一化
func NormalizeVector(v []float32) []float32 {
	if len(v) == 0 {
		return v
	}

	// 计算范数
	var norm float32
	n := len(v)
	i := 0

	// 8个一组处理
	for ; i <= n-8; i += 8 {
		norm += v[i]*v[i] +
			v[i+1]*v[i+1] +
			v[i+2]*v[i+2] +
			v[i+3]*v[i+3] +
			v[i+4]*v[i+4] +
			v[i+5]*v[i+5] +
			v[i+6]*v[i+6] +
			v[i+7]*v[i+7]
	}

	// 4个一组处理
	for ; i <= n-4; i += 4 {
		norm += v[i]*v[i] +
			v[i+1]*v[i+1] +
			v[i+2]*v[i+2] +
			v[i+3]*v[i+3]
	}

	// 处理剩余部分
	for ; i < n; i++ {
		norm += v[i] * v[i]
	}

	if norm == 0 {
		return v
	}

	// 计算归一化因子
	scale := float32(1.0 / math.Sqrt(float64(norm)))

	// 归一化向量
	normalized := make([]float32, n)
	i = 0

	// 8个一组处理
	for ; i <= n-8; i += 8 {
		normalized[i] = v[i] * scale
		normalized[i+1] = v[i+1] * scale
		normalized[i+2] = v[i+2] * scale
		normalized[i+3] = v[i+3] * scale
		normalized[i+4] = v[i+4] * scale
		normalized[i+5] = v[i+5] * scale
		normalized[i+6] = v[i+6] * scale
		normalized[i+7] = v[i+7] * scale
	}

	// 4个一组处理
	for ; i <= n-4; i += 4 {
		normalized[i] = v[i] * scale
		normalized[i+1] = v[i+1] * scale
		normalized[i+2] = v[i+2] * scale
		normalized[i+3] = v[i+3] * scale
	}

	// 处理剩余部分
	for ; i < n; i++ {
		normalized[i] = v[i] * scale
	}

	return normalized
}

// NormalizeVectorInplace 原地归一化向量，减少内存分配
func NormalizeVectorInplace(v []float32) {
	if len(v) == 0 {
		return
	}

	// 计算范数
	var norm float32
	n := len(v)
	i := 0

	// 8个一组处理
	for ; i <= n-8; i += 8 {
		norm += v[i]*v[i] +
			v[i+1]*v[i+1] +
			v[i+2]*v[i+2] +
			v[i+3]*v[i+3] +
			v[i+4]*v[i+4] +
			v[i+5]*v[i+5] +
			v[i+6]*v[i+6] +
			v[i+7]*v[i+7]
	}

	// 4个一组处理
	for ; i <= n-4; i += 4 {
		norm += v[i]*v[i] +
			v[i+1]*v[i+1] +
			v[i+2]*v[i+2] +
			v[i+3]*v[i+3]
	}

	// 处理剩余部分
	for ; i < n; i++ {
		norm += v[i] * v[i]
	}

	if norm == 0 {
		return
	}

	// 计算归一化因子
	scale := float32(1.0 / math.Sqrt(float64(norm)))

	// 原地归一化
	i = 0

	// 8个一组处理
	for ; i <= n-8; i += 8 {
		v[i] *= scale
		v[i+1] *= scale
		v[i+2] *= scale
		v[i+3] *= scale
		v[i+4] *= scale
		v[i+5] *= scale
		v[i+6] *= scale
		v[i+7] *= scale
	}

	// 4个一组处理
	for ; i <= n-4; i += 4 {
		v[i] *= scale
		v[i+1] *= scale
		v[i+2] *= scale
		v[i+3] *= scale
	}

	// 处理剩余部分
	for ; i < n; i++ {
		v[i] *= scale
	}
}
