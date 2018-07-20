package llbson

import "testing"

func TestType(t *testing.T) {
	testCases := []struct {
		name string
		t    Type
		want string
	}{
		{"double", TypeDouble, "double"},
		{"string", TypeString, "string"},
		{"embedded document", TypeEmbeddedDocument, "embedded document"},
		{"array", TypeArray, "array"},
		{"binary", TypeBinary, "binary"},
		{"undefined", TypeUndefined, "undefined"},
		{"objectID", TypeObjectID, "objectID"},
		{"boolean", TypeBoolean, "boolean"},
		{"UTC datetime", TypeDateTime, "UTC datetime"},
		{"null", TypeNull, "null"},
		{"regex", TypeRegex, "regex"},
		{"dbPointer", TypeDBPointer, "dbPointer"},
		{"javascript", TypeJavaScript, "javascript"},
		{"symbol", TypeSymbol, "symbol"},
		{"code with scope", TypeCodeWithScope, "code with scope"},
		{"32-bit integer", TypeInt32, "32-bit integer"},
		{"timestamp", TypeTimestamp, "timestamp"},
		{"64-bit integer", TypeInt64, "64-bit integer"},
		{"128-bit decimal", TypeDecimal128, "128-bit decimal"},
		{"min key", TypeMinKey, "min key"},
		{"max key", TypeMaxKey, "max key"},
		{"invalid", Type(0), "invalid"},
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
