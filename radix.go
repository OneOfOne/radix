//go:build go1.18
// +build go1.18

//go:generate ./gen.sh "interface{}"

package radix

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// WalkFn is used when walking the tree. Takes a
// key and value, returning if iteration should
// be terminated.
type WalkFn[VT any] func(s string, v VT) bool

// edge is used to represent an edge node
type edge[VT any] struct {
	node  *node[VT]
	label byte
}

type edges[VT any] []edge[VT]

func (e edges[VT]) Len() int {
	return len(e)
}

func (e edges[VT]) Less(i, j int) bool {
	return e[i].label < e[j].label
}

func (e edges[VT]) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

// leafNode is used to represent a value
type leafNode[VT any] struct {
	key string
	val VT
}

type node[VT any] struct {
	// leaf is used to store possible leaf
	leaf *leafNode[VT]

	// prefix is the common prefix we ignore
	prefix string

	// Edges should be stored in-order for iteration.
	// We avoid a fully materialized slice to save memory,
	// since in most cases we expect to be sparse
	edges []edge[VT]
}

func (n *node[VT]) isLeaf() bool {
	return n.leaf != nil
}

func (n *node[VT]) addEdge(e edge[VT]) {
	num := len(n.edges)
	idx := sort.Search(num, func(i int) bool {
		return n.edges[i].label >= e.label
	})

	n.edges = append(n.edges, edge[VT]{})
	copy(n.edges[idx+1:], n.edges[idx:])
	n.edges[idx] = e
}

func (n *node[VT]) updateEdge(label byte, node *node[VT]) {
	num := len(n.edges)
	idx := sort.Search(num, func(i int) bool {
		return n.edges[i].label >= label
	})
	if idx < num && n.edges[idx].label == label {
		n.edges[idx].node = node
		return
	}
	panic("replacing missing edge")
}

func (n *node[VT]) getEdge(label byte) *node[VT] {
	num := len(n.edges)
	idx := sort.Search(num, func(i int) bool {
		return n.edges[i].label >= label
	})
	if idx < num && n.edges[idx].label == label {
		return n.edges[idx].node
	}
	return nil
}

func (n *node[VT]) delEdge(label byte) {
	num := len(n.edges)
	idx := sort.Search(num, func(i int) bool {
		return n.edges[i].label >= label
	})
	if idx < num && n.edges[idx].label == label {
		copy(n.edges[idx:], n.edges[idx+1:])
		n.edges[len(n.edges)-1] = edge[VT]{}
		n.edges = n.edges[:len(n.edges)-1]
	}
}

// Tree implements a radix tree. This can be treated as a
// Dictionary abstract data type. The main advantage over
// a standard hash map is prefix-based lookups and
// ordered iteration.
// Can be used without calling New()
type Tree[VT any] struct {
	root node[VT]
	size int

	// if CaseInsensitive is set to true, all tree operations
	// will use Unicode case-folding, which is a more general
	// form of case-insensitivity.
	CaseInsensitive bool

	zero VT
}

// New returns an empty Tree.
// The same as just using `var t Tree[VT]`.
func New[VT any]() *Tree[VT] {
	var t Tree[VT]
	return &t
}

// NewFromMap returns a new tree containing the keys
// from an existing map
func NewFromMap[VT any](m map[string]VT) *Tree[VT] {
	var t Tree[VT]
	for k, v := range m {
		t.Insert(k, v)
	}
	return &t
}

// Len is used to return the number of elements in the tree.
func (t *Tree[VT]) Len() int {
	return t.size
}

