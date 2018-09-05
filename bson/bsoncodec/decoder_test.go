package bsoncodec

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson"
)

func TestDecoderv2(t *testing.T) {
	t.Run("Decode", func(t *testing.T) {
		for _, tc := range unmarshalingTestCases {
			t.Run(tc.name, func(t *testing.T) {
				got := reflect.New(tc.sType).Interface()
				vr := newValueReader(tc.data)
				var reg *Registry
				if tc.reg != nil {
					reg = tc.reg
				} else {
					reg = NewRegistryBuilder().Build()
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
			dec, err := NewDecoder(defaultRegistry, new(valueReader))
			noerr(t, err)
			want := ErrNoDecoder{Type: reflect.TypeOf(cdeih)}
			got := dec.Decode(&cdeih)
			if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
				t.Errorf("Received unexpected error. got %v; want %v", got, want)
			}
		})
		t.Run("Unmarshaler", func(t *testing.T) {
			testCases := []struct {
				name    string
				err     error
				vr      ValueReader
				invoked bool
			}{
				{
					"error",
					errors.New("Unmarshaler error"),
					&llValueReaderWriter{bsontype: bson.TypeEmbeddedDocument, err: ErrEOD, errAfter: llvrwReadElement},
					true,
				},
				{
					"copy error",
					errors.New("copy error"),
					&llValueReaderWriter{err: errors.New("copy error"), errAfter: llvrwReadDocument},
					false,
				},
				{
					"success",
					nil,
					&llValueReaderWriter{bsontype: bson.TypeEmbeddedDocument, err: ErrEOD, errAfter: llvrwReadElement},
					true,
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					unmarshaler := &testUnmarshaler{err: tc.err}
					dec, err := NewDecoder(defaultRegistry, tc.vr)
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
			_, got := NewDecoder(nil, &valueReader{})
			want := errors.New("cannot create a new Decoder with a nil Registry")
			if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
				t.Errorf("Was expecting error but got different error. got %v; want %v", got, want)
			}
			_, got = NewDecoder(defaultRegistry, nil)
			want = errors.New("cannot create a new Decoder with a nil ValueReader")
			if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
				t.Errorf("Was expecting error but got different error. got %v; want %v", got, want)
			}
		})
		t.Run("success", func(t *testing.T) {
			got, err := NewDecoder(defaultRegistry, &valueReader{})
			noerr(t, err)
			if got == nil {
				t.Errorf("Was expecting a non-nil Decoder, but got <nil>")
			}
		})
	})
	t.Run("Reset", func(t *testing.T) {
		vr1, vr2 := new(valueReader), new(documentValueReader)
		dec, err := NewDecoder(defaultRegistry, vr1)
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
		reg1, reg2 := defaultRegistry, NewRegistryBuilder().Build()
		dec, err := NewDecoder(reg1, new(valueReader))
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

func (tdc *testDecoderCodec) EncodeValue(EncodeContext, ValueWriter, interface{}) error {
	tdc.EncodeValueCalled = true
	return nil
}

func (tdc *testDecoderCodec) DecodeValue(DecodeContext, ValueReader, interface{}) error {
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
