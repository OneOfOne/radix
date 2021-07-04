radix [![Go Reference](https://pkg.go.dev/badge/go.oneofone.dev/radix.svg)](https://pkg.go.dev/go.oneofone.dev/radix)
=========

`radix` implements a [radix tree](http://en.wikipedia.org/wiki/Radix_tree).
The package only provides a single `Tree` implementation, optimized for sparse nodes.

Based on [armon/go-radix](github.com/armon/go-radix), with additional optimizations and a generic version.

As a radix tree, it provides the following:
* O(k) operations. In many cases, this can be faster than a hash table since the hash function is an O(k) operation, and hash tables have very poor cache locality.
* Minimum / Maximum value lookups
* Ordered iteration
* Can walk the tree from the nearest path
* Generic version for go1.18+

Example
=======

Below is a simple example of usage

```go
// Create a tree
r := radix.New()
r.Insert("foo", 1)
r.Insert("bar", 2)
r.Insert("foobar", 2)

// Find the longest prefix match
m, _, _ := r.LongestPrefix("foozip")
if m != "foo" {
    panic("should be foo")
}
```
