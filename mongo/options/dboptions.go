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

// DatabaseOptions represents arguments that can be used to configure a
// database.
type DatabaseOptions struct {
	// ReadConcern is the read concern to use for operations executed on the Database. The default value is nil, which means that
	// the read concern of the Client used to configure the Database will be used.
	ReadConcern *readconcern.ReadConcern

	// WriteConcern is the write concern to use for operations executed on the Database. The default value is nil, which means that the
	// write concern of the Client used to configure the Database will be used.
	WriteConcern *writeconcern.WriteConcern

	// ReadPreference is the read preference to use for operations executed on the Database. The default value is nil, which means that
	// the read preference of the Client used to configure the Database will be used.
	ReadPreference *readpref.ReadPref

	// BSONOptions configures optional BSON marshaling and unmarshaling
	// behavior.
	BSONOptions *BSONOptions

	// Registry is the BSON registry to marshal and unmarshal documents for operations executed on the Database. The default value
	// is nil, which means that the registry of the Client used to configure the Database will be used.
	Registry *bson.Registry
}

// DatabaseOptionsBuilder contains options to configure a database object. Each
// option can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type DatabaseOptionsBuilder struct {
	Opts []func(*DatabaseOptions) error
}

// Database creates a new DatabaseOptions instance.
func Database() *DatabaseOptionsBuilder {
	return &DatabaseOptionsBuilder{}
}

// ArgsSetters returns a list of DatabaseArgs setter functions.
func (d *DatabaseOptionsBuilder) ArgsSetters() []func(*DatabaseOptions) error {
	return d.Opts
}

// SetReadConcern sets the value for the ReadConcern field.
func (d *DatabaseOptionsBuilder) SetReadConcern(rc *readconcern.ReadConcern) *DatabaseOptionsBuilder {
	d.Opts = append(d.Opts, func(args *DatabaseOptions) error {
		args.ReadConcern = rc

		return nil
	})

	return d
}

// SetWriteConcern sets the value for the WriteConcern field.
func (d *DatabaseOptionsBuilder) SetWriteConcern(wc *writeconcern.WriteConcern) *DatabaseOptionsBuilder {
	d.Opts = append(d.Opts, func(args *DatabaseOptions) error {
		args.WriteConcern = wc

		return nil
	})

	return d
}

// SetReadPreference sets the value for the ReadPreference field.
func (d *DatabaseOptionsBuilder) SetReadPreference(rp *readpref.ReadPref) *DatabaseOptionsBuilder {
	d.Opts = append(d.Opts, func(args *DatabaseOptions) error {
		args.ReadPreference = rp

		return nil
	})

	return d
}

// SetBSONOptions configures optional BSON marshaling and unmarshaling behavior.
func (d *DatabaseOptionsBuilder) SetBSONOptions(opts *BSONOptions) *DatabaseOptionsBuilder {
	d.Opts = append(d.Opts, func(args *DatabaseOptions) error {
		args.BSONOptions = opts

		return nil
	})

	return d
}

// SetRegistry sets the value for the Registry field.
func (d *DatabaseOptionsBuilder) SetRegistry(r *bson.Registry) *DatabaseOptionsBuilder {
	d.Opts = append(d.Opts, func(args *DatabaseOptions) error {
		args.Registry = r

		return nil
	})
	return d
}
