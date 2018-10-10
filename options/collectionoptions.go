package options

import (
	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

// CollectionOptions represent all possible options to configure a Collection.
type CollectionOptions struct {
	ReadConcern    *readconcern.ReadConcern   // The read concern for operations in the collection.
	WriteConcern   *writeconcern.WriteConcern // The write concern for operations in the collection.
	ReadPreference *readpref.ReadPref         // The read preference for operations in the collection.
	Registry       *bsoncodec.Registry        // The registry to be used to construct BSON encoders and decoders for the collection.
}

// Collection creates a new CollectionOptions instance
func Collection() *CollectionOptions {
	return &CollectionOptions{}
}

// SetReadConcern sets the read concern for the collection.
func (c *CollectionOptions) SetReadConcern(rc *readconcern.ReadConcern) *CollectionOptions {
	c.ReadConcern = rc
	return c
}

// SetWriteConcern sets the write concern for the collection.
func (c *CollectionOptions) SetWriteConcern(wc *writeconcern.WriteConcern) *CollectionOptions {
	c.WriteConcern = wc
	return c
}

// SetReadPreference sets the read preference for the collection.
func (c *CollectionOptions) SetReadPreference(rp *readpref.ReadPref) *CollectionOptions {
	c.ReadPreference = rp
	return c
}

// SetRegistry sets the bsoncodec Registry for the collection.
func (c *CollectionOptions) SetRegistry(r *bsoncodec.Registry) *CollectionOptions {
	c.Registry = r
	return c
}

// ToCollectionOptions combines the *CollectionOptions arguments into a single *CollectionOptions in a last one wins
// fashion.
func ToCollectionOptions(opts ...*CollectionOptions) *CollectionOptions {
	c := Collection()

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.ReadConcern != nil {
			c.ReadConcern = opt.ReadConcern
		}
		if opt.WriteConcern != nil {
			c.WriteConcern = opt.WriteConcern
		}
		if opt.ReadPreference != nil {
			c.ReadPreference = opt.ReadPreference
		}
		if opt.Registry != nil {
			c.Registry = opt.Registry
		}
	}

	return c
}
