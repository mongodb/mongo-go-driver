package bson

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

func TestValue(t *testing.T) {
	longstr := "foobarbazqux, hello, world!"
	bytestr15 := "fifteen bytes!!"
	bin := BinaryPrimitive{Subtype: 0xFF, Data: []byte{0x01, 0x02, 0x03}}
	oid := objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
	now := time.Now().Truncate(time.Millisecond)
	nowdt := DateTimePrimitive(now.Unix()*1e3 + int64(now.Nanosecond()/1e6))
	regex := RegexPrimitive{Pattern: "/foobarbaz/", Options: "abr"}
	dbptr := DBPointerPrimitive{DB: "foobar", Pointer: oid}
	js := JavaScriptCodePrimitive("var hello ='world';")
	symbol := SymbolPrimitive("foobarbaz")
	cws := CodeWithScopePrimitive{Code: js, Scope: NewDocument()}
	ts := TimestampPrimitive{I: 12345, T: 67890}
	d128 := decimal.NewDecimal128(12345, 67890)

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
		{"Interface/Binary", Binary(bin).Interface, bin, nil},
		{"Interface/Undefined", Undefined(UndefinedPrimitive{}).Interface, UndefinedPrimitive{}, nil},
		{"Interface/Null", Null(NullPrimitive{}).Interface, NullPrimitive{}, nil},
		{"Interface/ObjectID", ObjectID(oid).Interface, oid, nil},
		{"Interface/Boolean", Boolean(true).Interface, bool(true), nil},
		{"Interface/DateTime", DateTime(1234567890).Interface, DateTimePrimitive(1234567890), nil},
		{"Interface/Time", Time(now).Interface, nowdt, nil},
		{"Interface/Regex", Regex(regex).Interface, regex, nil},
		{"Interface/DBPointer", DBPointer(dbptr).Interface, dbptr, nil},
		{"Interface/JavaScript", JavaScript(js).Interface, js, nil},
		{"Interface/Symbol", Symbol(symbol).Interface, symbol, nil},
		{"Interface/CodeWithScope", CodeWithScope(cws).Interface, cws, nil},
		{"Interface/Int32", Int32(12345).Interface, int32(12345), nil},
		{"Interface/Timestamp", Timestamp(ts).Interface, ts, nil},
		{"Interface/Int64", Int64(1234567890).Interface, int64(1234567890), nil},
		{"Interface/Decimal128", Decimal128(d128).Interface, d128, nil},
		{"Interface/MinKey", MinKey(MinKeyPrimitive{}).Interface, MinKeyPrimitive{}, nil},
		{"Interface/MaxKey", MaxKey(MaxKeyPrimitive{}).Interface, MaxKeyPrimitive{}, nil},
		{"Interface/Invalid", Valuev2{}.Interface, nil, nil},
		{"IsNumber/Double", Double(0).IsNumber, bool(true), nil},
		{"IsNumber/Int32", Int32(0).IsNumber, bool(true), nil},
		{"IsNumber/Int64", Int64(0).IsNumber, bool(true), nil},
		{"IsNumber/Decimal128", Decimal128(decimal.Decimal128{}).IsNumber, bool(true), nil},
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
		{"Binary/panic", Double(0).Binary, nil, ElementTypeError{"bson.Value.Binary", bsontype.Double}},
		{"Binary/success", Binary(bin).Binary, bin, nil},
		{"BinaryOK/error", Double(0).BinaryOK, BinaryPrimitive{}, false},
		{"BinaryOK/success", Binary(bin).BinaryOK, bin, true},
		{"Undefined/panic", Double(0).Undefined, nil, ElementTypeError{"bson.Value.Undefined", bsontype.Double}},
		{"Undefined/success", Undefined(UndefinedPrimitive{}).Undefined, UndefinedPrimitive{}, nil},
		{"UndefinedOK/error", Double(0).UndefinedOK, UndefinedPrimitive{}, false},
		{"UndefinedOK/success", Undefined(UndefinedPrimitive{}).UndefinedOK, UndefinedPrimitive{}, true},
		{"ObjectID/panic", Double(0).ObjectID, nil, ElementTypeError{"bson.Value.ObjectID", bsontype.Double}},
		{"ObjectID/success", ObjectID(oid).ObjectID, oid, nil},
		{"ObjectIDOK/error", Double(0).ObjectIDOK, objectid.ObjectID{}, false},
		{"ObjectIDOK/success", ObjectID(oid).ObjectIDOK, oid, true},
		{"Boolean/panic", Double(0).Boolean, nil, ElementTypeError{"bson.Value.Boolean", bsontype.Double}},
		{"Boolean/success", Boolean(true).Boolean, bool(true), nil},
		{"BooleanOK/error", Double(0).BooleanOK, bool(false), false},
		{"BooleanOK/success", Boolean(false).BooleanOK, false, true},
		{"DateTime/panic", Double(0).DateTime, nil, ElementTypeError{"bson.Value.DateTime", bsontype.Double}},
		{"DateTime/success", DateTime(1234567890).DateTime, DateTimePrimitive(1234567890), nil},
		{"DateTimeOK/error", Double(0).DateTimeOK, DateTimePrimitive(0), false},
		{"DateTimeOK/success", DateTime(987654321).DateTimeOK, DateTimePrimitive(987654321), true},
		{"Time/panic", Double(0).Time, nil, ElementTypeError{"bson.Value.Time", bsontype.Double}},
		{"Time/success", Time(now).Time, now, nil},
		{"TimeOK/error", Double(0).TimeOK, time.Time{}, false},
		{"TimeOK/success", Time(now).TimeOK, now, true},
		{"Time->DateTime", Time(now).DateTime, nowdt, nil},
		{"DateTime->Time", DateTime(nowdt).Time, now, nil},
		{"Null/panic", Double(0).Null, nil, ElementTypeError{"bson.Value.Null", bsontype.Double}},
		{"Null/success", Null(NullPrimitive{}).Null, NullPrimitive{}, nil},
		{"NullOK/error", Double(0).NullOK, NullPrimitive{}, false},
		{"NullOK/success", Null(NullPrimitive{}).NullOK, NullPrimitive{}, true},
		{"Regex/panic", Double(0).Regex, nil, ElementTypeError{"bson.Value.Regex", bsontype.Double}},
		{"Regex/success", Regex(regex).Regex, regex, nil},
		{"RegexOK/error", Double(0).RegexOK, RegexPrimitive{}, false},
		{"RegexOK/success", Regex(regex).RegexOK, regex, true},
		{"DBPointer/panic", Double(0).DBPointer, nil, ElementTypeError{"bson.Value.DBPointer", bsontype.Double}},
		{"DBPointer/success", DBPointer(dbptr).DBPointer, dbptr, nil},
		{"DBPointerOK/error", Double(0).DBPointerOK, DBPointerPrimitive{}, false},
		{"DBPointerOK/success", DBPointer(dbptr).DBPointerOK, dbptr, true},
		{"JavaScript/panic", Double(0).JavaScript, nil, ElementTypeError{"bson.Value.JavaScript", bsontype.Double}},
		{"JavaScript/success", JavaScript(js).JavaScript, js, nil},
		{"JavaScriptOK/error", Double(0).JavaScriptOK, JavaScriptCodePrimitive(""), false},
		{"JavaScriptOK/success", JavaScript(js).JavaScriptOK, js, true},
		{"Symbol/panic", Double(0).Symbol, nil, ElementTypeError{"bson.Value.Symbol", bsontype.Double}},
		{"Symbol/success", Symbol(symbol).Symbol, symbol, nil},
		{"SymbolOK/error", Double(0).SymbolOK, SymbolPrimitive(""), false},
		{"SymbolOK/success", Symbol(symbol).SymbolOK, symbol, true},
		{"CodeWithScope/panic", Double(0).CodeWithScope, nil, ElementTypeError{"bson.Value.CodeWithScope", bsontype.Double}},
		{"CodeWithScope/success", CodeWithScope(cws).CodeWithScope, cws, nil},
		{"CodeWithScopeOK/error", Double(0).CodeWithScopeOK, CodeWithScopePrimitive{}, false},
		{"CodeWithScopeOK/success", CodeWithScope(cws).CodeWithScopeOK, cws, true},
		{"Int32/panic", Double(0).Int32, nil, ElementTypeError{"bson.Value.Int32", bsontype.Double}},
		{"Int32/success", Int32(12345).Int32, int32(12345), nil},
		{"Int32OK/error", Double(0).Int32OK, int32(0), false},
		{"Int32OK/success", Int32(54321).Int32OK, int32(54321), true},
		{"Timestamp/panic", Double(0).Timestamp, nil, ElementTypeError{"bson.Value.Timestamp", bsontype.Double}},
		{"Timestamp/success", Timestamp(ts).Timestamp, ts, nil},
		{"TimestampOK/error", Double(0).TimestampOK, TimestampPrimitive{}, false},
		{"TimestampOK/success", Timestamp(ts).TimestampOK, ts, true},
		{"Int64/panic", Double(0).Int64, nil, ElementTypeError{"bson.Value.Int64", bsontype.Double}},
		{"Int64/success", Int64(1234567890).Int64, int64(1234567890), nil},
		{"Int64OK/error", Double(0).Int64OK, int64(0), false},
		{"Int64OK/success", Int64(9876543210).Int64OK, int64(9876543210), true},
		{"Decimal128/panic", Double(0).Decimal128, nil, ElementTypeError{"bson.Value.Decimal128", bsontype.Double}},
		{"Decimal128/success", Decimal128(d128).Decimal128, d128, nil},
		{"Decimal128OK/error", Double(0).Decimal128OK, decimal.Decimal128{}, false},
		{"Decimal128OK/success", Decimal128(d128).Decimal128OK, d128, true},
		{"MinKey/panic", Double(0).MinKey, nil, ElementTypeError{"bson.Value.MinKey", bsontype.Double}},
		{"MinKey/success", MinKey(MinKeyPrimitive{}).MinKey, MinKeyPrimitive{}, nil},
		{"MinKeyOK/error", Double(0).MinKeyOK, MinKeyPrimitive{}, false},
		{"MinKeyOK/success", MinKey(MinKeyPrimitive{}).MinKeyOK, MinKeyPrimitive{}, true},
		{"MaxKey/panic", Double(0).MaxKey, nil, ElementTypeError{"bson.Value.MaxKey", bsontype.Double}},
		{"MaxKey/success", MaxKey(MaxKeyPrimitive{}).MaxKey, MaxKeyPrimitive{}, nil},
		{"MaxKeyOK/error", Double(0).MaxKeyOK, MaxKeyPrimitive{}, false},
		{"MaxKeyOK/success", MaxKey(MaxKeyPrimitive{}).MaxKeyOK, MaxKeyPrimitive{}, true},
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
				if !cmp.Equal(got, want, cmp.Comparer(compareDecimal128)) {
					t.Errorf("error booleans should be equal. got %v; want %v", got, want)
				}
				fallthrough
			case 1:
				got, want := ret[0].Interface(), tc.ret
				if !cmp.Equal(got, want, cmp.Comparer(compareDecimal128)) {
					t.Errorf("return values should be equal. got %v; want %v", got, want)
				}
			default:
				t.Fatalf("fn should return one or two values, but returned %d values", len(ret))
			}
		})
	}

	t.Run("Equal", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name string
			v1   Valuev2
			v2   Valuev2
			res  bool
		}{
			{"Different Types", String(""), Double(0), false},
			{"Unknown Types", Valuev2{t: bsontype.Type(0x77)}, Valuev2{t: bsontype.Type(0x77)}, false},
			{"Empty Types", Valuev2{}, Valuev2{}, true},
			{"Double/Equal", Double(3.14159), Double(3.14159), true},
			{"Double/Not Equal", Double(3.14159), Double(9.51413), false},
			{"DateTime/Equal", DateTime(nowdt), DateTime(nowdt), true},
			{"DateTime/Not Equal", DateTime(nowdt), DateTime(0), false},
			{"String/Equal", String("hello"), String("hello"), true},
			{"String/Not Equal", String("hello"), String("world"), false},
			{"Document/Equal", Embed(NewDocument()), Embed(NewDocument()), true},
			{"Document/Not Equal", Embed(NewDocument()), Embed(NewDocument(EC.Null(""))), false},
			{"Array/Equal", Embed(NewArray()), Embed(NewArray()), true},
			{"Array/Not Equal", Embed(NewArray()), Embed(NewArray(VC.Null())), false},
			{"Binary/Equal", Binary(bin), Binary(bin), true},
			{"Binary/Not Equal", Binary(bin), Binary(BinaryPrimitive{}), false},
			{"Undefined/Equal", Undefined(UndefinedPrimitive{}), Undefined(UndefinedPrimitive{}), true},
			{"ObjectID/Equal", ObjectID(oid), ObjectID(oid), true},
			{"ObjectID/Not Equal", ObjectID(oid), ObjectID(objectid.ObjectID{}), false},
			{"Boolean/Equal", Boolean(true), Boolean(true), true},
			{"Boolean/Not Equal", Boolean(true), Boolean(false), false},
			{"Null/Equal", Null(NullPrimitive{}), Null(NullPrimitive{}), true},
			{"Regex/Equal", Regex(regex), Regex(regex), true},
			{"Regex/Not Equal", Regex(regex), Regex(RegexPrimitive{}), false},
			{"DBPointer/Equal", DBPointer(dbptr), DBPointer(dbptr), true},
			{"DBPointer/Not Equal", DBPointer(dbptr), DBPointer(DBPointerPrimitive{}), false},
			{"JavaScript/Equal", JavaScript(js), JavaScript(js), true},
			{"JavaScript/Not Equal", JavaScript(js), JavaScript(JavaScriptCodePrimitive("")), false},
			{"Symbol/Equal", Symbol(symbol), Symbol(symbol), true},
			{"Symbol/Not Equal", Symbol(symbol), Symbol(SymbolPrimitive("")), false},
			{"CodeWithScope/Equal", CodeWithScope(cws), CodeWithScope(cws), true},
			{"CodeWithScope/Not Equal", CodeWithScope(cws), CodeWithScope(CodeWithScopePrimitive{}), false},
			{"Int32/Equal", Int32(12345), Int32(12345), true},
			{"Int32/Not Equal", Int32(12345), Int32(54321), false},
			{"Timestamp/Equal", Timestamp(ts), Timestamp(ts), true},
			{"Timestamp/Not Equal", Timestamp(ts), Timestamp(TimestampPrimitive{}), false},
			{"Int64/Equal", Int64(1234567890), Int64(1234567890), true},
			{"Int64/Not Equal", Int64(1234567890), Int64(9876543210), false},
			{"Decimal128/Equal", Decimal128(d128), Decimal128(d128), true},
			{"Decimal128/Not Equal", Decimal128(d128), Decimal128(decimal.Decimal128{}), false},
			{"MinKey/Equal", MinKey(MinKeyPrimitive{}), MinKey(MinKeyPrimitive{}), true},
			{"MaxKey/Equal", MaxKey(MaxKeyPrimitive{}), MaxKey(MaxKeyPrimitive{}), true},
		}

		for _, tc := range testCases {
			tc := tc // capture range variable
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				res := tc.v1.Equal(tc.v2)
				if res != tc.res {
					t.Errorf("results do not match. got %v; want %v", res, tc.res)
				}
			})
		}
	})
}
