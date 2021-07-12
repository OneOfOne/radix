radix
[![Go Reference](https://pkg.go.dev/badge/go.oneofone.dev/radix.svg)](https://pkg.go.dev/go.oneofone.dev/radix)
[![Coverall](https://coveralls.io/repos/github/OneOfOne/radix/badge.svg?branch=main)](https://coveralls.io/github/OneOfOne/radix)
=========

`radix` implements a [radix tree](http://en.wikipedia.org/wiki/Radix_tree).
The package only provides a single `Tree` implementation, optimized for sparse nodes.

Based on [armon/go-radix](github.com/armon/go-radix), with additional optimizations and a generic version.

As a radix tree, it provides the following:
* O(k) operations. In many cases, this can be faster than a hash table since the hash function is an O(k) operation, and hash tables have very poor cache locality.
* Minimum / Maximum value lookups
* Ordered iteration
* Can walk the tree from the nearest path
* *optional* case-insensitive matching
* Concurrency-safe version
* UTF-8 safe.
* Generic

Example
=======

```go
// go get go.oneofone.dev/radix
// Create a tree
var t radix.Tree[int]
t.CaseInsensitive = true
// or thread-safe version
// var t radix.LockedTree[int]
t.Set("foo", 1)
t.Set("bar", 2)
t.Set("foobar", 2)

// Find the longest prefix match
m, _, _ := t.LongestPrefix("fOoZiP")
if m != "foo" {
    panic("should be foo")
}
```

Install Go with generics support (<small>[dev.typeparams](https://github.com/golang/go/tree/dev.typeparams)</small>)
======

```sh
$ go get golang.org/dl/gotip
$ gotip download dev.typeparams
```
