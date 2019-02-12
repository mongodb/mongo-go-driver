// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package wiremessage

import (
	"bytes"
	"testing"

	"go.mongodb.org/mongo-driver/x/bsonx"
)

var doc = bsonx.Doc{{"x", bsonx.Int32(5)}}

func oneSection(t *testing.T) []Section {
	rdr, err := doc.MarshalBSON()
	if err != nil {
		t.Errorf("error marshaling document: %s\n", err)
		return nil
	}

	arr := []Section{
		SectionBody{
			PayloadType: SingleDocument,
			Document:    rdr,
		},
	}

	return arr
}

func sectionBytes(t *testing.T) []byte {
	rdr, err := doc.MarshalBSON()
	if err != nil {
		t.Errorf("error marshaling document: %s\n", err)
		return nil
	}

	buf := make([]byte, 0)
	totalLen := 16 + 4 + 1 + len(rdr) // header + flags + payloadType + doc
	buf = appendInt32(buf, int32(totalLen))

	// append requestId, responseTo
	for i := 0; i < 8; i++ {
		buf = append(buf, 0x00)
	}

	buf = append(buf, 0xdd, 0x07, 0x00, 0x00) // opcode = OP_MSG = 2013
	buf = append(buf, 0x00, 0x00, 0x00, 0x00) // flags

	// append section
	buf = append(buf, 0x00)   // document type
	buf = append(buf, rdr...) // section document

	return buf
}

func TestMsg(t *testing.T) {
	t.Run("AppendWireMessage", func(t *testing.T) {
		testCases := []struct {
			name string
			m    Msg
			res  []byte
			err  error
		}{
			{
				"Success",
				Msg{
					MsgHeader: Header{},
					FlagBits:  0,
					Sections:  make([]Section, 0),
				},
				[]byte{
					0x14, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0xDD, 0x07, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00,
				},
				nil,
			},
			{
				"OneSection",
				Msg{
					MsgHeader: Header{},
					FlagBits:  0,
					Sections:  oneSection(t),
				},
				sectionBytes(t),
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				res := make([]byte, 0)
				res, err := tc.m.AppendWireMessage(res)
				if err != tc.err {
					t.Errorf("Did not get expected error. got %v; want %v", err, tc.err)
				}

				if !bytes.Equal(res, tc.res) {
					t.Errorf("Results do not match. got %#v; want %#v", res, tc.res)
				}
			})
		}
	})
}
