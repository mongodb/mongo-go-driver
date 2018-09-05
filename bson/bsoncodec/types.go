package bsoncodec

import (
	"encoding/json"
	"net/url"
	"reflect"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

var tDocument = reflect.TypeOf((*bson.Document)(nil))
var tArray = reflect.TypeOf((*bson.Array)(nil))
var tBinary = reflect.TypeOf(bson.Binary{})
var tBool = reflect.TypeOf(false)
var tCodeWithScope = reflect.TypeOf(bson.CodeWithScope{})
var tDBPointer = reflect.TypeOf(bson.DBPointer{})
var tDecimal = reflect.TypeOf(decimal.Decimal128{})
var tDateTime = reflect.TypeOf(bson.DateTime(0))
var tUndefined = reflect.TypeOf(bson.Undefinedv2{})
var tNull = reflect.TypeOf(bson.Nullv2{})
var tValue = reflect.TypeOf((*bson.Value)(nil))
var tFloat32 = reflect.TypeOf(float32(0))
var tFloat64 = reflect.TypeOf(float64(0))
var tInt = reflect.TypeOf(int(0))
var tInt8 = reflect.TypeOf(int8(0))
var tInt16 = reflect.TypeOf(int16(0))
var tInt32 = reflect.TypeOf(int32(0))
var tInt64 = reflect.TypeOf(int64(0))
var tJavaScriptCode = reflect.TypeOf(bson.JavaScriptCode(""))
var tOID = reflect.TypeOf(objectid.ObjectID{})
var tReader = reflect.TypeOf(bson.Reader(nil))
var tRegex = reflect.TypeOf(bson.Regex{})
var tString = reflect.TypeOf("")
var tSymbol = reflect.TypeOf(bson.Symbol(""))
var tTime = reflect.TypeOf(time.Time{})
var tTimestamp = reflect.TypeOf(bson.Timestamp{})
var tUint = reflect.TypeOf(uint(0))
var tUint8 = reflect.TypeOf(uint8(0))
var tUint16 = reflect.TypeOf(uint16(0))
var tUint32 = reflect.TypeOf(uint32(0))
var tUint64 = reflect.TypeOf(uint64(0))
var tMinKey = reflect.TypeOf(bson.MinKeyv2{})
var tMaxKey = reflect.TypeOf(bson.MaxKeyv2{})

var tEmpty = reflect.TypeOf((*interface{})(nil)).Elem()
var tElement = reflect.TypeOf((*bson.Element)(nil))
var tByteSlice = reflect.TypeOf([]byte(nil))
var tElementSlice = reflect.TypeOf(([]*bson.Element)(nil))
var tByte = reflect.TypeOf(byte(0x00))
var tURL = reflect.TypeOf(url.URL{})
var tJSONNumber = reflect.TypeOf(json.Number(""))

var tValueMarshaler = reflect.TypeOf((*ValueMarshaler)(nil)).Elem()
var tValueUnmarshaler = reflect.TypeOf((*ValueUnmarshaler)(nil)).Elem()
