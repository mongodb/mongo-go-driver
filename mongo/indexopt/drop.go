package indexopt

import (
	"reflect"
	"time"

	"github.com/mongodb/mongo-go-driver/core/option"
)

var dropBundle = new(DropBundle)

// Drop is options for the dropIndexes command.
type Drop interface {
	drop()
	ConvertDropOption() option.DropIndexesOptioner
}

// DropBundle is a bundle of Drop options
type DropBundle struct {
	option Drop
	next   *DropBundle
}

// BundleDrop bundles Drop options
func BundleDrop(opts ...Drop) *DropBundle {
	head := dropBundle

	for _, opt := range opts {
		newBundle := DropBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

func (db *DropBundle) drop() {}

// ConvertDropOption implements the Drop interface
func (db *DropBundle) ConvertDropOption() option.DropIndexesOptioner { return nil }

// MaxTime adds an option to specify the maximum amount of time to allow the query to run.
func (db *DropBundle) MaxTime(d time.Duration) *DropBundle {
	bundle := &DropBundle{
		option: MaxTime(d),
		next:   db,
	}

	return bundle
}

// Unbundle unwinds and deduplicates the options used to create it and those
// added after creation into a single slice of options.
//
// The deduplicate parameter is used to determine if the bundle is just flattened or
// if we actually deduplicate options.
func (db *DropBundle) Unbundle(deduplicate bool) ([]option.DropIndexesOptioner, error) {
	options, err := db.unbundle()
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
func (db *DropBundle) bundleLength() int {
	if db == nil {
		return 0
	}

	bundleLen := 0
	for ; db != nil && db.option != nil; db = db.next {
		if converted, ok := db.option.(*DropBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}
		bundleLen++
	}

	return bundleLen
}

// Helper that recursively unwraps bundle into slice of options
func (db *DropBundle) unbundle() ([]option.DropIndexesOptioner, error) {
	if db == nil {
		return nil, nil
	}

	listLen := db.bundleLength()
	options := make([]option.DropIndexesOptioner, listLen)
	index := listLen - 1

	for listHead := db; listHead != nil && listHead.option != nil; listHead = listHead.next {
		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*DropBundle); ok {
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
		options[index] = listHead.option.ConvertDropOption()
		index--
	}

	return options, nil
}

// String implements the Stringer interface
func (db *DropBundle) String() string {
	if db == nil {
		return ""
	}

	str := ""
	for head := db; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*DropBundle); ok {
			str += converted.String()
			continue
		}

		str += head.option.ConvertDropOption().String() + "\n"
	}

	return str
}
