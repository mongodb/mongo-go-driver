package wiremessage

import (
	"bytes"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
)

// WireMessage represents a MongoDB wire message in binary form.
type WireMessage []byte

// OpCode represents a MongoDB wire protocol opcode.
type OpCode = wiremessage.OpCode

// AppendHeaderStart appends a header to the dst slice and returns an index where the wire message
// starts in dst and the updated slice.
func AppendHeaderStart(dst []byte, reqid, respto int32, opcode OpCode) (index int32, b []byte) {
	index, dst = bsoncore.ReserveLength(dst)
	dst = appendi32(dst, reqid)
	dst = appendi32(dst, respto)
	dst = appendi32(dst, int32(opcode))
	return index, dst
}

// AppendHeader appends a header to dst.
func AppendHeader(dst []byte, length, reqid, respto int32, opcode OpCode) []byte {
	dst = appendi32(dst, length)
	dst = appendi32(dst, reqid)
	dst = appendi32(dst, respto)
	dst = appendi32(dst, int32(opcode))
	return dst
}

// ReadHeader reads a wire message header from src.
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

// AppendQueryFlags appends the flags for an OP_QUERY wire message.
func AppendQueryFlags(dst []byte, flags wiremessage.QueryFlag) []byte {
	return appendi32(dst, int32(flags))
}

// AppendMsgFlags appends the flags for an OP_MSG wire message.
func AppendMsgFlags(dst []byte, flags wiremessage.MsgFlag) []byte {
	return appendi32(dst, int32(flags))
}

// AppendReplyFlags appends the flags for an OP_REPLY wire message.
func AppendReplyFlags(dst []byte, flags wiremessage.ReplyFlag) []byte {
	return appendi32(dst, int32(flags))
}

// AppendMsgSectionType appends the section type to dst.
func AppendMsgSectionType(dst []byte, stype wiremessage.SectionType) []byte {
	return append(dst, byte(stype))
}

// AppendQueryFullCollectionName appends the full collection name to dst.
func AppendQueryFullCollectionName(dst []byte, ns string) []byte {
	return appendCString(dst, ns)
}

// AppendQueryNumberToSkip appends the number to skip to dst.
func AppendQueryNumberToSkip(dst []byte, skip int32) []byte {
	return appendi32(dst, skip)
}

// AppendQueryNumberToReturn appends the number to return to dst.
func AppendQueryNumberToReturn(dst []byte, nor int32) []byte {
	return appendi32(dst, nor)
}

// AppendReplyCursorID appends the cursor ID to dst.
func AppendReplyCursorID(dst []byte, id int64) []byte {
	return appendi64(dst, id)
}

// AppendReplyStartingFrom appends the starting from field to dst.
func AppendReplyStartingFrom(dst []byte, sf int32) []byte {
	return appendi32(dst, sf)
}

// AppendReplyNumberReturned appends the number returned to dst.
func AppendReplyNumberReturned(dst []byte, nr int32) []byte {
	return appendi32(dst, nr)
}

// AppendCompressedOriginalOpCode appends the original opcode to dst.
func AppendCompressedOriginalOpCode(dst []byte, opcode wiremessage.OpCode) []byte {
	return appendi32(dst, int32(opcode))
}

// AppendCompressedUncompressedSize appends the uncompressed size of a
// compressed wiremessage to dst.
func AppendCompressedUncompressedSize(dst []byte, size int32) []byte { return appendi32(dst, size) }

// AppendCompressedCompressorID appends the ID of the compressor to dst.
func AppendCompressedCompressorID(dst []byte, id wiremessage.CompressorID) []byte {
	return append(dst, byte(id))
}

// AppendCompressedCompressedMessage appends the compressed wiremessage to dst.
func AppendCompressedCompressedMessage(dst []byte, msg []byte) []byte { return append(dst, msg...) }

// AppendGetMoreZero appends the zero field to dst.
func AppendGetMoreZero(dst []byte) []byte {
	return appendi32(dst, 0)
}

