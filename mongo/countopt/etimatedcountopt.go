// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package countopt

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/session"
)

var estimatedDocumentCountBundle = new(EstimatedDocumentCountBundle)

//EstimatedDocumentCount represents all passable params for the estimatedDocumentCount() function
type EstimatedDocumentCount interface {
	estimatedCount()
}

// EstimatedDocumentCountOption is options for the estimatedDocumentCount() function
type EstimatedDocumentCountOption interface {
	EstimatedDocumentCount
	ConvertEstimateDocumentCountOption() option.CountOptioner
}

// EstimatedDocumentCountSession is the session for the estimatedDocumentCount() function
type EstimatedDocumentCountSession interface {
	EstimatedDocumentCount
	ConvertEstimateDocumentCountSession() *session.Client
}

// EstimatedDocumentCountBundle is a bundle of EstimatedDocumentCount options
type EstimatedDocumentCountBundle struct {
	option EstimatedDocumentCount
	next   *EstimatedDocumentCountBundle
}

// Implement the EstimateDocumentCount interface
func (cb *EstimatedDocumentCountBundle) estimatedCount() {}

// ConvertEstimateDocumentCountOption implements the EstimatedDocumentCount interface
func (cb *EstimatedDocumentCountBundle) ConvertEstimateDocumentCountOption() option.CountOptioner {
	return nil
}

// BundleEstimatedDocumentCount bundles EstimatedDocumentCount options
func BundleEstimatedDocumentCount(opts ...EstimatedDocumentCount) *EstimatedDocumentCountBundle {
	head := estimatedDocumentCountBundle

	for _, opt := range opts {
		newBundle := EstimatedDocumentCountBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// Unbundle transforms a bundle into a slice of options, optionally deduplicating.
func (cb *EstimatedDocumentCountBundle) Unbundle(deduplicate bool) ([]option.CountOptioner, *session.Client, error) {
	options, sess, err := cb.unbundle()
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

	return options, nil, nil
}

// Helper that recursively unwraps bundle into slice of options
func (cb *EstimatedDocumentCountBundle) unbundle() ([]option.CountOptioner, *session.Client, error) {
	if cb == nil {
		return nil, nil, nil
	}
	var sess *session.Client
	listLen := cb.bundleLength()

	options := make([]option.CountOptioner, listLen)
	index := listLen - 1

	for listHead := cb; listHead != nil && listHead.option != nil; listHead = listHead.next {
		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*EstimatedDocumentCountBundle); ok {
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
		case EstimatedDocumentCountOption:
			options[index] = t.ConvertEstimateDocumentCountOption()
			index--
		case EstimatedDocumentCountSession:
			if sess == nil {
				sess = t.ConvertEstimateDocumentCountSession()
			}
		}
	}

	return options, sess, nil
}

// Calculates the total length of a bundle, accounting for nested bundles.
func (cb *EstimatedDocumentCountBundle) bundleLength() int {
	if cb == nil {
		return 0
	}

	bundleLen := 0
	for ; cb != nil && cb.option != nil; cb = cb.next {
		if converted, ok := cb.option.(*EstimatedDocumentCountBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		bundleLen++
	}

	return bundleLen
}

// MaxTimeMs adds an option to specify the maximum amount of time to allow the operation to run.
func (cb *EstimatedDocumentCountBundle) MaxTimeMs(i int32) *EstimatedDocumentCountBundle {
	bundle := &EstimatedDocumentCountBundle{
		option: MaxTimeMs(i),
		next:   cb,
	}

	return bundle
}
