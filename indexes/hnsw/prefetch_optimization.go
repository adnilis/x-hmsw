package hnsw

import (
	"container/heap"
	"sort"
	"unsafe"

	"github.com/adnilis/x-hmsw/utils/prefetch"
)

// PrefetchOptimizedSearchLayer 使用预取优化的searchLayer
func (h *HNSWGraph) PrefetchOptimizedSearchLayer(query []float32, entryPoint *Node, ef, level int) []SearchResult {
	visited := h.visitedPool.Get()
	defer h.visitedPool.Put(visited)

	candidates := newMinHeap()
	results := newMaxHeap(ef)

	// 预取入口点向量
	prefetch.PrefetchVector(entryPoint.Vector)

	entryDist := h.innerProductDistance(query, entryPoint.Vector)
	heap.Push(candidates, &candidate{Node: entryPoint, Distance: entryDist})
	visited.Set(entryPoint.ID)

	heap.Push(results, &candidate{Node: entryPoint, Distance: entryDist})

	for candidates.Len() > 0 {
		current := heap.Pop(candidates).(*candidate)

		if results.Len() >= ef {
			peek := results.Peek()
			if peek != nil && current.Distance > peek.Distance {
				break
			}
		}

		current.Node.mu.RLock()
		var friends []*Node
		if level < len(current.Node.Friends) {
			friends = current.Node.Friends[level]
		}
		current.Node.mu.RUnlock()

		if friends == nil {
			continue
		}

		// 预取邻居节点
		for i := 0; i < len(friends) && i < 4; i++ {
			prefetch.PrefetchVector(friends[i].Vector)
		}

		for _, friend := range friends {
			if visited.Get(friend.ID) {
				continue
			}
			visited.Set(friend.ID)

			// 预取向量对
			prefetch.PrefetchVectorPair(query, friend.Vector)

			dist := h.innerProductDistance(query, friend.Vector)

			heap.Push(candidates, &candidate{Node: friend, Distance: dist})

			if results.Len() < ef {
				heap.Push(results, &candidate{Node: friend, Distance: dist})
			} else {
				peek := results.Peek()
				if peek != nil && dist < peek.Distance {
					heap.Pop(results)
					heap.Push(results, &candidate{Node: friend, Distance: dist})
				}
			}
		}
	}

	finalResults := make([]SearchResult, 0, results.Len())
	for results.Len() > 0 {
		item := heap.Pop(results).(*candidate)
		finalResults = append(finalResults, SearchResult{
			Node:     item.Node,
			Distance: item.Distance,
		})
	}

	// 反转结果
	for i, j := 0, len(finalResults)-1; i < j; i, j = i+1, j-1 {
		finalResults[i], finalResults[j] = finalResults[j], finalResults[i]
	}

	return finalResults
}

