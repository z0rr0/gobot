package random

import (
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"time"
)

// CryptoRandSource represents a source of uniformly-distributed random int64 values in the range [0, 1<<63).
type CryptoRandSource struct{}

// Int63 returns a non-negative random 63-bit integer as an int64 from CryptoRandSource.
func (CryptoRandSource) Int63() int64 {
	var b [8]byte
	_, err := crand.Read(b[:])
	if err != nil {
		panic(err) // fail - can't continue
	}
	return int64(binary.LittleEndian.Uint64(b[:]) & (1<<63 - 1))
}

// Seed is a fake implementation for rand.Source interface from math/rand.
func (CryptoRandSource) Seed(int64) {}

// New returns a new rand.Source.
func New(secure bool, seed int64) rand.Source {
	if secure {
		return CryptoRandSource{}
	}

	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return rand.NewSource(seed)
}
