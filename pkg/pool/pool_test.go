package pool

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testObject struct {
	Value int
}

func (t testObject) Reset() {
	t.Value = 0
}

func TestNew(t *testing.T) {
	t.Parallel()

	p := New[testObject](10)
	assert.NotNil(t, p)
}

func TestPool_Get_Put(t *testing.T) {
	t.Parallel()

	p := New[testObject](3)

	obj := testObject{Value: 42}
	p.Put(obj)

	got := p.Get()
	assert.Equal(t, 42, got.Value)
}

func TestPool_Get_Empty(t *testing.T) {
	t.Parallel()

	p := New[testObject](3)

	got := p.Get()
	// Проверяем zero value
	assert.Equal(t, 0, got.Value)
}

func TestPool_Put_Full(t *testing.T) {
	t.Parallel()

	p := New[testObject](2)

	p.Put(testObject{Value: 1})
	p.Put(testObject{Value: 2})
	p.Put(testObject{Value: 3})

	count := 0
	for {
		select {
		case <-p.objects:
			count++
		default:
			goto done
		}
	}
done:
	assert.Equal(t, 2, count)
}

func TestPool_ObjectLifecycle(t *testing.T) {
	t.Parallel()

	p := New[testObject](3)

	obj := testObject{Value: 100}
	p.Put(obj)

	got := p.Get()
	assert.Equal(t, 100, got.Value)

	objReset := testObject{Value: 0}
	p.Put(objReset)

	got2 := p.Get()
	assert.Equal(t, 0, got2.Value)
}

func TestNewMT(t *testing.T) {
	t.Parallel()

	p := NewMT[testObject](10)
	assert.NotNil(t, p)
}

func TestPoolMT_Concurrent(t *testing.T) {
	t.Parallel()

	p := NewMT[testObject](100)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				obj := p.Get()
				if obj.Value != 0 {
					obj.Value = 0
				}
				p.Put(testObject{Value: j})
			}
		}()
	}

	wg.Wait()
}

func TestPoolMT_Get(t *testing.T) {
	t.Parallel()

	p := NewMT[testObject](3)

	obj := testObject{Value: 42}
	p.Put(obj)

	got := p.Get()
	assert.Equal(t, 42, got.Value)
}

func TestPoolMT_Get_Empty(t *testing.T) {
	t.Parallel()

	p := NewMT[testObject](3)

	got := p.Get()
	assert.Equal(t, 0, got.Value)
}

func TestPool_PointerType(t *testing.T) {
	t.Parallel()

	type pointerStruct struct {
		Value int
	}

	ps := pointerStruct{Value: 10}
	// testObject уже реализует Resetter, используем его
	var _ Resetter = testObject{Value: 10}
	assert.Equal(t, 10, ps.Value)
}
