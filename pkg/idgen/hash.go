package idgen

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
)

// HashGenerator generates deterministic short codes from the long URL.
// It hashes the URL (SHA256), takes the first `nBytes` bytes of the digest,
// converts that slice to a uint64, and encodes with base62.
// nBytes controls length/collision risk; 6 bytes -> 48 bits -> ~2.8e14 values.
type HashGenerator struct {
	nBytes int // how many bytes from the hash to use (1..8)
}

// NewHashGenerator returns a HashGenerator which uses nBytes of the hash.
// nBytes must be 1..8. Recommended: 6 (good balance of length vs collisions).
func NewHashGenerator(nBytes int) (*HashGenerator, error) {
	if nBytes < 1 || nBytes > 8 {
		return nil, errors.New("nBytes must be between 1 and 8")
	}
	return &HashGenerator{nBytes: nBytes}, nil
}

// Generate creates short code by hashing the long URL (SHA256).
// Deterministic: same longURL => same short code (given same nBytes).
// Note: collisions are possible due to truncation; caller should check DB uniqueness.
func (g *HashGenerator) Generate(longURL string) (string, error) {
	hash := sha256.Sum256([]byte(longURL))

	// take first g.nBytes bytes and convert to uint64 (big-endian)
	var v uint64
	buf := make([]byte, 8)                  // zeroed
	copy(buf[8-g.nBytes:], hash[:g.nBytes]) // put bytes at the rightmost side for big-endian
	v = binary.BigEndian.Uint64(buf)

	// encode to base62
	code := Encode(v)
	return code, nil
}
