package dbopt

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
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
	ReadConcern    *readconcern.ReadConcern
	WriteConcern   *writeconcern.WriteConcern
	ReadPreference *readpref.ReadPref
}

// DatabaseBundle is a bundle of database options.
type DatabaseBundle struct {
	option Option
	next   *DatabaseBundle
}

func (*DatabaseBundle) dbOption() {}

func (optionFunc) dbOption() {}

// DropDB represents all possible params for the dropDatabase() function
type DropDB interface {
	dropDB()
}

// DropDBSession is the session for the dropDatabase() function.
type DropDBSession interface {
	DropDB
	ConvertDropDBSession() *session.Client
}

// DropDBSessionOpt is a dropDatabase session option.
type DropDBSessionOpt struct{}

func (DropDBSessionOpt) dropDB() {}

// ConvertDropDBSession implements the DropDBSession interface.
func (DropDBSessionOpt) ConvertDropDBSession() *session.Client {
	return nil
}

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

// String implements the Stringer interface
func (db *DatabaseBundle) String() string {
	if db == nil {
		return ""
	}

	str := ""
	for head := db; head != nil && head.option != nil; head = head.next {
		switch opt := head.option.(type) {
		case *DatabaseBundle:
			str += opt.String()
		case optionFunc:
			str += reflect.TypeOf(opt).String() + "\n"
		}
	}

	return str
}

// ReadConcern sets the read concern.
func ReadConcern(rc *readconcern.ReadConcern) Option {
	return optionFunc(
		func(d *Database) error {
			if d.ReadConcern == nil {
				d.ReadConcern = rc
			}
			return nil
		})
}

// WriteConcern sets the write concern.
func WriteConcern(wc *writeconcern.WriteConcern) Option {
	return optionFunc(
		func(d *Database) error {
			if d.WriteConcern == nil {
				d.WriteConcern = wc
			}
			return nil
		})
}

// ReadPreference sets the read preference.
func ReadPreference(rp *readpref.ReadPref) Option {
	return optionFunc(
		func(d *Database) error {
			if d.ReadPreference == nil {
				d.ReadPreference = rp
			}
			return nil
		})
}
