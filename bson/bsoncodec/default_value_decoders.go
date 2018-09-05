package bsoncodec

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"reflect"
	"strconv"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

var defaultValueDecoders DefaultValueDecoders

// DefaultValueDecoders is a namespace type for the default ValueDecoders used
// when creating a registry.
type DefaultValueDecoders struct{}

// BooleanDecodeValue is the ValueDecoderFunc for bool types.
func (dvd DefaultValueDecoders) BooleanDecodeValue(dctx DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != bson.TypeBoolean {
		return fmt.Errorf("cannot decode %v into a boolean", vr.Type())
	}

	var err error
	if target, ok := i.(*bool); ok && target != nil { // if it is nil, we go the slow path.
		*target, err = vr.ReadBoolean()
		return err
	}

	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || !val.Elem().CanSet() {
		return errors.New("BooleanDecodeValue can only be used to decode settable (non-nil) values")
	}
	val = val.Elem()
	if val.Type().Kind() != reflect.Bool {
		return ValueDecoderError{Name: "BooleanDecodeValue", Types: []interface{}{bool(true)}, Received: i}
	}

	b, err := vr.ReadBoolean()
	val.SetBool(b)
	return err
}

// IntDecodeValue is the ValueDecoderFunc for bool types.
func (dvd DefaultValueDecoders) IntDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	var i64 int64
	var err error
	switch vr.Type() {
	case bson.TypeInt32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		i64 = int64(i32)
	case bson.TypeInt64:
		i64, err = vr.ReadInt64()
		if err != nil {
			return err
		}
	case bson.TypeDouble:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		if !dc.Truncate && math.Floor(f64) != f64 {
			return errors.New("IntDecodeValue can only truncate float64 to an integer type when truncation is enabled")
		}
		if f64 > float64(math.MaxInt64) {
			return fmt.Errorf("%g overflows int64", f64)
		}
		i64 = int64(f64)
	default:
		return fmt.Errorf("cannot decode %v into an integer type", vr.Type())
	}

	switch target := i.(type) {
	case *int8:
		if target == nil {
			return errors.New("IntDecodeValue can only be used to decode non-nil *int8")
		}
		if i64 < math.MinInt8 || i64 > math.MaxInt8 {
			return fmt.Errorf("%d overflows int8", i64)
		}
		*target = int8(i64)
		return nil
	case *int16:
		if target == nil {
			return errors.New("IntDecodeValue can only be used to decode non-nil *int16")
		}
		if i64 < math.MinInt16 || i64 > math.MaxInt16 {
			return fmt.Errorf("%d overflows int16", i64)
		}
		*target = int16(i64)
		return nil
	case *int32:
		if target == nil {
			return errors.New("IntDecodeValue can only be used to decode non-nil *int32")
		}
		if i64 < math.MinInt32 || i64 > math.MaxInt32 {
			return fmt.Errorf("%d overflows int32", i64)
		}
		*target = int32(i64)
		return nil
	case *int64:
		if target == nil {
			return errors.New("IntDecodeValue can only be used to decode non-nil *int64")
		}
		*target = int64(i64)
		return nil
	case *int:
		if target == nil {
			return errors.New("IntDecodeValue can only be used to decode non-nil *int")
		}
		if int64(int(i64)) != i64 { // Can we fit this inside of an int
			return fmt.Errorf("%d overflows int", i64)
		}
		*target = int(i64)
		return nil
	}

	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || !val.Elem().CanSet() {
		return fmt.Errorf("IntDecodeValue can only be used to decode settable (non-nil) values")
	}
	val = val.Elem()

	switch val.Type().Kind() {
	case reflect.Int8:
		if i64 < math.MinInt8 || i64 > math.MaxInt8 {
			return fmt.Errorf("%d overflows int8", i64)
		}
	case reflect.Int16:
		if i64 < math.MinInt16 || i64 > math.MaxInt16 {
			return fmt.Errorf("%d overflows int16", i64)
		}
	case reflect.Int32:
		if i64 < math.MinInt32 || i64 > math.MaxInt32 {
			return fmt.Errorf("%d overflows int32", i64)
		}
	case reflect.Int64:
	case reflect.Int:
		if int64(int(i64)) != i64 { // Can we fit this inside of an int
			return fmt.Errorf("%d overflows int", i64)
		}
	default:
		return ValueDecoderError{
			Name:     "IntDecodeValue",
			Types:    []interface{}{(*int8)(nil), (*int16)(nil), (*int32)(nil), (*int64)(nil), (*int)(nil)},
			Received: i,
		}
	}

	val.SetInt(i64)
	return nil
}

