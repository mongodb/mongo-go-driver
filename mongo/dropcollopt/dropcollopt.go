package dropcollopt

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/session"
)

var dropCollBundle = new(DropCollBundle)

// DropColl represents all possible params for the dropColl() function.
type DropColl interface {
	dropColl()
}

// DropCollOption represents the options for the dropColl() function.
type DropCollOption interface {
	DropColl
	ConvertDropCollOption() option.DropCollectionsOptioner
}

// DropCollSession is the session for the DropColl() function.
type DropCollSession interface {
	DropColl
	ConvertDropCollSession() *session.Client
}

// DropCollBundle is a bundle of DropColl options
type DropCollBundle struct {
	option DropColl
	next   *DropCollBundle
}

// Implement the DropColl interface
func (lcb *DropCollBundle) dropColl() {}

// ConvertDropCollOption implements the DropColl interface
func (lcb *DropCollBundle) ConvertDropCollOption() option.DropCollectionsOptioner {
	return nil
}

// BundleDropColl bundles DropColl options
func BundleDropColl(opts ...DropColl) *DropCollBundle {
	head := dropCollBundle

	for _, opt := range opts {
		newBundle := DropCollBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// Unbundle transforms a bundle into a slice of options, optionally deduplicating.
func (lcb *DropCollBundle) Unbundle(deduplicate bool) ([]option.DropCollectionsOptioner, *session.Client, error) {
	options, sess, err := lcb.unbundle()
	if err != nil {
		return nil, nil, err
	}

	if !deduplicate {
		return options, sess, nil
	}

	// iterate backwards and make dedup slice
	optionsSet := make(map[reflect.Type]struct{})

	for i := len(options) - 1; i >= 0; i-- {
		currOption := options[i]
		optionType := reflect.TypeOf(currOption)

		if _, ok := optionsSet[optionType]; ok {
			// option already found
			options = append(options[:i], options[i+1:]...)
			continue
		}

		optionsSet[optionType] = struct{}{}
	}

	return options, sess, nil
}

// Calculates the total length of a bundle, accounting for nested bundles.
func (lcb *DropCollBundle) bundleLength() int {
	if lcb == nil {
		return 0
	}

	bundleLen := 0
	for ; lcb != nil; lcb = lcb.next {
		if lcb.option == nil {
			continue
		}
		if converted, ok := lcb.option.(*DropCollBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		if _, ok := lcb.option.(DropCollSessionOpt); !ok {
			bundleLen++
		}
	}

	return bundleLen
}

// Helper that recursively unwraps bundle into slice of options
func (lcb *DropCollBundle) unbundle() ([]option.DropCollectionsOptioner, *session.Client, error) {
	if lcb == nil {
		return nil, nil, nil
	}

	var sess *session.Client
	listLen := lcb.bundleLength()

	options := make([]option.DropCollectionsOptioner, listLen)
	index := listLen - 1

	for listHead := lcb; listHead != nil; listHead = listHead.next {
		if listHead.option == nil {
			continue
		}

		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*DropCollBundle); ok {
			nestedOptions, s, err := converted.unbundle()
			if err != nil {
				return nil, nil, err
			}
			if s != nil && sess == nil {
				sess = s
			}

			// where to start inserting nested options
			startIndex := index - len(nestedOptions) + 1

			// add nested options in order
			for _, nestedOp := range nestedOptions {
				options[startIndex] = nestedOp
				startIndex++
			}
			index -= len(nestedOptions)
			continue
		}

		switch t := listHead.option.(type) {
		case DropCollOption:
			options[index] = t.ConvertDropCollOption()
			index--
		case DropCollSession:
			if sess == nil {
				sess = t.ConvertDropCollSession()
			}
		}
	}

	return options, sess, nil
}

// String implements the Stringer interface
func (lcb *DropCollBundle) String() string {
	if lcb == nil {
		return ""
	}

	str := ""
	for head := lcb; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*DropCollBundle); ok {
			str += converted.String()
			continue
		}

		if conv, ok := head.option.(DropCollOption); !ok {
			str += conv.ConvertDropCollOption().String() + "\n"
		}
	}

	return str
}

// DropCollSessionOpt is a dropColl session option.
type DropCollSessionOpt struct{}

func (DropCollSessionOpt) dropColl() {}

// ConvertDropCollSession implements the DropCollSession interface.
func (DropCollSessionOpt) ConvertDropCollSession() *session.Client {
	return nil
}
