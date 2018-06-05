package mongo

import "context"

type Cursor interface {
	Next(context.Context) bool
	Decode(interface{}) error
	Close(context.Context) error
	Err() error
}
