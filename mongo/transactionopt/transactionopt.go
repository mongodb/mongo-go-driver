// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package transactionopt

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

var transactionBundle = new(TransactionBundle)

// Transaction represents options for creating client sessions.
type Transaction interface {
	session()
	ConvertTransactionOption() session.ClientOptioner
}

// TransactionBundle bundles session options
type TransactionBundle struct {
	option Transaction
	next   *TransactionBundle
}

func (sb *TransactionBundle) session() {}

// ConvertTransactionOption implements the Transaction interface
func (sb *TransactionBundle) ConvertTransactionOption() session.ClientOptioner {
	return nil
}

// BundleTransaction bundles session options
func BundleTransaction(opts ...Transaction) *TransactionBundle {
	head := transactionBundle

	for _, opt := range opts {
		newBundle := TransactionBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// ReadConcern specifies the default read concern for transactions started from this session.
func (sb *TransactionBundle) ReadConcern(rc *readconcern.ReadConcern) *TransactionBundle {
	return &TransactionBundle{
		option: ReadConcern(rc),
		next:   sb,
	}
}

// ReadPreference specifies the default read preference for transactions started from this session.
func (sb *TransactionBundle) ReadPreference(rp *readpref.ReadPref) *TransactionBundle {
	return &TransactionBundle{
		option: ReadPreference(rp),
		next:   sb,
	}
}

// WriteConcern specifies the default write concern for transactions started from this session.
func (sb *TransactionBundle) WriteConcern(wc *writeconcern.WriteConcern) *TransactionBundle {
	return &TransactionBundle{
		option: WriteConcern(wc),
		next:   sb,
	}
}

// Unbundle transforms a bundle into a slice of options, optionally deduplicating.
func (sb *TransactionBundle) Unbundle(deduplicate bool) ([]session.ClientOptioner, error) {
	opts, err := sb.unbundle()
	if err != nil {
		return nil, err
	}

	if !deduplicate {
		return opts, nil
	}

	optionsSet := make(map[reflect.Type]struct{})

	for i := len(opts) - 1; i >= 0; i-- {
		currOpt := opts[i]
		optType := reflect.TypeOf(currOpt)

		if _, ok := optionsSet[optType]; ok {
			// already found
			opts = append(opts[:i], opts[i+1:]...)
			continue
		}

		optionsSet[optType] = struct{}{}
	}

	return opts, nil
}

func (sb *TransactionBundle) unbundle() ([]session.ClientOptioner, error) {
	if sb == nil {
		return nil, nil
	}

	listLen := sb.bundleLength()

	options := make([]session.ClientOptioner, listLen)
	index := listLen - 1

	for listHead := sb; listHead != nil && listHead.option != nil; listHead = listHead.next {
		if converted, ok := listHead.option.(*TransactionBundle); ok {
			nestedOpts, err := converted.unbundle()
			if err != nil {
				return nil, err
			}

			startIndex := index - (len(nestedOpts)) + 1

			for _, nestedOpt := range nestedOpts {
				options[startIndex] = nestedOpt
				startIndex++
			}

			index -= len(nestedOpts)
			continue
		}

		options[index] = listHead.option.ConvertTransactionOption()
		index--
	}

	return options, nil
}

// Calculates the total length of a bundle, accounting for nested bundles.
func (sb *TransactionBundle) bundleLength() int {
	if sb == nil {
		return 0
	}

	bundleLen := 0
	for ; sb != nil && sb.option != nil; sb = sb.next {
		if converted, ok := sb.option.(*TransactionBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		bundleLen++
	}

	return bundleLen
}

// ReadConcern specifies the default read concern for transactions started from this session.
func ReadConcern(rc *readconcern.ReadConcern) OptReadConcern {
	return OptReadConcern{
		ReadConcern: rc,
	}
}

// ReadPreference specifies the default read preference for transactions started from this session.
func ReadPreference(rp *readpref.ReadPref) OptReadPreference {
	return OptReadPreference{
		ReadPref: rp,
	}
}

// WriteConcern specifies the default write concern for transactions started from this session.
func WriteConcern(wc *writeconcern.WriteConcern) OptWriteConcern {
	return OptWriteConcern{
		WriteConcern: wc,
	}
}

// OptReadConcern specifies the default read concern for transactions started from this session.
type OptReadConcern session.OptCurrentReadConcern

func (OptReadConcern) session() {}

// ConvertTransactionOption implements the Transaction interface.
func (opt OptReadConcern) ConvertTransactionOption() session.ClientOptioner {
	return session.OptCurrentReadConcern(opt)
}

// OptReadPreference specifies the default read preference for transactions started from this session.
type OptReadPreference session.OptCurrentReadPreference

func (OptReadPreference) session() {}

// ConvertTransactionOption implements the Transaction interface.
func (opt OptReadPreference) ConvertTransactionOption() session.ClientOptioner {
	return session.OptCurrentReadPreference(opt)
}

// OptWriteConcern specifies the default write concern for transactions started from this session.
type OptWriteConcern session.OptCurrentWriteConcern

func (OptWriteConcern) session() {}

// ConvertTransactionOption implements the Transaction interface.
func (opt OptWriteConcern) ConvertTransactionOption() session.ClientOptioner {
	return session.OptCurrentWriteConcern(opt)
}