// UintDecodeValue is the ValueDecoderFunc for uint types.
func (dvd DefaultValueDecoders) UintDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	var i64 int64
	var err error
	switch vr.Type() {
	case bson.TypeInt32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		i64 = int64(i32)
	case bson.TypeInt64:
		i64, err = vr.ReadInt64()
		if err != nil {
			return err
		}
	case bson.TypeDouble:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		if !dc.Truncate && math.Floor(f64) != f64 {
			return errors.New("UintDecodeValue can only truncate float64 to an integer type when truncation is enabled")
		}
		if f64 > float64(math.MaxInt64) {
			return fmt.Errorf("%g overflows int64", f64)
		}
		i64 = int64(f64)
	default:
		return fmt.Errorf("cannot decode %v into an integer type", vr.Type())
	}

	switch target := i.(type) {
	case *uint8:
		if target == nil {
			return errors.New("UintDecodeValue can only be used to decode non-nil *uint8")
		}
		if i64 < 0 || i64 > math.MaxUint8 {
			return fmt.Errorf("%d overflows uint8", i64)
		}
		*target = uint8(i64)
		return nil
	case *uint16:
		if target == nil {
			return errors.New("UintDecodeValue can only be used to decode non-nil *uint16")
		}
		if i64 < 0 || i64 > math.MaxUint16 {
			return fmt.Errorf("%d overflows uint16", i64)
		}
		*target = uint16(i64)
		return nil
	case *uint32:
		if target == nil {
			return errors.New("UintDecodeValue can only be used to decode non-nil *uint32")
		}
		if i64 < 0 || i64 > math.MaxUint32 {
			return fmt.Errorf("%d overflows uint32", i64)
		}
		*target = uint32(i64)
		return nil
	case *uint64:
		if target == nil {
			return errors.New("UintDecodeValue can only be used to decode non-nil *uint64")
		}
		if i64 < 0 {
			return fmt.Errorf("%d overflows uint64", i64)
		}
		*target = uint64(i64)
		return nil
	case *uint:
		if target == nil {
			return errors.New("UintDecodeValue can only be used to decode non-nil *uint")
		}
		if i64 < 0 || int64(uint(i64)) != i64 { // Can we fit this inside of an uint
			return fmt.Errorf("%d overflows uint", i64)
		}
		*target = uint(i64)
		return nil
	}

	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || !val.Elem().CanSet() {
		return errors.New("UintDecodeValue can only be used to decode settable (non-nil) values")
	}
	val = val.Elem()

	switch val.Type().Kind() {
	case reflect.Uint8:
		if i64 < 0 || i64 > math.MaxUint8 {
			return fmt.Errorf("%d overflows uint8", i64)
		}
	case reflect.Uint16:
		if i64 < 0 || i64 > math.MaxUint16 {
			return fmt.Errorf("%d overflows uint16", i64)
		}
	case reflect.Uint32:
		if i64 < 0 || i64 > math.MaxUint32 {
			return fmt.Errorf("%d overflows uint32", i64)
		}
	case reflect.Uint64:
		if i64 < 0 {
			return fmt.Errorf("%d overflows uint64", i64)
		}
	case reflect.Uint:
		if i64 < 0 || int64(uint(i64)) != i64 { // Can we fit this inside of an uint
			return fmt.Errorf("%d overflows uint", i64)
		}
	default:
		return ValueDecoderError{
			Name:     "UintDecodeValue",
			Types:    []interface{}{(*uint8)(nil), (*uint16)(nil), (*uint32)(nil), (*uint64)(nil), (*uint)(nil)},
			Received: i,
		}
	}

	val.SetUint(uint64(i64))
	return nil
}

// FloatDecodeValue is the ValueDecoderFunc for float types.
func (dvd DefaultValueDecoders) FloatDecodeValue(ec DecodeContext, vr ValueReader, i interface{}) error {
	var f float64
	var err error
	switch vr.Type() {
	case bson.TypeInt32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		f = float64(i32)
	case bson.TypeInt64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return err
		}
		f = float64(i64)
	case bson.TypeDouble:
		f, err = vr.ReadDouble()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot decode %v into a float32 or float64 type", vr.Type())
	}

	switch target := i.(type) {
	case *float32:
		if target == nil {
			return errors.New("FloatDecodeValue can only be used to decode non-nil *float32")
		}
		if !ec.Truncate && float64(float32(f)) != f {
			return errors.New("FloatDecodeValue can only convert float64 to float32 when truncation is allowed")
		}
		*target = float32(f)
		return nil
	case *float64:
		if target == nil {
			return errors.New("FloatDecodeValue can only be used to decode non-nil *float64")
		}
		*target = f
		return nil
	}

	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || !val.Elem().CanSet() {
		return errors.New("FloatDecodeValue can only be used to decode settable (non-nil) values")
	}
	val = val.Elem()

	switch val.Type().Kind() {
	case reflect.Float32:
		if !ec.Truncate && float64(float32(f)) != f {
			return errors.New("FloatDecodeValue can only convert float64 to float32 when truncation is allowed")
		}
	case reflect.Float64:
	default:
		return ValueDecoderError{Name: "FloatDecodeValue", Types: []interface{}{(*float32)(nil), (*float64)(nil)}, Received: i}
	}

	val.SetFloat(f)
	return nil
}

// StringDecodeValue is the ValueDecoderFunc for string types.
func (dvd DefaultValueDecoders) StringDecodeValue(dctx DecodeContext, vr ValueReader, i interface{}) error {
	var str string
	var err error
	switch vr.Type() {
	case bson.TypeString:
		str, err = vr.ReadString()
		if err != nil {
			return err
		}
	case bson.TypeJavaScript:
		str, err = vr.ReadJavascript()
		if err != nil {
			return err
		}
	case bson.TypeSymbol:
		str, err = vr.ReadSymbol()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot decode %v into a string type", vr.Type())
	}

	switch t := i.(type) {
	case *string:
		if t == nil {
			return errors.New("StringDecodeValue can only be used to decode non-nil *string")
		}
		*t = str
		return nil
	case *bson.JavaScriptCode:
		if t == nil {
			return errors.New("StringDecodeValue can only be used to decode non-nil *JavaScriptCode")
		}
		*t = bson.JavaScriptCode(str)
		return nil
	case *bson.Symbol:
		if t == nil {
			return errors.New("StringDecodeValue can only be used to decode non-nil *Symbol")
		}
		*t = bson.Symbol(str)
		return nil
	}

	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || !val.Elem().CanSet() {
		return errors.New("StringDecodeValue can only be used to decode settable (non-nil) values")
	}
	val = val.Elem()

	if val.Type().Kind() != reflect.String {
		return ValueDecoderError{
			Name:     "StringDecodeValue",
			Types:    []interface{}{(*string)(nil), (*bson.JavaScriptCode)(nil), (*bson.Symbol)(nil)},
			Received: i,
		}
	}

	val.SetString(str)
	return nil
}

