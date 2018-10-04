package bsontype

import "testing"

func TestType(t *testing.T) {
	testCases := []struct {
		name string
		t    Type
		want string
	}{
		{"double", Double, "double"},
		{"string", String, "string"},
		{"embedded document", EmbeddedDocument, "embedded document"},
		{"array", Array, "array"},
		{"binary", Binary, "binary"},
		{"undefined", Undefined, "undefined"},
		{"objectID", ObjectID, "objectID"},
		{"boolean", Boolean, "boolean"},
		{"UTC datetime", DateTime, "UTC datetime"},
		{"null", Null, "null"},
		{"regex", Regex, "regex"},
		{"dbPointer", DBPointer, "dbPointer"},
		{"javascript", JavaScript, "javascript"},
		{"symbol", Symbol, "symbol"},
		{"code with scope", CodeWithScope, "code with scope"},
		{"32-bit integer", Int32, "32-bit integer"},
		{"timestamp", Timestamp, "timestamp"},
		{"64-bit integer", Int64, "64-bit integer"},
		{"128-bit decimal", Decimal128, "128-bit decimal"},
		{"min key", MinKey, "min key"},
		{"max key", MaxKey, "max key"},
		{"invalid", (0), "invalid"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.t.String()
			if got != tc.want {
				t.Errorf("String outputs do not match. got %s; want %s", got, tc.want)
			}
		})
	}
}
