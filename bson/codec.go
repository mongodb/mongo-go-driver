package bson

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

var defaultBoolCodec = &BooleanCodec{}
var defaultIntCodec = &IntCodec{}
var defaultUintCodec = &UintCodec{}
var defaultFloatCodec = &FloatCodec{}
var defaultStringCodec = &StringCodec{}
var defaultDocumentCodec = &DocumentCodec{}
var defaultArrayCodec = &ArrayCodec{}
var defaultTimeCodec = &TimeCodec{}
var defaultElementCodec = &elementCodec{}
var defaultValueCodec = &ValueCodec{}
var defaultByteSliceCodec = &ByteSliceCodec{}
var defaultBinaryCodec = &BinaryCodec{}
var defaultUndefinedCodec = &UndefinedCodec{}
var defaultObjectIDCodec = &ObjectIDCodec{}
var defaultDateTimeCodec = &DateTimeCodec{}
var defaultNullCodec = &NullCodec{}
var defaultRegexCodec = &RegexCodec{}
var defaultDBPointerCodec = &DBPointerCodec{}
var defaultCodeWithScopeCodec = &CodeWithScopeCodec{}
var defaultTimestampCodec = &TimestampCodec{}
var defaultDecimal128Codec = &Decimal128Codec{}
var defaultMinKeyCodec = &MinKeyCodec{}
var defaultMaxKeyCodec = &MaxKeyCodec{}
var defaultJSONNumberCodec = &JSONNumberCodec{}
var defaultURLCodec = &URLCodec{}
var defaultReaderCodec = &ReaderCodec{}
var defaultElementSliceCodec = &ElementSliceCodec{}

var ptBool = reflect.TypeOf((*bool)(nil))
var ptInt8 = reflect.TypeOf((*int8)(nil))
var ptInt16 = reflect.TypeOf((*int16)(nil))
var ptInt32 = reflect.TypeOf((*int32)(nil))
var ptInt64 = reflect.TypeOf((*int64)(nil))
var ptInt = reflect.TypeOf((*int)(nil))
var ptUint8 = reflect.TypeOf((*uint8)(nil))
var ptUint16 = reflect.TypeOf((*uint16)(nil))
var ptUint32 = reflect.TypeOf((*uint32)(nil))
var ptUint64 = reflect.TypeOf((*uint64)(nil))
var ptUint = reflect.TypeOf((*uint)(nil))
var ptFloat32 = reflect.TypeOf((*float32)(nil))
var ptFloat64 = reflect.TypeOf((*float64)(nil))
var ptString = reflect.TypeOf((*string)(nil))

// CodecEncodeError is an error returned from a Codec's EncodeValue method when
// the provided value can't be encoded with the given Codec.
type CodecEncodeError struct {
	Codec    interface{}
	Types    []interface{}
	Received interface{}
}

func (cee CodecEncodeError) Error() string {
	types := make([]string, 0, len(cee.Types))
	for _, t := range cee.Types {
		types = append(types, fmt.Sprintf("%T", t))
	}
	return fmt.Sprintf("%T can only process %s, but got a %T", cee.Codec, strings.Join(types, ", "), cee.Received)
}

// CodecDecodeError is an error returned from a Codec's DecodeValue method when
// the provided value can't be decoded with the given Codec.
type CodecDecodeError struct {
	Codec    interface{}
	Types    []interface{}
	Received interface{}
}

func (dee CodecDecodeError) Error() string {
	types := make([]string, 0, len(dee.Types))
	for _, t := range dee.Types {
		types = append(types, fmt.Sprintf("%T", t))
	}
	return fmt.Sprintf("%T can only process %s, but got a %T", dee.Codec, strings.Join(types, ", "), dee.Received)
}

// EncodeContext is the contextual information required for a Codec to encode a
// value.
type EncodeContext struct {
	*Registry
	MinSize bool
}

// DecodeContext is the contextual information required for a Codec to decode a
// value.
type DecodeContext struct {
	*Registry
	Truncate bool
}

// Codec implementations handle encoding and decoding values. They can be
// registered in a registry which will handle invoking them. Callers of the
// DecodeValue methods pass in a pointer to the value, and
// implementations operate on a pointer to the value. This is true of pointer
// values as well, so a caller of DecodeValue for a pointer type *Foo will pass
// in **Foo.
type Codec interface {
	EncodeValue(EncodeContext, ValueWriter, interface{}) error
	DecodeValue(DecodeContext, ValueReader, interface{}) error
}

// CodecZeroer is the interface implemented by Codecs that can also determine if
// a value of the type that would be encoded is zero.
type CodecZeroer interface {
	Codec
	IsZero(interface{}) bool
}

// BooleanCodec is the Codec used for bool values.
type BooleanCodec struct{}

var _ Codec = &BooleanCodec{}

// EncodeValue implements the Codec interface.
func (bc *BooleanCodec) EncodeValue(ectx EncodeContext, vw ValueWriter, i interface{}) error {
	b, ok := i.(bool)
	if !ok {
		if reflect.TypeOf(i).Kind() != reflect.Bool {
			return CodecEncodeError{Codec: bc, Types: []interface{}{bool(true)}, Received: i}
		}

		b = reflect.ValueOf(i).Bool()
	}

	return vw.WriteBoolean(b)
}

// DecodeValue implements the Codec interface.
func (bc *BooleanCodec) DecodeValue(dctx DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != TypeBoolean {
		return fmt.Errorf("cannot decode %v into a boolean", vr.Type())
	}

	var err error
	if target, ok := i.(*bool); ok && target != nil { // if it is nil, we go the slow path.
		*target, err = vr.ReadBoolean()
		return err
	}

	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || !val.Elem().CanSet() {
		return fmt.Errorf("%T can only be used to decode settable (non-nil) values", bc)
	}
	val = val.Elem()
	if val.Type().Kind() != reflect.Bool {
		return CodecDecodeError{Codec: bc, Types: []interface{}{bool(true)}, Received: i}
	}

	b, err := vr.ReadBoolean()
	val.SetBool(b)
	return err
}

// IntCodec is the Codec used for int8, int16, int32, int64, and int values.
type IntCodec struct{}

var _ Codec = &IntCodec{}

// EncodeValue implements the Codec interface.
func (ic *IntCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	switch t := i.(type) {
	case int8:
		return vw.WriteInt32(int32(t))
	case int16:
		return vw.WriteInt32(int32(t))
	case int32:
		return vw.WriteInt32(t)
	case int64:
		if ec.MinSize && t <= math.MaxInt32 {
			return vw.WriteInt32(int32(t))
		}
		return vw.WriteInt64(t)
	case int:
		if ec.MinSize && t <= math.MaxInt32 {
			return vw.WriteInt32(int32(t))
		}
		return vw.WriteInt64(int64(t))
	}

	val := reflect.ValueOf(i)
	switch val.Type().Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32:
		return vw.WriteInt32(int32(val.Int()))
	case reflect.Int, reflect.Int64:
		i64 := val.Int()
		if ec.MinSize && i64 <= math.MaxInt32 {
			return vw.WriteInt32(int32(i64))
		}
		return vw.WriteInt64(i64)
	}

	return CodecEncodeError{Codec: ic, Types: []interface{}{int8(0), int16(0), int32(0), int64(0), int(0)}, Received: i}
}

