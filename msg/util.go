package msg

import (
	"fmt"

	"github.com/10gen/mongo-go-driver/bson"
)

func addCString(b []byte, s string) []byte {
	b = append(b, []byte(s)...)
	return append(b, 0)
}

func addInt32(b []byte, i int32) []byte {
	return append(b, byte(i), byte(i>>8), byte(i>>16), byte(i>>24))
}

func addInt64(b []byte, i int64) []byte {
	return append(b, byte(i), byte(i>>8), byte(i>>16), byte(i>>24), byte(i>>32), byte(i>>40), byte(i>>48), byte(i>>56))
}

func addMarshalled(b []byte, data interface{}) ([]byte, error) {
	if data == nil {
		return append(b, 5, 0, 0, 0, 0), nil
	}

	dataBytes, err := bson.Marshal(data)
	if err != nil {
		return nil, err
	}

	return append(b, dataBytes...), nil
}

func setInt32(b []byte, pos int32, i int32) {
	b[pos] = byte(i)
	b[pos+1] = byte(i >> 8)
	b[pos+2] = byte(i >> 16)
	b[pos+3] = byte(i >> 24)
}

func addHeader(b []byte, length, requestID, responseTo, opCode int32) []byte {
	b = addInt32(b, length)
	b = addInt32(b, requestID)
	b = addInt32(b, responseTo)
	return addInt32(b, opCode)
}

func readInt32(b []byte, pos int32) int32 {
	return (int32(b[pos+0])) |
		(int32(b[pos+1]) << 8) |
		(int32(b[pos+2]) << 16) |
		(int32(b[pos+3]) << 24)
}

func readInt64(b []byte, pos int32) int64 {
	return (int64(b[pos+0])) |
		(int64(b[pos+1]) << 8) |
		(int64(b[pos+2]) << 16) |
		(int64(b[pos+3]) << 24) |
		(int64(b[pos+4]) << 32) |
		(int64(b[pos+5]) << 40) |
		(int64(b[pos+6]) << 48) |
		(int64(b[pos+7]) << 56)
}

func readUint8(b []byte, pos int32) uint8 {
	return uint8(b[pos])
}

func bsonDocumentPartitioner(bytes []byte) (int, error) {
	if len(bytes) < 4 {
		return 0, fmt.Errorf("int32 requires 4 bytes but only %d available", len(bytes))
	}

	n := readInt32(bytes, 0)
	return int(n), nil
}
