package bson

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/bson/bsonrw"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

var primitiveCodecs PrimitiveCodecs

// PrimitiveCodecs is a namespace for all of the default bsoncodec.Codecs for the primitive types
// defined in this package.
type PrimitiveCodecs struct{}

// RegisterPrimitiveCodecs will register the encode and decode methods attached to PrimitiveCodecs
// with the provided RegistryBuilder. if rb is nil, a new empty RegistryBuilder will be created.
func (pc PrimitiveCodecs) RegisterPrimitiveCodecs(rb *bsoncodec.RegistryBuilder) {
	if rb == nil {
		panic(errors.New("argument to RegisterPrimitiveCodecs must not be nil"))
	}

	rb.
		RegisterEncoder(tDocument, bsoncodec.ValueEncoderFunc(pc.DocumentEncodeValue)).
		RegisterEncoder(tArray, bsoncodec.ValueEncoderFunc(pc.ArrayEncodeValue)).
		RegisterEncoder(tValue, bsoncodec.ValueEncoderFunc(pc.ValueEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tElementSlice), bsoncodec.ValueEncoderFunc(pc.ElementSliceEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tBinary), bsoncodec.ValueEncoderFunc(pc.BinaryEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tUndefined), bsoncodec.ValueEncoderFunc(pc.UndefinedEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tDateTime), bsoncodec.ValueEncoderFunc(pc.DateTimeEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tNull), bsoncodec.ValueEncoderFunc(pc.NullEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tRegex), bsoncodec.ValueEncoderFunc(pc.RegexEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tDBPointer), bsoncodec.ValueEncoderFunc(pc.DBPointerEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tCodeWithScope), bsoncodec.ValueEncoderFunc(pc.CodeWithScopeEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tTimestamp), bsoncodec.ValueEncoderFunc(pc.TimestampEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tMinKey), bsoncodec.ValueEncoderFunc(pc.MinKeyEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tMaxKey), bsoncodec.ValueEncoderFunc(pc.MaxKeyEncodeValue)).
		RegisterEncoder(reflect.PtrTo(tReader), bsoncodec.ValueEncoderFunc(pc.ReaderEncodeValue)).
		RegisterDecoder(tDocument, bsoncodec.ValueDecoderFunc(pc.DocumentDecodeValue)).
		RegisterDecoder(tArray, bsoncodec.ValueDecoderFunc(pc.ArrayDecodeValue)).
		RegisterDecoder(tValue, bsoncodec.ValueDecoderFunc(pc.ValueDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tElementSlice), bsoncodec.ValueDecoderFunc(pc.ElementSliceDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tBinary), bsoncodec.ValueDecoderFunc(pc.BinaryDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tUndefined), bsoncodec.ValueDecoderFunc(pc.UndefinedDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tDateTime), bsoncodec.ValueDecoderFunc(pc.DateTimeDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tNull), bsoncodec.ValueDecoderFunc(pc.NullDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tRegex), bsoncodec.ValueDecoderFunc(pc.RegexDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tDBPointer), bsoncodec.ValueDecoderFunc(pc.DBPointerDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tCodeWithScope), bsoncodec.ValueDecoderFunc(pc.CodeWithScopeDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tTimestamp), bsoncodec.ValueDecoderFunc(pc.TimestampDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tMinKey), bsoncodec.ValueDecoderFunc(pc.MinKeyDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tMaxKey), bsoncodec.ValueDecoderFunc(pc.MaxKeyDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tReader), bsoncodec.ValueDecoderFunc(pc.ReaderDecodeValue)).
		RegisterDecoder(reflect.PtrTo(tEmpty), bsoncodec.ValueDecoderFunc(pc.EmptyInterfaceDecodeValue))
}

// JavaScriptEncodeValue is the ValueEncoderFunc for the JavaScriptPrimitive type.
func (PrimitiveCodecs) JavaScriptEncodeValue(ectx bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var js JavaScriptCode
	switch t := i.(type) {
	case JavaScriptCode:
		js = t
	case *JavaScriptCode:
		js = *t
	default:
		return bsoncodec.ValueEncoderError{
			Name:     "JavaScriptEncodeValue",
			Types:    []interface{}{JavaScriptCode(""), (*JavaScriptCode)(nil)},
			Received: i,
		}
	}

	return vw.WriteJavascript(string(js))
}

// SymbolEncodeValue is the ValueEncoderFunc for the SymbolPrimitive type.
func (PrimitiveCodecs) SymbolEncodeValue(ectx bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var symbol Symbol
	switch t := i.(type) {
	case Symbol:
		symbol = t
	case *Symbol:
		symbol = *t
	default:
		return bsoncodec.ValueEncoderError{
			Name:     "SymbolEncodeValue",
			Types:    []interface{}{Symbol(""), (*Symbol)(nil)},
			Received: i,
		}
	}

	return vw.WriteJavascript(string(symbol))
}

