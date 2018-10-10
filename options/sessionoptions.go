package options

import (
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

// DefaultCausalConsistency is the default value for the CausalConsistency option.
var DefaultCausalConsistency = true

// SessionOptions represents all possible options for creating a new session.
type SessionOptions struct {
	CausalConsistency     *bool                      // Specifies if reads should be causally consistent. Defaults to true.
	DefaultReadConcern    *readconcern.ReadConcern   // The default read concern for transactions started in the session.
	DefaultReadPreference *readpref.ReadPref         // The default read preference for transactions started in the session.
	DefaultWriteConcern   *writeconcern.WriteConcern // The default write concern for transactions started in the session.
}

// Session creates a new *SessionOptions
func Session() *SessionOptions {
	return &SessionOptions{
		CausalConsistency: &DefaultCausalConsistency,
	}
}

// SetCausalConsistency specifies if a session should be causally consistent. Defaults to true.
func (s *SessionOptions) SetCausalConsistency(b bool) *SessionOptions {
	s.CausalConsistency = &b
	return s
}

// SetDefaultReadConcern sets the default read concern for transactions started in a session.
func (s *SessionOptions) SetDefaultReadConcern(rc *readconcern.ReadConcern) *SessionOptions {
	s.DefaultReadConcern = rc
	return s
}

// SetDefaultReadPreference sets the default read preference for transactions started in a session.
func (s *SessionOptions) SetDefaultReadPreference(rp *readpref.ReadPref) *SessionOptions {
	s.DefaultReadPreference = rp
	return s
}

// SetDefaultWriteConcern sets the default write concern for transactions started in a session.
func (s *SessionOptions) SetDefaultWriteConcern(wc *writeconcern.WriteConcern) *SessionOptions {
	s.DefaultWriteConcern = wc
	return s
}

// MergeSessionOptions combines the given *SessionOptions into a single *SessionOptions in a last one wins fashion.
func MergeSessionOptions(opts ...*SessionOptions) *SessionOptions {
	s := Session()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.CausalConsistency != nil {
			s.CausalConsistency = opt.CausalConsistency
		}
		if opt.DefaultReadConcern != nil {
			s.DefaultReadConcern = opt.DefaultReadConcern
		}
		if opt.DefaultReadPreference != nil {
			s.DefaultReadPreference = opt.DefaultReadPreference
		}
		if opt.DefaultWriteConcern != nil {
			s.DefaultWriteConcern = opt.DefaultWriteConcern
		}
	}

	return s
}
