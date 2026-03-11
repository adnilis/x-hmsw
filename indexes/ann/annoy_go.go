package ann

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// ANNOYIndex Annoy 索引实现
type ANNOYIndex struct {
	dimension    int
	numTrees     int
	trees        []*Tree
	distanceFunc func(a, b []float32) float32
	rng          *rand.Rand
	mu           sync.RWMutex
}

// Tree Annoy 树
type Tree struct {
	Root    *Node
	Vectors [][]float32
	IDs     []int
}

// Node Annoy 节点
type Node struct {
	ID         int
	Left       *Node
	Right      *Node
	Vector     []float32
	IsLeaf     bool
	SplitAxis  int
	SplitValue float32
}

// NewANNOYIndex 创建 ANNOY 索引
func NewANNOYIndex(dimension, numTrees int, distanceFunc func(a, b []float32) float32) *ANNOYIndex {
	return &ANNOYIndex{
		dimension:    dimension,
		numTrees:     numTrees,
		trees:        make([]*Tree, 0),
		distanceFunc: distanceFunc,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Add 添加向量
func (idx *ANNOYIndex) Add(id int, vector []float32) {
	// Annoy 需要批量构建树，这里先缓存
}

// Build 构建树
func (idx *ANNOYIndex) Build(vectors [][]float32, ids []int) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	for i := 0; i < idx.numTrees; i++ {
		tree := idx.buildTree(vectors, ids)
		idx.trees = append(idx.trees, tree)
	}

	return nil
}

// buildTree 构建单棵树
func (idx *ANNOYIndex) buildTree(vectors [][]float32, ids []int) *Tree {
	if len(vectors) == 0 {
		return &Tree{}
	}

	// 随机选择两个向量作为初始分割
	root := idx.split(vectors, ids, 0)
	return &Tree{
		Root:    root,
		Vectors: vectors,
		IDs:     ids,
	}
}

// split 递归分割
func (idx *ANNOYIndex) split(vectors [][]float32, ids []int, depth int) *Node {
	if len(vectors) <= 10 {
		// 叶子节点
		return &Node{
			ID:     ids[0],
			IsLeaf: true,
			Vector: avg(vectors),
		}
	}

	// 随机选择轴
	axis := idx.rng.Intn(idx.dimension)

	// 按轴排序
	sort.Slice(vectors, func(i, j int) bool {
		return vectors[i][axis] < vectors[j][axis]
	})

	// 分割
	mid := len(vectors) / 2
	splitValue := (vectors[mid][axis] + vectors[mid-1][axis]) / 2

	leftVectors := vectors[:mid]
	rightVectors := vectors[mid:]

	leftIDs := ids[:mid]
	rightIDs := ids[mid:]

	return &Node{
		SplitAxis:  axis,
		SplitValue: splitValue,
		Left:       idx.split(leftVectors, leftIDs, depth+1),
		Right:      idx.split(rightVectors, rightIDs, depth+1),
	}
}

// avg 计算平均向量
func avg(vectors [][]float32) []float32 {
	if len(vectors) == 0 {
		return nil
	}
	result := make([]float32, len(vectors[0]))
	for _, vec := range vectors {
		for i, v := range vec {
			result[i] += v
		}
	}
	for i := range result {
		result[i] /= float32(len(vectors))
	}
	return result
}

// Search 搜索
func (idx *ANNOYIndex) Search(query []float32, k int) ([]int, []float32, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if len(idx.trees) == 0 {
		return nil, nil, fmt.Errorf("index not built")
	}

	// 在每棵树中搜索
	candidates := make(map[int]float32)
	for _, tree := range idx.trees {
		idx.searchTree(tree, query, candidates)
	}

	// 排序
	type result struct {
		id   int
		dist float32
	}

	results := make([]result, 0, len(candidates))
	for id, dist := range candidates {
		results = append(results, result{id, dist})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].dist < results[j].dist
	})

	// 返回前 k 个
	resultSize := k
	if resultSize > len(results) {
		resultSize = len(results)
	}

	ids := make([]int, resultSize)
	dists := make([]float32, resultSize)

	for i := 0; i < resultSize; i++ {
		ids[i] = results[i].id
		dists[i] = results[i].dist
	}

	return ids, dists, nil
}

// searchTree 在树中搜索
func (idx *ANNOYIndex) searchTree(tree *Tree, query []float32, candidates map[int]float32) {
	if tree.Root == nil {
		return
	}

	stack := []*Node{tree.Root}
	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if node.IsLeaf {
			// 添加到候选
			candidates[node.ID] = idx.distanceFunc(query, node.Vector)
			continue
		}

		// 根据分割轴决定走哪边
		if query[node.SplitAxis] < node.SplitValue {
			if node.Left != nil {
				stack = append(stack, node.Left)
			}
			if node.Right != nil {
				stack = append(stack, node.Right)
			}
		} else {
			if node.Right != nil {
				stack = append(stack, node.Right)
			}
			if node.Left != nil {
				stack = append(stack, node.Left)
			}
		}
	}
}

