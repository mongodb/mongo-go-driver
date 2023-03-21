package bsoncodec

import (
	"reflect"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/bsonrw/bsonrwtest"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func BenchmarkSliceCodec_EncodeValue(b *testing.B) {
	codec := NewSliceCodec()
	eContext := EncodeContext{}
	dataToEncode := []byte(strings.Repeat("t", 4096))
	val := reflect.ValueOf(&dataToEncode).Elem()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		vw := &bsonrwtest.ValueReaderWriter{}
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
	dataToDecode := bsoncore.AppendBinary(nil, bsontype.BinaryGeneric, []byte(strings.Repeat("t", 4096)))

	var buf []byte
	val := reflect.ValueOf(&buf).Elem()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		vr := &bsonrwtest.ValueReaderWriter{BSONType: bsontype.Binary, Return: bsoncore.Value{
			Type: bsontype.Binary,
			Data: dataToDecode,
		}}
		err := codec.DecodeValue(dContext, vr, val)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
}

func TestSliceCodec_DecodeValue(t *testing.T) {
	t.Parallel()
	codec := NewSliceCodec()

	t.Run("decode binary", func(t *testing.T) {
		expected := []byte("test bytes")
		vr := &bsonrwtest.ValueReaderWriter{BSONType: bsontype.Binary, Return: bsoncore.Value{
			Type: bsontype.Binary,
			Data: bsoncore.AppendBinary(nil, bsontype.BinaryGeneric, expected),
		}}

		actual := []byte("dummy")
		val := reflect.ValueOf(&actual).Elem()
		err := codec.DecodeValue(DecodeContext{}, vr, val)
		require.NoError(t, err)

		assert.Equal(t, expected, actual)
	})

	t.Run("decode string", func(t *testing.T) {
		expected := "test string"
		vr := &bsonrwtest.ValueReaderWriter{BSONType: bsontype.String, Return: expected}

		actual := []byte("dummy")
		val := reflect.ValueOf(&actual).Elem()
		err := codec.DecodeValue(DecodeContext{}, vr, val)
		require.NoError(t, err)

		assert.Equal(t, expected, string(actual))
	})
}
