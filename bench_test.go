//go:build go1.18
// +build go1.18

package radix

import (
	"fmt"
	"math/rand"
	"testing"
)

var sink int


func BenchmarkTree(b *testing.B) {
	benchTree(b, New[int](false))
}

func BenchmarkTreeFold(b *testing.B) {
	benchTree(b, New[int](true))
}

func benchTree(b *testing.B, t *Tree[int]) {
	b.Helper()
	var keys [100000]string
	for i := range keys {
		keys[i] = fmt.Sprintf("/api/%02d/%03d/%04d", i%10, i%100, i+1)
		if i%100 == 0 {
			keys[i] += "ä"
		}
	}

	b.ResetTimer()

	b.Run("Set", func(b *testing.B) {
		first := true
		for i := 0; i < b.N; i++ {
			for x, k := range keys {
				if _, updated := t.Set(k, x); !updated && !first {
					b.Fatal("bad update")
				}
			}
			first = false
		}

		if t.Len() != len(keys) {
			b.Fatalf("wrong tree size: %d", t.Len())
		}
	})

	b.Run("Get", func(b *testing.B) {
		b.RunParallel(func(p *testing.PB) {
			r := rand.New(rand.NewSource(42))
			for p.Next() {
				idx := r.Intn(len(keys))
				if v, _ := t.Get(keys[idx]); v != idx {
					b.Fatalf("expected %v, got %v", idx, v)
				}
			}
		})
	})

	b.Run("LongestPrefix", func(b *testing.B) {
		b.RunParallel(func(p *testing.PB) {
			r := rand.New(rand.NewSource(42))
			for p.Next() {
				pre := keys[r.Intn(len(keys))]
				t.LongestPrefix(pre)
			}
		})
	})

	b.Run("Delete", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, k := range keys {
				t.Delete(k)
			}
		}
	})
}
