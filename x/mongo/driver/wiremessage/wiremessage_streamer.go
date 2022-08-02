// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package wiremessage

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// SrcStream ...
type SrcStream struct {
	io.ReadCloser

	RequestID  int32
	ResponseTo int32
	Opcode     OpCode
}

// NewSrcStream ...
func NewSrcStream(r io.Reader) (*SrcStream, error) {
	var err error
	var headerBuf [12]byte
	_, err = io.ReadFull(r, headerBuf[:])
	if err != nil {
		return nil, errors.New("incomplete read of message header")
	}
	stream := &SrcStream{
		ioutil.NopCloser(r),
		readi32unsafe(headerBuf[0:4]),
		readi32unsafe(headerBuf[4:8]),
		OpCode(readi32unsafe(headerBuf[8:12])),
	}

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

	opts := CompressionOpts{
		Compressor:       compressorID,
		UncompressedSize: uncompressedSize,
	}
	uncompressed, err := NewDecompressedReader(stream.ReadCloser, opts)
	if err != nil {
		return nil, err
	}

	piper, pipew := io.Pipe()
	stream.ReadCloser = piper
	go func(w *io.PipeWriter, r io.ReadCloser) {
		var err error
		defer func() {
			err = r.Close()
			err = w.CloseWithError(err)
		}()
		_, err = io.Copy(w, r)
	}(pipew, uncompressed)

	return stream, nil
}

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
	for err == nil {
		var b byte
		b, err = s.ReadByte()
		if err == nil {
			buf.WriteByte(b)
			if b == delim {
				break
			}
		}
	}
	return buf.Bytes(), err
}

// ReadI32 ...
func (s *SrcStream) ReadI32() (int32, error) {
	src := make([]byte, 4)
	_, err := io.ReadFull(s, src)
	if err != nil {
		return 0, err
	}

	return readi32unsafe(src), nil
}

// ReadI64 ...
func (s *SrcStream) ReadI64() (int64, error) {
	src := make([]byte, 8)
	_, err := io.ReadFull(s, src)
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
	buf := make([]byte, 4)
	_, err := s.Read(buf)
	if err != nil {
		return nil, err
	}
	l := readi32unsafe(buf)
	doc := make([]byte, l)
	n := copy(doc, buf)
	_, err = io.ReadFull(s, doc[n:])
	return doc, err
}

// ReadMsgSectionDocumentSequence reads an identifier and document sequence from src and returns the document sequence
// data parsed into a slice of BSON documents.
func (s *SrcStream) ReadMsgSectionDocumentSequence() (identifier string, docs []bsoncore.Document, err error) {
	_, err = s.ReadI32()
	if err != nil {
		return "", nil, err
	}

	slice, err := s.ReadSlice(0x00)
	if err != nil {
		return "", nil, err
	}
	identifier = string(slice[:len(slice)-1])

	docs, err = s.ReadReplyDocuments()

	return identifier, docs, err
}

// ReadReplyDocuments reads as many documents as possible from src
func (s *SrcStream) ReadReplyDocuments() ([]bsoncore.Document, error) {
	var docs []bsoncore.Document
	var err error
	for {
		var doc bsoncore.Document
		doc, err = s.ReadMsgSectionSingleDocument()
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}
		docs = append(docs, doc)
	}

	return docs, err
}
