package badger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/adnilis/x-hmsw/storage/backup"
	"github.com/adnilis/x-hmsw/storage/validation"
	"github.com/adnilis/x-hmsw/storage/wal"
	"github.com/adnilis/x-hmsw/types"
	"github.com/adnilis/x-hmsw/utils/logger"
	"github.com/dgraph-io/badger/v3"
)

// Vector 向量数据结构（别名）
type Vector = types.Vector

// BadgerStorageOptions Badger存储配置选项
type BadgerStorageOptions struct {
	Path         string
	SyncWrites   bool
	EnableWAL    bool
	EnableBackup bool
	BackupDir    string
}

// BadgerStorage BadgerDB存储引擎
type BadgerStorage struct {
	db           *badger.DB
	path         string
	logger       logger.Logger
	closed       bool
	wal          *wal.WAL
	validator    *validation.Validator
	backupMgr    *backup.BackupManager
	syncWrites   bool
	enableWAL    bool
	enableBackup bool
}

// NewBadgerStorage 创建Badger存储（默认配置）
func NewBadgerStorage(path string) (*BadgerStorage, error) {
	return NewBadgerStorageWithOptions(BadgerStorageOptions{
		Path:         path,
		SyncWrites:   false,
		EnableWAL:    true,
		EnableBackup: true,
		BackupDir:    filepath.Join(path, "backups"),
	})
}

// NewBadgerStorageWithOptions 使用自定义选项创建Badger存储
func NewBadgerStorageWithOptions(opts BadgerStorageOptions) (*BadgerStorage, error) {
	// 确保目录存在
	if err := os.MkdirAll(opts.Path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// 配置BadgerDB
	badgerOpts := badger.DefaultOptions(opts.Path)
	badgerOpts.Logger = nil
	badgerOpts.SyncWrites = opts.SyncWrites
	badgerOpts.NumVersionsToKeep = 1
	badgerOpts.ValueLogFileSize = 256 << 20 // 256MB

	db, err := badger.Open(badgerOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger db: %w", err)
	}

	storage := &BadgerStorage{
		db:           db,
		path:         opts.Path,
		logger:       logger.NewLogger("storage.badger"),
		closed:       false,
		syncWrites:   opts.SyncWrites,
		enableWAL:    opts.EnableWAL,
		enableBackup: opts.EnableBackup,
		validator:    validation.NewValidator(),
	}

	// 初始化WAL
	if opts.EnableWAL {
		walPath := filepath.Join(opts.Path, "wal.log")
		storage.wal, err = wal.NewWAL(walPath)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to create wal: %w", err)
		}

		// 从WAL恢复数据
		if err := storage.recoverFromWAL(); err != nil {
			storage.logger.Warn("failed to recover from wal", "error", err)
		}
	}

	// 初始化备份管理器
	if opts.EnableBackup {
		storage.backupMgr, err = backup.NewBackupManager(db, opts.BackupDir)
		if err != nil {
			if storage.wal != nil {
				storage.wal.Close()
			}
			db.Close()
			return nil, fmt.Errorf("failed to create backup manager: %w", err)
		}
	}

	storage.logger.Info("badger storage initialized",
		"syncWrites", opts.SyncWrites,
		"enableWAL", opts.EnableWAL,
		"enableBackup", opts.EnableBackup)

	return storage, nil
}

// recoverFromWAL 从WAL恢复数据
func (s *BadgerStorage) recoverFromWAL() error {
	vectors, ids, err := s.wal.Recover()
	if err != nil {
		return err
	}

	// 恢复插入的向量
	if len(vectors) > 0 {
		for _, vec := range vectors {
			if err := s.putWithoutWAL(vec); err != nil {
				s.logger.Warn("failed to recover vector",
					"id", vec.ID,
					"error", err)
			}
		}
	}

	// 恢复删除的ID
	if len(ids) > 0 {
		s.logger.Info("recovering deletions from wal", "count", len(ids))
		for _, id := range ids {
			if err := s.deleteWithoutWAL(id); err != nil {
				s.logger.Warn("failed to recover deletion",
					"id", id,
					"error", err)
			}
		}
	}

	// 清空WAL
	if err := s.wal.Truncate(); err != nil {
		s.logger.Warn("failed to truncate wal after recovery", "error", err)
	}

	return nil
}

