package sessionopt

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/core/session"
)

var sessionBundle = new(SessionBundle)

// Session represents options for creating client sessions.
type Session interface {
	session()
	ConvertSessionOption() session.ClientOptioner
}

// SessionBundle bundles session options
type SessionBundle struct {
	option Session
	next   *SessionBundle
}

func (sb *SessionBundle) session() {}

// ConvertSessionOption implements the Session interface
func (sb *SessionBundle) ConvertSessionOption() session.ClientOptioner {
	return nil
}

// BundleSession bundles session options
func BundleSession(opts ...Session) *SessionBundle {
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

// CausalConsistency specifies if a session should be causally consistent. Defaults to true. Causally consistent reads
// are not causally consistent with unacknowledged writes.
func (sb *SessionBundle) CausalConsistency(b bool) *SessionBundle {
	return &SessionBundle{
		option: CausalConsistency(b),
		next:   sb,
	}
}

// Unbundle transforms a bundle into a slice of options, optionally deduplicating.
func (sb *SessionBundle) Unbundle(deduplicate bool) ([]session.ClientOptioner, error) {
	opts, err := sb.unbundle()
	if err != nil {
		return nil, err
	}

	if !deduplicate {
		return opts, nil
	}

	optionsSet := make(map[reflect.Type]struct{})

	for i := len(opts) - 1; i >= 0; i-- {
		currOpt := opts[i]
		optType := reflect.TypeOf(currOpt)

		if _, ok := optionsSet[optType]; ok {
			// already found
			opts = append(opts[:i], opts[i+1:]...)
			continue
		}

		optionsSet[optType] = struct{}{}
	}

	return opts, nil
}

func (sb *SessionBundle) unbundle() ([]session.ClientOptioner, error) {
	if sb == nil {
		return nil, nil
	}

	listLen := sb.bundleLength()

	options := make([]session.ClientOptioner, listLen)
	index := listLen - 1

	for listHead := sb; listHead != nil && listHead.option != nil; listHead = listHead.next {
		if converted, ok := listHead.option.(*SessionBundle); ok {
			nestedOpts, err := converted.unbundle()
			if err != nil {
				return nil, err
			}

			startIndex := index - (len(nestedOpts)) + 1

			for _, nestedOpt := range nestedOpts {
				options[startIndex] = nestedOpt
				startIndex++
			}

			index -= len(nestedOpts)
			continue
		}

		options[index] = listHead.option.ConvertSessionOption()
		index--
	}

	return options, nil
}

// Calculates the total length of a bundle, accounting for nested bundles.
func (sb *SessionBundle) bundleLength() int {
	if sb == nil {
		return 0
	}

	bundleLen := 0
	for ; sb != nil && sb.option != nil; sb = sb.next {
		if converted, ok := sb.option.(*SessionBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		bundleLen++
	}

	return bundleLen
}

// CausalConsistency specifies if a client session should be causally consistent. Causally consistent reads are not
// causally consistent with unacknowledged writes.
func CausalConsistency(b bool) OptCausalConsistency {
	return OptCausalConsistency(b)
}

// OptCausalConsistency specifies if a client session should be causally consistent.
type OptCausalConsistency session.OptCausalConsistency

func (OptCausalConsistency) session() {}

// ConvertSessionOption implements the Session interface.
func (opt OptCausalConsistency) ConvertSessionOption() session.ClientOptioner {
	return session.OptCausalConsistency(opt)
}