// AppendGetMoreFullCollectionName appends the fullCollectionName field to dst.
func AppendGetMoreFullCollectionName(dst []byte, ns string) []byte {
	return appendCString(dst, ns)
}

// AppendGetMoreNumberToReturn appends the numberToReturn field to dst.
func AppendGetMoreNumberToReturn(dst []byte, numToReturn int32) []byte {
	return appendi32(dst, numToReturn)
}

// AppendGetMoreCursorID appends the cursorID field to dst.
func AppendGetMoreCursorID(dst []byte, cursorID int64) []byte {
	return appendi64(dst, cursorID)
}

// AppendKillCursorsZero appends the zero field to dst.
func AppendKillCursorsZero(dst []byte) []byte {
	return appendi32(dst, 0)
}

// AppendKillCursorsNumberIDs appends the numberOfCursorIDs field to dst.
func AppendKillCursorsNumberIDs(dst []byte, numIDs int32) []byte {
	return appendi32(dst, numIDs)
}

// AppendKillCursorsCursorIDs appends each the cursorIDs field to dst.
func AppendKillCursorsCursorIDs(dst []byte, cursors []int64) []byte {
	for _, cursor := range cursors {
		dst = appendi64(dst, cursor)
	}
	return dst
}

// ReadMsgFlags reads the OP_MSG flags from src.
func ReadMsgFlags(src []byte) (flags wiremessage.MsgFlag, rem []byte, ok bool) {
	i32, rem, ok := readi32(src)
	return wiremessage.MsgFlag(i32), rem, ok
}

// IsMsgMoreToCome returns if the provided wire message is an OP_MSG with the more to come flag set.
func IsMsgMoreToCome(wm []byte) bool {
	return len(wm) >= 20 &&
		wiremessage.OpCode(readi32unsafe(wm[12:16])) == wiremessage.OpMsg &&
		wiremessage.MsgFlag(readi32unsafe(wm[16:20]))&wiremessage.MoreToCome == wiremessage.MoreToCome
}

// ReadMsgSectionType reads the section type from src.
func ReadMsgSectionType(src []byte) (stype wiremessage.SectionType, rem []byte, ok bool) {
	if len(src) < 1 {
		return 0, src, false
	}
	return wiremessage.SectionType(src[0]), src[1:], true
}

// ReadMsgSectionSingleDocument reads a single document from src.
func ReadMsgSectionSingleDocument(src []byte) (doc bsoncore.Document, rem []byte, ok bool) {
	return bsoncore.ReadDocument(src)
}

// ReadMsgSectionDocumentSequence reads an identifier and document sequence from src.
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

// ReadMsgChecksum reads a checksum from src.
func ReadMsgChecksum(src []byte) (checksum uint32, rem []byte, ok bool) {
	i32, rem, ok := readi32(src)
	return uint32(i32), rem, ok
}

// ReadQueryFlags reads OP_QUERY flags from src.
func ReadQueryFlags(src []byte) (flags wiremessage.QueryFlag, rem []byte, ok bool) {
	i32, rem, ok := readi32(src)
	return wiremessage.QueryFlag(i32), rem, ok
}

// ReadQueryFullCollectionName reads the full collection name from src.
func ReadQueryFullCollectionName(src []byte) (collname string, rem []byte, ok bool) {
	return readcstring(src)
}

// ReadQueryNumberToSkip reads the number to skip from src.
func ReadQueryNumberToSkip(src []byte) (nts int32, rem []byte, ok bool) {
	return readi32(src)
}

// ReadQueryNumberToReturn reads the number to return from src.
func ReadQueryNumberToReturn(src []byte) (ntr int32, rem []byte, ok bool) {
	return readi32(src)
}

// ReadQueryQuery reads the query from src.
func ReadQueryQuery(src []byte) (query bsoncore.Document, rem []byte, ok bool) {
	return bsoncore.ReadDocument(src)
}

// ReadQueryReturnFieldsSelector reads a return fields selector document from src.
func ReadQueryReturnFieldsSelector(src []byte) (rfs bsoncore.Document, rem []byte, ok bool) {
	return bsoncore.ReadDocument(src)
}

