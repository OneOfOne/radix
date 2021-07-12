//go:build !go1.18
// +build !go1.18

package radix

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// WalkFn is used when walking the tree. Takes a
// key and value, returning if iteration should
// be terminated.
type WalkFn func(s string, v interface{}) bool

// edge is used to represent an edge node
type edge struct {
	node  *node
	label rune
}

type edges []edge

func (e edges) Len() int {
	return len(e)
}

func (e edges) Less(i, j int) bool {
	return e[i].label < e[j].label
}

func (e edges) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

// leafNode is used to represent a value
type leafNode struct {
	key string
	val interface{}
}

type node struct {
	// leaf is used to store possible leaf
	leaf *leafNode

	// prefix is the common prefix we ignore
	prefix string

	// Edges should be stored in-order for iteration.
	// We avoid a fully materialized slice to save memory,
	// since in most cases we expect to be sparse
	edges edges
}

func (n *node) dump(w io.Writer, indent string) (err error) {
	if n == nil {
		_, err = fmt.Fprintf(w, "%s<Node />", indent)
		return
	}
	fmt.Fprintf(w, "%s<Node prefix=%q", indent, n.prefix)
	if n.isLeafInTheWind() {
		fmt.Fprintf(w, " key=%q value=[%v]", n.leaf.key, n.leaf.val)
	}
	if len(n.edges) == 0 {
		_, err = fmt.Fprintln(w, "/>")
		return
	} else {
		_, err = fmt.Fprintln(w, ">")
	}
	indent = indent + "\t"
	for _, e := range n.edges {
		fmt.Fprintf(w, "%s<Edge label=%q rune=%d>\n", indent, e.label, e.label)
		if err = e.node.dump(w, indent+"\t"); err != nil {
			return
		}
		fmt.Fprintf(w, "%s</Edge label=%q rune=%d>\n", indent, e.label, e.label)
	}
	_, err = fmt.Fprintf(w, "%s</Node prefix=%q>\n", indent[:len(indent)-1], n.prefix)
	return
}

func (n *node) isLeafInTheWind() bool {
	return n.leaf != nil
}

func (n *node) addEdge(e edge, fold bool) {
	if fold {
		e.label = toLower(e.label)
	}
	num := len(n.edges)
	idx := sort.Search(num, func(i int) bool {
		return n.edges[i].label >= e.label
	})
	n.edges = append(n.edges, edge{})
	copy(n.edges[idx+1:], n.edges[idx:])
	n.edges[idx] = e
}