// JavaScriptDecodeValue is the ValueDecoderFunc for the JavaScriptPrimitive type.
func (PrimitiveCodecs) JavaScriptDecodeValue(dctx bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.JavaScript {
		return fmt.Errorf("cannot decode %v into a JavaScriptPrimitive", vr.Type())
	}

	js, err := vr.ReadJavascript()
	if err != nil {
		return err
	}

	if target, ok := i.(*JavaScriptCode); ok && target != nil {
		*target = JavaScriptCode(js)
		return nil
	}

	if target, ok := i.(**JavaScriptCode); ok && target != nil {
		pjs := *target
		if pjs == nil {
			pjs = new(JavaScriptCode)
		}
		*pjs = JavaScriptCode(js)
		*target = pjs
		return nil
	}

	return bsoncodec.ValueDecoderError{
		Name:     "JavaScriptDecodeValue",
		Types:    []interface{}{(*JavaScriptCode)(nil), (**JavaScriptCode)(nil)},
		Received: i,
	}
}

// SymbolDecodeValue is the ValueDecoderFunc for the SymbolPrimitive type.
func (PrimitiveCodecs) SymbolDecodeValue(dctx bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.Symbol {
		return fmt.Errorf("cannot decode %v into a SymbolPrimitive", vr.Type())
	}

	symbol, err := vr.ReadSymbol()
	if err != nil {
		return err
	}

	if target, ok := i.(*Symbol); ok && target != nil {
		*target = Symbol(symbol)
		return nil
	}

	if target, ok := i.(**Symbol); ok && target != nil {
		psymbol := *target
		if psymbol == nil {
			psymbol = new(Symbol)
		}
		*psymbol = Symbol(symbol)
		*target = psymbol
		return nil
	}

	return bsoncodec.ValueDecoderError{Name: "SymbolDecodeValue", Types: []interface{}{(*Symbol)(nil), (**Symbol)(nil)}, Received: i}
}

// BinaryEncodeValue is the ValueEncoderFunc for Binary.
func (PrimitiveCodecs) BinaryEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var b Binary
	switch t := i.(type) {
	case Binary:
		b = t
	case *Binary:
		b = *t
	default:
		return bsoncodec.ValueEncoderError{
			Name:     "BinaryEncodeValue",
			Types:    []interface{}{Binary{}, (*Binary)(nil)},
			Received: i,
		}
	}

	return vw.WriteBinaryWithSubtype(b.Data, b.Subtype)
}

// BinaryDecodeValue is the ValueDecoderFunc for Binary.
func (PrimitiveCodecs) BinaryDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.Binary {
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

	return bsoncodec.ValueDecoderError{Name: "BinaryDecodeValue", Types: []interface{}{(*Binary)(nil)}, Received: i}
}

// UndefinedEncodeValue is the ValueEncoderFunc for Undefined.
func (PrimitiveCodecs) UndefinedEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	switch i.(type) {
	case Undefinedv2, *Undefinedv2:
	default:
		return bsoncodec.ValueEncoderError{
			Name:     "UndefinedEncodeValue",
			Types:    []interface{}{Undefinedv2{}, (*Undefinedv2)(nil)},
			Received: i,
		}
	}

	return vw.WriteUndefined()
}

// UndefinedDecodeValue is the ValueDecoderFunc for Undefined.
func (PrimitiveCodecs) UndefinedDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.Undefined {
		return fmt.Errorf("cannot decode %v into an Undefined", vr.Type())
	}

	target, ok := i.(*Undefinedv2)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{Name: "UndefinedDecodeValue", Types: []interface{}{(*Undefinedv2)(nil)}, Received: i}
	}

	*target = Undefinedv2{}
	return vr.ReadUndefined()
}

// DateTimeEncodeValue is the ValueEncoderFunc for DateTime.
func (PrimitiveCodecs) DateTimeEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var dt DateTime
	switch t := i.(type) {
	case DateTime:
		dt = t
	case *DateTime:
		dt = *t
	default:
		return bsoncodec.ValueEncoderError{
			Name:     "DateTimeEncodeValue",
			Types:    []interface{}{DateTime(0), (*DateTime)(nil)},
			Received: i,
		}
	}

	return vw.WriteDateTime(int64(dt))
}

// DateTimeDecodeValue is the ValueDecoderFunc for DateTime.
func (PrimitiveCodecs) DateTimeDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.DateTime {
		return fmt.Errorf("cannot decode %v into a DateTime", vr.Type())
	}

	target, ok := i.(*DateTime)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{Name: "DateTimeDecodeValue", Types: []interface{}{(*DateTime)(nil)}, Received: i}
	}

	dt, err := vr.ReadDateTime()
	if err != nil {
		return err
	}

	*target = DateTime(dt)
	return nil
}

