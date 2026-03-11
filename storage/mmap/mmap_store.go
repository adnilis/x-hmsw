package mmap

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"reflect"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/adnilis/x-hmsw/types"
)

// Vector 向量数据结构（别名）
type Vector = types.Vector

// MMapStorage 内存映射存储
type MMapStorage struct {
	file   *os.File
	data   []byte
	index  map[string]int64
	mu     sync.RWMutex
	offset int64
	path   string
}

// NewMMapStorage 创建内存映射存储
func NewMMapStorage(path string) (*MMapStorage, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	// 获取文件大小
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	initialSize := int64(4096) // 默认大小

	// 如果文件为空，写入头部
	if stat.Size() == 0 {
		header := make([]byte, 4096)
		binary.LittleEndian.PutUint32(header[0:4], 0x4D4D4150) // "MMAP" magic
		binary.LittleEndian.PutUint64(header[4:12], 0)         // 记录数
		binary.LittleEndian.PutUint64(header[12:20], 4096)     // 数据起始偏移
		if _, err := file.Write(header); err != nil {
			file.Close()
			return nil, err
		}
		initialSize = 4096
	} else {
		initialSize = stat.Size()
	}

	// 内存映射 - 使用 syscall
	data, err := mmapFile(file, initialSize)
	if err != nil {
		file.Close()
		return nil, err
	}

	storage := &MMapStorage{
		file:   file,
		data:   data,
		index:  make(map[string]int64),
		offset: 4096, // 跳过头部
		path:   path,
	}

	// 重建索引
	if err := storage.rebuildIndex(); err != nil {
		file.Close()
		unmapFile(data)
		return nil, err
	}

	return storage, nil
}

// mmapFile 使用 syscall 进行内存映射
func mmapFile(file *os.File, size int64) ([]byte, error) {
	handle := syscall.Handle(file.Fd())

	// 创建内存映射文件
	// 在 Windows 上，需要指定映射的大小
	maxSizeHigh := uint32(size >> 32)
	maxSizeLow := uint32(size & 0xFFFFFFFF)
	mappedFile, err := syscall.CreateFileMapping(handle, nil, syscall.PAGE_READWRITE, maxSizeHigh, maxSizeLow, nil)
	if err != nil {
		return nil, err
	}

	// 映射视图
	addr, err := syscall.MapViewOfFile(mappedFile, syscall.FILE_MAP_WRITE|syscall.FILE_MAP_READ, 0, 0, uintptr(size))
	if err != nil {
		syscall.CloseHandle(mappedFile)
		return nil, err
	}

	// 映射成功后可以关闭映射句柄
	syscall.CloseHandle(mappedFile)

	// 转换为字节切片
	// 使用 reflect.SliceHeader 来创建切片
	var data []byte
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(&data))
	sliceHeader.Data = addr
	sliceHeader.Len = int(size)
	sliceHeader.Cap = int(size)

	return data, nil
}

// unmapFile 取消内存映射
func unmapFile(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	addr := uintptr(unsafe.Pointer(&data[0]))
	return syscall.UnmapViewOfFile(addr)
}

// Write 写入向量
func (s *MMapStorage) Write(id string, vector []float32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否需要扩展文件
	recordSize := 4 + len(id) + 4*len(vector) + 8 // id 长度 + id + 向量 + 时间戳
	if s.offset+int64(recordSize) > int64(len(s.data)) {
		if err := s.growFile(recordSize); err != nil {
			return err
		}
	}

	// 写入记录
	offset := s.offset

	// 写入 ID 长度
	binary.LittleEndian.PutUint32(s.data[offset:offset+4], uint32(len(id)))
	offset += 4

	// 写入 ID
	copy(s.data[offset:offset+int64(len(id))], id)
	offset += int64(len(id))

	// 写入向量长度
	binary.LittleEndian.PutUint32(s.data[offset:offset+4], uint32(len(vector)))
	offset += 4

	// 写入向量
	for i, v := range vector {
		binary.LittleEndian.PutUint32(s.data[offset+int64(i*4):offset+int64(i*4)+4], math.Float32bits(v))
	}
	offset += int64(len(vector) * 4)

	// 写入时间戳
	binary.LittleEndian.PutUint64(s.data[offset:offset+8], uint64(time.Now().UnixNano()))

	// 更新索引
	s.index[id] = s.offset

	// 更新偏移
	s.offset += int64(recordSize)

	// 更新头部计数
	count := binary.LittleEndian.Uint64(s.data[4:12])
	binary.LittleEndian.PutUint64(s.data[4:12], count+1)

	return nil
}

// Read 读取向量
func (s *MMapStorage) Read(id string) ([]float32, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	offset, exists := s.index[id]
	if !exists {
		return nil, fmt.Errorf("vector not found: %s", id)
	}

	// 读取 ID 长度
	idLen := binary.LittleEndian.Uint32(s.data[offset : offset+4])
	offset += 4

	// 跳过 ID
	offset += int64(idLen)

	// 读取向量长度
	vecLen := binary.LittleEndian.Uint32(s.data[offset : offset+4])
	offset += 4

	// 读取向量
	vector := make([]float32, vecLen)
	for i := 0; i < int(vecLen); i++ {
		bits := binary.LittleEndian.Uint32(s.data[offset+int64(i*4):])
		vector[i] = math.Float32frombits(bits)
	}

	return vector, nil
}

