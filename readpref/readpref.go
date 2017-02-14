package readpref

import (
	"time"

	"github.com/10gen/mongo-go-driver/server"
)

// New constructs a read preference from the mode and options.
func New(mode Mode, opts ...Option) *ReadPref {
	rp := &ReadPref{
		mode: mode,
	}

	for _, opt := range opts {
		opt(rp)
	}

	return rp
}

// Primary constructs a read preference with a PrimaryMode.
func Primary() *ReadPref {
	return New(PrimaryMode)
}

// PrimaryPreferred constructs a read preference with a PrimaryPreferredMode.
func PrimaryPreferred(opts ...Option) *ReadPref {
	return New(PrimaryPreferredMode, opts...)
}

// SecondaryPreferred constructs a read preference with a SecondaryPreferredMode.
func SecondaryPreferred(opts ...Option) *ReadPref {
	return New(SecondaryPreferredMode, opts...)
}

// Secondary constructs a read preference with a SecondaryMode.
func Secondary(opts ...Option) *ReadPref {
	return New(SecondaryMode, opts...)
}

// Nearest constructs a read preference with a NearestMode.
func Nearest(opts ...Option) *ReadPref {
	return New(NearestMode, opts...)
}

// ReadPref determines which servers are considered suitable for read operations.
type ReadPref struct {
	maxStaleness    time.Duration
	maxStalenessSet bool
	mode            Mode
	tagSets         []server.TagSet
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
func (r *ReadPref) TagSets() []server.TagSet {
	return r.tagSets
}