// Insert is used to add a newentry or update
// an existing entry. Returns if updated.
// *Not* safe for concurrent calls.
func (t *Tree[VT]) Insert(s string, v VT) (VT, bool) {
	var parent *node[VT]
	n := &t.root
	lcp := t.longestPrefixFn()
	search := s
	for {
		// Handle key exhaution
		if len(search) == 0 {
			if n.isLeaf() {
				old := n.leaf.val
				n.leaf.val = v
				return old, true
			}

			n.leaf = &leafNode[VT]{
				key: s,
				val: v,
			}
			t.size++
			return t.zero, false
		}

		// Look for the edge
		parent = n
		n = n.getEdge(search[0])

		// No edge, create one
		if n == nil {
			e := edge[VT]{
				label: search[0],
				node: &node[VT]{
					leaf: &leafNode[VT]{
						key: s,
						val: v,
					},
					prefix: search,
				},
			}
			parent.addEdge(e)
			t.size++
			return t.zero, false
		}

		// Determine longest prefix of the search key on match
		commonPrefix := lcp(search, n.prefix)
		if commonPrefix == len(n.prefix) {
			search = search[commonPrefix:]
			continue
		}

		// Split the node
		t.size++
		child := &node[VT]{
			prefix: search[:commonPrefix],
		}
		parent.updateEdge(search[0], child)

		// Restore the existing node
		child.addEdge(edge[VT]{
			label: n.prefix[commonPrefix],
			node:  n,
		})
		n.prefix = n.prefix[commonPrefix:]

		// Create a new leaf node
		leaf := &leafNode[VT]{
			key: s,
			val: v,
		}

		// If the new key is a subset, add to to this node
		search = search[commonPrefix:]
		if len(search) == 0 {
			child.leaf = leaf
			return t.zero, false
		}

		// Create a new edge for the node
		child.addEdge(edge[VT]{
			label: search[0],
			node: &node[VT]{
				leaf:   leaf,
				prefix: search,
			},
		})
		return t.zero, false
	}
}

// Delete is used to delete a key, returning the previous
// value and if it was deleted.
// *Not* safe for concurrent calls.
func (t *Tree[VT]) Delete(s string) (VT, bool) {
	var (
		parent *node[VT]
		label  byte
		n      = &t.root
		search = s
		hp     = t.hasPrefixFn()
	)

	for {
		// Check for key exhaution
		if len(search) == 0 {
			if !n.isLeaf() {
				break
			}
			goto DELETE
		}

		// Look for an edge
		parent = n
		label = search[0]
		n = n.getEdge(label)
		if n == nil {
			break
		}

		// Consume the search prefix
		if hp(search, n.prefix) {
			search = search[len(n.prefix):]
		} else {
			break
		}
	}
	return t.zero, false

DELETE:
	// Delete the leaf
	leaf := n.leaf
	n.leaf = nil
	t.size--

	// Check if we should delete this node from the parent
	if parent != nil && len(n.edges) == 0 {
		parent.delEdge(label)
	}

	// Check if we should merge this node
	if n != &t.root && len(n.edges) == 1 {
		n.mergeChild()
	}

	// Check if we should merge the parent's other child
	if parent != nil && parent != &t.root && len(parent.edges) == 1 && !parent.isLeaf() {
		parent.mergeChild()
	}

	return leaf.val, true
}

// DeletePrefix is used to delete the subtree under a prefix
// Returns how many nodes were deleted.
// Use this to delete large subtrees efficiently.
// *Not* safe for concurrent calls.
func (t *Tree[VT]) DeletePrefix(s string) int {
	return t.deletePrefix(nil, &t.root, s)
}

// delete does a recursive deletion
func (t *Tree[VT]) deletePrefix(parent, n *node[VT], prefix string) int {
	// Check for key exhaustion
	hp := t.hasPrefixFn()
	if len(prefix) == 0 {
		// Remove the leaf node
		subTreeSize := 0
		// recursively walk from all edges of the node to be deleted
		recursiveWalk(n, func(s string, v VT) bool {
			subTreeSize++
			return false
		})
		if n.isLeaf() {
			n.leaf = nil
		}
		n.edges = nil // deletes the entire subtree

		// Check if we should merge the parent's other child
		if parent != nil && parent != &t.root && len(parent.edges) == 1 && !parent.isLeaf() {
			parent.mergeChild()
		}
		t.size -= subTreeSize
		return subTreeSize
	}

	// Look for an edge
	label := prefix[0]
	child := n.getEdge(label)
	if child == nil || (!hp(child.prefix, prefix) && !hp(prefix, child.prefix)) {
		return 0
	}

	// Consume the search prefix
	if len(child.prefix) > len(prefix) {
		prefix = prefix[len(prefix):]
	} else {
		prefix = prefix[len(child.prefix):]
	}
	return t.deletePrefix(n, child, prefix)
}

