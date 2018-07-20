package bson

import (
	"bytes"
	"testing"
)

func TestEncoderEncode(t *testing.T) {
	for _, tc := range marshalingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			got := make(writer, 0, 1024)
			vw := newValueWriter(&got)
			reg := NewRegistryBuilder().Build()
			enc, err := NewEncoderv2(reg, vw)
			noerr(t, err)
			err = enc.Encode(tc.val)
			noerr(t, err)

			if !bytes.Equal(got, tc.want) {
				t.Errorf("Bytes are not equal. got %v; want %v", Reader(got), Reader(tc.want))
				t.Errorf("Bytes:\n%v\n%v", got, tc.want)
			}
		})
	}
}
