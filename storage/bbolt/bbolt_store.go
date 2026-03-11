package bbolt

import (
	"encoding/json"
	"fmt"

	"github.com/adnilis/x-hmsw/types"
	"go.etcd.io/bbolt"
)

// BBoltStorage BBolt 存储引擎
type BBoltStorage struct {
	db     *bbolt.DB
	bucket []byte
	closed bool
}

// NewBBoltStorage 创建 BBolt 存储
func NewBBoltStorage(path string) (*BBoltStorage, error) {
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open bbolt db: %w", err)
	}

	// 创建 bucket
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("vectors"))
		return err
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &BBoltStorage{
		db:     db,
		bucket: []byte("vectors"),
		closed: false,
	}, nil
}

// Get 获取向量
func (s *BBoltStorage) Get(id string) (*types.Vector, error) {
	if s.closed {
		return nil, fmt.Errorf("storage is closed")
	}

	var vec *types.Vector
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(s.bucket)
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("vector not found: %s", id)
		}
		return json.Unmarshal(data, &vec)
	})

	return vec, err
}

// BatchGet 批量获取
func (s *BBoltStorage) BatchGet(ids []string) ([]*types.Vector, error) {
	if s.closed {
		return nil, fmt.Errorf("storage is closed")
	}

	vectors := make([]*types.Vector, len(ids))
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(s.bucket)
		for i, id := range ids {
			data := bucket.Get([]byte(id))
			if data != nil {
				var vec types.Vector
				if err := json.Unmarshal(data, &vec); err == nil {
					vectors[i] = &vec
				}
			}
		}
		return nil
	})

	return vectors, err
}

// Put 存储向量
func (s *BBoltStorage) Put(vec *types.Vector) error {
	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	data, err := json.Marshal(vec)
	if err != nil {
		return err
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(s.bucket)
		return bucket.Put([]byte(vec.ID), data)
	})
}

// BatchPut 批量存储
func (s *BBoltStorage) BatchPut(vectors []*types.Vector) error {
	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(s.bucket)
		for _, vec := range vectors {
			data, err := json.Marshal(vec)
			if err != nil {
				return err
			}
			if err := bucket.Put([]byte(vec.ID), data); err != nil {
				return err
			}
		}
		return nil
	})
}

// Delete 删除向量
func (s *BBoltStorage) Delete(id string) error {
	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(s.bucket)
		return bucket.Delete([]byte(id))
	})
}

// BatchDelete 批量删除
func (s *BBoltStorage) BatchDelete(ids []string) error {
	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(s.bucket)
		for _, id := range ids {
			if err := bucket.Delete([]byte(id)); err != nil {
				return err
			}
		}
		return nil
	})
}

// Iterate 遍历所有向量
func (s *BBoltStorage) Iterate(fn func(*types.Vector) bool) error {
	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	return s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(s.bucket)
		return bucket.ForEach(func(key, value []byte) error {
			var vec types.Vector
			if err := json.Unmarshal(value, &vec); err != nil {
				return err
			}
			if !fn(&vec) {
				return fmt.Errorf("iteration stopped")
			}
			return nil
		})
	})
}

// Count 统计数量
func (s *BBoltStorage) Count() (int, error) {
	if s.closed {
		return 0, fmt.Errorf("storage is closed")
	}

	count := 0
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(s.bucket)
		count = bucket.Stats().KeyN
		return nil
	})
	return count, err
}

// Save 保存
func (s *BBoltStorage) Save(path string) error {
	return nil
}

// Load 加载
func (s *BBoltStorage) Load(path string) error {
	return nil
}

// Close 关闭
func (s *BBoltStorage) Close() error {
	s.closed = true
	return s.db.Close()
}