// DecodeValue implements the Codec interface.
func (ic *IntCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	var i64 int64
	var err error
	switch vr.Type() {
	case TypeInt32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		i64 = int64(i32)
	case TypeInt64:
		i64, err = vr.ReadInt64()
		if err != nil {
			return err
		}
	case TypeDouble:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		if !dc.Truncate && math.Floor(f64) != f64 {
			return fmt.Errorf("%T can only truncate float64 to an integer type when truncation is enabled", ic)
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
			return fmt.Errorf("%T can only be used to decode non-nil *int8", ic)
		}
		if i64 < math.MinInt8 || i64 > math.MaxInt8 {
			return fmt.Errorf("%d overflows int8", i64)
		}
		*target = int8(i64)
		return nil
	case *int16:
		if target == nil {
			return fmt.Errorf("%T can only be used to decode non-nil *int16", ic)
		}
		if i64 < math.MinInt16 || i64 > math.MaxInt16 {
			return fmt.Errorf("%d overflows int16", i64)
		}
		*target = int16(i64)
		return nil
	case *int32:
		if target == nil {
			return fmt.Errorf("%T can only be used to decode non-nil *int32", ic)
		}
		if i64 < math.MinInt32 || i64 > math.MaxInt32 {
			return fmt.Errorf("%d overflows int32", i64)
		}
		*target = int32(i64)
		return nil
	case *int64:
		if target == nil {
			return fmt.Errorf("%T can only be used to decode non-nil *int64", ic)
		}
		*target = int64(i64)
		return nil
	case *int:
		if target == nil {
			return fmt.Errorf("%T can only be used to decode non-nil *int", ic)
		}
		if int64(int(i64)) != i64 { // Can we fit this inside of an int
			return fmt.Errorf("%d overflows int", i64)
		}
		*target = int(i64)
		return nil
	}

	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || !val.Elem().CanSet() {
		return fmt.Errorf("%T can only be used to decode settable (non-nil) values", ic)
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
		return CodecDecodeError{
			Codec:    ic,
			Types:    []interface{}{(*int8)(nil), (*int16)(nil), (*int32)(nil), (*int64)(nil), (*int)(nil)},
			Received: i,
		}
	}

	val.SetInt(i64)
	return nil
}

// UintCodec is the Codec used for uint8, uint16, uint32, uint64, and uint
// values.
type UintCodec struct{}

var _ Codec = &UintCodec{}

// EncodeValue implements the Codec interface.
func (uc *UintCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	switch t := i.(type) {
	case uint8:
		return vw.WriteInt32(int32(t))
	case uint16:
		return vw.WriteInt32(int32(t))
	case uint:
		if ec.MinSize && t <= math.MaxInt32 {
			return vw.WriteInt32(int32(t))
		}
		if uint64(t) > math.MaxInt64 {
			return fmt.Errorf("%d overflows int64", t)
		}
		return vw.WriteInt64(int64(t))
	case uint32:
		if ec.MinSize && t <= math.MaxInt32 {
			return vw.WriteInt32(int32(t))
		}
		return vw.WriteInt64(int64(t))
	case uint64:
		if ec.MinSize && t <= math.MaxInt32 {
			return vw.WriteInt32(int32(t))
		}
		if t > math.MaxInt64 {
			return fmt.Errorf("%d overflows int64", t)
		}
		return vw.WriteInt64(int64(t))
	}

	val := reflect.ValueOf(i)
	switch val.Type().Kind() {
	case reflect.Uint8, reflect.Uint16:
		return vw.WriteInt32(int32(val.Uint()))
	case reflect.Uint, reflect.Uint32, reflect.Uint64:
		u64 := val.Uint()
		if ec.MinSize && u64 <= math.MaxInt32 {
			return vw.WriteInt32(int32(u64))
		}
		if u64 > math.MaxInt64 {
			return fmt.Errorf("%d overflows int64", u64)
		}
		return vw.WriteInt64(int64(u64))
	}

	return CodecEncodeError{Codec: uc, Types: []interface{}{uint8(0), uint16(0), uint32(0), uint64(0), uint(0)}, Received: i}
}

// DecodeValue implements the Codec interface.
func (uc *UintCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	var i64 int64
	var err error
	switch vr.Type() {
	case TypeInt32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		i64 = int64(i32)
	case TypeInt64:
		i64, err = vr.ReadInt64()
		if err != nil {
			return err
		}
	case TypeDouble:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		if !dc.Truncate && math.Floor(f64) != f64 {
			return fmt.Errorf("%T can only truncate float64 to an integer type when truncation is enabled", uc)
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
			return fmt.Errorf("%T can only be used to decode non-nil *uint8", uc)
		}
		if i64 < 0 || i64 > math.MaxUint8 {
			return fmt.Errorf("%d overflows uint8", i64)
		}
		*target = uint8(i64)
		return nil
	case *uint16:
		if target == nil {
			return fmt.Errorf("%T can only be used to decode non-nil *uint16", uc)
		}
		if i64 < 0 || i64 > math.MaxUint16 {
			return fmt.Errorf("%d overflows uint16", i64)
		}
		*target = uint16(i64)
		return nil
	case *uint32:
		if target == nil {
			return fmt.Errorf("%T can only be used to decode non-nil *uint32", uc)
		}
		if i64 < 0 || i64 > math.MaxUint32 {
			return fmt.Errorf("%d overflows uint32", i64)
		}
		*target = uint32(i64)
		return nil
	case *uint64:
		if target == nil {
			return fmt.Errorf("%T can only be used to decode non-nil *uint64", uc)
		}
		if i64 < 0 {
			return fmt.Errorf("%d overflows uint64", i64)
		}
		*target = uint64(i64)
		return nil
	case *uint:
		if target == nil {
			return fmt.Errorf("%T can only be used to decode non-nil *uint", uc)
		}
		if i64 < 0 || int64(uint(i64)) != i64 { // Can we fit this inside of an uint
			return fmt.Errorf("%d overflows uint", i64)
		}
		*target = uint(i64)
		return nil
	}

	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || !val.Elem().CanSet() {
		return fmt.Errorf("%T can only be used to decode settable (non-nil) values", uc)
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
		return CodecDecodeError{
			Codec:    uc,
			Types:    []interface{}{(*uint8)(nil), (*uint16)(nil), (*uint32)(nil), (*uint64)(nil), (*uint)(nil)},
			Received: i,
		}
	}

	val.SetUint(uint64(i64))
	return nil
}

