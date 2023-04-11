package random

import (
	"math/rand"
	"testing"
)

func TestNew(t *testing.T) {
	testCases := []struct {
		secure     bool
		seed       int64
		expected   int64
		unexpected int64
	}{
		{seed: 1, expected: 5577006791947779410},
		{seed: 1000, expected: 6278013164014963327},
		{secure: true, seed: 1000, unexpected: 6278013164014963327},
		{secure: true, unexpected: 6278013164014963327},
	}
	for _, tc := range testCases {
		s := New(tc.secure, tc.seed)
		r := rand.New(s)

		if tc.unexpected != 0 {
			noMatch := false
			// it's probability case, so we need to check it several times
			for i := 0; i < 100; i++ {
				if v := r.Int63(); v != tc.unexpected {
					noMatch = true // success case
					break
				}
			}
			if !noMatch {
				t.Errorf("expected not %d", tc.unexpected)
			}
		} else {
			if v := r.Int63(); v != tc.expected {
				t.Errorf("expected %v but got %d", tc.expected, v)
			}
		}
	}
}
