package wiremessage

import (
	"bytes"
	"errors"
)

func readInt32(b []byte, pos int32) int32 {
	return (int32(b[pos+0])) | (int32(b[pos+1]) << 8) | (int32(b[pos+2]) << 16) | (int32(b[pos+3]) << 24)
}

func readCString(b []byte, pos int32) (string, error) {
	null := bytes.IndexByte(b[pos:], 0x00)
	if null == -1 {
		return "", errors.New("invalid cstring")
	}
	return string(b[pos : int(pos)+null]), nil
}

func readInt64(b []byte, pos int32) int64 {
	return (int64(b[pos+0])) | (int64(b[pos+1]) << 8) | (int64(b[pos+2]) << 16) | (int64(b[pos+3]) << 24) | (int64(b[pos+4]) << 32) |
		(int64(b[pos+5]) << 40) | (int64(b[pos+6]) << 48) | (int64(b[pos+7]) << 56)

}