// PrefetchOptimizedPruneConnections 使用预取优化的pruneConnections
func (h *HNSWGraph) PrefetchOptimizedPruneConnections(node *Node, level, maxLinks int) {
	node.mu.RLock()
	if level >= len(node.Friends) || len(node.Friends[level]) <= maxLinks {
		node.mu.RUnlock()
		return
	}

	neighbors := make([]*Node, len(node.Friends[level]))
	copy(neighbors, node.Friends[level])
	node.mu.RUnlock()

	distances := h.floatSlicePool.Get()
	defer h.floatSlicePool.Put(distances)

	if cap(distances) < len(neighbors) {
		distances = make([]float32, len(neighbors))
	} else {
		distances = distances[:len(neighbors)]
	}

	// 预取节点向量
	prefetch.PrefetchVector(node.Vector)

	// 预取邻居向量
	for i := 0; i < len(neighbors) && i < 4; i++ {
		prefetch.PrefetchVector(neighbors[i].Vector)
	}

	for i, neighbor := range neighbors {
		// 预取向量对
		prefetch.PrefetchVectorPair(node.Vector, neighbor.Vector)
		distances[i] = h.innerProductDistance(node.Vector, neighbor.Vector)
	}

	selected := h.selectHeuristic(node, neighbors, distances, maxLinks)

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

// PrefetchOptimizedSelectHeuristic 使用预取优化的selectHeuristic
func (h *HNSWGraph) PrefetchOptimizedSelectHeuristic(node *Node, neighbors []*Node, distances []float32, maxLinks int) []int {
	selected := make([]int, 0, maxLinks)
	candidates := h.intSlicePool.Get()
	defer h.intSlicePool.Put(candidates)

	if cap(candidates) < len(neighbors) {
		candidates = make([]int, len(neighbors))
	} else {
		candidates = candidates[:len(neighbors)]
	}

	for i := range candidates {
		candidates[i] = i
	}

	sort.Slice(candidates, func(i, j int) bool {
		return distances[candidates[i]] < distances[candidates[j]]
	})

	selected = append(selected, candidates[0])

	for i := 1; i < len(candidates) && len(selected) < maxLinks; i++ {
		candidateIdx := candidates[i]
		shouldAdd := true
		candidateDist := distances[candidateIdx]

		// 预取已选节点向量
		for j := 0; j < len(selected) && j < 2; j++ {
			prefetch.PrefetchVector(neighbors[selected[j]].Vector)
		}

		for _, selectedIdx := range selected {
			// 预取向量对
			prefetch.PrefetchVectorPair(neighbors[selectedIdx].Vector, neighbors[candidateIdx].Vector)
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

// PrefetchOptimizedInnerProductDistance 使用预取优化的内积距离计算
func (h *HNSWGraph) PrefetchOptimizedInnerProductDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	// 预取向量数据
	prefetch.PrefetchVectorPair(a, b)

	return h.innerProductDistance(a, b)
}

// PrefetchOptimizedNormalizeVector 使用预取优化的向量归一化
func (h *HNSWGraph) PrefetchOptimizedNormalizeVector(v []float32) []float32 {
	if len(v) == 0 {
		return v
	}

	// 预取向量数据
	prefetch.PrefetchVector(v)

	return h.normalizeVector(v)
}

// PrefetchOptimizedBatchSearch 使用预取优化的批量搜索
func (h *HNSWGraph) PrefetchOptimizedBatchSearch(queries [][]float32, k int) [][]SearchResult {
	results := make([][]SearchResult, len(queries))

	for i, query := range queries {
		// 预取下一个查询
		if i+1 < len(queries) {
			prefetch.PrefetchVector(queries[i+1])
		}

		results[i] = h.Search(query, k)
	}

	return results
}

// PrefetchOptimizedBatchInsert 使用预取优化的批量插入
func (h *HNSWGraph) PrefetchOptimizedBatchInsert(ids []int, vectors [][]float32) {
	for i := range ids {
		// 预取下一个向量
		if i+1 < len(vectors) {
			prefetch.PrefetchVector(vectors[i+1])
		}

		h.Insert(ids[i], vectors[i])
	}
}

// PrefetchNodeData 预取节点数据
func (h *HNSWGraph) PrefetchNodeData(node *Node) {
	if node == nil {
		return
	}

	// 预取向量
	prefetch.PrefetchVector(node.Vector)

	// 预取邻居
	for level := 0; level < len(node.Friends) && level < 2; level++ {
		for i := 0; i < len(node.Friends[level]) && i < 4; i++ {
			prefetch.PrefetchVector(node.Friends[level][i].Vector)
		}
	}
}

// PrefetchUnsafe 使用unsafe指针预取
func PrefetchUnsafe(ptr unsafe.Pointer) {
	prefetch.PrefetchT0(ptr)
}

// PrefetchSliceUnsafe 使用unsafe指针预取切片
func PrefetchSliceUnsafe(slice []float32, offset int) {
	if offset < 0 || offset >= len(slice) {
		return
	}
	ptr := unsafe.Pointer(&slice[offset])
	prefetch.PrefetchT0(ptr)
}
