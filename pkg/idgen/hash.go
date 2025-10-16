package idgen

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
)

type HashGenerator struct {
	nBytes int // how many bytes from the hash to use (1..8)
}

func NewHashGenerator(nBytes int) (*HashGenerator, error) {
	if nBytes < 1 || nBytes > 8 {
		return nil, errors.New("nBytes must be between 1 and 8")
	}
	return &HashGenerator{nBytes: nBytes}, nil
}

// Generate creates short code by hashing the long URL (SHA256).
func (g *HashGenerator) Generate(longURL string) (string, error) {
	hash := sha256.Sum256([]byte(longURL))

	var v uint64
	buf := make([]byte, 8)
	copy(buf[8-g.nBytes:], hash[:g.nBytes])
	v = binary.BigEndian.Uint64(buf)

	code := Encode(v)
	return code, nil
}
