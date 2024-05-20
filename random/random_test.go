package random

import (
	"math/rand/v2"
	"testing"
)

func TestNew(t *testing.T) {
	const maxRetries = 100

	testCases := []struct {
		secure     bool
		seed1      uint64
		seed2      uint64
		expected   uint64
		unexpected uint64
	}{
		{seed1: 1, seed2: 2, expected: 14192431797130687760},
		{seed1: 1000, expected: 6558345869293885150},
		{seed2: 1000, expected: 1584297898518914641},
		{secure: true, seed1: 1000, unexpected: 14192431797130687760},
		{secure: true, seed2: 1000, unexpected: 14192431797130687760},
		{secure: true, unexpected: 1584297898518914641},
		{unexpected: 14192431797130687760},
	}
	for _, tc := range testCases {
		s := New(tc.secure, tc.seed1, tc.seed2)
		r := rand.New(s)

		if tc.unexpected != 0 {
			noMatch := false
			// it's probability case, so we need to check it several times
			for range maxRetries {
				if v := r.Uint64(); v != tc.unexpected {
					noMatch = true // success case
					break
				}
			}
			if !noMatch {
				t.Errorf("expected not %d", tc.unexpected)
			}
		} else {
			if v := r.Uint64(); v != tc.expected {
				t.Errorf("expected %v but got %d", tc.expected, v)
			}
		}
	}
}
