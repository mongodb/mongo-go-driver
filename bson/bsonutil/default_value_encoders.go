package bsonutil

//
// // DefaultValueEncoders is a namespace type for the default ValueEncoders used
// // when creating a registry.
// type DefaultValueEncoders struct{}
//
// // StringEncodeValue is the ValueEncoderFunc for string types.
// func (dve DefaultValueEncoders) StringEncodeValue(ectx EncodeContext, vw ValueWriter, i interface{}) error {
// 	switch t := i.(type) {
// 	case string:
// 		return vw.WriteString(t)
// 	case bson.JavaScriptCode:
// 		return vw.WriteJavascript(string(t))
// 	case bson.Symbol:
// 		return vw.WriteSymbol(string(t))
// 	}
//
// 	val := reflect.ValueOf(i)
// 	if val.Type().Kind() != reflect.String {
// 		return ValueEncoderError{
// 			Name:     "StringEncodeValue",
// 			Types:    []interface{}{string(""), bson.JavaScriptCode(""), bson.Symbol("")},
// 			Received: i,
// 		}
// 	}
//
// 	return vw.WriteString(val.String())
// }
//
// // DocumentEncodeValue is the ValueEncoderFunc for *bson.Document.
// func (dve DefaultValueEncoders) DocumentEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
// 	doc, ok := i.(*bson.Document)
// 	if !ok {
// 		return ValueEncoderError{Name: "DocumentEncodeValue", Types: []interface{}{(*bson.Document)(nil)}, Received: i}
// 	}
//
// 	dw, err := vw.WriteDocument()
// 	if err != nil {
// 		return err
// 	}
//
// 	return dve.encodeDocument(ec, dw, doc)
// }
//
// // ArrayEncodeValue is the ValueEncoderFunc for *bson.Array.
// func (dve DefaultValueEncoders) ArrayEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
// 	arr, ok := i.(*bson.Array)
// 	if !ok {
// 		return ValueEncoderError{Name: "ArrayEncodeValue", Types: []interface{}{(*bson.Array)(nil)}, Received: i}
// 	}
//
// 	aw, err := vw.WriteArray()
// 	if err != nil {
// 		return err
// 	}
//
// 	itr, err := arr.Iterator()
// 	if err != nil {
// 		return err
// 	}
//
// 	for itr.Next() {
// 		val := itr.Value()
// 		dvw, err := aw.WriteArrayElement()
// 		if err != nil {
// 			return err
// 		}
//
// 		err = dve.encodeValue(ec, dvw, val)
//
// 		if err != nil {
// 			return err
// 		}
// 	}
//
// 	if err := itr.Err(); err != nil {
// 		return err
// 	}
//
// 	return aw.WriteArrayEnd()
// }
//
// // encodeDocument is a separate function that we use because CodeWithScope
// // returns us a DocumentWriter and we need to do the same logic that we would do
// // for a document but cannot use a Codec.
// func (dve DefaultValueEncoders) encodeDocument(ec EncodeContext, dw DocumentWriter, doc *bson.Document) error {
// 	itr := doc.Iterator()
//
// 	for itr.Next() {
// 		elem := itr.Element()
// 		dvw, err := dw.WriteDocumentElement(elem.Key())
// 		if err != nil {
// 			return err
// 		}
//
// 		val := elem.Value()
// 		err = dve.encodeValue(ec, dvw, val)
//
// 		if err != nil {
// 			return err
// 		}
// 	}
//
// 	if err := itr.Err(); err != nil {
// 		return err
// 	}
//
// 	return dw.WriteDocumentEnd()
// }
//
// // BinaryEncodeValue is the ValueEncoderFunc for bson.Binary.
// func (dve DefaultValueEncoders) BinaryEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
// 	var b bson.Binary
// 	switch t := i.(type) {
// 	case bson.Binary:
// 		b = t
// 	case *bson.Binary:
// 		b = *t
// 	default:
// 		return ValueEncoderError{
// 			Name:     "BinaryEncodeValue",
// 			Types:    []interface{}{bson.Binary{}, (*bson.Binary)(nil)},
// 			Received: i,
// 		}
// 	}
//
// 	return vw.WriteBinaryWithSubtype(b.Data, b.Subtype)
// }
//
// // UndefinedEncodeValue is the ValueEncoderFunc for bson.Undefined.
// func (dve DefaultValueEncoders) UndefinedEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
// 	switch i.(type) {
// 	case bson.Undefinedv2, *bson.Undefinedv2:
// 	default:
// 		return ValueEncoderError{
// 			Name:     "UndefinedEncodeValue",
// 			Types:    []interface{}{bson.Undefinedv2{}, (*bson.Undefinedv2)(nil)},
// 			Received: i,
// 		}
// 	}
//
// 	return vw.WriteUndefined()
// }
//
// // DateTimeEncodeValue is the ValueEncoderFunc for bson.DateTime.
// func (dve DefaultValueEncoders) DateTimeEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
// 	var dt bson.DateTime
// 	switch t := i.(type) {
// 	case bson.DateTime:
// 		dt = t
// 	case *bson.DateTime:
// 		dt = *t
// 	default:
// 		return ValueEncoderError{
// 			Name:     "DateTimeEncodeValue",
// 			Types:    []interface{}{bson.DateTime(0), (*bson.DateTime)(nil)},
// 			Received: i,
// 		}
// 	}
//
// 	return vw.WriteDateTime(int64(dt))
// }
//
// // NullEncodeValue is the ValueEncoderFunc for bson.Null.
// func (dve DefaultValueEncoders) NullEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
// 	switch i.(type) {
// 	case bson.Nullv2, *bson.Nullv2:
// 	default:
// 		return ValueEncoderError{
// 			Name:     "NullEncodeValue",
// 			Types:    []interface{}{bson.Nullv2{}, (*bson.Nullv2)(nil)},
// 			Received: i,
// 		}
// 	}
//
// 	return vw.WriteNull()
// }
//
// // RegexEncodeValue is the ValueEncoderFunc for bson.Regex.
// func (dve DefaultValueEncoders) RegexEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
// 	var regex bson.Regex
// 	switch t := i.(type) {
// 	case bson.Regex:
// 		regex = t
// 	case *bson.Regex:
// 		regex = *t
// 	default:
// 		return ValueEncoderError{
// 			Name:     "RegexEncodeValue",
// 			Types:    []interface{}{bson.Regex{}, (*bson.Regex)(nil)},
// 			Received: i,
// 		}
// 	}
//
// 	return vw.WriteRegex(regex.Pattern, regex.Options)
// }
//
// // DBPointerEncodeValue is the ValueEncoderFunc for bson.DBPointer.
// func (dve DefaultValueEncoders) DBPointerEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
// 	var dbp bson.DBPointer
// 	switch t := i.(type) {
// 	case bson.DBPointer:
// 		dbp = t
// 	case *bson.DBPointer:
// 		dbp = *t
// 	default:
// 		return ValueEncoderError{
// 			Name:     "DBPointerEncodeValue",
// 			Types:    []interface{}{bson.DBPointer{}, (*bson.DBPointer)(nil)},
// 			Received: i,
// 		}
// 	}
//
// 	return vw.WriteDBPointer(dbp.DB, dbp.Pointer)
// }
//
// // CodeWithScopeEncodeValue is the ValueEncoderFunc for bson.CodeWithScope.
// func (dve DefaultValueEncoders) CodeWithScopeEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
// 	var cws bson.CodeWithScope
// 	switch t := i.(type) {
// 	case bson.CodeWithScope:
// 		cws = t
// 	case *bson.CodeWithScope:
// 		cws = *t
// 	default:
// 		return ValueEncoderError{
// 			Name:     "CodeWithScopeEncodeValue",
// 			Types:    []interface{}{bson.CodeWithScope{}, (*bson.CodeWithScope)(nil)},
// 			Received: i,
// 		}
// 	}
//
// 	dw, err := vw.WriteCodeWithScope(cws.Code)
// 	if err != nil {
// 		return err
// 	}
//
// 	return dve.encodeDocument(ec, dw, cws.Scope)
// }
//
// // TimestampEncodeValue is the ValueEncoderFunc for bson.Timestamp.
// func (dve DefaultValueEncoders) TimestampEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
// 	var ts bson.Timestamp
// 	switch t := i.(type) {
// 	case bson.Timestamp:
// 		ts = t
// 	case *bson.Timestamp:
// 		ts = *t
// 	default:
// 		return ValueEncoderError{
// 			Name:     "TimestampEncodeValue",
// 			Types:    []interface{}{bson.Timestamp{}, (*bson.Timestamp)(nil)},
// 			Received: i,
// 		}
// 	}
//
// 	return vw.WriteTimestamp(ts.T, ts.I)
// }
//
// // MinKeyEncodeValue is the ValueEncoderFunc for bson.MinKey.
// func (dve DefaultValueEncoders) MinKeyEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
// 	switch i.(type) {
// 	case bson.MinKeyv2, *bson.MinKeyv2:
// 	default:
// 		return ValueEncoderError{
// 			Name:     "MinKeyEncodeValue",
// 			Types:    []interface{}{bson.MinKeyv2{}, (*bson.MinKeyv2)(nil)},
// 			Received: i,
// 		}
// 	}
//
// 	return vw.WriteMinKey()
// }
//
// // MaxKeyEncodeValue is the ValueEncoderFunc for bson.MaxKey.
// func (dve DefaultValueEncoders) MaxKeyEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
// 	switch i.(type) {
// 	case bson.MaxKeyv2, *bson.MaxKeyv2:
// 	default:
// 		return ValueEncoderError{
// 			Name:     "MaxKeyEncodeValue",
// 			Types:    []interface{}{bson.MaxKeyv2{}, (*bson.MaxKeyv2)(nil)},
// 			Received: i,
// 		}
// 	}
//
// 	return vw.WriteMaxKey()
// }
//
// // elementEncodeValue is used internally to encode to values
// func (dve DefaultValueEncoders) elementEncodeValue(ectx EncodeContext, vw ValueWriter, i interface{}) error {
// 	elem, ok := i.(*bson.Element)
// 	if !ok {
// 		return ValueEncoderError{
// 			Name:     "elementEncodeValue",
// 			Types:    []interface{}{(*bson.Element)(nil)},
// 			Received: i,
// 		}
// 	}
//
// 	if _, err := elem.Validate(); err != nil {
// 		return err
// 	}
//
// 	return dve.encodeValue(ectx, vw, elem.Value())
// }
//
// // ValueEncodeValue is the ValueEncoderFunc for *bson.Value.
// func (dve DefaultValueEncoders) ValueEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
// 	val, ok := i.(*bson.Value)
// 	if !ok {
// 		return ValueEncoderError{
// 			Name:     "ValueEncodeValue",
// 			Types:    []interface{}{(*bson.Value)(nil)},
// 			Received: i,
// 		}
// 	}
//
// 	if err := val.Validate(); err != nil {
// 		return err
// 	}
//
// 	return dve.encodeValue(ec, vw, val)
// }
//
// // ReaderEncodeValue is the ValueEncoderFunc for bson.Reader.
// func (dve DefaultValueEncoders) ReaderEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
// 	rdr, ok := i.(bson.Reader)
// 	if !ok {
// 		return ValueEncoderError{
// 			Name:     "ReaderEncodeValue",
// 			Types:    []interface{}{bson.Reader{}},
// 			Received: i,
// 		}
// 	}
//
// 	return (Copier{r: ec.Registry}).CopyDocumentFromBytes(vw, rdr)
// }
//
// // ElementSliceEncodeValue is the ValueEncoderFunc for []*bson.Element.
// func (dve DefaultValueEncoders) ElementSliceEncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
// 	var slce []*bson.Element
// 	switch t := i.(type) {
// 	case []*bson.Element:
// 		slce = t
// 	case *[]*bson.Element:
// 		slce = *t
// 	default:
// 		return ValueEncoderError{
// 			Name:     "ElementSliceEncodeValue",
// 			Types:    []interface{}{[]*bson.Element{}, (*[]*bson.Element)(nil)},
// 			Received: i,
// 		}
// 	}
//
// 	return dve.DocumentEncodeValue(ec, vw, (&bson.Document{}).Append(slce...))
// }
//
// // encodeValue does not validation, and the callers must perform validation on val before calling
// // this method.
// func (dve DefaultValueEncoders) encodeValue(ec EncodeContext, vw ValueWriter, val *bson.Value) error {
// 	var err error
// 	switch val.Type() {
// 	case bson.TypeDouble:
// 		err = vw.WriteDouble(val.Double())
// 	case bson.TypeString:
// 		err = vw.WriteString(val.StringValue())
// 	case bson.TypeEmbeddedDocument:
// 		var encoder ValueEncoder
// 		encoder, err = ec.LookupEncoder(tDocument)
// 		if err != nil {
// 			break
// 		}
// 		err = encoder.EncodeValue(ec, vw, val.MutableDocument())
// 	case bson.TypeArray:
// 		var encoder ValueEncoder
// 		encoder, err = ec.LookupEncoder(tArray)
// 		if err != nil {
// 			break
// 		}
// 		err = encoder.EncodeValue(ec, vw, val.MutableArray())
// 	case bson.TypeBinary:
// 		// TODO: FIX THIS (╯°□°）╯︵ ┻━┻
// 		subtype, data := val.Binary()
// 		err = vw.WriteBinaryWithSubtype(data, subtype)
// 	case bson.TypeUndefined:
// 		err = vw.WriteUndefined()
// 	case bson.TypeObjectID:
// 		err = vw.WriteObjectID(val.ObjectID())
// 	case bson.TypeBoolean:
// 		err = vw.WriteBoolean(val.Boolean())
// 	case bson.TypeDateTime:
// 		err = vw.WriteDateTime(val.DateTime())
// 	case bson.TypeNull:
// 		err = vw.WriteNull()
// 	case bson.TypeRegex:
// 		err = vw.WriteRegex(val.Regex())
// 	case bson.TypeDBPointer:
// 		err = vw.WriteDBPointer(val.DBPointer())
// 	case bson.TypeJavaScript:
// 		err = vw.WriteJavascript(val.JavaScript())
// 	case bson.TypeSymbol:
// 		err = vw.WriteSymbol(val.Symbol())
// 	case bson.TypeCodeWithScope:
// 		code, scope := val.MutableJavaScriptWithScope()
//
// 		var cwsw DocumentWriter
// 		cwsw, err = vw.WriteCodeWithScope(code)
// 		if err != nil {
// 			break
// 		}
//
// 		err = dve.encodeDocument(ec, cwsw, scope)
// 	case bson.TypeInt32:
// 		err = vw.WriteInt32(val.Int32())
// 	case bson.TypeTimestamp:
// 		err = vw.WriteTimestamp(val.Timestamp())
// 	case bson.TypeInt64:
// 		err = vw.WriteInt64(val.Int64())
// 	case bson.TypeDecimal128:
// 		err = vw.WriteDecimal128(val.Decimal128())
// 	case bson.TypeMinKey:
// 		err = vw.WriteMinKey()
// 	case bson.TypeMaxKey:
// 		err = vw.WriteMaxKey()
// 	default:
// 		err = fmt.Errorf("%T is not a valid BSON type to encode", val.Type())
// 	}
//
// 	return err
// }
