package validation

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"

	"github.com/adnilis/x-hmsw/types"
	"github.com/adnilis/x-hmsw/utils/logger"
)

// Validator 数据验证器
type Validator struct {
	logger logger.Logger
}

// NewValidator 创建验证器
func NewValidator() *Validator {
	return &Validator{
		logger: logger.NewLogger("storage.validation"),
	}
}

// ValidateVector 验证向量数据完整性
func (v *Validator) ValidateVector(vec *types.Vector) error {
	if vec == nil {
		return fmt.Errorf("vector is nil")
	}

	if vec.ID == "" {
		return fmt.Errorf("vector id is empty")
	}

	// 允许空向量（当没有设置embedding函数时，向量将在搜索时生成）
	if len(vec.Vector) == 0 {
		return nil
	}

	// 验证数据是否为有效浮点数
	for i, val := range vec.Vector {
		if val != val { // NaN检查
			return fmt.Errorf("vector data contains NaN at index %d", i)
		}
	}

	return nil
}

// ValidateBatch 验证批量向量数据
func (v *Validator) ValidateBatch(vectors []*types.Vector) error {
	if len(vectors) == 0 {
		return fmt.Errorf("batch is empty")
	}

	// 检查维度一致性
	if len(vectors) > 1 {
		dim := len(vectors[0].Vector)
		for i, vec := range vectors {
			if len(vec.Vector) != dim {
				return fmt.Errorf("vector at index %d has inconsistent dimension: expected %d, got %d",
					i, dim, len(vec.Vector))
			}
		}
	}

	// 验证每个向量
	for i, vec := range vectors {
		if err := v.ValidateVector(vec); err != nil {
			return fmt.Errorf("vector at index %d validation failed: %w", i, err)
		}
	}

	return nil
}

// ComputeChecksum 计算向量的校验和
func (v *Validator) ComputeChecksum(vec *types.Vector) uint32 {
	// 使用CRC32计算校验和
	h := crc32.NewIEEE()

	// 写入ID
	binary.Write(h, binary.LittleEndian, uint32(len(vec.ID)))
	h.Write([]byte(vec.ID))

	// 写入数据
	for _, val := range vec.Vector {
		binary.Write(h, binary.LittleEndian, val)
	}

	return h.Sum32()
}

// VerifyChecksum 验证向量校验和
func (v *Validator) VerifyChecksum(vec *types.Vector, checksum uint32) bool {
	computed := v.ComputeChecksum(vec)
	return computed == checksum
}

// VerifyPayloadChecksum 验证存储的校验和
func (v *Validator) VerifyPayloadChecksum(payload []byte, storedChecksum uint32) bool {
	computed := crc32.ChecksumIEEE(payload)
	return computed == storedChecksum
}

// ValidateID 验证ID格式
func (v *Validator) ValidateID(id string) error {
	if id == "" {
		return fmt.Errorf("id is empty")
	}

	// 检查ID长度
	if len(id) > 1024 {
		return fmt.Errorf("id too long: %d > 1024", len(id))
	}

	// 检查非法字符
	for _, c := range id {
		if c < 32 || c > 126 {
			return fmt.Errorf("id contains invalid character: %c", c)
		}
	}

	return nil
}

// ValidateDimension 验证向量维度
func (v *Validator) ValidateDimension(data []float32, expectedDim int) error {
	if len(data) != expectedDim {
		return fmt.Errorf("dimension mismatch: expected %d, got %d", expectedDim, len(data))
	}
	return nil
}

// ValidateSearchOptions 验证搜索选项
func (v *Validator) ValidateSearchOptions(k int, ef int) error {
	if k <= 0 {
		return fmt.Errorf("k must be positive: %d", k)
	}

	if k > 10000 {
		return fmt.Errorf("k too large: %d > 10000", k)
	}

	if ef <= 0 {
		return fmt.Errorf("ef must be positive: %d", ef)
	}

	if ef > 100000 {
		return fmt.Errorf("ef too large: %d > 100000", ef)
	}

	return nil
}

// ValidateConfig 验证配置
func (v *Validator) ValidateConfig(dimension int, m int, efConstruction int) error {
	if dimension <= 0 {
		return fmt.Errorf("dimension must be positive: %d", dimension)
	}

	if dimension > 10000 {
		return fmt.Errorf("dimension too large: %d > 10000", dimension)
	}

	if m <= 0 {
		return fmt.Errorf("m must be positive: %d", m)
	}

	if m > 128 {
		return fmt.Errorf("m too large: %d > 128", m)
	}

	if efConstruction <= 0 {
		return fmt.Errorf("efConstruction must be positive: %d", efConstruction)
	}

	if efConstruction > 1000 {
		return fmt.Errorf("efConstruction too large: %d > 1000", efConstruction)
	}

	return nil
}