// FloatCodec is the Codec used for float32 and float64 values.
type FloatCodec struct{}

var _ Codec = &FloatCodec{}

// EncodeValue implements the Codec interface.
func (fc *FloatCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	switch t := i.(type) {
	case float32:
		return vw.WriteDouble(float64(t))
	case float64:
		return vw.WriteDouble(t)
	}

	val := reflect.ValueOf(i)
	switch val.Type().Kind() {
	case reflect.Float32, reflect.Float64:
		return vw.WriteDouble(val.Float())
	}

	return CodecEncodeError{Codec: fc, Types: []interface{}{float32(0), float64(0)}, Received: i}
}

// DecodeValue implements the Codec interface.
func (fc *FloatCodec) DecodeValue(ec DecodeContext, vr ValueReader, i interface{}) error {
	var f float64
	var err error
	switch vr.Type() {
	case TypeInt32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		f = float64(i32)
	case TypeInt64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return err
		}
		f = float64(i64)
	case TypeDouble:
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
			return fmt.Errorf("%T can only be used to decode non-nil *float32", fc)
		}
		if !ec.Truncate && float64(float32(f)) != f {
			return fmt.Errorf("%T can only convert float64 to float32 when truncation is allowed", fc)
		}
		*target = float32(f)
		return nil
	case *float64:
		if target == nil {
			return fmt.Errorf("%T can only be used to decode non-nil *float64", fc)
		}
		*target = f
		return nil
	}

	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || !val.Elem().CanSet() {
		return fmt.Errorf("%T can only be used to decode settable (non-nil) values", fc)
	}
	val = val.Elem()

	switch val.Type().Kind() {
	case reflect.Float32:
		if !ec.Truncate && float64(float32(f)) != f {
			return fmt.Errorf("%T can only convert float64 to float32 when truncation is allowed", fc)
		}
	case reflect.Float64:
	default:
		return CodecDecodeError{Codec: fc, Types: []interface{}{(*float32)(nil), (*float64)(nil)}, Received: i}
	}

	val.SetFloat(f)
	return nil
}

// StringCodec is the Codec used for string values.
type StringCodec struct{}

var _ Codec = &StringCodec{}

// EncodeValue implements the Codec interface.
func (sc *StringCodec) EncodeValue(ectx EncodeContext, vw ValueWriter, i interface{}) error {
	switch t := i.(type) {
	case string:
		return vw.WriteString(t)
	case JavaScriptCode:
		return vw.WriteJavascript(string(t))
	case Symbol:
		return vw.WriteSymbol(string(t))
	}

	val := reflect.ValueOf(i)
	if val.Type().Kind() != reflect.String {
		return CodecEncodeError{Codec: sc, Types: []interface{}{string(""), JavaScriptCode(""), Symbol("")}, Received: i}
	}

	return vw.WriteString(val.String())
}

// DecodeValue implements the Codec interface.
func (sc *StringCodec) DecodeValue(dctx DecodeContext, vr ValueReader, i interface{}) error {
	var str string
	var err error
	switch vr.Type() {
	case TypeString:
		str, err = vr.ReadString()
		if err != nil {
			return err
		}
	case TypeJavaScript:
		str, err = vr.ReadJavascript()
		if err != nil {
			return err
		}
	case TypeSymbol:
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
			return fmt.Errorf("%T can only be used to decode non-nil *string", sc)
		}
		*t = str
		return nil
	case *JavaScriptCode:
		if t == nil {
			return fmt.Errorf("%T can only be used to decode non-nil *JavaScriptCode", sc)
		}
		*t = JavaScriptCode(str)
		return nil
	case *Symbol:
		if t == nil {
			return fmt.Errorf("%T can only be used to decode non-nil *Symbol", sc)
		}
		*t = Symbol(str)
		return nil
	}

	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || !val.Elem().CanSet() {
		return fmt.Errorf("%T can only be used to decode settable (non-nil) values", sc)
	}
	val = val.Elem()

	if val.Type().Kind() != reflect.String {
		return CodecDecodeError{Codec: sc, Types: []interface{}{(*string)(nil), (*JavaScriptCode)(nil), (*Symbol)(nil)}, Received: i}
	}

	val.SetString(str)
	return nil
}

// DocumentCodec is the Codec used for *Document values.
type DocumentCodec struct{}

var _ Codec = &DocumentCodec{}

// EncodeValue implements the Codec interface.
func (dc *DocumentCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	doc, ok := i.(*Document)
	if !ok {
		return CodecEncodeError{Codec: dc, Types: []interface{}{(*Document)(nil)}, Received: i}
	}

	dw, err := vw.WriteDocument()
	if err != nil {
		return err
	}

	return dc.encodeDocument(ec, dw, doc)
}

// encodeDocument is a separate function that we use because CodeWithScope
// returns us a DocumentWriter and we need to do the same logic that we would do
// for a document but cannot use a Codec.
func (dc DocumentCodec) encodeDocument(ec EncodeContext, dw DocumentWriter, doc *Document) error {
	itr := doc.Iterator()

	for itr.Next() {
		elem := itr.Element()
		dvw, err := dw.WriteDocumentElement(elem.Key())
		if err != nil {
			return err
		}

		val := elem.Value()
		err = defaultValueCodec.encodeValue(ec, dvw, val)

		if err != nil {
			return err
		}
	}

	if err := itr.Err(); err != nil {
		return err
	}

	return dw.WriteDocumentEnd()

}

// DecodeValue implements the Codec interface.
func (dc *DocumentCodec) DecodeValue(dctx DecodeContext, vr ValueReader, i interface{}) error {
	doc, ok := i.(**Document)
	if !ok {
		return CodecDecodeError{Codec: dc, Types: []interface{}{(**Document)(nil)}, Received: i}
	}

	if doc == nil {
		return fmt.Errorf("%T can only be used to decode non-nil **Document", dc)
	}

	dr, err := vr.ReadDocument()
	if err != nil {
		return err
	}

	return dc.decodeDocument(dctx, dr, doc)
}

func (dc DocumentCodec) decodeDocument(dctx DecodeContext, dr DocumentReader, pdoc **Document) error {
	doc := NewDocument()
	for {
		key, vr, err := dr.ReadElement()
		if err == ErrEOD {
			break
		}
		if err != nil {
			return err
		}

		var elem *Element
		err = defaultElementCodec.decodeValue(dctx, vr, key, &elem)
		if err != nil {
			return err
		}

		doc.Append(elem)
	}

	*pdoc = doc
	return nil
}

