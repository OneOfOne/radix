//go:build go1.18
// +build go1.18

package radix

import "sync"

func NewLocked[VT any]() *LockedTree[VT] {
	var lt LockedTree[VT]
	return &lt
}

func NewLockedFromMap[VT any](m map[string]VT) *LockedTree[VT] {
	var lt LockedTree[VT]
	lt.t.MergeMap(m)
	return &lt
}

// LockedTree is a concurrency safe version of Tree.
type LockedTree[VT any] struct {
	t Tree[VT]
	m sync.RWMutex
}

// Update aquires a rw lock and calls fn with the underlying tree
func (lt *LockedTree[VT]) Update(fn func(t *Tree[VT])) {
	lt.m.Lock()
	defer lt.m.Unlock()
	fn(&lt.t)
}

func (lt *LockedTree[VT]) MergeMap(m map[string]VT) {
	lt.m.Lock()
	lt.t.MergeMap(m)
	lt.m.Unlock()
}

func (lt *LockedTree[VT]) Merge(ot *LockedTree[VT]) {
	ot.m.RLock()
	lt.m.Lock()
	lt.t.Merge(&ot.t)
	lt.m.Unlock()
	ot.m.RUnlock()
}

func (lt *LockedTree[VT]) Set(key string, value VT) (old VT, found bool) {
	lt.m.Lock()
	old, found = lt.t.Set(key, value)
	lt.m.Unlock()
	return
}

func (lt *LockedTree[VT]) Delete(key string) (old VT, found bool) {
	lt.m.Lock()
	old, found = lt.t.Delete(key)
	lt.m.Unlock()
	return
}

func (lt *LockedTree[VT]) DeletePrefix(prefix string) (count int) {
	lt.m.Lock()
	count = lt.t.DeletePrefix(prefix)
	lt.m.Unlock()
	return
}

func (lt *LockedTree[VT]) Get(key string) (val VT, found bool) {
	lt.m.RLock()
	val, found = lt.t.Get(key)
	lt.m.RUnlock()
	return
}

func (lt *LockedTree[VT]) LongestPrefix(prefix string) (key string, val VT, found bool) {
	lt.m.RLock()
	key, val, found = lt.t.LongestPrefix(prefix)
	lt.m.RUnlock()
	return
}

func (lt *LockedTree[VT]) Minimum() (key string, val VT, found bool) {
	lt.m.RLock()
	key, val, found = lt.t.Minimum()
	lt.m.RUnlock()
	return
}

func (lt *LockedTree[VT]) Maximum() (key string, val VT, found bool) {
	lt.m.RLock()
	key, val, found = lt.t.Maximum()
	lt.m.RUnlock()
	return
}

func (lt *LockedTree[VT]) Len() (ln int) {
	lt.m.RLock()
	ln = lt.t.Len()
	lt.m.RUnlock()
	return
}

// Walk
// It is *NOT* safe to modify the tree inside fn.
func (lt *LockedTree[VT]) Walk(fn WalkFn[VT]) {
	lt.m.RLock()
	defer lt.m.RUnlock()
	lt.t.Walk(fn)
}

// WalkPrefix
// It is *NOT* safe to modify the tree inside fn.
func (lt *LockedTree[VT]) WalkPrefix(prefix string, fn WalkFn[VT]) {
	lt.m.RLock()
	defer lt.m.RUnlock()
	lt.t.WalkPrefix(prefix, fn)
}

// WalkPath
// It is *NOT* safe to modify the tree inside fn.
func (lt *LockedTree[VT]) WalkPath(path string, fn WalkFn[VT]) {
	lt.m.RLock()
	defer lt.m.RUnlock()
	lt.t.WalkPath(path, fn)
}

// WalkNearestPath
// It is *NOT* safe to modify the tree inside fn.
func (lt *LockedTree[VT]) WalkNearestPath(path string, fn WalkFn[VT]) {
	lt.m.RLock()
	defer lt.m.RUnlock()
	lt.t.WalkNearestPath(path, fn)
}

func (lt *LockedTree[VT]) ToMap() map[string]VT {
	lt.m.RLock()
	out := make(map[string]VT, lt.t.size)
	lt.t.Walk(func(k string, v VT) bool {
		out[k] = v
		return false
	})
	lt.m.RUnlock()
	return out
}
