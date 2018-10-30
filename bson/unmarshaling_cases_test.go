package bson

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
)

type unmarshalingTestCase struct {
	name  string
	reg   *bsoncodec.Registry
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
		docToBytes(NewDocumentv2(EC("foo", Boolean(true)))),
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
		docToBytes(NewDocumentv2(EC("foo", EmbedElement("bar", Boolean(true))))),
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
		docToBytes(NewDocumentv2(EC("foo", EmbedValues(Boolean(true))))),
	},
}