// NullEncodeValue is the ValueEncoderFunc for Null.
func (PrimitiveCodecs) NullEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	switch i.(type) {
	case Nullv2, *Nullv2:
	default:
		return bsoncodec.ValueEncoderError{
			Name:     "NullEncodeValue",
			Types:    []interface{}{Nullv2{}, (*Nullv2)(nil)},
			Received: i,
		}
	}

	return vw.WriteNull()
}

// NullDecodeValue is the ValueDecoderFunc for Null.
func (PrimitiveCodecs) NullDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.Null {
		return fmt.Errorf("cannot decode %v into a Null", vr.Type())
	}

	target, ok := i.(*Nullv2)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{Name: "NullDecodeValue", Types: []interface{}{(*Nullv2)(nil)}, Received: i}
	}

	*target = Nullv2{}
	return vr.ReadNull()
}

// RegexEncodeValue is the ValueEncoderFunc for Regex.
func (PrimitiveCodecs) RegexEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var regex Regex
	switch t := i.(type) {
	case Regex:
		regex = t
	case *Regex:
		regex = *t
	default:
		return bsoncodec.ValueEncoderError{
			Name:     "RegexEncodeValue",
			Types:    []interface{}{Regex{}, (*Regex)(nil)},
			Received: i,
		}
	}

	return vw.WriteRegex(regex.Pattern, regex.Options)
}

// RegexDecodeValue is the ValueDecoderFunc for Regex.
func (PrimitiveCodecs) RegexDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.Regex {
		return fmt.Errorf("cannot decode %v into a Regex", vr.Type())
	}

	target, ok := i.(*Regex)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{Name: "RegexDecodeValue", Types: []interface{}{(*Regex)(nil)}, Received: i}
	}

	pattern, options, err := vr.ReadRegex()
	if err != nil {
		return err
	}

	*target = Regex{Pattern: pattern, Options: options}
	return nil
}

// DBPointerEncodeValue is the ValueEncoderFunc for DBPointer.
func (PrimitiveCodecs) DBPointerEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var dbp DBPointer
	switch t := i.(type) {
	case DBPointer:
		dbp = t
	case *DBPointer:
		dbp = *t
	default:
		return bsoncodec.ValueEncoderError{
			Name:     "DBPointerEncodeValue",
			Types:    []interface{}{DBPointer{}, (*DBPointer)(nil)},
			Received: i,
		}
	}

	return vw.WriteDBPointer(dbp.DB, dbp.Pointer)
}

// DBPointerDecodeValue is the ValueDecoderFunc for DBPointer.
func (PrimitiveCodecs) DBPointerDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.DBPointer {
		return fmt.Errorf("cannot decode %v into a DBPointer", vr.Type())
	}

	target, ok := i.(*DBPointer)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{Name: "DBPointerDecodeValue", Types: []interface{}{(*DBPointer)(nil)}, Received: i}
	}

	ns, pointer, err := vr.ReadDBPointer()
	if err != nil {
		return err
	}

	*target = DBPointer{DB: ns, Pointer: pointer}
	return nil
}

// DocumentEncodeValue is the ValueEncoderFunc for *Document.
func (pc PrimitiveCodecs) DocumentEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	doc, ok := i.(*Document)
	if !ok {
		return bsoncodec.ValueEncoderError{Name: "DocumentEncodeValue", Types: []interface{}{(*Document)(nil), (**Document)(nil)}, Received: i}
	}

	dw, err := vw.WriteDocument()
	if err != nil {
		return err
	}

	return pc.encodeDocument(ec, dw, doc)
}

// CodeWithScopeEncodeValue is the ValueEncoderFunc for CodeWithScope.
func (pc PrimitiveCodecs) CodeWithScopeEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var cws CodeWithScope
	switch t := i.(type) {
	case CodeWithScope:
		cws = t
	case *CodeWithScope:
		cws = *t
	default:
		return bsoncodec.ValueEncoderError{
			Name:     "CodeWithScopeEncodeValue",
			Types:    []interface{}{CodeWithScope{}, (*CodeWithScope)(nil)},
			Received: i,
		}
	}

	dw, err := vw.WriteCodeWithScope(cws.Code)
	if err != nil {
		return err
	}

	return pc.encodeDocument(ec, dw, cws.Scope)
}

