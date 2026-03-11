package binary

import (
	"encoding/binary"
	"io"
	"math"
	"time"
)

// Vector 向量数据结构
type Vector struct {
	ID        string
	Vector    []float32
	Payload   map[string]interface{}
	Timestamp time.Time
}

// Serializer 二进制序列化器
type Serializer struct{}

// NewSerializer 创建序列化器
func NewSerializer() *Serializer {
	return &Serializer{}
}

// Serialize 序列化向量
func (s *Serializer) Serialize(vec *Vector) ([]byte, error) {
	// 计算总大小
	size := 4 + len(vec.ID) + 4 + 4*len(vec.Vector) + 4
	data := make([]byte, 0, size)

	// ID 长度 + ID
	idLen := uint32(len(vec.ID))
	data = binary.LittleEndian.AppendUint32(data, idLen)
	data = append(data, []byte(vec.ID)...)

	// 向量长度 + 向量数据
	vecLen := uint32(len(vec.Vector))
	data = binary.LittleEndian.AppendUint32(data, vecLen)
	for _, v := range vec.Vector {
		data = binary.LittleEndian.AppendUint32(data, math.Float32bits(v))
	}

	// 时间戳
	ts := uint64(vec.Timestamp.UnixNano())
	data = binary.LittleEndian.AppendUint64(data, ts)

	return data, nil
}

// Deserialize 反序列化向量
func (s *Serializer) Deserialize(data []byte) (*Vector, error) {
	offset := 0

	// ID 长度
	idLen := binary.LittleEndian.Uint32(data[offset:])
	offset += 4

	// ID
	id := string(data[offset : offset+int(idLen)])
	offset += int(idLen)

	// 向量长度
	vecLen := binary.LittleEndian.Uint32(data[offset:])
	offset += 4

	// 向量数据
	vector := make([]float32, vecLen)
	for i := range vector {
		vector[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[offset:]))
		offset += 4
	}

	// 时间戳
	ts := binary.LittleEndian.Uint64(data[offset:])
	timestamp := time.Unix(0, int64(ts))

	return &Vector{
		ID:        id,
		Vector:    vector,
		Timestamp: timestamp,
	}, nil
}

// SerializeBatch 批量序列化
func (s *Serializer) SerializeBatch(vectors []*Vector) ([]byte, error) {
	// 数量
	data := binary.LittleEndian.AppendUint32(nil, uint32(len(vectors)))

	// 序列化每个向量
	for _, vec := range vectors {
		vecData, err := s.Serialize(vec)
		if err != nil {
			return nil, err
		}
		data = binary.LittleEndian.AppendUint32(data, uint32(len(vecData)))
		data = append(data, vecData...)
	}

	return data, nil
}

// DeserializeBatch 批量反序列化
func (s *Serializer) DeserializeBatch(data []byte) ([]*Vector, error) {
	offset := 0

	// 数量
	count := binary.LittleEndian.Uint32(data[offset:])
	offset += 4

	vectors := make([]*Vector, count)
	for i := range vectors {
		// 向量大小
		size := binary.LittleEndian.Uint32(data[offset:])
		offset += 4

		// 反序列化向量
		vec, err := s.Deserialize(data[offset : offset+int(size)])
		if err != nil {
			return nil, err
		}
		vectors[i] = vec
		offset += int(size)
	}

	return vectors, nil
}

// WriteTo 写入到流
func (s *Serializer) WriteTo(w io.Writer, vec *Vector) error {
	data, err := s.Serialize(vec)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// ReadFrom 从流读取
func (s *Serializer) ReadFrom(r io.Reader) (*Vector, error) {
	// 读取固定大小头部
	header := make([]byte, 8)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	// 解析 ID 长度
	idLen := binary.LittleEndian.Uint32(header)

	// 读取剩余数据
	data := make([]byte, idLen+4)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}

	// 构建完整数据
	fullData := make([]byte, 0, 8+len(data))
	fullData = binary.LittleEndian.AppendUint32(fullData, idLen)
	fullData = append(fullData, data...)

	return s.Deserialize(fullData)
}

// Encode 编码
func Encode(vec *Vector) ([]byte, error) {
	s := NewSerializer()
	return s.Serialize(vec)
}

// Decode 解码
func Decode(data []byte) (*Vector, error) {
	s := NewSerializer()
	return s.Deserialize(data)
}

// Write 写入
func Write(w io.Writer, vec *Vector) error {
	s := NewSerializer()
	return s.WriteTo(w, vec)
}

// Read 读取
func Read(r io.Reader) (*Vector, error) {
	s := NewSerializer()
	return s.ReadFrom(r)
}
