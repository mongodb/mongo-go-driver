package listdbopt

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/session"
)

var listDatabasesBundle = new(ListDatabasesBundle)

// ListDatabases represents all possible params for the listDatabases() function.
type ListDatabases interface {
	listDatabases()
}

// ListDatabasesOption represents the options for the listDatabases() function.
type ListDatabasesOption interface {
	ListDatabases
	ConvertListDatabasesOption() option.ListDatabasesOptioner
}

// ListDatabasesSession is the session for the ListDatabases() function.
type ListDatabasesSession interface {
	ListDatabases
	ConvertListDatabasesSession() *session.Client
}

// ListDatabasesBundle is a bundle of ListDatabases options
type ListDatabasesBundle struct {
	option ListDatabases
	next   *ListDatabasesBundle
}

// Implement the ListDatabases interface
func (lcb *ListDatabasesBundle) listDatabases() {}

// ConvertListDatabasesOption implements the ListDatabases interface
func (lcb *ListDatabasesBundle) ConvertListDatabasesOption() option.ListDatabasesOptioner {
	return nil
}

// BundleListDatabases bundles ListDatabases options
func BundleListDatabases(opts ...ListDatabases) *ListDatabasesBundle {
	head := listDatabasesBundle

	for _, opt := range opts {
		newBundle := ListDatabasesBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// NameOnly adds an option to specify whether to return only the collection names.
func (lcb *ListDatabasesBundle) NameOnly(b bool) *ListDatabasesBundle {
	bundle := &ListDatabasesBundle{
		option: NameOnly(b),
		next:   lcb,
	}

	return bundle
}

// Unbundle transforms a bundle into a slice of options, optionally deduplicating.
func (lcb *ListDatabasesBundle) Unbundle(deduplicate bool) ([]option.ListDatabasesOptioner, *session.Client, error) {
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
func (lcb *ListDatabasesBundle) bundleLength() int {
	if lcb == nil {
		return 0
	}

	bundleLen := 0
	for ; lcb != nil; lcb = lcb.next {
		if lcb.option == nil {
			continue
		}
		if converted, ok := lcb.option.(*ListDatabasesBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		if _, ok := lcb.option.(ListDatabasesSessionOpt); !ok {
			bundleLen++
		}
	}

	return bundleLen
}

// Helper that recursively unwraps bundle into slice of options
func (lcb *ListDatabasesBundle) unbundle() ([]option.ListDatabasesOptioner, *session.Client, error) {
	if lcb == nil {
		return nil, nil, nil
	}

	var sess *session.Client
	listLen := lcb.bundleLength()

	options := make([]option.ListDatabasesOptioner, listLen)
	index := listLen - 1

	for listHead := lcb; listHead != nil; listHead = listHead.next {
		if listHead.option == nil {
			continue
		}

		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*ListDatabasesBundle); ok {
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
		case ListDatabasesOption:
			options[index] = t.ConvertListDatabasesOption()
			index--
		case ListDatabasesSession:
			if sess == nil {
				sess = t.ConvertListDatabasesSession()
			}
		}
	}

	return options, sess, nil
}

// String implements the Stringer interface
func (lcb *ListDatabasesBundle) String() string {
	if lcb == nil {
		return ""
	}

	str := ""
	for head := lcb; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*ListDatabasesBundle); ok {
			str += converted.String()
			continue
		}

		if conv, ok := head.option.(ListDatabasesOption); !ok {
			str += conv.ConvertListDatabasesOption().String() + "\n"
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

func (OptNameOnly) listDatabases() {}

// ConvertListDatabasesOption implements the ListDatabases interface.
func (opt OptNameOnly) ConvertListDatabasesOption() option.ListDatabasesOptioner {
	return option.OptNameOnly(opt)
}

// ListDatabasesSessionOpt is a listDatabases session option.
type ListDatabasesSessionOpt struct{}

func (ListDatabasesSessionOpt) listDatabases() {}

// ConvertListDatabasesSession implements the ListDatabasesSession interface.
func (ListDatabasesSessionOpt) ConvertListDatabasesSession() *session.Client {
	return nil
}
