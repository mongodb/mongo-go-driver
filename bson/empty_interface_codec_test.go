package bson

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

func TestEmptyInterfaceCodec(t *testing.T) {
	testCases := []struct {
		name     string
		val      interface{}
		bsontype Type
	}{
		{
			"Double - float64",
			float64(3.14159),
			TypeDouble,
		},
		{
			"String - string",
			string("foo bar baz"),
			TypeString,
		},
		{
			"Embedded Document - *Document",
			NewDocument(EC.Null("foo")),
			TypeEmbeddedDocument,
		},
		{
			"Array - *Array",
			NewArray(VC.Double(3.14159)),
			TypeArray,
		},
		{
			"Binary - Binary",
			Binary{Subtype: 0xFF, Data: []byte{0x01, 0x02, 0x03}},
			TypeBinary,
		},
		{
			"Undefined - Undefined",
			Undefinedv2{},
			TypeUndefined,
		},
		{
			"ObjectID - objectid.ObjectID",
			objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			TypeObjectID,
		},
		{
			"Boolean - bool",
			bool(true),
			TypeBoolean,
		},
		{
			"DateTime - DateTime",
			DateTime(1234567890),
			TypeDateTime,
		},
		{
			"Null - Null",
			Nullv2{},
			TypeNull,
		},
		{
			"Regex - Regex",
			Regex{Pattern: "foo", Options: "bar"},
			TypeRegex,
		},
		{
			"DBPointer - DBPointer",
			DBPointer{
				DB:      "foobar",
				Pointer: objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			},
			TypeDBPointer,
		},
		{
			"JavaScript - JavaScriptCode",
			JavaScriptCode("var foo = 'bar';"),
			TypeJavaScript,
		},
		{
			"Symbol - Symbol",
			Symbol("foobarbazlolz"),
			TypeSymbol,
		},
		{
			"CodeWithScope - CodeWithScope",
			CodeWithScope{
				Code:  "var foo = 'bar';",
				Scope: NewDocument(EC.Double("foo", 3.14159)),
			},
			TypeCodeWithScope,
		},
		{
			"Int32 - int32",
			int32(123456),
			TypeInt32,
		},
		{
			"Int64 - int64",
			int64(1234567890),
			TypeInt64,
		},
		{
			"Timestamp - Timestamp",
			Timestamp{T: 12345, I: 67890},
			TypeTimestamp,
		},
		{
			"Decimal128 - decimal.Decimal128",
			decimal.NewDecimal128(12345, 67890),
			TypeDecimal128,
		},
		{
			"MinKey - MinKey",
			MinKeyv2{},
			TypeMinKey,
		},
		{
			"MaxKey - MaxKey",
			MaxKeyv2{},
			TypeMaxKey,
		},
	}

	t.Run("EncodeValue", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				llvr := &llValueReaderWriter{bsontype: tc.bsontype}
				eic := &EmptyInterfaceCodec{}

				t.Run("Lookup failure", func(t *testing.T) {
					ec := EncodeContext{Registry: NewEmptyRegistryBuilder().Build()}
					want := ErrNoCodec{Type: reflect.TypeOf(tc.val)}
					got := eic.EncodeValue(ec, llvr, tc.val)
					if !compareErrors(got, want) {
						t.Errorf("Errors are not equal. got %v; want %v", got, want)
					}
				})

				t.Run("Success", func(t *testing.T) {
					want := tc.val
					llc := &llCodec{t: t}
					ec := EncodeContext{
						Registry: NewEmptyRegistryBuilder().Register(reflect.TypeOf(tc.val), llc).Build(),
					}
					err := eic.EncodeValue(ec, llvr, tc.val)
					noerr(t, err)
					got := llc.encodeval
					if !cmp.Equal(got, want, cmp.Comparer(compareDecimal128)) {
						t.Errorf("Did not receive expected value. got %v; want %v", got, want)
					}
				})
			})
		}
	})

	t.Run("DecodeValue", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				llvr := &llValueReaderWriter{bsontype: tc.bsontype}
				eic := &EmptyInterfaceCodec{}

				t.Run("Lookup failure", func(t *testing.T) {
					val := new(interface{})
					dc := DecodeContext{Registry: NewEmptyRegistryBuilder().Build()}
					want := ErrNoCodec{Type: reflect.TypeOf(tc.val)}
					got := eic.DecodeValue(dc, llvr, val)
					if !compareErrors(got, want) {
						t.Errorf("Errors are not equal. got %v; want %v", got, want)
					}
				})

				t.Run("DecodeValue failure", func(t *testing.T) {
					want := errors.New("DecodeValue failure error")
					llc := &llCodec{t: t, err: want}
					dc := DecodeContext{
						Registry: NewEmptyRegistryBuilder().Register(reflect.TypeOf(tc.val), llc).Build(),
					}
					got := eic.DecodeValue(dc, llvr, new(interface{}))
					if !compareErrors(got, want) {
						t.Errorf("Errors are not equal. got %v; want %v", got, want)
					}
				})

				t.Run("Success", func(t *testing.T) {
					want := tc.val
					llc := &llCodec{t: t, decodeval: tc.val}
					dc := DecodeContext{
						Registry: NewEmptyRegistryBuilder().Register(reflect.TypeOf(tc.val), llc).Build(),
					}
					got := new(interface{})
					err := eic.DecodeValue(dc, llvr, got)
					noerr(t, err)
					if !cmp.Equal(*got, want, cmp.Comparer(compareDecimal128)) {
						t.Errorf("Did not receive expected value. got %v; want %v", *got, want)
					}
				})
			})
		}

		t.Run("non-*interface{}", func(t *testing.T) {
			eic := &EmptyInterfaceCodec{}
			val := uint64(1234567890)
			want := fmt.Errorf("%T can only be used to decode non-nil *interface{} values, provided type if %T", eic, &val)
			got := eic.DecodeValue(DecodeContext{}, nil, &val)
			if !compareErrors(got, want) {
				t.Errorf("Errors are not equal. got %v; want %v", got, want)
			}
		})

		t.Run("nil *interface{}", func(t *testing.T) {
			eic := &EmptyInterfaceCodec{}
			var val *interface{}
			want := fmt.Errorf("%T can only be used to decode non-nil *interface{} values, provided type if %T", eic, val)
			got := eic.DecodeValue(DecodeContext{}, nil, val)
			if !compareErrors(got, want) {
				t.Errorf("Errors are not equal. got %v; want %v", got, want)
			}
		})

		t.Run("unknown BSON type", func(t *testing.T) {
			llvr := &llValueReaderWriter{bsontype: Type(0)}
			eic := &EmptyInterfaceCodec{}
			want := fmt.Errorf("Type %s is not a valid BSON type and has no default Go type to decode into", Type(0))
			got := eic.DecodeValue(DecodeContext{}, llvr, new(interface{}))
			if !compareErrors(got, want) {
				t.Errorf("Errors are not equal. got %v; want %v", got, want)
			}
		})
	})
}

