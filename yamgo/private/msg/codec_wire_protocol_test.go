package msg_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/10gen/mongo-go-driver/bson"

	"encoding/json"

	. "github.com/10gen/mongo-go-driver/yamgo/private/msg"
)

func TestWireProtocolDecodeReply(t *testing.T) {
	t.Parallel()

	subject := NewWireProtocolCodec()

	tests := []struct {
		bytes    []byte
		expected *Reply
		docs     []bson.D
	}{
		{
			[]byte{0x31, 0, 0, 0, 2, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 8, 0, 0, 0, 9, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 1, 0, 0, 0, 0xD, 0, 0, 0, 8, 0x68, 0x6f, 0x77, 0x64, 0x79, 0, 1, 0},
			&Reply{
				ReqID:          2,
				RespTo:         1,
				ResponseFlags:  AwaitCapable,
				CursorID:       9,
				StartingFrom:   3,
				NumberReturned: 1,
				DocumentsBytes: []byte{0xD, 0, 0, 0, 8, 0x68, 0x6f, 0x77, 0x64, 0x79, 0, 1, 0},
			},
			[]bson.D{
				{{"howdy", true}},
			},
		},
	}

	for i, test := range tests {
		buf := bytes.NewBuffer(test.bytes)

		msg, err := subject.Decode(buf)
		if err != nil {
			t.Errorf("failed reading msg #%d: %v", i, err)
		}

		expectedBytes, _ := json.Marshal(test.expected)
		actualBytes, _ := json.Marshal(msg)

		if string(expectedBytes) != string(actualBytes) {
			t.Errorf("msg #%d is not the same as expected\n  expected: %s\n  actual  : %s", i, string(expectedBytes), string(actualBytes))
		}

		actualIter := msg.(*Reply).Iter()
		var result bson.D
		j := 0
		for actualIter.Next(&result) {
			expectedBytes, _ = json.Marshal(test.docs[j])
			actualBytes, _ = json.Marshal(result)
			if string(expectedBytes) != string(actualBytes) {
				t.Errorf("msg #%d document #%d is not the same as expected\n  expected: %s\n  actual  : %s", i, j, string(expectedBytes), string(actualBytes))
			}
			j++
		}
		if actualIter.Err() != nil {
			t.Errorf("msg #%d could not unmarshal document #%d: %v", i, j, actualIter.Err())
		}
		if j != len(test.docs) {
			t.Errorf("msg #%d did not iterate all the documents\n  expected: %d\n  actual  : %d", i, len(test.docs), j)
		}
	}
}

func TestWireProtocolEncodeQuery(t *testing.T) {
	t.Parallel()

	subject := NewWireProtocolCodec()

	tests := []struct {
		msg         *Query
		expectedHex string
	}{
		{
			&Query{
				ReqID:              1,
				Flags:              SlaveOK | NoCursorTimeout,
				FullCollectionName: "test.foo",
				NumberToSkip:       2,
				NumberToReturn:     1000,
				Query:              bson.D{{"howdy", true}},
			},
			"32 00 00 00 01 00 00 00 00 00 00 00 d4 07 00 00 14 00 00 00 74 65 73 74 2e 66 6f 6f 00 02 00 00 00 e8 03 00 00 0d 00 00 00 08 68 6f 77 64 79 00 01 00",
		},
		{
			&Query{
				ReqID:                2,
				FullCollectionName:   "test.foo",
				Query:                bson.D{{"howdy", true}},
				ReturnFieldsSelector: bson.D{{"one", 1}, {"two", 1}},
			},
			"49 00 00 00 02 00 00 00 00 00 00 00 d4 07 00 00 00 00 00 00 74 65 73 74 2e 66 6f 6f 00 00 00 00 00 00 00 00 00 0d 00 00 00 08 68 6f 77 64 79 00 01 00 17 00 00 00 10 6f 6e 65 00 01 00 00 00 10 74 77 6f 00 01 00 00 00 00",
		},
	}

	for i, test := range tests {
		var buf bytes.Buffer
		err := subject.Encode(&buf, test.msg)
		if err != nil {
			t.Errorf("failed writing msg #%d: %v", i, err)
		}

		actual := fmt.Sprintf("% x", buf.Bytes())
		if test.expectedHex != actual {
			t.Errorf("msg #%d does not match\n  expected: %s\n  actual  : %s", i, test.expectedHex, actual)
		}
	}

}
