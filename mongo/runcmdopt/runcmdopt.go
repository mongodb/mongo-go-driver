package runcmdopt

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/core/readpref"
)

var runCmdBundle = new(RunCmdBundle)

// Option represents a DB option.
type Option interface {
	runCmdOption()
}

// optionFunc adds the option to the DB
type optionFunc func(*RunCmd) error

// RunCmd represents a database.
type RunCmd struct {
	ReadPreference *readpref.ReadPref
}

// RunCmdBundle is a bundle of database options.
type RunCmdBundle struct {
	option Option
	next   *RunCmdBundle
}

func (*RunCmdBundle) runCmdOption() {}

func (optionFunc) runCmdOption() {}

// BundleRunCmd bundles RunCommand options
func BundleRunCmd(opts ...Option) *RunCmdBundle {
	head := runCmdBundle

	for _, opt := range opts {
		newBundle := RunCmdBundle{
			option: opt,
			next:   head,
		}
		head = &newBundle
	}

	return head
}

// ReadPreference sets the read preference.
func (db *RunCmdBundle) ReadPreference(rp *readpref.ReadPref) *RunCmdBundle {
	return &RunCmdBundle{
		option: ReadPreference(rp),
		next:   db,
	}
}

// Unbundle unbundles the options, returning a collection.
func (db *RunCmdBundle) Unbundle() (*RunCmd, error) {
	database := &RunCmd{}
	err := db.unbundle(database)
	if err != nil {
		return nil, err
	}

	return database, nil
}

// Helper that recursively unwraps the bundle.
func (db *RunCmdBundle) unbundle(database *RunCmd) error {
	if db == nil {
		return nil
	}

	for head := db; head != nil && head.option != nil; head = head.next {
		var err error
		switch opt := head.option.(type) {
		case *RunCmdBundle:
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
func (db *RunCmdBundle) String() string {
	if db == nil {
		return ""
	}

	str := ""
	for head := db; head != nil && head.option != nil; head = head.next {
		switch opt := head.option.(type) {
		case *RunCmdBundle:
			str += opt.String()
		case optionFunc:
			str += reflect.TypeOf(opt).String() + "\n"
		}
	}

	return str
}

// ReadPreference sets the read preference.
func ReadPreference(rp *readpref.ReadPref) Option {
	return optionFunc(
		func(rc *RunCmd) error {
			if rc.ReadPreference == nil {
				rc.ReadPreference = rp
			}
			return nil
		})
}
