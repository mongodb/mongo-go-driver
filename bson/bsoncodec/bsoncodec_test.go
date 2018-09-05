package bsoncodec

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
)

func TestBasicEncode(t *testing.T) {
	for _, tc := range marshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			got := make(writer, 0, 1024)
			vw := newValueWriter(&got)
			reg := NewRegistryBuilder().Build()
			codec, err := reg.Lookup(reflect.TypeOf(tc.val))
			noerr(t, err)
			err = codec.EncodeValue(EncodeContext{Registry: reg}, vw, tc.val)
			noerr(t, err)

			if !bytes.Equal(got, tc.want) {
				t.Errorf("Bytes are not equal. got %v; want %v", bson.Reader(got), bson.Reader(tc.want))
				t.Errorf("Bytes:\n%v\n%v", got, tc.want)
			}
		})
	}
}

func TestBasicDecode(t *testing.T) {
	for _, tc := range unmarshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			got := reflect.New(tc.sType).Interface()
			vr := newValueReader(tc.data)
			reg := NewRegistryBuilder().Build()
			codec, err := reg.Lookup(reflect.TypeOf(got))
			noerr(t, err)
			err = codec.DecodeValue(DecodeContext{Registry: reg}, vr, got)
			noerr(t, err)

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Results do not match. got %+v; want %+v", got, tc.want)
			}
		})
	}
}
