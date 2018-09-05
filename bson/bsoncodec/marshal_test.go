package bsoncodec

import (
	"bytes"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
)

func TestMarshalAppendWithRegistry(t *testing.T) {
	for _, tc := range marshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			dst := make([]byte, 0, 1024)
			var reg *Registry
			if tc.reg != nil {
				reg = tc.reg
			} else {
				reg = NewRegistryBuilder().Build()
			}
			got, err := MarshalAppendWithRegistry(reg, dst, tc.val)
			noerr(t, err)

			if !bytes.Equal(got, tc.want) {
				t.Errorf("Bytes are not equal. got %v; want %v", bson.Reader(got), bson.Reader(tc.want))
				t.Errorf("Bytes:\n%v\n%v", got, tc.want)
			}
		})
	}
}

func TestMarshalWithRegistry(t *testing.T) {
	for _, tc := range marshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			var reg *Registry
			if tc.reg != nil {
				reg = tc.reg
			} else {
				reg = NewRegistryBuilder().Build()
			}
			got, err := MarshalWithRegistry(reg, tc.val)
			noerr(t, err)

			if !bytes.Equal(got, tc.want) {
				t.Errorf("Bytes are not equal. got %v; want %v", bson.Reader(got), bson.Reader(tc.want))
				t.Errorf("Bytes:\n%v\n%v", got, tc.want)
			}
		})
	}
}

func TestMarshalAppend(t *testing.T) {
	for _, tc := range marshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.reg != nil {
				t.Skip() // test requires custom registry
			}
			dst := make([]byte, 0, 1024)
			got, err := MarshalAppend(dst, tc.val)
			noerr(t, err)

			if !bytes.Equal(got, tc.want) {
				t.Errorf("Bytes are not equal. got %v; want %v", bson.Reader(got), bson.Reader(tc.want))
				t.Errorf("Bytes:\n%v\n%v", got, tc.want)
			}
		})
	}
}

func TestMarshal(t *testing.T) {
	for _, tc := range marshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.reg != nil {
				t.Skip() // test requires custom registry
			}
			got, err := bson.Marshal(tc.val)
			noerr(t, err)

			if !bytes.Equal(got, tc.want) {
				t.Errorf("Bytes are not equal. got %v; want %v", bson.Reader(got), bson.Reader(tc.want))
				t.Errorf("Bytes:\n%v\n%v", got, tc.want)
			}
		})
	}
}
