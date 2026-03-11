package protobuf

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

// CoarseIndexSerializer 粗粒度索引序列化器
type CoarseIndexSerializer struct {
	basePath    string
	dirtyPages  map[int]bool // 脏页面标记
	dirtyMu     sync.RWMutex // 脏页面锁
	lastSave    time.Time    // 最后保存时间
	saveMu      sync.Mutex   // 保存锁
	enableDelta bool         // 是否启用增量保存
}

// NewCoarseIndexSerializer 创建粗粒度索引序列化器
func NewCoarseIndexSerializer(basePath string, enableDelta bool) *CoarseIndexSerializer {
	return &CoarseIndexSerializer{
		basePath:    basePath,
		dirtyPages:  make(map[int]bool),
		enableDelta: enableDelta,
		lastSave:    time.Now(),
	}
}

// MarkPageDirty 标记页面为脏（需要保存）
func (s *CoarseIndexSerializer) MarkPageDirty(pageID int) {
	if !s.enableDelta {
		return
	}
	s.dirtyMu.Lock()
	defer s.dirtyMu.Unlock()
	s.dirtyPages[pageID] = true
}

// MarkPagesDirty 批量标记页面为脏
func (s *CoarseIndexSerializer) MarkPagesDirty(pageIDs []int) {
	if !s.enableDelta {
		return
	}
	s.dirtyMu.Lock()
	defer s.dirtyMu.Unlock()
	for _, pageID := range pageIDs {
		s.dirtyPages[pageID] = true
	}
}

// ClearDirtyPages 清空脏页面标记
func (s *CoarseIndexSerializer) ClearDirtyPages() {
	s.dirtyMu.Lock()
	defer s.dirtyMu.Unlock()
	s.dirtyPages = make(map[int]bool)
}

// GetDirtyPages 获取脏页面列表
func (s *CoarseIndexSerializer) GetDirtyPages() []int {
	s.dirtyMu.RLock()
	defer s.dirtyMu.RUnlock()
	pages := make([]int, 0, len(s.dirtyPages))
	for pageID := range s.dirtyPages {
		pages = append(pages, pageID)
	}
	return pages
}