// DocumentDecodeValue is the ValueDecoderFunc for *bson.Document.
func (dvd DefaultValueDecoders) DocumentDecodeValue(dctx DecodeContext, vr ValueReader, i interface{}) error {
	doc, ok := i.(**bson.Document)
	if !ok {
		return ValueDecoderError{Name: "DocumentDecodeValue", Types: []interface{}{(**bson.Document)(nil)}, Received: i}
	}

	if doc == nil {
		return errors.New("DocumentDecodeValue can only be used to decode non-nil **Document")
	}

	dr, err := vr.ReadDocument()
	if err != nil {
		return err
	}

	return dvd.decodeDocument(dctx, dr, doc)
}

// ArrayDecodeValue is the ValueDecoderFunc for *bson.Array.
func (dvd DefaultValueDecoders) ArrayDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	parr, ok := i.(**bson.Array)
	if !ok {
		return ValueDecoderError{Name: "ArrayDecodeValue", Types: []interface{}{(**bson.Array)(nil)}, Received: i}
	}

	if parr == nil {
		return errors.New("ArrayDecodeValue can only be used to decode non-nil **Array")
	}

	ar, err := vr.ReadArray()
	if err != nil {
		return err
	}

	arr := bson.NewArray()
	for {
		vr, err := ar.ReadValue()
		if err == ErrEOA {
			break
		}
		if err != nil {
			return err
		}

		var val *bson.Value
		err = dvd.valueDecodeValue(dc, vr, &val)
		if err != nil {
			return err
		}

		arr.Append(val)
	}

	*parr = arr
	return nil
}

// BinaryDecodeValue is the ValueDecoderFunc for bson.Binary.
func (dvd DefaultValueDecoders) BinaryDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != bson.TypeBinary {
		return fmt.Errorf("cannot decode %v into a Binary", vr.Type())
	}

	data, subtype, err := vr.ReadBinary()
	if err != nil {
		return err
	}

	if target, ok := i.(*bson.Binary); ok && target != nil {
		*target = bson.Binary{Data: data, Subtype: subtype}
		return nil
	}

	if target, ok := i.(**bson.Binary); ok && target != nil {
		pb := *target
		if pb == nil {
			pb = new(bson.Binary)
		}
		*pb = bson.Binary{Data: data, Subtype: subtype}
		*target = pb
		return nil
	}

	return ValueDecoderError{Name: "BinaryDecodeValue", Types: []interface{}{(*bson.Binary)(nil)}, Received: i}
}

// UndefinedDecodeValue is the ValueDecoderFunc for bool types.
func (dvd DefaultValueDecoders) UndefinedDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != bson.TypeUndefined {
		return fmt.Errorf("cannot decode %v into an Undefined", vr.Type())
	}

	target, ok := i.(*bson.Undefinedv2)
	if !ok || target == nil {
		return ValueDecoderError{Name: "UndefinedDecodeValue", Types: []interface{}{(*bson.Undefinedv2)(nil)}, Received: i}
	}

	*target = bson.Undefinedv2{}
	return vr.ReadUndefined()
}

// ObjectIDDecodeValue is the ValueDecoderFunc for objectid.ObjectID.
func (dvd DefaultValueDecoders) ObjectIDDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != bson.TypeObjectID {
		return fmt.Errorf("cannot decode %v into an ObjectID", vr.Type())
	}

	target, ok := i.(*objectid.ObjectID)
	if !ok || target == nil {
		return ValueDecoderError{Name: "ObjectIDDecodeValue", Types: []interface{}{(*objectid.ObjectID)(nil)}, Received: i}
	}

	oid, err := vr.ReadObjectID()
	if err != nil {
		return err
	}

	*target = oid
	return nil
}

// DateTimeDecodeValue is the ValueDecoderFunc for bson.DateTime.
func (dvd DefaultValueDecoders) DateTimeDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != bson.TypeDateTime {
		return fmt.Errorf("cannot decode %v into a DateTime", vr.Type())
	}

	target, ok := i.(*bson.DateTime)
	if !ok || target == nil {
		return ValueDecoderError{Name: "DateTimeDecodeValue", Types: []interface{}{(*bson.DateTime)(nil)}, Received: i}
	}

	dt, err := vr.ReadDateTime()
	if err != nil {
		return err
	}

	*target = bson.DateTime(dt)
	return nil
}

// NullDecodeValue is the ValueDecoderFunc for bson.Null.
func (dvd DefaultValueDecoders) NullDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != bson.TypeNull {
		return fmt.Errorf("cannot decode %v into a Null", vr.Type())
	}

	target, ok := i.(*bson.Nullv2)
	if !ok || target == nil {
		return ValueDecoderError{Name: "NullDecodeValue", Types: []interface{}{(*bson.Nullv2)(nil)}, Received: i}
	}

	*target = bson.Nullv2{}
	return vr.ReadNull()
}

// RegexDecodeValue is the ValueDecoderFunc for bson.Regex.
func (dvd DefaultValueDecoders) RegexDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != bson.TypeRegex {
		return fmt.Errorf("cannot decode %v into a Regex", vr.Type())
	}

	target, ok := i.(*bson.Regex)
	if !ok || target == nil {
		return ValueDecoderError{Name: "RegexDecodeValue", Types: []interface{}{(*bson.Regex)(nil)}, Received: i}
	}

	pattern, options, err := vr.ReadRegex()
	if err != nil {
		return err
	}

	*target = bson.Regex{Pattern: pattern, Options: options}
	return nil
}

// DBPointerDecodeValue is the ValueDecoderFunc for bson.DBPointer.
func (dvd DefaultValueDecoders) DBPointerDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != bson.TypeDBPointer {
		return fmt.Errorf("cannot decode %v into a DBPointer", vr.Type())
	}

	target, ok := i.(*bson.DBPointer)
	if !ok || target == nil {
		return ValueDecoderError{Name: "DBPointerDecodeValue", Types: []interface{}{(*bson.DBPointer)(nil)}, Received: i}
	}

	ns, pointer, err := vr.ReadDBPointer()
	if err != nil {
		return err
	}

	*target = bson.DBPointer{DB: ns, Pointer: pointer}
	return nil
}

