package msg_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/10gen/mongo-go-driver/bson"
	. "github.com/10gen/mongo-go-driver/msg"
)

func TestWireProtocolDecodeReply(t *testing.T) {
	t.Parallel()

	subject := NewWireProtocolCodec()

	tests := []struct {
		desc     string
		bytes    []byte
		expected *Reply
		docs     []bson.D
	}{
		{
			"single document result",
			hexToBytes("3100000002000000010000000100000008000000090000000000000003000000010000000d00000008686f776479000100"),
			&Reply{
				ReqID:          2,
				RespTo:         1,
				ResponseFlags:  AwaitCapable,
				CursorID:       9,
				StartingFrom:   3,
				NumberReturned: 1,
				DocumentsBytes: hexToBytes("0d00000008686f776479000100"),
			},
			[]bson.D{
				{{"howdy", true}},
			},
		},
	}

	for i, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			buf := bytes.NewBuffer(test.bytes)

			msg, err := subject.Decode(buf)
			if err != nil {
				t.Errorf("failed reading msg #%d: %v", i, err)
			}

			expectedBytes, _ := bson.Marshal(test.expected)
			actualBytes, _ := bson.Marshal(msg)

			if string(expectedBytes) != string(actualBytes) {
				t.Errorf("msg #%d is not the same as expected\n  expected: %v\n  actual  : %v", i, expectedBytes, actualBytes)
			}

			actualIter := msg.(*Reply).Iter()
			var result bson.D
			j := 0
			for actualIter.Next(&result) {
				expectedBytes, _ = bson.Marshal(test.docs[j])
				actualBytes, _ = bson.Marshal(result)
				if string(expectedBytes) != string(actualBytes) {
					t.Errorf("msg #%d document #%d is not the same as expected\n  expected: %v\n  actual  : %v", i, j, expectedBytes, actualBytes)
				}
				j++
			}
			if actualIter.Err() != nil {
				t.Errorf("msg #%d could not unmarshal document #%d: %v", i, j, actualIter.Err())
			}
			if j != len(test.docs) {
				t.Errorf("msg #%d did not iterate all the documents\n  expected: %d\n  actual  : %d", i, len(test.docs), j)
			}
		})
	}
}

func TestWireProtocolEncodeQuery(t *testing.T) {
	t.Parallel()

	subject := NewWireProtocolCodec()

	tests := []struct {
		desc     string
		msg      *Query
		expected []byte
	}{
		{
			"with 2 flags",
			&Query{
				ReqID:              1,
				Flags:              SlaveOK | NoCursorTimeout,
				FullCollectionName: "test.foo",
				NumberToSkip:       2,
				NumberToReturn:     1000,
				Query:              bson.D{{"howdy", true}},
			},
			hexToBytes("320000000100000000000000d407000014000000746573742e666f6f0002000000e80300000d00000008686f776479000100"),
		},
		{
			"with no flags",
			&Query{
				ReqID:                2,
				FullCollectionName:   "test.foo",
				Query:                bson.D{{"howdy", true}},
				ReturnFieldsSelector: bson.D{{"one", 1}, {"two", 1}},
			},
			hexToBytes("490000000200000000000000d407000000000000746573742e666f6f0000000000000000000d00000008686f77647900010017000000106f6e6500010000001074776f000100000000"),
		},
	}

	for i, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			var buf bytes.Buffer
			err := subject.Encode(&buf, test.msg)
			if err != nil {
				t.Errorf("failed writing msg #%d: %v", i, err)
			}

			if !bytes.Equal(test.expected, buf.Bytes()) {
				t.Errorf("msg #%d does not match\n  expected: %v\n  actual  : %v", i, test.expected, buf.Bytes())
			}
		})
	}
}

func hexToBytes(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}