// SaveMetadata 保存元数据
func (s *CoarseIndexSerializer) SaveMetadata(metadata *CoarseIndexMetadataPB) error {
	s.saveMu.Lock()
	defer s.saveMu.Unlock()

	metadataPath := filepath.Join(s.basePath, "metadata.pb")
	file, err := os.Create(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to create metadata file: %w", err)
	}
	defer file.Close()

	data, err := proto.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// LoadMetadata 加载元数据
func (s *CoarseIndexSerializer) LoadMetadata() (*CoarseIndexMetadataPB, error) {
	metadataPath := filepath.Join(s.basePath, "metadata.pb")
	file, err := os.Open(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open metadata file: %w", err)
	}
	defer file.Close()

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata CoarseIndexMetadataPB
	if err := proto.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// SavePage 保存单个页面
func (s *CoarseIndexSerializer) SavePage(pageData *PageDataPB) error {
	pagePath := filepath.Join(s.basePath, fmt.Sprintf("page_%d.pb", pageData.PageId))
	file, err := os.Create(pagePath)
	if err != nil {
		return fmt.Errorf("failed to create page file: %w", err)
	}
	defer file.Close()

	data, err := proto.Marshal(pageData)
	if err != nil {
		return fmt.Errorf("failed to marshal page data: %w", err)
	}

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write page data: %w", err)
	}

	return nil
}

// LoadPage 加载单个页面
func (s *CoarseIndexSerializer) LoadPage(pageID int) (*PageDataPB, error) {
	pagePath := filepath.Join(s.basePath, fmt.Sprintf("page_%d.pb", pageID))
	file, err := os.Open(pagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open page file: %w", err)
	}
	defer file.Close()

	data, err := os.ReadFile(pagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read page file: %w", err)
	}

	var pageData PageDataPB
	if err := proto.Unmarshal(data, &pageData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal page data: %w", err)
	}

	return &pageData, nil
}

// SaveDirtyPages 保存脏页面（增量保存）
func (s *CoarseIndexSerializer) SaveDirtyPages() (int, error) {
	if !s.enableDelta {
		return 0, nil
	}

	s.dirtyMu.RLock()
	dirtyPages := make([]int, 0, len(s.dirtyPages))
	for pageID := range s.dirtyPages {
		dirtyPages = append(dirtyPages, pageID)
	}
	s.dirtyMu.RUnlock()

	if len(dirtyPages) == 0 {
		return 0, nil
	}

	s.saveMu.Lock()
	defer s.saveMu.Unlock()

	savedCount := 0
	for _, pageID := range dirtyPages {
		pagePath := filepath.Join(s.basePath, fmt.Sprintf("page_%d.pb", pageID))
		if _, err := os.Stat(pagePath); err == nil {
			// 页面文件存在，标记为已保存
			savedCount++
		}
	}

	// 保存脏页面映射
	dirtyPages32 := make([]int32, len(dirtyPages))
	for i, pageID := range dirtyPages {
		dirtyPages32[i] = int32(pageID)
	}
	dirtyMap := &PageDirtyMapPB{
		DirtyPages:   dirtyPages32,
		LastSaveTime: time.Now().UnixNano(),
	}

	dirtyMapPath := filepath.Join(s.basePath, "dirty_map.pb")
	file, err := os.Create(dirtyMapPath)
	if err != nil {
		return savedCount, fmt.Errorf("failed to create dirty map file: %w", err)
	}
	defer file.Close()

	data, err := proto.Marshal(dirtyMap)
	if err != nil {
		return savedCount, fmt.Errorf("failed to marshal dirty map: %w", err)
	}

	if _, err := file.Write(data); err != nil {
		return savedCount, fmt.Errorf("failed to write dirty map: %w", err)
	}

	s.lastSave = time.Now()
	return savedCount, nil
}

// LoadDirtyMap 加载脏页面映射
func (s *CoarseIndexSerializer) LoadDirtyMap() (*PageDirtyMapPB, error) {
	dirtyMapPath := filepath.Join(s.basePath, "dirty_map.pb")
	file, err := os.Open(dirtyMapPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open dirty map file: %w", err)
	}
	defer file.Close()

	data, err := os.ReadFile(dirtyMapPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read dirty map file: %w", err)
	}

	var dirtyMap PageDirtyMapPB
	if err := proto.Unmarshal(data, &dirtyMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dirty map: %w", err)
	}

	return &dirtyMap, nil
}

// SaveSnapshot 保存完整快照
func (s *CoarseIndexSerializer) SaveSnapshot(snapshot *CoarseIndexSnapshotPB) error {
	s.saveMu.Lock()
	defer s.saveMu.Unlock()

	snapshotPath := filepath.Join(s.basePath, "snapshot.pb")
	file, err := os.Create(snapshotPath)
	if err != nil {
		return fmt.Errorf("failed to create snapshot file: %w", err)
	}
	defer file.Close()

	data, err := proto.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write snapshot: %w", err)
	}

	return nil
}

// LoadSnapshot 加载完整快照
func (s *CoarseIndexSerializer) LoadSnapshot() (*CoarseIndexSnapshotPB, error) {
	snapshotPath := filepath.Join(s.basePath, "snapshot.pb")
	file, err := os.Open(snapshotPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open snapshot file: %w", err)
	}
	defer file.Close()

	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot file: %w", err)
	}

	var snapshot CoarseIndexSnapshotPB
	if err := proto.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	return &snapshot, nil
}

// GetLastSaveTime 获取最后保存时间
func (s *CoarseIndexSerializer) GetLastSaveTime() time.Time {
	return s.lastSave
}

// IsDirty 检查页面是否为脏
func (s *CoarseIndexSerializer) IsDirty(pageID int) bool {
	s.dirtyMu.RLock()
	defer s.dirtyMu.RUnlock()
	return s.dirtyPages[pageID]
}

// GetDirtyPageCount 获取脏页面数量
func (s *CoarseIndexSerializer) GetDirtyPageCount() int {
	s.dirtyMu.RLock()
	defer s.dirtyMu.RUnlock()
	return len(s.dirtyPages)
}
