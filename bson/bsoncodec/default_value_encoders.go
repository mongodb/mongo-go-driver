package bsoncodec

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"reflect"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

var defaultValueEncoders DefaultValueEncoders

// DefaultValueEncoders is a namespace type for the default ValueEncoders used
// when creating a registry.
type DefaultValueEncoders struct{}

// BooleanEncodeValue is the ValueEncoderFunc for bool types.
func (dve DefaultValueEncoders) BooleanEncodeValue(ectx EncodeContext, vw ValueWriter, i interface{}) error {
	b, ok := i.(bool)
	if !ok {
		if reflect.TypeOf(i).Kind() != reflect.Bool {
			return ValueEncoderError{Name: "BooleanEncodeValue", Types: []interface{}{bool(true)}, Received: i}
		}

		b = reflect.ValueOf(i).Bool()
	}

	return vw.WriteBoolean(b)
}

// IntEncodeValue is the ValueEncoderFunc for int types.
func (dve DefaultValueEncoders) IntEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
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

	return ValueEncoderError{
		Name:     "IntEncodeValue",
		Types:    []interface{}{int8(0), int16(0), int32(0), int64(0), int(0)},
		Received: i,
	}
}

// UintEncodeValue is the ValueEncoderFunc for uint types.
func (dve DefaultValueEncoders) UintEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
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

	return ValueEncoderError{
		Name:     "UintEncodeValue",
		Types:    []interface{}{uint8(0), uint16(0), uint32(0), uint64(0), uint(0)},
		Received: i,
	}
}

// FloatEncodeValue is the ValueEncoderFunc for float types.
func (dve DefaultValueEncoders) FloatEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
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

	return ValueEncoderError{Name: "FloatEncodeValue", Types: []interface{}{float32(0), float64(0)}, Received: i}
}

// StringEncodeValue is the ValueEncoderFunc for string types.
func (dve DefaultValueEncoders) StringEncodeValue(ectx EncodeContext, vw ValueWriter, i interface{}) error {
	switch t := i.(type) {
	case string:
		return vw.WriteString(t)
	case bson.JavaScriptCode:
		return vw.WriteJavascript(string(t))
	case bson.Symbol:
		return vw.WriteSymbol(string(t))
	}

	val := reflect.ValueOf(i)
	if val.Type().Kind() != reflect.String {
		return ValueEncoderError{
			Name:     "StringEncodeValue",
			Types:    []interface{}{string(""), bson.JavaScriptCode(""), bson.Symbol("")},
			Received: i,
		}
	}

	return vw.WriteString(val.String())
}

// DocumentEncodeValue is the ValueEncoderFunc for *bson.Document.
func (dve DefaultValueEncoders) DocumentEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	doc, ok := i.(*bson.Document)
	if !ok {
		return ValueEncoderError{Name: "DocumentEncodeValue", Types: []interface{}{(*bson.Document)(nil)}, Received: i}
	}

	dw, err := vw.WriteDocument()
	if err != nil {
		return err
	}

	return dve.encodeDocument(ec, dw, doc)
}

// ArrayEncodeValue is the ValueEncoderFunc for *bson.Array.
func (dve DefaultValueEncoders) ArrayEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	arr, ok := i.(*bson.Array)
	if !ok {
		return ValueEncoderError{Name: "ArrayEncodeValue", Types: []interface{}{(*bson.Array)(nil)}, Received: i}
	}

	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	itr, err := arr.Iterator()
	if err != nil {
		return err
	}

	for itr.Next() {
		val := itr.Value()
		dvw, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		err = dve.encodeValue(ec, dvw, val)

		if err != nil {
			return err
		}
	}

	if err := itr.Err(); err != nil {
		return err
	}

	return aw.WriteArrayEnd()
}

// encodeDocument is a separate function that we use because CodeWithScope
// returns us a DocumentWriter and we need to do the same logic that we would do
// for a document but cannot use a Codec.
func (dve DefaultValueEncoders) encodeDocument(ec EncodeContext, dw DocumentWriter, doc *bson.Document) error {
	itr := doc.Iterator()

	for itr.Next() {
		elem := itr.Element()
		dvw, err := dw.WriteDocumentElement(elem.Key())
		if err != nil {
			return err
		}

		val := elem.Value()
		err = dve.encodeValue(ec, dvw, val)

		if err != nil {
			return err
		}
	}

	if err := itr.Err(); err != nil {
		return err
	}

	return dw.WriteDocumentEnd()
}

