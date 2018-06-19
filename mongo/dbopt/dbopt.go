package dbopt

import (
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/mongo"
)

var dbBundle = new(DatabaseBundle)

// Option represents a DB option.
type Option interface {
	dbOption()
}

// optionFunc adds the option to the DB
type optionFunc func(*Database) error

// Database represents a database.
type Database struct {
	client         *mongo.Client
	name           string
	readConcern    *readconcern.ReadConcern
	writeConcern   *writeconcern.WriteConcern
	readPreference *readpref.ReadPref
	readSelector   description.ServerSelector
	writeSelector  description.ServerSelector
}

// DatabaseBundle is a bundle of database options.
type DatabaseBundle struct {
	option Option
	next   *DatabaseBundle
}

func (*DatabaseBundle) dbOption() {}

func (optionFunc) dbOption() {}

// BundleDatabase bundles database options
func BundleDatabase(opts ...Option) *DatabaseBundle {
	head := dbBundle

	for _, opt := range opts {
		newBundle := DatabaseBundle{
			option: opt,
			next:   head,
		}
		head = &newBundle
	}

	return head
}

// ReadConcern sets the read concern.
func (db *DatabaseBundle) ReadConcern(rc *readconcern.ReadConcern) *DatabaseBundle {
	return &DatabaseBundle{
		option: ReadConcern(rc),
		next:   db,
	}
}

// WriteConcern sets the write concern.
func (db *DatabaseBundle) WriteConcern(wc *writeconcern.WriteConcern) *DatabaseBundle {
	return &DatabaseBundle{
		option: WriteConcern(wc),
		next:   db,
	}
}

// ReadPreference sets the read preference.
func (db *DatabaseBundle) ReadPreference(rp *readpref.ReadPref) *DatabaseBundle {
	return &DatabaseBundle{
		option: ReadPreference(rp),
		next:   db,
	}
}

// Unbundle unbundles the options, returning a collection.
func (db *DatabaseBundle) Unbundle() (*Database, error) {
	database := &Database{}
	err := db.unbundle(database)
	if err != nil {
		return nil, err
	}

	return database, nil
}

// Helper that recursively unwraps the bundle.
func (db *DatabaseBundle) unbundle(database *Database) error {
	if db == nil {
		return nil
	}

	for head := db; head != nil && head.option != nil; head = head.next {
		var err error
		switch opt := head.option.(type) {
		case *DatabaseBundle:
			err = opt.unbundle(database) // add all bundle's options to database
		case optionFunc:
			err = opt(database) // add option to database
		default:
			return nil
		}
		if err != nil {
			return err
		}
	}

	return nil

}

// ReadConcern sets the read concern.
func ReadConcern(rc *readconcern.ReadConcern) Option {
	return optionFunc(
		func(d *Database) error {
			if d.readConcern == nil {
				d.readConcern = rc
			}
			return nil
		})
}

// WriteConcern sets the write concern.
func WriteConcern(wc *writeconcern.WriteConcern) Option {
	return optionFunc(
		func(d *Database) error {
			if d.writeConcern == nil {
				d.writeConcern = wc
			}
			return nil
		})
}

// ReadPreference sets the read preference.
func ReadPreference(rp *readpref.ReadPref) Option {
	return optionFunc(
		func(d *Database) error {
			if d.readPreference == nil {
				d.readPreference = rp
			}
			return nil
		})
}
