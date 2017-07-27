package compress

func addInt32(b []byte, i int32) []byte {
	return append(b, byte(i), byte(i>>8), byte(i>>16), byte(i>>24))
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