func (n *node[VT]) mergeChild() {
	e := n.edges[0]
	child := e.node
	n.prefix = n.prefix + child.prefix
	n.leaf = child.leaf
	n.edges = child.edges
}

// Get is used to lookup a specific key, returning
// the value and if it was found.
// Safe for concurrent calls.
func (t *Tree[VT]) Get(s string) (VT, bool) {
	n := &t.root
	hp := t.hasPrefixFn()
	search := s
	for {
		// Check for key exhaution
		if len(search) == 0 {
			if n.isLeaf() {
				return n.leaf.val, true
			}
			break
		}

		// Look for an edge
		n = n.getEdge(search[0])
		if n == nil {
			break
		}

		// Consume the search prefix
		if hp(search, n.prefix) {
			search = search[len(n.prefix):]
		} else {
			break
		}
	}
	return t.zero, false
}

// LongestPrefix is like Get, but instead of an
// exact match, it will return the longest prefix match.
// Safe for concurrent calls.
func (t *Tree[VT]) LongestPrefix(s string) (string, VT, bool) {
	var (
		last   *leafNode[VT]
		n      = &t.root
		search = s
		hp     = t.hasPrefixFn()
	)
	for {
		// Look for a leaf node
		if n.isLeaf() {
			last = n.leaf
		}

		// Check for key exhaution
		if len(search) == 0 {
			break
		}

		// Look for an edge
		n = n.getEdge(search[0])
		if n == nil {
			break
		}

		// Consume the search prefix
		if hp(search, n.prefix) {
			search = search[len(n.prefix):]
		} else {
			break
		}
	}
	if last != nil {
		return last.key, last.val, true
	}
	return "", t.zero, false
}

// Minimum is used to return the minimum value in the tree.
// Safe for concurrent calls.
func (t *Tree[VT]) Minimum() (string, VT, bool) {
	n := &t.root
	for {
		if n.isLeaf() {
			return n.leaf.key, n.leaf.val, true
		}
		if len(n.edges) > 0 {
			n = n.edges[0].node
		} else {
			break
		}
	}
	return "", t.zero, false
}

// Maximum is used to return the maximum value in the tree.
// Safe for concurrent calls.
func (t *Tree[VT]) Maximum() (string, VT, bool) {
	n := &t.root
	for {
		if num := len(n.edges); num > 0 {
			n = n.edges[num-1].node
			continue
		}
		if n.isLeaf() {
			return n.leaf.key, n.leaf.val, true
		}
		break
	}
	return "", t.zero, false
}

// Walk is used to walk the tree.
// Safe for concurrent calls.
func (t *Tree[VT]) Walk(fn WalkFn[VT]) {
	recursiveWalk(&t.root, fn)
}

// WalkPrefix is used to walk the tree under a prefix.
// Safe for concurrent calls.
func (t *Tree[VT]) WalkPrefix(prefix string, fn WalkFn[VT]) {
	n := &t.root
	hp := t.hasPrefixFn()
	search := prefix
	for {
		// Check for key exhaution
		if len(search) == 0 {
			recursiveWalk(n, fn)
			return
		}

		// Look for an edge
		n = n.getEdge(search[0])
		if n == nil {
			break
		}

		// Consume the search prefix
		if hp(search, n.prefix) {
			search = search[len(n.prefix):]
		} else if hp(n.prefix, search) {
			// Child may be under our search prefix
			recursiveWalk(n, fn)
			return
		} else {
			break
		}
	}
}

// WalkNearestPath is like WalkPath but will start at the longest common prefix.
// Safe for concurrent calls.
func (t *Tree[VT]) WalkNearestPath(path string, fn WalkFn[VT]) {
	var (
		last   *node[VT]
		n      = &t.root
		search = path
		hp     = t.hasPrefixFn()
	)
	for {
		// Look for a leaf node
		if n.isLeaf() {
			last = n
		}

		// Check for key exhaution
		if len(search) == 0 {
			break
		}

		// Look for an edge
		n = n.getEdge(search[0])
		if n == nil {
			break
		}
		last = n
		// Consume the search prefix
		if hp(search, n.prefix) {
			search = search[len(n.prefix):]
		} else {
			break
		}
	}

	if last != nil {
		recursiveWalk(last, fn)
	}
}

