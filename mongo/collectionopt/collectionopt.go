package collectionopt

import (
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/mongo"
	"reflect"
)

var collectionBundle = new (CollectionBundle)

// Option represents a collection option.
type Option interface {
	collectionOption()
}

// optionFunc adds the option to the client.
type optionFunc func(*Collection) error

// Collection represents a collection.
type Collection struct {
	client         *mongo.Client
	db             *mongo.Database
	name           string
	readConcern    *readconcern.ReadConcern
	writeConcern   *writeconcern.WriteConcern
	readPreference *readpref.ReadPref
	readSelector   description.ServerSelector
	writeSelector  description.ServerSelector
}

// CollectionBUndle is a bundle of collection options.
type CollectionBundle struct {
	option Option
	next *CollectionBundle
}

// CollectionBundle implements Option.
func (*CollectionBundle) collectionOption() {}
// OptionFunc implements Option.
func (optionFunc) collectionOption() {}

// BundleCollection bundles collection options.
func BundleCollection(opts ...Option) *CollectionBundle {
	head := collectionBundle

	for _, opt := range opts {
		newBundle := CollectionBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// ReadConcern sets the read concern.
func (cb *CollectionBundle) ReadConcern(rc *readconcern.ReadConcern) *CollectionBundle {
	return &CollectionBundle{
		option: ReadConcern(rc),
		next:   cb,
	}
}

// WriteConcern sets the write concern.
func (cb *CollectionBundle) WriteConcern(wc *writeconcern.WriteConcern) *CollectionBundle {
	return &CollectionBundle{
		option: WriteConcern(wc),
		next:   cb,
	}
}

// ReadPreference sets the read preference.
func (cb *CollectionBundle) ReadPreference(rp *readpref.ReadPref) *CollectionBundle {
	return &CollectionBundle{
		option: ReadPreference(rp),
		next:   cb,
	}
}

// String prints a string representation of the bundle for debug purposes
func (cb *CollectionBundle) String() string {
	if cb == nil {
		return ""
	}

	debugStr := ""
	for head := cb; head != nil && head.option != nil; head = head.next {
		switch opt := head.option.(type) {
		case *CollectionBundle:
			debugStr += opt.String() + "\n"
		case optionFunc:
			debugStr += reflect.TypeOf(opt).String() + "\n"
		default:
			return debugStr + "(error: CollectionOption can only be *CollectionBundle or optionFunc)"
		}
	}

	return debugStr
}

// Unbundle unbundles the options, returning a collection.
func (cb *CollectionBundle) Unbundle() (*Collection, error) {
	client := &Collection{}
	err := cb.unbundle(client)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Helper that recursively unwraps the bundle.
func (cb *CollectionBundle) unbundle(client *Collection) error{
	if cb == nil {
		return nil
	}

	for head := cb; head != nil && head.option != nil; head = head.next {
		var err error
		switch opt := head.option.(type) {
		case *CollectionBundle:
			err = opt.unbundle(client) // add all bundle's options to client
		case optionFunc:
			err = opt(client) // add option to client
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
		func(c *Collection) error {
			if c.readConcern == nil {
				c.readConcern = rc
			}
			return nil
		})
}

// WriteConcern sets the write concern.
func WriteConcern(wc *writeconcern.WriteConcern) Option {
	return optionFunc(
		func(c *Collection) error {
			if c.writeConcern == nil {
				c.writeConcern = wc
			}
			return nil
		})
}

// ReadPreference sets the read preference.
func ReadPreference(rp *readpref.ReadPref) Option {
	return optionFunc(
		func(c *Collection) error {
			if c.readPreference == nil {
				c.readPreference = rp
			}
			return nil
		})
}
