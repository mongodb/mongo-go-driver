// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"sort"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

var (
	emptyCoreDocument = bsoncore.NewDocumentBuilder().Build()
	emptyDocument     = bson.Raw(emptyCoreDocument)
	emptyRawValue     = bson.RawValue{}
)

func documentToRawValue(doc bson.Raw) bson.RawValue {
	return bson.RawValue{
		Type:  bsontype.EmbeddedDocument,
		Value: doc,
	}
}

func removeFieldsFromDocument(doc bson.Raw, keys ...string) bson.Raw {
	newDoc := bsoncore.NewDocumentBuilder()
	elems, _ := doc.Elements()

	keysMap := make(map[string]struct{})
	for _, key := range keys {
		keysMap[key] = struct{}{}
	}

	for _, elem := range elems {
		if _, ok := keysMap[elem.Key()]; ok {
			continue
		}

		val := elem.Value()
		newDoc.AppendValue(elem.Key(), bsoncore.Value{Type: val.Type, Data: val.Value})
	}
	return bson.Raw(newDoc.Build())
}

func sortDocument(doc bson.Raw) bson.Raw {
	elems, _ := doc.Elements()
	keys := make([]string, 0, len(elems))
	valuesMap := make(map[string]bson.RawValue)

	for _, elem := range elems {
		keys = append(keys, elem.Key())
		valuesMap[elem.Key()] = elem.Value()
	}

	sort.Strings(keys)
	sorted := bsoncore.NewDocumentBuilder()
	for _, key := range keys {
		val := valuesMap[key]
		sorted.AppendValue(key, bsoncore.Value{Type: val.Type, Data: val.Value})
	}
	return bson.Raw(sorted.Build())
}

func lookupString(doc bson.Raw, key string) string {
	return doc.Lookup(key).StringValue()
}

func lookupInteger(doc bson.Raw, key string) int64 {
	return doc.Lookup(key).AsInt64()
}

func mapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
