package bsoncodec

import "github.com/mongodb/mongo-go-driver/bson"

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
		bytesFromDoc(bson.NewDocument(bson.EC.Boolean("foo", true))),
	},
}
