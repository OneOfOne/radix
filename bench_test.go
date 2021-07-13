//go:build go1.18
// +build go1.18

package radix

import (
	"fmt"
	"testing"
	"unicode/utf8"
)

var sink int

func BenchmarkHasPrefix(b *testing.B) {
	k1 := "/api/v1/user/çç"
	k2 := "/api/v1/useR/x"
	b.Log(k1[:LongestPrefix(k1, k2)])
	b.Log(k1[:LongestPrefixFold(k1, k2)])
	b.Run("fold", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sink += LongestPrefixFold(k2, k1)
		}
	})
	b.Run("no-fold", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sink += LongestPrefix(k2, k1)
		}
	})
}

func BenchmarkNextRune(b *testing.B) {
	k1 := "/u/äpfêl/"
	b.Run("for-loop", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, r := range k1 {
				sink = int(r)
			}
		}
	})
	b.Run("decode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var r rune
			var sz int
			k1 := k1
			for k1 != "" {
				r, sz = utf8.DecodeRuneInString(k1)
				k1 = k1[sz:]
				sink = int(r)
			}
		}
	})
}

func BenchmarkTree(b *testing.B) {
	var keys [100000]string
	for i := range keys {
		keys[i] = fmt.Sprintf("/api/%02d/%03d/%04d", i%10, i%100, i+1)
	}

	b.ResetTimer()

	var t Tree[int]

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
