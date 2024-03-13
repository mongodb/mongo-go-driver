// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

// DatabaseArgs represents arguments that can be used to configure a Database.
type DatabaseArgs struct {
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
	Registry *bsoncodec.Registry
}

// DatabaseOptions contains options to configure a database object. Each option
// can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type DatabaseOptions struct {
	Opts []func(*DatabaseArgs) error
}

// Database creates a new DatabaseOptions instance.
func Database() *DatabaseOptions {
	return &DatabaseOptions{}
}

// ArgsSetters returns a list of DatabaseArgs setter functions.
func (d *DatabaseOptions) ArgsSetters() []func(*DatabaseArgs) error {
	return d.Opts
}

// SetReadConcern sets the value for the ReadConcern field.
func (d *DatabaseOptions) SetReadConcern(rc *readconcern.ReadConcern) *DatabaseOptions {
	d.Opts = append(d.Opts, func(args *DatabaseArgs) error {
		args.ReadConcern = rc

		return nil
	})

	return d
}

// SetWriteConcern sets the value for the WriteConcern field.
func (d *DatabaseOptions) SetWriteConcern(wc *writeconcern.WriteConcern) *DatabaseOptions {
	d.Opts = append(d.Opts, func(args *DatabaseArgs) error {
		args.WriteConcern = wc

		return nil
	})

	return d
}

// SetReadPreference sets the value for the ReadPreference field.
func (d *DatabaseOptions) SetReadPreference(rp *readpref.ReadPref) *DatabaseOptions {
	d.Opts = append(d.Opts, func(args *DatabaseArgs) error {
		args.ReadPreference = rp

		return nil
	})

	return d
}

// SetBSONOptions configures optional BSON marshaling and unmarshaling behavior.
func (d *DatabaseOptions) SetBSONOptions(opts *BSONOptions) *DatabaseOptions {
	d.Opts = append(d.Opts, func(args *DatabaseArgs) error {
		args.BSONOptions = opts

		return nil
	})

	return d
}

// SetRegistry sets the value for the Registry field.
func (d *DatabaseOptions) SetRegistry(r *bsoncodec.Registry) *DatabaseOptions {
	d.Opts = append(d.Opts, func(args *DatabaseArgs) error {
		args.Registry = r

		return nil
	})

	return d
}