// CodeWithScopeDecodeValue is the ValueDecoderFunc for CodeWithScope.
func (pc PrimitiveCodecs) CodeWithScopeDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.CodeWithScope {
		return fmt.Errorf("cannot decode %v into a CodeWithScope", vr.Type())
	}

	target, ok := i.(*CodeWithScope)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{
			Name:     "CodeWithScopeDecodeValue",
			Types:    []interface{}{(*CodeWithScope)(nil)},
			Received: i,
		}
	}

	code, dr, err := vr.ReadCodeWithScope()
	if err != nil {
		return err
	}

	var scope *Document
	err = pc.decodeDocument(dc, dr, &scope)
	if err != nil {
		return err
	}

	*target = CodeWithScope{Code: code, Scope: scope}
	return nil
}

// TimestampEncodeValue is the ValueEncoderFunc for Timestamp.
func (PrimitiveCodecs) TimestampEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var ts Timestamp
	switch t := i.(type) {
	case Timestamp:
		ts = t
	case *Timestamp:
		ts = *t
	default:
		return bsoncodec.ValueEncoderError{
			Name:     "TimestampEncodeValue",
			Types:    []interface{}{Timestamp{}, (*Timestamp)(nil)},
			Received: i,
		}
	}

	return vw.WriteTimestamp(ts.T, ts.I)
}

// TimestampDecodeValue is the ValueDecoderFunc for Timestamp.
func (PrimitiveCodecs) TimestampDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.Timestamp {
		return fmt.Errorf("cannot decode %v into a Timestamp", vr.Type())
	}

	target, ok := i.(*Timestamp)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{Name: "TimestampDecodeValue", Types: []interface{}{(*Timestamp)(nil)}, Received: i}
	}

	t, incr, err := vr.ReadTimestamp()
	if err != nil {
		return err
	}

	*target = Timestamp{T: t, I: incr}
	return nil
}

// MinKeyEncodeValue is the ValueEncoderFunc for MinKey.
func (PrimitiveCodecs) MinKeyEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	switch i.(type) {
	case MinKeyv2, *MinKeyv2:
	default:
		return bsoncodec.ValueEncoderError{
			Name:     "MinKeyEncodeValue",
			Types:    []interface{}{MinKeyv2{}, (*MinKeyv2)(nil)},
			Received: i,
		}
	}

	return vw.WriteMinKey()
}

// MinKeyDecodeValue is the ValueDecoderFunc for MinKey.
func (PrimitiveCodecs) MinKeyDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.MinKey {
		return fmt.Errorf("cannot decode %v into a MinKey", vr.Type())
	}

	target, ok := i.(*MinKeyv2)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{Name: "MinKeyDecodeValue", Types: []interface{}{(*MinKeyv2)(nil)}, Received: i}
	}

	*target = MinKeyv2{}
	return vr.ReadMinKey()
}

// MaxKeyEncodeValue is the ValueEncoderFunc for MaxKey.
func (PrimitiveCodecs) MaxKeyEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	switch i.(type) {
	case MaxKeyv2, *MaxKeyv2:
	default:
		return bsoncodec.ValueEncoderError{
			Name:     "MaxKeyEncodeValue",
			Types:    []interface{}{MaxKeyv2{}, (*MaxKeyv2)(nil)},
			Received: i,
		}
	}

	return vw.WriteMaxKey()
}

// MaxKeyDecodeValue is the ValueDecoderFunc for MaxKey.
func (PrimitiveCodecs) MaxKeyDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	if vr.Type() != bsontype.MaxKey {
		return fmt.Errorf("cannot decode %v into a MaxKey", vr.Type())
	}

	target, ok := i.(*MaxKeyv2)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{Name: "MaxKeyDecodeValue", Types: []interface{}{(*MaxKeyv2)(nil)}, Received: i}
	}

	*target = MaxKeyv2{}
	return vr.ReadMaxKey()
}

// ValueEncodeValue is the ValueEncoderFunc for *Value.
func (pc PrimitiveCodecs) ValueEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	val, ok := i.(*Value)
	if !ok {
		return bsoncodec.ValueEncoderError{
			Name:     "ValueEncodeValue",
			Types:    []interface{}{(*Value)(nil)},
			Received: i,
		}
	}

	if err := val.Validate(); err != nil {
		return err
	}

	return pc.encodeValue(ec, vw, val)
}

// ValueDecodeValue is the ValueDecoderFunc for *Value.
func (pc PrimitiveCodecs) ValueDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	pval, ok := i.(**Value)
	if !ok {
		return bsoncodec.ValueDecoderError{Name: "ValueDecodeValue", Types: []interface{}{(**Value)(nil)}, Received: i}
	}

	if pval == nil {
		return errors.New("ValueDecodeValue can only be used to decode non-nil **Value")
	}

	return pc.valueDecodeValue(dc, vr, pval)
}