// ArrayCodec is the Codec used for *Array values.
type ArrayCodec struct{}

var _ Codec = &ArrayCodec{}

// EncodeValue implements the Codec interface.
func (ac *ArrayCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	arr, ok := i.(*Array)
	if !ok {
		return CodecEncodeError{Codec: ac, Types: []interface{}{(*Array)(nil)}, Received: i}
	}

	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	itr := newArrayIterator(arr)

	for itr.Next() {
		val := itr.Value()
		dvw, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		err = defaultValueCodec.encodeValue(ec, dvw, val)

		if err != nil {
			return err
		}
	}

	if err := itr.Err(); err != nil {
		return err
	}

	return aw.WriteArrayEnd()
}

// DecodeValue implements the Codec interface.
func (ac *ArrayCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	parr, ok := i.(**Array)
	if !ok {
		return CodecDecodeError{Codec: ac, Types: []interface{}{(**Array)(nil)}, Received: i}
	}

	if parr == nil {
		return fmt.Errorf("%T can only be used to decode non-nil **Array", ac)
	}

	ar, err := vr.ReadArray()
	if err != nil {
		return err
	}

	arr := NewArray()
	for {
		vr, err := ar.ReadValue()
		if err == ErrEOA {
			break
		}
		if err != nil {
			return err
		}

		var val *Value
		err = defaultValueCodec.decodeValue(dc, vr, &val)
		if err != nil {
			return err
		}

		arr.Append(val)
	}

	*parr = arr
	return nil
}

// BinaryCodec is the Codec used for Binary values.
type BinaryCodec struct{}

var _ Codec = &BinaryCodec{}

// EncodeValue implements the Codec interface.
func (bc *BinaryCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var b Binary
	switch t := i.(type) {
	case Binary:
		b = t
	case *Binary:
		b = *t
	default:
		return CodecEncodeError{Codec: bc, Types: []interface{}{Binary{}, (*Binary)(nil)}, Received: i}
	}

	return vw.WriteBinaryWithSubtype(b.Data, b.Subtype)
}

// DecodeValue implements the Codec interface.
func (bc *BinaryCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != TypeBinary {
		return fmt.Errorf("cannot decode %v into a Binary", vr.Type())
	}

	data, subtype, err := vr.ReadBinary()
	if err != nil {
		return err
	}

	if target, ok := i.(*Binary); ok && target != nil {
		*target = Binary{Data: data, Subtype: subtype}
		return nil
	}

	if target, ok := i.(**Binary); ok && target != nil {
		pb := *target
		if pb == nil {
			pb = new(Binary)
		}
		*pb = Binary{Data: data, Subtype: subtype}
		*target = pb
		return nil
	}

	return fmt.Errorf("%T can only be used to decode non-nil *Binary values, got %T", bc, i)
}

// UndefinedCodec is the Codec for Undefined values.
type UndefinedCodec struct{}

var _ Codec = &UndefinedCodec{}

// EncodeValue implements the Codec interface.
func (uc *UndefinedCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	switch i.(type) {
	case Undefinedv2, *Undefinedv2:
	default:
		return CodecEncodeError{Codec: uc, Types: []interface{}{Undefinedv2{}, (*Undefinedv2)(nil)}, Received: i}
	}

	return vw.WriteUndefined()
}

// DecodeValue implements the Codec interface.
func (uc *UndefinedCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != TypeUndefined {
		return fmt.Errorf("cannot decode %v into an Undefined", vr.Type())
	}

	target, ok := i.(*Undefinedv2)
	if !ok || target == nil {
		return fmt.Errorf("%T can only be used to decode non-nil *Undefined values, got %T", uc, i)
	}

	*target = Undefinedv2{}
	return vr.ReadUndefined()
}

// ObjectIDCodec is the Codec for objectid.ObjectID values.
type ObjectIDCodec struct{}

var _ Codec = &ObjectIDCodec{}

// EncodeValue implements the Codec interface.
func (oidc *ObjectIDCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var oid objectid.ObjectID
	switch t := i.(type) {
	case objectid.ObjectID:
		oid = t
	case *objectid.ObjectID:
		oid = *t
	default:
		return CodecEncodeError{Codec: oidc, Types: []interface{}{objectid.ObjectID{}, (*objectid.ObjectID)(nil)}, Received: i}
	}

	return vw.WriteObjectID(oid)
}

// DecodeValue implements the Codec interface.
func (oidc *ObjectIDCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != TypeObjectID {
		return fmt.Errorf("cannot decode %v into an ObjectID", vr.Type())
	}

	target, ok := i.(*objectid.ObjectID)
	if !ok || target == nil {
		return fmt.Errorf("%T can only be used to decode non-nil *objectid.ObjectID values, got %T", oidc, i)
	}

	oid, err := vr.ReadObjectID()
	if err != nil {
		return err
	}

	*target = oid
	return nil
}

// DateTimeCodec is the Codec for DateTime values.
type DateTimeCodec struct{}

var _ Codec = &DateTimeCodec{}

// EncodeValue implements the Codec interface.
func (dtc *DateTimeCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var dt DateTime
	switch t := i.(type) {
	case DateTime:
		dt = t
	case *DateTime:
		dt = *t
	default:
		return CodecEncodeError{Codec: dtc, Types: []interface{}{DateTime(0), (*DateTime)(nil)}, Received: i}
	}

	return vw.WriteDateTime(int64(dt))
}

// DecodeValue implements the Codec interface.
func (dtc *DateTimeCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != TypeDateTime {
		return fmt.Errorf("cannot decode %v into a DateTime", vr.Type())
	}

	target, ok := i.(*DateTime)
	if !ok || target == nil {
		return fmt.Errorf("%T can only be used to decode non-nil *DateTime values, got %T", dtc, i)
	}

	dt, err := vr.ReadDateTime()
	if err != nil {
		return err
	}

	*target = DateTime(dt)
	return nil
}

// NullCodec is the Codec for Null values.
type NullCodec struct{}

var _ Codec = &NullCodec{}

// EncodeValue implements the Codec interface.
func (nc *NullCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	switch i.(type) {
	case Nullv2, *Nullv2:
	default:
		return CodecEncodeError{Codec: nc, Types: []interface{}{Nullv2{}, (*Nullv2)(nil)}, Received: i}
	}

	return vw.WriteNull()
}

// DecodeValue implements the Codec interface.
func (nc *NullCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != TypeNull {
		return fmt.Errorf("cannot decode %v into a Null", vr.Type())
	}

	target, ok := i.(*Nullv2)
	if !ok || target == nil {
		return fmt.Errorf("%T can only be used to decode non-nil *Null values, got %T", nc, i)
	}

	*target = Nullv2{}
	return vr.ReadNull()
}