// 获取向量
func (s *BadgerStorage) Get(id string) (*Vector, error) {
	if s.closed {
		return nil, fmt.Errorf("storage is closed")
	}

	var vec *Vector
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(id))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &vec)
		})
	})

	if err != nil {
		return nil, err
	}

	return vec, nil
}

// 批量获取
func (s *BadgerStorage) BatchGet(ids []string) ([]*Vector, error) {
	if s.closed {
		return nil, fmt.Errorf("storage is closed")
	}

	vectors := make([]*Vector, len(ids))

	err := s.db.View(func(txn *badger.Txn) error {
		for i, id := range ids {
			item, err := txn.Get([]byte(id))
			if err != nil {
				vectors[i] = nil
				continue
			}

			err = item.Value(func(val []byte) error {
				var vec Vector
				if err := json.Unmarshal(val, &vec); err != nil {
					return err
				}
				vectors[i] = &vec
				return nil
			})

			if err != nil {
				vectors[i] = nil
			}
		}
		return nil
	})

	return vectors, err
}

// 插入向量
func (s *BadgerStorage) Put(vec *Vector) error {
	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	// 验证向量数据
	if err := s.validator.ValidateVector(vec); err != nil {
		return fmt.Errorf("vector validation failed: %w", err)
	}

	// 写入WAL
	if s.enableWAL {
		if err := s.wal.WriteInsert(vec); err != nil {
			return fmt.Errorf("failed to write to wal: %w", err)
		}
	}

	// 写入存储
	if err := s.putWithoutWAL(vec); err != nil {
		return err
	}

	return nil
}

// putWithoutWAL 不写入WAL的内部插入方法
func (s *BadgerStorage) putWithoutWAL(vec *Vector) error {
	data, err := json.Marshal(vec)
	if err != nil {
		return err
	}

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(vec.ID), data)
	})
}

// 批量插入
func (s *BadgerStorage) BatchPut(vectors []*Vector) error {
	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	// 验证批量数据
	if err := s.validator.ValidateBatch(vectors); err != nil {
		return fmt.Errorf("batch validation failed: %w", err)
	}

	// 写入WAL
	if s.enableWAL {
		if err := s.wal.WriteBatch(vectors, nil); err != nil {
			return fmt.Errorf("failed to write batch to wal: %w", err)
		}
	}

	// 写入存储
	wb := s.db.NewWriteBatch()
	defer wb.Cancel()

	for _, vec := range vectors {
		data, err := json.Marshal(vec)
		if err != nil {
			return err
		}

		if err := wb.Set([]byte(vec.ID), data); err != nil {
			return err
		}
	}

	return wb.Flush()
}

// 删除向量
func (s *BadgerStorage) Delete(id string) error {
	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	// 验证ID
	if err := s.validator.ValidateID(id); err != nil {
		return fmt.Errorf("id validation failed: %w", err)
	}

	// 写入WAL
	if s.enableWAL {
		if err := s.wal.WriteDelete(id); err != nil {
			return fmt.Errorf("failed to write deletion to wal: %w", err)
		}
	}

	// 从存储删除
	if err := s.deleteWithoutWAL(id); err != nil {
		return err
	}

	return nil
}

// deleteWithoutWAL 不写入WAL的内部删除方法
func (s *BadgerStorage) deleteWithoutWAL(id string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(id))
	})
}

// 批量删除
func (s *BadgerStorage) BatchDelete(ids []string) error {
	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	// 验证所有ID
	for _, id := range ids {
		if err := s.validator.ValidateID(id); err != nil {
			return fmt.Errorf("id validation failed for %s: %w", id, err)
		}
	}

	// 写入WAL
	if s.enableWAL {
		if err := s.wal.WriteBatch(nil, ids); err != nil {
			return fmt.Errorf("failed to write batch deletion to wal: %w", err)
		}
	}

	// 从存储删除
	wb := s.db.NewWriteBatch()
	defer wb.Cancel()

	for _, id := range ids {
		if err := wb.Delete([]byte(id)); err != nil {
			return err
		}
	}

	return wb.Flush()
}

// 迭代所有向量
func (s *BadgerStorage) Iterate(fn func(*Vector) bool) error {
	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	return s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 100

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			var vec Vector

			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &vec)
			})

			if err != nil {
				continue
			}

			if !fn(&vec) {
				break
			}
		}

		return nil
	})
}

