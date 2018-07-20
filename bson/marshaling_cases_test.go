package bson

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
		bytesFromDoc(NewDocument(EC.Boolean("foo", true))),
	},
}
