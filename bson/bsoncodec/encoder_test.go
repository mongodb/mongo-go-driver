package bsoncodec

import (
	"bytes"
	"errors"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
)

func TestEncoderEncode(t *testing.T) {
	for _, tc := range marshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			got := make(writer, 0, 1024)
			vw := newValueWriter(&got)
			reg := NewRegistryBuilder().Build()
			enc, err := NewEncoder(reg, vw)
			noerr(t, err)
			err = enc.Encode(tc.val)
			noerr(t, err)

			if !bytes.Equal(got, tc.want) {
				t.Errorf("Bytes are not equal. got %v; want %v", bson.Reader(got), bson.Reader(tc.want))
				t.Errorf("Bytes:\n%v\n%v", got, tc.want)
			}
		})
	}

	t.Run("Marshaler", func(t *testing.T) {
		testCases := []struct {
			name    string
			buf     []byte
			err     error
			wanterr error
			vw      ValueWriter
		}{
			{
				"error",
				nil,
				errors.New("Marshaler error"),
				errors.New("Marshaler error"),
				&llValueReaderWriter{},
			},
			{
				"copy error",
				[]byte{0x05, 0x00, 0x00, 0x00, 0x00},
				nil,
				errors.New("copy error"),
				&llValueReaderWriter{err: errors.New("copy error"), errAfter: llvrwWriteDocument},
			},
			{
				"success",
				[]byte{0x07, 0x00, 0x00, 0x00, 0x0A, 0x00, 0x00},
				nil,
				nil,
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				marshaler := testMarshaler{buf: tc.buf, err: tc.err}

				var vw ValueWriter
				compareVW := false
				if tc.vw != nil {
					vw = tc.vw
				} else {
					compareVW = true
					vw = newValueWriterFromSlice([]byte{})
				}

				enc, err := NewEncoder(defaultRegistry, vw)
				noerr(t, err)
				got := enc.Encode(marshaler)
				want := tc.wanterr
				if !compareErrors(got, want) {
					t.Errorf("Did not receive expected error. got %v; want %v", got, want)
				}
				if compareVW {
					buf := vw.(*valueWriter).buf
					if !bytes.Equal(buf, tc.buf) {
						t.Errorf("Copied bytes do not match. got %v; want %v", buf, tc.buf)
					}
				}
			})
		}
	})
}

type testMarshaler struct {
	buf []byte
	err error
}

func (tm testMarshaler) MarshalBSON() ([]byte, error) { return tm.buf, tm.err }
