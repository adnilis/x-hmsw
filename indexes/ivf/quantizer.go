package ivf

// Quantizer 量化器接口
type Quantizer interface {
	Train(vectors [][]float32) error
	Encode(vector []float32) []byte
	Decode(encoded []byte) []float32
}

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

	// 初始化最小最大值
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
	if !sq.trained {
		return nil
	}

	bytesPerDim := (sq.bits + 7) / 8
	encoded := make([]byte, len(vector)*bytesPerDim)

	switch sq.bits {
	case 8:
		for i, val := range vector {
			normalized := (val - sq.minVals[i]) / (sq.maxVals[i] - sq.minVals[i])
			quantized := uint8(normalized * 255)
			encoded[i] = quantized
		}
	case 16:
		for i, val := range vector {
			normalized := (val - sq.minVals[i]) / (sq.maxVals[i] - sq.minVals[i])
			quantized := uint16(normalized * 65535)
			encoded[i*2] = byte(quantized)
			encoded[i*2+1] = byte(quantized >> 8)
		}
	}

	return encoded
}

// Decode 解码向量
func (sq *ScalarQuantizer) Decode(encoded []byte) []float32 {
	if !sq.trained {
		return nil
	}

	bytesPerDim := (sq.bits + 7) / 8
	dimension := len(encoded) / bytesPerDim
	decoded := make([]float32, dimension)

	switch sq.bits {
	case 8:
		for i := range decoded {
			normalized := float32(encoded[i]) / 255.0
			decoded[i] = sq.minVals[i] + normalized*(sq.maxVals[i]-sq.minVals[i])
		}
	case 16:
		for i := range decoded {
			val := uint16(encoded[i*2]) | (uint16(encoded[i*2+1]) << 8)
			normalized := float32(val) / 65535.0
			decoded[i] = sq.minVals[i] + normalized*(sq.maxVals[i]-sq.minVals[i])
		}
	}

	return decoded
}
