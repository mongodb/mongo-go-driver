package deleteopt

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

var deleteBundle = new(DeleteBundle)

// Delete is options for the delete() function.
type Delete interface {
	delete()
	ConvertDeleteOption() option.DeleteOptioner
}

// DeleteBundle is a bundle of Delete options
type DeleteBundle struct {
	option Delete
	next   *DeleteBundle
}

func (db *DeleteBundle) delete() {}

// ConvertDeleteOption implements the Delete interface.
func (db *DeleteBundle) ConvertDeleteOption() option.DeleteOptioner {
	return nil
}

// BundleDelete bundles Delete options.
func BundleDelete(opts ...Delete) *DeleteBundle {
	head := deleteBundle

	for _, opt := range opts {
		newBundle := DeleteBundle{
			option: opt,
			next:   head,
		}

		head = &newBundle
	}

	return head
}

// Collation adds an option to specify a collation.
func (db *DeleteBundle) Collation(c *mongoopt.Collation) *DeleteBundle {
	bundle := &DeleteBundle{
		option: Collation(c),
		next:   db,
	}

	return bundle
}

// Unbundle transforms a bundle into a slice of options, optionally deduplicating
func (db *DeleteBundle) Unbundle(deduplicate bool) ([]option.DeleteOptioner, error) {

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
func (db *DeleteBundle) bundleLength() int {
	if db == nil {
		return 0
	}

	bundleLen := 0
	for ; db != nil && db.option != nil; db = db.next {
		if converted, ok := db.option.(*DeleteBundle); ok {
			// nested bundle
			bundleLen += converted.bundleLength()
			continue
		}

		bundleLen++
	}

	return bundleLen
}

// Helper that recursively unwraps bundle into slice of options
func (db *DeleteBundle) unbundle() ([]option.DeleteOptioner, error) {
	if db == nil {
		return nil, nil
	}

	listLen := db.bundleLength()

	options := make([]option.DeleteOptioner, listLen)
	index := listLen - 1

	for listHead := db; listHead != nil && listHead.option != nil; listHead = listHead.next {
		// if the current option is a nested bundle, Unbundle it and add its options to the current array
		if converted, ok := listHead.option.(*DeleteBundle); ok {
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

		options[index] = listHead.option.ConvertDeleteOption()
		index--
	}

	return options, nil
}

// String implements the Stringer interface
func (db *DeleteBundle) String() string {
	if db == nil {
		return ""
	}

	str := ""
	for head := db; head != nil && head.option != nil; head = head.next {
		if converted, ok := head.option.(*DeleteBundle); ok {
			str += converted.String()
			continue
		}

		str += head.option.ConvertDeleteOption().String() + "\n"
	}

	return str
}

// Collation specifies a collation.
func Collation(c *mongoopt.Collation) OptCollation {
	return OptCollation{Collation: c.Convert()}
}

// OptCollation specifies a collation.
type OptCollation option.OptCollation

func (OptCollation) delete() {}

// ConvertDeleteOption implements the Delete interface.
func (opt OptCollation) ConvertDeleteOption() option.DeleteOptioner {
	return option.OptCollation(opt)
}
