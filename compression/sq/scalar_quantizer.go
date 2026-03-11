package sq

import (
	"math"
)

// ScalarQuantizer 标量量化器
type ScalarQuantizer struct {
	dimension int
	bits      int
	minVals   []float32
	maxVals   []float32
	trained   bool
}

// NewScalarQuantizer 创建标量量化器
func NewScalarQuantizer(dimension, bits int) *ScalarQuantizer {
	return &ScalarQuantizer{
		dimension: dimension,
		bits:      bits,
		minVals:   make([]float32, dimension),
		maxVals:   make([]float32, dimension),
		trained:   false,
	}
}

// Train 训练量化器
func (sq *ScalarQuantizer) Train(vectors [][]float32) error {
	if len(vectors) == 0 {
		return nil
	}

	// 初始化
	for i := range sq.minVals {
		sq.minVals[i] = vectors[0][i]
		sq.maxVals[i] = vectors[0][i]
	}

	// 找到每个维度的最小最大值
	for _, vec := range vectors {
		for i := range vec {
			if vec[i] < sq.minVals[i] {
				sq.minVals[i] = vec[i]
			}
			if vec[i] > sq.maxVals[i] {
				sq.maxVals[i] = vec[i]
			}
		}
	}

	sq.trained = true
	return nil
}

// Encode 编码向量
func (sq *ScalarQuantizer) Encode(vector []float32) []byte {
	if !sq.trained || len(vector) != sq.dimension {
		return nil
	}

	switch sq.bits {
	case 8:
		return sq.encode8(vector)
	case 16:
		return sq.encode16(vector)
	case 32:
		return sq.encode32(vector)
	default:
		return nil
	}
}

// encode8 8 位编码
func (sq *ScalarQuantizer) encode8(vector []float32) []byte {
	encoded := make([]byte, len(vector))
	for i, val := range vector {
		rng := sq.maxVals[i] - sq.minVals[i]
		if rng == 0 {
			encoded[i] = 128
		} else {
			normalized := (val - sq.minVals[i]) / rng
			quantized := uint8(math.Min(255.0, float64(normalized*255)))
			encoded[i] = quantized
		}
	}
	return encoded
}

// encode16 16 位编码
func (sq *ScalarQuantizer) encode16(vector []float32) []byte {
	encoded := make([]byte, len(vector)*2)
	for i, val := range vector {
		rng := sq.maxVals[i] - sq.minVals[i]
		if rng == 0 {
			encoded[i*2] = 0
			encoded[i*2+1] = 128
		} else {
			normalized := (val - sq.minVals[i]) / rng
			quantized := uint16(math.Min(65535.0, float64(normalized*65535)))
			encoded[i*2] = byte(quantized)
			encoded[i*2+1] = byte(quantized >> 8)
		}
	}
	return encoded
}

// encode32 32 位编码
func (sq *ScalarQuantizer) encode32(vector []float32) []byte {
	encoded := make([]byte, len(vector)*4)
	for i, val := range vector {
		rng := sq.maxVals[i] - sq.minVals[i]
		if rng == 0 {
			encoded[i*4] = 0
			encoded[i*4+1] = 0
			encoded[i*4+2] = 0
			encoded[i*4+3] = 0
		} else {
			normalized := (val - sq.minVals[i]) / rng
			quantized := uint32(math.Min(4294967295.0, float64(normalized*4294967295)))
			encoded[i*4] = byte(quantized)
			encoded[i*4+1] = byte(quantized >> 8)
			encoded[i*4+2] = byte(quantized >> 16)
			encoded[i*4+3] = byte(quantized >> 24)
		}
	}
	return encoded
}

// Decode 解码向量
func (sq *ScalarQuantizer) Decode(encoded []byte) []float32 {
	if !sq.trained {
		return nil
	}

	switch sq.bits {
	case 8:
		return sq.decode8(encoded)
	case 16:
		return sq.decode16(encoded)
	case 32:
		return sq.decode32(encoded)
	default:
		return nil
	}
}

// decode8 8 位解码
func (sq *ScalarQuantizer) decode8(encoded []byte) []float32 {
	decoded := make([]float32, len(encoded))
	for i := range decoded {
		normalized := float32(encoded[i]) / 255.0
		decoded[i] = sq.minVals[i] + normalized*(sq.maxVals[i]-sq.minVals[i])
	}
	return decoded
}

// decode16 16 位解码
func (sq *ScalarQuantizer) decode16(encoded []byte) []float32 {
	decoded := make([]float32, len(encoded)/2)
	for i := range decoded {
		val := uint16(encoded[i*2]) | (uint16(encoded[i*2+1]) << 8)
		normalized := float32(val) / 65535.0
		decoded[i] = sq.minVals[i] + normalized*(sq.maxVals[i]-sq.minVals[i])
	}
	return decoded
}

// decode32 32 位解码
func (sq *ScalarQuantizer) decode32(encoded []byte) []float32 {
	decoded := make([]float32, len(encoded)/4)
	for i := range decoded {
		val := uint32(encoded[i*4]) | (uint32(encoded[i*4+1]) << 8) |
			(uint32(encoded[i*4+2]) << 16) | (uint32(encoded[i*4+3]) << 24)
		normalized := float32(val) / 4294967295.0
		decoded[i] = sq.minVals[i] + normalized*(sq.maxVals[i]-sq.minVals[i])
	}
	return decoded
}

// Quantize 量化向量
func Quantize(vectors [][]float32, bits int) (*ScalarQuantizer, error) {
	if len(vectors) == 0 {
		return nil, nil
	}

	sq := NewScalarQuantizer(len(vectors[0]), bits)
	if err := sq.Train(vectors); err != nil {
		return nil, err
	}

	return sq, nil
}
