package bson

import (
	"testing"
)

func FuzzDecode(f *testing.F) {
	seedBSONCorpus(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		for _, typ := range []func() interface{}{
			func() interface{} { return new(D) },
			func() interface{} { return new([]E) },
			func() interface{} { return new(M) },
			func() interface{} { return new(interface{}) },
			func() interface{} { return make(map[string]interface{}) },
			func() interface{} { return new([]interface{}) },
		} {
			i := typ()
			if err := Unmarshal(data, i); err != nil {
				return
			}

			encoded, err := Marshal(i)
			if err != nil {
				t.Fatal("failed to marshal", err)
			}

			if err := Unmarshal(encoded, i); err != nil {
				t.Fatal("failed to unmarshal", err)
			}
		}
	})
}
