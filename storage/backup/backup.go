package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/adnilis/x-hmsw/utils/logger"
	"github.com/dgraph-io/badger/v3"
)

// BackupMetadata 备份元数据
type BackupMetadata struct {
	Timestamp   time.Time `json:"timestamp"`
	Size        int64     `json:"size"`
	VectorCount int       `json:"vectorCount"`
	Description string    `json:"description"`
}

// BackupManager 备份管理器
type BackupManager struct {
	db          *badger.DB
	backupDir   string
	logger      logger.Logger
	maxBackups  int
	retention   time.Duration
	autoBackup  bool
	backupTimer *time.Timer
	stopChan    chan struct{}
}

// NewBackupManager 创建备份管理器
func NewBackupManager(db *badger.DB, backupDir string) (*BackupManager, error) {
	// 确保备份目录存在
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	return &BackupManager{
		db:         db,
		backupDir:  backupDir,
		logger:     logger.NewLogger("storage.backup"),
		maxBackups: 10,                 // 最多保留10个备份
		retention:  7 * 24 * time.Hour, // 保留7天
		autoBackup: false,
		stopChan:   make(chan struct{}),
	}, nil
}

// CreateBackup 创建备份
func (bm *BackupManager) CreateBackup(description string) (*BackupMetadata, error) {
	timestamp := time.Now()
	backupName := fmt.Sprintf("backup_%s", timestamp.Format("20060102_150405"))
	backupPath := filepath.Join(bm.backupDir, backupName)

	bm.logger.Info("creating backup",
		"name", backupName,
		"description", description)

	// 创建备份目录
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// 创建备份文件
	backupFile := filepath.Join(backupPath, "data.backup")
	file, err := os.Create(backupFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	// 执行备份
	if _, err := bm.db.Backup(file, 0); err != nil {
		return nil, fmt.Errorf("failed to backup database: %w", err)
	}

	// 获取文件大小
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get backup file size: %w", err)
	}

	// 统计向量数量
	vectorCount := 0
	err = bm.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 100
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			vectorCount++
		}
		return nil
	})

	if err != nil {
		bm.logger.Warn("failed to count vectors in backup", "error", err)
	}

	// 创建元数据
	metadata := &BackupMetadata{
		Timestamp:   timestamp,
		Size:        fileInfo.Size(),
		VectorCount: vectorCount,
		Description: description,
	}

	// 保存元数据
	metadataFile := filepath.Join(backupPath, "metadata.json")
	if err := bm.saveMetadata(metadataFile, metadata); err != nil {
		return nil, fmt.Errorf("failed to save backup metadata: %w", err)
	}

	bm.logger.Info("backup created successfully",
		"name", backupName,
		"size", metadata.Size,
		"vectors", metadata.VectorCount)

	// 清理旧备份
	if err := bm.cleanupOldBackups(); err != nil {
		bm.logger.Warn("failed to cleanup old backups", "error", err)
	}

	return metadata, nil
}

// RestoreBackup 从备份恢复
func (bm *BackupManager) RestoreBackup(backupName string) error {
	backupPath := filepath.Join(bm.backupDir, backupName)
	backupFile := filepath.Join(backupPath, "data.backup")

	bm.logger.Info("restoring from backup", "name", backupName)

	// 检查备份是否存在
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		return fmt.Errorf("backup not found: %s", backupName)
	}

	// 打开备份文件
	file, err := os.Open(backupFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	// 恢复数据库
	if err := bm.db.Load(file, 0); err != nil {
		return fmt.Errorf("failed to restore database: %w", err)
	}

	bm.logger.Info("backup restored successfully", "name", backupName)

	return nil
}

// ListBackups 列出所有备份
func (bm *BackupManager) ListBackups() ([]*BackupMetadata, error) {
	entries, err := os.ReadDir(bm.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []*BackupMetadata

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if !strings.HasPrefix(entry.Name(), "backup_") {
			continue
		}

		metadataFile := filepath.Join(bm.backupDir, entry.Name(), "metadata.json")
		metadata, err := bm.loadMetadata(metadataFile)
		if err != nil {
			bm.logger.Warn("failed to load backup metadata",
				"backup", entry.Name(),
				"error", err)
			continue
		}

		backups = append(backups, metadata)
	}

	// 按时间戳排序（最新的在前）
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// DeleteBackup 删除备份
func (bm *BackupManager) DeleteBackup(backupName string) error {
	backupPath := filepath.Join(bm.backupDir, backupName)

	bm.logger.Info("deleting backup", "name", backupName)

	if err := os.RemoveAll(backupPath); err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	bm.logger.Info("backup deleted successfully", "name", backupName)

	return nil
}

// EnableAutoBackup 启用自动备份
func (bm *BackupManager) EnableAutoBackup(interval time.Duration) {
	bm.autoBackup = true
	bm.backupTimer = time.NewTimer(interval)

	bm.logger.Info("auto backup enabled", "interval", interval)

	go bm.autoBackupLoop(interval)
}

// DisableAutoBackup 禁用自动备份
func (bm *BackupManager) DisableAutoBackup() {
	bm.autoBackup = false

	if bm.backupTimer != nil {
		bm.backupTimer.Stop()
	}

	close(bm.stopChan)
	bm.stopChan = make(chan struct{})

	bm.logger.Info("auto backup disabled")
}

// autoBackupLoop 自动备份循环
func (bm *BackupManager) autoBackupLoop(interval time.Duration) {
	for {
		select {
		case <-bm.backupTimer.C:
			if !bm.autoBackup {
				return
			}

			description := fmt.Sprintf("Auto backup at %s", time.Now().Format("2006-01-02 15:04:05"))
			if _, err := bm.CreateBackup(description); err != nil {
				bm.logger.Error("auto backup failed", "error", err)
			}

			bm.backupTimer.Reset(interval)

		case <-bm.stopChan:
			return
		}
	}
}

// cleanupOldBackups 清理旧备份
func (bm *BackupManager) cleanupOldBackups() error {
	backups, err := bm.ListBackups()
	if err != nil {
		return err
	}

	// 删除超过保留期限的备份
	now := time.Now()
	for _, backup := range backups {
		if now.Sub(backup.Timestamp) > bm.retention {
			backupName := fmt.Sprintf("backup_%s", backup.Timestamp.Format("20060102_150405"))
			if err := bm.DeleteBackup(backupName); err != nil {
				bm.logger.Warn("failed to delete old backup",
					"backup", backupName,
					"error", err)
			}
		}
	}

	// 如果备份数量超过限制，删除最旧的
	backups, err = bm.ListBackups()
	if err != nil {
		return err
	}

	for len(backups) > bm.maxBackups {
		oldest := backups[len(backups)-1]
		backupName := fmt.Sprintf("backup_%s", oldest.Timestamp.Format("20060102_150405"))
		if err := bm.DeleteBackup(backupName); err != nil {
			bm.logger.Warn("failed to delete old backup",
				"backup", backupName,
				"error", err)
		}
		backups = backups[:len(backups)-1]
	}

	return nil
}

// saveMetadata 保存元数据
func (bm *BackupManager) saveMetadata(path string, metadata *BackupMetadata) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(metadata)
}

// loadMetadata 加载元数据
func (bm *BackupManager) loadMetadata(path string) (*BackupMetadata, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var metadata BackupMetadata
	if err := json.NewDecoder(file).Decode(&metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// Close 关闭备份管理器
func (bm *BackupManager) Close() error {
	bm.DisableAutoBackup()
	return nil
}
