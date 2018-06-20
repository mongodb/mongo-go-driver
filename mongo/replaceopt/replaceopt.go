// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package replaceopt

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

var replaceBundle = new(ReplaceBundle)

// Replace is options for the replace() function
type Replace interface {
	replace()
	ConvertReplaceOption() option.ReplaceOptioner
}

// ReplaceBundle is a bundle of Replace options
type ReplaceBundle struct {
	option Replace
	next   *ReplaceBundle
}

// Implement the Replace interface
func (rb *ReplaceBundle) replace() {}

// ConvertReplaceOption implements the Replace interface
func (rb *ReplaceBundle) ConvertReplaceOption() option.ReplaceOptioner { return nil }

// BundleReplace bundles Replace options
func BundleReplace(opts ...Replace) *ReplaceBundle {
	head := replaceBundle

	for _, opt := range opts {
		newBundle := ReplaceBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// BypassDocumentValidation adds an option to allow the write to opt-out of document-level validation.
func (rb *ReplaceBundle) BypassDocumentValidation(b bool) *ReplaceBundle {
	bundle := &ReplaceBundle{
		option: BypassDocumentValidation(b),
		next:   rb,
	}

	return bundle
}

// Collation adds an option to specify a Collation.
func (rb *ReplaceBundle) Collation(c *mongoopt.Collation) *ReplaceBundle {
	bundle := &ReplaceBundle{
		option: Collation(c),
		next:   rb,
	}

	return bundle
}

// Upsert adds an option to specify whether to insert a new document if it does not exist
func (rb *ReplaceBundle) Upsert(b bool) *ReplaceBundle {
	bundle := &ReplaceBundle{
		option: Upsert(b),
		next:   rb,
	}

	return bundle
}

// String implements the Stringer interface.
func (rb *ReplaceBundle) String() string {
	if rb == nil {
		return ""
	}

	str := ""
	for head := rb; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*ReplaceBundle); ok {
			str += converted.String()
			continue
		}

		str += head.option.ConvertReplaceOption().String() + "\n"
	}

	return str
}

// Calculates the total length of a bundle, accounting for nested bundles.
func (rb *ReplaceBundle) bundleLength() int {
	if rb == nil {
		return 0
	}

	bundleLen := 0
	for ; rb != nil && rb.option != nil; rb = rb.next {
		if converted, ok := rb.option.(*ReplaceBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		bundleLen++
	}

	return bundleLen
}

// Unbundle transforms a bundle into a slice of options, optionally deduplicating
func (rb *ReplaceBundle) Unbundle(deduplicate bool) ([]option.ReplaceOptioner, error) {

	options, err := rb.unbundle()
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
func (rb *ReplaceBundle) unbundle() ([]option.ReplaceOptioner, error) {
	if rb == nil {
		return nil, nil
	}

	listLen := rb.bundleLength()

	options := make([]option.ReplaceOptioner, listLen)
	index := listLen - 1

	for listHead := rb; listHead != nil && listHead.option != nil; listHead = listHead.next {
		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*ReplaceBundle); ok {
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

		options[index] = listHead.option.ConvertReplaceOption()
		index--
	}

	return options, nil

}

// BypassDocumentValidation allows the write to opt-out of document-level validation.
func BypassDocumentValidation(b bool) OptBypassDocumentValidation {
	return OptBypassDocumentValidation(b)
}

// Collation specifies a Collation.
func Collation(c *mongoopt.Collation) OptCollation {
	return OptCollation{Collation: c.Convert()}
}

// Upsert specifies whether to insert a new document if it does not exist
func Upsert(b bool) OptUpsert {
	return OptUpsert(b)
}

// OptBypassDocumentValidation allows the write to opt-out of document-level validation.
type OptBypassDocumentValidation option.OptBypassDocumentValidation

func (OptBypassDocumentValidation) replace() {}

// ConvertReplaceOption implements the Replace interface
func (opt OptBypassDocumentValidation) ConvertReplaceOption() option.ReplaceOptioner {
	return option.OptBypassDocumentValidation(opt)
}

// OptCollation specifies a Collation.
type OptCollation option.OptCollation

func (OptCollation) replace() {}

// ConvertReplaceOption implements the replace interface
func (opt OptCollation) ConvertReplaceOption() option.ReplaceOptioner {
	return option.OptCollation(opt)
}

// OptUpsert specifies whether to insert a new document if it does not exist
type OptUpsert option.OptUpsert

func (OptUpsert) replace() {}

// ConvertReplaceOption implements the Replace interface
func (opt OptUpsert) ConvertReplaceOption() option.ReplaceOptioner {
	return option.OptUpsert(opt)
}
