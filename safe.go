//go:build go1.18
// +build go1.18

package radix

import (
	"io"
	"sync"
)

// Safe returns a concurrency-safe version of the tree.
// it is *NOT* safe to modify the original tree after this call.
func (t *Tree[VT]) Safe() *SafeTree[VT] {
	return &SafeTree[VT]{
		t: *t,
	}
}

// NewSafe returns a concurrency-safe radix tree.
func NewSafe[VT any](caseInsensitive bool) *SafeTree[VT] {
	var lt SafeTree[VT]
	lt.t.fold = caseInsensitive
	return &lt
}

// SafeTree is a concurrency-safe radix tree.
type SafeTree[VT any] struct {
	t Tree[VT]
	m sync.RWMutex
}

// Update aquires a rw lock and calls fn with the underlying tree
func (lt *SafeTree[VT]) Update(fn func(t *Tree[VT])) {
	lt.m.Lock()
	defer lt.m.Unlock()
	fn(&lt.t)
}

func (lt *SafeTree[VT]) Set(key string, value VT) (old VT, found bool) {
	lt.m.Lock()
	old, found = lt.t.Set(key, value)
	lt.m.Unlock()
	return
}

func (lt *SafeTree[VT]) Delete(key string) (old VT, found bool) {
	lt.m.Lock()
	old, found = lt.t.Delete(key)
	lt.m.Unlock()
	return
}

func (lt *SafeTree[VT]) DeletePrefix(prefix string) (count int) {
	lt.m.Lock()
	count = lt.t.DeletePrefix(prefix)
	lt.m.Unlock()
	return
}

func (lt *SafeTree[VT]) Get(key string) (val VT, found bool) {
	lt.m.RLock()
	val, found = lt.t.Get(key)
	lt.m.RUnlock()
	return
}

func (lt *SafeTree[VT]) LongestPrefix(prefix string) (key string, val VT, found bool) {
	lt.m.RLock()
	key, val, found = lt.t.LongestPrefix(prefix)
	lt.m.RUnlock()
	return
}

func (lt *SafeTree[VT]) Minimum() (key string, val VT, found bool) {
	lt.m.RLock()
	key, val, found = lt.t.Minimum()
	lt.m.RUnlock()
	return
}

func (lt *SafeTree[VT]) Maximum() (key string, val VT, found bool) {
	lt.m.RLock()
	key, val, found = lt.t.Maximum()
	lt.m.RUnlock()
	return
}

func (lt *SafeTree[VT]) Len() (ln int) {
	lt.m.RLock()
	ln = lt.t.Len()
	lt.m.RUnlock()
	return
}

// Walk
// It is *NOT* safe to modify the tree inside fn.
func (lt *SafeTree[VT]) Walk(fn WalkFn[VT]) bool {
	lt.m.RLock()
	defer lt.m.RUnlock()
	return lt.t.Walk(fn)
}

// WalkPrefix
// It is *NOT* safe to modify the tree inside fn.
func (lt *SafeTree[VT]) WalkPrefix(prefix string, fn WalkFn[VT]) bool {
	lt.m.RLock()
	defer lt.m.RUnlock()
	return lt.t.WalkPrefix(prefix, fn)
}

// WalkPath
// It is *NOT* safe to modify the tree inside fn.
func (lt *SafeTree[VT]) WalkPath(path string, fn WalkFn[VT]) bool {
	lt.m.RLock()
	defer lt.m.RUnlock()
	return lt.t.WalkPath(path, fn)
}

// WalkNearestPath
// It is *NOT* safe to modify the tree inside fn.
func (lt *SafeTree[VT]) WalkNearestPath(path string, fn WalkFn[VT]) bool {
	lt.m.RLock()
	defer lt.m.RUnlock()
	return lt.t.WalkNearestPath(path, fn)
}

func (lt *SafeTree[VT]) MergeMap(m map[string]VT) {
	lt.m.Lock()
	lt.t.MergeMap(m)
	lt.m.Unlock()
}

func (lt *SafeTree[VT]) MergeTree(t *Tree[VT]) {
	lt.m.Lock()
	lt.t.Merge(t)
	lt.m.Unlock()
}

func (lt *SafeTree[VT]) Merge(ot *SafeTree[VT]) {
	ot.m.RLock()
	defer ot.m.RUnlock()
	lt.MergeTree(&ot.t)
}

func (lt *SafeTree[VT]) ToMap() map[string]VT {
	lt.m.RLock()
	out := make(map[string]VT, lt.t.size)
	lt.t.Walk(func(k string, v VT) bool {
		out[k] = v
		return false
	})
	lt.m.RUnlock()
	return out
}

func (lt *SafeTree[VT]) DumpTo(w io.Writer, asJSON bool) error {
	lt.m.RLock()
	defer lt.m.Unlock()
	return lt.t.DumpTo(w, asJSON)
}

func (lt *SafeTree[VT]) Dump(asJSON bool) string {
	lt.m.RLock()
	defer lt.m.Unlock()
	return lt.t.Dump(asJSON)
}
