package listcollectionopt

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/session"
)

var listCollectionsBundle = new(ListCollectionsBundle)

// ListCollections represents all possible params for the listCollections() function.
type ListCollections interface {
	listCollections()
}

// ListCollectionsOption represents the options for the listCollections() function.
type ListCollectionsOption interface {
	ListCollections
	ConvertListCollectionsOption() option.ListCollectionsOptioner
}

// ListCollectionsSession is the session for the ListCollections() function.
type ListCollectionsSession interface {
	ListCollections
	ConvertListCollectionsSession() *session.Client
}

// ListCollectionsBundle is a bundle of ListCollections options
type ListCollectionsBundle struct {
	option ListCollections
	next   *ListCollectionsBundle
}

// Implement the ListCollections interface
func (cb *ListCollectionsBundle) listCollections() {}

// ConvertListCollectionsOption implements the ListCollections interface
func (cb *ListCollectionsBundle) ConvertListCollectionsOption() option.ListCollectionsOptioner {
	return nil
}

// BundleListCollections bundles ListCollections options
func BundleListCollections(opts ...ListCollections) *ListCollectionsBundle {
	head := listCollectionsBundle

	for _, opt := range opts {
		newBundle := ListCollectionsBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// NameOnly adds an option to specify whether to return only the collection names.
func (lcb *ListCollectionsBundle) NameOnly(b bool) *ListCollectionsBundle {
	bundle := &ListCollectionsBundle{
		option: NameOnly(b),
		next:   lcb,
	}

	return bundle
}

// Unbundle transforms a bundle into a slice of options, optionally deduplicating.
func (cb *ListCollectionsBundle) Unbundle(deduplicate bool) ([]option.ListCollectionsOptioner, *session.Client, error) {
	options, sess, err := cb.unbundle()
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
func (cb *ListCollectionsBundle) bundleLength() int {
	if cb == nil {
		return 0
	}

	bundleLen := 0
	for ; cb != nil && cb.option != nil; cb = cb.next {
		if converted, ok := cb.option.(*ListCollectionsBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		bundleLen++
	}

	return bundleLen
}

// Helper that recursively unwraps bundle into slice of options
func (cb *ListCollectionsBundle) unbundle() ([]option.ListCollectionsOptioner, *session.Client, error) {
	if cb == nil {
		return nil, nil, nil
	}

	var sess *session.Client
	listLen := cb.bundleLength()

	options := make([]option.ListCollectionsOptioner, listLen)
	index := listLen - 1

	for listHead := cb; listHead != nil && listHead.option != nil; listHead = listHead.next {
		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*ListCollectionsBundle); ok {
			nestedOptions, s, err := converted.unbundle()
			if err != nil {
				return nil, nil, err
			}
			if s != nil {
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
		case ListCollectionsOption:
			options[index] = t.ConvertListCollectionsOption()
			index--
		case ListCollectionsSession:
			sess = t.ConvertListCollectionsSession()
		}
	}

	return options, sess, nil
}

// String implements the Stringer interface
func (cb *ListCollectionsBundle) String() string {
	if cb == nil {
		return ""
	}

	str := ""
	for head := cb; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*ListCollectionsBundle); ok {
			str += converted.String()
			continue
		}

		if conv, ok := head.option.(ListCollectionsOption); !ok {
			str += conv.ConvertListCollectionsOption().String() + "\n"
		}
	}

	return str
}

// NameOnly specifies whether to return only the collection names.
func NameOnly(b bool) OptNameOnly {
	return OptNameOnly(b)
}

// OptNameOnly specifies whether to return only the collection names.
type OptNameOnly option.OptNameOnly

func (OptNameOnly) listCollections() {}

// ConvertListCollectionsOption implements the ListCollections interface.
func (opt OptNameOnly) ConvertListCollectionsOption() option.ListCollectionsOptioner {
	return option.OptNameOnly(opt)
}

// ListCollectionsSessionOpt is a listCollections session option.
type ListCollectionsSessionOpt struct{}

func (ListCollectionsSessionOpt) listCollections() {}

// ConvertListCollectionsSession implements the ListCollectionsSession interface.
func (ListCollectionsSessionOpt) ConvertListCollectionsSession() *session.Client {
	return nil
}
