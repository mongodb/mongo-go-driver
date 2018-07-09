// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package insertopt

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/session"
)

var insertOneBundle = new(OneBundle)
var insertManyBundle = new(ManyBundle)

// One represents all passable params for the insertOne() function.
type One interface {
	insertOne()
}

// OneOption represents the options for the insertOne() function.
type OneOption interface {
	One
	ConvertInsertOption() option.InsertOptioner
}

// OneSession is the session for the insertOne() function
type OneSession interface {
	One
	ConvertInsertSession() *session.Client
}

// Many represents all passable params for the insertMany() function.
type Many interface {
	insertMany()
}

// ManyOption represents the options for the insertMany() function.
type ManyOption interface {
	Many
	ConvertInsertOption() option.InsertOptioner
}

// ManySession is the session for the insertMany() function
type ManySession interface {
	Many
	ConvertInsertSession() *session.Client
}

// OneBundle is a bundle of One options
type OneBundle struct {
	option One
	next   *OneBundle
}

// Implement the One interface
func (ob *OneBundle) insertOne() {}

// ConvertInsertOption implements the One interface
func (ob *OneBundle) ConvertInsertOption() option.InsertOptioner { return nil }

// BundleOne bundles One options
func BundleOne(opts ...One) *OneBundle {
	head := insertOneBundle

	for _, opt := range opts {
		newBundle := OneBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// BypassDocumentValidation adds an option allowing the write to opt-out of the document-level validation.
func (ob *OneBundle) BypassDocumentValidation(b bool) *OneBundle {
	bundle := &OneBundle{
		option: BypassDocumentValidation(b),
		next:   ob,
	}

	return bundle
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

// Unbundle unwinds and deduplicates the options used to create it and those
// added after creation into a single slice of options.
//
// The deduplicate parameter is used to determine if the bundle is just flattened or
// if we actually deduplicate options.
//
// Since a bundle can be recursive, this method will unwind all recursive bundles.
func (ob *OneBundle) Unbundle(deduplicate bool) ([]option.InsertOptioner, error) {
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
func (ob *OneBundle) unbundle() ([]option.InsertOptioner, error) {
	if ob == nil {
		return nil, nil
	}

	listLen := ob.bundleLength()

	options := make([]option.InsertOptioner, listLen)
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

		if conv, ok := listHead.option.(OneOption); ok {
			options[index] = conv.ConvertInsertOption()
			index--
		}
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

		if conv, ok := head.option.(OneOption); !ok {
			str += conv.ConvertInsertOption().String() + "\n"
		}
	}

	return str
}

// ManyBundle is a bundle of InsertInsertMany options
type ManyBundle struct {
	option Many
	next   *ManyBundle
}

// BundleMany bundles Many options
func BundleMany(opts ...Many) *ManyBundle {
	head := insertManyBundle

	for _, opt := range opts {
		newBundle := ManyBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// Implement the Many interface
func (mb *ManyBundle) insertMany() {}

// ConvertInsertOption implements the Many interface
func (mb *ManyBundle) ConvertInsertOption() option.InsertOptioner { return nil }

// BypassDocumentValidation adds an option allowing the write to opt-out of the document-level validation.
func (mb *ManyBundle) BypassDocumentValidation(b bool) *ManyBundle {
	bundle := &ManyBundle{
		option: BypassDocumentValidation(b),
		next:   mb,
	}

	return bundle
}

// Ordered adds an option that if true and insert fails, returns without performing remaining writes, otherwise continues
func (mb *ManyBundle) Ordered(b bool) *ManyBundle {
	bundle := &ManyBundle{
		option: Ordered(b),
		next:   mb,
	}

	return bundle
}

// Calculates the total length of a bundle, accounting for nested bundles.
func (mb *ManyBundle) bundleLength() int {
	if mb == nil {
		return 0
	}

	bundleLen := 0
	for ; mb != nil && mb.option != nil; mb = mb.next {
		if converted, ok := mb.option.(*ManyBundle); ok {
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
func (mb *ManyBundle) Unbundle(deduplicate bool) ([]option.InsertOptioner, error) {
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
func (mb *ManyBundle) unbundle() ([]option.InsertOptioner, error) {
	if mb == nil {
		return nil, nil
	}

	listLen := mb.bundleLength()

	options := make([]option.InsertOptioner, listLen)
	index := listLen - 1

	for listHead := mb; listHead != nil && listHead.option != nil; listHead = listHead.next {
		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*ManyBundle); ok {
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

		if conv, ok := listHead.option.(ManyOption); ok {
			options[index] = conv.ConvertInsertOption()
			index--
		}
	}

	return options, nil
}

// String implements the Stringer interface
func (mb *ManyBundle) String() string {
	if mb == nil {
		return ""
	}

	str := ""
	for head := mb; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*ManyBundle); ok {
			str += converted.String()
			continue
		}

		if conv, ok := head.option.(ManyOption); !ok {
			str += conv.ConvertInsertOption().String() + "\n"
		}
	}

	return str
}

// RetrieveSession retrieves the option that wraps a mongosession.
func (ob *OneBundle) RetrieveSession() *session.Client {
	if ob == nil {
		return nil
	}

	for head := ob; head != nil && head.option != nil; head = head.next {
		switch t := head.option.(type) {
		case *OneBundle:
			if res := t.RetrieveSession(); res != nil {
				return res
			}
		case OneSession:
			return t.ConvertInsertSession()
		}
	}

	return nil
}

// RetrieveSession retrieves the option that wraps a mongosession.
func (mb *ManyBundle) RetrieveSession() *session.Client {
	if mb == nil {
		return nil
	}

	for head := mb; head != nil && head.option != nil; head = head.next {
		switch t := head.option.(type) {
		case *ManyBundle:
			if res := t.RetrieveSession(); res != nil {
				return res
			}
		case ManySession:
			return t.ConvertInsertSession()
		}
	}

	return nil
}

// BypassDocumentValidation allows the write to opt-out of the document-level validation.
func BypassDocumentValidation(b bool) OptBypassDocumentValidation {
	return OptBypassDocumentValidation(b)
}

// Ordered if true and insert fails, returns without performing remaining writes, otherwise continues
func Ordered(b bool) OptOrdered {
	return OptOrdered(b)
}

// OptBypassDocumentValidation allows the write to opt-out of the document-level validation.
type OptBypassDocumentValidation option.OptBypassDocumentValidation

// OptOrdered if true and insert fails, returns without performing remaining writes, otherwise continues
type OptOrdered option.OptOrdered

func (OptBypassDocumentValidation) insertMany() {}

func (OptBypassDocumentValidation) insertOne() {}

// ConvertInsertOption implements the One,Many interface
func (opt OptBypassDocumentValidation) ConvertInsertOption() option.InsertOptioner {
	return option.OptBypassDocumentValidation(opt)
}

func (OptOrdered) insertMany() {}

// ConvertInsertOption implements the Many interface
func (opt OptOrdered) ConvertInsertOption() option.InsertOptioner {
	return option.OptOrdered(opt)
}

// InsertSessionOpt is a one,many mongosession option.
type InsertSessionOpt struct{}

func (InsertSessionOpt) insertOne()  {}
func (InsertSessionOpt) insertMany() {}

// ConvertInsertSession implements the InsertSession interface.
func (opt InsertSessionOpt) ConvertInsertSession() *session.Client {
	return nil
}
