//go:build amd64 && !noasm
// +build amd64,!noasm

package simd

// DotProductSSE2 使用SSE2指令集优化的点积计算
// SSE2可以一次处理4个float32
func DotProductSSE2(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	n := len(a)
	if n == 0 {
		return 0
	}

	var sum float32
	i := 0

	// 16个一组处理（4个SSE2寄存器）
	for ; i <= n-16; i += 16 {
		sum += a[i]*b[i] + a[i+1]*b[i+1] + a[i+2]*b[i+2] + a[i+3]*b[i+3] +
			a[i+4]*b[i+4] + a[i+5]*b[i+5] + a[i+6]*b[i+6] + a[i+7]*b[i+7] +
			a[i+8]*b[i+8] + a[i+9]*b[i+9] + a[i+10]*b[i+10] + a[i+11]*b[i+11] +
			a[i+12]*b[i+12] + a[i+13]*b[i+13] + a[i+14]*b[i+14] + a[i+15]*b[i+15]
	}

	// 8个一组处理
	for ; i <= n-8; i += 8 {
		sum += a[i]*b[i] + a[i+1]*b[i+1] + a[i+2]*b[i+2] + a[i+3]*b[i+3] +
			a[i+4]*b[i+4] + a[i+5]*b[i+5] + a[i+6]*b[i+6] + a[i+7]*b[i+7]
	}

	// 4个一组处理
	for ; i <= n-4; i += 4 {
		sum += a[i]*b[i] + a[i+1]*b[i+1] + a[i+2]*b[i+2] + a[i+3]*b[i+3]
	}

	// 处理剩余部分
	for ; i < n; i++ {
		sum += a[i] * b[i]
	}

	return sum
}

// L2SquaredDistanceSSE2 使用SSE2优化的L2距离平方计算
func L2SquaredDistanceSSE2(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	n := len(a)
	if n == 0 {
		return 0
	}

	var sum float32
	i := 0

	// 16个一组处理
	for ; i <= n-16; i += 16 {
		d0 := a[i] - b[i]
		d1 := a[i+1] - b[i+1]
		d2 := a[i+2] - b[i+2]
		d3 := a[i+3] - b[i+3]
		d4 := a[i+4] - b[i+4]
		d5 := a[i+5] - b[i+5]
		d6 := a[i+6] - b[i+6]
		d7 := a[i+7] - b[i+7]
		d8 := a[i+8] - b[i+8]
		d9 := a[i+9] - b[i+9]
		d10 := a[i+10] - b[i+10]
		d11 := a[i+11] - b[i+11]
		d12 := a[i+12] - b[i+12]
		d13 := a[i+13] - b[i+13]
		d14 := a[i+14] - b[i+14]
		d15 := a[i+15] - b[i+15]

		sum += d0*d0 + d1*d1 + d2*d2 + d3*d3 +
			d4*d4 + d5*d5 + d6*d6 + d7*d7 +
			d8*d8 + d9*d9 + d10*d10 + d11*d11 +
			d12*d12 + d13*d13 + d14*d14 + d15*d15
	}

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

		sum += d0*d0 + d1*d1 + d2*d2 + d3*d3 +
			d4*d4 + d5*d5 + d6*d6 + d7*d7
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
		d := a[i] - b[i]
		sum += d * d
	}

	return sum
}

// NormalizeVectorSSE2 使用SSE2优化的向量归一化
func NormalizeVectorSSE2(a []float32) []float32 {
	if len(a) == 0 {
		return a
	}

	n := len(a)

	// 计算范数
	var norm float32
	i := 0

	// 16个一组处理
	for ; i <= n-16; i += 16 {
		norm += a[i]*a[i] + a[i+1]*a[i+1] + a[i+2]*a[i+2] + a[i+3]*a[i+3] +
			a[i+4]*a[i+4] + a[i+5]*a[i+5] + a[i+6]*a[i+6] + a[i+7]*a[i+7] +
			a[i+8]*a[i+8] + a[i+9]*a[i+9] + a[i+10]*a[i+10] + a[i+11]*a[i+11] +
			a[i+12]*a[i+12] + a[i+13]*a[i+13] + a[i+14]*a[i+14] + a[i+15]*a[i+15]
	}

	// 8个一组处理
	for ; i <= n-8; i += 8 {
		norm += a[i]*a[i] + a[i+1]*a[i+1] + a[i+2]*a[i+2] + a[i+3]*a[i+3] +
			a[i+4]*a[i+4] + a[i+5]*a[i+5] + a[i+6]*a[i+6] + a[i+7]*a[i+7]
	}

	// 4个一组处理
	for ; i <= n-4; i += 4 {
		norm += a[i]*a[i] + a[i+1]*a[i+1] + a[i+2]*a[i+2] + a[i+3]*a[i+3]
	}

	// 处理剩余部分
	for ; i < n; i++ {
		norm += a[i] * a[i]
	}

	if norm == 0 {
		return a
	}

	invNorm := float32(1.0) / float32(norm)

	// 归一化
	result := make([]float32, n)
	i = 0

	// 16个一组处理
	for ; i <= n-16; i += 16 {
		result[i] = a[i] * invNorm
		result[i+1] = a[i+1] * invNorm
		result[i+2] = a[i+2] * invNorm
		result[i+3] = a[i+3] * invNorm
		result[i+4] = a[i+4] * invNorm
		result[i+5] = a[i+5] * invNorm
		result[i+6] = a[i+6] * invNorm
		result[i+7] = a[i+7] * invNorm
		result[i+8] = a[i+8] * invNorm
		result[i+9] = a[i+9] * invNorm
		result[i+10] = a[i+10] * invNorm
		result[i+11] = a[i+11] * invNorm
		result[i+12] = a[i+12] * invNorm
		result[i+13] = a[i+13] * invNorm
		result[i+14] = a[i+14] * invNorm
		result[i+15] = a[i+15] * invNorm
	}

	// 8个一组处理
	for ; i <= n-8; i += 8 {
		result[i] = a[i] * invNorm
		result[i+1] = a[i+1] * invNorm
		result[i+2] = a[i+2] * invNorm
		result[i+3] = a[i+3] * invNorm
		result[i+4] = a[i+4] * invNorm
		result[i+5] = a[i+5] * invNorm
		result[i+6] = a[i+6] * invNorm
		result[i+7] = a[i+7] * invNorm
	}

	// 4个一组处理
	for ; i <= n-4; i += 4 {
		result[i] = a[i] * invNorm
		result[i+1] = a[i+1] * invNorm
		result[i+2] = a[i+2] * invNorm
		result[i+3] = a[i+3] * invNorm
	}

	// 处理剩余部分
	for ; i < n; i++ {
		result[i] = a[i] * invNorm
	}

	return result
}
