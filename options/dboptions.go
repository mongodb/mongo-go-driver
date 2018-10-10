package options

import (
	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

// DatabaseOptions represent all possible options to configure a Database.
type DatabaseOptions struct {
	ReadConcern    *readconcern.ReadConcern   // The read concern for operations in the database.
	WriteConcern   *writeconcern.WriteConcern // The write concern for operations in the database.
	ReadPreference *readpref.ReadPref         // The read preference for operations in the database.
	Registry       *bsoncodec.Registry        // The registry to be used to construct BSON encoders and decoders for the database.
}

// Database creates a new DatabaseOptions instance
func Database() *DatabaseOptions {
	return &DatabaseOptions{}
}

// SetReadConcern sets the read concern for the database.
func (d *DatabaseOptions) SetReadConcern(rc *readconcern.ReadConcern) *DatabaseOptions {
	d.ReadConcern = rc
	return d
}

// SetWriteConcern sets the write concern for the database.
func (d *DatabaseOptions) SetWriteConcern(wc *writeconcern.WriteConcern) *DatabaseOptions {
	d.WriteConcern = wc
	return d
}

// SetReadPreference sets the read preference for the database.
func (d *DatabaseOptions) SetReadPreference(rp *readpref.ReadPref) *DatabaseOptions {
	d.ReadPreference = rp
	return d
}

// SetRegistry sets the bsoncodec Registry for the database.
func (d *DatabaseOptions) SetRegistry(r *bsoncodec.Registry) *DatabaseOptions {
	d.Registry = r
	return d
}

// MergeDatabaseOptions combines the *DatabaseOptions arguments into a single *DatabaseOptions in a last one wins
// fashion.
func MergeDatabaseOptions(opts ...*DatabaseOptions) *DatabaseOptions {
	d := Database()

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.ReadConcern != nil {
			d.ReadConcern = opt.ReadConcern
		}
		if opt.WriteConcern != nil {
			d.WriteConcern = opt.WriteConcern
		}
		if opt.ReadPreference != nil {
			d.ReadPreference = opt.ReadPreference
		}
		if opt.Registry != nil {
			d.Registry = opt.Registry
		}
	}

	return d
}
