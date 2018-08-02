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
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

var updateOneBundle = new(UpdateOneBundle)

// UpdateOne represents all passable params for the updateOne() function.
type UpdateOne interface {
	updateOne()
}

// UpdateOneOption represents the options for the updateOne() function.
type UpdateOneOption interface {
	UpdateOne
	ConvertUpdateOneOption() option.FindOneAndUpdateOptioner
}

// UpdateOneSession is the session for the updateOne() function
type UpdateOneSession interface {
	UpdateOne
	ConvertUpdateOneSession() *session.Client
}

// UpdateOneBundle is a bundle of FindOneAndUpdate options
type UpdateOneBundle struct {
	option UpdateOne
	next   *UpdateOneBundle
}

// BundleUpdateOne bundles FindOneAndUpdate options
func BundleUpdateOne(opts ...UpdateOne) *UpdateOneBundle {
	head := updateOneBundle

	for _, opt := range opts {
		newBundle := UpdateOneBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

func (uob *UpdateOneBundle) updateOne() {}

// ConvertUpdateOneOption implements the UpdateOne interface
func (uob *UpdateOneBundle) ConvertUpdateOneOption() option.FindOneAndUpdateOptioner { return nil }

// ArrayFilters adds an option to specify which array elements an update should apply.
func (uob *UpdateOneBundle) ArrayFilters(filters ...interface{}) *UpdateOneBundle {
	bundle := &UpdateOneBundle{
		option: ArrayFilters(filters),
		next:   uob,
	}

	return bundle
}

// BypassDocumentValidation adds an option to allow the write to opt-out of document-level validation
func (uob *UpdateOneBundle) BypassDocumentValidation(b bool) *UpdateOneBundle {
	bundle := &UpdateOneBundle{
		option: BypassDocumentValidation(b),
		next:   uob,
	}

	return bundle
}

// Collation adds an option to specify a collation.
func (uob *UpdateOneBundle) Collation(collation *mongoopt.Collation) *UpdateOneBundle {
	bundle := &UpdateOneBundle{
		option: Collation(collation),
		next:   uob,
	}

	return bundle
}

// MaxTime adds an option to specify the max time to allow the query to run.
func (uob *UpdateOneBundle) MaxTime(d time.Duration) *UpdateOneBundle {
	bundle := &UpdateOneBundle{
		option: MaxTime(d),
		next:   uob,
	}

	return bundle
}

// Projection adds an option to limit the fields returned for all documents.
func (uob *UpdateOneBundle) Projection(projection interface{}) *UpdateOneBundle {
	bundle := &UpdateOneBundle{
		option: Projection(projection),
		next:   uob,
	}

	return bundle
}

// ReturnDocument adds an option to specify whether to return the updated or original document.
func (uob *UpdateOneBundle) ReturnDocument(rd mongoopt.ReturnDocument) *UpdateOneBundle {
	bundle := &UpdateOneBundle{
		option: ReturnDocument(rd),
		next:   uob,
	}

	return bundle
}

// Sort adds an option to specify the order in which to return results.
func (uob *UpdateOneBundle) Sort(sort interface{}) *UpdateOneBundle {
	bundle := &UpdateOneBundle{
		option: Sort(sort),
		next:   uob,
	}

	return bundle
}

// Upsert adds an option to specify whether to create a new document if no document matches the query.
func (uob *UpdateOneBundle) Upsert(b bool) *UpdateOneBundle {
	bundle := &UpdateOneBundle{
		option: Upsert(b),
		next:   uob,
	}

	return bundle
}

// Calculates the total length of a bundle, accounting for nested bundles.
func (uob *UpdateOneBundle) bundleLength() int {
	if uob == nil {
		return 0
	}

	bundleLen := 0
	for ; uob != nil; uob = uob.next {
		if uob.option == nil {
			continue
		}
		if converted, ok := uob.option.(*UpdateOneBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		if _, ok := uob.option.(FindSessionOpt); !ok {
			bundleLen++
		}
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
func (uob *UpdateOneBundle) Unbundle(deduplicate bool) ([]option.FindOneAndUpdateOptioner, *session.Client, error) {
	options, sess, err := uob.unbundle()
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

// Helper that recursively unwraps bundle into slice of options
func (uob *UpdateOneBundle) unbundle() ([]option.FindOneAndUpdateOptioner, *session.Client, error) {
	if uob == nil {
		return nil, nil, nil
	}

	var sess *session.Client
	listLen := uob.bundleLength()

	options := make([]option.FindOneAndUpdateOptioner, listLen)
	index := listLen - 1

	for listHead := uob; listHead != nil; listHead = listHead.next {
		if listHead.option == nil {
			continue
		}

		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*UpdateOneBundle); ok {
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
		case UpdateOneOption:
			options[index] = t.ConvertUpdateOneOption()
			index--
		case UpdateOneSession:
			if sess == nil {
				sess = t.ConvertUpdateOneSession()
			}
		}
	}

	return options, sess, nil
}

// String implements the Stringer interface
func (uob *UpdateOneBundle) String() string {
	if uob == nil {
		return ""
	}

	str := ""
	for head := uob; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*UpdateOneBundle); ok {
			str += converted.String()
			continue
		}

		if conv, ok := head.option.(UpdateOneOption); !ok {
			str += conv.ConvertUpdateOneOption().String() + "\n"
		}
	}

	return str
}
