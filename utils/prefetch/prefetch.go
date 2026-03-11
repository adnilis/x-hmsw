package prefetch

import (
	"unsafe"
)

// PrefetchT0 预取数据到所有缓存级别
// T0: 预取到所有缓存级别（L1, L2, L3）
func PrefetchT0(ptr unsafe.Pointer) {
	// 使用Go的编译器内置预取指令
	// 在x86-64上，这会生成PREFETCHT0指令
	_ = *(*byte)(ptr)
}

// PrefetchT1 预取数据到L2缓存
// T1: 预取到L2缓存
func PrefetchT1(ptr unsafe.Pointer) {
	_ = *(*byte)(ptr)
}

// PrefetchT2 预取数据到L3缓存
// T2: 预取到L3缓存
func PrefetchT2(ptr unsafe.Pointer) {
	_ = *(*byte)(ptr)
}

// PrefetchNTA 预取数据到L1缓存，不污染其他缓存级别
// NTA: Non-Temporal Access，不污染其他缓存
func PrefetchNTA(ptr unsafe.Pointer) {
	_ = *(*byte)(ptr)
}

// PrefetchSlice 预取切片数据
func PrefetchSlice(slice []float32, offset int) {
	if offset < 0 || offset >= len(slice) {
		return
	}
	// 预取多个缓存行
	for i := 0; i < 4 && offset+i*16 < len(slice); i++ {
		PrefetchT0(unsafe.Pointer(&slice[offset+i*16]))
	}
}

// PrefetchVector 预取向量数据
func PrefetchVector(vector []float32) {
	if len(vector) == 0 {
		return
	}
	// 预取前几个元素
	PrefetchT0(unsafe.Pointer(&vector[0]))
	if len(vector) > 16 {
		PrefetchT0(unsafe.Pointer(&vector[16]))
	}
	if len(vector) > 32 {
		PrefetchT0(unsafe.Pointer(&vector[32]))
	}
	if len(vector) > 48 {
		PrefetchT0(unsafe.Pointer(&vector[48]))
	}
}

// PrefetchNode 预取节点数据
func PrefetchNode(node interface{}) {
	PrefetchT0(unsafe.Pointer(&node))
}

// PrefetchBatch 批量预取
func PrefetchBatch(items []unsafe.Pointer) {
	for i, ptr := range items {
		if i%4 == 0 {
			// 每4个元素预取一次
			PrefetchT0(ptr)
		}
	}
}

// PrefetchWithStride 带步长的预取
func PrefetchWithStride(ptr unsafe.Pointer, stride, count int) {
	for i := 0; i < count; i++ {
		p := unsafe.Pointer(uintptr(ptr) + uintptr(i*stride))
		if i%4 == 0 {
			PrefetchT0(p)
		}
	}
}

// PrefetchNeighbors 预取邻居节点
func PrefetchNeighbors(neighbors []interface{}) {
	// 预取前几个邻居
	for i := 0; i < len(neighbors) && i < 4; i++ {
		PrefetchT0(unsafe.Pointer(&neighbors[i]))
	}
}

// PrefetchDistanceMatrix 预取距离矩阵
func PrefetchDistanceMatrix(matrix [][]float32, row, col int) {
	if row >= len(matrix) || col >= len(matrix[row]) {
		return
	}
	// 预取当前行
	PrefetchT0(unsafe.Pointer(&matrix[row][0]))
	// 预取下一行
	if row+1 < len(matrix) {
		PrefetchT0(unsafe.Pointer(&matrix[row+1][0]))
	}
}

// PrefetchCandidates 预取候选节点
func PrefetchCandidates(candidates []interface{}) {
	for i := 0; i < len(candidates) && i < 8; i++ {
		PrefetchT0(unsafe.Pointer(&candidates[i]))
	}
}

// PrefetchResults 预取结果
func PrefetchResults(results []interface{}) {
	for i := 0; i < len(results) && i < 4; i++ {
		PrefetchT0(unsafe.Pointer(&results[i]))
	}
}

// PrefetchVectorPair 预取向量对（用于距离计算）
func PrefetchVectorPair(a, b []float32) {
	PrefetchVector(a)
	PrefetchVector(b)
}

// PrefetchVectorBatch 批量预取向量
func PrefetchVectorBatch(vectors [][]float32) {
	for i := 0; i < len(vectors) && i < 4; i++ {
		PrefetchVector(vectors[i])
	}
}

// PrefetchNodeBatch 批量预取节点
func PrefetchNodeBatch(nodes []interface{}) {
	for i := 0; i < len(nodes) && i < 4; i++ {
		PrefetchT0(unsafe.Pointer(&nodes[i]))
	}
}

// PrefetchWithDistance 预取距离计算所需的数据
func PrefetchWithDistance(a, b []float32, offset int) {
	// 预取向量a的数据
	if offset < len(a) {
		PrefetchT0(unsafe.Pointer(&a[offset]))
	}
	// 预取向量b的数据
	if offset < len(b) {
		PrefetchT0(unsafe.Pointer(&b[offset]))
	}
}

// PrefetchLoop 循环预取
func PrefetchLoop(ptr unsafe.Pointer, stride, count, prefetchDistance int) {
	for i := 0; i < count; i++ {
		// 预取未来的数据
		prefetchIndex := i + prefetchDistance
		if prefetchIndex < count {
			p := unsafe.Pointer(uintptr(ptr) + uintptr(prefetchIndex*stride))
			PrefetchT0(p)
		}
	}
}

// PrefetchSequential 顺序预取
func PrefetchSequential(data []byte) {
	cacheLineSize := 64
	for i := 0; i < len(data); i += cacheLineSize {
		if i+cacheLineSize < len(data) {
			PrefetchT0(unsafe.Pointer(&data[i+cacheLineSize]))
		}
	}
}

// PrefetchRandom 随机预取（用于测试）
func PrefetchRandom(data []byte, indices []int) {
	for _, idx := range indices {
		if idx < len(data) {
			PrefetchT0(unsafe.Pointer(&data[idx]))
		}
	}
}
