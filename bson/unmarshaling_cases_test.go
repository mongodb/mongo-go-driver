package bson

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
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
		bytesFromDoc(NewDocument(EC.Boolean("foo", true))),
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
		bytesFromDoc(NewDocument(EC.SubDocumentFromElements("foo", EC.Boolean("bar", true)))),
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
		bytesFromDoc(NewDocument(EC.ArrayFromElements("foo", VC.Boolean(true)))),
	},
}

func ioReaderFromDoc(doc *Document) io.Reader {
	b, err := doc.MarshalBSON()
	if err != nil {
		panic(fmt.Errorf("Couldn't marshal BSON document: %v", err))
	}
	return bytes.NewReader(b)
}
