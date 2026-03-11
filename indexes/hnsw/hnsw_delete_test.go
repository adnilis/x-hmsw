package hnsw

import (
	"testing"

	"github.com/adnilis/x-hmsw/utils/math"
)

func TestHNSWDelete(t *testing.T) {
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

	// 验证初始数量
	if len(index.Nodes) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(index.Nodes))
	}

	// 搜索删除前
	query := []float32{1.0, 0.0, 0.0, 0.0}
	resultsBefore := index.Search(query, 4)
	t.Logf("Before delete: Found %d results", len(resultsBefore))
	for _, r := range resultsBefore {
		t.Logf("  ID: %d, Distance: %f, Deleted: %v", r.Node.ID, r.Distance, r.Node.Deleted)
	}

	// 删除节点 0
	if err := index.Delete(0); err != nil {
		t.Fatalf("Failed to delete node 0: %v", err)
	}

	// 验证节点标记为已删除
	if !index.NodeMap[0].Deleted {
		t.Error("Node 0 should be marked as deleted")
	}

	// 验证 DeletedIDs 包含该节点
	if !index.DeletedIDs[0] {
		t.Error("DeletedIDs should contain node 0")
	}

	// 搜索删除后
	resultsAfter := index.Search(query, 4)
	t.Logf("After delete: Found %d results", len(resultsAfter))
	for _, r := range resultsAfter {
		t.Logf("  ID: %d, Distance: %f, Deleted: %v", r.Node.ID, r.Distance, r.Node.Deleted)
	}

	// 验证搜索结果不包含已删除的节点
	for _, r := range resultsAfter {
		if r.Node.ID == 0 {
			t.Error("Search results should not include deleted node 0")
		}
	}

	// 验证数量
	if len(resultsAfter) != 3 {
		t.Errorf("Expected 3 results after delete, got %d", len(resultsAfter))
	}

	t.Logf("Successfully deleted node and verified search excludes it")
}

func TestHNSWDeleteNonExistent(t *testing.T) {
	dimension := 4
	index := NewHNSW(dimension, 16, 200, 1000, math.CosineDistance)

	// 插入一个向量
	index.Insert(1, []float32{1.0, 0.0, 0.0, 0.0})

	// 删除不存在的节点
	err := index.Delete(999)
	if err == nil {
		t.Error("Expected error when deleting non-existent node")
	}

	t.Logf("Correctly returned error for non-existent node: %v", err)
}

func TestHNSWDeleteTwice(t *testing.T) {
	dimension := 4
	index := NewHNSW(dimension, 16, 200, 1000, math.CosineDistance)

	// 插入一个向量
	index.Insert(1, []float32{1.0, 0.0, 0.0, 0.0})

	// 第一次删除
	if err := index.Delete(1); err != nil {
		t.Fatalf("Failed to delete node 1: %v", err)
	}

	// 第二次删除应该报错
	err := index.Delete(1)
	if err == nil {
		t.Error("Expected error when deleting already deleted node")
	}

	t.Logf("Correctly returned error for already deleted node: %v", err)
}