// CodeWithScopeDecodeValue is the ValueDecoderFunc for bson.CodeWithScope.
func (dvd DefaultValueDecoders) CodeWithScopeDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != bson.TypeCodeWithScope {
		return fmt.Errorf("cannot decode %v into a CodeWithScope", vr.Type())
	}

	target, ok := i.(*bson.CodeWithScope)
	if !ok || target == nil {
		return ValueDecoderError{
			Name:     "CodeWithScopeDecodeValue",
			Types:    []interface{}{(*bson.CodeWithScope)(nil)},
			Received: i,
		}
	}

	code, dr, err := vr.ReadCodeWithScope()
	if err != nil {
		return err
	}

	var scope *bson.Document
	err = dvd.decodeDocument(dc, dr, &scope)
	if err != nil {
		return err
	}

	*target = bson.CodeWithScope{Code: code, Scope: scope}
	return nil
}

// TimestampDecodeValue is the ValueDecoderFunc for bson.Timestamp.
func (dvd DefaultValueDecoders) TimestampDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != bson.TypeTimestamp {
		return fmt.Errorf("cannot decode %v into a Timestamp", vr.Type())
	}

	target, ok := i.(*bson.Timestamp)
	if !ok || target == nil {
		return ValueDecoderError{Name: "TimestampDecodeValue", Types: []interface{}{(*bson.Timestamp)(nil)}, Received: i}
	}

	t, incr, err := vr.ReadTimestamp()
	if err != nil {
		return err
	}

	*target = bson.Timestamp{T: t, I: incr}
	return nil
}

// Decimal128DecodeValue is the ValueDecoderFunc for decimal.Decimal128.
func (dvd DefaultValueDecoders) Decimal128DecodeValue(dctx DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != bson.TypeDecimal128 {
		return fmt.Errorf("cannot decode %v into a decimal.Decimal128", vr.Type())
	}

	target, ok := i.(*decimal.Decimal128)
	if !ok || target == nil {
		return ValueDecoderError{Name: "Decimal128DecodeValue", Types: []interface{}{(*decimal.Decimal128)(nil)}, Received: i}
	}

	d128, err := vr.ReadDecimal128()
	if err != nil {
		return err
	}

	*target = d128
	return nil
}

// MinKeyDecodeValue is the ValueDecoderFunc for bson.MinKey.
func (dvd DefaultValueDecoders) MinKeyDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != bson.TypeMinKey {
		return fmt.Errorf("cannot decode %v into a MinKey", vr.Type())
	}

	target, ok := i.(*bson.MinKeyv2)
	if !ok || target == nil {
		return ValueDecoderError{Name: "MinKeyDecodeValue", Types: []interface{}{(*bson.MinKeyv2)(nil)}, Received: i}
	}

	*target = bson.MinKeyv2{}
	return vr.ReadMinKey()
}

// MaxKeyDecodeValue is the ValueDecoderFunc for bson.MaxKey.
func (dvd DefaultValueDecoders) MaxKeyDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != bson.TypeMaxKey {
		return fmt.Errorf("cannot decode %v into a MaxKey", vr.Type())
	}

	target, ok := i.(*bson.MaxKeyv2)
	if !ok || target == nil {
		return ValueDecoderError{Name: "MaxKeyDecodeValue", Types: []interface{}{(*bson.MaxKeyv2)(nil)}, Received: i}
	}

	*target = bson.MaxKeyv2{}
	return vr.ReadMaxKey()
}

// ValueDecodeValue is the ValueDecoderFunc for *bson.Value.
func (dvd DefaultValueDecoders) ValueDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	pval, ok := i.(**bson.Value)
	if !ok {
		return ValueDecoderError{Name: "ValueDecodeValue", Types: []interface{}{(**bson.Value)(nil)}, Received: i}
	}

	if pval == nil {
		return errors.New("ValueDecodeValue can only be used to decode non-nil **Value")
	}

	return dvd.valueDecodeValue(dc, vr, pval)
}

// JSONNumberDecodeValue is the ValueDecoderFunc for json.Number.
func (dvd DefaultValueDecoders) JSONNumberDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	target, ok := i.(*json.Number)
	if !ok || target == nil {
		return ValueDecoderError{Name: "JSONNumberDecodeValue", Types: []interface{}{(*json.Number)(nil)}, Received: i}
	}

	switch vr.Type() {
	case bson.TypeDouble:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		*target = json.Number(strconv.FormatFloat(f64, 'g', -1, 64))
	case bson.TypeInt32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		*target = json.Number(strconv.FormatInt(int64(i32), 10))
	case bson.TypeInt64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return err
		}
		*target = json.Number(strconv.FormatInt(i64, 10))
	default:
		return fmt.Errorf("cannot decode %v into a json.Number", vr.Type())
	}

	return nil
}

// URLDecodeValue is the ValueDecoderFunc for url.URL.
func (dvd DefaultValueDecoders) URLDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != bson.TypeString {
		return fmt.Errorf("cannot decode %v into a *url.URL", vr.Type())
	}

	str, err := vr.ReadString()
	if err != nil {
		return err
	}

	u, err := url.Parse(str)
	if err != nil {
		return err
	}

	err = ValueDecoderError{Name: "URLDecodeValue", Types: []interface{}{(*url.URL)(nil), (**url.URL)(nil)}, Received: i}

	// It's valid to use either a *url.URL or a url.URL
	switch target := i.(type) {
	case *url.URL:
		if target == nil {
			return err
		}
		*target = *u
	case **url.URL:
		if target == nil {
			return err
		}
		*target = u
	default:
		return err
	}
	return nil
}