// WalkPath is used to walk the tree, but only visiting nodes
// from the root down to a given leaf. Where WalkPrefix walks
// all the entries *under* the given prefix, this walks the
// entries *above* the given prefix.
// Safe for concurrent calls.
func (t *Tree[VT]) WalkPath(path string, fn WalkFn[VT]) {
	n := &t.root
	hp := t.hasPrefixFn()
	search := path
	for {
		// Visit the leaf values if any
		if n.leaf != nil && fn(n.leaf.key, n.leaf.val) {
			return
		}

		// Check for key exhaution
		if len(search) == 0 {
			return
		}

		// Look for an edge
		n = n.getEdge(search[0])
		if n == nil {
			return
		}

		// Consume the search prefix
		if hp(search, n.prefix) {
			search = search[len(n.prefix):]
		} else {
			break
		}
	}
}

// ToMap is used to walk the tree and convert it into a map.
// Safe for concurrent calls.
func (t *Tree[VT]) ToMap() map[string]VT {
	out := make(map[string]VT, t.size)
	t.Walk(func(k string, v VT) bool {
		out[k] = v
		return false
	})
	return out
}

func (t *Tree[VT]) hasPrefixFn() func(s, pre string) bool {
	if !t.CaseInsensitive {
		return strings.HasPrefix
	}

	return hasPrefixFold
}

func (t *Tree[VT]) longestPrefixFn() func(a, b string) int {
	if !t.CaseInsensitive {
		return longestPrefix
	}

	return longestPrefixFold
}

// recursiveWalk is used to do a pre-order walk of a node
// recursively. Returns true if the walk should be aborted
func recursiveWalk[VT any](n *node[VT], fn WalkFn[VT]) bool {
	// Visit the leaf values if any
	if n.leaf != nil && fn(n.leaf.key, n.leaf.val) {
		return true
	}

	// Recurse on the children
	for _, e := range n.edges {
		if recursiveWalk[VT](e.node, fn) {
			return true
		}
	}
	return false
}

// longestPrefix finds the length of the shared prefix
// of two strings
func longestPrefix(k1, k2 string) (i int) {
	max := len(k1)
	if l := len(k2); l < max {
		max = l
	}

	for i = 0; i < max; i++ {
		if k1[i] != k2[i] {
			return
		}
	}

	return
}

// longestPrefixFold finds the length of the shared prefix
// of two strings, ignoring case.
func longestPrefixFold(k1, k2 string) (i int) {
	if len(k1) > len(k2) {
		k1, k2 = k2, k1
	}

	var r1, r2 rune
	var sz int

	for i < len(k1) {
		if k1[i] < utf8.RuneSelf {
			if !asciiEq(k1[i], k2[i]) {
				return
			}
			i++
			continue
		}

		r1, sz = utf8.DecodeLastRuneInString(k1[i:])
		if r2, _ = utf8.DecodeRuneInString(k2[i:]); !runeEq(r1, r2) {
			return
		}
		i += sz
	}

	return i + 1
}

func hasPrefixFold(s, pre string) (_ bool) {
	if len(s) < len(pre) {
		return
	}

	var pr, sr rune
	var i, sz int

	for i < len(pre) {
		if pre[i] < utf8.RuneSelf {
			if !asciiEq(pre[i], s[i]) {
				return
			}
			i++
			continue
		}

		pr, sz = utf8.DecodeLastRuneInString(pre[i:])
		if sr, _ = utf8.DecodeRuneInString(s[i:]); !runeEq(pr, sr) {
			return
		}
		i += sz
	}

	return true
}

func runeEq(sr, tr rune) bool {
	return sr == tr || unicode.ToLower(sr) == unicode.ToLower(tr)
}

func asciiEq(sr, tr byte) bool {
	return sr == tr || asciiLower(sr) == asciiLower(tr)
}

func asciiLower(r byte) byte {
	if 'A' <= r && r <= 'Z' {
		r += 'a' - 'A'
	}
	return r
}