func (n *node) updateEdge(label rune, node *node, fold bool) {
	if fold {
		label = toLower(label)
	}
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

func (n *node) getEdge(label rune, fold bool) *node {
	if fold {
		label = toLower(label)
	}
	num := len(n.edges)
	idx := sort.Search(num, func(i int) bool {
		return n.edges[i].label >= label
	})
	if idx < num && n.edges[idx].label == label {
		return n.edges[idx].node
	}
	return nil
}

func (n *node) delEdge(label rune, fold bool) {
	if fold {
		label = toLower(label)
	}
	num := len(n.edges)
	idx := sort.Search(num, func(i int) bool {
		return n.edges[i].label >= label
	})
	if idx < num && n.edges[idx].label == label {
		copy(n.edges[idx:], n.edges[idx+1:])
		n.edges[len(n.edges)-1] = edge{}
		n.edges = n.edges[:len(n.edges)-1]
	}
}

// New returns an empty Tree.
// The same as just using `var t Tree`.
func New(caseInsensitive bool) *Tree {
	return &Tree{
		fold: caseInsensitive,
	}
}

// Tree implements a radix tree. This can be treated as a
// Dictionary abstract data type. The main advantage over
// a standard hash map is prefix-based lookups and
// ordered iteration.
// The zero value is usable.
type Tree struct {
	root node
	size int

	// if fold is set to true, all tree operations
	// will use Unicode case-folding, which is a more general
	// form of case-insensitivity.
	fold bool

	zero interface{}
}

// Len is used to return the number of elements in the tree.
func (t *Tree) Len() int {
	return t.size
}

// Set is used to add a newentry or update
// an existing entry. Returns if updated.
func (t *Tree) Set(key string, value interface{}) (interface{}, bool) {
	var (
		parent *node
		n      = &t.root
		lcp    = longestPrefixFn(t.fold)
		search = key
		r      rune
	)
	for {
		// Handle key exhaution
		if len(search) == 0 {
			if n.isLeafInTheWind() {
				old := n.leaf.val
				n.leaf.val = value
				return old, true
			}

			n.leaf = &leafNode{
				key: key,
				val: value,
			}
			t.size++
			return t.zero, false
		}

		// Look for the edge
		parent = n
		r = nextRune(search)
		n = n.getEdge(r, t.fold)

		// No edge, create one
		if n == nil {
			e := edge{
				label: r,
				node: &node{
					leaf: &leafNode{
						key: key,
						val: value,
					},
					prefix: search,
				},
			}
			parent.addEdge(e, t.fold)
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
		child := &node{
			prefix: search[:commonPrefix],
		}
		parent.updateEdge(r, child, t.fold)

		// Restore the existing node
		child.addEdge(edge{
			label: nextRune(n.prefix[commonPrefix:]),
			node:  n,
		}, t.fold)
		n.prefix = n.prefix[commonPrefix:]

		// Create a new leaf node
		leaf := &leafNode{
			key: key,
			val: value,
		}

		// If the new key is a subset, add to to this node
		search = search[commonPrefix:]
		if len(search) == 0 {
			child.leaf = leaf
			return t.zero, false
		}
		r = nextRune(search)
		// Create a new edge for the node
		child.addEdge(edge{
			label: r,
			node: &node{
				leaf:   leaf,
				prefix: search,
			},
		}, t.fold)
		return t.zero, false
	}
}

// Delete is used to delete a key, returning the previous
// value and if it was deleted.
func (t *Tree) Delete(s string) (interface{}, bool) {
	var (
		parent *node
		label  rune
		n      = &t.root
		search = s
		hp     = hasPrefixFn(t.fold)
	)

	for {
		// Check for key exhaution
		if len(search) == 0 {
			if !n.isLeafInTheWind() {
				break
			}
			goto DELETE
		}

		// Look for an edge
		parent = n
		label = nextRune(search)
		n = n.getEdge(label, t.fold)
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
		parent.delEdge(label, t.fold)
	}

	// Check if we should merge this node
	if n != &t.root && len(n.edges) == 1 {
		n.mergeChild()
	}

	// Check if we should merge the parent's other child
	if parent != nil && parent != &t.root && len(parent.edges) == 1 && !parent.isLeafInTheWind() {
		parent.mergeChild()
	}

	return leaf.val, true
}

// DeletePrefix is used to delete the subtree under a prefix
// Returns how many nodes were deleted.
// Use this to delete large subtrees efficiently.
func (t *Tree) DeletePrefix(s string) int {
	return t.deletePrefix(nil, &t.root, s)
}

// delete does a recursive deletion
func (t *Tree) deletePrefix(parent, n *node, prefix string) int {
	// Check for key exhaustion
	hp := hasPrefixFn(t.fold)
	if len(prefix) == 0 {
		// Remove the leaf node
		subTreeSize := 0
		// recursively walk from all edges of the node to be deleted
		recursiveWalk(n, func(s string, v interface{}) bool {
			subTreeSize++
			return false
		})
		if n.isLeafInTheWind() {
			n.leaf = nil
		}
		n.edges = nil // deletes the entire subtree

		if parent != nil {
			r := nextRune(n.prefix)
			// delete dangling edge
			parent.delEdge(r, t.fold)
		}

		// Check if we should merge the parent's other child
		if parent != nil && parent != &t.root && len(parent.edges) == 1 && !parent.isLeafInTheWind() {
			parent.mergeChild()
		}
		t.size -= subTreeSize
		return subTreeSize
	}

	// Look for an edge
	label := nextRune(prefix)
	child := n.getEdge(label, t.fold)
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

func (n *node) mergeChild() {
	e := n.edges[0]
	child := e.node
	n.prefix = n.prefix + child.prefix
	n.leaf = child.leaf
	n.edges = child.edges
}

// Get is used to lookup a specific key, returning
// the value and if it was found.
func (t *Tree) Get(s string) (interface{}, bool) {
	n := &t.root
	hp := hasPrefixFn(t.fold)
	search := s
	for {
		// Check for key exhaution
		if len(search) == 0 {
			if n.isLeafInTheWind() {
				return n.leaf.val, true
			}
			break
		}

		// Look for an edge
		r := nextRune(search)
		n = n.getEdge(r, t.fold)
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
func (t *Tree) LongestPrefix(s string) (string, interface{}, bool) {
	var (
		last   *leafNode
		n      = &t.root
		search = s
		hp     = hasPrefixFn(t.fold)
	)
	for {
		// Look for a leaf node
		if n.isLeafInTheWind() {
			last = n.leaf
		}

		// Check for key exhaution
		if len(search) == 0 {
			break
		}

		// Look for an edge
		r := nextRune(search)
		n = n.getEdge(r, t.fold)
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
func (t *Tree) Minimum() (string, interface{}, bool) {
	n := &t.root
	for {
		if n.isLeafInTheWind() {
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
func (t *Tree) Maximum() (string, interface{}, bool) {
	n := &t.root
	for {
		if num := len(n.edges); num > 0 {
			n = n.edges[num-1].node
			continue
		}
		if n.isLeafInTheWind() {
			return n.leaf.key, n.leaf.val, true
		}
		break
	}
	return "", t.zero, false
}

// Walk is used to walk the tree.
func (t *Tree) Walk(fn WalkFn) bool {
	return recursiveWalk(&t.root, fn)
}

// WalkPrefix is used to walk the tree under a prefix.
func (t *Tree) WalkPrefix(prefix string, fn WalkFn) bool {
	n := &t.root
	hp := hasPrefixFn(t.fold)
	search := prefix
	for {
		// Check for key exhaution
		if len(search) == 0 {
			return recursiveWalk(n, fn)
		}

		// Look for an edge
		r := nextRune(search)
		n = n.getEdge(r, t.fold)
		if n == nil {
			break
		}

		// Consume the search prefix
		if hp(search, n.prefix) {
			search = search[len(n.prefix):]
		} else if hp(n.prefix, search) {
			// Child may be under our search prefix
			return recursiveWalk(n, fn)
		} else {
			break
		}
	}

	return false
}

// WalkNearestPath is like WalkPath but will start at the longest common prefix.
func (t *Tree) WalkNearestPath(path string, fn WalkFn) bool {
	var (
		last   *node
		n      = &t.root
		search = path
		hp     = hasPrefixFn(t.fold)
	)

	for {
		// Look for a leaf node
		if n.isLeafInTheWind() {
			last = n
		}

		// Check for key exhaution
		if len(search) == 0 {
			break
		}

		// Look for an edge
		n = n.getEdge(nextRune(search), t.fold)

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
		return recursiveWalk(last, fn)
	}

	return false
}

// WalkPath is used to walk the tree, but only visiting nodes
// from the root down to a given leaf. Where WalkPrefix walks
// all the entries *under* the given prefix, this walks the
// entries *above* the given prefix.
func (t *Tree) WalkPath(path string, fn WalkFn) bool {
	n := &t.root
	hp := hasPrefixFn(t.fold)
	search := path
	for {
		// Visit the leaf values if any
		if n.leaf != nil && fn(n.leaf.key, n.leaf.val) {
			return true
		}

		// Check for key exhaution
		if len(search) == 0 {
			return false
		}

		// Look for an edge
		r := nextRune(search)
		n = n.getEdge(r, t.fold)
		if n == nil {
			return false
		}

		// Consume the search prefix
		if hp(search, n.prefix) {
			search = search[len(n.prefix):]
		} else {
			break
		}
	}
	return false
}

func (t *Tree) MergeMap(m map[string]interface{}) *Tree {
	for k, v := range m {
		t.Set(k, v)
	}
	return t
}

func (t *Tree) Merge(ot *Tree) *Tree {
	ot.Walk(func(k string, v interface{}) bool {
		t.Set(k, v)
		return false
	})
	return t
}

// ToMap is used to walk the tree and convert it into a map.
func (t *Tree) ToMap() map[string]interface{} {
	out := make(map[string]interface{}, t.size)
	t.Walk(func(k string, v interface{}) bool {
		out[k] = v
		return false
	})
	return out
}

func (t *Tree) Dump() string {
	var buf strings.Builder
	t.DumpTo(&buf)
	return buf.String()
}

func (t *Tree) DumpTo(w io.Writer) error {
	return t.root.dump(w, "")
}

// recursiveWalk is used to do a pre-order walk of a node
// recursively. Returns true if the walk should be aborted
func recursiveWalk(n *node, fn WalkFn) bool {
	if n == nil {
		return false
	}
	// Visit the leaf values if any
	if n.leaf != nil && fn(n.leaf.key, n.leaf.val) {
		return true
	}

	// Recurse on the children
	for _, e := range n.edges {
		if recursiveWalk(e.node, fn) {
			return true
		}
	}
	return false
}
