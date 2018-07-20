package bson

import (
	"fmt"
	"math"
	"reflect"
	"strings"
)

var defaultBoolCodec = &BooleanCodec{}
var defaultIntCodec = &IntCodec{}
var defaultUintCodec = &UintCodec{}
var defaultFloatCodec = &FloatCodec{}
var defaultStringCodec = &StringCodec{}

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

type EncodeContext struct {
	*Registry
	MinSize bool
}

type DecodeContext struct {
	*Registry
	Truncate bool
}

// Codec implementations handle encoding and decoding values. They can be
// registered in a registry which will handle invoking them.
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
		if t > math.MaxInt64 {
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
	if vr.Type() != TypeDouble {
		return fmt.Errorf("cannot decode %v into a float32 or float64 type", vr.Type())
	}

	f, err := vr.ReadDouble()
	if err != nil {
		return err
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
	s, ok := i.(string)
	if !ok {
		if reflect.TypeOf(i).Kind() != reflect.String {
			return CodecEncodeError{Codec: sc, Types: []interface{}{string("")}, Received: i}
		}

		s = reflect.ValueOf(i).String()
	}

	return vw.WriteString(s)
}

// DecodeValue implements the Codec interface.
func (sc *StringCodec) DecodeValue(dctx DecodeContext, vr ValueReader, i interface{}) error {
	if vr.Type() != TypeString {
		return fmt.Errorf("cannot decode %v into a string", vr.Type())
	}

	var err error
	if target, ok := i.(*string); ok && target != nil { // if it is nil, we go the slow path.
		*target, err = vr.ReadString()
		return err
	}

	val := reflect.ValueOf(i)
	if !val.IsValid() || val.Kind() != reflect.Ptr || !val.Elem().CanSet() {
		return fmt.Errorf("%T can only be used to decode settable (non-nil) values", sc)
	}
	val = val.Elem()

	if val.Type().Kind() != reflect.String {
		return CodecDecodeError{Codec: sc, Types: []interface{}{string("")}, Received: i}
	}

	s, err := vr.ReadString()
	val.SetString(s)
	return err
}

// DocumentCodec is the Codec used for *Document values.
type DocumentCodec struct{}

var _ Codec = &DocumentCodec{}

// EncodeValue implements the Codec interface.
func (dc *DocumentCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (dc *DocumentCodec) DecodeValue(dctx DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// ArrayCodec is the Codec used for *Array values.
type ArrayCodec struct{}

var _ Codec = &ArrayCodec{}

// EncodeValue implements the Codec interface.
func (ac *ArrayCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (ac *ArrayCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// BinaryCodec is the Codec used for Binary values.
type BinaryCodec struct{}

var _ Codec = &BinaryCodec{}

// EncodeValue implements the Codec interface.
func (b *BinaryCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (b *BinaryCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// UndefinedCodec is the Codec for Undefined values.
type UndefinedCodec struct{}

var _ Codec = &UndefinedCodec{}

// EncodeValue implements the Codec interface.
func (u *UndefinedCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (u *UndefinedCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// ObjectIDCodec is the Codec for objectid.ObjectID values.
type ObjectIDCodec struct{}

var _ Codec = &ObjectIDCodec{}

// EncodeValue implements the Codec interface.
func (o *ObjectIDCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (o *ObjectIDCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// DateTimeCodec is the Codec for DateTime values.
type DateTimeCodec struct{}

var _ Codec = &DateTimeCodec{}

// EncodeValue implements the Codec interface.
func (d *DateTimeCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (d *DateTimeCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// NullCodec is the Codec for Null values.
type NullCodec struct{}

var _ Codec = &NullCodec{}

// EncodeValue implements the Codec interface.
func (n *NullCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (n *NullCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// RegexCodec is the Codec for Regex values.
type RegexCodec struct{}

var _ Codec = &RegexCodec{}

// EncodeValue implements the Codec interface.
func (r *RegexCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (r *RegexCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// DBPointerCodec is the Codec for DBPointer values.
type DBPointerCodec struct{}

var _ Codec = &DBPointerCodec{}

// EncodeValue implements the Codec interface.
func (d *DBPointerCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (d *DBPointerCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// JavaScriptCodec is the Codec for JavaScript values.
type JavaScriptCodec struct{}

var _ Codec = &JavaScriptCodec{}

// EncodeValue implements the Codec interface.
func (j *JavaScriptCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (j *JavaScriptCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// SymbolCodec is the Codec for Symbol values.
type SymbolCodec struct{}

var _ Codec = &SymbolCodec{}

// EncodeValue implements the Codec interface.
func (s *SymbolCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (s *SymbolCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// CodeWithScopeCodec is the Codec for CodeWithScope values.
type CodeWithScopeCodec struct{}

var _ Codec = &CodeWithScopeCodec{}

// EncodeValue implements the Codec interface.
func (c *CodeWithScopeCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (c *CodeWithScopeCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// TimestampCodec is the Codec for Timestamp values.
type TimestampCodec struct{}

var _ Codec = &TimestampCodec{}

// EncodeValue implements the Codec interface.
func (t *TimestampCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (t *TimestampCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// Decimal128Codec is the Codec for decimal.Decimal128 values.
type Decimal128Codec struct{}

var _ Codec = &Decimal128Codec{}

// EncodeValue implements the Codec interface.
func (d *Decimal128Codec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (d *Decimal128Codec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// MinKeyCodec is the Codec for MinKey values.
type MinKeyCodec struct{}

var _ Codec = &MinKeyCodec{}

// EncodeValue implements the Codec interface.
func (m *MinKeyCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (m *MinKeyCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// MaxKeyCodec is the Codec for MaxKey values.
type MaxKeyCodec struct{}

var _ Codec = &MaxKeyCodec{}

// EncodeValue implements the Codec interface.
func (m *MaxKeyCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (m *MaxKeyCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// ElementCodec is the Codec for *Element values.
type ElementCodec struct{}

var _ Codec = &ElementCodec{}

// EncodeValue implements the Codec interface.
func (e *ElementCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (e *ElementCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// ValueCodec is the Codec for *Value values.
type ValueCodec struct{}

var _ Codec = &ValueCodec{}

// EncodeValue implements the Codec interface.
func (v *ValueCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (v *ValueCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// ReaderCodec is the Codec for Reader values.
type ReaderCodec struct{}

var _ Codec = &ReaderCodec{}

// EncodeValue implements the Codec interface.
func (r *ReaderCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (r *ReaderCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// JSONNumberCodec is the Codec for json.Number values.
type JSONNumberCodec struct{}

var _ Codec = &JSONNumberCodec{}

// EncodeValue implements the Codec interface.
func (j *JSONNumberCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (j *JSONNumberCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// URLCodec is the Codec for url.URL values.
type URLCodec struct{}

var _ Codec = &URLCodec{}

// EncodeValue implements the Codec interface.
func (u *URLCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (u *URLCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// TimeCodec is the Codec for time.Time values.
type TimeCodec struct{}

var _ Codec = &TimeCodec{}

// EncodeValue implements the Codec interface.
func (t *TimeCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (t *TimeCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}

// ByteSliceCodec is the Codec for []byte values.
type ByteSliceCodec struct{}

var _ Codec = &ByteSliceCodec{}

// EncodeValue implements the Codec interface.
func (b *ByteSliceCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	panic("not implemented")
}

// DecodeValue implements the Codec interface.
func (b *ByteSliceCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	panic("not implemented")
}
