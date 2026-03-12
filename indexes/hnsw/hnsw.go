package hnsw

import (
	"container/heap"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/adnilis/x-hmsw/utils/bitset"
	"github.com/adnilis/x-hmsw/utils/pool"
	"github.com/adnilis/x-hmsw/utils/simd"
)

// SearchResult HNSW 搜索结果
type SearchResult struct {
	Node     *Node   // 命中的节点
	Distance float32 // 距离值
}

// HNSW 图节点
type Node struct {
	ID       int
	Vector   []float32
	Level    int
	Friends  [][]*Node // 每层的邻居，使用 slice 替代 map 提高性能
	MaxLinks int
	Deleted  bool // 软删除标记
	mu       sync.RWMutex
}

// HNSW 图
type HNSWGraph struct {
	Nodes          []*Node
	MaxLevel       int
	LevelMult      float64
	EfConstruction int
	EfSearch       int
	M              int // 每层最大连接数
	M0             int // 底层最大连接数
	DistanceFunc   func(a, b []float32) float32
	EntryPoint     *Node
	MaxNodes       int
	rng            *rand.Rand
	mu             sync.RWMutex
	levelLocks     []sync.RWMutex
	NodeMap        map[int]*Node          // ID 到节点的映射，用于快速查找
	DeletedIDs     map[int]bool           // 已删除的 ID 集合
	visitedPool    *bitset.BitSetPool     // 复用 visited bitset 减少内存分配
	intSlicePool   *pool.IntSlicePool     // 复用 int 切片
	floatSlicePool *pool.Float32SlicePool // 复用 float32 切片
}

// 创建 HNSW 图
func NewHNSW(dim, M, efConstruction, maxNodes int, distanceFunc func(a, b []float32) float32) *HNSWGraph {
	if M <= 0 {
		M = 16
	}
	if efConstruction <= 0 {
		efConstruction = 200
	}
	levelMult := 1.0 / math.Log(float64(M))
	graph := &HNSWGraph{
		Nodes:          make([]*Node, 0, maxNodes),
		MaxLevel:       0,
		LevelMult:      levelMult,
		EfConstruction: efConstruction,
		EfSearch:       100, // 增加到 100 以确保找到足够的邻接节点
		M:              M,
		M0:             M * 2,
		DistanceFunc:   distanceFunc,
		EntryPoint:     nil,
		MaxNodes:       maxNodes,
		rng:            rand.New(rand.NewSource(time.Now().UnixNano())),
		levelLocks:     make([]sync.RWMutex, 32), // 假设最大 32 层
		NodeMap:        make(map[int]*Node),
		DeletedIDs:     make(map[int]bool),
		visitedPool:    bitset.NewBitSetPool(maxNodes),
		intSlicePool:   pool.NewIntSlicePool(M * 2),
		floatSlicePool: pool.NewFloat32SlicePool(M * 2),
	}

	return graph
}

// 插入节点
func (h *HNSWGraph) Insert(id int, vector []float32) *Node {
	// 预归一化向量（如果使用余弦距离）
	normalizedVector := h.normalizeVector(vector)

	// 检查节点是否已存在
	h.mu.Lock()
	if existingNode, exists := h.NodeMap[id]; exists {
		// 节点已存在，更新向量数据并取消删除标记
		existingNode.Vector = normalizedVector
		existingNode.Deleted = false
		delete(h.DeletedIDs, id)
		h.mu.Unlock()
		return existingNode
	}

	// 生成节点层级
	level := h.randomLevel()

	// 调试信息
	// fmt.Printf("Inserting node %d, level %d\n", id, level)

	// 创建节点
	node := &Node{
		ID:       id,
		Vector:   normalizedVector,
		Level:    level,
		Friends:  make([][]*Node, level+1),
		MaxLinks: h.M,
		Deleted:  false,
	}

	for i := 0; i <= level; i++ {
		if i == 0 {
			node.Friends[i] = make([]*Node, 0, h.M0)
		} else {
			node.Friends[i] = make([]*Node, 0, h.M)
		}
	}

	// 添加到节点列表和映射
	h.Nodes = append(h.Nodes, node)
	h.NodeMap[id] = node

	// 如果是第一个节点
	if h.EntryPoint == nil {
		h.EntryPoint = node
		h.MaxLevel = level
		h.mu.Unlock()
		return node
	}

	// 获取入口点和最大层级的快照（最小化锁定时间）
	entryPoint := h.EntryPoint
	currentMaxLevel := h.MaxLevel
	h.mu.Unlock()

	// 遍历每一层，从当前最大层开始往下（不持有全局锁）
	for l := currentMaxLevel; l >= 0; l-- {
		nearest := h.searchLayer(normalizedVector, entryPoint, 1, l)
		if len(nearest) > 0 {
			entryPoint = nearest[0].Node
		}
	}

	// 在底层搜索最近邻
	neighbors := h.searchLayer(normalizedVector, entryPoint, h.EfConstruction, 0)

	// 添加双向连接
	h.addConnections(node, neighbors, 0)

	// 在更高层插入（如果新节点有多层）
	for l := 1; l <= level; l++ {
		// 为这一层搜索
		entryPoint = h.getEntryPointForLevel(l)
		neighbors = h.searchLayer(normalizedVector, entryPoint, h.EfConstruction, l)

		// 添加连接
		h.addConnections(node, neighbors, l)

		// 更新该层的入口点
		h.setEntryPointForLevel(l, node)
	}

	// 更新最大层级（需要持有全局锁）
	h.mu.Lock()
	if level > h.MaxLevel {
		h.MaxLevel = level
		h.EntryPoint = node
	}
	h.mu.Unlock()

	return node
}

