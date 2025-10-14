package idgen

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var charIndex = func() map[rune]int {
	m := make(map[rune]int)
	for i, r := range alphabet {
		m[r] = i
	}
	return m
}()

// Time complexity is O(k^2) where k is length of the resulting string
// because of repeated prepending to the slice. Could be optimized if needed.
func Encode(n uint64) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		rem := n % 62
		b = append([]byte{alphabet[rem]}, b...) // prepend
		n /= 62
	}
	return string(b)
}

func Decode(s string) uint64 {
	var n uint64
	for _, ch := range s {
		n = n*62 + uint64(charIndex[ch])
	}
	return n
}
