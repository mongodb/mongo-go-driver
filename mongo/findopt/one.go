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

var oneBundle = new(OneBundle)

// One is an interface for FindOne options
type One interface {
	one()
	ConvertFindOneOption() option.FindOptioner
}

// OneBundle is a bundle of FindOne options
type OneBundle struct {
	option One
	next   *OneBundle
}

// BundleOne bundles FindOne options
func BundleOne(opts ...One) *OneBundle {
	head := oneBundle

	for _, opt := range opts {
		newBundle := OneBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

func (ob *OneBundle) one() {}

// ConvertFindOneOption implements the FindOne interface
func (ob *OneBundle) ConvertFindOneOption() option.FindOptioner { return nil }

// AllowPartialResults adds an option to get partial results if some shards are down.
func (ob *OneBundle) AllowPartialResults(b bool) *OneBundle {
	bundle := &OneBundle{
		option: AllowPartialResults(b),
		next:   ob,
	}

	return bundle
}

// BatchSize adds an option to specify the number of documents to return in every batch.
func (ob *OneBundle) BatchSize(i int32) *OneBundle {
	bundle := &OneBundle{
		option: BatchSize(i),
		next:   ob,
	}

	return bundle
}

// Collation adds an option to specify a Collation.
func (ob *OneBundle) Collation(collation *mongoopt.Collation) *OneBundle {
	bundle := &OneBundle{
		option: Collation(collation),
		next:   ob,
	}

	return bundle
}

// Comment adds an option to specify a string to help trace the operation through teh database profiler, currentOp, and logs
func (ob *OneBundle) Comment(s string) *OneBundle {
	bundle := &OneBundle{
		option: Comment(s),
		next:   ob,
	}

	return bundle
}

// CursorType adds an option to specify the type of cursor to use.
func (ob *OneBundle) CursorType(ct mongoopt.CursorType) *OneBundle {
	bundle := &OneBundle{
		option: CursorType(ct),
		next:   ob,
	}

	return bundle
}

// Hint adds an option to specify the index to use.
func (ob *OneBundle) Hint(hint interface{}) *OneBundle {
	bundle := &OneBundle{
		option: Hint(hint),
		next:   ob,
	}

	return bundle
}

// Max adds an option to set an exclusive upper bound for a specific index.
func (ob *OneBundle) Max(max interface{}) *OneBundle {
	bundle := &OneBundle{
		option: Max(max),
		next:   ob,
	}

	return bundle
}

// MaxAwaitTime adds an option to specify the max amount of time for the server to wait on new documents.
func (ob *OneBundle) MaxAwaitTime(d time.Duration) *OneBundle {
	bundle := &OneBundle{
		option: MaxAwaitTime(d),
		next:   ob,
	}

	return bundle
}

// MaxScan adds an option to specify the number of documents or index keys to scan.
func (ob *OneBundle) MaxScan(i int64) *OneBundle {
	bundle := &OneBundle{
		option: MaxScan(i),
		next:   ob,
	}

	return bundle
}

// MaxTime adds an option to specify the max time to allow the query to run.
func (ob *OneBundle) MaxTime(d time.Duration) *OneBundle {
	bundle := &OneBundle{
		option: MaxTime(d),
		next:   ob,
	}

	return bundle
}

// Min adds an option to specify the inclusive lower bound for a specific index.
func (ob *OneBundle) Min(min interface{}) *OneBundle {
	bundle := &OneBundle{
		option: Min(min),
		next:   ob,
	}

	return bundle
}

// NoCursorTimeout adds an option to prevent cursors from timing out after an inactivity period.
func (ob *OneBundle) NoCursorTimeout(b bool) *OneBundle {
	bundle := &OneBundle{
		option: NoCursorTimeout(b),
		next:   ob,
	}

	return bundle
}

// OplogReplay adds an option for internal use only and should not be set.
func (ob *OneBundle) OplogReplay(b bool) *OneBundle {
	bundle := &OneBundle{
		option: OplogReplay(b),
		next:   ob,
	}

	return bundle
}

// Projection adds an option to limit the fields returned for all documents.
func (ob *OneBundle) Projection(projection interface{}) *OneBundle {
	bundle := &OneBundle{
		option: Projection(projection),
		next:   ob,
	}

	return bundle
}

// ReturnKey adds an option to only return index keys for all results.
func (ob *OneBundle) ReturnKey(b bool) *OneBundle {
	bundle := &OneBundle{
		option: ReturnKey(b),
		next:   ob,
	}

	return bundle
}

// ShowRecordID adds an option to determine whether to return the record identifier for each document.
func (ob *OneBundle) ShowRecordID(b bool) *OneBundle {
	bundle := &OneBundle{
		option: ShowRecordID(b),
		next:   ob,
	}

	return bundle
}

// Skip adds an option to specify the number of documents to skip before returning.
func (ob *OneBundle) Skip(i int64) *OneBundle {
	bundle := &OneBundle{
		option: Skip(i),
		next:   ob,
	}

	return bundle
}

// Snapshot adds an option to prevent the server from returning multiple copies because of an intervening
// write operation
func (ob *OneBundle) Snapshot(b bool) *OneBundle {
	bundle := &OneBundle{
		option: Snapshot(b),
		next:   ob,
	}

	return bundle
}

// Sort adds an option to specify the order in which to return results.
func (ob *OneBundle) Sort(sort interface{}) *OneBundle {
	bundle := &OneBundle{
		option: Sort(sort),
		next:   ob,
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
func (ob *OneBundle) Unbundle(deduplicate bool) ([]option.FindOptioner, error) {
	options, err := ob.unbundle()
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
func (ob *OneBundle) bundleLength() int {
	if ob == nil {
		return 0
	}

	bundleLen := 0
	for ; ob != nil && ob.option != nil; ob = ob.next {
		if converted, ok := ob.option.(*OneBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		bundleLen++
	}

	return bundleLen
}

// Helper that recursively unwraps bundle into slice of options
func (ob *OneBundle) unbundle() ([]option.FindOptioner, error) {
	if ob == nil {
		return nil, nil
	}

	listLen := ob.bundleLength()

	options := make([]option.FindOptioner, listLen)
	index := listLen - 1

	for listHead := ob; listHead != nil && listHead.option != nil; listHead = listHead.next {
		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*OneBundle); ok {
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

		options[index] = listHead.option.ConvertFindOneOption()
		index--
	}

	return options, nil
}

// String implements the Stringer interface
func (ob *OneBundle) String() string {
	if ob == nil {
		return ""
	}

	str := ""
	for head := ob; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*OneBundle); ok {
			str += converted.String()
			continue
		}

		str += head.option.ConvertFindOneOption().String() + "\n"
	}

	return str
}
