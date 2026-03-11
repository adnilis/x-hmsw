package pebble

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/adnilis/x-hmsw/types"
	"github.com/cockroachdb/pebble"
)

// PebbleStorage PebbleDB 存储引擎
type PebbleStorage struct {
	db     *pebble.DB
	closed bool
}

// NewPebbleStorage 创建 PebbleDB 存储
func NewPebbleStorage(path string) (*PebbleStorage, error) {
	db, err := pebble.Open(path, &pebble.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to open pebble db: %w", err)
	}

	return &PebbleStorage{
		db:     db,
		closed: false,
	}, nil
}

// Get 获取向量
func (s *PebbleStorage) Get(id string) (*types.Vector, error) {
	if s.closed {
		return nil, fmt.Errorf("storage is closed")
	}

	data, closer, err := s.db.Get([]byte(id))
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	var vec types.Vector
	if err := json.Unmarshal(data, &vec); err != nil {
		return nil, err
	}

	return &vec, nil
}

// BatchGet 批量获取
func (s *PebbleStorage) BatchGet(ids []string) ([]*types.Vector, error) {
	if s.closed {
		return nil, fmt.Errorf("storage is closed")
	}

	vectors := make([]*types.Vector, len(ids))
	for i, id := range ids {
		vec, err := s.Get(id)
		if err == nil {
			vectors[i] = vec
		}
	}
	return vectors, nil
}

// Put 存储向量
func (s *PebbleStorage) Put(vec *types.Vector) error {
	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	if vec.Timestamp.IsZero() {
		vec.Timestamp = time.Now()
	}

	data, err := json.Marshal(vec)
	if err != nil {
		return err
	}

	return s.db.Set([]byte(vec.ID), data, pebble.Sync)
}

// BatchPut 批量存储
func (s *PebbleStorage) BatchPut(vectors []*types.Vector) error {
	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	batch := s.db.NewBatch()
	defer batch.Close()

	for _, vec := range vectors {
		if vec.Timestamp.IsZero() {
			vec.Timestamp = time.Now()
		}
		data, err := json.Marshal(vec)
		if err != nil {
			return err
		}
		if err := batch.Set([]byte(vec.ID), data, pebble.Sync); err != nil {
			return err
		}
	}

	return batch.Commit(pebble.Sync)
}

// Delete 删除向量
func (s *PebbleStorage) Delete(id string) error {
	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	return s.db.Delete([]byte(id), pebble.Sync)
}

// BatchDelete 批量删除
func (s *PebbleStorage) BatchDelete(ids []string) error {
	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	batch := s.db.NewBatch()
	defer batch.Close()

	for _, id := range ids {
		if err := batch.Delete([]byte(id), pebble.Sync); err != nil {
			return err
		}
	}

	return batch.Commit(pebble.Sync)
}

// Iterate 遍历所有向量
func (s *PebbleStorage) Iterate(fn func(*types.Vector) bool) error {
	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	iter, _ := s.db.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		var vec types.Vector
		if err := json.Unmarshal(iter.Value(), &vec); err != nil {
			return err
		}
		if !fn(&vec) {
			break
		}
	}

	return iter.Error()
}

// Count 统计数量
func (s *PebbleStorage) Count() (int, error) {
	if s.closed {
		return 0, fmt.Errorf("storage is closed")
	}

	count := 0
	iter, _ := s.db.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}

	return count, iter.Error()
}

// Save 保存
func (s *PebbleStorage) Save(path string) error {
	return nil
}

// Load 加载
func (s *PebbleStorage) Load(path string) error {
	return nil
}

// Close 关闭
func (s *PebbleStorage) Close() error {
	s.closed = true
	return s.db.Close()
}
