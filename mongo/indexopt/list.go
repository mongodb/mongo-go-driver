package indexopt

import (
	"reflect"
	"time"

	"github.com/mongodb/mongo-go-driver/core/option"
)

var listBundle = new(ListBundle)

// List is options for the listIndexes command
type List interface {
	list()
	ConvertListOption() option.ListIndexesOptioner
}

// ListBundle is a bundle of List options.
type ListBundle struct {
	option List
	next   *ListBundle
}

// BundleList bundles List options
func BundleList(opts ...List) *ListBundle {
	head := listBundle

	for _, opt := range opts {
		newBundle := ListBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

func (lb *ListBundle) list() {}

// ConvertListOption implements the List interface.
func (lb *ListBundle) ConvertListOption() option.ListIndexesOptioner { return nil }

// BatchSize adds an option to specify the number of documents to return in every batch
func (lb *ListBundle) BatchSize(i int32) *ListBundle {
	bundle := &ListBundle{
		option: BatchSize(i),
		next:   lb,
	}

	return bundle
}

// MaxTime adds an option to specify the maximum amount of time to allow the query to run.
func (lb *ListBundle) MaxTime(d time.Duration) *ListBundle {
	bundle := &ListBundle{
		option: MaxTime(d),
		next:   lb,
	}

	return bundle
}

// Unbundle unwinds and deduplicates the options used to create it and those
// added after creation into a single slice of options.
//
// The deduplicate parameter is used to determine if the bundle is just flattened or
// if we actually deduplicate options.
func (lb *ListBundle) Unbundle(deduplicate bool) ([]option.ListIndexesOptioner, error) {
	options, err := lb.unbundle()
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
func (lb *ListBundle) bundleLength() int {
	if lb == nil {
		return 0
	}

	bundleLen := 0
	for ; lb != nil && lb.option != nil; lb = lb.next {
		if converted, ok := lb.option.(*ListBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}
		bundleLen++
	}

	return bundleLen
}

// Helper that recursively unwraps bundle into slice of options
func (lb *ListBundle) unbundle() ([]option.ListIndexesOptioner, error) {
	if lb == nil {
		return nil, nil
	}

	listLen := lb.bundleLength()
	options := make([]option.ListIndexesOptioner, listLen)
	index := listLen - 1

	for listHead := lb; listHead != nil && listHead.option != nil; listHead = listHead.next {
		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*ListBundle); ok {
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
		options[index] = listHead.option.ConvertListOption()
		index--
	}

	return options, nil
}

// String implements the Stringer interface
func (lb *ListBundle) String() string {
	if lb == nil {
		return ""
	}

	str := ""
	for head := lb; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*ListBundle); ok {
			str += converted.String()
			continue
		}

		str += head.option.ConvertListOption().String() + "\n"
	}

	return str
}
