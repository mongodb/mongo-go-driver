package readpref

import (
	"fmt"
	"strings"
	"time"

	"github.com/10gen/mongo-go-driver/server"
)

// Primary constructs a read preference with a PrimaryMode.
func Primary() *ReadPref {
	return new(PrimaryMode)
}

// PrimaryPreferred constructs a read preference with a PrimaryPreferredMode.
func PrimaryPreferred(opts ...Option) *ReadPref {
	return new(PrimaryPreferredMode, opts...)
}

// SecondaryPreferred constructs a read preference with a SecondaryPreferredMode.
func SecondaryPreferred(opts ...Option) *ReadPref {
	return new(SecondaryPreferredMode, opts...)
}

// Secondary constructs a read preference with a SecondaryMode.
func Secondary(opts ...Option) *ReadPref {
	return new(SecondaryMode, opts...)
}

// Nearest constructs a read preference with a NearestMode.
func Nearest(opts ...Option) *ReadPref {
	return new(NearestMode, opts...)
}

func new(mode Mode, opts ...Option) *ReadPref {
	rp := &ReadPref{
		mode: mode,
	}

	for _, opt := range opts {
		opt(rp)
	}

	return rp
}

// ReadPref determines which servers are considered suitable for read operations.
type ReadPref struct {
	maxStaleness    time.Duration
	maxStalenessSet bool
	mode            Mode
	tagSets         []server.TagSet
}

// ModeFromString returns a mode corresponding to
// mode.
func ModeFromString(mode string) (Mode, error) {
	switch strings.ToLower(mode) {
	case "primary":
		return PrimaryMode, nil
	case "primarypreferred":
		return PrimaryPreferredMode, nil
	case "secondary":
		return SecondaryMode, nil
	case "secondarypreferred":
		return SecondaryPreferredMode, nil
	case "nearest":
		return NearestMode, nil
	}
	return Mode(uint8(0)), fmt.Errorf("unknown read preference %v", mode)
}

// WithMode takes a mode and creates a read preference using that mode.
func WithMode(m Mode) *ReadPref {
	return new(m)
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
