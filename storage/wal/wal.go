package wal

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/adnilis/x-hmsw/types"
	"github.com/adnilis/x-hmsw/utils/logger"
)

// WALRecordType WAL记录类型
type WALRecordType uint8

const (
	WALInsert WALRecordType = iota + 1
	WALDelete
	WALBatch
	WALCheckpoint
)

// WALRecord WAL记录
type WALRecord struct {
	Type      WALRecordType
	Timestamp int64
	Data      []byte
}

// WAL Write-Ahead Log实现
type WAL struct {
	file      *os.File
	path      string
	mu        sync.Mutex
	logger    logger.Logger
	closed    bool
	batchSize int
}

// NewWAL 创建新的WAL
func NewWAL(path string) (*WAL, error) {
	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("failed to create wal directory: %w", err)
	}

	// 打开或创建WAL文件
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open wal file: %w", err)
	}

	return &WAL{
		file:      file,
		path:      path,
		logger:    logger.NewLogger("storage.wal"),
		closed:    false,
		batchSize: 1000,
	}, nil
}

// WriteInsert 写入插入记录
func (w *WAL) WriteInsert(vec *types.Vector) error {
	if w.closed {
		return fmt.Errorf("wal is closed")
	}

	data, err := json.Marshal(vec)
	if err != nil {
		return fmt.Errorf("failed to marshal vector: %w", err)
	}

	record := WALRecord{
		Type:      WALInsert,
		Timestamp: time.Now().UnixNano(),
		Data:      data,
	}

	return w.writeRecord(&record)
}

// WriteDelete 写入删除记录
func (w *WAL) WriteDelete(id string) error {
	if w.closed {
		return fmt.Errorf("wal is closed")
	}

	data, err := json.Marshal(id)
	if err != nil {
		return fmt.Errorf("failed to marshal id: %w", err)
	}

	record := WALRecord{
		Type:      WALDelete,
		Timestamp: time.Now().UnixNano(),
		Data:      data,
	}

	return w.writeRecord(&record)
}

// WriteBatch 写入批量操作记录
func (w *WAL) WriteBatch(vectors []*types.Vector, ids []string) error {
	if w.closed {
		return fmt.Errorf("wal is closed")
	}

	batchData := struct {
		Vectors []*types.Vector `json:"vectors"`
		IDs     []string        `json:"ids"`
	}{
		Vectors: vectors,
		IDs:     ids,
	}

	data, err := json.Marshal(batchData)
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	record := WALRecord{
		Type:      WALBatch,
		Timestamp: time.Now().UnixNano(),
		Data:      data,
	}

	return w.writeRecord(&record)
}

// WriteCheckpoint 写入检查点记录
func (w *WAL) WriteCheckpoint() error {
	if w.closed {
		return fmt.Errorf("wal is closed")
	}

	record := WALRecord{
		Type:      WALCheckpoint,
		Timestamp: time.Now().UnixNano(),
		Data:      []byte{},
	}

	return w.writeRecord(&record)
}

// writeRecord 写入记录到WAL文件
func (w *WAL) writeRecord(record *WALRecord) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 序列化记录
	recordData, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	// 写入记录长度（4字节）
	length := uint32(len(recordData))
	if err := binary.Write(w.file, binary.LittleEndian, length); err != nil {
		return fmt.Errorf("failed to write record length: %w", err)
	}

	// 写入记录数据
	if _, err := w.file.Write(recordData); err != nil {
		return fmt.Errorf("failed to write record data: %w", err)
	}

	// 立即刷新到磁盘
	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync wal file: %w", err)
	}

	w.logger.Debug("wal record written",
		"type", uint8(record.Type),
		"size", len(recordData))

	return nil
}

// Recover 从WAL恢复数据
func (w *WAL) Recover() ([]*types.Vector, []string, error) {
	if w.closed {
		return nil, nil, fmt.Errorf("wal is closed")
	}

	// 关闭当前文件
	w.mu.Lock()
	w.file.Close()
	w.mu.Unlock()

	// 重新打开文件进行读取
	file, err := os.Open(w.path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open wal file for recovery: %w", err)
	}
	defer file.Close()

	var vectors []*types.Vector
	var ids []string

	// 读取所有记录
	for {
		// 读取记录长度
		var length uint32
		if err := binary.Read(file, binary.LittleEndian, &length); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, nil, fmt.Errorf("failed to read record length: %w", err)
		}

		// 读取记录数据
		recordData := make([]byte, length)
		if _, err := file.Read(recordData); err != nil {
			return nil, nil, fmt.Errorf("failed to read record data: %w", err)
		}

		// 反序列化记录
		var record WALRecord
		if err := json.Unmarshal(recordData, &record); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal record: %w", err)
		}

		// 处理记录
		switch record.Type {
		case WALInsert:
			var vec types.Vector
			if err := json.Unmarshal(record.Data, &vec); err != nil {
				w.logger.Warn("failed to unmarshal insert record", "error", err)
				continue
			}
			vectors = append(vectors, &vec)

		case WALDelete:
			var id string
			if err := json.Unmarshal(record.Data, &id); err != nil {
				w.logger.Warn("failed to unmarshal delete record", "error", err)
				continue
			}
			ids = append(ids, id)

		case WALBatch:
			var batchData struct {
				Vectors []*types.Vector `json:"vectors"`
				IDs     []string        `json:"ids"`
			}
			if err := json.Unmarshal(record.Data, &batchData); err != nil {
				w.logger.Warn("failed to unmarshal batch record", "error", err)
				continue
			}
			vectors = append(vectors, batchData.Vectors...)
			ids = append(ids, batchData.IDs...)

		case WALCheckpoint:
			// 检查点之前的记录可以忽略
			w.logger.Info("reached checkpoint, clearing recovered data")
			vectors = nil
			ids = nil
		}
	}

	w.logger.Info("wal recovery completed",
		"vectors", len(vectors),
		"deletes", len(ids))

	// 重新打开文件用于写入
	w.mu.Lock()
	w.file, err = os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	w.mu.Unlock()

	if err != nil {
		return nil, nil, fmt.Errorf("failed to reopen wal file: %w", err)
	}

	return vectors, ids, nil
}

// Truncate 清空WAL文件
func (w *WAL) Truncate() error {
	if w.closed {
		return fmt.Errorf("wal is closed")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// 关闭当前文件
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("failed to close wal file: %w", err)
	}

	// 删除并重新创建文件
	if err := os.Remove(w.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove wal file: %w", err)
	}

	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to recreate wal file: %w", err)
	}

	w.file = file
	w.logger.Info("wal truncated")

	return nil
}

// Close 关闭WAL
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}

	if err := w.file.Close(); err != nil {
		return fmt.Errorf("failed to close wal file: %w", err)
	}

	w.closed = true
	w.logger.Info("wal closed")

	return nil
}
