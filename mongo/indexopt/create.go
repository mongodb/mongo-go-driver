package indexopt

import (
	"reflect"
	"time"

	"github.com/mongodb/mongo-go-driver/core/option"
)

var createBundle = new(CreateBundle)

// Create is options for the createIndexes command.
type Create interface {
	create()
	ConvertCreateOption() option.CreateIndexesOptioner
}

// CreateBundle is a bundle of Create options
type CreateBundle struct {
	option Create
	next   *CreateBundle
}

// BundleCreate bundles Create options
func BundleCreate(opts ...Create) *CreateBundle {
	head := createBundle

	for _, opt := range opts {
		newBundle := CreateBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

func (cb *CreateBundle) create() {}

// ConvertCreateOption implements the Create interface.
func (cb *CreateBundle) ConvertCreateOption() option.CreateIndexesOptioner { return nil }

// MaxTime adds an option to specify the maximum amount of time to allow the query to run.
func (cb *CreateBundle) MaxTime(d time.Duration) *CreateBundle {
	bundle := &CreateBundle{
		option: MaxTime(d),
		next:   cb,
	}

	return bundle
}

// Unbundle unwinds and deduplicates the options used to create it and those
// added after creation into a single slice of options.
//
// The deduplicate parameter is used to determine if the bundle is just flattened or
// if we actually deduplicate options.
func (cb *CreateBundle) Unbundle(deduplicate bool) ([]option.CreateIndexesOptioner, error) {
	options, err := cb.unbundle()
	if err != nil {
		return nil, err
	}

	if !deduplicate {
		return options, nil
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

	return options, nil
}

// Calculates the total length of a bundle, accounting for nested bundles.
func (cb *CreateBundle) bundleLength() int {
	if cb == nil {
		return 0
	}

	bundleLen := 0
	for ; cb != nil && cb.option != nil; cb = cb.next {
		if converted, ok := cb.option.(*CreateBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}
		bundleLen++
	}

	return bundleLen
}

// Helper that recursively unwraps bundle into slice of options
func (cb *CreateBundle) unbundle() ([]option.CreateIndexesOptioner, error) {
	if cb == nil {
		return nil, nil
	}

	listLen := cb.bundleLength()
	options := make([]option.CreateIndexesOptioner, listLen)
	index := listLen - 1

	for listHead := cb; listHead != nil && listHead.option != nil; listHead = listHead.next {
		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*CreateBundle); ok {
			nestedOptions, err := converted.unbundle()
			if err != nil {
				return nil, err
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
		options[index] = listHead.option.ConvertCreateOption()
		index--
	}

	return options, nil
}

// String implements the Stringer interface
func (cb *CreateBundle) String() string {
	if cb == nil {
		return ""
	}

	str := ""
	for head := cb; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*CreateBundle); ok {
			str += converted.String()
			continue
		}

		str += head.option.ConvertCreateOption().String() + "\n"
	}

	return str
}
