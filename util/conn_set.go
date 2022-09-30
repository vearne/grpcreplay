package util

import (
	"github.com/vearne/grpcreplay/model"
	"sync"
)

type ConnSet struct {
	rw       sync.RWMutex
	internal map[model.DirectConn]int
}

func NewConnSet() *ConnSet {
	return &ConnSet{internal: make(map[model.DirectConn]int)}
}

func (set *ConnSet) Add(c model.DirectConn) {
	set.rw.Lock()
	defer set.rw.Unlock()

	set.internal[c] = 1
}

func (set *ConnSet) AddAll(cons []model.DirectConn) {
	set.rw.Lock()
	defer set.rw.Unlock()

	for _, conn := range cons {
		set.internal[conn] = 1
	}
}

func (set *ConnSet) Has(c model.DirectConn) bool {
	set.rw.RLock()
	defer set.rw.RUnlock()

	_, ok := set.internal[c]
	return ok
}

func (set *ConnSet) Remove(c model.DirectConn) {
	set.rw.Lock()
	defer set.rw.Unlock()

	delete(set.internal, c)
}

func (set *ConnSet) RemoveAll(other *ConnSet) {
	set.rw.Lock()
	defer set.rw.Unlock()

	for _, conn := range other.ToArray() {
		delete(set.internal, conn)
	}
}

func (set *ConnSet) ToArray() []model.DirectConn {
	set.rw.Lock()
	defer set.rw.Unlock()

	res := make([]model.DirectConn, len(set.internal))
	i := 0
	for key := range set.internal {
		res[i] = key
		i++
	}
	return res
}

func (set *ConnSet) Size() int {
	set.rw.RLock()
	defer set.rw.RUnlock()

	return len(set.internal)
}

func (set *ConnSet) Intersection(set2 *ConnSet) *ConnSet {
	set.rw.RLock()
	defer set.rw.RUnlock()

	result := NewConnSet()
	if set.Size() > set2.Size() {
		set, set2 = set2, set
	}

	for key := range set.internal {
		if _, ok := set2.internal[key]; ok {
			result.Add(key)
		}
	}
	return result
}

func (set *ConnSet) Clone() *ConnSet {
	set.rw.RLock()
	defer set.rw.RUnlock()

	result := NewConnSet()
	result.AddAll(set.ToArray())
	return result
}
