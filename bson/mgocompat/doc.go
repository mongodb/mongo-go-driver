// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package mgocompat provides Registry, a BSON registry compatible with globalsign/mgo's BSON,
// with some remaining differences. It also provides RegistryRespectNilValues for compatibility
// with mgo's BSON with RespectNilValues set to true. A registry can be configured on a
// mongo.Client with the SetRegistry option. See the bson docs for more details on registries.
//
// Registry supports Getter and Setter equivalents by registering hooks. Note that if a value
// matches the hook for bson.Marshaler, bson.ValueMarshaler, or bson.Proxy, that
// hook will take priority over the Getter hook. The same is true for the hooks for
// bson.Unmarshaler and bson.ValueUnmarshaler and the Setter hook.
//
// The functional differences between Registry and globalsign/mgo's BSON library are:
//
// 1) Registry errors instead of silently skipping mismatched types when decoding.
//
// 2) Registry does not have special handling for marshaling array ops ("$in", "$nin", "$all").
//
// The driver uses different types than mgo's bson. The differences are:
//
//  1. The driver's bson.RawValue is equivalent to mgo's bson.Raw, but uses Value instead of Data and uses Type,
//     which is a bsontype.Type object that wraps a byte, instead of bson.Raw's Kind, a byte.
//
//  2. The driver uses bson.ObjectID, which is a [12]byte instead of mgo's
//     bson.ObjectId, a string. Due to this, the zero value marshals and unmarshals differently
//     for Extended JSON, with the driver marshaling as {"ID":"000000000000000000000000"} and
//     mgo as {"Id":""}. The driver can unmarshal {"ID":""} to a bson.ObjectID.
//
//  3. The driver's bson.Symbol is equivalent to mgo's bson.Symbol.
//
//  4. The driver uses bson.Timestamp instead of mgo's bson.MongoTimestamp. While
//     MongoTimestamp is an int64, bson.Timestamp stores the time and counter as two separate
//     uint32 values, T and I respectively.
//
//  5. The driver uses bson.MinKey and bson.MaxKey, which are struct{}, instead
//     of mgo's bson.MinKey and bson.MaxKey, which are int64.
//
//  6. The driver's bson.Undefined is equivalent to mgo's bson.Undefined.
//
//  7. The driver's bson.Binary is equivalent to mgo's bson.Binary, with variables named Subtype
//     and Data instead of Kind and Data.
//
//  8. The driver's bson.Regex is equivalent to mgo's bson.RegEx.
//
//  9. The driver's bson.JavaScript is equivalent to mgo's bson.JavaScript with no
//     scope and bson.CodeWithScope is equivalent to mgo's bson.JavaScript with scope.
//
//  10. The driver's bson.DBPointer is equivalent to mgo's bson.DBPointer, with variables
//     named DB and Pointer instead of Namespace and Id.
//
//  11. When implementing the Setter interface, mgocompat.ErrSetZero is equivalent to mgo's
//     bson.ErrSetZero.
package mgocompat
