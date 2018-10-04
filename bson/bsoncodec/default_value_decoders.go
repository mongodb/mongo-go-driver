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

	"github.com/mongodb/mongo-go-driver/bson/bsonrw"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

var defaultValueDecoders DefaultValueDecoders

// DefaultValueDecoders is a namespace type for the default ValueDecoders used
// when creating a registry.
type DefaultValueDecoders struct{}

// RegisterDefaultDecoders will register the decoder methods attached to DefaultValueDecoders with
// the provided RegistryBuilder. If rb is nil, a new RegistryBuilder will be created.
//
// There is no support for decoding map[string]interface{} becuase there is no decoder for
// interface{}, so users must either register this decoder themselves or use the
// EmptyInterfaceDecoder avaialble in the bson package.
func (dvd DefaultValueDecoders) RegisterDefaultDecoders(rb *RegistryBuilder) *RegistryBuilder {
	if rb == nil {
		rb = NewEmptyRegistryBuilder()
	}

	return rb.
		RegisterDecoder(reflect.PtrTo(tByteSlice), ValueDecoderFunc(dvd.ByteSliceDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tTime), ValueDecoderFunc(dvd.TimeDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tEmpty), ValueDecoderFunc(dvd.EmptyInterfaceDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tOID), ValueDecoderFunc(dvd.ObjectIDDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tDecimal), ValueDecoderFunc(dvd.Decimal128DecodeValue)).
		RegisterDecoder(reflect.PtrTo(tJSONNumber), ValueDecoderFunc(dvd.JSONNumberDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tURL), ValueDecoderFunc(dvd.URLDecodeValue)).
		RegisterDecoder(tValueUnmarshaler, ValueDecoderFunc(dvd.ValueUnmarshalerDecodeValue)).
		RegisterDefaultDecoder(reflect.Bool, ValueDecoderFunc(dvd.BooleanDecodeValue)).
		RegisterDefaultDecoder(reflect.Int, ValueDecoderFunc(dvd.IntDecodeValue)).
		RegisterDefaultDecoder(reflect.Int8, ValueDecoderFunc(dvd.IntDecodeValue)).
		RegisterDefaultDecoder(reflect.Int16, ValueDecoderFunc(dvd.IntDecodeValue)).
		RegisterDefaultDecoder(reflect.Int32, ValueDecoderFunc(dvd.IntDecodeValue)).
		RegisterDefaultDecoder(reflect.Int64, ValueDecoderFunc(dvd.IntDecodeValue)).
		RegisterDefaultDecoder(reflect.Uint, ValueDecoderFunc(dvd.UintDecodeValue)).
		RegisterDefaultDecoder(reflect.Uint8, ValueDecoderFunc(dvd.UintDecodeValue)).
		RegisterDefaultDecoder(reflect.Uint16, ValueDecoderFunc(dvd.UintDecodeValue)).
		RegisterDefaultDecoder(reflect.Uint32, ValueDecoderFunc(dvd.UintDecodeValue)).
		RegisterDefaultDecoder(reflect.Uint64, ValueDecoderFunc(dvd.UintDecodeValue)).
		RegisterDefaultDecoder(reflect.Float32, ValueDecoderFunc(dvd.FloatDecodeValue)).
		RegisterDefaultDecoder(reflect.Float64, ValueDecoderFunc(dvd.FloatDecodeValue)).
		RegisterDefaultDecoder(reflect.Array, ValueDecoderFunc(dvd.SliceDecodeValue)).
		RegisterDefaultDecoder(reflect.Map, ValueDecoderFunc(dvd.MapDecodeValue)).
		RegisterDefaultDecoder(reflect.Slice, ValueDecoderFunc(dvd.SliceDecodeValue)).
		RegisterDefaultDecoder(reflect.String, ValueDecoderFunc(dvd.StringDecodeValue)).
		RegisterDefaultDecoder(reflect.Struct, &StructCodec{cache: make(map[reflect.Type]*structDescription), parser: DefaultStructTagParser})
}