// TimeDecodeValue is the ValueDecoderFunc for time.Time.
func (dvd DefaultValueDecoders) TimeDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != bson.TypeDateTime {
		return fmt.Errorf("cannot decode %v into a time.Time", vr.Type())
	}

	dt, err := vr.ReadDateTime()
	if err != nil {
		return err
	}

	if target, ok := i.(*time.Time); ok && target != nil {
		*target = time.Unix(dt/1000, dt%1000*1000000)
		return nil
	}

	if target, ok := i.(**time.Time); ok && target != nil {
		tt := *target
		if tt == nil {
			tt = new(time.Time)
		}
		*tt = time.Unix(dt/1000, dt%1000*1000000)
		*target = tt
		return nil
	}

	return ValueDecoderError{
		Name:     "TimeDecodeValue",
		Types:    []interface{}{(*time.Time)(nil), (**time.Time)(nil)},
		Received: i,
	}
}

// ReaderDecodeValue is the ValueDecoderFunc for bson.Reader.
func (dvd DefaultValueDecoders) ReaderDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	rdr, ok := i.(*bson.Reader)
	if !ok {
		return ValueDecoderError{Name: "ReaderDecodeValue", Types: []interface{}{(*bson.Reader)(nil)}, Received: i}
	}

	if rdr == nil {
		return errors.New("ReaderDecodeValue can only be used to decode non-nil *Reader")
	}

	if *rdr == nil {
		*rdr = make(bson.Reader, 0)
	} else {
		*rdr = (*rdr)[:0]
	}

	var err error
	*rdr, err = Copier{r: dc.Registry}.AppendDocumentBytes(*rdr, vr)
	return err
}

// ByteSliceDecodeValue is the ValueDecoderFunc for []byte.
func (dvd DefaultValueDecoders) ByteSliceDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != bson.TypeBinary {
		return fmt.Errorf("cannot decode %v into a *[]byte", vr.Type())
	}

	target, ok := i.(*[]byte)
	if !ok || target == nil {
		return ValueDecoderError{Name: "ByteSliceDecodeValue", Types: []interface{}{(*[]byte)(nil)}, Received: i}
	}

	data, subtype, err := vr.ReadBinary()
	if err != nil {
		return err
	}
	if subtype != 0x00 {
		return fmt.Errorf("ByteSliceDecodeValue can only be used to decode subtype 0x00 for %s, got %v", bson.TypeBinary, subtype)
	}

	*target = data
	return nil
}

// ElementSliceDecodeValue is the ValueDecoderFunc for []*bson.Element.
func (dvd DefaultValueDecoders) ElementSliceDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	dr, err := vr.ReadDocument()
	if err != nil {
		return err
	}
	elems := make([]*bson.Element, 0)
	for {
		key, vr, err := dr.ReadElement()
		if err == ErrEOD {
			break
		}
		if err != nil {
			return err
		}

		var elem *bson.Element
		err = dvd.elementDecodeValue(dc, vr, key, &elem)
		if err != nil {
			return err
		}

		elems = append(elems, elem)
	}

	target, ok := i.(*[]*bson.Element)
	if !ok || target == nil {
		return ValueDecoderError{Name: "ElementSliceDecodeValue", Types: []interface{}{(*[]*bson.Element)(nil)}, Received: i}
	}

	*target = elems
	return nil
}

// MapDecodeValue is the ValueDecoderFunc for map[string]* types.
func (dvd DefaultValueDecoders) MapDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("MapDecodeValue can only be used to decode non-nil pointers to map values, got %T", i)
	}

	if val.Elem().Kind() != reflect.Map || val.Elem().Type().Key().Kind() != reflect.String || !val.Elem().CanSet() {
		return errors.New("MapDecodeValue can only decode settable maps with string keys")
	}

	dr, err := vr.ReadDocument()
	if err != nil {
		return err
	}

	if val.Elem().IsNil() {
		val.Elem().Set(reflect.MakeMap(val.Elem().Type()))
	}

	mVal := val.Elem()

	dFn, err := dvd.decodeFn(dc, mVal)
	if err != nil {
		return err
	}

	for {
		var elem reflect.Value
		key, vr, err := dr.ReadElement()
		if err == ErrEOD {
			break
		}
		if err != nil {
			return err
		}
		key, elem, err = dFn(dc, vr, key)
		if err != nil {
			return err
		}

		mVal.SetMapIndex(reflect.ValueOf(key), elem)
	}
	return err
}

type decodeFn func(dc DecodeContext, vr ValueReader, key string) (updatedKey string, v reflect.Value, err error)

// decodeFn returns a function that can be used to decode the values of a map.
// The mapVal parameter should be a map type, not a pointer to a map type.
//
// If error is nil, decodeFn will return a non-nil decodeFn.
func (dvd DefaultValueDecoders) decodeFn(dc DecodeContext, mapVal reflect.Value) (decodeFn, error) {
	var dFn decodeFn
	switch mapVal.Type().Elem() {
	case tElement:
		// TODO(skriptble): We have to decide if we want to support this. We have
		// information loss because we can only store either the map key or the element
		// key. We could add a struct tag field that allows the user to make a decision.
		dFn = func(dc DecodeContext, vr ValueReader, key string) (string, reflect.Value, error) {
			var elem *bson.Element
			err := dvd.elementDecodeValue(dc, vr, key, &elem)
			if err != nil {
				return key, reflect.Value{}, err
			}
			return key, reflect.ValueOf(elem), nil
		}
	default:
		eType := mapVal.Type().Elem()
		decoder, err := dc.LookupDecoder(eType)
		if err != nil {
			return nil, err
		}

		dFn = func(dc DecodeContext, vr ValueReader, key string) (string, reflect.Value, error) {
			ptr := reflect.New(eType)

			err = decoder.DecodeValue(dc, vr, ptr.Interface())
			if err != nil {
				return key, reflect.Value{}, err
			}
			return key, ptr.Elem(), nil
		}
	}

	return dFn, nil
}

