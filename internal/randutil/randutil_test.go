package randutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCryptoSeed(t *testing.T) {
	seeds := make(map[int64]bool)
	for i := 1; i < 1000000; i++ {
		s := CryptoSeed()
		require.False(t, seeds[s], "CryptoSeed returned a duplicate value %d", s)
		seeds[s] = true
	}
}