// ReaderEncodeValue is the ValueEncoderFunc for Reader.
func (PrimitiveCodecs) ReaderEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	rdr, ok := i.(Reader)
	if !ok {
		return bsoncodec.ValueEncoderError{
			Name:     "ReaderEncodeValue",
			Types:    []interface{}{Reader{}},
			Received: i,
		}
	}

	return bsonrw.Copier{}.CopyDocumentFromBytes(vw, rdr)
}

// ReaderDecodeValue is the ValueDecoderFunc for Reader.
func (PrimitiveCodecs) ReaderDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	rdr, ok := i.(*Reader)
	if !ok {
		return bsoncodec.ValueDecoderError{Name: "ReaderDecodeValue", Types: []interface{}{(*Reader)(nil)}, Received: i}
	}

	if rdr == nil {
		return errors.New("ReaderDecodeValue can only be used to decode non-nil *Reader")
	}

	if *rdr == nil {
		*rdr = make(Reader, 0)
	} else {
		*rdr = (*rdr)[:0]
	}

	var err error
	*rdr, err = bsonrw.Copier{}.AppendDocumentBytes(*rdr, vr)
	return err
}

// ElementSliceEncodeValue is the ValueEncoderFunc for []*Element.
func (pc PrimitiveCodecs) ElementSliceEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var slce []*Element
	switch t := i.(type) {
	case []*Element:
		slce = t
	case *[]*Element:
		slce = *t
	default:
		return bsoncodec.ValueEncoderError{
			Name:     "ElementSliceEncodeValue",
			Types:    []interface{}{[]*Element{}, (*[]*Element)(nil)},
			Received: i,
		}
	}

	return pc.DocumentEncodeValue(ec, vw, (&Document{}).Append(slce...))
}

// ElementSliceDecodeValue is the ValueDecoderFunc for []*Element.
func (pc PrimitiveCodecs) ElementSliceDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	dr, err := vr.ReadDocument()
	if err != nil {
		return err
	}
	elems := make([]*Element, 0)
	for {
		key, vr, err := dr.ReadElement()
		if err == bsonrw.ErrEOD {
			break
		}
		if err != nil {
			return err
		}

		var elem *Element
		err = pc.elementDecodeValue(dc, vr, key, &elem)
		if err != nil {
			return err
		}

		elems = append(elems, elem)
	}

	target, ok := i.(*[]*Element)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{Name: "ElementSliceDecodeValue", Types: []interface{}{(*[]*Element)(nil)}, Received: i}
	}

	*target = elems
	return nil
}

// DocumentDecodeValue is the ValueDecoderFunc for *Document.
func (pc PrimitiveCodecs) DocumentDecodeValue(dctx bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	doc, ok := i.(**Document)
	if !ok {
		return bsoncodec.ValueDecoderError{Name: "DocumentDecodeValue", Types: []interface{}{(**Document)(nil)}, Received: i}
	}

	if doc == nil {
		return errors.New("DocumentDecodeValue can only be used to decode non-nil **Document")
	}

	dr, err := vr.ReadDocument()
	if err != nil {
		return err
	}

	return pc.decodeDocument(dctx, dr, doc)
}

// ArrayEncodeValue is the ValueEncoderFunc for *Array.
func (pc PrimitiveCodecs) ArrayEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	arr, ok := i.(*Array)
	if !ok {
		return bsoncodec.ValueEncoderError{Name: "ArrayEncodeValue", Types: []interface{}{(*Array)(nil)}, Received: i}
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

		err = pc.encodeValue(ec, dvw, val)

		if err != nil {
			return err
		}
	}

	if err := itr.Err(); err != nil {
		return err
	}

	return aw.WriteArrayEnd()
}

// ArrayDecodeValue is the ValueDecoderFunc for *Array.
func (pc PrimitiveCodecs) ArrayDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	parr, ok := i.(**Array)
	if !ok {
		return bsoncodec.ValueDecoderError{Name: "ArrayDecodeValue", Types: []interface{}{(**Array)(nil)}, Received: i}
	}

	if parr == nil {
		return errors.New("ArrayDecodeValue can only be used to decode non-nil **Array")
	}

	ar, err := vr.ReadArray()
	if err != nil {
		return err
	}

	arr := NewArray()
	for {
		vr, err := ar.ReadValue()
		if err == bsonrw.ErrEOA {
			break
		}
		if err != nil {
			return err
		}

		var val *Value
		err = pc.valueDecodeValue(dc, vr, &val)
		if err != nil {
			return err
		}

		arr.Append(val)
	}

	*parr = arr
	return nil
}