type llCodec struct {
	t         *testing.T
	decodeval interface{}
	encodeval interface{}
	err       error
}

func (llc *llCodec) EncodeValue(_ EncodeContext, _ ValueWriter, i interface{}) error {
	if llc.err != nil {
		return llc.err
	}

	llc.encodeval = i
	return nil
}

func (llc *llCodec) DecodeValue(_ DecodeContext, _ ValueReader, i interface{}) error {
	if llc.err != nil {
		return llc.err
	}

	val := reflect.ValueOf(i)
	if val.Type().Kind() != reflect.Ptr {
		llc.t.Errorf("Value provided to DecodeValue must be a pointer, but got %T", i)
		return nil
	}

	switch val.Type() {
	case tDocument:
		decodeval, ok := llc.decodeval.(*Document)
		if !ok {
			llc.t.Errorf("decodeval must be a *Document if the i is a *Document. decodeval %T", llc.decodeval)
			return nil
		}

		doc := i.(*Document)
		doc.Reset()
		err := doc.Concat(decodeval)
		if err != nil {
			llc.t.Errorf("could not concatenate the decoded val to doc: %v", err)
			return err
		}

		return nil
	case tArray:
		decodeval, ok := llc.decodeval.(*Array)
		if !ok {
			llc.t.Errorf("decodeval must be a *Array if the i is a *Array. decodeval %T", llc.decodeval)
			return nil
		}

		arr := i.(*Array)
		arr.Reset()
		err := arr.Concat(decodeval)
		if err != nil {
			llc.t.Errorf("could not concatenate the decoded val to array: %v", err)
			return err
		}

		return nil
	}

	if !reflect.TypeOf(llc.decodeval).AssignableTo(val.Type().Elem()) {
		llc.t.Errorf("decodeval must be assignable to i provided to DecodeValue, but is not. decodeval %T; i %T", llc.decodeval, i)
		return nil
	}

	val.Elem().Set(reflect.ValueOf(llc.decodeval))
	return nil
}
