// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	testhelpers "go.mongodb.org/mongo-driver/internal/testutil/helpers"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// newMissingArgumentError creates an error to convey that an argument that is required to run an operation is missing
// from the operation's arguments document.
func newMissingArgumentError(arg string) error {
	return fmt.Errorf("operation arguments document is missing required field %q", arg)
}

type updateArguments struct {
	filter bson.Raw
	update interface{}
	opts   *options.UpdateOptions
}

func createUpdateArguments(args bson.Raw) (*updateArguments, error) {
	ua := &updateArguments{
		opts: options.Update(),
	}
	var err error

	elems, _ := args.Elements()
	for _, elem := range elems {
		key := elem.Key()
		val := elem.Value()

		switch key {
		case "arrayFilters":
			ua.opts.SetArrayFilters(options.ArrayFilters{
				Filters: testhelpers.RawToInterfaceSlice(val.Array()),
			})
		case "bypassDocumentValidation":
			ua.opts.SetBypassDocumentValidation(val.Boolean())
		case "collation":
			collation, err := createCollation(val.Document())
			if err != nil {
				return nil, fmt.Errorf("error creating collation: %v", err)
			}
			ua.opts.SetCollation(collation)
		case "comment":
			ua.opts.SetComment(val)
		case "filter":
			ua.filter = val.Document()
		case "hint":
			hint, err := createHint(val)
			if err != nil {
				return nil, fmt.Errorf("error creating hint: %v", err)
			}
			ua.opts.SetHint(hint)
		case "let":
			ua.opts.SetLet(val.Document())
		case "update":
			ua.update, err = createUpdateValue(val)
			if err != nil {
				return nil, fmt.Errorf("error processing update value: %v", err)
			}
		case "upsert":
			ua.opts.SetUpsert(val.Boolean())
		default:
			return nil, fmt.Errorf("unrecognized update option %q", key)
		}
	}
	if ua.filter == nil {
		return nil, newMissingArgumentError("filter")
	}
	if ua.update == nil {
		return nil, newMissingArgumentError("update")
	}

	return ua, nil
}

type listCollectionsArguments struct {
	filter bson.Raw
	opts   *options.ListCollectionsOptions
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
	case bsontype.String:
		hint = val.StringValue()
	case bsontype.EmbeddedDocument:
		hint = val.Document()
	default:
		return nil, fmt.Errorf("unrecognized hint value type %s", val.Type)
	}
	return hint, nil
}

func createCommentString(val bson.RawValue) (string, error) {
	switch val.Type {
	case bsontype.String:
		return val.StringValue(), nil
	case bsontype.EmbeddedDocument:
		return val.String(), nil
	default:
		return "", fmt.Errorf("unrecognized 'comment' value type: %T", val)
	}
}
