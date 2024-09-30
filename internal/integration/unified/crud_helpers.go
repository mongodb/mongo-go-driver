// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"fmt"
	"reflect"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/bsonutil"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// newMissingArgumentError creates an error to convey that an argument that is required to run an operation is missing
// from the operation's arguments document.
func newMissingArgumentError(arg string) error {
	return fmt.Errorf("operation arguments document is missing required field %q", arg)
}

type updateArguments[Options options.UpdateManyOptions | options.UpdateOneOptions] struct {
	filter bson.Raw
	update interface{}
	opts   options.Lister[Options]
}

func createUpdateArguments[Options options.UpdateManyOptions | options.UpdateOneOptions](args bson.Raw) (*updateArguments[Options], error) {
	ua := &updateArguments[Options]{}
	var builder reflect.Value
	switch any((*Options)(nil)).(type) {
	case *options.UpdateManyOptions:
		builder = reflect.ValueOf(options.UpdateMany())
	case *options.UpdateOneOptions:
		builder = reflect.ValueOf(options.UpdateOne())
	}

	elems, _ := args.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		var arg reflect.Value
		switch key {
		case "arrayFilters":
			arg = reflect.ValueOf(
				bsonutil.RawToInterfaces(bsonutil.RawArrayToDocuments(val.Array())...),
			)
		case "bypassDocumentValidation":
			arg = reflect.ValueOf(val.Boolean())
		case "collation":
			collation, err := createCollation(val.Document())
			if err != nil {
				return nil, fmt.Errorf("error creating collation: %w", err)
			}
			arg = reflect.ValueOf(collation)
		case "comment":
			arg = reflect.ValueOf(val)
		case "filter":
			ua.filter = val.Document()
		case "hint":
			hint, err := createHint(val)
			if err != nil {
				return nil, fmt.Errorf("error creating hint: %w", err)
			}
			arg = reflect.ValueOf(hint)
		case "let", "sort":
			arg = reflect.ValueOf(val.Document())
		case "update":
			var err error
			ua.update, err = createUpdateValue(val)
			if err != nil {
				return nil, fmt.Errorf("error processing update value: %w", err)
			}
		case "upsert":
			arg = reflect.ValueOf(val.Boolean())
		default:
			return nil, fmt.Errorf("unrecognized update option %q", key)
		}
		if arg.IsValid() {
			fn := builder.MethodByName(
				fmt.Sprintf("Set%s%s", strings.ToUpper(string(key[0])), key[1:]),
			)
			if !fn.IsValid() {
				return nil, fmt.Errorf("unrecognized update option %q", key)
			}
			fn.Call([]reflect.Value{arg})
		}
	}
	if ua.filter == nil {
		return nil, newMissingArgumentError("filter")
	}
	if ua.update == nil {
		return nil, newMissingArgumentError("update")
	}
	ua.opts = builder.Interface().(options.Lister[Options])

	return ua, nil
}

type listCollectionsArguments struct {
	filter bson.Raw
	opts   *options.ListCollectionsOptionsBuilder
}

func createListCollectionsArguments(args bson.Raw) (*listCollectionsArguments, error) {
	lca := &listCollectionsArguments{
		opts: options.ListCollections(),
	}

	lca.filter = emptyDocument
	elems, _ := args.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "batchSize":
			lca.opts.SetBatchSize(val.Int32())
		case "filter":
			lca.filter = val.Document()
		case "nameOnly":
			lca.opts.SetNameOnly(val.Boolean())
		default:
			return nil, fmt.Errorf("unrecognized listCollections option %q", key)
		}
	}

	return lca, nil
}

func createCollation(args bson.Raw) (*options.Collation, error) {
	var collation options.Collation
	elems, _ := args.Elements()

	for _, elem := range elems {
		switch elem.Key() {
		case "locale":
			collation.Locale = elem.Value().StringValue()
		case "caseLevel":
			collation.CaseLevel = elem.Value().Boolean()
		case "caseFirst":
			collation.CaseFirst = elem.Value().StringValue()
		case "strength":
			collation.Strength = int(elem.Value().Int32())
		case "numericOrdering":
			collation.NumericOrdering = elem.Value().Boolean()
		case "alternate":
			collation.Alternate = elem.Value().StringValue()
		case "maxVariable":
			collation.MaxVariable = elem.Value().StringValue()
		case "normalization":
			collation.Normalization = elem.Value().Boolean()
		case "backwards":
			collation.Backwards = elem.Value().Boolean()
		default:
			return nil, fmt.Errorf("unrecognized collation option %q", elem.Key())
		}
	}
	return &collation, nil
}

func createHint(val bson.RawValue) (interface{}, error) {
	var hint interface{}

	switch val.Type {
	case bson.TypeString:
		hint = val.StringValue()
	case bson.TypeEmbeddedDocument:
		hint = val.Document()
	default:
		return nil, fmt.Errorf("unrecognized hint value type %s", val.Type)
	}
	return hint, nil
}

func createSort(val bson.RawValue) (interface{}, error) {
	var sort interface{}

	switch val.Type {
	case bson.TypeEmbeddedDocument:
		sort = val.Document()
	default:
		return nil, fmt.Errorf("unrecognized sort value type %s", val.Type)
	}
	return sort, nil
}
