package radix

import (
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

func hasPrefixFn(fold bool) func(s, pre string) bool {
	if !fold {
		return strings.HasPrefix
	}

	return HasPrefixFold
}

func longestPrefixFn(fold bool) func(a, b string) int {
	if !fold {
		return LongestPrefix
	}

	return LongestPrefixFold
}

// LongestPrefix finds the length of the shared prefix
// of two strings
func LongestPrefix(k1, k2 string) (i int) {
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

// LongestPrefixFold finds the length of the shared prefix
// of two strings, ignoring case.
func LongestPrefixFold(k1, k2 string) (i int) {
	if len(k1) > len(k2) {
		k1, k2 = k2, k1
	}

	var r1, r2 rune

	for i, r1 = range k1 {
		_ = k2[i]
		if r1 < utf8.RuneSelf {
			if !asciiEq(byte(r1), k2[i]) {
				return
			}
			continue
		}

		if r2, _ = utf8.DecodeRuneInString(k2[i:]); !runeEq(r1, r2) {
			return
		}
	}

	return len(k1)
}

// StringsEqualFold is a simplified version of strings.EqualFold,
// returns true if strings are equal, ignoring case.
func StringsEqualFold(k1, k2 string) bool {
	return len(k1) == len(k2) && LongestPrefixFold(k1, k2) == len(k1)
}

// StringCountByte is an optimized version of strings.Count for a single byte
func StringCountByte(s string, b byte) (n int) {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			n++
		}
	}
	return
}

// HasPrefixFold is a case-insenstive version of strings.HasPrefix.
func HasPrefixFold(s, pre string) (_ bool) {
	if len(s) < len(pre) {
		return
	}

	var pr, sr rune
	var i int

	for i, pr = range pre {
		_ = s[i]
		if pr < utf8.RuneSelf {
			if !asciiEq(byte(pr), s[i]) {
				return
			}
			continue
		}

		if sr, _ = utf8.DecodeRuneInString(s[i:]); !runeEq(pr, sr) {
			return
		}
	}

	return true
}

func runeEq(sr, tr rune) bool {
	if sr == tr {
		return true
	}
	return unicode.To(unicode.LowerCase, sr) == unicode.To(unicode.LowerCase, tr)
}

var asciiTable = func() (t [math.MaxUint8 + 1]byte) {
	for i := 0; i < len(t); i++ {
		t[i] = asciiLower(byte(i))
	}
	return
}()

func asciiEq(sr, tr byte) bool {
	return asciiTable[sr] == asciiTable[tr]
}

func asciiLower(r byte) byte {
	if 'A' <= r && r <= 'Z' {
		r += 'a' - 'A'
	}
	return r
}

func nextRune(s string) rune {
	if s[0] < utf8.RuneSelf {
		return rune(s[0])
	}
	r, _ := utf8.DecodeRuneInString(s)
	return r
}
