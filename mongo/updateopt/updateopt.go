// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package updateopt

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

var updateBundle = new(UpdateBundle)

// Update represents all passable params for the update() function.
type Update interface {
	update()
}

// UpdateOption represents the options for the update() function.
type UpdateOption interface {
	Update
	ConvertUpdateOption() option.UpdateOptioner
}

// UpdateSession is the session for the update() function
type UpdateSession interface {
	Update
	ConvertUpdateSession() *session.Client
}

// UpdateBundle bundles One options
type UpdateBundle struct {
	option Update
	next   *UpdateBundle
}

// Implement the Update interface
func (ub *UpdateBundle) update() {}

// ConvertUpdateOption implements the Update interface
func (ub *UpdateBundle) ConvertUpdateOption() option.UpdateOptioner { return nil }

// BundleUpdate bundles Update options
func BundleUpdate(opts ...Update) *UpdateBundle {
	head := updateBundle

	for _, opt := range opts {
		newBundle := UpdateBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// ArrayFilters adds an option to specify which array elements an update should apply.
func (ub *UpdateBundle) ArrayFilters(filter ...interface{}) *UpdateBundle {
	bundle := &UpdateBundle{
		option: ArrayFilters(filter),
		next:   ub,
	}

	return bundle
}

// BypassDocumentValidation adds an option to allow the write to opt-out of document-level validation.
func (ub *UpdateBundle) BypassDocumentValidation(b bool) *UpdateBundle {
	bundle := &UpdateBundle{
		option: BypassDocumentValidation(b),
		next:   ub,
	}

	return bundle
}

// Collation adds an option to specify a collation.
func (ub *UpdateBundle) Collation(c *mongoopt.Collation) *UpdateBundle {
	bundle := &UpdateBundle{
		option: Collation(c),
		next:   ub,
	}

	return bundle
}

// Upsert adds an option to specify whether to insert the document if it is not present.
func (ub *UpdateBundle) Upsert(b bool) *UpdateBundle {
	bundle := &UpdateBundle{
		option: Upsert(b),
		next:   ub,
	}

	return bundle
}

// String implements the Stringer interface
func (ub *UpdateBundle) String() string {
	if ub == nil {
		return ""
	}
	str := ""
	for head := ub; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*UpdateBundle); ok {
			str += converted.String()
			continue
		}
		if conv, ok := head.option.(UpdateOption); !ok {
			str += conv.ConvertUpdateOption().String() + "\n"
		}
	}
	return str
}

// Calculates the total length of a bundle, accounting for nested bundles.
func (ub *UpdateBundle) bundleLength() int {
	if ub == nil {
		return 0
	}

	bundleLen := 0
	for ; ub != nil; ub = ub.next {
		if ub.option == nil {
			continue
		}
		if converted, ok := ub.option.(*UpdateBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		if _, ok := ub.option.(UpdateSessionOpt); !ok {
			bundleLen++
		}
	}

	return bundleLen
}

// Unbundle transforms a bundle into a slice of options, optionally deduplicating
func (ub *UpdateBundle) Unbundle(deduplicate bool) ([]option.UpdateOptioner, *session.Client, error) {

	options, sess, err := ub.unbundle()
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
func (ub *UpdateBundle) unbundle() ([]option.UpdateOptioner, *session.Client, error) {
	if ub == nil {
		return nil, nil, nil
	}

	var sess *session.Client
	listLen := ub.bundleLength()

	options := make([]option.UpdateOptioner, listLen)
	index := listLen - 1

	for listHead := ub; listHead != nil; listHead = listHead.next {
		if listHead.option == nil {
			continue
		}

		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*UpdateBundle); ok {
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
		case UpdateOption:
			options[index] = t.ConvertUpdateOption()
			index--
		case UpdateSession:
			if sess == nil {
				sess = t.ConvertUpdateSession()
			}
		}
	}

	return options, sess, nil

}

// ArrayFilters specifies which array elements an update should apply.
func ArrayFilters(filter ...interface{}) OptArrayFilters {
	return OptArrayFilters(filter)
}

// BypassDocumentValidation allows the write to opt-out of document-level validation.
func BypassDocumentValidation(b bool) OptBypassDocumentValidation {
	return OptBypassDocumentValidation(b)
}

// Collation specifies a collation.
func Collation(c *mongoopt.Collation) OptCollation {
	return OptCollation{Collation: c.Convert()}
}

// Upsert specifies whether to insert the document if it is not present.
func Upsert(b bool) OptUpsert {
	return OptUpsert(b)
}

// OptArrayFilters specifies which array elements an update should apply.
type OptArrayFilters option.OptArrayFilters

func (OptArrayFilters) update() {}

// ConvertUpdateOption implements the Update interface
func (opt OptArrayFilters) ConvertUpdateOption() option.UpdateOptioner {
	return option.OptArrayFilters(opt)
}

// OptBypassDocumentValidation allows the write to opt-out of document-level validation.
type OptBypassDocumentValidation option.OptBypassDocumentValidation

func (OptBypassDocumentValidation) update() {}

// ConvertUpdateOption implements the Update interface
func (opt OptBypassDocumentValidation) ConvertUpdateOption() option.UpdateOptioner {
	return option.OptBypassDocumentValidation(opt)
}

// OptCollation specifies a collation.
type OptCollation option.OptCollation

func (OptCollation) update() {}

// ConvertUpdateOption implements the Update interface.
func (opt OptCollation) ConvertUpdateOption() option.UpdateOptioner {
	return option.OptCollation(opt)
}

// OptUpsert specifies whether to insert the document if it is not present.
type OptUpsert option.OptUpsert

func (OptUpsert) update() {}

// ConvertUpdateOption implements the Update interface.
func (opt OptUpsert) ConvertUpdateOption() option.UpdateOptioner {
	return option.OptUpsert(opt)
}

// UpdateSessionOpt is an update session option.
type UpdateSessionOpt struct{}

func (UpdateSessionOpt) update() {}

// ConvertUpdateSession implements the UpdateSession interface.
func (UpdateSessionOpt) ConvertUpdateSession() *session.Client {
	return nil
}