// BinaryEncodeValue is the ValueEncoderFunc for bson.Binary.
func (dve DefaultValueEncoders) BinaryEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var b bson.Binary
	switch t := i.(type) {
	case bson.Binary:
		b = t
	case *bson.Binary:
		b = *t
	default:
		return ValueEncoderError{
			Name:     "BinaryEncodeValue",
			Types:    []interface{}{bson.Binary{}, (*bson.Binary)(nil)},
			Received: i,
		}
	}

	return vw.WriteBinaryWithSubtype(b.Data, b.Subtype)
}

// UndefinedEncodeValue is the ValueEncoderFunc for bson.Undefined.
func (dve DefaultValueEncoders) UndefinedEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	switch i.(type) {
	case bson.Undefinedv2, *bson.Undefinedv2:
	default:
		return ValueEncoderError{
			Name:     "UndefinedEncodeValue",
			Types:    []interface{}{bson.Undefinedv2{}, (*bson.Undefinedv2)(nil)},
			Received: i,
		}
	}

	return vw.WriteUndefined()
}

// ObjectIDEncodeValue is the ValueEncoderFunc for objectid.ObjectID.
func (dve DefaultValueEncoders) ObjectIDEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var oid objectid.ObjectID
	switch t := i.(type) {
	case objectid.ObjectID:
		oid = t
	case *objectid.ObjectID:
		oid = *t
	default:
		return ValueEncoderError{
			Name:     "ObjectIDEncodeValue",
			Types:    []interface{}{objectid.ObjectID{}, (*objectid.ObjectID)(nil)},
			Received: i,
		}
	}

	return vw.WriteObjectID(oid)
}

// DateTimeEncodeValue is the ValueEncoderFunc for bson.DateTime.
func (dve DefaultValueEncoders) DateTimeEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var dt bson.DateTime
	switch t := i.(type) {
	case bson.DateTime:
		dt = t
	case *bson.DateTime:
		dt = *t
	default:
		return ValueEncoderError{
			Name:     "DateTimeEncodeValue",
			Types:    []interface{}{bson.DateTime(0), (*bson.DateTime)(nil)},
			Received: i,
		}
	}

	return vw.WriteDateTime(int64(dt))
}

// NullEncodeValue is the ValueEncoderFunc for bson.Null.
func (dve DefaultValueEncoders) NullEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	switch i.(type) {
	case bson.Nullv2, *bson.Nullv2:
	default:
		return ValueEncoderError{
			Name:     "NullEncodeValue",
			Types:    []interface{}{bson.Nullv2{}, (*bson.Nullv2)(nil)},
			Received: i,
		}
	}

	return vw.WriteNull()
}

// RegexEncodeValue is the ValueEncoderFunc for bson.Regex.
func (dve DefaultValueEncoders) RegexEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var regex bson.Regex
	switch t := i.(type) {
	case bson.Regex:
		regex = t
	case *bson.Regex:
		regex = *t
	default:
		return ValueEncoderError{
			Name:     "RegexEncodeValue",
			Types:    []interface{}{bson.Regex{}, (*bson.Regex)(nil)},
			Received: i,
		}
	}

	return vw.WriteRegex(regex.Pattern, regex.Options)
}

// DBPointerEncodeValue is the ValueEncoderFunc for bson.DBPointer.
func (dve DefaultValueEncoders) DBPointerEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var dbp bson.DBPointer
	switch t := i.(type) {
	case bson.DBPointer:
		dbp = t
	case *bson.DBPointer:
		dbp = *t
	default:
		return ValueEncoderError{
			Name:     "DBPointerEncodeValue",
			Types:    []interface{}{bson.DBPointer{}, (*bson.DBPointer)(nil)},
			Received: i,
		}
	}

	return vw.WriteDBPointer(dbp.DB, dbp.Pointer)
}

// CodeWithScopeEncodeValue is the ValueEncoderFunc for bson.CodeWithScope.
func (dve DefaultValueEncoders) CodeWithScopeEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var cws bson.CodeWithScope
	switch t := i.(type) {
	case bson.CodeWithScope:
		cws = t
	case *bson.CodeWithScope:
		cws = *t
	default:
		return ValueEncoderError{
			Name:     "CodeWithScopeEncodeValue",
			Types:    []interface{}{bson.CodeWithScope{}, (*bson.CodeWithScope)(nil)},
			Received: i,
		}
	}

	dw, err := vw.WriteCodeWithScope(cws.Code)
	if err != nil {
		return err
	}

	return dve.encodeDocument(ec, dw, cws.Scope)
}

