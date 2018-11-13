// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
)

func TestTransformDocument(t *testing.T) {
	testCases := []struct {
		name     string
		document interface{}
		want     bsonx.Doc
		err      error
	}{
		{
			"bson.Marshaler",
			bMarsh{bsonx.Doc{{"foo", bsonx.String("bar")}}},
			bsonx.Doc{{"foo", bsonx.String("bar")}},
			nil,
		},
		{
			"reflection",
			reflectStruct{Foo: "bar"},
			bsonx.Doc{{"foo", bsonx.String("bar")}},
			nil,
		},
		{
			"reflection pointer",
			&reflectStruct{Foo: "bar"},
			bsonx.Doc{{"foo", bsonx.String("bar")}},
			nil,
		},
		{
			"unsupported type",
			[]string{"foo", "bar"},
			nil,
			MarshalError{Value: []string{"foo", "bar"}, Err: errors.New("invalid state transition: TopLevel -> ArrayMode")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := transformDocument(bson.NewRegistryBuilder().Build(), tc.document)
			if !cmp.Equal(err, tc.err, cmp.Comparer(compareErrors)) {
				t.Errorf("Error does not match expected error. got %v; want %v", err, tc.err)
			}

			if diff := cmp.Diff(got, tc.want, cmp.AllowUnexported(bsonx.Elem{}, bsonx.Val{})); diff != "" {
				t.Errorf("Returned documents differ: (-got +want)\n%s", diff)
			}
		})
	}
}

func compareErrors(err1, err2 error) bool {
	if err1 == nil && err2 == nil {
		return true
	}

	if err1 == nil || err2 == nil {
		return false
	}

	if err1.Error() != err2.Error() {
		return false
	}

	return true
}

var _ bson.Marshaler = bMarsh{}

type bMarsh struct {
	bsonx.Doc
}

func (b bMarsh) MarshalBSON() ([]byte, error) {
	return b.Doc.MarshalBSON()
}

type reflectStruct struct {
	Foo string
}
