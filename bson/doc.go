// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package bson is a library for reading, writing, and manipulating BSON.
//
// Raw BSON
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
// Native Go Types
//
// This D and M types defined in this package can be used to build concise representations of BSON using native Go
// types.
//
// Example:
// 		bson.D{{"foo", "bar"}, {"hello", "world"}, {"pi", 3.14159}}
//
// When decoding BSON to a D or M, the following type mappings apply when unmarshalling:
//
// 		1. BSON int32 unmarshals to an int32.
// 		2. BSON int64 unmarshals to an int64.
// 		3. BSON double unmarshals to a float64.
// 		4. BSON string unmarshals to a string.
// 		5. BSON boolean unmarshals to a bool.
// 		6. BSON embedded document unmarshals to the parent type (i.e. D for a D, M for an M).
// 		7. BSON array unmarshals to a bson.A.
// 		8. BSON ObjectId unmarshals to a primitive.ObjectID.
// 		9. BSON datetime unmarshals to a primitive.Datetime.
// 		10. BSON binary unmarshals to a primitive.Binary.
// 		11. BSON regular expression unmarshals to a primitive.Regex.
// 		12. BSON JavaScript unmarshals to a primitive.JavaScript.
// 		13. BSON code with scope unmarshals to a primitive.CodeWithScope.
// 		14. BSON timestamp unmarshals to an primitive.Timestamp.
// 		15. BSON 128-bit decimal unmarshals to an primitive.Decimal128.
// 		16. BSON min key unmarshals to an primitive.MinKey.
// 		17. BSON max key unmarshals to an primitive.MaxKey.
// 		18. BSON undefined unmarshals to a primitive.Undefined.
// 		19. BSON null unmarshals to a primitive.Null.
// 		20. BSON DBPointer unmarshals to a primitive.DBPointer.
// 		21. BSON symbol unmarshals to a primitive.Symbol.
//
// The above mappings also apply when marshalling a D or M to BSON. Some other useful marshalling mappings are:
//
//       1. time.Time marshals to a BSON datetime.
//       2. int8, int16, and int32 marshal to a BSON int32.
//       3. int marshals to a BSON int32 if the value is between math.MinInt32 and math.MaxInt32, inclusive, and a BSON int64
//       otherwise.
//       4. int64 marshals to BSON int64.
//       5. uint8 and uint16 marshal to a BSON int32.
//       6. uint, uint32, and uint64 marshal to a BSON int32 if the value is between math.MinInt32 and math.MaxInt32,
//       inclusive, and BSON int64 otherwise.
//
// Structs
//
// Structs can be marshalled/unmarshalled to/from BSON. When transforming structs to/from BSON, the following rules
// apply:
//
//     1. Only exported fields in structs will be marshalled or unmarshalled.
//
//     2. When marshalling a struct, each field will be lowercased to generate the key for the corresponding BSON element.
//     For example, a struct field named "Foo" will generate key "foo". This can be overriden via a struct tag (e.g.
//     `bson:"fooField"` to generate key "fooField" instead).
//
//     3. An embedded struct field is marshalled as a subdocument. The key will be the lowercased name of the field's type.
//
//     4. A pointer field is marshalled as the underlying type if the pointer is non-nil. If the pointer is nil, it is
//     marshalled as a BSON null value.
//
// The following struct tags can be used to configure behavior:
//
//     1. omitempty: If the omitempty struct tag is specified on a field, the field will not be marshalled if it is set to
//     the zero value. By default, omitempty is not relevant for struct fields, which will be serialized as an embedded
//     document.
//
//     2. minsize: If the minsize struct tag is specified on a field of type int64, uint, uint32, or uint64 and the value of
//     the field can fit in a signed int32, the field will be serialized as a BSON int32 rather than a BSON int64. For other
//     types, this tag is ignored.
//
//     3. truncate: If the truncate struct tag is specified on a field with a non-float64 numeric type, BSON doubles unmarshalled
//     into that field will be trucated at the decimal point. For example, if 3.14 is unmarshalled into a field of type int,
//     it will be unmarshalled as 3. If this tag is not specified, the decoder will throw an error if the value cannot be
//     decoded without losing precision. For float64 or non-numeric types, this tag is ignored.
//
//     4. inline: If the inline struct tag is specified for a struct or map field, the field will be "flattened" when
//     marshalling and "un-flattened" when unmarshalling. This means that all of the fields in that struct/map will be
//     pulled up one level and will become top-level fields rather than being fields in a nested document. For example, if a
//     map field named "Map" with value map[string]interface{}{"foo": "bar"} is inlined, the resulting document will be
//     {"foo": "bar"} instead of {"map": {"foo": "bar"}}. This tag can be used with fields that are pointers to structs. If
//     an inlined pointer field is nil, it will not be marshalled.
//
// Marshalling and Unmarshalling
//
// Manually marshallhing and unmarshalling can be done with the Marshal and Unmarshal family of functions. To read or
// write BSON from a non-slice source, an Encoder or Decoder with a bsonrw.ValueWriter or bsonrw.ValueReader must be
// used.
package bson
