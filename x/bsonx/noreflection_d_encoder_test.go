package bsonx

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
)

func TestReflectionFreeDecoder(t *testing.T) {
	// TODO: json.Number, url.URL, bson.Raw/bsoncore.Document, map

	now := time.Now()
	oid := primitive.NewObjectID()
	d128 := primitive.NewDecimal128(20, 20)
	js := primitive.JavaScript("js")
	symbol := primitive.Symbol("sybmol")
	binary := primitive.Binary{Subtype: 2, Data: []byte("binary")}
	datetime := primitive.NewDateTimeFromTime(now)
	regex := primitive.Regex{Pattern: "pattern", Options: "i"}
	dbPointer := primitive.DBPointer{DB: "db", Pointer: oid}
	timestamp := primitive.Timestamp{T: 10, I: 10}
	cws := primitive.CodeWithScope{Code: js, Scope: bson.D{{"x", 1}}}

	doc := primitive.D{
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

	expected, err := bson.Marshal(doc)
	assert.Nil(t, err, "Marshal error: %v", err)
	registry := bson.NewRegistryBuilder().RegisterEncoder(tPrimitiveD, ReflectionFreeDEncoder).Build()
	actual, err := bson.MarshalWithRegistry(registry, doc)
	assert.Nil(t, err, "Marshal error: %v", err)
	assert.Equal(t, expected, actual, "expected doc %s, got %s", bson.Raw(expected), bson.Raw(actual))
}
