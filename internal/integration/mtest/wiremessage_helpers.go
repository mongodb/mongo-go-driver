// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mtest

import (
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

func copyBytes(original []byte) []byte {
	newSlice := make([]byte, len(original))
	copy(newSlice, original)
	return newSlice
}

// assertMsgSectionType asserts that the next section type in the OP_MSG wire message is equal to the provided type.
// It returns the remainder of the wire message and an error if the section type could not be read or was not equal
// to the expected type.
func assertMsgSectionType(wm []byte, expected wiremessage.SectionType) ([]byte, error) {
	var actual wiremessage.SectionType
	var ok bool

	actual, wm, ok = wiremessage.ReadMsgSectionType(wm)
	if !ok {
		return wm, errors.New("failed to read section type")
	}
	if expected != actual {
		return wm, fmt.Errorf("unexpected section type %v; expected %v", actual, expected)
	}
	return wm, nil
}

func parseOpCompressed(wm []byte) (wiremessage.OpCode, []byte, error) {
	// Store the original opcode to forward to another parser later.
	originalOpcode, wm, ok := wiremessage.ReadCompressedOriginalOpCode(wm)
	if !ok {
		return originalOpcode, nil, errors.New("failed to read original opcode")
	}

	uncompressedSize, wm, ok := wiremessage.ReadCompressedUncompressedSize(wm)
	if !ok {
		return originalOpcode, nil, errors.New("failed to read uncompressed size")
	}

	compressorID, wm, ok := wiremessage.ReadCompressedCompressorID(wm)
	if !ok {
		return originalOpcode, nil, errors.New("failed to read compressor ID")
	}

	compressedMsg, _, ok := wiremessage.ReadCompressedCompressedMessage(wm, int32(len(wm)))
	if !ok {
		return originalOpcode, nil, errors.New("failed to read compressed message")
	}

	opts := driver.CompressionOpts{
		Compressor:       compressorID,
		UncompressedSize: uncompressedSize,
	}
	decompressed, err := driver.DecompressPayload(compressedMsg, opts)
	if err != nil {
		return originalOpcode, nil, fmt.Errorf("error decompressing payload: %w", err)
	}

	return originalOpcode, decompressed, nil
}
