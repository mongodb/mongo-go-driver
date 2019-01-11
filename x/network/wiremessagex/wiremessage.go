package wiremessagex

import (
	"bytes"

	"github.com/mongodb/mongo-go-driver/x/bsonx/bsoncore"
	"github.com/mongodb/mongo-go-driver/x/network/wiremessage"
)

type WireMessage []byte

type OpCode = wiremessage.OpCode

func AppendHeaderStart(dst []byte, reqid, respto int32, opcode OpCode) (index int32, b []byte) {
	index, dst = bsoncore.ReserveLength(dst)
	dst = appendi32(dst, reqid)
	dst = appendi32(dst, respto)
	dst = appendi32(dst, int32(opcode))
	return index, dst
}

func ReadHeader(src []byte) (length, requestID, responseTo int32, opcode OpCode, rem []byte, ok bool) {
	if len(src) < 16 {
		return 0, 0, 0, 0, src, false
	}
	length = (int32(src[0]) | int32(src[1])<<8 | int32(src[2])<<16 | int32(src[3])<<24)
	requestID = (int32(src[4]) | int32(src[5])<<8 | int32(src[6])<<16 | int32(src[7])<<24)
	responseTo = (int32(src[8]) | int32(src[9])<<8 | int32(src[10])<<16 | int32(src[11])<<24)
	opcode = OpCode(int32(src[12]) | int32(src[13])<<8 | int32(src[14])<<16 | int32(src[15])<<24)
	return length, requestID, responseTo, opcode, src[16:], true
}

func AppendQueryFlags(dst []byte, flags wiremessage.QueryFlag) []byte {
	return appendi32(dst, int32(flags))
}

func AppendMsgFlags(dst []byte, flags wiremessage.MsgFlag) []byte {
	return appendi32(dst, int32(flags))
}

func AppendMsgSectionType(dst []byte, stype wiremessage.SectionType) []byte {
	return append(dst, byte(stype))
}

func AppendQueryFullCollectionName(dst []byte, ns string) []byte {
	return appendCString(dst, ns)
}

func AppendQueryNumberToSkip(dst []byte, skip int32) []byte {
	return appendi32(dst, skip)
}

func AppendQueryNumberToReturn(dst []byte, nor int32) []byte {
	return appendi32(dst, nor)
}

func ReadMsgFlags(src []byte) (flags wiremessage.MsgFlag, rem []byte, ok bool) {
	i32, rem, ok := readi32(src)
	return wiremessage.MsgFlag(i32), rem, ok
}

func ReadMsgSectionType(src []byte) (stype wiremessage.SectionType, rem []byte, ok bool) {
	if len(src) < 1 {
		return 0, src, false
	}
	return wiremessage.SectionType(src[0]), src[1:], true
}

func ReadMsgSectionSingleDocument(src []byte) (doc bsoncore.Document, rem []byte, ok bool) {
	return bsoncore.ReadDocument(src)
}

func ReadMsgSectionDocumentSequence(src []byte) (identifier string, docs []bsoncore.Document, rem []byte, ok bool) {
	length, rem, ok := readi32(src)
	if !ok || int(length) > len(src) {
		return "", nil, rem, false
	}

	rem, ret := rem[:length-4], rem[length-4:] // reslice so we can just iterate a loop later

	identifier, rem, ok = readcstring(rem)
	if !ok {
		return "", nil, rem, false
	}

	docs = make([]bsoncore.Document, 0)
	var doc bsoncore.Document
	for {
		doc, rem, ok = bsoncore.ReadDocument(rem)
		if !ok {
			break
		}
		docs = append(docs, doc)
	}
	if len(rem) > 0 {
		return "", nil, append(rem, ret...), false
	}

	return identifier, docs, ret, true
}

func ReadMsgChecksum(src []byte) (checksum uint32, rem []byte, ok bool) {
	i32, rem, ok := readi32(src)
	return uint32(i32), rem, ok
}

func ReadReplyFlags(src []byte) (flags wiremessage.ReplyFlag, rem []byte, ok bool) {
	i32, rem, ok := readi32(src)
	return wiremessage.ReplyFlag(i32), rem, ok
}

func ReadReplyCursorID(src []byte) (cursorID int64, rem []byte, ok bool) {
	return readi64(src)
}

func ReadReplyStartingFrom(src []byte) (startingFrom int32, rem []byte, ok bool) {
	return readi32(src)
}

func ReadReplyNumberReturned(src []byte) (numberReturned int32, rem []byte, ok bool) {
	return readi32(src)
}

func ReadReplyDocument(src []byte) (doc bsoncore.Document, rem []byte, ok bool) {
	return bsoncore.ReadDocument(src)
}

func appendi32(dst []byte, i32 int32) []byte {
	return append(dst, byte(i32), byte(i32>>8), byte(i32>>16), byte(i32>>24))
}

func appendCString(b []byte, str string) []byte {
	b = append(b, str...)
	return append(b, 0x00)
}

func readi32(src []byte) (int32, []byte, bool) {
	if len(src) < 4 {
		return 0, src, false
	}

	return (int32(src[0]) | int32(src[1])<<8 | int32(src[2])<<16 | int32(src[3])<<24), src[4:], true
}

func readi64(src []byte) (int64, []byte, bool) {
	if len(src) < 8 {
		return 0, src, false
	}
	i64 := (int64(src[0]) | int64(src[1])<<8 | int64(src[2])<<16 | int64(src[3])<<24 |
		int64(src[4])<<32 | int64(src[5])<<40 | int64(src[6])<<48 | int64(src[7])<<56)
	return i64, src[8:], true
}

func readcstring(src []byte) (string, []byte, bool) {
	idx := bytes.IndexByte(src, 0x00)
	if idx < 0 {
		return "", src, false
	}
	return string(src[:idx]), src[idx+1:], true
}