// growFile 扩展文件大小
func (s *MMapStorage) growFile(additionalSize int) error {
	// 计算新大小
	currentSize := len(s.data)
	newSize := currentSize + additionalSize + 4096 // 额外预留空间

	// 取消旧映射
	oldData := s.data
	if len(oldData) > 0 {
		if err := unmapFile(oldData); err != nil {
			return err
		}
	}
	s.data = nil

	// 关闭旧文件句柄
	if s.file != nil {
		if err := s.file.Close(); err != nil {
			return err
		}
	}

	// 截断/扩展文件
	file, err := os.OpenFile(s.path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	if err := file.Truncate(int64(newSize)); err != nil {
		file.Close()
		return err
	}

	// 重新映射
	s.file = file
	s.data, err = mmapFile(s.file, int64(newSize))
	if err != nil {
		return err
	}

	return nil
}

// rebuildIndex 重建索引
func (s *MMapStorage) rebuildIndex() error {
	s.index = make(map[string]int64)
	offset := int64(4096) // 跳过头部
	for offset < int64(len(s.data)) {
		recordStart := offset // 记录开始位置

		// 读取 ID 长度
		if offset+4 > int64(len(s.data)) {
			break
		}
		idLen := binary.LittleEndian.Uint32(s.data[offset : offset+4])
		if idLen == 0 || idLen > 1000 { // 合理的 ID 长度限制
			break
		}
		offset += 4
		// 读取 ID
		if offset+int64(idLen) > int64(len(s.data)) {
			break
		}
		id := string(s.data[offset : offset+int64(idLen)])
		offset += int64(idLen)
		// 读取向量长度
		if offset+4 > int64(len(s.data)) {
			break
		}
		vecLen := binary.LittleEndian.Uint32(s.data[offset : offset+4])
		if vecLen == 0 || vecLen > 100000 { // 合理的向量长度限制
			break
		}
		offset += 4
		// 跳过向量和时间戳
		offset += int64(vecLen)*4 + 8
		// 记录索引
		s.index[id] = recordStart
	}
	return nil
}

// Delete 删除向量
func (s *MMapStorage) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.index[id]
	if !exists {
		return fmt.Errorf("vector not found: %s", id)
	}

	// 简单实现：从索引中移除（文件空间不回收）
	delete(s.index, id)

	// 更新头部计数
	count := binary.LittleEndian.Uint64(s.data[4:12])
	if count > 0 {
		count--
		binary.LittleEndian.PutUint64(s.data[4:12], count)
	}

	return nil
}

// Close 关闭存储
func (s *MMapStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var err error

	if s.data != nil {
		if unmapErr := unmapFile(s.data); unmapErr != nil {
			err = unmapErr
		}
		s.data = nil
	}

	if s.file != nil {
		if closeErr := s.file.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			}
		}
		s.file = nil
	}

	return err
}

// Get 获取向量
func (s *MMapStorage) Get(id string) (*Vector, error) {
	vector, err := s.Read(id)
	if err != nil {
		return nil, err
	}
	return &Vector{
		ID:      id,
		Vector:  vector,
		Payload: nil,
	}, nil
}

// Put 写入向量
func (s *MMapStorage) Put(vec *Vector) error {
	return s.Write(vec.ID, vec.Vector)
}

// BatchGet 批量获取向量
func (s *MMapStorage) BatchGet(ids []string) ([]*Vector, error) {
	result := make([]*Vector, 0, len(ids))
	for _, id := range ids {
		vec, err := s.Get(id)
		if err != nil {
			continue // 跳过不存在的
		}
		result = append(result, vec)
	}
	return result, nil
}

// BatchPut 批量写入向量
func (s *MMapStorage) BatchPut(vectors []*Vector) error {
	for _, vec := range vectors {
		if err := s.Put(vec); err != nil {
			return err
		}
	}
	return nil
}

// BatchDelete 批量删除向量
func (s *MMapStorage) BatchDelete(ids []string) error {
	for _, id := range ids {
		if err := s.Delete(id); err != nil {
			return err
		}
	}
	return nil
}

// Iterate 遍历所有向量
func (s *MMapStorage) Iterate(fn func(*Vector) bool) error {
	for id := range s.index {
		vec, err := s.Get(id)
		if err != nil {
			continue
		}
		if !fn(vec) {
			break
		}
	}
	return nil
}

// Count 返回向量数量
func (s *MMapStorage) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.index), nil
}

// Save 保存到文件
func (s *MMapStorage) Save(path string) error {
	// MMap 已经是持久化的
	return nil
}

// Load 从文件加载
func (s *MMapStorage) Load(path string) error {
	// MMap 在打开时自动加载
	return nil
}
