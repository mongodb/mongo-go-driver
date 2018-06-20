// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package findopt

import (
	"reflect"
	"time"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

var deleteOneBundle = new(DeleteOneBundle)

// DeleteOne is an interface for FindOneAndDelete options
type DeleteOne interface {
	deleteOne()
	ConvertDeleteOneOption() option.FindOneAndDeleteOptioner
}

// DeleteOneBundle is a bundle of FindOneAndDelete options
type DeleteOneBundle struct {
	option DeleteOne
	next   *DeleteOneBundle
}

// BundleDeleteOne bundles FindOneAndDelete options
func BundleDeleteOne(opts ...DeleteOne) *DeleteOneBundle {
	head := deleteOneBundle

	for _, opt := range opts {
		newBundle := DeleteOneBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

func (dob *DeleteOneBundle) deleteOne() {}

// ConvertDeleteOneOption implements the DeleteOne interface
func (dob *DeleteOneBundle) ConvertDeleteOneOption() option.FindOneAndDeleteOptioner { return nil }

// Collation adds an option to specify a Collation
func (dob *DeleteOneBundle) Collation(collation *mongoopt.Collation) *DeleteOneBundle {
	bundle := &DeleteOneBundle{
		option: Collation(collation),
		next:   dob,
	}

	return bundle
}

// MaxTime adds an option to specify the max time to allow the query to run.
func (dob *DeleteOneBundle) MaxTime(d time.Duration) *DeleteOneBundle {
	bundle := &DeleteOneBundle{
		option: MaxTime(d),
		next:   dob,
	}

	return bundle
}

// Projection adds an option to limit the fields returned for all documents.
func (dob *DeleteOneBundle) Projection(projection interface{}) *DeleteOneBundle {
	bundle := &DeleteOneBundle{
		option: Projection(projection),
		next:   dob,
	}

	return bundle
}

// Sort adds an option to specify the order in which to return results.
func (dob *DeleteOneBundle) Sort(sort interface{}) *DeleteOneBundle {
	bundle := &DeleteOneBundle{
		option: Sort(sort),
		next:   dob,
	}

	return bundle
}

// Unbundle unwinds and deduplicates the options used to create it and those
// added after creation into a single slice of options.
//
// The deduplicate parameter is used to determine if the bundle is just flattened or
// if we actually deduplicate options.
//
// Since a FindBundle can be recursive, this method will unwind all recursive FindBundles.
func (dob *DeleteOneBundle) Unbundle(deduplicate bool) ([]option.FindOneAndDeleteOptioner, error) {
	options, err := dob.unbundle()
	if err != nil {
		return nil, err
	}

	if !deduplicate {
		return options, nil
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

	return options, nil
}

// Calculates the total length of a bundle, accounting for nested bundles.
func (dob *DeleteOneBundle) bundleLength() int {
	if dob == nil {
		return 0
	}

	bundleLen := 0
	for ; dob != nil && dob.option != nil; dob = dob.next {
		if converted, ok := dob.option.(*DeleteOneBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		bundleLen++
	}

	return bundleLen
}

// Helper that recursively unwraps bundle into slice of options
func (dob *DeleteOneBundle) unbundle() ([]option.FindOneAndDeleteOptioner, error) {
	if dob == nil {
		return nil, nil
	}

	listLen := dob.bundleLength()

	options := make([]option.FindOneAndDeleteOptioner, listLen)
	index := listLen - 1

	for listHead := dob; listHead != nil && listHead.option != nil; listHead = listHead.next {
		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*DeleteOneBundle); ok {
			nestedOptions, err := converted.unbundle()
			if err != nil {
				return nil, err
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

		options[index] = listHead.option.ConvertDeleteOneOption()
		index--
	}

	return options, nil
}

// String implements the Stringer interface
func (dob *DeleteOneBundle) String() string {
	if dob == nil {
		return ""
	}

	str := ""
	for head := dob; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*DeleteOneBundle); ok {
			str += converted.String()
			continue
		}

		str += head.option.ConvertDeleteOneOption().String() + "\n"
	}

	return str
}
