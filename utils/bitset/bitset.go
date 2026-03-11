package bitset

import (
	"sync"
)

// BitSet 高效的位集合实现，用于跟踪访问状态
type BitSet struct {
	bits []uint64
	size int
}

// NewBitSet 创建指定大小的位集合
func NewBitSet(size int) *BitSet {
	// 计算需要的uint64数量
	numWords := (size + 63) / 64
	return &BitSet{
		bits: make([]uint64, numWords),
		size: size,
	}
}

// Set 设置指定位
func (bs *BitSet) Set(index int) {
	if index < 0 || index >= bs.size {
		return
	}
	word := index / 64
	bit := uint64(1) << (index % 64)
	bs.bits[word] |= bit
}

// Get 获取指定位的值
func (bs *BitSet) Get(index int) bool {
	if index < 0 || index >= bs.size {
		return false
	}
	word := index / 64
	bit := uint64(1) << (index % 64)
	return (bs.bits[word] & bit) != 0
}

// Clear 清除指定位
func (bs *BitSet) Clear(index int) {
	if index < 0 || index >= bs.size {
		return
	}
	word := index / 64
	bit := uint64(1) << (index % 64)
	bs.bits[word] &^= bit
}

// ClearAll 清除所有位
func (bs *BitSet) ClearAll() {
	for i := range bs.bits {
		bs.bits[i] = 0
	}
}

// Size 返回位集合的大小
func (bs *BitSet) Size() int {
	return bs.size
}

// BitSetPool 位集合对象池，用于复用BitSet减少内存分配
type BitSetPool struct {
	pool sync.Pool
	size int
}

// NewBitSetPool 创建指定大小的位集合池
func NewBitSetPool(size int) *BitSetPool {
	return &BitSetPool{
		pool: sync.Pool{
			New: func() interface{} {
				return NewBitSet(size)
			},
		},
		size: size,
	}
}

// Get 从池中获取一个位集合
func (bsp *BitSetPool) Get() *BitSet {
	bs := bsp.pool.Get().(*BitSet)
	bs.ClearAll()
	return bs
}

// Put 将位集合归还到池中
func (bsp *BitSetPool) Put(bs *BitSet) {
	bsp.pool.Put(bs)
}
