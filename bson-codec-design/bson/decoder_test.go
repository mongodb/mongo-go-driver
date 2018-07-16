package bson

import (
	"reflect"
	"testing"
)

func TestDecoderDecode(t *testing.T) {
	for _, tc := range unmarshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			got := reflect.New(tc.sType).Interface()
			vr := newValueReader(tc.data)
			var reg *Registry
			if tc.reg != nil {
				reg = tc.reg
			} else {
				reg = NewRegistry()
			}
			dec, err := NewDecoder(reg, vr)
			noerr(t, err)
			err = dec.Decode(got)
			noerr(t, err)

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Results do not match. got %+v; want %+v", got, tc.want)
			}
		})
	}
}
