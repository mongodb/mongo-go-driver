// Package mgocompat provides MgoRegistry, a bson registry compatible with mgo's bson, with some
// remaining differences. For compatibility with mgo's bson with RespectNilValues set to true,
// use MgoRegistryRespectNilValues. See the bsoncodec docs for more details on registries.
//
// The differences between mgocompat and mgo's bson are:
//
// 1) mgocompat errors instead of silently skipping mismatched types when decoding.
//
// 2) mgocompat does not have special handling for marshaling array ops ("$in", "$nin", "$all").
//
// 4) mgocompat will error if you try to unmarshal an empty raw.
//
// The driver uses slightly different types than mgo's bson. The differences are:
//
// 1) The driver's bson.RawValue is equivalent to mgo's bson.Raw, but uses Value instead
//    of Data and uses Type, which is a bsontype.Type object that wraps a byte, instead of
//    bson.Raw's Kind, a byte.
//
// 3) The driver uses primitive.ObjectID, which is a [12]byte instead of mgo's
//    bson.ObjectId, a string. Due to this, the nil value marshals and unmarshals differently
//    for Extended JSON.
//
// 4) The driver's primitive.Symbol is equivalent to mgo's bson.Symbol.
//
// 5) The driver uses primitive.Timestamp instead of bson.MongoTimestamp. While
//    MongoTimestamp is an int64, primitive.Timestamp stores the time and counter as two separate
//    uint32 values, T and I respectively.
//
// 6) The driver uses primitive.MinKey and primitive.MaxKey, which are struct{}, instead
//    of mgo's bson.MinKey and bson.MaxKey, which are int64.
//
// 7) The driver's primitive.Undefined is equivalent to mgo's bson.Undefined.
//
// 8) The driver's primitive.Binary is equivalent to bson.Binary, with variables named Subtype
//    and Data instead of Kind and Data.
//
// 9) The driver's primitive.Regex is equivalent to mgo's bson.RegEx.
//
// 10) The driver's primitive.JavaScript is equivalent to mgo's bson.JavaScript with no
//     scope and primitive.CodeWithScope is equivalent to mgo's bson.JavaScript with scope.
// 
// 11) The driver's primitive.DBPointer is equivalent to bson.DBPointer, with variables
//     named DB and Pointer instead of Namespace and Id.
//
// 12) When implementing the setter interface, mgocompat.ErrSetZero is equivalent to mgo's
//     bson.ErrSetZero.
//
// Things to be aware of:
//
// 1) If a value matches the hook for bsoncodec.Marshaler, bsoncodec.ValueMarshaler, or
//    bsoncodec.Proxy, that hook will take priority over the Getter hook. The same is true for the
//    hooks for bsoncodec.Unmarshaler and bsoncodec.ValueUnmarshaler and the Setter hook.
// 
