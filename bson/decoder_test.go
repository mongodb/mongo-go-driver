package bson

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/bson/bsonrw"
	"github.com/mongodb/mongo-go-driver/bson/bsonrw/bsonrwtest"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
)

func TestBasicDecode(t *testing.T) {
	for _, tc := range unmarshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			got := reflect.New(tc.sType).Interface()
			vr := bsonrw.NewBSONValueReader(tc.data)
			reg := DefaultRegistry
			decoder, err := reg.LookupDecoder(reflect.TypeOf(got))
			noerr(t, err)
			err = decoder.DecodeValue(bsoncodec.DecodeContext{Registry: reg}, vr, got)
			noerr(t, err)

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Results do not match. got %+v; want %+v", got, tc.want)
			}
		})
	}
}

func TestDecoderv2(t *testing.T) {
	t.Run("Decode", func(t *testing.T) {
		for _, tc := range unmarshalingTestCases {
			t.Run(tc.name, func(t *testing.T) {
				got := reflect.New(tc.sType).Interface()
				vr := bsonrw.NewBSONValueReader(tc.data)
				var reg *bsoncodec.Registry
				if tc.reg != nil {
					reg = tc.reg
				} else {
					reg = DefaultRegistry
				}
				dec, err := NewDecoder(reg, vr)
				noerr(t, err)
				err = dec.Decode(got)
				noerr(t, err)

				if !reflect.DeepEqual(got, tc.want) {
					t.Errorf("Results do not match. got %+v; want %+v", got, tc.want)
				}
			})
		}
		t.Run("lookup error", func(t *testing.T) {
			type certainlydoesntexistelsewhereihope func(string, string) string
			cdeih := func(string, string) string { return "certainlydoesntexistelsewhereihope" }
			dec, err := NewDecoder(DefaultRegistry, bsonrw.NewBSONValueReader([]byte{}))
			noerr(t, err)
			want := bsoncodec.ErrNoDecoder{Type: reflect.TypeOf(cdeih)}
			got := dec.Decode(&cdeih)
			if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
				t.Errorf("Received unexpected error. got %v; want %v", got, want)
			}
		})
		t.Run("Unmarshaler", func(t *testing.T) {
			testCases := []struct {
				name    string
				err     error
				vr      bsonrw.ValueReader
				invoked bool
			}{
				{
					"error",
					errors.New("Unmarshaler error"),
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.EmbeddedDocument, Err: bsonrw.ErrEOD, ErrAfter: bsonrwtest.ReadElement},
					true,
				},
				{
					"copy error",
					errors.New("copy error"),
					&bsonrwtest.ValueReaderWriter{Err: errors.New("copy error"), ErrAfter: bsonrwtest.ReadDocument},
					false,
				},
				{
					"success",
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.EmbeddedDocument, Err: bsonrw.ErrEOD, ErrAfter: bsonrwtest.ReadElement},
					true,
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					unmarshaler := &testUnmarshaler{err: tc.err}
					dec, err := NewDecoder(DefaultRegistry, tc.vr)
					noerr(t, err)
					got := dec.Decode(unmarshaler)
					want := tc.err
					if !compareErrors(got, want) {
						t.Errorf("Did not receive expected error. got %v; want %v", got, want)
					}
					if unmarshaler.invoked != tc.invoked {
						if tc.invoked {
							t.Error("Expected to have UnmarshalBSON invoked, but it wasn't.")
						} else {
							t.Error("Expected UnmarshalBSON to not be invoked, but it was.")
						}
					}
				})
			}
		})
	})
	t.Run("NewDecoderv2", func(t *testing.T) {
		t.Run("errors", func(t *testing.T) {
			_, got := NewDecoder(nil, bsonrw.ValueReader(nil))
			want := errors.New("cannot create a new Decoder with a nil Registry")
			if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
				t.Errorf("Was expecting error but got different error. got %v; want %v", got, want)
			}
			_, got = NewDecoder(DefaultRegistry, nil)
			want = errors.New("cannot create a new Decoder with a nil ValueReader")
			if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
				t.Errorf("Was expecting error but got different error. got %v; want %v", got, want)
			}
		})
		t.Run("success", func(t *testing.T) {
			got, err := NewDecoder(DefaultRegistry, bsonrw.NewBSONValueReader([]byte{}))
			noerr(t, err)
			if got == nil {
				t.Errorf("Was expecting a non-nil Decoder, but got <nil>")
			}
		})
	})
	t.Run("Reset", func(t *testing.T) {
		vr1, vr2 := bsonrw.NewBSONValueReader([]byte{}), bsonrw.NewBSONValueReader([]byte{})
		dec, err := NewDecoder(DefaultRegistry, vr1)
		noerr(t, err)
		if dec.vr != vr1 {
			t.Errorf("Decoder should use the value reader provided. got %v; want %v", dec.vr, vr1)
		}
		err = dec.Reset(vr2)
		noerr(t, err)
		if dec.vr != vr2 {
			t.Errorf("Decoder should use the value reader provided. got %v; want %v", dec.vr, vr2)
		}
	})
	t.Run("SetRegistry", func(t *testing.T) {
		reg1, reg2 := DefaultRegistry, NewRegistryBuilder().Build()
		dec, err := NewDecoder(reg1, bsonrw.NewBSONValueReader([]byte{}))
		noerr(t, err)
		if dec.r != reg1 {
			t.Errorf("Decoder should use the Registry provided. got %v; want %v", dec.r, reg1)
		}
		err = dec.SetRegistry(reg2)
		noerr(t, err)
		if dec.r != reg2 {
			t.Errorf("Decoder should use the Registry provided. got %v; want %v", dec.r, reg2)
		}
	})
}

type testDecoderCodec struct {
	EncodeValueCalled bool
	DecodeValueCalled bool
}

func (tdc *testDecoderCodec) EncodeValue(bsoncodec.EncodeContext, bsonrw.ValueWriter, interface{}) error {
	tdc.EncodeValueCalled = true
	return nil
}

func (tdc *testDecoderCodec) DecodeValue(bsoncodec.DecodeContext, bsonrw.ValueReader, interface{}) error {
	tdc.DecodeValueCalled = true
	return nil
}

type testUnmarshaler struct {
	invoked bool
	err     error
}

func (tu *testUnmarshaler) UnmarshalBSON(_ []byte) error {
	tu.invoked = true
	return tu.err
}
