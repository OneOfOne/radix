//go:build !go1.18
// +build !go1.18

package radix

import (
	crand "crypto/rand"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func ExampleRadix() {
	var t Tree
	t.fold = true
	// or thread-safe version
	// var t radix.LockedTree
	t.Set("foo", 1)
	t.Set("bar", 2)
	t.Set("foobar", 2)

	// Find the longest prefix match
	m, _, _ := t.LongestPrefix("fOoZiP")
	if m != "foo" {
		panic("should be foo")
	}
}

func TestRadix(t *testing.T) {
	a := "aaabbbbccccdddd"
	b := "aaabbbbccD"
	t.Log(a[:LongestPrefixFold(a, b)])
	t.Log(a[:LongestPrefix(a, b)])
	var min, max string
	inp := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		gen := generateUUID()
		inp[gen] = i
		if gen < min || i == 0 {
			min = gen
		}
		if gen > max || i == 0 {
			max = gen
		}
	}

	r := New(false).MergeMap(inp)
	if r.Len() != len(inp) {
		t.Fatalf("bad length: %v %v", r.Len(), len(inp))
	}

	r.Walk(func(k string, v interface{}) bool {
		// println(k)
		return false
	})

	for k, v := range inp {
		out, ok := r.Get(k)
		if !ok {
			t.Fatalf("missing key: %v", k)
		}
		if out != v {
			t.Fatalf("value mis-match: %v %v", out, v)
		}
	}

	// Check min and max
	outMin, _, _ := r.Minimum()
	if outMin != min {
		t.Fatalf("bad minimum: %v %v", outMin, min)
	}
	outMax, _, _ := r.Maximum()
	if outMax != max {
		t.Fatalf("bad maximum: %v %v", outMax, max)
	}

	for k, v := range inp {
		out, ok := r.Delete(k)
		if !ok {
			t.Fatalf("missing key: %v", k)
		}
		if out != v {
			t.Fatalf("value mis-match: %v %v", out, v)
		}
	}
	if r.Len() != 0 {
		t.Fatalf("bad length: %v", r.Len())
	}
}

func TestRoot(t *testing.T) {
	r := New(false)
	_, ok := r.Delete("")
	if ok {
		t.Fatalf("bad")
	}
	_, ok = r.Set("", true)
	if ok {
		t.Fatalf("bad")
	}
	val, ok := r.Get("")
	if !ok || val != true {
		t.Fatalf("bad: %v", val)
	}
	val, ok = r.Delete("")
	if !ok || val != true {
		t.Fatalf("bad: %v", val)
	}
}

func TestDelete(t *testing.T) {
	r := New(false)

	s := []string{"", "A", "AB"}

	for _, ss := range s {
		r.Set(ss, true)
	}

	for _, ss := range s {
		_, ok := r.Delete(ss)
		if !ok {
			t.Fatalf("bad %q", ss)
		}
	}
}

func TestDeletePrefix(t *testing.T) {
	type exp struct {
		inp        []string
		prefix     string
		out        []string
		numDeleted int
	}

	cases := []exp{
		{[]string{"", "A", "AB", "ABC", "R", "S"}, "A", []string{"", "R", "S"}, 3},
		{[]string{"", "A", "AB", "ABC", "R", "S"}, "ABC", []string{"", "A", "AB", "R", "S"}, 1},
		{[]string{"", "A", "AB", "ABC", "R", "S"}, "", []string{}, 6},
		{[]string{"", "A", "AB", "ABC", "R", "S"}, "S", []string{"", "A", "AB", "ABC", "R"}, 1},
		{[]string{"", "A", "AB", "ABC", "R", "S"}, "SS", []string{"", "A", "AB", "ABC", "R", "S"}, 0},
	}

	for _, test := range cases {
		r := New(false)
		for _, ss := range test.inp {
			r.Set(ss, true)
		}

		deleted := r.DeletePrefix(test.prefix)
		if deleted != test.numDeleted {
			t.Fatalf("Bad delete, expected %v to be deleted but got %v", test.numDeleted, deleted)
		}

		out := []string{}
		fn := func(s string, v interface{}) bool {
			out = append(out, s)
			return false
		}
		r.Walk(fn)

		if !reflect.DeepEqual(out, test.out) {
			t.Fatalf("mis-match: %v %v", out, test.out)
		}
	}
}

func TestLongestPrefix(t *testing.T) {
	r := New(false)

	keys := []string{
		"",
		"foo",
		"foobar",
		"foobarbaz",
		"foobarbazzip",
		"foozip",
	}
	for _, k := range keys {
		r.Set(k, nil)
	}
	if r.Len() != len(keys) {
		t.Fatalf("bad len: %v %v", r.Len(), len(keys))
	}

	type exp struct {
		inp string
		out string
	}
	cases := []exp{
		{"a", ""},
		{"abc", ""},
		{"fo", ""},
		{"foo", "foo"},
		{"foob", "foo"},
		{"foobar", "foobar"},
		{"foobarba", "foobar"},
		{"foobarbaz", "foobarbaz"},
		{"foobarbazzi", "foobarbaz"},
		{"foobarbazzip", "foobarbazzip"},
		{"foozi", "foo"},
		{"foozip", "foozip"},
		{"foozipzap", "foozip"},
	}
	for _, test := range cases {
		m, _, ok := r.LongestPrefix(test.inp)
		if !ok {
			t.Fatalf("no match: %v", test)
		}
		if m != test.out {
			t.Fatalf("mis-match: %v %v", m, test)
		}
	}
}

