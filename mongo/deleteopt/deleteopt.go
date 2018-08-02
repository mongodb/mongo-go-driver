// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package deleteopt

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

var deleteBundle = new(DeleteBundle)

// Delete represents all passable params for the delete() function.
type Delete interface {
	delete()
}

// DeleteOption represents the options for the delete() function.
type DeleteOption interface {
	Delete
	ConvertDeleteOption() option.DeleteOptioner
}

// DeleteSession is the session for the delete() function
type DeleteSession interface {
	Delete
	ConvertDeleteSession() *session.Client
}

// DeleteBundle is a bundle of Delete options
type DeleteBundle struct {
	option Delete
	next   *DeleteBundle
}

func (db *DeleteBundle) delete() {}

// ConvertDeleteOption implements the Delete interface.
func (db *DeleteBundle) ConvertDeleteOption() option.DeleteOptioner {
	return nil
}

// BundleDelete bundles Delete options.
func BundleDelete(opts ...Delete) *DeleteBundle {
	head := deleteBundle

	for _, opt := range opts {
		newBundle := DeleteBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// Collation adds an option to specify a collation.
func (db *DeleteBundle) Collation(c *mongoopt.Collation) *DeleteBundle {
	bundle := &DeleteBundle{
		option: Collation(c),
		next:   db,
	}

	return bundle
}

// Unbundle transforms a bundle into a slice of options, optionally deduplicating
func (db *DeleteBundle) Unbundle(deduplicate bool) ([]option.DeleteOptioner, *session.Client, error) {

	options, sess, err := db.unbundle()
	if err != nil {
		return nil, nil, err
	}

	if !deduplicate {
		return options, sess, nil
	}

	// iterate backwards and make dedup slice
	optionsSet := make(map[reflect.Type]struct{})

	for i := len(options) - 1; i >= 0; i-- {
		currOption := options[i]
		optionType := reflect.TypeOf(currOption)

		if _, ok := optionsSet[optionType]; ok {
			// option already found
			options = append(options[:i], options[i+1:]...)
			continue
		}

		optionsSet[optionType] = struct{}{}
	}

	return options, sess, nil
}

// Calculates the total length of a bundle, accounting for nested bundles.
func (db *DeleteBundle) bundleLength() int {
	if db == nil {
		return 0
	}

	bundleLen := 0
	for ; db != nil; db = db.next {
		if db.option == nil {
			continue
		}
		if converted, ok := db.option.(*DeleteBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		if _, ok := db.option.(DeleteSessionOpt); !ok {
			bundleLen++
		}
	}

	return bundleLen
}

// Helper that recursively unwraps bundle into slice of options
func (db *DeleteBundle) unbundle() ([]option.DeleteOptioner, *session.Client, error) {
	if db == nil {
		return nil, nil, nil
	}

	var sess *session.Client
	listLen := db.bundleLength()

	options := make([]option.DeleteOptioner, listLen)
	index := listLen - 1

	for listHead := db; listHead != nil; listHead = listHead.next {
		if listHead.option == nil {
			continue
		}

		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*DeleteBundle); ok {
			nestedOptions, s, err := converted.unbundle()
			if err != nil {
				return nil, nil, err
			}
			if s != nil && sess == nil {
				sess = s
			}

			// where to start inserting nested options
			startIndex := index - len(nestedOptions) + 1

			// add nested options in order
			for _, nestedOp := range nestedOptions {
				options[startIndex] = nestedOp
				startIndex++
			}
			index -= len(nestedOptions)
			continue
		}

		switch t := listHead.option.(type) {
		case DeleteOption:
			options[index] = t.ConvertDeleteOption()
			index--
		case DeleteSession:
			if sess == nil {
				sess = t.ConvertDeleteSession()
			}
		}
	}

	return options, sess, nil
}

// String implements the Stringer interface
func (db *DeleteBundle) String() string {
	if db == nil {
		return ""
	}

	str := ""
	for head := db; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*DeleteBundle); ok {
			str += converted.String()
			continue
		}

		if conv, ok := head.option.(DeleteOption); !ok {
			str += conv.ConvertDeleteOption().String() + "\n"
		}
	}

	return str
}

// Collation specifies a collation.
func Collation(c *mongoopt.Collation) OptCollation {
	return OptCollation{Collation: c.Convert()}
}

// OptCollation specifies a collation.
type OptCollation option.OptCollation

func (OptCollation) delete() {}

// ConvertDeleteOption implements the Delete interface.
func (opt OptCollation) ConvertDeleteOption() option.DeleteOptioner {
	return option.OptCollation(opt)
}

// DeleteSessionOpt is an delete session option.
type DeleteSessionOpt struct{}

func (DeleteSessionOpt) delete() {}

// ConvertDeleteSession implements the DeleteSession interface.
func (DeleteSessionOpt) ConvertDeleteSession() *session.Client {
	return nil
}
