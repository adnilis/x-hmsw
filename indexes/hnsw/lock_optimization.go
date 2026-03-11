package hnsw

import (
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/adnilis/x-hmsw/utils/bitset"
	"github.com/adnilis/x-hmsw/utils/pool"
)

// ShardedLock 分片锁，减少锁竞争
// 将锁分成多个分片，每个分片负责一部分数据
type ShardedLock struct {
	shards []sync.RWMutex
	count  int
}

// NewShardedLock 创建分片锁
func NewShardedLock(shardCount int) *ShardedLock {
	if shardCount <= 0 {
		shardCount = 16 // 默认16个分片
	}
	return &ShardedLock{
		shards: make([]sync.RWMutex, shardCount),
		count:  shardCount,
	}
}

// getShard 获取对应的分片锁
func (sl *ShardedLock) getShard(key int) *sync.RWMutex {
	return &sl.shards[key%sl.count]
}

// Lock 获取写锁
func (sl *ShardedLock) Lock(key int) {
	sl.getShard(key).Lock()
}

// Unlock 释放写锁
func (sl *ShardedLock) Unlock(key int) {
	sl.getShard(key).Unlock()
}

// RLock 获取读锁
func (sl *ShardedLock) RLock(key int) {
	sl.getShard(key).RLock()
}

// RUnlock 释放读锁
func (sl *ShardedLock) RUnlock(key int) {
	sl.getShard(key).RUnlock()
}

// OptimizedNode 优化的节点结构，使用更细粒度的锁
type OptimizedNode struct {
	ID       int
	Vector   []float32
	Level    int
	Friends  [][]*Node // 每层的邻居
	MaxLinks int
	Deleted  bool
	// 使用atomic操作代替锁
	deletedFlag int32
	// 分片锁，每个层级一个锁
	levelLocks []sync.RWMutex
}

// IsDeleted 原子操作检查是否删除
func (on *OptimizedNode) IsDeleted() bool {
	return atomic.LoadInt32(&on.deletedFlag) == 1
}

// MarkDeleted 原子操作标记删除
func (on *OptimizedNode) MarkDeleted() {
	atomic.StoreInt32(&on.deletedFlag, 1)
}

// UnmarkDeleted 原子操作取消删除标记
func (on *OptimizedNode) UnmarkDeleted() {
	atomic.StoreInt32(&on.deletedFlag, 0)
}

// GetFriends 获取某层的邻居（使用读锁）
func (on *OptimizedNode) GetFriends(level int) []*Node {
	if level >= len(on.levelLocks) {
		return nil
	}
	on.levelLocks[level].RLock()
	defer on.levelLocks[level].RUnlock()

	if level >= len(on.Friends) {
		return nil
	}
	return on.Friends[level]
}

// AddFriend 添加邻居（使用写锁）
func (on *OptimizedNode) AddFriend(level int, friend *Node) {
	if level >= len(on.levelLocks) {
		return
	}
	on.levelLocks[level].Lock()
	defer on.levelLocks[level].Unlock()

	// 扩展Friends切片
	for len(on.Friends) <= level {
		on.Friends = append(on.Friends, make([]*Node, 0))
	}
	on.Friends[level] = append(on.Friends[level], friend)
}

// SetFriends 设置某层的邻居（使用写锁）
func (on *OptimizedNode) SetFriends(level int, friends []*Node) {
	if level >= len(on.levelLocks) {
		return
	}
	on.levelLocks[level].Lock()
	defer on.levelLocks[level].Unlock()

	// 扩展Friends切片
	for len(on.Friends) <= level {
		on.Friends = append(on.Friends, make([]*Node, 0))
	}
	on.Friends[level] = friends
}

// LockFreeCounter 无锁计数器
type LockFreeCounter struct {
	value int64
}

// Increment 原子递增
func (lfc *LockFreeCounter) Increment() int64 {
	return atomic.AddInt64(&lfc.value, 1)
}

// Decrement 原子递减
func (lfc *LockFreeCounter) Decrement() int64 {
	return atomic.AddInt64(&lfc.value, -1)
}