// TimestampEncodeValue is the ValueEncoderFunc for bson.Timestamp.
func (dve DefaultValueEncoders) TimestampEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var ts bson.Timestamp
	switch t := i.(type) {
	case bson.Timestamp:
		ts = t
	case *bson.Timestamp:
		ts = *t
	default:
		return ValueEncoderError{
			Name:     "TimestampEncodeValue",
			Types:    []interface{}{bson.Timestamp{}, (*bson.Timestamp)(nil)},
			Received: i,
		}
	}

	return vw.WriteTimestamp(ts.T, ts.I)
}

// Decimal128EncodeValue is the ValueEncoderFunc for decimal.Decimal128.
func (dve DefaultValueEncoders) Decimal128EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var d128 decimal.Decimal128
	switch t := i.(type) {
	case decimal.Decimal128:
		d128 = t
	case *decimal.Decimal128:
		d128 = *t
	default:
		return ValueEncoderError{
			Name:     "Decimal128EncodeValue",
			Types:    []interface{}{decimal.Decimal128{}, (*decimal.Decimal128)(nil)},
			Received: i,
		}
	}

	return vw.WriteDecimal128(d128)
}

// MinKeyEncodeValue is the ValueEncoderFunc for bson.MinKey.
func (dve DefaultValueEncoders) MinKeyEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	switch i.(type) {
	case bson.MinKeyv2, *bson.MinKeyv2:
	default:
		return ValueEncoderError{
			Name:     "MinKeyEncodeValue",
			Types:    []interface{}{bson.MinKeyv2{}, (*bson.MinKeyv2)(nil)},
			Received: i,
		}
	}

	return vw.WriteMinKey()
}

// MaxKeyEncodeValue is the ValueEncoderFunc for bson.MaxKey.
func (dve DefaultValueEncoders) MaxKeyEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	switch i.(type) {
	case bson.MaxKeyv2, *bson.MaxKeyv2:
	default:
		return ValueEncoderError{
			Name:     "MaxKeyEncodeValue",
			Types:    []interface{}{bson.MaxKeyv2{}, (*bson.MaxKeyv2)(nil)},
			Received: i,
		}
	}

	return vw.WriteMaxKey()
}

// elementEncodeValue is used internally to encode to values
func (dve DefaultValueEncoders) elementEncodeValue(ectx EncodeContext, vw ValueWriter, i interface{}) error {
	elem, ok := i.(*bson.Element)
	if !ok {
		return ValueEncoderError{
			Name:     "elementEncodeValue",
			Types:    []interface{}{(*bson.Element)(nil)},
			Received: i,
		}
	}

	if _, err := elem.Validate(); err != nil {
		return err
	}

	return dve.encodeValue(ectx, vw, elem.Value())
}

// ValueEncodeValue is the ValueEncoderFunc for *bson.Value.
func (dve DefaultValueEncoders) ValueEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	val, ok := i.(*bson.Value)
	if !ok {
		return ValueEncoderError{
			Name:     "ValueEncodeValue",
			Types:    []interface{}{(*bson.Value)(nil)},
			Received: i,
		}
	}

	if err := val.Validate(); err != nil {
		return err
	}

	return dve.encodeValue(ec, vw, val)
}

// JSONNumberEncodeValue is the ValueEncoderFunc for json.Number.
func (dve DefaultValueEncoders) JSONNumberEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var jsnum json.Number
	switch t := i.(type) {
	case json.Number:
		jsnum = t
	case *json.Number:
		jsnum = *t
	default:
		return ValueEncoderError{
			Name:     "JSONNumberEncodeValue",
			Types:    []interface{}{json.Number(""), (*json.Number)(nil)},
			Received: i,
		}
	}

	// Attempt int first, then float64
	if i64, err := jsnum.Int64(); err == nil {
		return dve.IntEncodeValue(ec, vw, i64)
	}

	f64, err := jsnum.Float64()
	if err != nil {
		return err
	}

	return dve.FloatEncodeValue(ec, vw, f64)
}