// RegexCodec is the Codec for Regex values.
type RegexCodec struct{}

var _ Codec = &RegexCodec{}

// EncodeValue implements the Codec interface.
func (rc *RegexCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var regex Regex
	switch t := i.(type) {
	case Regex:
		regex = t
	case *Regex:
		regex = *t
	default:
		return CodecEncodeError{Codec: rc, Types: []interface{}{Regex{}, (*Regex)(nil)}, Received: i}
	}

	return vw.WriteRegex(regex.Pattern, regex.Options)
}

// DecodeValue implements the Codec interface.
func (rc *RegexCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != TypeRegex {
		return fmt.Errorf("cannot decode %v into a Regex", vr.Type())
	}

	target, ok := i.(*Regex)
	if !ok || target == nil {
		return fmt.Errorf("%T can only be used to decode non-nil *Regex values, got %T", rc, i)
	}

	pattern, options, err := vr.ReadRegex()
	if err != nil {
		return err
	}

	*target = Regex{Pattern: pattern, Options: options}
	return nil
}

// DBPointerCodec is the Codec for DBPointer values.
type DBPointerCodec struct{}

var _ Codec = &DBPointerCodec{}

// EncodeValue implements the Codec interface.
func (dbpc *DBPointerCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var dbp DBPointer
	switch t := i.(type) {
	case DBPointer:
		dbp = t
	case *DBPointer:
		dbp = *t
	default:
		return CodecEncodeError{Codec: dbpc, Types: []interface{}{DBPointer{}, (*DBPointer)(nil)}, Received: i}
	}

	return vw.WriteDBPointer(dbp.DB, dbp.Pointer)
}

// DecodeValue implements the Codec interface.
func (dbpc *DBPointerCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != TypeDBPointer {
		return fmt.Errorf("cannot decode %v into a DBPointer", vr.Type())
	}

	target, ok := i.(*DBPointer)
	if !ok || target == nil {
		return fmt.Errorf("%T can only be used to decode non-nil *DBPointer values, got %T", dbpc, i)
	}

	ns, pointer, err := vr.ReadDBPointer()
	if err != nil {
		return err
	}

	*target = DBPointer{DB: ns, Pointer: pointer}
	return nil
}

// CodeWithScopeCodec is the Codec for CodeWithScope values.
type CodeWithScopeCodec struct{}

var _ Codec = &CodeWithScopeCodec{}

// EncodeValue implements the Codec interface.
func (cwsc *CodeWithScopeCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var cws CodeWithScope
	switch t := i.(type) {
	case CodeWithScope:
		cws = t
	case *CodeWithScope:
		cws = *t
	default:
		return CodecEncodeError{Codec: cwsc, Types: []interface{}{CodeWithScope{}, (*CodeWithScope)(nil)}, Received: i}
	}

	dw, err := vw.WriteCodeWithScope(cws.Code)
	if err != nil {
		return err
	}
	return defaultDocumentCodec.encodeDocument(ec, dw, cws.Scope)
}

// DecodeValue implements the Codec interface.
func (cwsc *CodeWithScopeCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != TypeCodeWithScope {
		return fmt.Errorf("cannot decode %v into a CodeWithScope", vr.Type())
	}

	target, ok := i.(*CodeWithScope)
	if !ok || target == nil {
		return fmt.Errorf("%T can only be used to decode non-nil *CodeWithScope values, got %T", cwsc, i)
	}

	code, dr, err := vr.ReadCodeWithScope()
	if err != nil {
		return err
	}

	var scope *Document
	err = defaultDocumentCodec.decodeDocument(dc, dr, &scope)
	if err != nil {
		return err
	}

	*target = CodeWithScope{Code: code, Scope: scope}
	return nil
}

// TimestampCodec is the Codec for Timestamp values.
type TimestampCodec struct{}

var _ Codec = &TimestampCodec{}

// EncodeValue implements the Codec interface.
func (tc *TimestampCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var ts Timestamp
	switch t := i.(type) {
	case Timestamp:
		ts = t
	case *Timestamp:
		ts = *t
	default:
		return CodecEncodeError{Codec: tc, Types: []interface{}{Timestamp{}, (*Timestamp)(nil)}, Received: i}
	}

	return vw.WriteTimestamp(ts.T, ts.I)
}

// DecodeValue implements the Codec interface.
func (tc *TimestampCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != TypeTimestamp {
		return fmt.Errorf("cannot decode %v into a Timestamp", vr.Type())
	}

	target, ok := i.(*Timestamp)
	if !ok || target == nil {
		return fmt.Errorf("%T can only be used to decode non-nil *Timestamp values, got %T", tc, i)
	}

	t, incr, err := vr.ReadTimestamp()
	if err != nil {
		return err
	}

	*target = Timestamp{T: t, I: incr}
	return nil
}

// Decimal128Codec is the Codec for decimal.Decimal128 values.
type Decimal128Codec struct{}

var _ Codec = &Decimal128Codec{}

// EncodeValue implements the Codec interface.
func (dc *Decimal128Codec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var d128 decimal.Decimal128
	switch t := i.(type) {
	case decimal.Decimal128:
		d128 = t
	case *decimal.Decimal128:
		d128 = *t
	default:
		return CodecEncodeError{Codec: dc, Types: []interface{}{decimal.Decimal128{}, (*decimal.Decimal128)(nil)}, Received: i}
	}

	return vw.WriteDecimal128(d128)
}

// DecodeValue implements the Codec interface.
func (dc *Decimal128Codec) DecodeValue(dctx DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != TypeDecimal128 {
		return fmt.Errorf("cannot decode %v into a decimal.Decimal128", vr.Type())
	}

	target, ok := i.(*decimal.Decimal128)
	if !ok || target == nil {
		return fmt.Errorf("%T can only be used to decode non-nil *decimal.Decimal128 values, got %T", dc, i)
	}

	d128, err := vr.ReadDecimal128()
	if err != nil {
		return err
	}

	*target = d128
	return nil
}

// MinKeyCodec is the Codec for MinKey values.
type MinKeyCodec struct{}

var _ Codec = &MinKeyCodec{}

// EncodeValue implements the Codec interface.
func (mkc *MinKeyCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	switch i.(type) {
	case MinKeyv2, *MinKeyv2:
	default:
		return CodecEncodeError{Codec: mkc, Types: []interface{}{MinKeyv2{}, (*MinKeyv2)(nil)}, Received: i}
	}

	return vw.WriteMinKey()
}

// DecodeValue implements the Codec interface.
func (mkc *MinKeyCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != TypeMinKey {
		return fmt.Errorf("cannot decode %v into a MinKey", vr.Type())
	}

	target, ok := i.(*MinKeyv2)
	if !ok || target == nil {
		return fmt.Errorf("%T can only be used to decode non-nil *MinKey values, got %T", mkc, i)
	}

	*target = MinKeyv2{}
	return vr.ReadMinKey()
}

