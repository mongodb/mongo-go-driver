package flate

import (
	"io/ioutil"
	"testing"
)

type testFatal interface {
	Fatal(args ...interface{})
}

// loadTestTokens will load test tokens.
// First block from enwik9, varint encoded.
func loadTestTokens(t testFatal) *tokens {
	b, err := ioutil.ReadFile("testdata/tokens.bin")
	if err != nil {
		t.Fatal(err)
	}
	var tokens tokens
	err = tokens.FromVarInt(b)
	if err != nil {
		t.Fatal(err)
	}
	return &tokens
}

func Test_tokens_EstimatedBits(t *testing.T) {
	tok := loadTestTokens(t)
	// The estimated size, update if method changes.
	const expect = 199380
	if n := tok.EstimatedBits(); n != expect {
		t.Error("want:", expect, "bits, got:", n)
	}
}

func Benchmark_tokens_EstimatedBits(b *testing.B) {
	tok := loadTestTokens(b)
	b.ResetTimer()
	// One "byte", one token iteration.
	b.SetBytes(1)
	for i := 0; i < b.N; i++ {
		_ = tok.EstimatedBits()
	}
}