// ReadReplyFlags reads OP_REPLY flags from src.
func ReadReplyFlags(src []byte) (flags wiremessage.ReplyFlag, rem []byte, ok bool) {
	i32, rem, ok := readi32(src)
	return wiremessage.ReplyFlag(i32), rem, ok
}

// ReadReplyCursorID reads a cursor ID from src.
func ReadReplyCursorID(src []byte) (cursorID int64, rem []byte, ok bool) {
	return readi64(src)
}

// ReadReplyStartingFrom reads the starting from from src.
func ReadReplyStartingFrom(src []byte) (startingFrom int32, rem []byte, ok bool) {
	return readi32(src)
}

// ReadReplyNumberReturned reads the numbered returned from src.
func ReadReplyNumberReturned(src []byte) (numberReturned int32, rem []byte, ok bool) {
	return readi32(src)
}

// ReadReplyDocuments reads as many documents as possible from src
func ReadReplyDocuments(src []byte) (docs []bsoncore.Document, rem []byte, ok bool) {
	rem = src
	for {
		var doc bsoncore.Document
		doc, rem, ok = bsoncore.ReadDocument(rem)
		if !ok {
			break
		}

		docs = append(docs, doc)
	}

	return docs, rem, true
}

// ReadReplyDocument reads a reply document from src.
func ReadReplyDocument(src []byte) (doc bsoncore.Document, rem []byte, ok bool) {
	return bsoncore.ReadDocument(src)
}

// ReadCompressedOriginalOpCode reads the original opcode from src.
func ReadCompressedOriginalOpCode(src []byte) (opcode wiremessage.OpCode, rem []byte, ok bool) {
	i32, rem, ok := readi32(src)
	return wiremessage.OpCode(i32), rem, ok
}

// ReadCompressedUncompressedSize reads the uncompressed size of a
// compressed wiremessage to dst.
func ReadCompressedUncompressedSize(src []byte) (size int32, rem []byte, ok bool) { return readi32(src) }

// ReadCompressedCompressorID reads the ID of the compressor to dst.
func ReadCompressedCompressorID(src []byte) (id wiremessage.CompressorID, rem []byte, ok bool) {
	if len(src) < 1 {
		return 0, src, false
	}
	return wiremessage.CompressorID(src[0]), src[1:], true
}

// ReadCompressedCompressedMessage reads the compressed wiremessage to dst.
func ReadCompressedCompressedMessage(src []byte, length int32) (msg []byte, rem []byte, ok bool) {
	if len(src) < int(length) {
		return nil, src, false
	}
	return src[:length], src[length:], true
}

// ReadKillCursorsZero reads the zero field from src.
func ReadKillCursorsZero(src []byte) (zero int32, rem []byte, ok bool) {
	return readi32(src)
}

// ReadKillCursorsNumberIDs reads the numberOfCursorIDs field from src.
func ReadKillCursorsNumberIDs(src []byte) (numIDs int32, rem []byte, ok bool) {
	return readi32(src)
}

// ReadKillCursorsCursorIDs reads numIDs cursor IDs from src.
func ReadKillCursorsCursorIDs(src []byte, numIDs int32) (cursorIDs []int64, rem []byte, ok bool) {
	var i int32
	var id int64
	for i = 0; i < numIDs; i++ {
		id, src, ok = readi64(src)
		if !ok {
			return cursorIDs, src, false
		}

		cursorIDs = append(cursorIDs, id)
	}
	return cursorIDs, src, true
}

func appendi32(dst []byte, i32 int32) []byte {
	return append(dst, byte(i32), byte(i32>>8), byte(i32>>16), byte(i32>>24))
}

func appendi64(b []byte, i int64) []byte {
	return append(b, byte(i), byte(i>>8), byte(i>>16), byte(i>>24), byte(i>>32), byte(i>>40), byte(i>>48), byte(i>>56))
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

func readi32unsafe(src []byte) int32 {
	return (int32(src[0]) | int32(src[1])<<8 | int32(src[2])<<16 | int32(src[3])<<24)
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