// ANNOYSnapshot 用于序列化的快照结构体
type ANNOYSnapshot struct {
	Dimension int            `json:"dimension"`
	NumTrees  int            `json:"num_trees"`
	Trees     []TreeSnapshot `json:"trees"`
}

// TreeSnapshot 树的快照
type TreeSnapshot struct {
	Vectors [][]float32  `json:"vectors"`
	IDs     []int        `json:"ids"`
	Root    NodeSnapshot `json:"root"`
}

// NodeSnapshot 节点快照
type NodeSnapshot struct {
	ID         int           `json:"id"`
	Left       *NodeSnapshot `json:"left,omitempty"`
	Right      *NodeSnapshot `json:"right,omitempty"`
	Vector     []float32     `json:"vector,omitempty"`
	IsLeaf     bool          `json:"is_leaf"`
	SplitAxis  int           `json:"split_axis,omitempty"`
	SplitValue float32       `json:"split_value,omitempty"`
}

// Save 保存索引
func (idx *ANNOYIndex) Save(path string) error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// 创建快照
	snapshot := ANNOYSnapshot{
		Dimension: idx.dimension,
		NumTrees:  idx.numTrees,
		Trees:     make([]TreeSnapshot, len(idx.trees)),
	}

	// 序列化每棵树
	for i, tree := range idx.trees {
		snapshot.Trees[i] = idx.treeToSnapshot(tree)
	}

	// JSON 序列化
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal ANNOY index: %w", err)
	}

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write ANNOY index: %w", err)
	}

	return nil
}

// treeToSnapshot 将树转换为快照
func (idx *ANNOYIndex) treeToSnapshot(tree *Tree) TreeSnapshot {
	return TreeSnapshot{
		Vectors: tree.Vectors,
		IDs:     tree.IDs,
		Root:    *idx.nodeToSnapshotPtr(tree.Root),
	}
}

// nodeToSnapshot 将节点转换为快照
func (idx *ANNOYIndex) nodeToSnapshot(node *Node) NodeSnapshot {
	if node == nil {
		return NodeSnapshot{}
	}

	snapshot := NodeSnapshot{
		ID:         node.ID,
		IsLeaf:     node.IsLeaf,
		SplitAxis:  node.SplitAxis,
		SplitValue: node.SplitValue,
		Vector:     node.Vector,
	}

	if node.Left != nil {
		left := idx.nodeToSnapshot(node.Left)
		snapshot.Left = &left
	}

	if node.Right != nil {
		right := idx.nodeToSnapshot(node.Right)
		snapshot.Right = &right
	}

	return snapshot
}

// nodeToSnapshotPtr 将节点转换为快照指针
func (idx *ANNOYIndex) nodeToSnapshotPtr(node *Node) *NodeSnapshot {
	if node == nil {
		return nil
	}
	snapshot := idx.nodeToSnapshot(node)
	return &snapshot
}

// snapshotToNodePtr 将快照指针转换为节点
func (idx *ANNOYIndex) snapshotToNodePtr(snapshot *NodeSnapshot) *Node {
	if snapshot == nil {
		return nil
	}
	return idx.snapshotToNode(*snapshot)
}

// Load 加载索引
func (idx *ANNOYIndex) Load(path string) error {
	// 读取文件
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read ANNOY index: %w", err)
	}

	// JSON 反序列化
	var snapshot ANNOYSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("failed to unmarshal ANNOY index: %w", err)
	}

	// 重建索引
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.dimension = snapshot.Dimension
	idx.numTrees = snapshot.NumTrees
	idx.trees = make([]*Tree, len(snapshot.Trees))

	// 重建树
	for i, treeSnap := range snapshot.Trees {
		idx.trees[i] = idx.snapshotToTree(treeSnap)
	}

	return nil
}

// snapshotToTree 将快照转换为树
func (idx *ANNOYIndex) snapshotToTree(treeSnap TreeSnapshot) *Tree {
	return &Tree{
		Vectors: treeSnap.Vectors,
		IDs:     treeSnap.IDs,
		Root:    idx.snapshotToNodePtr(&treeSnap.Root),
	}
}

// snapshotToNode 将快照转换为节点
func (idx *ANNOYIndex) snapshotToNode(snapshot NodeSnapshot) *Node {
	node := &Node{
		ID:         snapshot.ID,
		IsLeaf:     snapshot.IsLeaf,
		SplitAxis:  snapshot.SplitAxis,
		SplitValue: snapshot.SplitValue,
		Vector:     snapshot.Vector,
	}

	if snapshot.Left != nil {
		node.Left = idx.snapshotToNode(*snapshot.Left)
	}

	if snapshot.Right != nil {
		node.Right = idx.snapshotToNode(*snapshot.Right)
	}

	return node
}
