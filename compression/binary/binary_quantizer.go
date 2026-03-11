package binary

import (
	"math"
)

// BinaryQuantizer 二进制量化器
type BinaryQuantizer struct {
	dimension int
	threshold float32
	trained   bool
}

// NewBinaryQuantizer 创建二进制量化器
func NewBinaryQuantizer(dimension int) *BinaryQuantizer {
	return &BinaryQuantizer{
		dimension: dimension,
		threshold: 0.0,
		trained:   false,
	}
}

// Train 训练量化器
func (bq *BinaryQuantizer) Train(vectors [][]float32) error {
	if len(vectors) == 0 {
		return nil
	}

	// 计算每个维度的中位数作为阈值
	for i := 0; i < bq.dimension; i++ {
		values := make([]float32, len(vectors))
		for j, vec := range vectors {
			values[j] = vec[i]
		}

		// 排序找中位数
		median := findMedian(values)
		if i == 0 {
			bq.threshold = median
		} else {
			bq.threshold = (bq.threshold + median) / 2
		}
	}

	bq.trained = true
	return nil
}

// findMedian 找中位数
func findMedian(values []float32) float32 {
	if len(values) == 0 {
		return 0
	}

	// 简单排序
	sorted := make([]float32, len(values))
	copy(sorted, values)

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted[len(sorted)/2]
}

// Encode 编码向量
func (bq *BinaryQuantizer) Encode(vector []float32) []byte {
	if !bq.trained || len(vector) != bq.dimension {
		return nil
	}

	// 每个字节存储 8 个维度
	bytesNeeded := (bq.dimension + 7) / 8
	encoded := make([]byte, bytesNeeded)

	for i, val := range vector {
		if val > bq.threshold {
			byteIdx := i / 8
			bitIdx := 7 - (i % 8)
			encoded[byteIdx] |= (1 << bitIdx)
		}
	}

	return encoded
}

// Decode 解码向量（近似）
func (bq *BinaryQuantizer) Decode(encoded []byte) []float32 {
	if !bq.trained {
		return nil
	}

	decoded := make([]float32, bq.dimension)
	for i := 0; i < bq.dimension; i++ {
		byteIdx := i / 8
		bitIdx := 7 - (i % 8)

		if byteIdx < len(encoded) && (encoded[byteIdx]&(1<<bitIdx)) != 0 {
			decoded[i] = 1.0
		} else {
			decoded[i] = -1.0
		}
	}

	return decoded
}

// HammingDistance 汉明距离
func HammingDistance(a, b []byte) int {
	if len(a) != len(b) {
		return -1
	}

	distance := 0
	for i := range a {
		xor := a[i] ^ b[i]
		// 计算置位比特数
		for xor != 0 {
			distance += int(xor & 1)
			xor >>= 1
		}
	}

	return distance
}

// Popcount 计算置位比特数
func Popcount(x uint64) int {
	count := 0
	for x != 0 {
		count += int(x & 1)
		x >>= 1
	}
	return count
}

// CosineSimilarityBinary 二进制余弦相似度
func CosineSimilarityBinary(a, b []byte) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dot, normA, normB int
	for i := range a {
		// 计算点积
		common := a[i] & b[i]
		for common != 0 {
			dot += int(common & 1)
			common >>= 1
		}

		// 计算范数
		for x := a[i]; x != 0; {
			normA += int(x & 1)
			x >>= 1
		}
		for x := b[i]; x != 0; {
			normB += int(x & 1)
			x >>= 1
		}
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return float32(dot) / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// Quantize 量化向量
func Quantize(vectors [][]float32) (*BinaryQuantizer, error) {
	if len(vectors) == 0 {
		return nil, nil
	}

	bq := NewBinaryQuantizer(len(vectors[0]))
	if err := bq.Train(vectors); err != nil {
		return nil, err
	}

	return bq, nil
}
