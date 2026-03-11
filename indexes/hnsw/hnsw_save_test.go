package hnsw

import (
	"os"
	"testing"

	"github.com/adnilis/x-hmsw/utils/math"
)

func TestHNSWSaveLoadBasic(t *testing.T) {
	tempDir := "./test_save_load"
	defer os.RemoveAll(tempDir)

	// 创建 HNSW 索引
	dimension := 4
	index := NewHNSW(dimension, 16, 200, 1000, math.CosineDistance)

	// 插入测试向量
	vectors := [][]float32{
		{1.0, 0.0, 0.0, 0.0},
		{0.0, 1.0, 0.0, 0.0},
		{0.0, 0.0, 1.0, 0.0},
		{0.0, 0.0, 0.0, 1.0},
	}

	for i, vec := range vectors {
		index.Insert(i, vec)
	}

	// 保存
	if err := index.Save(tempDir); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// 创建新索引并加载
	newIndex := NewHNSW(dimension, 16, 200, 1000, math.CosineDistance)
	if err := newIndex.Load(tempDir); err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// 验证节点数量
	if len(newIndex.Nodes) != len(vectors) {
		t.Errorf("Expected %d nodes, got %d", len(vectors), len(newIndex.Nodes))
	}

	// 验证搜索功能正常
	query := []float32{1.0, 0.0, 0.0, 0.0}
	results := newIndex.Search(query, 2)
	if len(results) == 0 {
		t.Error("Expected search results")
	}

	t.Logf("Successfully saved and loaded HNSW index with %d nodes", len(newIndex.Nodes))
}

func TestHNSWSaveLoadEmpty(t *testing.T) {
	tempDir := "./test_save_load_empty"
	defer os.RemoveAll(tempDir)

	dimension := 4
	index := NewHNSW(dimension, 16, 200, 1000, math.CosineDistance)

	// 保存空索引
	if err := index.Save(tempDir); err != nil {
		t.Fatalf("Failed to save empty index: %v", err)
	}

	// 加载空索引
	newIndex := NewHNSW(dimension, 16, 200, 1000, math.CosineDistance)
	if err := newIndex.Load(tempDir); err != nil {
		t.Fatalf("Failed to load empty index: %v", err)
	}

	if len(newIndex.Nodes) != 0 {
		t.Errorf("Expected 0 nodes, got %d", len(newIndex.Nodes))
	}
}

func TestHNSWSaveLoadNonExistent(t *testing.T) {
	dimension := 4
	index := NewHNSW(dimension, 16, 200, 1000, math.CosineDistance)

	// 尝试加载不存在的文件
	err := index.Load("./non_existent_path")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
