// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package insertopt

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

var insertOneBundle = new(InsertOneBundle)
var insertManyBundle = new(InsertManyBundle)

// InsertOne is options for InsertInsertOne
type InsertOne interface {
	insertOne()
	ConvertInsertOneOption() option.InsertOptioner
}

// InsertMany is options for InsertInsertMany
type InsertMany interface {
	insertMany()
	ConvertInsertManyOption() option.InsertOptioner
}

// InsertOneBundle is a bundle of InsertOne options
type InsertOneBundle struct {
	option InsertOne
	next   *InsertOneBundle
}

// Implement the InsertOne interface
func (ob *InsertOneBundle) insertOne() {}

// ConvertInsertOneOption implements the InsertOne interface
func (ob *InsertOneBundle) ConvertInsertOneOption() option.InsertOptioner { return nil }

// BundleOne bundles InsertOne options
func BundleOne(opts ...InsertOne) *InsertOneBundle {
	head := insertOneBundle

	for _, opt := range opts {
		newBundle := InsertOneBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// BypassDocumentValidation adds an option allowing the write to opt-out of the document-level validation.
func (ob *InsertOneBundle) BypassDocumentValidation(b bool) *InsertOneBundle {
	bundle := &InsertOneBundle{
		option: BypassDocumentValidation(b),
		next:   ob,
	}

	return bundle
}

// WriteConcern adds an option to specify a write concern
func (ob *InsertOneBundle) WriteConcern(wc *writeconcern.WriteConcern) *InsertOneBundle {
	bundle := &InsertOneBundle{
		option: WriteConcern(wc),
		next:   ob,
	}

	return bundle
}

// Calculates the total length of a bundle, accounting for nested bundles.
func (ob *InsertOneBundle) bundleLength() int {
	if ob == nil {
		return 0
	}

	bundleLen := 0
	for ; ob != nil && ob.option != nil; ob = ob.next {
		if converted, ok := ob.option.(*InsertOneBundle); ok {
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
// Since a bundle can be recursive, this method will unwind all recursive bundles.
func (ob *InsertOneBundle) Unbundle(deduplicate bool) ([]option.InsertOptioner, error) {
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

// Helper that recursively unwraps bundle into slice of options
func (ob *InsertOneBundle) unbundle() ([]option.InsertOptioner, error) {
	if ob == nil {
		return nil, nil
	}

	listLen := ob.bundleLength()

	options := make([]option.InsertOptioner, listLen)
	index := listLen - 1

	for listHead := ob; listHead != nil && listHead.option != nil; listHead = listHead.next {
		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*InsertOneBundle); ok {
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

		options[index] = listHead.option.ConvertInsertOneOption()
		index--
	}

	return options, nil
}

// String implements the Stringer interface
func (ob *InsertOneBundle) String() string {
	if ob == nil {
		return ""
	}

	str := ""
	for head := ob; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*InsertOneBundle); ok {
			str += converted.String()
			continue
		}

		str += head.option.ConvertInsertOneOption().String() + "\n"
	}

	return str
}

// InsertManyBundle is a bundle of InsertInsertMany options
type InsertManyBundle struct {
	option InsertMany
	next   *InsertManyBundle
}

// BundleMany bundles InsertMany options
func BundleMany(opts ...InsertMany) *InsertManyBundle {
	head := insertManyBundle

	for _, opt := range opts {
		newBundle := InsertManyBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// Implement the InsertMany interface
func (mb *InsertManyBundle) insertMany() {}

// ConvertInsertManyOption implements the InsertMany interface
func (mb *InsertManyBundle) ConvertInsertManyOption() option.InsertOptioner { return nil }

// BypassDocumentValidation adds an option allowing the write to opt-out of the document-level validation.
func (mb *InsertManyBundle) BypassDocumentValidation(b bool) *InsertManyBundle {
	bundle := &InsertManyBundle{
		option: BypassDocumentValidation(b),
		next:   mb,
	}

	return bundle
}

// Ordered adds an option that if true and insert fails, returns without performing remaining writes, otherwise continues
func (mb *InsertManyBundle) Ordered(b bool) *InsertManyBundle {
	bundle := &InsertManyBundle{
		option: Ordered(b),
		next:   mb,
	}

	return bundle
}

// WriteConcern adds an option to specify a write concern
func (mb *InsertManyBundle) WriteConcern(wc *writeconcern.WriteConcern) *InsertManyBundle {
	bundle := &InsertManyBundle{
		option: WriteConcern(wc),
		next:   mb,
	}

	return bundle
}

// Calculates the total length of a bundle, accounting for nested bundles.
func (mb *InsertManyBundle) bundleLength() int {
	if mb == nil {
		return 0
	}

	bundleLen := 0
	for ; mb != nil && mb.option != nil; mb = mb.next {
		if converted, ok := mb.option.(*InsertManyBundle); ok {
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
// Since a bundle can be recursive, this method will unwind all recursive bundles.
func (mb *InsertManyBundle) Unbundle(deduplicate bool) ([]option.InsertOptioner, error) {
	options, err := mb.unbundle()
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
func (mb *InsertManyBundle) unbundle() ([]option.InsertOptioner, error) {
	if mb == nil {
		return nil, nil
	}

	listLen := mb.bundleLength()

	options := make([]option.InsertOptioner, listLen)
	index := listLen - 1

	for listHead := mb; listHead != nil && listHead.option != nil; listHead = listHead.next {
		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*InsertManyBundle); ok {
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

		options[index] = listHead.option.ConvertInsertManyOption()
		index--
	}

	return options, nil
}

// String implements the Stringer interface
func (mb *InsertManyBundle) String() string {
	if mb == nil {
		return ""
	}

	str := ""
	for head := mb; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*InsertManyBundle); ok {
			str += converted.String()
			continue
		}

		str += head.option.ConvertInsertManyOption().String() + "\n"
	}

	return str
}

// BypassDocumentValidation allows the write to opt-out of the document-level validation.
func BypassDocumentValidation(b bool) OptBypassDocumentValidation {
	return OptBypassDocumentValidation(b)
}

// Ordered if true and insert fails, returns without performing remaining writes, otherwise continues
func Ordered(b bool) OptOrdered {
	return OptOrdered(b)
}

// WriteConcern specifies a write concern
func WriteConcern(wc *writeconcern.WriteConcern) OptWriteConcern {
	return OptWriteConcern{
		WriteConcern: wc,
	}
}

// OptBypassDocumentValidation allows the write to opt-out of the document-level validation.
type OptBypassDocumentValidation option.OptBypassDocumentValidation

// OptOrdered if true and insert fails, returns without performing remaining writes, otherwise continues
type OptOrdered option.OptOrdered

// OptWriteConcern specifies a write concern
type OptWriteConcern option.OptWriteConcern

func (OptBypassDocumentValidation) insertOne() {}

// ConvertInsertOneOption implements the InsertOne interface
func (opt OptBypassDocumentValidation) ConvertInsertOneOption() option.InsertOptioner {
	return option.OptBypassDocumentValidation(opt)
}

func (OptWriteConcern) insertOne() {}

// ConvertInsertOneOption implements the InsertOne interface
func (opt OptWriteConcern) ConvertInsertOneOption() option.InsertOptioner {
	return option.OptWriteConcern(opt)
}

func (OptWriteConcern) insertMany() {}

// ConvertInsertManyOption implements the InsertMany interface
func (opt OptWriteConcern) ConvertInsertManyOption() option.InsertOptioner {
	return option.OptWriteConcern(opt)
}

func (OptBypassDocumentValidation) insertMany() {}

// ConvertInsertManyOption implements the InsertMany interface
func (opt OptBypassDocumentValidation) ConvertInsertManyOption() option.InsertOptioner {
	return option.OptBypassDocumentValidation(opt)
}

func (OptOrdered) insertMany() {}

// ConvertInsertManyOption implements the InsertMany interface
func (opt OptOrdered) ConvertInsertManyOption() option.InsertOptioner {
	return option.OptOrdered(opt)
}
