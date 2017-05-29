package readpref

import (
	"time"

	"github.com/10gen/mongo-go-driver/model"
)

// Option configures a read preference
type Option func(*ReadPref) error

// WithMaxStaleness sets the maximum staleness a
// server is allowed.
func WithMaxStaleness(ms time.Duration) Option {
	return func(rp *ReadPref) error {
		rp.maxStaleness = ms
		rp.maxStalenessSet = true
		return nil
	}
}

// WithTags sets a single tag set used to match
// a server. The last call to WithTags or WithTagSets
// overrides all previous calls to either method.
func WithTags(tags ...string) Option {
	return WithTagSets(model.NewTagSet(tags...))
}

// WithTagSets sets the tag sets used to match
// a server. The last call to WithTags or WithTagSets
// overrides all previous calls to either method.
func WithTagSets(tagSets ...model.TagSet) Option {
	return func(rp *ReadPref) error {
		rp.tagSets = tagSets
		return nil
	}
}
