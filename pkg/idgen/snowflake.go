package idgen

import (
	"errors"
	"sync"
	"time"
)

// SnowflakeGenerator implements a Snowflake-like ID generator.
// Layout (64 bits):
// 41 bits timestamp (ms since custom epoch)
// 10 bits node ID (0..1023)
// 12 bits sequence (0..4095)
type SnowflakeGenerator struct {
	mu          sync.Mutex
	epoch       int64  // custom epoch in ms
	nodeID      uint64 // up to 10 bits
	lastTs      int64
	sequence    uint64 // 12 bits
	maxSequence uint64
	maxNodeID   uint64
}

// NewSnowflakeGenerator creates a SnowflakeGenerator with given epoch (ms) and nodeID.
// nodeID must fit in 10 bits (0..1023). If epoch==0, a sensible default is used:
// example default epoch: 2020-01-01 00:00:00 UTC
func NewSnowflakeGenerator(nodeID uint64, epochMs int64) (*SnowflakeGenerator, error) {
	const (
		nodeBits     = 10
		sequenceBits = 12
	)

	maxNode := uint64((1 << nodeBits) - 1)
	if nodeID > maxNode {
		return nil, errors.New("nodeID out of range")
	}

	if epochMs == 0 {
		// default epoch: 2020-01-01T00:00:00Z
		epochMs = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano() / 1e6
	}

	return &SnowflakeGenerator{
		epoch:       epochMs,
		nodeID:      nodeID,
		lastTs:      -1,
		sequence:    0,
		maxSequence: (1 << sequenceBits) - 1,
		maxNodeID:   maxNode,
	}, nil
}

// Generate returns a base62-encoded Snowflake ID (opaque short code).
// Caller should be prepared to store and use these codes directly.
func (s *SnowflakeGenerator) Generate() (string, error) {
	const (
		timestampBits = 41
		nodeBits      = 10
		sequenceBits  = 12
	)

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixNano() / 1e6 // ms
	ts := now - s.epoch
	if ts < 0 {
		return "", errors.New("current time is before epoch")
	}

	// If same millisecond as last, increment sequence
	if ts == s.lastTs {
		s.sequence = (s.sequence + 1) & s.maxSequence
		if s.sequence == 0 {
			// sequence overflow within same millisecond -> wait for next millisecond
			for ts <= s.lastTs {
				time.Sleep(time.Millisecond)
				now = time.Now().UnixNano() / 1e6
				ts = now - s.epoch
			}
		}
	} else {
		// new millisecond, reset sequence
		s.sequence = 0
	}

	s.lastTs = ts

	// Compose ID: (ts << (nodeBits+sequenceBits)) | (nodeID << sequenceBits) | sequence
	id := (uint64(ts) << (nodeBits + sequenceBits)) |
		((s.nodeID & s.maxNodeID) << sequenceBits) |
		(s.sequence & s.maxSequence)

	// encode to base62 for short human-safe code
	code := Encode(id)
	return code, nil
}
