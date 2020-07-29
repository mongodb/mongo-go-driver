package bsonx

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
)

func TestReflectionFreeDCodec(t *testing.T) {
	assert.RegisterOpts(reflect.TypeOf(primitive.D{}), cmp.AllowUnexported(primitive.Decimal128{}))

	now := time.Now()
	oid := primitive.NewObjectID()
	d128 := primitive.NewDecimal128(10, 20)
	js := primitive.JavaScript("js")
	symbol := primitive.Symbol("sybmol")
	binary := primitive.Binary{Subtype: 2, Data: []byte("binary")}
	datetime := primitive.NewDateTimeFromTime(now)
	regex := primitive.Regex{Pattern: "pattern", Options: "i"}
	dbPointer := primitive.DBPointer{DB: "db", Pointer: oid}
	timestamp := primitive.Timestamp{T: 5, I: 10}
	cws := primitive.CodeWithScope{Code: js, Scope: bson.D{{"x", 1}}}
	noReflectionRegistry := bson.NewRegistryBuilder().RegisterCodec(tPrimitiveD, ReflectionFreeDCodec).Build()
	docWithAllTypes := primitive.D{
		{"byteSlice", []byte("foobar")},
		{"sliceByteSlice", [][]byte{[]byte("foobar")}},
		{"timeTime", now},
		{"sliceTimeTime", []time.Time{now}},
		{"objectID", oid},
		{"sliceObjectID", []primitive.ObjectID{oid}},
		{"decimal128", d128},
		{"sliceDecimal128", []primitive.Decimal128{d128}},
		{"js", js},
		{"sliceJS", []primitive.JavaScript{js}},
		{"symbol", symbol},
		{"sliceSymbol", []primitive.Symbol{symbol}},
		{"binary", binary},
		{"sliceBinary", []primitive.Binary{binary}},
		{"undefined", primitive.Undefined{}},
		{"sliceUndefined", []primitive.Undefined{{}}},
		{"datetime", datetime},
		{"sliceDateTime", []primitive.DateTime{datetime}},
		{"null", primitive.Null{}},
		{"sliceNull", []primitive.Null{{}}},
		{"regex", regex},
		{"sliceRegex", []primitive.Regex{regex}},
		{"dbPointer", dbPointer},
		{"sliceDBPointer", []primitive.DBPointer{dbPointer}},
		{"timestamp", timestamp},
		{"sliceTimestamp", []primitive.Timestamp{timestamp}},
		{"minKey", primitive.MinKey{}},
		{"sliceMinKey", []primitive.MinKey{{}}},
		{"maxKey", primitive.MaxKey{}},
		{"sliceMaxKey", []primitive.MaxKey{{}}},
		{"cws", cws},
		{"sliceCWS", []primitive.CodeWithScope{cws}},
		{"bool", true},
		{"sliceBool", []bool{true}},
		{"int", int(10)},
		{"sliceInt", []int{10}},
		{"int8", int8(10)},
		{"sliceInt8", []int8{10}},
		{"int16", int16(10)},
		{"sliceInt16", []int16{10}},
		{"int32", int32(10)},
		{"sliceInt32", []int32{10}},
		{"int64", int64(10)},
		{"sliceInt64", []int64{10}},
		{"uint", uint(10)},
		{"sliceUint", []uint{10}},
		{"uint8", uint8(10)},
		{"sliceUint8", []uint8{10}},
		{"uint16", uint16(10)},
		{"sliceUint16", []uint16{10}},
		{"uint32", uint32(10)},
		{"sliceUint32", []uint32{10}},
		{"uint64", uint64(10)},
		{"sliceUint64", []uint64{10}},
		{"float32", float32(10)},
		{"sliceFloat32", []float32{10}},
		{"float64", float64(10)},
		{"sliceFloat64", []float64{10}},
		{"primitiveA", primitive.A{"foo", "bar"}},
	}

	t.Run("encode", func(t *testing.T) {
		// Assert that bson.Marshal returns the same result when using the default registry and noReflectionRegistry.

		expected, err := bson.Marshal(docWithAllTypes)
		assert.Nil(t, err, "Marshal error with default registry: %v", err)
		actual, err := bson.MarshalWithRegistry(noReflectionRegistry, docWithAllTypes)
		assert.Nil(t, err, "Marshal error with noReflectionRegistry: %v", err)
		assert.Equal(t, expected, actual, "expected doc %s, got %s", bson.Raw(expected), bson.Raw(actual))
	})
	t.Run("decode", func(t *testing.T) {
		// Assert that bson.Unmarshal returns the same result when using the default registry and noReflectionRegistry.

		// docWithAllTypes contains some types that can't be roundtripped. For example, any slices besides primitive.A
		// would start of as []T but unmarshal to primitive.A. To get around this, we first marshal docWithAllTypes to
		// raw bytes and then Unmarshal to another primitive.D rather than asserting directly against docWithAllTypes.
		docBytes, err := bson.Marshal(docWithAllTypes)
		assert.Nil(t, err, "Marshal error: %v", err)

		var expected, actual primitive.D
		err = bson.Unmarshal(docBytes, &expected)
		assert.Nil(t, err, "Unmarshal error with default registry: %v", err)
		err = bson.UnmarshalWithRegistry(noReflectionRegistry, docBytes, &actual)
		assert.Nil(t, err, "Unmarshal error with noReflectionRegistry: %v", err)

		assert.Equal(t, expected, actual, "expected document %v, got %v", expected, actual)
	})
}