// SliceDecodeValue is the ValueDecoderFunc for []* types.
func (dvd DefaultValueDecoders) SliceDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("SliceDecodeValue can only be used to decode non-nil pointers to slice or array values, got %T", i)
	}

	switch val.Elem().Kind() {
	case reflect.Slice, reflect.Array:
		if !val.Elem().CanSet() {
			return errors.New("SliceDecodeValue can only decode settable slice and array values")
		}
	default:
		return fmt.Errorf("SliceDecodeValue can only decode settable slice and array values, got %T", i)
	}

	switch vr.Type() {
	case bson.TypeArray:
	case bson.TypeNull:
		if val.Elem().Kind() != reflect.Slice {
			return fmt.Errorf("cannot decode %v into an array", vr.Type())
		}
		null := reflect.Zero(val.Elem().Type())
		val.Elem().Set(null)
		return vr.ReadNull()
	default:
		return fmt.Errorf("cannot decode %v into a slice", vr.Type())
	}

	eType := val.Type().Elem().Elem()

	ar, err := vr.ReadArray()
	if err != nil {
		return err
	}

	var elems []reflect.Value
	switch eType {
	case tElement:
		elems, err = dvd.decodeElement(dc, ar)
	default:
		elems, err = dvd.decodeDefault(dc, ar, eType)
	}

	if err != nil {
		return err
	}

	switch val.Elem().Kind() {
	case reflect.Slice:
		slc := reflect.MakeSlice(val.Elem().Type(), len(elems), len(elems))

		for idx, elem := range elems {
			slc.Index(idx).Set(elem)
		}

		val.Elem().Set(slc)
	case reflect.Array:
		if len(elems) > val.Elem().Len() {
			return fmt.Errorf("more elements returned in array than can fit inside %s", val.Elem().Type())
		}

		for idx, elem := range elems {
			val.Elem().Index(idx).Set(elem)
		}
	}

	return nil
}

// EmptyInterfaceDecodeValue is the ValueDecoderFunc for interface{}.
func (dvd DefaultValueDecoders) EmptyInterfaceDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	target, ok := i.(*interface{})
	if !ok || target == nil {
		return fmt.Errorf("EmptyInterfaceDecodeValue can only be used to decode non-nil *interface{} values, provided type if %T", i)
	}

	// fn is a function we call to assign val back to the target, we do this so
	// we can keep down on the repeated code in this method. In all of the
	// implementations this is a closure, so we don't need to provide the
	// target as a parameter.
	var fn func()
	var val interface{}
	var rtype reflect.Type

	switch vr.Type() {
	case bson.TypeDouble:
		val = new(float64)
		rtype = tFloat64
		fn = func() { *target = *(val.(*float64)) }
	case bson.TypeString:
		val = new(string)
		rtype = tString
		fn = func() { *target = *(val.(*string)) }
	case bson.TypeEmbeddedDocument:
		val = new(*bson.Document)
		rtype = tDocument
		fn = func() { *target = *val.(**bson.Document) }
	case bson.TypeArray:
		val = new(*bson.Array)
		rtype = tArray
		fn = func() { *target = *val.(**bson.Array) }
	case bson.TypeBinary:
		val = new(bson.Binary)
		rtype = tBinary
		fn = func() { *target = *(val.(*bson.Binary)) }
	case bson.TypeUndefined:
		val = new(bson.Undefinedv2)
		rtype = tUndefined
		fn = func() { *target = *(val.(*bson.Undefinedv2)) }
	case bson.TypeObjectID:
		val = new(objectid.ObjectID)
		rtype = tOID
		fn = func() { *target = *(val.(*objectid.ObjectID)) }
	case bson.TypeBoolean:
		val = new(bool)
		rtype = tBool
		fn = func() { *target = *(val.(*bool)) }
	case bson.TypeDateTime:
		val = new(bson.DateTime)
		rtype = tDateTime
		fn = func() { *target = *(val.(*bson.DateTime)) }
	case bson.TypeNull:
		val = new(bson.Nullv2)
		rtype = tNull
		fn = func() { *target = *(val.(*bson.Nullv2)) }
	case bson.TypeRegex:
		val = new(bson.Regex)
		rtype = tRegex
		fn = func() { *target = *(val.(*bson.Regex)) }
	case bson.TypeDBPointer:
		val = new(bson.DBPointer)
		rtype = tDBPointer
		fn = func() { *target = *(val.(*bson.DBPointer)) }
	case bson.TypeJavaScript:
		val = new(bson.JavaScriptCode)
		rtype = tJavaScriptCode
		fn = func() { *target = *(val.(*bson.JavaScriptCode)) }
	case bson.TypeSymbol:
		val = new(bson.Symbol)
		rtype = tSymbol
		fn = func() { *target = *(val.(*bson.Symbol)) }
	case bson.TypeCodeWithScope:
		val = new(bson.CodeWithScope)
		rtype = tCodeWithScope
		fn = func() { *target = *(val.(*bson.CodeWithScope)) }
	case bson.TypeInt32:
		val = new(int32)
		rtype = tInt32
		fn = func() { *target = *(val.(*int32)) }
	case bson.TypeInt64:
		val = new(int64)
		rtype = tInt64
		fn = func() { *target = *(val.(*int64)) }
	case bson.TypeTimestamp:
		val = new(bson.Timestamp)
		rtype = tTimestamp
		fn = func() { *target = *(val.(*bson.Timestamp)) }
	case bson.TypeDecimal128:
		val = new(decimal.Decimal128)
		rtype = tDecimal
		fn = func() { *target = *(val.(*decimal.Decimal128)) }
	case bson.TypeMinKey:
		val = new(bson.MinKeyv2)
		rtype = tMinKey
		fn = func() { *target = *(val.(*bson.MinKeyv2)) }
	case bson.TypeMaxKey:
		val = new(bson.MaxKeyv2)
		rtype = tMaxKey
		fn = func() { *target = *(val.(*bson.MaxKeyv2)) }
	default:
		return fmt.Errorf("Type %s is not a valid BSON type and has no default Go type to decode into", vr.Type())
	}

	decoder, err := dc.LookupDecoder(rtype)
	if err != nil {
		return err
	}
	err = decoder.DecodeValue(dc, vr, val)
	if err != nil {
		return err
	}

	fn()
	return nil
}

