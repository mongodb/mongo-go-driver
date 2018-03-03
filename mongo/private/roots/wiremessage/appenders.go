package wiremessage

func appendInt32(b []byte, i int32) []byte {
	return append(b, byte(i), byte(i>>8), byte(i>>16), byte(i>>24))
}

func appendCString(b []byte, str string) []byte {
	b = append(b, str...)
	return append(b, 0x00)
}

func appendInt64(b []byte, i int64) []byte {
	return append(b, byte(i), byte(i>>8), byte(i>>16), byte(i>>24), byte(i>>32), byte(i>>40), byte(i>>48), byte(i>>56))
}
