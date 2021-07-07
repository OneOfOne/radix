//go:build !go1.18
// +build !go1.18

package radix

import "sync"

func NewLocked() *LockedTree {
	var lt LockedTree
	return &lt
}

func NewLockedFromMap(m map[string]interface{}) *LockedTree {
	var lt LockedTree
	lt.t.MergeMap(m)
	return &lt
}

// LockedTree is a concurrency safe version of Tree.
type LockedTree struct {
	t Tree
	m sync.RWMutex
}

// Update aquires a rw lock and calls fn with the underlying tree
func (lt *LockedTree) Update(fn func(t *Tree)) {
	lt.m.Lock()
	defer lt.m.Unlock()
	fn(&lt.t)
}

func (lt *LockedTree) MergeMap(m map[string]interface{}) {
	lt.m.Lock()
	lt.t.MergeMap(m)
	lt.m.Unlock()
}

func (lt *LockedTree) Merge(ot *LockedTree) {
	ot.m.RLock()
	lt.m.Lock()
	lt.t.Merge(&ot.t)
	lt.m.Unlock()
	ot.m.RUnlock()
}

func (lt *LockedTree) Set(key string, value interface{}) (old interface{}, found bool) {
	lt.m.Lock()
	old, found = lt.t.Set(key, value)
	lt.m.Unlock()
	return
}

func (lt *LockedTree) Delete(key string) (old interface{}, found bool) {
	lt.m.Lock()
	old, found = lt.t.Delete(key)
	lt.m.Unlock()
	return
}

func (lt *LockedTree) DeletePrefix(prefix string) (count int) {
	lt.m.Lock()
	count = lt.t.DeletePrefix(prefix)
	lt.m.Unlock()
	return
}

func (lt *LockedTree) Get(key string) (val interface{}, found bool) {
	lt.m.RLock()
	val, found = lt.t.Get(key)
	lt.m.RUnlock()
	return
}

func (lt *LockedTree) LongestPrefix(prefix string) (key string, val interface{}, found bool) {
	lt.m.RLock()
	key, val, found = lt.t.LongestPrefix(prefix)
	lt.m.RUnlock()
	return
}

func (lt *LockedTree) Minimum() (key string, val interface{}, found bool) {
	lt.m.RLock()
	key, val, found = lt.t.Minimum()
	lt.m.RUnlock()
	return
}

func (lt *LockedTree) Maximum() (key string, val interface{}, found bool) {
	lt.m.RLock()
	key, val, found = lt.t.Maximum()
	lt.m.RUnlock()
	return
}

func (lt *LockedTree) Len() (ln int) {
	lt.m.RLock()
	ln = lt.t.Len()
	lt.m.RUnlock()
	return
}

// Walk
// It is *NOT* safe to modify the tree inside fn.
func (lt *LockedTree) Walk(fn WalkFn) bool {
	lt.m.RLock()
	defer lt.m.RUnlock()
	return lt.t.Walk(fn)
}

// WalkPrefix
// It is *NOT* safe to modify the tree inside fn.
func (lt *LockedTree) WalkPrefix(prefix string, fn WalkFn) bool {
	lt.m.RLock()
	defer lt.m.RUnlock()
	return lt.t.WalkPrefix(prefix, fn)
}

// WalkPath
// It is *NOT* safe to modify the tree inside fn.
func (lt *LockedTree) WalkPath(path string, fn WalkFn) bool {
	lt.m.RLock()
	defer lt.m.RUnlock()
	return lt.t.WalkPath(path, fn)
}

// WalkNearestPath
// It is *NOT* safe to modify the tree inside fn.
func (lt *LockedTree) WalkNearestPath(path string, fn WalkFn) bool {
	lt.m.RLock()
	defer lt.m.RUnlock()
	return lt.t.WalkNearestPath(path, fn)
}

func (lt *LockedTree) ToMap() map[string]interface{} {
	lt.m.RLock()
	out := make(map[string]interface{}, lt.t.size)
	lt.t.Walk(func(k string, v interface{}) bool {
		out[k] = v
		return false
	})
	lt.m.RUnlock()
	return out
}
