package msgpack

import (
	"bytes"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

// Vector 向量数据结构
type Vector struct {
	ID        string
	Vector    []float32
	Payload   map[string]interface{}
	Timestamp time.Time
}

// Serializer Msgpack 序列化器
type Serializer struct{}

// NewSerializer 创建序列化器
func NewSerializer() *Serializer {
	return &Serializer{}
}

// Serialize 序列化向量
func (s *Serializer) Serialize(vec *Vector) ([]byte, error) {
	return msgpack.Marshal(vec)
}

// Deserialize 反序列化向量
func (s *Serializer) Deserialize(data []byte) (*Vector, error) {
	var vec Vector
	err := msgpack.Unmarshal(data, &vec)
	return &vec, err
}

// SerializeBatch 批量序列化
func (s *Serializer) SerializeBatch(vectors []*Vector) ([]byte, error) {
	return msgpack.Marshal(vectors)
}

// DeserializeBatch 批量反序列化
func (s *Serializer) DeserializeBatch(data []byte) ([]*Vector, error) {
	var vectors []*Vector
	err := msgpack.Unmarshal(data, &vectors)
	return vectors, err
}

// Encode 编码
func Encode(vec *Vector) ([]byte, error) {
	return msgpack.Marshal(vec)
}

// Decode 解码
func Decode(data []byte) (*Vector, error) {
	var vec Vector
	err := msgpack.Unmarshal(data, &vec)
	return &vec, err
}

// EncodeBatch 批量编码
func EncodeBatch(vectors []*Vector) ([]byte, error) {
	return msgpack.Marshal(vectors)
}

// DecodeBatch 批量解码
func DecodeBatch(data []byte) ([]*Vector, error) {
	var vectors []*Vector
	err := msgpack.Unmarshal(data, &vectors)
	return vectors, err
}

// Compress 压缩（使用 msgpack 的压缩特性）
func Compress(vec *Vector) ([]byte, error) {
	data, err := msgpack.Marshal(vec)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Decompress 解压缩
func Decompress(data []byte) (*Vector, error) {
	var vec Vector
	err := msgpack.Unmarshal(data, &vec)
	return &vec, err
}

// Serializer 接口实现
type VectorData interface {
	GetID() string
	GetVector() []float32
	GetPayload() map[string]interface{}
	GetTimestamp() time.Time
}

// EncodeInterface 编码接口数据
func EncodeInterface(data VectorData) ([]byte, error) {
	return msgpack.Marshal(data)
}

// DecodeInterface 解码接口数据
func DecodeInterface(data []byte) (*Vector, error) {
	var vec Vector
	err := msgpack.Unmarshal(data, &vec)
	return &vec, err
}

// Buffer 序列化到缓冲区
func Buffer(buf *bytes.Buffer, vec *Vector) error {
	data, err := msgpack.Marshal(vec)
	if err != nil {
		return err
	}
	_, err = buf.Write(data)
	return err
}

// BufferDecode 从缓冲区解码
func BufferDecode(buf *bytes.Buffer) (*Vector, error) {
	data := buf.Bytes()
	var vec Vector
	err := msgpack.Unmarshal(data, &vec)
	return &vec, err
}
