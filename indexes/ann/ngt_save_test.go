package ann

import (
	"os"
	"testing"
)

func TestNGTSaveLoadBasic(t *testing.T) {
	// 创建索引
	idx := NewNGTIndex(4, 10, 20, func(a, b []float32) float32 {
		sum := float32(0)
		for i := range a {
			diff := a[i] - b[i]
			sum += diff * diff
		}
		return sum
	})

	// 插入向量
	vectors := [][]float32{
		{1.0, 0.0, 0.0, 0.0},
		{0.0, 1.0, 0.0, 0.0},
		{0.0, 0.0, 1.0, 0.0},
		{0.0, 0.0, 0.0, 1.0},
	}

	for i, vec := range vectors {
		if err := idx.Add(i, vec); err != nil {
			t.Fatalf("插入向量失败：%v", err)
		}
	}

	// 搜索测试（保存前）
	query := []float32{1.0, 0.0, 0.0, 0.0}
	beforeIDs, _, err := idx.Search(query, 2)
	if err != nil {
		t.Fatalf("搜索失败：%v", err)
	}
	t.Logf("保存前搜索结果：%v", beforeIDs)

	// 保存
	tmpPath := "test_ngt_save.json"
	if err := idx.Save(tmpPath); err != nil {
		t.Fatalf("保存失败：%v", err)
	}
	defer os.Remove(tmpPath)

	// 创建新索引并加载
	idx2 := NewNGTIndex(4, 10, 20, func(a, b []float32) float32 {
		sum := float32(0)
		for i := range a {
			diff := a[i] - b[i]
			sum += diff * diff
		}
		return sum
	})
	if err := idx2.Load(tmpPath); err != nil {
		t.Fatalf("加载失败：%v", err)
	}

	// 搜索测试（加载后）
	afterIDs, _, err := idx2.Search(query, 2)
	if err != nil {
		t.Fatalf("加载后搜索失败：%v", err)
	}
	t.Logf("加载后搜索结果：%v", afterIDs)

	// 验证结果一致
	if len(beforeIDs) != len(afterIDs) {
		t.Errorf("结果数量不一致：保存前=%d, 加载后=%d", len(beforeIDs), len(afterIDs))
	}

	t.Logf("成功保存和加载 NGT 索引，包含 %d 个节点", len(idx2.graph))
}

func TestNGTSaveLoadEmpty(t *testing.T) {
	idx := NewNGTIndex(4, 10, 20, nil)

	tmpPath := "test_ngt_empty.json"
	if err := idx.Save(tmpPath); err != nil {
		t.Fatalf("保存空索引失败：%v", err)
	}
	defer os.Remove(tmpPath)

	idx2 := NewNGTIndex(4, 10, 20, nil)
	if err := idx2.Load(tmpPath); err != nil {
		t.Fatalf("加载空索引失败：%v", err)
	}

	t.Log("成功保存和加载空 NGT 索引")
}

func TestNGTSaveLoadNonExistent(t *testing.T) {
	idx := NewNGTIndex(4, 10, 20, nil)

	err := idx.Load("non_existent_file.json")
	if err == nil {
		t.Error("加载不存在的文件应该返回错误")
	}

	t.Logf("正确返回错误：%v", err)
}
