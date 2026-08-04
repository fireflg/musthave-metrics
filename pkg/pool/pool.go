package pool

import "sync"

// Resetter — интерфейс для объектов с методом Reset().
type Resetter interface {
	Reset()
}

// Pool — обобщённая версия sync.Pool с ограничением T Resetter.
type Pool[T Resetter] struct {
	pool sync.Pool
}

// New создаёт и возвращает указатель на новый пул.
func New[T Resetter]() *Pool[T] {
	return &Pool[T]{}
}

// Get возвращает объект из пула. Если пул пуст, возвращает нулевое значение T.
func (p *Pool[T]) Get() T {
	obj, ok := p.pool.Get().(T)
	if !ok {
		var zero T
		return zero
	}
	return obj
}

// Put помещает объект обратно в пул, предварительно сбрасывая его состояние.
func (p *Pool[T]) Put(obj T) {
	obj.Reset()
	p.pool.Put(obj)
}