// 搜索最近邻
func (h *HNSWGraph) Search(query []float32, k int) []SearchResult {
	return h.SearchWithEf(query, k, h.EfSearch)
}

// SearchWithEf 搜索最近邻（自定义ef参数）
// ef: 搜索时扩展因子，越大越精确但越慢
func (h *HNSWGraph) SearchWithEf(query []float32, k, ef int) []SearchResult {
	if ef <= 0 {
		ef = h.EfSearch
	}
	if ef < k {
		ef = k
	}

	h.mu.RLock()
	if h.EntryPoint == nil {
		h.mu.RUnlock()
		return nil
	}

	// 从顶层开始
	entryPoint := h.EntryPoint
	currentMaxLevel := h.MaxLevel
	h.mu.RUnlock()

	// 归一化查询向量
	normalizedQuery := h.normalizeVector(query)

	// 遍历每一层，找到最近的入口点
	for l := currentMaxLevel; l > 0; l-- {
		nearest := h.searchLayer(normalizedQuery, entryPoint, 1, l)
		if len(nearest) > 0 {
			entryPoint = nearest[0].Node
		}
	}

	// 在底层搜索
	results := h.searchLayer(normalizedQuery, entryPoint, ef, 0)

	// 过滤已删除的节点
	validResults := make([]SearchResult, 0, len(results))
	for _, result := range results {
		if !result.Node.Deleted {
			validResults = append(validResults, result)
		}
	}

	// 取前 k 个
	if len(validResults) > k {
		validResults = validResults[:k]
	}

	return validResults
}

// 搜索单层
func (h *HNSWGraph) searchLayer(query []float32, entryPoint *Node, ef, level int) []SearchResult {
	// 从池中获取 visited bitset，减少内存分配
	visited := h.visitedPool.Get()
	defer h.visitedPool.Put(visited)

	candidates := newMinHeap()
	results := newMaxHeap(ef)

	// 计算入口点距离（使用内积距离）
	entryDist := h.innerProductDistance(query, entryPoint.Vector)
	heap.Push(candidates, &candidate{Node: entryPoint, Distance: entryDist})
	visited.Set(entryPoint.ID)

	// 立即添加入口点到结果
	heap.Push(results, &candidate{Node: entryPoint, Distance: entryDist})

	for candidates.Len() > 0 {
		current := heap.Pop(candidates).(*candidate)

		// 如果当前节点比结果中最远的还远，就停止
		if results.Len() >= ef {
			peek := results.Peek()
			if peek != nil && current.Distance > peek.Distance {
				break
			}
		}

		// 遍历邻居
		current.Node.mu.RLock()
		var friends []*Node
		if level < len(current.Node.Friends) {
			friends = current.Node.Friends[level]
		}
		current.Node.mu.RUnlock()

		if friends == nil {
			continue
		}

		// 批量处理邻居，减少锁操作
		for _, friend := range friends {
			if visited.Get(friend.ID) {
				continue
			}
			visited.Set(friend.ID)

			// 使用内积距离（向量已预归一化）
			dist := h.innerProductDistance(query, friend.Vector)

			// 添加到候选
			heap.Push(candidates, &candidate{Node: friend, Distance: dist})

			// 添加到结果
			if results.Len() < ef {
				// 结果未满，直接添加
				heap.Push(results, &candidate{Node: friend, Distance: dist})
			} else {
				// 结果已满，只在距离更近时添加
				peek := results.Peek()
				if peek != nil && dist < peek.Distance {
					heap.Pop(results)
					heap.Push(results, &candidate{Node: friend, Distance: dist})
				}
			}
		}
	}

	// 转换结果
	finalResults := make([]SearchResult, 0, results.Len())
	for results.Len() > 0 {
		item := heap.Pop(results).(*candidate)
		finalResults = append(finalResults, SearchResult{
			Node:     item.Node,
			Distance: item.Distance,
		})
	}

	// 反转结果（从近到远）
	for i, j := 0, len(finalResults)-1; i < j; i, j = i+1, j-1 {
		finalResults[i], finalResults[j] = finalResults[j], finalResults[i]
	}

	return finalResults
}