func TestWalkPrefix(t *testing.T) {
	r := New(false)

	keys := []string{
		"foobar",
		"foo/bar/baz",
		"foo/baz/bar",
		"foo/zip/zap",
		"zipzap",
	}
	for _, k := range keys {
		r.Set(k, nil)
	}
	if r.Len() != len(keys) {
		t.Fatalf("bad len: %v %v", r.Len(), len(keys))
	}

	type exp struct {
		inp string
		out []string
	}
	cases := []exp{
		{
			"f",
			[]string{"foobar", "foo/bar/baz", "foo/baz/bar", "foo/zip/zap"},
		},
		{
			"foo",
			[]string{"foobar", "foo/bar/baz", "foo/baz/bar", "foo/zip/zap"},
		},
		{
			"foob",
			[]string{"foobar"},
		},
		{
			"foo/",
			[]string{"foo/bar/baz", "foo/baz/bar", "foo/zip/zap"},
		},
		{
			"foo/b",
			[]string{"foo/bar/baz", "foo/baz/bar"},
		},
		{
			"foo/ba",
			[]string{"foo/bar/baz", "foo/baz/bar"},
		},
		{
			"foo/bar",
			[]string{"foo/bar/baz"},
		},
		{
			"foo/bar/baz",
			[]string{"foo/bar/baz"},
		},
		{
			"foo/bar/bazoo",
			[]string{},
		},
		{
			"z",
			[]string{"zipzap"},
		},
	}

	for _, test := range cases {
		out := []string{}
		fn := func(s string, v interface{}) bool {
			out = append(out, s)
			return false
		}
		r.WalkPrefix(test.inp, fn)
		sort.Strings(out)
		sort.Strings(test.out)
		if !reflect.DeepEqual(out, test.out) {
			t.Fatalf("mis-match: %v %v", out, test.out)
		}
	}
}

func TestWalkPath(t *testing.T) {
	r := New(true)

	keys := []string{
		"foo",
		"foo/bar",
		"foo/bar/baz",
		"foo/baz/bar",
		"foo/zip/zap",
		"zipzap",
		"/u/äpfêl/",
	}
	for _, k := range keys {
		r.Set(k, nil)
	}
	if r.Len() != len(keys) {
		t.Fatalf("bad len: %v %v", r.Len(), len(keys))
	}

	type exp struct {
		in      string
		out     []string
		nearest bool
	}
	cases := []exp{
		{
			"f",
			nil,
			false,
		},
		{
			"foo",
			[]string{"foo"},
			false,
		},
		{
			"foo/",
			[]string{"foo"},
			false,
		},
		{
			"foo/ba",
			[]string{"foo"},
			false,
		},
		{
			"foo/bar",
			[]string{"foo", "foo/bar"},
			false,
		},
		{
			"foo/bar/baz",
			[]string{"foo", "foo/bar", "foo/bar/baz"},
			false,
		},
		{
			"foo/bar/bazoo",
			[]string{"foo", "foo/bar", "foo/bar/baz"},
			false,
		},
		{
			"z",
			nil,
			false,
		},
		{"/", []string{"/u/äpfêl/"}, true},
		{"/u/aPFÊL/", []string{"/u/äpfêl/"}, true},
		{"/u/ÄPFÊL/", []string{"/u/äpfêl/"}, true},
	}

	for _, test := range cases {
		out := []string{}
		fn := func(s string, v interface{}) bool {
			out = append(out, s)
			return false
		}
		if test.nearest {
			r.WalkNearestPath(test.in, fn)
		} else {
			r.WalkPath(test.in, fn)
		}
		sort.Strings(out)
		sort.Strings(test.out)
		if !reflect.DeepEqual(out, test.out) {
			if len(test.out) != len(out) {
				t.Log(r.Dump(true))
				t.Fatalf("mismatch(%s): expected %v, got %v", test.in, test.out, out)
			}
			for i, s0 := range test.out {
				s1 := out[i]
				if !strings.EqualFold(s0, s1) {
					t.Fatalf("mismatch(%s): %+v %v %s (expected: %v, got %v)", test.in, test, out, s0, []rune(s0), []rune(s1))
				}
			}
		}
	}
}

// generateUUID is used to generate a random UUID
func generateUUID() string {
	buf := make([]byte, 16)
	if _, err := crand.Read(buf); err != nil {
		panic(fmt.Errorf("failed to read random bytes: %v", err))
	}

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x",
		buf[0:4],
		buf[4:6],
		buf[6:8],
		buf[8:10],
		buf[10:16])
}
