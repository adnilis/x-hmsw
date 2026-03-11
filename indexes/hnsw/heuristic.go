package hnsw

import (
	"container/heap"
)

// Candidate 候选节点
type Candidate struct {
	ID       int
	Distance float32
}

// CandidateHeap 候选节点堆
type CandidateHeap []Candidate

func (h CandidateHeap) Len() int           { return len(h) }
func (h CandidateHeap) Less(i, j int) bool { return h[i].Distance < h[j].Distance }
func (h CandidateHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *CandidateHeap) Push(x interface{}) {
	*h = append(*h, x.(Candidate))
}

func (h *CandidateHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// SearchLayer 在某一层搜索
func (h *HNSWGraph) SearchLayer(entryPoints []Candidate, query []float32, layer int, ef int) []Candidate {
	visited := make(map[int]bool)
	candidates := make(CandidateHeap, 0, len(entryPoints))
	best := make(CandidateHeap, 0, ef)

	// 初始化候选队列
	for _, ep := range entryPoints {
		if !visited[ep.ID] {
			visited[ep.ID] = true
			heap.Push(&candidates, ep)
		}
	}

	// 搜索
	for candidates.Len() > 0 {
		current := heap.Pop(&candidates).(Candidate)

		// 如果当前候选比最好的最差，停止
		if best.Len() >= ef && current.Distance > best[best.Len()-1].Distance {
			break
		}

		// 添加到最好结果
		heap.Push(&best, current)
		if best.Len() > ef {
			heap.Pop(&best)
		}

		// 扩展邻居
		h.mu.RLock()
		node := h.Nodes[current.ID]
		h.mu.RUnlock()

		if node != nil && layer < len(node.Friends) {
			node.mu.RLock()
			friends := node.Friends[layer]
			node.mu.RUnlock()

			for _, neighbor := range friends {
				if !visited[neighbor.ID] {
					visited[neighbor.ID] = true
					dist := h.DistanceFunc(query, neighbor.Vector)
					heap.Push(&candidates, Candidate{
						ID:       neighbor.ID,
						Distance: dist,
					})
				}
			}
		}
	}

	return best
}
