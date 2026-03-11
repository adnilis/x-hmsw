package ivf

import (
	"os"
	"testing"

	"github.com/adnilis/x-hmsw/utils/math"
)

func TestIVFSaveLoadBasic(t *testing.T) {
	tempDir := "./test_save_load"
	defer os.RemoveAll(tempDir)

	// 创建 IVF 索引
	dimension := 4
	numClusters := 2
	index := NewIVFIndex(dimension, numClusters, math.CosineDistance)

	// 插入测试向量
	vectors := [][]float32{
		{1.0, 0.0, 0.0, 0.0},
		{0.0, 1.0, 0.0, 0.0},
		{0.0, 0.0, 1.0, 0.0},
		{0.0, 0.0, 0.0, 1.0},
	}

	// 训练索引
	if err := index.Train(vectors, 10); err != nil {
		t.Fatalf("Failed to train: %v", err)
	}

	// 保存
	if err := index.Save(tempDir); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// 创建新索引并加载
	newIndex := NewIVFIndex(dimension, numClusters, math.CosineDistance)
	if err := newIndex.Load(tempDir); err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// 验证向量数量
	if len(newIndex.vectors) != len(vectors) {
		t.Errorf("Expected %d vectors, got %d", len(vectors), len(newIndex.vectors))
	}

	// 验证聚类数量
	if len(newIndex.clusters) != numClusters {
		t.Errorf("Expected %d clusters, got %d", numClusters, len(newIndex.clusters))
	}

	// 验证搜索功能正常
	query := []float32{1.0, 0.0, 0.0, 0.0}
	ids, dists, err := newIndex.Search(query, 2, 1)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(ids) == 0 {
		t.Error("Expected search results")
	}

	t.Logf("Successfully saved and loaded IVF index with %d vectors and %d clusters", len(newIndex.vectors), len(newIndex.clusters))
	t.Logf("Search results: ids=%v, distances=%v", ids, dists)
}

func TestIVFSaveLoadEmpty(t *testing.T) {
	tempDir := "./test_save_load_empty"
	defer os.RemoveAll(tempDir)

	dimension := 4
	numClusters := 2
	index := NewIVFIndex(dimension, numClusters, math.CosineDistance)

	// 保存空索引
	if err := index.Save(tempDir); err != nil {
		t.Fatalf("Failed to save empty index: %v", err)
	}

	// 加载空索引
	newIndex := NewIVFIndex(dimension, numClusters, math.CosineDistance)
	if err := newIndex.Load(tempDir); err != nil {
		t.Fatalf("Failed to load empty index: %v", err)
	}

	if len(newIndex.vectors) != 0 {
		t.Errorf("Expected 0 vectors, got %d", len(newIndex.vectors))
	}
}

func TestIVFSaveLoadNonExistent(t *testing.T) {
	dimension := 4
	numClusters := 2
	index := NewIVFIndex(dimension, numClusters, math.CosineDistance)

	// 尝试加载不存在的文件
	err := index.Load("./non_existent_path")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
