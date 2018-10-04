package bsoncodec

import "github.com/mongodb/mongo-go-driver/bson/bsoncore"

type marshalingTestCase struct {
	name string
	reg  *Registry
	val  interface{}
	want []byte
}

var marshalingTestCases = []marshalingTestCase{
	{
		"small struct",
		nil,
		struct {
			Foo bool
		}{Foo: true},
		buildDocument(func(doc []byte) []byte {
			return bsoncore.AppendBooleanElement(doc, "foo", true)
		}(nil)),
	},
}