// ValueUnmarshalerDecodeValue is the ValueDecoderFunc for ValueUnmarshaler implementations.
func (dvd DefaultValueDecoders) ValueUnmarshalerDecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	val := reflect.ValueOf(i)
	var valueUnmarshaler ValueUnmarshaler
	if val.Kind() == reflect.Ptr && val.IsNil() {
		return fmt.Errorf("ValueUnmarshalerDecodeValue can only unmarshal into non-nil ValueUnmarshaler values, got %T", i)
	}
	if val.Type().Implements(tValueUnmarshaler) {
		valueUnmarshaler = val.Interface().(ValueUnmarshaler)
	} else if val.Type().Kind() == reflect.Ptr && val.Elem().Type().Implements(tValueUnmarshaler) {
		if val.Elem().Kind() == reflect.Ptr && val.Elem().IsNil() {
			val.Elem().Set(reflect.New(val.Type().Elem().Elem()))
		}
		valueUnmarshaler = val.Elem().Interface().(ValueUnmarshaler)
	} else {
		return fmt.Errorf("ValueUnmarshalerDecodeValue can only handle types or pointers to types that are a ValueUnmarshaler, got %T", i)
	}

	t, src, err := Copier{r: dc.Registry}.CopyValueToBytes(vr)
	if err != nil {
		return err
	}

	return valueUnmarshaler.UnmarshalBSONValue(t, src)
}

func (dvd DefaultValueDecoders) decodeElement(dc DecodeContext, ar ArrayReader) ([]reflect.Value, error) {
	elems := make([]reflect.Value, 0)
	for {
		vr, err := ar.ReadValue()
		if err == ErrEOA {
			break
		}
		if err != nil {
			return nil, err
		}

		var elem *bson.Element
		err = dvd.elementDecodeValue(dc, vr, "", &elem)
		if err != nil {
			return nil, err
		}
		elems = append(elems, reflect.ValueOf(elem))
	}

	return elems, nil
}

func (dvd DefaultValueDecoders) decodeDefault(dc DecodeContext, ar ArrayReader, eType reflect.Type) ([]reflect.Value, error) {
	elems := make([]reflect.Value, 0)

	decoder, err := dc.LookupDecoder(eType)
	if err != nil {
		return nil, err
	}

	for {
		vr, err := ar.ReadValue()
		if err == ErrEOA {
			break
		}
		if err != nil {
			return nil, err
		}

		ptr := reflect.New(eType)

		err = decoder.DecodeValue(dc, vr, ptr.Interface())
		if err != nil {
			return nil, err
		}
		elems = append(elems, ptr.Elem())
	}

	return elems, nil
}

func (dvd DefaultValueDecoders) decodeDocument(dctx DecodeContext, dr DocumentReader, pdoc **bson.Document) error {
	doc := bson.NewDocument()
	for {
		key, vr, err := dr.ReadElement()
		if err == ErrEOD {
			break
		}
		if err != nil {
			return err
		}

		var elem *bson.Element
		err = dvd.elementDecodeValue(dctx, vr, key, &elem)
		if err != nil {
			return err
		}

		doc.Append(elem)
	}

	*pdoc = doc
	return nil
}

func (dvd DefaultValueDecoders) elementDecodeValue(dc DecodeContext, vr ValueReader, key string, elem **bson.Element) error {
	switch vr.Type() {
	case bson.TypeDouble:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		*elem = bson.EC.Double(key, f64)
	case bson.TypeString:
		str, err := vr.ReadString()
		if err != nil {
			return err
		}
		*elem = bson.EC.String(key, str)
	case bson.TypeEmbeddedDocument:
		decoder, err := dc.LookupDecoder(tDocument)
		if err != nil {
			return err
		}
		var embeddedDoc *bson.Document
		err = decoder.DecodeValue(dc, vr, &embeddedDoc)
		if err != nil {
			return err
		}
		*elem = bson.EC.SubDocument(key, embeddedDoc)
	case bson.TypeArray:
		decoder, err := dc.LookupDecoder(tArray)
		if err != nil {
			return err
		}
		var arr *bson.Array
		err = decoder.DecodeValue(dc, vr, &arr)
		if err != nil {
			return err
		}
		*elem = bson.EC.Array(key, arr)
	case bson.TypeBinary:
		data, subtype, err := vr.ReadBinary()
		if err != nil {
			return err
		}
		*elem = bson.EC.BinaryWithSubtype(key, data, subtype)
	case bson.TypeUndefined:
		err := vr.ReadUndefined()
		if err != nil {
			return err
		}
		*elem = bson.EC.Undefined(key)
	case bson.TypeObjectID:
		oid, err := vr.ReadObjectID()
		if err != nil {
			return err
		}
		*elem = bson.EC.ObjectID(key, oid)
	case bson.TypeBoolean:
		b, err := vr.ReadBoolean()
		if err != nil {
			return err
		}
		*elem = bson.EC.Boolean(key, b)
	case bson.TypeDateTime:
		dt, err := vr.ReadDateTime()
		if err != nil {
			return err
		}
		*elem = bson.EC.DateTime(key, dt)
	case bson.TypeNull:
		err := vr.ReadNull()
		if err != nil {
			return err
		}
		*elem = bson.EC.Null(key)
	case bson.TypeRegex:
		pattern, options, err := vr.ReadRegex()
		if err != nil {
			return err
		}
		*elem = bson.EC.Regex(key, pattern, options)
	case bson.TypeDBPointer:
		ns, pointer, err := vr.ReadDBPointer()
		if err != nil {
			return err
		}
		*elem = bson.EC.DBPointer(key, ns, pointer)
	case bson.TypeJavaScript:
		js, err := vr.ReadJavascript()
		if err != nil {
			return err
		}
		*elem = bson.EC.JavaScript(key, js)
	case bson.TypeSymbol:
		symbol, err := vr.ReadSymbol()
		if err != nil {
			return err
		}
		*elem = bson.EC.Symbol(key, symbol)
	case bson.TypeCodeWithScope:
		code, scope, err := vr.ReadCodeWithScope()
		if err != nil {
			return err
		}
		scopeDoc := new(*bson.Document)
		err = dvd.decodeDocument(dc, scope, scopeDoc)
		if err != nil {
			return err
		}
		*elem = bson.EC.CodeWithScope(key, code, *scopeDoc)
	case bson.TypeInt32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		*elem = bson.EC.Int32(key, i32)
	case bson.TypeTimestamp:
		t, i, err := vr.ReadTimestamp()
		if err != nil {
			return err
		}
		*elem = bson.EC.Timestamp(key, t, i)
	case bson.TypeInt64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return err
		}
		*elem = bson.EC.Int64(key, i64)
	case bson.TypeDecimal128:
		d128, err := vr.ReadDecimal128()
		if err != nil {
			return err
		}
		*elem = bson.EC.Decimal128(key, d128)
	case bson.TypeMinKey:
		err := vr.ReadMinKey()
		if err != nil {
			return err
		}
		*elem = bson.EC.MinKey(key)
	case bson.TypeMaxKey:
		err := vr.ReadMaxKey()
		if err != nil {
			return err
		}
		*elem = bson.EC.MaxKey(key)
	default:
		return fmt.Errorf("Cannot read unknown BSON type %s", vr.Type())
	}

	return nil
}