// MaxKeyCodec is the Codec for MaxKey values.
type MaxKeyCodec struct{}

var _ Codec = &MaxKeyCodec{}

// EncodeValue implements the Codec interface.
func (mkc *MaxKeyCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	switch i.(type) {
	case MaxKeyv2, *MaxKeyv2:
	default:
		return CodecEncodeError{Codec: mkc, Types: []interface{}{MaxKeyv2{}, (*MaxKeyv2)(nil)}, Received: i}
	}

	return vw.WriteMaxKey()
}

// DecodeValue implements the Codec interface.
func (mkc *MaxKeyCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != TypeMaxKey {
		return fmt.Errorf("cannot decode %v into a MaxKey", vr.Type())
	}

	target, ok := i.(*MaxKeyv2)
	if !ok || target == nil {
		return fmt.Errorf("%T can only be used to decode non-nil *MaxKey values, got %T", mkc, i)
	}

	*target = MaxKeyv2{}
	return vr.ReadMaxKey()
}

// elementCodec is the Codec for *Element values.
//
// This is a codec used internally.
type elementCodec struct{}

func (ec *elementCodec) EncodeValue(ectx EncodeContext, vw ValueWriter, i interface{}) error {
	elem, ok := i.(*Element)
	if !ok {
		return CodecEncodeError{Codec: ec, Types: []interface{}{(*Element)(nil)}, Received: i}
	}

	if _, err := elem.Validate(); err != nil {
		return err
	}

	return ec.encodeValue(ectx, vw, elem)
}

func (*elementCodec) DecodeValue(DecodeContext, ValueReader, interface{}) error {
	return errors.New("elementCodec's DecodeValue method should not be used directly")
}

func (ec *elementCodec) encodeValue(ectx EncodeContext, vw ValueWriter, elem *Element) error {
	return defaultValueCodec.encodeValue(ectx, vw, elem.value)
}

func (ec *elementCodec) decodeValue(dc DecodeContext, vr ValueReader, key string, elem **Element) error {
	switch vr.Type() {
	case TypeDouble:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		*elem = EC.Double(key, f64)
	case TypeString:
		str, err := vr.ReadString()
		if err != nil {
			return err
		}
		*elem = EC.String(key, str)
	case TypeEmbeddedDocument:
		codec, err := dc.Lookup(tDocument)
		if err != nil {
			return err
		}
		var embeddedDoc *Document
		err = codec.DecodeValue(dc, vr, &embeddedDoc)
		if err != nil {
			return err
		}
		*elem = EC.SubDocument(key, embeddedDoc)
	case TypeArray:
		codec, err := dc.Lookup(tArray)
		if err != nil {
			return err
		}
		var arr *Array
		err = codec.DecodeValue(dc, vr, &arr)
		if err != nil {
			return err
		}
		*elem = EC.Array(key, arr)
	case TypeBinary:
		data, subtype, err := vr.ReadBinary()
		if err != nil {
			return err
		}
		*elem = EC.BinaryWithSubtype(key, data, subtype)
	case TypeUndefined:
		err := vr.ReadUndefined()
		if err != nil {
			return err
		}
		*elem = EC.Undefined(key)
	case TypeObjectID:
		oid, err := vr.ReadObjectID()
		if err != nil {
			return err
		}
		*elem = EC.ObjectID(key, oid)
	case TypeBoolean:
		b, err := vr.ReadBoolean()
		if err != nil {
			return err
		}
		*elem = EC.Boolean(key, b)
	case TypeDateTime:
		dt, err := vr.ReadDateTime()
		if err != nil {
			return err
		}
		*elem = EC.DateTime(key, dt)
	case TypeNull:
		err := vr.ReadNull()
		if err != nil {
			return err
		}
		*elem = EC.Null(key)
	case TypeRegex:
		pattern, options, err := vr.ReadRegex()
		if err != nil {
			return err
		}
		*elem = EC.Regex(key, pattern, options)
	case TypeDBPointer:
		ns, pointer, err := vr.ReadDBPointer()
		if err != nil {
			return err
		}
		*elem = EC.DBPointer(key, ns, pointer)
	case TypeJavaScript:
		js, err := vr.ReadJavascript()
		if err != nil {
			return err
		}
		*elem = EC.JavaScript(key, js)
	case TypeSymbol:
		symbol, err := vr.ReadSymbol()
		if err != nil {
			return err
		}
		*elem = EC.Symbol(key, symbol)
	case TypeCodeWithScope:
		code, scope, err := vr.ReadCodeWithScope()
		if err != nil {
			return err
		}
		scopeDoc := new(*Document)
		err = defaultDocumentCodec.decodeDocument(dc, scope, scopeDoc)
		if err != nil {
			return err
		}
		*elem = EC.CodeWithScope(key, code, *scopeDoc)
	case TypeInt32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		*elem = EC.Int32(key, i32)
	case TypeTimestamp:
		t, i, err := vr.ReadTimestamp()
		if err != nil {
			return err
		}
		*elem = EC.Timestamp(key, t, i)
	case TypeInt64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return err
		}
		*elem = EC.Int64(key, i64)
	case TypeDecimal128:
		d128, err := vr.ReadDecimal128()
		if err != nil {
			return err
		}
		*elem = EC.Decimal128(key, d128)
	case TypeMinKey:
		err := vr.ReadMinKey()
		if err != nil {
			return err
		}
		*elem = EC.MinKey(key)
	case TypeMaxKey:
		err := vr.ReadMaxKey()
		if err != nil {
			return err
		}
		*elem = EC.MaxKey(key)
	default:
		return fmt.Errorf("Cannot read unknown BSON type %s", vr.Type())
	}

	return nil
}

// ValueCodec is the Codec for *Value values.
type ValueCodec struct{}

var _ Codec = &ValueCodec{}

// EncodeValue implements the Codec interface.
func (vc *ValueCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	val, ok := i.(*Value)
	if !ok {
		return CodecEncodeError{Codec: vc, Types: []interface{}{(*Value)(nil)}, Received: i}
	}

	if _, err := val.validate(false); err != nil {
		return err
	}

	return vc.encodeValue(ec, vw, val)
}

