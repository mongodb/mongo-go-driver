package bsoncodec

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/bson/bsoncore"
)

type unmarshalingTestCase struct {
	name  string
	reg   *Registry
	sType reflect.Type
	want  interface{}
	data  []byte
}

var unmarshalingTestCases = []unmarshalingTestCase{
	{
		"small struct",
		nil,
		reflect.TypeOf(struct {
			Foo bool
		}{}),
		&struct {
			Foo bool
		}{Foo: true},
		buildDocument(func(doc []byte) []byte { return bsoncore.AppendBooleanElement(doc, "foo", true) }(nil)),
	},
	{
		"nested document",
		nil,
		reflect.TypeOf(struct {
			Foo struct {
				Bar bool
			}
		}{}),
		&struct {
			Foo struct {
				Bar bool
			}
		}{
			Foo: struct {
				Bar bool
			}{Bar: true},
		},
		buildDocument(func(doc []byte) []byte {
			return buildDocumentElement("foo", bsoncore.AppendBooleanElement(nil, "bar", true))
		}(nil)),
	},
	{
		"simple array",
		nil,
		reflect.TypeOf(struct {
			Foo []bool
		}{}),
		&struct {
			Foo []bool
		}{
			Foo: []bool{true},
		},
		buildDocument(func(doc []byte) []byte {
			return appendArrayElement(doc, "foo", bsoncore.AppendBooleanElement(nil, "0", true))
		}(nil)),
	},
}
