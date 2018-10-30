package bson

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
)

func TestValue(t *testing.T) {
	longstr := "foobarbazqux, hello, world!"
	bytestr15 := "fifteen bytes!!"
	t.Parallel()
	testCases := []struct {
		name string
		fn   interface{} // method to call
		ret  interface{} // return value
		err  interface{} // panic result or bool
	}{
		{"Interface/Double", Double(3.14159).Interface, float64(3.14159), nil},
		{"Interface/String", String("foo").Interface, string("foo"), nil},
		{"Interface/Document", Embed(NewDocument()).Interface, NewDocument(), nil},
		{"Interface/Array", Embed(NewArray()).Interface, NewArray(), nil},
		{"IsNumber/Double", Double(0).IsNumber, bool(true), nil},
		{"IsNumber/String", String("").IsNumber, bool(false), nil},
		{"Double/panic", String("").Double, nil, ElementTypeError{"bson.Value.Double", bsontype.String}},
		{"Double/success", Double(3.14159).Double, float64(3.14159), nil},
		{"DoubleOK/error", String("").DoubleOK, float64(0), false},
		{"DoubleOK/success", Double(3.14159).DoubleOK, float64(3.14159), true},
		{"String/panic", Double(0).StringValue, nil, ElementTypeError{"bson.Value.StringValue", bsontype.Double}},
		{"String/success", String("bar").StringValue, string("bar"), nil},
		{"String/15bytes", String(bytestr15).StringValue, string(bytestr15), nil},
		{"String/success(long)", String(longstr).StringValue, string(longstr), nil},
		{"StringOK/error", Double(0).StringValueOK, string(""), false},
		{"StringOK/success", String("bar").StringValueOK, string("bar"), true},
		{"Document/panic", Double(0).Document, nil, ElementTypeError{"bson.Value.Document", bsontype.Double}},
		{"Document/success", Embed(NewDocument()).Document, NewDocument(), nil},
		{"DocumentOK/error", Double(0).DocumentOK, (*Document)(nil), false},
		{"DocumentOK/success", Embed(NewDocument()).DocumentOK, NewDocument(), true},
		{"Array/panic", Double(0).Array, nil, ElementTypeError{"bson.Value.Array", bsontype.Double}},
		{"Array/success", Embed(NewArray()).Array, NewArray(), nil},
		{"ArrayOK/error", Double(0).ArrayOK, (*Array)(nil), false},
		{"ArrayOK/success", Embed(NewArray()).ArrayOK, NewArray(), true},
		{"Embed/NilDocument", Embed((*Document)(nil)).Interface, NullPrimitive{}, nil},
		{"Embed/NilArray", Embed((*Array)(nil)).Interface, NullPrimitive{}, nil},
		{"Embed/Nil", Embed(nil).Interface, NullPrimitive{}, nil},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				err := recover()
				if err != nil && !cmp.Equal(err, tc.err) {
					t.Errorf("panic errors are not equal. got %v; want %v", err, tc.err)
					if tc.err == nil {
						panic(err)
					}
				}
			}()
			fn := reflect.ValueOf(tc.fn)
			if fn.Kind() != reflect.Func {
				t.Fatalf("fn must be a function, but is a %s", fn.Kind())
			}
			ret := fn.Call(nil)
			switch len(ret) {
			case 2:
				got, want := ret[1].Interface(), tc.err
				if !cmp.Equal(got, want) {
					t.Errorf("error booleans should be equal. got %v; want %v", got, want)
				}
				fallthrough
			case 1:
				got, want := ret[0].Interface(), tc.ret
				if !cmp.Equal(got, want) {
					t.Errorf("return values should be equal. got %v; want %v", got, want)
				}
			default:
				t.Fatalf("fn should return one or two values, but returned %d values", len(ret))
			}
		})
	}
}
