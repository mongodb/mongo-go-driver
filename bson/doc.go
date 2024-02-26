// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package bson is a library for reading, writing, and manipulating BSON. BSON is a binary serialization format used to
// store documents and make remote procedure calls in MongoDB. The BSON specification is located at https://bsonspec.org.
// The BSON library handles marshaling and unmarshaling of values through a configurable codec system. For a description
// of the codec system and examples of registering custom codecs, see the bsoncodec package. For additional information
// and usage examples, check out the [Work with BSON] page in the Go Driver docs site.
//
// # Raw BSON
//
// The Raw family of types is used to validate and retrieve elements from a slice of bytes. This
// type is most useful when you want do lookups on BSON bytes without unmarshaling it into another
// type.
//
// Example:
//
//	var raw bson.Raw = ... // bytes from somewhere
//	err := raw.Validate()
//	if err != nil { return err }
//	val := raw.Lookup("foo")
//	i32, ok := val.Int32OK()
//	// do something with i32...
//
// # Native Go Types
//
// The D and M types defined in this package can be used to build representations of BSON using native Go types. D is a
// slice and M is a map. For more information about the use cases for these types, see the documentation on the type
// definitions.
//
// Note that a D should not be constructed with duplicate key names, as that can cause undefined server behavior.
//
// Example:
//
//	bson.D{{"foo", "bar"}, {"hello", "world"}, {"pi", 3.14159}}
//	bson.M{"foo": "bar", "hello": "world", "pi": 3.14159}
//
// When decoding BSON to a D or M, the following type mappings apply when unmarshaling:
//
//  1. BSON int32 unmarshals to an int32.
//  2. BSON int64 unmarshals to an int64.
//  3. BSON double unmarshals to a float64.
//  4. BSON string unmarshals to a string.
//  5. BSON boolean unmarshals to a bool.
//  6. BSON embedded document unmarshals to the parent type (i.e. D for a D, M for an M).
//  7. BSON array unmarshals to a bson.A.
//  8. BSON ObjectId unmarshals to a primitive.ObjectID.
//  9. BSON datetime unmarshals to a primitive.DateTime.
//  10. BSON binary unmarshals to a primitive.Binary.
//  11. BSON regular expression unmarshals to a primitive.Regex.
//  12. BSON JavaScript unmarshals to a primitive.JavaScript.
//  13. BSON code with scope unmarshals to a primitive.CodeWithScope.
//  14. BSON timestamp unmarshals to an primitive.Timestamp.
//  15. BSON 128-bit decimal unmarshals to an primitive.Decimal128.
//  16. BSON min key unmarshals to an primitive.MinKey.
//  17. BSON max key unmarshals to an primitive.MaxKey.
//  18. BSON undefined unmarshals to a primitive.Undefined.
//  19. BSON null unmarshals to nil.
//  20. BSON DBPointer unmarshals to a primitive.DBPointer.
//  21. BSON symbol unmarshals to a primitive.Symbol.
//
// The above mappings also apply when marshaling a D or M to BSON. Some other useful marshaling mappings are:
//
//  1. time.Time marshals to a BSON datetime.
//  2. int8, int16, and int32 marshal to a BSON int32.
//  3. int marshals to a BSON int32 if the value is between math.MinInt32 and math.MaxInt32, inclusive, and a BSON int64
//     otherwise.
//  4. int64 marshals to BSON int64 (unless [Encoder.IntMinSize] is set).
//  5. uint8 and uint16 marshal to a BSON int32.
//  6. uint, uint32, and uint64 marshal to a BSON int64 (unless [Encoder.IntMinSize] is set).
//  7. BSON null and undefined values will unmarshal into the zero value of a field (e.g. unmarshaling a BSON null or
//     undefined value into a string will yield the empty string.).
//
// # Structs
//
// Structs can be marshaled/unmarshaled to/from BSON or Extended JSON. When transforming structs to/from BSON or Extended
// JSON, the following rules apply:
//
//  1. Only exported fields in structs will be marshaled or unmarshaled.
//
//  2. When marshaling a struct, each field will be lowercased to generate the key for the corresponding BSON element.
//     For example, a struct field named "Foo" will generate key "foo". This can be overridden via a struct tag (e.g.
//     `bson:"fooField"` to generate key "fooField" instead).
//
//  3. An embedded struct field is marshaled as a subdocument. The key will be the lowercased name of the field's type.
//
//  4. A pointer field is marshaled as the underlying type if the pointer is non-nil. If the pointer is nil, it is
//     marshaled as a BSON null value.
//
//  5. When unmarshaling, a field of type interface{} will follow the D/M type mappings listed above. BSON documents
//     unmarshaled into an interface{} field will be unmarshaled as a D.
//
// The encoding of each struct field can be customized by the "bson" struct tag.
//
// This tag behavior is configurable, and different struct tag behavior can be configured by initializing a new
// bsoncodec.StructCodec with the desired tag parser and registering that StructCodec onto the Registry. By default, JSON
// tags are not honored, but that can be enabled by creating a StructCodec with JSONFallbackStructTagParser, like below:
//
// Example:
//
//	structcodec, _ := bsoncodec.NewStructCodec(bsoncodec.JSONFallbackStructTagParser)
//
// The bson tag gives the name of the field, possibly followed by a comma-separated list of options.
// The name may be empty in order to specify options without overriding the default field name. The following options can
// be used to configure behavior:
//
//  1. omitempty: If the omitempty struct tag is specified on a field, the field will be omitted from the marshaling if
//     the field has an empty value, defined as false, 0, a nil pointer, a nil interface value, and any empty array,
//     slice, map, or string.
//     NOTE: It is recommended that this tag be used for all slice and map fields.
//
//  2. minsize: If the minsize struct tag is specified on a field of type int64, uint, uint32, or uint64 and the value of
//     the field can fit in a signed int32, the field will be serialized as a BSON int32 rather than a BSON int64. For
//     other types, this tag is ignored.
//
//  3. truncate: If the truncate struct tag is specified on a field with a non-float numeric type, BSON doubles
//     unmarshaled into that field will be truncated at the decimal point. For example, if 3.14 is unmarshaled into a
//     field of type int, it will be unmarshaled as 3. If this tag is not specified, the decoder will throw an error if
//     the value cannot be decoded without losing precision. For float64 or non-numeric types, this tag is ignored.
//
//  4. inline: If the inline struct tag is specified for a struct or map field, the field will be "flattened" when
//     marshaling and "un-flattened" when unmarshaling. This means that all of the fields in that struct/map will be
//     pulled up one level and will become top-level fields rather than being fields in a nested document. For example,
//     if a map field named "Map" with value map[string]interface{}{"foo": "bar"} is inlined, the resulting document will
//     be {"foo": "bar"} instead of {"map": {"foo": "bar"}}. There can only be one inlined map field in a struct. If
//     there are duplicated fields in the resulting document when an inlined struct is marshaled, the inlined field will
//     be overwritten. If there are duplicated fields in the resulting document when an inlined map is marshaled, an
//     error will be returned. This tag can be used with fields that are pointers to structs. If an inlined pointer field
//     is nil, it will not be marshaled. For fields that are not maps or structs, this tag is ignored.
//
// # Marshaling and Unmarshaling
//
// Manually marshaling and unmarshaling can be done with the Marshal and Unmarshal family of functions.
//
// ---
//
// bsoncodec code provides a system for encoding values to BSON representations and decoding
// values from BSON representations. This package considers both binary BSON and ExtendedJSON as
// BSON representations. The types in this package enable a flexible system for handling this
// encoding and decoding.
//
// The codec system is composed of two parts:
//
// 1) ValueEncoders and ValueDecoders that handle encoding and decoding Go values to and from BSON
// representations.
//
// 2) A Registry that holds these ValueEncoders and ValueDecoders and provides methods for
// retrieving them.
//
// # ValueEncoders and ValueDecoders
//
// The ValueEncoder interface is implemented by types that can encode a provided Go type to BSON.
// The value to encode is provided as a reflect.Value and a bsonrw.ValueWriter is used within the
// EncodeValue method to actually create the BSON representation. For convenience, ValueEncoderFunc
// is provided to allow use of a function with the correct signature as a ValueEncoder. An
// EncodeContext instance is provided to allow implementations to lookup further ValueEncoders and
// to provide configuration information.
//
// The ValueDecoder interface is the inverse of the ValueEncoder. Implementations should ensure that
// the value they receive is settable. Similar to ValueEncoderFunc, ValueDecoderFunc is provided to
// allow the use of a function with the correct signature as a ValueDecoder. A DecodeContext
// instance is provided and serves similar functionality to the EncodeContext.
//
// # Registry
//
// A Registry is a store for ValueEncoders, ValueDecoders, and a type map. See the Registry type
// documentation for examples of registering various custom encoders and decoders. A Registry can
// have three main types of codecs:
//
// 1. Type encoders/decoders - These can be registered using the RegisterTypeEncoder and
// RegisterTypeDecoder methods. The registered codec will be invoked when encoding/decoding a value
// whose type matches the registered type exactly.
// If the registered type is an interface, the codec will be invoked when encoding or decoding
// values whose type is the interface, but not for values with concrete types that implement the
// interface.
//
// 2. Hook encoders/decoders - These can be registered using the RegisterHookEncoder and
// RegisterHookDecoder methods. These methods only accept interface types and the registered codecs
// will be invoked when encoding or decoding values whose types implement the interface. An example
// of a hook defined by the driver is bson.Marshaler. The driver will call the MarshalBSON method
// for any value whose type implements bson.Marshaler, regardless of the value's concrete type.
//
// 3. Type map entries - This can be used to associate a BSON type with a Go type. These type
// associations are used when decoding into a bson.D/bson.M or a struct field of type interface{}.
// For example, by default, BSON int32 and int64 values decode as Go int32 and int64 instances,
// respectively, when decoding into a bson.D. The following code would change the behavior so these
// values decode as Go int instances instead:
//
//	intType := reflect.TypeOf(int(0))
//	registry.RegisterTypeMapEntry(bson.TypeInt32, intType).RegisterTypeMapEntry(bson.TypeInt64, intType)
//
// 4. Kind encoder/decoders - These can be registered using the RegisterDefaultEncoder and
// RegisterDefaultDecoder methods. The registered codec will be invoked when encoding or decoding
// values whose reflect.Kind matches the registered reflect.Kind as long as the value's type doesn't
// match a registered type or hook encoder/decoder first. These methods should be used to change the
// behavior for all values for a specific kind.
//
// # Registry Lookup Procedure
//
// When looking up an encoder in a Registry, the precedence rules are as follows:
//
// 1. A type encoder registered for the exact type of the value.
//
// 2. A hook encoder registered for an interface that is implemented by the value or by a pointer to
// the value. If the value matches multiple hooks (e.g. the type implements bsoncodec.Marshaler and
// bsoncodec.ValueMarshaler), the first one registered will be selected. Note that registries
// constructed using bson.NewRegistry have driver-defined hooks registered for the
// bsoncodec.Marshaler, bsoncodec.ValueMarshaler, and bsoncodec.Proxy interfaces, so those will take
// precedence over any new hooks.
//
// 3. A kind encoder registered for the value's kind.
//
// If all of these lookups fail to find an encoder, an error of type ErrNoEncoder is returned. The
// same precedence rules apply for decoders, with the exception that an error of type ErrNoDecoder
// will be returned if no decoder is found.
//
// # DefaultValueEncoders and DefaultValueDecoders
//
// The DefaultValueEncoders and DefaultValueDecoders types provide a full set of ValueEncoders and
// ValueDecoders for handling a wide range of Go types, including all of the types within the
// primitive package. To make registering these codecs easier, a helper method on each type is
// provided. For the DefaultValueEncoders type the method is called RegisterDefaultEncoders and for
// the DefaultValueDecoders type the method is called RegisterDefaultDecoders, this method also
// handles registering type map entries for each BSON type.
//
// [Work with BSON]: https://www.mongodb.com/docs/drivers/go/current/fundamentals/bson/
package bson
