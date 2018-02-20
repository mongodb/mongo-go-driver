// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"bytes"
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
)

type singleResultCursor struct {
	err error
	rdr bson.Reader
}

func (s *singleResultCursor) ID() int64 {
	return -1
}

// Next implements the Cursor interface.
func (s *singleResultCursor) Next(context.Context) bool {
	if len(s.rdr) == 0 {
		return false
	}

	return true
}

// Decode implements the Cursor interface.
func (s *singleResultCursor) Decode(v interface{}) error {
	br, err := s.DecodeBytes()
	if err != nil {
		return err
	}

	return bson.NewDecoder(bytes.NewReader(br)).Decode(v)
}

// DecodeBytes implements the Cursor interface.
func (s *singleResultCursor) DecodeBytes() (bson.Reader, error) {
	if s.err != nil {
		return nil, s.err
	}

	if len(s.rdr) == 0 {
		return nil, nil
	}

	return s.rdr, nil
}

// Err implements the Cursor interface.
func (s *singleResultCursor) Err() error {
	return s.err
}

// Close implements the Cursor interface.
func (s *singleResultCursor) Close(context.Context) error {
	return nil
}