// EmptyInterfaceDecodeValue is the ValueDecoderFunc for interface{}.
func (PrimitiveCodecs) EmptyInterfaceDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
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
	case bsontype.EmbeddedDocument:
		val = new(*Document)
		rtype = tDocument
		fn = func() { *target = *val.(**Document) }
	case bsontype.Array:
		val = new(*Array)
		rtype = tArray
		fn = func() { *target = *val.(**Array) }
	case bsontype.Binary:
		val = new(Binary)
		rtype = tBinary
		fn = func() { *target = *(val.(*Binary)) }
	case bsontype.Undefined:
		val = new(Undefinedv2)
		rtype = tUndefined
		fn = func() { *target = *(val.(*Undefinedv2)) }
	case bsontype.ObjectID:
		val = new(objectid.ObjectID)
		rtype = tOID
		fn = func() { *target = *(val.(*objectid.ObjectID)) }
	case bsontype.Boolean:
		val = new(bool)
		rtype = tBool
		fn = func() { *target = *(val.(*bool)) }
	case bsontype.DateTime:
		val = new(DateTime)
		rtype = tDateTime
		fn = func() { *target = *(val.(*DateTime)) }
	case bsontype.Null:
		val = new(Nullv2)
		rtype = tNull
		fn = func() { *target = *(val.(*Nullv2)) }
	case bsontype.Regex:
		val = new(Regex)
		rtype = tRegex
		fn = func() { *target = *(val.(*Regex)) }
	case bsontype.DBPointer:
		val = new(DBPointer)
		rtype = tDBPointer
		fn = func() { *target = *(val.(*DBPointer)) }
	case bsontype.JavaScript:
		val = new(JavaScriptCode)
		rtype = tJavaScriptCode
		fn = func() { *target = *(val.(*JavaScriptCode)) }
	case bsontype.Symbol:
		val = new(Symbol)
		rtype = tSymbol
		fn = func() { *target = *(val.(*Symbol)) }
	case bsontype.CodeWithScope:
		val = new(CodeWithScope)
		rtype = tCodeWithScope
		fn = func() { *target = *(val.(*CodeWithScope)) }
	case bsontype.Int32:
		val = new(int32)
		rtype = tInt32
		fn = func() { *target = *(val.(*int32)) }
	case bsontype.Int64:
		val = new(int64)
		rtype = tInt64
		fn = func() { *target = *(val.(*int64)) }
	case bsontype.Timestamp:
		val = new(Timestamp)
		rtype = tTimestamp
		fn = func() { *target = *(val.(*Timestamp)) }
	case bsontype.Decimal128:
		val = new(decimal.Decimal128)
		rtype = tDecimal
		fn = func() { *target = *(val.(*decimal.Decimal128)) }
	case bsontype.MinKey:
		val = new(MinKeyv2)
		rtype = tMinKey
		fn = func() { *target = *(val.(*MinKeyv2)) }
	case bsontype.MaxKey:
		val = new(MaxKeyv2)
		rtype = tMaxKey
		fn = func() { *target = *(val.(*MaxKeyv2)) }
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

// encodeDocument is a separate function that we use because CodeWithScope
// returns us a DocumentWriter and we need to do the same logic that we would do
// for a document but cannot use a Codec.
func (pc PrimitiveCodecs) encodeDocument(ec bsoncodec.EncodeContext, dw bsonrw.DocumentWriter, doc *Document) error {
	itr := doc.Iterator()

	for itr.Next() {
		elem := itr.Element()
		dvw, err := dw.WriteDocumentElement(elem.Key())
		if err != nil {
			return err
		}

		val := elem.Value()
		err = pc.encodeValue(ec, dvw, val)

		if err != nil {
			return err
		}
	}

	if err := itr.Err(); err != nil {
		return err
	}

	return dw.WriteDocumentEnd()
}

