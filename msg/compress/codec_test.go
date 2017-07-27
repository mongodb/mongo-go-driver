package compress_test

import (
	"bytes"
	"testing"

	"encoding/hex"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/msg"
	. "github.com/10gen/mongo-go-driver/msg/compress"
)

func TestCompressDecodeReply(t *testing.T) {
	t.Parallel()

	subject := NewCodec(msg.NewWireProtocolCodec(), NewZLibCompressor())

	tests := []struct {
		desc     string
		bytes    []byte
		expected *msg.Reply
		docs     []bson.D
	}{
		{
			"uncompressed single document result",
			hexToBytes("3100000002000000010000000100000008000000090000000000000003000000010000000d00000008686f776479000100"),
			&msg.Reply{
				ReqID:          2,
				RespTo:         1,
				ResponseFlags:  msg.AwaitCapable,
				CursorID:       9,
				StartingFrom:   3,
				NumberReturned: 1,
				DocumentsBytes: hexToBytes("0d00000008686f776479000100"),
			},
			[]bson.D{
				{{"howdy", true}},
			},
		},
		{
			"zlib compressed single document result",
			hexToBytes("420000000200000001000000dc070000010000002100000002789ce2606060e064800066060606460606065e0606068e8cfcf2944a06460640000000ffff105c0257"),
			&msg.Reply{
				ReqID:          2,
				RespTo:         1,
				ResponseFlags:  msg.AwaitCapable,
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

			m, err := subject.Decode(buf)
			if err != nil {
				t.Errorf("failed reading msg #%d: %v", i, err)
			}

			expectedBytes, _ := bson.Marshal(test.expected)
			actualBytes, _ := bson.Marshal(m)

			if string(expectedBytes) != string(actualBytes) {
				t.Errorf("msg #%d is not the same as expected\n  expected: %v\n  actual  : %v", i, expectedBytes, actualBytes)
			}

			actualIter := m.(*msg.Reply).Iter()
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

func TestCompressEncodeQuery(t *testing.T) {
	t.Parallel()

	subject := NewCodec(msg.NewWireProtocolCodec(), NewZLibCompressor())
	subject.SetCompressors([]string{"zlib"})

	tests := []struct {
		desc     string
		msg      *msg.Query
		expected []byte
	}{
		{
			"query",
			&msg.Query{
				ReqID:              1,
				Flags:              msg.SlaveOK | msg.NoCursorTimeout,
				FullCollectionName: "test.foo",
				NumberToSkip:       2,
				NumberToReturn:     1000,
				Query:              bson.D{{"howdy", true}},
			},
			hexToBytes("470000000100000000000000dc070000d40700002200000002789c1261606028492d2ed14bcbcf676062606078c1ccc0c0cbc0c0c091915f9e52c9c0c800080000ffff75460675"),
		},
		{
			"compress-able command",
			&msg.Query{
				ReqID:              2,
				FullCollectionName: "test.$cmd",
				Query:              bson.D{{"howdy", 1}},
			},
			hexToBytes("400000000200000000000000dc070000d40700002600000002789c6260606028492d2ed15349ce4d6180010110cec82f4fa9646004f101010000ffff78230593"),
		},
		{
			"uncompress-able command",
			&msg.Query{
				ReqID:              2,
				FullCollectionName: "test.$cmd",
				Query:              bson.D{{"saslStart", 1}},
			},
			hexToBytes("3a0000000200000000000000d407000000000000746573742e24636d6400000000000000000014000000107361736c5374617274000100000000"),
		},
		{
			"compress-able command in $query",
			&msg.Query{
				ReqID:              2,
				FullCollectionName: "test.$cmd",
				Query:              bson.D{{"$query", bson.D{{"howdy", 1}}}},
			},
			hexToBytes("4e0000000200000000000000dc070000d40700003300000002789c6260606028492d2ed15349ce4d618001590606066695c2d2d4a24a0601060606818cfcf2944a0646b024200000ffffda5f080d"),
		},
		{
			"uncompress-able command in $query",
			&msg.Query{
				ReqID:              2,
				FullCollectionName: "test.$cmd",
				Query:              bson.D{{"$query", bson.D{{"saslStart", 1}}}},
			},
			hexToBytes("470000000200000000000000d407000000000000746573742e24636d6400000000000000000021000000032471756572790014000000107361736c537461727400010000000000"),
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
