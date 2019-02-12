// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package wiremessage

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/bson"
)

func TestReply(t *testing.T) {
	t.Run("UnmarshalWireMessage", func(t *testing.T) {
		testCases := []struct {
			name string
			b    []byte
			r    Reply
			err  error
		}{
			{
				"success",
				[]byte{
					0x38, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00,
					0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x0A, 0x00, 0x00, 0x00, 0x0A, 'f', 'o', 'o', 0x00, 0x00,
					0x0A, 0x00, 0x00, 0x00, 0x0A, 'f', 'o', 'o', 0x00, 0x00,
				},
				Reply{
					MsgHeader: Header{
						MessageLength: 56,
					},
					CursorID: 256,
					Documents: []bson.Raw{
						{0x0A, 0x00, 0x00, 0x00, 0x0A, 'f', 'o', 'o', 0x00, 0x00},
						{0x0A, 0x00, 0x00, 0x00, 0x0A, 'f', 'o', 'o', 0x00, 0x00},
					},
				},
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var r Reply
				err := r.UnmarshalWireMessage(tc.b)
				if err != tc.err {
					t.Errorf("Errors do not match. got %v; want %v", err, tc.err)
				}
				if diff := cmp.Diff(r, tc.r); diff != "" {
					t.Errorf("Reply's differ: (-got +want)\n%s", diff)
				}
			})
		}
	})
}
