//go:build go1.18
// +build go1.18

//go:generate ./gen.sh "interface{}"

package radix

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"
)

// WalkFn is used when walking the tree. Takes a
// key and value, returning if iteration should
// be terminated.
type WalkFn[VT any] func(s string, v VT) bool

// edge is used to represent an edge node
type edge[VT any] struct {
	Node  *node[VT] `json:"node,omitempty"`
	Label rune      `json:"label"`
}

// leafNode is used to represent a value
type leafNode[VT any] struct {
	Key   string `json:"key,omitempty"`
	Value VT     `json:"value,omitempty"`
}

type node[VT any] struct {
	// Leaf is used to store possible Leaf
	Leaf *leafNode[VT] `json:"leaf,omitempty"`

	// Prefix is the common Prefix we ignore
	Prefix string `json:"prefix,omitempty"`

	// Edges should be stored in-order for iteration.
	// We avoid a fully materialized slice to save memory,
	// since in most cases we expect to be sparse
	Edges []edge[VT] `json:"edges,omitempty"`
}

func (n *node[VT]) dump(w io.Writer, indent string) (err error) {
	if n == nil {
		_, err = fmt.Fprintf(w, "%s<Node />", indent)
		return
	}
	fmt.Fprintf(w, "%s<Node prefix=%q", indent, n.Prefix)
	if n.isLeafInTheWind() {
		fmt.Fprintf(w, " key=%q value=[%v]", n.Leaf.Key, n.Leaf.Value)
	}
	if len(n.Edges) == 0 {
		_, err = fmt.Fprintln(w, "/>")
		return
	} else {
		_, err = fmt.Fprintln(w, ">")
	}
	indent = indent + "\t"
	for _, e := range n.Edges {
		fmt.Fprintf(w, "%s<Edge label=%q rune=%d>\n", indent, e.Label, e.Label)
		if err = e.Node.dump(w, indent+"\t"); err != nil {
			return
		}
		fmt.Fprintf(w, "%s</Edge label=%q rune=%d>\n", indent, e.Label, e.Label)
	}
	_, err = fmt.Fprintf(w, "%s</Node prefix=%q>\n", indent[:len(indent)-1], n.Prefix)
	return
}

func (n *node[VT]) isLeafInTheWind() bool {
	return n.Leaf != nil
}

func (n *node[VT]) addEdge(e edge[VT], fold bool) {
	if fold {
		e.Label = unicode.ToLower(e.Label)
	}

	num := len(n.Edges)
	idx := sort.Search(num, func(i int) bool {
		return n.Edges[i].Label >= e.Label
	})

	n.Edges = append(n.Edges, edge[VT]{})
	copy(n.Edges[idx+1:], n.Edges[idx:])
	n.Edges[idx] = e
}

func (n *node[VT]) updateEdge(label rune, node *node[VT], fold bool) {
	if fold {
		label = unicode.ToLower(label)
	}

	i, j := 0, len(n.Edges)
	for i < j {
		h := int(uint(i+j) >> 1)
		e := &n.Edges[h]
		if e.Label == label {
			e.Node = node
			return
		} else if e.Label < label {
			i = h + 1
		} else {
			j = h
		}
	}

	panic("replacing missing edge")
}

func (n *node[VT]) getEdge(label rune, fold bool) *node[VT] {
	if fold {
		label = unicode.ToLower(label)
	}

	i, j := 0, len(n.Edges)
	for i < j {
		h := int(uint(i+j) >> 1)
		e := n.Edges[h]
		if e.Label == label {
			return e.Node
		} else if e.Label < label {
			i = h + 1
		} else {
			j = h
		}
	}

	return nil
}

func (n *node[VT]) delEdge(label rune, fold bool) {
	if fold {
		label = unicode.ToLower(label)
	}
	num := len(n.Edges)
	idx := sort.Search(num, func(i int) bool {
		return n.Edges[i].Label >= label
	})
	if idx < num && n.Edges[idx].Label == label {
		copy(n.Edges[idx:], n.Edges[idx+1:])
		n.Edges[len(n.Edges)-1] = edge[VT]{}
		n.Edges = n.Edges[:len(n.Edges)-1]
	}
}

// New returns an empty Tree.
// The same as just using `var t Tree[VT]`.
func New[VT any](caseInsensitive bool) *Tree[VT] {
	return &Tree[VT]{
		fold: caseInsensitive,
	}
}

// Tree implements a radix tree. This can be treated as a
// Dictionary abstract data type. The main advantage over
// a standard hash map is prefix-based lookups and
// ordered iteration.
// The zero value is usable.
type Tree[VT any] struct {
	root node[VT]
	size int

	// if fold is set to true, all tree operations
	// will use Unicode case-folding, which is a more general
	// form of case-insensitivity.
	fold bool

	zero VT
}

