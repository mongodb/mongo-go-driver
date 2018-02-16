// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson"
)

func TestTransformDocument(t *testing.T) {
	testCases := []struct {
		name     string
		document interface{}
		want     *bson.Document
		err      error
	}{
		{
			"bson.Marshaler",
			bMarsh{bson.NewDocument(bson.EC.String("foo", "bar"))},
			bson.NewDocument(bson.EC.String("foo", "bar")),
			nil,
		},
		{
			"bson.DocumentMarshaler",
			dMarsh{bson.NewDocument(bson.EC.String("foo", "bar"))},
			bson.NewDocument(bson.EC.String("foo", "bar")),
			nil,
		},
		{
			"reflection",
			reflectStruct{Foo: "bar"},
			bson.NewDocument(bson.EC.String("foo", "bar")),
			nil,
		},
		{
			"reflection pointer",
			&reflectStruct{Foo: "bar"},
			bson.NewDocument(bson.EC.String("foo", "bar")),
			nil,
		},
		{
			"unsupported type",
			[]string{"foo", "bar"},
			nil,
			fmt.Errorf("cannot transform type %s to a *bson.Document", reflect.TypeOf([]string{})),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := TransformDocument(tc.document)
			if !cmp.Equal(err, tc.err, cmp.Comparer(compareErrors)) {
				t.Errorf("Error does not match expected error. got %v; want %v", err, tc.err)
			}

			if diff := cmp.Diff(got, tc.want, cmp.AllowUnexported(bson.Document{}, bson.Element{}, bson.Value{})); diff != "" {
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
	*bson.Document
}

func (b bMarsh) MarshalBSON() ([]byte, error) {
	return b.Document.MarshalBSON()
}

var _ bson.DocumentMarshaler = dMarsh{}

type dMarsh struct {
	d *bson.Document
}

func (d dMarsh) MarshalBSONDocument() (*bson.Document, error) {
	return d.d, nil
}

type reflectStruct struct {
	Foo string
}
