// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package bson is a library for reading, writing, and manipulating BSON. The
// library has two families of types for representing BSON.
//
// The Raw family of types is used to validate and retrieve elements from a slice of bytes. This
// type is most useful when you want do lookups on BSON bytes without unmarshaling it into another
// type.
//
// Example:
// 		var raw bson.Raw = ... // bytes from somewhere
// 		err := raw.Validate()
// 		if err != nil { return err }
// 		val := raw.Lookup("foo")
// 		i32, ok := val.Int32OK()
// 		// do something with i32...
//
// The D family of types is used to build concise representations of BSON using native Go types.
// These types do not support automatic lookup.
//
// Example:
// 		bson.D{{"foo", "bar"}, {"hello", "world"}, {"pi", 3.14159}}
//
//
// Marshaling and Unmarshaling are handled with the Marshal and Unmarshal family of functions. If
// you need to write or read BSON from a non-slice source, an Encoder or Decoder can be used with a
// bsonrw.ValueWriter or bsonrw.ValueReader.
//
// Example:
// 		b, err := bson.Marshal(bson.D{{"foo", "bar"}})
// 		if err != nil { return err }
// 		var fooer struct {
// 			Foo string
// 		}
// 		err = bson.Unmarshal(b, &fooer)
// 		if err != nil { return err }
// 		// do something with fooer...
//
// The default encoding/decoding mappings for Go to BSON types are:
//
//     1. []byte encodes to a BSON binary with subtype 0. See DefaultValueEncoders.ByteSliceEncodeValue in bsoncodec for
//     more information.
//
//     2. time.Time encodes to a BSON datetime. This can be decoded from a BSON datetime, string (in RFC-3339 format),
//     int64, or timestamp. See TimeCodec in bsoncodec for more information.
//
//     3. interface{} encodes to any BSON type, depending on the underlying concrete type. When decoding, interface{} will
//     default to bson.D. See DefaultValueEncoders.EmptyInterfaceEncodeValue in bsoncodec for more information.
//
//     4. primitive.ObjectID encodes to a BSON ObjectId. See DefaultValueEncoders.ObjectIDEncodeValue in bsoncodec for more
//     information.
//
//     5. primitive.Decimal128 encodes to a BSON decimal128. See DefaultValueEncoders.Decimal128EncodeValue in bsoncodec for
//     more information.
//
//     6. json.Number encodes to a BSON int64 or double depending on the number. See
//     DefaultValueEncoders.JSONNumberEncodeValue in bsoncodec for more information.
//
//     7. url.URL encodes to a BSON string. See DefaultValueEncoders.URLEncodeValue in bsoncodec for more information.
//
//     8. primitive.JavaScript encodes to a BSON JavaScript code value. See DefaultValueEncoders.JavaScriptEncodeValue
//     in bsoncodec for more information.
//
//     9. primitive.CodeWithScope encodes to a BSON code with scope value. See
//     DefaultValueEncoders.CodeWWithScopeEncodeValue in bsoncodec for more information.
//
//     10. primitive.Symbol encodes to a BSON symbol value. See DefaultValueEncoders.SymbolEncodeValue in bsoncodec for more
//     information.
//
//     11. primitive.Binary encodes to a BSON binary. See DefaultValueEncoders.BinaryEncodeValue in bsoncodec for more
//     information.
//
//     12. primitive.Undefined encodes to a BSON undefined value. See DefaultValueEncoders.UndefinedEncodeValue in bsoncodec
//     for more information.
//
//     13. primitive.DateTime encodes to BSON datetime. See DefaultValueEncoders.DateTimeEncodeValue in bsoncodec for more
//     information.
//
//     14. primitive.Null encodes to a BSON null value. See DefaultValueEncoders.NullEncodeValue in bsoncodec for more
//     information.
//
//     15. primitive.Regex encodes to a BSON regex. See DefaultValueEncoders.RegexEncodeValue in bsoncodec for more
//     information.
//
//     16. primitive.DBPointer encodes to a BSON DBPointer. See DefaultValueEncoders.DBPointerEncodeValue in bsoncodec for
//     more information.
//
//     17. primitive.Timestamp encodes to a BSON timestamp. See DefaultValueEncoders.TimestampEncodeValue in bsoncodec for
//     more information.
//
//     18. primitive.MinKey encodes to a BSON min key value. See DefaultValueEncoders.MinKeyEncodeValue in bsoncodec for
//     more information.
//
//     19. primitive.MaxKey encodes to a BSON max key value. See DefaultValueEncoders.MaxKeyEncodeValue in bsoncodec for
//     more information.
//
//     20. bson.Raw encodes to a BSON document. See PrimitiveCodecs.RawEncodeValue for more information.
//
//     21. bson.D encodes to a BSON document. See DefaultValueEncoders.SliceEncodeValue in bsoncodec for more information.
//
//     22. Maps, such as bson.M, encode to a BSON document. Maps must have string keys. See
//     DefaultValueEncoders.MapEncodeValue in bsoncodec for more information.
//
//     23. int8, int16, and int32 encode to a BSON int32. int encodes to a BSON int32 if it's between math.MinInt32 and
//     math.MaxInt32, inclusive, and a BSON int64 otherwise. int64 encodes to a BSON int64. See
//     DefaultValueEncoders.IntEncodeValue in bsoncodec for more information.
//
//     26. bool encodes to a BSON boolean. See DefaultValueEncoders.BooleanEncodeValue in bsoncodec for more information.
//
//     27. uint8 and uint16 encode to a BSON int32. uint, uint32, and uint64 encode to a BSON int64. See
//     DefaultValueEncoders.UintEncodeValue in bsoncodec for more information.
//
//     28. float32 and float64 encode to a BSON double. See DefaultValueEncoders.FloatEncodeValue in bsoncodec for more
//     information.
//
//     29. Arrays and slices encode to BSON arrays. Arrays and slices of primitive.E are treated as a primitive.D and encode
//     to a BSON subdocument. See DefaultValueEncoders.ArrayEncodeValue in bsoncodec for more information.
//
//     30. string encodes to a BSON string. Strings can be decoded from a BSON string, ObjectId, or symbol. See
//     StringCodec in bsoncodec for more information.
//
//     31. Structs encode to BSON subdocuments. See StructCodec in bsoncodec for more information.
package bson
