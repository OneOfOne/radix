//go:build go1.18
// +build go1.18

package radix

import (
	"fmt"
	"testing"
)

var sink int

func BenchmarkHasPrefix(b *testing.B) {
	k1 := "/api/v1/user/çç"
	k2 := "/api/v1/useR/x"
	b.Log(k1[:longestPrefix(k1, k2)])
	b.Run("fold", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sink += longestPrefixFold(k2, k1)
		}
	})
	b.Run("no-fold", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sink += longestPrefix(k2, k1)
		}
	})
}

func BenchmarkTree(b *testing.B) {
	var keys [10000]string
	for i := range keys {
		keys[i] = fmt.Sprintf("/api/%02d/%03d/%04d", i%10, i%100, i+1)
	}
	b.ResetTimer()

	var t Tree[int]

	b.Run("Insert", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for x, k := range keys {
				t.Insert(k, x)
			}
		}

		if t.Len() != len(keys) {
			b.Fatalf("wrong tree size: %d", t.Len())
		}
	})

	b.Run("GetParallel", func(b *testing.B) {
		b.RunParallel(func(p *testing.PB) {
			for p.Next() {
				for x, k := range keys {
					if v, _ := t.Get(k); v != x {
						b.Fatalf("expected %v, got %v", x, v)
					}
				}
			}
		})
	})

	b.Run("LongestPrefixParallel", func(b *testing.B) {
		pre := keys[len(keys)/2]
		pre = pre[:len(pre)-4]
		b.RunParallel(func(p *testing.PB) {
			for p.Next() {
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
