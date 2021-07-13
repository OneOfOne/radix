# radix
[![Go Reference](https://pkg.go.dev/badge/go.oneofone.dev/radix.svg)](https://pkg.go.dev/go.oneofone.dev/radix)
![test status](https://github.com/OneOfOne/radix/actions/workflows/test.yml/badge.svg)
[![Coverall](https://coveralls.io/repos/github/OneOfOne/radix/badge.svg?branch=main)](https://coveralls.io/github/OneOfOne/radix)

Implements a [radix tree](http://en.wikipedia.org/wiki/Radix_tree), optimized for sparse nodes.

This is a hard-fork based of [armon/go-radix](https://github.com/armon/go-radix), with some additions.

# Features

* O(k) operations. In many cases, this can be faster than a hash table since the hash function is an O(k) operation, and hash tables have very poor cache locality.
* Minimum / Maximum value lookups
* Ordered iteration

## New
* Can walk the tree from the nearest path
* Includes a thread-safe version.
* Unicode safe.
* Case-insensitive matching support.
* Go Generics support.

# TODO

* Marshaling/Unmarshaling support (currently you can dump to json).
* More optimizations.
* More benchmarks.

# Usage

```go
// go get go.oneofone.dev/radix@main
// Create a case-insensistive tree
r := radix.New[int](true)

// or thread-safe version
// t := NewSafe[int](true)
t.Set("foo", 1)
t.Set("bar", 2)
t.Set("foobar", 2)

// Find the longest prefix match
m, _, _ := t.LongestPrefix("fOoZiP")
if m != "foo" {
    panic("should be foo")
}
```

### Generating a typed version for go < 1.18 (requires perl and posix shell)

```sh
# outside a module
$ go get go.oneofone.dev/radix
$ sh $(go env GOPATH)/src/go.oneofone.dev/radix/gen.sh "interface{}" # or "string" or "[]pkg.SomeStruct"
```

# Using the generic version

Install Go 1.18 (<small>[dev.typeparams](https://github.com/golang/go/tree/dev.typeparams)</small>)

```sh
$ go get golang.org/dl/gotip
$ gotip download dev.typeparams
$ gotip get go.oneofone.dev/radix@main # note that the go proxy does not support go 1.18 yet.
```

# Contributions

Anyone is more than welcome to open pull requests provided they adhere to the Go community [code of conduct](https://golang.org/conduct).

It's recommended that any modifications should be on the [generic](radix.go),
however if that's not possible, I'll port any changes from the [compact](radix_go117.go).

# License

[MIT](LICENSE)
