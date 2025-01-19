package random

import (
	crand "crypto/rand"
	"encoding/binary"
	"math/rand/v2"
	"time"
)

// CryptoRandSource represents a source of uniformly-distributed random uint64 values.
type CryptoRandSource struct{}

// Uint64 returns a non-negative random integer as an uint64 from CryptoRandSource.
func (CryptoRandSource) Uint64() uint64 {
	var b [8]byte
	_, err := crand.Read(b[:])
	if err != nil {
		panic(err) // fail - can't continue
	}

	return binary.BigEndian.Uint64(b[:]) & (1<<63 - 1)
}

// New returns a new rand.Source.
func New(secure bool, seed1, seed2 uint64) rand.Source {
	if secure {
		return CryptoRandSource{}
	}

	if seed1 == 0 && seed2 == 0 {
		seed1 = uint64(time.Now().UnixNano()) // #nosec: G115
		seed2 = seed1
	}

	return rand.NewPCG(seed1, seed2)
}
