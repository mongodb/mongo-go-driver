package mnet

// Pinner represents a Connection that can be pinned by one or more cursors or
// transactions. Implementations of this interface should maintain the following
// invariants:
//
//  1. Each Pin* call should increment the number of references for the
//     connection.
//  2. Each Unpin* call should decrement the number of references for the
//     connection.
//  3. Calls to Close() should be ignored until all resources have unpinned the
//     connection.
type Pinner interface {
	PinToCursor() error
	PinToTransaction() error
	UnpinFromCursor() error
	UnpinFromTransaction() error
}

type defaultPinner struct{}

// Assert defaultPinner implements Pinner at compile-time.
var _ Pinner = &defaultPinner{}

func (*defaultPinner) PinToCursor() error          { return nil }
func (*defaultPinner) PinToTransaction() error     { return nil }
func (*defaultPinner) UnpinFromCursor() error      { return nil }
func (*defaultPinner) UnpinFromTransaction() error { return nil }