// 添加连接
func (h *HNSWGraph) addConnections(node *Node, neighbors []SearchResult, level int) {
	maxLinks := h.M
	if level == 0 {
		maxLinks = h.M0
	}

	// 按距离排序
	sort.Slice(neighbors, func(i, j int) bool {
		return neighbors[i].Distance < neighbors[j].Distance
	})

	// 收集需要修剪的节点（先不调用pruneConnections避免死锁）
	needsPruning := make([]*Node, 0)

	// 连接到最近的 maxLinks 个节点
	connected := 0
	for _, neighbor := range neighbors {
		if connected >= maxLinks {
			break
		}

		// 跳过自连接
		if neighbor.Node.ID == node.ID {
			continue
		}

		neighborNode := neighbor.Node

		// 使用一致的锁顺序：总是先锁 ID 较小的节点
		var first, second *Node
		if node.ID < neighborNode.ID {
			first, second = node, neighborNode
		} else {
			first, second = neighborNode, node
		}

		first.mu.Lock()
		second.mu.Lock()

		// 确保两个节点在该层都有足够的空间
		if level >= len(neighborNode.Friends) {
			newFriends := make([][]*Node, level+1)
			copy(newFriends, neighborNode.Friends)
			for i := len(neighborNode.Friends); i <= level; i++ {
				newFriends[i] = make([]*Node, 0)
			}
			neighborNode.Friends = newFriends
		}

		if level >= len(node.Friends) {
			second.mu.Unlock()
			first.mu.Unlock()
			continue
		}

		// 双向连接（使用 slice append，比 map 更快）
		neighborNode.Friends[level] = append(neighborNode.Friends[level], node)
		node.Friends[level] = append(node.Friends[level], neighborNode)

		// 检查是否需要修剪
		if len(neighborNode.Friends[level]) > maxLinks {
			needsPruning = append(needsPruning, neighborNode)
		}

		second.mu.Unlock()
		first.mu.Unlock()

		connected++
	}

	// 检查自身是否需要修剪
	node.mu.RLock()
	hasLevel := level < len(node.Friends)
	nodeCount := 0
	if hasLevel {
		nodeCount = len(node.Friends[level])
	}
	node.mu.RUnlock()

	if hasLevel && nodeCount > maxLinks {
		needsPruning = append(needsPruning, node)
	}

	// 在释放所有锁后再修剪
	for _, nodeToprune := range needsPruning {
		h.pruneConnections(nodeToprune, level, maxLinks)
	}
}

// 修剪连接（使用对象池优化）
func (h *HNSWGraph) pruneConnections(node *Node, level, maxLinks int) {
	node.mu.RLock()
	if level >= len(node.Friends) || len(node.Friends[level]) <= maxLinks {
		node.mu.RUnlock()
		return
	}

	// 获取所有邻居（使用 slice copy）
	neighbors := make([]*Node, len(node.Friends[level]))
	copy(neighbors, node.Friends[level])
	node.mu.RUnlock()

	// 计算距离矩阵（使用内积距离和对象池）
	distances := h.floatSlicePool.Get()
	defer h.floatSlicePool.Put(distances)

	// 扩展distances到所需大小
	if cap(distances) < len(neighbors) {
		distances = make([]float32, len(neighbors))
	} else {
		distances = distances[:len(neighbors)]
	}

	for i, neighbor := range neighbors {
		distances[i] = h.innerProductDistance(node.Vector, neighbor.Vector)
	}

	// 使用启发式算法选择要保留的连接
	selected := h.selectHeuristic(node, neighbors, distances, maxLinks)

	// 更新连接（使用 slice）
	newFriends := make([]*Node, 0, len(selected))
	for _, idx := range selected {
		newFriends = append(newFriends, neighbors[idx])
	}

	node.mu.Lock()
	if level < len(node.Friends) {
		node.Friends[level] = newFriends
	}
	node.mu.Unlock()
}

