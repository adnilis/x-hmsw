package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/adnilis/x-hmsw/types"
)

// MemoryStorage 内存存储引擎
type MemoryStorage struct {
	data   map[string]*types.Vector
	mu     sync.RWMutex
	closed bool
}

// NewMemoryStorage 创建内存存储
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]*types.Vector),
	}
}

// Get 获取单个向量
func (s *MemoryStorage) Get(id string) (*types.Vector, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, fmt.Errorf("storage is closed")
	}

	vec, exists := s.data[id]
	if !exists {
		return nil, fmt.Errorf("vector not found: %s", id)
	}

	return vec, nil
}

// BatchGet 批量获取向量
func (s *MemoryStorage) BatchGet(ids []string) ([]*types.Vector, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, fmt.Errorf("storage is closed")
	}

	vectors := make([]*types.Vector, 0, len(ids))
	for _, id := range ids {
		if vec, exists := s.data[id]; exists {
			vectors = append(vectors, vec)
		} else {
			vectors = append(vectors, nil)
		}
	}

	return vectors, nil
}

// Put 存储单个向量
func (s *MemoryStorage) Put(vec *types.Vector) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	if vec.Timestamp.IsZero() {
		vec.Timestamp = time.Now()
	}

	s.data[vec.ID] = vec
	return nil
}

// BatchPut 批量存储向量
func (s *MemoryStorage) BatchPut(vectors []*types.Vector) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	for _, vec := range vectors {
		if vec.Timestamp.IsZero() {
			vec.Timestamp = time.Now()
		}
		s.data[vec.ID] = vec
	}

	return nil
}

// Delete 删除单个向量
func (s *MemoryStorage) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	if _, exists := s.data[id]; !exists {
		return fmt.Errorf("vector not found: %s", id)
	}

	delete(s.data, id)
	return nil
}

// BatchDelete 批量删除向量
func (s *MemoryStorage) BatchDelete(ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	for _, id := range ids {
		delete(s.data, id)
	}

	return nil
}

// Iterate 迭代所有向量
func (s *MemoryStorage) Iterate(fn func(*types.Vector) bool) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	for _, vec := range s.data {
		if !fn(vec) {
			break
		}
	}

	return nil
}

// Count 获取向量数量
func (s *MemoryStorage) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return 0, fmt.Errorf("storage is closed")
	}

	return len(s.data), nil
}

// Save 保存到文件
func (s *MemoryStorage) Save(path string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	// 创建保存目录
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 准备保存的数据
	data := make(map[string]*types.Vector, len(s.data))
	for id, vec := range s.data {
		data[id] = vec
	}

	// 序列化为 JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal vectors: %w", err)
	}

	// 写入文件
	savePath := filepath.Join(path, "vectors.json")
	if err := os.WriteFile(savePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Load 从文件加载
func (s *MemoryStorage) Load(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	// 读取文件
	loadPath := filepath.Join(path, "vectors.json")
	jsonData, err := os.ReadFile(loadPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", loadPath)
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	// 反序列化
	var data map[string]*types.Vector
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return fmt.Errorf("failed to unmarshal vectors: %w", err)
	}

	// 加载到内存
	s.data = data

	return nil
}

// Close 关闭存储
func (s *MemoryStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	return nil
}
