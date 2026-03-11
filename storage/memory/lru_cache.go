package memory

import (
	"sync"
)

// LRUCache LRU 缓存
type LRUCache[K comparable, V any] struct {
	capacity int
	cache    map[K]*listNode[K, V]
	head     *listNode[K, V]
	tail     *listNode[K, V]
	mu       sync.RWMutex
}

type listNode[K comparable, V any] struct {
	key   K
	value V
	prev  *listNode[K, V]
	next  *listNode[K, V]
}

// NewLRUCache 创建 LRU 缓存
func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	if capacity <= 0 {
		capacity = 1000
	}

	cache := &LRUCache[K, V]{
		capacity: capacity,
		cache:    make(map[K]*listNode[K, V], capacity),
	}

	// 初始化双向链表头尾哨兵
	cache.head = &listNode[K, V]{}
	cache.tail = &listNode[K, V]{}
	cache.head.next = cache.tail
	cache.tail.prev = cache.head

	return cache
}

// Get 获取缓存项
func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, exists := c.cache[key]
	if !exists {
		var zero V
		return zero, false
	}

	// 移动到头部（最近使用）
	c.moveToFront(node)
	return node.value, true
}

// Put 放入缓存项
func (c *LRUCache[K, V]) Put(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 如果已存在，更新并移到头部
	if node, exists := c.cache[key]; exists {
		node.value = value
		c.moveToFront(node)
		return
	}

	// 如果超出容量，删除最久未使用的项
	if len(c.cache) >= c.capacity {
		c.removeOldest()
	}

	// 添加新节点到头部
	node := &listNode[K, V]{
		key:   key,
		value: value,
	}
	c.addToFront(node)
	c.cache[key] = node
}

// Delete 删除缓存项
func (c *LRUCache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if node, exists := c.cache[key]; exists {
		c.removeNode(node)
		delete(c.cache, key)
	}
}

// Clear 清空缓存
func (c *LRUCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[K]*listNode[K, V])
	c.head.next = c.tail
	c.tail.prev = c.head
}

// Size 返回缓存大小
func (c *LRUCache[K, V]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// Capacity 返回缓存容量
func (c *LRUCache[K, V]) Capacity() int {
	return c.capacity
}

// 辅助方法：移动到头部
func (c *LRUCache[K, V]) moveToFront(node *listNode[K, V]) {
	c.removeNode(node)
	c.addToFront(node)
}

// 辅助方法：添加到头部
func (c *LRUCache[K, V]) addToFront(node *listNode[K, V]) {
	node.next = c.head.next
	node.prev = c.head
	c.head.next.prev = node
	c.head.next = node
}

// 辅助方法：删除节点
func (c *LRUCache[K, V]) removeNode(node *listNode[K, V]) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

// 辅助方法：删除最久未使用的项
func (c *LRUCache[K, V]) removeOldest() {
	oldest := c.tail.prev
	if oldest == c.head {
		return
	}
	c.removeNode(oldest)
	delete(c.cache, oldest.key)
}
