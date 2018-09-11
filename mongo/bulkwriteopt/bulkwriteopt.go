// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bulkwriteopt

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/core/session"
)

var bulkWriteBundle = new(BulkWriteBundle)

// BulkWriteOptions represents all the bulk write options.
type BulkWriteOptions struct {
	Ordered                     bool
	OrderedSet                  bool
	BypassDocumentValidation    bool
	BypassDocumentValidationSet bool
}

// BulkWrite represents all passable params for the bulkWrite() function.
type BulkWrite interface {
	bulkWrite()
}

// optionFunc adds the option to the bulkwrite options
type optionFunc func(*BulkWriteOptions)

// BulkWriteSession is the session for the bulkWrite() function.
type BulkWriteSession interface {
	BulkWrite
	ConvertBulkWriteSession() *session.Client
}

// BulkWriteBundle is a bundle of BulkWrite options.
type BulkWriteBundle struct {
	option BulkWrite
	next   *BulkWriteBundle
}

// Implement the BulkWrite interface.
func (bwb *BulkWriteBundle) bulkWrite() {}

// optionFunc implements bulkWrite.
func (optionFunc) bulkWrite() {}

// BundleBulkWrite bundles BulkWrite options
func BundleBulkWrite(opts ...BulkWrite) *BulkWriteBundle {
	head := bulkWriteBundle

	for _, opt := range opts {
		newBundle := BulkWriteBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// BypassDocumentValidation adds an option to allow the write to opt-out of document-level validation.
func (bwb *BulkWriteBundle) BypassDocumentValidation(b bool) *BulkWriteBundle {
	bundle := &BulkWriteBundle{
		option: BypassDocumentValidation(b),
		next:   bwb,
	}

	return bundle
}

// Ordered specifies whether later writes should be attempted if an earlier one fails. If false, when a write fails,
// the operation attempts later writes.
func (bwb *BulkWriteBundle) Ordered(b bool) *BulkWriteBundle {
	bundle := &BulkWriteBundle{
		option: Ordered(b),
		next:   bwb,
	}

	return bundle
}

// Unbundle transforms a bundle into a BulkWriteOptions object.
func (bwb *BulkWriteBundle) Unbundle() (*BulkWriteOptions, *session.Client, error) {
	bwo := &BulkWriteOptions{
		Ordered: true, // defaults to true
	}
	sess, err := bwb.unbundle(bwo)
	if err != nil {
		return nil, nil, err
	}

	return bwo, sess, nil
}

// Helper that populates the BulkWriteOptions object and returns a session, if present.
func (bwb *BulkWriteBundle) unbundle(bwo *BulkWriteOptions) (*session.Client, error) {
	var sess *session.Client

	for head := bwb; head != nil; head = head.next {
		var err error
		switch opt := head.option.(type) {
		case *BulkWriteBundle:
			s, err := opt.unbundle(bwo) // add all bundle's options to options
			if s != nil && sess == nil {
				sess = s
			}
			if err != nil {
				return nil, err
			}
		case optionFunc:
			opt(bwo) // add option to options
		case BulkWriteSession:
			if sess == nil {
				sess = opt.ConvertBulkWriteSession()
			}
		}
		if err != nil {
			return nil, err
		}
	}

	return sess, nil
}

// String implements the Stringer interface.
func (bwb *BulkWriteBundle) String() string {
	if bwb == nil {
		return ""
	}

	str := ""
	for head := bwb; head != nil && head.option != nil; head = head.next {
		switch opt := head.option.(type) {
		case *BulkWriteBundle:
			str = opt.String() + str
		case optionFunc:
			str = reflect.TypeOf(opt).String() + "\n" + str
		}
	}

	return str
}

// BypassDocumentValidation allows the write to opt-out of document-level validation.
func BypassDocumentValidation(b bool) BulkWrite {
	return optionFunc(
		func(bwo *BulkWriteOptions) {
			if !bwo.BypassDocumentValidationSet {
				bwo.BypassDocumentValidation = b
				bwo.BypassDocumentValidationSet = true
			}
		})
}

// Ordered allows the write to opt-out of document-level validation.
func Ordered(b bool) BulkWrite {
	return optionFunc(
		func(bwo *BulkWriteOptions) {
			if !bwo.OrderedSet {
				bwo.Ordered = b
				bwo.OrderedSet = true
			}
		})
}

// BulkWriteSessionOpt is an aggregate session option.
type BulkWriteSessionOpt struct{}

func (BulkWriteSessionOpt) bulkWrite() {}

// ConvertBulkWriteSession implements the BulkWriteSession interface.
func (BulkWriteSessionOpt) ConvertBulkWriteSession() *session.Client {
	return nil
}
