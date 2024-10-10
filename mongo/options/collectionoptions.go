// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

// CollectionOptions represents arguments that can be used to configure a Collection.
//
// See corresponding setter methods for documentation.
type CollectionOptions struct {
	ReadConcern    *readconcern.ReadConcern
	WriteConcern   *writeconcern.WriteConcern
	ReadPreference *readpref.ReadPref
	BSONOptions    *BSONOptions
	Registry       *bson.Registry
}

// CollectionOptionsBuilder contains options to configure a Collection instance.
// Each option can be set through setter functions. See documentation for each
// setter function for an explanation of the option.
type CollectionOptionsBuilder struct {
	Opts []func(*CollectionOptions) error
}

// Collection creates a new CollectionOptions instance.
func Collection() *CollectionOptionsBuilder {
	return &CollectionOptionsBuilder{}
}

// List returns a list of CollectionOptions setter functions.
func (c *CollectionOptionsBuilder) List() []func(*CollectionOptions) error {
	return c.Opts
}

// SetReadConcern sets the value for the ReadConcern field. ReadConcern is the read concern to use for
// operations executed on the Collection. The default value is nil, which means that the read concern
// of the Database used to configure the Collection will be used.
func (c *CollectionOptionsBuilder) SetReadConcern(rc *readconcern.ReadConcern) *CollectionOptionsBuilder {
	c.Opts = append(c.Opts, func(opts *CollectionOptions) error {
		opts.ReadConcern = rc

		return nil
	})

	return c
}

// SetWriteConcern sets the value for the WriteConcern field. WriteConcern is the write concern to
// use for operations executed on the Collection. The default value is nil, which means that the write
// concern of the Database used to configure the Collection will be used.
func (c *CollectionOptionsBuilder) SetWriteConcern(wc *writeconcern.WriteConcern) *CollectionOptionsBuilder {
	c.Opts = append(c.Opts, func(opts *CollectionOptions) error {
		opts.WriteConcern = wc

		return nil
	})

	return c
}

// SetReadPreference sets the value for the ReadPreference field. ReadPreference is the read preference
// to use for operations executed on the Collection. The default value is nil, which means that the
// read preference of the Database used to configure the Collection will be used.
func (c *CollectionOptionsBuilder) SetReadPreference(rp *readpref.ReadPref) *CollectionOptionsBuilder {
	c.Opts = append(c.Opts, func(opts *CollectionOptions) error {
		opts.ReadPreference = rp

		return nil
	})

	return c
}

// SetBSONOptions configures optional BSON marshaling and unmarshaling behavior. BSONOptions configures
// optional BSON marshaling and unmarshaling behavior.
func (c *CollectionOptionsBuilder) SetBSONOptions(bopts *BSONOptions) *CollectionOptionsBuilder {
	c.Opts = append(c.Opts, func(opts *CollectionOptions) error {
		opts.BSONOptions = bopts

		return nil
	})

	return c
}

// SetRegistry sets the value for the Registry field. Registry is the BSON registry to marshal and
// unmarshal documents for operations executed on the Collection. The default value is nil, which
// means that the registry of the Database used to configure the Collection will be used.
func (c *CollectionOptionsBuilder) SetRegistry(r *bson.Registry) *CollectionOptionsBuilder {
	c.Opts = append(c.Opts, func(opts *CollectionOptions) error {
		opts.Registry = r

		return nil
	})
	return c
}
