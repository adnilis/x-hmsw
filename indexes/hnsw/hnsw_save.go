package hnsw

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// HNSWSnapshot 用于序列化的 HNSW 快照
type HNSWSnapshot struct {
	MaxLevel       int            `json:"max_level"`
	M              int            `json:"m"`
	M0             int            `json:"m0"`
	EfConstruction int            `json:"ef_construction"`
	EfSearch       int            `json:"ef_search"`
	MaxNodes       int            `json:"max_nodes"`
	Nodes          []NodeSnapshot `json:"nodes"`
}

// NodeSnapshot 用于序列化的节点快照
type NodeSnapshot struct {
	ID      int           `json:"id"`
	Vector  []float32     `json:"vector"`
	Level   int           `json:"level"`
	Friends []FriendLayer `json:"friends"`
}

// FriendLayer 某一层的朋友关系
type FriendLayer struct {
	Level int   `json:"level"`
	IDs   []int `json:"ids"`
}

// Save 保存 HNSW 索引
func (h *HNSWGraph) Save(path string) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 创建目录
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建快照
	snapshot := HNSWSnapshot{
		MaxLevel:       h.MaxLevel,
		M:              h.M,
		M0:             h.M0,
		EfConstruction: h.EfConstruction,
		EfSearch:       h.EfSearch,
		MaxNodes:       h.MaxNodes,
		Nodes:          make([]NodeSnapshot, len(h.Nodes)),
	}

	// 序列化所有节点
	for i, node := range h.Nodes {
		snapshot.Nodes[i] = NodeSnapshot{
			ID:      node.ID,
			Vector:  node.Vector,
			Level:   node.Level,
			Friends: make([]FriendLayer, len(node.Friends)),
		}

		// 序列化每层的朋友关系
		for level, friends := range node.Friends {
			ids := make([]int, 0, len(friends))
			for friendID := range friends {
				ids = append(ids, friendID)
			}
			snapshot.Nodes[i].Friends[level] = FriendLayer{
				Level: level,
				IDs:   ids,
			}
		}
	}

	// 序列化
	jsonData, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal HNSW snapshot: %w", err)
	}

	// 写入文件
	savePath := filepath.Join(path, "hnsw_index.json")
	if err := ioutil.WriteFile(savePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Load 加载 HNSW 索引
func (h *HNSWGraph) Load(path string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 读取文件
	loadPath := filepath.Join(path, "hnsw_index.json")
	jsonData, err := ioutil.ReadFile(loadPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", loadPath)
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	// 反序列化
	var snapshot HNSWSnapshot
	if err := json.Unmarshal(jsonData, &snapshot); err != nil {
		return fmt.Errorf("failed to unmarshal HNSW snapshot: %w", err)
	}

	// 重建索引
	h.MaxLevel = snapshot.MaxLevel
	h.M = snapshot.M
	h.M0 = snapshot.M0
	h.EfConstruction = snapshot.EfConstruction
	h.EfSearch = snapshot.EfSearch
	h.MaxNodes = snapshot.MaxNodes

	// 重建节点
	h.Nodes = make([]*Node, len(snapshot.Nodes))
	nodeMap := make(map[int]*Node)

	// 第一步：创建所有节点
	for i, nodeSnap := range snapshot.Nodes {
		node := &Node{
			ID:       nodeSnap.ID,
			Vector:   nodeSnap.Vector,
			Level:    nodeSnap.Level,
			Friends:  make([][]*Node, len(nodeSnap.Friends)),
			MaxLinks: h.M,
		}
		h.Nodes[i] = node
		nodeMap[nodeSnap.ID] = node // 使用节点 ID 作为键
	}

	// 第二步：建立连接关系
	for i, nodeSnap := range snapshot.Nodes {
		node := h.Nodes[i]

		// 确保 Friends 数组足够大
		maxLevel := 0
		for _, fl := range nodeSnap.Friends {
			if fl.Level >= maxLevel {
				maxLevel = fl.Level
			}
		}

		// 初始化所有层
		node.Friends = make([][]*Node, maxLevel+1)

		// 建立朋友连接
		for _, friendLayer := range nodeSnap.Friends {
			level := friendLayer.Level
			if level >= len(node.Friends) {
				continue
			}

			node.Friends[level] = make([]*Node, 0, len(friendLayer.IDs))
			for _, friendID := range friendLayer.IDs {
				if friend, exists := nodeMap[friendID]; exists {
					node.Friends[level] = append(node.Friends[level], friend)
				}
			}
		}
	}

	// 重新设置入口点为第一个节点（最简单的方式）
	if len(h.Nodes) > 0 {
		h.EntryPoint = h.Nodes[0]
		// 找到最高层的节点作为入口点
		for _, node := range h.Nodes {
			if node.Level > h.MaxLevel {
				h.MaxLevel = node.Level
				h.EntryPoint = node
			}
		}
	}

	return nil
}

// SaveJSON 保存为 JSON 格式（用于调试）
func (h *HNSWGraph) SaveJSON(path string) error {
	return h.Save(path)
}

// LoadJSON 从 JSON 格式加载（用于调试）
func (h *HNSWGraph) LoadJSON(path string) error {
	return h.Load(path)
}
