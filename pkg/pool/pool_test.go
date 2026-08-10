package pool

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testObject struct {
	Value int
}

func (t *testObject) Reset() { t.Value = 0 }

func TestNew(t *testing.T) {
	t.Parallel()

	p := New[*testObject]()
	assert.NotNil(t, p)
}

func TestPut_ResetsObject(t *testing.T) {
	t.Parallel()

	p := New[*testObject]()

	obj := &testObject{Value: 42}
	p.Put(obj)

	assert.Equal(t, 0, obj.Value, "Put должен сбрасывать состояние объекта")
}

func TestGet_EmptyPoolReturnsZeroValue(t *testing.T) {
	t.Parallel()

	p := New[*testObject]()

	assert.Nil(t, p.Get(), "у пула нет фабрики, поэтому пустой пул отдаёт zero value")
}

func TestGet_NeverReturnsStaleState(t *testing.T) {
	t.Parallel()

	p := New[*testObject]()
	p.Put(&testObject{Value: 100})

	if got := p.Get(); got != nil {
		assert.Equal(t, 0, got.Value)
	}
}

func TestPool_Concurrent(t *testing.T) {
	t.Parallel()

	p := New[*testObject]()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if obj := p.Get(); obj != nil {
					obj.Value = j
					p.Put(obj)
					continue
				}
				p.Put(&testObject{Value: j})
			}
		}()
	}

	wg.Wait()
}

func TestResetterConstraint(t *testing.T) {
	t.Parallel()

	var r Resetter = &testObject{Value: 10}
	r.Reset()

	assert.Equal(t, 0, r.(*testObject).Value)
}