// encodeValue does not validation, and the callers must perform validation on val before calling
// this method.
func (vc *ValueCodec) encodeValue(ec EncodeContext, vw ValueWriter, val *Value) error {
	var err error
	switch val.Type() {
	case TypeDouble:
		err = vw.WriteDouble(val.Double())
	case TypeString:
		err = vw.WriteString(val.StringValue())
	case TypeEmbeddedDocument:
		var codec Codec
		codec, err = ec.Lookup(tDocument)
		if err != nil {
			break
		}
		err = codec.EncodeValue(ec, vw, val.MutableDocument())
	case TypeArray:
		var codec Codec
		codec, err = ec.Lookup(tArray)
		if err != nil {
			break
		}
		err = codec.EncodeValue(ec, vw, val.MutableArray())
	case TypeBinary:
		// TODO: FIX THIS (╯°□°）╯︵ ┻━┻
		subtype, data := val.Binary()
		err = vw.WriteBinaryWithSubtype(data, subtype)
	case TypeUndefined:
		err = vw.WriteUndefined()
	case TypeObjectID:
		err = vw.WriteObjectID(val.ObjectID())
	case TypeBoolean:
		err = vw.WriteBoolean(val.Boolean())
	case TypeDateTime:
		err = vw.WriteDateTime(val.DateTime())
	case TypeNull:
		err = vw.WriteNull()
	case TypeRegex:
		err = vw.WriteRegex(val.Regex())
	case TypeDBPointer:
		err = vw.WriteDBPointer(val.DBPointer())
	case TypeJavaScript:
		err = vw.WriteJavascript(val.JavaScript())
	case TypeSymbol:
		err = vw.WriteSymbol(val.Symbol())
	case TypeCodeWithScope:
		code, scope := val.MutableJavaScriptWithScope()

		var cwsw DocumentWriter
		cwsw, err = vw.WriteCodeWithScope(code)
		if err != nil {
			break
		}

		err = defaultDocumentCodec.encodeDocument(ec, cwsw, scope)
	case TypeInt32:
		err = vw.WriteInt32(val.Int32())
	case TypeTimestamp:
		err = vw.WriteTimestamp(val.Timestamp())
	case TypeInt64:
		err = vw.WriteInt64(val.Int64())
	case TypeDecimal128:
		err = vw.WriteDecimal128(val.Decimal128())
	case TypeMinKey:
		err = vw.WriteMinKey()
	case TypeMaxKey:
		err = vw.WriteMaxKey()
	default:
		err = fmt.Errorf("%T is not a valid BSON type to encode", val.Type())
	}

	return err
}

// DecodeValue implements the Codec interface.
func (vc *ValueCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	pval, ok := i.(**Value)
	if !ok {
		return CodecDecodeError{Codec: vc, Types: []interface{}{(**Value)(nil)}, Received: i}
	}

	if pval == nil {
		return fmt.Errorf("%T can only be used to decode non-nil **Value", vc)
	}

	return vc.decodeValue(dc, vr, pval)
}

func (vc *ValueCodec) decodeValue(dc DecodeContext, vr ValueReader, val **Value) error {
	switch vr.Type() {
	case TypeDouble:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		*val = VC.Double(f64)
	case TypeString:
		str, err := vr.ReadString()
		if err != nil {
			return err
		}
		*val = VC.String(str)
	case TypeEmbeddedDocument:
		codec, err := dc.Lookup(tDocument)
		if err != nil {
			return err
		}
		var embeddedDoc *Document
		err = codec.DecodeValue(dc, vr, &embeddedDoc)
		if err != nil {
			return err
		}
		*val = VC.Document(embeddedDoc)
	case TypeArray:
		codec, err := dc.Lookup(tArray)
		if err != nil {
			return err
		}
		var arr *Array
		err = codec.DecodeValue(dc, vr, &arr)
		if err != nil {
			return err
		}
		*val = VC.Array(arr)
	case TypeBinary:
		data, subtype, err := vr.ReadBinary()
		if err != nil {
			return err
		}
		*val = VC.BinaryWithSubtype(data, subtype)
	case TypeUndefined:
		err := vr.ReadUndefined()
		if err != nil {
			return err
		}
		*val = VC.Undefined()
	case TypeObjectID:
		oid, err := vr.ReadObjectID()
		if err != nil {
			return err
		}
		*val = VC.ObjectID(oid)
	case TypeBoolean:
		b, err := vr.ReadBoolean()
		if err != nil {
			return err
		}
		*val = VC.Boolean(b)
	case TypeDateTime:
		dt, err := vr.ReadDateTime()
		if err != nil {
			return err
		}
		*val = VC.DateTime(dt)
	case TypeNull:
		err := vr.ReadNull()
		if err != nil {
			return err
		}
		*val = VC.Null()
	case TypeRegex:
		pattern, options, err := vr.ReadRegex()
		if err != nil {
			return err
		}
		*val = VC.Regex(pattern, options)
	case TypeDBPointer:
		ns, pointer, err := vr.ReadDBPointer()
		if err != nil {
			return err
		}
		*val = VC.DBPointer(ns, pointer)
	case TypeJavaScript:
		js, err := vr.ReadJavascript()
		if err != nil {
			return err
		}
		*val = VC.JavaScript(js)
	case TypeSymbol:
		symbol, err := vr.ReadSymbol()
		if err != nil {
			return err
		}
		*val = VC.Symbol(symbol)
	case TypeCodeWithScope:
		code, scope, err := vr.ReadCodeWithScope()
		if err != nil {
			return err
		}
		scopeDoc := new(*Document)
		err = defaultDocumentCodec.decodeDocument(dc, scope, scopeDoc)
		if err != nil {
			return err
		}
		*val = VC.CodeWithScope(code, *scopeDoc)
	case TypeInt32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		*val = VC.Int32(i32)
	case TypeTimestamp:
		t, i, err := vr.ReadTimestamp()
		if err != nil {
			return err
		}
		*val = VC.Timestamp(t, i)
	case TypeInt64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return err
		}
		*val = VC.Int64(i64)
	case TypeDecimal128:
		d128, err := vr.ReadDecimal128()
		if err != nil {
			return err
		}
		*val = VC.Decimal128(d128)
	case TypeMinKey:
		err := vr.ReadMinKey()
		if err != nil {
			return err
		}
		*val = VC.MinKey()
	case TypeMaxKey:
		err := vr.ReadMaxKey()
		if err != nil {
			return err
		}
		*val = VC.MaxKey()
	default:
		return fmt.Errorf("Cannot read unknown BSON type %s", vr.Type())
	}

	return nil
}

// JSONNumberCodec is the Codec for json.Number values.
type JSONNumberCodec struct{}

var _ Codec = &JSONNumberCodec{}

// EncodeValue implements the Codec interface.
func (jnc *JSONNumberCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var jsnum json.Number
	switch t := i.(type) {
	case json.Number:
		jsnum = t
	case *json.Number:
		jsnum = *t
	default:
		return CodecEncodeError{Codec: jnc, Types: []interface{}{json.Number(""), (*json.Number)(nil)}, Received: i}
	}

	// Attempt int first, then float64
	if i64, err := jsnum.Int64(); err == nil {
		return defaultIntCodec.EncodeValue(ec, vw, i64)
	}

	f64, err := jsnum.Float64()
	if err != nil {
		return err
	}

	return defaultFloatCodec.EncodeValue(ec, vw, f64)
}

