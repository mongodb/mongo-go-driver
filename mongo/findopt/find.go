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
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

var findBundle = new(FindBundle)

// Find represents all passable params for the find() function.
type Find interface {
	find()
}

// FindOption represents the options for the find() function.
type FindOption interface {
	Find
	ConvertFindOption() option.FindOptioner
}

// FindSession is the session for the find() function
type FindSession interface {
	Find
	ConvertFindSession() *session.Client
}

// FindBundle is a bundle of Find options
type FindBundle struct {
	option Find
	next   *FindBundle
}

// BundleFind bundles Find options
func BundleFind(opts ...Find) *FindBundle {
	head := findBundle

	for _, opt := range opts {
		newBundle := FindBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

func (fb *FindBundle) find() {}

// ConvertFindOption implements the Find interface
func (fb *FindBundle) ConvertFindOption() option.FindOptioner { return nil }

// AllowPartialResults adds an option to get partial results if some shards are down.
func (fb *FindBundle) AllowPartialResults(b bool) *FindBundle {
	bundle := &FindBundle{
		option: AllowPartialResults(b),
		next:   fb,
	}

	return bundle
}

// BatchSize adds an option to specify the number of documents to return in every batch.
func (fb *FindBundle) BatchSize(i int32) *FindBundle {
	bundle := &FindBundle{
		option: BatchSize(i),
		next:   fb,
	}

	return bundle
}

// Collation adds an option to specify a Collation
func (fb *FindBundle) Collation(collation *mongoopt.Collation) *FindBundle {
	bundle := &FindBundle{
		option: Collation(collation),
		next:   fb,
	}

	return bundle
}

// Comment adds an option to specify a string to help trace the operation through the database profiler, currentOp, and logs.
func (fb *FindBundle) Comment(s string) *FindBundle {
	bundle := &FindBundle{
		option: Comment(s),
		next:   fb,
	}

	return bundle
}

// CursorType adds an option to specify the type of cursor to use.
func (fb *FindBundle) CursorType(ct mongoopt.CursorType) *FindBundle {
	bundle := &FindBundle{
		option: CursorType(ct),
		next:   fb,
	}

	return bundle
}

// Hint adds an option to specify the index to use.
func (fb *FindBundle) Hint(hint interface{}) *FindBundle {
	bundle := &FindBundle{
		option: Hint(hint),
		next:   fb,
	}

	return bundle
}

// Limit adds an option to set the limit on the number of results.
func (fb *FindBundle) Limit(i int64) *FindBundle {
	bundle := &FindBundle{
		option: Limit(i),
		next:   fb,
	}

	return bundle
}

// Max adds an option to set an exclusive upper bound for a specific index.
func (fb *FindBundle) Max(max interface{}) *FindBundle {
	bundle := &FindBundle{
		option: Max(max),
		next:   fb,
	}

	return bundle
}

// MaxAwaitTime adds an option to specify the max amount of time for the server to wait on new documents.
func (fb *FindBundle) MaxAwaitTime(d time.Duration) *FindBundle {
	bundle := &FindBundle{
		option: MaxAwaitTime(d),
		next:   fb,
	}

	return bundle
}

// MaxScan adds an option to specify the max number of documents or index keys to scan.
func (fb *FindBundle) MaxScan(i int64) *FindBundle {
	bundle := &FindBundle{
		option: MaxScan(i),
		next:   fb,
	}

	return bundle
}

// MaxTime adds an option to specify the max time to allow the query to run.
func (fb *FindBundle) MaxTime(d time.Duration) *FindBundle {
	bundle := &FindBundle{
		option: MaxTime(d),
		next:   fb,
	}

	return bundle
}

// Min adds an option to specify the inclusive lower bound for a specific index.
func (fb *FindBundle) Min(min interface{}) *FindBundle {
	bundle := &FindBundle{
		option: Min(min),
		next:   fb,
	}

	return bundle
}

// NoCursorTimeout adds an option to prevent cursors from timing out after an inactivity period.
func (fb *FindBundle) NoCursorTimeout(b bool) *FindBundle {
	bundle := &FindBundle{
		option: NoCursorTimeout(b),
		next:   fb,
	}

	return bundle
}

// OplogReplay adds an option for internal use only and should not be set.
func (fb *FindBundle) OplogReplay(b bool) *FindBundle {
	bundle := &FindBundle{
		option: OplogReplay(b),
		next:   fb,
	}

	return bundle
}

// Projection adds an option to limit the fields returned for all documents.
func (fb *FindBundle) Projection(projection interface{}) *FindBundle {
	bundle := &FindBundle{
		option: Projection(projection),
		next:   fb,
	}

	return bundle
}

// ReturnKey adds an option to only return index keys for all result documents.
func (fb *FindBundle) ReturnKey(b bool) *FindBundle {
	bundle := &FindBundle{
		option: ReturnKey(b),
		next:   fb,
	}

	return bundle
}

// ShowRecordID sets an option to determine whether to return the record identifier for each document.
func (fb *FindBundle) ShowRecordID(b bool) *FindBundle {
	bundle := &FindBundle{
		option: ShowRecordID(b),
		next:   fb,
	}

	return bundle
}

// Skip adds an option to specify the number of documents to skip before returning.
func (fb *FindBundle) Skip(i int64) *FindBundle {
	bundle := &FindBundle{
		option: Skip(i),
		next:   fb,
	}

	return bundle
}

// Snapshot adds an option to prevent the cursor from returning a document more than once because of an
// intervening write operation.
func (fb *FindBundle) Snapshot(b bool) *FindBundle {
	bundle := &FindBundle{
		option: Snapshot(b),
		next:   fb,
	}

	return bundle
}

// Sort adds an option to specify the order in which to return results.
func (fb *FindBundle) Sort(sort interface{}) *FindBundle {
	bundle := &FindBundle{
		option: Sort(sort),
		next:   fb,
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
func (fb *FindBundle) Unbundle(deduplicate bool) ([]option.FindOptioner, *session.Client, error) {
	options, sess, err := fb.unbundle()
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
func (fb *FindBundle) bundleLength() int {
	if fb == nil {
		return 0
	}

	bundleLen := 0
	for ; fb != nil; fb = fb.next {
		if fb.option == nil {
			continue
		}
		if converted, ok := fb.option.(*FindBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		if _, ok := fb.option.(FindSessionOpt); !ok {
			bundleLen++
		}
	}

	return bundleLen
}

// Helper that recursively unwraps bundle into slice of options
func (fb *FindBundle) unbundle() ([]option.FindOptioner, *session.Client, error) {
	if fb == nil {
		return nil, nil, nil
	}

	var sess *session.Client
	listLen := fb.bundleLength()

	options := make([]option.FindOptioner, listLen)
	index := listLen - 1

	for listHead := fb; listHead != nil; listHead = listHead.next {
		if listHead.option == nil {
			continue
		}

		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*FindBundle); ok {
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
		case FindOption:
			options[index] = t.ConvertFindOption()
			index--
		case FindSession:
			if sess == nil {
				sess = t.ConvertFindSession()
			}
		}
	}

	return options, sess, nil

}

// String implements the Stringer interface
func (fb *FindBundle) String() string {
	if fb == nil {
		return ""
	}

	str := ""
	for head := fb; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*FindBundle); ok {
			str += converted.String()
			continue
		}

		if conv, ok := head.option.(FindOption); !ok {
			str += conv.ConvertFindOption().String() + "\n"
		}
	}

	return str
}
