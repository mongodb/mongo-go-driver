package readpref

import (
	"time"

	"github.com/10gen/mongo-go-driver/server"
)

// Option configures a read preference
type Option func(*ReadPref)

// WithMaxStaleness sets the maximum staleness a
// server is allowed.
func WithMaxStaleness(ms time.Duration) Option {
	return func(rp *ReadPref) {
		rp.maxStaleness = ms
		rp.maxStalenessSet = true
	}
}

// WithTags sets a single tag set used to match
// a server. The last call to WithTags or WithTagSets
// overrides all previous calls to either method.
func WithTags(tags ...string) Option {
	return WithTagSets(server.NewTagSet(tags...))
}

// WithTagSets sets the tag sets used to match
// a server. The last call to WithTags or WithTagSets
// overrides all previous calls to either method.
func WithTagSets(tagSets ...server.TagSet) Option {
	return func(rp *ReadPref) {
		if rp.mode != PrimaryMode {
			rp.tagSets = tagSets
		} else {
			panic("can't set tag set on primary")
		}
	}
}
