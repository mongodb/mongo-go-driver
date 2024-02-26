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

// DatabaseOptions represents options that can be used to configure a Database.
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

// Database creates a new DatabaseOptions instance.
func Database() *DatabaseOptions {
	return &DatabaseOptions{}
}

// SetReadConcern sets the value for the ReadConcern field.
func (d *DatabaseOptions) SetReadConcern(rc *readconcern.ReadConcern) *DatabaseOptions {
	d.ReadConcern = rc
	return d
}

// SetWriteConcern sets the value for the WriteConcern field.
func (d *DatabaseOptions) SetWriteConcern(wc *writeconcern.WriteConcern) *DatabaseOptions {
	d.WriteConcern = wc
	return d
}

// SetReadPreference sets the value for the ReadPreference field.
func (d *DatabaseOptions) SetReadPreference(rp *readpref.ReadPref) *DatabaseOptions {
	d.ReadPreference = rp
	return d
}

// SetBSONOptions configures optional BSON marshaling and unmarshaling behavior.
func (d *DatabaseOptions) SetBSONOptions(opts *BSONOptions) *DatabaseOptions {
	d.BSONOptions = opts
	return d
}

// SetRegistry sets the value for the Registry field.
func (d *DatabaseOptions) SetRegistry(r *bson.Registry) *DatabaseOptions {
	d.Registry = r
	return d
}