// URLEncodeValue is the ValueEncoderFunc for url.URL.
func (dve DefaultValueEncoders) URLEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var u *url.URL
	switch t := i.(type) {
	case url.URL:
		u = &t
	case *url.URL:
		u = t
	default:
		return ValueEncoderError{
			Name:     "URLEncodeValue",
			Types:    []interface{}{url.URL{}, (*url.URL)(nil)},
			Received: i,
		}
	}

	return vw.WriteString(u.String())
}

// TimeEncodeValue is the ValueEncoderFunc for time.TIme.
func (dve DefaultValueEncoders) TimeEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var tt time.Time
	switch t := i.(type) {
	case time.Time:
		tt = t
	case *time.Time:
		tt = *t
	default:
		return ValueEncoderError{
			Name:     "TimeEncodeValue",
			Types:    []interface{}{time.Time{}, (*time.Time)(nil)},
			Received: i,
		}
	}

	return vw.WriteDateTime(tt.Unix()*1000 + int64(tt.Nanosecond()/1e6))
}

// ReaderEncodeValue is the ValueEncoderFunc for bson.Reader.
func (dve DefaultValueEncoders) ReaderEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	rdr, ok := i.(bson.Reader)
	if !ok {
		return ValueEncoderError{
			Name:     "ReaderEncodeValue",
			Types:    []interface{}{bson.Reader{}},
			Received: i,
		}
	}

	return (Copier{r: ec.Registry}).CopyDocumentFromBytes(vw, rdr)
}

// ByteSliceEncodeValue is the ValueEncoderFunc for []byte.
func (dve DefaultValueEncoders) ByteSliceEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var slcb []byte
	switch t := i.(type) {
	case []byte:
		slcb = t
	case *[]byte:
		slcb = *t
	default:
		return ValueEncoderError{
			Name:     "ByteSliceEncodeValue",
			Types:    []interface{}{[]byte{}, (*[]byte)(nil)},
			Received: i,
		}
	}

	return vw.WriteBinary(slcb)
}

// ElementSliceEncodeValue is the ValueEncoderFunc for []*bson.Element.
func (dve DefaultValueEncoders) ElementSliceEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	var slce []*bson.Element
	switch t := i.(type) {
	case []*bson.Element:
		slce = t
	case *[]*bson.Element:
		slce = *t
	default:
		return ValueEncoderError{
			Name:     "ElementSliceEncodeValue",
			Types:    []interface{}{[]*bson.Element{}, (*[]*bson.Element)(nil)},
			Received: i,
		}
	}

	return dve.DocumentEncodeValue(ec, vw, (&bson.Document{}).Append(slce...))
}

// MapEncodeValue is the ValueEncoderFunc for map[string]* types.
func (dve DefaultValueEncoders) MapEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	val := reflect.ValueOf(i)
	if val.Kind() != reflect.Map || val.Type().Key().Kind() != reflect.String {
		return errors.New("MapEncodeValue can only encode maps with string keys")
	}

	dw, err := vw.WriteDocument()
	if err != nil {
		return err
	}

	return dve.mapEncodeValue(ec, dw, val, nil)
}

// mapEncodeValue handles encoding of the values of a map. The collisionFn returns
// true if the provided key exists, this is mainly used for inline maps in the
// struct codec.
func (dve DefaultValueEncoders) mapEncodeValue(ec EncodeContext, dw DocumentWriter, val reflect.Value, collisionFn func(string) bool) error {

	var err error
	var encoder ValueEncoder
	switch val.Type().Elem() {
	case tElement:
		encoder = ValueEncoderFunc(dve.elementEncodeValue)
	default:
		encoder, err = ec.LookupEncoder(val.Type().Elem())
		if err != nil {
			return err
		}
	}

	keys := val.MapKeys()
	for _, key := range keys {
		if collisionFn != nil && collisionFn(key.String()) {
			return fmt.Errorf("Key %s of inlined map conflicts with a struct field name", key)
		}
		vw, err := dw.WriteDocumentElement(key.String())
		if err != nil {
			return err
		}

		err = encoder.EncodeValue(ec, vw, val.MapIndex(key).Interface())
		if err != nil {
			return err
		}
	}

	return dw.WriteDocumentEnd()
}