// 获取数量
func (s *BadgerStorage) Count() (int, error) {
	if s.closed {
		return 0, fmt.Errorf("storage is closed")
	}

	count := 0
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			count++
		}

		return nil
	})

	return count, err
}

// Save 保存到文件
func (s *BadgerStorage) Save(path string) error {
	if s.closed {
		return fmt.Errorf("storage is closed")
	}
	// BadgerDB 已经有自己的持久化机制
	// 这里创建一个备份文件
	backupFile := filepath.Join(path, "backup.db")

	// 确保目录存在
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	file, err := os.Create(backupFile)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	_, err = s.db.Backup(file, 0)
	if err != nil {
		return fmt.Errorf("failed to backup database: %w", err)
	}

	return nil
}

// 从文件加载
func (s *BadgerStorage) Load(path string) error {
	// BadgerDB 在打开时自动加载
	return nil
}

// 关闭存储
func (s *BadgerStorage) Close() error {
	if s.closed {
		return nil
	}

	s.logger.Info("closing badger storage")

	// 关闭备份管理器
	if s.backupMgr != nil {
		if err := s.backupMgr.Close(); err != nil {
			s.logger.Warn("failed to close backup manager", "error", err)
		}
	}

	// 关闭WAL
	if s.wal != nil {
		if err := s.wal.Close(); err != nil {
			s.logger.Warn("failed to close wal", "error", err)
		}
	}

	// 关闭数据库
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("failed to close badger db: %w", err)
	}

	s.closed = true
	s.logger.Info("badger storage closed successfully")

	return nil
}

// Backup 备份数据库
func (s *BadgerStorage) Backup(path string) error {
	// BadgerDB 已经有自己的持久化机制
	// 这里创建一个备份
	_, err := s.db.Backup(nil, 0)
	return err
}

// CreateBackup 创建备份
func (s *BadgerStorage) CreateBackup(description string) error {
	if s.backupMgr == nil {
		return fmt.Errorf("backup manager not enabled")
	}

	_, err := s.backupMgr.CreateBackup(description)
	return err
}

// RestoreBackup 从备份恢复
func (s *BadgerStorage) RestoreBackup(backupName string) error {
	if s.backupMgr == nil {
		return fmt.Errorf("backup manager not enabled")
	}

	return s.backupMgr.RestoreBackup(backupName)
}

// ListBackups 列出所有备份
func (s *BadgerStorage) ListBackups() ([]*backup.BackupMetadata, error) {
	if s.backupMgr == nil {
		return nil, fmt.Errorf("backup manager not enabled")
	}

	return s.backupMgr.ListBackups()
}

// DeleteBackup 删除备份
func (s *BadgerStorage) DeleteBackup(backupName string) error {
	if s.backupMgr == nil {
		return fmt.Errorf("backup manager not enabled")
	}

	return s.backupMgr.DeleteBackup(backupName)
}

// EnableAutoBackup 启用自动备份
func (s *BadgerStorage) EnableAutoBackup(interval string) error {
	if s.backupMgr == nil {
		return fmt.Errorf("backup manager not enabled")
	}

	d, err := time.ParseDuration(interval)
	if err != nil {
		return fmt.Errorf("invalid interval: %w", err)
	}

	s.backupMgr.EnableAutoBackup(d)
	return nil
}

// DisableAutoBackup 禁用自动备份
func (s *BadgerStorage) DisableAutoBackup() error {
	if s.backupMgr == nil {
		return fmt.Errorf("backup manager not enabled")
	}

	s.backupMgr.DisableAutoBackup()
	return nil
}

// Checkpoint 创建检查点并清空WAL
func (s *BadgerStorage) Checkpoint() error {
	if s.wal == nil {
		return fmt.Errorf("wal not enabled")
	}

	// 写入检查点记录
	if err := s.wal.WriteCheckpoint(); err != nil {
		return fmt.Errorf("failed to write checkpoint: %w", err)
	}

	// 清空WAL
	if err := s.wal.Truncate(); err != nil {
		return fmt.Errorf("failed to truncate wal: %w", err)
	}

	s.logger.Info("checkpoint created and wal truncated")

	return nil
}