// 启发式选择连接（使用对象池优化）
func (h *HNSWGraph) selectHeuristic(node *Node, neighbors []*Node, distances []float32, maxLinks int) []int {
	selected := make([]int, 0, maxLinks)
	candidates := h.intSlicePool.Get()
	defer h.intSlicePool.Put(candidates)

	// 扩展candidates到所需大小
	if cap(candidates) < len(neighbors) {
		candidates = make([]int, len(neighbors))
	} else {
		candidates = candidates[:len(neighbors)]
	}

	for i := range candidates {
		candidates[i] = i
	}

	// 先添加最近的
	sort.Slice(candidates, func(i, j int) bool {
		return distances[candidates[i]] < distances[candidates[j]]
	})

	selected = append(selected, candidates[0])

	// 使用启发式算法选择其他连接
	for i := 1; i < len(candidates) && len(selected) < maxLinks; i++ {
		candidateIdx := candidates[i]
		shouldAdd := true
		candidateDist := distances[candidateIdx]

		// 检查候选节点是否比已选节点更远
		for _, selectedIdx := range selected {
			// 使用预归一化向量的内积距离
			dist := h.innerProductDistance(neighbors[selectedIdx].Vector, neighbors[candidateIdx].Vector)
			if dist < candidateDist {
				shouldAdd = false
				break
			}
		}

		if shouldAdd {
			selected = append(selected, candidateIdx)
		}
	}

	return selected
}

// innerProductDistance 预归一化向量的内积距离（使用SIMD优化）
func (h *HNSWGraph) innerProductDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	dot := simd.DotProduct(a, b)
	return 1.0 - dot
}

// 随机层级
func (h *HNSWGraph) randomLevel() int {
	level := 0
	threshold := 1.0 / float64(h.M)
	for h.rng.Float64() < threshold && level < 31 {
		level++
	}
	return level
}

// 辅助结构
type candidate struct {
	Node     *Node
	Distance float32
}

type minHeap []*candidate

type maxHeap []*candidate

func (h minHeap) Len() int           { return len(h) }
func (h minHeap) Less(i, j int) bool { return h[i].Distance < h[j].Distance }
func (h minHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *minHeap) Push(x interface{}) {
	*h = append(*h, x.(*candidate))
}

func (h *minHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

func (h *minHeap) Peek() *candidate {
	if len(*h) == 0 {
		return nil
	}
	return (*h)[0]
}

func (h maxHeap) Len() int           { return len(h) }
func (h maxHeap) Less(i, j int) bool { return h[i].Distance > h[j].Distance }
func (h maxHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *maxHeap) Push(x interface{}) {
	*h = append(*h, x.(*candidate))
}

func (h *maxHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

func (h *maxHeap) Peek() *candidate {
	if len(*h) == 0 {
		return nil
	}
	return (*h)[0]
}

// newMinHeap 创建最小堆
func newMinHeap() *minHeap {
	h := make(minHeap, 0)
	return &h
}

// newMaxHeap 创建最大堆
func newMaxHeap(size int) *maxHeap {
	h := make(maxHeap, 0, size)
	return &h
}

// getEntryPointForLevel 获取某层的入口点
func (h *HNSWGraph) getEntryPointForLevel(level int) *Node {
	// 简单实现：返回第一个有该层的节点
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, node := range h.Nodes {
		if node.Level >= level {
			return node
		}
	}
	return h.EntryPoint
}

// setEntryPointForLevel 设置某层的入口点
func (h *HNSWGraph) setEntryPointForLevel(level int, node *Node) {
	h.mu.Lock()
	defer h.mu.Unlock()
	// 简单实现：更新入口点
	if level > h.MaxLevel {
		h.MaxLevel = level
		h.EntryPoint = node
	}
}

// Delete 软删除节点
func (h *HNSWGraph) Delete(id int) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 查找节点
	node, exists := h.NodeMap[id]
	if !exists {
		return fmt.Errorf("node %d not found", id)
	}

	// 检查是否已经删除
	if node.Deleted {
		return fmt.Errorf("node %d is already deleted", id)
	}

	// 标记为已删除
	node.Deleted = true
	h.DeletedIDs[id] = true

	// 从邻居中移除该节点的引用（不需要重新锁定 h.mu）
	for level := 0; level <= node.Level; level++ {
		for _, neighbor := range node.Friends[level] {
			neighbor.mu.Lock()
			if level < len(neighbor.Friends) {
				// 从 slice 中删除节点
				for i, n := range neighbor.Friends[level] {
					if n.ID == node.ID {
						neighbor.Friends[level] = append(neighbor.Friends[level][:i], neighbor.Friends[level][i+1:]...)
						break
					}
				}
			}
			neighbor.mu.Unlock()
		}
	}

	return nil
}

// Count 返回节点数量（不包括已删除的节点）
func (h *HNSWGraph) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.NodeMap) - len(h.DeletedIDs)
}

// normalizeVector 归一化向量（使用SIMD优化）
func (h *HNSWGraph) normalizeVector(vector []float32) []float32 {
	return simd.NormalizeVector(vector)
}
