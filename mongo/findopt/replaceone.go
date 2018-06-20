// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package findopt

import (
	"time"

	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

var replaceOneBundle = new(ReplaceOneBundle)

// ReplaceOne is an interface for FindOneAndReplace options
type ReplaceOne interface {
	replaceOne()
	ConvertReplaceOneOption() option.FindOneAndReplaceOptioner
}

// ReplaceOneBundle is a bundle of FindOneAndReplace options
type ReplaceOneBundle struct {
	option ReplaceOne
	next   *ReplaceOneBundle
}

// BundleReplaceOne bundles FindOneAndReplace options
func BundleReplaceOne(opts ...ReplaceOne) *ReplaceOneBundle {
	head := replaceOneBundle

	for _, opt := range opts {
		newBundle := ReplaceOneBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

func (rob *ReplaceOneBundle) replaceOne() {}

// ConvertReplaceOneOption implements ReplaceOne interface
func (rob *ReplaceOneBundle) ConvertReplaceOneOption() option.FindOneAndReplaceOptioner { return nil }

// BypassDocumentValidation adds an option to allow the write to opt-out of document-level validation.
func (rob *ReplaceOneBundle) BypassDocumentValidation(b bool) *ReplaceOneBundle {
	bundle := &ReplaceOneBundle{
		option: BypassDocumentValidation(b),
		next:   rob,
	}

	return bundle
}

// Collation adds an option to specify a Collation.
func (rob *ReplaceOneBundle) Collation(collation *mongoopt.Collation) *ReplaceOneBundle {
	bundle := &ReplaceOneBundle{
		option: Collation(collation),
		next:   rob,
	}

	return bundle
}

// MaxTime adds an option to specify the max time to allow the query to run.
func (rob *ReplaceOneBundle) MaxTime(d time.Duration) *ReplaceOneBundle {
	bundle := &ReplaceOneBundle{
		option: MaxTime(d),
		next:   rob,
	}

	return bundle
}

// Projection adds an option to limit the fields returned for all documents.
func (rob *ReplaceOneBundle) Projection(projection interface{}) *ReplaceOneBundle {
	bundle := &ReplaceOneBundle{
		option: Projection(projection),
		next:   rob,
	}

	return bundle
}

// ReturnDocument adds an option to specify whether to return the updated or original document.
func (rob *ReplaceOneBundle) ReturnDocument(rd mongoopt.ReturnDocument) *ReplaceOneBundle {
	bundle := &ReplaceOneBundle{
		option: ReturnDocument(rd),
		next:   rob,
	}

	return bundle
}

// Sort adds an option to specify the order in which to return results.
func (rob *ReplaceOneBundle) Sort(sort interface{}) *ReplaceOneBundle {
	bundle := &ReplaceOneBundle{
		option: Sort(sort),
		next:   rob,
	}

	return bundle
}

// Upsert adds an option to specify whether to create a new document if no document matches the query.
func (rob *ReplaceOneBundle) Upsert(b bool) *ReplaceOneBundle {
	bundle := &ReplaceOneBundle{
		option: Upsert(b),
		next:   rob,
	}

	return bundle
}

// Calculates the total length of a bundle, accounting for nested bundles.
func (rob *ReplaceOneBundle) bundleLength() int {
	if rob == nil {
		return 0
	}

	bundleLen := 0
	for ; rob != nil && rob.option != nil; rob = rob.next {
		if converted, ok := rob.option.(*ReplaceOneBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		bundleLen++
	}

	return bundleLen
}

// Unbundle unwinds and deduplicates the options used to create it and those
// added after creation into a single slice of options.
//
// The deduplicate parameter is used to determine if the bundle is just flattened or
// if we actually deduplicate options.
//
// Since a FindBundle can be recursive, this method will unwind all recursive FindBundles.
func (rob *ReplaceOneBundle) Unbundle(deduplicate bool) ([]option.FindOneAndReplaceOptioner, error) {
	options, err := rob.unbundle()
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

// Helper that recursively unwraps bundle into slice of options
func (rob *ReplaceOneBundle) unbundle() ([]option.FindOneAndReplaceOptioner, error) {
	if rob == nil {
		return nil, nil
	}

	listLen := rob.bundleLength()

	options := make([]option.FindOneAndReplaceOptioner, listLen)
	index := listLen - 1

	for listHead := rob; listHead != nil && listHead.option != nil; listHead = listHead.next {
		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*ReplaceOneBundle); ok {
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

		options[index] = listHead.option.ConvertReplaceOneOption()
		index--
	}

	return options, nil
}

// String implements the Stringer interface
func (rob *ReplaceOneBundle) String() string {
	if rob == nil {
		return ""
	}

	str := ""
	for head := rob; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*ReplaceOneBundle); ok {
			str += converted.String()
			continue
		}

		str += head.option.ConvertReplaceOneOption().String() + "\n"
	}

	return str
}