func (dvd DefaultValueDecoders) valueDecodeValue(dc DecodeContext, vr ValueReader, val **bson.Value) error {
	switch vr.Type() {
	case bson.TypeDouble:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		*val = bson.VC.Double(f64)
	case bson.TypeString:
		str, err := vr.ReadString()
		if err != nil {
			return err
		}
		*val = bson.VC.String(str)
	case bson.TypeEmbeddedDocument:
		decoder, err := dc.LookupDecoder(tDocument)
		if err != nil {
			return err
		}
		var embeddedDoc *bson.Document
		err = decoder.DecodeValue(dc, vr, &embeddedDoc)
		if err != nil {
			return err
		}
		*val = bson.VC.Document(embeddedDoc)
	case bson.TypeArray:
		decoder, err := dc.LookupDecoder(tArray)
		if err != nil {
			return err
		}
		var arr *bson.Array
		err = decoder.DecodeValue(dc, vr, &arr)
		if err != nil {
			return err
		}
		*val = bson.VC.Array(arr)
	case bson.TypeBinary:
		data, subtype, err := vr.ReadBinary()
		if err != nil {
			return err
		}
		*val = bson.VC.BinaryWithSubtype(data, subtype)
	case bson.TypeUndefined:
		err := vr.ReadUndefined()
		if err != nil {
			return err
		}
		*val = bson.VC.Undefined()
	case bson.TypeObjectID:
		oid, err := vr.ReadObjectID()
		if err != nil {
			return err
		}
		*val = bson.VC.ObjectID(oid)
	case bson.TypeBoolean:
		b, err := vr.ReadBoolean()
		if err != nil {
			return err
		}
		*val = bson.VC.Boolean(b)
	case bson.TypeDateTime:
		dt, err := vr.ReadDateTime()
		if err != nil {
			return err
		}
		*val = bson.VC.DateTime(dt)
	case bson.TypeNull:
		err := vr.ReadNull()
		if err != nil {
			return err
		}
		*val = bson.VC.Null()
	case bson.TypeRegex:
		pattern, options, err := vr.ReadRegex()
		if err != nil {
			return err
		}
		*val = bson.VC.Regex(pattern, options)
	case bson.TypeDBPointer:
		ns, pointer, err := vr.ReadDBPointer()
		if err != nil {
			return err
		}
		*val = bson.VC.DBPointer(ns, pointer)
	case bson.TypeJavaScript:
		js, err := vr.ReadJavascript()
		if err != nil {
			return err
		}
		*val = bson.VC.JavaScript(js)
	case bson.TypeSymbol:
		symbol, err := vr.ReadSymbol()
		if err != nil {
			return err
		}
		*val = bson.VC.Symbol(symbol)
	case bson.TypeCodeWithScope:
		code, scope, err := vr.ReadCodeWithScope()
		if err != nil {
			return err
		}
		scopeDoc := new(*bson.Document)
		err = dvd.decodeDocument(dc, scope, scopeDoc)
		if err != nil {
			return err
		}
		*val = bson.VC.CodeWithScope(code, *scopeDoc)
	case bson.TypeInt32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		*val = bson.VC.Int32(i32)
	case bson.TypeTimestamp:
		t, i, err := vr.ReadTimestamp()
		if err != nil {
			return err
		}
		*val = bson.VC.Timestamp(t, i)
	case bson.TypeInt64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return err
		}
		*val = bson.VC.Int64(i64)
	case bson.TypeDecimal128:
		d128, err := vr.ReadDecimal128()
		if err != nil {
			return err
		}
		*val = bson.VC.Decimal128(d128)
	case bson.TypeMinKey:
		err := vr.ReadMinKey()
		if err != nil {
			return err
		}
		*val = bson.VC.MinKey()
	case bson.TypeMaxKey:
		err := vr.ReadMaxKey()
		if err != nil {
			return err
		}
		*val = bson.VC.MaxKey()
	default:
		return fmt.Errorf("Cannot read unknown BSON type %s", vr.Type())
	}

	return nil
}
