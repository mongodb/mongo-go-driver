package bsoncodec

import (
	"reflect"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

func BenchmarkSliceCodec_EncodeValue(b *testing.B) {
	codec := NewSliceCodec()
	eContext := EncodeContext{}
	vw := &valueWriterMock{}
	dataToEncode := []byte(strings.Repeat("t", 4096))
	val := reflect.ValueOf(&dataToEncode).Elem()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		err := codec.EncodeValue(eContext, vw, val)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
}

func BenchmarkSliceCodec_DecodeValue(b *testing.B) {
	codec := NewSliceCodec()
	dContext := DecodeContext{}
	vr := &valueReaderMock{}
	var buf []byte
	val := reflect.ValueOf(&buf).Elem()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		err := codec.DecodeValue(dContext, vr, val)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
}

type valueWriterMock struct {
	bsonrw.ValueWriter
}

func (m *valueWriterMock) WriteBinary(_ []byte) error {
	return nil
}

type valueReaderMock struct {
	bsonrw.ValueReader
}

func (m *valueReaderMock) Type() bsontype.Type {
	return bsontype.Binary
}

var dataToDecode = []byte(strings.Repeat("t", 4096))

func (m *valueReaderMock) ReadBinary() (b []byte, btype byte, err error) {
	return dataToDecode, bsontype.BinaryGeneric, nil
}
