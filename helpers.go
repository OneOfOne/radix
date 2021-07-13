package radix

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func firstChar(s string) rune {
	if s[0] < utf8.RuneSelf {
		return rune(s[0])
	}
	r, _ := utf8.DecodeRuneInString(s)
	return r
}

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
	var sz int

	for i < len(k1) {
		if k1[i] < utf8.RuneSelf {
			if !asciiEq(k1[i], k2[i]) {
				return
			}
			i++
			continue
		}

		r1, sz = utf8.DecodeRuneInString(k1[i:])
		if r2, _ = utf8.DecodeRuneInString(k2[i:]); !runeEq(r1, r2) {
			return
		}
		i += sz
	}

	return i
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
	var i, sz int

	for i < len(pre) {
		if pre[i] < utf8.RuneSelf {
			if !asciiEq(pre[i], s[i]) {
				return
			}
			i++
			continue
		}

		pr, sz = utf8.DecodeRuneInString(pre[i:])
		if sr, _ = utf8.DecodeRuneInString(s[i:]); !runeEq(pr, sr) {
			return
		}
		i += sz
	}

	return true
}

func runeEq(sr, tr rune) bool {
	if sr == tr {
		return true
	}
	return toLower(sr) == toLower(tr)
}

func asciiEq(sr, tr byte) bool {
	return sr == tr || asciiLower(sr) == asciiLower(tr)
}

func asciiLower(r byte) byte {
	if 'A' <= r && r <= 'Z' {
		r += 'a' - 'A'
	}
	return r
}

func toLower(r rune) rune {
	return unicode.ToLower(r)
}

func nextRune(s string) rune {
	if s[0] < utf8.RuneSelf {
		return rune(s[0])
	}
	r, _ := utf8.DecodeRuneInString(s)
	return r
}
