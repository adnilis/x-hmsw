package protobuf

import (
	"time"

	"google.golang.org/protobuf/proto"
)

//go:generate protoc --go_out=. --go_opt=paths=source_relative vector.proto

// Vector 向量数据
type Vector struct {
	ID        string
	Vector    []float32
	Payload   map[string]string
	Timestamp time.Time
}

// Serializer Protobuf 序列化器
type Serializer struct{}

// NewSerializer 创建序列化器
func NewSerializer() *Serializer {
	return &Serializer{}
}

// Serialize 序列化向量
func (s *Serializer) Serialize(vec *Vector) ([]byte, error) {
	// 转换为 protobuf 消息
	vector32 := make([]float32, len(vec.Vector))
	copy(vector32, vec.Vector)
	pbVec := &VectorPB{
		Id:        vec.ID,
		Vector:    vector32,
		Payload:   vec.Payload,
		Timestamp: vec.Timestamp.UnixNano(),
	}
	return proto.Marshal(pbVec)
}

// Deserialize 反序列化向量
func (s *Serializer) Deserialize(data []byte) (*Vector, error) {
	var pbVec VectorPB
	if err := proto.Unmarshal(data, &pbVec); err != nil {
		return nil, err
	}

	vector32 := make([]float32, len(pbVec.Vector))
	for i, v := range pbVec.Vector {
		vector32[i] = float32(v)
	}

	return &Vector{
		ID:        pbVec.Id,
		Vector:    vector32,
		Payload:   pbVec.Payload,
		Timestamp: time.Unix(0, pbVec.Timestamp),
	}, nil
}

// SerializeBatch 批量序列化
func (s *Serializer) SerializeBatch(vectors []*Vector) ([]byte, error) {
	pbVectors := make([]*VectorPB, len(vectors))
	for i, vec := range vectors {
		vector32 := make([]float32, len(vec.Vector))
		copy(vector32, vec.Vector)
		pbVectors[i] = &VectorPB{
			Id:        vec.ID,
			Vector:    vector32,
			Payload:   vec.Payload,
			Timestamp: vec.Timestamp.UnixNano(),
		}
	}

	batch := &VectorBatch{Vectors: pbVectors}
	return proto.Marshal(batch)
}

// DeserializeBatch 批量反序列化
func (s *Serializer) DeserializeBatch(data []byte) ([]*Vector, error) {
	var batch VectorBatch
	if err := proto.Unmarshal(data, &batch); err != nil {
		return nil, err
	}

	vectors := make([]*Vector, len(batch.Vectors))
	for i, pbVec := range batch.Vectors {
		vector32 := make([]float32, len(pbVec.Vector))
		for j, v := range pbVec.Vector {
			vector32[j] = float32(v)
		}
		vectors[i] = &Vector{
			ID:        pbVec.Id,
			Vector:    pbVec.Vector,
			Payload:   pbVec.Payload,
			Timestamp: time.Unix(0, pbVec.Timestamp),
		}
	}

	return vectors, nil
}

// Encode 编码
func Encode(vec *Vector) ([]byte, error) {
	pbVec := &VectorPB{
		Id:        vec.ID,
		Vector:    vec.Vector,
		Payload:   vec.Payload,
		Timestamp: vec.Timestamp.UnixNano(),
	}
	return proto.Marshal(pbVec)
}

// Decode 解码
func Decode(data []byte) (*Vector, error) {
	var pbVec VectorPB
	if err := proto.Unmarshal(data, &pbVec); err != nil {
		return nil, err
	}

	vector32 := make([]float32, len(pbVec.Vector))
	for i, v := range pbVec.Vector {
		vector32[i] = float32(v)
	}

	return &Vector{
		ID:        pbVec.Id,
		Vector:    vector32,
		Payload:   pbVec.Payload,
		Timestamp: time.Unix(0, pbVec.Timestamp),
	}, nil
}

// Marshal 序列化
func (m *VectorPB) Marshal() ([]byte, error) {
	return proto.Marshal(m)
}

// Unmarshal 反序列化
func (m *VectorPB) Unmarshal(data []byte) error {
	return proto.Unmarshal(data, m)
}

// Marshal 批量序列化
func (b *VectorBatch) Marshal() ([]byte, error) {
	return proto.Marshal(b)
}

// Unmarshal 批量反序列化
func (b *VectorBatch) Unmarshal(data []byte) error {
	return proto.Unmarshal(data, b)
}
