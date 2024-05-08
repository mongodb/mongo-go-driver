// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

// CollectionArgs represents arguments that can be used to configure a
// Collection.
type CollectionArgs struct {
	// ReadConcern is the read concern to use for operations executed on the Collection. The default value is nil, which means that
	// the read concern of the Database used to configure the Collection will be used.
	ReadConcern *readconcern.ReadConcern

	// WriteConcern is the write concern to use for operations executed on the Collection. The default value is nil, which means that
	// the write concern of the Database used to configure the Collection will be used.
	WriteConcern *writeconcern.WriteConcern

	// ReadPreference is the read preference to use for operations executed on the Collection. The default value is nil, which means that
	// the read preference of the Database used to configure the Collection will be used.
	ReadPreference *readpref.ReadPref

	// BSONOptions configures optional BSON marshaling and unmarshaling
	// behavior.
	BSONOptions *BSONOptions

	// Registry is the BSON registry to marshal and unmarshal documents for operations executed on the Collection. The default value
	// is nil, which means that the registry of the Database used to configure the Collection will be used.
	Registry *bson.Registry
}

// CollectionOptions contains options to configure a Collection instance. Each
// option can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type CollectionOptions struct {
	Opts []func(*CollectionArgs) error
}

// Collection creates a new CollectionOptions instance.
func Collection() *CollectionOptions {
	return &CollectionOptions{}
}

// ArgsSetters returns a list of CollectionArgs setter functions.
func (c *CollectionOptions) ArgsSetters() []func(*CollectionArgs) error {
	return c.Opts
}

// SetReadConcern sets the value for the ReadConcern field.
func (c *CollectionOptions) SetReadConcern(rc *readconcern.ReadConcern) *CollectionOptions {
	c.Opts = append(c.Opts, func(args *CollectionArgs) error {
		args.ReadConcern = rc

		return nil
	})

	return c
}

// SetWriteConcern sets the value for the WriteConcern field.
func (c *CollectionOptions) SetWriteConcern(wc *writeconcern.WriteConcern) *CollectionOptions {
	c.Opts = append(c.Opts, func(args *CollectionArgs) error {
		args.WriteConcern = wc

		return nil
	})

	return c
}

// SetReadPreference sets the value for the ReadPreference field.
func (c *CollectionOptions) SetReadPreference(rp *readpref.ReadPref) *CollectionOptions {
	c.Opts = append(c.Opts, func(args *CollectionArgs) error {
		args.ReadPreference = rp

		return nil
	})

	return c
}

// SetBSONOptions configures optional BSON marshaling and unmarshaling behavior.
func (c *CollectionOptions) SetBSONOptions(opts *BSONOptions) *CollectionOptions {
	c.Opts = append(c.Opts, func(args *CollectionArgs) error {
		args.BSONOptions = opts

		return nil
	})

	return c
}

// SetRegistry sets the value for the Registry field.
func (c *CollectionOptions) SetRegistry(r *bson.Registry) *CollectionOptions {
	c.Opts = append(c.Opts, func(args *CollectionArgs) error {
		args.Registry = r

		return nil
	})
	return c
}
