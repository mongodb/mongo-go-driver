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

// CollectionOptions represents options that can be used to configure a Collection.
type CollectionOptions struct {
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

// Collection creates a new CollectionOptions instance.
func Collection() *CollectionOptions {
	return &CollectionOptions{}
}

// SetReadConcern sets the value for the ReadConcern field.
func (c *CollectionOptions) SetReadConcern(rc *readconcern.ReadConcern) *CollectionOptions {
	c.ReadConcern = rc
	return c
}

// SetWriteConcern sets the value for the WriteConcern field.
func (c *CollectionOptions) SetWriteConcern(wc *writeconcern.WriteConcern) *CollectionOptions {
	c.WriteConcern = wc
	return c
}

// SetReadPreference sets the value for the ReadPreference field.
func (c *CollectionOptions) SetReadPreference(rp *readpref.ReadPref) *CollectionOptions {
	c.ReadPreference = rp
	return c
}

// SetBSONOptions configures optional BSON marshaling and unmarshaling behavior.
func (c *CollectionOptions) SetBSONOptions(opts *BSONOptions) *CollectionOptions {
	c.BSONOptions = opts
	return c
}

// SetRegistry sets the value for the Registry field.
func (c *CollectionOptions) SetRegistry(r *bson.Registry) *CollectionOptions {
	c.Registry = r
	return c
}
