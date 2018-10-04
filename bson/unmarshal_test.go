package bson

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
)

func TestUnmarshal(t *testing.T) {
	for _, tc := range unmarshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.reg != nil {
				t.Skip() // test requires custom registry
			}
			got := reflect.New(tc.sType).Interface()
			err := Unmarshal(tc.data, got)
			noerr(t, err)
			if !cmp.Equal(got, tc.want) {
				t.Errorf("Did not unmarshal as expected. got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestUnmarshalWithRegistry(t *testing.T) {
	for _, tc := range unmarshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			var reg *bsoncodec.Registry
			if tc.reg != nil {
				reg = tc.reg
			} else {
				reg = DefaultRegistry
			}
			got := reflect.New(tc.sType).Interface()
			err := UnmarshalWithRegistry(reg, tc.data, got)
			noerr(t, err)
			if !cmp.Equal(got, tc.want) {
				t.Errorf("Did not unmarshal as expected. got %v; want %v", got, tc.want)
			}
		})
	}
}
