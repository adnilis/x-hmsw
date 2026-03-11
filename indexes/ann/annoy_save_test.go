package ann

import (
	"os"
	"testing"
)

func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func TestANNOYSaveLoadBasic(t *testing.T) {
	// 创建索引
	idx := NewANNOYIndex(4, 2, func(a, b []float32) float32 {
		sum := float32(0)
		for i := range a {
			diff := a[i] - b[i]
			sum += diff * diff
		}
		return sum
	})

	// 准备数据
	vectors := [][]float32{
		{1.0, 0.0, 0.0, 0.0},
		{0.0, 1.0, 0.0, 0.0},
		{0.0, 0.0, 1.0, 0.0},
		{0.0, 0.0, 0.0, 1.0},
	}
	ids := []int{0, 1, 2, 3}

	// 构建树
	if err := idx.Build(vectors, ids); err != nil {
		t.Fatalf("构建索引失败：%v", err)
	}

	// 搜索测试（保存前）
	query := []float32{1.0, 0.0, 0.0, 0.0}
	beforeIDs, _, err := idx.Search(query, 2)
	if err != nil {
		t.Fatalf("搜索失败：%v", err)
	}
	t.Logf("保存前搜索结果：%v", beforeIDs)

	// 保存
	tmpPath := "test_annoy_save.json"
	if err := idx.Save(tmpPath); err != nil {
		t.Fatalf("保存失败：%v", err)
	}
	t.Logf("保存成功，文件路径：%s", tmpPath)

	// 检查文件是否存在
	if _, err := os.Stat(tmpPath); err != nil {
		t.Fatalf("保存后文件不存在：%v", err)
	}
	t.Logf("文件存在，大小：%d bytes", getFileSize(tmpPath))
	// defer os.Remove(tmpPath) // 暂时不删除，用于调试

	// 创建新索引并加载
	idx2 := NewANNOYIndex(4, 2, func(a, b []float32) float32 {
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

	t.Logf("加载后索引有 %d 棵树", len(idx2.trees))
	if len(idx2.trees) > 0 {
		t.Logf("第一棵树有 %d 个向量", len(idx2.trees[0].Vectors))
		if idx2.trees[0].Root != nil {
			t.Logf("树根节点是叶子节点：%v", idx2.trees[0].Root.IsLeaf)
			t.Logf("树根节点 ID: %d", idx2.trees[0].Root.ID)
			t.Logf("树根节点 Vector: %v", idx2.trees[0].Root.Vector)
			if idx2.trees[0].Root.Vector == nil {
				t.Logf("警告：树根节点 Vector 为 nil!")
			}
		} else {
			t.Logf("树根节点为 nil")
		}
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

	t.Logf("成功保存和加载 ANNOY 索引，包含 %d 棵树", len(idx2.trees))
}

func TestANNOYSaveLoadEmpty(t *testing.T) {
	idx := NewANNOYIndex(4, 2, nil)

	tmpPath := "test_annoy_empty.json"
	if err := idx.Save(tmpPath); err != nil {
		t.Fatalf("保存空索引失败：%v", err)
	}
	defer os.Remove(tmpPath)

	idx2 := NewANNOYIndex(4, 2, nil)
	if err := idx2.Load(tmpPath); err != nil {
		t.Fatalf("加载空索引失败：%v", err)
	}

	t.Log("成功保存和加载空 ANNOY 索引")
}

func TestANNOYSaveLoadNonExistent(t *testing.T) {
	idx := NewANNOYIndex(4, 2, nil)

	err := idx.Load("non_existent_file.json")
	if err == nil {
		t.Error("加载不存在的文件应该返回错误")
	}

	t.Logf("正确返回错误：%v", err)
}
