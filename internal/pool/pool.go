package pool

import "sync"

// Pool is a generic wrapper around sync.Pool with typed Get/Put and an
// optional destructor that resets items before they return to the pool.
type Pool[T any] struct {
	pool       sync.Pool
	destructor func(T) T
}

// New creates a Pool. The allocator creates fresh items when the pool is
// empty. The destructor (may be nil) resets an item before it is returned
// to the pool; it receives the item and must return the (possibly modified)
// value to store.
func New[T any](allocator func() T, destructor func(T) T) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any { return allocator() },
		},
		destructor: destructor,
	}
}

// Get retrieves an item from the pool (or allocates a new one).
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T) //nolint:forcetypeassert
}

// Put returns an item to the pool. If a destructor was provided, the item
// is reset before being stored.
func (p *Pool[T]) Put(item T) {
	if p.destructor != nil {
		item = p.destructor(item)
	}
	p.pool.Put(item)
}