// BooleanDecodeValue is the ValueDecoderFunc for bool types.
func (dvd DefaultValueDecoders) BooleanDecodeValue(dctx DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.Boolean {
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
func (dvd DefaultValueDecoders) IntDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	var i64 int64
	var err error
	switch vr.Type() {
	case bsontype.Int32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		i64 = int64(i32)
	case bsontype.Int64:
		i64, err = vr.ReadInt64()
		if err != nil {
			return err
		}
	case bsontype.Double:
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
func (dvd DefaultValueDecoders) UintDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	var i64 int64
	var err error
	switch vr.Type() {
	case bsontype.Int32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		i64 = int64(i32)
	case bsontype.Int64:
		i64, err = vr.ReadInt64()
		if err != nil {
			return err
		}
	case bsontype.Double:
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
func (dvd DefaultValueDecoders) FloatDecodeValue(ec DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	var f float64
	var err error
	switch vr.Type() {
	case bsontype.Int32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		f = float64(i32)
	case bsontype.Int64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return err
		}
		f = float64(i64)
	case bsontype.Double:
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
func (dvd DefaultValueDecoders) StringDecodeValue(dctx DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	var str string
	var err error
	switch vr.Type() {
	case bsontype.String:
		str, err = vr.ReadString()
		if err != nil {
			return err
		}
	// TODO(GODRIVER-577): Handle JavaScript and Symbol BSON types when allowed.
	// case bsontype.JavaScript:
	// 	str, err = vr.ReadJavascript()
	// 	if err != nil {
	// 		return err
	// 	}
	// case bsontype.Symbol:
	// 	str, err = vr.ReadSymbol()
	// 	if err != nil {
	// 		return err
	// 	}
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
		// case *bson.JavaScriptCode:
		// 	if t == nil {
		// 		return errors.New("StringDecodeValue can only be used to decode non-nil *JavaScriptCode")
		// 	}
		// 	*t = bson.JavaScriptCode(str)
		// 	return nil
		// case *bson.Symbol:
		// 	if t == nil {
		// 		return errors.New("StringDecodeValue can only be used to decode non-nil *Symbol")
		// 	}
		// 	*t = bson.Symbol(str)
		// 	return nil
	}

	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || !val.Elem().CanSet() {
		return errors.New("StringDecodeValue can only be used to decode settable (non-nil) values")
	}
	val = val.Elem()

	if val.Type().Kind() != reflect.String {
		return ValueDecoderError{
			Name: "StringDecodeValue",
			// Types:    []interface{}{(*string)(nil), (*bson.JavaScriptCode)(nil), (*bson.Symbol)(nil)},
			Types:    []interface{}{(*string)(nil)},
			Received: i,
		}
	}

	val.SetString(str)
	return nil
}

// ObjectIDDecodeValue is the ValueDecoderFunc for objectid.ObjectID.
func (dvd DefaultValueDecoders) ObjectIDDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.ObjectID {
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

// Decimal128DecodeValue is the ValueDecoderFunc for decimal.Decimal128.
func (dvd DefaultValueDecoders) Decimal128DecodeValue(dctx DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.Decimal128 {
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

// JSONNumberDecodeValue is the ValueDecoderFunc for json.Number.
func (dvd DefaultValueDecoders) JSONNumberDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	target, ok := i.(*json.Number)
	if !ok || target == nil {
		return ValueDecoderError{Name: "JSONNumberDecodeValue", Types: []interface{}{(*json.Number)(nil)}, Received: i}
	}

	switch vr.Type() {
	case bsontype.Double:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		*target = json.Number(strconv.FormatFloat(f64, 'g', -1, 64))
	case bsontype.Int32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		*target = json.Number(strconv.FormatInt(int64(i32), 10))
	case bsontype.Int64:
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
func (dvd DefaultValueDecoders) URLDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.String {
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
func (dvd DefaultValueDecoders) TimeDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.DateTime {
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

// ByteSliceDecodeValue is the ValueDecoderFunc for []byte.
func (dvd DefaultValueDecoders) ByteSliceDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.Binary {
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
		return fmt.Errorf("ByteSliceDecodeValue can only be used to decode subtype 0x00 for %s, got %v", bsontype.Binary, subtype)
	}

	*target = data
	return nil
}

// MapDecodeValue is the ValueDecoderFunc for map[string]* types.
func (dvd DefaultValueDecoders) MapDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
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
		if err == bsonrw.ErrEOD {
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

type decodeFn func(dc DecodeContext, vr bsonrw.ValueReader, key string) (updatedKey string, v reflect.Value, err error)

// decodeFn returns a function that can be used to decode the values of a map.
// The mapVal parameter should be a map type, not a pointer to a map type.
//
// If error is nil, decodeFn will return a non-nil decodeFn.
func (dvd DefaultValueDecoders) decodeFn(dc DecodeContext, mapVal reflect.Value) (decodeFn, error) {
	var dFn decodeFn
	switch mapVal.Type().Elem() {
	// case tElement:
	// 	// TODO(skriptble): We have to decide if we want to support this. We have
	// 	// information loss because we can only store either the map key or the element
	// 	// key. We could add a struct tag field that allows the user to make a decision.
	// 	dFn = func(dc DecodeContext, vr bsonrw.ValueReader, key string) (string, reflect.Value, error) {
	// 		var elem *bson.Element
	// 		err := dvd.elementDecodeValue(dc, vr, key, &elem)
	// 		if err != nil {
	// 			return key, reflect.Value{}, err
	// 		}
	// 		return key, reflect.ValueOf(elem), nil
	// 	}
	default:
		eType := mapVal.Type().Elem()
		decoder, err := dc.LookupDecoder(eType)
		if err != nil {
			return nil, err
		}

		dFn = func(dc DecodeContext, vr bsonrw.ValueReader, key string) (string, reflect.Value, error) {
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
func (dvd DefaultValueDecoders) SliceDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
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
	case bsontype.Array:
	case bsontype.Null:
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
	// case tElement:
	// 	elems, err = dvd.decodeElement(dc, ar)
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

// ValueUnmarshalerDecodeValue is the ValueDecoderFunc for ValueUnmarshaler implementations.
//
// TODO(skriptble): Performance test this versus a regular encoder.
func (dvd DefaultValueDecoders) ValueUnmarshalerDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
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

	t, src, err := bsonrw.Copier{}.CopyValueToBytes(vr)
	if err != nil {
		return err
	}

	return valueUnmarshaler.UnmarshalBSONValue(t, src)
}

// EmptyInterfaceDecodeValue is the ValueDecoderFunc for interface{}.
func (dvd DefaultValueDecoders) EmptyInterfaceDecodeValue(dc DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
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
	case bsontype.Double:
		val = new(float64)
		rtype = tFloat64
		fn = func() { *target = *(val.(*float64)) }
	case bsontype.String:
		val = new(string)
		rtype = tString
		fn = func() { *target = *(val.(*string)) }
	// case bsontype.EmbeddedDocument:
	// 	val = new(*bson.Document)
	// 	rtype = tDocument
	// 	fn = func() { *target = *val.(**bson.Document) }
	// case bsontype.Array:
	// 	val = new(*bson.Array)
	// 	rtype = tArray
	// 	fn = func() { *target = *val.(**bson.Array) }
	// case bsontype.Binary:
	// 	val = new(bson.Binary)
	// 	rtype = tBinary
	// 	fn = func() { *target = *(val.(*bson.Binary)) }
	// case bsontype.Undefined:
	// 	val = new(bson.Undefinedv2)
	// 	rtype = tUndefined
	// 	fn = func() { *target = *(val.(*bson.Undefinedv2)) }
	// case bsontype.ObjectID:
	// 	val = new(objectid.ObjectID)
	// 	rtype = tOID
	// 	fn = func() { *target = *(val.(*objectid.ObjectID)) }
	case bsontype.Boolean:
		val = new(bool)
		rtype = tBool
		fn = func() { *target = *(val.(*bool)) }
	// case bsontype.DateTime:
	// 	val = new(bson.DateTime)
	// 	rtype = tDateTime
	// 	fn = func() { *target = *(val.(*bson.DateTime)) }
	// case bsontype.Null:
	// 	val = new(bson.Nullv2)
	// 	rtype = tNull
	// 	fn = func() { *target = *(val.(*bson.Nullv2)) }
	// case bsontype.Regex:
	// 	val = new(bson.Regex)
	// 	rtype = tRegex
	// 	fn = func() { *target = *(val.(*bson.Regex)) }
	// case bsontype.DBPointer:
	// 	val = new(bson.DBPointer)
	// 	rtype = tDBPointer
	// 	fn = func() { *target = *(val.(*bson.DBPointer)) }
	// case bsontype.JavaScript:
	// 	val = new(bson.JavaScriptCode)
	// 	rtype = tJavaScriptCode
	// 	fn = func() { *target = *(val.(*bson.JavaScriptCode)) }
	// case bsontype.Symbol:
	// 	val = new(bson.Symbol)
	// 	rtype = tSymbol
	// 	fn = func() { *target = *(val.(*bson.Symbol)) }
	// case bsontype.CodeWithScope:
	// 	val = new(bson.CodeWithScope)
	// 	rtype = tCodeWithScope
	// 	fn = func() { *target = *(val.(*bson.CodeWithScope)) }
	case bsontype.Int32:
		val = new(int32)
		rtype = tInt32
		fn = func() { *target = *(val.(*int32)) }
	case bsontype.Int64:
		val = new(int64)
		rtype = tInt64
		fn = func() { *target = *(val.(*int64)) }
	// case bsontype.Timestamp:
	// 	val = new(bson.Timestamp)
	// 	rtype = tTimestamp
	// 	fn = func() { *target = *(val.(*bson.Timestamp)) }
	case bsontype.Decimal128:
		val = new(decimal.Decimal128)
		rtype = tDecimal
		fn = func() { *target = *(val.(*decimal.Decimal128)) }
	// case bsontype.MinKey:
	// 	val = new(bson.MinKeyv2)
	// 	rtype = tMinKey
	// 	fn = func() { *target = *(val.(*bson.MinKeyv2)) }
	// case bsontype.MaxKey:
	// 	val = new(bson.MaxKeyv2)
	// 	rtype = tMaxKey
	// 	fn = func() { *target = *(val.(*bson.MaxKeyv2)) }
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

func (dvd DefaultValueDecoders) decodeDefault(dc DecodeContext, ar bsonrw.ArrayReader, eType reflect.Type) ([]reflect.Value, error) {
	elems := make([]reflect.Value, 0)

	decoder, err := dc.LookupDecoder(eType)
	if err != nil {
		return nil, err
	}

	for {
		vr, err := ar.ReadValue()
		if err == bsonrw.ErrEOA {
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
