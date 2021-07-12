//go:build !go1.18
// +build !go1.18

package radix

import (
	"io"
	"sync"
)

// Safe returns a concurrency-safe version of the tree.
// it is *NOT* safe to modify the original tree after this call.
func (t *Tree) Safe() *SafeTree {
	return &SafeTree{
		t: *t,
	}
}

// NewSafe returns a concurrency-safe radix tree.
func NewSafe(caseInsensitive bool) *SafeTree {
	var lt SafeTree
	lt.t.fold = caseInsensitive
	return &lt
}

// SafeTree is a concurrency-safe radix tree.
type SafeTree struct {
	t Tree
	m sync.RWMutex
}

// Update aquires a rw lock and calls fn with the underlying tree
func (lt *SafeTree) Update(fn func(t *Tree)) {
	lt.m.Lock()
	defer lt.m.Unlock()
	fn(&lt.t)
}

func (lt *SafeTree) Set(key string, value interface{}) (old interface{}, found bool) {
	lt.m.Lock()
	old, found = lt.t.Set(key, value)
	lt.m.Unlock()
	return
}

func (lt *SafeTree) Delete(key string) (old interface{}, found bool) {
	lt.m.Lock()
	old, found = lt.t.Delete(key)
	lt.m.Unlock()
	return
}

func (lt *SafeTree) DeletePrefix(prefix string) (count int) {
	lt.m.Lock()
	count = lt.t.DeletePrefix(prefix)
	lt.m.Unlock()
	return
}

func (lt *SafeTree) Get(key string) (val interface{}, found bool) {
	lt.m.RLock()
	val, found = lt.t.Get(key)
	lt.m.RUnlock()
	return
}

func (lt *SafeTree) LongestPrefix(prefix string) (key string, val interface{}, found bool) {
	lt.m.RLock()
	key, val, found = lt.t.LongestPrefix(prefix)
	lt.m.RUnlock()
	return
}

func (lt *SafeTree) Minimum() (key string, val interface{}, found bool) {
	lt.m.RLock()
	key, val, found = lt.t.Minimum()
	lt.m.RUnlock()
	return
}

func (lt *SafeTree) Maximum() (key string, val interface{}, found bool) {
	lt.m.RLock()
	key, val, found = lt.t.Maximum()
	lt.m.RUnlock()
	return
}

func (lt *SafeTree) Len() (ln int) {
	lt.m.RLock()
	ln = lt.t.Len()
	lt.m.RUnlock()
	return
}

// Walk
// It is *NOT* safe to modify the tree inside fn.
func (lt *SafeTree) Walk(fn WalkFn) bool {
	lt.m.RLock()
	defer lt.m.RUnlock()
	return lt.t.Walk(fn)
}

// WalkPrefix
// It is *NOT* safe to modify the tree inside fn.
func (lt *SafeTree) WalkPrefix(prefix string, fn WalkFn) bool {
	lt.m.RLock()
	defer lt.m.RUnlock()
	return lt.t.WalkPrefix(prefix, fn)
}

// WalkPath
// It is *NOT* safe to modify the tree inside fn.
func (lt *SafeTree) WalkPath(path string, fn WalkFn) bool {
	lt.m.RLock()
	defer lt.m.RUnlock()
	return lt.t.WalkPath(path, fn)
}

// WalkNearestPath
// It is *NOT* safe to modify the tree inside fn.
func (lt *SafeTree) WalkNearestPath(path string, fn WalkFn) bool {
	lt.m.RLock()
	defer lt.m.RUnlock()
	return lt.t.WalkNearestPath(path, fn)
}

func (lt *SafeTree) MergeMap(m map[string]interface{}) {
	lt.m.Lock()
	lt.t.MergeMap(m)
	lt.m.Unlock()
}

func (lt *SafeTree) MergeTree(t *Tree) {
	lt.m.Lock()
	lt.t.Merge(t)
	lt.m.Unlock()
}

func (lt *SafeTree) Merge(ot *SafeTree) {
	ot.m.RLock()
	defer ot.m.RUnlock()
	lt.MergeTree(&ot.t)
}

func (lt *SafeTree) ToMap() map[string]interface{} {
	lt.m.RLock()
	out := make(map[string]interface{}, lt.t.size)
	lt.t.Walk(func(k string, v interface{}) bool {
		out[k] = v
		return false
	})
	lt.m.RUnlock()
	return out
}

func (lt *SafeTree) DumpTo(w io.Writer, asJSON bool) error {
	lt.m.RLock()
	defer lt.m.Unlock()
	return lt.t.DumpTo(w, asJSON)
}

func (lt *SafeTree) Dump(asJSON bool) string {
	lt.m.RLock()
	defer lt.m.Unlock()
	return lt.t.Dump(asJSON)
}
