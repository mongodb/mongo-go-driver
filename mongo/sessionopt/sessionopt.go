package sessionopt

var sessionBundle = new(SessionBundle)

// Option represents a session option
type Option interface {
	sessionOption()
}

type optionFunc func(s *Session) error

// Session represents a client session.
type Session struct {
	Consistent    bool
	ConsistentSet bool
}

// SessionBundle is a bundle of session options
type SessionBundle struct {
	option Option
	next   *SessionBundle
}

func (*SessionBundle) sessionOption() {}
func (optionFunc) sessionOption()     {}

// BundleSession bundles session options
func BundleSession(opts ...Option) *SessionBundle {
	head := sessionBundle

	for _, opt := range opts {
		newBundle := SessionBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// CausalConsistency specifies if a session should be causally consistent.
func (sb *SessionBundle) CausalConsistency(b bool) *SessionBundle {
	return &SessionBundle{
		option: CausalConsistency(b),
		next:   sb,
	}
}

// Unbundle creates a Session.
func (sb *SessionBundle) Unbundle() (*Session, error) {
	session := &Session{}

	err := sb.unbundle(session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// Helper that recursively unwraps bundle.
func (sb *SessionBundle) unbundle(s *Session) error {
	if sb == nil {
		return nil
	}

	for head := sb; head != nil && head.option != nil; head = head.next {
		var err error
		switch opt := head.option.(type) {
		case *SessionBundle:
			err = opt.unbundle(s) // add all bundle's options to client
		case optionFunc:
			err = opt(s) // add option to client
		default:
			return nil
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// CausalConsistency specifies if a session should be causally consistent.
func CausalConsistency(b bool) Option {
	return optionFunc(
		func(s *Session) error {
			if !s.ConsistentSet {
				s.Consistent = b
				s.ConsistentSet = true
			}

			return nil
		})
}
