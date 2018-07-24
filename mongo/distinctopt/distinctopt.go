// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package distinctopt

import (
	"reflect"
	"time"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

var distinctBundle = new(DistinctBundle)

// Distinct represents all passable params for the distinct() function.
type Distinct interface {
	distinct()
}

// DistinctOption represents the options for the distinct() function.
type DistinctOption interface {
	Distinct
	ConvertDistinctOption() option.DistinctOptioner
}

// DistinctSession is the session for the distinct() function
type DistinctSession interface {
	Distinct
	ConvertDistinctSession() *session.Client
}

// DistinctBundle is a bundle of Distinct options.
type DistinctBundle struct {
	option Distinct
	next   *DistinctBundle
}

func (db *DistinctBundle) distinct() {}

// ConvertDistinctOption implements the Distinct interface
func (db *DistinctBundle) ConvertDistinctOption() option.DistinctOptioner { return nil }

// BundleDistinct bundles Distinct options.
func BundleDistinct(opts ...Distinct) *DistinctBundle {
	head := distinctBundle

	for _, opt := range opts {
		newBundle := DistinctBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// Collation adds an option to specify a collation.
func (db *DistinctBundle) Collation(collation *mongoopt.Collation) *DistinctBundle {
	bundle := &DistinctBundle{
		option: Collation(collation),
		next:   db,
	}
	return bundle
}

// MaxTime adds an option to specify the maximum amount of time to allow the query to run.
func (db *DistinctBundle) MaxTime(d time.Duration) *DistinctBundle {
	bundle := &DistinctBundle{
		option: MaxTime(d),
		next:   db,
	}
	return bundle
}

// Unbundle transofrms a bundle into a slice of DistinctOptioner, optionally deduplicating.
func (db *DistinctBundle) Unbundle(deduplicate bool) ([]option.DistinctOptioner, *session.Client, error) {
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
func (db *DistinctBundle) bundleLength() int {
	if db == nil {
		return 0
	}

	bundleLen := 0
	for ; db != nil; db = db.next {
		if db.option == nil {
			continue
		}
		if converted, ok := db.option.(*DistinctBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		if _, ok := db.option.(DistinctSessionOpt); !ok {
			bundleLen++
		}
	}

	return bundleLen
}

// Helper that recursively unwraps bundle into slice of options
func (db *DistinctBundle) unbundle() ([]option.DistinctOptioner, *session.Client, error) {
	if db == nil {
		return nil, nil, nil
	}

	var sess *session.Client
	listLen := db.bundleLength()

	options := make([]option.DistinctOptioner, listLen)
	index := listLen - 1

	for listHead := db; listHead != nil; listHead = listHead.next {
		if listHead.option == nil {
			continue
		}

		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*DistinctBundle); ok {
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
		case DistinctOption:
			options[index] = t.ConvertDistinctOption()
			index--
		case DistinctSession:
			if sess == nil {
				sess = t.ConvertDistinctSession()
			}
		}
	}

	return options, sess, nil
}

// String implements the Stringer interface
func (db *DistinctBundle) String() string {
	if db == nil {
		return ""
	}

	str := ""
	for head := db; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*DistinctBundle); ok {
			str += converted.String()
			continue
		}

		if conv, ok := head.option.(DistinctOption); !ok {
			str += conv.ConvertDistinctOption().String() + "\n"
		}
	}

	return str
}

// Collation specifies a collation.
func Collation(collation *mongoopt.Collation) OptCollation {
	return OptCollation{
		Collation: collation.Convert(),
	}
}

// MaxTime adds an optin to specify the maximum amount of time to allow the query to run.
func MaxTime(d time.Duration) OptMaxTime {
	return OptMaxTime(d)
}

// OptCollation specifies a collation
type OptCollation option.OptCollation

func (OptCollation) distinct() {}

// ConvertDistinctOption implements the Distinct interface.
func (opt OptCollation) ConvertDistinctOption() option.DistinctOptioner {
	return option.OptCollation(opt)
}

// OptMaxTime specifies the maximum amount of time to allow the query to run.
type OptMaxTime option.OptMaxTime

func (OptMaxTime) distinct() {}

// ConvertDistinctOption implements the Distinct interface.
func (opt OptMaxTime) ConvertDistinctOption() option.DistinctOptioner {
	return option.OptMaxTime(opt)
}

// DistinctSessionOpt is an distinct session option.
type DistinctSessionOpt struct{}

func (DistinctSessionOpt) distinct() {}

// ConvertDistinctSession implements the DistinctSession interface.
func (DistinctSessionOpt) ConvertDistinctSession() *session.Client {
	return nil
}