func (pc PrimitiveCodecs) elementDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, key string, elem **Element) error {
	switch vr.Type() {
	case bsontype.Double:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		*elem = EC.Double(key, f64)
	case bsontype.String:
		str, err := vr.ReadString()
		if err != nil {
			return err
		}
		*elem = EC.String(key, str)
	case bsontype.EmbeddedDocument:
		decoder, err := dc.LookupDecoder(tDocument)
		if err != nil {
			return err
		}
		var embeddedDoc *Document
		err = decoder.DecodeValue(dc, vr, &embeddedDoc)
		if err != nil {
			return err
		}
		*elem = EC.SubDocument(key, embeddedDoc)
	case bsontype.Array:
		decoder, err := dc.LookupDecoder(tArray)
		if err != nil {
			return err
		}
		var arr *Array
		err = decoder.DecodeValue(dc, vr, &arr)
		if err != nil {
			return err
		}
		*elem = EC.Array(key, arr)
	case bsontype.Binary:
		data, subtype, err := vr.ReadBinary()
		if err != nil {
			return err
		}
		*elem = EC.BinaryWithSubtype(key, data, subtype)
	case bsontype.Undefined:
		err := vr.ReadUndefined()
		if err != nil {
			return err
		}
		*elem = EC.Undefined(key)
	case bsontype.ObjectID:
		oid, err := vr.ReadObjectID()
		if err != nil {
			return err
		}
		*elem = EC.ObjectID(key, oid)
	case bsontype.Boolean:
		b, err := vr.ReadBoolean()
		if err != nil {
			return err
		}
		*elem = EC.Boolean(key, b)
	case bsontype.DateTime:
		dt, err := vr.ReadDateTime()
		if err != nil {
			return err
		}
		*elem = EC.DateTime(key, dt)
	case bsontype.Null:
		err := vr.ReadNull()
		if err != nil {
			return err
		}
		*elem = EC.Null(key)
	case bsontype.Regex:
		pattern, options, err := vr.ReadRegex()
		if err != nil {
			return err
		}
		*elem = EC.Regex(key, pattern, options)
	case bsontype.DBPointer:
		ns, pointer, err := vr.ReadDBPointer()
		if err != nil {
			return err
		}
		*elem = EC.DBPointer(key, ns, pointer)
	case bsontype.JavaScript:
		js, err := vr.ReadJavascript()
		if err != nil {
			return err
		}
		*elem = EC.JavaScript(key, js)
	case bsontype.Symbol:
		symbol, err := vr.ReadSymbol()
		if err != nil {
			return err
		}
		*elem = EC.Symbol(key, symbol)
	case bsontype.CodeWithScope:
		code, scope, err := vr.ReadCodeWithScope()
		if err != nil {
			return err
		}
		scopeDoc := new(*Document)
		err = pc.decodeDocument(dc, scope, scopeDoc)
		if err != nil {
			return err
		}
		*elem = EC.CodeWithScope(key, code, *scopeDoc)
	case bsontype.Int32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		*elem = EC.Int32(key, i32)
	case bsontype.Timestamp:
		t, i, err := vr.ReadTimestamp()
		if err != nil {
			return err
		}
		*elem = EC.Timestamp(key, t, i)
	case bsontype.Int64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return err
		}
		*elem = EC.Int64(key, i64)
	case bsontype.Decimal128:
		d128, err := vr.ReadDecimal128()
		if err != nil {
			return err
		}
		*elem = EC.Decimal128(key, d128)
	case bsontype.MinKey:
		err := vr.ReadMinKey()
		if err != nil {
			return err
		}
		*elem = EC.MinKey(key)
	case bsontype.MaxKey:
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

func (pc PrimitiveCodecs) decodeDocument(dctx bsoncodec.DecodeContext, dr bsonrw.DocumentReader, pdoc **Document) error {
	doc := NewDocument()
	for {
		key, vr, err := dr.ReadElement()
		if err == bsonrw.ErrEOD {
			break
		}
		if err != nil {
			return err
		}

		var elem *Element
		err = pc.elementDecodeValue(dctx, vr, key, &elem)
		if err != nil {
			return err
		}

		doc.Append(elem)
	}

	*pdoc = doc
	return nil
}

// encodeValue does not validation, and the callers must perform validation on val before calling
// this method.
func (pc PrimitiveCodecs) encodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val *Value) error {
	var err error
	switch val.Type() {
	case bsontype.Double:
		err = vw.WriteDouble(val.Double())
	case bsontype.String:
		err = vw.WriteString(val.StringValue())
	case bsontype.EmbeddedDocument:
		var encoder bsoncodec.ValueEncoder
		encoder, err = ec.LookupEncoder(tDocument)
		if err != nil {
			break
		}
		err = encoder.EncodeValue(ec, vw, val.MutableDocument())
	case bsontype.Array:
		var encoder bsoncodec.ValueEncoder
		encoder, err = ec.LookupEncoder(tArray)
		if err != nil {
			break
		}
		err = encoder.EncodeValue(ec, vw, val.MutableArray())
	case bsontype.Binary:
		// TODO: FIX THIS (╯°□°）╯︵ ┻━┻
		subtype, data := val.Binary()
		err = vw.WriteBinaryWithSubtype(data, subtype)
	case bsontype.Undefined:
		err = vw.WriteUndefined()
	case bsontype.ObjectID:
		err = vw.WriteObjectID(val.ObjectID())
	case bsontype.Boolean:
		err = vw.WriteBoolean(val.Boolean())
	case bsontype.DateTime:
		err = vw.WriteDateTime(val.DateTime())
	case bsontype.Null:
		err = vw.WriteNull()
	case bsontype.Regex:
		err = vw.WriteRegex(val.Regex())
	case bsontype.DBPointer:
		err = vw.WriteDBPointer(val.DBPointer())
	case bsontype.JavaScript:
		err = vw.WriteJavascript(val.JavaScript())
	case bsontype.Symbol:
		err = vw.WriteSymbol(val.Symbol())
	case bsontype.CodeWithScope:
		code, scope := val.MutableJavaScriptWithScope()

		var cwsw bsonrw.DocumentWriter
		cwsw, err = vw.WriteCodeWithScope(code)
		if err != nil {
			break
		}

		err = pc.encodeDocument(ec, cwsw, scope)
	case bsontype.Int32:
		err = vw.WriteInt32(val.Int32())
	case bsontype.Timestamp:
		err = vw.WriteTimestamp(val.Timestamp())
	case bsontype.Int64:
		err = vw.WriteInt64(val.Int64())
	case bsontype.Decimal128:
		err = vw.WriteDecimal128(val.Decimal128())
	case bsontype.MinKey:
		err = vw.WriteMinKey()
	case bsontype.MaxKey:
		err = vw.WriteMaxKey()
	default:
		err = fmt.Errorf("%T is not a valid BSON type to encode", val.Type())
	}

	return err
}

