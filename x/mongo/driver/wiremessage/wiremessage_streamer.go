// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package wiremessage

import (
	"bufio"
	"bytes"
	"errors"
	"io"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// SrcStream ...
type SrcStream struct {
	io.LimitedReader

	Length          int32
	RequestID       int32
	ResponseTo      int32
	Opcode          OpCode
	IsMsgMoreToCome bool
}

type ayncReader struct {
	R io.Reader
	E chan error
}

// Read ...
func (r *ayncReader) Read(p []byte) (int, error) {
	n, err := r.R.Read(p)
	select {
	case err = <-r.E:
	default:
	}
	return n, err
}

// NewSrcStream ...
func NewSrcStream(r *io.LimitedReader) (*SrcStream, error) {
	src := make([]byte, 16)
	if r.N < int64(len(src)) {
		return nil, errIncompleteMsg
	}
	_, err := r.Read(src)
	if err != nil {
		return nil, err
	}
	var stream = new(SrcStream)
	stream.Length = readi32unsafe(src[0:4])
	stream.RequestID = readi32unsafe(src[4:8])
	stream.ResponseTo = readi32unsafe(src[8:12])
	stream.Opcode = OpCode(readi32unsafe(src[12:16]))

	bufreader := bufio.NewReader(r.R)
	src, err = bufreader.Peek(4)
	if err != nil {
		return nil, errIncompleteMsg
	}
	stream.IsMsgMoreToCome = stream.Opcode == OpMsg &&
		MsgFlag(readi32unsafe(src[0:4]))&MoreToCome == MoreToCome

	stream.R = bufreader
	stream.N = r.N

	if stream.Opcode != OpCompressed {
		return stream, nil
	}
	// get the original opcode and uncompressed size
	stream.Opcode, err = stream.ReadOpCode()
	if err != nil {
		return nil, errors.New("malformed OP_COMPRESSED: missing original opcode")
	}
	uncompressedSize, err := stream.ReadI32()
	if err != nil {
		return nil, errors.New("malformed OP_COMPRESSED: missing uncompressed size")
	}
	// get the compressor ID and decompress the message
	compressorID, err := stream.ReadCompressorID()
	if err != nil {
		return nil, errors.New("malformed OP_COMPRESSED: missing compressor ID")
	}
	compressedSize := stream.Length - 25 // header (16) + original opcode (4) + uncompressed size (4) + compressor ID (1)

	opts := CompressionOpts{
		Compressor:       compressorID,
		UncompressedSize: uncompressedSize,
	}
	uncompressed, err := NewDecompressedReader(io.LimitReader(stream.R, int64(compressedSize)), opts)
	if err != nil {
		return nil, err
	}

	piper, pipew := io.Pipe()
	errCh := make(chan error)
	stream.R = &ayncReader{piper, errCh}
	stream.Length = uncompressedSize + 16
	go func(w io.Writer, r io.Reader, errCh chan error) {
		_, err = io.Copy(w, r)
		if err != nil {
			errCh <- err
		}
		errCh <- nil
	}(pipew, uncompressed, errCh)

	return stream, nil
}

var errIncompleteMsg = errors.New("incomplete message")

// ReadByte ...
func (s *SrcStream) ReadByte() (byte, error) {
	b := make([]byte, 1)
	_, err := s.Read(b)
	return b[0], err
}

// ReadSlice ...
func (s *SrcStream) ReadSlice(delim byte) ([]byte, error) {
	buf := bytes.Buffer{}
	var err error
	for {
		var b byte
		b, err = s.ReadByte()
		if err != nil || b == delim {
			break
		}
		buf.WriteByte(b)
	}
	return buf.Bytes(), err
}

// ReadI32 ...
func (s *SrcStream) ReadI32() (int32, error) {
	src := make([]byte, 4)
	if s.N < int64(len(src)) {
		return 0, errIncompleteMsg
	}
	_, err := s.Read(src)
	if err != nil {
		return 0, err
	}

	return readi32unsafe(src), nil
}

// ReadI64 ...
func (s *SrcStream) ReadI64() (int64, error) {
	src := make([]byte, 8)
	if s.N < int64(len(src)) {
		return 0, errIncompleteMsg
	}
	_, err := s.Read(src)
	if err != nil {
		return 0, err
	}

	return readi64unsafe(src), nil
}

// ReadMsgFlags reads the OP_MSG flags from src.
func (s *SrcStream) ReadMsgFlags() (MsgFlag, error) {
	i32, err := s.ReadI32()
	return MsgFlag(i32), err
}

// ReadOpCode reads an opcode from src.
func (s *SrcStream) ReadOpCode() (OpCode, error) {
	i32, err := s.ReadI32()
	return OpCode(i32), err
}

// ReadReplyFlags reads OP_REPLY flags from src.
func (s *SrcStream) ReadReplyFlags() (ReplyFlag, error) {
	i32, err := s.ReadI32()
	return ReplyFlag(i32), err
}

// ReadMsgSectionType reads the section type from src.
func (s *SrcStream) ReadMsgSectionType() (SectionType, error) {
	b, err := s.ReadByte()
	return SectionType(b), err
}

// ReadCompressorID reads the ID of the compressor to dst.
func (s *SrcStream) ReadCompressorID() (CompressorID, error) {
	b, err := s.ReadByte()
	return CompressorID(b), err
}

// ReadMsgSectionSingleDocument reads a single document from src.
func (s *SrcStream) ReadMsgSectionSingleDocument() (bsoncore.Document, error) {
	length := s.N
	buf := make([]byte, 4)
	if length < int64(len(buf)) {
		return nil, errIncompleteMsg
	}
	_, err := s.Read(buf)
	if err != nil {
		return nil, errIncompleteMsg
	}
	l := readi32unsafe(buf)
	if length < int64(l) {
		return nil, errIncompleteMsg
	}
	doc := make([]byte, l)
	n := copy(doc, buf)
	_, err = s.Read(doc[n:])
	return doc, err
}

// ReadMsgSectionDocumentSequence reads an identifier and document sequence from src and returns the document sequence
// data parsed into a slice of BSON documents.
func (s *SrcStream) ReadMsgSectionDocumentSequence() (identifier string, docs []bsoncore.Document, err error) {
	length, err := s.ReadI32()
	if err != nil {
		return "", nil, err
	} else if int64(length) > s.N {
		return "", nil, errIncompleteMsg
	}

	s.N -= 4
	defer func() {
		s.N += 4
	}()

	slice, err := s.ReadSlice(0x00)
	if err != nil {
		return "", nil, err
	}
	identifier = string(slice)

	docs, err = s.ReadReplyDocuments()
	if s.N > 0 {
		return "", nil, errIncompleteMsg
	}

	return identifier, docs, err
}

// ReadReplyDocuments reads as many documents as possible from src
func (s *SrcStream) ReadReplyDocuments() ([]bsoncore.Document, error) {
	var docs []bsoncore.Document
	for s.N > 0 {
		length := s.N

		buf := make([]byte, 4)
		if length < int64(len(buf)) {
			return nil, errIncompleteMsg
		}
		_, err := s.Read(buf)
		if err != nil {
			return nil, errIncompleteMsg
		}
		l := readi32unsafe(buf)
		if length < int64(l) {
			return nil, errIncompleteMsg
		}
		doc := make([]byte, l)
		n := copy(doc, buf)
		_, err = s.Read(doc[n:])
		if err != nil {
			return nil, err
		}

		docs = append(docs, doc)
	}

	return docs, nil
}