// Get 获取当前值
func (lfc *LockFreeCounter) Get() int64 {
	return atomic.LoadInt64(&lfc.value)
}

// OptimizedHNSWGraph 优化的HNSW图，使用分片锁和无锁数据结构
type OptimizedHNSWGraph struct {
	Nodes          []*Node
	MaxLevel       int32 // 使用atomic
	LevelMult      float64
	EfConstruction int
	EfSearch       int
	M              int
	M0             int
	DistanceFunc   func(a, b []float32) float32
	EntryPoint     *Node
	MaxNodes       int
	rng            *rand.Rand
	// 分片锁替代全局锁
	nodeLocks      *ShardedLock
	levelLocks     []sync.RWMutex
	NodeMap        map[int]*Node
	DeletedIDs     map[int]bool
	visitedPool    *bitset.BitSetPool
	intSlicePool   *pool.IntSlicePool
	floatSlicePool *pool.Float32SlicePool
	// 无锁计数器
	insertCounter LockFreeCounter
	searchCounter LockFreeCounter
}

// NewOptimizedHNSWGraph 创建优化的HNSW图
func NewOptimizedHNSWGraph(dim, M, efConstruction, maxNodes int, distanceFunc func(a, b []float32) float32) *OptimizedHNSWGraph {
	if M <= 0 {
		M = 16
	}
	if efConstruction <= 0 {
		efConstruction = 200
	}
	levelMult := 1.0 / math.Log(float64(M))

	graph := &OptimizedHNSWGraph{
		Nodes:          make([]*Node, 0, maxNodes),
		MaxLevel:       0,
		LevelMult:      levelMult,
		EfConstruction: efConstruction,
		EfSearch:       100,
		M:              M,
		M0:             M * 2,
		DistanceFunc:   distanceFunc,
		EntryPoint:     nil,
		MaxNodes:       maxNodes,
		rng:            rand.New(rand.NewSource(time.Now().UnixNano())),
		nodeLocks:      NewShardedLock(32), // 32个分片
		levelLocks:     make([]sync.RWMutex, 32),
		NodeMap:        make(map[int]*Node),
		DeletedIDs:     make(map[int]bool),
		visitedPool:    bitset.NewBitSetPool(maxNodes),
		intSlicePool:   pool.NewIntSlicePool(M * 2),
		floatSlicePool: pool.NewFloat32SlicePool(M * 2),
	}

	return graph
}

// GetNode 使用分片锁获取节点
func (oh *OptimizedHNSWGraph) GetNode(id int) *Node {
	oh.nodeLocks.RLock(id)
	defer oh.nodeLocks.RUnlock(id)
	return oh.NodeMap[id]
}

// SetNode 使用分片锁设置节点
func (oh *OptimizedHNSWGraph) SetNode(id int, node *Node) {
	oh.nodeLocks.Lock(id)
	defer oh.nodeLocks.Unlock(id)
	oh.NodeMap[id] = node
}

// GetMaxLevel 原子操作获取最大层级
func (oh *OptimizedHNSWGraph) GetMaxLevel() int32 {
	return atomic.LoadInt32(&oh.MaxLevel)
}

// SetMaxLevel 原子操作设置最大层级
func (oh *OptimizedHNSWGraph) SetMaxLevel(level int32) {
	atomic.StoreInt32(&oh.MaxLevel, level)
}

// CompareAndSwapMaxLevel 原子操作比较并交换最大层级
func (oh *OptimizedHNSWGraph) CompareAndSwapMaxLevel(old, new int32) bool {
	return atomic.CompareAndSwapInt32(&oh.MaxLevel, old, new)
}

// GetInsertCount 获取插入计数
func (oh *OptimizedHNSWGraph) GetInsertCount() int64 {
	return oh.insertCounter.Get()
}

// GetSearchCount 获取搜索计数
func (oh *OptimizedHNSWGraph) GetSearchCount() int64 {
	return oh.searchCounter.Get()
}
