package bson

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

func TestBasicDecodeDocumentReader(t *testing.T) {
	for _, tc := range unmarshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			got := reflect.New(tc.sType).Interface()
			doc, err := ReadDocument(tc.data)
			noerr(t, err)
			vr, err := NewDocumentValueReader(doc)
			noerr(t, err)
			reg := NewRegistryBuilder().Build()
			codec, err := reg.Lookup(reflect.TypeOf(got))
			noerr(t, err)
			err = codec.DecodeValue(DecodeContext{Registry: reg}, vr, got)
			noerr(t, err)

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Results do not match. got %+v; want %+v", got, tc.want)
			}
		})
	}
}

func TestDocumentValueReader(t *testing.T) {
	oid := objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
	now := time.Now().UnixNano() / int64(time.Millisecond)
	d128 := decimal.NewDecimal128(1, 2)
	testCases := []struct {
		name    string
		v       *Value
		fn      reflect.Value
		results []interface{}
	}{
		{
			"ReadBinary/incorrect type",
			nil,
			reflect.ValueOf((*documentValueReader).ReadBinary),
			[]interface{}{[]byte(nil), byte(0), (&documentValueReader{stack: []dvrState{{}}}).typeError(TypeBinary)},
		},
		{
			"ReadBinary/success",
			VC.BinaryWithSubtype([]byte{0x01, 0x02, 0x03}, 0xEA),
			reflect.ValueOf((*documentValueReader).ReadBinary),
			[]interface{}{[]byte{0x01, 0x02, 0x03}, byte(0xEA), nil},
		},
		{
			"ReadBoolean/incorrect type",
			nil,
			reflect.ValueOf((*documentValueReader).ReadBoolean),
			[]interface{}{false, (&documentValueReader{stack: []dvrState{{}}}).typeError(TypeBoolean)},
		},
		{
			"ReadBoolean/success",
			VC.Boolean(true),
			reflect.ValueOf((*documentValueReader).ReadBoolean),
			[]interface{}{true, nil},
		},
		{
			"ReadDBPointer/incorrect type",
			nil,
			reflect.ValueOf((*documentValueReader).ReadDBPointer),
			[]interface{}{"", objectid.ObjectID{}, (&documentValueReader{stack: []dvrState{{}}}).typeError(TypeDBPointer)},
		},
		{
			"ReadDBPointer/success",
			VC.DBPointer("foobar", oid),
			reflect.ValueOf((*documentValueReader).ReadDBPointer),
			[]interface{}{"foobar", oid, nil},
		},
		{
			"ReadDateTime/incorrect type",
			nil,
			reflect.ValueOf((*documentValueReader).ReadDateTime),
			[]interface{}{int64(0), (&documentValueReader{stack: []dvrState{{}}}).typeError(TypeDateTime)},
		},
		{
			"ReadDateTime/success",
			VC.DateTime(now),
			reflect.ValueOf((*documentValueReader).ReadDateTime),
			[]interface{}{now, nil},
		},
		{
			"ReadDecimal128/incorrect type",
			nil,
			reflect.ValueOf((*documentValueReader).ReadDecimal128),
			[]interface{}{decimal.Decimal128{}, (&documentValueReader{stack: []dvrState{{}}}).typeError(TypeDecimal128)},
		},
		{
			"ReadDecimal128/success",
			VC.Decimal128(d128),
			reflect.ValueOf((*documentValueReader).ReadDecimal128),
			[]interface{}{d128, nil},
		},
		{
			"ReadDouble/incorrect type",
			nil,
			reflect.ValueOf((*documentValueReader).ReadDouble),
			[]interface{}{float64(0), (&documentValueReader{stack: []dvrState{{}}}).typeError(TypeDouble)},
		},
		{
			"ReadDouble/success",
			VC.Double(1.2345),
			reflect.ValueOf((*documentValueReader).ReadDouble),
			[]interface{}{1.2345, nil},
		},
		{
			"ReadInt32/incorrect type",
			nil,
			reflect.ValueOf((*documentValueReader).ReadInt32),
			[]interface{}{int32(0), (&documentValueReader{stack: []dvrState{{}}}).typeError(TypeInt32)},
		},
		{
			"ReadInt32/success",
			VC.Int32(12345),
			reflect.ValueOf((*documentValueReader).ReadInt32),
			[]interface{}{int32(12345), nil},
		},
		{
			"ReadInt64/incorrect type",
			nil,
			reflect.ValueOf((*documentValueReader).ReadInt64),
			[]interface{}{int64(0), (&documentValueReader{stack: []dvrState{{}}}).typeError(TypeInt64)},
		},
		{
			"ReadInt64/success",
			VC.Int64(1234567890),
			reflect.ValueOf((*documentValueReader).ReadInt64),
			[]interface{}{int64(1234567890), nil},
		},
		{
			"ReadJavascript/incorrect type",
			nil,
			reflect.ValueOf((*documentValueReader).ReadJavascript),
			[]interface{}{"", (&documentValueReader{stack: []dvrState{{}}}).typeError(TypeJavaScript)},
		},
		{
			"ReadJavascript/success",
			VC.JavaScript("var foo = bar;"),
			reflect.ValueOf((*documentValueReader).ReadJavascript),
			[]interface{}{"var foo = bar;", nil},
		},
		{
			"ReadMaxKey/incorrect type",
			nil,
			reflect.ValueOf((*documentValueReader).ReadMaxKey),
			[]interface{}{(&documentValueReader{stack: []dvrState{{}}}).typeError(TypeMaxKey)},
		},
		{
			"ReadMaxKey/success",
			VC.MaxKey(),
			reflect.ValueOf((*documentValueReader).ReadMaxKey),
			[]interface{}{nil},
		},
		{
			"ReadMinKey/incorrect type",
			nil,
			reflect.ValueOf((*documentValueReader).ReadMinKey),
			[]interface{}{(&documentValueReader{stack: []dvrState{{}}}).typeError(TypeMinKey)},
		},
		{
			"ReadMinKey/success",
			VC.MinKey(),
			reflect.ValueOf((*documentValueReader).ReadMinKey),
			[]interface{}{nil},
		},
		{
			"ReadNull/incorrect type",
			nil,
			reflect.ValueOf((*documentValueReader).ReadNull),
			[]interface{}{(&documentValueReader{stack: []dvrState{{}}}).typeError(TypeNull)},
		},
		{
			"ReadNull/success",
			VC.Null(),
			reflect.ValueOf((*documentValueReader).ReadNull),
			[]interface{}{nil},
		},
		{
			"ReadObjectID/incorrect type",
			nil,
			reflect.ValueOf((*documentValueReader).ReadObjectID),
			[]interface{}{objectid.ObjectID{}, (&documentValueReader{stack: []dvrState{{}}}).typeError(TypeObjectID)},
		},
		{
			"ReadObjectID/success",
			VC.ObjectID(oid),
			reflect.ValueOf((*documentValueReader).ReadObjectID),
			[]interface{}{oid, nil},
		},
		{
			"ReadRegex/incorrect type",
			nil,
			reflect.ValueOf((*documentValueReader).ReadRegex),
			[]interface{}{"", "", (&documentValueReader{stack: []dvrState{{}}}).typeError(TypeRegex)},
		},
		{
			"ReadRegex/success",
			VC.Regex("foo", "bar"),
			reflect.ValueOf((*documentValueReader).ReadRegex),
			[]interface{}{"foo", "bar", nil},
		},
		{
			"ReadString/incorrect type",
			nil,
			reflect.ValueOf((*documentValueReader).ReadString),
			[]interface{}{"", (&documentValueReader{stack: []dvrState{{}}}).typeError(TypeString)},
		},
		{
			"ReadString/success",
			VC.String("hello, world!"),
			reflect.ValueOf((*documentValueReader).ReadString),
			[]interface{}{"hello, world!", nil},
		},
		{
			"ReadSymbol/incorrect type",
			nil,
			reflect.ValueOf((*documentValueReader).ReadSymbol),
			[]interface{}{"", (&documentValueReader{stack: []dvrState{{}}}).typeError(TypeSymbol)},
		},
		{
			"ReadSymbol/success",
			VC.Symbol("hello, world!"),
			reflect.ValueOf((*documentValueReader).ReadSymbol),
			[]interface{}{"hello, world!", nil},
		},
		{
			"ReadTimestamp/incorrect type",
			nil,
			reflect.ValueOf((*documentValueReader).ReadTimestamp),
			[]interface{}{uint32(0), uint32(0), (&documentValueReader{stack: []dvrState{{}}}).typeError(TypeTimestamp)},
		},
		{
			"ReadTimestamp/success",
			VC.Timestamp(10, 20),
			reflect.ValueOf((*documentValueReader).ReadTimestamp),
			[]interface{}{uint32(10), uint32(20), nil},
		},
		{
			"ReadUndefined/incorrect type",
			nil,
			reflect.ValueOf((*documentValueReader).ReadUndefined),
			[]interface{}{(&documentValueReader{stack: []dvrState{{}}}).typeError(TypeUndefined)},
		},
		{
			"ReadUndefined/success",
			VC.Undefined(),
			reflect.ValueOf((*documentValueReader).ReadUndefined),
			[]interface{}{nil},
		},
		{
			"ReadCodeWithScope/incorrect type",
			nil,
			reflect.ValueOf((*documentValueReader).ReadCodeWithScope),
			[]interface{}{"", nil, (&documentValueReader{stack: []dvrState{{}}}).typeError(TypeCodeWithScope)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dvr := &documentValueReader{
				stack: []dvrState{
					{mode: mTopLevel},
					{
						mode: mElement,
						v:    tc.v,
					},
				},
				frame: 1,
			}

			results := tc.fn.Call([]reflect.Value{reflect.ValueOf(dvr)})
			if len(results) != len(tc.results) {
				t.Fatalf("Length of results does not match. got %d; want %d", len(results), len(tc.results))
			}
			for idx := range results {
				got := results[idx].Interface()
				want := tc.results[idx]
				if !cmp.Equal(got, want, cmp.Comparer(compareErrors), cmp.Comparer(compareDecimal128)) {
					t.Errorf("Result %d does not match. got %v; want %v", idx, got, want)
				}
			}
		})
	}

	t.Run("ReadCodeWithScope/success", func(t *testing.T) {
		wantkey, wantcode := "foo", "var hello = world;"
		doc := NewDocument(
			EC.CodeWithScope(wantkey, wantcode,
				NewDocument(EC.Undefined("bar")),
			))
		dvr := &documentValueReader{
			stack: []dvrState{
				{mode: mTopLevel, d: doc},
			},
		}
		dr, err := dvr.ReadDocument()
		noerr(t, err)
		key, er, err := dr.ReadElement()
		noerr(t, err)
		if key != wantkey {
			t.Errorf("Incorrect key returned. got %s; want %s", key, wantkey)
		}
		code, dr, err := er.ReadCodeWithScope()
		noerr(t, err)
		if code != wantcode {
			t.Errorf("Incorrect code returned. got %s; want %s", code, wantcode)
		}
		wantmode := mCodeWithScope
		gotmode := dvr.stack[dvr.frame].mode
		if gotmode != wantmode {
			t.Errorf("Incorrect mode after reading code with scope. got %v; want %v", gotmode, wantmode)
		}
	})

	t.Run("Skip/invalid mode", func(t *testing.T) {
		dvr := &documentValueReader{stack: []dvrState{{mode: mTopLevel}}}
		wanterr := (&documentValueReader{stack: []dvrState{{mode: mTopLevel}}}).invalidTransitionErr(0)
		goterr := dvr.Skip()
		if !cmp.Equal(goterr, wanterr, cmp.Comparer(compareErrors)) {
			t.Errorf("Expected correct invalid transition error. got %v; want %v", goterr, wanterr)
		}
	})

	t.Run("Skip/success", func(t *testing.T) {
		firstkey, secondkey := "foo", "baz"
		doc := NewDocument(EC.String(firstkey, "bar"), EC.Null(secondkey))
		dvr, err := NewDocumentValueReader(doc)
		noerr(t, err)
		dr, err := dvr.ReadDocument()
		noerr(t, err)
		key, er, err := dr.ReadElement()
		noerr(t, err)
		if key != firstkey {
			t.Errorf("First key does not match. got %s; want %s", key, firstkey)
		}
		err = er.Skip()
		noerr(t, err)
		key, _, err = dr.ReadElement()
		noerr(t, err)
		if key != secondkey {
			t.Errorf("Second key does not match. got %s; want %s", key, secondkey)
		}
	})
}

func compareErrors(err1, err2 error) bool {
	if err1 == nil && err2 == nil {
		return true
	}

	if err1 == nil || err2 == nil {
		return false
	}

	if err1.Error() != err2.Error() {
		return false
	}

	return true
}

func compareDecimal128(d1, d2 decimal.Decimal128) bool {
	d1H, d1L := d1.GetBytes()
	d2H, d2L := d2.GetBytes()

	if d1H != d2H {
		return false
	}

	if d1L != d2L {
		return false
	}

	return true
}