// SliceEncodeValue is the ValueEncoderFunc for []* types.
func (dve DefaultValueEncoders) SliceEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	val := reflect.ValueOf(i)
	switch val.Kind() {
	case reflect.Array:
	case reflect.Slice:
		if val.IsNil() { // When nil, special case to null
			return vw.WriteNull()
		}
	default:
		return errors.New("SliceEncodeValue can only encode arrays and slices")
	}

	length := val.Len()

	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	// We do this outside of the loop because an array or a slice can only have
	// one element type. If it's the empty interface, we'll use the empty
	// interface codec.
	var encoder ValueEncoder
	switch val.Type().Elem() {
	case tElement:
		encoder = ValueEncoderFunc(dve.elementEncodeValue)
	default:
		encoder, err = ec.LookupEncoder(val.Type().Elem())
		if err != nil {
			return err
		}
	}
	for idx := 0; idx < length; idx++ {
		vw, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		err = encoder.EncodeValue(ec, vw, val.Index(idx).Interface())
		if err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

// EmptyInterfaceEncodeValue is the ValueEncoderFunc for interface{}.
func (dve DefaultValueEncoders) EmptyInterfaceEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	encoder, err := ec.LookupEncoder(reflect.TypeOf(i))
	if err != nil {
		return err
	}

	return encoder.EncodeValue(ec, vw, i)
}

// ValueMarshalerEncodeValue is the ValueEncoderFunc for ValueMarshaler implementations.
func (dve DefaultValueEncoders) ValueMarshalerEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	vm, ok := i.(ValueMarshaler)
	if !ok {
		return ValueEncoderError{
			Name:     "ValueMarshalerEncodeValue",
			Types:    []interface{}{(ValueMarshaler)(nil)},
			Received: i,
		}
	}

	t, val, err := vm.MarshalBSONValue()
	if err != nil {
		return err
	}
	return Copier{r: ec.Registry}.CopyValueFromBytes(vw, t, val)
}

// encodeValue does not validation, and the callers must perform validation on val before calling
// this method.
func (dve DefaultValueEncoders) encodeValue(ec EncodeContext, vw ValueWriter, val *bson.Value) error {
	var err error
	switch val.Type() {
	case bson.TypeDouble:
		err = vw.WriteDouble(val.Double())
	case bson.TypeString:
		err = vw.WriteString(val.StringValue())
	case bson.TypeEmbeddedDocument:
		var encoder ValueEncoder
		encoder, err = ec.LookupEncoder(tDocument)
		if err != nil {
			break
		}
		err = encoder.EncodeValue(ec, vw, val.MutableDocument())
	case bson.TypeArray:
		var encoder ValueEncoder
		encoder, err = ec.LookupEncoder(tArray)
		if err != nil {
			break
		}
		err = encoder.EncodeValue(ec, vw, val.MutableArray())
	case bson.TypeBinary:
		// TODO: FIX THIS (╯°□°）╯︵ ┻━┻
		subtype, data := val.Binary()
		err = vw.WriteBinaryWithSubtype(data, subtype)
	case bson.TypeUndefined:
		err = vw.WriteUndefined()
	case bson.TypeObjectID:
		err = vw.WriteObjectID(val.ObjectID())
	case bson.TypeBoolean:
		err = vw.WriteBoolean(val.Boolean())
	case bson.TypeDateTime:
		err = vw.WriteDateTime(val.DateTime())
	case bson.TypeNull:
		err = vw.WriteNull()
	case bson.TypeRegex:
		err = vw.WriteRegex(val.Regex())
	case bson.TypeDBPointer:
		err = vw.WriteDBPointer(val.DBPointer())
	case bson.TypeJavaScript:
		err = vw.WriteJavascript(val.JavaScript())
	case bson.TypeSymbol:
		err = vw.WriteSymbol(val.Symbol())
	case bson.TypeCodeWithScope:
		code, scope := val.MutableJavaScriptWithScope()

		var cwsw DocumentWriter
		cwsw, err = vw.WriteCodeWithScope(code)
		if err != nil {
			break
		}

		err = dve.encodeDocument(ec, cwsw, scope)
	case bson.TypeInt32:
		err = vw.WriteInt32(val.Int32())
	case bson.TypeTimestamp:
		err = vw.WriteTimestamp(val.Timestamp())
	case bson.TypeInt64:
		err = vw.WriteInt64(val.Int64())
	case bson.TypeDecimal128:
		err = vw.WriteDecimal128(val.Decimal128())
	case bson.TypeMinKey:
		err = vw.WriteMinKey()
	case bson.TypeMaxKey:
		err = vw.WriteMaxKey()
	default:
		err = fmt.Errorf("%T is not a valid BSON type to encode", val.Type())
	}

	return err
}
