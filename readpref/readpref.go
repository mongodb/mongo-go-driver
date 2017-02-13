package readpref

import (
	"time"

	"github.com/10gen/mongo-go-driver/desc"
)

// New constructs a read preference from the mode and optionally
// some name value pairs.
func New(mode Mode, tagSets ...desc.TagSet) *ReadPref {
	// TODO: think about having an error for invalid construction

	return &ReadPref{
		maxStaleness: time.Duration(-1) * time.Millisecond,
		mode:         mode,
		tagSets:      tagSets,
	}
}

// NewWithMaxStaleness constructs a read preference from a mode, a
// maximum staleness, and optionally some name value pairs.
func NewWithMaxStaleness(mode Mode, maxStaleness time.Duration, tagSets ...desc.TagSet) *ReadPref {
	// TODO: think about having an error for invalid construction

	return &ReadPref{
		maxStaleness:    maxStaleness,
		maxStalenessSet: true,
		mode:            mode,
		tagSets:         tagSets,
	}
}

var defaultReadPref = &ReadPref{}

// Default returns the default read preference.
func Default() *ReadPref {
	return defaultReadPref
}

// ReadPref determines which servers are considered suitable for read operations.
type ReadPref struct {
	maxStaleness    time.Duration
	maxStalenessSet bool
	mode            Mode
	tagSets         []desc.TagSet
}

// MaxStaleness is the maximum amount of time to allow
// a server to be considered eligible for selection. The
// second return value indicates if this value has been set.
func (r *ReadPref) MaxStaleness() (time.Duration, bool) {
	return r.maxStaleness, r.maxStalenessSet
}

// Mode indicates the mode of the read preference.
func (r *ReadPref) Mode() Mode {
	return r.mode
}

// TagSets are multiple tag sets indicating
// which servers should be considered.
func (r *ReadPref) TagSets() []desc.TagSet {
	return r.tagSets
}