// Len is used to return the number of elements in the tree.
func (t *Tree[VT]) Len() int {
	return t.size
}

// Set is used to set a value and return the previous one if any.
func (t *Tree[VT]) Set(key string, value VT) (VT, bool) {
	var (
		parent *node[VT]
		n      = &t.root
		lcp    = longestPrefixFn(t.fold)
		search = key
		r      rune
	)

	for {
		// Handle key exhaution
		if len(search) == 0 {
			if n.isLeafInTheWind() {
				old := n.Leaf.Value
				n.Leaf.Value = value
				return old, true
			}

			n.Leaf = &leafNode[VT]{
				Key:   key,
				Value: value,
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

			e := edge[VT]{
				Label: r,
				Node: &node[VT]{
					Leaf: &leafNode[VT]{
						Key:   key,
						Value: value,
					},
					Prefix: search,
				},
			}
			parent.addEdge(e, t.fold)
			t.size++
			return t.zero, false
		}

		// Determine longest prefix of the search key on match
		commonPrefix := lcp(search, n.Prefix)
		if commonPrefix == len(n.Prefix) {
			search = search[commonPrefix:]
			continue
		}

		// Split the node
		t.size++
		child := &node[VT]{
			Prefix: search[:commonPrefix],
		}
		parent.updateEdge(r, child, t.fold)

		// Restore the existing node
		child.addEdge(edge[VT]{
			Label: nextRune(n.Prefix[commonPrefix:]),
			Node:  n,
		}, t.fold)
		n.Prefix = n.Prefix[commonPrefix:]

		// Create a new leaf node
		leaf := &leafNode[VT]{
			Key:   key,
			Value: value,
		}

		// If the new key is a subset, add to to this node
		search = search[commonPrefix:]
		if len(search) == 0 {
			child.Leaf = leaf
			return t.zero, false
		}

		r = nextRune(search)
		// Create a new edge for the node
		child.addEdge(edge[VT]{
			Label: r,
			Node: &node[VT]{
				Leaf:   leaf,
				Prefix: search,
			},
		}, t.fold)
		return t.zero, false
	}
}

// Delete is used to delete a key, returning the previous
// value and if it was deleted.
func (t *Tree[VT]) Delete(s string) (VT, bool) {
	var (
		parent *node[VT]
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
		if hp(search, n.Prefix) {
			search = search[len(n.Prefix):]
		} else {
			break
		}
	}
	return t.zero, false

DELETE:
	// Delete the leaf
	leaf := n.Leaf
	n.Leaf = nil
	t.size--

	// Check if we should delete this node from the parent
	if parent != nil && len(n.Edges) == 0 {
		parent.delEdge(label, t.fold)
	}

	// Check if we should merge this node
	if n != &t.root && len(n.Edges) == 1 {
		n.mergeChild()
	}

	// Check if we should merge the parent's other child
	if parent != nil && parent != &t.root && len(parent.Edges) == 1 && !parent.isLeafInTheWind() {
		parent.mergeChild()
	}

	return leaf.Value, true
}

// DeletePrefix is used to delete the subtree under a prefix
// Returns how many nodes were deleted.
// Use this to delete large subtrees efficiently.
func (t *Tree[VT]) DeletePrefix(s string) int {
	return t.deletePrefix(nil, &t.root, s)
}

// delete does a recursive deletion
func (t *Tree[VT]) deletePrefix(parent, n *node[VT], prefix string) int {
	// Check for key exhaustion
	hp := hasPrefixFn(t.fold)
	if len(prefix) == 0 {
		// Remove the leaf node
		subTreeSize := 0
		// recursively walk from all edges of the node to be deleted
		recursiveWalk(n, func(s string, v VT) bool {
			subTreeSize++
			return false
		})
		if n.isLeafInTheWind() {
			n.Leaf = nil
		}
		n.Edges = nil // deletes the entire subtree

		if parent != nil {
			r := nextRune(n.Prefix)
			// delete dangling edge
			parent.delEdge(r, t.fold)
		}

		// Check if we should merge the parent's other child
		if parent != nil && parent != &t.root && len(parent.Edges) == 1 && !parent.isLeafInTheWind() {
			parent.mergeChild()
		}
		t.size -= subTreeSize
		return subTreeSize
	}

	// Look for an edge
	label := nextRune(prefix)
	child := n.getEdge(label, t.fold)
	if child == nil || (!hp(child.Prefix, prefix) && !hp(prefix, child.Prefix)) {
		return 0
	}

	// Consume the search prefix
	if len(child.Prefix) > len(prefix) {
		prefix = prefix[len(prefix):]
	} else {
		prefix = prefix[len(child.Prefix):]
	}
	return t.deletePrefix(n, child, prefix)
}

func (n *node[VT]) mergeChild() {
	e := n.Edges[0]
	child := e.Node
	n.Prefix = n.Prefix + child.Prefix
	n.Leaf = child.Leaf
	n.Edges = child.Edges
}

// Get is used to lookup a specific key, returning
// the value and if it was found.
func (t *Tree[VT]) Get(s string) (VT, bool) {
	n := &t.root
	hp := hasPrefixFn(t.fold)
	search := s
	for {
		// Check for key exhaution
		if len(search) == 0 {
			if n.isLeafInTheWind() {
				return n.Leaf.Value, true
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
		if hp(search, n.Prefix) {
			search = search[len(n.Prefix):]
		} else {
			break
		}
	}
	return t.zero, false
}

// LongestPrefix is like Get, but instead of an
// exact match, it will return the longest prefix match.
func (t *Tree[VT]) LongestPrefix(s string) (string, VT, bool) {
	var (
		last   *leafNode[VT]
		n      = &t.root
		search = s
		hp     = hasPrefixFn(t.fold)
	)
	for {
		// Look for a leaf node
		if n.isLeafInTheWind() {
			last = n.Leaf
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
		if hp(search, n.Prefix) {
			search = search[len(n.Prefix):]
		} else {
			break
		}
	}
	if last != nil {
		return last.Key, last.Value, true
	}
	return "", t.zero, false
}

// Minimum is used to return the minimum value in the tree.
func (t *Tree[VT]) Minimum() (string, VT, bool) {
	n := &t.root
	for {
		if n.isLeafInTheWind() {
			return n.Leaf.Key, n.Leaf.Value, true
		}
		if len(n.Edges) > 0 {
			n = n.Edges[0].Node
		} else {
			break
		}
	}
	return "", t.zero, false
}

// Maximum is used to return the maximum value in the tree.
func (t *Tree[VT]) Maximum() (string, VT, bool) {
	n := &t.root
	for {
		if num := len(n.Edges); num > 0 {
			n = n.Edges[num-1].Node
			continue
		}
		if n.isLeafInTheWind() {
			return n.Leaf.Key, n.Leaf.Value, true
		}
		break
	}
	return "", t.zero, false
}

// Walk is used to walk the tree.
func (t *Tree[VT]) Walk(fn WalkFn[VT]) bool {
	return recursiveWalk(&t.root, fn)
}

// WalkPrefix is used to walk the tree under a prefix.
func (t *Tree[VT]) WalkPrefix(prefix string, fn WalkFn[VT]) bool {
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
		if hp(search, n.Prefix) {
			search = search[len(n.Prefix):]
		} else if hp(n.Prefix, search) {
			// Child may be under our search prefix
			return recursiveWalk(n, fn)
		} else {
			break
		}
	}

	return false
}

// WalkNearestPath is like WalkPath but will start at the longest common prefix.
func (t *Tree[VT]) WalkNearestPath(path string, fn WalkFn[VT]) bool {
	var (
		last   *node[VT]
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
		if hp(search, n.Prefix) {
			search = search[len(n.Prefix):]
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
func (t *Tree[VT]) WalkPath(path string, fn WalkFn[VT]) bool {
	n := &t.root
	hp := hasPrefixFn(t.fold)
	search := path
	for {
		// Visit the leaf values if any
		if n.Leaf != nil && fn(n.Leaf.Key, n.Leaf.Value) {
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
		if hp(search, n.Prefix) {
			search = search[len(n.Prefix):]
		} else {
			break
		}
	}
	return false
}

func (t *Tree[VT]) MergeMap(m map[string]VT) *Tree[VT] {
	for k, v := range m {
		t.Set(k, v)
	}
	return t
}

func (t *Tree[VT]) Merge(ot *Tree[VT]) *Tree[VT] {
	ot.Walk(func(k string, v VT) bool {
		t.Set(k, v)
		return false
	})
	return t
}

// ToMap is used to walk the tree and convert it into a map.
func (t *Tree[VT]) ToMap() map[string]VT {
	out := make(map[string]VT, t.size)
	t.Walk(func(k string, v VT) bool {
		out[k] = v
		return false
	})
	return out
}

func (t *Tree[VT]) DumpTo(w io.Writer, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "\t")
		return enc.Encode(t.root.Edges)
	}
	return t.root.dump(w, "")
}

func (t *Tree[VT]) Dump(asJSON bool) string {
	var buf strings.Builder
	t.DumpTo(&buf, asJSON)
	return buf.String()
}

// recursiveWalk is used to do a pre-order walk of a node
// recursively. Returns true if the walk should be aborted
func recursiveWalk[VT any](n *node[VT], fn WalkFn[VT]) bool {
	if n == nil {
		return false
	}
	// Visit the leaf values if any
	if n.Leaf != nil && fn(n.Leaf.Key, n.Leaf.Value) {
		return true
	}

	// Recurse on the children
	for _, e := range n.Edges {
		if recursiveWalk[VT](e.Node, fn) {
			return true
		}
	}
	return false
}
