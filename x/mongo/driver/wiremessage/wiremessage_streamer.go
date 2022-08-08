// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package wiremessage

import (
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

	headerBuf [12]byte
}

// NewSrcStream ...
func NewSrcStream(r io.Reader) (*SrcStream, error) {
	stream := &SrcStream{}
	err := stream.Reset(r)
	return stream, err
}

// Reset ...
func (s *SrcStream) Reset(r io.Reader) error {
	var err error
	_, err = io.ReadFull(r, s.headerBuf[:])
	if err != nil {
		return errors.New("incomplete read of message header")
	}
	s.ReadCloser = ioutil.NopCloser(r)
	s.RequestID = readi32unsafe(s.headerBuf[0:4])
	s.ResponseTo = readi32unsafe(s.headerBuf[4:8])
	s.Opcode = OpCode(readi32unsafe(s.headerBuf[8:12]))

	if s.Opcode != OpCompressed {
		return nil
	}
	// get the original opcode and uncompressed size
	s.Opcode, err = s.ReadOpCode()
	if err != nil {
		return errors.New("malformed OP_COMPRESSED: missing original opcode")
	}
	uncompressedSize, err := s.ReadI32()
	if err != nil {
		return errors.New("malformed OP_COMPRESSED: missing uncompressed size")
	}
	// get the compressor ID and decompress the message
	compressorID, err := s.ReadCompressorID()
	if err != nil {
		return errors.New("malformed OP_COMPRESSED: missing compressor ID")
	}

	opts := CompressionOpts{
		Compressor:       compressorID,
		UncompressedSize: uncompressedSize,
	}
	uncompressed, err := NewDecompressedReader(s.ReadCloser, opts)
	if err != nil {
		return err
	}

	piper, pipew := io.Pipe()
	s.ReadCloser = piper
	go func(w *io.PipeWriter, r io.ReadCloser) {
		var err error
		defer func() {
			err = r.Close()
			err = w.CloseWithError(err)
		}()
		_, err = io.Copy(w, r)
	}(pipew, uncompressed)

	return nil
}

// ReadByte ...
func (s *SrcStream) ReadByte() (byte, error) {
	_, err := io.ReadFull(s, s.headerBuf[:1])
	return s.headerBuf[0], err
}

// ReadI32 ...
func (s *SrcStream) ReadI32() (int32, error) {
	_, err := io.ReadFull(s, s.headerBuf[:4])
	if err != nil {
		return 0, err
	}

	return readi32unsafe(s.headerBuf[:4]), nil
}

// ReadI64 ...
func (s *SrcStream) ReadI64() (int64, error) {
	_, err := io.ReadFull(s, s.headerBuf[:8])
	if err != nil {
		return 0, err
	}

	return readi64unsafe(s.headerBuf[:8]), nil
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
	_, err := io.ReadFull(s, s.headerBuf[:4])
	if err != nil {
		return nil, err
	}
	l := readi32unsafe(s.headerBuf[:4])
	doc := make([]byte, l)
	n := copy(doc, s.headerBuf[:4])
	_, err = io.ReadFull(s, doc[n:])
	return doc, err
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
