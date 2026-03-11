package mmap

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestMMapStorage_WriteAndRead(t *testing.T) {
	// 创建临时文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mmap")

	// 创建存储
	storage, err := NewMMapStorage(testFile)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// 写入向量
	id := "test-1"
	vector := []float32{1.0, 2.0, 3.0, 4.0, 5.0}
	err = storage.Write(id, vector)
	if err != nil {
		t.Fatalf("Failed to write vector: %v", err)
	}

	// 读取向量
	readVector, err := storage.Read(id)
	if err != nil {
		t.Fatalf("Failed to read vector: %v", err)
	}

	// 验证向量
	if len(readVector) != len(vector) {
		t.Errorf("Vector length mismatch: got %d, want %d", len(readVector), len(vector))
	}

	for i := range vector {
		if readVector[i] != vector[i] {
			t.Errorf("Vector value mismatch at index %d: got %f, want %f", i, readVector[i], vector[i])
		}
	}
}

func TestMMapStorage_PutAndGet(t *testing.T) {
	// 创建临时文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mmap")

	// 创建存储
	storage, err := NewMMapStorage(testFile)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// 写入向量
	vec := &Vector{
		ID:      "test-2",
		Vector:  []float32{0.1, 0.2, 0.3},
		Payload: map[string]interface{}{"key": "value"},
	}
	err = storage.Put(vec)
	if err != nil {
		t.Fatalf("Failed to put vector: %v", err)
	}

	// 获取向量
	readVec, err := storage.Get(vec.ID)
	if err != nil {
		t.Fatalf("Failed to get vector: %v", err)
	}

	// 验证向量
	if readVec.ID != vec.ID {
		t.Errorf("ID mismatch: got %s, want %s", readVec.ID, vec.ID)
	}

	if len(readVec.Vector) != len(vec.Vector) {
		t.Errorf("Vector length mismatch: got %d, want %d", len(readVec.Vector), len(vec.Vector))
	}
}

func TestMMapStorage_BatchOperations(t *testing.T) {
	// 创建临时文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mmap")

	// 创建存储
	storage, err := NewMMapStorage(testFile)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// 批量写入
	vectors := []*Vector{
		{ID: "batch-1", Vector: []float32{1.0, 2.0}},
		{ID: "batch-2", Vector: []float32{3.0, 4.0}},
		{ID: "batch-3", Vector: []float32{5.0, 6.0}},
	}
	err = storage.BatchPut(vectors)
	if err != nil {
		t.Fatalf("Failed to batch put vectors: %v", err)
	}

	// 批量获取
	ids := []string{"batch-1", "batch-2", "batch-3"}
	readVectors, err := storage.BatchGet(ids)
	if err != nil {
		t.Fatalf("Failed to batch get vectors: %v", err)
	}

	if len(readVectors) != len(vectors) {
		t.Errorf("Batch get count mismatch: got %d, want %d", len(readVectors), len(vectors))
	}

	// 批量删除
	err = storage.BatchDelete([]string{"batch-1", "batch-2"})
	if err != nil {
		t.Fatalf("Failed to batch delete vectors: %v", err)
	}

	// 验证删除
	_, err = storage.Get("batch-1")
	if err == nil {
		t.Error("Expected error when getting deleted vector")
	}
}

func TestMMapStorage_Count(t *testing.T) {
	// 创建临时文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mmap")

	// 创建存储
	storage, err := NewMMapStorage(testFile)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// 初始计数
	count, err := storage.Count()
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}
	if count != 0 {
		t.Errorf("Initial count should be 0, got %d", count)
	}

	// 写入向量
	storage.Write("test-1", []float32{1.0, 2.0})
	storage.Write("test-2", []float32{3.0, 4.0})

	// 检查计数
	count, err = storage.Count()
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}
	if count != 2 {
		t.Errorf("Count should be 2, got %d", count)
	}
}

func TestMMapStorage_Delete(t *testing.T) {
	// 创建临时文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mmap")

	// 创建存储
	storage, err := NewMMapStorage(testFile)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// 写入向量
	id := "test-delete"
	storage.Write(id, []float32{1.0, 2.0})

	// 删除向量
	err = storage.Delete(id)
	if err != nil {
		t.Fatalf("Failed to delete vector: %v", err)
	}

	// 验证删除
	_, err = storage.Read(id)
	if err == nil {
		t.Error("Expected error when reading deleted vector")
	}
}

func TestMMapStorage_Iterate(t *testing.T) {
	// 创建临时文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mmap")

	// 创建存储
	storage, err := NewMMapStorage(testFile)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// 写入多个向量
	vectors := []*Vector{
		{ID: "iter-1", Vector: []float32{1.0}},
		{ID: "iter-2", Vector: []float32{2.0}},
		{ID: "iter-3", Vector: []float32{3.0}},
	}
	storage.BatchPut(vectors)

	// 遍历
	count := 0
	err = storage.Iterate(func(vec *Vector) bool {
		count++
		return true
	})
	if err != nil {
		t.Fatalf("Failed to iterate: %v", err)
	}

	if count != len(vectors) {
		t.Errorf("Iterate count mismatch: got %d, want %d", count, len(vectors))
	}
}

func TestMMapStorage_Reopen(t *testing.T) {
	// 创建临时文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mmap")

	// 创建存储并写入数据
	storage1, err := NewMMapStorage(testFile)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	id := "reopen-test"
	vector := []float32{7.0, 8.0, 9.0}
	err = storage1.Write(id, vector)
	if err != nil {
		storage1.Close()
		t.Fatalf("Failed to write vector: %v", err)
	}
	storage1.Close()

	// 重新打开存储
	storage2, err := NewMMapStorage(testFile)
	if err != nil {
		t.Fatalf("Failed to reopen storage: %v", err)
	}
	defer storage2.Close()

	// 读取数据
	readVector, err := storage2.Read(id)
	if err != nil {
		t.Fatalf("Failed to read vector after reopen: %v", err)
	}

	// 验证数据
	if len(readVector) != len(vector) {
		t.Errorf("Vector length mismatch after reopen: got %d, want %d", len(readVector), len(vector))
	}

	for i := range vector {
		if readVector[i] != vector[i] {
			t.Errorf("Vector value mismatch at index %d after reopen: got %f, want %f", i, readVector[i], vector[i])
		}
	}
}

func TestMMapStorage_GrowFile(t *testing.T) {
	// 创建临时文件
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mmap")

	// 创建存储
	storage, err := NewMMapStorage(testFile)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// 写入适量数据以触发文件扩展
	for i := 0; i < 100; i++ {
		id := fmt.Sprintf("vec-%05d", i)
		vector := make([]float32, 10)
		for j := range vector {
			vector[j] = float32(i*10 + j)
		}
		err = storage.Write(id, vector)
		if err != nil {
			t.Fatalf("Failed to write vector %d: %v", i, err)
		}
	}

	// 验证数据
	count, err := storage.Count()
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}
	if count != 100 {
		t.Errorf("Count should be 100, got %d", count)
	}
}