func (pc PrimitiveCodecs) valueDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, val **Value) error {
	switch vr.Type() {
	case bsontype.Double:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		*val = VC.Double(f64)
	case bsontype.String:
		str, err := vr.ReadString()
		if err != nil {
			return err
		}
		*val = VC.String(str)
	case bsontype.EmbeddedDocument:
		decoder, err := dc.LookupDecoder(tDocument)
		if err != nil {
			return err
		}
		var embeddedDoc *Document
		err = decoder.DecodeValue(dc, vr, &embeddedDoc)
		if err != nil {
			return err
		}
		*val = VC.Document(embeddedDoc)
	case bsontype.Array:
		decoder, err := dc.LookupDecoder(tArray)
		if err != nil {
			return err
		}
		var arr *Array
		err = decoder.DecodeValue(dc, vr, &arr)
		if err != nil {
			return err
		}
		*val = VC.Array(arr)
	case bsontype.Binary:
		data, subtype, err := vr.ReadBinary()
		if err != nil {
			return err
		}
		*val = VC.BinaryWithSubtype(data, subtype)
	case bsontype.Undefined:
		err := vr.ReadUndefined()
		if err != nil {
			return err
		}
		*val = VC.Undefined()
	case bsontype.ObjectID:
		oid, err := vr.ReadObjectID()
		if err != nil {
			return err
		}
		*val = VC.ObjectID(oid)
	case bsontype.Boolean:
		b, err := vr.ReadBoolean()
		if err != nil {
			return err
		}
		*val = VC.Boolean(b)
	case bsontype.DateTime:
		dt, err := vr.ReadDateTime()
		if err != nil {
			return err
		}
		*val = VC.DateTime(dt)
	case bsontype.Null:
		err := vr.ReadNull()
		if err != nil {
			return err
		}
		*val = VC.Null()
	case bsontype.Regex:
		pattern, options, err := vr.ReadRegex()
		if err != nil {
			return err
		}
		*val = VC.Regex(pattern, options)
	case bsontype.DBPointer:
		ns, pointer, err := vr.ReadDBPointer()
		if err != nil {
			return err
		}
		*val = VC.DBPointer(ns, pointer)
	case bsontype.JavaScript:
		js, err := vr.ReadJavascript()
		if err != nil {
			return err
		}
		*val = VC.JavaScript(js)
	case bsontype.Symbol:
		symbol, err := vr.ReadSymbol()
		if err != nil {
			return err
		}
		*val = VC.Symbol(symbol)
	case bsontype.CodeWithScope:
		code, scope, err := vr.ReadCodeWithScope()
		if err != nil {
			return err
		}
		scopeDoc := new(*Document)
		err = pc.decodeDocument(dc, scope, scopeDoc)
		if err != nil {
			return err
		}
		*val = VC.CodeWithScope(code, *scopeDoc)
	case bsontype.Int32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		*val = VC.Int32(i32)
	case bsontype.Timestamp:
		t, i, err := vr.ReadTimestamp()
		if err != nil {
			return err
		}
		*val = VC.Timestamp(t, i)
	case bsontype.Int64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return err
		}
		*val = VC.Int64(i64)
	case bsontype.Decimal128:
		d128, err := vr.ReadDecimal128()
		if err != nil {
			return err
		}
		*val = VC.Decimal128(d128)
	case bsontype.MinKey:
		err := vr.ReadMinKey()
		if err != nil {
			return err
		}
		*val = VC.MinKey()
	case bsontype.MaxKey:
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
