package http2

import (
	"strings"
	"sync"
)

type ConnSet struct {
	rw       sync.RWMutex
	internal map[DirectConn]int
}

func NewConnSet() *ConnSet {
	return &ConnSet{internal: make(map[DirectConn]int)}
}

func (set *ConnSet) Add(c DirectConn) {
	set.rw.Lock()
	defer set.rw.Unlock()

	set.internal[c] = 1
}

func (set *ConnSet) AddAll(cons []DirectConn) {
	set.rw.Lock()
	defer set.rw.Unlock()

	for _, conn := range cons {
		set.internal[conn] = 1
	}
}

func (set *ConnSet) Has(c DirectConn) bool {
	set.rw.RLock()
	defer set.rw.RUnlock()

	_, ok := set.internal[c]
	return ok
}

func (set *ConnSet) Remove(c DirectConn) {
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

func (set *ConnSet) ToArray() []DirectConn {
	set.rw.RLock()
	defer set.rw.RUnlock()

	res := make([]DirectConn, len(set.internal))
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

func (set *ConnSet) String() string {
	set.rw.RLock()
	defer set.rw.RUnlock()

	strList := make([]string, 0)
	for _, conn := range set.ToArray() {
		strList = append(strList, conn.String())
	}
	return strings.Join(strList, ",")
}

func (set *ConnSet) Clone() *ConnSet {
	set.rw.RLock()
	defer set.rw.RUnlock()

	result := NewConnSet()
	result.AddAll(set.ToArray())
	return result
}
