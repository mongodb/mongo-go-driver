// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package collectionopt

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

var collectionBundle = new(CollectionBundle)

// BSONAppender is an interface implemented by types that can marshal a
// provided type into BSON bytes and append those bytes to the provided []byte.
// The AppendBSON can return a non-nil error and non-nil []byte. The AppendBSON
// method may also write incomplete BSON to the []byte.
type BSONAppender interface {
	AppendBSON([]byte, interface{}) ([]byte, error)
}

// Option represents a collection option.
type Option interface {
	collectionOption()
}

// optionFunc adds the option to the client.
type optionFunc func(*Collection) error

// Collection represents a collection.
type Collection struct {
	ReadConcern    *readconcern.ReadConcern
	WriteConcern   *writeconcern.WriteConcern
	ReadPreference *readpref.ReadPref
	BSONAppender   BSONAppender
}

// CollectionBundle is a bundle of collection options.
type CollectionBundle struct {
	option Option
	next   *CollectionBundle
}

// CollectionBundle implements Option.
func (*CollectionBundle) collectionOption() {}

// OptionFunc implements Option.
func (optionFunc) collectionOption() {}

// BundleCollection bundles collection options.
func BundleCollection(opts ...Option) *CollectionBundle {
	head := collectionBundle

	for _, opt := range opts {
		newBundle := CollectionBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// ReadConcern sets the read concern.
func (cb *CollectionBundle) ReadConcern(rc *readconcern.ReadConcern) *CollectionBundle {
	return &CollectionBundle{
		option: ReadConcern(rc),
		next:   cb,
	}
}

// WriteConcern sets the write concern.
func (cb *CollectionBundle) WriteConcern(wc *writeconcern.WriteConcern) *CollectionBundle {
	return &CollectionBundle{
		option: WriteConcern(wc),
		next:   cb,
	}
}

// ReadPreference sets the read preference.
func (cb *CollectionBundle) ReadPreference(rp *readpref.ReadPref) *CollectionBundle {
	return &CollectionBundle{
		option: ReadPreference(rp),
		next:   cb,
	}
}

// String prints a string representation of the bundle for debug purposes
func (cb *CollectionBundle) String() string {
	if cb == nil {
		return ""
	}

	debugStr := ""
	for head := cb; head != nil && head.option != nil; head = head.next {
		switch opt := head.option.(type) {
		case *CollectionBundle:
			debugStr += opt.String()
		case optionFunc:
			debugStr += reflect.TypeOf(opt).String() + "\n"
		default:
			return debugStr + "(error: CollectionOption can only be *CollectionBundle or optionFunc)"
		}
	}

	return debugStr
}

// Unbundle unbundles the options, returning a collection.
func (cb *CollectionBundle) Unbundle() (*Collection, error) {
	client := &Collection{}
	err := cb.unbundle(client)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Helper that recursively unwraps the bundle.
func (cb *CollectionBundle) unbundle(client *Collection) error {
	if cb == nil {
		return nil
	}

	for head := cb; head != nil && head.option != nil; head = head.next {
		var err error
		switch opt := head.option.(type) {
		case *CollectionBundle:
			err = opt.unbundle(client) // add all bundle's options to client
		case optionFunc:
			err = opt(client) // add option to client
		default:
			return nil
		}
		if err != nil {
			return err
		}
	}

	return nil

}

// ReadConcern sets the read concern.
func ReadConcern(rc *readconcern.ReadConcern) Option {
	return optionFunc(
		func(c *Collection) error {
			if c.ReadConcern == nil {
				c.ReadConcern = rc
			}
			return nil
		})
}

// WriteConcern sets the write concern.
func WriteConcern(wc *writeconcern.WriteConcern) Option {
	return optionFunc(
		func(c *Collection) error {
			if c.WriteConcern == nil {
				c.WriteConcern = wc
			}
			return nil
		})
}

// ReadPreference sets the read preference.
func ReadPreference(rp *readpref.ReadPref) Option {
	return optionFunc(
		func(c *Collection) error {
			if c.ReadPreference == nil {
				c.ReadPreference = rp
			}
			return nil
		})
}
