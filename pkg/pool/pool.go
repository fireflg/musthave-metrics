package pool

import "sync"

// Resetter — интерфейс для объектов с методом Reset().
type Resetter interface {
	Reset()
}

// Pool — пул объектов с generic-параметром.
type Pool[T Resetter] struct {
	objects chan T
}

// New создаёт и возвращает указатель на новый пул объектов.
func New[T Resetter](size int) *Pool[T] {
	return &Pool[T]{
		objects: make(chan T, size),
	}
}

// Get возвращает объект из пула.
func (p *Pool[T]) Get() T {
	select {
	case obj := <-p.objects:
		return obj
	default:
		var zero T
		return zero
	}
}

// Put помещает объект обратно в пул.
func (p *Pool[T]) Put(obj T) {
	select {
	case p.objects <- obj:
	default:
	}
}

// PoolMT — потокобезопасная версия пула.
type PoolMT[T Resetter] struct {
	pool Pool[T]
	mu   sync.Mutex
}

// NewMT создаёт и возвращает указатель на потокобезопасный пул.
func NewMT[T Resetter](size int) *PoolMT[T] {
	return &PoolMT[T]{
		pool: Pool[T]{
			objects: make(chan T, size),
		},
	}
}

// Get возвращает объект из пула в потокобезопасном режиме.
func (p *PoolMT[T]) Get() T {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.pool.Get()
}

// Put помещает объект в пул в потокобезопасном режиме.
func (p *PoolMT[T]) Put(obj T) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pool.Put(obj)
}
