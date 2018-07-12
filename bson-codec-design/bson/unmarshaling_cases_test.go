package bson

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
)

type unmarshalingTestCase struct {
	name   string
	reg    *Registry
	sType  reflect.Type
	want   interface{}
	reader io.Reader
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
		ioReaderFromDoc(NewDocument(EC.Boolean("foo", true))),
	},
}

func ioReaderFromDoc(doc *Document) io.Reader {
	b, err := doc.MarshalBSON()
	if err != nil {
		panic(fmt.Errorf("Couldn't marshal BSON document: %v", err))
	}
	return bytes.NewReader(b)
}