// DecodeValue implements the Codec interface.
func (jnc *JSONNumberCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	target, ok := i.(*json.Number)
	if !ok || target == nil {
		return fmt.Errorf("%T can only be used to decode non-nil *json.Number values, got %T", jnc, i)
	}

	switch vr.Type() {
	case TypeDouble:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		*target = json.Number(strconv.FormatFloat(f64, 'g', -1, 64))
	case TypeInt32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		*target = json.Number(strconv.FormatInt(int64(i32), 10))
	case TypeInt64:
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

// URLCodec is the Codec for url.URL values.
type URLCodec struct{}

var _ Codec = &URLCodec{}

// EncodeValue implements the Codec interface.
func (uc *URLCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var u *url.URL
	switch t := i.(type) {
	case url.URL:
		u = &t
	case *url.URL:
		u = t
	default:
		return CodecEncodeError{Codec: uc, Types: []interface{}{url.URL{}, (*url.URL)(nil)}, Received: i}
	}

	return vw.WriteString(u.String())
}

// DecodeValue implements the Codec interface.
func (uc *URLCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != TypeString {
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

	// It's valid to use either a *url.URL or a url.URL
	switch target := i.(type) {
	case *url.URL:
		if target == nil {
			return fmt.Errorf("%T can only be used to decode non-nil *url.URL values, got %T", uc, i)
		}
		*target = *u
	case **url.URL:
		if target == nil {
			return fmt.Errorf("%T can only be used to decode non-nil *url.URL values, got %T", uc, i)
		}
		*target = u
	default:
		return fmt.Errorf("%T can only be used to decode non-nil *url.URL values, got %T", uc, i)
	}
	return nil
}

// TimeCodec is the Codec for time.Time values.
type TimeCodec struct{}

var _ Codec = &TimeCodec{}

// EncodeValue implements the Codec interface.
func (tc *TimeCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var tt time.Time
	switch t := i.(type) {
	case time.Time:
		tt = t
	case *time.Time:
		tt = *t
	default:
		return CodecEncodeError{Codec: tc, Types: []interface{}{time.Time{}, (*time.Time)(nil)}, Received: i}
	}

	return vw.WriteDateTime(tt.UnixNano() / int64(time.Millisecond))
}

// DecodeValue implements the Codec interface.
func (tc *TimeCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != TypeDateTime {
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

	return fmt.Errorf("%T can only be used to decode non-nil *time.Time values, got %T", tc, i)
}

// ReaderCodec is the Codec for Reader values.
type ReaderCodec struct{}

var _ Codec = &ReaderCodec{}

// EncodeValue implements the Codec interface.
func (rc *ReaderCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	rdr, ok := i.(Reader)
	if !ok {
		return CodecEncodeError{Codec: rc, Types: []interface{}{Reader{}}, Received: i}
	}

	// TODO: Handle fast path, we should just copy the bytes into the
	// *valueWriter and then do the cleanup of the state.
	dw, err := vw.WriteDocument()
	if err != nil {
		return err
	}

	itr, err := rdr.Iterator()
	if err != nil {
		return err
	}

	for itr.Next() {
		elem := itr.Element()
		dvw, err := dw.WriteDocumentElement(elem.Key())
		if err != nil {
			return err
		}

		val := elem.Value()
		err = defaultValueCodec.encodeValue(ec, dvw, val)

		if err != nil {
			return err
		}
	}

	if err := itr.Err(); err != nil {
		return err
	}

	return dw.WriteDocumentEnd()
}

// DecodeValue implements the Codec interface.
func (rc *ReaderCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	rdr, ok := i.(*Reader)
	if !ok {
		return CodecDecodeError{Codec: rc, Types: []interface{}{(*Reader)(nil)}, Received: i}
	}

	if rdr == nil {
		return fmt.Errorf("%T can only be used to decode non-nil *Reader", rc)
	}

	if *rdr == nil {
		*rdr = make(Reader, 256)
	}

	// TODO: handle the fast path, if we have a *valueReader, just copy the
	// bytes.
	vw := vwPool.Get().(*valueWriter)
	defer vwPool.Put(vw)

	vw.reset((*rdr)[:0])

	err := CopyDocument(vw, vr)
	if err != nil {
		return err
	}

	*rdr = vw.buf
	return nil
}

// ByteSliceCodec is the Codec for []byte values.
type ByteSliceCodec struct{}

var _ Codec = &ByteSliceCodec{}

// EncodeValue implements the Codec interface.
func (bsc *ByteSliceCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var slcb []byte
	switch t := i.(type) {
	case []byte:
		slcb = t
	case *[]byte:
		slcb = *t
	default:
		return CodecEncodeError{Codec: bsc, Types: []interface{}{[]byte{}, (*[]byte)(nil)}, Received: i}
	}

	return vw.WriteBinary(slcb)
}

// DecodeValue implements the Codec interface.
func (bsc *ByteSliceCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != TypeBinary {
		return fmt.Errorf("cannot decode %v into a *[]byte", vr.Type())
	}

	target, ok := i.(*[]byte)
	if !ok || target == nil {
		return fmt.Errorf("%T can only be used to decode non-nil *[]byte values, got %T", bsc, i)
	}

	data, subtype, err := vr.ReadBinary()
	if err != nil {
		return err
	}
	if subtype != 0x00 {
		return fmt.Errorf("%T can only be used to decode subtype 0x00 for %s, got %v", bsc, TypeBinary, subtype)
	}

	*target = data
	return nil
}

// ElementSliceCodec is the Codec for []*Element values.
type ElementSliceCodec struct{}

var _ Codec = &ElementSliceCodec{}

// EncodeValue implements the Codec interface.
func (esc *ElementSliceCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var slce []*Element
	switch t := i.(type) {
	case []*Element:
		slce = t
	case *[]*Element:
		slce = *t
	default:
		return CodecEncodeError{Codec: esc, Types: []interface{}{[]*Element{}, (*[]*Element)(nil)}, Received: i}
	}

	return defaultDocumentCodec.EncodeValue(ec, vw, &Document{elems: slce})
}

// DecodeValue implements the Codec interface.
func (esc *ElementSliceCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	var doc *Document
	err := defaultDocumentCodec.DecodeValue(dc, vr, &doc)
	if err != nil {
		return err
	}

	target, ok := i.(*[]*Element)
	if !ok || target == nil {
		return fmt.Errorf("%T can only be used to decode non-nil *[]*Element values, got %T", esc, i)
	}

	*target = doc.elems
	return nil
}
