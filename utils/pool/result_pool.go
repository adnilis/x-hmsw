package pool

import (
	"sync"
)

// SearchResultPool 搜索结果对象池
type SearchResultPool struct {
	pool sync.Pool
}

// NewSearchResultPool 创建搜索结果池
func NewSearchResultPool() *SearchResultPool {
	return &SearchResultPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]interface{}, 0, 100)
			},
		},
	}
}

// Get 从池中获取一个切片
func (p *SearchResultPool) Get() []interface{} {
	return p.pool.Get().([]interface{})
}

// Put 将切片归还到池中
func (p *SearchResultPool) Put(s []interface{}) {
	// 清空切片
	s = s[:0]
	p.pool.Put(s)
}

// IntSlicePool int切片对象池
type IntSlicePool struct {
	pool sync.Pool
	size int
}

// NewIntSlicePool 创建int切片池
func NewIntSlicePool(size int) *IntSlicePool {
	return &IntSlicePool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]int, 0, size)
			},
		},
		size: size,
	}
}

// Get 从池中获取一个切片
func (p *IntSlicePool) Get() []int {
	return p.pool.Get().([]int)
}

// Put 将切片归还到池中
func (p *IntSlicePool) Put(s []int) {
	// 清空切片
	s = s[:0]
	p.pool.Put(s)
}

// Float32SlicePool float32切片对象池
type Float32SlicePool struct {
	pool sync.Pool
	size int
}

// NewFloat32SlicePool 创建float32切片池
func NewFloat32SlicePool(size int) *Float32SlicePool {
	return &Float32SlicePool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]float32, 0, size)
			},
		},
		size: size,
	}
}

// Get 从池中获取一个切片
func (p *Float32SlicePool) Get() []float32 {
	return p.pool.Get().([]float32)
}

// Put 将切片归还到池中
func (p *Float32SlicePool) Put(s []float32) {
	// 清空切片
	s = s[:0]
	p.pool.Put(s)
}
