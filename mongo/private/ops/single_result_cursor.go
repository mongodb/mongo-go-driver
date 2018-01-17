package ops

import (
	"bytes"
	"context"

	"github.com/skriptble/wilson/bson"
)

type singleResultCursor struct {
	err error
	rdr bson.Reader
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
	if s.err != nil {
		return s.err
	}

	if len(s.rdr) == 0 {
		return nil
	}

	return bson.NewDecoder(bytes.NewReader(s.rdr)).Decode(v)
}

// Err implements the Cursor interface.
func (s *singleResultCursor) Err() error {
	return s.err
}

// Close implements the Cursor interface.
func (s *singleResultCursor) Close(context.Context) error {
	return nil
}
