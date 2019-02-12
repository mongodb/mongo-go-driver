// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package wiremessage

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/bson"
)

func TestQuery(t *testing.T) {
	t.Run("AppendWireMessage", func(t *testing.T) {
		testCases := []struct {
			name string
			q    Query
			res  []byte
			err  error
		}{
			{
				"success",
				Query{
					MsgHeader:          Header{},
					FullCollectionName: "foo.bar",
					Query:              bson.Raw{0x05, 0x00, 0x00, 0x00, 0x00},
				},
				[]byte{
					0x29, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0xD4, 0x07, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f, 0x2e,
					0x62, 0x61, 0x72, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00,
					0x00,
				},
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				res := make([]byte, 0)
				res, err := tc.q.AppendWireMessage(res)
				if err != tc.err {
					t.Errorf("Did not get expected error. got %v; want %v", err, tc.err)
				}
				if !bytes.Equal(res, tc.res) {
					t.Errorf("Results do not match. got %#v; want %#v", res, tc.res)
				}
			})
		}
	})
	t.Run("UnmarshalWireMessage", func(t *testing.T) {
		testCases := []struct {
			name string
			req  []byte
			q    Query
			err  error
		}{
			{
				"success",
				[]byte{
					0x29, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0xD4, 0x07, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f, 0x2e,
					0x62, 0x61, 0x72, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00,
					0x00,
				},
				Query{
					MsgHeader: Header{
						MessageLength: 41,
						OpCode:        OpQuery,
					},
					FullCollectionName: "foo.bar",
					Query:              bson.Raw{0x05, 0x00, 0x00, 0x00, 0x00},
				},
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var q Query
				err := q.UnmarshalWireMessage(tc.req)
				if err != tc.err {
					t.Errorf("Did not get expected error. got %v; want %v", err, tc.err)
				}
				if diff := cmp.Diff(q, tc.q); diff != "" {
					t.Errorf("Results do not match. (-got +want):\n%s", diff)
				}
			})
		}
	})
}
